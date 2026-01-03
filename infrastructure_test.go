package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanupFunctions(t *testing.T) {
	// Create a temporary directory for testing
	tmpHome, err := os.MkdirTemp("", "cceasy-infra-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Set environment variables to override UserHomeDir
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)
	
	if os.Getenv("USERPROFILE") != "" {
		originalUserProfile := os.Getenv("USERPROFILE")
		defer os.Setenv("USERPROFILE", originalUserProfile)
		os.Setenv("USERPROFILE", tmpHome)
	}

	app := &App{testHomeDir: tmpHome}

	// 1. Test Claude Cleanup
	claudeDir := filepath.Join(tmpHome, ".claude")
	claudeLegacy := filepath.Join(tmpHome, ".claude.json")
	os.MkdirAll(claudeDir, 0755)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{}"), 0644)
	os.WriteFile(claudeLegacy, []byte("{}"), 0644)

	app.clearClaudeConfig()

	if _, err := os.Stat(claudeDir); !os.IsNotExist(err) {
		t.Errorf("Claude directory was not removed")
	}
	if _, err := os.Stat(claudeLegacy); !os.IsNotExist(err) {
		t.Errorf("Claude legacy file was not removed")
	}

	// 2. Test Gemini Cleanup
	geminiDir := filepath.Join(tmpHome, ".gemini")
	geminiLegacy := filepath.Join(tmpHome, ".geminirc")
	os.MkdirAll(geminiDir, 0755)
	os.WriteFile(filepath.Join(geminiDir, "config.json"), []byte("{}"), 0644)
	os.WriteFile(geminiLegacy, []byte("{}"), 0644)

	app.clearGeminiConfig()

	if _, err := os.Stat(geminiDir); !os.IsNotExist(err) {
		t.Errorf("Gemini directory was not removed")
	}
	if _, err := os.Stat(geminiLegacy); !os.IsNotExist(err) {
		t.Errorf("Gemini legacy file was not removed")
	}

	// 3. Test Codex Cleanup
	codexDir := filepath.Join(tmpHome, ".codex")
	os.MkdirAll(codexDir, 0755)
	os.WriteFile(filepath.Join(codexDir, "auth.json"), []byte("{}"), 0644)

	app.clearCodexConfig()

	if _, err := os.Stat(codexDir); !os.IsNotExist(err) {
		t.Errorf("Codex directory was not removed")
	}

	// 4. Test Env Vars Cleanup
	os.Setenv("ANTHROPIC_AUTH_TOKEN", "test")
	os.Setenv("OPENAI_API_KEY", "test")
	os.Setenv("WIRE_API", "test")
	os.Setenv("GEMINI_API_KEY", "test")

	app.clearEnvVars()

	if os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN was not cleared")
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		t.Errorf("OPENAI_API_KEY was not cleared")
	}
	if os.Getenv("WIRE_API") != "" {
		t.Errorf("WIRE_API was not cleared")
	}
	if os.Getenv("GEMINI_API_KEY") != "" {
		t.Errorf("GEMINI_API_KEY was not cleared")
	}
}

func TestSyncToClaudeSettings_Original(t *testing.T) {
	tmpHome, _ := os.MkdirTemp("", "claude-original-test")
	defer os.RemoveAll(tmpHome)

	os.Setenv("HOME", tmpHome)
	if os.Getenv("USERPROFILE") != "" {
		os.Setenv("USERPROFILE", tmpHome)
	}

	app := &App{testHomeDir: tmpHome}
	
	// Create some files to be deleted
	dir, settings, legacy := app.getClaudeConfigPaths()
	os.MkdirAll(dir, 0755)
	os.WriteFile(settings, []byte("junk"), 0644)
	os.WriteFile(legacy, []byte("junk"), 0644)

	config := AppConfig{
		Claude: ToolConfig{
			CurrentModel: "Original",
			Models: []ModelConfig{
				{ModelName: "Original"},
			},
		},
	}

	err := app.syncToClaudeSettings(config)
	if err != nil {
		t.Fatalf("syncToClaudeSettings failed: %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("Expected .claude directory to be gone")
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("Expected legacy .claude.json to be gone")
	}
}

func TestSyncToGeminiSettings_Original(t *testing.T) {
	tmpHome, _ := os.MkdirTemp("", "gemini-original-test")
	defer os.RemoveAll(tmpHome)

	os.Setenv("HOME", tmpHome)
	if os.Getenv("USERPROFILE") != "" {
		os.Setenv("USERPROFILE", tmpHome)
	}

	app := &App{testHomeDir: tmpHome}
	
	// Create some files to be deleted
	dir, configPath, legacy := app.getGeminiConfigPaths()
	os.MkdirAll(dir, 0755)
	os.WriteFile(configPath, []byte("junk"), 0644)
	os.WriteFile(legacy, []byte("junk"), 0644)

	config := AppConfig{
		Gemini: ToolConfig{
			CurrentModel: "Original",
			Models: []ModelConfig{
				{ModelName: "Original"},
			},
		},
	}

	err := app.syncToGeminiSettings(config)
	if err != nil {
		t.Fatalf("syncToGeminiSettings failed: %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("Expected .gemini directory to be gone")
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("Expected legacy .geminirc to be gone")
	}
}

func TestSyncToCodexSettings_Original(t *testing.T) {
	tmpHome, _ := os.MkdirTemp("", "codex-original-test")
	defer os.RemoveAll(tmpHome)

	os.Setenv("HOME", tmpHome)
	if os.Getenv("USERPROFILE") != "" {
		os.Setenv("USERPROFILE", tmpHome)
	}

	app := &App{testHomeDir: tmpHome}
	
	// Create some files to be deleted
	dir, auth := app.getCodexConfigPaths()
	os.MkdirAll(dir, 0755)
	os.WriteFile(auth, []byte("junk"), 0644)
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("junk"), 0644)

	config := AppConfig{
		Codex: ToolConfig{
			CurrentModel: "Original",
			Models: []ModelConfig{
				{ModelName: "Original"},
			},
		},
	}

	err := app.syncToCodexSettings(config)
	if err != nil {
		t.Fatalf("syncToCodexSettings failed: %v", err)
	}

		if _, err := os.Stat(dir); !os.IsNotExist(err) {

			t.Errorf("Expected .codex directory to be gone")

		}

	}

	

	func TestSyncToCodeBuddySettings(t *testing.T) {

		tmpHome, _ := os.MkdirTemp("", "codebuddy-sync-test")

		defer os.RemoveAll(tmpHome)

	

		os.Setenv("HOME", tmpHome)

		if os.Getenv("USERPROFILE") != "" {

			os.Setenv("USERPROFILE", tmpHome)

		}

	

		app := &App{testHomeDir: tmpHome}

		

		projectPath := filepath.Join(tmpHome, "test-project")

		os.MkdirAll(projectPath, 0755)

	

			config := AppConfig{

	

				CodeBuddy: ToolConfig{

	

					CurrentModel: "DeepSeek",

	

					Models: []ModelConfig{

	

						{ModelName: "Original"},

	

						{ModelName: "DeepSeek", ModelId: "ds-1", ModelUrl: "https://ds.api/v1", ApiKey: "sk-ds"},

	

					},

	

				},

	

				Projects: []ProjectConfig{

				{Id: "p1", Name: "P1", Path: projectPath},

			},

					CurrentProject: "p1",

				}

			

								// Save config so GetCurrentProjectPath can find it

			

								if err := app.SaveConfig(config); err != nil {

			

									t.Fatalf("Failed to save config: %v", err)

			

								}

			

							

			

								err := app.syncToCodeBuddySettings(config, "")

			

								if err != nil {

			

							t.Fatalf("syncToCodeBuddySettings failed: %v", err)

			

						}

			

				

	

		cbFilePath := filepath.Join(projectPath, ".codebuddy", "models.json")

		if _, err := os.Stat(cbFilePath); os.IsNotExist(err) {

			t.Fatalf("Expected .codebuddy/models.json to be created")

		}

	

		data, err := os.ReadFile(cbFilePath)

		if err != nil {

			t.Fatalf("Failed to read models.json: %v", err)

		}

	

		if !strings.Contains(string(data), "ds-1") {

			t.Errorf("Expected model ID ds-1 in models.json")

		}

			if !strings.Contains(string(data), "sk-ds") {

				t.Errorf("Expected API key sk-ds in models.json")

			}

				if !strings.Contains(string(data), "https://ds.api/v1/chat/completions") {

					t.Errorf("Expected completed URL in models.json, got: %s", string(data))

				}

			

								// Test multi-model ID support

			

								config.CodeBuddy.Models[1].ModelId = "model-1, model-2"

			

								app.syncToCodeBuddySettings(config, "")

			

								data, _ = os.ReadFile(cbFilePath)

			

				

				

						if !strings.Contains(string(data), "\"id\": \"model-1\"") || !strings.Contains(string(data), "\"id\": \"model-2\"") {

				

							t.Errorf("Expected both model-1 and model-2 IDs in models.json, got: %s", string(data))

				

						}

				

					}

				

					

				

					func TestSyncAllProviderApiKeys_CustomExclusion(t *testing.T) {

				

						app := &App{}

				

					

				

						// Setup initial config: Claude has a Custom key, Gemini has a different Custom key

				

						oldConfig := AppConfig{

				

							Claude: ToolConfig{

				

								Models: []ModelConfig{

				

									{ModelName: "Custom", ApiKey: "key-claude", IsCustom: true},

				

									{ModelName: "DeepSeek", ApiKey: "key-common"},

				

								},

				

							},

				

							Gemini: ToolConfig{

				

								Models: []ModelConfig{

				

									{ModelName: "Custom", ApiKey: "key-gemini", IsCustom: true},

				

									{ModelName: "DeepSeek", ApiKey: "key-common"},

				

								},

				

							},

				

						}

				

					

				

						// New config: User updates Claude Custom key

				

						newConfig := AppConfig{

				

							Claude: ToolConfig{

				

								Models: []ModelConfig{

				

									{ModelName: "Custom", ApiKey: "key-claude-updated", IsCustom: true},

				

									{ModelName: "DeepSeek", ApiKey: "key-common"},

				

								},

				

							},

				

							Gemini: ToolConfig{

				

								Models: []ModelConfig{

				

									{ModelName: "Custom", ApiKey: "key-gemini", IsCustom: true},

				

									{ModelName: "DeepSeek", ApiKey: "key-common"},

				

								},

				

							},

				

						}

				

					

				

						syncAllProviderApiKeys(app, &oldConfig, &newConfig)

				

					

				

						// Verify: Gemini Custom key should NOT change

				

						if newConfig.Gemini.Models[0].ApiKey != "key-gemini" {

				

							t.Errorf("Custom provider key should not sync! Expected 'key-gemini', got '%s'", newConfig.Gemini.Models[0].ApiKey)

				

						}

				

					

				

						// Verify: Common provider should still sync (if we updated it)

				

						// Let's test common provider update now

				

						newConfig.Claude.Models[1].ApiKey = "key-common-updated"

				

						

				

						// Reset foundChange logic by calling sync again with the updated struct as input

				

						syncAllProviderApiKeys(app, &oldConfig, &newConfig)

				

					

				

								if newConfig.Gemini.Models[1].ApiKey != "key-common-updated" {

				

					

				

									t.Errorf("Common provider key should sync! Expected 'key-common-updated', got '%s'", newConfig.Gemini.Models[1].ApiKey)

				

					

				

								}

				

					

				

							}

				

					

				

						

				

					

				

						func TestDeepSeekConfigGeneration(t *testing.T) {

				

					

				

							// Create a temporary directory for testing

				

					

				

							tmpHome, err := os.MkdirTemp("", "codex-config-test")

				

					

				

							if err != nil {

				

					

				

								t.Fatalf("Failed to create temp dir: %v", err)

				

					

				

							}

				

					

				

							defer os.RemoveAll(tmpHome)

				

					

				

						

				

					

				

							// Mock home directory

				

					

				

							os.Setenv("HOME", tmpHome)

				

					

				

							if os.Getenv("USERPROFILE") != "" {

				

					

				

								defer os.Setenv("USERPROFILE", os.Getenv("USERPROFILE"))

				

					

				

								os.Setenv("USERPROFILE", tmpHome)

				

					

				

							}

				

					

				

						

				

					

				

							app := &App{testHomeDir: tmpHome}

				

					

				

						

				

					

				

							// Setup config for DeepSeek

				

					

				

							config := AppConfig{

				

					

				

								Codex: ToolConfig{

				

					

				

									CurrentModel: "DeepSeek",

				

					

				

									Models: []ModelConfig{

				

					

				

										{

				

					

				

											ModelName: "DeepSeek",

				

					

				

											ModelId:   "deepseek-coder",

				

					

				

											ModelUrl:  "https://api.deepseek.com",

				

					

				

											ApiKey:    "test-key",

				

					

				

										},

				

					

				

									},

				

					

				

								},

				

					

				

							}

				

					

				

						

				

					

				

							// Run sync

				

					

				

							err = app.syncToCodexSettings(config)

				

					

				

							if err != nil {

				

					

				

								t.Fatalf("syncToCodexSettings failed: %v", err)

				

					

				

							}

				

					

				

						

				

					

				

							// Check config.toml content

				

					

				

							codexDir := filepath.Join(tmpHome, ".codex")

				

					

				

							configPath := filepath.Join(codexDir, "config.toml")

				

					

				

						

				

					

				

							data, err := os.ReadFile(configPath)

				

					

				

							if err != nil {

				

					

				

								t.Fatalf("Failed to read config.toml: %v", err)

				

					

				

							}

				

					

				

						

				

					

				

							content := string(data)

				

					

				

							

				

					

				

							// Assertions

				

					

				

							if strings.Contains(content, "[model.limit]") {

				

					

				

								t.Errorf("config.toml should NOT contain '[model.limit]' block")

				

					

				

							}

				

					

				

						

				

					

				

							if !strings.Contains(content, "model = \"deepseek-coder\"") {

				

					

				

								t.Errorf("config.toml should contain 'model = \"deepseek-coder\"'")

				

					

				

							}

				

					

				

						

				

					

				

							if !strings.Contains(content, "model_provider = \"deepseek\"") {

				

					

				

								t.Errorf("config.toml should contain 'model_provider = \"deepseek\"'")

				

					

				

							}

				

					

				

							

				

					

				

							// Verify that we didn't accidentally remove too much

				

					

				

							if !strings.Contains(content, "[model_providers.deepseek]") {

				

					

				

								t.Errorf("config.toml should contain '[model_providers.deepseek]'")

				

					

				

							}

				

					

				

						}

				

					

	