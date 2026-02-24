package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDefaultConfig tests that DefaultConfig returns a valid config with all defaults set.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Test version
	if cfg.Version != DefaultVersion {
		t.Errorf("Version = %q, want %q", cfg.Version, DefaultVersion)
	}

	// Test AICli defaults
	if cfg.AICli.Command != DefaultAICliCommand {
		t.Errorf("AICli.Command = %q, want %q", cfg.AICli.Command, DefaultAICliCommand)
	}
	if cfg.AICli.EnvVar != DefaultAICliEnvVar {
		t.Errorf("AICli.EnvVar = %q, want %q", cfg.AICli.EnvVar, DefaultAICliEnvVar)
	}
	if cfg.AICli.DefaultTimeout != DefaultAICliDefaultTimeout {
		t.Errorf("AICli.DefaultTimeout = %q, want %q", cfg.AICli.DefaultTimeout, DefaultAICliDefaultTimeout)
	}
	if cfg.AICli.MaxTimeout != DefaultAICliMaxTimeout {
		t.Errorf("AICli.MaxTimeout = %q, want %q", cfg.AICli.MaxTimeout, DefaultAICliMaxTimeout)
	}
	if cfg.AICli.EnableSkipPermissions != DefaultAICliEnableSkipPermissions {
		t.Errorf("AICli.EnableSkipPermissions = %v, want %v", cfg.AICli.EnableSkipPermissions, DefaultAICliEnableSkipPermissions)
	}
	if cfg.AICli.OutputFormat != DefaultAICliOutputFormat {
		t.Errorf("AICli.OutputFormat = %q, want %q", cfg.AICli.OutputFormat, DefaultAICliOutputFormat)
	}

	// Test Execution defaults
	if cfg.Execution.MaxRetryCount != DefaultExecutionMaxRetryCount {
		t.Errorf("Execution.MaxRetryCount = %d, want %d", cfg.Execution.MaxRetryCount, DefaultExecutionMaxRetryCount)
	}
	if cfg.Execution.AutoGitCommit != DefaultExecutionAutoGitCommit {
		t.Errorf("Execution.AutoGitCommit = %v, want %v", cfg.Execution.AutoGitCommit, DefaultExecutionAutoGitCommit)
	}
	if cfg.Execution.ContinueOnError != DefaultExecutionContinueOnError {
		t.Errorf("Execution.ContinueOnError = %v, want %v", cfg.Execution.ContinueOnError, DefaultExecutionContinueOnError)
	}
	if cfg.Execution.ParallelJobs != DefaultExecutionParallelJobs {
		t.Errorf("Execution.ParallelJobs = %d, want %d", cfg.Execution.ParallelJobs, DefaultExecutionParallelJobs)
	}

	// Test Logging defaults
	if cfg.Logging.Level != DefaultLoggingLevel {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, DefaultLoggingLevel)
	}
	if cfg.Logging.Format != DefaultLoggingFormat {
		t.Errorf("Logging.Format = %q, want %q", cfg.Logging.Format, DefaultLoggingFormat)
	}
	if cfg.Logging.Output != DefaultLoggingOutput {
		t.Errorf("Logging.Output = %q, want %q", cfg.Logging.Output, DefaultLoggingOutput)
	}
	if cfg.Logging.File.Enabled != DefaultLoggingFileEnabled {
		t.Errorf("Logging.File.Enabled = %v, want %v", cfg.Logging.File.Enabled, DefaultLoggingFileEnabled)
	}
	if cfg.Logging.File.Path != DefaultLoggingFilePath {
		t.Errorf("Logging.File.Path = %q, want %q", cfg.Logging.File.Path, DefaultLoggingFilePath)
	}
	if cfg.Logging.File.MaxSize != DefaultLoggingFileMaxSize {
		t.Errorf("Logging.File.MaxSize = %q, want %q", cfg.Logging.File.MaxSize, DefaultLoggingFileMaxSize)
	}
	if cfg.Logging.File.MaxBackups != DefaultLoggingFileMaxBackups {
		t.Errorf("Logging.File.MaxBackups = %d, want %d", cfg.Logging.File.MaxBackups, DefaultLoggingFileMaxBackups)
	}
	if cfg.Logging.File.MaxAge != DefaultLoggingFileMaxAge {
		t.Errorf("Logging.File.MaxAge = %d, want %d", cfg.Logging.File.MaxAge, DefaultLoggingFileMaxAge)
	}

	// Test State defaults
	if cfg.State.File != DefaultStateFile {
		t.Errorf("State.File = %q, want %q", cfg.State.File, DefaultStateFile)
	}
	if cfg.State.AutoSave != DefaultStateAutoSave {
		t.Errorf("State.AutoSave = %v, want %v", cfg.State.AutoSave, DefaultStateAutoSave)
	}
	if cfg.State.SaveInterval != DefaultStateSaveInterval {
		t.Errorf("State.SaveInterval = %q, want %q", cfg.State.SaveInterval, DefaultStateSaveInterval)
	}

	// Test Git defaults
	if cfg.Git.CommitPrefix != DefaultGitCommitPrefix {
		t.Errorf("Git.CommitPrefix = %q, want %q", cfg.Git.CommitPrefix, DefaultGitCommitPrefix)
	}
	if cfg.Git.AutoCommit != DefaultGitAutoCommit {
		t.Errorf("Git.AutoCommit = %v, want %v", cfg.Git.AutoCommit, DefaultGitAutoCommit)
	}
	if cfg.Git.RequireCleanWorktree != DefaultGitRequireCleanWorktree {
		t.Errorf("Git.RequireCleanWorktree = %v, want %v", cfg.Git.RequireCleanWorktree, DefaultGitRequireCleanWorktree)
	}

	// Test Plan defaults
	if cfg.Plan.Dir != DefaultPlanDir {
		t.Errorf("Plan.Dir = %q, want %q", cfg.Plan.Dir, DefaultPlanDir)
	}
	if cfg.Plan.FileExtension != DefaultPlanFileExtension {
		t.Errorf("Plan.FileExtension = %q, want %q", cfg.Plan.FileExtension, DefaultPlanFileExtension)
	}
	if cfg.Plan.AutoValidate != DefaultPlanAutoValidate {
		t.Errorf("Plan.AutoValidate = %v, want %v", cfg.Plan.AutoValidate, DefaultPlanAutoValidate)
	}

	// Test Prompts defaults
	if cfg.Prompts.Dir != DefaultPromptsDir {
		t.Errorf("Prompts.Dir = %q, want %q", cfg.Prompts.Dir, DefaultPromptsDir)
	}
	if cfg.Prompts.Research != DefaultPromptsResearch {
		t.Errorf("Prompts.Research = %q, want %q", cfg.Prompts.Research, DefaultPromptsResearch)
	}
	if cfg.Prompts.Plan != DefaultPromptsPlan {
		t.Errorf("Prompts.Plan = %q, want %q", cfg.Prompts.Plan, DefaultPromptsPlan)
	}
	if cfg.Prompts.Doing != DefaultPromptsDoing {
		t.Errorf("Prompts.Doing = %q, want %q", cfg.Prompts.Doing, DefaultPromptsDoing)
	}
}

// TestConfigSerialization tests JSON serialization and deserialization.
func TestConfigSerialization(t *testing.T) {
	cfg := DefaultConfig()

	// Serialize to JSON
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Verify JSON is valid
	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("Generated JSON is invalid: %v", err)
	}

	// Deserialize back
	var cfg2 Config
	if err := json.Unmarshal(data, &cfg2); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify values match
	if cfg2.Version != cfg.Version {
		t.Errorf("Version mismatch: got %q, want %q", cfg2.Version, cfg.Version)
	}
	if cfg2.AICli.Command != cfg.AICli.Command {
		t.Errorf("AICli.Command mismatch: got %q, want %q", cfg2.AICli.Command, cfg.AICli.Command)
	}
}

// TestConfigStructTags tests that all structs have proper JSON tags.
func TestConfigStructTags(t *testing.T) {
	cfg := DefaultConfig()
	data, _ := json.Marshal(cfg)

	// Check that JSON keys use snake_case
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	// Check that nested structs are properly serialized
	if _, ok := raw["ai_cli"]; !ok {
		t.Error("Missing 'ai_cli' field in JSON")
	}
	if _, ok := raw["execution"]; !ok {
		t.Error("Missing 'execution' field in JSON")
	}
	if _, ok := raw["logging"]; !ok {
		t.Error("Missing 'logging' field in JSON")
	}
	if _, ok := raw["state"]; !ok {
		t.Error("Missing 'state' field in JSON")
	}
	if _, ok := raw["git"]; !ok {
		t.Error("Missing 'git' field in JSON")
	}
	if _, ok := raw["plan"]; !ok {
		t.Error("Missing 'plan' field in JSON")
	}
	if _, ok := raw["prompts"]; !ok {
		t.Error("Missing 'prompts' field in JSON")
	}
}

// TestAICliDuration tests the Duration helper method.
func TestAICliDuration(t *testing.T) {
	tests := []struct {
		name        string
		timeout     string
		defaultVal  time.Duration
		want        time.Duration
	}{
		{
			name:        "valid 10m",
			timeout:     "10m",
			defaultVal:  5 * time.Minute,
			want:        10 * time.Minute,
		},
		{
			name:        "valid 1h30s",
			timeout:     "1h30s",
			defaultVal:  5 * time.Minute,
			want:        time.Hour + 30*time.Second,
		},
		{
			name:        "invalid uses default",
			timeout:     "invalid",
			defaultVal:  5 * time.Minute,
			want:        5 * time.Minute,
		},
		{
			name:        "empty uses default",
			timeout:     "",
			defaultVal:  5 * time.Minute,
			want:        5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := AICliConfig{DefaultTimeout: tt.timeout}
			got := cfg.Duration(tt.defaultVal)
			if got != tt.want {
				t.Errorf("Duration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAICliMaxTimeoutDuration tests the MaxTimeoutDuration helper method.
func TestAICliMaxTimeoutDuration(t *testing.T) {
	tests := []struct {
		name        string
		maxTimeout  string
		defaultVal  time.Duration
		want        time.Duration
	}{
		{
			name:        "valid 30m",
			maxTimeout:  "30m",
			defaultVal:  1 * time.Hour,
			want:        30 * time.Minute,
		},
		{
			name:        "invalid uses default",
			maxTimeout:  "bad",
			defaultVal:  1 * time.Hour,
			want:        1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := AICliConfig{MaxTimeout: tt.maxTimeout}
			got := cfg.MaxTimeoutDuration(tt.defaultVal)
			if got != tt.want {
				t.Errorf("MaxTimeoutDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStateSaveIntervalDuration tests the SaveIntervalDuration helper method.
func TestStateSaveIntervalDuration(t *testing.T) {
	tests := []struct {
		name       string
		interval   string
		defaultVal time.Duration
		want       time.Duration
	}{
		{
			name:       "valid 30s",
			interval:   "30s",
			defaultVal: 1 * time.Minute,
			want:       30 * time.Second,
		},
		{
			name:       "invalid uses default",
			interval:   "wrong",
			defaultVal: 1 * time.Minute,
			want:       1 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := StateConfig{SaveInterval: tt.interval}
			got := cfg.SaveIntervalDuration(tt.defaultVal)
			if got != tt.want {
				t.Errorf("SaveIntervalDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDefaultConfigJSONFile tests that the default config can be loaded from settings.json.
func TestDefaultConfigJSONFile(t *testing.T) {
	// Read the actual settings.json file
	settingsPath := filepath.Join("..", "..", "configs", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read settings.json: %v", err)
	}

	// Parse the JSON
	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		t.Fatalf("Failed to parse settings.json: %v", err)
	}

	// Compare with DefaultConfig
	defaultCfg := DefaultConfig()

	if fileCfg.Version != defaultCfg.Version {
		t.Errorf("settings.json Version = %q, want %q", fileCfg.Version, defaultCfg.Version)
	}
	if fileCfg.AICli.Command != defaultCfg.AICli.Command {
		t.Errorf("settings.json AICli.Command = %q, want %q", fileCfg.AICli.Command, defaultCfg.AICli.Command)
	}
	if fileCfg.Logging.Level != defaultCfg.Logging.Level {
		t.Errorf("settings.json Logging.Level = %q, want %q", fileCfg.Logging.Level, defaultCfg.Logging.Level)
	}
}

// TestConstants tests that all default constants are properly defined.
func TestConstants(t *testing.T) {
	// Test AI CLI constants
	if DefaultAICliCommand == "" {
		t.Error("DefaultAICliCommand should not be empty")
	}
	if DefaultAICliEnvVar == "" {
		t.Error("DefaultAICliEnvVar should not be empty")
	}

	// Test execution constants
	if DefaultExecutionMaxRetryCount < 0 {
		t.Error("DefaultExecutionMaxRetryCount should be >= 0")
	}
	if DefaultExecutionParallelJobs < 1 {
		t.Error("DefaultExecutionParallelJobs should be >= 1")
	}

	// Test logging constants
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[DefaultLoggingLevel] {
		t.Errorf("DefaultLoggingLevel = %q is not a valid level", DefaultLoggingLevel)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[DefaultLoggingFormat] {
		t.Errorf("DefaultLoggingFormat = %q is not a valid format", DefaultLoggingFormat)
	}

	// Test path constants
	if DefaultWorkDir == "" {
		t.Error("DefaultWorkDir should not be empty")
	}
	if DefaultPlanDir == "" {
		t.Error("DefaultPlanDir should not be empty")
	}
}

// TestConfigFieldsSelfDocumenting tests that config fields are self-documenting.
func TestConfigFieldsSelfDocumenting(t *testing.T) {
	cfg := DefaultConfig()

	// All string fields should have meaningful values (not just placeholders)
	if cfg.AICli.Command == "" {
		t.Error("AICli.Command should have a meaningful default")
	}
	if cfg.Logging.Level == "" {
		t.Error("Logging.Level should have a meaningful default")
	}
	if cfg.Git.CommitPrefix == "" {
		t.Error("Git.CommitPrefix should have a meaningful default")
	}
	if cfg.Plan.Dir == "" {
		t.Error("Plan.Dir should have a meaningful default")
	}
	if cfg.State.File == "" {
		t.Error("State.File should have a meaningful default")
	}
}

// BenchmarkConfigSerialization benchmarks JSON serialization performance.
func BenchmarkConfigSerialization(b *testing.B) {
	cfg := DefaultConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(cfg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDefaultConfig benchmarks DefaultConfig performance.
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}
