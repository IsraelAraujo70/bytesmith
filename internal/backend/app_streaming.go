package backend

import (
	"strings"
	"time"

	"bytesmith/internal/session"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) appendStreamChunk(connectionID, sessionID, text, contentType string) {
	chunkType := normalizeMessageType(contentType)

	var previous *streamMessage

	a.streamMessagesMu.Lock()
	stream := a.streamMessages[sessionID]
	if stream == nil {
		stream = &streamMessage{
			MessageID:   uuid.NewString(),
			ContentType: chunkType,
			StartedAt:   time.Now(),
		}
		a.streamMessages[sessionID] = stream
	} else if stream.ContentType != chunkType {
		previous = stream
		stream = &streamMessage{
			MessageID:   uuid.NewString(),
			ContentType: chunkType,
			StartedAt:   time.Now(),
		}
		a.streamMessages[sessionID] = stream
	}
	stream.Content.WriteString(text)
	messageID := stream.MessageID
	a.streamMessagesMu.Unlock()

	if previous != nil {
		a.flushStreamMessage(connectionID, sessionID, previous)
	}

	wailsRuntime.EventsEmit(a.ctx, "agent:message", map[string]interface{}{
		"connectionId": connectionID,
		"sessionId":    sessionID,
		"messageId":    messageID,
		"text":         text,
		"type":         chunkType,
		"isFinal":      false,
	})
}

func (a *App) finalizeStreamMessage(connectionID, sessionID string) {
	a.streamMessagesMu.Lock()
	stream := a.streamMessages[sessionID]
	delete(a.streamMessages, sessionID)
	a.streamMessagesMu.Unlock()

	if stream == nil {
		return
	}

	a.flushStreamMessage(connectionID, sessionID, stream)
}

func (a *App) flushStreamMessage(connectionID, sessionID string, stream *streamMessage) {
	if stream == nil {
		return
	}

	content := stream.Content.String()
	if content == "" {
		return
	}
	if strings.TrimSpace(content) == "" {
		return
	}

	a.sessions.AddMessage(sessionID, session.Message{
		ID:        stream.MessageID,
		Role:      "agent",
		Content:   content,
		Timestamp: stream.StartedAt,
	})

	wailsRuntime.EventsEmit(a.ctx, "agent:message", map[string]interface{}{
		"connectionId": connectionID,
		"sessionId":    sessionID,
		"messageId":    stream.MessageID,
		"text":         "",
		"type":         normalizeMessageType(stream.ContentType),
		"isFinal":      true,
		"content":      content,
	})
}
