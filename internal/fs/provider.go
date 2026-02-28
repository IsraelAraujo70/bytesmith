package fs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bytesmith/internal/acp"
)

// FileChange records a single file modification made by an agent,
// capturing before/after content for undo and review.
type FileChange struct {
	Path       string
	OldContent string
	NewContent string
	Timestamp  time.Time
	SessionID  string
	AgentName  string
}

// Provider handles fs/read_text_file and fs/write_text_file requests from agents.
// It reads and writes files on disk, tracks all modifications for undo/review,
// and emits events when files are changed.
type Provider struct {
	changes       []FileChange
	mu            sync.RWMutex
	onFileChanged func(FileChange)
}

// NewProvider creates a new file system Provider.
func NewProvider() *Provider {
	return &Provider{
		changes: make([]FileChange, 0),
	}
}

// HandleReadTextFile reads a text file from disk, applying optional line offset
// and limit. Offset is 1-based. If offset is 0 or negative, it defaults to 1.
// If limit is 0 or negative, all lines from offset onward are returned.
func (p *Provider) HandleReadTextFile(params acp.FSReadTextFileParams) (*acp.FSReadTextFileResult, error) {
	f, err := os.Open(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", params.Path, err)
	}
	defer f.Close()

	var allLines []string
	scanner := bufio.NewScanner(f)
	// Increase scanner buffer for long lines (1MB).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", params.Path, err)
	}

	totalLines := len(allLines)

	offset := params.Line
	if offset <= 0 {
		offset = 1
	}
	if offset > totalLines {
		// Offset beyond file length — return empty content.
		return &acp.FSReadTextFileResult{
			Content: "",
		}, nil
	}

	startIdx := offset - 1 // convert to 0-based
	endIdx := totalLines   // exclusive

	if params.Limit > 0 {
		candidate := startIdx + params.Limit
		if candidate < endIdx {
			endIdx = candidate
		}
	}

	selected := allLines[startIdx:endIdx]
	content := strings.Join(selected, "\n")
	// Preserve trailing newline if original file had one and we're reading to the end.
	if endIdx == totalLines && totalLines > 0 {
		content += "\n"
	}

	return &acp.FSReadTextFileResult{
		Content: content,
	}, nil
}

// HandleWriteTextFile writes content to a file, creating parent directories
// if needed. It reads the existing content first to record the change for
// undo capability and emits a FileChanged event.
func (p *Provider) HandleWriteTextFile(params acp.FSWriteTextFileParams) error {
	// Read existing content for change tracking (ignore error if file doesn't exist).
	var oldContent string
	if data, err := os.ReadFile(params.Path); err == nil {
		oldContent = string(data)
	}

	// Create parent directories if they don't exist.
	dir := filepath.Dir(params.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directories for %s: %w", params.Path, err)
	}

	// Write the file.
	if err := os.WriteFile(params.Path, []byte(params.Content), 0o644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", params.Path, err)
	}

	change := FileChange{
		Path:       params.Path,
		OldContent: oldContent,
		NewContent: params.Content,
		Timestamp:  time.Now(),
	}

	p.mu.Lock()
	p.changes = append(p.changes, change)
	handler := p.onFileChanged
	p.mu.Unlock()

	if handler != nil {
		handler(change)
	}

	return nil
}

// GetChanges returns a copy of all recorded file changes.
func (p *Provider) GetChanges() []FileChange {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]FileChange, len(p.changes))
	copy(out, p.changes)
	return out
}

// OnFileChanged registers a callback that is invoked whenever a file is written.
// Only one handler is supported; subsequent calls replace the previous handler.
func (p *Provider) OnFileChanged(handler func(FileChange)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onFileChanged = handler
}
