// Package acp implements the Agent Client Protocol (ACP) types and client.
// ACP uses JSON-RPC 2.0 over stdio for communication between a client
// (this desktop app) and an AI coding agent subprocess.
// Spec: https://agentclientprotocol.com
package acp

import (
	"encoding/json"
	"fmt"
)

// ---------------------------------------------------------------------------
// JSON-RPC 2.0 base types
// ---------------------------------------------------------------------------

// JSONRPCMessage represents a JSON-RPC 2.0 message. It can be a request,
// response, or notification depending on which fields are populated.
//
// A request has Method and optionally Params, plus an ID.
// A notification has Method and optionally Params, but no ID.
// A response has ID and either Result or Error.
type JSONRPCMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

// IsRequest returns true if the message is a request (has method and ID).
func (m *JSONRPCMessage) IsRequest() bool {
	return m.Method != "" && m.ID != nil
}

// IsNotification returns true if the message is a notification (has method but no ID).
func (m *JSONRPCMessage) IsNotification() bool {
	return m.Method != "" && m.ID == nil
}

// IsResponse returns true if the message is a response (has ID but no method).
func (m *JSONRPCMessage) IsResponse() bool {
	return m.Method == "" && m.ID != nil
}

// IDAsInt64 parses the message ID as an int64. Returns 0 if the ID is nil
// or cannot be parsed as a number.
func (m *JSONRPCMessage) IDAsInt64() int64 {
	if m.ID == nil {
		return 0
	}
	var id int64
	if err := json.Unmarshal(*m.ID, &id); err != nil {
		return 0
	}
	return id
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

// Standard JSON-RPC 2.0 error codes.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

// ---------------------------------------------------------------------------
// Initialize
// ---------------------------------------------------------------------------

// InitializeParams is sent by the client as the first message after starting
// the agent process. It advertises the client's capabilities and identity.
type InitializeParams struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities"`
	ClientInfo         ImplementationInfo `json:"clientInfo"`
}

// InitializeResult is the agent's response to the initialize request.
// It advertises the agent's capabilities, identity, and any required auth.
type InitializeResult struct {
	ProtocolVersion   int                `json:"protocolVersion"`
	AgentCapabilities AgentCapabilities  `json:"agentCapabilities"`
	AgentInfo         ImplementationInfo `json:"agentInfo"`
	AuthMethods       []AuthMethod       `json:"authMethods,omitempty"`
}

// ClientCapabilities describes what the client can do on behalf of the agent.
type ClientCapabilities struct {
	FS       *FSCapabilities `json:"fs,omitempty"`
	Terminal bool            `json:"terminal,omitempty"`
}

// FSCapabilities describes which file system operations the client supports.
type FSCapabilities struct {
	ReadTextFile  bool `json:"readTextFile,omitempty"`
	WriteTextFile bool `json:"writeTextFile,omitempty"`
}

// AgentCapabilities describes what the agent supports.
type AgentCapabilities struct {
	LoadSession         bool                 `json:"loadSession,omitempty"`
	PromptCapabilities  *PromptCapabilities  `json:"promptCapabilities,omitempty"`
	MCP                 *MCPCapabilities     `json:"mcp,omitempty"`
	SessionCapabilities *SessionCapabilities `json:"sessionCapabilities,omitempty"`
}

// PromptCapabilities describes what content types the agent accepts in prompts.
type PromptCapabilities struct {
	Image           bool `json:"image,omitempty"`
	Audio           bool `json:"audio,omitempty"`
	EmbeddedContext bool `json:"embeddedContext,omitempty"`
}

// MCPCapabilities describes which MCP (Model Context Protocol) transports are supported.
type MCPCapabilities struct {
	HTTP bool `json:"http,omitempty"`
	SSE  bool `json:"sse,omitempty"`
}

// SessionCapabilities is a future extension point for session-level features.
type SessionCapabilities struct{}

// ImplementationInfo identifies an ACP implementation (client or agent).
type ImplementationInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// AuthMethod describes an authentication method the agent requires.
type AuthMethod struct {
	Type string `json:"type"`
}

// ---------------------------------------------------------------------------
// Session management
// ---------------------------------------------------------------------------

// SessionNewParams requests the agent to create a new session.
type SessionNewParams struct {
	CWD        string      `json:"cwd"`
	MCPServers []MCPServer `json:"mcpServers,omitempty"`
}

// SessionNewResult is the agent's response to a session/new request.
type SessionNewResult struct {
	SessionID string `json:"sessionId"`
}

// SessionLoadParams requests the agent to reload an existing session.
type SessionLoadParams struct {
	SessionID  string      `json:"sessionId"`
	CWD        string      `json:"cwd"`
	MCPServers []MCPServer `json:"mcpServers,omitempty"`
}

// MCPServer describes an MCP server to attach to the session.
// For stdio transport, Command and Args are used.
// For HTTP transport, Type, URL, and Headers are used.
type MCPServer struct {
	Name    string        `json:"name"`
	Command string        `json:"command,omitempty"`
	Args    []string      `json:"args,omitempty"`
	Env     []EnvVariable `json:"env,omitempty"`
	Type    string        `json:"type,omitempty"`
	URL     string        `json:"url,omitempty"`
	Headers []HTTPHeader  `json:"headers,omitempty"`
}

// EnvVariable is a name/value pair for environment variables.
type EnvVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HTTPHeader is a name/value pair for HTTP headers.
type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ---------------------------------------------------------------------------
// Prompt
// ---------------------------------------------------------------------------

// SessionPromptParams sends a user prompt to an active session.
type SessionPromptParams struct {
	SessionID string         `json:"sessionId"`
	Prompt    []ContentBlock `json:"prompt"`
}

// SessionPromptResult is returned when the agent finishes processing a prompt.
type SessionPromptResult struct {
	// StopReason indicates why the agent stopped.
	// Possible values: end_turn, max_tokens, max_turn_requests, refusal, cancelled.
	StopReason string `json:"stopReason"`
}

// SessionCancelParams requests cancellation of an in-progress prompt.
type SessionCancelParams struct {
	SessionID string `json:"sessionId"`
}

// ---------------------------------------------------------------------------
// Content blocks
// ---------------------------------------------------------------------------

// ContentBlock represents a piece of content in a prompt or agent response.
// The Type field determines which other fields are relevant.
type ContentBlock struct {
	// Type of content: text, image, audio, resource, resource_link.
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	Resource *Resource `json:"resource,omitempty"`
	// Image/audio fields.
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// Resource represents an embedded or linked resource.
type Resource struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// ---------------------------------------------------------------------------
// Session updates (notifications from agent -> client)
// ---------------------------------------------------------------------------

// SessionUpdateParams wraps a session update notification sent by the agent.
type SessionUpdateParams struct {
	SessionID string        `json:"sessionId"`
	Update    SessionUpdate `json:"update"`
}

// SessionUpdate type constants.
const (
	UpdateAgentMessageChunk = "agent_message_chunk"
	UpdateUserMessageChunk  = "user_message_chunk"
	UpdateToolCall          = "tool_call"
	UpdateToolCallUpdate    = "tool_call_update"
	UpdatePlan              = "plan"
	UpdateAvailableCommands = "available_commands_update"
)

// SessionUpdate represents a single update from the agent during a session.
//
// The update type is indicated by the Type field. Different fields are
// populated depending on the type:
//
//   - agent_message_chunk / user_message_chunk: MessageContent is set.
//   - tool_call / tool_call_update: ToolCallID, Title, Kind, Status,
//     ToolContent, Locations, RawInput, RawOutput may be set.
//   - plan: Entries is set.
//   - available_commands_update: AvailableCommands is set.
//
// The ACP spec overloads the JSON "content" field for both message chunks
// (a single ContentBlock) and tool calls ([]ToolCallContent). We resolve
// this with separate Go fields and a custom JSON un/marshaler.
type SessionUpdate struct {
	// Type is the discriminator (JSON key: "sessionUpdate").
	Type string `json:"-"`

	// MessageContent is populated for agent_message_chunk / user_message_chunk.
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
	case UpdateAgentMessageChunk, UpdateUserMessageChunk:
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
	case UpdateAgentMessageChunk, UpdateUserMessageChunk:
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

// ---------------------------------------------------------------------------
// Permission (agent -> client request)
// ---------------------------------------------------------------------------

// RequestPermissionParams is sent by the agent to ask the user for permission
// before performing a sensitive action.
type RequestPermissionParams struct {
	SessionID string             `json:"sessionId"`
	ToolCall  ToolCallUpdate     `json:"toolCall"`
	Options   []PermissionOption `json:"options"`
}

// ToolCallUpdate carries tool call details within a permission request.
type ToolCallUpdate struct {
	ToolCallID string            `json:"toolCallId"`
	Title      string            `json:"title,omitempty"`
	Kind       string            `json:"kind,omitempty"`
	Status     string            `json:"status,omitempty"`
	Content    []ToolCallContent `json:"content,omitempty"`
}

// PermissionOption is a single choice presented to the user.
type PermissionOption struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name"`
	// Kind: allow_once, allow_always, reject_once, reject_always.
	Kind string `json:"kind"`
}

// RequestPermissionResult is the client's response to a permission request.
type RequestPermissionResult struct {
	Outcome PermissionOutcome `json:"outcome"`
}

// PermissionOutcome describes the user's decision on a permission request.
type PermissionOutcome struct {
	// Outcome: selected, cancelled.
	Outcome  string `json:"outcome"`
	OptionID string `json:"optionId,omitempty"`
}

// ---------------------------------------------------------------------------
// File system (agent -> client requests)
// ---------------------------------------------------------------------------

// FSReadTextFileParams requests the client to read a text file on disk.
type FSReadTextFileParams struct {
	SessionID string `json:"sessionId"`
	Path      string `json:"path"`
	Line      int    `json:"line,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

// FSReadTextFileResult is the client's response containing file content.
type FSReadTextFileResult struct {
	Content string `json:"content"`
}

// FSWriteTextFileParams requests the client to write content to a text file.
type FSWriteTextFileParams struct {
	SessionID string `json:"sessionId"`
	Path      string `json:"path"`
	Content   string `json:"content"`
}

// ---------------------------------------------------------------------------
// Terminal (agent -> client requests)
// ---------------------------------------------------------------------------

// TerminalCreateParams requests the client to spawn a terminal subprocess.
type TerminalCreateParams struct {
	SessionID       string        `json:"sessionId"`
	Command         string        `json:"command"`
	Args            []string      `json:"args,omitempty"`
	Env             []EnvVariable `json:"env,omitempty"`
	CWD             string        `json:"cwd,omitempty"`
	OutputByteLimit int           `json:"outputByteLimit,omitempty"`
}

// TerminalCreateResult is returned after a terminal subprocess is created.
type TerminalCreateResult struct {
	TerminalID string `json:"terminalId"`
}

// TerminalOutputParams requests the current output of a terminal.
type TerminalOutputParams struct {
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
}

// TerminalOutputResult contains the terminal's accumulated output.
type TerminalOutputResult struct {
	Output     string              `json:"output"`
	Truncated  bool                `json:"truncated"`
	ExitStatus *TerminalExitStatus `json:"exitStatus,omitempty"`
}

// TerminalExitStatus describes how a terminal process exited.
type TerminalExitStatus struct {
	ExitCode *int   `json:"exitCode,omitempty"`
	Signal   string `json:"signal,omitempty"`
}

// TerminalWaitParams requests the client to block until a terminal exits.
type TerminalWaitParams struct {
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
}

// TerminalWaitResult is returned when the terminal process exits.
type TerminalWaitResult struct {
	ExitCode *int   `json:"exitCode,omitempty"`
	Signal   string `json:"signal,omitempty"`
}

// TerminalKillParams requests the client to kill a terminal process.
type TerminalKillParams struct {
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
}

// TerminalReleaseParams tells the client it may release terminal resources.
type TerminalReleaseParams struct {
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
}

// ---------------------------------------------------------------------------
// Session modes
// ---------------------------------------------------------------------------

// SessionSetModeParams requests the agent to switch its operating mode.
type SessionSetModeParams struct {
	SessionID string `json:"sessionId"`
	Mode      string `json:"mode"`
}

// ---------------------------------------------------------------------------
// ACP method names (JSON-RPC method strings)
// ---------------------------------------------------------------------------

const (
	MethodInitialize        = "initialize"
	MethodSessionNew        = "session/new"
	MethodSessionLoad       = "session/load"
	MethodSessionPrompt     = "session/prompt"
	MethodSessionCancel     = "session/cancel"
	MethodSessionSetMode    = "session/setMode"
	MethodSessionUpdate     = "session/update"
	MethodRequestPermission = "requestPermission"
	MethodFSReadTextFile    = "fs/readTextFile"
	MethodFSWriteTextFile   = "fs/writeTextFile"
	MethodTerminalCreate    = "terminal/create"
	MethodTerminalOutput    = "terminal/output"
	MethodTerminalWait      = "terminal/wait"
	MethodTerminalKill      = "terminal/kill"
	MethodTerminalRelease   = "terminal/release"
)
