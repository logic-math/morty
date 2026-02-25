package callcli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOutputCapture tests the default OutputCapture mode
func TestOutputCapture(t *testing.T) {
	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputCapture,
		},
	}

	result, err := caller.CallWithOptions(context.Background(), "echo", []string{"hello", "world"}, opts)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout != "hello world" {
		t.Errorf("Expected stdout 'hello world', got '%s'", result.Stdout)
	}
}

// TestOutputStream tests the OutputStream mode
func TestOutputStream(t *testing.T) {
	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputStream,
		},
	}

	// Create custom stdout to capture streaming output
	var captured bytes.Buffer
	opts.Output.CustomStdout = &captured

	result, err := caller.CallWithOptions(context.Background(), "echo", []string{"stream", "test"}, opts)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// In stream mode, result.Stdout should be empty
	if result.Stdout != "" {
		t.Errorf("Expected empty stdout in stream mode, got '%s'", result.Stdout)
	}

	// But our custom stdout should have the output
	if !strings.Contains(captured.String(), "stream test") {
		t.Errorf("Expected custom stdout to contain 'stream test', got '%s'", captured.String())
	}
}

// TestOutputCaptureAndStream tests the OutputCaptureAndStream mode
func TestOutputCaptureAndStream(t *testing.T) {
	caller := New()
	var captured bytes.Buffer
	opts := Options{
		Output: OutputConfig{
			Mode:         OutputCaptureAndStream,
			CustomStdout: &captured,
		},
	}

	result, err := caller.CallWithOptions(context.Background(), "echo", []string{"capture", "and", "stream"}, opts)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Both captured and result should have the output
	if result.Stdout != "capture and stream" {
		t.Errorf("Expected stdout 'capture and stream', got '%s'", result.Stdout)
	}

	if !strings.Contains(captured.String(), "capture and stream") {
		t.Errorf("Expected custom stdout to contain 'capture and stream', got '%s'", captured.String())
	}
}

// TestOutputSilent tests the OutputSilent mode
func TestOutputSilent(t *testing.T) {
	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputSilent,
		},
	}

	result, err := caller.CallWithOptions(context.Background(), "echo", []string{"silent", "mode"}, opts)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// In silent mode, result.Stdout should be empty
	if result.Stdout != "" {
		t.Errorf("Expected empty stdout in silent mode, got '%s'", result.Stdout)
	}
}

// TestOutputToFile tests output redirection to a file
func TestOutputToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode:       OutputCapture,
			OutputFile: outputFile,
		},
	}

	result, err := caller.CallWithOptions(context.Background(), "echo", []string{"file", "output"}, opts)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Check result has output
	if result.Stdout != "file output" {
		t.Errorf("Expected stdout 'file output', got '%s'", result.Stdout)
	}

	// Check file has output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "file output") {
		t.Errorf("Expected file to contain 'file output', got '%s'", string(content))
	}
}

// TestOutputMaxCaptureSize tests the output size limiting
func TestOutputMaxCaptureSize(t *testing.T) {
	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode:           OutputCapture,
			MaxCaptureSize: 10, // Only capture first 10 bytes
		},
	}

	result, err := caller.CallWithOptions(context.Background(), "echo", []string{"this is a very long string"}, opts)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Should be truncated to 10 bytes (excluding trailing newline that echo adds)
	if len(result.Stdout) > 10 {
		t.Errorf("Expected output to be truncated to 10 bytes, got %d bytes: '%s'", len(result.Stdout), result.Stdout)
	}
}

// TestOutputCaptureStderr tests stderr capture
func TestOutputCaptureStderr(t *testing.T) {
	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputCapture,
		},
	}

	// Use bash to write to stderr
	result, err := caller.CallWithOptions(context.Background(), "bash", []string{"-c", "echo 'error message' >&2"}, opts)
	if err != nil {
		// Non-zero exit is expected for commands writing to stderr
		t.Logf("Command returned error (may be expected): %v", err)
	}

	if result.Stderr != "error message" {
		t.Errorf("Expected stderr 'error message', got '%s'", result.Stderr)
	}
}

// TestOutputStreamStderr tests stderr streaming
func TestOutputStreamStderr(t *testing.T) {
	caller := New()
	var captured bytes.Buffer
	opts := Options{
		Output: OutputConfig{
			Mode:         OutputStream,
			CustomStderr: &captured,
		},
	}

	result, err := caller.CallWithOptions(context.Background(), "bash", []string{"-c", "echo 'stderr test' >&2"}, opts)
	if err != nil {
		t.Logf("Command returned error (may be expected): %v", err)
	}

	// In stream mode, result.Stderr should be empty
	if result.Stderr != "" {
		t.Errorf("Expected empty stderr in stream mode, got '%s'", result.Stderr)
	}

	// But our custom stderr should have the output
	if !strings.Contains(captured.String(), "stderr test") {
		t.Errorf("Expected custom stderr to contain 'stderr test', got '%s'", captured.String())
	}
}

// TestOutputModeString tests the mode string conversion
func TestOutputModeString(t *testing.T) {
	tests := []struct {
		mode     OutputMode
		expected string
	}{
		{OutputCapture, "capture"},
		{OutputStream, "stream"},
		{OutputCaptureAndStream, "capture_and_stream"},
		{OutputSilent, "silent"},
		{OutputMode(999), "unknown"},
	}

	for _, tc := range tests {
		result := OutputModeString(tc.mode)
		if result != tc.expected {
			t.Errorf("OutputModeString(%v) = '%s', expected '%s'", tc.mode, result, tc.expected)
		}
	}
}

// TestNewOutputHandler_InvalidFile tests error handling for invalid output file
func TestNewOutputHandler_InvalidFile(t *testing.T) {
	config := OutputConfig{
		Mode:       OutputCapture,
		OutputFile: "/nonexistent/path/to/file.txt",
	}

	_, err := NewOutputHandler(config)
	if err == nil {
		t.Error("Expected error for invalid output file path, got nil")
	}
}

// TestOutputCaptureWithAsync tests capture mode with async execution
func TestOutputCaptureWithAsync(t *testing.T) {
	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputCapture,
		},
	}

	handler, err := caller.CallAsyncWithOptions(context.Background(), "echo", []string{"async", "output"}, opts)
	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout != "async output" {
		t.Errorf("Expected stdout 'async output', got '%s'", result.Stdout)
	}
}

// TestOutputSilentWithAsync tests silent mode with async execution
func TestOutputSilentWithAsync(t *testing.T) {
	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputSilent,
		},
	}

	handler, err := caller.CallAsyncWithOptions(context.Background(), "echo", []string{"silent", "async"}, opts)
	if err != nil {
		t.Fatalf("CallAsync failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout != "" {
		t.Errorf("Expected empty stdout in silent mode, got '%s'", result.Stdout)
	}
}

// TestOutputCaptureWithContext tests capture mode with context execution
func TestOutputCaptureWithContext(t *testing.T) {
	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputCapture,
		},
	}

	handler, err := caller.CallWithCtx(context.Background(), "echo", []string{"ctx", "output"}, opts)
	if err != nil {
		t.Fatalf("CallWithCtx failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout != "ctx output" {
		t.Errorf("Expected stdout 'ctx output', got '%s'", result.Stdout)
	}
}

// TestOutputSilentWithContext tests silent mode with context execution
func TestOutputSilentWithContext(t *testing.T) {
	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputSilent,
		},
	}

	handler, err := caller.CallWithCtx(context.Background(), "echo", []string{"silent", "ctx"}, opts)
	if err != nil {
		t.Fatalf("CallWithCtx failed: %v", err)
	}

	result, err := handler.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.Stdout != "" {
		t.Errorf("Expected empty stdout in silent mode, got '%s'", result.Stdout)
	}
}

// TestOutputStreamToFile tests streaming mode with file output
func TestOutputStreamToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "stream_output.txt")

	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode:       OutputStream,
			OutputFile: outputFile,
		},
	}

	result, err := caller.CallWithOptions(context.Background(), "echo", []string{"stream", "to", "file"}, opts)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// In stream mode without capture, result should be empty
	if result.Stdout != "" {
		t.Errorf("Expected empty stdout in stream mode, got '%s'", result.Stdout)
	}

	// Check file has output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "stream to file") {
		t.Errorf("Expected file to contain 'stream to file', got '%s'", string(content))
	}
}

// TestOutputCaptureAndStreamToFile tests capture and stream mode with file output
func TestOutputCaptureAndStreamToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "capture_stream_output.txt")

	caller := New()
	var captured bytes.Buffer
	opts := Options{
		Output: OutputConfig{
			Mode:         OutputCaptureAndStream,
			OutputFile:   outputFile,
			CustomStdout: &captured,
		},
	}

	result, err := caller.CallWithOptions(context.Background(), "echo", []string{"all", "outputs"}, opts)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Result should have the output
	if result.Stdout != "all outputs" {
		t.Errorf("Expected stdout 'all outputs', got '%s'", result.Stdout)
	}

	// Custom stdout should have the output
	if !strings.Contains(captured.String(), "all outputs") {
		t.Errorf("Expected custom stdout to contain 'all outputs', got '%s'", captured.String())
	}

	// File should have the output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "all outputs") {
		t.Errorf("Expected file to contain 'all outputs', got '%s'", string(content))
	}
}

// TestOutputSilentToFile tests silent mode with file output
func TestOutputSilentToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "silent_output.txt")

	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode:       OutputSilent,
			OutputFile: outputFile,
		},
	}

	result, err := caller.CallWithOptions(context.Background(), "echo", []string{"silent", "file"}, opts)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Result should be empty in silent mode
	if result.Stdout != "" {
		t.Errorf("Expected empty stdout in silent mode, got '%s'", result.Stdout)
	}

	// But file should still have the output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if !strings.Contains(string(content), "silent file") {
		t.Errorf("Expected file to contain 'silent file', got '%s'", string(content))
	}
}

// TestLimitedWriter tests the limited writer functionality
func TestLimitedWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	lw := &limitedWriter{
		buf:     buf,
		maxSize: 10,
	}

	// Write data that fits
	n, err := lw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Expected to write 5 bytes, wrote %d", n)
	}

	// Write more data that exceeds limit
	n, err = lw.Write([]byte(" world this is too much"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	// Should report writing all bytes even though only part is stored
	if n != 23 {
		t.Errorf("Expected to report writing 23 bytes, wrote %d", n)
	}

	// Buffer should only contain 10 bytes
	if buf.Len() != 10 {
		t.Errorf("Expected buffer to have 10 bytes, got %d", buf.Len())
	}

	if buf.String() != "hello worl" {
		t.Errorf("Expected 'hello worl', got '%s'", buf.String())
	}

	// Write when already at limit
	n, err = lw.Write([]byte("more"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 4 {
		t.Errorf("Expected to report writing 4 bytes, wrote %d", n)
	}

	// Buffer should still contain only 10 bytes
	if buf.Len() != 10 {
		t.Errorf("Expected buffer to still have 10 bytes, got %d", buf.Len())
	}
}

// TestOutputHandlerClose tests closing the output handler
func TestOutputHandlerClose(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "close_test.txt")

	config := OutputConfig{
		Mode:       OutputCapture,
		OutputFile: outputFile,
	}

	handler, err := NewOutputHandler(config)
	if err != nil {
		t.Fatalf("Failed to create output handler: %v", err)
	}

	// Write some data
	writer := handler.StdoutWriter()
	writer.Write([]byte("test data"))

	// Close the handler
	err = handler.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// File should be closed and have content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if !strings.Contains(string(content), "test data") {
		t.Errorf("Expected file to contain 'test data', got '%s'", string(content))
	}
}

// TestOutputHandlerGetStdoutGetStderr tests the getter methods
func TestOutputHandlerGetStdoutGetStderr(t *testing.T) {
	config := OutputConfig{
		Mode: OutputCapture,
	}

	handler, err := NewOutputHandler(config)
	if err != nil {
		t.Fatalf("Failed to create output handler: %v", err)
	}

	// Initially should be empty
	if handler.GetStdout() != "" {
		t.Errorf("Expected empty stdout initially, got '%s'", handler.GetStdout())
	}
	if handler.GetStderr() != "" {
		t.Errorf("Expected empty stderr initially, got '%s'", handler.GetStderr())
	}

	// Write some data
	stdoutWriter := handler.StdoutWriter()
	stdoutWriter.Write([]byte("stdout data"))

	stderrWriter := handler.StderrWriter()
	stderrWriter.Write([]byte("stderr data"))

	// Check getters
	if handler.GetStdout() != "stdout data" {
		t.Errorf("Expected 'stdout data', got '%s'", handler.GetStdout())
	}
	if handler.GetStderr() != "stderr data" {
		t.Errorf("Expected 'stderr data', got '%s'", handler.GetStderr())
	}
}

// TestOutputDefaultMode tests that default mode is capture
func TestOutputDefaultMode(t *testing.T) {
	caller := New()
	// Don't specify output config, should default to capture mode
	opts := Options{}

	result, err := caller.CallWithOptions(context.Background(), "echo", []string{"default", "mode"}, opts)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if result.Stdout != "default mode" {
		t.Errorf("Expected 'default mode', got '%s'", result.Stdout)
	}
}

// BenchmarkOutputCapture benchmarks the capture mode
func BenchmarkOutputCapture(b *testing.B) {
	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputCapture,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = caller.CallWithOptions(context.Background(), "echo", []string{"benchmark"}, opts)
	}
}

// BenchmarkOutputSilent benchmarks the silent mode
func BenchmarkOutputSilent(b *testing.B) {
	caller := New()
	opts := Options{
		Output: OutputConfig{
			Mode: OutputSilent,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = caller.CallWithOptions(context.Background(), "echo", []string{"benchmark"}, opts)
	}
}

// MockWriter is a writer that can simulate errors for testing
type MockWriter struct {
	WriteFunc func(p []byte) (n int, err error)
}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	if m.WriteFunc != nil {
		return m.WriteFunc(p)
	}
	return len(p), nil
}

// Ensure MockWriter implements io.Writer
var _ io.Writer = (*MockWriter)(nil)
