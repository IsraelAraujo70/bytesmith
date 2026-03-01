package main

import (
	"context"
	"fmt"
	"time"

	"bytesmith/internal/acp"
	"bytesmith/internal/session"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------------------
// Session lifecycle and prompts
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
