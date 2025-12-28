//go:build darwin

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	wails_runtime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) platformStartup() {
	// No terminal to hide on macOS
}

func (a *App) CheckEnvironment() {
	go func() {
		a.log("Checking Node.js installation...")
		
		home, _ := os.UserHomeDir()
		localNodeDir := filepath.Join(home, ".cceasy", "node")
		localBinDir := filepath.Join(localNodeDir, "bin")

		// 1. Setup PATH correctly for GUI apps on macOS
		envPath := os.Getenv("PATH")
		commonPaths := []string{"/usr/local/bin", "/opt/homebrew/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"}
		
		commonPaths = append(commonPaths, filepath.Join(home, ".npm-global/bin"))
		
		// Add local node bin to PATH
		commonPaths = append([]string{localBinDir}, commonPaths...)

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
			a.log("Updated PATH: " + envPath)
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

			if brewExec != "" {
				a.log("Installing Node.js via Homebrew...")
				cmd := exec.Command(brewExec, "install", "node")
				if err := cmd.Run(); err != nil {
					a.log("Homebrew installation failed.")
				} else {
					a.log("Node.js installed via Homebrew.")
				}
			} else {
				a.log("Homebrew not found. Attempting manual installation...")
				if err := a.installNodeJSManually(localNodeDir); err != nil {
					a.log("Manual installation failed: " + err.Error())
					wails_runtime.EventsEmit(a.ctx, "env-check-done")
					return
				}
				a.log("Node.js manually installed to " + localNodeDir)
			}
			
			a.log("Verifying Node.js installation...")
			
			// Re-check for node
			nodePath, err = exec.LookPath("node")
			if err != nil {
				// Check explicitly in local bin if LookPath fails
				localNodePath := filepath.Join(localBinDir, "node")
				if _, err := os.Stat(localNodePath); err == nil {
					nodePath = localNodePath
				}
			}
			
			if nodePath == "" {
				a.log("Node.js installation completed but binary not found.")
				wails_runtime.EventsEmit(a.ctx, "env-check-done")
				return
			}
		}

		a.log("Node.js found at: " + nodePath)

		// 4. Search for npm
		npmExec, err := exec.LookPath("npm")
		if err != nil {
			localNpmPath := filepath.Join(localBinDir, "npm")
			if _, err := os.Stat(localNpmPath); err == nil {
				npmExec = localNpmPath
			}
		}
		
		if npmExec == "" {
			a.log("npm not found.")
			wails_runtime.EventsEmit(a.ctx, "env-check-done")
			return
		}

		// 5. Search for Claude
		claudePath, _ := exec.LookPath("claude")
		if claudePath == "" {
			// Check if we are using local node, claude might be in local bin
			localClaude := filepath.Join(localBinDir, "claude")
			if _, err := os.Stat(localClaude); err == nil {
				claudePath = localClaude
			} else {
				prefixCmd := exec.Command(npmExec, "config", "get", "prefix")
				if out, err := prefixCmd.Output(); err == nil {
					prefix := strings.TrimSpace(string(out))
					globalClaude := filepath.Join(prefix, "bin", "claude")
					if _, err := os.Stat(globalClaude); err == nil {
						claudePath = globalClaude
					}
				}
			}
		}

		if claudePath == "" {
			a.log("Claude Code not found. Installing...")
			
			// Use local npm install -g if we are using manual node
			// If npm is in ~/.cceasy/node/bin/npm, it should default global install to ~/.cceasy/node/lib...
			
			installCmd := exec.Command(npmExec, "install", "-g", "@anthropic-ai/claude-code")
			installCmd.Env = os.Environ() // Explicitly pass environment with updated PATH
			if err := installCmd.Run(); err != nil {
				a.log("Standard installation failed. Trying with sudo...")
				script := fmt.Sprintf(`do shell script "%s install -g @anthropic-ai/claude-code" with administrator privileges`, npmExec)
				adminCmd := exec.Command("osascript", "-e", script)
				if err := adminCmd.Run(); err != nil {
					a.log("Installation failed.")
				} else {
					a.log("Claude Code installed.")
				}
			} else {
				a.log("Claude Code installed.")
			}
		}

		a.log("Environment check complete.")
		wails_runtime.EventsEmit(a.ctx, "env-check-done")
	}()
}

func (a *App) installNodeJSManually(destDir string) error {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64" // Node uses x64 for amd64
	}
	
	version := "v" + RequiredNodeVersion
	fileName := fmt.Sprintf("node-%s-darwin-%s.tar.gz", version, arch)
	url := fmt.Sprintf("https://nodejs.org/dist/%s/%s", version, fileName)
	
	a.log("Downloading Node.js from " + url)
	
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create a temp file for the tarball
	tempFile, err := os.CreateTemp("", "node-*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	a.log("Saving download to temp file...")
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save download: %v", err)
	}
	
	// Close file to ensure all data is flushed before tar reads it
	tempFile.Close()

	// Clean destination directory if it exists to avoid conflicts
	if _, err := os.Stat(destDir); err == nil {
		a.log("Cleaning existing Node.js directory...")
		os.RemoveAll(destDir)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	
	a.log("Extracting Node.js (this should be fast)...")
	
	// Using native tar command on the file
	// -x: extract, -z: gunzip, -f: file, -C: directory, --strip-components 1: remove root folder
	cmd := exec.Command("tar", "-xzf", tempFile.Name(), "-C", destDir, "--strip-components", "1")
	
	var stderr strings.Builder
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar extraction failed: %v, stderr: %s", err, stderr.String())
	}

	// Final verification: check if bin/node exists
	if _, err := os.Stat(filepath.Join(destDir, "bin", "node")); err != nil {
		return fmt.Errorf("verification failed: bin/node not found after extraction")
	}
	
	return nil
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
		wails_runtime.Quit(a.ctx)
		return
	}
	exec.Command("open", "-n", appBundle).Start()
	wails_runtime.Quit(a.ctx)
}

func (a *App) LaunchClaude(yoloMode bool, projectDir string) {
	a.log("Launching Claude Code...")
	config, err := a.LoadConfig()
	if err != nil {
		a.log("Error loading config: " + err.Error())
		return
	}
	
	var selectedModel *ModelConfig
	for _, m := range config.Models {
		if m.ModelName == config.CurrentModel {
			selectedModel = &m
			break
		}
	}

	if selectedModel == nil {
		a.log("No model selected or model not found in config.")
		return
	}

	baseUrl := getBaseUrl(selectedModel)
	claudePath, _ := exec.LookPath("claude")
	if claudePath == "" {
		// Try fallback to local bin
		home, _ := os.UserHomeDir()
		claudePath = filepath.Join(home, ".cceasy", "node", "bin", "claude")
	}
	a.log("Using claude at: " + claudePath)
	
	// Prepare the launch script
	home, _ := os.UserHomeDir()
	localBinDir := filepath.Join(home, ".cceasy", "node", "bin")
	scriptsDir := filepath.Join(home, ".cceasy", "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		a.log("Failed to create scripts dir: " + err.Error())
		return
	}
	launchScriptPath := filepath.Join(scriptsDir, "launch.sh")

	// Construct script content
	var sb strings.Builder
	sb.WriteString("#!/bin/bash\n")
	// Export local bin to PATH
	sb.WriteString(fmt.Sprintf("export PATH=\"%s:$PATH\"\n", localBinDir))
	// Export Auth Tokens
	sb.WriteString(fmt.Sprintf("export ANTHROPIC_AUTH_TOKEN=\"%s\"\n", selectedModel.ApiKey))
	sb.WriteString(fmt.Sprintf("export ANTHROPIC_BASE_URL=\"%s\"\n", baseUrl))
	
	// Navigate to project directory
	if projectDir != "" {
		// Escape quotes in project path for bash
		safeProjectDir := strings.ReplaceAll(projectDir, "\"", "\\\"")
		sb.WriteString(fmt.Sprintf("cd \"%s\" || exit\n", safeProjectDir))
	}

	// Clear screen to hide the command invocation
	sb.WriteString("clear\n")
	
	// Execute claude (using exec to replace the shell process)
	sb.WriteString(fmt.Sprintf("exec \"%s\"", claudePath))
	if yoloMode {
		sb.WriteString(" --dangerously-skip-permissions")
	}
	sb.WriteString("\n")

	// Write script to file
	if err := os.WriteFile(launchScriptPath, []byte(sb.String()), 0700); err != nil {
		a.log("Failed to write launch script: " + err.Error())
		return
	}

	// Launch Terminal via AppleScript
	// We run the script file. We quote the path in case of spaces in home dir.
	// We escape double quotes for AppleScript string logic.
	safeLaunchPath := strings.ReplaceAll(launchScriptPath, "\"", "\\\"")
	
	script := fmt.Sprintf(`try
	tell application "Terminal" to do script "\"%s\""
	tell application "Terminal" to activate
on error errMsg
	display dialog "Failed to launch Terminal: " & errMsg
end try`, safeLaunchPath)
	
	a.log("Executing AppleScript...")
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Start(); err != nil {
		a.log("Failed to launch Terminal: " + err.Error())
	}
}

func (a *App) syncToSystemEnv(config AppConfig) {
}