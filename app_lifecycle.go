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
// re-applying maximise shortly after DOM ready is more reliable than relying
// only on initial window options.
func (a *App) domReady(ctx context.Context) {
	apply := func() {
		wailsRuntime.WindowUnfullscreen(ctx)
		// Wayland can retain stale geometry hints; keep max size effectively unset.
		wailsRuntime.WindowSetMaxSize(ctx, 16384, 16384)

		if screens, err := wailsRuntime.ScreenGetAll(ctx); err == nil {
			for _, screen := range screens {
				if screen.IsCurrent || screen.IsPrimary {
					width := screen.Size.Width
					height := screen.Size.Height
					if width > 0 && height > 0 {
						wailsRuntime.WindowSetSize(ctx, width, height)
					}
					break
				}
			}
		}

		wailsRuntime.WindowMaximise(ctx)
	}

	apply()

	go func() {
		for _, delay := range []time.Duration{
			180 * time.Millisecond,
			600 * time.Millisecond,
		} {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			timer.Stop()

			if !wailsRuntime.WindowIsMaximised(ctx) {
				apply()
			}
		}
	}()
}

// shutdown is called by Wails when the application is closing.
func (a *App) shutdown(ctx context.Context) {
	a.App.Shutdown(ctx)
}
