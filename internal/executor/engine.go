// Package executor provides job execution engine for Morty.
// It handles the lifecycle of job execution including state transitions,
// retry logic, prerequisite checking, and Git integration.
package executor

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/morty/morty/internal/git"
	"github.com/morty/morty/internal/logging"
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
}

// NewEngine creates a new execution engine with the given dependencies.
func NewEngine(
	stateManager *state.Manager,
	gitManager *git.Manager,
	logger logging.Logger,
	config *Config,
) Engine {
	if config == nil {
		config = DefaultConfig()
	}
	return &engine{
		stateManager: stateManager,
		gitManager:   gitManager,
		logger:       logger,
		config:       config,
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

	// Step 1: Check prerequisites
	if err := e.checkPrerequisites(ctx, module, job); err != nil {
		e.logger.Error("Prerequisite check failed",
			logging.String("module", module),
			logging.String("job", job),
			logging.String("error", err.Error()),
		)
		return fmt.Errorf("prerequisite check failed: %w", err)
	}

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

	// Step 5: Transition to COMPLETED
	if err := e.transitionState(module, job, state.StatusCompleted); err != nil {
		return fmt.Errorf("failed to transition to COMPLETED: %w", err)
	}

	// Clear current job
	if err := e.stateManager.SetCurrent("", "", state.StatusPending); err != nil {
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

	// This is a basic implementation that marks the task as completed.
	// In a full implementation, this would call the AI CLI through the TaskRunner.
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

// executeTasks executes all pending tasks for a job.
// Returns the number of tasks completed.
func (e *engine) executeTasks(ctx context.Context, module, job string) (int, error) {
	jobState, err := e.getJobState(module, job)
	if err != nil {
		return 0, err
	}

	completed := 0
	tasksTotal := len(jobState.Tasks)

	for i, task := range jobState.Tasks {
		// Skip already completed tasks
		if task.Status == state.StatusCompleted {
			completed++
			continue
		}

		// Execute the task
		if err := e.ExecuteTask(ctx, module, job, i, task.Description); err != nil {
			return completed, fmt.Errorf("task %d failed: %w", i, err)
		}

		completed++

		// Update job's tasks completed count
		if err := e.updateTasksCompleted(module, job, completed); err != nil {
			e.logger.Warn("Failed to update tasks completed count",
				logging.String("error", err.Error()),
			)
		}
	}

	// Verify all tasks completed
	if completed < tasksTotal {
		return completed, fmt.Errorf("only %d of %d tasks completed", completed, tasksTotal)
	}

	return completed, nil
}

// markTaskCompleted marks a single task as completed.
func (e *engine) markTaskCompleted(module, job string, taskIndex int) error {
	// This would typically be implemented by accessing the state directly
	// For now, we use the UpdateJobStatus which saves the state
	// In a full implementation, this would update individual task status
	e.logger.Debug("Task marked as completed",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("task_index", taskIndex),
	)
	return nil
}

// updateTasksCompleted updates the count of completed tasks for a job.
func (e *engine) updateTasksCompleted(module, job string, count int) error {
	// Note: In the current state.Manager, we don't have a direct method
	// to update TasksCompleted. This would need to be added or we access
	// the state directly. For now, we log the progress.
	e.logger.Debug("Tasks completed updated",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("completed", count),
	)
	return nil
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

// Ensure engine implements Engine interface
var _ Engine = (*engine)(nil)
