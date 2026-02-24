// Package config provides configuration management for Morty.
// It supports hierarchical configuration loading with 5 levels of priority:
// 1. Built-in defaults (compiled into binary)
// 2. User global config (~/.morty/config.json)
// 3. Project config (.morty/settings.json)
// 4. Environment variables (MORTY_* prefix)
// 5. Command-line arguments (highest priority)
package config

import "time"

// Config represents the complete configuration structure for Morty.
// It contains all configuration sections organized by functionality.
type Config struct {
	// Version is the configuration format version.
	Version string `json:"version"`

	// AICli contains AI CLI configuration settings.
	AICli AICliConfig `json:"ai_cli"`

	// Execution contains execution-related configuration.
	Execution ExecutionConfig `json:"execution"`

	// Logging contains logging configuration settings.
	Logging LoggingConfig `json:"logging"`

	// State contains state management configuration.
	State StateConfig `json:"state"`

	// Git contains Git operation configuration.
	Git GitConfig `json:"git"`

	// Plan contains plan management configuration.
	Plan PlanConfig `json:"plan"`

	// Prompts contains prompt file path configuration.
	Prompts PromptsConfig `json:"prompts"`
}

// AICliConfig contains AI CLI configuration settings.
// This defines how Morty interacts with the AI CLI tool.
type AICliConfig struct {
	// Command is the AI CLI command name (e.g., "ai_cli", "claude").
	Command string `json:"command"`

	// EnvVar is the environment variable name for CLI path override.
	EnvVar string `json:"env_var"`

	// DefaultTimeout is the default timeout for CLI operations (e.g., "10m").
	DefaultTimeout string `json:"default_timeout"`

	// MaxTimeout is the maximum allowed timeout for CLI operations.
	MaxTimeout string `json:"max_timeout"`

	// EnableSkipPermissions enables the --dangerously-skip-permissions flag.
	EnableSkipPermissions bool `json:"enable_skip_permissions"`

	// DefaultArgs contains default arguments passed to the CLI.
	DefaultArgs []string `json:"default_args"`

	// OutputFormat specifies the output format ("json" or "text").
	OutputFormat string `json:"output_format"`
}

// ExecutionConfig contains execution-related configuration.
// This controls how jobs and tasks are executed.
type ExecutionConfig struct {
	// MaxRetryCount is the maximum number of retries for failed operations.
	MaxRetryCount int `json:"max_retry_count"`

	// AutoGitCommit enables automatic Git commits after job completion.
	AutoGitCommit bool `json:"auto_git_commit"`

	// ContinueOnError allows continuing execution when errors occur.
	ContinueOnError bool `json:"continue_on_error"`

	// ParallelJobs is the number of parallel jobs to run (reserved for future).
	ParallelJobs int `json:"parallel_jobs"`
}

// LoggingConfig contains logging configuration settings.
// This controls log output, format, and file rotation.
type LoggingConfig struct {
	// Level is the log level ("debug", "info", "warn", "error").
	Level string `json:"level"`

	// Format is the log format ("json" or "text").
	Format string `json:"format"`

	// Output is the output destination ("stdout", "file", or "both").
	Output string `json:"output"`

	// File contains file logging configuration.
	File FileConfig `json:"file"`
}

// FileConfig contains file logging configuration.
// This defines log file rotation and retention policies.
type FileConfig struct {
	// Enabled enables file logging.
	Enabled bool `json:"enabled"`

	// Path is the log file path.
	Path string `json:"path"`

	// MaxSize is the maximum size of a single log file (e.g., "10MB").
	MaxSize string `json:"max_size"`

	// MaxBackups is the number of backup files to retain.
	MaxBackups int `json:"max_backups"`

	// MaxAge is the number of days to retain log files.
	MaxAge int `json:"max_age"`
}

// StateConfig contains state management configuration.
// This controls how execution state is tracked and persisted.
type StateConfig struct {
	// File is the path to the status file.
	File string `json:"file"`

	// AutoSave enables automatic state saving.
	AutoSave bool `json:"auto_save"`

	// SaveInterval is the interval for auto-saving state.
	SaveInterval string `json:"save_interval"`
}

// GitConfig contains Git operation configuration.
// This controls Git integration and auto-commit behavior.
type GitConfig struct {
	// CommitPrefix is the prefix for auto-generated commit messages.
	CommitPrefix string `json:"commit_prefix"`

	// AutoCommit enables automatic Git commits.
	AutoCommit bool `json:"auto_commit"`

	// RequireCleanWorktree requires a clean working tree before operations.
	RequireCleanWorktree bool `json:"require_clean_worktree"`
}

// PlanConfig contains plan management configuration.
// This defines where plan files are stored and how they are processed.
type PlanConfig struct {
	// Dir is the directory containing plan files.
	Dir string `json:"dir"`

	// FileExtension is the file extension for plan files.
	FileExtension string `json:"file_extension"`

	// AutoValidate enables automatic plan format validation.
	AutoValidate bool `json:"auto_validate"`
}

// PromptsConfig contains prompt file path configuration.
// This defines where system prompt templates are located.
type PromptsConfig struct {
	// Dir is the directory containing prompt files.
	Dir string `json:"dir"`

	// Research is the path to the research prompt template.
	Research string `json:"research"`

	// Plan is the path to the plan prompt template.
	Plan string `json:"plan"`

	// Doing is the path to the doing prompt template.
	Doing string `json:"doing"`
}

// DefaultConfig returns a Config with all default values set.
// This represents Level 1 of the configuration hierarchy.
func DefaultConfig() *Config {
	return &Config{
		Version: DefaultVersion,
		AICli: AICliConfig{
			Command:               DefaultAICliCommand,
			EnvVar:                DefaultAICliEnvVar,
			DefaultTimeout:        DefaultAICliDefaultTimeout,
			MaxTimeout:            DefaultAICliMaxTimeout,
			EnableSkipPermissions: DefaultAICliEnableSkipPermissions,
			DefaultArgs:           DefaultAICliDefaultArgs,
			OutputFormat:          DefaultAICliOutputFormat,
		},
		Execution: ExecutionConfig{
			MaxRetryCount:   DefaultExecutionMaxRetryCount,
			AutoGitCommit:   DefaultExecutionAutoGitCommit,
			ContinueOnError: DefaultExecutionContinueOnError,
			ParallelJobs:    DefaultExecutionParallelJobs,
		},
		Logging: LoggingConfig{
			Level:  DefaultLoggingLevel,
			Format: DefaultLoggingFormat,
			Output: DefaultLoggingOutput,
			File: FileConfig{
				Enabled:    DefaultLoggingFileEnabled,
				Path:       DefaultLoggingFilePath,
				MaxSize:    DefaultLoggingFileMaxSize,
				MaxBackups: DefaultLoggingFileMaxBackups,
				MaxAge:     DefaultLoggingFileMaxAge,
			},
		},
		State: StateConfig{
			File:         DefaultStateFile,
			AutoSave:     DefaultStateAutoSave,
			SaveInterval: DefaultStateSaveInterval,
		},
		Git: GitConfig{
			CommitPrefix:         DefaultGitCommitPrefix,
			AutoCommit:           DefaultGitAutoCommit,
			RequireCleanWorktree: DefaultGitRequireCleanWorktree,
		},
		Plan: PlanConfig{
			Dir:           DefaultPlanDir,
			FileExtension: DefaultPlanFileExtension,
			AutoValidate:  DefaultPlanAutoValidate,
		},
		Prompts: PromptsConfig{
			Dir:      DefaultPromptsDir,
			Research: DefaultPromptsResearch,
			Plan:     DefaultPromptsPlan,
			Doing:    DefaultPromptsDoing,
		},
	}
}

// Duration returns the DefaultTimeout as a time.Duration.
// Returns the provided default if parsing fails.
func (c *AICliConfig) Duration(defaultVal time.Duration) time.Duration {
	d, err := time.ParseDuration(c.DefaultTimeout)
	if err != nil {
		return defaultVal
	}
	return d
}

// MaxTimeoutDuration returns the MaxTimeout as a time.Duration.
// Returns the provided default if parsing fails.
func (c *AICliConfig) MaxTimeoutDuration(defaultVal time.Duration) time.Duration {
	d, err := time.ParseDuration(c.MaxTimeout)
	if err != nil {
		return defaultVal
	}
	return d
}

// SaveIntervalDuration returns the SaveInterval as a time.Duration.
// Returns the provided default if parsing fails.
func (c *StateConfig) SaveIntervalDuration(defaultVal time.Duration) time.Duration {
	d, err := time.ParseDuration(c.SaveInterval)
	if err != nil {
		return defaultVal
	}
	return d
}
