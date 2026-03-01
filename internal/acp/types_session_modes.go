package acp

// SessionSetModeParams requests the agent to switch its operating mode.
type SessionSetModeParams struct {
	SessionID string `json:"sessionId"`
	ModeID    string `json:"modeId"`
}

// SessionSetModeLegacyParams is kept for older agents that still use
// session/setMode with the "mode" field.
type SessionSetModeLegacyParams struct {
	SessionID string `json:"sessionId"`
	Mode      string `json:"mode"`
}

// SessionSetConfigOptionParams requests the agent to set a session config
// option (for example model selection).
type SessionSetConfigOptionParams struct {
	SessionID string `json:"sessionId"`
	ConfigID  string `json:"configId"`
	Value     string `json:"value"`
}

// SessionSetModelParams requests the agent to switch the active model.
// This method is currently an OpenCode ACP extension.
type SessionSetModelParams struct {
	SessionID string `json:"sessionId"`
	ModelID   string `json:"modelId"`
}
