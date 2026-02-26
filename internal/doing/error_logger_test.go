package doing

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func setupTestErrorLogger(t *testing.T) (*ErrorLogger, string, func()) {
	tmpDir, err := os.MkdirTemp("", "error_logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	logger := &mockLogger{}
	el := NewErrorLogger(logger, tmpDir)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return el, tmpDir, cleanup
}

func TestNewErrorLogger(t *testing.T) {
	logger := &mockLogger{}
	el := NewErrorLogger(logger, "")

	if el == nil {
		t.Fatal("NewErrorLogger returned nil")
	}
	if el.logger == nil {
		t.Error("Logger should not be nil")
	}
	if el.logDir != ".morty/logs" {
		t.Errorf("logDir = %s, want .morty/logs", el.logDir)
	}
}

func TestErrorLogger_LogError(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	testErr := errors.New("test error")
	el.LogError(testErr, "test-module", "test-job", 1, 0)

	if len(el.entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(el.entries))
	}

	entry := el.entries[0]
	if entry.Level != "ERROR" {
		t.Errorf("Level = %s, want ERROR", entry.Level)
	}
	if entry.Module != "test-module" {
		t.Errorf("Module = %s, want test-module", entry.Module)
	}
	if entry.Job != "test-job" {
		t.Errorf("Job = %s, want test-job", entry.Job)
	}
	if entry.LoopCount != 1 {
		t.Errorf("LoopCount = %d, want 1", entry.LoopCount)
	}

	// Verify file was created
	if _, err := os.Stat(el.logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestErrorLogger_LogError_Nil(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	el.LogError(nil, "module", "job", 1, 0)

	if len(el.entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(el.entries))
	}
}

func TestErrorLogger_LogWarning(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	context := map[string]interface{}{"key": "value"}
	el.LogWarning("warning message", "test-module", "test-job", context)

	if len(el.entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(el.entries))
	}

	entry := el.entries[0]
	if entry.Level != "WARNING" {
		t.Errorf("Level = %s, want WARNING", entry.Level)
	}
	if entry.Message != "warning message" {
		t.Errorf("Message = %s, want warning message", entry.Message)
	}
}

func TestErrorLogger_LogRetry(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	testErr := errors.New("transient error")
	el.LogRetry("test-module", "test-job", 2, 3, testErr)

	if len(el.entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(el.entries))
	}

	entry := el.entries[0]
	if entry.Level != "INFO" {
		t.Errorf("Level = %s, want INFO", entry.Level)
	}
	if entry.Category != "Retry" {
		t.Errorf("Category = %s, want Retry", entry.Category)
	}
	if entry.RetryCount != 2 {
		t.Errorf("RetryCount = %d, want 2", entry.RetryCount)
	}
}

func TestErrorLogger_MaxEntries(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	el.maxEntries = 5

	for i := 0; i < 10; i++ {
		el.LogError(errors.New("error"), "module", "job", i, 0)
	}

	if len(el.entries) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(el.entries))
	}
}

func TestErrorLogger_GetRecentErrors(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		el.LogError(errors.New("error"), "module", "job", i, 0)
	}

	recent := el.GetRecentErrors(3)
	if len(recent) != 3 {
		t.Errorf("Expected 3 recent errors, got %d", len(recent))
	}

	// Should be the last 3 entries
	if recent[0].LoopCount != 2 {
		t.Errorf("First recent entry should have LoopCount=2, got %d", recent[0].LoopCount)
	}
}

func TestErrorLogger_GetRecentErrors_All(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	el.LogError(errors.New("error"), "module", "job", 1, 0)

	recent := el.GetRecentErrors(0) // 0 should return all
	if len(recent) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(recent))
	}
}

func TestErrorLogger_GetErrorsByModule(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	el.LogError(errors.New("error1"), "module-a", "job1", 1, 0)
	el.LogError(errors.New("error2"), "module-b", "job1", 1, 0)
	el.LogError(errors.New("error3"), "module-a", "job2", 1, 0)

	moduleAErrors := el.GetErrorsByModule("module-a")
	if len(moduleAErrors) != 2 {
		t.Errorf("Expected 2 errors for module-a, got %d", len(moduleAErrors))
	}
}

func TestErrorLogger_GetErrorsByJob(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	el.LogError(errors.New("error1"), "module-a", "job1", 1, 0)
	el.LogError(errors.New("error2"), "module-a", "job2", 1, 0)
	el.LogError(errors.New("error3"), "module-b", "job1", 1, 0)

	jobErrors := el.GetErrorsByJob("module-a", "job1")
	if len(jobErrors) != 1 {
		t.Errorf("Expected 1 error for module-a/job1, got %d", len(jobErrors))
	}
}

func TestErrorLogger_LoadErrorLog(t *testing.T) {
	el, tmpDir, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	// Create some entries and persist
	el.LogError(errors.New("error1"), "module", "job1", 1, 0)
	el.LogError(errors.New("error2"), "module", "job2", 1, 0)

	// Create new logger pointing to same directory
	el2 := NewErrorLogger(&mockLogger{}, tmpDir)

	err := el2.LoadErrorLog()
	if err != nil {
		t.Errorf("LoadErrorLog failed: %v", err)
	}

	if len(el2.entries) != 2 {
		t.Errorf("Expected 2 entries after load, got %d", len(el2.entries))
	}
}

func TestErrorLogger_LoadErrorLog_NotExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "error_logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	el := NewErrorLogger(&mockLogger{}, tmpDir)

	err = el.LoadErrorLog()
	if err != nil {
		t.Errorf("LoadErrorLog should not fail for non-existent file: %v", err)
	}

	if len(el.entries) != 0 {
		t.Errorf("Expected 0 entries, got %d", len(el.entries))
	}
}

func TestErrorLogger_Clear(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	el.LogError(errors.New("error"), "module", "job", 1, 0)

	if len(el.entries) != 1 {
		t.Fatal("Expected 1 entry before clear")
	}

	el.Clear()

	if len(el.entries) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(el.entries))
	}
}

func TestErrorLogger_FormatErrorReport(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	el.LogError(errors.New("error1"), "test-module", "test-job", 1, 0)
	el.LogError(errors.New("error2"), "test-module", "test-job", 2, 1)

	report := el.FormatErrorReport("test-module", "test-job")

	if report == "" {
		t.Error("Expected non-empty report")
	}
	if report == "No errors recorded for this job." {
		t.Error("Expected error report, got empty message")
	}
}

func TestErrorLogger_FormatErrorReport_NoErrors(t *testing.T) {
	el, _, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	report := el.FormatErrorReport("test-module", "test-job")

	if report != "No errors recorded for this job." {
		t.Errorf("Expected 'No errors recorded' message, got: %s", report)
	}
}

func TestErrorLogger_PersistError(t *testing.T) {
	el, tmpDir, cleanup := setupTestErrorLogger(t)
	defer cleanup()

	// Make directory read-only to cause persist error
	// Skip this test on Windows
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping on Windows")
	}

	// Create a file where directory should be
	fakeDir := filepath.Join(tmpDir, "fakefile")
	os.WriteFile(fakeDir, []byte("content"), 0644)

	el.logDir = filepath.Join(fakeDir, "logs")
	err := el.persist()
	if err == nil {
		t.Error("Expected error when persist cannot create directory")
	}
}
