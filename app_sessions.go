package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bytesmith/internal/acp"
	"bytesmith/internal/session"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------------------
// Session management
// ---------------------------------------------------------------------------

// NewSession creates a new session on an existing agent connection and returns
// the session ID.
func (a *App) NewSession(connectionID, cwd string) (string, error) {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return "", fmt.Errorf("connection %q not found", connectionID)
	}

	result, err := conn.Client.NewSession(context.Background(), cwd, nil)
	if err != nil {
		return "", err
	}
	sessionID := result.SessionID

	// Track session locally.
	a.sessions.Create(sessionID, conn.Agent.Name, connectionID, cwd)
	appendSessionIfMissing(conn, sessionID)

	if result.Models != nil {
		models := make([]SessionModelInfo, 0, len(result.Models.AvailableModels))
		for _, m := range result.Models.AvailableModels {
			models = append(models, SessionModelInfo{
				ModelID: m.ModelID,
				Name:    m.Name,
			})
		}

		info := SessionModelsInfo{
			CurrentModelID: result.Models.CurrentModelID,
			Models:         models,
		}

		a.sessionModelsMu.Lock()
		a.sessionModels[sessionID] = info
		a.sessionModelsMu.Unlock()

		wailsRuntime.EventsEmit(a.ctx, "agent:models", map[string]interface{}{
			"connectionId":   connectionID,
			"sessionId":      sessionID,
			"currentModelId": info.CurrentModelID,
			"models":         info.Models,
		})
	}

	if modes, ok := resolveSessionModes(conn.IntegratorID, result.Modes); ok {
		a.sessionModesMu.Lock()
		a.sessionModes[sessionID] = modes
		a.sessionModesMu.Unlock()
		a.emitSessionModes(connectionID, sessionID, modes)
	}

	return sessionID, nil
}

// SendPrompt sends a user prompt to the agent asynchronously. Real-time
// updates arrive via Wails events ("agent:message", "agent:toolcall", etc.).
// When the agent finishes, an "agent:prompt-done" event is emitted.
func (a *App) SendPrompt(connectionID, sessionID, text string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}

	// Record the user message.
	a.sessions.AddMessage(sessionID, session.Message{
		ID:      uuid.NewString(),
		Role:    "user",
		Content: text,
	})

	go func() {
		// A prompt can take a very long time; use a generous timeout.
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
		defer cancel()

		a.activePromptsMu.Lock()
		a.activePrompts[sessionID] = cancel
		a.activePromptsMu.Unlock()

		defer func() {
			a.activePromptsMu.Lock()
			delete(a.activePrompts, sessionID)
			a.activePromptsMu.Unlock()
		}()

		prompt := []acp.ContentBlock{
			{Type: "text", Text: text},
		}

		result, err := conn.Client.Prompt(ctx, sessionID, prompt)
		if err != nil {
			a.finalizeStreamMessage(connectionID, sessionID)
			wailsRuntime.EventsEmit(a.ctx, "agent:error", map[string]string{
				"connectionId": connectionID,
				"sessionId":    sessionID,
				"error":        err.Error(),
			})
			return
		}

		a.finalizeStreamMessage(connectionID, sessionID)
		wailsRuntime.EventsEmit(a.ctx, "agent:prompt-done", map[string]string{
			"connectionId": connectionID,
			"sessionId":    sessionID,
			"stopReason":   result.StopReason,
		})
	}()

	return nil
}

// CancelPrompt cancels an in-progress prompt by sending the ACP cancel
// notification and aborting the local context.
func (a *App) CancelPrompt(connectionID, sessionID string) error {
	// Cancel the local context so the Prompt call unblocks.
	a.activePromptsMu.Lock()
	cancel, ok := a.activePrompts[sessionID]
	a.activePromptsMu.Unlock()
	if ok {
		cancel()
	}

	// Also tell the agent to stop.
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}
	return conn.Client.Cancel(sessionID)
}

// GetSessionHistory returns the full conversation history for a session.
func (a *App) GetSessionHistory(sessionID string) *SessionHistoryInfo {
	rec := a.sessions.Get(sessionID)
	if rec == nil {
		return nil
	}

	messages := make([]MessageInfo, 0, len(rec.Messages))
	for _, m := range rec.Messages {
		messages = append(messages, MessageInfo{
			ID:        m.ID,
			Role:      m.Role,
			Content:   m.Content,
			Timestamp: m.Timestamp.Format(time.RFC3339),
		})
	}

	toolCalls := make([]ToolCallInfo, 0, len(rec.ToolCalls))
	for _, tc := range rec.ToolCalls {
		toolCalls = append(toolCalls, toToolCallInfo(tc))
	}

	return &SessionHistoryInfo{
		ID:           rec.ID,
		AgentName:    rec.AgentName,
		ConnectionID: rec.ConnectionID,
		CWD:          rec.CWD,
		Messages:     messages,
		ToolCalls:    toolCalls,
		CreatedAt:    rec.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    rec.UpdatedAt.Format(time.RFC3339),
	}
}

// ListSessions returns lightweight summaries for all sessions.
func (a *App) ListSessions() []SessionListItem {
	records := a.sessions.List()
	result := make([]SessionListItem, 0, len(records))
	for _, r := range records {
		result = append(result, SessionListItem{
			ID:           r.ID,
			AgentName:    r.AgentName,
			ConnectionID: r.ConnectionID,
			CWD:          r.CWD,
			MessageCount: len(r.Messages),
			CreatedAt:    r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result
}

// ResumeHistoricalSession opens a previously persisted session and tries to
// reattach it to a live connection for continued chat.
func (a *App) ResumeHistoricalSession(sessionID string) (ResumeHistoricalResult, error) {
	rec := a.sessions.Get(sessionID)
	if rec == nil {
		return ResumeHistoricalResult{}, fmt.Errorf("session %q not found", sessionID)
	}

	result := ResumeHistoricalResult{
		ConnectionID: rec.ConnectionID,
		SessionID:    sessionID,
		Resumed:      false,
	}

	conn := a.manager.GetConnection(rec.ConnectionID)
	if conn == nil {
		conn = findConnectionByAgent(a.manager.ListConnections(), rec.AgentName)
	}

	if conn == nil {
		connected, err := a.manager.Connect(rec.AgentName, rec.CWD)
		if err != nil {
			result.Reason = fmt.Sprintf("failed to connect agent: %v", err)
			return result, nil
		}
		a.wireConnection(connected)
		conn = connected
	}

	result.ConnectionID = conn.ID

	if err := a.ResumeSession(conn.ID, sessionID, rec.CWD); err != nil {
		result.Reason = err.Error()
		return result, nil
	}

	result.Resumed = true
	return result, nil
}

// GetSessionModels returns the known model selection info for a session.
func (a *App) GetSessionModels(sessionID string) *SessionModelsInfo {
	a.sessionModelsMu.RLock()
	defer a.sessionModelsMu.RUnlock()

	info, ok := a.sessionModels[sessionID]
	if !ok {
		return nil
	}

	copyModels := make([]SessionModelInfo, len(info.Models))
	copy(copyModels, info.Models)

	result := info
	result.Models = copyModels
	return &result
}

// SetSessionModel updates the current model for a session.
func (a *App) SetSessionModel(connectionID, sessionID, modelID string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}

	if err := conn.Client.SetModel(context.Background(), sessionID, modelID); err != nil {
		return err
	}

	a.sessionModelsMu.Lock()
	info, ok := a.sessionModels[sessionID]
	if ok {
		info.CurrentModelID = modelID
		a.sessionModels[sessionID] = info
	}
	a.sessionModelsMu.Unlock()

	if ok {
		wailsRuntime.EventsEmit(a.ctx, "agent:models", map[string]interface{}{
			"connectionId":   connectionID,
			"sessionId":      sessionID,
			"currentModelId": modelID,
			"models":         info.Models,
		})
	}

	return nil
}

// GetSessionModes returns the known mode/profile selection info for a session.
func (a *App) GetSessionModes(sessionID string) *SessionModesInfo {
	a.sessionModesMu.RLock()
	defer a.sessionModesMu.RUnlock()

	info, ok := a.sessionModes[sessionID]
	if !ok {
		return nil
	}

	copyModes := make([]SessionModeInfo, len(info.Modes))
	copy(copyModes, info.Modes)

	result := info
	result.Modes = copyModes
	return &result
}

// SetSessionMode updates the current mode for a session.
func (a *App) SetSessionMode(connectionID, sessionID, modeID string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}
	if err := conn.Client.SetMode(context.Background(), sessionID, modeID); err != nil {
		return err
	}

	a.sessionModesMu.Lock()
	info, ok := a.sessionModes[sessionID]
	if !ok && conn.IntegratorID == "codex" {
		info = codexFallbackModes()
		ok = true
	}
	if ok {
		found := false
		for _, mode := range info.Modes {
			if mode.ModeID == modeID {
				found = true
				break
			}
		}
		if !found && modeID != "" {
			info.Modes = append(info.Modes, SessionModeInfo{
				ModeID: modeID,
				Name:   modeID,
			})
		}
		info.CurrentModeID = modeID
		a.sessionModes[sessionID] = info
	}
	a.sessionModesMu.Unlock()

	if ok {
		a.emitSessionModes(connectionID, sessionID, info)
	}

	return nil
}

// SetSessionConfigOption sets a generic session config option.
func (a *App) SetSessionConfigOption(connectionID, sessionID, configID, value string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}
	return conn.Client.SetConfigOption(context.Background(), sessionID, configID, value)
}

func codexFallbackModes() SessionModesInfo {
	return SessionModesInfo{
		CurrentModeID: "restricted",
		Modes: []SessionModeInfo{
			{ModeID: "full-access", Name: "Full Access"},
			{ModeID: "restricted", Name: "Restricted"},
			{ModeID: "plan", Name: "Plan"},
		},
	}
}

func resolveSessionModes(integratorID string, state *acp.SessionModesState) (SessionModesInfo, bool) {
	if state != nil {
		modes := make([]SessionModeInfo, 0, len(state.AvailableModes))
		for _, m := range state.AvailableModes {
			name := m.Name
			if strings.TrimSpace(name) == "" {
				name = m.ID
			}
			modes = append(modes, SessionModeInfo{
				ModeID: m.ID,
				Name:   name,
			})
		}

		current := state.CurrentModeID
		if current == "" && len(modes) > 0 {
			current = modes[0].ModeID
		}

		if len(modes) > 0 || current != "" {
			return SessionModesInfo{
				CurrentModeID: current,
				Modes:         modes,
			}, true
		}
	}

	if integratorID == "codex" {
		return codexFallbackModes(), true
	}

	return SessionModesInfo{}, false
}

func (a *App) emitSessionModes(connectionID, sessionID string, info SessionModesInfo) {
	wailsRuntime.EventsEmit(a.ctx, "agent:modes", map[string]interface{}{
		"connectionId":  connectionID,
		"sessionId":     sessionID,
		"currentModeId": info.CurrentModeID,
		"modes":         info.Modes,
	})
}
