package callcli

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/morty/morty/pkg/errors"
)

// getSleepCommand returns the appropriate sleep command for the platform
func getSleepCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "ping", []string{"-n", "2", "127.0.0.1"}
	}
	return "sleep", []string{"0.1"}
}

// getLongSleepCommand returns a command that takes longer to complete
func getLongSleepCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "ping", []string{"-n", "10", "127.0.0.1"}
	}
	return "sleep", []string{"10"}
}

// TestCallAsync_SimpleCommand tests basic async command execution
func TestCallAsync_SimpleCommand(t *testing.T) {
	caller := New()
	ctx := context.Background()

	handler, err := caller.CallAsync(ctx, "echo", "hello")
	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout != "hello" {
		t.Errorf("Expected stdout 'hello', got '%s'", result.Stdout)
	}
}

// TestCallAsync_ReturnsHandlerImmediately tests that async returns handler without waiting
func TestCallAsync_ReturnsHandlerImmediately(t *testing.T) {
	caller := New()
	ctx := context.Background()

	name, args := getSleepCommand()
	start := time.Now()
	handler, err := caller.CallAsync(ctx, name, args...)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	// Should return immediately (much faster than the sleep duration)
	if elapsed > 100*time.Millisecond {
		t.Errorf("CallAsync took too long: %v, expected immediate return", elapsed)
	}

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}

	// Wait for completion
	_, err = handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
}

// TestCallHandler_Wait tests the Wait method
func TestCallHandler_Wait(t *testing.T) {
	caller := New()
	ctx := context.Background()

	handler, err := caller.CallAsync(ctx, "echo", "test output")
	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.Stdout != "test output" {
		t.Errorf("Expected stdout 'test output', got '%s'", result.Stdout)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
}

// TestCallHandler_Kill tests the Kill method
func TestCallHandler_Kill(t *testing.T) {
	caller := New()
	ctx := context.Background()

	name, args := getLongSleepCommand()
	handler, err := caller.CallAsync(ctx, name, args...)
	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	// Give process time to start
	time.Sleep(50 * time.Millisecond)

	// Verify it's running
	if !handler.Running() {
		t.Skip("Process not running, skipping Kill test")
	}

	// Kill the process
	err = handler.Kill()
	if err != nil {
		t.Fatalf("Kill failed: %v", err)
	}

	// Wait for result
	result, err := handler.Wait()

	// Process should have been killed
	if runtime.GOOS != "windows" {
		// On Unix, we expect a signal error
		if err == nil {
			t.Error("Expected error when process is killed")
		}
	}

	if result == nil {
		t.Fatal("Expected non-nil result after kill")
	}
}

// TestCallHandler_Running tests the Running method
func TestCallHandler_Running(t *testing.T) {
	caller := New()
	ctx := context.Background()

	name, args := getSleepCommand()
	handler, err := caller.CallAsync(ctx, name, args...)
	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	// Should be running initially
	time.Sleep(10 * time.Millisecond)

	// Wait for completion
	_, _ = handler.Wait()

	// Should not be running after completion
	if handler.Running() {
		t.Error("Expected Running() to return false after completion")
	}
}

// TestCallHandler_PID tests the PID method
func TestCallHandler_PID(t *testing.T) {
	caller := New()
	ctx := context.Background()

	handler, err := caller.CallAsync(ctx, "echo", "test")
	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	pid := handler.PID()
	if pid <= 0 {
		t.Errorf("Expected positive PID, got %d", pid)
	}

	_, _ = handler.Wait()
}

// TestCallAsyncWithOptions tests async with options
func TestCallAsyncWithOptions(t *testing.T) {
	caller := New()
	ctx := context.Background()

	opts := Options{
		WorkingDir: t.TempDir(),
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	var handler CallHandler
	var err error

	if runtime.GOOS == "windows" {
		handler, err = caller.CallAsyncWithOptions(ctx, "cmd", []string{"/C", "echo %TEST_VAR%"}, opts)
	} else {
		handler, err = caller.CallAsyncWithOptions(ctx, "sh", []string{"-c", "echo $TEST_VAR"}, opts)
	}

	if err != nil {
		t.Fatalf("CallAsyncWithOptions failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "test_value") {
		t.Errorf("Expected stdout to contain 'test_value', got '%s'", result.Stdout)
	}
}

// TestCallAsync_CommandNotFound tests error handling for missing command
func TestCallAsync_CommandNotFound(t *testing.T) {
	caller := New()
	ctx := context.Background()

	_, err := caller.CallAsync(ctx, "nonexistent_command_xyz")
	if err == nil {
		t.Fatal("Expected error for nonexistent command")
	}

	appErr, ok := err.(*errors.MortyError)
	if !ok {
		t.Fatalf("Expected errors.Error type, got %T", err)
	}

	if appErr.Code != "M5001" {
		t.Errorf("Expected error code M5001, got %s", appErr.Code)
	}
}

// TestCallAsync_ExitCode tests that exit codes are captured correctly
func TestCallAsync_ExitCode(t *testing.T) {
	caller := New()
	ctx := context.Background()

	var handler CallHandler
	var err error

	if runtime.GOOS == "windows" {
		handler, err = caller.CallAsync(ctx, "cmd", "/C", "exit", "42")
	} else {
		handler, err = caller.CallAsync(ctx, "sh", "-c", "exit 42")
	}

	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	result, err := handler.Wait()
	if err == nil {
		t.Fatal("Expected error for non-zero exit")
	}

	if result.ExitCode != 42 {
		t.Errorf("Expected exit code 42, got %d", result.ExitCode)
	}
}

// TestCallAsync_StdoutCapture tests stdout capture
func TestCallAsync_StdoutCapture(t *testing.T) {
	caller := New()
	ctx := context.Background()

	handler, err := caller.CallAsync(ctx, "echo", "captured output")
	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.Stdout != "captured output" {
		t.Errorf("Expected stdout 'captured output', got '%s'", result.Stdout)
	}
}

// TestCallAsync_StderrCapture tests stderr capture
func TestCallAsync_StderrCapture(t *testing.T) {
	caller := New()
	ctx := context.Background()

	var handler CallHandler
	var err error

	if runtime.GOOS == "windows" {
		handler, err = caller.CallAsync(ctx, "cmd", "/C", "echo error message >&2")
	} else {
		handler, err = caller.CallAsync(ctx, "sh", "-c", "echo error message >&2")
	}

	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if !strings.Contains(result.Stderr, "error message") {
		t.Errorf("Expected stderr to contain 'error message', got '%s'", result.Stderr)
	}
}

// TestCallAsync_MultipleArgs tests multiple argument handling
func TestCallAsync_MultipleArgs(t *testing.T) {
	caller := New()
	ctx := context.Background()

	handler, err := caller.CallAsync(ctx, "echo", "arg1", "arg2", "arg3")
	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	expected := "arg1 arg2 arg3"
	if result.Stdout != expected {
		t.Errorf("Expected stdout '%s', got '%s'", expected, result.Stdout)
	}
}

// TestCallAsync_CancelledContext tests cancelled context handling
func TestCallAsync_CancelledContext(t *testing.T) {
	caller := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := caller.CallAsync(ctx, "echo", "test")
	if err == nil {
		t.Fatal("Expected error for cancelled context")
	}

	appErr, ok := err.(*errors.MortyError)
	if !ok {
		t.Fatalf("Expected errors.Error type, got %T", err)
	}

	if appErr.Code != "M5007" {
		t.Errorf("Expected error code M5007, got %s", appErr.Code)
	}
}

// TestCallAsync_Timeout tests timeout handling
func TestCallAsync_Timeout(t *testing.T) {
	caller := New()
	ctx := context.Background()

	opts := Options{
		Timeout: 50 * time.Millisecond,
	}

	name, args := getLongSleepCommand()
	handler, err := caller.CallAsyncWithOptions(ctx, name, args, opts)
	if err != nil {
		t.Fatalf("CallAsyncWithOptions failed: %v", err)
	}

	result, err := handler.Wait()

	// Should have timed out
	if err == nil {
		t.Error("Expected timeout error")
	}

	if result.ExitCode != -1 {
		t.Errorf("Expected exit code -1 for timeout, got %d", result.ExitCode)
	}

	if err != nil {
		appErr, ok := err.(*errors.MortyError)
		if ok && appErr.Code != "M5003" {
			t.Errorf("Expected error code M5003 for timeout, got %s", appErr.Code)
		}
	}
}

// TestCallAsync_DurationTracking tests duration measurement
func TestCallAsync_DurationTracking(t *testing.T) {
	caller := New()
	ctx := context.Background()

	name, args := getSleepCommand()
	start := time.Now()
	handler, err := caller.CallAsync(ctx, name, args...)
	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	result, err := handler.Wait()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.Duration <= 0 {
		t.Error("Expected positive duration")
	}

	// Duration should be close to elapsed time
	if result.Duration > elapsed+50*time.Millisecond {
		t.Errorf("Duration %v is too different from elapsed %v", result.Duration, elapsed)
	}
}

// TestCallAsyncWithOptions_Stdin tests stdin input
func TestCallAsyncWithOptions_Stdin(t *testing.T) {
	caller := New()
	ctx := context.Background()

	opts := Options{
		Stdin: "input data",
	}

	var handler CallHandler
	var err error

	if runtime.GOOS == "windows" {
		handler, err = caller.CallAsyncWithOptions(ctx, "findstr", []string{"input"}, opts)
	} else {
		handler, err = caller.CallAsyncWithOptions(ctx, "cat", []string{}, opts)
	}

	if err != nil {
		t.Fatalf("CallAsyncWithOptions failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "input") {
		t.Errorf("Expected stdout to contain 'input', got '%s'", result.Stdout)
	}
}

// BenchmarkCallAsync benchmarks async execution
func BenchmarkCallAsync_Echo(b *testing.B) {
	caller := New()
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		handler, err := caller.CallAsync(ctx, "echo", "benchmark")
		if err != nil {
			b.Fatalf("CallAsync failed: %v", err)
		}
		_, _ = handler.Wait()
	}
}
