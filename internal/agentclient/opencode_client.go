package agentclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"bytesmith/internal/acp"
)

// OpenCodeClient speaks to a local "opencode serve" HTTP server.
type OpenCodeClient struct {
	baseURL    string
	defaultCWD string
	httpClient *http.Client
	eventHTTP  *http.Client
	stderrCh   <-chan string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	notifMu         sync.RWMutex
	onSessionUpdate func(acp.SessionUpdateParams)

	handlerMu           sync.RWMutex
	onRequestPermission func(acp.RequestPermissionParams) acp.RequestPermissionResult
	onFSReadTextFile    func(acp.FSReadTextFileParams) (*acp.FSReadTextFileResult, error)
	onFSWriteTextFile   func(acp.FSWriteTextFileParams) error
	onTerminalCreate    func(acp.TerminalCreateParams) (*acp.TerminalCreateResult, error)
	onTerminalOutput    func(acp.TerminalOutputParams) (*acp.TerminalOutputResult, error)
	onTerminalWait      func(acp.TerminalWaitParams) (*acp.TerminalWaitResult, error)
	onTerminalKill      func(acp.TerminalKillParams) error
	onTerminalRelease   func(acp.TerminalReleaseParams) error

	sessionMu    sync.RWMutex
	sessionCWD   map[string]string
	toolCallSeen map[string]map[string]bool
	sessionModel map[string]openCodeModelRef

	promptMu      sync.Mutex
	promptWaiters map[string][]chan string
}

var _ Client = (*OpenCodeClient)(nil)

func NewOpenCode(baseURL, defaultCWD string) (*OpenCodeClient, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return nil, fmt.Errorf("opencode: empty base URL")
	}
	if _, err := url.Parse(trimmed); err != nil {
		return nil, fmt.Errorf("opencode: invalid base URL: %w", err)
	}

	transport := &http.Transport{
		Proxy: nil,
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &OpenCodeClient{
		baseURL:    strings.TrimRight(trimmed, "/"),
		defaultCWD: strings.TrimSpace(defaultCWD),
		httpClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		},
		eventHTTP: &http.Client{
			Transport: transport,
		},
		stderrCh:      closedStringChannel(),
		ctx:           ctx,
		cancel:        cancel,
		sessionCWD:    make(map[string]string),
		toolCallSeen:  make(map[string]map[string]bool),
		sessionModel:  make(map[string]openCodeModelRef),
		promptWaiters: make(map[string][]chan string),
	}

	c.wg.Add(1)
	go c.eventLoop()
	return c, nil
}

func (c *OpenCodeClient) Close() error {
	c.cancel()
	c.wg.Wait()
	return nil
}

func (c *OpenCodeClient) StderrCh() <-chan string {
	return c.stderrCh
}

func (c *OpenCodeClient) NewSession(ctx context.Context, cwd string, _ []acp.MCPServer) (*acp.SessionNewResult, error) {
	var resp openCodeSession
	if err := c.requestJSON(ctx, http.MethodPost, "/session", directoryQuery(cwd), map[string]any{}, &resp); err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.ID) == "" {
		return nil, fmt.Errorf("opencode: new session returned empty id")
	}

	finalCWD := resolveSessionDir(cwd, resp.Directory, c.defaultCWD)
	c.trackSession(resp.ID, finalCWD)
	models, _ := c.loadModels(ctx, finalCWD)

	return &acp.SessionNewResult{
		SessionID: resp.ID,
		Models:    models,
	}, nil
}

func (c *OpenCodeClient) LoadSession(ctx context.Context, sessionID, cwd string, _ []acp.MCPServer) error {
	var resp openCodeSession
	path := fmt.Sprintf("/session/%s", url.PathEscape(sessionID))
	if err := c.requestJSON(ctx, http.MethodGet, path, directoryQuery(cwd), nil, &resp); err != nil {
		return err
	}
	finalCWD := resolveSessionDir(cwd, resp.Directory, c.defaultCWD)
	c.trackSession(sessionID, finalCWD)
	if model, ok := c.loadSessionModel(ctx, sessionID, finalCWD); ok {
		c.setSessionModel(sessionID, model)
	}
	return nil
}

func (c *OpenCodeClient) ResumeSession(ctx context.Context, sessionID, cwd string, _ []acp.MCPServer) (*acp.SessionResumeResult, error) {
	if err := c.LoadSession(ctx, sessionID, cwd, nil); err != nil {
		return nil, err
	}
	finalCWD := c.sessionDirectory(sessionID)
	models, _ := c.loadModels(ctx, finalCWD)
	if models != nil {
		if model, ok := c.getSessionModel(sessionID); ok {
			fullID := model.String()
			if containsModel(models.AvailableModels, fullID) {
				models.CurrentModelID = fullID
			}
		}
	}
	return &acp.SessionResumeResult{
		SessionID: sessionID,
		Models:    models,
	}, nil
}

func (c *OpenCodeClient) ListSessions(ctx context.Context, cwd, _ string) (*acp.SessionListResult, error) {
	var sessions []openCodeSession
	if err := c.requestJSON(ctx, http.MethodGet, "/session", directoryQuery(cwd), nil, &sessions); err != nil {
		return nil, err
	}

	result := &acp.SessionListResult{
		Sessions: make([]acp.SessionInfo, 0, len(sessions)),
	}
	for _, s := range sessions {
		updatedAt := ""
		if s.Time.Updated > 0 {
			updatedAt = time.Unix(int64(s.Time.Updated), 0).UTC().Format(time.RFC3339)
		}
		result.Sessions = append(result.Sessions, acp.SessionInfo{
			SessionID: s.ID,
			CWD:       s.Directory,
			Title:     s.Title,
			UpdatedAt: updatedAt,
		})
	}
	return result, nil
}

func (c *OpenCodeClient) Prompt(ctx context.Context, sessionID string, prompt []acp.ContentBlock) (*acp.SessionPromptResult, error) {
	cwd := c.sessionDirectory(sessionID)
	parts := make([]map[string]string, 0, len(prompt))
	for _, block := range prompt {
		if block.Type != "text" {
			continue
		}
		text := strings.TrimSpace(block.Text)
		if text == "" {
			continue
		}
		parts = append(parts, map[string]string{
			"type": "text",
			"text": text,
		})
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("opencode: empty text prompt")
	}

	waitCh, cleanup := c.registerPromptWaiter(sessionID)
	defer cleanup()

	payload := map[string]any{
		"parts": parts,
	}
	if model, ok := c.getSessionModel(sessionID); ok {
		payload["model"] = map[string]string{
			"providerID": model.ProviderID,
			"modelID":    model.ModelID,
		}
	}

	path := fmt.Sprintf("/session/%s/message", url.PathEscape(sessionID))
	if err := c.requestJSON(ctx, http.MethodPost, path, directoryQuery(cwd), payload, nil); err != nil {
		return nil, err
	}

	reason, err := c.waitPromptDone(ctx, waitCh)
	if err != nil {
		return nil, err
	}
	if reason == "" {
		reason = "end_turn"
	}
	return &acp.SessionPromptResult{StopReason: reason}, nil
}

func (c *OpenCodeClient) Cancel(sessionID string) error {
	cwd := c.sessionDirectory(sessionID)
	path := fmt.Sprintf("/session/%s/abort", url.PathEscape(sessionID))
	if err := c.requestJSON(context.Background(), http.MethodPost, path, directoryQuery(cwd), map[string]any{}, nil); err != nil {
		return err
	}
	c.signalPromptDone(sessionID, "cancelled")
	return nil
}

func (c *OpenCodeClient) SetMode(ctx context.Context, sessionID, mode string) error {
	cwd := c.sessionDirectory(sessionID)
	if err := c.runCommand(ctx, sessionID, cwd, "mode", mode); err != nil {
		return fmt.Errorf("session/set_mode unsupported by opencode server: %w", err)
	}
	return nil
}

func (c *OpenCodeClient) SetModel(ctx context.Context, sessionID, modelID string) error {
	cwd := c.sessionDirectory(sessionID)
	providers, err := c.fetchProviders(ctx, cwd)
	if err != nil {
		return fmt.Errorf("failed to load available models: %w", err)
	}
	model, err := resolveModelID(modelID, providers.Providers)
	if err != nil {
		return err
	}
	c.setSessionModel(sessionID, model)
	return nil
}

func (c *OpenCodeClient) SetConfigOption(ctx context.Context, sessionID, configID, value string) error {
	cwd := c.sessionDirectory(sessionID)
	if err := c.runCommand(ctx, sessionID, cwd, configID, value); err != nil {
		return fmt.Errorf("session/set_config_option unsupported by opencode server: %w", err)
	}
	return nil
}

func (c *OpenCodeClient) OnSessionUpdate(handler func(acp.SessionUpdateParams)) {
	c.notifMu.Lock()
	c.onSessionUpdate = handler
	c.notifMu.Unlock()
}

func (c *OpenCodeClient) OnRequestPermission(handler func(acp.RequestPermissionParams) acp.RequestPermissionResult) {
	c.handlerMu.Lock()
	c.onRequestPermission = handler
	c.handlerMu.Unlock()
}

func (c *OpenCodeClient) OnFSReadTextFile(handler func(acp.FSReadTextFileParams) (*acp.FSReadTextFileResult, error)) {
	c.handlerMu.Lock()
	c.onFSReadTextFile = handler
	c.handlerMu.Unlock()
}

func (c *OpenCodeClient) OnFSWriteTextFile(handler func(acp.FSWriteTextFileParams) error) {
	c.handlerMu.Lock()
	c.onFSWriteTextFile = handler
	c.handlerMu.Unlock()
}

func (c *OpenCodeClient) OnTerminalCreate(handler func(acp.TerminalCreateParams) (*acp.TerminalCreateResult, error)) {
	c.handlerMu.Lock()
	c.onTerminalCreate = handler
	c.handlerMu.Unlock()
}

func (c *OpenCodeClient) OnTerminalOutput(handler func(acp.TerminalOutputParams) (*acp.TerminalOutputResult, error)) {
	c.handlerMu.Lock()
	c.onTerminalOutput = handler
	c.handlerMu.Unlock()
}

func (c *OpenCodeClient) OnTerminalWait(handler func(acp.TerminalWaitParams) (*acp.TerminalWaitResult, error)) {
	c.handlerMu.Lock()
	c.onTerminalWait = handler
	c.handlerMu.Unlock()
}

func (c *OpenCodeClient) OnTerminalKill(handler func(acp.TerminalKillParams) error) {
	c.handlerMu.Lock()
	c.onTerminalKill = handler
	c.handlerMu.Unlock()
}

func (c *OpenCodeClient) OnTerminalRelease(handler func(acp.TerminalReleaseParams) error) {
	c.handlerMu.Lock()
	c.onTerminalRelease = handler
	c.handlerMu.Unlock()
}

func (c *OpenCodeClient) eventLoop() {
	defer c.wg.Done()

	for {
		if c.ctx.Err() != nil {
			return
		}
		err := c.consumeEvents(c.ctx)
		if err == nil || errors.Is(err, context.Canceled) {
			return
		}

		log.Printf("opencode: event stream ended (%v); retrying", err)
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (c *OpenCodeClient) consumeEvents(ctx context.Context) error {
	values := directoryQuery(c.defaultCWD)
	endpoint := c.baseURL + "/event"
	if values != nil {
		endpoint += "?" + values.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.eventHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return fmt.Errorf("event stream failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	var payload bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if payload.Len() > 0 {
				raw := strings.TrimSpace(payload.String())
				payload.Reset()
				c.handleEvent(raw)
			}
			continue
		}

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			if len(data) > 0 && data[0] == ' ' {
				data = data[1:]
			}
			payload.WriteString(data)
			payload.WriteByte('\n')
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return io.EOF
}

func (c *OpenCodeClient) handleEvent(raw string) {
	if raw == "" {
		return
	}

	event, err := decodeOpenCodeEvent(raw)
	if err != nil {
		log.Printf("opencode: failed to decode event: %v", err)
		return
	}

	switch event.Type {
	case "message.part.updated":
		c.handleMessagePartUpdated(event.Properties)
	case "message.updated":
		c.handleMessageUpdated(event.Properties)
	case "session.idle":
		c.handleSessionIdle(event.Properties)
	case "session.status":
		c.handleSessionStatus(event.Properties)
	case "permission.asked":
		c.handlePermissionAsked(event.Properties)
	case "permission.updated":
		c.handlePermissionUpdated(event.Properties)
	}
}

func (c *OpenCodeClient) handleMessagePartUpdated(raw json.RawMessage) {
	var props openCodeMessagePartUpdated
	if err := json.Unmarshal(raw, &props); err != nil {
		log.Printf("opencode: invalid message.part.updated: %v", err)
		return
	}

	if !c.sessionTracked(props.Part.SessionID) {
		return
	}

	switch props.Part.Type {
	case "text":
		delta := props.Delta
		if delta == "" && props.Part.Time != nil && props.Part.Time.End != nil {
			delta = props.Part.Text
		}
		if delta == "" || props.Part.Ignored {
			return
		}
		c.emitSessionUpdate(acp.SessionUpdateParams{
			SessionID: props.Part.SessionID,
			Update: acp.SessionUpdate{
				Type: acp.UpdateAgentMessageChunk,
				MessageContent: &acp.ContentBlock{
					Type: "text",
					Text: delta,
				},
			},
		})
	case "reasoning":
		delta := props.Delta
		if delta == "" && props.Part.Time != nil && props.Part.Time.End != nil {
			delta = props.Part.Text
		}
		if delta == "" || props.Part.Ignored {
			return
		}
		c.emitSessionUpdate(acp.SessionUpdateParams{
			SessionID: props.Part.SessionID,
			Update: acp.SessionUpdate{
				Type: acp.UpdateAgentThoughtChunk,
				MessageContent: &acp.ContentBlock{
					Type: "text",
					Text: delta,
				},
			},
		})
	case "tool":
		c.handleToolPart(props.Part)
	}
}

func (c *OpenCodeClient) handleMessageUpdated(raw json.RawMessage) {
	var props openCodeMessageUpdated
	if err := json.Unmarshal(raw, &props); err != nil {
		log.Printf("opencode: invalid message.updated: %v", err)
		return
	}
	if !c.sessionTracked(props.Info.SessionID) {
		return
	}
	if strings.EqualFold(props.Info.Role, "assistant") && strings.TrimSpace(props.Info.Finish) != "" {
		c.signalPromptDone(props.Info.SessionID, props.Info.Finish)
	}
}

func (c *OpenCodeClient) handleSessionIdle(raw json.RawMessage) {
	var props struct {
		SessionID string `json:"sessionID"`
	}
	if err := json.Unmarshal(raw, &props); err != nil {
		return
	}
	if strings.TrimSpace(props.SessionID) != "" {
		c.signalPromptDone(props.SessionID, "end_turn")
	}
}

func (c *OpenCodeClient) handleSessionStatus(raw json.RawMessage) {
	var props struct {
		SessionID string `json:"sessionID"`
		Status    struct {
			Type string `json:"type"`
		} `json:"status"`
	}
	if err := json.Unmarshal(raw, &props); err != nil {
		return
	}
	if strings.EqualFold(props.Status.Type, "idle") && strings.TrimSpace(props.SessionID) != "" {
		c.signalPromptDone(props.SessionID, "end_turn")
	}
}

func (c *OpenCodeClient) handleToolPart(part openCodePart) {
	callID := nonEmpty(part.CallID, part.ID)
	if callID == "" {
		return
	}

	updateType := c.nextToolUpdateType(part.SessionID, callID)
	if part.State.Status == "pending" {
		updateType = acp.UpdateToolCall
	}

	kind := mapToolKind(part.Tool)
	status := normalizeToolStatus(part.State.Status)
	title := nonEmpty(part.State.Title, part.Tool, "Tool")

	update := acp.SessionUpdate{
		Type:       updateType,
		ToolCallID: callID,
		Title:      title,
		Kind:       kind,
		Status:     status,
		RawInput:   marshalRaw(part.State.Input),
	}

	if len(part.State.Input) > 0 {
		update.Locations = toLocations(kind, part.State.Input)
	}

	switch part.State.Status {
	case "completed":
		update.Status = "completed"
		update.RawOutput = marshalRaw(map[string]any{
			"output":   part.State.Output,
			"metadata": part.State.Metadata,
		})
		content := []acp.ToolCallContent{}
		if strings.TrimSpace(part.State.Output) != "" {
			content = append(content, acp.ToolCallContent{
				Type: "content",
				Content: &acp.ContentBlock{
					Type: "text",
					Text: part.State.Output,
				},
			})
		}
		if kind == "edit" {
			path := asString(part.State.Input["filePath"])
			oldText := asString(part.State.Input["oldString"])
			newText := asString(part.State.Input["newString"])
			if newText == "" {
				newText = asString(part.State.Input["content"])
			}
			if path != "" || oldText != "" || newText != "" {
				content = append(content, acp.ToolCallContent{
					Type:    "diff",
					Path:    path,
					OldText: oldText,
					NewText: newText,
				})
			}
		}
		update.ToolContent = content
	case "error":
		update.Status = "failed"
		update.RawOutput = marshalRaw(map[string]any{"error": part.State.Error})
		if strings.TrimSpace(part.State.Error) != "" {
			update.ToolContent = []acp.ToolCallContent{
				{
					Type: "content",
					Content: &acp.ContentBlock{
						Type: "text",
						Text: part.State.Error,
					},
				},
			}
		}
	case "running":
		update.Status = "in_progress"
	case "pending":
		update.Status = "pending"
	}

	c.emitSessionUpdate(acp.SessionUpdateParams{
		SessionID: part.SessionID,
		Update:    update,
	})
}

func (c *OpenCodeClient) handlePermissionAsked(raw json.RawMessage) {
	var perm openCodePermissionAsked
	if err := json.Unmarshal(raw, &perm); err != nil {
		log.Printf("opencode: invalid permission.asked: %v", err)
		return
	}
	c.handlePermission(
		perm.SessionID,
		perm.ID,
		nonEmpty(perm.Tool.CallID, perm.ID),
		nonEmpty(asString(perm.Metadata["title"]), perm.Permission, "Permission"),
		perm.Permission,
	)
}

func (c *OpenCodeClient) handlePermissionUpdated(raw json.RawMessage) {
	var perm openCodePermissionUpdated
	if err := json.Unmarshal(raw, &perm); err != nil {
		log.Printf("opencode: invalid permission.updated: %v", err)
		return
	}
	c.handlePermission(
		perm.SessionID,
		perm.ID,
		nonEmpty(perm.CallID, perm.ID),
		nonEmpty(perm.Title, perm.Type, "Permission"),
		perm.Type,
	)
}

func (c *OpenCodeClient) handlePermission(sessionID, permissionID, toolCallID, title, kind string) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(permissionID) == "" || !c.sessionTracked(sessionID) {
		return
	}

	c.handlerMu.RLock()
	handler := c.onRequestPermission
	c.handlerMu.RUnlock()

	result := acp.RequestPermissionResult{
		Outcome: acp.PermissionOutcome{Outcome: "cancelled"},
	}
	if handler != nil {
		result = handler(acp.RequestPermissionParams{
			SessionID: sessionID,
			ToolCall: acp.ToolCallUpdate{
				ToolCallID: toolCallID,
				Title:      title,
				Kind:       mapToolKind(kind),
				Status:     "pending",
			},
			Options: []acp.PermissionOption{
				{OptionID: "approved", Name: "Allow once", Kind: "allow_once"},
				{OptionID: "approved_for_session", Name: "Always allow", Kind: "allow_always"},
				{OptionID: "denied", Name: "Deny", Kind: "reject_once"},
			},
		})
	}

	response := mapPermissionSelection(result)
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	cwd := c.sessionDirectory(sessionID)
	path := fmt.Sprintf("/session/%s/permissions/%s", url.PathEscape(sessionID), url.PathEscape(permissionID))
	_ = c.requestJSON(ctx, http.MethodPost, path, directoryQuery(cwd), map[string]string{
		"response": response,
	}, nil)
}

func (c *OpenCodeClient) runCommand(ctx context.Context, sessionID, cwd, command, arguments string) error {
	path := fmt.Sprintf("/session/%s/command", url.PathEscape(sessionID))
	return c.requestJSON(ctx, http.MethodPost, path, directoryQuery(cwd), map[string]string{
		"command":   command,
		"arguments": arguments,
	}, nil)
}

func (c *OpenCodeClient) loadModels(ctx context.Context, cwd string) (*acp.SessionModelsState, error) {
	providersResp, err := c.fetchProviders(ctx, cwd)
	if err != nil {
		return nil, err
	}

	modelByID := make(map[string]string)
	for _, provider := range providersResp.Providers {
		providerID := strings.TrimSpace(provider.ID)
		if providerID == "" {
			continue
		}
		for id, model := range provider.Models {
			localModelID := strings.TrimSpace(nonEmpty(model.ID, id))
			if localModelID == "" {
				continue
			}
			fullID := joinModelID(providerID, localModelID)
			name := strings.TrimSpace(nonEmpty(model.Name, localModelID))
			modelByID[fullID] = name
		}
	}
	if len(modelByID) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(modelByID))
	for id := range modelByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	models := make([]acp.SessionModel, 0, len(ids))
	for _, id := range ids {
		models = append(models, acp.SessionModel{
			ModelID: id,
			Name:    modelByID[id],
		})
	}

	current := ""
	var cfg struct {
		Model string `json:"model"`
	}
	if err := c.requestJSON(ctx, http.MethodGet, "/config", directoryQuery(cwd), nil, &cfg); err == nil {
		cfgModelID := strings.TrimSpace(cfg.Model)
		if _, ok := modelByID[cfgModelID]; ok {
			current = cfgModelID
		} else if model, err := resolveModelID(cfgModelID, providersResp.Providers); err == nil {
			fullID := model.String()
			if _, exists := modelByID[fullID]; exists {
				current = fullID
			}
		}
	}
	if current == "" {
		for _, provider := range providersResp.Providers {
			providerID := strings.TrimSpace(provider.ID)
			defID := strings.TrimSpace(providersResp.Default[providerID])
			if defID != "" {
				fullID := joinModelID(providerID, defID)
				if _, ok := modelByID[fullID]; ok {
					current = fullID
					break
				}
			}
		}
	}
	if current == "" && len(ids) > 0 {
		current = ids[0]
	}

	return &acp.SessionModelsState{
		CurrentModelID:  current,
		AvailableModels: models,
	}, nil
}

func (c *OpenCodeClient) fetchProviders(ctx context.Context, cwd string) (openCodeProvidersResponse, error) {
	var providersResp openCodeProvidersResponse
	configErr := c.requestJSON(ctx, http.MethodGet, "/config/providers", directoryQuery(cwd), nil, &providersResp)
	if configErr == nil && len(providersResp.Providers) > 0 {
		return providersResp, nil
	}

	var providerListResp openCodeProviderListResponse
	if err := c.requestJSON(ctx, http.MethodGet, "/provider", directoryQuery(cwd), nil, &providerListResp); err != nil {
		if configErr == nil {
			return providersResp, nil
		}
		return openCodeProvidersResponse{}, err
	}

	return openCodeProvidersResponse{
		Providers: providerListResp.All,
		Default:   providerListResp.Default,
	}, nil
}

func (c *OpenCodeClient) loadSessionModel(ctx context.Context, sessionID, cwd string) (openCodeModelRef, bool) {
	path := fmt.Sprintf("/session/%s/message", url.PathEscape(sessionID))
	var messages []openCodeMessageWithParts
	if err := c.requestJSON(ctx, http.MethodGet, path, directoryQuery(cwd), nil, &messages); err != nil {
		return openCodeModelRef{}, false
	}

	for idx := len(messages) - 1; idx >= 0; idx-- {
		info := messages[idx].Info
		if !strings.EqualFold(strings.TrimSpace(info.Role), "user") {
			continue
		}
		providerID := strings.TrimSpace(info.Model.ProviderID)
		modelID := strings.TrimSpace(info.Model.ModelID)
		if providerID == "" || modelID == "" {
			continue
		}
		return openCodeModelRef{
			ProviderID: providerID,
			ModelID:    modelID,
		}, true
	}

	return openCodeModelRef{}, false
}

func (c *OpenCodeClient) setSessionModel(sessionID string, model openCodeModelRef) {
	if strings.TrimSpace(model.ProviderID) == "" || strings.TrimSpace(model.ModelID) == "" {
		c.sessionMu.Lock()
		delete(c.sessionModel, sessionID)
		c.sessionMu.Unlock()
		return
	}

	c.sessionMu.Lock()
	c.sessionModel[sessionID] = model
	c.sessionMu.Unlock()
}

func (c *OpenCodeClient) getSessionModel(sessionID string) (openCodeModelRef, bool) {
	c.sessionMu.RLock()
	model, ok := c.sessionModel[sessionID]
	c.sessionMu.RUnlock()
	if !ok || strings.TrimSpace(model.ProviderID) == "" || strings.TrimSpace(model.ModelID) == "" {
		return openCodeModelRef{}, false
	}
	return model, ok
}

func (c *OpenCodeClient) emitSessionUpdate(params acp.SessionUpdateParams) {
	c.notifMu.RLock()
	handler := c.onSessionUpdate
	c.notifMu.RUnlock()
	if handler != nil {
		handler(params)
	}
}

func (c *OpenCodeClient) trackSession(sessionID, cwd string) {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()

	c.sessionCWD[sessionID] = resolveSessionDir(cwd, "", c.defaultCWD)
	if _, ok := c.toolCallSeen[sessionID]; !ok {
		c.toolCallSeen[sessionID] = make(map[string]bool)
	}
}

func (c *OpenCodeClient) sessionTracked(sessionID string) bool {
	c.sessionMu.RLock()
	_, ok := c.sessionCWD[sessionID]
	c.sessionMu.RUnlock()
	return ok
}

func (c *OpenCodeClient) sessionDirectory(sessionID string) string {
	c.sessionMu.RLock()
	cwd := c.sessionCWD[sessionID]
	c.sessionMu.RUnlock()
	if strings.TrimSpace(cwd) != "" {
		return cwd
	}
	return c.defaultCWD
}

func (c *OpenCodeClient) nextToolUpdateType(sessionID, callID string) string {
	c.sessionMu.Lock()
	defer c.sessionMu.Unlock()

	seenMap, ok := c.toolCallSeen[sessionID]
	if !ok {
		seenMap = make(map[string]bool)
		c.toolCallSeen[sessionID] = seenMap
	}
	if seenMap[callID] {
		return acp.UpdateToolCallUpdate
	}
	seenMap[callID] = true
	return acp.UpdateToolCall
}

func (c *OpenCodeClient) registerPromptWaiter(sessionID string) (chan string, func()) {
	ch := make(chan string, 1)
	c.promptMu.Lock()
	c.promptWaiters[sessionID] = append(c.promptWaiters[sessionID], ch)
	c.promptMu.Unlock()

	cleanup := func() {
		c.promptMu.Lock()
		defer c.promptMu.Unlock()

		waiters := c.promptWaiters[sessionID]
		next := make([]chan string, 0, len(waiters))
		for _, existing := range waiters {
			if existing != ch {
				next = append(next, existing)
			}
		}
		if len(next) == 0 {
			delete(c.promptWaiters, sessionID)
		} else {
			c.promptWaiters[sessionID] = next
		}
	}

	return ch, cleanup
}

func (c *OpenCodeClient) waitPromptDone(ctx context.Context, ch <-chan string) (string, error) {
	select {
	case reason := <-ch:
		return reason, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *OpenCodeClient) signalPromptDone(sessionID, reason string) {
	if strings.TrimSpace(sessionID) == "" {
		return
	}
	if strings.TrimSpace(reason) == "" {
		reason = "end_turn"
	}

	c.promptMu.Lock()
	waiters := c.promptWaiters[sessionID]
	delete(c.promptWaiters, sessionID)
	c.promptMu.Unlock()

	for _, ch := range waiters {
		select {
		case ch <- reason:
		default:
		}
	}
}

func (c *OpenCodeClient) requestJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	fullURL := c.baseURL + path
	u, err := url.Parse(fullURL)
	if err != nil {
		return fmt.Errorf("opencode: parse request URL: %w", err)
	}
	if query != nil {
		q := u.Query()
		for key, values := range query {
			for _, v := range values {
				q.Add(key, v)
			}
		}
		u.RawQuery = q.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("opencode: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("opencode: create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("opencode: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return fmt.Errorf("opencode: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("opencode: %s %s returned %s: %s", method, path, resp.Status, strings.TrimSpace(string(respBody)))
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("opencode: decode response: %w", err)
		}
	}
	return nil
}

func mapPermissionSelection(result acp.RequestPermissionResult) string {
	switch strings.ToLower(strings.TrimSpace(result.Outcome.OptionID)) {
	case "approved", "once":
		return "once"
	case "approved_for_session", "always":
		return "always"
	case "denied", "reject":
		return "reject"
	}
	if strings.EqualFold(result.Outcome.Outcome, "selected") {
		return "once"
	}
	return "reject"
}

func mapToolKind(toolName string) string {
	tool := strings.ToLower(strings.TrimSpace(toolName))
	switch tool {
	case "bash":
		return "execute"
	case "webfetch":
		return "fetch"
	case "edit", "patch", "write":
		return "edit"
	case "grep", "glob", "context7_resolve_library_id", "context7_get_library_docs":
		return "search"
	case "list", "read":
		return "read"
	default:
		return "other"
	}
}

func toLocations(kind string, input map[string]any) []acp.ToolCallLocation {
	switch kind {
	case "read", "edit":
		if path := asString(input["filePath"]); path != "" {
			return []acp.ToolCallLocation{{Path: path}}
		}
	case "search":
		if path := asString(input["path"]); path != "" {
			return []acp.ToolCallLocation{{Path: path}}
		}
	}
	return nil
}

func normalizeToolStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "pending"
	case "running", "in_progress":
		return "in_progress"
	case "completed", "done":
		return "completed"
	case "error", "failed":
		return "failed"
	default:
		return "pending"
	}
}

func containsModel(models []acp.SessionModel, modelID string) bool {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return false
	}
	for _, m := range models {
		if strings.EqualFold(strings.TrimSpace(m.ModelID), modelID) {
			return true
		}
	}
	return false
}

func resolveModelID(input string, providers []openCodeProvider) (openCodeModelRef, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return openCodeModelRef{}, fmt.Errorf("invalid model id format, expected provider/model")
	}

	if strings.Contains(input, "/") {
		providerID, modelID, ok := splitModelID(input)
		if !ok {
			return openCodeModelRef{}, fmt.Errorf("invalid model id format, expected provider/model")
		}
		if providerHasModel(providers, providerID, modelID) {
			return openCodeModelRef{
				ProviderID: providerID,
				ModelID:    modelID,
			}, nil
		}
		return openCodeModelRef{}, fmt.Errorf("model not available for current opencode server: %s", input)
	}

	modelID := input
	matches := make([]openCodeModelRef, 0, 1)
	seen := make(map[string]bool)
	for _, provider := range providers {
		providerID := strings.TrimSpace(provider.ID)
		if providerID == "" {
			continue
		}
		if !providerContainsModel(provider, modelID) {
			continue
		}
		fullID := joinModelID(providerID, modelID)
		if seen[fullID] {
			continue
		}
		seen[fullID] = true
		matches = append(matches, openCodeModelRef{
			ProviderID: providerID,
			ModelID:    modelID,
		})
	}

	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return openCodeModelRef{}, fmt.Errorf("invalid model id format, expected provider/model")
	}
	return openCodeModelRef{}, fmt.Errorf("model not available for current opencode server: %s", input)
}

func splitModelID(input string) (string, string, bool) {
	input = strings.TrimSpace(input)
	parts := strings.SplitN(input, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	providerID := strings.TrimSpace(parts[0])
	modelID := strings.TrimSpace(parts[1])
	if providerID == "" || modelID == "" {
		return "", "", false
	}
	return providerID, modelID, true
}

func joinModelID(providerID, modelID string) string {
	return strings.TrimSpace(providerID) + "/" + strings.TrimSpace(modelID)
}

func providerHasModel(providers []openCodeProvider, providerID, modelID string) bool {
	providerID = strings.TrimSpace(providerID)
	modelID = strings.TrimSpace(modelID)
	for _, provider := range providers {
		if !strings.EqualFold(strings.TrimSpace(provider.ID), providerID) {
			continue
		}
		return providerContainsModel(provider, modelID)
	}
	return false
}

func providerContainsModel(provider openCodeProvider, modelID string) bool {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return false
	}
	for key, model := range provider.Models {
		candidate := strings.TrimSpace(nonEmpty(model.ID, key))
		if strings.EqualFold(candidate, modelID) {
			return true
		}
	}
	return false
}

func directoryQuery(cwd string) url.Values {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return nil
	}
	values := url.Values{}
	values.Set("directory", cwd)
	return values
}

func resolveSessionDir(requested, returned, fallback string) string {
	for _, candidate := range []string{requested, returned, fallback} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func marshalRaw(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return data
}

func nonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func decodeOpenCodeEvent(raw string) (openCodeEvent, error) {
	var event openCodeEvent
	if err := json.Unmarshal([]byte(raw), &event); err == nil {
		if strings.TrimSpace(event.Type) != "" {
			return event, nil
		}
	}

	var wrapped struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapped); err != nil {
		return openCodeEvent{}, err
	}
	if len(wrapped.Payload) == 0 {
		return openCodeEvent{}, fmt.Errorf("missing event payload")
	}
	if err := json.Unmarshal(wrapped.Payload, &event); err != nil {
		return openCodeEvent{}, err
	}
	if strings.TrimSpace(event.Type) == "" {
		return openCodeEvent{}, fmt.Errorf("missing event type")
	}
	return event, nil
}

type openCodeModelRef struct {
	ProviderID string
	ModelID    string
}

func (m openCodeModelRef) String() string {
	return joinModelID(m.ProviderID, m.ModelID)
}

type openCodeEvent struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
}

type openCodeMessagePartUpdated struct {
	Part  openCodePart `json:"part"`
	Delta string       `json:"delta"`
}

type openCodeMessageUpdated struct {
	Info openCodeMessageInfo `json:"info"`
}

type openCodeMessageInfo struct {
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
	Finish    string `json:"finish"`
}

type openCodeMessageWithParts struct {
	Info openCodePromptMessageInfo `json:"info"`
}

type openCodePromptMessageInfo struct {
	Role  string `json:"role"`
	Model struct {
		ProviderID string `json:"providerID"`
		ModelID    string `json:"modelID"`
	} `json:"model"`
}

type openCodePart struct {
	ID        string            `json:"id"`
	CallID    string            `json:"callID"`
	MessageID string            `json:"messageID"`
	SessionID string            `json:"sessionID"`
	Type      string            `json:"type"`
	Text      string            `json:"text"`
	Tool      string            `json:"tool"`
	Ignored   bool              `json:"ignored"`
	Time      *openCodePartTime `json:"time"`
	State     openCodeToolState `json:"state"`
}

type openCodePartTime struct {
	Start float64  `json:"start"`
	End   *float64 `json:"end,omitempty"`
}

type openCodeToolState struct {
	Status   string         `json:"status"`
	Title    string         `json:"title"`
	Output   string         `json:"output"`
	Error    string         `json:"error"`
	Input    map[string]any `json:"input"`
	Metadata map[string]any `json:"metadata"`
}

type openCodePermissionAsked struct {
	ID         string         `json:"id"`
	SessionID  string         `json:"sessionID"`
	Permission string         `json:"permission"`
	Metadata   map[string]any `json:"metadata"`
	Tool       struct {
		CallID string `json:"callID"`
	} `json:"tool"`
}

type openCodePermissionUpdated struct {
	ID        string `json:"id"`
	CallID    string `json:"callID"`
	SessionID string `json:"sessionID"`
	Title     string `json:"title"`
	Type      string `json:"type"`
}

type openCodeSession struct {
	ID        string `json:"id"`
	Directory string `json:"directory"`
	Title     string `json:"title"`
	Time      struct {
		Updated float64 `json:"updated"`
	} `json:"time"`
}

type openCodeProvider struct {
	ID     string                   `json:"id"`
	Models map[string]openCodeModel `json:"models"`
}

type openCodeModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type openCodeProvidersResponse struct {
	Providers []openCodeProvider `json:"providers"`
	Default   map[string]string  `json:"default"`
}

type openCodeProviderListResponse struct {
	All     []openCodeProvider `json:"all"`
	Default map[string]string  `json:"default"`
}
