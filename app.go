package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"bytesmith/internal/acp"
	"bytesmith/internal/agent"
	bfs "bytesmith/internal/fs"
	"bytesmith/internal/session"
	"bytesmith/internal/terminal"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------------------
// DTO types – JSON-serializable structs exposed to the frontend via Wails
// bindings. They intentionally avoid internal pointers so the TypeScript
// code generator produces clean interfaces.
// ---------------------------------------------------------------------------

// AgentInfo describes an ACP agent and whether it is installed locally.
type AgentInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Command     string `json:"command"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
}

// ConnectionInfo is a snapshot of a live agent connection.
type ConnectionInfo struct {
	ID          string   `json:"id"`
	AgentName   string   `json:"agentName"`
	DisplayName string   `json:"displayName"`
	Sessions    []string `json:"sessions"`
}

// SessionHistoryInfo carries the full conversation history for one session.
type SessionHistoryInfo struct {
	ID           string         `json:"id"`
	AgentName    string         `json:"agentName"`
	ConnectionID string         `json:"connectionId"`
	CWD          string         `json:"cwd"`
	Messages     []MessageInfo  `json:"messages"`
	ToolCalls    []ToolCallInfo `json:"toolCalls"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
}

// MessageInfo is a single message in a session's conversation.
type MessageInfo struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// ToolCallInfo is a single tool invocation record.
type ToolCallInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// SessionListItem is a lightweight summary for the session list view.
type SessionListItem struct {
	ID           string `json:"id"`
	AgentName    string `json:"agentName"`
	ConnectionID string `json:"connectionId"`
	CWD          string `json:"cwd"`
	MessageCount int    `json:"messageCount"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// AppSettingsInfo mirrors agent.AppSettings for frontend consumption.
type AppSettingsInfo struct {
	Theme        string `json:"theme"`
	DefaultAgent string `json:"defaultAgent"`
	DefaultCWD   string `json:"defaultCwd"`
	AutoApprove  bool   `json:"autoApprove"`
}

// FileEntry represents a single file or directory for the file explorer.
type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
}

// PermissionRequestInfo is emitted to the frontend when an agent asks for
// permission before performing a sensitive operation.
type PermissionRequestInfo struct {
	ConnectionID string                 `json:"connectionId"`
	SessionID    string                 `json:"sessionId"`
	ToolCallID   string                 `json:"toolCallId"`
	Title        string                 `json:"title"`
	Kind         string                 `json:"kind"`
	Options      []PermissionOptionInfo `json:"options"`
}

// PermissionOptionInfo is one choice in a permission dialog.
type PermissionOptionInfo struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

// ---------------------------------------------------------------------------
// App – the main Wails-bound struct
// ---------------------------------------------------------------------------

// App is the primary backend struct whose exported methods are exposed to the
// frontend as TypeScript bindings. It orchestrates agent connections, session
// management, file system and terminal providers, and pushes real-time
// updates to the frontend via Wails runtime events.
type App struct {
	ctx context.Context

	manager  *agent.Manager
	config   *agent.Config
	fs       *bfs.Provider
	terminal *terminal.Provider
	sessions *session.Store

	// pendingPermissions stores channels keyed by connectionID. When the
	// agent sends a requestPermission request the handler creates a channel,
	// emits an event to the UI, and blocks. The UI calls RespondPermission
	// which delivers the chosen optionID through the channel.
	pendingPermissions   map[string]chan string
	pendingPermissionsMu sync.Mutex

	// activePrompts tracks running prompt goroutines so CancelPrompt can
	// both cancel the context and send the ACP cancel notification.
	activePrompts   map[string]context.CancelFunc
	activePromptsMu sync.Mutex

	configPath string
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{
		pendingPermissions: make(map[string]chan string),
		activePrompts:      make(map[string]context.CancelFunc),
	}
}

// startup is called by Wails when the application starts. It initialises
// configuration, the agent manager, and all providers.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load or create configuration.
	a.configPath = agent.ConfigPath()
	cfg, err := agent.LoadConfig(a.configPath)
	if err != nil {
		log.Printf("bytesmith: failed to load config, using defaults: %v", err)
		cfg = agent.DefaultConfig()
	}
	a.config = cfg

	// Initialise subsystems.
	a.manager = agent.NewManager(a.config)
	a.fs = bfs.NewProvider()
	a.terminal = terminal.NewProvider()
	a.sessions = session.NewStore()

	// Forward file-change events to the frontend.
	a.fs.OnFileChanged(func(change bfs.FileChange) {
		wailsRuntime.EventsEmit(a.ctx, "file:changed", map[string]string{
			"path":      change.Path,
			"sessionId": change.SessionID,
			"agentName": change.AgentName,
		})
	})

	// Forward terminal output events to the frontend.
	a.terminal.OnOutput(func(terminalID string, data string) {
		wailsRuntime.EventsEmit(a.ctx, "terminal:output", map[string]string{
			"terminalId": terminalID,
			"data":       data,
		})
	})
}

// shutdown is called by Wails when the application is closing. It tears down
// all terminals and agent connections.
func (a *App) shutdown(ctx context.Context) {
	a.terminal.CloseAll()
	a.manager.DisconnectAll()
}

// ---------------------------------------------------------------------------
// Agent management
// ---------------------------------------------------------------------------

// ListAvailableAgents returns configured agents merged with well-known agents,
// annotated with whether they are installed on the system.
func (a *App) ListAvailableAgents() []AgentInfo {
	seen := make(map[string]bool)
	var result []AgentInfo

	// Configured agents first.
	for _, ac := range a.config.Agents {
		seen[ac.Name] = true
		result = append(result, AgentInfo{
			Name:        ac.Name,
			DisplayName: ac.DisplayName,
			Command:     ac.Command,
			Description: ac.Description,
			Installed:   agent.IsInstalled(ac.Command),
		})
	}

	// Well-known agents that aren't already configured.
	for _, ac := range agent.WellKnownAgents() {
		if !seen[ac.Name] {
			result = append(result, AgentInfo{
				Name:        ac.Name,
				DisplayName: ac.DisplayName,
				Command:     ac.Command,
				Description: ac.Description,
				Installed:   agent.IsInstalled(ac.Command),
			})
		}
	}

	return result
}

// ListInstalledAgents returns only agents whose binary is found in PATH.
func (a *App) ListInstalledAgents() []AgentInfo {
	detected := agent.DetectInstalled()
	result := make([]AgentInfo, 0, len(detected))
	for _, ac := range detected {
		result = append(result, AgentInfo{
			Name:        ac.Name,
			DisplayName: ac.DisplayName,
			Command:     ac.Command,
			Description: ac.Description,
			Installed:   true,
		})
	}
	return result
}

// ConnectAgent starts an agent subprocess, performs the ACP handshake, wires
// up all callbacks, and returns the connection ID.
func (a *App) ConnectAgent(agentName, cwd string) (string, error) {
	conn, err := a.manager.Connect(agentName, cwd)
	if err != nil {
		return "", err
	}

	a.wireConnection(conn)
	return conn.ID, nil
}

// DisconnectAgent gracefully shuts down a connection by ID.
func (a *App) DisconnectAgent(connectionID string) error {
	return a.manager.Disconnect(connectionID)
}

// ListConnections returns a snapshot of all active agent connections.
func (a *App) ListConnections() []ConnectionInfo {
	conns := a.manager.ListConnections()
	result := make([]ConnectionInfo, 0, len(conns))
	for _, c := range conns {
		sessions := make([]string, len(c.Sessions))
		copy(sessions, c.Sessions)
		result = append(result, ConnectionInfo{
			ID:          c.ID,
			AgentName:   c.Agent.Name,
			DisplayName: c.Agent.DisplayName,
			Sessions:    sessions,
		})
	}
	return result
}

// ---------------------------------------------------------------------------
// Session management
// ---------------------------------------------------------------------------

// NewSession creates a new session on an existing agent connection and returns
// the session ID.
func (a *App) NewSession(connectionID, cwd string) (string, error) {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return "", fmt.Errorf("connection %q not found", connectionID)
	}

	sessionID, err := conn.Client.NewSession(context.Background(), cwd, nil)
	if err != nil {
		return "", err
	}

	// Track session locally.
	a.sessions.Create(sessionID, conn.Agent.Name, connectionID, cwd)
	conn.Sessions = append(conn.Sessions, sessionID)

	return sessionID, nil
}

// SendPrompt sends a user prompt to the agent asynchronously. Real-time
// updates arrive via Wails events ("agent:message", "agent:toolcall", etc.).
// When the agent finishes, an "agent:prompt-done" event is emitted.
func (a *App) SendPrompt(connectionID, sessionID, text string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}

	// Record the user message.
	a.sessions.AddMessage(sessionID, session.Message{
		Role:    "user",
		Content: text,
	})

	go func() {
		// A prompt can take a very long time; use a generous timeout.
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
		defer cancel()

		a.activePromptsMu.Lock()
		a.activePrompts[sessionID] = cancel
		a.activePromptsMu.Unlock()

		defer func() {
			a.activePromptsMu.Lock()
			delete(a.activePrompts, sessionID)
			a.activePromptsMu.Unlock()
		}()

		prompt := []acp.ContentBlock{
			{Type: "text", Text: text},
		}

		result, err := conn.Client.Prompt(ctx, sessionID, prompt)
		if err != nil {
			wailsRuntime.EventsEmit(a.ctx, "agent:error", map[string]string{
				"connectionId": connectionID,
				"sessionId":    sessionID,
				"error":        err.Error(),
			})
			return
		}

		wailsRuntime.EventsEmit(a.ctx, "agent:prompt-done", map[string]string{
			"connectionId": connectionID,
			"sessionId":    sessionID,
			"stopReason":   result.StopReason,
		})
	}()

	return nil
}

// CancelPrompt cancels an in-progress prompt by sending the ACP cancel
// notification and aborting the local context.
func (a *App) CancelPrompt(connectionID, sessionID string) error {
	// Cancel the local context so the Prompt call unblocks.
	a.activePromptsMu.Lock()
	cancel, ok := a.activePrompts[sessionID]
	a.activePromptsMu.Unlock()
	if ok {
		cancel()
	}

	// Also tell the agent to stop.
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}
	return conn.Client.Cancel(sessionID)
}

// GetSessionHistory returns the full conversation history for a session.
func (a *App) GetSessionHistory(sessionID string) *SessionHistoryInfo {
	rec := a.sessions.Get(sessionID)
	if rec == nil {
		return nil
	}

	messages := make([]MessageInfo, 0, len(rec.Messages))
	for _, m := range rec.Messages {
		messages = append(messages, MessageInfo{
			Role:      m.Role,
			Content:   m.Content,
			Timestamp: m.Timestamp.Format(time.RFC3339),
		})
	}

	toolCalls := make([]ToolCallInfo, 0, len(rec.ToolCalls))
	for _, tc := range rec.ToolCalls {
		toolCalls = append(toolCalls, ToolCallInfo{
			ID:        tc.ID,
			Title:     tc.Title,
			Kind:      tc.Kind,
			Status:    tc.Status,
			Content:   tc.Content,
			Timestamp: tc.Timestamp.Format(time.RFC3339),
		})
	}

	return &SessionHistoryInfo{
		ID:           rec.ID,
		AgentName:    rec.AgentName,
		ConnectionID: rec.ConnectionID,
		CWD:          rec.CWD,
		Messages:     messages,
		ToolCalls:    toolCalls,
		CreatedAt:    rec.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    rec.UpdatedAt.Format(time.RFC3339),
	}
}

// ListSessions returns lightweight summaries for all sessions.
func (a *App) ListSessions() []SessionListItem {
	records := a.sessions.List()
	result := make([]SessionListItem, 0, len(records))
	for _, r := range records {
		result = append(result, SessionListItem{
			ID:           r.ID,
			AgentName:    r.AgentName,
			ConnectionID: r.ConnectionID,
			CWD:          r.CWD,
			MessageCount: len(r.Messages),
			CreatedAt:    r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result
}

// ---------------------------------------------------------------------------
// Permission handling
// ---------------------------------------------------------------------------

// RespondPermission is called by the UI when the user clicks allow/deny on a
// permission dialog. It unblocks the ACP requestPermission handler which is
// waiting for the user's decision.
func (a *App) RespondPermission(connectionID string, optionID string) {
	a.pendingPermissionsMu.Lock()
	ch, ok := a.pendingPermissions[connectionID]
	a.pendingPermissionsMu.Unlock()

	if ok {
		ch <- optionID
	}
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

// GetSettings returns the current application settings.
func (a *App) GetSettings() AppSettingsInfo {
	return AppSettingsInfo{
		Theme:        a.config.Settings.Theme,
		DefaultAgent: a.config.Settings.DefaultAgent,
		DefaultCWD:   a.config.Settings.DefaultCWD,
		AutoApprove:  a.config.Settings.AutoApprove,
	}
}

// SaveSettings persists new application settings to the config file.
func (a *App) SaveSettings(settings AppSettingsInfo) error {
	a.config.Settings = agent.AppSettings{
		Theme:        settings.Theme,
		DefaultAgent: settings.DefaultAgent,
		DefaultCWD:   settings.DefaultCWD,
		AutoApprove:  settings.AutoApprove,
	}
	return agent.SaveConfig(a.configPath, a.config)
}

// ---------------------------------------------------------------------------
// File system
// ---------------------------------------------------------------------------

// SelectDirectory opens the native directory picker dialog and returns the
// selected path, or an empty string if the user cancelled.
func (a *App) SelectDirectory() (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Directory",
	})
}

// ListFiles returns directory entries sorted with directories first.
func (a *App) ListFiles(dir string) ([]FileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	result := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		var size int64
		if info, infoErr := e.Info(); infoErr == nil {
			size = info.Size()
		}
		result = append(result, FileEntry{
			Name:  e.Name(),
			Path:  filepath.Join(dir, e.Name()),
			IsDir: e.IsDir(),
			Size:  size,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// ---------------------------------------------------------------------------
// Internal: connection wiring
// ---------------------------------------------------------------------------

// wireConnection registers all ACP client callbacks on a newly created
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
		for line := range conn.Transport.StderrCh() {
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

// handleSessionUpdate dispatches an incoming ACP session update notification
// to the appropriate Wails event and records data in the session store.
func (a *App) handleSessionUpdate(connectionID string, params acp.SessionUpdateParams) {
	update := params.Update
	sid := params.SessionID

	switch update.Type {
	case acp.UpdateAgentMessageChunk:
		if update.MessageContent != nil {
			a.sessions.AddMessage(sid, session.Message{
				Role:    "agent",
				Content: update.MessageContent.Text,
			})
			wailsRuntime.EventsEmit(a.ctx, "agent:message", map[string]string{
				"connectionId": connectionID,
				"sessionId":    sid,
				"text":         update.MessageContent.Text,
				"type":         update.MessageContent.Type,
			})
		}

	case acp.UpdateToolCall:
		a.sessions.AddToolCall(sid, session.ToolCallRecord{
			ID:     update.ToolCallID,
			Title:  update.Title,
			Kind:   update.Kind,
			Status: update.Status,
		})
		wailsRuntime.EventsEmit(a.ctx, "agent:toolcall", map[string]interface{}{
			"connectionId": connectionID,
			"sessionId":    sid,
			"toolCallId":   update.ToolCallID,
			"title":        update.Title,
			"kind":         update.Kind,
			"status":       update.Status,
			"isUpdate":     false,
		})

	case acp.UpdateToolCallUpdate:
		a.sessions.UpdateToolCall(sid, update.ToolCallID, update.Status, "")
		wailsRuntime.EventsEmit(a.ctx, "agent:toolcall", map[string]interface{}{
			"connectionId": connectionID,
			"sessionId":    sid,
			"toolCallId":   update.ToolCallID,
			"title":        update.Title,
			"kind":         update.Kind,
			"status":       update.Status,
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

// ---------------------------------------------------------------------------
// Internal: permission request handling
// ---------------------------------------------------------------------------

// handlePermissionRequest is called synchronously by the ACP client when the
// agent asks for user permission. It emits an event to the UI and blocks
// until RespondPermission is called.
func (a *App) handlePermissionRequest(connectionID string, params acp.RequestPermissionParams) acp.RequestPermissionResult {
	ch := make(chan string, 1)

	a.pendingPermissionsMu.Lock()
	a.pendingPermissions[connectionID] = ch
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
	delete(a.pendingPermissions, connectionID)
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
