package acp

import "encoding/json"

// ExecCommandApprovalParams is the legacy codex app-server approval request for
// shell/unified_exec actions.
type ExecCommandApprovalParams struct {
	ConversationID string          `json:"conversationId"`
	CallID         string          `json:"callId"`
	ApprovalID     *string         `json:"approvalId"`
	Command        []string        `json:"command"`
	CWD            string          `json:"cwd"`
	Reason         *string         `json:"reason"`
	ParsedCmd      []ParsedCommand `json:"parsedCmd"`
}

// ParsedCommand is a best-effort parsed action for command display.
type ParsedCommand struct {
	Type string `json:"type"`
	Cmd  string `json:"cmd,omitempty"`
}

// ExecCommandApprovalResponse is returned to legacy codex app-server approval
// requests.
type ExecCommandApprovalResponse struct {
	Decision string `json:"decision"`
}

// FileChangePreview describes one patch target in applyPatchApproval.
type FileChangePreview struct {
	Type        string  `json:"type"`
	UnifiedDiff string  `json:"unified_diff,omitempty"`
	MovePath    *string `json:"move_path,omitempty"`
	// Alternate key names observed in some protocol variants.
	UnifiedDiffAlt string  `json:"unifiedDiff,omitempty"`
	MovePathAlt    *string `json:"movePath,omitempty"`
}

// ApplyPatchApprovalParams is the legacy codex app-server approval request for
// patch application.
type ApplyPatchApprovalParams struct {
	ConversationID string                       `json:"conversationId"`
	CallID         string                       `json:"callId"`
	FileChanges    map[string]FileChangePreview `json:"fileChanges"`
	Reason         *string                      `json:"reason"`
	GrantRoot      *string                      `json:"grantRoot"`
}

// ApplyPatchApprovalResponse is returned to legacy codex patch approval
// requests.
type ApplyPatchApprovalResponse struct {
	Decision string `json:"decision"`
}

// CommandAction is a best-effort parsed action for v2 command approvals.
type CommandAction struct {
	Type string `json:"type"`
	Cmd  string `json:"cmd,omitempty"`
}

// CommandExecutionRequestApprovalParams is the v2 codex request for command
// execution approval (method item/commandExecution/requestApproval).
type CommandExecutionRequestApprovalParams struct {
	ThreadID                    string            `json:"threadId"`
	TurnID                      string            `json:"turnId"`
	ItemID                      string            `json:"itemId"`
	ApprovalID                  *string           `json:"approvalId,omitempty"`
	Reason                      *string           `json:"reason,omitempty"`
	Command                     *string           `json:"command,omitempty"`
	CWD                         *string           `json:"cwd,omitempty"`
	CommandActions              []CommandAction   `json:"commandActions,omitempty"`
	AvailableDecisions          []json.RawMessage `json:"availableDecisions,omitempty"`
	ProposedExecpolicyAmendment any               `json:"proposedExecpolicyAmendment,omitempty"`
}

// CommandExecutionRequestApprovalResponse is returned to v2 command approval
// requests.
type CommandExecutionRequestApprovalResponse struct {
	Decision string `json:"decision"`
}

// FileChangeRequestApprovalParams is the v2 codex request for file-change
// approval (method item/fileChange/requestApproval).
type FileChangeRequestApprovalParams struct {
	ThreadID  string  `json:"threadId"`
	TurnID    string  `json:"turnId"`
	ItemID    string  `json:"itemId"`
	Reason    *string `json:"reason,omitempty"`
	GrantRoot *string `json:"grantRoot,omitempty"`
}

// FileChangeRequestApprovalResponse is returned to v2 file-change approval
// requests.
type FileChangeRequestApprovalResponse struct {
	Decision string `json:"decision"`
}

// ToolRequestUserInputOption is one selectable option in a codex user-input
// request.
type ToolRequestUserInputOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ToolRequestUserInputQuestion is one question in a codex user-input request.
type ToolRequestUserInputQuestion struct {
	ID       string                       `json:"id"`
	Header   string                       `json:"header"`
	Question string                       `json:"question"`
	IsOther  bool                         `json:"isOther"`
	IsSecret bool                         `json:"isSecret"`
	Options  []ToolRequestUserInputOption `json:"options"`
}

// ToolRequestUserInputParams is the v2 codex request for explicit user input
// (method item/tool/requestUserInput).
type ToolRequestUserInputParams struct {
	ThreadID  string                         `json:"threadId"`
	TurnID    string                         `json:"turnId"`
	ItemID    string                         `json:"itemId"`
	Questions []ToolRequestUserInputQuestion `json:"questions"`
}

// ToolRequestUserInputAnswer is the answer payload for one question id.
type ToolRequestUserInputAnswer struct {
	Answers []string `json:"answers"`
}

// ToolRequestUserInputResponse is returned to v2 user-input requests.
type ToolRequestUserInputResponse struct {
	Answers map[string]ToolRequestUserInputAnswer `json:"answers"`
}
