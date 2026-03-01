package acp

import (
	"encoding/json"
	"testing"
)

func TestSessionUpdateUnmarshalMessageChunk(t *testing.T) {
	payload := []byte(`{
		"sessionUpdate":"agent_message_chunk",
		"content":{"type":"text","text":"hello"}
	}`)

	var got SessionUpdate
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Type != UpdateAgentMessageChunk {
		t.Fatalf("type = %q, want %q", got.Type, UpdateAgentMessageChunk)
	}
	if got.MessageContent == nil || got.MessageContent.Text != "hello" {
		t.Fatalf("message content = %#v, want text hello", got.MessageContent)
	}
	if len(got.ToolContent) != 0 {
		t.Fatalf("tool content len = %d, want 0", len(got.ToolContent))
	}
}

func TestSessionUpdateUnmarshalToolCallArray(t *testing.T) {
	payload := []byte(`{
		"sessionUpdate":"tool_call",
		"toolCallId":"tc1",
		"content":[{"type":"diff","path":"app.go","oldText":"a","newText":"b"}]
	}`)

	var got SessionUpdate
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Type != UpdateToolCall {
		t.Fatalf("type = %q, want %q", got.Type, UpdateToolCall)
	}
	if len(got.ToolContent) != 1 {
		t.Fatalf("tool content len = %d, want 1", len(got.ToolContent))
	}
	if got.ToolContent[0].Path != "app.go" {
		t.Fatalf("tool content path = %q, want app.go", got.ToolContent[0].Path)
	}
	if got.MessageContent != nil {
		t.Fatalf("message content = %#v, want nil", got.MessageContent)
	}
}

func TestSessionUpdateUnmarshalUnknownFallback(t *testing.T) {
	arrayPayload := []byte(`{
		"sessionUpdate":"unknown_update",
		"content":[{"type":"terminal","terminalId":"t1"}]
	}`)
	var fromArray SessionUpdate
	if err := json.Unmarshal(arrayPayload, &fromArray); err != nil {
		t.Fatalf("array unmarshal: %v", err)
	}
	if len(fromArray.ToolContent) != 1 || fromArray.ToolContent[0].TerminalID != "t1" {
		t.Fatalf("array fallback tool content = %#v", fromArray.ToolContent)
	}

	objectPayload := []byte(`{
		"sessionUpdate":"unknown_update",
		"content":{"type":"text","text":"fallback"}
	}`)
	var fromObject SessionUpdate
	if err := json.Unmarshal(objectPayload, &fromObject); err != nil {
		t.Fatalf("object unmarshal: %v", err)
	}
	if fromObject.MessageContent == nil || fromObject.MessageContent.Text != "fallback" {
		t.Fatalf("object fallback message content = %#v", fromObject.MessageContent)
	}
}

func TestSessionUpdateMarshalShapes(t *testing.T) {
	messageUpdate := SessionUpdate{
		Type: UpdateAgentThoughtChunk,
		MessageContent: &ContentBlock{
			Type: "text",
			Text: "thinking",
		},
	}
	messageRaw, err := json.Marshal(messageUpdate)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	var messageDecoded struct {
		SessionUpdate string                 `json:"sessionUpdate"`
		Content       map[string]interface{} `json:"content"`
	}
	if err := json.Unmarshal(messageRaw, &messageDecoded); err != nil {
		t.Fatalf("unmarshal message json: %v", err)
	}
	if messageDecoded.SessionUpdate != UpdateAgentThoughtChunk {
		t.Fatalf("message sessionUpdate = %q", messageDecoded.SessionUpdate)
	}
	if _, ok := messageDecoded.Content["text"]; !ok {
		t.Fatalf("message content does not look like an object: %#v", messageDecoded.Content)
	}

	toolUpdate := SessionUpdate{
		Type: UpdateToolCall,
		ToolContent: []ToolCallContent{{
			Type:       "terminal",
			TerminalID: "term-1",
		}},
	}
	toolRaw, err := json.Marshal(toolUpdate)
	if err != nil {
		t.Fatalf("marshal tool: %v", err)
	}

	var toolDecoded struct {
		SessionUpdate string            `json:"sessionUpdate"`
		Content       []ToolCallContent `json:"content"`
	}
	if err := json.Unmarshal(toolRaw, &toolDecoded); err != nil {
		t.Fatalf("unmarshal tool json: %v", err)
	}
	if toolDecoded.SessionUpdate != UpdateToolCall {
		t.Fatalf("tool sessionUpdate = %q", toolDecoded.SessionUpdate)
	}
	if len(toolDecoded.Content) != 1 || toolDecoded.Content[0].TerminalID != "term-1" {
		t.Fatalf("tool content = %#v", toolDecoded.Content)
	}
}
