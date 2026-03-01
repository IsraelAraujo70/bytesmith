package acp

import (
	"encoding/json"
	"fmt"
)

// JSONRPCMessage represents a JSON-RPC 2.0 message. It can be a request,
// response, or notification depending on which fields are populated.
//
// A request has Method and optionally Params, plus an ID.
// A notification has Method and optionally Params, but no ID.
// A response has ID and either Result or Error.
type JSONRPCMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

// IsRequest returns true if the message is a request (has method and ID).
func (m *JSONRPCMessage) IsRequest() bool {
	return m.Method != "" && m.ID != nil
}

// IsNotification returns true if the message is a notification (has method but no ID).
func (m *JSONRPCMessage) IsNotification() bool {
	return m.Method != "" && m.ID == nil
}

// IsResponse returns true if the message is a response (has ID but no method).
func (m *JSONRPCMessage) IsResponse() bool {
	return m.Method == "" && m.ID != nil
}

// IDAsInt64 parses the message ID as an int64. Returns 0 if the ID is nil
// or cannot be parsed as a number.
func (m *JSONRPCMessage) IDAsInt64() int64 {
	if m.ID == nil {
		return 0
	}
	var id int64
	if err := json.Unmarshal(*m.ID, &id); err != nil {
		return 0
	}
	return id
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

// Standard JSON-RPC 2.0 error codes.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)
