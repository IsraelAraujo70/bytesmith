package acp

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
