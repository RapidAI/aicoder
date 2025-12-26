//go:build linux
// +build linux

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

		go func() {
			systray.Run(func() {
				// We need an icon for Linux. Using a placeholder or the one from resources if available.
				// For now, let's assume 'icon' is defined globally or we use nil.
				// Based on windows/darwin files, 'icon' seems to be available (likely in a resources file).
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
						go func() {
							currentConfig, _ := app.LoadConfig()
							for _, m := range currentConfig.Models {
								if m.ModelName == modelName {
									if m.ApiKey == "" {
										runtime.WindowShow(app.ctx)
										return
									}
									break
								}
							}
							currentConfig.CurrentModel = modelName
							app.SaveConfig(currentConfig)
						}()
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
					go runtime.WindowShow(app.ctx)
				})

							mLaunch.Click(func() {
								go func() {
									cfg, _ := app.LoadConfig()
									projectPath := cfg.ProjectDir
									for _, p := range cfg.Projects {
										if p.Id == cfg.CurrentProject {
											projectPath = p.Path
											break
										}
									}
									app.LaunchClaude(false, projectPath)
								}()
							})
				mQuit.Click(func() {
					go func() {
						systray.Quit()
						runtime.Quit(app.ctx)
					}()
				})

				if app.CurrentLanguage != "" {
					go func() {
						time.Sleep(500 * time.Millisecond)
						UpdateTrayMenu(app.CurrentLanguage)
					}()
				}
			}, func() {
				// Cleanup logic on exit
			})
		}()
	}
}