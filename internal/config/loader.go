// Package config provides configuration management for Morty.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Loader implements configuration loading and merging with support for
// hierarchical configuration sources.
type Loader struct {
	config     *Config
	configFile string
}

// NewLoader creates a new configuration loader with default values.
func NewLoader() *Loader {
	return &Loader{
		config: DefaultConfig(),
	}
}

// Config returns the current configuration.
func (l *Loader) Config() *Config {
	return l.config
}

// Load loads configuration from the specified file path.
// The file should be in JSON format. If the file doesn't exist,
// returns an error.
func (l *Loader) Load(path string) error {
	if path == "" {
		return fmt.Errorf("config path cannot be empty")
	}

	// Expand ~ to home directory
	expandedPath, err := expandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path %s: %w", path, err)
	}

	// Check if file exists
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", expandedPath)
	}

	// Read file content
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", expandedPath, err)
	}

	// Parse JSON
	var fileConfig Config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("failed to parse JSON config from %s: %w", expandedPath, err)
	}

	// Merge with defaults (file config overrides defaults)
	l.config = mergeConfigs(DefaultConfig(), &fileConfig)
	l.configFile = expandedPath

	// Validate the loaded configuration
	if err := l.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

// LoadWithMerge loads and merges configuration from multiple sources.
// Loading order: defaults → user config → project config → environment variables.
// Later sources override earlier ones.
func (l *Loader) LoadWithMerge(userConfigPath string) error {
	// Start with defaults
	l.config = DefaultConfig()

	// Load user config if exists (optional)
	if userConfigPath != "" {
		expandedPath, _ := expandPath(userConfigPath)
		if _, err := os.Stat(expandedPath); err == nil {
			userConfig, err := l.loadConfigFromFile(expandedPath)
			if err == nil {
				l.config = mergeConfigs(l.config, userConfig)
			}
		}
	}

	// Load project config if exists (optional)
	projectConfigPath := filepath.Join(l.GetWorkDir(), "settings.json")
	if _, err := os.Stat(projectConfigPath); err == nil {
		projectConfig, err := l.loadConfigFromFile(projectConfigPath)
		if err == nil {
			l.config = mergeConfigs(l.config, projectConfig)
		}
	}

	// Apply environment variables (highest priority)
	l.applyEnvironmentVariables()

	// Validate the final configuration
	if err := l.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

// LoadWithDefaults loads configuration from file if it exists,
// otherwise uses default configuration.
func (l *Loader) LoadWithDefaults(path string) error {
	if path == "" {
		// Use defaults
		l.config = DefaultConfig()
		return nil
	}

	expandedPath, err := expandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path %s: %w", path, err)
	}

	// Check if file exists
	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		// Use defaults when file doesn't exist
		l.config = DefaultConfig()
		return nil
	}

	// File exists, load it
	return l.Load(path)
}

// loadConfigFromFile loads a Config from a file without validation.
func (l *Loader) loadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate validates the current configuration.
func (l *Loader) Validate() error {
	validator := NewConfigValidator()
	return validator.Validate(l.config)
}

// Get retrieves a configuration value using dot notation.
func (l *Loader) Get(key string, defaultValue ...interface{}) (interface{}, error) {
	if l.config == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	value, err := getFieldByPath(l.config, key)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0], nil
		}
		return nil, err
	}

	return value, nil
}

// GetString retrieves a string configuration value using dot notation.
func (l *Loader) GetString(key string, defaultValue ...string) string {
	val, err := l.Get(key)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}

	if str, ok := val.(string); ok {
		return str
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// GetInt retrieves an integer configuration value using dot notation.
func (l *Loader) GetInt(key string, defaultValue ...int) int {
	val, err := l.Get(key)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// GetBool retrieves a boolean configuration value using dot notation.
func (l *Loader) GetBool(key string, defaultValue ...bool) bool {
	val, err := l.Get(key)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case string:
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

// GetDuration retrieves a duration configuration value using dot notation.
func (l *Loader) GetDuration(key string, defaultValue ...time.Duration) time.Duration {
	val, err := l.Get(key)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	switch v := val.(type) {
	case time.Duration:
		return v
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

// Set sets a configuration value using dot notation.
func (l *Loader) Set(key string, value interface{}) error {
	if l.config == nil {
		return fmt.Errorf("config not loaded")
	}

	return setFieldByPath(l.config, key, value)
}

// Save saves the current configuration to the default location.
func (l *Loader) Save() error {
	if l.configFile == "" {
		return fmt.Errorf("no config file path set")
	}
	return l.SaveTo(l.configFile)
}

// SaveTo saves the current configuration to the specified path.
func (l *Loader) SaveTo(path string) error {
	if l.config == nil {
		return fmt.Errorf("config not loaded")
	}

	expandedPath, err := expandPath(path)
	if err != nil {
		return fmt.Errorf("failed to expand path %s: %w", path, err)
	}

	// Ensure directory exists
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(l.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(expandedPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", expandedPath, err)
	}

	return nil
}

// GetWorkDir returns the Morty working directory path.
func (l *Loader) GetWorkDir() string {
	return ".morty"
}

// GetLogDir returns the log directory path.
func (l *Loader) GetLogDir() string {
	return filepath.Join(l.GetWorkDir(), "doing", "logs")
}

// GetResearchDir returns the research directory path.
func (l *Loader) GetResearchDir() string {
	return filepath.Join(l.GetWorkDir(), "research")
}

// GetPlanDir returns the plan directory path.
func (l *Loader) GetPlanDir() string {
	if l.config != nil && l.config.Plan.Dir != "" {
		return l.config.Plan.Dir
	}
	return DefaultPlanDir
}

// GetStatusFile returns the status file path.
func (l *Loader) GetStatusFile() string {
	if l.config != nil && l.config.State.File != "" {
		return l.config.State.File
	}
	return DefaultStateFile
}

// GetConfigFile returns the current configuration file path.
func (l *Loader) GetConfigFile() string {
	return l.configFile
}

// applyEnvironmentVariables applies environment variable overrides.
func (l *Loader) applyEnvironmentVariables() {
	if l.config == nil {
		return
	}

	// MORTY_LOG_LEVEL
	if level := os.Getenv(EnvMortyLogLevel); level != "" {
		l.config.Logging.Level = level
	}

	// MORTY_DEBUG
	if debug := os.Getenv(EnvMortyDebug); debug != "" {
		if b, err := strconv.ParseBool(debug); err == nil {
			// Map debug to log level
			if b {
				l.config.Logging.Level = "debug"
			}
		}
	}

	// MORTY_HOME - affects paths
	if home := os.Getenv(EnvMortyHome); home != "" {
		// Update paths that depend on MORTY_HOME
		l.config.State.File = filepath.Join(home, "status.json")
		l.config.Logging.File.Path = filepath.Join(home, "doing", "logs", "morty.log")
		l.config.Plan.Dir = filepath.Join(home, "plan")
	}

	// MORTY_CONFIG - specific config file path
	// This is handled during Load(), not here
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Handle ~ at the start
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path, nil // Return original if can't make absolute
	}

	return absPath, nil
}

// mergeConfigs merges two configurations, with src overriding dst.
func mergeConfigs(dst, src *Config) *Config {
	if src == nil {
		if dst == nil {
			return DefaultConfig()
		}
		return dst
	}

	if dst == nil {
		dst = DefaultConfig()
	}

	result := *dst // Copy dst

	// Merge Version
	if src.Version != "" {
		result.Version = src.Version
	}

	// Merge AICli
	if src.AICli.Command != "" {
		result.AICli.Command = src.AICli.Command
	}
	if src.AICli.EnvVar != "" {
		result.AICli.EnvVar = src.AICli.EnvVar
	}
	if src.AICli.DefaultTimeout != "" {
		result.AICli.DefaultTimeout = src.AICli.DefaultTimeout
	}
	if src.AICli.MaxTimeout != "" {
		result.AICli.MaxTimeout = src.AICli.MaxTimeout
	}
	if src.AICli.OutputFormat != "" {
		result.AICli.OutputFormat = src.AICli.OutputFormat
	}
	if len(src.AICli.DefaultArgs) > 0 {
		result.AICli.DefaultArgs = src.AICli.DefaultArgs
	}
	result.AICli.EnableSkipPermissions = src.AICli.EnableSkipPermissions

	// Merge Execution
	if src.Execution.MaxRetryCount != 0 {
		result.Execution.MaxRetryCount = src.Execution.MaxRetryCount
	}
	result.Execution.AutoGitCommit = src.Execution.AutoGitCommit
	result.Execution.ContinueOnError = src.Execution.ContinueOnError
	if src.Execution.ParallelJobs != 0 {
		result.Execution.ParallelJobs = src.Execution.ParallelJobs
	}

	// Merge Logging
	if src.Logging.Level != "" {
		result.Logging.Level = src.Logging.Level
	}
	if src.Logging.Format != "" {
		result.Logging.Format = src.Logging.Format
	}
	if src.Logging.Output != "" {
		result.Logging.Output = src.Logging.Output
	}
	result.Logging.File.Enabled = src.Logging.File.Enabled
	if src.Logging.File.Path != "" {
		result.Logging.File.Path = src.Logging.File.Path
	}
	if src.Logging.File.MaxSize != "" {
		result.Logging.File.MaxSize = src.Logging.File.MaxSize
	}
	if src.Logging.File.MaxBackups != 0 {
		result.Logging.File.MaxBackups = src.Logging.File.MaxBackups
	}
	if src.Logging.File.MaxAge != 0 {
		result.Logging.File.MaxAge = src.Logging.File.MaxAge
	}

	// Merge State
	if src.State.File != "" {
		result.State.File = src.State.File
	}
	result.State.AutoSave = src.State.AutoSave
	if src.State.SaveInterval != "" {
		result.State.SaveInterval = src.State.SaveInterval
	}

	// Merge Git
	if src.Git.CommitPrefix != "" {
		result.Git.CommitPrefix = src.Git.CommitPrefix
	}
	result.Git.AutoCommit = src.Git.AutoCommit
	result.Git.RequireCleanWorktree = src.Git.RequireCleanWorktree

	// Merge Plan
	if src.Plan.Dir != "" {
		result.Plan.Dir = src.Plan.Dir
	}
	if src.Plan.FileExtension != "" {
		result.Plan.FileExtension = src.Plan.FileExtension
	}
	result.Plan.AutoValidate = src.Plan.AutoValidate

	// Merge Prompts
	if src.Prompts.Dir != "" {
		result.Prompts.Dir = src.Prompts.Dir
	}
	if src.Prompts.Research != "" {
		result.Prompts.Research = src.Prompts.Research
	}
	if src.Prompts.Plan != "" {
		result.Prompts.Plan = src.Prompts.Plan
	}
	if src.Prompts.Doing != "" {
		result.Prompts.Doing = src.Prompts.Doing
	}

	return &result
}

// getFieldByPath gets a field value from config using dot notation.
func getFieldByPath(cfg *Config, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	v := reflect.ValueOf(cfg).Elem()

	for _, part := range parts {
		if v.Kind() != reflect.Struct {
			return nil, fmt.Errorf("path %s references non-struct field", path)
		}

		field := findFieldByJSONTag(v, part)
		if !field.IsValid() {
			return nil, fmt.Errorf("field %s not found", part)
		}

		v = field
	}

	return v.Interface(), nil
}

// setFieldByPath sets a field value in config using dot notation.
func setFieldByPath(cfg *Config, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}

	v := reflect.ValueOf(cfg).Elem()

	// Navigate to the parent of the target field
	for i := 0; i < len(parts)-1; i++ {
		if v.Kind() != reflect.Struct {
			return fmt.Errorf("path %s references non-struct field", path)
		}

		field := findFieldByJSONTag(v, parts[i])
		if !field.IsValid() {
			return fmt.Errorf("field %s not found", parts[i])
		}

		v = field
	}

	// Set the final field
	lastPart := parts[len(parts)-1]
	field := findFieldByJSONTag(v, lastPart)
	if !field.IsValid() {
		return fmt.Errorf("field %s not found", lastPart)
	}

	if !field.CanSet() {
		return fmt.Errorf("field %s cannot be set", lastPart)
	}

	// Convert value to the correct type
	val := reflect.ValueOf(value)
	if val.Type().ConvertibleTo(field.Type()) {
		field.Set(val.Convert(field.Type()))
	} else {
		return fmt.Errorf("cannot convert %T to %s", value, field.Type())
	}

	return nil
}

// findFieldByJSONTag finds a struct field by its json tag name.
func findFieldByJSONTag(v reflect.Value, name string) reflect.Value {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")

		// Handle json tag options like `json:"name,omitempty"`
		if idx := strings.Index(jsonTag, ","); idx != -1 {
			jsonTag = jsonTag[:idx]
		}

		if jsonTag == name {
			return v.Field(i)
		}
	}

	return reflect.Value{}
}

// Ensure Loader implements Manager interface
var _ Manager = (*Loader)(nil)
