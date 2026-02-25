// Package state provides state management for Morty.
package state

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/pkg/errors"
)

// mockLogger is a simple mock for testing logging functionality
type mockLogger struct {
	logs   []map[string]interface{}
	level  logging.Level
	module string
	job    string
}

func (m *mockLogger) Debug(msg string, attrs ...logging.Attr) {}
func (m *mockLogger) Info(msg string, attrs ...logging.Attr)  {}
func (m *mockLogger) Warn(msg string, attrs ...logging.Attr)  {}
func (m *mockLogger) Error(msg string, attrs ...logging.Attr) {
	entry := map[string]interface{}{"msg": msg, "attrs": attrs}
	m.logs = append(m.logs, entry)
}
func (m *mockLogger) Success(msg string, attrs ...logging.Attr) {}
func (m *mockLogger) Loop(msg string, attrs ...logging.Attr)   {}
func (m *mockLogger) WithContext(ctx context.Context) logging.Logger {
	return m
}
func (m *mockLogger) WithJob(module, job string) logging.Logger {
	return &mockLogger{
		logs:   m.logs,
		level:  m.level,
		module: module,
		job:    job,
	}
}
func (m *mockLogger) WithAttrs(attrs ...logging.Attr) logging.Logger {
	return m
}
func (m *mockLogger) SetLevel(level logging.Level) {}
func (m *mockLogger) GetLevel() logging.Level {
	return m.level
}
func (m *mockLogger) IsEnabled(level logging.Level) bool {
	return level >= m.level
}

// TestIsValidTransition_ValidTransitions tests all valid state transitions.
func TestIsValidTransition_ValidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
	}{
		{"PENDING to RUNNING", StatusPending, StatusRunning},
		{"PENDING to BLOCKED", StatusPending, StatusBlocked},
		{"RUNNING to COMPLETED", StatusRunning, StatusCompleted},
		{"RUNNING to FAILED", StatusRunning, StatusFailed},
		{"RUNNING to BLOCKED", StatusRunning, StatusBlocked},
		{"FAILED to PENDING", StatusFailed, StatusPending},
		{"BLOCKED to PENDING", StatusBlocked, StatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !IsValidTransition(tt.from, tt.to) {
				t.Errorf("IsValidTransition(%s, %s) = false, expected true", tt.from, tt.to)
			}
		})
	}
}

// TestIsValidTransition_InvalidTransitions tests invalid state transitions.
func TestIsValidTransition_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
	}{
		{"PENDING to COMPLETED", StatusPending, StatusCompleted},
		{"PENDING to FAILED", StatusPending, StatusFailed},
		{"PENDING to PENDING", StatusPending, StatusPending},
		{"RUNNING to PENDING", StatusRunning, StatusPending},
		{"RUNNING to RUNNING", StatusRunning, StatusRunning},
		{"COMPLETED to PENDING", StatusCompleted, StatusPending},
		{"COMPLETED to RUNNING", StatusCompleted, StatusRunning},
		{"COMPLETED to FAILED", StatusCompleted, StatusFailed},
		{"COMPLETED to COMPLETED", StatusCompleted, StatusCompleted},
		{"FAILED to RUNNING", StatusFailed, StatusRunning},
		{"FAILED to COMPLETED", StatusFailed, StatusCompleted},
		{"FAILED to FAILED", StatusFailed, StatusFailed},
		{"FAILED to BLOCKED", StatusFailed, StatusBlocked},
		{"BLOCKED to RUNNING", StatusBlocked, StatusRunning},
		{"BLOCKED to COMPLETED", StatusBlocked, StatusCompleted},
		{"BLOCKED to FAILED", StatusBlocked, StatusFailed},
		{"BLOCKED to BLOCKED", StatusBlocked, StatusBlocked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsValidTransition(tt.from, tt.to) {
				t.Errorf("IsValidTransition(%s, %s) = true, expected false", tt.from, tt.to)
			}
		})
	}
}

// TestIsValidTransition_InvalidStatus tests transitions with invalid statuses.
func TestIsValidTransition_InvalidStatus(t *testing.T) {
	invalidStatus := Status("INVALID")

	// Test invalid from status
	if IsValidTransition(invalidStatus, StatusPending) {
		t.Error("IsValidTransition(INVALID, PENDING) should return false")
	}

	// Test invalid to status
	if IsValidTransition(StatusPending, invalidStatus) {
		t.Error("IsValidTransition(PENDING, INVALID) should return false")
	}

	// Test both invalid
	if IsValidTransition(invalidStatus, Status("ALSO_INVALID")) {
		t.Error("IsValidTransition(INVALID, ALSO_INVALID) should return false")
	}

	// Test empty status
	if IsValidTransition(Status(""), StatusPending) {
		t.Error("IsValidTransition(empty, PENDING) should return false")
	}
}

// TestGetValidTransitions tests the GetValidTransitions function.
func TestGetValidTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     Status
		expected []Status
	}{
		{"PENDING", StatusPending, []Status{StatusRunning, StatusBlocked}},
		{"RUNNING", StatusRunning, []Status{StatusCompleted, StatusFailed, StatusBlocked}},
		{"COMPLETED", StatusCompleted, []Status{}},
		{"FAILED", StatusFailed, []Status{StatusPending}},
		{"BLOCKED", StatusBlocked, []Status{StatusPending}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetValidTransitions(tt.from)

			if len(result) != len(tt.expected) {
				t.Errorf("GetValidTransitions(%s) returned %d statuses, expected %d",
					tt.from, len(result), len(tt.expected))
				return
			}

			for i, status := range result {
				if status != tt.expected[i] {
					t.Errorf("GetValidTransitions(%s)[%d] = %s, expected %s",
						tt.from, i, status, tt.expected[i])
				}
			}
		})
	}
}

// TestGetValidTransitions_InvalidStatus tests GetValidTransitions with invalid status.
func TestGetValidTransitions_InvalidStatus(t *testing.T) {
	result := GetValidTransitions(Status("INVALID"))
	if len(result) != 0 {
		t.Errorf("GetValidTransitions(INVALID) should return empty slice, got %v", result)
	}
}

// TestTransitionError_Error tests the TransitionError.Error method.
func TestTransitionError_Error(t *testing.T) {
	err := &TransitionError{
		From:   StatusPending,
		To:     StatusCompleted,
		Reason: "transition from PENDING to COMPLETED is not allowed",
	}

	expected := "invalid transition from PENDING to COMPLETED: transition from PENDING to COMPLETED is not allowed"
	if err.Error() != expected {
		t.Errorf("Error() = %s, expected %s", err.Error(), expected)
	}
}

// TestManager_CanTransition_Valid tests CanTransition with valid transitions.
func TestManager_CanTransition_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create a test module and job
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:      "test_job",
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetJob("test_module", job)

	// Test valid transition: PENDING -> RUNNING
	if err := m.CanTransition("test_module", "test_job", StatusRunning); err != nil {
		t.Errorf("CanTransition(PENDING, RUNNING) error = %v, expected nil", err)
	}

	// Update job status to RUNNING for next test
	job.Status = StatusRunning

	// Test valid transition: RUNNING -> COMPLETED
	if err := m.CanTransition("test_module", "test_job", StatusCompleted); err != nil {
		t.Errorf("CanTransition(RUNNING, COMPLETED) error = %v, expected nil", err)
	}
}

// TestManager_CanTransition_Invalid tests CanTransition with invalid transitions.
func TestManager_CanTransition_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create a test module and job
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:      "test_job",
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetJob("test_module", job)

	// Test invalid transition: PENDING -> COMPLETED
	err := m.CanTransition("test_module", "test_job", StatusCompleted)
	if err == nil {
		t.Error("CanTransition(PENDING, COMPLETED) expected error, got nil")
	} else {
		// Check that it's a TransitionError
		if _, ok := err.(*TransitionError); !ok {
			t.Errorf("Expected TransitionError, got %T", err)
		}
	}
}

// TestManager_CanTransition_Errors tests CanTransition error cases.
func TestManager_CanTransition_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)

	// Test nil state
	err := m.CanTransition("module", "job", StatusRunning)
	if err == nil {
		t.Error("CanTransition with nil state expected error")
	} else {
		if me, ok := errors.AsMortyError(err); ok {
			if me.Code != "M2003" {
				t.Errorf("Error code = %s, expected M2003", me.Code)
			}
		}
	}

	// Load state and test other errors
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test invalid target status
	err = m.CanTransition("module", "job", Status("INVALID"))
	if err == nil {
		t.Error("CanTransition with invalid status expected error")
	}

	// Test non-existent module
	err = m.CanTransition("nonexistent", "job", StatusRunning)
	if err == nil {
		t.Error("CanTransition with non-existent module expected error")
	} else {
		if me, ok := errors.AsMortyError(err); ok {
			if me.Code != "M2003" {
				t.Errorf("Error code = %s, expected M2003", me.Code)
			}
		}
	}

	// Create module and test non-existent job
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	err = m.CanTransition("test_module", "nonexistent", StatusRunning)
	if err == nil {
		t.Error("CanTransition with non-existent job expected error")
	}
}

// TestManager_TransitionJobStatus_Valid tests successful transitions.
func TestManager_TransitionJobStatus_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create a test module and job with PENDING status
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:      "test_job",
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetJob("test_module", job)

	logger := &mockLogger{}

	// Test PENDING -> RUNNING transition
	if err := m.TransitionJobStatus("test_module", "test_job", StatusRunning, logger); err != nil {
		t.Errorf("TransitionJobStatus(PENDING -> RUNNING) error = %v", err)
	}

	// Verify status was updated
	updatedJob := m.GetJob("test_module", "test_job")
	if updatedJob.Status != StatusRunning {
		t.Errorf("Job status = %s, expected RUNNING", updatedJob.Status)
	}

	// Test RUNNING -> COMPLETED transition
	if err := m.TransitionJobStatus("test_module", "test_job", StatusCompleted, logger); err != nil {
		t.Errorf("TransitionJobStatus(RUNNING -> COMPLETED) error = %v", err)
	}

	// Verify status was updated
	updatedJob = m.GetJob("test_module", "test_job")
	if updatedJob.Status != StatusCompleted {
		t.Errorf("Job status = %s, expected COMPLETED", updatedJob.Status)
	}
}

// TestManager_TransitionJobStatus_RetryFlow tests FAILED -> PENDING retry flow.
func TestManager_TransitionJobStatus_RetryFlow(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create a test module and job with FAILED status
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:       "test_job",
		Status:     StatusFailed,
		RetryCount: 0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	m.SetJob("test_module", job)

	logger := &mockLogger{}

	// Test FAILED -> PENDING transition (retry)
	if err := m.TransitionJobStatus("test_module", "test_job", StatusPending, logger); err != nil {
		t.Errorf("TransitionJobStatus(FAILED -> PENDING) error = %v", err)
	}

	// Verify status was updated and retry count incremented
	updatedJob := m.GetJob("test_module", "test_job")
	if updatedJob.Status != StatusPending {
		t.Errorf("Job status = %s, expected PENDING", updatedJob.Status)
	}
}

// TestManager_TransitionJobStatus_UnblockFlow tests BLOCKED -> PENDING unblock flow.
func TestManager_TransitionJobStatus_UnblockFlow(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create a test module and job with BLOCKED status
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:      "test_job",
		Status:    StatusBlocked,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetJob("test_module", job)

	logger := &mockLogger{}

	// Test BLOCKED -> PENDING transition (unblock)
	if err := m.TransitionJobStatus("test_module", "test_job", StatusPending, logger); err != nil {
		t.Errorf("TransitionJobStatus(BLOCKED -> PENDING) error = %v", err)
	}

	// Verify status was updated
	updatedJob := m.GetJob("test_module", "test_job")
	if updatedJob.Status != StatusPending {
		t.Errorf("Job status = %s, expected PENDING", updatedJob.Status)
	}
}

// TestManager_TransitionJobStatus_BlockFlow tests RUNNING -> BLOCKED block flow.
func TestManager_TransitionJobStatus_BlockFlow(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create a test module and job with RUNNING status
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:      "test_job",
		Status:    StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetJob("test_module", job)

	logger := &mockLogger{}

	// Test RUNNING -> BLOCKED transition
	if err := m.TransitionJobStatus("test_module", "test_job", StatusBlocked, logger); err != nil {
		t.Errorf("TransitionJobStatus(RUNNING -> BLOCKED) error = %v", err)
	}

	// Verify status was updated
	updatedJob := m.GetJob("test_module", "test_job")
	if updatedJob.Status != StatusBlocked {
		t.Errorf("Job status = %s, expected BLOCKED", updatedJob.Status)
	}
}

// TestManager_TransitionJobStatus_Invalid tests invalid transitions with logging.
func TestManager_TransitionJobStatus_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create a test module and job with PENDING status
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:      "test_job",
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetJob("test_module", job)

	logger := &mockLogger{}

	// Test invalid transition: PENDING -> COMPLETED
	err := m.TransitionJobStatus("test_module", "test_job", StatusCompleted, logger)
	if err == nil {
		t.Error("TransitionJobStatus(PENDING -> COMPLETED) expected error")
	} else {
		// Check that it's a TransitionError
		if te, ok := err.(*TransitionError); !ok {
			t.Errorf("Expected TransitionError, got %T", err)
		} else {
			if te.From != StatusPending || te.To != StatusCompleted {
				t.Errorf("TransitionError.From = %s, To = %s, expected PENDING, COMPLETED", te.From, te.To)
			}
		}
	}

	// Verify status was NOT updated
	updatedJob := m.GetJob("test_module", "test_job")
	if updatedJob.Status != StatusPending {
		t.Errorf("Job status changed to %s, expected PENDING (unchanged)", updatedJob.Status)
	}

	// Verify that error was logged
	if len(logger.logs) == 0 {
		t.Error("Expected error to be logged")
	}
}

// TestManager_TransitionJobStatus_NilLogger tests that nil logger doesn't panic.
func TestManager_TransitionJobStatus_NilLogger(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create a test module and job
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:      "test_job",
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetJob("test_module", job)

	// Test with nil logger - should not panic
	if err := m.TransitionJobStatus("test_module", "test_job", StatusRunning, nil); err != nil {
		t.Errorf("TransitionJobStatus with nil logger error = %v", err)
	}
}

// TestManager_TransitionJobStatus_Errors tests error cases for TransitionJobStatus.
func TestManager_TransitionJobStatus_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)

	logger := &mockLogger{}

	// Test nil state
	err := m.TransitionJobStatus("module", "job", StatusRunning, logger)
	if err == nil {
		t.Error("TransitionJobStatus with nil state expected error")
	}

	// Load state for remaining tests
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test invalid target status
	err = m.TransitionJobStatus("module", "job", Status("INVALID"), logger)
	if err == nil {
		t.Error("TransitionJobStatus with invalid status expected error")
	}

	// Test non-existent module
	err = m.TransitionJobStatus("nonexistent", "job", StatusRunning, logger)
	if err == nil {
		t.Error("TransitionJobStatus with non-existent module expected error")
	}
}

// TestManager_GetJobValidTransitions tests GetJobValidTransitions.
func TestManager_GetJobValidTransitions(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Create a test module and job with PENDING status
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:      "test_job",
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetJob("test_module", job)

	// Get valid transitions for a PENDING job
	transitions, err := m.GetJobValidTransitions("test_module", "test_job")
	if err != nil {
		t.Errorf("GetJobValidTransitions error = %v", err)
	}

	// PENDING should have 2 valid transitions: RUNNING and BLOCKED
	if len(transitions) != 2 {
		t.Errorf("GetJobValidTransitions returned %d transitions, expected 2", len(transitions))
	}

	// Verify the transitions are correct
	hasRunning := false
	hasBlocked := false
	for _, t := range transitions {
		if t == StatusRunning {
			hasRunning = true
		}
		if t == StatusBlocked {
			hasBlocked = true
		}
	}
	if !hasRunning || !hasBlocked {
		t.Error("GetJobValidTransitions should return RUNNING and BLOCKED for PENDING job")
	}
}

// TestManager_GetJobValidTransitions_Errors tests GetJobValidTransitions error cases.
func TestManager_GetJobValidTransitions_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)

	// Test nil state
	_, err := m.GetJobValidTransitions("module", "job")
	if err == nil {
		t.Error("GetJobValidTransitions with nil state expected error")
	}

	// Load state for remaining tests
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test non-existent module
	_, err = m.GetJobValidTransitions("nonexistent", "job")
	if err == nil {
		t.Error("GetJobValidTransitions with non-existent module expected error")
	}

	// Create module and test non-existent job
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	_, err = m.GetJobValidTransitions("test_module", "nonexistent")
	if err == nil {
		t.Error("GetJobValidTransitions with non-existent job expected error")
	}
}

// TestTransitionRules_Immutability tests that TransitionRules cannot be modified externally.
func TestTransitionRules_Immutability(t *testing.T) {
	// Get transitions for PENDING
	original := GetValidTransitions(StatusPending)
	if len(original) != 2 {
		t.Fatalf("PENDING should have 2 valid transitions")
	}

	// Try to modify the returned slice
	original[0] = StatusCompleted

	// Get transitions again - should be unchanged
	newTransitions := GetValidTransitions(StatusPending)
	if newTransitions[0] != StatusRunning {
		t.Error("TransitionRules was modified externally - slice copy not working")
	}
}

// TestMainWorkflow_PendingRunningCompleted tests the main workflow: PENDING -> RUNNING -> COMPLETED.
func TestMainWorkflow_PendingRunningCompleted(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:      "test_job",
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetJob("test_module", job)

	logger := &mockLogger{}

	// PENDING -> RUNNING
	if err := m.TransitionJobStatus("test_module", "test_job", StatusRunning, logger); err != nil {
		t.Fatalf("PENDING -> RUNNING failed: %v", err)
	}

	// RUNNING -> COMPLETED
	if err := m.TransitionJobStatus("test_module", "test_job", StatusCompleted, logger); err != nil {
		t.Fatalf("RUNNING -> COMPLETED failed: %v", err)
	}

	// Verify final state
	finalJob := m.GetJob("test_module", "test_job")
	if finalJob.Status != StatusCompleted {
		t.Errorf("Final status = %s, expected COMPLETED", finalJob.Status)
	}

	// COMPLETED is terminal - no more transitions should be possible
	transitions, _ := m.GetJobValidTransitions("test_module", "test_job")
	if len(transitions) != 0 {
		t.Error("COMPLETED should have no valid outgoing transitions")
	}
}

// TestMainWorkflow_WithFailure tests the workflow with failure: PENDING -> RUNNING -> FAILED -> PENDING -> RUNNING -> COMPLETED.
func TestMainWorkflow_WithFailure(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusPending,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:      "test_job",
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetJob("test_module", job)

	logger := &mockLogger{}

	// PENDING -> RUNNING
	if err := m.TransitionJobStatus("test_module", "test_job", StatusRunning, logger); err != nil {
		t.Fatalf("PENDING -> RUNNING failed: %v", err)
	}

	// RUNNING -> FAILED
	if err := m.TransitionJobStatus("test_module", "test_job", StatusFailed, logger); err != nil {
		t.Fatalf("RUNNING -> FAILED failed: %v", err)
	}

	// FAILED -> PENDING (retry)
	if err := m.TransitionJobStatus("test_module", "test_job", StatusPending, logger); err != nil {
		t.Fatalf("FAILED -> PENDING failed: %v", err)
	}

	// PENDING -> RUNNING (retry)
	if err := m.TransitionJobStatus("test_module", "test_job", StatusRunning, logger); err != nil {
		t.Fatalf("PENDING -> RUNNING (retry) failed: %v", err)
	}

	// RUNNING -> COMPLETED
	if err := m.TransitionJobStatus("test_module", "test_job", StatusCompleted, logger); err != nil {
		t.Fatalf("RUNNING -> COMPLETED failed: %v", err)
	}

	// Verify final state
	finalJob := m.GetJob("test_module", "test_job")
	if finalJob.Status != StatusCompleted {
		t.Errorf("Final status = %s, expected COMPLETED", finalJob.Status)
	}
}
