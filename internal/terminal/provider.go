package terminal

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"bytesmith/internal/acp"

	"github.com/google/uuid"
)

// Terminal represents a single subprocess spawned on behalf of an agent.
type Terminal struct {
	ID         string
	SessionID  string
	Command    string
	Args       []string
	CWD        string
	Output     bytes.Buffer
	Truncated  bool
	ByteLimit  int
	ExitStatus *acp.TerminalExitStatus
	cmd        *exec.Cmd
	done       chan struct{}
	mu         sync.Mutex
}

// Provider manages terminal instances created by agents.
// It starts subprocesses, captures their combined stdout/stderr output,
// and provides methods to query output, wait for exit, kill, and release.
type Provider struct {
	terminals map[string]*Terminal
	mu        sync.RWMutex
	onOutput  func(terminalID string, data string)
}

// NewProvider creates a new terminal Provider.
func NewProvider() *Provider {
	return &Provider{
		terminals: make(map[string]*Terminal),
	}
}

// HandleCreate starts a new subprocess and returns its terminal ID.
// The subprocess runs immediately with combined stdout/stderr piped into
// an in-memory buffer. Output is truncated from the beginning if it
// exceeds the configured byte limit.
func (p *Provider) HandleCreate(params acp.TerminalCreateParams) (*acp.TerminalCreateResult, error) {
	id := uuid.New().String()

	cmd := exec.Command(params.Command, params.Args...)
	if params.CWD != "" {
		cmd.Dir = params.CWD
	}

	// Combine stdout and stderr into a single pipe.
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout // merge stderr into stdout pipe

	byteLimit := params.OutputByteLimit
	if byteLimit <= 0 {
		byteLimit = 1024 * 1024 // default 1MB
	}

	t := &Terminal{
		ID:        id,
		Command:   params.Command,
		Args:      params.Args,
		CWD:       params.CWD,
		ByteLimit: byteLimit,
		cmd:       cmd,
		done:      make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command %q: %w", params.Command, err)
	}

	p.mu.Lock()
	p.terminals[id] = t
	p.mu.Unlock()

	// Read output in background.
	go p.readOutput(t, stdoutPipe)

	// Wait for process exit in background.
	go p.waitForProcess(t)

	return &acp.TerminalCreateResult{
		TerminalID: id,
	}, nil
}

// readOutput reads from the pipe and appends to the terminal's output buffer,
// truncating from the beginning if the byte limit is exceeded.
func (p *Provider) readOutput(t *Terminal, r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := buf[:n]

			t.mu.Lock()
			t.Output.Write(chunk)
			// Truncate from beginning if over limit.
			if t.Output.Len() > t.ByteLimit {
				data := t.Output.Bytes()
				excess := len(data) - t.ByteLimit
				t.Output.Reset()
				t.Output.Write(data[excess:])
				t.Truncated = true
			}
			t.mu.Unlock()

			p.mu.RLock()
			handler := p.onOutput
			p.mu.RUnlock()

			if handler != nil {
				handler(t.ID, string(chunk))
			}
		}
		if err != nil {
			break
		}
	}
}

// waitForProcess waits for the subprocess to exit and records its exit status.
func (p *Provider) waitForProcess(t *Terminal) {
	err := t.cmd.Wait()

	t.mu.Lock()
	defer t.mu.Unlock()

	status := acp.TerminalExitStatus{}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			status.ExitCode = &code
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				status.Signal = ws.Signal().String()
			}
		} else {
			// Non-exit error (shouldn't normally happen after Start succeeds).
			code := -1
			status.ExitCode = &code
		}
	} else {
		code := 0
		status.ExitCode = &code
	}

	t.ExitStatus = &status
	close(t.done)
}

// get retrieves a terminal by ID, returning an error if not found.
func (p *Provider) get(id string) (*Terminal, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	t, ok := p.terminals[id]
	if !ok {
		return nil, fmt.Errorf("terminal %q not found", id)
	}
	return t, nil
}

// HandleOutput returns the current buffered output for a terminal and its
// exit status if the process has finished.
func (p *Provider) HandleOutput(params acp.TerminalOutputParams) (*acp.TerminalOutputResult, error) {
	t, err := p.get(params.TerminalID)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	output := t.Output.String()
	truncated := t.Truncated
	exitStatus := t.ExitStatus
	t.mu.Unlock()

	return &acp.TerminalOutputResult{
		Output:     output,
		Truncated:  truncated,
		ExitStatus: exitStatus,
	}, nil
}

// HandleWaitForExit blocks until the terminal's subprocess exits and returns
// the exit status.
func (p *Provider) HandleWaitForExit(params acp.TerminalWaitParams) (*acp.TerminalWaitResult, error) {
	t, err := p.get(params.TerminalID)
	if err != nil {
		return nil, err
	}

	<-t.done

	t.mu.Lock()
	status := *t.ExitStatus
	t.mu.Unlock()

	return &acp.TerminalWaitResult{
		ExitCode: status.ExitCode,
		Signal:   status.Signal,
	}, nil
}

// HandleKill sends SIGTERM to the subprocess. If it hasn't exited after 5
// seconds, it sends SIGKILL.
func (p *Provider) HandleKill(params acp.TerminalKillParams) error {
	t, err := p.get(params.TerminalID)
	if err != nil {
		return err
	}

	t.mu.Lock()
	if t.ExitStatus != nil {
		t.mu.Unlock()
		return nil // already exited
	}
	process := t.cmd.Process
	t.mu.Unlock()

	if process == nil {
		return nil
	}

	// Send SIGTERM.
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited.
		return nil
	}

	// Wait up to 5 seconds for graceful exit, then SIGKILL.
	select {
	case <-t.done:
		return nil
	case <-time.After(5 * time.Second):
		_ = process.Signal(syscall.SIGKILL)
		<-t.done
		return nil
	}
}

// HandleRelease kills the subprocess if still running and removes the terminal
// from the provider's map, freeing resources.
func (p *Provider) HandleRelease(params acp.TerminalReleaseParams) error {
	t, err := p.get(params.TerminalID)
	if err != nil {
		return err
	}

	// Kill if still running.
	_ = p.HandleKill(acp.TerminalKillParams{TerminalID: t.ID})

	p.mu.Lock()
	delete(p.terminals, params.TerminalID)
	p.mu.Unlock()

	return nil
}

// OnOutput registers a callback invoked whenever new output is read from any
// terminal. Only one handler is supported; subsequent calls replace the
// previous handler.
func (p *Provider) OnOutput(handler func(terminalID string, data string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onOutput = handler
}

// CloseAll kills and releases all active terminals.
func (p *Provider) CloseAll() {
	p.mu.RLock()
	ids := make([]string, 0, len(p.terminals))
	for id := range p.terminals {
		ids = append(ids, id)
	}
	p.mu.RUnlock()

	for _, id := range ids {
		_ = p.HandleRelease(acp.TerminalReleaseParams{TerminalID: id})
	}
}
