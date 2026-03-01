// Package executor provides job execution engine for Morty.
// It handles the lifecycle of job execution including state transitions,
// retry logic, prerequisite checking, and Git integration.
package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/morty/morty/internal/callcli"
	"github.com/morty/morty/internal/git"
	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/parser/plan"
	"github.com/morty/morty/internal/state"
)

// Engine defines the interface for job execution.
type Engine interface {
	// ExecuteJob executes a job within a module.
	// It handles the full lifecycle: prerequisite check, state transitions,
	// task execution, retry logic, and Git commit.
	ExecuteJob(ctx context.Context, module, job string) error

	// ExecuteTask executes a single task.
	// This is used internally by ExecuteJob for each task in the job.
	ExecuteTask(ctx context.Context, module, job string, taskIndex int, taskDesc string) error

	// ResumeJob resumes a previously interrupted job.
	// It continues execution from the last uncompleted task.
	ResumeJob(ctx context.Context, module, job string) error
}

// Config holds the configuration for the executor engine.
type Config struct {
	// MaxRetries is the maximum number of retry attempts for a failed job.
	MaxRetries int
	// AutoCommit enables automatic Git commit after job completion.
	AutoCommit bool
	// CommitPrefix is the prefix used for Git commit messages.
	CommitPrefix string
	// WorkingDir is the working directory for Git operations.
	WorkingDir string
	// PromptsDir is the directory containing prompt templates.
	PromptsDir string
	// PlanDir is the directory containing plan files.
	PlanDir string
}

// DefaultConfig returns the default executor configuration.
func DefaultConfig() *Config {
	return &Config{
		MaxRetries:   3,
		AutoCommit:   true,
		CommitPrefix: "morty:",
		WorkingDir:   ".",
	}
}

// ExecutionResult represents the result of job execution.
type ExecutionResult struct {
	// Status is the final status of the job.
	Status state.Status
	// TasksCompleted is the number of tasks completed.
	TasksCompleted int
	// TasksTotal is the total number of tasks.
	TasksTotal int
	// Summary is a brief summary of the execution.
	Summary string
	// Module is the module name.
	Module string
	// Job is the job name.
	Job string
	// RetryCount is the number of retries performed.
	RetryCount int
	// Error is the error message if the job failed.
	Error string
}

// engine implements the Engine interface.
type engine struct {
	stateManager *state.Manager
	gitManager   *git.Manager
	logger       logging.Logger
	config       *Config
	cliCaller    callcli.AICliCaller
}

// NewEngine creates a new execution engine with the given dependencies.
func NewEngine(
	stateManager *state.Manager,
	gitManager *git.Manager,
	logger logging.Logger,
	config *Config,
	cliCaller callcli.AICliCaller,
) Engine {
	if config == nil {
		config = DefaultConfig()
	}
	return &engine{
		stateManager: stateManager,
		gitManager:   gitManager,
		logger:       logger,
		config:       config,
		cliCaller:    cliCaller,
	}
}

// ExecuteJob executes a job within a module.
// It performs the following steps:
// 1. Check prerequisites
// 2. Transition state from PENDING to RUNNING
// 3. Execute tasks
// 4. Handle failures with retry logic
// 5. Transition to COMPLETED or FAILED
// 6. Create Git commit if configured
func (e *engine) ExecuteJob(ctx context.Context, module, job string) error {
	e.logger.Info("Starting job execution",
		logging.String("module", module),
		logging.String("job", job),
	)

	// Step 1: Check prerequisites (V2: skipped - order guaranteed by topological sort)
	// In V2 format, modules and jobs are already sorted topologically at generation time,
	// so we don't need runtime prerequisite checking.
	// if err := e.checkPrerequisites(ctx, module, job); err != nil {
	// 	e.logger.Error("Prerequisite check failed",
	// 		logging.String("module", module),
	// 		logging.String("job", job),
	// 		logging.String("error", err.Error()),
	// 	)
	// 	return fmt.Errorf("prerequisite check failed: %w", err)
	// }

	// Get job state to check current status and retry count
	jobState, err := e.getJobState(module, job)
	if err != nil {
		return err
	}

	// Check if we've exceeded max retries for failed jobs
	if jobState.Status == state.StatusFailed {
		if jobState.RetryCount >= e.config.MaxRetries {
			return fmt.Errorf("max retries exceeded (%d)", e.config.MaxRetries)
		}
		// For retry: first transition from FAILED to PENDING
		e.logger.Info("Retrying failed job",
			logging.String("module", module),
			logging.String("job", job),
			logging.Int("retry_count", jobState.RetryCount),
		)
		if err := e.transitionState(module, job, state.StatusPending); err != nil {
			return fmt.Errorf("failed to transition from FAILED to PENDING for retry: %w", err)
		}
	}

	// Step 2: Transition to RUNNING
	if err := e.transitionState(module, job, state.StatusRunning); err != nil {
		return fmt.Errorf("failed to transition to RUNNING: %w", err)
	}

	// Set as current job
	if err := e.stateManager.SetCurrent(module, job, state.StatusRunning); err != nil {
		e.logger.Warn("Failed to set current job", logging.String("error", err.Error()))
	}

	// Step 3: Execute tasks
	tasksCompleted, err := e.executeTasks(ctx, module, job)

	// Step 4 & 5: Handle result and state transition
	if err != nil {
		e.logger.Error("Job execution failed",
			logging.String("module", module),
			logging.String("job", job),
			logging.String("error", err.Error()),
		)

		// Transition to FAILED
		if transErr := e.transitionState(module, job, state.StatusFailed); transErr != nil {
			e.logger.Error("Failed to transition to FAILED", logging.String("error", transErr.Error()))
		}

		// Update failure reason
		if updateErr := e.updateFailureReason(module, job, err.Error()); updateErr != nil {
			e.logger.Warn("Failed to update failure reason", logging.String("error", updateErr.Error()))
		}

		return fmt.Errorf("job execution failed: %w", err)
	}

	// Step 5: Verify completion marking in plan file before transitioning to COMPLETED
	completionVerified, err := e.verifyJobCompletionInPlan(module, job)
	if err != nil {
		e.logger.Warn("Failed to verify job completion in plan file",
			logging.String("error", err.Error()),
		)
		// Continue anyway - verification is best-effort
	}

	if !completionVerified {
		e.logger.Warn("Job completion not marked in plan file",
			logging.String("module", module),
			logging.String("job", job),
		)
		// For now, we'll proceed anyway, but log the warning
		// In stricter mode, we could return an error here
	}

	// Step 5: Transition to COMPLETED
	if err := e.transitionState(module, job, state.StatusCompleted); err != nil {
		return fmt.Errorf("failed to transition to COMPLETED: %w", err)
	}

	// Clear current job
	if err := e.stateManager.ClearCurrent(); err != nil {
		e.logger.Warn("Failed to clear current job", logging.String("error", err.Error()))
	}

	e.logger.Info("Job completed successfully",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("tasks_completed", tasksCompleted),
	)

	// Step 6: Create Git commit
	if e.config.AutoCommit {
		if err := e.createGitCommit(module, job); err != nil {
			// Log warning but don't fail the job
			e.logger.Warn("Failed to create Git commit",
				logging.String("error", err.Error()),
			)
		}
	}

	return nil
}

// ExecuteTask executes a single task.
// This is a placeholder implementation that would be extended by TaskRunner.
func (e *engine) ExecuteTask(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
	e.logger.Info("Executing task",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("task_index", taskIndex),
		logging.String("task_desc", taskDesc),
	)

	// Build the prompt for this task
	prompt, err := e.buildTaskPrompt(module, job, taskIndex, taskDesc)
	if err != nil {
		return fmt.Errorf("failed to build task prompt: %w", err)
	}

	// Execute the task using AI CLI (doing mode - non-interactive)
	opts := callcli.Options{
		Timeout:    0, // No timeout for task execution
		Stdin:      prompt,
		WorkingDir: e.config.WorkingDir,
		Output: callcli.OutputConfig{
			Mode: callcli.OutputStream, // Stream output to terminal
		},
	}

	// Build args for non-interactive mode (doing)
	// Use bypassPermissions mode for automated execution
	baseArgs := e.cliCaller.BuildArgs()
	args := append([]string{"--permission-mode", "bypassPermissions", "-p"}, baseArgs...)

	// Execute the command
	result, err := e.cliCaller.GetBaseCaller().CallWithOptions(ctx, e.cliCaller.GetCLIPath(), args, opts)

	if err != nil {
		e.logger.Error("Task execution failed",
			logging.String("module", module),
			logging.String("job", job),
			logging.Int("task_index", taskIndex),
			logging.String("error", err.Error()),
		)
		return fmt.Errorf("task execution failed: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("task execution failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// Mark task as completed
	return e.markTaskCompleted(module, job, taskIndex)
}

// ResumeJob resumes a previously interrupted job.
func (e *engine) ResumeJob(ctx context.Context, module, job string) error {
	e.logger.Info("Resuming job",
		logging.String("module", module),
		logging.String("job", job),
	)

	// Get job state
	jobState, err := e.getJobState(module, job)
	if err != nil {
		return err
	}

	// Check if job can be resumed
	if jobState.Status != state.StatusRunning && jobState.Status != state.StatusFailed {
		return fmt.Errorf("job cannot be resumed: current status is %s", jobState.Status)
	}

	// Re-run the job
	return e.ExecuteJob(ctx, module, job)
}

// checkPrerequisites checks if all prerequisites for a job are met.
func (e *engine) checkPrerequisites(ctx context.Context, module, job string) error {
	// Get job state
	jobState, err := e.getJobState(module, job)
	if err != nil {
		return err
	}

	// Check if job is in a valid state to start
	switch jobState.Status {
	case state.StatusPending, state.StatusFailed:
		// Can start from these states
	case state.StatusCompleted:
		return fmt.Errorf("job already completed")
	case state.StatusRunning:
		return fmt.Errorf("job already running")
	case state.StatusBlocked:
		return fmt.Errorf("job is blocked by dependencies")
	default:
		return fmt.Errorf("invalid job status: %s", jobState.Status)
	}

	// Check if module exists
	if _, err := e.stateManager.GetJobStatus(module, job); err != nil {
		return fmt.Errorf("module or job not found: %w", err)
	}

	e.logger.Debug("Prerequisites check passed",
		logging.String("module", module),
		logging.String("job", job),
	)

	return nil
}

// transitionState transitions a job to a new state with validation.
func (e *engine) transitionState(module, job string, toStatus state.Status) error {
	// Use the state manager's TransitionJobStatus which validates the transition
	if err := e.stateManager.TransitionJobStatus(module, job, toStatus, e.logger); err != nil {
		return err
	}

	e.logger.Info("State transition successful",
		logging.String("module", module),
		logging.String("job", job),
		logging.String("new_status", string(toStatus)),
	)

	return nil
}

// executeTasks executes all tasks for a job in a single AI CLI call.
// Returns the number of tasks completed.
// This method creates one comprehensive prompt for the entire job and lets
// the AI CLI handle all tasks autonomously.
func (e *engine) executeTasks(ctx context.Context, module, job string) (int, error) {
	jobState, err := e.getJobState(module, job)
	if err != nil {
		return 0, err
	}

	tasksTotal := len(jobState.Tasks)

	e.logger.Info("Executing job with all tasks",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("tasks_total", tasksTotal),
	)

	// Create job-specific log file
	logFilePath, logFile, err := e.createJobLogFile(module, job)
	if err != nil {
		e.logger.Warn("Failed to create job log file",
			logging.String("error", err.Error()),
		)
		// Continue without log file - not critical
		logFile = nil
	} else {
		defer logFile.Close()
		e.logger.Info("Job log file created",
			logging.String("log_file", logFilePath),
		)
	}

	// Build comprehensive job-level prompt
	prompt, err := e.buildJobPrompt(module, job)
	if err != nil {
		return 0, fmt.Errorf("failed to build job prompt: %w", err)
	}

	// Write prompt to log file for debugging
	if logFile != nil {
		promptHeader := "========================================\n"
		promptHeader += "MORTY JOB PROMPT (for debugging)\n"
		promptHeader += "========================================\n\n"
		if _, err := logFile.WriteString(promptHeader + prompt + "\n\n"); err != nil {
			e.logger.Warn("Failed to write prompt to log file",
				logging.String("error", err.Error()),
			)
		}
		promptFooter := "========================================\n"
		promptFooter += "END OF PROMPT - CLI OUTPUT BELOW\n"
		promptFooter += "========================================\n\n"
		if _, err := logFile.WriteString(promptFooter); err != nil {
			e.logger.Warn("Failed to write prompt footer to log file",
				logging.String("error", err.Error()),
			)
		}
	}

	// Execute the entire job using AI CLI with log file capture
	opts := callcli.Options{
		Timeout:    0, // No timeout for job execution
		Stdin:      prompt,
		WorkingDir: e.config.WorkingDir,
		Output: callcli.OutputConfig{
			Mode: callcli.OutputCapture, // Capture output to memory (don't pollute console)
		},
	}

	// If we have a log file, also write to it
	if logFile != nil {
		opts.Output.OutputFile = logFilePath
	}

	// Build args for non-interactive mode (doing)
	// Use bypassPermissions mode for automated execution
	baseArgs := e.cliCaller.BuildArgs()
	args := append([]string{"--permission-mode", "bypassPermissions", "-p"}, baseArgs...)

	// Execute the command
	result, err := e.cliCaller.GetBaseCaller().CallWithOptions(ctx, e.cliCaller.GetCLIPath(), args, opts)

	// Write captured output to log file
	if logFile != nil && result != nil {
		e.writeJobLog(logFile, module, job, result.Stdout, result.Stderr, result.ExitCode)
	}

	if err != nil {
		e.logger.Error("Job execution failed",
			logging.String("module", module),
			logging.String("job", job),
			logging.String("error", err.Error()),
		)
		return 0, fmt.Errorf("job execution failed: %w", err)
	}

	if result.ExitCode != 0 {
		return 0, fmt.Errorf("job execution failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// After successful execution, mark all tasks as completed
	// The AI is expected to handle all tasks, so we mark them all as done
	for i := range jobState.Tasks {
		if err := e.markTaskCompleted(module, job, i); err != nil {
			e.logger.Warn("Failed to mark task as completed",
				logging.Int("task_index", i),
				logging.String("error", err.Error()),
			)
		}
	}

	// Update tasks completed count
	if err := e.updateTasksCompleted(module, job, tasksTotal); err != nil {
		e.logger.Warn("Failed to update tasks completed count",
			logging.String("error", err.Error()),
		)
	}

	e.logger.Success("Job execution completed",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("tasks_completed", tasksTotal),
	)

	return tasksTotal, nil
}

// markTaskCompleted marks a single task as completed.
func (e *engine) markTaskCompleted(module, job string, taskIndex int) error {
	e.logger.Debug("Task marked as completed",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("task_index", taskIndex),
	)

	// Update the task status in state manager
	return e.stateManager.UpdateTaskStatusByName(module, job, taskIndex, state.StatusCompleted)
}

// updateTasksCompleted updates the count of completed tasks for a job.
func (e *engine) updateTasksCompleted(module, job string, count int) error {
	e.logger.Debug("Tasks completed updated",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("completed", count),
	)

	// Update tasks_completed count in state
	return e.stateManager.UpdateTasksCompleted(module, job, count)
}

// updateFailureReason updates the failure reason for a job.
func (e *engine) updateFailureReason(module, job, reason string) error {
	// Similar to updateTasksCompleted, this would need state access
	e.logger.Debug("Failure reason updated",
		logging.String("module", module),
		logging.String("job", job),
		logging.String("reason", reason),
	)
	return nil
}

// getJobState retrieves the job state from the state manager.
func (e *engine) getJobState(module, job string) (*state.JobState, error) {
	// Load state to get job details
	if err := e.stateManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Get the full job state using GetJob
	jobState := e.stateManager.GetJob(module, job)
	if jobState == nil {
		return nil, fmt.Errorf("job not found: %s in module %s", job, module)
	}

	return jobState, nil
}

// createGitCommit creates a Git commit after job completion.
// It uses the git.Manager's CreateLoopCommit method which handles staging and committing.
func (e *engine) createGitCommit(module, job string) error {
	return e.createGitCommitUsingCommitter(module, job)
}

// createGitCommitUsingCommitter creates a Git commit using the Committer interface.
func (e *engine) createGitCommitUsingCommitter(module, job string) error {
	if e.gitManager == nil {
		return fmt.Errorf("git manager not available")
	}

	workDir := e.config.WorkingDir
	if workDir == "" {
		workDir = "."
	}

	absPath, err := filepath.Abs(workDir)
	if err != nil {
		absPath = workDir
	}

	// Check if there are changes to commit
	hasChanges, err := e.gitManager.HasUncommittedChanges(absPath)
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	if !hasChanges {
		e.logger.Info("No changes to commit")
		return nil
	}

	// Get current loop number
	loopNum, err := e.gitManager.GetCurrentLoopNumber(absPath)
	if err != nil {
		loopNum = 1
	}

	// Use CreateLoopCommit with job-specific status
	status := fmt.Sprintf("%s/%s - COMPLETED", module, job)
	_, err = e.gitManager.CreateLoopCommit(loopNum, status, absPath)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	e.logger.Info("Git commit created",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("loop", loopNum),
	)

	return nil
}

// buildJobPrompt builds a comprehensive prompt for executing an entire job.
// This includes all tasks, context, and instructions for the AI to handle autonomously.
func (e *engine) buildJobPrompt(module, job string) (string, error) {
	// Load the doing prompt template
	doingPromptPath := filepath.Join(e.config.PromptsDir, "doing.md")
	promptTemplate, err := os.ReadFile(doingPromptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read doing prompt: %w", err)
	}

	// Get the plan file name from module state
	planFileName := module + ".md"

	// Get status and find module
	if execStatus := e.stateManager.GetStatus(); execStatus != nil {
		// Find module by name
		for _, mod := range execStatus.Modules {
			if mod.Name == module && mod.PlanFile != "" {
				planFileName = mod.PlanFile
				break
			}
		}
	}

	// Load the plan file for context
	planFilePath := filepath.Join(e.config.PlanDir, planFileName)
	planContent, err := os.ReadFile(planFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read plan file: %w", err)
	}

	// Get job state for detailed information
	jobState, err := e.getJobState(module, job)
	if err != nil {
		return "", fmt.Errorf("failed to get job state: %w", err)
	}

	// Build task list
	taskList := ""
	for i, task := range jobState.Tasks {
		status := "[ ]"
		if task.Status == state.StatusCompleted {
			status = "[x]"
		}
		taskList += fmt.Sprintf("- %s Task %d: %s\n", status, i+1, task.Description)
	}

	// Build the comprehensive job-level prompt
	prompt := fmt.Sprintf(`%s

# Current Job

**Module**: %s
**Job**: %s

## Job Tasks

%s

## Job Details

**Tasks Total**: %d
**Tasks Completed**: %d

# Plan Context

%s

# Job-Level Execution Instructions

You are executing the entire job "%s" in module "%s". This is a job-level execution where you should:

1. Review all tasks listed above
2. Execute each task in sequence
3. Skip tasks that are already marked as completed [x]
4. Follow the doing prompt template for task execution
5. Ensure all validation criteria are met before completing
6. Update the plan file with any issues encountered in the debug logs section
7. Mark the job as complete when all tasks are done and validated

Execute the job autonomously and handle all tasks. Report any issues or blockers encountered.
`, string(promptTemplate), module, job, taskList, len(jobState.Tasks), jobState.TasksCompleted, string(planContent), job, module)

	return prompt, nil
}

// buildTaskPrompt builds the prompt for executing a single task (legacy method, kept for compatibility).
func (e *engine) buildTaskPrompt(module, job string, taskIndex int, taskDesc string) (string, error) {
	// Load the doing prompt template
	doingPromptPath := filepath.Join(e.config.PromptsDir, "doing.md")
	promptTemplate, err := os.ReadFile(doingPromptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read doing prompt: %w", err)
	}

	// Get the plan file name from module state
	planFileName := module + ".md"
	if execStatus := e.stateManager.GetState(); execStatus != nil {
		// Find module by name
		for _, mod := range execStatus.Modules {
			if mod.Name == module && mod.PlanFile != "" {
				planFileName = mod.PlanFile
				break
			}
		}
	}

	// Load the plan file for context
	planFilePath := filepath.Join(e.config.PlanDir, planFileName)
	planContent, err := os.ReadFile(planFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read plan file: %w", err)
	}

	// Build the full prompt
	prompt := fmt.Sprintf(`%s

# Current Task

**Module**: %s
**Job**: %s
**Task %d**: %s

# Plan Context

%s

# Instructions

Please execute the task described above. Follow the doing prompt template and ensure all validation criteria are met.
`, string(promptTemplate), module, job, taskIndex+1, taskDesc, string(planContent))

	return prompt, nil
}

// createJobLogFile creates a log file for a job.
// Returns the file path and an open file handle.
// The log file path format is: .morty/logs/{module}_{job}_{timestamp}.log
func (e *engine) createJobLogFile(module, job string) (string, *os.File, error) {
	// Create logs directory if it doesn't exist
	logsDir := ".morty/logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Generate timestamp for unique filename
	timestamp := time.Now().Format("20060102_150405")

	// Build log file path
	logFileName := fmt.Sprintf("%s_%s_%s.log", module, job, timestamp)
	logFilePath := filepath.Join(logsDir, logFileName)

	// Create log file
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Write header
	fmt.Fprintf(logFile, "=== Job Log: %s/%s ===\n", module, job)
	fmt.Fprintf(logFile, "Started: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(logFile, "==========================================\n\n")

	return logFilePath, logFile, nil
}

// writeJobLog writes execution results to the job log file.
func (e *engine) writeJobLog(logFile *os.File, module, job, stdout, stderr string, exitCode int) {
	if logFile == nil {
		return
	}

	// Parse and format stdout as event stream if it looks like JSON
	if stdout != "" && strings.HasPrefix(strings.TrimSpace(stdout), "[") {
		// Looks like JSON event stream, format it
		formatter := NewEventFormatter(logFile)
		if err := formatter.FormatEventStream(stdout); err != nil {
			// If formatting fails, fall back to raw output
			e.logger.Warn("Failed to format event stream, using raw output",
				logging.String("error", err.Error()),
			)
			fmt.Fprintf(logFile, "\n=== STDOUT (Raw) ===\n")
			fmt.Fprintf(logFile, "%s\n", stdout)
		}
	} else if stdout != "" {
		// Not JSON, write as-is
		fmt.Fprintf(logFile, "\n=== STDOUT ===\n")
		fmt.Fprintf(logFile, "%s\n", stdout)
	}

	// Write stderr section
	if stderr != "" {
		fmt.Fprintf(logFile, "\n=== STDERR ===\n")
		fmt.Fprintf(logFile, "%s\n", stderr)
	}

	// Write footer
	fmt.Fprintf(logFile, "\n==========================================\n")
	fmt.Fprintf(logFile, "Exit Code: %d\n", exitCode)
	fmt.Fprintf(logFile, "Completed: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}

// verifyJobCompletionInPlan verifies that the job is marked as completed in the plan file.
// Returns true if the job has a completion status marker in the plan file.
func (e *engine) verifyJobCompletionInPlan(module, job string) (bool, error) {
	// Get the plan file name
	planFileName := module + ".md"

	// Get status and find module
	if execStatus := e.stateManager.GetState(); execStatus != nil {
		// Find module by name
		for _, mod := range execStatus.Modules {
			if mod.Name == module && mod.PlanFile != "" {
				planFileName = mod.PlanFile
				break
			}
		}
	}

	// Load and parse the plan file
	planFilePath := filepath.Join(e.config.PlanDir, planFileName)
	planContent, err := os.ReadFile(planFilePath)
	if err != nil {
		return false, fmt.Errorf("failed to read plan file: %w", err)
	}

	// Parse the plan
	planData, err := plan.ParsePlan(string(planContent))
	if err != nil {
		return false, fmt.Errorf("failed to parse plan file: %w", err)
	}

	// Find the specific job
	for _, jobData := range planData.Jobs {
		if jobData.Name == job {
			// Check if the job has a completion marker
			if jobData.IsCompleted {
				e.logger.Debug("Job completion verified in plan file",
					logging.String("module", module),
					logging.String("job", job),
					logging.String("completion_status", jobData.CompletionStatus),
				)
				return true, nil
			}
			// Job found but not marked as completed
			return false, nil
		}
	}

	// Job not found in plan
	return false, fmt.Errorf("job %s not found in plan file", job)
}

// Ensure engine implements Engine interface
var _ Engine = (*engine)(nil)
