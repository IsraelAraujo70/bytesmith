package acp

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
