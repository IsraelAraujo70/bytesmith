package main

import (
	"context"

	"bytesmith/internal/backend"
)

// App is the Wails-bound adapter that forwards calls to backend.App.
type App struct {
	*backend.App
}

// NewApp creates a new application adapter.
func NewApp() *App {
	return &App{App: backend.NewApp()}
}

// startup is called by Wails when the application starts.
func (a *App) startup(ctx context.Context) {
	a.App.Startup(ctx)
}

// shutdown is called by Wails when the application is closing.
func (a *App) shutdown(ctx context.Context) {
	a.App.Shutdown(ctx)
}
