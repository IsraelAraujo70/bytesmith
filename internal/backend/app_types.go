package backend

import (
	"context"
	"strings"
	"sync"
	"time"

	"bytesmith/internal/acp"
	"bytesmith/internal/agent"
	bfs "bytesmith/internal/fs"
	"bytesmith/internal/session"
	"bytesmith/internal/terminal"
	"bytesmith/internal/uixterm"
)

// ---------------------------------------------------------------------------
// DTO types – JSON-serializable structs exposed to the frontend via Wails
// bindings. They intentionally avoid internal pointers so the TypeScript
// code generator produces clean interfaces.
// ---------------------------------------------------------------------------

// AgentInfo describes a supported agent runtime and whether it is installed locally.
type AgentInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Command     string `json:"command"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
}

// ConnectionInfo is a snapshot of a live agent connection.
type ConnectionInfo struct {
	ID          string   `json:"id"`
	AgentName   string   `json:"agentName"`
	DisplayName string   `json:"displayName"`
	Sessions    []string `json:"sessions"`
	Integrator  string   `json:"integrator"`
}

// SessionHistoryInfo carries the full conversation history for one session.
type SessionHistoryInfo struct {
	ID           string         `json:"id"`
	AgentName    string         `json:"agentName"`
	ConnectionID string         `json:"connectionId"`
	CWD          string         `json:"cwd"`
	Messages     []MessageInfo  `json:"messages"`
	ToolCalls    []ToolCallInfo `json:"toolCalls"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
}

// MessageInfo is a single message in a session's conversation.
type MessageInfo struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// ToolCallInfo is a single tool invocation record.
type ToolCallInfo struct {
	ID          string                   `json:"id"`
	Title       string                   `json:"title"`
	Kind        string                   `json:"kind"`
	Status      string                   `json:"status"`
	Content     string                   `json:"content"`
	Parts       []ToolCallPartInfo       `json:"parts,omitempty"`
	DiffSummary *ToolCallDiffSummaryInfo `json:"diffSummary,omitempty"`
	Timestamp   string                   `json:"timestamp"`
}

// ToolCallPartInfo is one structured section of a tool call payload.
type ToolCallPartInfo struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	Path       string `json:"path,omitempty"`
	OldText    string `json:"oldText,omitempty"`
	NewText    string `json:"newText,omitempty"`
	TerminalID string `json:"terminalId,omitempty"`
}

// ToolCallDiffSummaryInfo aggregates line changes for tool call diffs.
type ToolCallDiffSummaryInfo struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Files     int `json:"files"`
}

// SessionListItem is a lightweight summary for the session list view.
type SessionListItem struct {
	ID           string `json:"id"`
	AgentName    string `json:"agentName"`
	ConnectionID string `json:"connectionId"`
	CWD          string `json:"cwd"`
	MessageCount int    `json:"messageCount"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// AppSettingsInfo mirrors agent.AppSettings for frontend consumption.
type AppSettingsInfo struct {
	Theme        string `json:"theme"`
	DefaultAgent string `json:"defaultAgent"`
	DefaultCWD   string `json:"defaultCwd"`
	AutoApprove  bool   `json:"autoApprove"`
}

// FileEntry represents a single file or directory for the file explorer.
type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
}

// PermissionRequestInfo is emitted to the frontend when an agent asks for
// permission before performing a sensitive operation.
type PermissionRequestInfo struct {
	RequestID    string                 `json:"requestId"`
	ConnectionID string                 `json:"connectionId"`
	SessionID    string                 `json:"sessionId"`
	ToolCallID   string                 `json:"toolCallId"`
	Title        string                 `json:"title"`
	Kind         string                 `json:"kind"`
	Options      []PermissionOptionInfo `json:"options"`
}

// PermissionOptionInfo is one choice in a permission dialog.
type PermissionOptionInfo struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

// QuestionRequestInfo is emitted to the frontend when an agent asks explicit
// user input via item/tool/requestUserInput.
type QuestionRequestInfo struct {
	RequestID    string         `json:"requestId"`
	ConnectionID string         `json:"connectionId"`
	SessionID    string         `json:"sessionId"`
	ToolCallID   string         `json:"toolCallId"`
	Questions    []QuestionInfo `json:"questions"`
}

// QuestionInfo is one user-input question requested by the agent.
type QuestionInfo struct {
	ID       string               `json:"id"`
	Header   string               `json:"header"`
	Question string               `json:"question"`
	Multiple bool                 `json:"multiple"`
	IsOther  bool                 `json:"isOther"`
	IsSecret bool                 `json:"isSecret"`
	Options  []QuestionOptionInfo `json:"options"`
}

// QuestionOptionInfo is one selectable option for a question.
type QuestionOptionInfo struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// SessionModelInfo is one available model option for a session.
type SessionModelInfo struct {
	ModelID string `json:"modelId"`
	Name    string `json:"name"`
}

// SessionModelsInfo contains model selection state for a session.
type SessionModelsInfo struct {
	CurrentModelID string             `json:"currentModelId"`
	Models         []SessionModelInfo `json:"models"`
}

// SessionModeInfo is one available mode/profile option for a session.
type SessionModeInfo struct {
	ModeID string `json:"modeId"`
	Name   string `json:"name"`
}

// SessionModesInfo contains mode selection state for a session.
type SessionModesInfo struct {
	CurrentModeID string            `json:"currentModeId"`
	Modes         []SessionModeInfo `json:"modes"`
}

// SessionListPage is a page of remote sessions queried from an integrator.
type SessionListPage struct {
	Sessions    []SessionListItem `json:"sessions"`
	NextCursor  string            `json:"nextCursor,omitempty"`
	Unsupported bool              `json:"unsupported,omitempty"`
}

// ResumeHistoricalResult reports the outcome of reopening an old session.
type ResumeHistoricalResult struct {
	ConnectionID string `json:"connectionId"`
	SessionID    string `json:"sessionId"`
	Resumed      bool   `json:"resumed"`
	Reason       string `json:"reason,omitempty"`
}

// EmbeddedTerminalInfo is metadata for one UI terminal tab.
type EmbeddedTerminalInfo struct {
	ID    string `json:"id"`
	CWD   string `json:"cwd"`
	Shell string `json:"shell"`
}

// ---------------------------------------------------------------------------
// App – the main Wails-bound struct
// ---------------------------------------------------------------------------

// App is the primary backend struct whose exported methods are exposed to the
// frontend as TypeScript bindings. It orchestrates agent connections, session
// management, file system and terminal providers, and pushes real-time
// updates to the frontend via Wails runtime events.
type App struct {
	ctx context.Context

	manager  *agent.Manager
	config   *agent.Config
	fs       *bfs.Provider
	terminal *terminal.Provider
	uiTerm   *uixterm.Manager
	sessions session.Store

	// sessionModels stores model options returned by session/new per session.
	sessionModels   map[string]SessionModelsInfo
	sessionModelsMu sync.RWMutex

	// sessionModes stores mode/profile options returned by session/new per session.
	sessionModes   map[string]SessionModesInfo
	sessionModesMu sync.RWMutex

	// sessionAccessModes stores execution access policy options per session
	// (currently used by codex compatibility sessions).
	sessionAccessModes   map[string]SessionModesInfo
	sessionAccessModesMu sync.RWMutex

	// pendingPermissions stores channels keyed by requestID.
	// pendingPermissionOrder stores request IDs FIFO by session+toolCall.
	pendingPermissions     map[string]chan string
	pendingPermissionOrder map[string][]string
	pendingPermissionsMu   sync.Mutex

	// pendingQuestions stores channels keyed by requestID for
	// item/tool/requestUserInput interactions.
	pendingQuestions   map[string]chan acp.ToolRequestUserInputResponse
	pendingQuestionsMu sync.Mutex

	// activePrompts tracks running prompt goroutines so CancelPrompt can
	// both cancel the context and send the ACP cancel notification.
	activePrompts   map[string]context.CancelFunc
	activePromptsMu sync.Mutex

	// streamMessages aggregates streaming chunks so each turn is stored as a
	// single final agent message.
	streamMessages   map[string]*streamMessage
	streamMessagesMu sync.Mutex

	configPath string
}

type streamMessage struct {
	MessageID   string
	ContentType string
	Content     strings.Builder
	StartedAt   time.Time
}
