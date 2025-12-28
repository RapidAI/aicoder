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

type AppConfig struct {
	CurrentModel   string          `json:"current_model"`
	ProjectDir     string          `json:"project_dir"` // Deprecated, kept for migration
	Models         []ModelConfig   `json:"models"`
	Projects       []ProjectConfig `json:"projects"`
	CurrentProject string          `json:"current_project"` // ID of the current project
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
	case "minimax":
		baseUrl = "https://api.minimaxi.com/anthropic"
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
			Projects: []ProjectConfig{
				{
					Id:       "default",
					Name:     "Project 1",
					Path:     home,
					YoloMode: false,
				},
			},
			CurrentProject: "default",
			Models: []ModelConfig{
				{
					ModelName: "GLM",
					ModelUrl:  "https://open.bigmodel.cn/api/anthropic",
					ApiKey:    "",
				},
				{
					ModelName: "kimi",
					ModelUrl:  "https://api.kimi.com/coding",
					ApiKey:    "",
				},
				{
					ModelName: "doubao",
					ModelUrl:  "https://ark.cn-beijing.volces.com/api/coding",
					ApiKey:    "",
				},
				{
					ModelName: "MiniMax",
					ModelUrl:  "https://api.minimaxi.com/anthropic",
					ApiKey:    "",
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
	
	// Migration: If Projects list is empty but ProjectDir exists
	if len(config.Projects) == 0 {
		pDir := config.ProjectDir
		if pDir == "" {
			pDir, _ = os.UserHomeDir()
		}
		config.Projects = []ProjectConfig{
			{
				Id:       "default",
				Name:     "Project 1",
				Path:     pDir,
				YoloMode: false, // Default to false for safety
			},
		}
		config.CurrentProject = "default"
	}
	
	// Ensure CurrentProject is valid
	validProj := false
	for _, p := range config.Projects {
		if p.Id == config.CurrentProject {
			validProj = true
			break
		}
	}
	if !validProj && len(config.Projects) > 0 {
		config.CurrentProject = config.Projects[0].Id
	}

	// Ensure ModelUrls are populated and migrate names for existing configs
	hasCustom := false
	hasMiniMax := false
	for i := range config.Models {
		// Migrate to "GLM" for display
		lowerName := strings.ToLower(config.Models[i].ModelName)
		if lowerName == "glm" || lowerName == "glm-4.7" {
			config.Models[i].ModelName = "GLM"
			if strings.ToLower(config.CurrentModel) == "glm" || strings.ToLower(config.CurrentModel) == "glm-4.7" {
				config.CurrentModel = "GLM"
			}
		}

		if lowerName == "minimax" {
			hasMiniMax = true
			config.Models[i].ModelName = "MiniMax"
			if strings.ToLower(config.CurrentModel) == "minimax" {
				config.CurrentModel = "MiniMax"
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
			case "minimax":
				config.Models[i].ModelUrl = "https://api.minimaxi.com/anthropic"
			}
		}
	}

	if !hasMiniMax {
		// Insert MiniMax before Custom if Custom exists, otherwise append
		newModels := []ModelConfig{}
		inserted := false
		for _, m := range config.Models {
			if (m.IsCustom || m.ModelName == "Custom") && !inserted {
				newModels = append(newModels, ModelConfig{
					ModelName: "MiniMax",
					ModelUrl:  "https://api.minimaxi.com/anthropic",
					ApiKey:    "your_minimax_api_key_here",
				})
				inserted = true
			}
			newModels = append(newModels, m)
		}
		if !inserted {
			newModels = append(newModels, ModelConfig{
				ModelName: "MiniMax",
				ModelUrl:  "https://api.minimaxi.com/anthropic",
				ApiKey:    "your_minimax_api_key_here",
			})
		}
		config.Models = newModels
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
	req.Header.Set("User-Agent", "Claude-Code-Easy-Suite")

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
		// We can't easily access syscall.SysProcAttr here without importing syscall
		// but since this is just getting version, it should be quick.
		// If we really need to hide it, we might need platform specific helpers.
		// For now, let's assume it's fine or we handle it in platform code.
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
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
