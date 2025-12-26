package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Platform specific early initialization (like hiding console on Windows)
	app.platformStartup()

	// Create application with options
	appOptions := &options.App{
		Title:     "Claude Code Easy Suite",
		Frameless: true,
		Width:     602,
		Height:    200,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "claude-code-easy-suite-lock",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				runtime.WindowUnminimise(app.ctx)
				runtime.WindowShow(app.ctx)
				runtime.WindowSetAlwaysOnTop(app.ctx, true)
				runtime.WindowSetAlwaysOnTop(app.ctx, false)
			},
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 0},
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			BackdropType:         windows.Auto,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHidden(),
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		Linux: &linux.Options{
			Icon: icon,
		},
	}

	// Platform specific tray/menu setup
	setupTray(app, appOptions)

	err := wails.Run(appOptions)

	if err != nil {
		println("Error:", err.Error())
	}
}