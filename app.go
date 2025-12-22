package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

var OnConfigChanged func(AppConfig)

type ModelConfig struct {
	ModelName string `json:"model_name"`
	ModelUrl  string `json:"model_url"`
	ApiKey    string `json:"api_key"`
}

type AppConfig struct {
	CurrentModel string        `json:"current_model"`
	ProjectDir   string        `json:"project_dir"`
	Models       []ModelConfig `json:"models"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Force sync system env vars using current config on startup
	config, _ := a.LoadConfig()
	a.syncToSystemEnv(config)
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) ResizeWindow(width, height int) {
	runtime.WindowSetSize(a.ctx, width, height)
	runtime.WindowCenter(a.ctx)
}

func (a *App) SelectProjectDir() string {
	selection, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Project Directory",
	})
	if err != nil {
		return ""
	}
	return selection
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
			// Create a window specifically for winget interaction if strictly needed, but silent is better.
			// However, winget often requires admin. This might fail if not run as admin.
			// We will try.
			if out, err := cmd.CombinedOutput(); err != nil {
				a.log("Error installing Node.js: " + string(out))
				// Fallback or stop? We'll try to continue but likely fail.
			} else {
				a.log("Node.js installed successfully.")
				// Update npm path assumption
				npmPath = `C:\Program Files\nodejs\npm.cmd`
			}
		} else {
			a.log("Node.js is installed.")
		}

		a.log("Checking Claude Code...")
		
		// Check if claude exists first
		claudeCheckCmd := exec.Command("claude", "--version")
		claudeCheckCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		claudeExists := claudeCheckCmd.Run() == nil

		if !claudeExists {
			a.log("Claude Code not found. Installing...")
			installCmd := exec.Command(npmPath, "install", "-g", "@anthropic-ai/claude-code")
			installCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			
			if out, err := installCmd.CombinedOutput(); err != nil {
				// Fallback to absolute path
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
			// Always try to update claude-code if it exists
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

func (a *App) syncToClaudeSettings(config AppConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// 1. Sync to ~/.claude/settings.json
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return err
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")

	var selectedModel *ModelConfig
	for _, m := range config.Models {
		if m.ModelName == config.CurrentModel {
			selectedModel = &m
			break
		}
	}

	if selectedModel == nil {
		return fmt.Errorf("selected model not found")
	}

	settings := make(map[string]interface{})
	env := make(map[string]string)

	env["ANTHROPIC_AUTH_TOKEN"] = selectedModel.ApiKey

	switch strings.ToLower(selectedModel.ModelName) {
	case "kimi":
		env["ANTHROPIC_BASE_URL"] = "https://api.kimi.com/coding"
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = "kimi-k2-thinking"
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = "kimi-k2-thinking"
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = "kimi-k2-thinking"
		env["ANTHROPIC_MODEL"] = "kimi-k2-thinking"
	case "glm":
		env["ANTHROPIC_BASE_URL"] = "https://open.bigmodel.cn/api/anthropic"
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = "glm-4.5-air"
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = "glm-4.6"
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = "glm-4.6"
		env["ANTHROPIC_MODEL"] = "glm-4.6"
		settings["permissions"] = map[string]string{"defaultMode": "dontAsk"}
	case "doubao":
		env["ANTHROPIC_BASE_URL"] = "https://ark.cn-beijing.volces.com/api/coding"
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = "doubao-seed-code-preview-latest"
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = "doubao-seed-code-preview-latest"
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = "doubao-seed-code-preview-latest"
		env["ANTHROPIC_MODEL"] = "doubao-seed-code-preview-latest"
	default:
		env["ANTHROPIC_BASE_URL"] = selectedModel.ModelUrl
		env["ANTHROPIC_MODEL"] = selectedModel.ModelName
	}

	settings["env"] = env

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return err
	}

	// 2. Sync to ~/.claude.json for customApiKeyResponses
	claudeJsonPath := filepath.Join(home, ".claude.json")
	var claudeJson map[string]interface{}
	
	if jsonData, err := os.ReadFile(claudeJsonPath); err == nil {
		json.Unmarshal(jsonData, &claudeJson)
	}
	if claudeJson == nil {
		claudeJson = make(map[string]interface{})
	}

	claudeJson["customApiKeyResponses"] = map[string]interface{}{
		"approved": []string{selectedModel.ApiKey},
		"rejected": []string{},
	}

	data2, err := json.MarshalIndent(claudeJson, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(claudeJsonPath, data2, 0644)
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
	
	// Use standard environment
	cmd.Env = os.Environ()
	
	if err := cmd.Start(); err != nil {
		a.log("Failed to launch Claude: " + err.Error())
	}
}

func (a *App) log(message string) {
	runtime.EventsEmit(a.ctx, "env-log", message)
}

func (a *App) getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude_model_config.json"), nil
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

	baseUrl := selectedModel.ModelUrl
	// Match the specific URLs used in settings.json
	switch strings.ToLower(selectedModel.ModelName) {
	case "kimi":
		baseUrl = "https://api.kimi.com/coding"
	case "glm":
		baseUrl = "https://open.bigmodel.cn/api/anthropic"
	case "doubao":
		baseUrl = "https://ark.cn-beijing.volces.com/api/coding"
	}

	// Set persistent environment variables on Windows
	cmd1 := exec.Command("setx", "ANTHROPIC_AUTH_TOKEN", selectedModel.ApiKey)
	cmd1.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd1.Run()

	cmd2 := exec.Command("setx", "ANTHROPIC_BASE_URL", baseUrl)
	cmd2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd2.Run()
}

func (a *App) LoadConfig() (AppConfig, error) {
	path, err := a.getConfigPath()
	if err != nil {
		return AppConfig{}, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create default config
		home, _ := os.UserHomeDir()
		defaultConfig := AppConfig{
			ProjectDir: home,
			Models: []ModelConfig{
				{
					ModelName: "glm",
					ModelUrl:  "https://open.bigmodel.cn/api/anthropic",
					ApiKey:    "your_glm_api_key_here",
				},
				{
					ModelName: "kimi",
					ModelUrl:  "https://api.kimi.com/coding",
					ApiKey:    "your_kimi_api_key_here",
				},
				{
					ModelName: "doubao",
					ModelUrl:  "https://ark.cn-beijing.volces.com/api/coding",
					ApiKey:    "your_doubao_api_key_here",
				},
			},
		}
		if len(defaultConfig.Models) > 0 {
			defaultConfig.CurrentModel = defaultConfig.Models[0].ModelName
		}

		err = a.SaveConfig(defaultConfig)
		return defaultConfig, err
	}

	var config AppConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	if config.CurrentModel == "" && len(config.Models) > 0 {
		config.CurrentModel = config.Models[0].ModelName
	}
	
	if config.ProjectDir == "" {
		config.ProjectDir, _ = os.UserHomeDir()
	}

	return config, nil
}

func (a *App) SaveConfig(config AppConfig) error {
	// Sync to Claude Code settings
	a.syncToClaudeSettings(config)

	path, err := a.getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if OnConfigChanged != nil {
		OnConfigChanged(config)
	}

	return os.WriteFile(path, data, 0644)
}
