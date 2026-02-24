package config

// Default configuration constants.
// These values are compiled into the binary and serve as Level 1
// of the configuration hierarchy (lowest priority).
//
// The configuration hierarchy (highest to lowest priority):
// 1. Command-line arguments
// 2. Environment variables (MORTY_* prefix)
// 3. Project config (.morty/settings.json)
// 4. User global config (~/.morty/config.json)
// 5. Built-in defaults (these constants)
const (
	// DefaultVersion is the configuration format version.
	DefaultVersion = "2.0"
)

// AI CLI default constants.
const (
	// DefaultAICliCommand is the default AI CLI command name.
	DefaultAICliCommand = "ai_cli"

	// DefaultAICliEnvVar is the environment variable for CLI path override.
	DefaultAICliEnvVar = "CLAUDE_CODE_CLI"

	// DefaultAICliDefaultTimeout is the default timeout for CLI operations.
	DefaultAICliDefaultTimeout = "10m"

	// DefaultAICliMaxTimeout is the maximum allowed timeout.
	DefaultAICliMaxTimeout = "30m"

	// DefaultAICliEnableSkipPermissions enables skip permissions by default.
	DefaultAICliEnableSkipPermissions = true

	// DefaultAICliOutputFormat is the default output format.
	DefaultAICliOutputFormat = "json"
)

// DefaultAICliDefaultArgs contains default CLI arguments.
var DefaultAICliDefaultArgs = []string{"--verbose", "--debug"}

// Execution default constants.
const (
	// DefaultExecutionMaxRetryCount is the default maximum retry count.
	DefaultExecutionMaxRetryCount = 3

	// DefaultExecutionAutoGitCommit enables auto-git-commit by default.
	DefaultExecutionAutoGitCommit = true

	// DefaultExecutionContinueOnError disables continue-on-error by default.
	DefaultExecutionContinueOnError = false

	// DefaultExecutionParallelJobs is the default number of parallel jobs.
	DefaultExecutionParallelJobs = 1
)

// Logging default constants.
const (
	// DefaultLoggingLevel is the default log level.
	DefaultLoggingLevel = "info"

	// DefaultLoggingFormat is the default log format.
	DefaultLoggingFormat = "json"

	// DefaultLoggingOutput is the default log output destination.
	DefaultLoggingOutput = "stdout"

	// DefaultLoggingFileEnabled enables file logging by default.
	DefaultLoggingFileEnabled = true

	// DefaultLoggingFilePath is the default log file path.
	DefaultLoggingFilePath = ".morty/doing/logs/morty.log"

	// DefaultLoggingFileMaxSize is the default maximum log file size.
	DefaultLoggingFileMaxSize = "10MB"

	// DefaultLoggingFileMaxBackups is the default number of backup files.
	DefaultLoggingFileMaxBackups = 5

	// DefaultLoggingFileMaxAge is the default log file retention in days.
	DefaultLoggingFileMaxAge = 7
)

// State default constants.
const (
	// DefaultStateFile is the default status file path.
	DefaultStateFile = ".morty/status.json"

	// DefaultStateAutoSave enables auto-save by default.
	DefaultStateAutoSave = true

	// DefaultStateSaveInterval is the default auto-save interval.
	DefaultStateSaveInterval = "30s"
)

// Git default constants.
const (
	// DefaultGitCommitPrefix is the default commit message prefix.
	DefaultGitCommitPrefix = "morty"

	// DefaultGitAutoCommit enables auto-commit by default.
	DefaultGitAutoCommit = true

	// DefaultGitRequireCleanWorktree disables clean worktree requirement by default.
	DefaultGitRequireCleanWorktree = false
)

// Plan default constants.
const (
	// DefaultPlanDir is the default plan directory.
	DefaultPlanDir = ".morty/plan"

	// DefaultPlanFileExtension is the default plan file extension.
	DefaultPlanFileExtension = ".md"

	// DefaultPlanAutoValidate enables auto-validation by default.
	DefaultPlanAutoValidate = true
)

// Prompts default constants.
const (
	// DefaultPromptsDir is the default prompts directory.
	DefaultPromptsDir = "prompts"

	// DefaultPromptsResearch is the default research prompt path.
	DefaultPromptsResearch = "prompts/research.md"

	// DefaultPromptsPlan is the default plan prompt path.
	DefaultPromptsPlan = "prompts/plan.md"

	// DefaultPromptsDoing is the default doing prompt path.
	DefaultPromptsDoing = "prompts/doing.md"
)

// Environment variable names.
const (
	// EnvMortyHome is the environment variable for Morty home directory.
	EnvMortyHome = "MORTY_HOME"

	// EnvMortyConfig is the environment variable for config file path.
	EnvMortyConfig = "MORTY_CONFIG"

	// EnvMortyLogLevel is the environment variable for log level override.
	EnvMortyLogLevel = "MORTY_LOG_LEVEL"

	// EnvMortyDebug is the environment variable for debug mode.
	EnvMortyDebug = "MORTY_DEBUG"
)

// Path constants.
const (
	// DefaultWorkDir is the default Morty working directory.
	DefaultWorkDir = ".morty"

	// DefaultUserConfigDir is the default user config directory.
	DefaultUserConfigDir = "~/.morty"

	// DefaultUserConfigFile is the default user config file path.
	DefaultUserConfigFile = "~/.morty/config.json"

	// DefaultProjectConfigFile is the default project config file path.
	DefaultProjectConfigFile = ".morty/settings.json"
)
