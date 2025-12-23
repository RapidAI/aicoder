package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	CurrentLanguage string
}

var OnConfigChanged func(AppConfig)
var UpdateTrayMenu func(string)

type ModelConfig struct {
	ModelName string `json:"model_name"`
	ModelUrl  string `json:"model_url"`
	ApiKey    string `json:"api_key"`
	IsCustom  bool   `json:"is_custom"`
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
	// Platform specific initialization
	a.platformStartup()
	// Force sync system env vars using current config on startup
	config, _ := a.LoadConfig()
	a.syncToSystemEnv(config)
}

func (a *App) SetLanguage(lang string) {
	a.CurrentLanguage = lang
	if UpdateTrayMenu != nil {
		UpdateTrayMenu(lang)
	}
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
	case "glm", "glm-4.7":
		env["ANTHROPIC_BASE_URL"] = "https://open.bigmodel.cn/api/anthropic"
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = "glm-4.7"
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = "glm-4.7"
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = "glm-4.7"
		env["ANTHROPIC_MODEL"] = "glm-4.7"
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

func getBaseUrl(selectedModel *ModelConfig) string {
	baseUrl := selectedModel.ModelUrl
	// Match the specific URLs used in settings.json
	switch strings.ToLower(selectedModel.ModelName) {
	case "kimi":
		baseUrl = "https://api.kimi.com/coding"
	case "glm", "glm-4.7":
		baseUrl = "https://open.bigmodel.cn/api/anthropic"
	case "doubao":
		baseUrl = "https://ark.cn-beijing.volces.com/api/coding"
	}
	return baseUrl
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
					ModelName: "GLM",
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
				{
					ModelName: "Custom",
					ModelUrl:  "",
					ApiKey:    "",
					IsCustom:  true,
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

	// Ensure ModelUrls are populated and migrate names for existing configs
	hasCustom := false
	for i := range config.Models {
		// Migrate to "GLM" for display
		lowerName := strings.ToLower(config.Models[i].ModelName)
		if lowerName == "glm" || lowerName == "glm-4.7" {
			config.Models[i].ModelName = "GLM"
			if strings.ToLower(config.CurrentModel) == "glm" || strings.ToLower(config.CurrentModel) == "glm-4.7" {
				config.CurrentModel = "GLM"
			}
		}

		if config.Models[i].IsCustom || config.Models[i].ModelName == "Custom" {
			hasCustom = true
			config.Models[i].IsCustom = true
		}
		if config.Models[i].ModelUrl == "" {
			switch strings.ToLower(config.Models[i].ModelName) {
			case "glm", "glm-4.7":
				config.Models[i].ModelUrl = "https://open.bigmodel.cn/api/anthropic"
			case "kimi":
				config.Models[i].ModelUrl = "https://api.kimi.com/coding"
			case "doubao":
				config.Models[i].ModelUrl = "https://ark.cn-beijing.volces.com/api/coding"
			}
		}
	}

	if !hasCustom {
		config.Models = append(config.Models, ModelConfig{
			ModelName: "Custom",
			ModelUrl:  "",
			ApiKey:    "",
			IsCustom:  true,
		})
	}

	return config, nil
}

func (a *App) SaveConfig(config AppConfig) error {
	// Sync to Claude Code settings
	a.syncToClaudeSettings(config)
	// Sync system environment variables
	a.syncToSystemEnv(config)

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
