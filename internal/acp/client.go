package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
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

	// codexSessions stores compatibility state for sessions created via
	// `codex app-server` (non-ACP JSON-RPC dialect).
	codexSessions map[string]*codexSessionState
	codexMu       sync.RWMutex
}

type codexSessionState struct {
	CWD               string
	ModelID           string
	ReasoningEffort   string
	Summary           string
	ApprovalPolicy    string
	SandboxPolicyType string
	PromptDone        chan string
}

// NewClient creates an ACP client bound to the given transport. The transport
// must not be started yet; call Initialize to perform the handshake which
// also starts the transport if it hasn't been started.
func NewClient(transport *StdioTransport) *Client {
	c := &Client{
		transport:      transport,
		pending:        make(map[int64]chan json.RawMessage),
		RequestTimeout: DefaultRequestTimeout,
		codexSessions:  make(map[string]*codexSessionState),
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

// NewSession asks the agent to create a new session and returns the full
// session/new result.
func (c *Client) NewSession(ctx context.Context, cwd string, mcpServers []MCPServer) (*SessionNewResult, error) {
	if mcpServers == nil {
		mcpServers = []MCPServer{}
	}

	params := SessionNewParams{
		CWD:        cwd,
		MCPServers: mcpServers,
	}

	raw, err := c.call(ctx, MethodSessionNew, params)
	if err != nil {
		if isMethodUnavailable(err, MethodSessionNew) {
			return c.newSessionCodex(ctx, cwd)
		}
		return nil, fmt.Errorf("session/new: %w", err)
	}

	var result SessionNewResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("session/new: unmarshal result: %w", err)
	}
	return &result, nil
}

func (c *Client) newSessionCodex(ctx context.Context, cwd string) (*SessionNewResult, error) {
	raw, err := c.call(ctx, "newConversation", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("codex/newConversation: %w", err)
	}

	var conversation struct {
		ConversationID  string `json:"conversationId"`
		Model           string `json:"model"`
		ReasoningEffort string `json:"reasoningEffort"`
		RolloutPath     string `json:"rolloutPath"`
	}
	if err := json.Unmarshal(raw, &conversation); err != nil {
		return nil, fmt.Errorf("codex/newConversation: unmarshal result: %w", err)
	}
	if conversation.ConversationID == "" {
		return nil, fmt.Errorf("codex/newConversation: empty conversationId")
	}

	_, err = c.call(ctx, "addConversationListener", map[string]any{
		"conversationId": conversation.ConversationID,
	})
	if err != nil {
		return nil, fmt.Errorf("codex/addConversationListener: %w", err)
	}

	models := []SessionModel{}
	modelsRaw, modelErr := c.call(ctx, "model/list", map[string]any{})
	if modelErr == nil {
		var list struct {
			Data []struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
				Model       string `json:"model"`
			} `json:"data"`
		}
		if err := json.Unmarshal(modelsRaw, &list); err == nil {
			models = make([]SessionModel, 0, len(list.Data))
			for _, m := range list.Data {
				id := m.ID
				if id == "" {
					id = m.Model
				}
				if id == "" {
					continue
				}
				name := m.DisplayName
				if name == "" {
					name = id
				}
				models = append(models, SessionModel{ModelID: id, Name: name})
			}
		}
	}

	if conversation.Model != "" && !sessionModelExists(models, conversation.Model) {
		models = append(models, SessionModel{ModelID: conversation.Model, Name: conversation.Model})
	}

	result := &SessionNewResult{
		SessionID: conversation.ConversationID,
		Models: &SessionModelsState{
			CurrentModelID:  conversation.Model,
			AvailableModels: models,
		},
	}

	c.codexMu.Lock()
	c.codexSessions[conversation.ConversationID] = &codexSessionState{
		CWD:               cwd,
		ModelID:           conversation.Model,
		ReasoningEffort:   conversation.ReasoningEffort,
		Summary:           "none",
		ApprovalPolicy:    "on-request",
		SandboxPolicyType: "workspace-write",
	}
	c.codexMu.Unlock()

	return result, nil
}

func sessionModelExists(models []SessionModel, modelID string) bool {
	for _, m := range models {
		if m.ModelID == modelID {
			return true
		}
	}
	return false
}

// LoadSession asks the agent to load an existing session.
func (c *Client) LoadSession(ctx context.Context, sessionID, cwd string, mcpServers []MCPServer) error {
	if mcpServers == nil {
		mcpServers = []MCPServer{}
	}

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

// ResumeSession asks the agent to resume an existing session.
func (c *Client) ResumeSession(ctx context.Context, sessionID, cwd string, mcpServers []MCPServer) (*SessionResumeResult, error) {
	if c.isCodexSession(sessionID) {
		return nil, fmt.Errorf("session/resume unsupported for codex compatibility sessions")
	}

	if mcpServers == nil {
		mcpServers = []MCPServer{}
	}

	params := SessionResumeParams{
		SessionID:  sessionID,
		CWD:        cwd,
		MCPServers: mcpServers,
	}

	raw, err := c.call(ctx, MethodSessionResume, params)
	if err != nil {
		if isMethodUnavailable(err, MethodSessionResume) {
			raw, err = c.call(ctx, "unstable_resumeSession", params)
			if err != nil {
				return nil, fmt.Errorf("session/resume: %w", err)
			}
		} else {
			return nil, fmt.Errorf("session/resume: %w", err)
		}
	}

	var result SessionResumeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("session/resume: unmarshal result: %w", err)
	}

	return &result, nil
}

// ListSessions returns a page of remote sessions, if supported by the agent.
func (c *Client) ListSessions(ctx context.Context, cwd, cursor string) (*SessionListResult, error) {
	params := SessionListParams{
		CWD:    cwd,
		Cursor: cursor,
	}

	raw, err := c.call(ctx, MethodSessionList, params)
	if err != nil {
		if isMethodUnavailable(err, MethodSessionList) {
			raw, err = c.call(ctx, "unstable_listSessions", params)
			if err != nil {
				return nil, fmt.Errorf("session/list: %w", err)
			}
		} else {
			return nil, fmt.Errorf("session/list: %w", err)
		}
	}

	var result SessionListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("session/list: unmarshal result: %w", err)
	}

	return &result, nil
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
		if isMethodUnavailable(err, MethodSessionPrompt) {
			return c.promptCodex(ctx, sessionID, prompt)
		}
		return nil, fmt.Errorf("session/prompt: %w", err)
	}

	var result SessionPromptResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("session/prompt: unmarshal result: %w", err)
	}
	return &result, nil
}

func (c *Client) promptCodex(ctx context.Context, sessionID string, prompt []ContentBlock) (*SessionPromptResult, error) {
	textParts := make([]string, 0, len(prompt))
	for _, block := range prompt {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			textParts = append(textParts, block.Text)
		}
	}

	if len(textParts) == 0 {
		return nil, fmt.Errorf("codex/sendUserTurn: empty text prompt")
	}
	text := strings.Join(textParts, "\n\n")

	c.codexMu.Lock()
	state, ok := c.codexSessions[sessionID]
	if !ok {
		c.codexMu.Unlock()
		return nil, fmt.Errorf("codex/sendUserTurn: session %q not found", sessionID)
	}

	doneCh := make(chan string, 1)
	state.PromptDone = doneCh

	modelID := state.ModelID
	reasoning := state.ReasoningEffort
	summary := state.Summary
	approval := state.ApprovalPolicy
	sandboxType := state.SandboxPolicyType
	cwd := state.CWD
	c.codexMu.Unlock()

	if summary == "" {
		summary = "none"
	}
	if approval == "" {
		approval = "on-request"
	}
	if sandboxType == "" {
		sandboxType = "workspace-write"
	}

	params := map[string]any{
		"conversationId": sessionID,
		"cwd":            cwd,
		"items": []map[string]any{{
			"type": "text",
			"data": map[string]any{
				"text": text,
			},
		}},
		"approvalPolicy": approval,
		"sandboxPolicy": map[string]any{
			"type": sandboxType,
		},
		"summary": summary,
	}
	if modelID != "" {
		params["model"] = modelID
	}
	if reasoning != "" {
		params["reasoningEffort"] = reasoning
	}

	_, err := c.call(ctx, "sendUserTurn", params)
	if err != nil {
		c.codexMu.Lock()
		if current, exists := c.codexSessions[sessionID]; exists && current.PromptDone == doneCh {
			current.PromptDone = nil
		}
		c.codexMu.Unlock()
		return nil, fmt.Errorf("codex/sendUserTurn: %w", err)
	}

	select {
	case reason := <-doneCh:
		if reason == "" {
			reason = "end_turn"
		}
		c.codexMu.Lock()
		if current, exists := c.codexSessions[sessionID]; exists && current.PromptDone == doneCh {
			current.PromptDone = nil
		}
		c.codexMu.Unlock()
		return &SessionPromptResult{StopReason: reason}, nil
	case <-ctx.Done():
		c.codexMu.Lock()
		if current, exists := c.codexSessions[sessionID]; exists && current.PromptDone == doneCh {
			current.PromptDone = nil
		}
		c.codexMu.Unlock()
		return nil, ctx.Err()
	}
}

// Cancel requests cancellation of an in-progress prompt. This is a
// notification (fire-and-forget).
func (c *Client) Cancel(sessionID string) error {
	if c.isCodexSession(sessionID) {
		// Best effort: codex app-server uses a different interrupt API.
		// We keep this non-fatal so UI cancel remains responsive.
		_, _ = c.call(context.Background(), "interruptConversation", map[string]any{
			"conversationId": sessionID,
		})
		return nil
	}

	params := SessionCancelParams{
		SessionID: sessionID,
	}
	return c.notify(MethodSessionCancel, params)
}

// SetMode asks the agent to switch operating modes.
func (c *Client) SetMode(ctx context.Context, sessionID, mode string) error {
	if c.isCodexSession(sessionID) {
		approvalPolicy, sandboxPolicy, ok := codexPoliciesForMode(mode)
		if !ok {
			return fmt.Errorf("codex/set_mode: unsupported mode %q", mode)
		}

		c.codexMu.Lock()
		state, exists := c.codexSessions[sessionID]
		if !exists {
			c.codexMu.Unlock()
			return fmt.Errorf("codex/set_mode: session %q not found", sessionID)
		}
		state.ApprovalPolicy = approvalPolicy
		state.SandboxPolicyType = sandboxPolicy
		c.codexMu.Unlock()
		return nil
	}

	params := SessionSetModeParams{
		SessionID: sessionID,
		ModeID:    mode,
	}

	_, err := c.call(ctx, MethodSessionSetMode, params)
	if err == nil {
		return nil
	}

	legacy := SessionSetModeLegacyParams{
		SessionID: sessionID,
		Mode:      mode,
	}
	_, legacyErr := c.call(ctx, MethodSessionSetModeOld, legacy)
	if legacyErr == nil {
		return nil
	}

	return fmt.Errorf("session/set_mode: %w (legacy fallback: %v)", err, legacyErr)
}

func codexPoliciesForMode(mode string) (approvalPolicy, sandboxPolicy string, ok bool) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "full-access", "full_access", "full":
		return "never", "danger-full-access", true
	case "restricted", "safe", "workspace", "workspace-write":
		return "on-request", "workspace-write", true
	case "plan", "planning", "read-only", "readonly":
		return "on-request", "read-only", true
	default:
		return "", "", false
	}
}

// SetConfigOption asks the agent to set a session configuration option.
func (c *Client) SetConfigOption(ctx context.Context, sessionID, configID, value string) error {
	params := SessionSetConfigOptionParams{
		SessionID: sessionID,
		ConfigID:  configID,
		Value:     value,
	}

	_, err := c.call(ctx, MethodSessionSetConfig, params)
	if err != nil {
		return fmt.Errorf("session/set_config_option: %w", err)
	}
	return nil
}

// SetModel asks the agent to switch the active session model.
// It first tries the OpenCode extension method `session/set_model`, then
// falls back to ACP `session/set_config_option` for agents that implement it.
func (c *Client) SetModel(ctx context.Context, sessionID, modelID string) error {
	if c.isCodexSession(sessionID) {
		c.codexMu.Lock()
		if state, ok := c.codexSessions[sessionID]; ok {
			state.ModelID = modelID
		}
		c.codexMu.Unlock()
		return nil
	}

	params := SessionSetModelParams{
		SessionID: sessionID,
		ModelID:   modelID,
	}

	_, err := c.call(ctx, MethodSessionSetModel, params)
	if err == nil {
		return nil
	}

	if !isMethodUnavailable(err, MethodSessionSetModel) {
		return fmt.Errorf("session/set_model: %w", err)
	}

	if cfgErr := c.SetConfigOption(ctx, sessionID, "model", modelID); cfgErr != nil {
		return fmt.Errorf("session/set_model unavailable and set_config_option failed: %w", cfgErr)
	}

	return nil
}

func isMethodUnavailable(err error, method string) bool {
	var rpcErr *JSONRPCError
	if errors.As(err, &rpcErr) {
		if rpcErr.Code == ErrCodeMethodNotFound {
			return true
		}
		if rpcErr.Code == ErrCodeInvalidRequest && strings.Contains(rpcErr.Message, "unknown variant") {
			if method == "" {
				return true
			}
			return strings.Contains(rpcErr.Message, "`"+method+"`")
		}
	}
	return false
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

	c.codexMu.Lock()
	for _, state := range c.codexSessions {
		state.PromptDone = nil
	}
	c.codexMu.Unlock()

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

	case "item/agentMessage/delta":
		c.notifMu.RLock()
		h := c.onSessionUpdate
		c.notifMu.RUnlock()

		if h != nil {
			var params struct {
				ThreadID string `json:"threadId"`
				Delta    string `json:"delta"`
			}
			if err := json.Unmarshal(msg.Params, &params); err != nil {
				log.Printf("acp: failed to unmarshal codex item/agentMessage/delta params: %v", err)
				return
			}
			if strings.TrimSpace(params.Delta) == "" || params.ThreadID == "" {
				return
			}

			h(SessionUpdateParams{
				SessionID: params.ThreadID,
				Update: SessionUpdate{
					Type: UpdateAgentMessageChunk,
					MessageContent: &ContentBlock{
						Type: "text",
						Text: params.Delta,
					},
				},
			})
		}

	case "codex/event/task_complete":
		var params struct {
			ConversationID string `json:"conversationId"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			log.Printf("acp: failed to unmarshal codex task_complete params: %v", err)
			return
		}
		if params.ConversationID != "" {
			c.signalCodexPromptDone(params.ConversationID, "end_turn")
		}

	case "task/completed":
		var params struct {
			ThreadID   string `json:"threadId"`
			StopReason string `json:"stopReason"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			log.Printf("acp: failed to unmarshal codex task/completed params: %v", err)
			return
		}
		if params.ThreadID != "" {
			c.signalCodexPromptDone(params.ThreadID, params.StopReason)
		}

	default:
		if strings.HasPrefix(msg.Method, "codex/") ||
			strings.HasPrefix(msg.Method, "item/") ||
			strings.HasPrefix(msg.Method, "thread/") ||
			strings.HasPrefix(msg.Method, "turn/") ||
			strings.HasPrefix(msg.Method, "account/") {
			return
		}
		log.Printf("acp: unhandled notification: %s", msg.Method)
	}
}

func (c *Client) isCodexSession(sessionID string) bool {
	c.codexMu.RLock()
	defer c.codexMu.RUnlock()
	_, ok := c.codexSessions[sessionID]
	return ok
}

func (c *Client) signalCodexPromptDone(sessionID, stopReason string) {
	c.codexMu.RLock()
	state, ok := c.codexSessions[sessionID]
	c.codexMu.RUnlock()
	if !ok || state.PromptDone == nil {
		return
	}

	if stopReason == "" {
		stopReason = "end_turn"
	}

	select {
	case state.PromptDone <- stopReason:
	default:
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
