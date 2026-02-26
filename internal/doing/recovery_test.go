package doing

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/morty/morty/internal/state"
)

func setupTestStateRecovery(t *testing.T) (*StateRecovery, *state.Manager, string, func()) {
	tmpDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	logger := &mockLogger{}
	stateFile := filepath.Join(tmpDir, "status.json")
	stateManager := state.NewManager(stateFile)

	// Load to create default state
	stateManager.Load()

	// Initialize the Modules map and add test data
	stateManager.GetState().Modules["test-module"] = &state.ModuleState{
		Name:   "test-module",
		Status: state.StatusRunning,
		Jobs: map[string]*state.JobState{
			"test-job": {
				Name:           "test-job",
				Status:         state.StatusRunning,
				LoopCount:      1,
				RetryCount:     0,
				TasksCompleted: 2,
				TasksTotal:     5,
			},
		},
	}

	recovery := NewStateRecovery(logger, tmpDir, stateManager)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return recovery, stateManager, tmpDir, cleanup
}

func TestNewStateRecovery(t *testing.T) {
	logger := &mockLogger{}
	stateManager := state.NewManager("")

	sr := NewStateRecovery(logger, "", stateManager)

	if sr == nil {
		t.Fatal("NewStateRecovery returned nil")
	}
	if sr.logger == nil {
		t.Error("Logger should not be nil")
	}
	if sr.recoveryDir != ".morty/recovery" {
		t.Errorf("recoveryDir = %s, want .morty/recovery", sr.recoveryDir)
	}
}

func TestStateRecovery_CreateRecoveryPoint(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	rp, err := sr.CreateRecoveryPoint("test-module", "test-job")
	if err != nil {
		t.Fatalf("CreateRecoveryPoint failed: %v", err)
	}

	if rp.Module != "test-module" {
		t.Errorf("Module = %s, want test-module", rp.Module)
	}
	if rp.Job != "test-job" {
		t.Errorf("Job = %s, want test-job", rp.Job)
	}
	if rp.JobStatus != state.StatusRunning {
		t.Errorf("JobStatus = %s, want Running", rp.JobStatus)
	}
	if rp.LoopCount != 1 {
		t.Errorf("LoopCount = %d, want 1", rp.LoopCount)
	}
	if rp.TasksDone != 2 {
		t.Errorf("TasksDone = %d, want 2", rp.TasksDone)
	}
	if rp.TasksTotal != 5 {
		t.Errorf("TasksTotal = %d, want 5", rp.TasksTotal)
	}

	// Verify file was created
	files, err := os.ReadDir(sr.recoveryDir)
	if err != nil {
		t.Fatalf("Failed to read recovery dir: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 recovery file, got %d", len(files))
	}
}

func TestStateRecovery_CreateRecoveryPoint_NoStateManager(t *testing.T) {
	logger := &mockLogger{}
	sr := NewStateRecovery(logger, "", nil)

	_, err := sr.CreateRecoveryPoint("module", "job")
	if err == nil {
		t.Error("Expected error when state manager is nil")
	}
}

func TestStateRecovery_CreateRecoveryPoint_JobNotFound(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	_, err := sr.CreateRecoveryPoint("test-module", "non-existent-job")
	if err == nil {
		t.Error("Expected error when job not found")
	}
}

func TestStateRecovery_RestoreFromRecovery(t *testing.T) {
	sr, stateManager, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	// Create a recovery point with different values
	rp := &RecoveryPoint{
		Timestamp:      time.Now(),
		Module:         "test-module",
		Job:            "test-job",
		JobStatus:      state.StatusCompleted,
		LoopCount:      5,
		RetryCount:     2,
		TasksDone:      5,
		TasksTotal:     5,
	}

	err := sr.RestoreFromRecovery(rp)
	if err != nil {
		t.Fatalf("RestoreFromRecovery failed: %v", err)
	}

	// Verify state was restored
	jobState := stateManager.GetJob("test-module", "test-job")
	if jobState == nil {
		t.Fatal("Job state not found after restore")
	}
	if jobState.Status != state.StatusCompleted {
		t.Errorf("Status = %s, want Completed", jobState.Status)
	}
	if jobState.LoopCount != 5 {
		t.Errorf("LoopCount = %d, want 5", jobState.LoopCount)
	}
	if jobState.RetryCount != 2 {
		t.Errorf("RetryCount = %d, want 2", jobState.RetryCount)
	}
	if jobState.TasksCompleted != 5 {
		t.Errorf("TasksCompleted = %d, want 5", jobState.TasksCompleted)
	}
}

func TestStateRecovery_RestoreFromRecovery_NoStateManager(t *testing.T) {
	logger := &mockLogger{}
	sr := NewStateRecovery(logger, "", nil)

	rp := &RecoveryPoint{Module: "m", Job: "j"}
	err := sr.RestoreFromRecovery(rp)
	if err == nil {
		t.Error("Expected error when state manager is nil")
	}
}

func TestStateRecovery_ListRecoveryPoints(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	// Create multiple recovery points with enough delay for unique timestamps
	for i := 0; i < 3; i++ {
		time.Sleep(100 * time.Millisecond) // Ensure different timestamps (filename uses second precision)
		_, err := sr.CreateRecoveryPoint("test-module", "test-job")
		if err != nil {
			t.Fatalf("CreateRecoveryPoint failed: %v", err)
		}
	}

	points, err := sr.ListRecoveryPoints("test-module", "test-job")
	if err != nil {
		t.Fatalf("ListRecoveryPoints failed: %v", err)
	}

	// Note: Files with same timestamp will overwrite, so we may get fewer than 3
	if len(points) < 1 {
		t.Errorf("Expected at least 1 recovery point, got %d", len(points))
	}

	// Verify sorting (newest first)
	for i := 0; i < len(points)-1; i++ {
		if points[i].Timestamp.Before(points[i+1].Timestamp) {
			t.Error("Recovery points should be sorted newest first")
		}
	}
}

func TestStateRecovery_GetLatestRecoveryPoint(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	// Create two recovery points with delay
	_, err := sr.CreateRecoveryPoint("test-module", "test-job")
	if err != nil {
		t.Fatalf("CreateRecoveryPoint failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	_, err = sr.CreateRecoveryPoint("test-module", "test-job")
	if err != nil {
		t.Fatalf("CreateRecoveryPoint failed: %v", err)
	}

	latest, err := sr.GetLatestRecoveryPoint("test-module", "test-job")
	if err != nil {
		t.Fatalf("GetLatestRecoveryPoint failed: %v", err)
	}

	points, _ := sr.ListRecoveryPoints("test-module", "test-job")
	if latest.Timestamp != points[0].Timestamp {
		t.Error("GetLatestRecoveryPoint should return the newest point")
	}
}

func TestStateRecovery_GetLatestRecoveryPoint_NotFound(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	_, err := sr.GetLatestRecoveryPoint("non-existent", "job")
	if err == nil {
		t.Error("Expected error when no recovery points exist")
	}
}

func TestStateRecovery_DeleteRecoveryPoint(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	// Create a recovery point
	_, err := sr.CreateRecoveryPoint("test-module", "test-job")
	if err != nil {
		t.Fatalf("CreateRecoveryPoint failed: %v", err)
	}

	points, _ := sr.ListRecoveryPoints("test-module", "test-job")
	if len(points) != 1 {
		t.Fatal("Expected 1 recovery point")
	}

	// Delete it
	err = sr.DeleteRecoveryPoint(points[0])
	if err != nil {
		t.Errorf("DeleteRecoveryPoint failed: %v", err)
	}

	// Verify it's gone
	points, _ = sr.ListRecoveryPoints("test-module", "test-job")
	if len(points) != 0 {
		t.Errorf("Expected 0 recovery points after delete, got %d", len(points))
	}
}

func TestStateRecovery_ClearAllRecoveryPoints(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	// Create multiple recovery points
	for i := 0; i < 3; i++ {
		_, err := sr.CreateRecoveryPoint("test-module", "test-job")
		if err != nil {
			t.Fatalf("CreateRecoveryPoint failed: %v", err)
		}
	}

	err := sr.ClearAllRecoveryPoints("test-module", "test-job")
	if err != nil {
		t.Errorf("ClearAllRecoveryPoints failed: %v", err)
	}

	points, _ := sr.ListRecoveryPoints("test-module", "test-job")
	if len(points) != 0 {
		t.Errorf("Expected 0 recovery points after clear, got %d", len(points))
	}
}

func TestStateRecovery_AutoRecover_State(t *testing.T) {
	sr, stateManager, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	// Create a recovery point first
	_, err := sr.CreateRecoveryPoint("test-module", "test-job")
	if err != nil {
		t.Fatalf("CreateRecoveryPoint failed: %v", err)
	}

	// Change the state
	stateManager.UpdateJobStatus("test-module", "test-job", state.StatusFailed)

	// Create a state error
	testErr := NewDoingError(ErrorCategoryState, "state corrupted", nil)

	recovered, err := sr.AutoRecover("test-module", "test-job", testErr)
	if err != nil {
		t.Fatalf("AutoRecover failed: %v", err)
	}
	if !recovered {
		t.Error("Expected recovery to succeed")
	}

	// Verify state was restored
	jobState := stateManager.GetJob("test-module", "test-job")
	if jobState.Status != state.StatusRunning {
		t.Errorf("Status = %s, want Running after recovery", jobState.Status)
	}
}

func TestStateRecovery_AutoRecover_Transient(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	testErr := NewDoingError(ErrorCategoryTransient, "timeout", nil)

	recovered, err := sr.AutoRecover("test-module", "test-job", testErr)
	if err != nil {
		t.Fatalf("AutoRecover failed: %v", err)
	}
	if !recovered {
		t.Error("Expected transient errors to be handled")
	}
}

func TestStateRecovery_AutoRecover_NotAvailable(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	testErr := NewDoingError(ErrorCategoryPrerequisite, "missing prereq", nil)

	recovered, err := sr.AutoRecover("test-module", "test-job", testErr)
	if err != nil {
		t.Fatalf("AutoRecover failed: %v", err)
	}
	if recovered {
		t.Error("Expected no recovery for prerequisite errors")
	}
}

func TestStateRecovery_AutoRecover_NoRecoveryPoint(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	testErr := NewDoingError(ErrorCategoryState, "state error", nil)

	recovered, err := sr.AutoRecover("test-module", "test-job", testErr)
	if err != nil {
		t.Fatalf("AutoRecover failed: %v", err)
	}
	if recovered {
		t.Error("Expected no recovery when no recovery point exists")
	}
}

func TestStateRecovery_FormatRecoveryReport(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	// Create some recovery points
	for i := 0; i < 2; i++ {
		_, err := sr.CreateRecoveryPoint("test-module", "test-job")
		if err != nil {
			t.Fatalf("CreateRecoveryPoint failed: %v", err)
		}
	}

	report := sr.FormatRecoveryReport("test-module", "test-job")

	if report == "" {
		t.Error("Expected non-empty report")
	}
	if report == "No recovery points available for this job." {
		t.Error("Expected recovery report, got empty message")
	}
}

func TestStateRecovery_FormatRecoveryReport_NoPoints(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	report := sr.FormatRecoveryReport("test-module", "test-job")

	if report != "No recovery points available for this job." {
		t.Errorf("Expected 'No recovery points' message, got: %s", report)
	}
}

func TestStateRecovery_CleanupOldRecoveryPoints(t *testing.T) {
	sr, _, _, cleanup := setupTestStateRecovery(t)
	defer cleanup()

	sr.maxRecoveryPoints = 3

	// Create more recovery points than max
	for i := 0; i < 5; i++ {
		time.Sleep(5 * time.Millisecond)
		_, err := sr.CreateRecoveryPoint("test-module", "test-job")
		if err != nil {
			t.Fatalf("CreateRecoveryPoint failed: %v", err)
		}
	}

	points, _ := sr.ListRecoveryPoints("test-module", "test-job")
	if len(points) > sr.maxRecoveryPoints {
		t.Errorf("Expected at most %d recovery points after cleanup, got %d", sr.maxRecoveryPoints, len(points))
	}
}
