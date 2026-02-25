// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
