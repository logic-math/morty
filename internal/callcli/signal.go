// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/morty/morty/pkg/errors"
)

// SignalHandler manages signal handling for subprocesses.
type SignalHandler struct {
	mu              sync.RWMutex
	cmd             *exec.Cmd
	pid             int
	started         bool
	finished        bool
	interrupted     bool
	signalReceived  os.Signal
	result          *Result
	err             error
	done            chan struct{}
	outputHandler   *OutputHandler
	timeout         time.Duration
	gracefulPeriod  time.Duration
	commandStr      string
	signalCh        chan os.Signal
	stopSignalCh    chan struct{}
	childProcesses  []*exec.Cmd
}

// InterruptState represents the state of an interrupted process.
type InterruptState struct {
	Command       string        `json:"command"`
	Args          []string      `json:"args"`
	PID           int           `json:"pid"`
	Signal        string        `json:"signal"`
	Timestamp     time.Time     `json:"timestamp"`
	PartialStdout string        `json:"partial_stdout,omitempty"`
	PartialStderr string        `json:"partial_stderr,omitempty"`
	Duration      time.Duration `json:"duration"`
}

// globalSignalHandler manages process-level signal handling for all subprocesses.
type globalSignalHandler struct {
	mu        sync.RWMutex
	handlers  map[int]*SignalHandler
	stopCh    chan struct{}
	started   bool
}

var (
	globalHandler     *globalSignalHandler
	globalHandlerOnce sync.Once
)

// getGlobalSignalHandler returns the singleton global signal handler.
func getGlobalSignalHandler() *globalSignalHandler {
	globalHandlerOnce.Do(func() {
		globalHandler = &globalSignalHandler{
			handlers: make(map[int]*SignalHandler),
			stopCh:   make(chan struct{}),
		}
	})
	return globalHandler
}

// Start begins listening for OS signals.
func (g *globalSignalHandler) Start() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.started {
		return
	}

	g.started = true

	// Set up signal handling for SIGINT and SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case sig := <-sigCh:
				g.handleSignal(sig)
			case <-g.stopCh:
				signal.Stop(sigCh)
				return
			}
		}
	}()
}

// Stop stops the global signal handler.
func (g *globalSignalHandler) Stop() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.started {
		return
	}

	close(g.stopCh)
	g.started = false
}

// Register registers a signal handler for a process.
func (g *globalSignalHandler) Register(handler *SignalHandler) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.handlers[handler.pid] = handler
}

// Unregister removes a signal handler.
func (g *globalSignalHandler) Unregister(pid int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.handlers, pid)
}

// handleSignal forwards signals to all registered handlers.
func (g *globalSignalHandler) handleSignal(sig os.Signal) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, handler := range g.handlers {
		handler.handleSignal(sig)
	}
}

// CallWithSignal executes a command with signal handling support.
func (c *CallerImpl) CallWithSignal(ctx context.Context, name string, args []string, opts Options) (*SignalHandler, error) {
	// Start global signal handler
	getGlobalSignalHandler().Start()

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
		return nil, errors.Wrap(err, "M5001", "command not found").
			WithDetail("command", name)
	}

	// Determine timeout
	timeout := opts.Timeout
	if timeout == 0 && c.defaultTimeout > 0 {
		timeout = c.defaultTimeout
	}

	// Create command with process group for proper signal forwarding
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
	handler := &SignalHandler{
		cmd:            cmd,
		done:           make(chan struct{}),
		timeout:        timeout,
		gracefulPeriod: opts.GracefulPeriod,
		outputHandler:  outputHandler,
		commandStr:     commandStr,
		signalCh:       make(chan os.Signal, 1),
		stopSignalCh:   make(chan struct{}),
		childProcesses: make([]*exec.Cmd, 0),
	}

	// Set up stdout and stderr writers
	cmd.Stdout = outputHandler.StdoutWriter()
	cmd.Stderr = outputHandler.StderrWriter()

	// Set up process group (Unix only) for signal forwarding
	setupProcessGroup(cmd)

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

	// Register with global handler
	getGlobalSignalHandler().Register(handler)

	// Start goroutine to wait for completion
	go handler.waitWithSignal(ctx, timeout)

	return handler, nil
}

// waitWithSignal waits for the command to finish with signal and timeout support.
func (h *SignalHandler) waitWithSignal(ctx context.Context, timeout time.Duration) {
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

	// Wait for completion, timeout, context cancellation, or signal
	select {
	case waitErr = <-done:
		// Command finished normally
	case <-timeoutChan:
		// Timeout occurred
		timedOut = true
		h.mu.Lock()
		h.interrupted = true
		h.mu.Unlock()
		h.terminateProcess()
		waitErr = <-done
	case <-ctx.Done():
		// Context cancelled
		cancelled = true
		h.mu.Lock()
		h.interrupted = true
		h.mu.Unlock()
		h.terminateProcess()
		waitErr = <-done
	case sig := <-h.signalCh:
		// Signal received
		h.mu.Lock()
		h.interrupted = true
		h.signalReceived = sig
		h.mu.Unlock()
		h.terminateProcess()
		waitErr = <-done
	}

	duration := time.Since(startTime)

	// Unregister from global handler
	getGlobalSignalHandler().Unregister(h.pid)

	// Close output handler
	h.outputHandler.Close()

	// Wait for any child processes to prevent zombies
	h.waitForChildren()

	// Get captured output
	stdout := h.outputHandler.GetStdout()
	stderr := h.outputHandler.GetStderr()

	// Build result
	h.mu.Lock()
	defer h.mu.Unlock()

	result := &Result{
		Stdout:      strings.TrimSpace(stdout),
		Stderr:      strings.TrimSpace(stderr),
		ExitCode:    0,
		Duration:    duration,
		Command:     h.commandStr,
		TimedOut:    timedOut,
		Interrupted: h.interrupted || cancelled,
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
		} else if h.interrupted {
			result.ExitCode = -1
			h.err = errors.Wrap(waitErr, "M5004", "process interrupted by signal").
				WithDetail("command", h.commandStr).
				WithDetail("signal", h.signalReceived.String())
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

// handleSignal handles incoming signals.
func (h *SignalHandler) handleSignal(sig os.Signal) {
	h.mu.Lock()
	if !h.started || h.finished {
		h.mu.Unlock()
		return
	}

	h.interrupted = true
	h.signalReceived = sig
	h.mu.Unlock()

	// Forward signal to process group (outside of lock)
	h.forwardSignal(sig)

	// Also send to signal channel
	h.mu.Lock()
	select {
	case h.signalCh <- sig:
	default:
	}
	h.mu.Unlock()
}

// terminateProcess performs graceful termination.
func (h *SignalHandler) terminateProcess() {
	if h.cmd == nil || h.cmd.Process == nil {
		return
	}

	// Forward SIGTERM to process group first
	h.forwardSignal(syscall.SIGTERM)

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
			h.forwardSignal(syscall.SIGKILL)
		}
	} else {
		// No graceful period, send SIGKILL immediately
		h.forwardSignal(syscall.SIGKILL)
	}
}

// forwardSignal forwards a signal to the process group.
func (h *SignalHandler) forwardSignal(sig os.Signal) {
	if h.cmd == nil || h.cmd.Process == nil {
		return
	}

	// Try to signal the process group first (Unix)
	if err := signalProcessGroup(h.cmd.Process.Pid, sig); err != nil {
		// Fall back to signaling just the process
		h.cmd.Process.Signal(sig)
	}

	// Also forward to child processes
	h.mu.RLock()
	children := make([]*exec.Cmd, len(h.childProcesses))
	copy(children, h.childProcesses)
	h.mu.RUnlock()

	for _, child := range children {
		if child != nil && child.Process != nil {
			child.Process.Signal(sig)
		}
	}
}

// waitForChildren waits for any child processes to prevent zombies.
func (h *SignalHandler) waitForChildren() {
	h.mu.Lock()
	children := make([]*exec.Cmd, len(h.childProcesses))
	copy(children, h.childProcesses)
	h.mu.Unlock()

	for _, child := range children {
		if child != nil && child.Process != nil {
			// Non-blocking wait to prevent zombies
			go func(c *exec.Cmd) {
				c.Wait()
			}(child)
		}
	}
}

// Wait blocks until the command finishes executing and returns the result.
func (h *SignalHandler) Wait() (*Result, error) {
	<-h.done
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.result, h.err
}

// Kill terminates the running process.
func (h *SignalHandler) Kill() error {
	h.mu.Lock()
	if !h.started || h.finished {
		h.mu.Unlock()
		return nil
	}

	cmd := h.cmd
	h.interrupted = true
	h.signalReceived = syscall.SIGKILL
	h.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		return h.forwardSignalAndWait(syscall.SIGKILL)
	}

	return nil
}

// forwardSignalAndWait forwards a signal and waits briefly for process to exit.
func (h *SignalHandler) forwardSignalAndWait(sig os.Signal) error {
	h.forwardSignal(sig)

	// Brief wait to allow process to exit
	done := make(chan struct{})
	go func() {
		h.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

// PID returns the process ID of the running command.
func (h *SignalHandler) PID() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.pid
}

// Running returns true if the process is still running.
func (h *SignalHandler) Running() bool {
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

// Interrupted returns true if the process was interrupted by a signal.
func (h *SignalHandler) Interrupted() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.interrupted
}

// SignalReceived returns the signal that interrupted the process, if any.
func (h *SignalHandler) SignalReceived() os.Signal {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.signalReceived
}

// GetInterruptState returns the current interrupt state for potential resume.
func (h *SignalHandler) GetInterruptState() *InterruptState {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var args []string
	if h.cmd != nil {
		args = h.cmd.Args[1:]
	}

	signal := ""
	if h.signalReceived != nil {
		signal = h.signalReceived.String()
	}

	var duration time.Duration
	if h.result != nil {
		duration = h.result.Duration
	}

	return &InterruptState{
		Command:       h.commandStr,
		Args:          args,
		PID:           h.pid,
		Signal:        signal,
		Timestamp:     time.Now(),
		PartialStdout: h.outputHandler.GetStdout(),
		PartialStderr: h.outputHandler.GetStderr(),
		Duration:      duration,
	}
}

// Resume attempts to resume execution after an interrupt.
// Note: This creates a new process as the original cannot be resumed.
func (h *SignalHandler) Resume(ctx context.Context, c *CallerImpl, opts Options) (*SignalHandler, error) {
	h.mu.RLock()
	signal := ""
	if h.signalReceived != nil {
		signal = h.signalReceived.String()
	}
	state := &InterruptState{
		Command: h.commandStr,
		PID:     h.pid,
		Signal:  signal,
	}
	if h.cmd != nil && len(h.cmd.Args) > 0 {
		state.Args = h.cmd.Args[1:]
	}
	h.mu.RUnlock()

	if state.Command == "" {
		return nil, errors.New("M5006", "no command to resume")
	}

	// Parse command name from command string
	parts := strings.Fields(state.Command)
	if len(parts) == 0 {
		return nil, errors.New("M5006", "invalid command string")
	}

	name := parts[0]
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	// Resume with same options but mark as resumed
	return c.CallWithSignal(ctx, name, args, opts)
}

// AddChildProcess adds a child process to track for signal forwarding and zombie prevention.
func (h *SignalHandler) AddChildProcess(cmd *exec.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.childProcesses == nil {
		h.childProcesses = make([]*exec.Cmd, 0)
	}
	h.childProcesses = append(h.childProcesses, cmd)
}

// RemoveChildProcess removes a child process from tracking.
func (h *SignalHandler) RemoveChildProcess(cmd *exec.Cmd) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i, child := range h.childProcesses {
		if child == cmd {
			// Remove by swapping with last element
			h.childProcesses[i] = h.childProcesses[len(h.childProcesses)-1]
			h.childProcesses = h.childProcesses[:len(h.childProcesses)-1]
			break
		}
	}
}

// Ensure SignalHandler implements CallHandler interface
var _ CallHandler = (*SignalHandler)(nil)
