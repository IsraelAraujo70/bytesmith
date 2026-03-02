package main

import (
	"context"
	"time"

	"bytesmith/internal/backend"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
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

// domReady is called when the frontend DOM is ready. On Wayland compositors,
// applying maximise/fullscreen here (and re-applying shortly after) is more
// reliable than relying only on initial window options.
func (a *App) domReady(ctx context.Context) {
	apply := func() {
		wailsRuntime.WindowMaximise(ctx)
		wailsRuntime.WindowFullscreen(ctx)
	}

	apply()

	go func() {
		timer := time.NewTimer(180 * time.Millisecond)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if !wailsRuntime.WindowIsFullscreen(ctx) {
				apply()
			}
		}
	}()
}

// shutdown is called by Wails when the application is closing.
func (a *App) shutdown(ctx context.Context) {
	a.App.Shutdown(ctx)
}
