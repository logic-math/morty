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
	logger             logging.Logger
	timeout            time.Duration
	aiCliCaller        callcli.AICliCaller
	conversationParser *callcli.ConversationParser
	logsDir            string
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
	// ConversationJSON contains the raw conversation JSON from AI CLI (if available)
	ConversationJSON string
	// ConversationLogPath is the path to the saved conversation log file
	ConversationLogPath string
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

	logsDir := ".morty/logs"
	return &TaskRunner{
		logger:             logger,
		timeout:            DefaultTaskTimeout,
		aiCliCaller:        aiCliCaller,
		conversationParser: callcli.NewConversationParser(logsDir),
		logsDir:            logsDir,
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

// NewTaskRunnerWithConfig creates a new TaskRunner with custom configuration.
//
// Parameters:
//   - logger: The logger for recording execution progress
//   - aiCliCaller: Optional AI CLI caller. If nil, a new one is created.
//   - timeout: The timeout duration for task execution
//   - logsDir: The directory for storing conversation logs
//
// Returns:
//   - A pointer to a new TaskRunner instance
func NewTaskRunnerWithConfig(logger logging.Logger, aiCliCaller callcli.AICliCaller, timeout time.Duration, logsDir string) *TaskRunner {
	if aiCliCaller == nil {
		aiCliCaller = callcli.NewAICliCaller()
	}

	if logsDir == "" {
		logsDir = ".morty/logs"
	}

	return &TaskRunner{
		logger:             logger,
		timeout:            timeout,
		aiCliCaller:        aiCliCaller,
		conversationParser: callcli.NewConversationParser(logsDir),
		logsDir:            logsDir,
	}
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

// RunWithLogging executes a task and automatically parses/saves conversation logs.
//
// Parameters:
//   - ctx: The context for cancellation and timeout
//   - module: The module name (for organizing logs)
//   - job: The job name (for organizing logs)
//   - taskDesc: Description of the task being executed
//   - prompt: The prompt content to pass to the AI CLI
//
// Returns:
//   - A TaskResult containing execution details and log paths
//   - An error if the task fails to execute
func (tr *TaskRunner) RunWithLogging(ctx context.Context, module, job, taskDesc string, prompt string) (*TaskResult, error) {
	// Execute the task normally
	result, err := tr.Run(ctx, taskDesc, prompt)

	// Try to parse conversation logs from stdout
	// Claude Code may return JSON conversation data in stdout
	if result != nil && result.Stdout != "" {
		tr.parseAndSaveConversationLog(result, module, job)
	}

	return result, err
}

// parseAndSaveConversationLog attempts to parse and save conversation logs from task output.
func (tr *TaskRunner) parseAndSaveConversationLog(result *TaskResult, module, job string) {
	if tr.conversationParser == nil {
		return
	}

	// Try to extract JSON from stdout
	// Claude Code might output JSON in various formats
	jsonData := tr.extractConversationJSON(result.Stdout)
	if jsonData == "" {
		tr.logger.Debug("No conversation JSON found in output")
		return
	}

	result.ConversationJSON = jsonData

	// Parse and save the conversation log
	logPath, err := tr.conversationParser.ParseAndSave(jsonData, module, job)
	if err != nil {
		tr.logger.Warn("Failed to parse and save conversation log",
			logging.String("error", err.Error()))
		return
	}

	result.ConversationLogPath = logPath
	tr.logger.Info("Conversation log saved",
		logging.String("log_path", logPath))
}

// extractConversationJSON attempts to extract conversation JSON from output.
// It looks for JSON blocks that contain conversation structure.
func (tr *TaskRunner) extractConversationJSON(output string) string {
	// Look for JSON that starts with { and contains "messages" field
	// This is a simple heuristic - you may need to adjust based on actual format

	// Try to find JSON block with messages array
	startIdx := -1
	braceCount := 0
	inString := false
	escapeNext := false

	for i := 0; i < len(output); i++ {
		ch := output[i]

		if escapeNext {
			escapeNext = false
			continue
		}

		if ch == '\\' {
			escapeNext = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if ch == '{' {
			if braceCount == 0 {
				startIdx = i
			}
			braceCount++
		} else if ch == '}' {
			braceCount--
			if braceCount == 0 && startIdx != -1 {
				// Found a complete JSON block
				jsonBlock := output[startIdx : i+1]
				// Check if it looks like a conversation JSON
				if stringContains(jsonBlock, "\"messages\"") ||
					stringContains(jsonBlock, "\"role\"") {
					return jsonBlock
				}
				startIdx = -1
			}
		}
	}

	return ""
}

// SetLogsDir updates the logs directory for conversation logs.
func (tr *TaskRunner) SetLogsDir(logsDir string) {
	tr.logsDir = logsDir
	tr.conversationParser = callcli.NewConversationParser(logsDir)
}

// GetLogsDir returns the current logs directory.
func (tr *TaskRunner) GetLogsDir() string {
	return tr.logsDir
}
