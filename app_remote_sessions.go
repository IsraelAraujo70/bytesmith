package main

import (
	"context"
	"fmt"
	"strings"

	"bytesmith/internal/integrator"
)

// ListRemoteSessions lists sessions directly from the connected integrator.
func (a *App) ListRemoteSessions(connectionID, cwd, cursor string) (SessionListPage, error) {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return SessionListPage{}, fmt.Errorf("connection %q not found", connectionID)
	}

	if !integrator.ForAgent(conn.Agent.Name).Capabilities().ListSessions {
		return SessionListPage{Unsupported: true}, nil
	}

	list, err := conn.Client.ListSessions(context.Background(), cwd, cursor)
	if err != nil {
		if strings.Contains(err.Error(), "method not found") || strings.Contains(err.Error(), "unknown variant") {
			return SessionListPage{Unsupported: true}, nil
		}
		return SessionListPage{}, err
	}

	sessions := make([]SessionListItem, 0, len(list.Sessions))
	for _, s := range list.Sessions {
		sessions = append(sessions, SessionListItem{
			ID:           s.SessionID,
			AgentName:    conn.Agent.Name,
			ConnectionID: connectionID,
			CWD:          s.CWD,
			CreatedAt:    "",
			UpdatedAt:    s.UpdatedAt,
		})
	}

	return SessionListPage{
		Sessions:   sessions,
		NextCursor: list.NextCursor,
	}, nil
}

// LoadRemoteSession asks the remote agent to load a session and tracks it locally.
func (a *App) LoadRemoteSession(connectionID, sessionID, cwd string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}

	if !integrator.ForAgent(conn.Agent.Name).Capabilities().LoadSession {
		return fmt.Errorf("integrator %q does not support session load", conn.Agent.Name)
	}

	if err := conn.Client.LoadSession(context.Background(), sessionID, cwd, nil); err != nil {
		return err
	}

	a.sessions.Create(sessionID, conn.Agent.Name, connectionID, cwd)
	appendSessionIfMissing(conn, sessionID)
	return nil
}

// ResumeSession asks the remote agent to resume a session and tracks it locally.
func (a *App) ResumeSession(connectionID, sessionID, cwd string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}

	caps := integrator.ForAgent(conn.Agent.Name).Capabilities()
	if !caps.ResumeSession {
		if !caps.LoadSession {
			return fmt.Errorf("integrator %q does not support session resume", conn.Agent.Name)
		}
		return a.LoadRemoteSession(connectionID, sessionID, cwd)
	}

	result, err := conn.Client.ResumeSession(context.Background(), sessionID, cwd, nil)
	if err != nil {
		return err
	}

	a.sessions.Create(sessionID, conn.Agent.Name, connectionID, cwd)
	appendSessionIfMissing(conn, sessionID)

	if result != nil && result.Models != nil {
		models := make([]SessionModelInfo, 0, len(result.Models.AvailableModels))
		for _, m := range result.Models.AvailableModels {
			models = append(models, SessionModelInfo{
				ModelID: m.ModelID,
				Name:    m.Name,
			})
		}
		a.sessionModelsMu.Lock()
		a.sessionModels[sessionID] = SessionModelsInfo{
			CurrentModelID: result.Models.CurrentModelID,
			Models:         models,
		}
		a.sessionModelsMu.Unlock()
	}

	if result != nil {
		if modes, ok := resolveSessionModes(conn.IntegratorID, result.Modes); ok {
			a.sessionModesMu.Lock()
			a.sessionModes[sessionID] = modes
			a.sessionModesMu.Unlock()
			a.emitSessionModes(connectionID, sessionID, modes)
		}
	}

	return nil
}
