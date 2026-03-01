package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"bytesmith/internal/acp"
	"bytesmith/internal/agent"
	bfs "bytesmith/internal/fs"
	"bytesmith/internal/integrator"
	"bytesmith/internal/session"
	"bytesmith/internal/terminal"

	"github.com/google/uuid"
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
	Integrator  string   `json:"integrator"`
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
	ID        string `json:"id"`
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
	RequestID    string                 `json:"requestId"`
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

// SessionModelInfo is one available model option for a session.
type SessionModelInfo struct {
	ModelID string `json:"modelId"`
	Name    string `json:"name"`
}

// SessionModelsInfo contains model selection state for a session.
type SessionModelsInfo struct {
	CurrentModelID string             `json:"currentModelId"`
	Models         []SessionModelInfo `json:"models"`
}

// SessionListPage is a page of remote sessions queried from an integrator.
type SessionListPage struct {
	Sessions    []SessionListItem `json:"sessions"`
	NextCursor  string            `json:"nextCursor,omitempty"`
	Unsupported bool              `json:"unsupported,omitempty"`
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
	sessions session.Store

	// sessionModels stores model options returned by session/new per session.
	sessionModels   map[string]SessionModelsInfo
	sessionModelsMu sync.RWMutex

	// pendingPermissions stores channels keyed by requestID.
	// pendingPermissionOrder stores request IDs FIFO by session+toolCall.
	pendingPermissions     map[string]chan string
	pendingPermissionOrder map[string][]string
	pendingPermissionsMu   sync.Mutex

	// activePrompts tracks running prompt goroutines so CancelPrompt can
	// both cancel the context and send the ACP cancel notification.
	activePrompts   map[string]context.CancelFunc
	activePromptsMu sync.Mutex

	// streamMessages aggregates streaming chunks so each turn is stored as a
	// single final agent message.
	streamMessages   map[string]*streamMessage
	streamMessagesMu sync.Mutex

	configPath string
}

type streamMessage struct {
	MessageID   string
	ContentType string
	Content     strings.Builder
	StartedAt   time.Time
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{
		pendingPermissions:     make(map[string]chan string),
		pendingPermissionOrder: make(map[string][]string),
		activePrompts:          make(map[string]context.CancelFunc),
		sessionModels:          make(map[string]SessionModelsInfo),
		streamMessages:         make(map[string]*streamMessage),
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
	if a.sessions != nil {
		_ = a.sessions.Close()
	}
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
			Integrator:  c.IntegratorID,
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

	result, err := conn.Client.NewSession(context.Background(), cwd, nil)
	if err != nil {
		return "", err
	}
	sessionID := result.SessionID

	// Track session locally.
	a.sessions.Create(sessionID, conn.Agent.Name, connectionID, cwd)
	appendSessionIfMissing(conn, sessionID)

	if result.Models != nil {
		models := make([]SessionModelInfo, 0, len(result.Models.AvailableModels))
		for _, m := range result.Models.AvailableModels {
			models = append(models, SessionModelInfo{
				ModelID: m.ModelID,
				Name:    m.Name,
			})
		}

		info := SessionModelsInfo{
			CurrentModelID: result.Models.CurrentModelID,
			Models:         models,
		}

		a.sessionModelsMu.Lock()
		a.sessionModels[sessionID] = info
		a.sessionModelsMu.Unlock()

		wailsRuntime.EventsEmit(a.ctx, "agent:models", map[string]interface{}{
			"connectionId":   connectionID,
			"sessionId":      sessionID,
			"currentModelId": info.CurrentModelID,
			"models":         info.Models,
		})
	}

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
		ID:      uuid.NewString(),
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
			a.finalizeStreamMessage(connectionID, sessionID)
			wailsRuntime.EventsEmit(a.ctx, "agent:error", map[string]string{
				"connectionId": connectionID,
				"sessionId":    sessionID,
				"error":        err.Error(),
			})
			return
		}

		a.finalizeStreamMessage(connectionID, sessionID)
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
			ID:        m.ID,
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

// ListRemoteSessions lists sessions directly from the connected integrator.
func (a *App) ListRemoteSessions(connectionID, cwd, cursor string) (SessionListPage, error) {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return SessionListPage{}, fmt.Errorf("connection %q not found", connectionID)
	}

	if !integrator.ForAgent(conn.Agent.Name).Capabilities().ListSessions {
		return SessionListPage{Unsupported: true}, nil
	}

	list, err := conn.Client.ListSessions(context.Background(), cwd, cursor)
	if err != nil {
		if strings.Contains(err.Error(), "method not found") || strings.Contains(err.Error(), "unknown variant") {
			return SessionListPage{Unsupported: true}, nil
		}
		return SessionListPage{}, err
	}

	sessions := make([]SessionListItem, 0, len(list.Sessions))
	for _, s := range list.Sessions {
		sessions = append(sessions, SessionListItem{
			ID:           s.SessionID,
			AgentName:    conn.Agent.Name,
			ConnectionID: connectionID,
			CWD:          s.CWD,
			CreatedAt:    "",
			UpdatedAt:    s.UpdatedAt,
		})
	}

	return SessionListPage{
		Sessions:   sessions,
		NextCursor: list.NextCursor,
	}, nil
}

// LoadRemoteSession asks the remote agent to load a session and tracks it locally.
func (a *App) LoadRemoteSession(connectionID, sessionID, cwd string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}

	if !integrator.ForAgent(conn.Agent.Name).Capabilities().LoadSession {
		return fmt.Errorf("integrator %q does not support session load", conn.Agent.Name)
	}

	if err := conn.Client.LoadSession(context.Background(), sessionID, cwd, nil); err != nil {
		return err
	}

	a.sessions.Create(sessionID, conn.Agent.Name, connectionID, cwd)
	appendSessionIfMissing(conn, sessionID)
	return nil
}

// ResumeSession asks the remote agent to resume a session and tracks it locally.
func (a *App) ResumeSession(connectionID, sessionID, cwd string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}

	caps := integrator.ForAgent(conn.Agent.Name).Capabilities()
	if !caps.ResumeSession {
		if !caps.LoadSession {
			return fmt.Errorf("integrator %q does not support session resume", conn.Agent.Name)
		}
		return a.LoadRemoteSession(connectionID, sessionID, cwd)
	}

	result, err := conn.Client.ResumeSession(context.Background(), sessionID, cwd, nil)
	if err != nil {
		return err
	}

	a.sessions.Create(sessionID, conn.Agent.Name, connectionID, cwd)
	appendSessionIfMissing(conn, sessionID)

	if result != nil && result.Models != nil {
		models := make([]SessionModelInfo, 0, len(result.Models.AvailableModels))
		for _, m := range result.Models.AvailableModels {
			models = append(models, SessionModelInfo{
				ModelID: m.ModelID,
				Name:    m.Name,
			})
		}
		a.sessionModelsMu.Lock()
		a.sessionModels[sessionID] = SessionModelsInfo{
			CurrentModelID: result.Models.CurrentModelID,
			Models:         models,
		}
		a.sessionModelsMu.Unlock()
	}

	return nil
}

// GetSessionModels returns the known model selection info for a session.
func (a *App) GetSessionModels(sessionID string) *SessionModelsInfo {
	a.sessionModelsMu.RLock()
	defer a.sessionModelsMu.RUnlock()

	info, ok := a.sessionModels[sessionID]
	if !ok {
		return nil
	}

	copyModels := make([]SessionModelInfo, len(info.Models))
	copy(copyModels, info.Models)

	result := info
	result.Models = copyModels
	return &result
}

// SetSessionModel updates the current model for a session.
func (a *App) SetSessionModel(connectionID, sessionID, modelID string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}

	if err := conn.Client.SetModel(context.Background(), sessionID, modelID); err != nil {
		return err
	}

	a.sessionModelsMu.Lock()
	info, ok := a.sessionModels[sessionID]
	if ok {
		info.CurrentModelID = modelID
		a.sessionModels[sessionID] = info
	}
	a.sessionModelsMu.Unlock()

	if ok {
		wailsRuntime.EventsEmit(a.ctx, "agent:models", map[string]interface{}{
			"connectionId":   connectionID,
			"sessionId":      sessionID,
			"currentModelId": modelID,
			"models":         info.Models,
		})
	}

	return nil
}

// SetSessionMode updates the current mode for a session.
func (a *App) SetSessionMode(connectionID, sessionID, modeID string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}
	return conn.Client.SetMode(context.Background(), sessionID, modeID)
}

// SetSessionConfigOption sets a generic session config option.
func (a *App) SetSessionConfigOption(connectionID, sessionID, configID, value string) error {
	conn := a.manager.GetConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connectionID)
	}
	return conn.Client.SetConfigOption(context.Background(), sessionID, configID, value)
}

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
		content := formatToolCallContent(update)
		a.sessions.AddToolCall(sid, session.ToolCallRecord{
			ID:      update.ToolCallID,
			Title:   update.Title,
			Kind:    update.Kind,
			Status:  update.Status,
			Content: content,
		})
		wailsRuntime.EventsEmit(a.ctx, "agent:toolcall", map[string]interface{}{
			"connectionId": connectionID,
			"sessionId":    sid,
			"toolCallId":   update.ToolCallID,
			"title":        update.Title,
			"kind":         update.Kind,
			"status":       update.Status,
			"content":      content,
			"isUpdate":     false,
		})

	case acp.UpdateToolCallUpdate:
		content := formatToolCallContent(update)
		a.sessions.UpdateToolCall(sid, update.ToolCallID, update.Status, content)
		wailsRuntime.EventsEmit(a.ctx, "agent:toolcall", map[string]interface{}{
			"connectionId": connectionID,
			"sessionId":    sid,
			"toolCallId":   update.ToolCallID,
			"title":        update.Title,
			"kind":         update.Kind,
			"status":       update.Status,
			"content":      content,
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

func normalizeMessageType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "thought", "reasoning":
		return "thought"
	default:
		return "text"
	}
}

func prettyJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return strings.TrimSpace(string(raw))
	}

	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return strings.TrimSpace(string(raw))
	}
	return string(formatted)
}

func formatToolCallContent(update acp.SessionUpdate) string {
	sections := make([]string, 0, 6)

	for _, part := range update.ToolContent {
		switch part.Type {
		case "content":
			if part.Content != nil && strings.TrimSpace(part.Content.Text) != "" {
				sections = append(sections, "Content:\n"+part.Content.Text)
			}
		case "diff":
			var b strings.Builder
			if strings.TrimSpace(part.Path) != "" {
				b.WriteString("Diff: " + part.Path + "\n")
			} else {
				b.WriteString("Diff:\n")
			}
			if part.OldText != "" || part.NewText != "" {
				b.WriteString("--- old\n")
				b.WriteString(part.OldText)
				if !strings.HasSuffix(part.OldText, "\n") {
					b.WriteString("\n")
				}
				b.WriteString("+++ new\n")
				b.WriteString(part.NewText)
			}
			diff := strings.TrimSpace(b.String())
			if diff != "" {
				sections = append(sections, diff)
			}
		case "terminal":
			terminalText := ""
			if part.Content != nil {
				terminalText = part.Content.Text
			}
			switch {
			case strings.TrimSpace(part.TerminalID) != "" && strings.TrimSpace(terminalText) != "":
				sections = append(sections, fmt.Sprintf("Terminal (%s):\n%s", part.TerminalID, terminalText))
			case strings.TrimSpace(part.TerminalID) != "":
				sections = append(sections, fmt.Sprintf("Terminal: %s", part.TerminalID))
			case strings.TrimSpace(terminalText) != "":
				sections = append(sections, "Terminal:\n"+terminalText)
			}
		default:
			if part.Content != nil && strings.TrimSpace(part.Content.Text) != "" {
				sections = append(sections, part.Content.Text)
			}
		}
	}

	if len(update.Locations) > 0 {
		lines := make([]string, 0, len(update.Locations))
		for _, loc := range update.Locations {
			if loc.Line > 0 {
				lines = append(lines, fmt.Sprintf("- %s:%d", loc.Path, loc.Line))
			} else {
				lines = append(lines, fmt.Sprintf("- %s", loc.Path))
			}
		}
		sections = append(sections, "Locations:\n"+strings.Join(lines, "\n"))
	}

	if input := prettyJSON(update.RawInput); input != "" {
		sections = append(sections, "Input:\n"+input)
	}
	if output := prettyJSON(update.RawOutput); output != "" {
		sections = append(sections, "Output:\n"+output)
	}

	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}

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

func appendSessionIfMissing(conn *agent.Connection, sessionID string) {
	for _, existing := range conn.Sessions {
		if existing == sessionID {
			return
		}
	}
	conn.Sessions = append(conn.Sessions, sessionID)
}
