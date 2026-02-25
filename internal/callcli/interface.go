// Package callcli provides functionality for executing external CLI commands.
// It wraps os/exec to provide a testable, configurable interface for running
// subprocesses with proper error handling, output capture, and context support.
package callcli

import (
	"context"
	"time"
)

// Result represents the outcome of executing a command.
type Result struct {
	// Stdout contains the command's standard output
	Stdout string
	// Stderr contains the command's standard error
	Stderr string
	// ExitCode is the command's exit code (0 typically means success)
	ExitCode int
	// Duration is how long the command took to execute
	Duration time.Duration
	// Command is the executed command with arguments (for debugging)
	Command string
	// TimedOut is true if the command was terminated due to timeout
	TimedOut bool
	// Interrupted is true if the command was interrupted by a signal
	Interrupted bool
}

// Options contains configuration options for command execution.
type Options struct {
	// WorkingDir sets the working directory for the command
	WorkingDir string
	// Env is a map of environment variables to set (in addition to current env)
	Env map[string]string
	// Timeout is the maximum time to wait for the command (0 means no timeout)
	Timeout time.Duration
	// Stdin is the input to provide to the command
	Stdin string
	// GracefulPeriod is the time to wait after SIGTERM before sending SIGKILL (0 means no graceful termination)
	GracefulPeriod time.Duration
}

// Caller defines the interface for executing CLI commands.
// Implementations should handle command execution, output capture,
// and proper error handling with context support.
type Caller interface {
	// Call executes a command synchronously with the given arguments.
	// Returns a Result containing stdout, stderr, exit code, and duration.
	// The command name is resolved via PATH if not an absolute path.
	Call(ctx context.Context, name string, args ...string) (*Result, error)

	// CallWithOptions executes a command with additional options like
	// working directory, environment variables, and timeout.
	CallWithOptions(ctx context.Context, name string, args []string, opts Options) (*Result, error)

	// CallWithCtx executes a command with context control and returns a CallHandler
	// for managing the running process with timeout and cancellation support.
	CallWithCtx(ctx context.Context, name string, args []string, opts Options) (CallHandler, error)

	// CallAsync executes a command asynchronously and returns a CallHandler
	// for managing the running process.
	CallAsync(ctx context.Context, name string, args ...string) (CallHandler, error)

	// CallAsyncWithOptions executes a command asynchronously with additional options.
	CallAsyncWithOptions(ctx context.Context, name string, args []string, opts Options) (CallHandler, error)

	// SetDefaultTimeout sets the default timeout for all calls.
	SetDefaultTimeout(timeout time.Duration)

	// GetDefaultTimeout returns the current default timeout.
	GetDefaultTimeout() time.Duration
}

// CallHandler provides control over an asynchronously running command.
// It allows waiting for completion, checking status, and terminating the process.
type CallHandler interface {
	// Wait blocks until the command finishes executing and returns the result.
	Wait() (*Result, error)

	// Kill terminates the running process.
	Kill() error

	// PID returns the process ID of the running command.
	// Returns -1 if the process hasn't started or has already finished.
	PID() int

	// Running returns true if the process is still running.
	Running() bool
}

// Ensure CallerImpl implements Caller interface
// This will be defined in caller.go
