package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncToClaudeSettings(t *testing.T) {
	// Create a temporary directory for testing
	tmpHome, err := os.MkdirTemp("", "cceasy-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Set environment variable to trick UserHomeDir (or mock the function if possible)
	// Since we can't easily mock UserHomeDir without modifying the App struct to use an interface or variable,
	// we'll modify the test to use a temporary home directory if we can.
	// However, `os.UserHomeDir` reads from environment variables on most systems.
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)
	
	// For Windows, it might check USERPROFILE
	if os.Getenv("USERPROFILE") != "" {
		originalUserProfile := os.Getenv("USERPROFILE")
		defer os.Setenv("USERPROFILE", originalUserProfile)
		os.Setenv("USERPROFILE", tmpHome)
	}

	app := &App{}

	// Define a test configuration
	config := AppConfig{
		CurrentModel: "TestModel",
		Models: []ModelConfig{
			{
				ModelName: "TestModel",
				ApiKey:    "sk-test-key-123",
				ModelUrl:  "https://api.test.com",
			},
		},
	}

	// Run the sync function
	err = app.syncToClaudeSettings(config)
	if err != nil {
		t.Fatalf("syncToClaudeSettings failed: %v", err)
	}

	// 1. Verify settings.json
	settingsPath := filepath.Join(tmpHome, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Errorf("settings.json was not created at %s", settingsPath)
	}

	settingsData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings.json: %v", err)
	}

	var settingsMap map[string]interface{}
	if err := json.Unmarshal(settingsData, &settingsMap); err != nil {
		t.Fatalf("Failed to unmarshal settings.json: %v", err)
	}

	env, ok := settingsMap["env"].(map[string]interface{})
	if !ok {
		t.Fatalf("settings.json missing 'env' object")
	}

	if env["ANTHROPIC_AUTH_TOKEN"] != "sk-test-key-123" {
		t.Errorf("Expected ANTHROPIC_AUTH_TOKEN to be 'sk-test-key-123', got '%v'", env["ANTHROPIC_AUTH_TOKEN"])
	}

	if env["ANTHROPIC_API_KEY"] != "sk-test-key-123" {
		t.Errorf("Expected ANTHROPIC_API_KEY to be 'sk-test-key-123', got '%v'", env["ANTHROPIC_API_KEY"])
	}
	
	if env["CLAUDE_CODE_USE_COLORS"] != "true" {
		t.Errorf("Expected CLAUDE_CODE_USE_COLORS to be 'true', got '%v'", env["CLAUDE_CODE_USE_COLORS"])
	}

	// 2. Verify .claude.json
	claudeJsonPath := filepath.Join(tmpHome, ".claude.json")
	if _, err := os.Stat(claudeJsonPath); os.IsNotExist(err) {
		t.Errorf(".claude.json was not created at %s", claudeJsonPath)
	}

	claudeJsonData, err := os.ReadFile(claudeJsonPath)
	if err != nil {
		t.Fatalf("Failed to read .claude.json: %v", err)
	}

	var claudeJsonMap map[string]interface{}
	if err := json.Unmarshal(claudeJsonData, &claudeJsonMap); err != nil {
		t.Fatalf("Failed to unmarshal .claude.json: %v", err)
	}

	customResponses, ok := claudeJsonMap["customApiKeyResponses"].(map[string]interface{})
	if !ok {
		t.Fatalf(".claude.json missing 'customApiKeyResponses' object")
	}

	approved, ok := customResponses["approved"].([]interface{})
	if !ok {
		t.Fatalf("'approved' field is missing or not an array")
	}

	foundKey := false
	for _, k := range approved {
		if k == "sk-test-key-123" {
			foundKey = true
			break
		}
	}

	if !foundKey {
		t.Errorf("Key 'sk-test-key-123' not found in approved list: %v", approved)
	}
}
