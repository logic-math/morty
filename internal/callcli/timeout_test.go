// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/morty/morty/pkg/errors"
)

// TestCallWithCtx_SimpleCommand tests basic execution with CallWithCtx
func TestCallWithCtx_SimpleCommand(t *testing.T) {
	caller := New()
	ctx := context.Background()

	result, err := caller.CallWithCtx(ctx, "echo", []string{"hello"}, Options{})
	if err != nil {
		t.Fatalf("CallWithCtx() failed: %v", err)
	}

	// Wait for completion
	res, err := result.Wait()
	if err != nil {
		t.Fatalf("Wait() failed: %v", err)
	}

	if res.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", res.ExitCode)
	}

	if res.Stdout != "hello" {
		t.Errorf("Expected stdout 'hello', got '%s'", res.Stdout)
	}

	if res.TimedOut {
		t.Error("Expected TimedOut to be false")
	}
}

// TestCallWithCtx_Timeout tests timeout functionality
func TestCallWithCtx_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping timeout test on Windows")
	}

	caller := New()
	ctx := context.Background()

	opts := Options{
		Timeout: 100 * time.Millisecond,
	}

	result, err := caller.CallWithCtx(ctx, "sleep", []string{"5"}, opts)
	if err != nil {
		t.Fatalf("CallWithCtx() failed: %v", err)
	}

	// Wait for completion
	res, err := result.Wait()

	// Should have timed out
	if !res.TimedOut {
		t.Error("Expected TimedOut to be true")
	}

	if res.ExitCode != -1 {
		t.Errorf("Expected exit code -1 for timeout, got %d", res.ExitCode)
	}

	// Check error code
	if mortyErr, ok := errors.AsMortyError(err); ok {
		if mortyErr.Code != "M5003" {
			t.Errorf("Expected error code M5003, got %s", mortyErr.Code)
		}
	} else {
		t.Error("Expected MortyError for timeout")
	}
}

// TestCallWithCtx_ContextCancel tests context cancellation
func TestCallWithCtx_ContextCancel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping context cancel test on Windows")
	}

	caller := New()
	ctx, cancel := context.WithCancel(context.Background())

	// Start a long-running command
	result, err := caller.CallWithCtx(ctx, "sleep", []string{"10"}, Options{})
	if err != nil {
		t.Fatalf("CallWithCtx() failed: %v", err)
	}

	// Cancel context after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Wait for completion
	res, err := result.Wait()

	// Should have been cancelled
	if res.ExitCode != -1 {
		t.Errorf("Expected exit code -1 for cancellation, got %d", res.ExitCode)
	}

	// Check error code
	if mortyErr, ok := errors.AsMortyError(err); ok {
		if mortyErr.Code != "M5007" {
			t.Errorf("Expected error code M5007, got %s", mortyErr.Code)
		}
	} else {
		t.Error("Expected MortyError for cancellation")
	}
}

// TestCallWithCtx_GracefulTermination tests graceful termination with SIGTERM
func TestCallWithCtx_GracefulTermination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping graceful termination test on Windows")
	}

	caller := New()
	ctx := context.Background()

	// Use a script that handles SIGTERM
	opts := Options{
		Timeout:        100 * time.Millisecond,
		GracefulPeriod: 200 * time.Millisecond,
	}

	// Start a sleep command with timeout
	result, err := caller.CallWithCtx(ctx, "sleep", []string{"5"}, opts)
	if err != nil {
		t.Fatalf("CallWithCtx() failed: %v", err)
	}

	// Wait for completion
	res, _ := result.Wait()

	if !res.TimedOut {
		t.Error("Expected TimedOut to be true")
	}

	if res.ExitCode != -1 {
		t.Errorf("Expected exit code -1, got %d", res.ExitCode)
	}
}

// TestCallWithCtx_CommandNotFound tests error handling for missing command
func TestCallWithCtx_CommandNotFound(t *testing.T) {
	caller := New()
	ctx := context.Background()

	_, err := caller.CallWithCtx(ctx, "nonexistent_command_xyz_12345", []string{}, Options{})
	if err == nil {
		t.Fatal("Expected error for non-existent command")
	}

	// Check error code
	if mortyErr, ok := errors.AsMortyError(err); ok {
		if mortyErr.Code != "M5001" {
			t.Errorf("Expected error code M5001, got %s", mortyErr.Code)
		}
	} else {
		t.Error("Expected MortyError")
	}
}

// TestCallWithCtx_CancelledContextBeforeExecution tests cancelled context before execution
func TestCallWithCtx_CancelledContextBeforeExecution(t *testing.T) {
	caller := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := caller.CallWithCtx(ctx, "echo", []string{"hello"}, Options{})
	if err == nil {
		t.Fatal("Expected error for cancelled context")
	}

	// Check error code
	if mortyErr, ok := errors.AsMortyError(err); ok {
		if mortyErr.Code != "M5007" {
			t.Errorf("Expected error code M5007, got %s", mortyErr.Code)
		}
	} else {
		t.Error("Expected MortyError")
	}
}

// TestCallWithCtx_PID tests PID retrieval
func TestCallWithCtx_PID(t *testing.T) {
	caller := New()
	ctx := context.Background()

	result, err := caller.CallWithCtx(ctx, "echo", []string{"hello"}, Options{})
	if err != nil {
		t.Fatalf("CallWithCtx() failed: %v", err)
	}

	pid := result.PID()
	if pid <= 0 {
		t.Errorf("Expected positive PID, got %d", pid)
	}

	// Wait for completion
	result.Wait()
}

// TestCallWithCtx_Running tests running status
func TestCallWithCtx_Running(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping running test on Windows")
	}

	caller := New()
	ctx := context.Background()

	// Start a long-running command
	result, err := caller.CallWithCtx(ctx, "sleep", []string{"2"}, Options{})
	if err != nil {
		t.Fatalf("CallWithCtx() failed: %v", err)
	}

	// Should be running immediately after start
	if !result.Running() {
		t.Error("Expected process to be running")
	}

	// Wait for completion
	result.Wait()

	// Should not be running after completion
	if result.Running() {
		t.Error("Expected process to not be running after completion")
	}
}

// TestCallWithCtx_Kill tests process termination
func TestCallWithCtx_Kill(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping kill test on Windows")
	}

	caller := New()
	ctx := context.Background()

	// Start a long-running command
	result, err := caller.CallWithCtx(ctx, "sleep", []string{"10"}, Options{})
	if err != nil {
		t.Fatalf("CallWithCtx() failed: %v", err)
	}

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Kill the process
	err = result.Kill()
	if err != nil {
		t.Logf("Kill() returned error (may be expected): %v", err)
	}

	// Wait for completion
	res, _ := result.Wait()

	// Should have non-zero exit code
	if res.ExitCode == 0 {
		t.Error("Expected non-zero exit code after kill")
	}
}

// TestCallWithCtx_InterruptedFlag tests the Interrupted flag
func TestCallWithCtx_InterruptedFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping interrupted flag test on Windows")
	}

	caller := New()
	ctx := context.Background()

	// Test timeout
	opts := Options{
		Timeout: 100 * time.Millisecond,
	}

	result, _ := caller.CallWithCtx(ctx, "sleep", []string{"5"}, opts)
	res, _ := result.Wait()

	if !res.Interrupted {
		t.Error("Expected Interrupted to be true for timeout")
	}

	// Test context cancellation
	ctx2, cancel := context.WithCancel(context.Background())
	result2, _ := caller.CallWithCtx(ctx2, "sleep", []string{"10"}, Options{})

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	res2, _ := result2.Wait()
	if !res2.Interrupted {
		t.Error("Expected Interrupted to be true for cancellation")
	}
}

// TestCallWithCtx_StderrCapture tests stderr capture
func TestCallWithCtx_StderrCapture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping stderr test on Windows")
	}

	caller := New()
	ctx := context.Background()

	// Use a command that writes to stderr
	result, err := caller.CallWithCtx(ctx, "sh", []string{"-c", "echo 'error message' >&2"}, Options{})
	if err != nil {
		t.Fatalf("CallWithCtx() failed: %v", err)
	}

	res, _ := result.Wait()

	if res.Stderr != "error message" {
		t.Errorf("Expected stderr 'error message', got '%s'", res.Stderr)
	}
}

// TestCallWithCtx_Stdin tests stdin input
func TestCallWithCtx_Stdin(t *testing.T) {
	caller := New()
	ctx := context.Background()

	opts := Options{
		Stdin: "hello from stdin",
	}

	result, err := caller.CallWithCtx(ctx, "cat", []string{}, opts)
	if err != nil {
		t.Fatalf("CallWithCtx() failed: %v", err)
	}

	res, _ := result.Wait()

	if res.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", res.ExitCode)
	}

	if res.Stdout != "hello from stdin" {
		t.Errorf("Expected stdout 'hello from stdin', got '%s'", res.Stdout)
	}
}

// TestCallWithCtx_DurationTracking tests duration tracking
func TestCallWithCtx_DurationTracking(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping duration test on Windows")
	}

	caller := New()
	ctx := context.Background()

	result, err := caller.CallWithCtx(ctx, "sleep", []string{"1"}, Options{})
	if err != nil {
		t.Fatalf("CallWithCtx() failed: %v", err)
	}

	res, _ := result.Wait()

	if res.Duration < 1*time.Second {
		t.Errorf("Expected duration >= 1s, got %v", res.Duration)
	}
}

// BenchmarkCallWithCtx_Echo benchmarks the CallWithCtx function
func BenchmarkCallWithCtx_Echo(b *testing.B) {
	caller := New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler, _ := caller.CallWithCtx(ctx, "echo", []string{"hello"}, Options{})
		handler.Wait()
	}
}
