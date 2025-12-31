package main

import (
	"os"
	"path/filepath"
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

	app := &App{}

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
	os.Setenv("ANTHROPIC_API_KEY", "test")
	os.Setenv("OPENAI_API_KEY", "test")
	os.Setenv("WIRE_API", "test")
	os.Setenv("GEMINI_API_KEY", "test")

	app.clearEnvVars()

	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		t.Errorf("ANTHROPIC_API_KEY was not cleared")
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

	app := &App{}
	
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

	app := &App{}
	
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
