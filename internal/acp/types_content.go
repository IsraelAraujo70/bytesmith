package acp

// ContentBlock represents a piece of content in a prompt or agent response.
// The Type field determines which other fields are relevant.
type ContentBlock struct {
	// Type of content: text, image, audio, resource, resource_link.
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	Resource *Resource `json:"resource,omitempty"`
	// Image/audio fields.
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	URI      string `json:"uri,omitempty"`
}

// Resource represents an embedded or linked resource.
type Resource struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}
