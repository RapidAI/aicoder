package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type ToolStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Path      string `json:"path"`
}

type ToolManager struct {
	app *App
}

func NewToolManager(app *App) *ToolManager {
	return &ToolManager{app: app}
}

func (tm *ToolManager) GetToolStatus(name string) ToolStatus {
	status := ToolStatus{Name: name}
	
	binaryName := name
	// Check for specific binary names if different from tool name
	// currently claude -> claude, codex -> codex, gemini -> gemini

	path, err := exec.LookPath(binaryName)
	if err != nil {
		// Fallback: Check local node bin directly
		// This handles cases where PATH hasn't been updated yet or is missing the local bin
		home, _ := os.UserHomeDir()
		var localBin string
		
		if runtime.GOOS == "windows" {
			// Windows npm global installs usually put .cmd or .ps1 in the prefix root or bin
			// We check both the bin folder and the prefix root just in case
			localBin = filepath.Join(home, ".cceasy", "node", binaryName+".cmd")
			if _, err := os.Stat(localBin); err != nil {
				localBin = filepath.Join(home, ".cceasy", "node", "bin", binaryName+".cmd")
			}
		} else {
			localBin = filepath.Join(home, ".cceasy", "node", "bin", binaryName)
		}

		if _, err := os.Stat(localBin); err == nil {
			path = localBin
		} else {
			return status
		}
	}

	status.Installed = true
	status.Path = path
	
	version, err := tm.getToolVersion(binaryName, path)
	if err == nil {
		status.Version = version
	}

	return status
}

func (tm *ToolManager) getToolVersion(name, path string) (string, error) {
	var cmd *exec.Cmd
	// Use --version for all tools
	cmd = exec.Command(path, "--version")

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	output := strings.TrimSpace(string(out))
	// Parse version based on tool output format
	if strings.Contains(name, "claude") {
		// claude-code/0.2.29 darwin-arm64 node-v22.12.0
		parts := strings.Split(output, " ")
		if len(parts) > 0 {
			verParts := strings.Split(parts[0], "/")
			if len(verParts) == 2 {
				return verParts[1], nil
			}
		}
	}

	return output, nil
}

func (tm *ToolManager) InstallTool(name string) error {
	npmPath := tm.getNpmPath()
	if npmPath == "" {
		return fmt.Errorf("npm not found. Please ensure Node.js is installed.")
	}

	home, _ := os.UserHomeDir()
	localNodeDir := filepath.Join(home, ".cceasy", "node")
	
	// Ensure the local node directory exists for prefix usage
	if err := os.MkdirAll(localNodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create local node directory: %w", err)
	}

	var packageName string
	switch name {
	case "claude":
		packageName = "@anthropic-ai/claude-code"
	case "gemini":
		packageName = "@google/gemini-cli"
	case "codex":
		packageName = "@openai/codex"
	default:
		return fmt.Errorf("unknown tool: %s", name)
	}

	// Use --prefix to install to our local folder, avoiding sudo/permission issues
	// This works with both system npm and local npm.
	args := []string{"install", "-g", packageName, "--prefix", localNodeDir}
	
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// On Windows, if we are using the system npm, it might need 'cmd /c'
		if !strings.Contains(strings.ToLower(npmPath), ".cmd") && !strings.Contains(strings.ToLower(npmPath), ".exe") {
			cmd = exec.Command("cmd", append([]string{"/c", npmPath}, args...)...)
		} else {
			cmd = exec.Command(npmPath, args...)
		}
	} else {
		cmd = exec.Command(npmPath, args...)
	}

	// Set environment to include local node bin for the installation process
	localBinDir := filepath.Join(localNodeDir, "bin")
	if runtime.GOOS == "windows" {
		localBinDir = localNodeDir
	}

	env := os.Environ()
	pathFound := false
	for i, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			env[i] = fmt.Sprintf("PATH=%s%c%s", localBinDir, os.PathListSeparator, e[5:])
			pathFound = true
			break
		}
	}
	if !pathFound {
		env = append(env, "PATH="+localBinDir)
	}
	cmd.Env = env

	tm.app.log(fmt.Sprintf("Running installation: %s %s", cmd.Path, strings.Join(cmd.Args[1:], " ")))

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install %s: %v\nOutput: %s", name, err, string(out))
	}
	return nil
}

func (tm *ToolManager) getNpmPath() string {
	// 1. Check local node environment first
	home, _ := os.UserHomeDir()
	var localNpm string
	if runtime.GOOS == "windows" {
		localNpm = filepath.Join(home, ".cceasy", "node", "npm.cmd")
	} else {
		localNpm = filepath.Join(home, ".cceasy", "node", "bin", "npm")
	}

	if _, err := os.Stat(localNpm); err == nil {
		return localNpm
	}

	// 2. Fallback to system npm
	path, err := exec.LookPath("npm")
	if err == nil {
		return path
	}

	return ""
}

func (a *App) InstallTool(name string) error {
	tm := NewToolManager(a)
	return tm.InstallTool(name)
}

func (a *App) CheckToolsStatus() []ToolStatus {
	tm := NewToolManager(a)
	tools := []string{"claude", "gemini", "codex"}
	statuses := make([]ToolStatus, len(tools))
	for i, name := range tools {
		statuses[i] = tm.GetToolStatus(name)
	}
	return statuses
}
