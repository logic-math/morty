package config

import (
	"time"
)

// Manager defines the interface for configuration management.
// It provides methods for loading, reading, setting, and saving configuration
// with support for dot notation access to nested values.
//
// The configuration follows a 5-level hierarchy:
// 1. Built-in defaults (lowest priority)
// 2. User global config (~/.morty/config.json)
// 3. Project config (.morty/settings.json)
// 4. Environment variables (MORTY_* prefix)
// 5. Command-line arguments (highest priority)
type Manager interface {
	// Load loads configuration from the specified file path.
	// The file should be in JSON format. If the file doesn't exist,
	// an error is returned.
	//
	// Example:
	//   err := mgr.Load(".morty/settings.json")
	Load(path string) error

	// LoadWithMerge loads and merges configuration from multiple sources.
	// It loads in this order: defaults → user config → project config → environment.
	// Later sources override earlier ones.
	//
	// The userConfigPath is typically ~/.morty/config.json
	// The project config is loaded from .morty/settings.json if it exists.
	//
	// Example:
	//   err := mgr.LoadWithMerge("~/.morty/config.json")
	LoadWithMerge(userConfigPath string) error

	// Get retrieves a configuration value using dot notation.
	// It returns the value as interface{} or an error if the key doesn't exist.
	// An optional default value can be provided as the second argument.
	//
	// Example:
	//   val, err := mgr.Get("ai_cli.command")
	//   val, err := mgr.Get("execution.max_retry_count", 3)
	Get(key string, defaultValue ...interface{}) (interface{}, error)

	// GetString retrieves a string configuration value using dot notation.
	// If the key doesn't exist or the value is not a string, the default is returned.
	//
	// Example:
	//   cmd := mgr.GetString("ai_cli.command", "ai_cli")
	GetString(key string, defaultValue ...string) string

	// GetInt retrieves an integer configuration value using dot notation.
	// If the key doesn't exist or the value is not an integer, the default is returned.
	//
	// Example:
	//   retries := mgr.GetInt("execution.max_retry_count", 3)
	GetInt(key string, defaultValue ...int) int

	// GetBool retrieves a boolean configuration value using dot notation.
	// If the key doesn't exist or the value is not a boolean, the default is returned.
	//
	// Example:
	//   autoCommit := mgr.GetBool("execution.auto_git_commit", true)
	GetBool(key string, defaultValue ...bool) bool

	// GetDuration retrieves a duration configuration value using dot notation.
	// The value should be a string in Go duration format (e.g., "10m", "1h30s").
	// If the key doesn't exist or parsing fails, the default is returned.
	//
	// Example:
	//   timeout := mgr.GetDuration("ai_cli.default_timeout", 10*time.Minute)
	GetDuration(key string, defaultValue ...time.Duration) time.Duration

	// Set sets a configuration value using dot notation.
	// It creates nested structures as needed.
	//
	// Example:
	//   err := mgr.Set("execution.max_retry_count", 5)
	//   err := mgr.Set("ai_cli.command", "claude")
	Set(key string, value interface{}) error

	// Save saves the current configuration to the default location.
	// The default location is the last loaded config file or ~/.morty/config.json.
	//
	// Example:
	//   err := mgr.Save()
	Save() error

	// SaveTo saves the current configuration to the specified path.
	// The file is written in JSON format with indentation.
	//
	// Example:
	//   err := mgr.SaveTo(".morty/settings.json")
	SaveTo(path string) error

	// Path Helpers

	// GetWorkDir returns the Morty working directory path (typically ".morty").
	GetWorkDir() string

	// GetLogDir returns the log directory path (typically ".morty/doing/logs").
	GetLogDir() string

	// GetResearchDir returns the research directory path (typically ".morty/research").
	GetResearchDir() string

	// GetPlanDir returns the plan directory path (typically ".morty/plan").
	GetPlanDir() string

	// GetStatusFile returns the status file path (typically ".morty/status.json").
	GetStatusFile() string

	// GetConfigFile returns the current configuration file path.
	GetConfigFile() string
}

// Ensure managerImpl implements Manager interface.
// This will be uncommented when managerImpl is implemented.
// var _ Manager = (*managerImpl)(nil)
