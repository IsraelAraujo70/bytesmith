package agentclient

import (
	"context"

	"bytesmith/internal/acp"
)

// Client is the protocol/runtime abstraction consumed by the backend.
// It intentionally mirrors the subset currently used by App methods.
type Client interface {
	Close() error
	StderrCh() <-chan string

	NewSession(ctx context.Context, cwd string, mcpServers []acp.MCPServer) (*acp.SessionNewResult, error)
	LoadSession(ctx context.Context, sessionID, cwd string, mcpServers []acp.MCPServer) error
	ResumeSession(ctx context.Context, sessionID, cwd string, mcpServers []acp.MCPServer) (*acp.SessionResumeResult, error)
	ListSessions(ctx context.Context, cwd, cursor string) (*acp.SessionListResult, error)
	Prompt(ctx context.Context, sessionID string, prompt []acp.ContentBlock) (*acp.SessionPromptResult, error)
	Cancel(sessionID string) error
	SetMode(ctx context.Context, sessionID, mode string) error
	SetModel(ctx context.Context, sessionID, modelID string) error
	SetConfigOption(ctx context.Context, sessionID, configID, value string) error

	OnSessionUpdate(handler func(acp.SessionUpdateParams))
	OnRequestPermission(handler func(acp.RequestPermissionParams) acp.RequestPermissionResult)
	OnFSReadTextFile(handler func(acp.FSReadTextFileParams) (*acp.FSReadTextFileResult, error))
	OnFSWriteTextFile(handler func(acp.FSWriteTextFileParams) error)
	OnTerminalCreate(handler func(acp.TerminalCreateParams) (*acp.TerminalCreateResult, error))
	OnTerminalOutput(handler func(acp.TerminalOutputParams) (*acp.TerminalOutputResult, error))
	OnTerminalWait(handler func(acp.TerminalWaitParams) (*acp.TerminalWaitResult, error))
	OnTerminalKill(handler func(acp.TerminalKillParams) error)
	OnTerminalRelease(handler func(acp.TerminalReleaseParams) error)
}

func closedStringChannel() <-chan string {
	ch := make(chan string)
	close(ch)
	return ch
}
