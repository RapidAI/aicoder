//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func init() {
	hideConsole()
}

func hideConsole() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	user32 := syscall.NewLazyDLL("user32.dll")

	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	showWindow := user32.NewProc("ShowWindow")

	if getConsoleWindow.Find() == nil && showWindow.Find() == nil {
		hwnd, _, _ := getConsoleWindow.Call()
		if hwnd != 0 {
			showWindow.Call(hwnd, 0) // SW_HIDE = 0
		}
	}
}

func (a *App) platformStartup() {
	hideConsole()
}

func (a *App) CheckEnvironment() {
	go func() {
		a.log("Checking Node.js installation...")
		
		npmPath := "npm"
		// Check for node
		nodeCmd := exec.Command("node", "--version")
		nodeCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := nodeCmd.Run(); err != nil {
			a.log("Node.js not found. Installing via Winget (this may take a while)...")
			// Try installing Node.js
			cmd := exec.Command("winget", "install", "-e", "--id", "OpenJS.NodeJS", "--silent", "--accept-package-agreements", "--accept-source-agreements")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if out, err := cmd.CombinedOutput(); err != nil {
				a.log("Error installing Node.js: " + string(out))
			} else {
				a.log("Node.js installed successfully.")
				npmPath = `C:\Program Files\nodejs\npm.cmd`
			}
		} else {
			a.log("Node.js is installed.")
		}

		a.log("Checking Claude Code...")
		
		claudeCheckCmd := exec.Command("claude", "--version")
		claudeCheckCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		claudeExists := claudeCheckCmd.Run() == nil

		if !claudeExists {
			a.log("Claude Code not found. Installing...")
			installCmd := exec.Command(npmPath, "install", "-g", "@anthropic-ai/claude-code")
			installCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			
			if out, err := installCmd.CombinedOutput(); err != nil {
				if npmPath == "npm" {
					installCmd = exec.Command(`C:\Program Files\nodejs\npm.cmd`, "install", "-g", "@anthropic-ai/claude-code")
					installCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
					if out2, err2 := installCmd.CombinedOutput(); err2 != nil {
						a.log("Failed to install Claude Code: " + string(out) + " / " + string(out2))
					} else {
						a.log("Claude Code installed successfully. Restarting app to apply changes...")
						a.restartApp()
						return
					}
				} else {
					a.log("Failed to install Claude Code: " + string(out))
				}
			} else {
				a.log("Claude Code installed successfully. Restarting app to apply changes...")
				a.restartApp()
				return
			}
		} else {
			a.log("Claude Code found. Checking for updates (npm install -g @anthropic-ai/claude-code)...")
			
			installCmd := exec.Command(npmPath, "install", "-g", "@anthropic-ai/claude-code")
			installCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if out, err := installCmd.CombinedOutput(); err != nil {
				if npmPath == "npm" {
					installCmd = exec.Command(`C:\Program Files\nodejs\npm.cmd`, "install", "-g", "@anthropic-ai/claude-code")
					installCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
					if out2, err2 := installCmd.CombinedOutput(); err2 != nil {
						a.log("Failed to update Claude Code: " + string(out) + " / " + string(out2))
					} else {
						a.log("Claude Code updated successfully.")
					}
				} else {
					a.log("Failed to update Claude Code: " + string(out))
				}
			} else {
				a.log("Claude Code updated successfully.")
			}
		}

		a.log("Environment check complete.")
		runtime.EventsEmit(a.ctx, "env-check-done")
	}()
}

func (a *App) restartApp() {
	executable, err := os.Executable()
	if err != nil {
		a.log("Failed to get executable path: " + err.Error())
		return
	}

	cmd := exec.Command("cmd", "/c", "start", "", executable)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		a.log("Failed to restart: " + err.Error())
	} else {
		runtime.Quit(a.ctx)
	}
}

func (a *App) LaunchClaude(yoloMode bool, projectDir string) {
	args := []string{"/c", "start", "cmd.exe", "/k", "claude"}
	if yoloMode {
		args = append(args, "--dangerously-skip-permissions")
	}
	
	cmd := exec.Command("cmd.exe", args...)
	if projectDir != "" {
		cmd.Dir = projectDir
	}
	
	cmd.Env = os.Environ()
	
	if err := cmd.Start(); err != nil {
		a.log("Failed to launch Claude: " + err.Error())
	}
}

func (a *App) syncToSystemEnv(config AppConfig) {
	var selectedModel *ModelConfig
	for _, m := range config.Models {
		if m.ModelName == config.CurrentModel {
			selectedModel = &m
			break
		}
	}

	if selectedModel == nil {
		return
	}

	baseUrl := getBaseUrl(selectedModel)

	// Set environment variables for the current process immediately
	os.Setenv("ANTHROPIC_AUTH_TOKEN", selectedModel.ApiKey)
	os.Setenv("ANTHROPIC_BASE_URL", baseUrl)

	// Set persistent environment variables on Windows in a goroutine because setx is slow
	go func() {
		cmd1 := exec.Command("setx", "ANTHROPIC_AUTH_TOKEN", selectedModel.ApiKey)
		cmd1.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd1.Run()

		cmd2 := exec.Command("setx", "ANTHROPIC_BASE_URL", baseUrl)
		cmd2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd2.Run()
	}()
}
