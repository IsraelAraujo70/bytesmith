package acp

// ACP method names (JSON-RPC method strings).
const (
	MethodInitialize        = "initialize"
	MethodSessionNew        = "session/new"
	MethodSessionLoad       = "session/load"
	MethodSessionResume     = "session/resume"
	MethodSessionList       = "session/list"
	MethodSessionPrompt     = "session/prompt"
	MethodSessionCancel     = "session/cancel"
	MethodSessionSetMode    = "session/set_mode"
	MethodSessionSetModeOld = "session/setMode"
	MethodSessionSetModel   = "session/set_model"
	MethodSessionSetConfig  = "session/set_config_option"
	MethodSessionUpdate     = "session/update"
	MethodRequestPermission = "requestPermission"
	// Legacy codex approval request methods.
	MethodExecCommandApproval = "execCommandApproval"
	MethodApplyPatchApproval  = "applyPatchApproval"
	// Codex v2 request methods.
	MethodItemCommandExecutionRequestApproval = "item/commandExecution/requestApproval"
	MethodItemFileChangeRequestApproval       = "item/fileChange/requestApproval"
	MethodItemToolRequestUserInput            = "item/tool/requestUserInput"
	MethodFSReadTextFile                      = "fs/readTextFile"
	MethodFSWriteTextFile                     = "fs/writeTextFile"
	MethodTerminalCreate                      = "terminal/create"
	MethodTerminalOutput                      = "terminal/output"
	MethodTerminalWait                        = "terminal/wait"
	MethodTerminalKill                        = "terminal/kill"
	MethodTerminalRelease                     = "terminal/release"
)
