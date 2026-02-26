// Package executor provides job execution engine for Morty.
package executor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/state"
)

// setupJobRunnerTestEnv creates a test environment specifically for JobRunner tests.
func setupJobRunnerTestEnv(t testing.TB) (string, *state.Manager, logging.Logger, func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "jobrunner-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create state file
	stateFile := filepath.Join(tempDir, "status.json")
	stateManager := state.NewManager(stateFile)

	// Save initial state with test data
	if err := os.WriteFile(stateFile, []byte(`{
		"version": "1.0",
		"global": {
			"status": "PENDING",
			"start_time": "2024-01-01T00:00:00Z",
			"last_update": "2024-01-01T00:00:00Z"
		},
		"modules": {
			"test-module": {
				"name": "test-module",
				"status": "PENDING",
				"jobs": {
					"pending-job": {
						"name": "pending-job",
						"status": "PENDING",
						"tasks_total": 3,
						"tasks_completed": 0,
						"tasks": [
							{"index": 0, "status": "PENDING", "description": "Task 1"},
							{"index": 1, "status": "PENDING", "description": "Task 2"},
							{"index": 2, "status": "PENDING", "description": "Task 3"}
						],
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					},
					"partial-job": {
						"name": "partial-job",
						"status": "RUNNING",
						"tasks_total": 3,
						"tasks_completed": 1,
						"tasks": [
							{"index": 0, "status": "COMPLETED", "description": "Task 1"},
							{"index": 1, "status": "PENDING", "description": "Task 2"},
							{"index": 2, "status": "PENDING", "description": "Task 3"}
						],
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					},
					"completed-job": {
						"name": "completed-job",
						"status": "COMPLETED",
						"tasks_total": 2,
						"tasks_completed": 2,
						"tasks": [
							{"index": 0, "status": "COMPLETED", "description": "Task 1"},
							{"index": 1, "status": "COMPLETED", "description": "Task 2"}
						],
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					},
					"failed-job": {
						"name": "failed-job",
						"status": "FAILED",
						"tasks_total": 2,
						"tasks_completed": 0,
						"failure_reason": "",
						"tasks": [
							{"index": 0, "status": "PENDING", "description": "Task 1"},
							{"index": 1, "status": "PENDING", "description": "Task 2"}
						],
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					},
					"all-completed-tasks": {
						"name": "all-completed-tasks",
						"status": "RUNNING",
						"tasks_total": 3,
						"tasks_completed": 3,
						"tasks": [
							{"index": 0, "status": "COMPLETED", "description": "Task 1"},
							{"index": 1, "status": "COMPLETED", "description": "Task 2"},
							{"index": 2, "status": "COMPLETED", "description": "Task 3"}
						],
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					},
					"empty-tasks": {
						"name": "empty-tasks",
						"status": "PENDING",
						"tasks_total": 0,
						"tasks_completed": 0,
						"tasks": [],
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					}
				},
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z"
			}
		}
	}`), 0644); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Create logger
	logger := &mockLogger{}

	// Cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	// Load state
	if err := stateManager.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	return tempDir, stateManager, logger, cleanup
}

func TestNewJobRunner(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	tests := []struct {
		name         string
		taskExecutor TaskExecutor
		wantNil      bool
	}{
		{
			name:         "with nil task executor",
			taskExecutor: nil,
			wantNil:      false,
		},
		{
			name: "with custom task executor",
			taskExecutor: func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
				return nil
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jr := NewJobRunner(stateManager, logger, tt.taskExecutor)
			if jr == nil && !tt.wantNil {
				t.Fatal("NewJobRunner returned nil")
			}
			if jr != nil {
				if jr.stateManager != stateManager {
					t.Error("JobRunner.stateManager mismatch")
				}
				if jr.logger != logger {
					t.Error("JobRunner.logger mismatch")
				}
				if jr.taskExecutor == nil {
					t.Error("JobRunner.taskExecutor should not be nil")
				}
			}
		})
	}
}

func TestJobRunner_Run_Success(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	// Track executed tasks
	executedTasks := []int{}

	// Create custom task executor that records which tasks were executed
	taskExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		executedTasks = append(executedTasks, taskIndex)
		return nil
	}

	jr := NewJobRunner(stateManager, logger, taskExecutor)

	ctx := context.Background()
	completed, err := jr.Run(ctx, "test-module", "pending-job")

	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
	if completed != 3 {
		t.Errorf("Run() completed = %d, want 3", completed)
	}

	// Verify all tasks were executed
	if len(executedTasks) != 3 {
		t.Errorf("Expected 3 tasks to be executed, got %d", len(executedTasks))
	}
	for i := 0; i < 3; i++ {
		found := false
		for _, taskIdx := range executedTasks {
			if taskIdx == i {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Task %d was not executed", i)
		}
	}

	// Verify state was updated - reload and check
	stateManager.Load()
	jobState := stateManager.GetJob("test-module", "pending-job")
	if jobState == nil {
		t.Fatal("Job state is nil after reload")
	}
	if jobState.TasksCompleted != 3 {
		t.Errorf("TasksCompleted = %d, want 3", jobState.TasksCompleted)
	}
	for i, task := range jobState.Tasks {
		if task.Status != state.StatusCompleted {
			t.Errorf("Task %d status = %s, want COMPLETED", i, task.Status)
		}
	}
}

func TestJobRunner_Run_SkipCompletedTasks(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	// Track executed tasks
	executedTasks := []int{}

	taskExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		executedTasks = append(executedTasks, taskIndex)
		return nil
	}

	jr := NewJobRunner(stateManager, logger, taskExecutor)

	ctx := context.Background()
	completed, err := jr.Run(ctx, "test-module", "partial-job")

	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
	if completed != 3 {
		t.Errorf("Run() completed = %d, want 3", completed)
	}

	// Verify only pending tasks (1 and 2) were executed, not the already completed task 0
	if len(executedTasks) != 2 {
		t.Errorf("Expected 2 tasks to be executed (skipping completed), got %d", len(executedTasks))
	}
	for _, taskIdx := range executedTasks {
		if taskIdx == 0 {
			t.Error("Task 0 (already completed) should not have been executed")
		}
	}

	// Verify state
	stateManager.Load()
	jobState := stateManager.GetJob("test-module", "partial-job")
	if jobState.TasksCompleted != 3 {
		t.Errorf("TasksCompleted = %d, want 3", jobState.TasksCompleted)
	}
}

func TestJobRunner_Run_AllTasksAlreadyCompleted(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	executedTasks := []int{}

	taskExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		executedTasks = append(executedTasks, taskIndex)
		return nil
	}

	jr := NewJobRunner(stateManager, logger, taskExecutor)

	ctx := context.Background()
	completed, err := jr.Run(ctx, "test-module", "all-completed-tasks")

	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
	if completed != 3 {
		t.Errorf("Run() completed = %d, want 3", completed)
	}

	// Verify no tasks were executed (all were already completed)
	if len(executedTasks) != 0 {
		t.Errorf("Expected 0 tasks to be executed (all already completed), got %d", len(executedTasks))
	}
}

func TestJobRunner_Run_TaskFailure(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	expectedErr := errors.New("task execution failed")

	taskExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		if taskIndex == 1 {
			return expectedErr
		}
		return nil
	}

	jr := NewJobRunner(stateManager, logger, taskExecutor)

	ctx := context.Background()
	completed, err := jr.Run(ctx, "test-module", "pending-job")

	if err == nil {
		t.Error("Run() expected error, got nil")
	}
	if completed != 1 {
		t.Errorf("Run() completed = %d, want 1 (only first task should complete)", completed)
	}

	// Verify failure reason was updated
	stateManager.Load()
	jobState := stateManager.GetJob("test-module", "pending-job")
	if jobState.FailureReason == "" {
		t.Error("FailureReason should be set after task failure")
	}
}

func TestJobRunner_Run_InvalidJobStatus(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	jr := NewJobRunner(stateManager, logger, nil)

	ctx := context.Background()

	// Test with COMPLETED job
	_, err := jr.Run(ctx, "test-module", "completed-job")
	if err == nil {
		t.Error("Run() with completed job should return error")
	}

	// Test with FAILED job
	_, err = jr.Run(ctx, "test-module", "failed-job")
	if err == nil {
		t.Error("Run() with failed job should return error")
	}
}

func TestJobRunner_Run_NonExistentJob(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	jr := NewJobRunner(stateManager, logger, nil)

	ctx := context.Background()
	_, err := jr.Run(ctx, "test-module", "non-existent-job")

	if err == nil {
		t.Error("Run() with non-existent job should return error")
	}
}

func TestJobRunner_Run_NonExistentModule(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	jr := NewJobRunner(stateManager, logger, nil)

	ctx := context.Background()
	_, err := jr.Run(ctx, "non-existent-module", "pending-job")

	if err == nil {
		t.Error("Run() with non-existent module should return error")
	}
}

func TestJobRunner_Run_EmptyTasks(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	jr := NewJobRunner(stateManager, logger, nil)

	ctx := context.Background()
	completed, err := jr.Run(ctx, "test-module", "empty-tasks")

	if err != nil {
		t.Errorf("Run() error = %v, want nil for empty tasks", err)
	}
	if completed != 0 {
		t.Errorf("Run() completed = %d, want 0 for empty tasks", completed)
	}
}

func TestJobRunner_Run_ContextCancellation(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	taskExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		// Simulate some work
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	jr := NewJobRunner(stateManager, logger, taskExecutor)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := jr.Run(ctx, "test-module", "pending-job")

	if err == nil {
		t.Error("Run() with cancelled context should return error")
	}
}

func TestJobRunner_SetTaskExecutor(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	jr := NewJobRunner(stateManager, logger, nil)

	// Verify default executor exists
	if jr.GetTaskExecutor() == nil {
		t.Error("Default task executor should not be nil")
	}

	// Set custom executor
	customExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		return nil
	}

	jr.SetTaskExecutor(customExecutor)

	// Verify custom executor is set (just check it's not nil, can't compare funcs directly)
	if jr.GetTaskExecutor() == nil {
		t.Error("Custom task executor should be set")
	}

	// Execute to verify it doesn't panic with custom executor
	ctx := context.Background()
	jr.Run(ctx, "test-module", "empty-tasks")

	// Create a new runner with the custom executor directly to verify it works
	jr2 := NewJobRunner(stateManager, logger, customExecutor)
	if jr2 == nil {
		t.Error("NewJobRunner with custom executor should not be nil")
	}
}

func TestJobRunner_SetTaskExecutor_Nil(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	// Create with a valid executor
	originalExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		return nil
	}

	jr := NewJobRunner(stateManager, logger, originalExecutor)

	// Try to set nil - should not change the executor (can't compare funcs, but shouldn't panic)
	jr.SetTaskExecutor(nil)

	// Verify executor is still set (not nil)
	if jr.GetTaskExecutor() == nil {
		t.Error("SetTaskExecutor(nil) should not change the existing executor to nil")
	}
}

func TestJobRunner_Run_MultipleTasksWithFailure(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	// Track which task fails
	failAtTask := 2
	executedTasks := []int{}

	taskExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		executedTasks = append(executedTasks, taskIndex)
		if taskIndex == failAtTask {
			return errors.New("intentional failure")
		}
		return nil
	}

	jr := NewJobRunner(stateManager, logger, taskExecutor)

	ctx := context.Background()
	completed, err := jr.Run(ctx, "test-module", "pending-job")

	if err == nil {
		t.Error("Run() expected error when task fails")
	}

	// Should have completed tasks 0 and 1, then failed at task 2
	if completed != 2 {
		t.Errorf("Run() completed = %d, want 2", completed)
	}

	if len(executedTasks) != 3 {
		t.Errorf("Expected 3 task execution attempts, got %d", len(executedTasks))
	}

	// Verify state reflects partial completion
	stateManager.Load()
	jobState := stateManager.GetJob("test-module", "pending-job")
	if jobState.TasksCompleted != 2 {
		t.Errorf("TasksCompleted = %d, want 2", jobState.TasksCompleted)
	}

	// Tasks 0 and 1 should be COMPLETED, task 2 should still be PENDING
	if jobState.Tasks[0].Status != state.StatusCompleted {
		t.Errorf("Task 0 status = %s, want COMPLETED", jobState.Tasks[0].Status)
	}
	if jobState.Tasks[1].Status != state.StatusCompleted {
		t.Errorf("Task 1 status = %s, want COMPLETED", jobState.Tasks[1].Status)
	}
	if jobState.Tasks[2].Status != state.StatusPending {
		t.Errorf("Task 2 status = %s, want PENDING (failed task should not be marked completed)", jobState.Tasks[2].Status)
	}
}

func TestJobRunner_Run_UpdatesTaskState(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	taskExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		return nil
	}

	jr := NewJobRunner(stateManager, logger, taskExecutor)

	ctx := context.Background()
	jr.Run(ctx, "test-module", "pending-job")

	// Reload state and verify individual task states
	stateManager.Load()
	jobState := stateManager.GetJob("test-module", "pending-job")

	for i, task := range jobState.Tasks {
		if task.Status != state.StatusCompleted {
			t.Errorf("Task %d status = %s, want COMPLETED", i, task.Status)
		}
		if task.UpdatedAt.IsZero() {
			t.Errorf("Task %d UpdatedAt should not be zero", i)
		}
	}
}

// Benchmark tests
func BenchmarkJobRunner_Run(b *testing.B) {
	// Setup test environment directly (can't use setupJobRunnerTestEnv with *testing.B)
	tempDir, err := os.MkdirTemp("", "jobrunner-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stateFile := filepath.Join(tempDir, "status.json")
	stateManager := state.NewManager(stateFile)

	// Create initial state
	if err := os.WriteFile(stateFile, []byte(`{
		"version": "1.0",
		"global": {"status": "PENDING", "start_time": "2024-01-01T00:00:00Z", "last_update": "2024-01-01T00:00:00Z"},
		"modules": {
			"test-module": {
				"name": "test-module",
				"status": "PENDING",
				"jobs": {
					"pending-job": {
						"name": "pending-job",
						"status": "PENDING",
						"tasks_total": 3,
						"tasks_completed": 0,
						"tasks": [
							{"index": 0, "status": "PENDING", "description": "Task 1"},
							{"index": 1, "status": "PENDING", "description": "Task 2"},
							{"index": 2, "status": "PENDING", "description": "Task 3"}
						],
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					}
				},
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z"
			}
		}
	}`), 0644); err != nil {
		b.Fatalf("Failed to write state file: %v", err)
	}

	if err := stateManager.Load(); err != nil {
		b.Fatalf("Failed to load state: %v", err)
	}

	logger := &mockLogger{}

	taskExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		return nil
	}

	jr := NewJobRunner(stateManager, logger, taskExecutor)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset state for each iteration
		stateManager.Load()
		statePtr := stateManager.GetState()
		if statePtr != nil {
			// Reset task statuses to PENDING
			for _, task := range statePtr.Modules["test-module"].Jobs["pending-job"].Tasks {
				task.Status = state.StatusPending
			}
			statePtr.Modules["test-module"].Jobs["pending-job"].TasksCompleted = 0
			stateManager.Save()
		}

		jr.Run(ctx, "test-module", "pending-job")
	}
}

// Test for helper method error paths
func TestJobRunner_GetJobState_Errors(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	jr := NewJobRunner(stateManager, logger, nil)

	// Test non-existent module (via getJobState)
	_, err := jr.Run(context.Background(), "non-existent-module", "pending-job")
	if err == nil {
		t.Error("Expected error for non-existent module")
	}

	// Test non-existent job (via getJobState)
	_, err = jr.Run(context.Background(), "test-module", "non-existent-job")
	if err == nil {
		t.Error("Expected error for non-existent job")
	}
}

func TestJobRunner_HelperMethods_ErrorPaths(t *testing.T) {
	// Create a test environment
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	jr := NewJobRunner(stateManager, logger, nil)

	// Test updateTasksCompleted with non-existent module
	// This tests the error path at line 259
	err := jr.updateTasksCompleted("non-existent-module", "job", 1)
	if err == nil {
		t.Error("updateTasksCompleted should return error for non-existent module")
	}

	// Test updateTasksCompleted with non-existent job
	err = jr.updateTasksCompleted("test-module", "non-existent-job", 1)
	if err == nil {
		t.Error("updateTasksCompleted should return error for non-existent job")
	}

	// Test updateFailureReason with non-existent module
	err = jr.updateFailureReason("non-existent-module", "job", "error")
	if err == nil {
		t.Error("updateFailureReason should return error for non-existent module")
	}

	// Test updateFailureReason with non-existent job
	err = jr.updateFailureReason("test-module", "non-existent-job", "error")
	if err == nil {
		t.Error("updateFailureReason should return error for non-existent job")
	}

	// Test markTaskCompleted with non-existent module
	err = jr.markTaskCompleted("non-existent-module", "job", 0)
	if err == nil {
		t.Error("markTaskCompleted should return error for non-existent module")
	}

	// Test markTaskCompleted with non-existent job
	err = jr.markTaskCompleted("test-module", "non-existent-job", 0)
	if err == nil {
		t.Error("markTaskCompleted should return error for non-existent job")
	}

	// Test markTaskCompleted with invalid task index
	err = jr.markTaskCompleted("test-module", "pending-job", 100)
	if err == nil {
		t.Error("markTaskCompleted should return error for invalid task index")
	}
}

func TestJobRunner_Run_RunningJobAllowed(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	// Create a job with RUNNING status
	stateManager.Load()
	statePtr := stateManager.GetState()
	if statePtr != nil {
		statePtr.Modules["test-module"].Jobs["pending-job"].Status = state.StatusRunning
		stateManager.Save()
	}

	taskExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		return nil
	}

	jr := NewJobRunner(stateManager, logger, taskExecutor)

	ctx := context.Background()
	completed, err := jr.Run(ctx, "test-module", "pending-job")

	if err != nil {
		t.Errorf("Run() with RUNNING status should succeed, got error: %v", err)
	}
	if completed != 3 {
		t.Errorf("Run() completed = %d, want 3", completed)
	}
}

func TestJobRunner_SetTaskExecutor_Integration(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	executed := []int{}

	// Create runner with nil executor (uses default)
	jr := NewJobRunner(stateManager, logger, nil)

	// Set a custom executor that tracks execution
	jr.SetTaskExecutor(func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		executed = append(executed, taskIndex)
		return nil
	})

	ctx := context.Background()
	completed, err := jr.Run(ctx, "test-module", "pending-job")

	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
	if completed != 3 {
		t.Errorf("Run() completed = %d, want 3", completed)
	}
	if len(executed) != 3 {
		t.Errorf("Expected 3 tasks executed, got %d", len(executed))
	}
}

func TestJobRunner_TaskState_Preservation(t *testing.T) {
	_, stateManager, logger, cleanup := setupJobRunnerTestEnv(t)
	defer cleanup()

	// Track which tasks were passed to executor
	receivedDescs := []string{}

	taskExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		receivedDescs = append(receivedDescs, taskDesc)
		return nil
	}

	jr := NewJobRunner(stateManager, logger, taskExecutor)

	ctx := context.Background()
	jr.Run(ctx, "test-module", "pending-job")

	// Verify task descriptions were preserved
	expectedDescs := []string{"Task 1", "Task 2", "Task 3"}
	for i, expected := range expectedDescs {
		if i >= len(receivedDescs) {
			t.Errorf("Task %d description missing", i)
			continue
		}
		if receivedDescs[i] != expected {
			t.Errorf("Task %d description = %q, want %q", i, receivedDescs[i], expected)
		}
	}
}
