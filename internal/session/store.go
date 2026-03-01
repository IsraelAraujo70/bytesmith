package session

import (
	"log"
	"os"
	"path/filepath"
)

// Store is the session persistence contract used by the app.
type Store interface {
	Create(id, agentName, connectionID, cwd string) *SessionRecord
	Get(id string) *SessionRecord
	AddMessage(sessionID string, msg Message)
	AddToolCall(sessionID string, tc ToolCallRecord)
	UpdateToolCall(
		sessionID,
		toolCallID,
		status,
		content string,
		parts []ToolCallPart,
		diffSummary ToolCallDiffSummary,
	)
	List() []*SessionRecord
	Delete(id string)
	Close() error
}

// NewStore returns a SQLite-backed store with an in-memory fallback.
func NewStore() Store {
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("session: user config dir unavailable, using memory store: %v", err)
		return NewMemoryStore()
	}

	dbPath := filepath.Join(dir, "bytesmith", "sessions.db")
	sqliteStore, err := NewSQLiteStore(dbPath)
	if err != nil {
		log.Printf("session: sqlite unavailable, using memory store: %v", err)
		return NewMemoryStore()
	}

	return sqliteStore
}
