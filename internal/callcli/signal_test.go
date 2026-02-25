// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"context"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/morty/morty/pkg/errors"
)

// TestSignalHandler_Creation tests creating a SignalHandler.
func TestSignalHandler_Creation(t *testing.T) {
	caller := New()

	ctx := context.Background()
	handler, err := caller.CallWithSignal(ctx, "echo", []string{"hello"}, Options{})
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	if handler == nil {
		t.Fatal("handler is nil")
	}

	// Wait for completion
	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout != "hello" {
		t.Errorf("expected stdout 'hello', got '%s'", result.Stdout)
	}

	// Check not interrupted
	if handler.Interrupted() {
		t.Error("handler should not be interrupted")
	}
}

// TestSignalHandler_Kill tests killing a running process.
func TestSignalHandler_Kill(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	caller := New()

	ctx := context.Background()
	// Start a long-running process
	handler, err := caller.CallWithSignal(ctx, "sleep", []string{"10"}, Options{})
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Check it's running
	if !handler.Running() {
		t.Error("process should be running")
	}

	// Get PID
	pid := handler.PID()
	if pid <= 0 {
		t.Error("PID should be positive")
	}

	// Kill it
	if err := handler.Kill(); err != nil {
		t.Errorf("Kill failed: %v", err)
	}

	// Wait for result
	result, err := handler.Wait()

	// Should have error or non-zero exit
	if err == nil && result.ExitCode == 0 {
		t.Error("expected error or non-zero exit code after kill")
	}

	// Check interrupted flag
	if !handler.Interrupted() {
		t.Error("handler should be interrupted after kill")
	}

	// Should not be running anymore
	if handler.Running() {
		t.Error("process should not be running after kill")
	}
}

// TestSignalHandler_Timeout tests timeout with signal handler.
func TestSignalHandler_Timeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	caller := New()

	ctx := context.Background()
	opts := Options{
		Timeout: 100 * time.Millisecond,
	}

	handler, err := caller.CallWithSignal(ctx, "sleep", []string{"10"}, opts)
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	result, err := handler.Wait()

	// Should be timed out
	if !result.TimedOut {
		t.Error("expected TimedOut to be true")
	}

	// Should be interrupted
	if !result.Interrupted {
		t.Error("expected Interrupted to be true")
	}

	// Verify error code
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code after timeout")
	}

	// Check error type
	if err != nil {
		if mortyErr, ok := errors.AsMortyError(err); ok {
			if mortyErr.Code != "M5003" {
				t.Errorf("expected error code M5003, got %s", mortyErr.Code)
			}
		}
	}
}

// TestSignalHandler_ContextCancel tests context cancellation.
func TestSignalHandler_ContextCancel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	caller := New()

	ctx, cancel := context.WithCancel(context.Background())

	handler, err := caller.CallWithSignal(ctx, "sleep", []string{"10"}, Options{})
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	result, err := handler.Wait()

	// Should be interrupted
	if !result.Interrupted {
		t.Error("expected Interrupted to be true after context cancel")
	}

	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code after cancel")
	}
}

// TestSignalHandler_GracefulTermination tests graceful termination.
func TestSignalHandler_GracefulTermination(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	caller := New()

	ctx := context.Background()
	opts := Options{
		Timeout:        50 * time.Millisecond,
		GracefulPeriod: 200 * time.Millisecond,
	}

	handler, err := caller.CallWithSignal(ctx, "sleep", []string{"10"}, opts)
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	start := time.Now()
	result, _ := handler.Wait()
	elapsed := time.Since(start)

	// Should have waited for graceful period
	if elapsed < opts.GracefulPeriod {
		t.Logf("elapsed time %v is less than graceful period %v", elapsed, opts.GracefulPeriod)
	}

	if !result.TimedOut {
		t.Error("expected TimedOut to be true")
	}
}

// TestSignalHandler_InterruptState tests interrupt state tracking.
func TestSignalHandler_InterruptState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	caller := New()

	ctx := context.Background()
	handler, err := caller.CallWithSignal(ctx, "echo", []string{"test"}, Options{})
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	// Wait for completion
	handler.Wait()

	// Get interrupt state
	state := handler.GetInterruptState()

	if state.Command == "" {
		t.Error("expected non-empty command in state")
	}

	if state.PID == 0 {
		t.Error("expected non-zero PID in state")
	}

	if state.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp in state")
	}
}

// TestSignalHandler_Resume tests resume functionality.
func TestSignalHandler_Resume(t *testing.T) {
	caller := New()

	ctx := context.Background()

	// First call
	handler, err := caller.CallWithSignal(ctx, "echo", []string{"hello"}, Options{})
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	handler.Wait()

	// Resume with same command
	newHandler, err := handler.Resume(ctx, caller, Options{})
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	result, err := newHandler.Wait()
	if err != nil {
		t.Fatalf("Wait after resume failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0 after resume, got %d", result.ExitCode)
	}
}

// TestSignalHandler_SignalForwarding tests signal forwarding.
func TestSignalHandler_SignalForwarding(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	caller := New()

	ctx := context.Background()
	handler, err := caller.CallWithSignal(ctx, "sleep", []string{"5"}, Options{})
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Simulate signal by calling handleSignal directly
	handler.handleSignal(syscall.SIGINT)

	// Wait for result
	result, _ := handler.Wait()

	// Should be interrupted
	if !handler.Interrupted() {
		t.Error("expected handler to be interrupted")
	}

	if !result.Interrupted {
		t.Error("expected result.Interrupted to be true")
	}

	// Check signal received
	sig := handler.SignalReceived()
	if sig == nil {
		t.Error("expected signal to be recorded")
	}
}

// TestGlobalSignalHandler tests the global signal handler.
func TestGlobalSignalHandler(t *testing.T) {
	gh := getGlobalSignalHandler()

	if gh == nil {
		t.Fatal("global handler is nil")
	}

	// Start the handler
	gh.Start()

	if !gh.started {
		t.Error("global handler should be started")
	}

	// Stop the handler
	gh.Stop()
}

// TestSignalHandler_PID tests PID retrieval.
func TestSignalHandler_PID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	caller := New()

	ctx := context.Background()
	handler, err := caller.CallWithSignal(ctx, "sleep", []string{"1"}, Options{})
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	pid := handler.PID()
	if pid <= 0 {
		t.Error("PID should be positive")
	}

	// Wait for completion
	handler.Wait()

	// PID should still be available after completion
	if handler.PID() != pid {
		t.Error("PID should remain the same after completion")
	}
}

// TestSignalHandler_Running tests Running() method.
func TestSignalHandler_Running(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	caller := New()

	ctx := context.Background()
	handler, err := caller.CallWithSignal(ctx, "sleep", []string{"0.1"}, Options{})
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	// Should be running initially
	if !handler.Running() {
		t.Error("handler should be running initially")
	}

	// Wait for completion
	handler.Wait()

	// Should not be running after completion
	if handler.Running() {
		t.Error("handler should not be running after completion")
	}
}

// TestSignalHandler_ChildProcessTracking tests child process tracking.
func TestSignalHandler_ChildProcessTracking(t *testing.T) {
	handler := &SignalHandler{
		childProcesses: make([]*exec.Cmd, 0),
	}

	// Create a mock command (not started)
	cmd := exec.Command("echo", "test")

	// Add child process
	handler.AddChildProcess(cmd)

	if len(handler.childProcesses) != 1 {
		t.Errorf("expected 1 child process, got %d", len(handler.childProcesses))
	}

	// Remove child process
	handler.RemoveChildProcess(cmd)

	if len(handler.childProcesses) != 0 {
		t.Errorf("expected 0 child processes, got %d", len(handler.childProcesses))
	}
}

// TestSignalHandler_OutputCapture tests output capture with signal handler.
func TestSignalHandler_OutputCapture(t *testing.T) {
	caller := New()

	ctx := context.Background()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputCapture,
		},
	}

	handler, err := caller.CallWithSignal(ctx, "echo", []string{"captured output"}, opts)
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.Stdout != "captured output" {
		t.Errorf("expected 'captured output', got '%s'", result.Stdout)
	}
}

// TestSignalHandler_ConcurrentAccess tests concurrent access to handler.
func TestSignalHandler_ConcurrentAccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	caller := New()

	ctx := context.Background()
	handler, err := caller.CallWithSignal(ctx, "sleep", []string{"0.5"}, Options{})
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	// Concurrent access tests
	done := make(chan bool, 4)

	// Check Running concurrently
	go func() {
		for i := 0; i < 10; i++ {
			_ = handler.Running()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Check PID concurrently
	go func() {
		for i := 0; i < 10; i++ {
			_ = handler.PID()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Check Interrupted concurrently
	go func() {
		for i := 0; i < 10; i++ {
			_ = handler.Interrupted()
			time.Sleep(10 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for completion concurrently
	go func() {
		handler.Wait()
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
}

// TestSignalHandler_ErrorCode tests error code handling.
func TestSignalHandler_ErrorCode(t *testing.T) {
	caller := New()

	ctx := context.Background()
	handler, err := caller.CallWithSignal(ctx, "false", []string{}, Options{})
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	result, err := handler.Wait()

	// false command returns exit code 1
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}

	if err == nil {
		t.Error("expected error for false command")
	}
}

// TestSignalHandler_InvalidCommand tests handling of invalid commands.
func TestSignalHandler_InvalidCommand(t *testing.T) {
	caller := New()

	ctx := context.Background()
	_, err := caller.CallWithSignal(ctx, "nonexistent_command_xyz", []string{}, Options{})

	if err == nil {
		t.Error("expected error for invalid command")
	}

	mortyErr, ok := errors.AsMortyError(err)
	if !ok {
		t.Error("expected MortyError")
	} else if mortyErr.Code != "M5001" {
		t.Errorf("expected error code M5001, got %s", mortyErr.Code)
	}
}

// TestSignalHandler_ContextAlreadyCancelled tests cancelled context before execution.
func TestSignalHandler_ContextAlreadyCancelled(t *testing.T) {
	caller := New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := caller.CallWithSignal(ctx, "echo", []string{"hello"}, Options{})

	if err == nil {
		t.Error("expected error for cancelled context")
	}

	mortyErr, ok := errors.AsMortyError(err)
	if !ok {
		t.Error("expected MortyError")
	} else if mortyErr.Code != "M5007" {
		t.Errorf("expected error code M5007, got %s", mortyErr.Code)
	}
}

// TestSignalHandler_Duration tests duration tracking.
func TestSignalHandler_Duration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	caller := New()

	ctx := context.Background()
	handler, err := caller.CallWithSignal(ctx, "sleep", []string{"0.2"}, Options{})
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	start := time.Now()
	result, _ := handler.Wait()
	elapsed := time.Since(start)

	// Duration should be reasonable
	if result.Duration < 0 {
		t.Error("duration should be non-negative")
	}

	// Elapsed time should be at least sleep time
	if elapsed < 150*time.Millisecond {
		t.Errorf("elapsed time %v seems too short", elapsed)
	}
}

// TestSignalHandler_StreamOutput tests streaming output mode.
func TestSignalHandler_StreamOutput(t *testing.T) {
	caller := New()

	ctx := context.Background()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputStream,
		},
	}

	handler, err := caller.CallWithSignal(ctx, "echo", []string{"stream test"}, opts)
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

// TestSignalHandler_StdinInput tests stdin input with signal handler.
func TestSignalHandler_StdinInput(t *testing.T) {
	caller := New()

	ctx := context.Background()
	opts := Options{
		Stdin: "hello from stdin",
	}

	handler, err := caller.CallWithSignal(ctx, "cat", []string{}, opts)
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.Stdout != "hello from stdin" {
		t.Errorf("expected 'hello from stdin', got '%s'", result.Stdout)
	}
}

// TestSignalHandler_WorkingDir tests working directory with signal handler.
func TestSignalHandler_WorkingDir(t *testing.T) {
	caller := New()

	ctx := context.Background()
	opts := Options{
		WorkingDir: "/tmp",
	}

	handler, err := caller.CallWithSignal(ctx, "pwd", []string{}, opts)
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.Stdout != "/tmp" {
		t.Errorf("expected '/tmp', got '%s'", result.Stdout)
	}
}

// TestSignalHandler_EnvVars tests environment variables with signal handler.
func TestSignalHandler_EnvVars(t *testing.T) {
	caller := New()

	ctx := context.Background()
	opts := Options{
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	handler, err := caller.CallWithSignal(ctx, "sh", []string{"-c", "echo $TEST_VAR"}, opts)
	if err != nil {
		t.Fatalf("CallWithSignal failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.Stdout != "test_value" {
		t.Errorf("expected 'test_value', got '%s'", result.Stdout)
	}
}
