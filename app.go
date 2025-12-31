package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	stdruntime "runtime"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	CurrentLanguage string
	watcher         *fsnotify.Watcher
}

var OnConfigChanged func(AppConfig)
var UpdateTrayMenu func(string)

type ModelConfig struct {
	ModelName string `json:"model_name"`
	ModelUrl  string `json:"model_url"`
	ApiKey    string `json:"api_key"`
	IsCustom  bool   `json:"is_custom"`
}

type ProjectConfig struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	YoloMode bool   `json:"yolo_mode"`
}

type ToolConfig struct {
	CurrentModel string        `json:"current_model"`
	Models       []ModelConfig `json:"models"`
}

type AppConfig struct {
	Claude         ToolConfig      `json:"claude"`
	Gemini         ToolConfig      `json:"gemini"`
	Codex          ToolConfig      `json:"codex"`
	Projects       []ProjectConfig `json:"projects"`
	CurrentProject string          `json:"current_project"` // ID of the current project
	ActiveTool     string          `json:"active_tool"`     // "claude", "gemini", or "codex"
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
	a.startConfigWatcher()
	a.startConfigWatcher()
}

func (a *App) startConfigWatcher() {
	var err error
	a.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		a.log("Failed to create file watcher: " + err.Error())
		return
	}

	go func() {
		for {
			select {
			case event, ok := <-a.watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					a.log("Config file modified: " + event.Name)
					// Reload config and emit event
					// We use a debounce-like approach or just reload. 
					// Since Wails events are async, it should be fine.
					// However, writing the config (SaveConfig) also triggers a write event.
					// We should probably check if the change was internal or external, 
					// but that's hard. For now, simply reloading might be okay, 
					// but it could cause a loop if we are not careful.
					// Actually, if we just emit 'config-updated', the frontend updates.
					// But if the frontend updates, it might save... 
					// Let's assume for now this is for external edits.
					
					config, err := a.LoadConfig()
					if err == nil {
						runtime.EventsEmit(a.ctx, "config-updated", config)
						// Also re-sync system envs
						a.syncToSystemEnv(config)
					}
				}
			case err, ok := <-a.watcher.Errors:
				if !ok {
					return
				}
				a.log("Watcher error: " + err.Error())
			}
		}
	}()

	configPath, err := a.getConfigPath()
	if err == nil {
		if err := a.watcher.Add(configPath); err != nil {
			a.log("Failed to watch config file: " + err.Error())
		} else {
			a.log("Watching config file: " + configPath)
		}
	}
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

func (a *App) GetUserHomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

func (a *App) GetCurrentProjectPath() string {
	config, err := a.LoadConfig()
	if err != nil {
		return ""
	}
	
	for _, p := range config.Projects {
		if p.Id == config.CurrentProject {
			return p.Path
		}
	}
	
	if len(config.Projects) > 0 {
		return config.Projects[0].Path
	}
	
	home, _ := os.UserHomeDir()
	return home // Fallback
}

func (a *App) getClaudeConfigPaths() (string, string, string) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".claude")
	settings := filepath.Join(dir, "settings.json")
	legacy := filepath.Join(home, ".claude.json")
	return dir, settings, legacy
}

func (a *App) getGeminiConfigPaths() (string, string, string) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".gemini")
	config := filepath.Join(dir, "config.json")
	legacy := filepath.Join(home, ".geminirc")
	return dir, config, legacy
}

func (a *App) getCodexConfigPaths() (string, string) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".codex")
	auth := filepath.Join(dir, "auth.json")
	// config.toml is also used
	return dir, auth
}

func (a *App) clearClaudeConfig() {
	dir, _, legacy := a.getClaudeConfigPaths()
	os.RemoveAll(dir)
	os.Remove(legacy)
	a.log("Cleared Claude configuration files")
}

func (a *App) clearGeminiConfig() {
	dir, _, legacy := a.getGeminiConfigPaths()
	os.RemoveAll(dir)
	os.Remove(legacy)
	a.log("Cleared Gemini configuration files")
}

func (a *App) clearCodexConfig() {
	dir, _ := a.getCodexConfigPaths()
	os.RemoveAll(dir)
	a.log("Cleared Codex configuration directory")
}

func (a *App) clearEnvVars() {
	vars := []string{
		"ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN",
		"OPENAI_API_KEY", "OPENAI_BASE_URL", "WIRE_API",
		"GEMINI_API_KEY", "GEMINI_BASE_URL",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}
}

func (a *App) syncToClaudeSettings(config AppConfig) error {
	var selectedModel *ModelConfig
	for _, m := range config.Claude.Models {
		if m.ModelName == config.Claude.CurrentModel {
			selectedModel = &m
			break
		}
	}

	if selectedModel == nil {
		return fmt.Errorf("selected model not found")
	}

	dir, settingsPath, legacyPath := a.getClaudeConfigPaths()

	if strings.ToLower(selectedModel.ModelName) == "original" {
		a.clearClaudeConfig()
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	settings := make(map[string]interface{})
	env := make(map[string]string)

	env["ANTHROPIC_AUTH_TOKEN"] = selectedModel.ApiKey
	env["ANTHROPIC_API_KEY"] = selectedModel.ApiKey // Compatibility fallback
	env["CLAUDE_CODE_USE_COLORS"] = "true"

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
	case "minimax":
		env["ANTHROPIC_BASE_URL"] = "https://api.minimaxi.com/anthropic"
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = "MiniMax-M2.1"
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = "MiniMax-M2.1"
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = "MiniMax-M2.1"
		env["ANTHROPIC_MODEL"] = "MiniMax-M2.1"
		env["ANTHROPIC_SMALL_FAST_MODEL"] = "MiniMax-M2.1"
		env["API_TIMEOUT_MS"] = "3000000"
		env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"] = "1"
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
	var claudeJson map[string]interface{}
	
	if jsonData, err := os.ReadFile(legacyPath); err == nil {
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

	return os.WriteFile(legacyPath, data2, 0644)
}

func (a *App) syncToCodexSettings(config AppConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	var selectedModel *ModelConfig
	for _, m := range config.Codex.Models {
		if m.ModelName == config.Codex.CurrentModel {
			selectedModel = &m
			break
		}
	}

	if selectedModel == nil {
		return fmt.Errorf("selected codex model not found")
	}

	// If "Original" is selected, delete all config files and DO NOT write anything
	codexDir := filepath.Join(home, ".codex")
	if strings.ToLower(selectedModel.ModelName) == "original" {
		os.RemoveAll(codexDir)
		return nil
	}

	// Create .codex directory for custom providers
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		return err
	}

	// Create auth.json
	authPath := filepath.Join(codexDir, "auth.json")
	authData := map[string]string{
		"OPENAI_API_KEY": selectedModel.ApiKey,
	}
	authJson, err := json.MarshalIndent(authData, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(authPath, authJson, 0644); err != nil {
		return err
	}

	// Create config.toml
	configPath := filepath.Join(codexDir, "config.toml")
	baseUrl := selectedModel.ModelUrl
	if baseUrl == "" {
		baseUrl = "https://api.aicodemirror.com/api/codex/backend-api/codex"
	}

	configToml := fmt.Sprintf(`model_provider = "aicodemirror"
model = "gpt-5.2-codex"
model_reasoning_effort = "xhigh"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.aicodemirror]
name = "aicodemirror"
base_url = "%s"
wire_api = "responses"
`, baseUrl)

	return os.WriteFile(configPath, []byte(configToml), 0644)
}

func (a *App) syncToGeminiSettings(config AppConfig) error {
	var selectedModel *ModelConfig
	for _, m := range config.Gemini.Models {
		if m.ModelName == config.Gemini.CurrentModel {
			selectedModel = &m
			break
		}
	}

	if selectedModel == nil {
		return fmt.Errorf("selected gemini model not found")
	}

	dir, configPath, _ := a.getGeminiConfigPaths()

	if strings.ToLower(selectedModel.ModelName) == "original" {
		a.clearGeminiConfig()
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	configData := map[string]interface{}{
		"apiKey":  selectedModel.ApiKey,
		"baseUrl": selectedModel.ModelUrl,
	}

	configJson, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, configJson, 0644)
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
	case "minimax":
		baseUrl = "https://api.minimaxi.com/anthropic"
	}
	return baseUrl
}

func (a *App) LaunchTool(toolName string, yoloMode bool, projectDir string) {
	a.log(fmt.Sprintf("Launching %s...", toolName))
	
	config, err := a.LoadConfig()
	if err != nil {
		a.log("Error loading config: " + err.Error())
		return
	}

	var toolCfg ToolConfig
	var envKey, envBaseUrl string
	var binaryName string

	switch strings.ToLower(toolName) {
	case "claude":
		toolCfg = config.Claude
		envKey = "ANTHROPIC_API_KEY"
		envBaseUrl = "ANTHROPIC_BASE_URL"
		binaryName = "claude"
	case "gemini":
		toolCfg = config.Gemini
		envKey = "GEMINI_API_KEY"
		envBaseUrl = "GEMINI_BASE_URL"
		binaryName = "gemini"
	case "codex":
		toolCfg = config.Codex
		envKey = "OPENAI_API_KEY"
		envBaseUrl = "OPENAI_BASE_URL"
		binaryName = "codex"
	default:
		a.log("Unknown tool: " + toolName)
		return
	}

	var selectedModel *ModelConfig
	for _, m := range toolCfg.Models {
		if m.ModelName == toolCfg.CurrentModel {
			selectedModel = &m
			break
		}
	}

		if selectedModel == nil {

			a.log("No model selected.")

			return

		}

	

		// 1. CLEAR EVERYTHING FIRST (Safety First)

		a.clearEnvVars()

		// Also clear tool configs to ensure a fresh start every time

		switch strings.ToLower(toolName) {

		case "claude":

			a.clearClaudeConfig()

		case "gemini":

			a.clearGeminiConfig()

		case "codex":

			a.clearCodexConfig()

		}

	

		env := make(map[string]string)

		if strings.ToLower(selectedModel.ModelName) != "original" {

			// --- OTHER PROVIDER MODE: WRITE CONFIG & SET ENV ---

			

			// Set process environment variables

			os.Setenv(envKey, selectedModel.ApiKey)

			env[envKey] = selectedModel.ApiKey

			if selectedModel.ModelUrl != "" {

				os.Setenv(envBaseUrl, selectedModel.ModelUrl)

				env[envBaseUrl] = selectedModel.ModelUrl

			}

	

			// Tool-specific configurations

			switch strings.ToLower(toolName) {

			case "claude":

				os.Setenv("ANTHROPIC_AUTH_TOKEN", selectedModel.ApiKey)

				env["ANTHROPIC_AUTH_TOKEN"] = selectedModel.ApiKey

				a.syncToClaudeSettings(config)

			case "gemini":

				a.syncToGeminiSettings(config)

			case "codex":

				os.Setenv("WIRE_API", "responses")

				env["WIRE_API"] = "responses"

				// Ensure OpenAI standard vars for Codex

				os.Setenv("OPENAI_API_KEY", selectedModel.ApiKey)

				env["OPENAI_API_KEY"] = selectedModel.ApiKey

				if selectedModel.ModelUrl != "" {

					os.Setenv("OPENAI_BASE_URL", selectedModel.ModelUrl)

					env["OPENAI_BASE_URL"] = selectedModel.ModelUrl

				}

				a.syncToCodexSettings(config)

			}

		} else {

			// --- ORIGINAL MODE: ALREADY CLEARED ABOVE ---

			a.log("Running in Original mode: All custom configurations cleared.")

		}

	

		// Platform specific launch

		a.platformLaunch(binaryName, yoloMode, projectDir, env)

	}

func (a *App) log(message string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "env-log", message)
	}
}

func (a *App) getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aicoder_config.json"), nil
}

func (a *App) LoadConfig() (AppConfig, error) {
	path, err := a.getConfigPath()
	if err != nil {
		return AppConfig{}, err
	}

	// Helper for default models
	defaultClaudeModels := []ModelConfig{
		{ModelName: "Original", ModelUrl: "", ApiKey: ""},
		{ModelName: "GLM", ModelUrl: "https://open.bigmodel.cn/api/anthropic", ApiKey: ""},
		{ModelName: "kimi", ModelUrl: "https://api.kimi.com/coding", ApiKey: ""},
		{ModelName: "doubao", ModelUrl: "https://ark.cn-beijing.volces.com/api/coding", ApiKey: ""},
		{ModelName: "MiniMax", ModelUrl: "https://api.minimaxi.com/anthropic", ApiKey: ""},
		{ModelName: "AICodeMirror", ModelUrl: "https://api.aicodemirror.com/api/claudecode", ApiKey: ""},
		{ModelName: "Custom", ModelUrl: "", ApiKey: "", IsCustom: true},
	}
	defaultGeminiModels := []ModelConfig{
		{ModelName: "Original", ModelUrl: "", ApiKey: ""},
		{ModelName: "AiCodeMirror", ModelUrl: "https://api.aicodemirror.com/api/gemini", ApiKey: ""},
		{ModelName: "Custom", ModelUrl: "", ApiKey: "", IsCustom: true},
	}
	defaultCodexModels := []ModelConfig{
		{ModelName: "Original", ModelUrl: "", ApiKey: ""},
		{ModelName: "AiCodeMirror", ModelUrl: "https://api.aicodemirror.com/api/codex/backend-api/codex", ApiKey: ""},
		{ModelName: "Custom", ModelUrl: "", ApiKey: "", IsCustom: true},
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Check for old config file for migration
		home, _ := os.UserHomeDir()
		oldPath := filepath.Join(home, ".claude_model_config.json")
		if _, err := os.Stat(oldPath); err == nil {
			// Migrate old config
			data, err := os.ReadFile(oldPath)
			if err == nil {
				var oldConfig struct {
					CurrentModel string          `json:"current_model"`
					Models       []ModelConfig   `json:"models"`
					Projects     []ProjectConfig `json:"projects"`
					CurrentProj  string          `json:"current_project"`
				}
				if err := json.Unmarshal(data, &oldConfig); err == nil {
					config := AppConfig{
						Claude: ToolConfig{
							CurrentModel: oldConfig.CurrentModel,
							Models:       oldConfig.Models,
						},
						Gemini: ToolConfig{
							CurrentModel: "Gemini 1.5 Pro",
							Models:       defaultGeminiModels,
						},
						Codex: ToolConfig{
							CurrentModel: "Codex",
							Models:       defaultCodexModels,
						},
						Projects:       oldConfig.Projects,
						CurrentProject: oldConfig.CurrentProj,
						ActiveTool:     "message",
					}
					a.SaveConfig(config)
					// Optional: os.Remove(oldPath)
					return config, nil
				}
			}
		}

		// Create default config
		defaultConfig := AppConfig{
			Claude: ToolConfig{
				CurrentModel: "GLM",
				Models:       defaultClaudeModels,
			},
			Gemini: ToolConfig{
				CurrentModel: "Gemini 1.5 Pro",
				Models:       defaultGeminiModels,
			},
			Codex: ToolConfig{
				CurrentModel: "Codex",
				Models:       defaultCodexModels,
			},
			Projects: []ProjectConfig{
				{
					Id:       "default",
					Name:     "Project 1",
					Path:     home,
					YoloMode: false,
				},
			},
			CurrentProject: "default",
			ActiveTool:     "message",
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

	// Ensure defaults for new fields
	if config.Claude.CurrentModel == "" && len(config.Claude.Models) > 0 {
		config.Claude.CurrentModel = config.Claude.Models[0].ModelName
	}
	
	// Inject AiCodeMirror if missing
	ensureAiCodeMirror := func(models *[]ModelConfig, name, url string, strictlyOnly bool) {
		if strictlyOnly {
			// For Gemini/Codex, we keep Original, AiCodeMirror AND Custom
			var newModels []ModelConfig
			foundOriginal := false
			foundName := false
			foundCustom := false
			for _, m := range *models {
				if m.ModelName == "Original" {
					newModels = append(newModels, m)
					foundOriginal = true
				} else if m.ModelName == name || m.ModelName == "AiCodeMirror" {
					newModels = append(newModels, m)
					foundName = true
				} else if m.IsCustom || m.ModelName == "Custom" {
					newModels = append(newModels, m)
					foundCustom = true
				}
			}
			if !foundOriginal {
				// Insert Original at the beginning
				newModels = append([]ModelConfig{{ModelName: "Original", ModelUrl: "", ApiKey: ""}}, newModels...)
			}
			if !foundName {
				newModels = append(newModels, ModelConfig{ModelName: name, ModelUrl: url, ApiKey: ""})
			}
			if !foundCustom {
				newModels = append(newModels, ModelConfig{ModelName: "Custom", ModelUrl: "", ApiKey: "", IsCustom: true})
			}
			*models = newModels
			return
		}

		found := false
		foundCustom := false
		foundOriginal := false
		for _, m := range *models {
			if m.ModelName == name {
				found = true
			}
			if m.IsCustom || m.ModelName == "Custom" {
				foundCustom = true
			}
			if m.ModelName == "Original" {
				foundOriginal = true
			}
		}
		if !foundOriginal {
			// Insert Original at the beginning
			*models = append([]ModelConfig{{ModelName: "Original", ModelUrl: "", ApiKey: ""}}, *models...)
		}
		if !found {
			*models = append(*models, ModelConfig{ModelName: name, ModelUrl: url, ApiKey: ""})
		}
		if !foundCustom {
			*models = append(*models, ModelConfig{ModelName: "Custom", ModelUrl: "", ApiKey: "", IsCustom: true})
		}
	}

	if config.Gemini.Models == nil || len(config.Gemini.Models) == 0 {
		config.Gemini.Models = defaultGeminiModels
		config.Gemini.CurrentModel = "AiCodeMirror"
	}
	if config.Codex.Models == nil || len(config.Codex.Models) == 0 {
		config.Codex.Models = defaultCodexModels
		config.Codex.CurrentModel = "AiCodeMirror"
	}

	ensureAiCodeMirror(&config.Claude.Models, "AiCodeMirror", "https://api.aicodemirror.com/api/claudecode", false)
	ensureAiCodeMirror(&config.Gemini.Models, "AiCodeMirror", "https://api.aicodemirror.com/api/gemini", true)
	ensureAiCodeMirror(&config.Codex.Models, "AiCodeMirror", "https://api.aicodemirror.com/api/codex/backend-api/codex", true)

	// Ensure 'Custom' is always last for all tools
	moveCustomToLast := func(models *[]ModelConfig) {
		var customModel *ModelConfig
		var newModels []ModelConfig
		for _, m := range *models {
			if m.IsCustom || m.ModelName == "Custom" {
				m.IsCustom = true // Ensure flag is set
				customModel = &m
			} else {
				newModels = append(newModels, m)
			}
		}
		if customModel != nil {
			newModels = append(newModels, *customModel)
		}
		*models = newModels
	}

	// Ensure 'Original' is always first for all tools
	ensureOriginalFirst := func(models *[]ModelConfig) {
		var originalModel *ModelConfig
		var newModels []ModelConfig
		for i := range *models {
			m := &(*models)[i]
			if m.ModelName == "Original" {
				originalModel = m
			} else {
				newModels = append(newModels, *m)
			}
		}
		if originalModel != nil {
			*models = append([]ModelConfig{*originalModel}, newModels...)
		}
	}

	moveCustomToLast(&config.Claude.Models)
	moveCustomToLast(&config.Gemini.Models)
	moveCustomToLast(&config.Codex.Models)

	ensureOriginalFirst(&config.Claude.Models)
	ensureOriginalFirst(&config.Gemini.Models)
	ensureOriginalFirst(&config.Codex.Models)

	// Ensure CurrentModel is valid after filtering
	config.Gemini.CurrentModel = "AiCodeMirror"
	config.Codex.CurrentModel = "AiCodeMirror"

	if config.ActiveTool == "" {
		config.ActiveTool = "message"
	}

	return config, nil
}

func (a *App) SaveConfig(config AppConfig) error {
	// Sync settings for all tools
	a.syncToClaudeSettings(config)
	a.syncToGeminiSettings(config)
	a.syncToCodexSettings(config)
	
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

type UpdateResult struct {
	HasUpdate     bool   `json:"has_update"`
	LatestVersion string `json:"latest_version"`
}

func (a *App) CheckUpdate(currentVersion string) (UpdateResult, error) {
	url := "https://github.com/RapidAI/cceasy/releases"
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return UpdateResult{}, err
	}
	req.Header.Set("User-Agent", "AICoder")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return UpdateResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UpdateResult{}, err
	}

	// Regex to find versions in the page text
	// Matches ">V1.2.1", "> 1.2.2", etc.
	// We look for a closing tag '>', optional whitespace, optional 'V' or 'v', then the version numbers.
	// This captures standard semver-ish versions like 1.2, 1.2.1, 1.2.1.1005
	re := regexp.MustCompile(`>[\s]*[Vv]?(\d+(?:\.\d+)+)`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	var highestVersion string
	
	for _, match := range matches {
		if len(match) >= 2 {
			ver := match[1]
			// ver is just the number part (e.g. "1.2.1") due to the regex group
			
			// If we haven't found a version yet, or this one is higher
			if highestVersion == "" {
				highestVersion = ver
			} else {
				if compareVersions(ver, highestVersion) > 0 {
					highestVersion = ver
				}
			}
		}
	}

	if highestVersion == "" {
		return UpdateResult{}, fmt.Errorf("no release versions found")
	}

	// Compare versions
	cleanCurrent := strings.TrimPrefix(strings.ToLower(currentVersion), "v")
	cleanCurrent = strings.Split(cleanCurrent, " ")[0]
	
	// highestVersion is already clean (just numbers and dots) from the regex
	
	if compareVersions(highestVersion, cleanCurrent) > 0 {
		return UpdateResult{HasUpdate: true, LatestVersion: highestVersion}, nil
	}

	return UpdateResult{HasUpdate: false, LatestVersion: highestVersion}, nil
}

func (a *App) RecoverCC() error {
	a.emitRecoverLog("Starting recovery process...")

	home, err := os.UserHomeDir()
	if err != nil {
		a.emitRecoverLog(fmt.Sprintf("Error getting home dir: %v", err))
		return err
	}

	// Remove ~/.claude directory
	claudeDir := filepath.Join(home, ".claude")
	a.emitRecoverLog(fmt.Sprintf("Checking directory: %s", claudeDir))
	if _, err := os.Stat(claudeDir); !os.IsNotExist(err) {
		a.emitRecoverLog("Found .claude directory. Removing...")
		if err := os.RemoveAll(claudeDir); err != nil {
			a.emitRecoverLog(fmt.Sprintf("Failed to remove .claude directory: %v", err))
			return fmt.Errorf("failed to remove .claude directory: %w", err)
		}
		a.emitRecoverLog("Successfully removed .claude directory.")
	} else {
		a.emitRecoverLog(".claude directory not found, skipping.")
	}

	// Remove ~/.claude.json file
	claudeJsonPath := filepath.Join(home, ".claude.json")
	a.emitRecoverLog(fmt.Sprintf("Checking file: %s", claudeJsonPath))
	if _, err := os.Stat(claudeJsonPath); !os.IsNotExist(err) {
		a.emitRecoverLog("Found .claude.json file. Removing...")
		if err := os.Remove(claudeJsonPath); err != nil && !os.IsNotExist(err) {
			a.emitRecoverLog(fmt.Sprintf("Failed to remove .claude.json file: %v", err))
			return fmt.Errorf("failed to remove .claude.json file: %w", err)
		}
		a.emitRecoverLog("Successfully removed .claude.json file.")
	} else {
		a.emitRecoverLog(".claude.json file not found, skipping.")
	}

	a.emitRecoverLog("Recovery process completed successfully.")
	return nil
}

func (a *App) emitRecoverLog(msg string) {
	runtime.EventsEmit(a.ctx, "recover-log", msg)
}

func (a *App) ShowMessage(title, message string) {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   title,
		Message: message,
	})
}

func (a *App) ReadBBS() (string, error) {
	url := "https://raw.githubusercontent.com/RapidAI/cceasy/main/bbs.md"
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(url)
	if err != nil {
		return "Failed to fetch remote message: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "Remote message unavailable (Status: " + resp.Status + ")", nil
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Error reading remote content: " + err.Error(), nil
	}

	return string(data), nil
}

// compareVersions returns 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		val1 := 0
		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &val1)
		}

		val2 := 0
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &val2)
		}

		if val1 > val2 {
			return 1
		}
		if val1 < val2 {
			return -1
		}
	}
	return 0
}

func (a *App) getInstalledClaudeVersion(claudePath string) (string, error) {
	// Check if the path exists
	if _, err := os.Stat(claudePath); err != nil {
		// If explicit path fails, try finding it in PATH if it's just "claude"
		if claudePath == "claude" {
			path, err := exec.LookPath("claude")
			if err != nil {
				return "", err
			}
			claudePath = path
		} else {
			return "", err
		}
	}

	cmd := exec.Command(claudePath, "--version")
	// Hide window on Windows
	if stdruntime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// Output format example: claude-code/0.2.29 darwin-arm64 node-v22.12.0
	output := strings.TrimSpace(string(out))
	parts := strings.Split(output, " ")
	if len(parts) > 0 {
		// "claude-code/0.2.29"
		verParts := strings.Split(parts[0], "/")
		if len(verParts) == 2 {
			return verParts[1], nil
		}
		// If output is just the version (unlikely but possible)
		if len(parts) == 1 && strings.Contains(parts[0], ".") {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("unexpected output format: %s", output)
}

func (a *App) getLatestClaudeVersion(npmPath string) (string, error) {
	cmd := exec.Command(npmPath, "view", "@anthropic-ai/claude-code", "version")
	if stdruntime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
