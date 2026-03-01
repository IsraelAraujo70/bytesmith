package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var deprecatedBuiltInAgents = map[string]struct{}{
	"gemini":          {},
	"claude-code-acp": {},
	"goose":           {},
	"kiro":            {},
	"augment":         {},
}

// AgentConfig represents the configuration for a single agent.
type AgentConfig struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env,omitempty"`
	Description string            `json:"description,omitempty"`
	AutoDetect  bool              `json:"autoDetect"`
}

// Config is the top-level configuration.
type Config struct {
	Agents     []AgentConfig     `json:"agents"`
	MCPServers []MCPServerConfig `json:"mcpServers,omitempty"`
	Settings   AppSettings       `json:"settings"`
}

// MCPServerConfig describes an MCP server that can be launched alongside agents.
type MCPServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// AppSettings holds application-wide preferences.
type AppSettings struct {
	Theme        string `json:"theme"`
	DefaultAgent string `json:"defaultAgent"`
	DefaultCWD   string `json:"defaultCwd"`
	AutoApprove  bool   `json:"autoApprove"`
}

// ConfigPath returns the default configuration file path
// (~/.config/bytesmith/config.json).
func ConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "bytesmith", "config.json")
}

// DefaultConfig returns a Config populated with well-known ACP agents and
// sensible default settings.
func DefaultConfig() *Config {
	return &Config{
		Agents: []AgentConfig{
			{
				Name:        "opencode",
				DisplayName: "OpenCode",
				Command:     "opencode",
				Args:        []string{"acp"},
				Description: "OpenCode ACP agent",
				AutoDetect:  true,
			},
			{
				Name:        "codex-app-server",
				DisplayName: "Codex App Server",
				Command:     "codex",
				Args:        []string{"app-server"},
				Description: "OpenAI Codex app-server",
				AutoDetect:  true,
			},
		},
		Settings: AppSettings{
			Theme:        "dark",
			DefaultAgent: "opencode",
			DefaultCWD:   "",
			AutoApprove:  false,
		},
	}
}

// LoadConfig reads the configuration from the given path. If the file does not
// exist, a default configuration is created, written to disk, and returned.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if writeErr := SaveConfig(path, cfg); writeErr != nil {
				return nil, fmt.Errorf("agent: create default config: %w", writeErr)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("agent: read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("agent: parse config: %w", err)
	}

	changed := migrateCodexACPEntries(&cfg)
	if removeDeprecatedAgents(&cfg) {
		changed = true
	}
	if len(cfg.Agents) == 0 {
		cfg.Agents = DefaultConfig().Agents
		changed = true
	}
	if ensureValidDefaultAgent(&cfg) {
		changed = true
	}

	if changed {
		if err := SaveConfig(path, &cfg); err != nil {
			return nil, fmt.Errorf("agent: migrate config: %w", err)
		}
	}
	return &cfg, nil
}

func migrateCodexACPEntries(cfg *Config) bool {
	changed := false
	for i := range cfg.Agents {
		if cfg.Agents[i].Name == "codex-acp" || cfg.Agents[i].Command == "codex-acp" {
			cfg.Agents[i].Name = "codex-app-server"
			cfg.Agents[i].DisplayName = "Codex App Server"
			cfg.Agents[i].Command = "codex"
			cfg.Agents[i].Args = []string{"app-server"}
			cfg.Agents[i].Description = "OpenAI Codex app-server"
			cfg.Agents[i].AutoDetect = true
			changed = true
		}
	}
	return changed
}

func removeDeprecatedAgents(cfg *Config) bool {
	kept := make([]AgentConfig, 0, len(cfg.Agents))
	removedAny := false
	for _, a := range cfg.Agents {
		if _, isDeprecated := deprecatedBuiltInAgents[a.Name]; isDeprecated {
			removedAny = true
			continue
		}
		kept = append(kept, a)
	}
	if removedAny {
		cfg.Agents = kept
	}
	return removedAny
}

func ensureValidDefaultAgent(cfg *Config) bool {
	if cfg.Settings.DefaultAgent == "" {
		cfg.Settings.DefaultAgent = pickFallbackDefaultAgent(cfg.Agents)
		return true
	}
	for _, a := range cfg.Agents {
		if a.Name == cfg.Settings.DefaultAgent {
			return false
		}
	}
	cfg.Settings.DefaultAgent = pickFallbackDefaultAgent(cfg.Agents)
	return true
}

func pickFallbackDefaultAgent(agents []AgentConfig) string {
	for _, a := range agents {
		if a.Name == "opencode" {
			return "opencode"
		}
	}
	for _, a := range agents {
		if a.Name == "codex-app-server" {
			return "codex-app-server"
		}
	}
	if len(agents) > 0 {
		return agents[0].Name
	}
	return "opencode"
}

// SaveConfig writes the configuration to the given path, creating parent
// directories as needed.
func SaveConfig(path string, config *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("agent: create config dir: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("agent: marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("agent: write config: %w", err)
	}
	return nil
}
