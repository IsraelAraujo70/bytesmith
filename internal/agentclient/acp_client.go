package agentclient

import (
	"context"
	"fmt"

	"bytesmith/internal/acp"
)

// ACPClient is a thin adapter over internal/acp.Client.
type ACPClient struct {
	client    *acp.Client
	transport *acp.StdioTransport
}

var _ Client = (*ACPClient)(nil)

func NewACP(command string, args []string, env []string, cwd string) (*ACPClient, error) {
	transport := acp.NewStdioTransport(command, args, env, cwd)
	client := acp.NewClient(transport)
	if _, err := client.Initialize(context.Background()); err != nil {
		_ = transport.Close()
		return nil, fmt.Errorf("initialize acp client: %w", err)
	}
	return &ACPClient{
		client:    client,
		transport: transport,
	}, nil
}

func (c *ACPClient) Close() error {
	return c.client.Close()
}

func (c *ACPClient) StderrCh() <-chan string {
	return c.transport.StderrCh()
}

func (c *ACPClient) NewSession(ctx context.Context, cwd string, mcpServers []acp.MCPServer) (*acp.SessionNewResult, error) {
	return c.client.NewSession(ctx, cwd, mcpServers)
}

func (c *ACPClient) LoadSession(ctx context.Context, sessionID, cwd string, mcpServers []acp.MCPServer) error {
	return c.client.LoadSession(ctx, sessionID, cwd, mcpServers)
}

func (c *ACPClient) ResumeSession(ctx context.Context, sessionID, cwd string, mcpServers []acp.MCPServer) (*acp.SessionResumeResult, error) {
	return c.client.ResumeSession(ctx, sessionID, cwd, mcpServers)
}

func (c *ACPClient) ListSessions(ctx context.Context, cwd, cursor string) (*acp.SessionListResult, error) {
	return c.client.ListSessions(ctx, cwd, cursor)
}

func (c *ACPClient) Prompt(ctx context.Context, sessionID string, prompt []acp.ContentBlock) (*acp.SessionPromptResult, error) {
	return c.client.Prompt(ctx, sessionID, prompt)
}

func (c *ACPClient) Cancel(sessionID string) error {
	return c.client.Cancel(sessionID)
}

func (c *ACPClient) SetMode(ctx context.Context, sessionID, mode string) error {
	return c.client.SetMode(ctx, sessionID, mode)
}

func (c *ACPClient) SetAccessMode(ctx context.Context, sessionID, mode string) error {
	return c.client.SetAccessMode(ctx, sessionID, mode)
}

func (c *ACPClient) SetModel(ctx context.Context, sessionID, modelID string) error {
	return c.client.SetModel(ctx, sessionID, modelID)
}

func (c *ACPClient) SetConfigOption(ctx context.Context, sessionID, configID, value string) error {
	return c.client.SetConfigOption(ctx, sessionID, configID, value)
}

func (c *ACPClient) OnSessionUpdate(handler func(acp.SessionUpdateParams)) {
	c.client.OnSessionUpdate(handler)
}

func (c *ACPClient) OnRequestPermission(handler func(acp.RequestPermissionParams) acp.RequestPermissionResult) {
	c.client.OnRequestPermission(handler)
}

func (c *ACPClient) OnRequestUserInput(handler func(acp.ToolRequestUserInputParams) acp.ToolRequestUserInputResponse) {
	c.client.OnRequestUserInput(handler)
}

func (c *ACPClient) OnFSReadTextFile(handler func(acp.FSReadTextFileParams) (*acp.FSReadTextFileResult, error)) {
	c.client.OnFSReadTextFile(handler)
}

func (c *ACPClient) OnFSWriteTextFile(handler func(acp.FSWriteTextFileParams) error) {
	c.client.OnFSWriteTextFile(handler)
}

func (c *ACPClient) OnTerminalCreate(handler func(acp.TerminalCreateParams) (*acp.TerminalCreateResult, error)) {
	c.client.OnTerminalCreate(handler)
}

func (c *ACPClient) OnTerminalOutput(handler func(acp.TerminalOutputParams) (*acp.TerminalOutputResult, error)) {
	c.client.OnTerminalOutput(handler)
}

func (c *ACPClient) OnTerminalWait(handler func(acp.TerminalWaitParams) (*acp.TerminalWaitResult, error)) {
	c.client.OnTerminalWait(handler)
}

func (c *ACPClient) OnTerminalKill(handler func(acp.TerminalKillParams) error) {
	c.client.OnTerminalKill(handler)
}

func (c *ACPClient) OnTerminalRelease(handler func(acp.TerminalReleaseParams) error) {
	c.client.OnTerminalRelease(handler)
}
