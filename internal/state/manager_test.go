// Package state provides state management for Morty.
package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestManager creates a temporary manager for testing.
func setupTestManager(t *testing.T) (*Manager, string) {
	t.Helper()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "test_status.json")

	manager := NewManager(stateFile)
	if err := manager.Load(); err != nil {
		t.Fatalf("Failed to load manager: %v", err)
	}

	return manager, stateFile
}

// createTestJob creates a test job with the given status.
func createTestJob(name string, status Status) *JobState {
	now := time.Now()
	return &JobState{
		Name:           name,
		Status:         status,
		LoopCount:      0,
		RetryCount:     0,
		TasksTotal:     5,
		TasksCompleted: 0,
		Tasks:          make([]TaskState, 0),
		DebugLogs:      make([]DebugLogEntry, 0),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// createTestModule creates a test module with jobs.
func createTestModule(name string, jobs map[string]*JobState) *ModuleState {
	now := time.Now()
	if jobs == nil {
		jobs = make(map[string]*JobState)
	}
	return &ModuleState{
		Name:      name,
		Status:    StatusPending,
		Jobs:      jobs,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// TestGetJobStatus tests the GetJobStatus method.
func TestGetJobStatus(t *testing.T) {
	t.Run("Get existing job status returns correct value", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup test data
		job := createTestJob("test_job", StatusRunning)
		module := createTestModule("test_module", map[string]*JobState{
			"test_job": job,
		})
		manager.SetModule(module)

		// Test GetJobStatus
		status, err := manager.GetJobStatus("test_module", "test_job")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if status != StatusRunning {
			t.Errorf("Expected status %s, got: %s", StatusRunning, status)
		}
	})

	t.Run("Get non-existent module returns error", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		_, err := manager.GetJobStatus("non_existent_module", "test_job")
		if err == nil {
			t.Error("Expected error for non-existent module, got nil")
		}
	})

	t.Run("Get non-existent job returns error", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup module without the job
		module := createTestModule("test_module", map[string]*JobState{})
		manager.SetModule(module)

		_, err := manager.GetJobStatus("test_module", "non_existent_job")
		if err == nil {
			t.Error("Expected error for non-existent job, got nil")
		}
	})

	t.Run("GetJobStatus with nil state returns error", func(t *testing.T) {
		manager := NewManager("/tmp/test.json")
		// Don't load state, keep it nil

		_, err := manager.GetJobStatus("test_module", "test_job")
		if err == nil {
			t.Error("Expected error when state is nil, got nil")
		}
	})
}

// TestUpdateJobStatus tests the UpdateJobStatus method.
func TestUpdateJobStatus(t *testing.T) {
	t.Run("Update job status and save to file", func(t *testing.T) {
		manager, stateFile := setupTestManager(t)

		// Setup test data
		job := createTestJob("test_job", StatusPending)
		module := createTestModule("test_module", map[string]*JobState{
			"test_job": job,
		})
		manager.SetModule(module)

		// Update job status
		err := manager.UpdateJobStatus("test_module", "test_job", StatusRunning)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify status was updated in memory
		status, _ := manager.GetJobStatus("test_module", "test_job")
		if status != StatusRunning {
			t.Errorf("Expected status %s, got: %s", StatusRunning, status)
		}

		// Verify file was saved
		if _, err := os.Stat(stateFile); os.IsNotExist(err) {
			t.Error("Expected state file to exist after UpdateJobStatus")
		}
	})

	t.Run("Update job status with invalid status returns error", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup test data
		job := createTestJob("test_job", StatusPending)
		module := createTestModule("test_module", map[string]*JobState{
			"test_job": job,
		})
		manager.SetModule(module)

		// Try to update with invalid status
		err := manager.UpdateJobStatus("test_module", "test_job", Status("INVALID"))
		if err == nil {
			t.Error("Expected error for invalid status, got nil")
		}
	})

	t.Run("Update non-existent module returns error", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		err := manager.UpdateJobStatus("non_existent_module", "test_job", StatusRunning)
		if err == nil {
			t.Error("Expected error for non-existent module, got nil")
		}
	})

	t.Run("Update non-existent job returns error", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup module without the job
		module := createTestModule("test_module", map[string]*JobState{})
		manager.SetModule(module)

		err := manager.UpdateJobStatus("test_module", "non_existent_job", StatusRunning)
		if err == nil {
			t.Error("Expected error for non-existent job, got nil")
		}
	})

	t.Run("UpdateJobStatus with nil state returns error", func(t *testing.T) {
		manager := NewManager("/tmp/test.json")

		err := manager.UpdateJobStatus("test_module", "test_job", StatusRunning)
		if err == nil {
			t.Error("Expected error when state is nil, got nil")
		}
	})
}

// TestGetCurrent tests the GetCurrent method.
func TestGetCurrent(t *testing.T) {
	t.Run("GetCurrent returns current executing job from global state", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup test data
		job := createTestJob("test_job", StatusRunning)
		module := createTestModule("test_module", map[string]*JobState{
			"test_job": job,
		})
		manager.SetModule(module)

		// Set current via SetCurrent
		manager.SetCurrent("test_module", "test_job", StatusRunning)

		// Get current
		current, err := manager.GetCurrent()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if current == nil {
			t.Fatal("Expected current job, got nil")
		}
		if current.Module != "test_module" {
			t.Errorf("Expected module 'test_module', got: %s", current.Module)
		}
		if current.Job != "test_job" {
			t.Errorf("Expected job 'test_job', got: %s", current.Job)
		}
	})

	t.Run("GetCurrent finds running job when global not set", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup test data with a running job
		job := createTestJob("running_job", StatusRunning)
		module := createTestModule("test_module", map[string]*JobState{
			"running_job": job,
		})
		manager.SetModule(module)

		// Get current without setting global
		current, err := manager.GetCurrent()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if current == nil {
			t.Fatal("Expected current job, got nil")
		}
		if current.Status != StatusRunning {
			t.Errorf("Expected status RUNNING, got: %s", current.Status)
		}
	})

	t.Run("GetCurrent returns nil when no job is running", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup test data with only pending jobs
		job := createTestJob("pending_job", StatusPending)
		module := createTestModule("test_module", map[string]*JobState{
			"pending_job": job,
		})
		manager.SetModule(module)

		current, err := manager.GetCurrent()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if current != nil {
			t.Errorf("Expected nil, got: %+v", current)
		}
	})

	t.Run("GetCurrent with nil state returns error", func(t *testing.T) {
		manager := NewManager("/tmp/test.json")

		_, err := manager.GetCurrent()
		if err == nil {
			t.Error("Expected error when state is nil, got nil")
		}
	})
}

// TestSetCurrent tests the SetCurrent method.
func TestSetCurrent(t *testing.T) {
	t.Run("SetCurrent updates global state", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup test data
		job := createTestJob("test_job", StatusPending)
		module := createTestModule("test_module", map[string]*JobState{
			"test_job": job,
		})
		manager.SetModule(module)

		// Set current
		err := manager.SetCurrent("test_module", "test_job", StatusRunning)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify global state was updated
		state := manager.GetState()
		if state.Global.CurrentModule != "test_module" {
			t.Errorf("Expected global current module 'test_module', got: %s", state.Global.CurrentModule)
		}
		if state.Global.CurrentJob != "test_job" {
			t.Errorf("Expected global current job 'test_job', got: %s", state.Global.CurrentJob)
		}
		if state.Global.Status != StatusRunning {
			t.Errorf("Expected global status RUNNING, got: %s", state.Global.Status)
		}
	})

	t.Run("SetCurrent with invalid status returns error", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup test data
		job := createTestJob("test_job", StatusPending)
		module := createTestModule("test_module", map[string]*JobState{
			"test_job": job,
		})
		manager.SetModule(module)

		// Try to set with invalid status
		err := manager.SetCurrent("test_module", "test_job", Status("INVALID"))
		if err == nil {
			t.Error("Expected error for invalid status, got nil")
		}
	})

	t.Run("SetCurrent with non-existent module returns error", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		err := manager.SetCurrent("non_existent_module", "test_job", StatusRunning)
		if err == nil {
			t.Error("Expected error for non-existent module, got nil")
		}
	})

	t.Run("SetCurrent with non-existent job returns error", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup module without the job
		module := createTestModule("test_module", map[string]*JobState{})
		manager.SetModule(module)

		err := manager.SetCurrent("test_module", "non_existent_job", StatusRunning)
		if err == nil {
			t.Error("Expected error for non-existent job, got nil")
		}
	})

	t.Run("SetCurrent with nil state returns error", func(t *testing.T) {
		manager := NewManager("/tmp/test.json")

		err := manager.SetCurrent("test_module", "test_job", StatusRunning)
		if err == nil {
			t.Error("Expected error when state is nil, got nil")
		}
	})
}

// TestGetSummary tests the GetSummary method.
func TestGetSummary(t *testing.T) {
	t.Run("GetSummary returns correct statistics", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup test data
		module1 := createTestModule("module1", map[string]*JobState{
			"job1": createTestJob("job1", StatusPending),
			"job2": createTestJob("job2", StatusRunning),
			"job3": createTestJob("job3", StatusCompleted),
		})
		module2 := createTestModule("module2", map[string]*JobState{
			"job4": createTestJob("job4", StatusFailed),
			"job5": createTestJob("job5", StatusBlocked),
		})
		manager.SetModule(module1)
		manager.SetModule(module2)

		// Get summary
		summary, err := manager.GetSummary()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify counts
		if summary.TotalModules != 2 {
			t.Errorf("Expected 2 total modules, got: %d", summary.TotalModules)
		}
		if summary.TotalJobs != 5 {
			t.Errorf("Expected 5 total jobs, got: %d", summary.TotalJobs)
		}
		if summary.Pending != 1 {
			t.Errorf("Expected 1 pending job, got: %d", summary.Pending)
		}
		if summary.Running != 1 {
			t.Errorf("Expected 1 running job, got: %d", summary.Running)
		}
		if summary.Completed != 1 {
			t.Errorf("Expected 1 completed job, got: %d", summary.Completed)
		}
		if summary.Failed != 1 {
			t.Errorf("Expected 1 failed job, got: %d", summary.Failed)
		}
		if summary.Blocked != 1 {
			t.Errorf("Expected 1 blocked job, got: %d", summary.Blocked)
		}

		// Verify per-module statistics
		if len(summary.Modules) != 2 {
			t.Errorf("Expected 2 module summaries, got: %d", len(summary.Modules))
		}
	})

	t.Run("GetSummary with empty state returns zero counts", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		summary, err := manager.GetSummary()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if summary.TotalModules != 0 {
			t.Errorf("Expected 0 total modules, got: %d", summary.TotalModules)
		}
		if summary.TotalJobs != 0 {
			t.Errorf("Expected 0 total jobs, got: %d", summary.TotalJobs)
		}
	})

	t.Run("GetSummary with nil state returns error", func(t *testing.T) {
		manager := NewManager("/tmp/test.json")

		_, err := manager.GetSummary()
		if err == nil {
			t.Error("Expected error when state is nil, got nil")
		}
	})
}

// TestGetPendingJobs tests the GetPendingJobs method.
func TestGetPendingJobs(t *testing.T) {
	t.Run("GetPendingJobs returns only pending jobs", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup test data with mixed statuses
		module1 := createTestModule("module1", map[string]*JobState{
			"pending1":  createTestJob("pending1", StatusPending),
			"running1":  createTestJob("running1", StatusRunning),
			"pending2":  createTestJob("pending2", StatusPending),
		})
		module2 := createTestModule("module2", map[string]*JobState{
			"completed": createTestJob("completed", StatusCompleted),
			"pending3":  createTestJob("pending3", StatusPending),
		})
		manager.SetModule(module1)
		manager.SetModule(module2)

		// Get pending jobs
		pending := manager.GetPendingJobs()

		// Should have 3 pending jobs
		if len(pending) != 3 {
			t.Errorf("Expected 3 pending jobs, got: %d", len(pending))
		}

		// Verify all returned jobs are pending
		for _, jobRef := range pending {
			if jobRef.Status != StatusPending {
				t.Errorf("Expected status PENDING, got: %s", jobRef.Status)
			}
		}
	})

	t.Run("GetPendingJobs returns empty slice when no pending jobs", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		// Setup test data with no pending jobs
		module := createTestModule("module1", map[string]*JobState{
			"running":   createTestJob("running", StatusRunning),
			"completed": createTestJob("completed", StatusCompleted),
		})
		manager.SetModule(module)

		pending := manager.GetPendingJobs()

		if len(pending) != 0 {
			t.Errorf("Expected 0 pending jobs, got: %d", len(pending))
		}
	})

	t.Run("GetPendingJobs returns empty slice when state is nil", func(t *testing.T) {
		manager := NewManager("/tmp/test.json")

		pending := manager.GetPendingJobs()

		if len(pending) != 0 {
			t.Errorf("Expected 0 pending jobs when state is nil, got: %d", len(pending))
		}
	})

	t.Run("GetPendingJobs returns empty slice for empty state", func(t *testing.T) {
		manager, _ := setupTestManager(t)

		pending := manager.GetPendingJobs()

		if len(pending) != 0 {
			t.Errorf("Expected 0 pending jobs for empty state, got: %d", len(pending))
		}
	})
}

// TestIntegration tests the integration of manager methods.
func TestIntegration(t *testing.T) {
	t.Run("Full workflow: create, update, query jobs", func(t *testing.T) {
		manager, stateFile := setupTestManager(t)

		// 1. Create a module with jobs
		module := createTestModule("test_module", map[string]*JobState{
			"job1": createTestJob("job1", StatusPending),
			"job2": createTestJob("job2", StatusPending),
		})
		manager.SetModule(module)

		// 2. Verify initial state
		summary, _ := manager.GetSummary()
		if summary.TotalJobs != 2 {
			t.Errorf("Expected 2 jobs, got: %d", summary.TotalJobs)
		}
		if summary.Pending != 2 {
			t.Errorf("Expected 2 pending, got: %d", summary.Pending)
		}

		// 3. Set current job
		err := manager.SetCurrent("test_module", "job1", StatusRunning)
		if err != nil {
			t.Errorf("SetCurrent failed: %v", err)
		}

		// 4. Update job status
		err = manager.UpdateJobStatus("test_module", "job1", StatusRunning)
		if err != nil {
			t.Errorf("UpdateJobStatus failed: %v", err)
		}

		// 5. Verify current job
		current, _ := manager.GetCurrent()
		if current == nil {
			t.Fatal("Expected current job")
		}
		if current.Job != "job1" {
			t.Errorf("Expected current job 'job1', got: %s", current.Job)
		}

		// 6. Complete job
		err = manager.UpdateJobStatus("test_module", "job1", StatusCompleted)
		if err != nil {
			t.Errorf("Failed to complete job: %v", err)
		}

		// 7. Verify updated summary
		summary, _ = manager.GetSummary()
		if summary.Completed != 1 {
			t.Errorf("Expected 1 completed, got: %d", summary.Completed)
		}
		if summary.Pending != 1 {
			t.Errorf("Expected 1 pending, got: %d", summary.Pending)
		}

		// 8. Verify state file exists
		if _, err := os.Stat(stateFile); os.IsNotExist(err) {
			t.Error("Expected state file to exist")
		}
	})
}
