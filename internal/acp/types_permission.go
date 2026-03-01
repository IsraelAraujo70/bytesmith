package acp

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
