package agent

import (
	"os/exec"
)

// wellKnownAgent is a compile-time table entry for a supported agent runtime.
type wellKnownAgent struct {
	Name        string
	DisplayName string
	Command     string
	Args        []string
	Description string
}

// wellKnownAgents is the canonical list of supported agent runtimes.
var wellKnownAgents = []wellKnownAgent{
	{
		Name:        "opencode",
		DisplayName: "OpenCode",
		Command:     "opencode",
		Args:        []string{},
		Description: "OpenCode local server",
	},
	{
		Name:        "codex-app-server",
		DisplayName: "Codex App Server",
		Command:     "codex",
		Args:        []string{"app-server"},
		Description: "OpenAI Codex app-server",
	},
}

// WellKnownAgents returns AgentConfig entries for every known agent runtime,
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
