package agent

import (
	"context"
	"fmt"
	"sync"

	"bytesmith/internal/acp"

	"github.com/google/uuid"
)

// Connection represents a live connection to an agent subprocess.
type Connection struct {
	ID        string
	Agent     AgentConfig
	Client    *acp.Client
	Transport *acp.StdioTransport
	Sessions  []string
}

// Manager handles the lifecycle of multiple agent connections.
type Manager struct {
	connections map[string]*Connection
	config      *Config
	mu          sync.RWMutex
}

// NewManager creates a Manager with the given configuration.
func NewManager(config *Config) *Manager {
	return &Manager{
		connections: make(map[string]*Connection),
		config:      config,
	}
}

// findAgent looks up an AgentConfig by name.
func (m *Manager) findAgent(name string) (AgentConfig, bool) {
	for _, a := range m.config.Agents {
		if a.Name == name {
			return a, true
		}
	}
	return AgentConfig{}, false
}

// Connect starts an agent subprocess, sets up the ACP transport and client,
// performs the initialize handshake, and registers the connection.
func (m *Manager) Connect(agentName string, cwd string) (*Connection, error) {
	agent, ok := m.findAgent(agentName)
	if !ok {
		return nil, fmt.Errorf("agent: unknown agent %q", agentName)
	}

	// Build environment slice from agent config.
	var env []string
	if len(agent.Env) > 0 {
		for k, v := range agent.Env {
			env = append(env, k+"="+v)
		}
	}

	transport := acp.NewStdioTransport(agent.Command, agent.Args, env, cwd)

	client := acp.NewClient(transport)
	// Initialize starts the transport and performs the ACP handshake.
	if _, err := client.Initialize(context.Background()); err != nil {
		transport.Close()
		return nil, fmt.Errorf("agent: initialize %s: %w", agentName, err)
	}

	conn := &Connection{
		ID:        uuid.New().String(),
		Agent:     agent,
		Client:    client,
		Transport: transport,
		Sessions:  make([]string, 0),
	}

	m.mu.Lock()
	m.connections[conn.ID] = conn
	m.mu.Unlock()

	return conn, nil
}

// Disconnect gracefully shuts down a single connection by ID.
func (m *Manager) Disconnect(connectionID string) error {
	m.mu.Lock()
	conn, ok := m.connections[connectionID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("agent: connection %q not found", connectionID)
	}
	delete(m.connections, connectionID)
	m.mu.Unlock()

	if err := conn.Client.Close(); err != nil {
		return fmt.Errorf("agent: close connection %s: %w", connectionID, err)
	}

	// Wait for the subprocess to exit so we don't leak zombie processes.
	_ = conn.Transport.Process().Wait()
	return nil
}

// GetConnection returns the connection with the given ID, or nil if not found.
func (m *Manager) GetConnection(connectionID string) *Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connections[connectionID]
}

// ListConnections returns a snapshot of all active connections.
func (m *Manager) ListConnections() []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Connection, 0, len(m.connections))
	for _, c := range m.connections {
		result = append(result, c)
	}
	return result
}

// DisconnectAll shuts down every active connection. Errors are silently
// ignored so the method can be used in defer / cleanup paths.
func (m *Manager) DisconnectAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.connections))
	for id := range m.connections {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		_ = m.Disconnect(id)
	}
}
