// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/morty/morty/pkg/errors"
)

// CallerImpl implements the Caller interface for executing CLI commands.
type CallerImpl struct {
	defaultTimeout time.Duration
}

// New creates a new Caller with default settings.
func New() *CallerImpl {
	return &CallerImpl{
		defaultTimeout: 0, // No timeout by default
	}
}

// NewWithTimeout creates a new Caller with a default timeout.
func NewWithTimeout(timeout time.Duration) *CallerImpl {
	return &CallerImpl{
		defaultTimeout: timeout,
	}
}

// SetDefaultTimeout sets the default timeout for all calls.
func (c *CallerImpl) SetDefaultTimeout(timeout time.Duration) {
	c.defaultTimeout = timeout
}

// GetDefaultTimeout returns the current default timeout.
func (c *CallerImpl) GetDefaultTimeout() time.Duration {
	return c.defaultTimeout
}

// Call executes a command synchronously with the given arguments.
func (c *CallerImpl) Call(ctx context.Context, name string, args ...string) (*Result, error) {
	return c.CallWithOptions(ctx, name, args, Options{})
}

// CallWithOptions executes a command with additional options.
func (c *CallerImpl) CallWithOptions(ctx context.Context, name string, args []string, opts Options) (*Result, error) {
	startTime := time.Now()

	// Build the full command string for debugging
	commandStr := buildCommandString(name, args)

	// Check if context is already cancelled
	if err := ctx.Err(); err != nil {
		return nil, errors.Wrap(err, "M5007", "context cancelled before execution").
			WithDetail("command", commandStr)
	}

	// Look up the executable path
	execPath, err := exec.LookPath(name)
	if err != nil {
		return nil, errors.Wrap(err, "M5001", "AI CLI command not found").
			WithDetail("command", name)
	}

	// Apply timeout if specified and create timeout context
	timeout := opts.Timeout
	if timeout == 0 && c.defaultTimeout > 0 {
		timeout = c.defaultTimeout
	}

	// If timeout is set, wrap context with timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// Create command with the (possibly wrapped) context
	cmd := exec.CommandContext(ctx, execPath, args...)

	// Set working directory if specified
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}

	// Set up environment variables
	cmd.Env = c.buildEnv(opts.Env)

	// Set up stdin if provided
	if opts.Stdin != "" {
		cmd.Stdin = strings.NewReader(opts.Stdin)
	}

	// Capture stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Execute the command
	runErr := cmd.Run()

	duration := time.Since(startTime)

	// Build result
	result := &Result{
		Stdout:   strings.TrimSpace(stdoutBuf.String()),
		Stderr:   strings.TrimSpace(stderrBuf.String()),
		ExitCode: 0,
		Duration: duration,
		Command:  commandStr,
	}

	// Handle execution result
	if runErr != nil {
		// First check if context was cancelled (timeout takes precedence)
		if ctx.Err() == context.DeadlineExceeded {
			result.ExitCode = -1
			return result, errors.Wrap(runErr, "M5003", "execution timeout").
				WithDetail("command", commandStr).
				WithDetail("timeout", timeout.String())
		}

		// Try to get the exit code
		if exitError, ok := runErr.(*exec.ExitError); ok {
			// Get exit code from the process state
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				result.ExitCode = status.ExitStatus()
				// Check if it was killed by a signal (and not due to timeout)
				if status.Signaled() {
					return result, errors.Wrap(runErr, "M5004", "process killed by signal").
						WithDetail("command", commandStr).
						WithDetail("signal", status.Signal().String())
				}
			} else {
				result.ExitCode = exitError.ExitCode()
			}
		} else {
			result.ExitCode = -1
		}

		// Return the result with an error
		return result, errors.Wrap(runErr, "M5002", "execution failed").
			WithDetail("command", commandStr).
			WithDetail("exit_code", result.ExitCode).
			WithDetail("stderr", result.Stderr)
	}

	return result, nil
}

// buildEnv builds the environment variable slice.
func (c *CallerImpl) buildEnv(additionalEnv map[string]string) []string {
	// Start with current environment
	env := os.Environ()

	// Add/override with provided environment variables
	for key, value := range additionalEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

// buildCommandString builds a command string for debugging.
func buildCommandString(name string, args []string) string {
	if len(args) == 0 {
		return name
	}
	return fmt.Sprintf("%s %s", name, strings.Join(args, " "))
}

// Ensure CallerImpl implements Caller interface
var _ Caller = (*CallerImpl)(nil)
