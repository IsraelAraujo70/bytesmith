package uixterm

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

const (
	defaultCols = 120
	defaultRows = 32
)

// SessionInfo is the UI-facing metadata for an embedded terminal process.
type SessionInfo struct {
	ID    string
	CWD   string
	Shell string
}

type session struct {
	info SessionInfo
	cmd  *exec.Cmd
	pty  *os.File
}

// Manager controls all embedded terminal PTY sessions.
type Manager struct {
	sessions map[string]*session
	mu       sync.RWMutex

	onOutput func(terminalID, data string)
	onExit   func(terminalID string, exitCode int)
}

// NewManager creates a new embedded terminal manager.
func NewManager() *Manager {
	return &Manager{sessions: make(map[string]*session)}
}

// OnOutput registers a callback for terminal output chunks.
func (m *Manager) OnOutput(handler func(terminalID, data string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onOutput = handler
}

// OnExit registers a callback for terminal exit events.
func (m *Manager) OnExit(handler func(terminalID string, exitCode int)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onExit = handler
}

// Create starts a new interactive shell PTY in the provided working directory.
func (m *Manager) Create(cwd string) (SessionInfo, error) {
	dir, err := validateDirectory(cwd)
	if err != nil {
		return SessionInfo{}, err
	}

	shell, args := resolveShell()
	cmd := exec.Command(shell, args...)
	cmd.Dir = dir
	cmd.Env = ensureTermEnv(os.Environ())

	ptyFile, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: defaultCols, Rows: defaultRows})
	if err != nil {
		return SessionInfo{}, fmt.Errorf("failed to start embedded terminal: %w", err)
	}

	id := uuid.NewString()
	info := SessionInfo{
		ID:    id,
		CWD:   dir,
		Shell: filepath.Base(shell),
	}

	s := &session{info: info, cmd: cmd, pty: ptyFile}

	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()

	go m.readLoop(s)
	go m.waitLoop(s)

	return info, nil
}

// Write sends input bytes to the active PTY process.
func (m *Manager) Write(terminalID, data string) error {
	s, err := m.get(terminalID)
	if err != nil {
		return err
	}
	if data == "" {
		return nil
	}

	if _, err := io.WriteString(s.pty, data); err != nil {
		return fmt.Errorf("failed to write terminal input: %w", err)
	}
	return nil
}

// Resize updates PTY dimensions.
func (m *Manager) Resize(terminalID string, cols, rows int) error {
	s, err := m.get(terminalID)
	if err != nil {
		return err
	}
	if cols <= 0 || rows <= 0 {
		return fmt.Errorf("invalid terminal size %dx%d", cols, rows)
	}

	if err := pty.Setsize(s.pty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}); err != nil {
		return fmt.Errorf("failed to resize terminal: %w", err)
	}
	return nil
}

// Close terminates and releases a terminal session.
func (m *Manager) Close(terminalID string) error {
	s, err := m.detach(terminalID)
	if err != nil {
		return err
	}

	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	if s.pty != nil {
		_ = s.pty.Close()
	}

	return nil
}

// CloseAll terminates all active embedded terminals.
func (m *Manager) CloseAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	for _, id := range ids {
		_ = m.Close(id)
	}
}

func (m *Manager) readLoop(s *session) {
	buf := make([]byte, 4096)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			m.emitOutput(s.info.ID, string(buf[:n]))
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				// Non-EOF read errors are expected during shutdown/kill.
			}
			return
		}
	}
}

func (m *Manager) waitLoop(s *session) {
	err := s.cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	if s.pty != nil {
		_ = s.pty.Close()
	}

	m.mu.Lock()
	if current, ok := m.sessions[s.info.ID]; ok && current == s {
		delete(m.sessions, s.info.ID)
	}
	m.mu.Unlock()

	m.emitExit(s.info.ID, exitCode)
}

func (m *Manager) get(terminalID string) (*session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[terminalID]
	if !ok {
		return nil, fmt.Errorf("terminal %q not found", terminalID)
	}
	return s, nil
}

func (m *Manager) detach(terminalID string) (*session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[terminalID]
	if !ok {
		return nil, fmt.Errorf("terminal %q not found", terminalID)
	}
	delete(m.sessions, terminalID)
	return s, nil
}

func (m *Manager) emitOutput(terminalID, data string) {
	m.mu.RLock()
	handler := m.onOutput
	m.mu.RUnlock()
	if handler != nil {
		handler(terminalID, data)
	}
}

func (m *Manager) emitExit(terminalID string, exitCode int) {
	m.mu.RLock()
	handler := m.onExit
	m.mu.RUnlock()
	if handler != nil {
		handler(terminalID, exitCode)
	}
}

func validateDirectory(cwd string) (string, error) {
	dir := strings.TrimSpace(cwd)
	if dir == "" {
		return "", fmt.Errorf("working directory is empty")
	}

	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("invalid working directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("working directory %q is not a directory", dir)
	}

	return dir, nil
}

func resolveShell() (string, []string) {
	if shell := strings.TrimSpace(os.Getenv("SHELL")); shell != "" {
		return shell, []string{"-l"}
	}

	for _, candidate := range []string{"zsh", "bash", "sh"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, []string{"-l"}
		}
	}

	return "sh", []string{"-l"}
}

func ensureTermEnv(env []string) []string {
	for _, entry := range env {
		if strings.HasPrefix(entry, "TERM=") {
			return env
		}
	}

	out := make([]string, 0, len(env)+1)
	out = append(out, env...)
	out = append(out, "TERM=xterm-256color")
	return out
}
