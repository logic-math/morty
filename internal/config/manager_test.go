package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestManagerInterface ensures Loader implements Manager interface.
func TestManagerInterface(t *testing.T) {
	// This test verifies that *Loader implements Manager
	var _ Manager = (*Loader)(nil)

	loader := NewLoader()
	var mgr Manager = loader

	if mgr == nil {
		t.Fatal("Loader does not implement Manager interface")
	}
}

// TestManagerGet tests the Get method with dot notation.
func TestManagerGet(t *testing.T) {
	loader := NewLoader()
	loader.config = &Config{
		Version: "2.0",
		AICli: AICliConfig{
			Command: "ai_cli",
			EnvVar:  "CLAUDE_CODE_CLI",
		},
		Execution: ExecutionConfig{
			MaxRetryCount: 3,
		},
	}

	tests := []struct {
		name        string
		key         string
		expectValue interface{}
		expectErr   bool
	}{
		{
			name:        "get top-level field",
			key:         "version",
			expectValue: "2.0",
			expectErr:   false,
		},
		{
			name:        "get nested field - ai_cli.command",
			key:         "ai_cli.command",
			expectValue: "ai_cli",
			expectErr:   false,
		},
		{
			name:        "get nested field - execution.max_retry_count",
			key:         "execution.max_retry_count",
			expectValue: 3,
			expectErr:   false,
		},
		{
			name:        "get non-existent field",
			key:         "nonexistent",
			expectValue: nil,
			expectErr:   true,
		},
		{
			name:        "get non-existent nested field",
			key:         "ai_cli.nonexistent",
			expectValue: nil,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := loader.Get(tt.key)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Get(%q) expected error but got none", tt.key)
				}
			} else {
				if err != nil {
					t.Errorf("Get(%q) unexpected error: %v", tt.key, err)
				}
				if val != tt.expectValue {
					t.Errorf("Get(%q) = %v, want %v", tt.key, val, tt.expectValue)
				}
			}
		})
	}
}

// TestManagerGetWithDefault tests Get with default values.
func TestManagerGetWithDefault(t *testing.T) {
	loader := NewLoader()

	// Test Get with default for non-existent key
	val, err := loader.Get("nonexistent", "default_value")
	if err != nil {
		t.Errorf("Get() with default should not error: %v", err)
	}
	if val != "default_value" {
		t.Errorf("Get() with default = %v, want 'default_value'", val)
	}

	// Test Get without default for non-existent key (should error)
	_, err = loader.Get("nonexistent")
	if err == nil {
		t.Error("Get() without default should error for non-existent key")
	}
}

// TestManagerGetString tests the GetString method.
func TestManagerGetString(t *testing.T) {
	loader := NewLoader()
	loader.config = &Config{
		AICli: AICliConfig{
			Command: "test_command",
		},
	}

	// Test GetString returns correct value
	if got := loader.GetString("ai_cli.command"); got != "test_command" {
		t.Errorf("GetString() = %v, want 'test_command'", got)
	}

	// Test GetString with default for non-existent key
	if got := loader.GetString("nonexistent", "default"); got != "default" {
		t.Errorf("GetString() with default = %v, want 'default'", got)
	}

	// Test GetString without default for non-existent key
	if got := loader.GetString("nonexistent"); got != "" {
		t.Errorf("GetString() without default = %v, want empty string", got)
	}
}

// TestManagerGetInt tests the GetInt method.
func TestManagerGetInt(t *testing.T) {
	loader := NewLoader()
	loader.config = &Config{
		Execution: ExecutionConfig{
			MaxRetryCount: 42,
		},
	}

	// Test GetInt returns correct value
	if got := loader.GetInt("execution.max_retry_count"); got != 42 {
		t.Errorf("GetInt() = %v, want 42", got)
	}

	// Test GetInt with default for non-existent key
	if got := loader.GetInt("nonexistent", 10); got != 10 {
		t.Errorf("GetInt() with default = %v, want 10", got)
	}

	// Test GetInt without default for non-existent key
	if got := loader.GetInt("nonexistent"); got != 0 {
		t.Errorf("GetInt() without default = %v, want 0", got)
	}

	// Test GetInt with type conversion from float64
	loader.config = &Config{
		Execution: ExecutionConfig{
			MaxRetryCount: 0, // will test with manual set
		},
	}
	loader.Set("execution.max_retry_count", float64(100))
	if got := loader.GetInt("execution.max_retry_count"); got != 100 {
		t.Errorf("GetInt() with float64 conversion = %v, want 100", got)
	}
}

// TestManagerGetBool tests the GetBool method.
func TestManagerGetBool(t *testing.T) {
	loader := NewLoader()
	loader.config = &Config{
		Execution: ExecutionConfig{
			AutoGitCommit: true,
		},
	}

	// Test GetBool returns correct value
	if got := loader.GetBool("execution.auto_git_commit"); got != true {
		t.Errorf("GetBool() = %v, want true", got)
	}

	// Test GetBool with default for non-existent key
	if got := loader.GetBool("nonexistent", true); got != true {
		t.Errorf("GetBool() with default = %v, want true", got)
	}

	// Test GetBool without default for non-existent key
	if got := loader.GetBool("nonexistent"); got != false {
		t.Errorf("GetBool() without default = %v, want false", got)
	}

	// Test GetBool with string parsing (via Get method)
	loader.config = &Config{
		Execution: ExecutionConfig{
			AutoGitCommit: true,
		},
	}
	if got := loader.GetBool("execution.auto_git_commit"); got != true {
		t.Errorf("GetBool() = %v, want true", got)
	}
}

// TestManagerGetDuration tests the GetDuration method.
func TestManagerGetDuration(t *testing.T) {
	loader := NewLoader()
	loader.config = &Config{
		AICli: AICliConfig{
			DefaultTimeout: "30m",
		},
	}

	// Test GetDuration returns correct value
	if got := loader.GetDuration("ai_cli.default_timeout"); got != 30*time.Minute {
		t.Errorf("GetDuration() = %v, want 30m", got)
	}

	// Test GetDuration with default for non-existent key
	if got := loader.GetDuration("nonexistent", 5*time.Minute); got != 5*time.Minute {
		t.Errorf("GetDuration() with default = %v, want 5m", got)
	}

	// Test GetDuration without default for non-existent key
	if got := loader.GetDuration("nonexistent"); got != 0 {
		t.Errorf("GetDuration() without default = %v, want 0", got)
	}

	// Test GetDuration with invalid duration string
	loader.Set("ai_cli.default_timeout", "invalid")
	if got := loader.GetDuration("ai_cli.default_timeout", 10*time.Minute); got != 10*time.Minute {
		t.Errorf("GetDuration() with invalid string = %v, want default 10m", got)
	}
}

// TestManagerSet tests the Set method.
func TestManagerSet(t *testing.T) {
	loader := NewLoader()

	tests := []struct {
		name       string
		key        string
		value      interface{}
		expectErr  bool
		expectGet  interface{}
	}{
		{
			name:      "set top-level field",
			key:       "version",
			value:     "3.0",
			expectErr: false,
			expectGet: "3.0",
		},
		{
			name:      "set nested field - ai_cli.command",
			key:       "ai_cli.command",
			value:     "new_command",
			expectErr: false,
			expectGet: "new_command",
		},
		{
			name:      "set nested field - execution.max_retry_count",
			key:       "execution.max_retry_count",
			value:     100,
			expectErr: false,
			expectGet: 100,
		},
		{
			name:      "set non-existent field",
			key:       "nonexistent.field",
			value:     "value",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.Set(tt.key, tt.value)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Set(%q) expected error but got none", tt.key)
				}
			} else {
				if err != nil {
					t.Errorf("Set(%q) unexpected error: %v", tt.key, err)
				}

				// Verify the value was set
				got, err := loader.Get(tt.key)
				if err != nil {
					t.Errorf("Get(%q) after Set error: %v", tt.key, err)
				}
				if got != tt.expectGet {
					t.Errorf("Get(%q) after Set = %v, want %v", tt.key, got, tt.expectGet)
				}
			}
		})
	}
}

// TestManagerSave tests the Save method.
func TestManagerSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create initial config file
	initialConfig := map[string]interface{}{
		"version": "2.0",
		"ai_cli": map[string]interface{}{
			"command": "initial_cli",
		},
	}
	data, _ := json.Marshal(initialConfig)
	os.WriteFile(configPath, data, 0644)

	// Load config
	loader := NewLoader()
	if err := loader.Load(configPath); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Modify config
	loader.Set("ai_cli.command", "updated_cli")

	// Save config
	if err := loader.Save(); err != nil {
		t.Errorf("Save() error: %v", err)
	}

	// Load saved config and verify
	newLoader := NewLoader()
	if err := newLoader.Load(configPath); err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if newLoader.Config().AICli.Command != "updated_cli" {
		t.Errorf("Saved config command = %s, want 'updated_cli'", newLoader.Config().AICli.Command)
	}
}

// TestManagerSaveWithoutConfigFile tests Save without config file path.
func TestManagerSaveWithoutConfigFile(t *testing.T) {
	loader := NewLoader()

	err := loader.Save()
	if err == nil {
		t.Error("Save() without config file should error")
	}
}

// TestManagerSaveTo tests the SaveTo method.
func TestManagerSaveTo(t *testing.T) {
	tempDir := t.TempDir()

	loader := NewLoader()
	loader.config = &Config{
		Version: "2.0",
		AICli: AICliConfig{
			Command: "saved_cli",
			EnvVar:  "SAVED_VAR",
		},
		Execution: ExecutionConfig{
			MaxRetryCount: 5,
		},
	}

	savePath := filepath.Join(tempDir, "saved_config.json")

	// Save to specific path
	if err := loader.SaveTo(savePath); err != nil {
		t.Fatalf("SaveTo() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Fatal("SaveTo() did not create file")
	}

	// Load and verify
	newLoader := NewLoader()
	if err := newLoader.Load(savePath); err != nil {
		t.Fatalf("Load() saved config error: %v", err)
	}

	if newLoader.Config().AICli.Command != "saved_cli" {
		t.Errorf("Loaded command = %s, want 'saved_cli'", newLoader.Config().AICli.Command)
	}

	if newLoader.Config().Execution.MaxRetryCount != 5 {
		t.Errorf("Loaded max_retry_count = %d, want 5", newLoader.Config().Execution.MaxRetryCount)
	}
}

// TestManagerSaveToCreatesDirectory tests that SaveTo creates directories.
func TestManagerSaveToCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	deepPath := filepath.Join(tempDir, "deep", "nested", "config.json")

	loader := NewLoader()
	loader.config = DefaultConfig()

	if err := loader.SaveTo(deepPath); err != nil {
		t.Fatalf("SaveTo() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(deepPath); os.IsNotExist(err) {
		t.Fatal("SaveTo() did not create file in nested directory")
	}
}

// TestManagerPathHelpers tests all path helper methods.
func TestManagerPathHelpers(t *testing.T) {
	loader := NewLoader()

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{
			name:     "GetWorkDir",
			got:      loader.GetWorkDir(),
			expected: ".morty",
		},
		{
			name:     "GetLogDir",
			got:      loader.GetLogDir(),
			expected: ".morty/doing/logs",
		},
		{
			name:     "GetResearchDir",
			got:      loader.GetResearchDir(),
			expected: ".morty/research",
		},
		{
			name:     "GetPlanDir",
			got:      loader.GetPlanDir(),
			expected: DefaultPlanDir,
		},
		{
			name:     "GetStatusFile",
			got:      loader.GetStatusFile(),
			expected: DefaultStateFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s() = %s, want %s", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestManagerGetConfigFile tests the GetConfigFile method.
func TestManagerGetConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.json")

	loader := NewLoader()

	// Before loading, config file should be empty
	if got := loader.GetConfigFile(); got != "" {
		t.Errorf("GetConfigFile() before load = %s, want empty", got)
	}

	// Create and load config
	config := map[string]interface{}{
		"version": "2.0",
		"ai_cli": map[string]interface{}{
			"command": "test_cli",
		},
	}
	data, _ := json.Marshal(config)
	os.WriteFile(configPath, data, 0644)

	if err := loader.Load(configPath); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// After loading, config file should be set
	if got := loader.GetConfigFile(); got != configPath {
		t.Errorf("GetConfigFile() after load = %s, want %s", got, configPath)
	}
}

// TestManagerDefaultValues tests default value handling.
func TestManagerDefaultValues(t *testing.T) {
	loader := NewLoader()
	loader.config = DefaultConfig()

	// Test GetString with default
	if got := loader.GetString("nonexistent", "fallback"); got != "fallback" {
		t.Errorf("GetString() with default = %v, want 'fallback'", got)
	}

	// Test GetInt with default
	if got := loader.GetInt("nonexistent", 42); got != 42 {
		t.Errorf("GetInt() with default = %v, want 42", got)
	}

	// Test GetBool with default
	if got := loader.GetBool("nonexistent", true); got != true {
		t.Errorf("GetBool() with default = %v, want true", got)
	}

	// Test GetDuration with default
	if got := loader.GetDuration("nonexistent", 5*time.Minute); got != 5*time.Minute {
		t.Errorf("GetDuration() with default = %v, want 5m", got)
	}
}

// TestManagerGetWithWrongType tests getting values with wrong types.
func TestManagerGetWithWrongType(t *testing.T) {
	loader := NewLoader()
	loader.config = &Config{
		Version: "2.0",
		AICli: AICliConfig{
			Command: "test_command",
		},
		Execution: ExecutionConfig{
			MaxRetryCount: 10,
			AutoGitCommit: true,
		},
	}

	// GetString on int field should return default
	if got := loader.GetString("execution.max_retry_count", "default"); got != "default" {
		t.Errorf("GetString() on int field = %v, want 'default'", got)
	}

	// GetInt on string field should return default
	if got := loader.GetInt("ai_cli.command", 99); got != 99 {
		t.Errorf("GetInt() on string field = %v, want 99", got)
	}

	// GetBool on string field should return default
	if got := loader.GetBool("ai_cli.command", true); got != true {
		t.Errorf("GetBool() on string field = %v, want true", got)
	}
}

// TestManagerLoadWithMerge tests LoadWithMerge method.
func TestManagerLoadWithMerge(t *testing.T) {
	tempDir := t.TempDir()

	// Create user config
	userConfig := map[string]interface{}{
		"version": "2.0",
		"ai_cli": map[string]interface{}{
			"command": "user_cli",
		},
		"logging": map[string]interface{}{
			"level": "info",
		},
	}
	userConfigPath := filepath.Join(tempDir, "user_config.json")
	userData, _ := json.Marshal(userConfig)
	os.WriteFile(userConfigPath, userData, 0644)

	// Create project config
	workDir := filepath.Join(tempDir, ".morty")
	os.MkdirAll(workDir, 0755)
	projectConfig := map[string]interface{}{
		"logging": map[string]interface{}{
			"level": "debug",
		},
		"execution": map[string]interface{}{
			"max_retry_count": 5,
		},
	}
	projectConfigPath := filepath.Join(workDir, "settings.json")
	projectData, _ := json.Marshal(projectConfig)
	os.WriteFile(projectConfigPath, projectData, 0644)

	// Change to temp directory
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	loader := NewLoader()
	if err := loader.LoadWithMerge(userConfigPath); err != nil {
		t.Fatalf("LoadWithMerge() error: %v", err)
	}

	// Verify merged config
	if loader.Config().AICli.Command != "user_cli" {
		t.Errorf("Command = %s, want 'user_cli'", loader.Config().AICli.Command)
	}

	// Project config should override user config
	if loader.Config().Logging.Level != "debug" {
		t.Errorf("Log level = %s, want 'debug'", loader.Config().Logging.Level)
	}

	if loader.Config().Execution.MaxRetryCount != 5 {
		t.Errorf("Max retry count = %d, want 5", loader.Config().Execution.MaxRetryCount)
	}
}

// TestManagerDotNotationComprehensive tests comprehensive dot notation scenarios.
func TestManagerDotNotationComprehensive(t *testing.T) {
	loader := NewLoader()
	loader.config = DefaultConfig()

	tests := []struct {
		key       string
		wantValue interface{}
		wantErr   bool
	}{
		// Top-level (technically not dot notation but should work)
		{"version", "2.0", false},

		// Single level nesting
		{"ai_cli.command", DefaultAICliCommand, false},
		{"ai_cli.env_var", DefaultAICliEnvVar, false},
		{"logging.level", DefaultLoggingLevel, false},

		// Double nesting
		{"logging.file.enabled", DefaultLoggingFileEnabled, false},
		{"logging.file.path", DefaultLoggingFilePath, false},
		{"logging.file.max_size", DefaultLoggingFileMaxSize, false},

		// Non-existent paths
		{"nonexistent", nil, true},
		{"ai_cli.nonexistent", nil, true},
		{"logging.file.nonexistent", nil, true},
		{"deep.nested.nonexistent.path", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := loader.Get(tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Get(%q) expected error but got none", tt.key)
				}
			} else {
				if err != nil {
					t.Errorf("Get(%q) unexpected error: %v", tt.key, err)
				}
				if got != tt.wantValue {
					t.Errorf("Get(%q) = %v, want %v", tt.key, got, tt.wantValue)
				}
			}
		})
	}
}

// TestManagerSetComprehensive tests comprehensive Set scenarios.
func TestManagerSetComprehensive(t *testing.T) {
	loader := NewLoader()
	loader.config = DefaultConfig()

	// Test setting various types
	tests := []struct {
		key   string
		value interface{}
	}{
		{"version", "3.0"},
		{"ai_cli.command", "updated_cli"},
		{"ai_cli.env_var", "UPDATED_VAR"},
		{"execution.max_retry_count", 10},
		{"execution.auto_git_commit", false},
		{"logging.level", "error"},
		{"logging.file.enabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if err := loader.Set(tt.key, tt.value); err != nil {
				t.Errorf("Set(%q, %v) error: %v", tt.key, tt.value, err)
				return
			}

			got, err := loader.Get(tt.key)
			if err != nil {
				t.Errorf("Get(%q) after Set error: %v", tt.key, err)
				return
			}

			if got != tt.value {
				t.Errorf("Get(%q) after Set = %v, want %v", tt.key, got, tt.value)
			}
		})
	}
}

// BenchmarkManagerGet benchmarks the Get method.
func BenchmarkManagerGet(b *testing.B) {
	loader := NewLoader()
	loader.config = DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loader.Get("ai_cli.command")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkManagerSet benchmarks the Set method.
func BenchmarkManagerSet(b *testing.B) {
	loader := NewLoader()
	loader.config = DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := loader.Set("ai_cli.command", "test")
		if err != nil {
			b.Fatal(err)
		}
	}
}
