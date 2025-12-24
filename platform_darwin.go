//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) platformStartup() {
	// No terminal to hide on macOS
}

func (a *App) CheckEnvironment() {
	go func() {
		a.log("Checking Node.js installation...")

		// 1. Setup PATH correctly for GUI apps on macOS
		envPath := os.Getenv("PATH")
		commonPaths := []string{"/usr/local/bin", "/opt/homebrew/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"}
		
		home, _ := os.UserHomeDir()
		commonPaths = append(commonPaths, filepath.Join(home, ".npm-global/bin"))

		newPathParts := strings.Split(envPath, ":")
		pathChanged := false
		for _, p := range commonPaths {
			if !contains(newPathParts, p) {
				newPathParts = append([]string{p}, newPathParts...) // Prepend for priority
				pathChanged = true
			}
		}
		
		if pathChanged {
			envPath = strings.Join(newPathParts, ":")
			os.Setenv("PATH", envPath)
		}

		// 2. Search for Node.js
		nodePath, err := exec.LookPath("node")
		if err != nil {
			for _, p := range commonPaths {
				fullPath := filepath.Join(p, "node")
				if _, err := os.Stat(fullPath); err == nil {
					nodePath = fullPath
					break
				}
			}
		}

		// 3. If still not found, try to install
		if nodePath == "" {
			a.log("Node.js not found. Checking for Homebrew...")
			
			brewExec, _ := exec.LookPath("brew")
			if brewExec == "" {
				for _, p := range []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"} {
					if _, err := os.Stat(p); err == nil {
						brewExec = p
						break
					}
				}
			}

			if brewExec == "" {
				a.log("Homebrew not found. Please install Node.js manually.")
				runtime.EventsEmit(a.ctx, "env-check-done")
				return
			}

			a.log("Installing Node.js via Homebrew...")
			cmd := exec.Command(brewExec, "install", "node")
			if err := cmd.Run(); err != nil {
				a.log("Installation failed.")
				runtime.EventsEmit(a.ctx, "env-check-done")
				return
			}
			
			a.log("Node.js installed. Restarting...")
			a.restartApp()
			return
		}

		a.log("Node.js found at: " + nodePath)

		// 4. Search for npm
		npmExec, err := exec.LookPath("npm")
		if err != nil {
			for _, p := range commonPaths {
				fullPath := filepath.Join(p, "npm")
				if _, err := os.Stat(fullPath); err == nil {
					npmExec = fullPath
					break
				}
			}
		}

		if npmExec == "" {
			a.log("npm not found.")
			runtime.EventsEmit(a.ctx, "env-check-done")
			return
		}

		// 5. Search for Claude
		claudePath, _ := exec.LookPath("claude")
		if claudePath == "" {
			prefixCmd := exec.Command(npmExec, "config", "get", "prefix")
			if out, err := prefixCmd.Output(); err == nil {
				prefix := strings.TrimSpace(string(out))
				globalClaude := filepath.Join(prefix, "bin", "claude")
				if _, err := os.Stat(globalClaude); err == nil {
					claudePath = globalClaude
				}
			}
		}

		if claudePath == "" {
			a.log("Claude Code not found. Installing...")
			installCmd := exec.Command(npmExec, "install", "-g", "@anthropic-ai/claude-code")
			if err := installCmd.Run(); err != nil {
				a.log("Standard installation failed. Trying with sudo...")
				script := fmt.Sprintf(`do shell script "%s install -g @anthropic-ai/claude-code" with administrator privileges`, npmExec)
				adminCmd := exec.Command("osascript", "-e", script)
				if err := adminCmd.Run(); err != nil {
					a.log("Installation failed.")
				} else {
					a.log("Claude Code installed. Restarting...")
					a.restartApp()
					return
				}
			} else {
				a.log("Claude Code installed. Restarting...")
				a.restartApp()
				return
			}
		}

		a.log("Environment check complete.")
		runtime.EventsEmit(a.ctx, "env-check-done")
	}()
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (a *App) restartApp() {
	executable, err := os.Executable()
	if err != nil {
		return
	}
	appBundle := filepath.Dir(filepath.Dir(filepath.Dir(executable)))
	if !strings.HasSuffix(appBundle, ".app") {
		runtime.Quit(a.ctx)
		return
	}
	exec.Command("open", "-n", appBundle).Start()
	runtime.Quit(a.ctx)
}

func (a *App) LaunchClaude(yoloMode bool, projectDir string) {
	config, _ := a.LoadConfig()
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
	claudePath, _ := exec.LookPath("claude")
	if claudePath == "" {
		claudePath = "claude"
	}

	command := fmt.Sprintf("export ANTHROPIC_AUTH_TOKEN=%s && export ANTHROPIC_BASE_URL=%s && %s", 
		selectedModel.ApiKey, baseUrl, claudePath)
	
	if yoloMode {
		command += " --dangerously-skip-permissions"
	}

	script := fmt.Sprintf(`tell application "Terminal" to do script "cd %s && %s"`, projectDir, command)
	if projectDir == "" {
		script = fmt.Sprintf(`tell application "Terminal" to do script "%s"`, command)
	}
	
	exec.Command("osascript", "-e", script).Start()
}

func (a *App) syncToSystemEnv(config AppConfig) {
}
