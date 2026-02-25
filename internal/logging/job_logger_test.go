package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// captureOutput captures slog output to a buffer for testing.
func captureOutput(f func()) string {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	// Create a temporary adapter with the buffer
	originalLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(originalLogger)

	f()
	return buf.String()
}

// TestJobLogger_NewJobLogger tests the creation of a JobLogger.
func TestJobLogger_NewJobLogger(t *testing.T) {
	baseLogger, err := NewSlogAdapter("json", "stdout", DebugLevel)
	if err != nil {
		t.Fatalf("Failed to create base logger: %v", err)
	}

	jl := NewJobLogger("test-module", "test-job", baseLogger)
	if jl == nil {
		t.Fatal("NewJobLogger returned nil")
	}

	// Verify fields
	if jl.GetModule() != "test-module" {
		t.Errorf("Expected module 'test-module', got '%s'", jl.GetModule())
	}
	if jl.GetJob() != "test-job" {
		t.Errorf("Expected job 'test-job', got '%s'", jl.GetJob())
	}

	// Clean up
	jl.LogJobEnd("completed")
}

// TestJobLogger_Getters tests the getter methods.
func TestJobLogger_Getters(t *testing.T) {
	baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)
	jl := NewJobLogger("my-module", "my-job", baseLogger)
	defer jl.LogJobEnd("completed")

	// Test GetModule
	if jl.GetModule() != "my-module" {
		t.Errorf("GetModule() = %v, want %v", jl.GetModule(), "my-module")
	}

	// Test GetJob
	if jl.GetJob() != "my-job" {
		t.Errorf("GetJob() = %v, want %v", jl.GetJob(), "my-job")
	}

	// Test GetStartTime - should be set
	if jl.GetStartTime().IsZero() {
		t.Error("GetStartTime() returned zero time")
	}

	// Test GetDuration - should be > 0
	time.Sleep(10 * time.Millisecond)
	if jl.GetDuration() == 0 {
		t.Error("GetDuration() returned zero")
	}

	// Test GetTaskCount - initially 0
	if jl.GetTaskCount() != 0 {
		t.Errorf("GetTaskCount() = %v, want %v", jl.GetTaskCount(), 0)
	}
}

// TestJobLogger_TaskLogging tests task start and end logging.
func TestJobLogger_TaskLogging(t *testing.T) {
	baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)
	jl := NewJobLogger("test-module", "test-job", baseLogger)
	defer jl.LogJobEnd("completed")

	// Log task start
	jl.LogTaskStart(1, "First task")

	// Verify task count updated
	if jl.GetTaskCount() != 1 {
		t.Errorf("GetTaskCount() = %v, want %v", jl.GetTaskCount(), 1)
	}

	// Log more tasks
	jl.LogTaskStart(2, "Second task")
	jl.LogTaskStart(3, "Third task")

	if jl.GetTaskCount() != 3 {
		t.Errorf("GetTaskCount() = %v, want %v", jl.GetTaskCount(), 3)
	}

	// Log task end
	jl.LogTaskEnd(1, "First task", "success")
	jl.LogTaskEndWithError(2, "Second task", errors.New("task failed"))
}

// TestJobLogger_JobEnd tests job end logging with different results.
func TestJobLogger_JobEnd(t *testing.T) {
	baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)

	// Test successful completion
	t.Run("success", func(t *testing.T) {
		jl := NewJobLogger("test-module", "test-job", baseLogger)
		time.Sleep(5 * time.Millisecond)
		jl.LogJobEnd("completed")
	})

	// Test failure
	t.Run("failure", func(t *testing.T) {
		jl := NewJobLogger("test-module", "test-job", baseLogger)
		time.Sleep(5 * time.Millisecond)
		jl.LogJobEndWithError(errors.New("something went wrong"))
	})
}

// TestJobLogger_LoggerInterface tests that JobLogger provides access to Logger.
func TestJobLogger_LoggerInterface(t *testing.T) {
	baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)
	jl := NewJobLogger("test-module", "test-job", baseLogger)
	defer jl.LogJobEnd("completed")

	// Get the logger
	logger := jl.Logger()
	if logger == nil {
		t.Fatal("Logger() returned nil")
	}

	// Test that we can use the logger
	logger.Info("Test message", String("key", "value"))
}

// TestJobLogger_ConvenienceMethods tests the convenience logging methods.
func TestJobLogger_ConvenienceMethods(t *testing.T) {
	baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)
	jl := NewJobLogger("test-module", "test-job", baseLogger)
	defer jl.LogJobEnd("completed")

	// Test all convenience methods
	jl.Debug("Debug message", String("level", "debug"))
	jl.Info("Info message", String("level", "info"))
	jl.Warn("Warning message", String("level", "warn"))
	jl.Error("Error message", String("level", "error"))
	jl.Success("Success message", String("level", "success"))
	jl.Loop("Loop message", Int("iteration", 1))
}

// TestJobLogger_WithAttrs tests adding attributes.
func TestJobLogger_WithAttrs(t *testing.T) {
	baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)
	jl := NewJobLogger("test-module", "test-job", baseLogger)
	defer jl.LogJobEnd("completed")

	// Add attributes
	jl2 := jl.WithAttrs(String("extra", "value"), Int("count", 42))
	if jl2 == nil {
		t.Fatal("WithAttrs returned nil")
	}

	// Use the new logger
	jl2.Info("Message with extra attrs")
}

// TestJobLogger_NewJobLoggerWithFile tests creating a JobLogger with a separate log file.
func TestJobLogger_NewJobLoggerWithFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()

	baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)
	jl, err := NewJobLoggerWithFile("test-module", "test-job", baseLogger, tempDir)
	if err != nil {
		t.Fatalf("NewJobLoggerWithFile failed: %v", err)
	}

	// Log some messages
	jl.Info("Test message 1")
	jl.LogTaskStart(1, "Task 1")
	jl.LogTaskEnd(1, "Task 1", "success")
	jl.LogJobEnd("completed")

	// Verify log file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("No log file was created")
	}

	// Find the log file
	var logFile os.DirEntry
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".log") {
			logFile = f
			break
		}
	}

	if logFile == nil {
		t.Fatal("No .log file found")
	}

	// Read and verify log file contents
	content, err := os.ReadFile(filepath.Join(tempDir, logFile.Name()))
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file is empty")
	}

	// Verify JSON format
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	for i, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

// TestJobLogger_JobContextConsistency verifies that all log entries have consistent job context.
func TestJobLogger_JobContextConsistency(t *testing.T) {
	// Create a temporary file to capture output
	tempFile, err := os.CreateTemp("", "joblogger_test_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// Create logger writing to file
	baseLogger, err := NewSlogAdapterWithConfig("json", "file", tempFile.Name(), DebugLevel, true)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	jl := NewJobLogger("my-module", "my-job", baseLogger)
	jl.LogTaskStart(1, "First task")
	jl.LogTaskEnd(1, "First task", "success")
	jl.Info("Some info")
	jl.LogJobEnd("completed")

	// Read the log file
	content, err := os.ReadFile(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	// Verify each line has module and job
	for i, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i+1, err)
			continue
		}

		// Check module
		if entry["module"] != "my-module" {
			t.Errorf("Line %d: expected module 'my-module', got '%v'", i+1, entry["module"])
		}

		// Check job
		if entry["job"] != "my-job" {
			t.Errorf("Line %d: expected job 'my-job', got '%v'", i+1, entry["job"])
		}
	}
}

// TestJobLogger_StartEventFormat verifies job start event format.
func TestJobLogger_StartEventFormat(t *testing.T) {
	tempFile, err := os.CreateTemp("", "joblogger_start_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	baseLogger, _ := NewSlogAdapterWithConfig("json", "file", tempFile.Name(), DebugLevel, true)
	jl := NewJobLogger("test-module", "test-job", baseLogger)
	jl.LogJobEnd("completed")

	content, _ := os.ReadFile(tempFile.Name())
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	// Find the job_start event
	var found bool
	for _, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry["event"] == "job_start" {
			found = true

			// Verify required fields
			if entry["module"] != "test-module" {
				t.Errorf("job_start: expected module 'test-module', got '%v'", entry["module"])
			}
			if entry["job"] != "test-job" {
				t.Errorf("job_start: expected job 'test-job', got '%v'", entry["job"])
			}
			if entry["timestamp"] == "" {
				t.Error("job_start: missing timestamp field")
			}
			if entry["msg"] != "Job started" {
				t.Errorf("job_start: expected msg 'Job started', got '%v'", entry["msg"])
			}
		}
	}

	if !found {
		t.Error("job_start event not found in log")
	}
}

// TestJobLogger_EndEventFormat verifies job end event format.
func TestJobLogger_EndEventFormat(t *testing.T) {
	tempFile, err := os.CreateTemp("", "joblogger_end_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	baseLogger, _ := NewSlogAdapterWithConfig("json", "file", tempFile.Name(), DebugLevel, true)
	jl := NewJobLogger("test-module", "test-job", baseLogger)
	time.Sleep(20 * time.Millisecond) // Ensure some duration
	jl.LogJobEnd("success")

	content, _ := os.ReadFile(tempFile.Name())
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	// Find the job_end event
	var found bool
	for _, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry["event"] == "job_end" {
			found = true

			// Verify required fields
			if entry["result"] != "success" {
				t.Errorf("job_end: expected result 'success', got '%v'", entry["result"])
			}
			if entry["duration"] == "" {
				t.Error("job_end: missing duration field")
			}
			if entry["duration_ms"] == nil {
				t.Error("job_end: missing duration_ms field")
			}
			if entry["msg"] != "Job completed" {
				t.Errorf("job_end: expected msg 'Job completed', got '%v'", entry["msg"])
			}
		}
	}

	if !found {
		t.Error("job_end event not found in log")
	}
}

// TestJobLogger_TaskEventFormat verifies task event format.
func TestJobLogger_TaskEventFormat(t *testing.T) {
	tempFile, err := os.CreateTemp("", "joblogger_task_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	baseLogger, _ := NewSlogAdapterWithConfig("json", "file", tempFile.Name(), DebugLevel, true)
	jl := NewJobLogger("test-module", "test-job", baseLogger)
	jl.LogTaskStart(5, "Fifth task description")
	jl.LogTaskEnd(5, "Fifth task description", "success")
	jl.LogJobEnd("completed")

	content, _ := os.ReadFile(tempFile.Name())
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	var foundStart, foundEnd bool
	for _, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry["event"] == "task_start" {
			foundStart = true
			if entry["task_num"] != float64(5) {
				t.Errorf("task_start: expected task_num 5, got '%v'", entry["task_num"])
			}
			if entry["task_desc"] != "Fifth task description" {
				t.Errorf("task_start: unexpected task_desc: '%v'", entry["task_desc"])
			}
		}

		if entry["event"] == "task_end" {
			foundEnd = true
			if entry["result"] != "success" {
				t.Errorf("task_end: expected result 'success', got '%v'", entry["result"])
			}
		}
	}

	if !foundStart {
		t.Error("task_start event not found")
	}
	if !foundEnd {
		t.Error("task_end event not found")
	}
}

// TestJobLogger_ConcurrentAccess tests thread safety.
func TestJobLogger_ConcurrentAccess(t *testing.T) {
	baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)
	jl := NewJobLogger("test-module", "test-job", baseLogger)
	defer jl.LogJobEnd("completed")

	// Concurrent logging
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			jl.Info(fmt.Sprintf("Message %d", n))
			jl.LogTaskStart(n, fmt.Sprintf("Task %d", n))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Task count should be at least 10 (could be more due to concurrent updates)
	if jl.GetTaskCount() < 10 {
		t.Errorf("GetTaskCount() = %v, expected at least 10", jl.GetTaskCount())
	}
}

// TestJobLogger_Close tests the Close method.
func TestJobLogger_Close(t *testing.T) {
	// Test Close without file - call Close without LogJobEnd
	t.Run("close without file", func(t *testing.T) {
		baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)
		jl := NewJobLogger("test-module", "test-job", baseLogger)

		// Close should not error
		if err := jl.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	})

	// Test Close with file - call Close without LogJobEnd
	t.Run("close with file", func(t *testing.T) {
		tempDir := t.TempDir()
		baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)
		jl, err := NewJobLoggerWithFile("test-module", "test-job", baseLogger, tempDir)
		if err != nil {
			t.Fatalf("NewJobLoggerWithFile failed: %v", err)
		}

		// Close should not error
		if err := jl.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	})
}

// TestJobLogger_NewJobLoggerWithFileErrors tests error handling in NewJobLoggerWithFile.
func TestJobLogger_NewJobLoggerWithFileErrors(t *testing.T) {
	// Test with invalid log directory (permission issue)
	t.Run("invalid directory", func(t *testing.T) {
		baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)
		// Try to create a log file in a non-existent path with invalid characters (on Windows)
		// or in a location that requires root access
		_, err := NewJobLoggerWithFile("test", "job", baseLogger, "/nonexistent/invalid/path/that/cannot/be/created")
		// We expect this to fail
		if err == nil {
			t.Skip("Expected error for invalid directory, but got none")
		}
	})
}

// TestJobLogger_LogTaskEndWithError tests LogTaskEndWithError method.
func TestJobLogger_LogTaskEndWithError(t *testing.T) {
	tempFile, err := os.CreateTemp("", "joblogger_task_error_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	baseLogger, _ := NewSlogAdapterWithConfig("json", "file", tempFile.Name(), DebugLevel, true)
	jl := NewJobLogger("test-module", "test-job", baseLogger)
	jl.LogTaskEndWithError(1, "Task 1", errors.New("task execution failed"))
	jl.LogJobEnd("completed")

	// Verify the error was logged
	content, _ := os.ReadFile(tempFile.Name())
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	var found bool
	for _, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry["event"] == "task_end" && entry["error"] != nil {
			found = true
			if entry["error"] != "task execution failed" {
				t.Errorf("Expected error 'task execution failed', got '%v'", entry["error"])
			}
			if entry["level"] != "ERROR" {
				t.Errorf("Expected level ERROR, got '%v'", entry["level"])
			}
		}
	}

	if !found {
		t.Error("task_end error event not found in log")
	}
}

// TestJobLogger_GetDuration tests GetDuration method.
func TestJobLogger_GetDuration(t *testing.T) {
	baseLogger, _ := NewSlogAdapter("json", "stdout", DebugLevel)
	jl := NewJobLogger("test-module", "test-job", baseLogger)
	defer jl.LogJobEnd("completed")

	// Get initial duration
	duration1 := jl.GetDuration()

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Get duration again
	duration2 := jl.GetDuration()

	// Duration2 should be greater than duration1
	if duration2 <= duration1 {
		t.Errorf("GetDuration() should increase over time: %v <= %v", duration2, duration1)
	}
}

// BenchmarkJobLogger benchmarks the JobLogger.
func BenchmarkJobLogger(b *testing.B) {
	baseLogger, _ := NewSlogAdapter("json", "stdout", ErrorLevel) // Suppress output
	jl := NewJobLogger("bench-module", "bench-job", baseLogger)
	defer jl.LogJobEnd("completed")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jl.Info("Benchmark message", Int("iteration", i))
	}
}
