// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/morty/morty/internal/config"
)

// AICliCaller defines the interface for AI CLI operations.
// It provides methods for calling AI CLI with prompts and content.
type AICliCaller interface {
	// CallWithPrompt calls the AI CLI with a prompt file.
	// The prompt file path is passed to the CLI.
	CallWithPrompt(ctx context.Context, promptFile string) (*Result, error)

	// CallWithPromptContent calls the AI CLI with prompt content via stdin.
	// The content is passed directly to the CLI's stdin.
	CallWithPromptContent(ctx context.Context, content string) (*Result, error)

	// GetCLIPath returns the resolved CLI path (from env var or config).
	GetCLIPath() string

	// BuildArgs builds the CLI arguments based on configuration.
	BuildArgs() []string
}

// AICliCallerImpl implements the AICliCaller interface.
type AICliCallerImpl struct {
	config     *config.AICliConfig
	loader     config.Manager
	cliPath    string
	baseCaller Caller
}

// NewAICliCaller creates a new AI CLI caller with default configuration.
func NewAICliCaller() *AICliCallerImpl {
	cfg := config.DefaultConfig().AICli
	return &AICliCallerImpl{
		config:     &cfg,
		baseCaller: New(),
	}
}

// NewAICliCallerWithLoader creates a new AI CLI caller with a config loader.
func NewAICliCallerWithLoader(loader config.Manager) *AICliCallerImpl {
	cfg := config.DefaultConfig().AICli
	if loader != nil {
		// Try to get config values from loader
		cmd := loader.GetString("ai_cli.command")
		if cmd != "" {
			cfg.Command = cmd
		}
		envVar := loader.GetString("ai_cli.env_var")
		if envVar != "" {
			cfg.EnvVar = envVar
		}
		timeout := loader.GetString("ai_cli.default_timeout")
		if timeout != "" {
			cfg.DefaultTimeout = timeout
		}
		cfg.EnableSkipPermissions = loader.GetBool("ai_cli.enable_skip_permissions")
		outputFmt := loader.GetString("ai_cli.output_format")
		if outputFmt != "" {
			cfg.OutputFormat = outputFmt
		}
		if defaultArgs, err := loader.Get("ai_cli.default_args"); err == nil {
			if args, ok := defaultArgs.([]string); ok {
				cfg.DefaultArgs = args
			}
		}
	}

	return &AICliCallerImpl{
		config:     &cfg,
		loader:     loader,
		baseCaller: New(),
	}
}

// GetCLIPath returns the resolved CLI path.
// It checks the environment variable first, then falls back to config.
func (a *AICliCallerImpl) GetCLIPath() string {
	// Check environment variable first
	envVar := a.config.EnvVar
	if envVar == "" {
		envVar = config.DefaultAICliEnvVar
	}

	if cliPath := os.Getenv(envVar); cliPath != "" {
		return cliPath
	}

	// Fall back to config command
	return a.config.Command
}

// BuildArgs builds the CLI arguments based on configuration.
func (a *AICliCallerImpl) BuildArgs() []string {
	var args []string

	// Add default args from config
	args = append(args, a.config.DefaultArgs...)

	// Add output format flag if specified
	if a.config.OutputFormat != "" {
		args = append(args, "--output-format", a.config.OutputFormat)
	}

	// Add skip permissions flag if enabled
	if a.config.EnableSkipPermissions {
		args = append(args, "--dangerously-skip-permissions")
	}

	return args
}

// CallWithPrompt calls the AI CLI with a prompt file.
func (a *AICliCallerImpl) CallWithPrompt(ctx context.Context, promptFile string) (*Result, error) {
	cliPath := a.GetCLIPath()

	// Resolve absolute path for prompt file
	absPromptFile, err := filepath.Abs(promptFile)
	if err != nil {
		absPromptFile = promptFile
	}

	// Build arguments
	args := a.BuildArgs()
	args = append(args, absPromptFile)

	// Parse timeout from config
	timeout := a.config.Duration(0)

	// Create options
	opts := Options{
		Timeout: timeout,
		Output: OutputConfig{
			Mode: OutputCaptureAndStream,
		},
	}

	// Execute the command
	return a.baseCaller.CallWithOptions(ctx, cliPath, args, opts)
}

// CallWithPromptContent calls the AI CLI with prompt content via stdin.
func (a *AICliCallerImpl) CallWithPromptContent(ctx context.Context, content string) (*Result, error) {
	cliPath := a.GetCLIPath()

	// Build arguments
	args := a.BuildArgs()

	// Parse timeout from config
	timeout := a.config.Duration(0)

	// Create options with stdin
	opts := Options{
		Timeout: timeout,
		Stdin:   content,
		Output: OutputConfig{
			Mode: OutputCaptureAndStream,
		},
	}

	// Execute the command
	return a.baseCaller.CallWithOptions(ctx, cliPath, args, opts)
}

// SetBaseCaller sets a custom base caller (useful for testing).
func (a *AICliCallerImpl) SetBaseCaller(caller Caller) {
	a.baseCaller = caller
}

// GetConfig returns the AI CLI configuration.
func (a *AICliCallerImpl) GetConfig() *config.AICliConfig {
	return a.config
}

// SetCLITimeout sets the timeout for CLI calls.
func (a *AICliCallerImpl) SetCLITimeout(timeout time.Duration) {
	a.baseCaller.SetDefaultTimeout(timeout)
}

// Ensure AICliCallerImpl implements AICliCaller interface
var _ AICliCaller = (*AICliCallerImpl)(nil)
