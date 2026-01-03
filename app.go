package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	CurrentLanguage string
	watcher         *fsnotify.Watcher
	testHomeDir     string // For testing purposes
}

var OnConfigChanged func(AppConfig)
var UpdateTrayMenu func(string)

type ModelConfig struct {
	ModelName string `json:"model_name"`
	ModelId   string `json:"model_id"`
	ModelUrl  string `json:"model_url"`
	ApiKey    string `json:"api_key"`
	WireApi   string `json:"wire_api"`
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

type CodeBuddyModel struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	Vendor           string `json:"vendor"`
	ApiKey           string `json:"apiKey"`
	MaxInputTokens   int    `json:"maxInputTokens"`
	MaxOutputTokens  int    `json:"maxOutputTokens"`
	Url              string `json:"url"`
	SupportsToolCall bool   `json:"supportsToolCall"`
	SupportsImages   bool   `json:"supportsImages"`
}

type CodeBuddyFileConfig struct {
	Models          []CodeBuddyModel `json:"Models"`
	AvailableModels []string         `json:"availableModels"`
}

type AppConfig struct {
	Claude           ToolConfig      `json:"claude"`
	Gemini           ToolConfig      `json:"gemini"`
	Codex            ToolConfig      `json:"codex"`
	Opencode         ToolConfig      `json:"opencode"`
	CodeBuddy        ToolConfig      `json:"codebuddy"`
	Qoder            ToolConfig      `json:"qoder"`
	Projects         []ProjectConfig `json:"projects"`
	CurrentProject   string          `json:"current_project"` // ID of the current project
	ActiveTool       string          `json:"active_tool"`     // "claude", "gemini", or "codex"
	HideStartupPopup bool            `json:"hide_startup_popup"`
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
	a.startConfigWatcher()

	// Initialize CodeBuddy config in project directory
	if _, err := a.LoadConfig(); err == nil {
		// a.syncToCodeBuddySettings(config, "")
	}
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
	if a.testHomeDir != "" {
		return a.testHomeDir
	}
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

func (a *App) getOpencodeConfigPaths() (string, string) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "opencode")
	config := filepath.Join(dir, "opencode.json")
	return dir, config
}

func (a *App) clearClaudeConfig() {
	dir, _, legacy := a.getClaudeConfigPaths()
	home, _ := os.UserHomeDir()

	os.RemoveAll(dir)
	os.Remove(legacy)
	os.Remove(filepath.Join(home, ".claude.json.backup"))
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

func (a *App) clearOpencodeConfig() {
	dir, _ := a.getOpencodeConfigPaths()
	os.RemoveAll(dir)
	a.log("Cleared Opencode configuration directory")
}

func (a *App) clearEnvVars() {
	vars := []string{
		"ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL", "ANTHROPIC_AUTH_TOKEN",
		"OPENAI_API_KEY", "OPENAI_BASE_URL", "WIRE_API",
		"GEMINI_API_KEY", "GOOGLE_GEMINI_BASE_URL",
		"OPENCODE_API_KEY", "OPENCODE_BASE_URL",
		"CODEBUDDY_API_KEY", "CODEBUDDY_BASE_URL", "CODEBUDDY_CODE_MAX_OUTPUT_TOKENS",
		"QODER_API_KEY", "QODER_BASE_URL",
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

	// Exclusively use AUTH_TOKEN for custom providers
	env["ANTHROPIC_AUTH_TOKEN"] = selectedModel.ApiKey
	env["CLAUDE_CODE_USE_COLORS"] = "true"
	env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "64000"
	env["MAX_THINKING_TOKENS"] = "31999"

	switch strings.ToLower(selectedModel.ModelName) {
	case "kimi":
		env["ANTHROPIC_BASE_URL"] = "https://api.kimi.com/coding"
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_MODEL"] = selectedModel.ModelId
	case "glm", "glm-4.7":
		env["ANTHROPIC_BASE_URL"] = "https://open.bigmodel.cn/api/anthropic"
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_MODEL"] = selectedModel.ModelId
		settings["permissions"] = map[string]string{"defaultMode": "dontAsk"}
	case "doubao":
		env["ANTHROPIC_BASE_URL"] = "https://ark.cn-beijing.volces.com/api/coding"
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_MODEL"] = selectedModel.ModelId
	case "minimax":
		env["ANTHROPIC_BASE_URL"] = "https://api.minimaxi.com/anthropic"
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_MODEL"] = selectedModel.ModelId
		env["ANTHROPIC_SMALL_FAST_MODEL"] = selectedModel.ModelId
		env["API_TIMEOUT_MS"] = "3000000"
		env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"] = "1"
	case "deepseek":
		env["ANTHROPIC_BASE_URL"] = "https://api.deepseek.com/anthropic"
		modelId := selectedModel.ModelId
		if modelId == "" {
			modelId = "deepseek-chat"
		}
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = modelId
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = modelId
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = modelId
		env["ANTHROPIC_MODEL"] = modelId
	case "gaccode":
		env["ANTHROPIC_BASE_URL"] = "https://gaccode.com/claudecode"
		modelId := selectedModel.ModelId
		if modelId == "" {
			modelId = "sonnet"
		}
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = modelId
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = modelId
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = modelId
		env["ANTHROPIC_MODEL"] = modelId
	default:
		env["ANTHROPIC_BASE_URL"] = selectedModel.ModelUrl
		env["ANTHROPIC_MODEL"] = selectedModel.ModelId
	}

	settings["env"] = env

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	// Check if settings file needs update
	if existingData, err := os.ReadFile(settingsPath); err == nil {
		if bytes.Equal(existingData, data) {
			// Settings file is already up to date, skip main settings write
			// But still need to update .claude.json for API key responses
			goto updateLegacyJson
		}
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return err
	}

	// 2. Sync to ~/.claude.json for customApiKeyResponses
updateLegacyJson:
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

	// Check if legacy file needs update
	if existingData, err := os.ReadFile(legacyPath); err == nil {
		if bytes.Equal(existingData, data2) {
			return nil
		}
	}

	return os.WriteFile(legacyPath, data2, 0644)
}

func (a *App) syncToCodexSettings(config AppConfig) error {
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

	dir, authPath := a.getCodexConfigPaths()

	if strings.ToLower(selectedModel.ModelName) == "original" {
		a.clearCodexConfig()
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create auth.json
	authData := map[string]string{
		"OPENAI_API_KEY": selectedModel.ApiKey,
	}
	authJson, err := json.MarshalIndent(authData, "", "  ")
	if err != nil {
		return err
	}

	// Check if auth.json needs update
	if existingData, err := os.ReadFile(authPath); err == nil {
		if bytes.Equal(existingData, authJson) {
			// Auth file is already up to date, skip writing
			goto writeConfigToml
		}
	}

	if err := os.WriteFile(authPath, authJson, 0644); err != nil {
		return err
	}

	// Create config.toml
writeConfigToml:
	configPath := filepath.Join(dir, "config.toml")
	baseUrl := selectedModel.ModelUrl
	
	var configToml string
	if strings.ToLower(selectedModel.ModelName) == "aigocode" {
		if baseUrl == "" {
			baseUrl = "https://api.aigocode.com/openai"
		}
		modelId := selectedModel.ModelId
		if modelId == "" {
			modelId = "gpt-5.2-codex"
		}
		configToml = fmt.Sprintf(`model_provider = "aigocode"
model = "%s"
model_reasoning_effort = "high"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.aigocode]
name = "aigocode"
base_url = "%s"
wire_api = "responses"
requires_openai_auth = true
`, modelId, baseUrl)
	} else if strings.ToLower(selectedModel.ModelName) == "deepseek" {
		if baseUrl == "" {
			baseUrl = "https://api.deepseek.com/v1"
		}
		modelId := selectedModel.ModelId
		if modelId == "" {
			modelId = "deepseek-chat"
		}
		configToml = fmt.Sprintf(`model_provider = "deepseek"
model = "%s"
model_reasoning_effort = "xhigh"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.deepseek]
name = "deepseek"
base_url = "%s"
wire_api = "chat"
request_max_retries = 4
stream_max_retries = 8
stream_idle_timeout_ms = 120000
`, modelId, baseUrl)
	} else if strings.ToLower(selectedModel.ModelName) == "glm" {
		if baseUrl == "" {
			baseUrl = "https://open.bigmodel.cn/api/paas/v4"
		}
		modelId := selectedModel.ModelId
		if modelId == "" {
			modelId = "glm-4.7"
		}
		configToml = fmt.Sprintf(`model_provider = "glm"
model = "%s"
model_reasoning_effort = "xhigh"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.glm]
name = "glm"
base_url = "%s"
wire_api = "chat"
request_max_retries = 4
stream_max_retries = 8
stream_idle_timeout_ms = 120000
`, modelId, baseUrl)
	} else if strings.ToLower(selectedModel.ModelName) == "doubao" {
		if baseUrl == "" {
			baseUrl = "https://ark.cn-beijing.volces.com/api/coding/v3"
		}
		modelId := selectedModel.ModelId
		if modelId == "" {
			modelId = "doubao-seed-code-preview-latest"
		}
		configToml = fmt.Sprintf(`model_provider = "doubao"
model = "%s"
model_reasoning_effort = "xhigh"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.doubao]
name = "doubao"
base_url = "%s"
wire_api = "chat"
request_max_retries = 4
stream_max_retries = 8
stream_idle_timeout_ms = 120000
`, modelId, baseUrl)
	} else if strings.ToLower(selectedModel.ModelName) == "kimi" {
		if baseUrl == "" {
			baseUrl = "https://api.kimi.com/coding/v1"
		}
		modelId := selectedModel.ModelId
		if modelId == "" {
			modelId = "kimi-for-coding"
		}
		configToml = fmt.Sprintf(`model_provider = "kimi"
model = "%s"
model_reasoning_effort = "xhigh"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.kimi]
name = "kimi"
base_url = "%s"
wire_api = "chat"
request_max_retries = 4
stream_max_retries = 8
stream_idle_timeout_ms = 120000
`, modelId, baseUrl)
	} else if strings.ToLower(selectedModel.ModelName) == "minimax" {
		if baseUrl == "" {
			baseUrl = "https://api.minimaxi.com/v1"
		}
		modelId := selectedModel.ModelId
		if modelId == "" {
			modelId = "MiniMax-M2.1"
		}
		configToml = fmt.Sprintf(`model_provider = "minimax"
model = "%s"
model_reasoning_effort = "xhigh"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.minimax]
name = "minimax"
base_url = "%s"
wire_api = "chat"
request_max_retries = 4
stream_max_retries = 8
stream_idle_timeout_ms = 120000
`, modelId, baseUrl)
	} else {
		if baseUrl == "" {
			baseUrl = "https://api.aicodemirror.com/api/codex/backend-api/codex"
		}
		modelId := selectedModel.ModelId
		if modelId == "" {
			modelId = "gpt-5.2-codex"
		}
		configToml = fmt.Sprintf(`model_provider = "aicodemirror"
model = "%s"
model_reasoning_effort = "xhigh"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.aicodemirror]
name = "aicodemirror"
base_url = "%s"
wire_api = "responses"
`, modelId, baseUrl)
	}

	if selectedModel.IsCustom || (strings.ToLower(selectedModel.ModelName) != "aigocode" && 
		strings.ToLower(selectedModel.ModelName) != "deepseek" && 
		strings.ToLower(selectedModel.ModelName) != "glm" && 
		strings.ToLower(selectedModel.ModelName) != "doubao" && 
		strings.ToLower(selectedModel.ModelName) != "kimi" && 
		strings.ToLower(selectedModel.ModelName) != "minimax" && 
		strings.ToLower(selectedModel.ModelName) != "aicodemirror") {
		// --- CUSTOM OR OTHER PROVIDERS ---
		wireApi := selectedModel.WireApi
		if wireApi == "" {
			wireApi = "chat"
		}
		
		providerName := strings.ToLower(selectedModel.ModelName)
		if providerName == "" || providerName == "custom" {
			providerName = "custom"
		}

		modelId := selectedModel.ModelId
		if modelId == "" {
			modelId = "gpt-5.2-codex"
		}

		configToml = fmt.Sprintf(`model_provider = "%s"
model = "%s"
model_reasoning_effort = "high"
disable_response_storage = true
preferred_auth_method = "apikey"

[model_providers.%s]
name = "%s"
base_url = "%s"
wire_api = "%s"
`, providerName, modelId, providerName, providerName, baseUrl, wireApi)
	}

	configBytes := []byte(configToml)

	// Check if config.toml needs update
	if existingData, err := os.ReadFile(configPath); err == nil {
		if bytes.Equal(existingData, configBytes) {
			return nil
		}
	}

	return os.WriteFile(configPath, configBytes, 0644)
}

func (a *App) syncToOpencodeSettings(config AppConfig) error {
	var selectedModel *ModelConfig
	for _, m := range config.Opencode.Models {
		if m.ModelName == config.Opencode.CurrentModel {
			selectedModel = &m
			break
		}
	}

	if selectedModel == nil {
		return fmt.Errorf("selected opencode model not found")
	}

	dir, configPath := a.getOpencodeConfigPaths()

	if strings.ToLower(selectedModel.ModelName) == "original" {
		a.clearOpencodeConfig()
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	baseUrl := selectedModel.ModelUrl
	modelId := selectedModel.ModelId
	providerName := selectedModel.ModelName

	// Fallback logic for Opencode (align with Codex providers)
	if modelId == "" {
		switch strings.ToLower(providerName) {
		case "deepseek":
			modelId = "deepseek-chat"
			if baseUrl == "" { baseUrl = "https://api.deepseek.com/v1" }
		case "glm":
			modelId = "glm-4.7"
			if baseUrl == "" { baseUrl = "https://open.bigmodel.cn/api/paas/v4" }
		case "doubao":
			modelId = "doubao-seed-code-preview-latest"
			if baseUrl == "" { baseUrl = "https://ark.cn-beijing.volces.com/api/coding/v3" }
		case "kimi":
			modelId = "kimi-for-coding"
			if baseUrl == "" { baseUrl = "https://api.kimi.com/coding/v1" }
		case "minimax":
			modelId = "MiniMax-M2.1"
			if baseUrl == "" { baseUrl = "https://api.minimaxi.com/v1" }
		default:
			modelId = "opencode-1.0"
			if baseUrl == "" { baseUrl = "https://api.aicodemirror.com/api/opencode/v1" }
		}
	}

	// Build the JSON structure
	opencodeJson := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
		"provider": map[string]interface{}{
			"myprovider": map[string]interface{}{
				"npm": "@ai-sdk/openai-compatible",
				"name": providerName,
				"options": map[string]interface{}{
					"baseURL": baseUrl,
					"apiKey": selectedModel.ApiKey,
					"maxTokens": 8192,
				},
				"models": map[string]interface{}{
					modelId: map[string]interface{}{
						"name": modelId,
						"limit": map[string]interface{}{
							"context": 8192,
							"output":  8192,
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(opencodeJson, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
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

	// Use compact JSON format for faster serialization
	configJson, err := json.Marshal(configData)
	if err != nil {
		return err
	}

	// Check if file exists and has same content to avoid unnecessary writes
	if existingData, err := os.ReadFile(configPath); err == nil {
		if bytes.Equal(existingData, configJson) {
			// File already has the correct content, skip writing
			return nil
		}
	}

	return os.WriteFile(configPath, configJson, 0644)
}

func (a *App) syncToCodeBuddySettings(config AppConfig, projectPath string) error {
	if projectPath == "" {
		projectPath = a.GetCurrentProjectPath()
	}
	
	if projectPath == "" {
		return nil
	}

	cbDir := filepath.Join(projectPath, ".codebuddy")
	if err := os.MkdirAll(cbDir, 0755); err != nil {
		return err
	}

	cbFilePath := filepath.Join(cbDir, "models.json")

	var cbModels []CodeBuddyModel
	var availableModelIds []string

	for _, m := range config.CodeBuddy.Models {
		// Only sync the currently selected model
		if m.ModelName != config.CodeBuddy.CurrentModel {
			continue
		}

		if strings.ToLower(m.ModelName) == "original" {
			continue
		}

		vendor := strings.ToLower(m.ModelName)
		
		idStr := m.ModelId
		if idStr == "" {
			switch vendor {
			case "deepseek":
				idStr = "deepseek-chat"
			case "glm":
				idStr = "glm-4.7"
			case "doubao":
				idStr = "doubao-seed-code-preview-latest"
			case "kimi":
				idStr = "kimi-for-coding"
			case "minimax":
				idStr = "MiniMax-M2.1"
			default:
				idStr = vendor + "-model"
			}
		}

		modelIds := strings.Split(idStr, ",")
		
		modelUrl := m.ModelUrl
		if modelUrl != "" && !strings.HasSuffix(modelUrl, "/chat/completions") {
			if strings.HasSuffix(modelUrl, "/") {
				modelUrl += "chat/completions"
			} else {
				modelUrl += "/chat/completions"
			}
		}

		for _, id := range modelIds {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}

			availableModelIds = append(availableModelIds, id)
			cbModels = append(cbModels, CodeBuddyModel{
				Id:               id,
				Name:             id,
				Vendor:           vendor,
				ApiKey:           m.ApiKey,
				MaxInputTokens:   200000,
				MaxOutputTokens:  8192,
				Url:              modelUrl,
				SupportsToolCall: true,
				SupportsImages:   true,
			})
		}
	}

	cbConfig := CodeBuddyFileConfig{
		Models:          cbModels,
		AvailableModels: availableModelIds,
	}

	data, err := json.MarshalIndent(cbConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cbFilePath, data, 0644)
}

func (a *App) syncToQoderSettings(config AppConfig, projectPath string) error {
	if projectPath == "" {
		projectPath = a.GetCurrentProjectPath()
	}
	
	if projectPath == "" {
		return nil
	}

	qDir := filepath.Join(projectPath, ".qoder")
	if err := os.MkdirAll(qDir, 0755); err != nil {
		return err
	}

	qFilePath := filepath.Join(qDir, "models.json")

	var qModels []CodeBuddyModel
	var availableModelIds []string

	for _, m := range config.Qoder.Models {
		// Only sync the currently selected model
		if m.ModelName != config.Qoder.CurrentModel {
			continue
		}

		if strings.ToLower(m.ModelName) == "original" {
			continue
		}

		vendor := strings.ToLower(m.ModelName)
		
		idStr := m.ModelId
		if idStr == "" {
			switch vendor {
			case "deepseek":
				idStr = "deepseek-chat"
			case "glm":
				idStr = "glm-4.7"
			case "doubao":
				idStr = "doubao-seed-code-preview-latest"
			case "kimi":
				idStr = "kimi-for-coding"
			case "minimax":
				idStr = "MiniMax-M2.1"
			default:
				idStr = vendor + "-model"
			}
		}

		modelIds := strings.Split(idStr, ",")
		
		modelUrl := m.ModelUrl
		if modelUrl != "" && !strings.HasSuffix(modelUrl, "/chat/completions") {
			if strings.HasSuffix(modelUrl, "/") {
				modelUrl += "chat/completions"
			} else {
				modelUrl += "/chat/completions"
			}
		}

		for _, id := range modelIds {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}

			availableModelIds = append(availableModelIds, id)
			qModels = append(qModels, CodeBuddyModel{
				Id:               id,
				Name:             id,
				Vendor:           vendor,
				ApiKey:           m.ApiKey,
				MaxInputTokens:   200000,
				MaxOutputTokens:  8192,
				Url:              modelUrl,
				SupportsToolCall: true,
				SupportsImages:   true,
			})
		}
	}

	qConfig := CodeBuddyFileConfig{
		Models:          qModels,
		AvailableModels: availableModelIds,
	}

	data, err := json.MarshalIndent(qConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(qFilePath, data, 0644)
}


func getBaseUrl(selectedModel *ModelConfig) string {
	// If user provided a URL for the selected model, always prefer it.
	if selectedModel.ModelUrl != "" {
		return selectedModel.ModelUrl
	}

	// Otherwise, fall back to hardcoded defaults for known providers that have them.
	baseUrl := "" // Default to empty string
	switch strings.ToLower(selectedModel.ModelName) {
	case "kimi":
		baseUrl = "https://api.kimi.com/coding"
	case "glm", "glm-4.7":
		baseUrl = "https://open.bigmodel.cn/api/anthropic"
	case "doubao":
		baseUrl = "https://ark.cn-beijing.volces.com/api/coding"
	case "minimax":
		baseUrl = "https://api.minimaxi.com/anthropic"
	case "deepseek":
		baseUrl = "https://api.deepseek.com/anthropic"
	case "gaccode":
		baseUrl = "https://gaccode.com/claudecode"
	}
	return baseUrl
}

func (a *App) LaunchTool(toolName string, yoloMode bool, projectDir string) {
	a.log(fmt.Sprintf("Launching %s...", toolName))
	
	if projectDir == "" {
		projectDir = a.GetCurrentProjectPath()
	}

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
		envKey = "ANTHROPIC_AUTH_TOKEN"
		envBaseUrl = "ANTHROPIC_BASE_URL"
		binaryName = "claude"
	case "gemini":
		toolCfg = config.Gemini
		envKey = "GEMINI_API_KEY"
		envBaseUrl = "GOOGLE_GEMINI_BASE_URL"
		binaryName = "gemini"
	case "codex":
		toolCfg = config.Codex
		envKey = "OPENAI_API_KEY"
		envBaseUrl = "OPENAI_BASE_URL"
		binaryName = "codex"
	case "opencode":
		toolCfg = config.Opencode
		envKey = "OPENCODE_API_KEY"
		envBaseUrl = "OPENCODE_BASE_URL"
		binaryName = "opencode"
	case "codebuddy":
		toolCfg = config.CodeBuddy
		envKey = "CODEBUDDY_API_KEY"
		envBaseUrl = "CODEBUDDY_BASE_URL"
		binaryName = "codebuddy"
	case "qoder":
		toolCfg = config.Qoder
		envKey = "QODER_PERSONAL_ACCESS_TOKEN"
		envBaseUrl = "" // Qoder doesn't use a base URL env var in this context
		binaryName = "qodercli"
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

	if selectedModel == nil || toolCfg.CurrentModel == "" {
		title := "提示"
		message := "请先选择一个服务商。"
		if a.CurrentLanguage == "en" {
			title = "Notice"
			message = "Please select a provider first."
		}
		a.ShowMessage(title, message)
		return
	}

	// Ensure ActiveTool is set correctly for syncToSystemEnv
	config.ActiveTool = strings.ToLower(toolName)
	a.syncToSystemEnv(config)

	// 1. CLEAR PROCESS ENV VARS (Safety First - avoid leaks from current process)
	a.clearEnvVars()

	env := make(map[string]string)
	if strings.ToLower(selectedModel.ModelName) != "original" {
		// --- OTHER PROVIDER MODE: WRITE CONFIG & SET ENV ---
		
		// Set process environment variables
		os.Setenv(envKey, selectedModel.ApiKey)
		env[envKey] = selectedModel.ApiKey
		if selectedModel.ModelUrl != "" && envBaseUrl != "" {
			os.Setenv(envBaseUrl, selectedModel.ModelUrl)
			env[envBaseUrl] = selectedModel.ModelUrl
		}

		// Add CODEBUDDY_CODE_MAX_OUTPUT_TOKENS for DeepSeek
		if strings.ToLower(selectedModel.ModelName) == "deepseek" {
			os.Setenv("CODEBUDDY_CODE_MAX_OUTPUT_TOKENS", "8192")
			env["CODEBUDDY_CODE_MAX_OUTPUT_TOKENS"] = "8192"
		}
		
		// Set generic model name env var if applicable
		if selectedModel.ModelId != "" {
			switch strings.ToLower(toolName) {
			case "claude":
				os.Setenv("ANTHROPIC_MODEL", selectedModel.ModelId)
				env["ANTHROPIC_MODEL"] = selectedModel.ModelId
			case "gemini":
				os.Setenv("GOOGLE_GEMINI_MODEL", selectedModel.ModelId)
				env["GOOGLE_GEMINI_MODEL"] = selectedModel.ModelId
			case "codex":
				os.Setenv("OPENAI_MODEL", selectedModel.ModelId)
				env["OPENAI_MODEL"] = selectedModel.ModelId
			case "opencode":
				os.Setenv("OPENCODE_MODEL", selectedModel.ModelId)
				env["OPENCODE_MODEL"] = selectedModel.ModelId
			case "codebuddy":
				// os.Setenv("CODEBUDDY_MODEL", selectedModel.ModelId)
				// env["CODEBUDDY_MODEL"] = selectedModel.ModelId
			case "qoder":
				// Qoder doesn't use model env var
			}
		}

		// Tool-specific configurations
		switch strings.ToLower(toolName) {
		case "claude":
			// Ensure AUTH_TOKEN is unset when using API_KEY to avoid conflict
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
		case "opencode":
			// Opencode might use similar settings to Codex or its own
			a.syncToOpencodeSettings(config)
		case "codebuddy":
			// a.syncToCodeBuddySettings(config, projectDir)
		case "qoder":
			a.syncToQoderSettings(config, projectDir)
		}
	} else {
		// --- ORIGINAL MODE: CLEANUP SPECIFIC TOOL ONLY ---
		
		// Clear process environment variables for this tool
		os.Unsetenv(envKey)
		os.Unsetenv(envBaseUrl)
		if strings.ToLower(toolName) == "claude" {
			os.Unsetenv("ANTHROPIC_AUTH_TOKEN")
			a.clearClaudeConfig()
		} else if strings.ToLower(toolName) == "gemini" {
			a.clearGeminiConfig()
		} else if strings.ToLower(toolName) == "codex" {
			os.Unsetenv("WIRE_API")
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("OPENAI_BASE_URL")
			a.clearCodexConfig()
		} else if strings.ToLower(toolName) == "opencode" {
			os.Unsetenv("OPENCODE_API_KEY")
			os.Unsetenv("OPENCODE_BASE_URL")
			a.clearOpencodeConfig()
		} else if strings.ToLower(toolName) == "codebuddy" {
			os.Unsetenv("CODEBUDDY_API_KEY")
			os.Unsetenv("CODEBUDDY_BASE_URL")
			os.Unsetenv("CODEBUDDY_CODE_MAX_OUTPUT_TOKENS")
			// Codebuddy might need cleanup too if we added a clear function
		} else if strings.ToLower(toolName) == "qoder" {
			os.Unsetenv("QODER_PERSONAL_ACCESS_TOKEN")
			// No base URL to unset for Qoder
		}
		
		a.log(fmt.Sprintf("Running %s in Original mode: Custom configurations cleared.", toolName))
	}

	// Platform specific launch

		a.platformLaunch(binaryName, yoloMode, projectDir, env, selectedModel.ModelId)

	}

func (a *App) log(message string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "env-log", message)
	}
}

func (a *App) getConfigPath() (string, error) {
	if a.testHomeDir != "" {
		return filepath.Join(a.testHomeDir, ".aicoder_config.json"), nil
	}
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
		{ModelName: "Original", ModelId: "", ModelUrl: "", ApiKey: ""},
		{ModelName: "GLM", ModelId: "glm-4.7", ModelUrl: "https://open.bigmodel.cn/api/anthropic", ApiKey: ""},
		{ModelName: "Kimi", ModelId: "kimi-k2-thinking", ModelUrl: "https://api.kimi.com/coding", ApiKey: ""},
		{ModelName: "Doubao", ModelId: "doubao-seed-code-preview-latest", ModelUrl: "https://ark.cn-beijing.volces.com/api/coding", ApiKey: ""},
		{ModelName: "MiniMax", ModelId: "MiniMax-M2.1", ModelUrl: "https://api.minimaxi.com/anthropic", ApiKey: ""},
		{ModelName: "DeepSeek", ModelId: "deepseek-chat", ModelUrl: "https://api.deepseek.com/anthropic", ApiKey: ""},
		{ModelName: "AIgoCode", ModelId: "sonnet", ModelUrl: "https://api.aigocode.com/api", ApiKey: ""},
		{ModelName: "AiCodeMirror", ModelId: "sonnet", ModelUrl: "https://api.aicodemirror.com/api/claudecode", ApiKey: ""},
		{ModelName: "GACCode", ModelId: "sonnet", ModelUrl: "https://gaccode.com/claudecode", ApiKey: ""},
		{ModelName: "Custom", ModelId: "", ModelUrl: "", ApiKey: "", IsCustom: true},
	}
	defaultGeminiModels := []ModelConfig{
		{ModelName: "Original", ModelId: "", ModelUrl: "", ApiKey: ""},
		{ModelName: "AIgoCode", ModelId: "gemini-2.0-flash-exp", ModelUrl: "https://api.aigocode.com/gemini", ApiKey: ""},
		{ModelName: "AiCodeMirror", ModelId: "gemini-2.0-flash-exp", ModelUrl: "https://api.aicodemirror.com/api/gemini", ApiKey: ""},
		{ModelName: "Custom", ModelId: "", ModelUrl: "", ApiKey: "", IsCustom: true},
	}
	defaultCodexModels := []ModelConfig{
		{ModelName: "Original", ModelId: "", ModelUrl: "", ApiKey: ""},
		{ModelName: "AIgoCode", ModelId: "gpt-5.2-codex", ModelUrl: "https://api.aigocode.com/openai", ApiKey: ""},
		{ModelName: "AiCodeMirror", ModelId: "gpt-5.2-codex", ModelUrl: "https://api.aicodemirror.com/api/codex/backend-api/codex", ApiKey: ""},
		{ModelName: "DeepSeek", ModelId: "deepseek-chat", ModelUrl: "https://api.deepseek.com/v1", ApiKey: ""},
		{ModelName: "GLM", ModelId: "glm-4.7", ModelUrl: "https://open.bigmodel.cn/api/paas/v4", ApiKey: ""},
		{ModelName: "Doubao", ModelId: "doubao-seed-code-preview-latest", ModelUrl: "https://ark.cn-beijing.volces.com/api/coding/v3", ApiKey: ""},
		{ModelName: "Kimi", ModelId: "kimi-for-coding", ModelUrl: "https://api.kimi.com/coding/v1", ApiKey: ""},
		{ModelName: "MiniMax", ModelId: "MiniMax-M2.1", ModelUrl: "https://api.minimaxi.com/v1", ApiKey: ""},
		{ModelName: "Custom", ModelId: "", ModelUrl: "", ApiKey: "", IsCustom: true},
	}
	defaultOpencodeModels := []ModelConfig{
		{ModelName: "Original", ModelId: "", ModelUrl: "", ApiKey: ""},
		{ModelName: "DeepSeek", ModelId: "deepseek-chat", ModelUrl: "https://api.deepseek.com/v1", ApiKey: ""},
		{ModelName: "GLM", ModelId: "glm-4.7", ModelUrl: "https://open.bigmodel.cn/api/paas/v4", ApiKey: ""},
		{ModelName: "Doubao", ModelId: "doubao-seed-code-preview-latest", ModelUrl: "https://ark.cn-beijing.volces.com/api/coding/v3", ApiKey: ""},
		{ModelName: "Kimi", ModelId: "kimi-for-coding", ModelUrl: "https://api.kimi.com/coding/v1", ApiKey: ""},
		{ModelName: "MiniMax", ModelId: "MiniMax-M2.1", ModelUrl: "https://api.minimaxi.com/v1", ApiKey: ""},
		{ModelName: "Custom", ModelId: "", ModelUrl: "", ApiKey: "", IsCustom: true},
	}
	defaultQoderModels := []ModelConfig{
		{ModelName: "Original", ModelId: "", ModelUrl: "", ApiKey: ""},
		{ModelName: "Qoder", ModelId: "qoder-1.0", ModelUrl: "https://api.qoder.com/v1", ApiKey: ""},
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
						Opencode: ToolConfig{
							CurrentModel: "AiCodeMirror",
							Models:       defaultOpencodeModels,
						},
						CodeBuddy: ToolConfig{
							CurrentModel: "AiCodeMirror",
							Models:       defaultOpencodeModels,
						},
						Qoder: ToolConfig{
							CurrentModel: "Original",
							Models:       defaultQoderModels,
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
			Opencode: ToolConfig{
				CurrentModel: "AiCodeMirror",
				Models:       defaultOpencodeModels,
			},
			CodeBuddy: ToolConfig{
				CurrentModel: "AiCodeMirror",
				Models:       defaultOpencodeModels,
			},
			Qoder: ToolConfig{
				CurrentModel: "Original",
				Models:       defaultQoderModels,
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
	
	// Helper to ensure a model exists in the list
	ensureModel := func(models *[]ModelConfig, name, url string) {
		for i := range *models {
			if strings.EqualFold((*models)[i].ModelName, name) {
				(*models)[i].ModelName = name // Update to canonical casing
				return
			}
		}
		*models = append(*models, ModelConfig{ModelName: name, ModelUrl: url, ApiKey: ""})
	}

	if config.Gemini.Models == nil || len(config.Gemini.Models) == 0 {
		config.Gemini.Models = defaultGeminiModels
		config.Gemini.CurrentModel = "AiCodeMirror"
	}
	if config.Codex.Models == nil || len(config.Codex.Models) == 0 {
		config.Codex.Models = defaultCodexModels
		config.Codex.CurrentModel = "AiCodeMirror"
	}
	if config.Opencode.Models == nil || len(config.Opencode.Models) == 0 {
		config.Opencode.Models = defaultOpencodeModels
		config.Opencode.CurrentModel = "AiCodeMirror"
	}
	if config.CodeBuddy.Models == nil || len(config.CodeBuddy.Models) == 0 {
		config.CodeBuddy.Models = defaultOpencodeModels
		config.CodeBuddy.CurrentModel = "AiCodeMirror"
	}
	if config.Qoder.Models == nil || len(config.Qoder.Models) == 0 {
		config.Qoder.Models = defaultQoderModels
		config.Qoder.CurrentModel = "Original"
	}

	ensureModel(&config.Claude.Models, "AiCodeMirror", "https://api.aicodemirror.com/api/claudecode")
	ensureModel(&config.Claude.Models, "GACCode", "https://gaccode.com/claudecode")
	ensureModel(&config.Claude.Models, "DeepSeek", "https://api.deepseek.com/anthropic")
	ensureModel(&config.Claude.Models, "Kimi", "https://api.kimi.com/coding")
	ensureModel(&config.Claude.Models, "Doubao", "https://ark.cn-beijing.volces.com/api/coding")
	ensureModel(&config.Claude.Models, "GLM", "https://open.bigmodel.cn/api/anthropic")
	ensureModel(&config.Claude.Models, "MiniMax", "https://api.minimaxi.com/anthropic")
	
	// Deduplicate AiCodeMirror for Claude if both AICodeMirror and AiCodeMirror exist
	dedupeAiCodeMirror := func(models *[]ModelConfig) {
		var newModels []ModelConfig
		foundAi := false
		for _, m := range *models {
			if strings.EqualFold(m.ModelName, "AiCodeMirror") {
				if !foundAi {
					m.ModelName = "AiCodeMirror" // Standardize
					newModels = append(newModels, m)
					foundAi = true
				}
			} else {
				newModels = append(newModels, m)
			}
		}
		*models = newModels
	}
	dedupeAiCodeMirror(&config.Claude.Models)

	ensureModel(&config.Gemini.Models, "AiCodeMirror", "https://api.aicodemirror.com/api/gemini")
	ensureModel(&config.Codex.Models, "AiCodeMirror", "https://api.aicodemirror.com/api/codex/backend-api/codex")
	ensureModel(&config.Codex.Models, "DeepSeek", "https://api.deepseek.com/v1")
	ensureModel(&config.Codex.Models, "GLM", "https://open.bigmodel.cn/api/paas/v4")
	ensureModel(&config.Codex.Models, "Doubao", "https://ark.cn-beijing.volces.com/api/coding/v3")
	ensureModel(&config.Codex.Models, "Kimi", "https://api.kimi.com/coding/v1")
	ensureModel(&config.Codex.Models, "MiniMax", "https://api.minimaxi.com/v1")

	ensureModel(&config.Opencode.Models, "DeepSeek", "https://api.deepseek.com/v1")
	ensureModel(&config.Opencode.Models, "GLM", "https://open.bigmodel.cn/api/paas/v4")
	ensureModel(&config.Opencode.Models, "Doubao", "https://ark.cn-beijing.volces.com/api/coding/v3")
	ensureModel(&config.Opencode.Models, "Kimi", "https://api.kimi.com/coding/v1")
	ensureModel(&config.Opencode.Models, "MiniMax", "https://api.minimaxi.com/v1")

	ensureModel(&config.CodeBuddy.Models, "DeepSeek", "https://api.deepseek.com/v1")
	ensureModel(&config.CodeBuddy.Models, "GLM", "https://open.bigmodel.cn/api/paas/v4")
	ensureModel(&config.CodeBuddy.Models, "Doubao", "https://ark.cn-beijing.volces.com/api/coding/v3")
	ensureModel(&config.CodeBuddy.Models, "Kimi", "https://api.kimi.com/coding/v1")
	ensureModel(&config.CodeBuddy.Models, "MiniMax", "https://api.minimaxi.com/v1")

	// Ensure 'Original' is always present and first
	ensureOriginal := func(models *[]ModelConfig) {
		found := false
		for _, m := range *models {
			if m.ModelName == "Original" {
				found = true
				break
			}
		}
		if !found {
			*models = append([]ModelConfig{{ModelName: "Original", ModelUrl: "", ApiKey: ""}}, *models...)
		}
	}
	
	// Opencode does NOT use common relay providers
	cleanOpencodeModels := func(models *[]ModelConfig) {
		var newModels []ModelConfig
		for _, m := range *models {
			name := strings.ToLower(m.ModelName)
			if name != "aigocode" && name != "aicodemirror" {
				newModels = append(newModels, m)
			}
		}
		*models = newModels
	}

	ensureOriginal(&config.Claude.Models)
	ensureOriginal(&config.Gemini.Models)
	ensureOriginal(&config.Codex.Models)
	ensureOriginal(&config.Opencode.Models)
	ensureOriginal(&config.CodeBuddy.Models)
	ensureOriginal(&config.Qoder.Models)

	cleanOpencodeModels(&config.Opencode.Models)
	cleanOpencodeModels(&config.CodeBuddy.Models)

	// Ensure 'Custom' is always present
	ensureCustom := func(models *[]ModelConfig) {
		found := false
		for _, m := range *models {
			if m.ModelName == "Custom" || m.IsCustom {
				found = true
				break
			}
		}
		if !found {
			*models = append(*models, ModelConfig{ModelName: "Custom", ModelUrl: "", ApiKey: "", IsCustom: true})
		}
	}
	ensureCustom(&config.Claude.Models)
	ensureCustom(&config.Gemini.Models)
	ensureCustom(&config.Codex.Models)
	ensureCustom(&config.Opencode.Models)
	ensureCustom(&config.CodeBuddy.Models)
	// Qoder only has Original and Qoder
	// Preserve existing Qoder key if present
	var existingQoderKey string
	for _, m := range config.Qoder.Models {
		if m.ModelName == "Qoder" {
			existingQoderKey = m.ApiKey
			break
		}
	}
	config.Qoder.Models = defaultQoderModels
	if existingQoderKey != "" {
		for i := range config.Qoder.Models {
			if config.Qoder.Models[i].ModelName == "Qoder" {
				config.Qoder.Models[i].ApiKey = existingQoderKey
				break
			}
		}
	}

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
	moveCustomToLast(&config.Opencode.Models)
	moveCustomToLast(&config.CodeBuddy.Models)

	ensureOriginalFirst(&config.Claude.Models)
	ensureOriginalFirst(&config.Gemini.Models)
	ensureOriginalFirst(&config.Codex.Models)
	ensureOriginalFirst(&config.Opencode.Models)
	ensureOriginalFirst(&config.CodeBuddy.Models)
	ensureOriginalFirst(&config.Qoder.Models)

	// Ensure CurrentModel is valid
	if config.Gemini.CurrentModel == "" {
		config.Gemini.CurrentModel = "Original"
	}
	if config.Codex.CurrentModel == "" {
		config.Codex.CurrentModel = "Original"
	}
	if config.Opencode.CurrentModel == "" {
		config.Opencode.CurrentModel = "Original"
	}
	if config.CodeBuddy.CurrentModel == "" {
		config.CodeBuddy.CurrentModel = "Original"
	}
	if config.Qoder.CurrentModel == "" {
		config.Qoder.CurrentModel = "Original"
	}

	if config.ActiveTool == "" {
		config.ActiveTool = "message"
	}

	// Normalize CurrentModel casing for all tools
	normalizeCurrentModel := func(toolCfg *ToolConfig) {
		for _, m := range toolCfg.Models {
			if strings.EqualFold(m.ModelName, toolCfg.CurrentModel) {
				toolCfg.CurrentModel = m.ModelName
				break
			}
		}
	}
	normalizeCurrentModel(&config.Claude)
	normalizeCurrentModel(&config.Gemini)
	normalizeCurrentModel(&config.Codex)
	normalizeCurrentModel(&config.Opencode)
	normalizeCurrentModel(&config.CodeBuddy)
	normalizeCurrentModel(&config.Qoder)

	return config, nil
}

// getProviderModel gets the model for a specific provider name from a tool config
func getProviderModel(toolConfig *ToolConfig, providerName string) *ModelConfig {
	for i := range toolConfig.Models {
		if strings.EqualFold(toolConfig.Models[i].ModelName, providerName) {
			return &toolConfig.Models[i]
		}
	}
	return nil
}

// syncAllProviderApiKeys synchronizes apikeys of all providers (except 'Original') across all tools
func syncAllProviderApiKeys(a *App, oldConfig, newConfig *AppConfig) {
	// List of tools to sync
	tools := []*ToolConfig{&newConfig.Claude, &newConfig.Gemini, &newConfig.Codex, &newConfig.Opencode, &newConfig.CodeBuddy, &newConfig.Qoder}
	oldTools := []*ToolConfig{&oldConfig.Claude, &oldConfig.Gemini, &oldConfig.Codex, &oldConfig.Opencode, &oldConfig.CodeBuddy, &oldConfig.Qoder}

	// 1. Identify which provider's ApiKey has changed
	var changedProvider string
	var updatedApiKey string
	foundChange := false

	// Iterate through all tools and their models to find a change compared to oldConfig
	for i, tool := range tools {
		oldTool := oldTools[i]

		// Check for ApiKey changes
		for _, model := range tool.Models {
			if strings.EqualFold(model.ModelName, "Original") {
				continue
			}
			
			// Exclude "Custom" providers or any provider marked as IsCustom
			if strings.EqualFold(model.ModelName, "Custom") || model.IsCustom {
				continue
			}

			oldModel := getProviderModel(oldTool, model.ModelName)
			if oldModel != nil {
				// If it existed before, check if ApiKey changed
				if model.ApiKey != oldModel.ApiKey {
					changedProvider = model.ModelName
					updatedApiKey = model.ApiKey
					foundChange = true
					a.log(fmt.Sprintf("Sync: detected %s apikey change in tool config", changedProvider))
					break
				}
			} else {
				// New model added (not in oldTool)
				if model.ApiKey != "" {
					changedProvider = model.ModelName
					updatedApiKey = model.ApiKey
					foundChange = true
					a.log(fmt.Sprintf("Sync: detected new provider %s with apikey", changedProvider))
					break
				}
			}
		}
		if foundChange {
			break
		}
	}

	if foundChange {
		a.log(fmt.Sprintf("Sync: propagating %s apikey to all tools", changedProvider))
		for _, toolCfg := range tools {
			for i := range toolCfg.Models {
				if strings.EqualFold(toolCfg.Models[i].ModelName, changedProvider) {
					toolCfg.Models[i].ApiKey = updatedApiKey
				}
			}
		}
	}
}

func (a *App) SaveConfig(config AppConfig) error {
	// Load old config to compare for sync logic
	var oldConfig AppConfig
	path, _ := a.getConfigPath()
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &oldConfig)
	}

	// Sync all apikeys across all tools before saving
	syncAllProviderApiKeys(a, &oldConfig, &config)

	if err := a.saveToPath(path, config); err != nil {
		return err
	}

	if OnConfigChanged != nil {
		OnConfigChanged(config)
	}

	return nil
}

func (a *App) saveToPath(path string, config AppConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

type UpdateResult struct {
	HasUpdate     bool   `json:"has_update"`
	LatestVersion string `json:"latest_version"`
	ReleaseUrl    string `json:"release_url"`
}

func (a *App) CheckUpdate(currentVersion string) (UpdateResult, error) {
	// Use GitHub API instead of web scraping
	// Updated URL: aicoder instead of cceasy
	url := "https://api.github.com/repos/RapidAI/aicoder/releases/latest"

	a.log(fmt.Sprintf("CheckUpdate: Starting check against %s", url))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		a.log(fmt.Sprintf("CheckUpdate: Failed to create request: %v", err))
		return UpdateResult{LatestVersion: "检查失败", ReleaseUrl: ""}, err
	}
	req.Header.Set("User-Agent", "AICoder")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		a.log(fmt.Sprintf("CheckUpdate: Failed to fetch GitHub API: %v", err))
		return UpdateResult{LatestVersion: "网络错误", ReleaseUrl: ""}, err
	}
	defer resp.Body.Close()

	a.log(fmt.Sprintf("CheckUpdate: HTTP Status: %d", resp.StatusCode))

	// Check HTTP status
	if resp.StatusCode != 200 {
		a.log(fmt.Sprintf("CheckUpdate: GitHub API returned status %d", resp.StatusCode))
		bodyText, _ := io.ReadAll(resp.Body)
		a.log(fmt.Sprintf("CheckUpdate: Response: %s", string(bodyText[:min(len(bodyText), 200)])))
		return UpdateResult{LatestVersion: "API错误", ReleaseUrl: ""}, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		a.log(fmt.Sprintf("CheckUpdate: Failed to read response body: %v", err))
		return UpdateResult{LatestVersion: "读取失败", ReleaseUrl: ""}, err
	}

	// Log raw response for debugging
	a.log(fmt.Sprintf("CheckUpdate: Raw response length: %d bytes", len(body)))
	a.log(fmt.Sprintf("CheckUpdate: Response body: %s", string(body[:min(len(body), 500)])))

	// Parse JSON response
	var release map[string]interface{}
	err = json.Unmarshal(body, &release)
	if err != nil {
		a.log(fmt.Sprintf("CheckUpdate: Failed to parse JSON: %v", err))
		a.log(fmt.Sprintf("CheckUpdate: Response body: %s", string(body[:min(len(body), 500)])))
		return UpdateResult{LatestVersion: "解析失败", ReleaseUrl: ""}, err
	}

	// Log parsed keys
	a.log(fmt.Sprintf("CheckUpdate: Parsed keys: %v", getMapKeys(release)))

	// Extract version from either 'name' or 'tag_name'
	var tagName string

	// Try 'tag_name' field first (e.g., "v2.0.0.2")
	if tag, ok := release["tag_name"].(string); ok && tag != "" {
		tagName = tag
		a.log(fmt.Sprintf("CheckUpdate: Found version in 'tag_name' field: %s", tagName))
	} else if name, ok := release["name"].(string); ok && name != "" {
		// Fallback to 'name' field (e.g., "V2.0.0.2")
		tagName = name
		a.log(fmt.Sprintf("CheckUpdate: Found version in 'name' field: %s", tagName))
	} else {
		a.log(fmt.Sprintf("CheckUpdate: Neither 'name' nor 'tag_name' found. Available: %v", release))
		return UpdateResult{LatestVersion: "找不到版本号", ReleaseUrl: ""}, fmt.Errorf("no version found in release")
	}

	a.log(fmt.Sprintf("CheckUpdate: Using version: %s", tagName))

	// Extract release URL
	htmlURL, _ := release["html_url"].(string)

	// Keep original version with V prefix for display
	displayVersion := strings.TrimSpace(tagName)
	if !strings.HasPrefix(strings.ToUpper(displayVersion), "V") {
		displayVersion = "V" + displayVersion
	}

	// Clean version strings for comparison (lowercase, no V prefix)
	latestVersionForComparison := strings.TrimPrefix(strings.ToLower(tagName), "v")
	cleanCurrent := strings.TrimPrefix(strings.ToLower(currentVersion), "v")
	cleanCurrent = strings.Split(cleanCurrent, " ")[0]

	// Log for debugging
	a.log(fmt.Sprintf("CheckUpdate: Latest version: %s, Current version: %s, Display version: %s", latestVersionForComparison, cleanCurrent, displayVersion))

	// Compare versions
	if compareVersions(latestVersionForComparison, cleanCurrent) > 0 {
		a.log(fmt.Sprintf("CheckUpdate: Update available! %s > %s", latestVersionForComparison, cleanCurrent))
		return UpdateResult{HasUpdate: true, LatestVersion: displayVersion, ReleaseUrl: htmlURL}, nil
	}

	a.log(fmt.Sprintf("CheckUpdate: Already on latest version"))
	return UpdateResult{HasUpdate: false, LatestVersion: displayVersion, ReleaseUrl: htmlURL}, nil
}

// Helper function to get map keys
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

	// Remove ~/.claude.json.backup file
	claudeJsonBackupPath := filepath.Join(home, ".claude.json.backup")
	a.emitRecoverLog(fmt.Sprintf("Checking file: %s", claudeJsonBackupPath))
	if _, err := os.Stat(claudeJsonBackupPath); !os.IsNotExist(err) {
		a.emitRecoverLog("Found .claude.json.backup file. Removing...")
		if err := os.Remove(claudeJsonBackupPath); err != nil && !os.IsNotExist(err) {
			a.emitRecoverLog(fmt.Sprintf("Failed to remove .claude.json.backup file: %v", err))
			return fmt.Errorf("failed to remove .claude.json.backup file: %w", err)
		}
		a.emitRecoverLog("Successfully removed .claude.json.backup file.")
	} else {
		a.emitRecoverLog(".claude.json.backup file not found, skipping.")
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

func (a *App) ClipboardGetText() (string, error) {
	// Try Wails runtime first
	if a.ctx != nil {
		text, err := runtime.ClipboardGetText(a.ctx)
		if err == nil && text != "" {
			return text, nil
		}
	}

	// Fallback for macOS: use pbpaste command
	cmd := exec.Command("pbpaste")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		return out.String(), nil
	}

	return "", nil
}

func (a *App) ReadBBS() (string, error) {
	// Use GitHub API with timestamp to bypass all caches
	url := fmt.Sprintf("https://api.github.com/repos/RapidAI/aicoder/contents/bbs.md?ref=main&t=%d", time.Now().UnixNano())

	// Create a new transport to avoid connection reuse caching
	transport := &http.Transport{
		DisableKeepAlives: true,
		ForceAttemptHTTP2: false,
	}
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "Failed to create request: " + err.Error(), nil
	}

	// GitHub API headers - request raw content directly
	req.Header.Set("Accept", "application/vnd.github.v3.raw")
	req.Header.Set("User-Agent", "AICoder-App")
	req.Header.Set("Cache-Control", "no-cache, no-store")
	req.Header.Set("Pragma", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return "Failed to fetch remote message: " + err.Error(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Remote message unavailable (Status: %d %s)", resp.StatusCode, resp.Status), nil
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

	var cmd *exec.Cmd
	cmd = createVersionCmd(claudePath)
	
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

func (a *App) getLatestNpmVersion(npmPath string, packageName string) (string, error) {
	var cmd *exec.Cmd
	// Use npm view <package> version
	args := []string{"view", packageName, "version"}
	if strings.HasPrefix(strings.ToLower(a.CurrentLanguage), "zh") {
		args = append(args, "--registry=https://registry.npmmirror.com")
	}
	cmd = createNpmInstallCmd(npmPath, args) // Using createNpmInstallCmd as it's a general npm command runner
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
