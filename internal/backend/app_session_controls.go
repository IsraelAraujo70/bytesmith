package backend

import (
	"context"
	"fmt"
	"strings"

	"bytesmith/internal/acp"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------------------
// Session model/mode controls
// ---------------------------------------------------------------------------

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
