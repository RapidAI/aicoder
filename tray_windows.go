//go:build windows

package main

import (
	"context"
	"time"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func setupTray(app *App, appOptions *options.App) {
	appOptions.OnStartup = func(ctx context.Context) {
		app.startup(ctx)

		go systray.Run(func() {
			systray.SetIcon(icon)
			systray.SetTitle("Claude Config Manager")
			systray.SetTooltip("Claude Config Manager")

			mShow := systray.AddMenuItem("Show", "Show Main Window")
			mLaunch := systray.AddMenuItem("Launch Claude Code", "Launch Claude Code in Terminal")
			systray.AddSeparator()

			// Model menu items map
			modelItems := make(map[string]*systray.MenuItem)

			// Load config to populate tray
			config, _ := app.LoadConfig()
			for _, model := range config.Models {
				m := systray.AddMenuItemCheckbox(model.ModelName, "Switch to "+model.ModelName, model.ModelName == config.CurrentModel)
				modelItems[model.ModelName] = m
				
				modelName := model.ModelName
				m.Click(func() {
					currentConfig, _ := app.LoadConfig()
					currentConfig.CurrentModel = modelName
					app.SaveConfig(currentConfig)
				})
			}

			systray.AddSeparator()
			mQuit := systray.AddMenuItem("Quit", "Quit Application")

			// Register update function
			UpdateTrayMenu = func(lang string) {
				t, ok := trayTranslations[lang]
				if !ok {
					t = trayTranslations["en"]
				}
				systray.SetTitle(t["title"])
				systray.SetTooltip(t["title"])
				mShow.SetTitle(t["show"])
				mLaunch.SetTitle(t["launch"])
				mQuit.SetTitle(t["quit"])
			}

			// Register config change listener
			OnConfigChanged = func(cfg AppConfig) {
				for name, item := range modelItems {
					if name == cfg.CurrentModel {
						item.Check()
					} else {
						item.Uncheck()
					}
				}
				runtime.EventsEmit(app.ctx, "config-changed", cfg)
			}

			// Handle menu clicks
			mShow.Click(func() {
				runtime.WindowShow(app.ctx)
			})

			mLaunch.Click(func() {
				cfg, _ := app.LoadConfig()
				app.LaunchClaude(false, cfg.ProjectDir)
			})

			mQuit.Click(func() {
				systray.Quit()
				runtime.Quit(app.ctx)
			})

			if app.CurrentLanguage != "" {
				go func() {
					time.Sleep(500 * time.Millisecond)
					UpdateTrayMenu(app.CurrentLanguage)
				}()
			}
		}, func() {})
	}
}
