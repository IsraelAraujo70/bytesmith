package agent

import (
	"fmt"
	"sync"

	"bytesmith/internal/agentclient"
	"bytesmith/internal/integrator"

	"github.com/google/uuid"
)

// Connection represents a live connection to an agent runtime.
type Connection struct {
	ID           string
	Agent        AgentConfig
	Client       agentclient.Client
	Sessions     []string
	IntegratorID string
	release      func()
}

// Manager handles the lifecycle of multiple agent connections.
type Manager struct {
	connections   map[string]*Connection
	config        *Config
	openCodeServe *openCodeServerManager
	mu            sync.RWMutex
}

// NewManager creates a Manager with the given configuration.
func NewManager(config *Config) *Manager {
	return &Manager{
		connections:   make(map[string]*Connection),
		config:        config,
		openCodeServe: newOpenCodeServerManager(),
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

// Connect starts and registers a connection to the selected runtime.
func (m *Manager) Connect(agentName string, cwd string) (*Connection, error) {
	agent, ok := m.findAgent(agentName)
	if !ok {
		return nil, fmt.Errorf("agent: unknown agent %q", agentName)
	}

	var (
		client  agentclient.Client
		release func()
		err     error
	)

	if agent.Name == "opencode" {
		var baseURL string
		baseURL, release, err = m.openCodeServe.acquire(agent.Command, agent.Env)
		if err != nil {
			return nil, err
		}
		client, err = agentclient.NewOpenCode(baseURL, cwd)
		if err != nil {
			if release != nil {
				release()
			}
			return nil, fmt.Errorf("agent: initialize opencode runtime: %w", err)
		}
	} else {
		env := make([]string, 0, len(agent.Env))
		for k, v := range agent.Env {
			env = append(env, k+"="+v)
		}
		client, err = agentclient.NewACP(agent.Command, agent.Args, env, cwd)
		if err != nil {
			return nil, fmt.Errorf("agent: initialize %s: %w", agentName, err)
		}
	}

	conn := &Connection{
		ID:           uuid.New().String(),
		Agent:        agent,
		Client:       client,
		Sessions:     make([]string, 0),
		IntegratorID: integrator.ForAgent(agent.Name).ID(),
		release:      release,
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
	if conn.release != nil {
		conn.release()
	}
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
	m.openCodeServe.shutdown()
}
