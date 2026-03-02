package backend

import (
	"strings"
	"time"

	"bytesmith/internal/acp"
	"bytesmith/internal/agent"
	"bytesmith/internal/session"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------------------
// Internal: connection wiring
// ---------------------------------------------------------------------------

// wireConnection registers all runtime callbacks on a newly created
// connection so that session updates, permission requests, FS operations,
// and terminal operations are correctly routed.
func (a *App) wireConnection(conn *agent.Connection) {
	connID := conn.ID

	// --- Session update notifications ---
	conn.Client.OnSessionUpdate(func(params acp.SessionUpdateParams) {
		a.handleSessionUpdate(connID, params)
	})

	// --- Permission requests ---
	conn.Client.OnRequestPermission(func(params acp.RequestPermissionParams) acp.RequestPermissionResult {
		return a.handlePermissionRequest(connID, params)
	})

	// --- FS handlers ---
	conn.Client.OnFSReadTextFile(func(params acp.FSReadTextFileParams) (*acp.FSReadTextFileResult, error) {
		return a.fs.HandleReadTextFile(params)
	})
	conn.Client.OnFSWriteTextFile(func(params acp.FSWriteTextFileParams) error {
		return a.fs.HandleWriteTextFile(params)
	})

	// --- Terminal handlers ---
	conn.Client.OnTerminalCreate(func(params acp.TerminalCreateParams) (*acp.TerminalCreateResult, error) {
		return a.terminal.HandleCreate(params)
	})
	conn.Client.OnTerminalOutput(func(params acp.TerminalOutputParams) (*acp.TerminalOutputResult, error) {
		return a.terminal.HandleOutput(params)
	})
	conn.Client.OnTerminalWait(func(params acp.TerminalWaitParams) (*acp.TerminalWaitResult, error) {
		return a.terminal.HandleWaitForExit(params)
	})
	conn.Client.OnTerminalKill(func(params acp.TerminalKillParams) error {
		return a.terminal.HandleKill(params)
	})
	conn.Client.OnTerminalRelease(func(params acp.TerminalReleaseParams) error {
		return a.terminal.HandleRelease(params)
	})

	// --- Forward stderr to frontend ---
	go func() {
		for line := range conn.Client.StderrCh() {
			wailsRuntime.EventsEmit(a.ctx, "agent:stderr", map[string]string{
				"connectionId": connID,
				"line":         line,
			})
		}
	}()
}

// ---------------------------------------------------------------------------
// Internal: session update routing
// ---------------------------------------------------------------------------

// handleSessionUpdate dispatches an incoming session update notification
// to the appropriate Wails event and records data in the session store.
func (a *App) handleSessionUpdate(connectionID string, params acp.SessionUpdateParams) {
	update := params.Update
	sid := params.SessionID

	switch update.Type {
	case acp.UpdateAgentMessageChunk:
		if update.MessageContent != nil {
			chunkType := update.MessageContent.Type
			if strings.TrimSpace(chunkType) == "" {
				chunkType = "text"
			}
			a.appendStreamChunk(connectionID, sid, update.MessageContent.Text, chunkType)
		}

	case acp.UpdateAgentThoughtChunk:
		if update.MessageContent != nil {
			a.appendStreamChunk(connectionID, sid, update.MessageContent.Text, "thought")
		}

	case acp.UpdateUserMessageChunk:
		if update.MessageContent != nil {
			a.sessions.AddMessage(sid, session.Message{
				ID:        uuid.NewString(),
				Role:      "user",
				Content:   update.MessageContent.Text,
				Timestamp: time.Now(),
			})
		}

	case acp.UpdateToolCall:
		parts := normalizeToolCallParts(update.ToolContent)
		content := formatToolCallContent(parts, update)
		diffSummary := summarizeDiffParts(parts)
		record := session.ToolCallRecord{
			ID:          update.ToolCallID,
			Title:       update.Title,
			Kind:        update.Kind,
			Status:      update.Status,
			Content:     content,
			Parts:       parts,
			DiffSummary: diffSummary,
		}
		a.sessions.AddToolCall(sid, record)
		info := toToolCallInfo(record)
		wailsRuntime.EventsEmit(a.ctx, "agent:toolcall", map[string]interface{}{
			"connectionId": connectionID,
			"sessionId":    sid,
			"toolCallId":   update.ToolCallID,
			"title":        update.Title,
			"kind":         update.Kind,
			"status":       update.Status,
			"content":      content,
			"parts":        info.Parts,
			"diffSummary":  info.DiffSummary,
			"isUpdate":     false,
		})

	case acp.UpdateToolCallUpdate:
		parts := normalizeToolCallParts(update.ToolContent)
		content := formatToolCallContent(parts, update)
		diffSummary := summarizeDiffParts(parts)
		a.sessions.UpdateToolCall(sid, update.ToolCallID, update.Status, content, parts, diffSummary)
		info := toToolCallInfo(session.ToolCallRecord{
			ID:          update.ToolCallID,
			Title:       update.Title,
			Kind:        update.Kind,
			Status:      update.Status,
			Content:     content,
			Parts:       parts,
			DiffSummary: diffSummary,
		})
		wailsRuntime.EventsEmit(a.ctx, "agent:toolcall", map[string]interface{}{
			"connectionId": connectionID,
			"sessionId":    sid,
			"toolCallId":   update.ToolCallID,
			"title":        update.Title,
			"kind":         update.Kind,
			"status":       update.Status,
			"content":      content,
			"parts":        info.Parts,
			"diffSummary":  info.DiffSummary,
			"isUpdate":     true,
		})

	case acp.UpdatePlan:
		entries := make([]map[string]string, 0, len(update.Entries))
		for _, e := range update.Entries {
			entries = append(entries, map[string]string{
				"content":  e.Content,
				"priority": e.Priority,
				"status":   e.Status,
			})
		}
		wailsRuntime.EventsEmit(a.ctx, "agent:plan", map[string]interface{}{
			"connectionId": connectionID,
			"sessionId":    sid,
			"entries":      entries,
		})

	case acp.UpdateAvailableCommands:
		cmds := make([]map[string]string, 0, len(update.AvailableCommands))
		for _, c := range update.AvailableCommands {
			entry := map[string]string{
				"name":        c.Name,
				"description": c.Description,
			}
			if c.Input != nil {
				entry["inputHint"] = c.Input.Hint
			}
			cmds = append(cmds, entry)
		}
		wailsRuntime.EventsEmit(a.ctx, "agent:commands", map[string]interface{}{
			"connectionId": connectionID,
			"sessionId":    sid,
			"commands":     cmds,
		})
	}
}
