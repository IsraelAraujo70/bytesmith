package acp

// FSReadTextFileParams requests the client to read a text file on disk.
type FSReadTextFileParams struct {
	SessionID string `json:"sessionId"`
	Path      string `json:"path"`
	Line      int    `json:"line,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

// FSReadTextFileResult is the client's response containing file content.
type FSReadTextFileResult struct {
	Content string `json:"content"`
}

// FSWriteTextFileParams requests the client to write content to a text file.
type FSWriteTextFileParams struct {
	SessionID string `json:"sessionId"`
	Path      string `json:"path"`
	Content   string `json:"content"`
}
