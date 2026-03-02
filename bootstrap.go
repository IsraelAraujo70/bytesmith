package main

import (
	"fmt"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func run() {
	app := NewApp()
	if err := wails.Run(appOptions(app)); err != nil {
		fmt.Println("Error:", err.Error())
	}
}

func appOptions(app *App) *options.App {
	return &options.App{
		Title:            "ByteSmith",
		Width:            1920,
		Height:           1080,
		MinWidth:         800,
		MinHeight:        600,
		DisableResize:    false,
		Frameless:        false,
		WindowStartState: options.Maximised,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 24, G: 24, B: 27, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	}
}
