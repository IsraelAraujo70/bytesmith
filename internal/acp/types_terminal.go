package acp

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
