//go:build linux
// +build linux

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
	"time"

	wails_runtime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) platformStartup() {
}

func (a *App) CheckEnvironment() {
	go func() {
		a.log("Checking Node.js installation...")
		
		home, _ := os.UserHomeDir()
		localNodeDir := filepath.Join(home, ".cceasy", "node")
		localBinDir := filepath.Join(localNodeDir, "bin")

		// 1. Setup PATH
	envPath := os.Getenv("PATH")
		commonPaths := []string{"/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin"}
		
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
			a.log("Node.js not found. Attempting manual installation...")
			if err := a.installNodeJSManually(localNodeDir); err != nil {
				a.log("Manual installation failed: " + err.Error())
				wails_runtime.EventsEmit(a.ctx, "env-check-done")
				return
			}
			a.log("Node.js manually installed to " + localNodeDir)
			
			// Re-check for node
			localNodePath := filepath.Join(localBinDir, "node")
			if _, err := os.Stat(localNodePath); err == nil {
				nodePath = localNodePath
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
			home, _ := os.UserHomeDir()
			localClaude := filepath.Join(home, ".cceasy", "node", "bin", "claude")
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
			installCmd := exec.Command(npmExec, "install", "-g", "@anthropic-ai/claude-code")
			installCmd.Env = os.Environ()
			if out, err := installCmd.CombinedOutput(); err != nil {
				a.log("Installation failed: " + string(out))
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
		arch = "x64"
	}
	
	version := "22.14.0"
	fileName := fmt.Sprintf("node-v%s-linux-%s.tar.xz", version, arch)
	
downloadURL := fmt.Sprintf("https://nodejs.org/dist/v%s/%s", version, fileName)
	if strings.HasPrefix(strings.ToLower(a.CurrentLanguage), "zh") {
		// Use a mirror in China for faster download
		downloadURL = fmt.Sprintf("https://mirrors.tuna.tsinghua.edu.cn/nodejs-release/v%s/%s", version, fileName)
	}

	a.log(fmt.Sprintf("Downloading Node.js v%s from %s...", version, downloadURL))
	
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("network error during download: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	size := resp.ContentLength
	tempFile, err := os.CreateTemp("", "node-*.tar.xz")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	var downloaded int64
	buffer := make([]byte, 32768)
	lastReport := time.Now()

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			tempFile.Write(buffer[:n])
			downloaded += int64(n)
			if size > 0 && time.Since(lastReport) > 500*time.Millisecond {
				percent := float64(downloaded) / float64(size) * 100
				a.log(fmt.Sprintf("Downloading Node.js (%.1f%%): %d/%d bytes", percent, downloaded, size))
				lastReport = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("interrupted download: %v", err)
		}
	}
	tempFile.Close()

	if _, err := os.Stat(destDir); err == nil {
		a.log("Cleaning existing Node.js directory...")
		os.RemoveAll(destDir)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	
a.log("Extracting Node.js...")
	cmd := exec.Command("tar", "-xJf", tempFile.Name(), "-C", destDir, "--strip-components", "1")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tar extraction failed: %v, output: %s", err, string(out))
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

func (a *App) LaunchClaude(yoloMode bool, projectDir string) {
	a.log("Launching Claude Code...")
	fmt.Printf("Launching Claude Code: yoloMode=%v, projectDir=%s\n", yoloMode, projectDir)

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
		a.log("No model selected.")
		return
	}

	baseUrl := getBaseUrl(selectedModel)
	
	home, _ := os.UserHomeDir()
	localBinDir := filepath.Join(home, ".cceasy", "node", "bin")

	// Search for Claude
	claudePath, _ := exec.LookPath("claude")
	if claudePath == "" {
		// 1. Try local bin
		localClaude := filepath.Join(localBinDir, "claude")
		if _, err := os.Stat(localClaude); err == nil {
			claudePath = localClaude
		} else {
			// 2. Try global npm prefix
			npmExec, _ := exec.LookPath("npm")
			if npmExec == "" {
				localNpmPath := filepath.Join(localBinDir, "npm")
				if _, err := os.Stat(localNpmPath); err == nil {
					npmExec = localNpmPath
				}
			}
			
			if npmExec != "" {
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
	}

	scriptsDir := filepath.Join(home, ".cceasy", "scripts")
	os.MkdirAll(scriptsDir, 0755)
	launchScriptPath := filepath.Join(scriptsDir, "launch.sh")

	var sb strings.Builder
	sb.WriteString("#!/bin/bash\n")
	
	// Add both local and global bin to PATH in script
	pathDirs := []string{localBinDir}
	if claudePath != "" {
		pathDirs = append(pathDirs, filepath.Dir(claudePath))
	}
	sb.WriteString(fmt.Sprintf("export PATH=\"%s:$PATH\"\n", strings.Join(pathDirs, ":")))
	
	sb.WriteString(fmt.Sprintf("export ANTHROPIC_AUTH_TOKEN=\"%s\"\n", selectedModel.ApiKey))
	sb.WriteString(fmt.Sprintf("export ANTHROPIC_BASE_URL=\"%s\"\n", baseUrl))

	if projectDir != "" {
		sb.WriteString(fmt.Sprintf("cd \"%s\" || exit\n", projectDir))
	}

	sb.WriteString("clear\n")

	claudeArgs := ""
	if yoloMode {
		claudeArgs = " --dangerously-skip-permissions"
	}

	if claudePath != "" {
		sb.WriteString(fmt.Sprintf("if [ -f \"%s\" ]; then\n", claudePath))
		sb.WriteString(fmt.Sprintf("  exec \"%s\"%s\n", claudePath, claudeArgs))
		sb.WriteString("elif command -v claude >/dev/null 2>&1; then\n")
		sb.WriteString(fmt.Sprintf("  exec claude%s\n", claudeArgs))
	} else {
		sb.WriteString("if command -v claude >/dev/null 2>&1; then\n")
		sb.WriteString(fmt.Sprintf("  exec claude%s\n", claudeArgs))
	}

	sb.WriteString("elif command -v npx >/dev/null 2>&1; then\n")
	sb.WriteString("  echo \"claude command not found, trying npx...\"\n")
	sb.WriteString(fmt.Sprintf("  exec npx @anthropic-ai/claude-code%s\n", claudeArgs))
	sb.WriteString("else\n")
	sb.WriteString("  echo \"Error: Claude Code ('claude' command) not found and 'npx' is not available.\"\n")
	sb.WriteString("  echo \"Please make sure Node.js and Claude Code are installed correctly.\"\n")
	sb.WriteString("  read -p \"Press Enter to close...\"\n")
	sb.WriteString("fi\n")

	if err := os.WriteFile(launchScriptPath, []byte(sb.String()), 0700); err != nil {
		a.log("Failed to write launch script: " + err.Error())
		return
	}

	// Terminal fallbacks
	terminals := []struct {
		name string
		args []string
	}{
		{"x-terminal-emulator", []string{"-e", launchScriptPath}},
		{"gnome-terminal", []string{"--", launchScriptPath}},
		{"konsole", []string{"--", launchScriptPath}},
		{"xfce4-terminal", []string{"-e", launchScriptPath}},
		{"lxterminal", []string{"-e", launchScriptPath}},
		{"mate-terminal", []string{"-e", launchScriptPath}},
		{"xterm", []string{"-e", launchScriptPath}},
	}

	launched := false
	for _, t := range terminals {
		if _, err := exec.LookPath(t.name); err == nil {
			a.log("Attempting to launch via " + t.name)
			if err := exec.Command(t.name, t.args...).Start(); err == nil {
				launched = true
				break
			}
		}
	}

	if !launched {
		a.log("Failed to launch any terminal emulator.")
	}
}

func (a *App) syncToSystemEnv(config AppConfig) {
}