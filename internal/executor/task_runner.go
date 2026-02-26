// Package executor provides job execution engine for Morty.
package executor

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/morty/morty/internal/callcli"
	"github.com/morty/morty/internal/logging"
)

// DefaultTaskTimeout is the default timeout for task execution (10 minutes).
const DefaultTaskTimeout = 10 * time.Minute

// TaskRunner handles the execution of a single task.
// It manages the AI CLI call, timeout control, output capture, and exit code handling.
type TaskRunner struct {
	logger      logging.Logger
	timeout     time.Duration
	aiCliCaller callcli.AICliCaller
}

// TaskResult represents the outcome of executing a task.
type TaskResult struct {
	// Success indicates whether the task executed successfully (exit code 0)
	Success bool
	// ExitCode is the process exit code
	ExitCode int
	// Stdout contains the captured standard output
	Stdout string
	// Stderr contains the captured standard error
	Stderr string
	// Duration is how long the task took to execute
	Duration time.Duration
	// TimedOut indicates if the task was terminated due to timeout
	TimedOut bool
	// Error contains any execution error
	Error error
}

// NewTaskRunner creates a new TaskRunner with the given dependencies.
//
// Parameters:
//   - logger: The logger for recording execution progress
//   - aiCliCaller: Optional AI CLI caller. If nil, a new one is created.
//
// Returns:
//   - A pointer to a new TaskRunner instance
func NewTaskRunner(logger logging.Logger, aiCliCaller callcli.AICliCaller) *TaskRunner {
	if aiCliCaller == nil {
		aiCliCaller = callcli.NewAICliCaller()
	}

	return &TaskRunner{
		logger:      logger,
		timeout:     DefaultTaskTimeout,
		aiCliCaller: aiCliCaller,
	}
}

// NewTaskRunnerWithTimeout creates a new TaskRunner with a custom timeout.
//
// Parameters:
//   - logger: The logger for recording execution progress
//   - aiCliCaller: Optional AI CLI caller. If nil, a new one is created.
//   - timeout: The timeout duration for task execution
//
// Returns:
//   - A pointer to a new TaskRunner instance
func NewTaskRunnerWithTimeout(logger logging.Logger, aiCliCaller callcli.AICliCaller, timeout time.Duration) *TaskRunner {
	tr := NewTaskRunner(logger, aiCliCaller)
	tr.timeout = timeout
	return tr
}

// Run executes a single task with the given description and prompt.
// It handles timeout control, AI CLI execution, output capture, and exit code handling.
//
// Parameters:
//   - ctx: The context for cancellation and timeout
//   - taskDesc: Description of the task being executed
//   - prompt: The prompt content to pass to the AI CLI
//
// Returns:
//   - A TaskResult containing execution details
//   - An error if the task fails to execute
func (tr *TaskRunner) Run(ctx context.Context, taskDesc string, prompt string) (*TaskResult, error) {
	tr.logger.Info("Starting task execution",
		logging.String("task_desc", taskDesc),
		logging.Any("timeout", tr.timeout),
	)

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, tr.timeout)
	defer cancel()

	// Execute the AI CLI call with prompt content
	callResult, err := tr.aiCliCaller.CallWithPromptContent(ctx, prompt)

	// Build task result from call result
	result := &TaskResult{
		Success:  callResult != nil && callResult.ExitCode == 0 && err == nil,
		ExitCode: 0,
		Stdout:   "",
		Stderr:   "",
		Duration: 0,
		TimedOut: ctx.Err() == context.DeadlineExceeded,
		Error:    err,
	}

	if callResult != nil {
		result.ExitCode = callResult.ExitCode
		result.Stdout = callResult.Stdout
		result.Stderr = callResult.Stderr
		result.Duration = callResult.Duration
		result.TimedOut = callResult.TimedOut
	}

	// Handle different execution outcomes
	if err != nil {
		// Check for timeout
		if ctx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
			result.Success = false
			result.ExitCode = -1
			tr.logger.Error("Task execution timed out",
				logging.String("task_desc", taskDesc),
				logging.Any("timeout", tr.timeout),
			)
			return result, fmt.Errorf("task execution timed out after %v: %w", tr.timeout, err)
		}

		// Check for command not found
		if isCommandNotFound(err) {
			result.Success = false
			tr.logger.Error("AI CLI command not found",
				logging.String("task_desc", taskDesc),
				logging.String("error", err.Error()),
			)
			return result, fmt.Errorf("AI CLI command not found: %w", err)
		}

		// Other execution errors
		result.Success = false
		tr.logger.Error("Task execution failed",
			logging.String("task_desc", taskDesc),
			logging.Int("exit_code", result.ExitCode),
			logging.String("error", err.Error()),
		)
		return result, fmt.Errorf("task execution failed: %w", err)
	}

	// Check exit code
	if result.ExitCode != 0 {
		result.Success = false
		tr.logger.Error("Task exited with non-zero code",
			logging.String("task_desc", taskDesc),
			logging.Int("exit_code", result.ExitCode),
			logging.String("stderr", result.Stderr),
		)
		return result, fmt.Errorf("task exited with code %d: %s", result.ExitCode, result.Stderr)
	}

	// Success
	tr.logger.Success("Task completed successfully",
		logging.String("task_desc", taskDesc),
		logging.Any("duration", result.Duration),
	)

	return result, nil
}

// SetTimeout updates the timeout for task execution.
func (tr *TaskRunner) SetTimeout(timeout time.Duration) {
	tr.timeout = timeout
}

// GetTimeout returns the current timeout setting.
func (tr *TaskRunner) GetTimeout() time.Duration {
	return tr.timeout
}

// SetAICaller sets a custom AI CLI caller (useful for testing).
func (tr *TaskRunner) SetAICaller(caller callcli.AICliCaller) {
	if caller != nil {
		tr.aiCliCaller = caller
	}
}

// GetAICaller returns the current AI CLI caller.
func (tr *TaskRunner) GetAICaller() callcli.AICliCaller {
	return tr.aiCliCaller
}

// isCommandNotFound checks if an error is due to command not found.
func isCommandNotFound(err error) bool {
	if err == nil {
		return false
	}
	// Check for exec.Error which is returned when command is not found
	if _, ok := err.(*exec.Error); ok {
		return true
	}
	// Check error message for common patterns
	errStr := err.Error()
	return stringContains(errStr, "executable file not found") ||
		stringContains(errStr, "command not found") ||
		stringContains(errStr, "M5001")
}

// stringContains checks if a string contains a substring.
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || stringContainsSubstring(s, substr))
}

// stringContainsSubstring checks if s contains substr.
func stringContainsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
