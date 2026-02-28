package agent

import (
	"os/exec"
)

// wellKnownAgent is a compile-time table entry for an ACP-compatible agent.
type wellKnownAgent struct {
	Name        string
	DisplayName string
	Command     string
	Args        []string
	Description string
}

// wellKnownAgents is the canonical list of known ACP agents.
var wellKnownAgents = []wellKnownAgent{
	{
		Name:        "opencode",
		DisplayName: "OpenCode",
		Command:     "opencode",
		Args:        []string{"acp"},
		Description: "OpenCode ACP agent",
	},
	{
		Name:        "codex-acp",
		DisplayName: "Codex CLI",
		Command:     "codex-acp",
		Args:        []string{},
		Description: "OpenAI Codex CLI with ACP support",
	},
	{
		Name:        "gemini",
		DisplayName: "Gemini CLI",
		Command:     "gemini",
		Args:        []string{"--acp"},
		Description: "Google Gemini CLI with ACP support",
	},
	{
		Name:        "claude-code-acp",
		DisplayName: "Claude Code",
		Command:     "claude-code-acp",
		Args:        []string{},
		Description: "Anthropic Claude Code with ACP support",
	},
	{
		Name:        "goose",
		DisplayName: "Goose",
		Command:     "goose",
		Args:        []string{"--acp"},
		Description: "Block Goose with ACP support",
	},
	{
		Name:        "kiro",
		DisplayName: "Kiro",
		Command:     "kiro",
		Args:        []string{"--acp"},
		Description: "Kiro with ACP support",
	},
	{
		Name:        "augment",
		DisplayName: "Augment",
		Command:     "augment",
		Args:        []string{"acp"},
		Description: "Augment with ACP support",
	},
}

// WellKnownAgents returns AgentConfig entries for every known ACP agent,
// regardless of whether they are installed.
func WellKnownAgents() []AgentConfig {
	configs := make([]AgentConfig, 0, len(wellKnownAgents))
	for _, wk := range wellKnownAgents {
		configs = append(configs, AgentConfig{
			Name:        wk.Name,
			DisplayName: wk.DisplayName,
			Command:     wk.Command,
			Args:        wk.Args,
			Description: wk.Description,
			AutoDetect:  true,
		})
	}
	return configs
}

// IsInstalled reports whether the given command is available in PATH.
func IsInstalled(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// DetectInstalled returns AgentConfig entries for every well-known agent whose
// binary is found in PATH.
func DetectInstalled() []AgentConfig {
	var installed []AgentConfig
	for _, wk := range wellKnownAgents {
		if IsInstalled(wk.Command) {
			installed = append(installed, AgentConfig{
				Name:        wk.Name,
				DisplayName: wk.DisplayName,
				Command:     wk.Command,
				Args:        wk.Args,
				Description: wk.Description,
				AutoDetect:  true,
			})
		}
	}
	return installed
}
