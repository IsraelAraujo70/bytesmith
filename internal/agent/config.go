package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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
				Name:        "codex-acp",
				DisplayName: "Codex CLI",
				Command:     "codex-acp",
				Args:        []string{},
				Description: "OpenAI Codex CLI with ACP support",
				AutoDetect:  true,
			},
			{
				Name:        "gemini",
				DisplayName: "Gemini CLI",
				Command:     "gemini",
				Args:        []string{"--acp"},
				Description: "Google Gemini CLI with ACP support",
				AutoDetect:  true,
			},
			{
				Name:        "claude-code-acp",
				DisplayName: "Claude Code",
				Command:     "claude-code-acp",
				Args:        []string{},
				Description: "Anthropic Claude Code with ACP support",
				AutoDetect:  true,
			},
			{
				Name:        "goose",
				DisplayName: "Goose",
				Command:     "goose",
				Args:        []string{"--acp"},
				Description: "Block Goose with ACP support",
				AutoDetect:  true,
			},
			{
				Name:        "kiro",
				DisplayName: "Kiro",
				Command:     "kiro",
				Args:        []string{"--acp"},
				Description: "Kiro with ACP support",
				AutoDetect:  true,
			},
			{
				Name:        "augment",
				DisplayName: "Augment",
				Command:     "augment",
				Args:        []string{"acp"},
				Description: "Augment with ACP support",
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
	return &cfg, nil
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
