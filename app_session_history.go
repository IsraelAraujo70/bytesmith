package main

import (
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// Session history and reattach
// ---------------------------------------------------------------------------

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
