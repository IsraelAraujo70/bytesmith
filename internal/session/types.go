package session

import "time"

// Message represents a single message in a session's conversation history.
type Message struct {
	ID        string
	Role      string // "user", "agent", "system"
	Content   string
	Timestamp time.Time
}

// ToolCallRecord tracks a tool invocation made during a session.
type ToolCallRecord struct {
	ID        string
	Title     string
	Kind      string
	Status    string
	Content   string // summary of the result
	Timestamp time.Time
}

// SessionRecord holds the full state of a single agent session including
// its conversation history and tool call records.
type SessionRecord struct {
	ID           string
	AgentName    string
	ConnectionID string
	CWD          string
	Messages     []Message
	ToolCalls    []ToolCallRecord
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
