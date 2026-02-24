package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewLoader tests creating a new loader with defaults.
func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}

	if loader.Config() == nil {
		t.Error("NewLoader().Config() returned nil")
	}

	// Verify defaults are set
	cfg := loader.Config()
	if cfg.Version != DefaultVersion {
		t.Errorf("Expected version %s, got %s", DefaultVersion, cfg.Version)
	}
}

// TestLoaderLoad tests loading configuration from a file.
func TestLoaderLoad(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		config      map[string]interface{}
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"version": "2.0",
				"ai_cli": map[string]interface{}{
					"command":       "claude",
					"env_var":       "CLAUDE_CODE_CLI",
					"output_format": "json",
				},
				"logging": map[string]interface{}{
					"level":  "debug",
					"format": "json",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid JSON",
			config: map[string]interface{}{
				"invalid": "this is not valid json {{{",
			},
			wantErr:     true,
			errContains: "parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tempDir, tt.name+"_config.json")
			var data []byte
			if tt.name == "invalid JSON" {
				data = []byte(`{invalid json}`)
			} else {
				var err error
				data, err = json.Marshal(tt.config)
				if err != nil {
					t.Fatalf("Failed to marshal config: %v", err)
				}
			}

			if err := os.WriteFile(configPath, data, 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			loader := NewLoader()
			err := loader.Load(configPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Load() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Load() unexpected error: %v", err)
				}
				if loader.Config().AICli.Command != "claude" {
					t.Errorf("Expected command 'claude', got %s", loader.Config().AICli.Command)
				}
			}
		})
	}
}

// TestLoaderLoadNonExistentFile tests loading a non-existent file.
func TestLoaderLoadNonExistentFile(t *testing.T) {
	loader := NewLoader()
	err := loader.Load("/nonexistent/path/config.json")

	if err == nil {
		t.Error("Load() with non-existent file expected error but got none")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Load() error should contain 'not found', got: %v", err)
	}
}

// TestLoaderLoadWithDefaults tests loading with defaults fallback.
func TestLoaderLoadWithDefaults(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		path         string
		fileExists   bool
		expectLoaded bool
	}{
		{
			name:         "existing file",
			path:         filepath.Join(tempDir, "existing.json"),
			fileExists:   true,
			expectLoaded: true,
		},
		{
			name:         "non-existent file",
			path:         filepath.Join(tempDir, "nonexistent.json"),
			fileExists:   false,
			expectLoaded: false,
		},
		{
			name:         "empty path",
			path:         "",
			fileExists:   false,
			expectLoaded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fileExists {
				config := map[string]interface{}{
					"version": "2.0",
					"ai_cli": map[string]interface{}{
						"command": "custom_cli",
						"env_var": "TEST_VAR",
					},
				}
				data, _ := json.Marshal(config)
				os.WriteFile(tt.path, data, 0644)
			}

			loader := NewLoader()
			err := loader.LoadWithDefaults(tt.path)

			if err != nil {
				t.Errorf("LoadWithDefaults() unexpected error: %v", err)
			}

			if tt.expectLoaded {
				if loader.Config().AICli.Command != "custom_cli" {
					t.Errorf("Expected loaded config command 'custom_cli', got %s", loader.Config().AICli.Command)
				}
			} else {
				if loader.Config().AICli.Command != DefaultAICliCommand {
					t.Errorf("Expected default command %s, got %s", DefaultAICliCommand, loader.Config().AICli.Command)
				}
			}
		})
	}
}

// TestLoaderLoadWithMerge tests the hierarchical config merging.
func TestLoaderLoadWithMerge(t *testing.T) {
	tempDir := t.TempDir()

	// Create user config
	userConfig := map[string]interface{}{
		"version": "2.0",
		"ai_cli": map[string]interface{}{
			"command": "user_cli",
			"env_var": "USER_VAR",
		},
		"logging": map[string]interface{}{
			"level": "info",
		},
	}
	userConfigPath := filepath.Join(tempDir, "user_config.json")
	userData, _ := json.Marshal(userConfig)
	os.WriteFile(userConfigPath, userData, 0644)

	// Create project config directory and file
	workDir := filepath.Join(tempDir, ".morty")
	os.MkdirAll(workDir, 0755)
	projectConfig := map[string]interface{}{
		"version": "2.0",
		"logging": map[string]interface{}{
			"level":  "debug",
			"format": "text",
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
	err := loader.LoadWithMerge(userConfigPath)
	if err != nil {
		t.Fatalf("LoadWithMerge() unexpected error: %v", err)
	}

	cfg := loader.Config()

	if cfg.AICli.Command != "user_cli" {
		t.Errorf("Expected user command 'user_cli', got %s", cfg.AICli.Command)
	}

	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected project logging level 'debug', got %s", cfg.Logging.Level)
	}

	if cfg.Logging.Format != "text" {
		t.Errorf("Expected project logging format 'text', got %s", cfg.Logging.Format)
	}

	if cfg.Execution.MaxRetryCount != 5 {
		t.Errorf("Expected max_retry_count 5, got %d", cfg.Execution.MaxRetryCount)
	}
}

// TestLoaderMergeConfigs tests the config merging logic.
func TestLoaderMergeConfigs(t *testing.T) {
	base := &Config{
		Version: "2.0",
		AICli: AICliConfig{
			Command: "base_cli",
			EnvVar:  "BASE_VAR",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	override := &Config{
		AICli: AICliConfig{
			Command: "override_cli",
		},
		Logging: LoggingConfig{
			Level: "debug",
		},
	}

	result := mergeConfigs(base, override)

	if result.AICli.Command != "override_cli" {
		t.Errorf("Expected command 'override_cli', got %s", result.AICli.Command)
	}

	if result.AICli.EnvVar != "BASE_VAR" {
		t.Errorf("Expected env_var 'BASE_VAR', got %s", result.AICli.EnvVar)
	}

	if result.Logging.Level != "debug" {
		t.Errorf("Expected level 'debug', got %s", result.Logging.Level)
	}

	if result.Logging.Format != "json" {
		t.Errorf("Expected format 'json', got %s", result.Logging.Format)
	}
}

// TestLoaderGet tests the Get method with dot notation.
func TestLoaderGet(t *testing.T) {
	loader := NewLoader()
	loader.config = &Config{
		Version: "2.0",
		AICli: AICliConfig{
			Command: "test_cli",
			EnvVar:  "TEST_VAR",
		},
		Execution: ExecutionConfig{
			MaxRetryCount: 5,
		},
		Logging: LoggingConfig{
			Level: "debug",
		},
	}

	tests := []struct {
		key         string
		expectValue interface{}
		expectErr   bool
	}{
		{"version", "2.0", false},
		{"ai_cli.command", "test_cli", false},
		{"ai_cli.env_var", "TEST_VAR", false},
		{"execution.max_retry_count", 5, false},
		{"logging.level", "debug", false},
		{"nonexistent", nil, true},
		{"ai_cli.nonexistent", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
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

// TestLoaderGetWithDefault tests Get with default value.
func TestLoaderGetWithDefault(t *testing.T) {
	loader := NewLoader()

	val, err := loader.Get("nonexistent", "default_value")
	if err != nil {
		t.Errorf("Get() with default should not error: %v", err)
	}
	if val != "default_value" {
		t.Errorf("Get() with default = %v, want 'default_value'", val)
	}
}

// TestLoaderGetString tests the GetString method.
func TestLoaderGetString(t *testing.T) {
	loader := NewLoader()
	loader.config = &Config{
		AICli: AICliConfig{
			Command: "test_cli",
		},
	}

	if got := loader.GetString("ai_cli.command"); got != "test_cli" {
		t.Errorf("GetString() = %v, want 'test_cli'", got)
	}

	if got := loader.GetString("nonexistent", "default"); got != "default" {
		t.Errorf("GetString() with default = %v, want 'default'", got)
	}
}

// TestLoaderGetInt tests the GetInt method.
func TestLoaderGetInt(t *testing.T) {
	loader := NewLoader()
	loader.config = &Config{
		Execution: ExecutionConfig{
			MaxRetryCount: 10,
		},
	}

	if got := loader.GetInt("execution.max_retry_count"); got != 10 {
		t.Errorf("GetInt() = %v, want 10", got)
	}

	if got := loader.GetInt("nonexistent", 5); got != 5 {
		t.Errorf("GetInt() with default = %v, want 5", got)
	}
}

// TestLoaderGetBool tests the GetBool method.
func TestLoaderGetBool(t *testing.T) {
	loader := NewLoader()
	loader.config = &Config{
		Execution: ExecutionConfig{
			AutoGitCommit: true,
		},
	}

	if got := loader.GetBool("execution.auto_git_commit"); got != true {
		t.Errorf("GetBool() = %v, want true", got)
	}

	if got := loader.GetBool("nonexistent", true); got != true {
		t.Errorf("GetBool() with default = %v, want true", got)
	}
}

// TestLoaderGetDuration tests the GetDuration method.
func TestLoaderGetDuration(t *testing.T) {
	loader := NewLoader()
	loader.config = &Config{
		AICli: AICliConfig{
			DefaultTimeout: "30m",
		},
	}

	if got := loader.GetDuration("ai_cli.default_timeout"); got != 30*time.Minute {
		t.Errorf("GetDuration() = %v, want 30m", got)
	}

	if got := loader.GetDuration("nonexistent", 5*time.Minute); got != 5*time.Minute {
		t.Errorf("GetDuration() with default = %v, want 5m", got)
	}
}

// TestLoaderSet tests the Set method.
func TestLoaderSet(t *testing.T) {
	loader := NewLoader()

	tests := []struct {
		key       string
		value     interface{}
		expectErr bool
	}{
		{"version", "3.0", false},
		{"ai_cli.command", "new_cli", false},
		{"execution.max_retry_count", 10, false},
		{"nonexistent", "value", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			err := loader.Set(tt.key, tt.value)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Set(%q) expected error but got none", tt.key)
				}
			} else {
				if err != nil {
					t.Errorf("Set(%q) unexpected error: %v", tt.key, err)
				}

				val, err := loader.Get(tt.key)
				if err != nil {
					t.Errorf("Get(%q) after Set error: %v", tt.key, err)
				}
				if val != tt.value {
					t.Errorf("Get(%q) after Set = %v, want %v", tt.key, val, tt.value)
				}
			}
		})
	}
}

// TestLoaderSaveAndSaveTo tests saving configuration.
func TestLoaderSaveAndSaveTo(t *testing.T) {
	tempDir := t.TempDir()

	loader := NewLoader()
	loader.config = &Config{
		Version: "2.0",
		AICli: AICliConfig{
			Command: "saved_cli",
			EnvVar:  "SAVED_VAR",
		},
	}

	savePath := filepath.Join(tempDir, "saved_config.json")
	if err := loader.SaveTo(savePath); err != nil {
		t.Fatalf("SaveTo() error: %v", err)
	}

	if _, err := os.Stat(savePath); os.IsNotExist(err) {
		t.Fatal("SaveTo() did not create file")
	}

	newLoader := NewLoader()
	if err := newLoader.Load(savePath); err != nil {
		t.Fatalf("Load() saved config error: %v", err)
	}

	if newLoader.Config().AICli.Command != "saved_cli" {
		t.Errorf("Loaded saved config command = %s, want 'saved_cli'", newLoader.Config().AICli.Command)
	}
}

// TestLoaderValidate tests the Validate method.
func TestLoaderValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid version",
			config: &Config{
				Version: "1.0",
				AICli: AICliConfig{
					Command: "cli",
					EnvVar:  "VAR",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader()
			loader.config = tt.config

			err := loader.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

// TestLoaderEnvironmentVariables tests environment variable overrides.
func TestLoaderEnvironmentVariables(t *testing.T) {
	os.Setenv(EnvMortyLogLevel, "error")
	defer os.Unsetenv(EnvMortyLogLevel)

	loader := NewLoader()
	loader.LoadWithMerge("")

	if loader.Config().Logging.Level != "error" {
		t.Errorf("Expected log level 'error' from env var, got %s", loader.Config().Logging.Level)
	}
}

// TestLoaderPathHelpers tests the path helper methods.
func TestLoaderPathHelpers(t *testing.T) {
	loader := NewLoader()

	if loader.GetWorkDir() != ".morty" {
		t.Errorf("GetWorkDir() = %s, want '.morty'", loader.GetWorkDir())
	}

	if loader.GetLogDir() != ".morty/doing/logs" {
		t.Errorf("GetLogDir() = %s, want '.morty/doing/logs'", loader.GetLogDir())
	}

	if loader.GetPlanDir() != DefaultPlanDir {
		t.Errorf("GetPlanDir() = %s, want %s", loader.GetPlanDir(), DefaultPlanDir)
	}

	if loader.GetStatusFile() != DefaultStateFile {
		t.Errorf("GetStatusFile() = %s, want %s", loader.GetStatusFile(), DefaultStateFile)
	}
}

// TestLoaderExpandPath tests the path expansion.
func TestLoaderExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"~", home},
		{"~/config.json", filepath.Join(home, "config.json")},
	}

	for _, tt := range tests {
		result, err := expandPath(tt.input)
		if err != nil {
			t.Errorf("expandPath(%q) error: %v", tt.input, err)
			continue
		}

		if tt.input == "" && result != "" {
			t.Errorf("expandPath(%q) = %q, want empty", tt.input, result)
			continue
		}

		if tt.input != "" && result == "" {
			t.Errorf("expandPath(%q) returned empty", tt.input)
		}
	}
}

// TestLoaderJSONParsingErrors tests invalid JSON handling.
func TestLoaderJSONParsingErrors(t *testing.T) {
	tempDir := t.TempDir()

	invalidPath := filepath.Join(tempDir, "invalid.json")
	os.WriteFile(invalidPath, []byte(`{not valid json}`), 0644)

	loader := NewLoader()
	err := loader.Load(invalidPath)

	if err == nil {
		t.Error("Load() with invalid JSON expected error but got none")
	}

	if !strings.Contains(err.Error(), "JSON") {
		t.Errorf("Load() error should mention JSON, got: %v", err)
	}
}

// TestLoaderLoadWithMergeNoUserConfig tests LoadWithMerge without user config.
func TestLoaderLoadWithMergeNoUserConfig(t *testing.T) {
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	loader := NewLoader()
	err := loader.LoadWithMerge("")

	if err != nil {
		t.Errorf("LoadWithMerge() with no user config unexpected error: %v", err)
	}

	if loader.Config().Version != DefaultVersion {
		t.Errorf("Expected default version, got %s", loader.Config().Version)
	}
}
