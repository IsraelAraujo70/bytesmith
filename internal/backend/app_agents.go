package backend

import (
	"bytesmith/internal/agent"
)

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

func appendSessionIfMissing(conn *agent.Connection, sessionID string) {
	for _, existing := range conn.Sessions {
		if existing == sessionID {
			return
		}
	}
	conn.Sessions = append(conn.Sessions, sessionID)
}

func findConnectionByAgent(conns []*agent.Connection, agentName string) *agent.Connection {
	for _, conn := range conns {
		if conn.Agent.Name == agentName {
			return conn
		}
	}
	return nil
}
