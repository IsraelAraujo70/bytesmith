package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigMigratesCodexAndRemovesDeprecated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	seed := `{
		"agents": [
			{"name":"codex-acp","displayName":"Codex ACP","command":"codex-acp","args":[]},
			{"name":"goose","displayName":"Goose","command":"goose","args":["--acp"]},
			{"name":"custom-agent","displayName":"Custom","command":"custom-bin","args":["serve"]}
		],
		"settings": {"theme":"dark","defaultAgent":"goose","defaultCwd":"","autoApprove":false}
	}`
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatalf("write seed config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if hasAgent(cfg.Agents, "goose") {
		t.Fatalf("deprecated agent goose was not removed: %#v", cfg.Agents)
	}
	if !hasAgent(cfg.Agents, "codex-app-server") {
		t.Fatalf("codex-acp was not migrated: %#v", cfg.Agents)
	}
	if !hasAgent(cfg.Agents, "custom-agent") {
		t.Fatalf("custom agent should be preserved: %#v", cfg.Agents)
	}
	if cfg.Settings.DefaultAgent != "codex-app-server" {
		t.Fatalf("defaultAgent = %q, want codex-app-server", cfg.Settings.DefaultAgent)
	}
}

func TestLoadConfigRepopulatesDefaultsWhenOnlyDeprecatedRemain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	seed := `{
		"agents": [
			{"name":"goose","displayName":"Goose","command":"goose","args":["--acp"]},
			{"name":"claude-code-acp","displayName":"Claude","command":"claude-code-acp","args":[]}
		],
		"settings": {"theme":"dark","defaultAgent":"goose","defaultCwd":"","autoApprove":false}
	}`
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatalf("write seed config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(cfg.Agents) == 0 {
		t.Fatal("agents should be repopulated with defaults")
	}
	if !hasAgent(cfg.Agents, "opencode") || !hasAgent(cfg.Agents, "codex-app-server") {
		t.Fatalf("default agents not restored: %#v", cfg.Agents)
	}
	if cfg.Settings.DefaultAgent != "opencode" {
		t.Fatalf("defaultAgent = %q, want opencode", cfg.Settings.DefaultAgent)
	}
}

func hasAgent(agents []AgentConfig, name string) bool {
	for _, a := range agents {
		if a.Name == name {
			return true
		}
	}
	return false
}
