package backend

import (
	"bytesmith/internal/acp"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------------------
// Permission handling
// ---------------------------------------------------------------------------

// RespondPermission is called by the UI when the user clicks allow/deny on a
// permission dialog. It unblocks the ACP requestPermission handler which is
// waiting for the user's decision.
func (a *App) RespondPermission(sessionID, toolCallID, optionID string) {
	key := sessionPermissionKey(sessionID, toolCallID)

	a.pendingPermissionsMu.Lock()
	queue := a.pendingPermissionOrder[key]
	if len(queue) == 0 {
		a.pendingPermissionsMu.Unlock()
		return
	}

	requestID := queue[0]
	next := queue[1:]
	if len(next) == 0 {
		delete(a.pendingPermissionOrder, key)
	} else {
		a.pendingPermissionOrder[key] = next
	}

	ch, ok := a.pendingPermissions[requestID]
	a.pendingPermissionsMu.Unlock()

	if ok {
		ch <- optionID
	}
}

// handlePermissionRequest is called synchronously by the ACP client when the
// agent asks for user permission. It emits an event to the UI and blocks
// until RespondPermission is called.
func (a *App) handlePermissionRequest(connectionID string, params acp.RequestPermissionParams) acp.RequestPermissionResult {
	ch := make(chan string, 1)
	requestID := uuid.NewString()
	orderKey := sessionPermissionKey(params.SessionID, params.ToolCall.ToolCallID)

	a.pendingPermissionsMu.Lock()
	a.pendingPermissions[requestID] = ch
	a.pendingPermissionOrder[orderKey] = append(a.pendingPermissionOrder[orderKey], requestID)
	a.pendingPermissionsMu.Unlock()

	// Build options for the frontend.
	options := make([]PermissionOptionInfo, 0, len(params.Options))
	for _, opt := range params.Options {
		options = append(options, PermissionOptionInfo{
			OptionID: opt.OptionID,
			Name:     opt.Name,
			Kind:     opt.Kind,
		})
	}

	wailsRuntime.EventsEmit(a.ctx, "agent:permission", PermissionRequestInfo{
		RequestID:    requestID,
		ConnectionID: connectionID,
		SessionID:    params.SessionID,
		ToolCallID:   params.ToolCall.ToolCallID,
		Title:        params.ToolCall.Title,
		Kind:         params.ToolCall.Kind,
		Options:      options,
	})

	// Block until the UI responds.
	optionID, ok := <-ch

	// Clean up.
	a.pendingPermissionsMu.Lock()
	delete(a.pendingPermissions, requestID)
	a.pendingPermissionsMu.Unlock()

	if !ok || optionID == "" {
		return acp.RequestPermissionResult{
			Outcome: acp.PermissionOutcome{
				Outcome: "cancelled",
			},
		}
	}

	return acp.RequestPermissionResult{
		Outcome: acp.PermissionOutcome{
			Outcome:  "selected",
			OptionID: optionID,
		},
	}
}

func sessionPermissionKey(sessionID, toolCallID string) string {
	return sessionID + "::" + toolCallID
}
