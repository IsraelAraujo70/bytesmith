package integrator

// Capabilities describes what an integrator supports over ACP.
type Capabilities struct {
	ListSessions    bool
	LoadSession     bool
	ResumeSession   bool
	SetMode         bool
	SetModel        bool
	SetConfigOption bool
}

// AgentServer is a lightweight adapter descriptor for an ACP integrator.
type AgentServer interface {
	ID() string
	DisplayName() string
	Capabilities() Capabilities
}

type adapter struct {
	id           string
	displayName  string
	capabilities Capabilities
}

func (a adapter) ID() string {
	return a.id
}

func (a adapter) DisplayName() string {
	return a.displayName
}

func (a adapter) Capabilities() Capabilities {
	return a.capabilities
}

var (
	openCode = adapter{
		id:          "opencode",
		displayName: "OpenCode",
		capabilities: Capabilities{
			ListSessions:    true,
			LoadSession:     true,
			ResumeSession:   true,
			SetMode:         true,
			SetModel:        true,
			SetConfigOption: true,
		},
	}
	codex = adapter{
		id:          "codex",
		displayName: "Codex App Server",
		capabilities: Capabilities{
			ListSessions:    false,
			LoadSession:     false,
			ResumeSession:   false,
			SetMode:         false,
			SetModel:        true,
			SetConfigOption: false,
		},
	}
	defaultACP = adapter{
		id:          "acp-generic",
		displayName: "Generic ACP",
		capabilities: Capabilities{
			ListSessions:    true,
			LoadSession:     true,
			ResumeSession:   true,
			SetMode:         true,
			SetModel:        true,
			SetConfigOption: true,
		},
	}
)

// ForAgent resolves the integrator descriptor for a configured agent name.
func ForAgent(agentName string) AgentServer {
	switch agentName {
	case "opencode":
		return openCode
	case "codex-app-server":
		return codex
	default:
		return defaultACP
	}
}
