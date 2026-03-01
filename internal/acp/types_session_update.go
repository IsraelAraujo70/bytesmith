package acp

import (
	"encoding/json"
	"fmt"
)

// SessionUpdateParams wraps a session update notification sent by the agent.
type SessionUpdateParams struct {
	SessionID string        `json:"sessionId"`
	Update    SessionUpdate `json:"update"`
}

// SessionUpdate type constants.
const (
	UpdateAgentMessageChunk = "agent_message_chunk"
	UpdateUserMessageChunk  = "user_message_chunk"
	UpdateAgentThoughtChunk = "agent_thought_chunk"
	UpdateToolCall          = "tool_call"
	UpdateToolCallUpdate    = "tool_call_update"
	UpdatePlan              = "plan"
	UpdateAvailableCommands = "available_commands_update"
)

// SessionUpdate represents a single update from the agent during a session.
//
// The ACP spec overloads the JSON "content" field for both message chunks
// (a single ContentBlock) and tool calls ([]ToolCallContent). We resolve
// this with separate Go fields and a custom JSON un/marshaler.
type SessionUpdate struct {
	// Type is the discriminator (JSON key: "sessionUpdate").
	Type string `json:"-"`

	// MessageContent is populated for message/thought chunk updates.
	MessageContent *ContentBlock `json:"-"`

	// ToolCallID identifies the tool call (for tool_call / tool_call_update).
	ToolCallID string `json:"toolCallId,omitempty"`

	// Title is a human-readable label for the tool call.
	Title string `json:"title,omitempty"`

	// Kind categorizes the tool call: read, edit, delete, move, search,
	// execute, think, fetch, other.
	Kind string `json:"kind,omitempty"`

	// Status of the tool call: pending, in_progress, completed, failed.
	Status string `json:"status,omitempty"`

	// ToolContent is the structured content of a tool call or its result.
	ToolContent []ToolCallContent `json:"-"`

	// Locations are file references associated with a tool call.
	Locations []ToolCallLocation `json:"locations,omitempty"`

	// RawInput is the raw JSON input sent to the tool (for debugging).
	RawInput json.RawMessage `json:"rawInput,omitempty"`

	// RawOutput is the raw JSON output returned by the tool (for debugging).
	RawOutput json.RawMessage `json:"rawOutput,omitempty"`

	// Entries is populated for plan updates.
	Entries []PlanEntry `json:"entries,omitempty"`

	// AvailableCommands is populated for available_commands_update.
	AvailableCommands []AvailableCommand `json:"availableCommands,omitempty"`
}

// sessionUpdateJSON is the raw JSON shape used for custom un/marshaling.
// It mirrors the wire format where "content" is overloaded.
type sessionUpdateJSON struct {
	SessionUpdate     string             `json:"sessionUpdate"`
	Content           json.RawMessage    `json:"content,omitempty"`
	ToolCallID        string             `json:"toolCallId,omitempty"`
	Title             string             `json:"title,omitempty"`
	Kind              string             `json:"kind,omitempty"`
	Status            string             `json:"status,omitempty"`
	Locations         []ToolCallLocation `json:"locations,omitempty"`
	RawInput          json.RawMessage    `json:"rawInput,omitempty"`
	RawOutput         json.RawMessage    `json:"rawOutput,omitempty"`
	Entries           []PlanEntry        `json:"entries,omitempty"`
	AvailableCommands []AvailableCommand `json:"availableCommands,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling to resolve the "content" field
// ambiguity. For message chunks, "content" is a single ContentBlock object.
// For tool calls, "content" is an array of ToolCallContent objects.
func (u *SessionUpdate) UnmarshalJSON(data []byte) error {
	var raw sessionUpdateJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal SessionUpdate: %w", err)
	}

	u.Type = raw.SessionUpdate
	u.ToolCallID = raw.ToolCallID
	u.Title = raw.Title
	u.Kind = raw.Kind
	u.Status = raw.Status
	u.Locations = raw.Locations
	u.RawInput = raw.RawInput
	u.RawOutput = raw.RawOutput
	u.Entries = raw.Entries
	u.AvailableCommands = raw.AvailableCommands

	if len(raw.Content) == 0 {
		return nil
	}

	switch raw.SessionUpdate {
	case UpdateAgentMessageChunk, UpdateUserMessageChunk, UpdateAgentThoughtChunk:
		var cb ContentBlock
		if err := json.Unmarshal(raw.Content, &cb); err != nil {
			return fmt.Errorf("unmarshal message content: %w", err)
		}
		u.MessageContent = &cb

	case UpdateToolCall, UpdateToolCallUpdate:
		var tcc []ToolCallContent
		if err := json.Unmarshal(raw.Content, &tcc); err != nil {
			return fmt.Errorf("unmarshal tool call content: %w", err)
		}
		u.ToolContent = tcc

	default:
		// Unknown update type: try array first, then single object.
		var tcc []ToolCallContent
		if err := json.Unmarshal(raw.Content, &tcc); err == nil {
			u.ToolContent = tcc
		} else {
			var cb ContentBlock
			if err2 := json.Unmarshal(raw.Content, &cb); err2 == nil {
				u.MessageContent = &cb
			}
		}
	}

	return nil
}

// MarshalJSON implements custom marshaling that writes the correct "content"
// field based on the update type.
func (u SessionUpdate) MarshalJSON() ([]byte, error) {
	raw := sessionUpdateJSON{
		SessionUpdate:     u.Type,
		ToolCallID:        u.ToolCallID,
		Title:             u.Title,
		Kind:              u.Kind,
		Status:            u.Status,
		Locations:         u.Locations,
		RawInput:          u.RawInput,
		RawOutput:         u.RawOutput,
		Entries:           u.Entries,
		AvailableCommands: u.AvailableCommands,
	}

	switch u.Type {
	case UpdateAgentMessageChunk, UpdateUserMessageChunk, UpdateAgentThoughtChunk:
		if u.MessageContent != nil {
			b, err := json.Marshal(u.MessageContent)
			if err != nil {
				return nil, err
			}
			raw.Content = b
		}
	case UpdateToolCall, UpdateToolCallUpdate:
		if u.ToolContent != nil {
			b, err := json.Marshal(u.ToolContent)
			if err != nil {
				return nil, err
			}
			raw.Content = b
		}
	}

	return json.Marshal(raw)
}

// ToolCallContent represents a single content element within a tool call.
type ToolCallContent struct {
	// Type discriminator: content, diff, terminal.
	Type string `json:"type"`

	// Content is set when Type is "content".
	Content *ContentBlock `json:"content,omitempty"`

	// Diff fields (when Type is "diff").
	Path    string `json:"path,omitempty"`
	OldText string `json:"oldText,omitempty"`
	NewText string `json:"newText,omitempty"`

	// Terminal fields (when Type is "terminal").
	TerminalID string `json:"terminalId,omitempty"`
}

// ToolCallLocation is a file path and optional line number associated with
// a tool call.
type ToolCallLocation struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
}

// PlanEntry is a single item in a plan update.
type PlanEntry struct {
	Content  string `json:"content"`
	Priority string `json:"priority,omitempty"` // high, medium, low
	Status   string `json:"status,omitempty"`   // pending, in_progress, completed
}

// AvailableCommand describes a slash command or action available in the session.
type AvailableCommand struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       *AvailableCommandInput `json:"input,omitempty"`
}

// AvailableCommandInput describes expected input for an available command.
type AvailableCommandInput struct {
	Hint string `json:"hint"`
}
