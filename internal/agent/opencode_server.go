package agent

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

const (
	defaultOpenCodeHost = "127.0.0.1"
	defaultOpenCodePort = 4096
)

type openCodeServerManager struct {
	mu      sync.Mutex
	host    string
	port    int
	baseURL string
	refs    int

	managed bool
	cmd     *exec.Cmd
	done    chan error
}

func newOpenCodeServerManager() *openCodeServerManager {
	port := parseOpenCodePort()
	return &openCodeServerManager{
		host:    defaultOpenCodeHost,
		port:    port,
		baseURL: fmt.Sprintf("http://%s:%d", defaultOpenCodeHost, port),
	}
}

func (m *openCodeServerManager) acquire(command string, env map[string]string) (string, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if command == "" {
		command = "opencode"
	}

	if !checkOpenCodeHealth(m.baseURL) {
		if m.managed && m.cmd != nil {
			_ = m.cmd.Process.Kill()
			m.waitDoneLocked(2 * time.Second)
			m.clearManagedLocked()
		}

		if err := m.spawnLocked(command, env); err != nil {
			return "", nil, err
		}
	}

	m.refs++
	var once sync.Once
	release := func() {
		once.Do(func() {
			m.release()
		})
	}

	return m.baseURL, release, nil
}

func (m *openCodeServerManager) release() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.refs > 0 {
		m.refs--
	}
	if m.refs != 0 {
		return
	}
	if !m.managed || m.cmd == nil {
		return
	}

	_ = m.cmd.Process.Kill()
	m.waitDoneLocked(3 * time.Second)
	m.clearManagedLocked()
}

func (m *openCodeServerManager) shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.refs = 0
	if m.managed && m.cmd != nil {
		_ = m.cmd.Process.Kill()
		m.waitDoneLocked(3 * time.Second)
	}
	m.clearManagedLocked()
}

func (m *openCodeServerManager) spawnLocked(command string, env map[string]string) error {
	cmd := exec.Command(
		command,
		"serve",
		"--hostname", m.host,
		"--port", strconv.Itoa(m.port),
	)

	fullEnv := os.Environ()
	for k, v := range env {
		fullEnv = append(fullEnv, k+"="+v)
	}
	cmd.Env = fullEnv

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("agent: start opencode serve: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if checkOpenCodeHealth(m.baseURL) {
			m.managed = true
			m.cmd = cmd
			m.done = done
			return nil
		}

		select {
		case err := <-done:
			return fmt.Errorf("agent: opencode serve exited before health check: %w", err)
		default:
		}

		time.Sleep(200 * time.Millisecond)
	}

	_ = cmd.Process.Kill()
	select {
	case <-done:
	default:
	}
	return fmt.Errorf("agent: timed out waiting for opencode server health at %s", m.baseURL)
}

func (m *openCodeServerManager) waitDoneLocked(timeout time.Duration) {
	if m.done == nil {
		return
	}
	done := m.done
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

func (m *openCodeServerManager) clearManagedLocked() {
	m.managed = false
	m.cmd = nil
	m.done = nil
}

func checkOpenCodeHealth(baseURL string) bool {
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}

	resp, err := client.Get(baseURL + "/global/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func parseOpenCodePort() int {
	for _, key := range []string{"BYTESMITH_OPENCODE_PORT", "OPENCODE_PORT"} {
		raw := os.Getenv(key)
		if raw == "" {
			continue
		}
		port, err := strconv.Atoi(raw)
		if err == nil && port > 0 && port <= 65535 {
			return port
		}
	}
	return defaultOpenCodePort
}
