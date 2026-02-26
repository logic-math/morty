// Package executor provides job execution engine for Morty.
package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/state"
)

// TaskExecutor is a function type for executing a single task.
// It receives the context, module, job, task index, and task description.
// It should return an error if the task fails.
type TaskExecutor func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error

// JobRunner handles the execution of a job's tasks.
// It manages the task loop, state updates, and error handling.
type JobRunner struct {
	stateManager *state.Manager
	logger       logging.Logger
	taskExecutor TaskExecutor
}

// NewJobRunner creates a new JobRunner with the given dependencies.
//
// Parameters:
//   - stateManager: The state manager for persisting job and task state
//   - logger: The logger for recording execution progress
//   - taskExecutor: Optional custom task executor. If nil, a default no-op executor is used.
//
// Returns:
//   - A pointer to a new JobRunner instance
func NewJobRunner(
	stateManager *state.Manager,
	logger logging.Logger,
	taskExecutor TaskExecutor,
) *JobRunner {
	if taskExecutor == nil {
		// Default no-op task executor that just marks tasks as completed
		taskExecutor = func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
			return nil
		}
	}

	return &JobRunner{
		stateManager: stateManager,
		logger:       logger,
		taskExecutor: taskExecutor,
	}
}

// Run executes all pending tasks for a job.
// It handles the complete task execution lifecycle:
// 1. Load job state
// 2. Iterate through tasks, skipping completed ones
// 3. Execute each pending task
// 4. Update task status after completion
// 5. Update job's tasks_completed count
// 6. Handle task failures with error reporting
//
// Parameters:
//   - ctx: The context for cancellation
//   - module: The module name
//   - job: The job name
//
// Returns:
//   - The number of tasks completed
//   - An error if any task fails
func (jr *JobRunner) Run(ctx context.Context, module, job string) (int, error) {
	jr.logger.Info("Starting job runner",
		logging.String("module", module),
		logging.String("job", job),
	)

	// Load job state
	jobState, err := jr.getJobState(module, job)
	if err != nil {
		return 0, fmt.Errorf("failed to get job state: %w", err)
	}

	// Validate job is in a runnable state
	if jobState.Status != state.StatusRunning && jobState.Status != state.StatusPending {
		return 0, fmt.Errorf("job cannot be executed: current status is %s", jobState.Status)
	}

	completed := 0
	tasksTotal := len(jobState.Tasks)

	jr.logger.Info("Executing tasks",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("total_tasks", tasksTotal),
	)

	for i, task := range jobState.Tasks {
		// Check for context cancellation
		if err := ctx.Err(); err != nil {
			jr.logger.Warn("Context cancelled, stopping task execution",
				logging.String("module", module),
				logging.String("job", job),
				logging.Int("task_index", i),
			)
			return completed, fmt.Errorf("context cancelled: %w", err)
		}

		// Task 7: Skip already completed tasks
		if task.Status == state.StatusCompleted {
			jr.logger.Debug("Skipping completed task",
				logging.String("module", module),
				logging.String("job", job),
				logging.Int("task_index", i),
			)
			completed++
			continue
		}

		// Log task execution start
		jr.logger.Info("Executing task",
			logging.String("module", module),
			logging.String("job", job),
			logging.Int("task_index", i),
			logging.String("task_desc", task.Description),
		)

		// Execute the task using the configured task executor
		if err := jr.taskExecutor(ctx, module, job, i, task.Description); err != nil {
			jr.logger.Error("Task execution failed",
				logging.String("module", module),
				logging.String("job", job),
				logging.Int("task_index", i),
				logging.String("error", err.Error()),
			)

			// Task 6: Job-level error handling - update failure reason
			if updateErr := jr.updateFailureReason(module, job, fmt.Sprintf("task %d failed: %v", i, err)); updateErr != nil {
				jr.logger.Warn("Failed to update failure reason",
					logging.String("error", updateErr.Error()),
				)
			}

			return completed, fmt.Errorf("task %d failed: %w", i, err)
		}

		// Task 5: Mark task as completed and update state
		if err := jr.markTaskCompleted(module, job, i); err != nil {
			jr.logger.Warn("Failed to mark task as completed",
				logging.String("module", module),
				logging.String("job", job),
				logging.Int("task_index", i),
				logging.String("error", err.Error()),
			)
			// Continue even if state update fails - task was executed successfully
		}

		completed++

		// Update job's tasks_completed count
		if err := jr.updateTasksCompleted(module, job, completed); err != nil {
			jr.logger.Warn("Failed to update tasks completed count",
				logging.String("error", err.Error()),
			)
		}

		jr.logger.Success("Task completed",
			logging.String("module", module),
			logging.String("job", job),
			logging.Int("task_index", i),
		)
	}

	// Verify all tasks completed
	if completed < tasksTotal {
		jr.logger.Warn("Not all tasks completed",
			logging.String("module", module),
			logging.String("job", job),
			logging.Int("completed", completed),
			logging.Int("total", tasksTotal),
		)
		return completed, fmt.Errorf("only %d of %d tasks completed", completed, tasksTotal)
	}

	jr.logger.Success("All tasks completed",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("tasks_completed", completed),
	)

	return completed, nil
}

// getJobState retrieves the job state from the state manager.
func (jr *JobRunner) getJobState(module, job string) (*state.JobState, error) {
	// Load state to get job details
	if err := jr.stateManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Get the full job state using GetJob
	jobState := jr.stateManager.GetJob(module, job)
	if jobState == nil {
		return nil, fmt.Errorf("job not found: %s in module %s", job, module)
	}

	return jobState, nil
}

// markTaskCompleted marks a single task as completed.
func (jr *JobRunner) markTaskCompleted(module, job string, taskIndex int) error {
	jr.stateManager.Load()
	statePtr := jr.stateManager.GetState()
	if statePtr == nil {
		return fmt.Errorf("state not loaded")
	}

	moduleState, ok := statePtr.Modules[module]
	if !ok {
		return fmt.Errorf("module not found: %s", module)
	}

	jobState, ok := moduleState.Jobs[job]
	if !ok {
		return fmt.Errorf("job not found: %s in module %s", job, module)
	}

	// Update task status
	now := time.Now()
	if taskIndex >= 0 && taskIndex < len(jobState.Tasks) {
		jobState.Tasks[taskIndex].Status = state.StatusCompleted
		jobState.Tasks[taskIndex].UpdatedAt = now
	} else {
		return fmt.Errorf("invalid task index: %d", taskIndex)
	}

	// Save state
	if err := jr.stateManager.Save(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	jr.logger.Debug("Task marked as completed",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("task_index", taskIndex),
	)

	return nil
}

// updateTasksCompleted updates the count of completed tasks for a job.
func (jr *JobRunner) updateTasksCompleted(module, job string, count int) error {
	jr.stateManager.Load()
	statePtr := jr.stateManager.GetState()
	if statePtr == nil {
		return fmt.Errorf("state not loaded")
	}

	moduleState, ok := statePtr.Modules[module]
	if !ok {
		return fmt.Errorf("module not found: %s", module)
	}

	jobState, ok := moduleState.Jobs[job]
	if !ok {
		return fmt.Errorf("job not found: %s in module %s", job, module)
	}

	// Update tasks completed count
	now := time.Now()
	jobState.TasksCompleted = count
	jobState.UpdatedAt = now

	// Save state
	if err := jr.stateManager.Save(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// updateFailureReason updates the failure reason for a job.
func (jr *JobRunner) updateFailureReason(module, job, reason string) error {
	jr.stateManager.Load()
	statePtr := jr.stateManager.GetState()
	if statePtr == nil {
		return fmt.Errorf("state not loaded")
	}

	moduleState, ok := statePtr.Modules[module]
	if !ok {
		return fmt.Errorf("module not found: %s", module)
	}

	jobState, ok := moduleState.Jobs[job]
	if !ok {
		return fmt.Errorf("job not found: %s in module %s", job, module)
	}

	// Update failure reason
	now := time.Now()
	jobState.FailureReason = reason
	jobState.UpdatedAt = now

	// Save state
	if err := jr.stateManager.Save(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// SetTaskExecutor updates the task executor function.
// This allows customizing task execution behavior after creation.
func (jr *JobRunner) SetTaskExecutor(executor TaskExecutor) {
	if executor != nil {
		jr.taskExecutor = executor
	}
}

// GetTaskExecutor returns the current task executor function.
func (jr *JobRunner) GetTaskExecutor() TaskExecutor {
	return jr.taskExecutor
}
