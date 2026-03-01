package acp

// SessionNewParams requests the agent to create a new session.
type SessionNewParams struct {
	CWD        string      `json:"cwd"`
	MCPServers []MCPServer `json:"mcpServers"`
}

// SessionNewResult is the agent's response to a session/new request.
type SessionNewResult struct {
	SessionID string              `json:"sessionId"`
	Models    *SessionModelsState `json:"models,omitempty"`
	Modes     *SessionModesState  `json:"modes,omitempty"`
	Meta      map[string]any      `json:"_meta,omitempty"`
}

// SessionModelsState represents model information returned by some agents
// (for example OpenCode) during session setup.
type SessionModelsState struct {
	CurrentModelID  string         `json:"currentModelId"`
	AvailableModels []SessionModel `json:"availableModels,omitempty"`
}

// SessionModel is one selectable model option.
type SessionModel struct {
	ModelID string `json:"modelId"`
	Name    string `json:"name"`
}

// SessionModesState represents optional mode metadata returned on session/new.
type SessionModesState struct {
	CurrentModeID  string        `json:"currentModeId"`
	AvailableModes []SessionMode `json:"availableModes,omitempty"`
}

// SessionMode is one available mode option.
type SessionMode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// SessionLoadParams requests the agent to reload an existing session.
type SessionLoadParams struct {
	SessionID  string      `json:"sessionId"`
	CWD        string      `json:"cwd"`
	MCPServers []MCPServer `json:"mcpServers"`
}

// SessionResumeParams requests the agent to resume an existing session.
type SessionResumeParams struct {
	SessionID  string      `json:"sessionId"`
	CWD        string      `json:"cwd"`
	MCPServers []MCPServer `json:"mcpServers"`
}

// SessionResumeResult contains metadata returned by a resume operation.
type SessionResumeResult struct {
	SessionID string              `json:"sessionId"`
	Models    *SessionModelsState `json:"models,omitempty"`
	Modes     *SessionModesState  `json:"modes,omitempty"`
	Meta      map[string]any      `json:"_meta,omitempty"`
}

// SessionListParams requests a paginated list of sessions.
type SessionListParams struct {
	CWD    string `json:"cwd,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// SessionInfo is one entry in a remote session listing.
type SessionInfo struct {
	SessionID string         `json:"sessionId"`
	CWD       string         `json:"cwd,omitempty"`
	Title     string         `json:"title,omitempty"`
	UpdatedAt string         `json:"updatedAt,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
}

// SessionListResult is the response payload for session/list.
type SessionListResult struct {
	Sessions   []SessionInfo  `json:"sessions"`
	NextCursor string         `json:"nextCursor,omitempty"`
	Meta       map[string]any `json:"meta,omitempty"`
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
