// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/morty/morty/pkg/errors"
)

func TestNew(t *testing.T) {
	caller := New()
	if caller == nil {
		t.Fatal("New() returned nil")
	}
	if caller.GetDefaultTimeout() != 0 {
		t.Errorf("Expected default timeout 0, got %v", caller.GetDefaultTimeout())
	}
}

func TestNewWithTimeout(t *testing.T) {
	timeout := 5 * time.Second
	caller := NewWithTimeout(timeout)
	if caller == nil {
		t.Fatal("NewWithTimeout() returned nil")
	}
	if caller.GetDefaultTimeout() != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, caller.GetDefaultTimeout())
	}
}

func TestSetDefaultTimeout(t *testing.T) {
	caller := New()
	timeout := 10 * time.Second
	caller.SetDefaultTimeout(timeout)
	if caller.GetDefaultTimeout() != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, caller.GetDefaultTimeout())
	}
}

func TestCall_SimpleCommand(t *testing.T) {
	caller := New()
	ctx := context.Background()

	// Use echo command which works on both Unix and Windows
	result, err := caller.Call(ctx, "echo", "hello")
	if err != nil {
		t.Fatalf("Call() failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// On Windows, echo adds a space, on Unix it doesn't
	// So we check for "hello" in the output
	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("Expected stdout to contain 'hello', got '%s'", result.Stdout)
	}

	if result.Stderr != "" {
		t.Errorf("Expected empty stderr, got '%s'", result.Stderr)
	}

	if result.Duration < 0 {
		t.Errorf("Expected non-negative duration, got %v", result.Duration)
	}

	if result.Command != "echo hello" {
		t.Errorf("Expected command 'echo hello', got '%s'", result.Command)
	}
}

func TestCall_CommandNotFound(t *testing.T) {
	caller := New()
	ctx := context.Background()

	_, err := caller.Call(ctx, "nonexistent_command_xyz_12345")
	if err == nil {
		t.Fatal("Expected error for non-existent command")
	}

	// Check that it's a MortyError with the right code
	if mortyErr, ok := errors.AsMortyError(err); ok {
		if mortyErr.Code != "M5001" {
			t.Errorf("Expected error code M5001, got %s", mortyErr.Code)
		}
	} else {
		t.Error("Expected MortyError")
	}
}

func TestCall_ExitCode(t *testing.T) {
	caller := New()
	ctx := context.Background()

	var result *Result
	var err error

	// Use a command that returns non-zero exit code
	if runtime.GOOS == "windows" {
		result, err = caller.Call(ctx, "cmd", "/c", "exit", "42")
	} else {
		result, err = caller.Call(ctx, "sh", "-c", "exit 42")
	}

	if err == nil {
		t.Fatal("Expected error for non-zero exit code")
	}

	if result == nil {
		t.Fatal("Expected result even on error")
	}

	if result.ExitCode != 42 {
		t.Errorf("Expected exit code 42, got %d", result.ExitCode)
	}

	// Check that it's a MortyError with the right code
	if mortyErr, ok := errors.AsMortyError(err); ok {
		if mortyErr.Code != "M5002" {
			t.Errorf("Expected error code M5002, got %s", mortyErr.Code)
		}
	} else {
		t.Error("Expected MortyError")
	}
}

func TestCall_StdoutCapture(t *testing.T) {
	caller := New()
	ctx := context.Background()

	var result *Result
	if runtime.GOOS == "windows" {
		result, _ = caller.Call(ctx, "cmd", "/c", "echo stdout_test")
	} else {
		result, _ = caller.Call(ctx, "sh", "-c", "echo stdout_test")
	}

	if !strings.Contains(result.Stdout, "stdout_test") {
		t.Errorf("Expected stdout to contain 'stdout_test', got '%s'", result.Stdout)
	}
}

func TestCall_StderrCapture(t *testing.T) {
	caller := New()
	ctx := context.Background()

	var result *Result
	var err error

	// Write to stderr and exit with non-zero
	if runtime.GOOS == "windows" {
		result, err = caller.Call(ctx, "cmd", "/c", "echo stderr_test>&2 && exit 1")
	} else {
		result, err = caller.Call(ctx, "sh", "-c", "echo stderr_test >&2; exit 1")
	}

	if err == nil {
		t.Fatal("Expected error")
	}

	if !strings.Contains(result.Stderr, "stderr_test") {
		t.Errorf("Expected stderr to contain 'stderr_test', got '%s'", result.Stderr)
	}
}

func TestCall_WorkingDir(t *testing.T) {
	caller := New()
	ctx := context.Background()

	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a test file in the temp directory
	testFile := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	opts := Options{
		WorkingDir: tempDir,
	}

	var result *Result
	var err error

	// List files in the working directory
	if runtime.GOOS == "windows" {
		result, err = caller.CallWithOptions(ctx, "cmd", []string{"/c", "dir"}, opts)
	} else {
		result, err = caller.CallWithOptions(ctx, "ls", []string{"-la"}, opts)
	}

	if err != nil {
		t.Fatalf("CallWithOptions() failed: %v", err)
	}

	// Should be able to see the test file
	if !strings.Contains(result.Stdout, "testfile.txt") {
		t.Errorf("Expected stdout to contain 'testfile.txt', got '%s'", result.Stdout)
	}
}

func TestCall_EnvironmentVariables(t *testing.T) {
	caller := New()
	ctx := context.Background()

	opts := Options{
		Env: map[string]string{
			"MORTY_TEST_VAR": "test_value_123",
		},
	}

	var result *Result
	var err error

	// Print the environment variable
	if runtime.GOOS == "windows" {
		result, err = caller.CallWithOptions(ctx, "cmd", []string{"/c", "echo %MORTY_TEST_VAR%"}, opts)
	} else {
		result, err = caller.CallWithOptions(ctx, "sh", []string{"-c", "echo $MORTY_TEST_VAR"}, opts)
	}

	if err != nil {
		t.Fatalf("CallWithOptions() failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "test_value_123") {
		t.Errorf("Expected stdout to contain 'test_value_123', got '%s'", result.Stdout)
	}
}

func TestCall_Timeout(t *testing.T) {
	caller := New()
	ctx := context.Background()

	opts := Options{
		Timeout: 100 * time.Millisecond,
	}

	var err error

	// Run a command that takes longer than the timeout
	if runtime.GOOS == "windows" {
		_, err = caller.CallWithOptions(ctx, "ping", []string{"-n", "10", "localhost"}, opts)
	} else {
		_, err = caller.CallWithOptions(ctx, "sleep", []string{"10"}, opts)
	}

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	// Check that it's a timeout error
	if mortyErr, ok := errors.AsMortyError(err); ok {
		if mortyErr.Code != "M5003" {
			t.Errorf("Expected error code M5003 for timeout, got %s", mortyErr.Code)
		}
	} else {
		t.Error("Expected MortyError for timeout")
	}
}

func TestCall_DefaultTimeout(t *testing.T) {
	caller := NewWithTimeout(100 * time.Millisecond)
	ctx := context.Background()

	var err error

	// Run a command that takes longer than the timeout
	if runtime.GOOS == "windows" {
		_, err = caller.Call(ctx, "ping", "-n", "10", "localhost")
	} else {
		_, err = caller.Call(ctx, "sleep", "10")
	}

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	// Check that it's a timeout error
	if mortyErr, ok := errors.AsMortyError(err); ok {
		if mortyErr.Code != "M5003" {
			t.Errorf("Expected error code M5003 for timeout, got %s", mortyErr.Code)
		}
	}
}

func TestCall_Stdin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping stdin test on Windows")
	}

	caller := New()
	ctx := context.Background()

	opts := Options{
		Stdin: "hello from stdin",
	}

	result, err := caller.CallWithOptions(ctx, "cat", []string{}, opts)
	if err != nil {
		t.Fatalf("CallWithOptions() failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "hello from stdin") {
		t.Errorf("Expected stdout to contain 'hello from stdin', got '%s'", result.Stdout)
	}
}

func TestCall_CancelledContext(t *testing.T) {
	caller := New()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately
	cancel()

	_, err := caller.Call(ctx, "echo", "hello")
	if err == nil {
		t.Fatal("Expected error for cancelled context")
	}

	// Check that it's a MortyError
	if mortyErr, ok := errors.AsMortyError(err); ok {
		if mortyErr.Code != "M5007" {
			t.Errorf("Expected error code M5007, got %s", mortyErr.Code)
		}
	} else {
		t.Error("Expected MortyError")
	}
}

func TestCall_MultipleArgs(t *testing.T) {
	caller := New()
	ctx := context.Background()

	// Test with multiple arguments
	result, err := caller.Call(ctx, "echo", "arg1", "arg2", "arg3")
	if err != nil {
		t.Fatalf("Call() failed: %v", err)
	}

	// All arguments should be in output
	if !strings.Contains(result.Stdout, "arg1") {
		t.Errorf("Expected stdout to contain 'arg1', got '%s'", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "arg2") {
		t.Errorf("Expected stdout to contain 'arg2', got '%s'", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "arg3") {
		t.Errorf("Expected stdout to contain 'arg3', got '%s'", result.Stdout)
	}
}

func TestCallWithOptions_EmptyArgs(t *testing.T) {
	caller := New()
	ctx := context.Background()

	// Test with empty args slice
	result, err := caller.CallWithOptions(ctx, "echo", []string{}, Options{})
	if err != nil {
		t.Fatalf("CallWithOptions() failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}

func TestResult_Struct(t *testing.T) {
	result := &Result{
		Stdout:   "stdout content",
		Stderr:   "stderr content",
		ExitCode: 0,
		Duration: time.Second,
		Command:  "test command",
	}

	if result.Stdout != "stdout content" {
		t.Errorf("Expected stdout 'stdout content', got '%s'", result.Stdout)
	}
	if result.Stderr != "stderr content" {
		t.Errorf("Expected stderr 'stderr content', got '%s'", result.Stderr)
	}
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	if result.Duration != time.Second {
		t.Errorf("Expected duration 1s, got %v", result.Duration)
	}
	if result.Command != "test command" {
		t.Errorf("Expected command 'test command', got '%s'", result.Command)
	}
}

func TestOptions_Struct(t *testing.T) {
	opts := Options{
		WorkingDir: "/tmp",
		Env: map[string]string{
			"KEY": "value",
		},
		Timeout: time.Minute,
		Stdin:   "input",
	}

	if opts.WorkingDir != "/tmp" {
		t.Errorf("Expected working dir '/tmp', got '%s'", opts.WorkingDir)
	}
	if opts.Env["KEY"] != "value" {
		t.Errorf("Expected env KEY='value', got '%s'", opts.Env["KEY"])
	}
	if opts.Timeout != time.Minute {
		t.Errorf("Expected timeout 1m, got %v", opts.Timeout)
	}
	if opts.Stdin != "input" {
		t.Errorf("Expected stdin 'input', got '%s'", opts.Stdin)
	}
}

func TestCall_PathResolution(t *testing.T) {
	caller := New()
	ctx := context.Background()

	// Test that we can find common commands in PATH
	commands := []string{"echo", "cat", "ls"}
	if runtime.GOOS == "windows" {
		commands = []string{"echo", "dir", "type"}
	}

	for _, cmd := range commands {
		// Just check that it doesn't return "not found" error
		// Some commands might not support --version, but that's ok
		_, err := caller.Call(ctx, cmd)
		if err != nil {
			// If error is "not found", that's a failure
			if mortyErr, ok := errors.AsMortyError(err); ok {
				if mortyErr.Code == "M5001" {
					t.Errorf("Command '%s' not found in PATH", cmd)
				}
			}
		}
	}
}

func TestCall_DurationTracking(t *testing.T) {
	caller := New()
	ctx := context.Background()

	start := time.Now()
	result, err := caller.Call(ctx, "sleep", "0.1")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Call() failed: %v", err)
	}

	// Duration should be at least 100ms
	if result.Duration < 100*time.Millisecond {
		t.Errorf("Expected duration >= 100ms, got %v", result.Duration)
	}

	// Duration should be reasonable (not more than actual elapsed + margin)
	if result.Duration > elapsed+50*time.Millisecond {
		t.Errorf("Duration %v seems too high compared to elapsed %v", result.Duration, elapsed)
	}
}

func TestBuildCommandString(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"echo", []string{}, "echo"},
		{"echo", []string{"hello"}, "echo hello"},
		{"echo", []string{"hello", "world"}, "echo hello world"},
		{"ls", []string{"-la", "/tmp"}, "ls -la /tmp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCommandString(tt.name, tt.args)
			if result != tt.expected {
				t.Errorf("buildCommandString(%q, %v) = %q, expected %q",
					tt.name, tt.args, result, tt.expected)
			}
		})
	}
}

// Benchmark test for performance testing
func BenchmarkCall_Echo(b *testing.B) {
	caller := New()
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		_, err := caller.Call(ctx, "echo", "hello")
		if err != nil {
			b.Fatalf("Call() failed: %v", err)
		}
	}
}

func TestExecutionLogSimple(t *testing.T) {
	t.Log("Test running!")
}


func TestExecutionLog(t *testing.T) {
	t.Run("Struct creation", func(t *testing.T) {
		log := &ExecutionLog{
			ID:              "test_id",
			Command:         "echo",
			Args:            []string{"hello"},
			FullCommand:     "echo hello",
			ExitCode:        0,
			Success:         true,
			StdoutSize:      5,
			StderrSize:      0,
			TotalOutputSize: 5,
		}
		if log.Command != "echo" {
			t.Errorf("Expected 'echo', got '%s'", log.Command)
		}
	})

	t.Run("FromResult success", func(t *testing.T) {
		result := &Result{
			Stdout:   "output",
			Stderr:   "",
			ExitCode: 0,
			Duration: 100 * time.Millisecond,
		}
		log := NewExecutionLogFromResult(result, "echo", []string{"test"}, "/tmp", 0)
		if !log.Success {
			t.Error("Expected success")
		}
		if log.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", log.ExitCode)
		}
	})

	t.Run("FromResult error", func(t *testing.T) {
		result := &Result{
			Stdout:   "",
			Stderr:   "error",
			ExitCode: 1,
			Duration: 50 * time.Millisecond,
		}
		log := NewExecutionLogFromResult(result, "cmd", []string{}, "/tmp", 0)
		if log.Success {
			t.Error("Expected failure")
		}
	})

	t.Run("FromResult timeout", func(t *testing.T) {
		result := &Result{
			ExitCode: -1,
			Duration: 5 * time.Second,
			TimedOut: true,
		}
		log := NewExecutionLogFromResult(result, "sleep", []string{"10"}, "/tmp", 5*time.Second)
		if !log.TimedOut {
			t.Error("Expected TimedOut to be true")
		}
	})
}

func TestExecutionLogger(t *testing.T) {
	t.Run("Creates directory", func(t *testing.T) {
		tmpDir := filepath.Join(t.TempDir(), "logs", "subdir")
		logger, err := NewExecutionLogger(tmpDir, 0, 0, 0)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()
		if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
			t.Error("Log directory was not created")
		}
	})

	t.Run("Writes JSON log", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, err := NewExecutionLogger(tmpDir, 0, 0, 0)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		log := &ExecutionLog{
			ID:         "test_001",
			Command:    "echo",
			ExitCode:   0,
			Success:    true,
			StdoutSize: 4,
		}
		if err := logger.LogExecution(log); err != nil {
			t.Fatalf("Failed to log: %v", err)
		}

		files, _ := filepath.Glob(filepath.Join(tmpDir, "execution_*.log"))
		if len(files) == 0 {
			t.Fatal("No log file created")
		}
	})

	t.Run("Closed logger fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, _ := NewExecutionLogger(tmpDir, 0, 0, 0)
		logger.Close()
		log := &ExecutionLog{ID: "test", Command: "echo"}
		if err := logger.LogExecution(log); err == nil {
			t.Error("Expected error when logging to closed logger")
		}
	})

	t.Run("Multiple entries", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, _ := NewExecutionLogger(tmpDir, 0, 0, 0)
		defer logger.Close()

		for i := 0; i < 5; i++ {
			log := &ExecutionLog{
				ID:       fmt.Sprintf("test_%d", i),
				Command:  "echo",
				ExitCode: i % 2,
				Success:  i%2 == 0,
			}
			if err := logger.LogExecution(log); err != nil {
				t.Fatalf("Failed to log: %v", err)
			}
		}

		logs, _ := ReadLogs(tmpDir)
		if len(logs) != 5 {
			t.Errorf("Expected 5 logs, got %d", len(logs))
		}
	})
}

func TestLogRotation(t *testing.T) {
	t.Run("Rotation by size", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, _ := NewExecutionLogger(tmpDir, 500, 5, 0)
		defer logger.Close()

		for i := 0; i < 20; i++ {
			log := &ExecutionLog{
				ID:          fmt.Sprintf("rot_%d", i),
				Command:     "echo",
				FullCommand: "echo " + strings.Repeat("x", 50),
				StdoutSize:  50,
				Success:     true,
			}
			logger.LogExecution(log)
		}

		files, _ := filepath.Glob(filepath.Join(tmpDir, "execution_*.log"))
		if len(files) < 1 {
			t.Error("Expected at least one log file")
		}
	})
}

func TestExecutionStats(t *testing.T) {
	t.Run("Stats calculation", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, _ := NewExecutionLogger(tmpDir, 0, 0, 0)
		defer logger.Close()

		executions := []struct {
			exitCode int
			success  bool
		}{
			{0, true},
			{0, true},
			{1, false},
			{0, true},
			{127, false},
		}

		for i, exec := range executions {
			log := &ExecutionLog{
				ID:       fmt.Sprintf("stat_%d", i),
				Command:  "cmd",
				ExitCode: exec.exitCode,
				Success:  exec.success,
				Duration: 100 * time.Millisecond,
			}
			logger.LogExecution(log)
		}

		stats := logger.GetStats()
		if stats.TotalExecutions != 5 {
			t.Errorf("Expected 5 executions, got %d", stats.TotalExecutions)
		}
		if stats.SuccessfulExecutions != 3 {
			t.Errorf("Expected 3 successful, got %d", stats.SuccessfulExecutions)
		}
		if stats.FailedExecutions != 2 {
			t.Errorf("Expected 2 failed, got %d", stats.FailedExecutions)
		}
	})

	t.Run("Success rate", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, _ := NewExecutionLogger(tmpDir, 0, 0, 0)
		defer logger.Close()

		logger.LogExecution(&ExecutionLog{ID: "1", Command: "echo", ExitCode: 0, Success: true})
		logger.LogExecution(&ExecutionLog{ID: "2", Command: "echo", ExitCode: 0, Success: true})
		logger.LogExecution(&ExecutionLog{ID: "3", Command: "fail", ExitCode: 1, Success: false})

		stats := logger.GetStats()
		rate := stats.GetSuccessRate()
		if rate != 66.66666666666667 {
			t.Errorf("Expected 66.67%% success rate, got %f", rate)
		}
	})

	t.Run("Command stats", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, _ := NewExecutionLogger(tmpDir, 0, 0, 0)
		defer logger.Close()

		for i := 0; i < 3; i++ {
			logger.LogExecution(&ExecutionLog{ID: fmt.Sprintf("e%d", i), Command: "echo", Success: true})
		}
		for i := 0; i < 2; i++ {
			logger.LogExecution(&ExecutionLog{ID: fmt.Sprintf("l%d", i), Command: "ls", Success: true})
		}

		stats := logger.GetStats()
		if stats.CommandStats["echo"].TotalExecutions != 3 {
			t.Errorf("Expected 3 echo executions, got %d", stats.CommandStats["echo"].TotalExecutions)
		}
		if stats.CommandStats["ls"].TotalExecutions != 2 {
			t.Errorf("Expected 2 ls executions, got %d", stats.CommandStats["ls"].TotalExecutions)
		}
	})
}

func TestReadLogs(t *testing.T) {
	t.Run("Read logs from directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, _ := NewExecutionLogger(tmpDir, 0, 0, 0)

		logger.LogExecution(&ExecutionLog{ID: "r1", Command: "echo", ExitCode: 0, Success: true})
		logger.LogExecution(&ExecutionLog{ID: "r2", Command: "ls", ExitCode: 1, Success: false})
		logger.Close()

		logs, err := ReadLogs(tmpDir)
		if err != nil {
			t.Fatalf("Failed to read logs: %v", err)
		}
		if len(logs) != 2 {
			t.Errorf("Expected 2 logs, got %d", len(logs))
		}
	})

	t.Run("Empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		logs, err := ReadLogs(tmpDir)
		if err != nil {
			t.Fatalf("Failed: %v", err)
		}
		if len(logs) != 0 {
			t.Errorf("Expected 0 logs, got %d", len(logs))
		}
	})
}

func TestExecutionLogConcurrency(t *testing.T) {
	t.Run("Concurrent writes", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, _ := NewExecutionLogger(tmpDir, 0, 0, 0)
		defer logger.Close()

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					log := &ExecutionLog{
						ID:      fmt.Sprintf("c%d_%d", id, j),
						Command: "echo",
						Success: true,
					}
					logger.LogExecution(log)
				}
			}(i)
		}
		wg.Wait()

		stats := logger.GetStats()
		if stats.TotalExecutions != 100 {
			t.Errorf("Expected 100 executions, got %d", stats.TotalExecutions)
		}
	})
}

func TestExecutionLogIntegration(t *testing.T) {
	t.Run("Full workflow", func(t *testing.T) {
		tmpDir := t.TempDir()
		logger, _ := NewExecutionLogger(tmpDir, 1024*1024, 5, 7)

		commands := []struct {
			result *Result
			name   string
		}{
			{&Result{Stdout: "out1", ExitCode: 0, Duration: 10 * time.Millisecond}, "echo"},
			{&Result{Stdout: "out2", ExitCode: 0, Duration: 20 * time.Millisecond}, "ls"},
			{&Result{Stderr: "err", ExitCode: 1, Duration: 5 * time.Millisecond}, "cat"},
		}

		for _, cmd := range commands {
			log := NewExecutionLogFromResult(cmd.result, cmd.name, []string{}, "/tmp", 0)
			logger.LogExecution(log)
		}

		stats := logger.GetStats()
		if stats.TotalExecutions != 3 {
			t.Errorf("Expected 3 executions, got %d", stats.TotalExecutions)
		}
		if stats.SuccessfulExecutions != 2 {
			t.Errorf("Expected 2 successful, got %d", stats.SuccessfulExecutions)
		}
		if stats.FailedExecutions != 1 {
			t.Errorf("Expected 1 failed, got %d", stats.FailedExecutions)
		}

		logger.Close()

		logs, _ := ReadLogs(tmpDir)
		if len(logs) != 3 {
			t.Errorf("Expected 3 logs, got %d", len(logs))
		}
	})
}
