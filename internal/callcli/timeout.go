// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/morty/morty/pkg/errors"
)

// CallWithCtx executes a command with context control and returns a CallHandler
// for managing the running process with timeout and cancellation support.
func (c *CallerImpl) CallWithCtx(ctx context.Context, name string, args []string, opts Options) (CallHandler, error) {
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

	// Create command
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

	// Create handler
	handler := &ctxHandler{
		cmd:            cmd,
		done:           make(chan struct{}),
		timeout:        timeout,
		gracefulPeriod: opts.GracefulPeriod,
		commandStr:     commandStr,
	}

	// Capture stdout and stderr
	cmd.Stdout = &handler.stdout
	cmd.Stderr = &handler.stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "M5002", "failed to start command").
			WithDetail("command", commandStr)
	}

	// Record PID and started state
	handler.mu.Lock()
	handler.pid = cmd.Process.Pid
	handler.started = true
	handler.mu.Unlock()

	// Start goroutine to wait for completion with context and timeout support
	go handler.waitWithContext(ctx, timeout)

	return handler, nil
}

// ctxHandler implements the CallHandler interface for context-controlled processes.
type ctxHandler struct {
	cmd            *exec.Cmd
	pid            int
	mu             sync.RWMutex
	started        bool
	finished       bool
	result         *Result
	err            error
	done           chan struct{}
	stdout         bytes.Buffer
	stderr         bytes.Buffer
	timeout        time.Duration
	gracefulPeriod time.Duration
	commandStr     string
}

// waitWithContext waits for the command to finish with context and timeout support.
func (h *ctxHandler) waitWithContext(ctx context.Context, timeout time.Duration) {
	startTime := time.Now()

	// Create channels for completion and context cancellation
	done := make(chan error, 1)
	go func() {
		done <- h.cmd.Wait()
	}()

	var waitErr error
	var timedOut bool
	var cancelled bool

	// Determine which context to use for timeout
	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timeoutChan = time.After(timeout)
	}

	// Wait for completion, timeout, or context cancellation
	select {
	case waitErr = <-done:
		// Command finished normally
	case <-timeoutChan:
		// Timeout occurred
		timedOut = true
		h.terminateProcess()
		waitErr = <-done
	case <-ctx.Done():
		// Context cancelled
		cancelled = true
		h.terminateProcess()
		waitErr = <-done
	}

	duration := time.Since(startTime)

	// Build result
	h.mu.Lock()
	defer h.mu.Unlock()

	result := &Result{
		Stdout:      strings.TrimSpace(h.stdout.String()),
		Stderr:      strings.TrimSpace(h.stderr.String()),
		ExitCode:    0,
		Duration:    duration,
		Command:     h.commandStr,
		TimedOut:    timedOut,
		Interrupted: timedOut || cancelled,
	}

	// Handle execution result
	if waitErr != nil {
		if timedOut {
			result.ExitCode = -1
			h.err = errors.Wrap(waitErr, "M5003", "execution timeout").
				WithDetail("command", h.commandStr).
				WithDetail("timeout", timeout.String())
		} else if cancelled {
			result.ExitCode = -1
			h.err = errors.Wrap(waitErr, "M5007", "context cancelled during execution").
				WithDetail("command", h.commandStr)
		} else if exitError, ok := waitErr.(*exec.ExitError); ok {
			// Get exit code from the process state
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				result.ExitCode = status.ExitStatus()
				// Check if it was killed by a signal
				if status.Signaled() {
					h.err = errors.Wrap(waitErr, "M5004", "process killed by signal").
						WithDetail("command", h.commandStr).
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
				WithDetail("command", h.commandStr).
				WithDetail("exit_code", result.ExitCode).
				WithDetail("stderr", result.Stderr)
		}
	}

	h.result = result
	h.finished = true
	close(h.done)
}

// terminateProcess performs graceful termination (SIGTERM -> wait -> SIGKILL).
func (h *ctxHandler) terminateProcess() {
	if h.cmd == nil || h.cmd.Process == nil {
		return
	}

	// Send SIGTERM first for graceful termination
	if err := h.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGKILL immediately
		h.cmd.Process.Kill()
		return
	}

	// If graceful period is set, wait for it before sending SIGKILL
	if h.gracefulPeriod > 0 {
		done := make(chan struct{})
		go func() {
			h.cmd.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Process exited gracefully
			return
		case <-time.After(h.gracefulPeriod):
			// Grace period expired, send SIGKILL
			h.cmd.Process.Kill()
		}
	} else {
		// No graceful period, send SIGKILL immediately
		h.cmd.Process.Kill()
	}
}

// Wait blocks until the command finishes executing and returns the result.
func (h *ctxHandler) Wait() (*Result, error) {
	<-h.done
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.result, h.err
}

// Kill terminates the running process.
func (h *ctxHandler) Kill() error {
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
func (h *ctxHandler) PID() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.pid
}

// Running returns true if the process is still running.
func (h *ctxHandler) Running() bool {
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

// Ensure ctxHandler implements CallHandler interface
var _ CallHandler = (*ctxHandler)(nil)
