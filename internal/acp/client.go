package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultRequestTimeout is the default timeout for JSON-RPC requests.
const DefaultRequestTimeout = 30 * time.Second

// Client is the main ACP protocol client. It orchestrates communication with
// an AI coding agent over a StdioTransport by:
//
//  1. Managing the transport lifecycle.
//  2. Dispatching outgoing JSON-RPC requests and tracking responses via ID.
//  3. Routing incoming notifications to registered handlers.
//  4. Routing incoming requests (from agent) to registered method handlers and
//     sending back responses.
type Client struct {
	transport *StdioTransport

	nextID atomic.Int64

	// pending tracks in-flight requests by their numeric ID.
	pending   map[int64]chan json.RawMessage
	pendingMu sync.Mutex

	// Request timeout for outgoing calls.
	RequestTimeout time.Duration

	// --- notification handlers ---

	onSessionUpdate func(SessionUpdateParams)
	notifMu         sync.RWMutex

	// --- agent-to-client request handlers ---

	onRequestPermission func(RequestPermissionParams) RequestPermissionResult
	onFSReadTextFile    func(FSReadTextFileParams) (*FSReadTextFileResult, error)
	onFSWriteTextFile   func(FSWriteTextFileParams) error
	onTerminalCreate    func(TerminalCreateParams) (*TerminalCreateResult, error)
	onTerminalOutput    func(TerminalOutputParams) (*TerminalOutputResult, error)
	onTerminalWait      func(TerminalWaitParams) (*TerminalWaitResult, error)
	onTerminalKill      func(TerminalKillParams) error
	onTerminalRelease   func(TerminalReleaseParams) error
	handlerMu           sync.RWMutex
}

// NewClient creates an ACP client bound to the given transport. The transport
// must not be started yet; call Initialize to perform the handshake which
// also starts the transport if it hasn't been started.
func NewClient(transport *StdioTransport) *Client {
	c := &Client{
		transport:      transport,
		pending:        make(map[int64]chan json.RawMessage),
		RequestTimeout: DefaultRequestTimeout,
	}
	transport.SetHandler(c.dispatch)
	return c
}

// ---------------------------------------------------------------------------
// Protocol methods (client -> agent)
// ---------------------------------------------------------------------------

// Initialize starts the transport (if not running), sends the ACP initialize
// handshake, and returns the agent's capabilities.
func (c *Client) Initialize(ctx context.Context) (*InitializeResult, error) {
	if !c.transport.IsRunning() {
		if err := c.transport.Start(); err != nil {
			return nil, err
		}
	}

	params := InitializeParams{
		ProtocolVersion: 1,
		ClientCapabilities: ClientCapabilities{
			FS: &FSCapabilities{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
			Terminal: true,
		},
		ClientInfo: ImplementationInfo{
			Name:    "bytesmith",
			Title:   "ByteSmith",
			Version: "0.1.0",
		},
	}

	raw, err := c.call(ctx, MethodInitialize, params)
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	var result InitializeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("initialize: unmarshal result: %w", err)
	}
	return &result, nil
}

// NewSession asks the agent to create a new session and returns the session ID.
func (c *Client) NewSession(ctx context.Context, cwd string, mcpServers []MCPServer) (string, error) {
	params := SessionNewParams{
		CWD:        cwd,
		MCPServers: mcpServers,
	}

	raw, err := c.call(ctx, MethodSessionNew, params)
	if err != nil {
		return "", fmt.Errorf("session/new: %w", err)
	}

	var result SessionNewResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("session/new: unmarshal result: %w", err)
	}
	return result.SessionID, nil
}

// LoadSession asks the agent to load an existing session.
func (c *Client) LoadSession(ctx context.Context, sessionID, cwd string, mcpServers []MCPServer) error {
	params := SessionLoadParams{
		SessionID:  sessionID,
		CWD:        cwd,
		MCPServers: mcpServers,
	}

	_, err := c.call(ctx, MethodSessionLoad, params)
	if err != nil {
		return fmt.Errorf("session/load: %w", err)
	}
	return nil
}

// Prompt sends a user prompt to the agent session and blocks until the agent
// finishes processing. Session updates arrive via the OnSessionUpdate callback
// while this method is blocked.
func (c *Client) Prompt(ctx context.Context, sessionID string, prompt []ContentBlock) (*SessionPromptResult, error) {
	params := SessionPromptParams{
		SessionID: sessionID,
		Prompt:    prompt,
	}

	raw, err := c.call(ctx, MethodSessionPrompt, params)
	if err != nil {
		return nil, fmt.Errorf("session/prompt: %w", err)
	}

	var result SessionPromptResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("session/prompt: unmarshal result: %w", err)
	}
	return &result, nil
}

// Cancel requests cancellation of an in-progress prompt. This is a
// notification (fire-and-forget).
func (c *Client) Cancel(sessionID string) error {
	params := SessionCancelParams{
		SessionID: sessionID,
	}
	return c.notify(MethodSessionCancel, params)
}

// SetMode asks the agent to switch operating modes.
func (c *Client) SetMode(ctx context.Context, sessionID, mode string) error {
	params := SessionSetModeParams{
		SessionID: sessionID,
		Mode:      mode,
	}

	_, err := c.call(ctx, MethodSessionSetMode, params)
	if err != nil {
		return fmt.Errorf("session/setMode: %w", err)
	}
	return nil
}

// Close performs a clean shutdown: cancels pending requests, closes the
// transport, and waits for the subprocess to exit.
func (c *Client) Close() error {
	// Fail all pending requests.
	c.pendingMu.Lock()
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.pendingMu.Unlock()

	return c.transport.Close()
}

// Transport returns the underlying transport for direct access if needed.
func (c *Client) Transport() *StdioTransport {
	return c.transport
}

// ---------------------------------------------------------------------------
// Notification handlers (agent -> client notifications)
// ---------------------------------------------------------------------------

// OnSessionUpdate registers a handler for session/update notifications from
// the agent. Only one handler is supported; subsequent calls replace the
// previous handler.
func (c *Client) OnSessionUpdate(handler func(SessionUpdateParams)) {
	c.notifMu.Lock()
	c.onSessionUpdate = handler
	c.notifMu.Unlock()
}

// ---------------------------------------------------------------------------
// Agent-to-client request handlers
// ---------------------------------------------------------------------------

// OnRequestPermission registers a handler for requestPermission requests.
func (c *Client) OnRequestPermission(handler func(RequestPermissionParams) RequestPermissionResult) {
	c.handlerMu.Lock()
	c.onRequestPermission = handler
	c.handlerMu.Unlock()
}

// OnFSReadTextFile registers a handler for fs/readTextFile requests.
func (c *Client) OnFSReadTextFile(handler func(FSReadTextFileParams) (*FSReadTextFileResult, error)) {
	c.handlerMu.Lock()
	c.onFSReadTextFile = handler
	c.handlerMu.Unlock()
}

// OnFSWriteTextFile registers a handler for fs/writeTextFile requests.
func (c *Client) OnFSWriteTextFile(handler func(FSWriteTextFileParams) error) {
	c.handlerMu.Lock()
	c.onFSWriteTextFile = handler
	c.handlerMu.Unlock()
}

// OnTerminalCreate registers a handler for terminal/create requests.
func (c *Client) OnTerminalCreate(handler func(TerminalCreateParams) (*TerminalCreateResult, error)) {
	c.handlerMu.Lock()
	c.onTerminalCreate = handler
	c.handlerMu.Unlock()
}

// OnTerminalOutput registers a handler for terminal/output requests.
func (c *Client) OnTerminalOutput(handler func(TerminalOutputParams) (*TerminalOutputResult, error)) {
	c.handlerMu.Lock()
	c.onTerminalOutput = handler
	c.handlerMu.Unlock()
}

// OnTerminalWait registers a handler for terminal/wait requests.
func (c *Client) OnTerminalWait(handler func(TerminalWaitParams) (*TerminalWaitResult, error)) {
	c.handlerMu.Lock()
	c.onTerminalWait = handler
	c.handlerMu.Unlock()
}

// OnTerminalKill registers a handler for terminal/kill requests.
func (c *Client) OnTerminalKill(handler func(TerminalKillParams) error) {
	c.handlerMu.Lock()
	c.onTerminalKill = handler
	c.handlerMu.Unlock()
}

// OnTerminalRelease registers a handler for terminal/release requests.
func (c *Client) OnTerminalRelease(handler func(TerminalReleaseParams) error) {
	c.handlerMu.Lock()
	c.onTerminalRelease = handler
	c.handlerMu.Unlock()
}

// ---------------------------------------------------------------------------
// Internal: outgoing call / notify
// ---------------------------------------------------------------------------

// call sends a JSON-RPC request and blocks until a response is received or
// the context expires. Returns the raw result JSON on success.
func (c *Client) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal params: %w", err)
	}

	idJSON := json.RawMessage(fmt.Sprintf("%d", id))

	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &idJSON,
		Method:  method,
		Params:  paramsJSON,
	}

	// Register a channel for the response before sending.
	ch := make(chan json.RawMessage, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	if err := c.transport.Send(msg); err != nil {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, err
	}

	// Determine timeout from context or default.
	timeout := c.RequestTimeout
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case raw, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("request %d cancelled (client closing)", id)
		}
		// raw contains the full response message. We need to check for errors.
		var resp JSONRPCMessage
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil

	case <-timer.C:
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, fmt.Errorf("request %s (id=%d) timed out after %v", method, id, timeout)

	case <-ctx.Done():
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return nil, ctx.Err()
	}
}

// notify sends a JSON-RPC notification (no ID, no response expected).
func (c *Client) notify(method string, params any) error {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}

	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}

	return c.transport.Send(msg)
}

// ---------------------------------------------------------------------------
// Internal: incoming message dispatch
// ---------------------------------------------------------------------------

// dispatch is the handler registered with the transport. It routes each
// incoming JSON-RPC message to the appropriate handler.
func (c *Client) dispatch(msg JSONRPCMessage) {
	switch {
	case msg.IsResponse():
		c.handleResponse(msg)
	case msg.IsNotification():
		c.handleNotification(msg)
	case msg.IsRequest():
		c.handleRequest(msg)
	default:
		log.Printf("acp: received unrecognized message: %+v", msg)
	}
}

// handleResponse matches a response to a pending request by ID and delivers
// the raw message to the waiting goroutine.
func (c *Client) handleResponse(msg JSONRPCMessage) {
	id := msg.IDAsInt64()
	if id == 0 {
		log.Printf("acp: received response with non-numeric or zero ID")
		return
	}

	c.pendingMu.Lock()
	ch, ok := c.pending[id]
	if ok {
		delete(c.pending, id)
	}
	c.pendingMu.Unlock()

	if !ok {
		log.Printf("acp: received response for unknown request id=%d", id)
		return
	}

	// Re-marshal the full message so the caller can inspect error/result.
	raw, err := json.Marshal(msg)
	if err != nil {
		log.Printf("acp: failed to re-marshal response id=%d: %v", id, err)
		close(ch)
		return
	}
	ch <- raw
}

// handleNotification routes incoming notifications to registered handlers.
func (c *Client) handleNotification(msg JSONRPCMessage) {
	switch msg.Method {
	case MethodSessionUpdate:
		c.notifMu.RLock()
		h := c.onSessionUpdate
		c.notifMu.RUnlock()

		if h != nil {
			var params SessionUpdateParams
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				log.Printf("acp: failed to unmarshal session/update params: %v", err)
				return
			}
			h(params)
		}

	default:
		log.Printf("acp: unhandled notification: %s", msg.Method)
	}
}

// handleRequest routes incoming requests from the agent, calls the registered
// handler, and sends back a JSON-RPC response.
func (c *Client) handleRequest(msg JSONRPCMessage) {
	c.handlerMu.RLock()
	defer c.handlerMu.RUnlock()

	var result any
	var handlerErr error

	switch msg.Method {
	case MethodRequestPermission:
		if c.onRequestPermission != nil {
			var params RequestPermissionParams
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				c.sendError(msg.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
				return
			}
			res := c.onRequestPermission(params)
			result = res
		} else {
			c.sendError(msg.ID, ErrCodeMethodNotFound, "no handler for "+msg.Method)
			return
		}

	case MethodFSReadTextFile:
		if c.onFSReadTextFile != nil {
			var params FSReadTextFileParams
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				c.sendError(msg.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
				return
			}
			result, handlerErr = c.onFSReadTextFile(params)
		} else {
			c.sendError(msg.ID, ErrCodeMethodNotFound, "no handler for "+msg.Method)
			return
		}

	case MethodFSWriteTextFile:
		if c.onFSWriteTextFile != nil {
			var params FSWriteTextFileParams
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				c.sendError(msg.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
				return
			}
			handlerErr = c.onFSWriteTextFile(params)
			if handlerErr == nil {
				result = struct{}{}
			}
		} else {
			c.sendError(msg.ID, ErrCodeMethodNotFound, "no handler for "+msg.Method)
			return
		}

	case MethodTerminalCreate:
		if c.onTerminalCreate != nil {
			var params TerminalCreateParams
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				c.sendError(msg.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
				return
			}
			result, handlerErr = c.onTerminalCreate(params)
		} else {
			c.sendError(msg.ID, ErrCodeMethodNotFound, "no handler for "+msg.Method)
			return
		}

	case MethodTerminalOutput:
		if c.onTerminalOutput != nil {
			var params TerminalOutputParams
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				c.sendError(msg.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
				return
			}
			result, handlerErr = c.onTerminalOutput(params)
		} else {
			c.sendError(msg.ID, ErrCodeMethodNotFound, "no handler for "+msg.Method)
			return
		}

	case MethodTerminalWait:
		if c.onTerminalWait != nil {
			var params TerminalWaitParams
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				c.sendError(msg.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
				return
			}
			result, handlerErr = c.onTerminalWait(params)
		} else {
			c.sendError(msg.ID, ErrCodeMethodNotFound, "no handler for "+msg.Method)
			return
		}

	case MethodTerminalKill:
		if c.onTerminalKill != nil {
			var params TerminalKillParams
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				c.sendError(msg.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
				return
			}
			handlerErr = c.onTerminalKill(params)
			if handlerErr == nil {
				result = struct{}{}
			}
		} else {
			c.sendError(msg.ID, ErrCodeMethodNotFound, "no handler for "+msg.Method)
			return
		}

	case MethodTerminalRelease:
		if c.onTerminalRelease != nil {
			var params TerminalReleaseParams
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				c.sendError(msg.ID, ErrCodeInvalidParams, "invalid params: "+err.Error())
				return
			}
			handlerErr = c.onTerminalRelease(params)
			if handlerErr == nil {
				result = struct{}{}
			}
		} else {
			c.sendError(msg.ID, ErrCodeMethodNotFound, "no handler for "+msg.Method)
			return
		}

	default:
		c.sendError(msg.ID, ErrCodeMethodNotFound, "unknown method: "+msg.Method)
		return
	}

	if handlerErr != nil {
		c.sendError(msg.ID, ErrCodeInternal, handlerErr.Error())
		return
	}

	c.sendResult(msg.ID, result)
}

// sendResult sends a successful JSON-RPC response.
func (c *Client) sendResult(id *json.RawMessage, result any) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		log.Printf("acp: failed to marshal result: %v", err)
		c.sendError(id, ErrCodeInternal, "failed to marshal result")
		return
	}

	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resultJSON,
	}

	if err := c.transport.Send(msg); err != nil {
		log.Printf("acp: failed to send response: %v", err)
	}
}

// sendError sends a JSON-RPC error response.
func (c *Client) sendError(id *json.RawMessage, code int, message string) {
	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}

	if err := c.transport.Send(msg); err != nil {
		log.Printf("acp: failed to send error response: %v", err)
	}
}
