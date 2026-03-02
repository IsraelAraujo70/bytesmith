package backend

import (
	"bytesmith/internal/uixterm"
)

// CreateEmbeddedTerminal opens a new in-app PTY terminal tab.
func (a *App) CreateEmbeddedTerminal(cwd string) (EmbeddedTerminalInfo, error) {
	manager := a.ensureUITerminalManager()
	info, err := manager.Create(cwd)
	if err != nil {
		return EmbeddedTerminalInfo{}, err
	}
	return EmbeddedTerminalInfo{
		ID:    info.ID,
		CWD:   info.CWD,
		Shell: info.Shell,
	}, nil
}

// WriteEmbeddedTerminal forwards raw keyboard input bytes to the PTY.
func (a *App) WriteEmbeddedTerminal(terminalID, data string) error {
	return a.ensureUITerminalManager().Write(terminalID, data)
}

// ResizeEmbeddedTerminal updates PTY dimensions.
func (a *App) ResizeEmbeddedTerminal(terminalID string, cols, rows int) error {
	return a.ensureUITerminalManager().Resize(terminalID, cols, rows)
}

// CloseEmbeddedTerminal closes and removes one in-app terminal tab.
func (a *App) CloseEmbeddedTerminal(terminalID string) error {
	return a.ensureUITerminalManager().Close(terminalID)
}

func (a *App) ensureUITerminalManager() *uixterm.Manager {
	if a.uiTerm == nil {
		a.uiTerm = uixterm.NewManager()
	}
	return a.uiTerm
}
