package session

import "time"

// Message represents a single message in a session's conversation history.
type Message struct {
	ID        string
	Role      string // "user", "agent", "system"
	Content   string
	Timestamp time.Time
}

// ToolCallPart is one structured section inside a tool call update.
type ToolCallPart struct {
	Type       string
	Text       string
	Path       string
	OldText    string
	NewText    string
	TerminalID string
}

// ToolCallDiffSummary tracks aggregate line changes for diff parts.
type ToolCallDiffSummary struct {
	Additions int
	Deletions int
	Files     int
}

// ToolCallRecord tracks a tool invocation made during a session.
type ToolCallRecord struct {
	ID          string
	Title       string
	Kind        string
	Status      string
	Content     string // summary of the result
	Parts       []ToolCallPart
	DiffSummary ToolCallDiffSummary
	Timestamp   time.Time
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
