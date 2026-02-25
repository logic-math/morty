// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/morty/morty/pkg/errors"
)

// asyncHandler implements the CallHandler interface for managing async processes.
type asyncHandler struct {
	cmd           *exec.Cmd
	pid           int
	mu            sync.RWMutex
	started       bool
	finished      bool
	result        *Result
	err           error
	done          chan struct{}
	outputHandler *OutputHandler
	timeout       time.Duration
	commandStr    string
}

// CallAsync executes a command asynchronously and returns a CallHandler.
func (c *CallerImpl) CallAsync(ctx context.Context, name string, args ...string) (CallHandler, error) {
	return c.CallAsyncWithOptions(ctx, name, args, Options{})
}

// CallAsyncWithOptions executes a command asynchronously with additional options.
func (c *CallerImpl) CallAsyncWithOptions(ctx context.Context, name string, args []string, opts Options) (CallHandler, error) {
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

	// Determine timeout
	timeout := opts.Timeout
	if timeout == 0 && c.defaultTimeout > 0 {
		timeout = c.defaultTimeout
	}

	// Create command without context (we'll handle cancellation manually for async)
	cmd := exec.Command(execPath, args...)

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

	// Create output handler
	outputHandler, err := NewOutputHandler(opts.Output)
	if err != nil {
		return nil, errors.Wrap(err, "M5002", "failed to create output handler").
			WithDetail("command", commandStr)
	}

	// Create handler
	handler := &asyncHandler{
		cmd:           cmd,
		done:          make(chan struct{}),
		timeout:       timeout,
		outputHandler: outputHandler,
		commandStr:    commandStr,
	}

	// Set up stdout and stderr writers
	cmd.Stdout = outputHandler.StdoutWriter()
	cmd.Stderr = outputHandler.StderrWriter()

	// Start the command
	if err := cmd.Start(); err != nil {
		outputHandler.Close()
		return nil, errors.Wrap(err, "M5002", "failed to start command").
			WithDetail("command", commandStr)
	}

	// Record PID and started state
	handler.mu.Lock()
	handler.pid = cmd.Process.Pid
	handler.started = true
	handler.mu.Unlock()

	// Start goroutine to wait for completion
	go handler.waitForCompletion(commandStr, timeout)

	return handler, nil
}

// waitForCompletion waits for the command to finish and records the result.
func (h *asyncHandler) waitForCompletion(commandStr string, timeout time.Duration) {
	startTime := time.Now()

	// Create a channel to signal completion
	done := make(chan error, 1)
	go func() {
		done <- h.cmd.Wait()
	}()

	var waitErr error
	var timedOut bool

	// Wait for completion or timeout
	if timeout > 0 {
		select {
		case waitErr = <-done:
			// Command finished normally
		case <-time.After(timeout):
			// Timeout occurred
			timedOut = true
			h.cmd.Process.Kill()
			waitErr = <-done
		}
	} else {
		waitErr = <-done
	}

	duration := time.Since(startTime)

	// Close output handler
	h.outputHandler.Close()

	// Get captured output
	stdout := h.outputHandler.GetStdout()
	stderr := h.outputHandler.GetStderr()

	// Build result
	h.mu.Lock()
	defer h.mu.Unlock()

	result := &Result{
		Stdout:   strings.TrimSpace(stdout),
		Stderr:   strings.TrimSpace(stderr),
		ExitCode: 0,
		Duration: duration,
		Command:  commandStr,
		TimedOut: timedOut,
	}

	// Handle execution result
	if waitErr != nil {
		if timedOut {
			result.ExitCode = -1
			h.err = errors.Wrap(waitErr, "M5003", "execution timeout").
				WithDetail("command", commandStr).
				WithDetail("timeout", timeout.String())
		} else if exitError, ok := waitErr.(*exec.ExitError); ok {
			// Get exit code from the process state
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				result.ExitCode = status.ExitStatus()
				// Check if it was killed by a signal
				if status.Signaled() {
					h.err = errors.Wrap(waitErr, "M5004", "process killed by signal").
						WithDetail("command", commandStr).
						WithDetail("signal", status.Signal().String())
				}
			} else {
				result.ExitCode = exitError.ExitCode()
			}
		} else {
			result.ExitCode = -1
		}

		// If not already set an error, set a generic one
		if h.err == nil {
			h.err = errors.Wrap(waitErr, "M5002", "execution failed").
				WithDetail("command", commandStr).
				WithDetail("exit_code", result.ExitCode).
				WithDetail("stderr", result.Stderr)
		}
	}

	h.result = result
	h.finished = true
	close(h.done)
}

// Wait blocks until the command finishes executing and returns the result.
func (h *asyncHandler) Wait() (*Result, error) {
	<-h.done
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.result, h.err
}

// Kill terminates the running process.
func (h *asyncHandler) Kill() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.started || h.finished {
		return nil
	}

	if h.cmd != nil && h.cmd.Process != nil {
		return h.cmd.Process.Kill()
	}

	return nil
}

// PID returns the process ID of the running command.
func (h *asyncHandler) PID() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.pid
}

// Running returns true if the process is still running.
func (h *asyncHandler) Running() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.started || h.finished {
		return false
	}

	// Try to signal the process to check if it's alive
	if h.cmd != nil && h.cmd.Process != nil {
		// On Unix, Signal(0) checks if process exists without sending a real signal
		err := h.cmd.Process.Signal(syscall.Signal(0))
		return err == nil
	}

	return false
}

// Ensure asyncHandler implements CallHandler interface
var _ CallHandler = (*asyncHandler)(nil)
