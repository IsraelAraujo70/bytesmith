package backend

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"bytesmith/internal/agent"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

// GetSettings returns the current application settings.
func (a *App) GetSettings() AppSettingsInfo {
	return AppSettingsInfo{
		Theme:        a.config.Settings.Theme,
		DefaultAgent: a.config.Settings.DefaultAgent,
		DefaultCWD:   a.config.Settings.DefaultCWD,
		AutoApprove:  a.config.Settings.AutoApprove,
	}
}

// SaveSettings persists new application settings to the config file.
func (a *App) SaveSettings(settings AppSettingsInfo) error {
	a.config.Settings = agent.AppSettings{
		Theme:        settings.Theme,
		DefaultAgent: settings.DefaultAgent,
		DefaultCWD:   settings.DefaultCWD,
		AutoApprove:  settings.AutoApprove,
	}
	return agent.SaveConfig(a.configPath, a.config)
}

// ---------------------------------------------------------------------------
// File system
// ---------------------------------------------------------------------------

// SelectDirectory opens the native directory picker dialog and returns the
// selected path, or an empty string if the user cancelled.
func (a *App) SelectDirectory() (string, error) {
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Directory",
	})
}

// ListFiles returns directory entries sorted with directories first.
func (a *App) ListFiles(dir string) ([]FileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	result := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		var size int64
		if info, infoErr := e.Info(); infoErr == nil {
			size = info.Size()
		}
		result = append(result, FileEntry{
			Name:  e.Name(),
			Path:  filepath.Join(dir, e.Name()),
			IsDir: e.IsDir(),
			Size:  size,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}
