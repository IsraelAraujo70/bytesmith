package acp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"sync/atomic"
)

// StdioTransport manages a subprocess agent connection over stdin/stdout.
//
// Messages are newline-delimited JSON (one JSON-RPC message per line).
// Incoming messages are dispatched to a registered handler function on a
// dedicated goroutine. Stderr output from the subprocess is forwarded to
// a channel for logging.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	handler   func(JSONRPCMessage)
	handlerMu sync.RWMutex

	writeMu sync.Mutex // serializes writes to stdin

	stderrCh  chan string   // stderr lines for external consumers
	done      chan struct{} // closed when the read loop exits
	running   atomic.Bool
	closeOnce sync.Once
}

// NewStdioTransport prepares a transport for the given command but does not
// start it. Call Start to spawn the subprocess and begin reading.
func NewStdioTransport(command string, args []string, env []string, cwd string) *StdioTransport {
	cmd := exec.Command(command, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	if len(env) > 0 {
		cmd.Env = env
	}

	return &StdioTransport{
		cmd:      cmd,
		stderrCh: make(chan string, 256),
		done:     make(chan struct{}),
	}
}

// Start spawns the subprocess and begins reading stdout and stderr.
// The handler (set via SetHandler) is called for each incoming JSON-RPC
// message. If no handler is set, incoming messages are silently dropped.
func (t *StdioTransport) Start() error {
	var err error

	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("acp: stdin pipe: %w", err)
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("acp: stdout pipe: %w", err)
	}

	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("acp: stderr pipe: %w", err)
	}

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("acp: start process: %w", err)
	}

	t.running.Store(true)

	go t.readLoop()
	go t.stderrLoop()

	return nil
}

// SetHandler registers the function that will be called for each incoming
// JSON-RPC message read from the subprocess stdout. Must be called before
// Start or messages may be missed.
func (t *StdioTransport) SetHandler(h func(JSONRPCMessage)) {
	t.handlerMu.Lock()
	t.handler = h
	t.handlerMu.Unlock()
}

// Send marshals a JSON-RPC message and writes it as a single line to the
// subprocess stdin. It is safe to call from multiple goroutines.
func (t *StdioTransport) Send(msg JSONRPCMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("acp: marshal message: %w", err)
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if !t.running.Load() {
		return fmt.Errorf("acp: transport is closed")
	}

	// Write the JSON line followed by a newline.
	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("acp: write stdin: %w", err)
	}
	if _, err := t.stdin.Write([]byte("\n")); err != nil {
		return fmt.Errorf("acp: write stdin newline: %w", err)
	}

	return nil
}

// StderrCh returns a channel that receives lines written to the subprocess
// stderr. The channel is buffered and will drop lines if the consumer falls
// behind.
func (t *StdioTransport) StderrCh() <-chan string {
	return t.stderrCh
}

// Done returns a channel that is closed when the read loop exits, meaning
// the subprocess stdout has been closed or the process has exited.
func (t *StdioTransport) Done() <-chan struct{} {
	return t.done
}

// IsRunning reports whether the subprocess is still running.
func (t *StdioTransport) IsRunning() bool {
	return t.running.Load()
}

// Process returns the underlying exec.Cmd.
func (t *StdioTransport) Process() *exec.Cmd {
	return t.cmd
}

// Close performs a clean shutdown: closes stdin to signal EOF to the agent,
// waits for the process to exit, and cleans up resources. If the process
// does not exit after stdin is closed, it is killed.
func (t *StdioTransport) Close() error {
	var firstErr error

	t.closeOnce.Do(func() {
		t.running.Store(false)

		// Close stdin to signal the agent to exit.
		if t.stdin != nil {
			if err := t.stdin.Close(); err != nil {
				firstErr = fmt.Errorf("acp: close stdin: %w", err)
			}
		}

		// Wait for the read loop to finish (stdout EOF).
		<-t.done

		// Wait for the process to exit and collect its status.
		if err := t.cmd.Wait(); err != nil {
			// Only record if we don't already have an error.
			if firstErr == nil {
				firstErr = fmt.Errorf("acp: wait process: %w", err)
			}
		}

		close(t.stderrCh)
	})

	return firstErr
}

// readLoop reads newline-delimited JSON-RPC messages from stdout and
// dispatches them to the registered handler.
func (t *StdioTransport) readLoop() {
	defer func() {
		t.running.Store(false)
		close(t.done)
	}()

	scanner := bufio.NewScanner(t.stdout)
	// Allow up to 10 MB per line to handle large tool outputs.
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg JSONRPCMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Printf("acp: invalid JSON from agent: %v (line: %s)", err, string(line))
			continue
		}

		t.handlerMu.RLock()
		h := t.handler
		t.handlerMu.RUnlock()

		if h != nil {
			h(msg)
		}
	}

	if err := scanner.Err(); err != nil {
		if t.running.Load() {
			log.Printf("acp: stdout read error: %v", err)
		}
	}
}

// stderrLoop reads lines from the subprocess stderr and sends them to the
// stderr channel.
func (t *StdioTransport) stderrLoop() {
	scanner := bufio.NewScanner(t.stderr)
	scanner.Buffer(make([]byte, 0, 64*1024), 1*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		select {
		case t.stderrCh <- line:
		default:
			// Drop line if channel is full to avoid blocking the agent.
		}
	}
}
