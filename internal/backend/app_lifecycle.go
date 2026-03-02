package backend

import (
	"context"
	"log"

	"bytesmith/internal/agent"
	bfs "bytesmith/internal/fs"
	"bytesmith/internal/session"
	"bytesmith/internal/terminal"
	"bytesmith/internal/uixterm"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{
		pendingPermissions:     make(map[string]chan string),
		pendingPermissionOrder: make(map[string][]string),
		activePrompts:          make(map[string]context.CancelFunc),
		sessionModels:          make(map[string]SessionModelsInfo),
		sessionModes:           make(map[string]SessionModesInfo),
		streamMessages:         make(map[string]*streamMessage),
	}
}

// Startup is called by the Wails adapter when the application starts.
// It initialises configuration, the agent manager, and all providers.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.loadConfig()
	a.initSubsystems()
	a.wireRuntimeEvents()
}

func (a *App) loadConfig() {
	a.configPath = agent.ConfigPath()
	cfg, err := agent.LoadConfig(a.configPath)
	if err != nil {
		log.Printf("bytesmith: failed to load config, using defaults: %v", err)
		cfg = agent.DefaultConfig()
	}
	a.config = cfg
}

func (a *App) initSubsystems() {
	a.manager = agent.NewManager(a.config)
	a.fs = bfs.NewProvider()
	a.terminal = terminal.NewProvider()
	a.uiTerm = uixterm.NewManager()
	a.sessions = session.NewStore()
}

func (a *App) wireRuntimeEvents() {
	a.fs.OnFileChanged(func(change bfs.FileChange) {
		wailsRuntime.EventsEmit(a.ctx, "file:changed", map[string]string{
			"path":      change.Path,
			"sessionId": change.SessionID,
			"agentName": change.AgentName,
		})
	})

	a.terminal.OnOutput(func(terminalID string, data string) {
		wailsRuntime.EventsEmit(a.ctx, "terminal:output", map[string]string{
			"terminalId": terminalID,
			"data":       data,
		})
	})

	a.uiTerm.OnOutput(func(terminalID string, data string) {
		wailsRuntime.EventsEmit(a.ctx, "ui:terminal-output", map[string]string{
			"terminalId": terminalID,
			"data":       data,
		})
	})

	a.uiTerm.OnExit(func(terminalID string, exitCode int) {
		wailsRuntime.EventsEmit(a.ctx, "ui:terminal-exit", map[string]interface{}{
			"terminalId": terminalID,
			"exitCode":   exitCode,
		})
	})
}

// Shutdown is called by the Wails adapter when the application is closing.
// It tears down all terminals and agent connections.
func (a *App) Shutdown(ctx context.Context) {
	_ = ctx
	a.terminal.CloseAll()
	if a.uiTerm != nil {
		a.uiTerm.CloseAll()
	}
	a.manager.DisconnectAll()
	if a.sessions != nil {
		_ = a.sessions.Close()
	}
}
