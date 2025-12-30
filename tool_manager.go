package main

import (
	"fmt"
	"os/exec"
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
	if name == "gemini" {
		binaryName = "claude"
	}

	path, err := exec.LookPath(binaryName)
	if err != nil {
		return status
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
	var cmd *exec.Cmd
	switch name {
	case "claude":
		cmd = exec.Command("npm", "install", "-g", "@anthropic-ai/claude-code")
	case "gemini":
		// Gemini uses the same Claude Code CLI as per user instructions
		cmd = exec.Command("npm", "install", "-g", "@anthropic-ai/claude-code")
	case "codex":
		cmd = exec.Command("npm", "install", "-g", "@openai/codex")
	default:
		return fmt.Errorf("unknown tool: %s", name)
	}

	// For Windows, we might need to handle the .cmd extension for npm
	if runtime.GOOS == "windows" {
		cmd.Args = append([]string{"/c", "npm"}, cmd.Args[1:]...)
		cmd.Path = "cmd"
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install %s: %v\nOutput: %s", name, err, string(out))
	}
	return nil
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
