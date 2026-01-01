//go:build darwin

package main

import (
	"context"
	"time"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func setupTray(app *App, appOptions *options.App) {
	// We still use a basic Application Menu for macOS to support standard shortcuts
	appMenu := menu.NewMenu()
	appMenu.Append(menu.AppMenu())
	appMenu.Append(menu.EditMenu())
	appOptions.Menu = appMenu

	appOptions.OnStartup = func(ctx context.Context) {
		app.startup(ctx)

		// Start energye/systray in a goroutine
		go systray.Run(func() {
			systray.SetIcon(icon)
			// Do not set title for macOS as requested
			systray.SetTooltip("AICoder Dashboard")

			// Ensure clicking the icon shows the menu immediately on macOS
			systray.CreateMenu()

			mShow := systray.AddMenuItem("Show Main Window", "Show Main Window")
			mLaunch := systray.AddMenuItem("开始编程", "Start Coding")
			systray.AddSeparator()

			// Model menu items map
			modelItems := make(map[string]*systray.MenuItem)

			// Load config to populate tray
			config, _ := app.LoadConfig()

			// Claude Code menu items
			for _, model := range config.Claude.Models {
				displayName := "Claude Code - " + model.ModelName
				m := systray.AddMenuItemCheckbox(displayName, "Switch to "+model.ModelName, model.ModelName == config.Claude.CurrentModel && config.ActiveTool == "claude")
				modelItems["claude-"+model.ModelName] = m

				modelName := model.ModelName
				m.Click(func() {
					go func() {
						currentConfig, _ := app.LoadConfig()
						// Check if target model has API key
						for _, m := range currentConfig.Claude.Models {
							if m.ModelName == modelName {
								if m.ApiKey == "" {
									runtime.WindowShow(app.ctx)
									return
								}
								break
							}
						}
						currentConfig.Claude.CurrentModel = modelName
						currentConfig.ActiveTool = "claude"
						app.SaveConfig(currentConfig)
					}()
				})
			}

			// Gemini CLI menu items
			for _, model := range config.Gemini.Models {
				displayName := "Gemini CLI - " + model.ModelName
				m := systray.AddMenuItemCheckbox(displayName, "Switch to "+model.ModelName, model.ModelName == config.Gemini.CurrentModel && config.ActiveTool == "gemini")
				modelItems["gemini-"+model.ModelName] = m

				modelName := model.ModelName
				m.Click(func() {
					go func() {
						currentConfig, _ := app.LoadConfig()
						// Check if target model has API key
						for _, m := range currentConfig.Gemini.Models {
							if m.ModelName == modelName {
								if m.ApiKey == "" {
									runtime.WindowShow(app.ctx)
									return
								}
								break
							}
						}
						currentConfig.Gemini.CurrentModel = modelName
						currentConfig.ActiveTool = "gemini"
						app.SaveConfig(currentConfig)
					}()
				})
			}

			// Codex menu items
			for _, model := range config.Codex.Models {
				displayName := "Codex - " + model.ModelName
				m := systray.AddMenuItemCheckbox(displayName, "Switch to "+model.ModelName, model.ModelName == config.Codex.CurrentModel && config.ActiveTool == "codex")
				modelItems["codex-"+model.ModelName] = m

				modelName := model.ModelName
				m.Click(func() {
					go func() {
						currentConfig, _ := app.LoadConfig()
						// Check if target model has API key
						for _, m := range currentConfig.Codex.Models {
							if m.ModelName == modelName {
								if m.ApiKey == "" {
									runtime.WindowShow(app.ctx)
									return
								}
								break
							}
						}
						currentConfig.Codex.CurrentModel = modelName
						currentConfig.ActiveTool = "codex"
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
				systray.SetTooltip(t["title"])
				mShow.SetTitle(t["show"])
				mLaunch.SetTitle(t["launch"])
				mQuit.SetTitle(t["quit"])
			}

			// Register config change listener
			OnConfigChanged = func(cfg AppConfig) {
				if modelItems == nil {
					return
				}
				for name, item := range modelItems {
					// Only check the currently active tool's current model
					if (cfg.ActiveTool == "claude" && name == "claude-"+cfg.Claude.CurrentModel) ||
						(cfg.ActiveTool == "gemini" && name == "gemini-"+cfg.Gemini.CurrentModel) ||
						(cfg.ActiveTool == "codex" && name == "codex-"+cfg.Codex.CurrentModel) {
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
					path := app.GetCurrentProjectPath()
					app.LaunchTool("claude", false, path)
				}()
			})

			mQuit.Click(func() {
				go func() {
					systray.Quit()
					runtime.Quit(app.ctx)
				}()
			})

			// Initial language sync
			if app.CurrentLanguage != "" {
				go func() {
					time.Sleep(500 * time.Millisecond)
					UpdateTrayMenu(app.CurrentLanguage)
				}()
			}
		}, func() {})
	}
}
