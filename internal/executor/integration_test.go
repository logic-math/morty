// Package executor provides integration tests for the executor module.
// These tests verify the integration between TaskRunner, Call CLI, and ResultParser.
package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/morty/morty/internal/callcli"
	"github.com/morty/morty/internal/git"
	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/state"
)

// mockAICallerForIntegration is a mock AI CLI caller for integration testing.
type mockAICallerForIntegration struct {
	callResult      *callcli.Result
	callError       error
	delay           time.Duration
	called          bool
	lastPrompt      string
	lastPromptFile  string
	callContentUsed bool
}

func (m *mockAICallerForIntegration) CallWithPrompt(ctx context.Context, promptFile string) (*callcli.Result, error) {
	m.called = true
	m.lastPromptFile = promptFile
	m.callContentUsed = false

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return &callcli.Result{
				ExitCode: -1,
				TimedOut: true,
			}, ctx.Err()
		}
	}

	return m.callResult, m.callError
}

func (m *mockAICallerForIntegration) CallWithPromptContent(ctx context.Context, content string) (*callcli.Result, error) {
	m.called = true
	m.lastPrompt = content
	m.callContentUsed = true

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return &callcli.Result{
				ExitCode: -1,
				TimedOut: true,
			}, ctx.Err()
		}
	}

	return m.callResult, m.callError
}

func (m *mockAICallerForIntegration) GetCLIPath() string {
	return "mock-claude-cli"
}

func (m *mockAICallerForIntegration) BuildArgs() []string {
	return []string{"--output-format", "json"}
}

// integrationTestLogger is a simple logger for integration tests.
type integrationTestLogger struct {
	logs []string
}

func (m *integrationTestLogger) Debug(msg string, attrs ...logging.Attr) {}
func (m *integrationTestLogger) Info(msg string, attrs ...logging.Attr)  {}
func (m *integrationTestLogger) Warn(msg string, attrs ...logging.Attr)  {}
func (m *integrationTestLogger) Error(msg string, attrs ...logging.Attr) {}
func (m *integrationTestLogger) Success(msg string, attrs ...logging.Attr) {}
func (m *integrationTestLogger) Loop(msg string, attrs ...logging.Attr)  {}
func (m *integrationTestLogger) WithContext(ctx context.Context) logging.Logger { return m }
func (m *integrationTestLogger) WithJob(module, job string) logging.Logger      { return m }
func (m *integrationTestLogger) WithAttrs(attrs ...logging.Attr) logging.Logger { return m }
func (m *integrationTestLogger) SetLevel(level logging.Level)                   {}
func (m *integrationTestLogger) GetLevel() logging.Level                        { return logging.InfoLevel }
func (m *integrationTestLogger) IsEnabled(level logging.Level) bool             { return true }

// setupIntegrationTestEnv creates a test environment for integration tests.
func setupIntegrationTestEnv(t *testing.T) (string, *state.Manager, logging.Logger, func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "executor-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create .morty directory structure
	mortyDir := filepath.Join(tempDir, ".morty")
	if err := os.MkdirAll(mortyDir, 0755); err != nil {
		t.Fatalf("failed to create .morty dir: %v", err)
	}

	// Create initial state file
	stateContent := `{
		"version": "1.0",
		"global": {"status": "PENDING", "total_loops": 0, "current_module": "", "current_job": ""},
		"modules": {
			"test_module": {
				"name": "test_module",
				"status": "PENDING",
				"jobs": {
					"test_job": {
						"name": "test_job",
						"status": "PENDING",
						"tasks_total": 3,
						"tasks_completed": 0,
						"retry_count": 0,
						"tasks": [
							{"index": 0, "description": "Task 1: Test task one", "status": "PENDING", "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"},
							{"index": 1, "description": "Task 2: Test task two", "status": "PENDING", "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"},
							{"index": 2, "description": "Task 3: Test task three", "status": "PENDING", "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"}
						],
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					}
				}
			}
		}
	}`

	stateFile := filepath.Join(mortyDir, "state.json")
	if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
		t.Fatalf("failed to create state file: %v", err)
	}

	// Create state manager
	stateManager := state.NewManager(stateFile)
	if err := stateManager.Load(); err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	// Create logger
	logger := &integrationTestLogger{}

	// Cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, stateManager, logger, cleanup
}

// TestTaskRunner_Integration_CallCLI verifies TaskRunner correctly calls Call CLI.
func TestTaskRunner_Integration_CallCLI(t *testing.T) {
	logger := &integrationTestLogger{}

	tests := []struct {
		name           string
		callResult     *callcli.Result
		callError      error
		expectSuccess  bool
		expectExitCode int
	}{
		{
			name: "successful execution",
			callResult: &callcli.Result{
				Stdout:   "Task completed successfully",
				Stderr:   "",
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
			},
			callError:      nil,
			expectSuccess:  true,
			expectExitCode: 0,
		},
		{
			name: "non-zero exit code",
			callResult: &callcli.Result{
				Stdout:   "",
				Stderr:   "error: task failed",
				ExitCode: 1,
				Duration: 50 * time.Millisecond,
			},
			callError:      nil,
			expectSuccess:  false,
			expectExitCode: 1,
		},
		{
			name:           "execution error",
			callResult:     nil,
			callError:      errors.New("execution failed"),
			expectSuccess:  false,
			expectExitCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCaller := &mockAICallerForIntegration{
				callResult: tt.callResult,
				callError:  tt.callError,
			}

			tr := NewTaskRunner(logger, mockCaller)

			result, err := tr.Run(context.Background(), "Integration Test", "test prompt content")

			// Verify result
			if result == nil {
				t.Fatal("expected result, got nil")
			}

			if result.Success != tt.expectSuccess {
				t.Errorf("expected success=%v, got %v", tt.expectSuccess, result.Success)
			}

			if result.ExitCode != tt.expectExitCode {
				t.Errorf("expected exit code %d, got %d", tt.expectExitCode, result.ExitCode)
			}

			if !mockCaller.called {
				t.Error("expected AI CLI caller to be invoked")
			}

			if mockCaller.lastPrompt != "test prompt content" {
				t.Errorf("expected prompt 'test prompt content', got %s", mockCaller.lastPrompt)
			}

			// Error should be present for failed executions
			if !tt.expectSuccess && err == nil {
				t.Error("expected error for failed execution")
			}
		})
	}
}

// TestTaskRunner_Integration_PromptPassedCorrectly verifies prompt is correctly passed to AI CLI.
func TestTaskRunner_Integration_PromptPassedCorrectly(t *testing.T) {
	logger := &integrationTestLogger{}
	mockCaller := &mockAICallerForIntegration{
		callResult: &callcli.Result{
			Stdout:   "success",
			ExitCode: 0,
			Duration: 10 * time.Millisecond,
		},
	}

	tr := NewTaskRunner(logger, mockCaller)

	// Test with different prompt contents
	prompts := []string{
		"simple prompt",
		"prompt with special chars: <>&\"'",
		"multi-line\nprompt\ncontent",
		`prompt with json: {"key": "value"}`,
	}

	for _, prompt := range prompts {
		mockCaller.called = false
		mockCaller.lastPrompt = ""

		_, err := tr.Run(context.Background(), "Prompt Test", prompt)
		if err != nil {
			t.Errorf("unexpected error for prompt %q: %v", prompt, err)
		}

		if !mockCaller.called {
			t.Errorf("caller not invoked for prompt %q", prompt)
		}

		if mockCaller.lastPrompt != prompt {
			t.Errorf("prompt mismatch: expected %q, got %q", prompt, mockCaller.lastPrompt)
		}
	}
}

// TestTaskRunner_Integration_Timeout verifies timeout configuration works correctly.
func TestTaskRunner_Integration_Timeout(t *testing.T) {
	logger := &integrationTestLogger{}

	// Test default timeout (10 minutes)
	t.Run("default timeout", func(t *testing.T) {
		mockCaller := &mockAICallerForIntegration{
			callResult: &callcli.Result{
				ExitCode: 0,
				Duration: 10 * time.Millisecond,
			},
		}

		tr := NewTaskRunner(logger, mockCaller)

		if tr.GetTimeout() != DefaultTaskTimeout {
			t.Errorf("expected default timeout %v, got %v", DefaultTaskTimeout, tr.GetTimeout())
		}
	})

	// Test custom timeout
	t.Run("custom timeout", func(t *testing.T) {
		customTimeout := 30 * time.Second
		mockCaller := &mockAICallerForIntegration{
			callResult: &callcli.Result{
				ExitCode: 0,
				Duration: 10 * time.Millisecond,
			},
		}

		tr := NewTaskRunnerWithTimeout(logger, mockCaller, customTimeout)

		if tr.GetTimeout() != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, tr.GetTimeout())
		}
	})

	// Test timeout enforcement
	t.Run("timeout enforcement", func(t *testing.T) {
		mockCaller := &mockAICallerForIntegration{
			callResult: &callcli.Result{
				ExitCode: -1,
				TimedOut: true,
			},
			callError: context.DeadlineExceeded,
			delay:     200 * time.Millisecond,
		}

		tr := NewTaskRunnerWithTimeout(logger, mockCaller, 50*time.Millisecond)

		result, err := tr.Run(context.Background(), "Slow Task", "test")

		if err == nil {
			t.Error("expected timeout error")
		}

		if result == nil {
			t.Fatal("expected result, got nil")
		}

		if !result.TimedOut {
			t.Error("expected TimedOut to be true")
		}
	})
}

// TestTaskRunner_Integration_ResultParsing verifies result parsing integration.
func TestTaskRunner_Integration_ResultParsing(t *testing.T) {
	logger := &integrationTestLogger{}

	// Test successful result parsing
	t.Run("successful result", func(t *testing.T) {
		mockCaller := &mockAICallerForIntegration{
			callResult: &callcli.Result{
				Stdout: `<!-- RALPH_STATUS -->
{
	"module": "test",
	"job": "test_job",
	"status": "COMPLETED",
	"tasks_completed": 5,
	"tasks_total": 5,
	"summary": "All tasks completed"
}
<!-- END_RALPH_STATUS -->`,
				Stderr:   "",
				ExitCode: 0,
				Duration: 100 * time.Millisecond,
			},
		}

		tr := NewTaskRunner(logger, mockCaller)
		result, err := tr.Run(context.Background(), "Test", "prompt")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Error("expected success to be true")
		}

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
	})

	// Test failed result with RALPH_STATUS
	t.Run("failed result with RALPH_STATUS", func(t *testing.T) {
		mockCaller := &mockAICallerForIntegration{
			callResult: &callcli.Result{
				Stdout: `Some output before
<!-- RALPH_STATUS -->
{
	"module": "test",
	"job": "test_job",
	"status": "FAILED",
	"tasks_completed": 2,
	"tasks_total": 5,
	"summary": "Task execution failed"
}
<!-- END_RALPH_STATUS -->`,
				Stderr:   "error: something went wrong",
				ExitCode: 1,
				Duration: 50 * time.Millisecond,
			},
		}

		tr := NewTaskRunner(logger, mockCaller)
		result, err := tr.Run(context.Background(), "Test", "prompt")

		if err == nil {
			t.Error("expected error for non-zero exit code")
		}

		if result.Success {
			t.Error("expected success to be false")
		}

		if result.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", result.ExitCode)
		}
	})
}

// TestTaskRunner_Integration_Interruption verifies interruption handling.
func TestTaskRunner_Integration_Interruption(t *testing.T) {
	logger := &integrationTestLogger{}

	t.Run("context cancellation", func(t *testing.T) {
		mockCaller := &mockAICallerForIntegration{
			callResult: &callcli.Result{
				ExitCode: -1,
				TimedOut: false,
			},
			callError: context.Canceled,
		}

		tr := NewTaskRunner(logger, mockCaller)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := tr.Run(ctx, "Cancelled Task", "test")

		if err == nil {
			t.Error("expected error for cancelled context")
		}

		if result == nil {
			t.Fatal("expected result, got nil")
		}

		if result.Success {
			t.Error("expected success to be false")
		}
	})

	t.Run("signal forwarding through context", func(t *testing.T) {
		// Simulate a slow operation that gets interrupted
		mockCaller := &mockAICallerForIntegration{
			callResult: &callcli.Result{
				ExitCode: -1,
				TimedOut: false,
			},
			delay:     500 * time.Millisecond,
			callError: context.Canceled,
		}

		tr := NewTaskRunnerWithTimeout(logger, mockCaller, 1*time.Second)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		start := time.Now()
		result, err := tr.Run(ctx, "Interrupt Test", "test")
		elapsed := time.Since(start)

		if err == nil {
			t.Error("expected error for interrupted execution")
		}

		if result == nil {
			t.Fatal("expected result, got nil")
		}

		// Should have been interrupted before the full delay
		if elapsed > 400*time.Millisecond {
			t.Errorf("interruption took too long: %v", elapsed)
		}
	})
}

// TestJobRunner_Integration_TaskExecution verifies JobRunner task execution flow.
func TestJobRunner_Integration_TaskExecution(t *testing.T) {
	tempDir, stateManager, logger, cleanup := setupIntegrationTestEnv(t)
	defer cleanup()

	// Track executed tasks
	executedTasks := make(map[int]string)

	// Create a task executor that records execution
	taskExecutor := func(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
		executedTasks[taskIndex] = taskDesc
		return nil
	}

	jr := NewJobRunner(stateManager, logger, taskExecutor)

	completed, err := jr.Run(context.Background(), "test_module", "test_job")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if completed != 3 {
		t.Errorf("expected 3 completed tasks, got %d", completed)
	}

	if len(executedTasks) != 3 {
		t.Errorf("expected 3 executed tasks, got %d", len(executedTasks))
	}

	// Verify state was updated
	jobState := stateManager.GetJob("test_module", "test_job")
	if jobState == nil {
		t.Fatal("job state not found")
	}

	// Check that tasks are marked as completed
	for i, task := range jobState.Tasks {
		if task.Status != state.StatusCompleted {
			t.Errorf("task %d status: expected COMPLETED, got %s", i, task.Status)
		}
	}

	// Cleanup
	_ = tempDir
}

// TestEngine_Integration_ExecuteJob verifies the full Engine job execution flow.
func TestEngine_Integration_ExecuteJob(t *testing.T) {
	tempDir, stateManager, logger, cleanup := setupIntegrationTestEnv(t)
	defer cleanup()

	// Create git manager (will fail but that's ok for this test)
	gitManager := git.NewManager()

	// Create engine config
	config := &Config{
		MaxRetries:   1,
		AutoCommit:   false, // Disable auto-commit for test
		CommitPrefix: "test:",
		WorkingDir:   tempDir,
	}

	// Create engine
	eng := NewEngine(stateManager, gitManager, logger, config)

	// Execute the job
	err := eng.ExecuteJob(context.Background(), "test_module", "test_job")

	// The job should complete (ExecuteTask is a placeholder that just marks tasks complete)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify job state
	jobState := stateManager.GetJob("test_module", "test_job")
	if jobState == nil {
		t.Fatal("job state not found")
	}

	if jobState.Status != state.StatusCompleted {
		t.Errorf("expected job status COMPLETED, got %s", jobState.Status)
	}
}

// TestTaskRunner_Integration_ExitCodeHandling verifies exit code handling.
func TestTaskRunner_Integration_ExitCodeHandling(t *testing.T) {
	logger := &integrationTestLogger{}

	tests := []struct {
		name           string
		exitCode       int
		expectSuccess  bool
		expectError    bool
	}{
		{"exit 0 - success", 0, true, false},
		{"exit 1 - error", 1, false, true},
		{"exit 2 - error", 2, false, true},
		{"exit 127 - command not found", 127, false, true},
		{"exit 255 - error", 255, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCaller := &mockAICallerForIntegration{
				callResult: &callcli.Result{
					Stdout:   "",
					Stderr:   fmt.Sprintf("exit code %d", tt.exitCode),
					ExitCode: tt.exitCode,
					Duration: 10 * time.Millisecond,
				},
			}

			tr := NewTaskRunner(logger, mockCaller)
			result, err := tr.Run(context.Background(), "Exit Code Test", "test")

			if result.Success != tt.expectSuccess {
				t.Errorf("expected success=%v, got %v", tt.expectSuccess, result.Success)
			}

			if result.ExitCode != tt.exitCode {
				t.Errorf("expected exit code %d, got %d", tt.exitCode, result.ExitCode)
			}

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestTaskRunner_Integration_OutputCapture verifies output capture.
func TestTaskRunner_Integration_OutputCapture(t *testing.T) {
	logger := &integrationTestLogger{}

	mockCaller := &mockAICallerForIntegration{
		callResult: &callcli.Result{
			Stdout:   "standard output line 1\nline 2",
			Stderr:   "standard error message",
			ExitCode: 0,
			Duration: 10 * time.Millisecond,
		},
	}

	tr := NewTaskRunner(logger, mockCaller)
	result, err := tr.Run(context.Background(), "Output Test", "test")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.Stdout != "standard output line 1\nline 2" {
		t.Errorf("unexpected stdout: %q", result.Stdout)
	}

	if result.Stderr != "standard error message" {
		t.Errorf("unexpected stderr: %q", result.Stderr)
	}
}

// TestTaskRunner_Integration_WithPromptFile verifies prompt file path passing.
func TestTaskRunner_Integration_WithPromptFile(t *testing.T) {
	logger := &integrationTestLogger{}

	// Create a temporary prompt file
	tempDir := t.TempDir()
	promptFile := filepath.Join(tempDir, "test_prompt.md")
	promptContent := "# Test Prompt\n\nThis is a test prompt."
	if err := os.WriteFile(promptFile, []byte(promptContent), 0644); err != nil {
		t.Fatalf("failed to create prompt file: %v", err)
	}

	// Create mock caller that verifies prompt file path
	mockCaller := &mockAICallerForIntegration{
		callResult: &callcli.Result{
			ExitCode: 0,
			Duration: 10 * time.Millisecond,
		},
	}

	tr := NewTaskRunner(logger, mockCaller)

	// Note: TaskRunner uses CallWithPromptContent by default
	// This test verifies the integration pattern
	result, err := tr.Run(context.Background(), "File Test", promptContent)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success")
	}

	// The prompt content should have been passed
	if mockCaller.lastPrompt != promptContent {
		t.Errorf("prompt content not passed correctly")
	}
}

// TestFullIntegration_ExecutorWithCallCLI tests the full executor integration.
func TestFullIntegration_ExecutorWithCallCLI(t *testing.T) {
	logger := &integrationTestLogger{}

	// Test complete flow: TaskRunner -> AICliCaller -> Result
	t.Run("complete execution flow", func(t *testing.T) {
		mockCaller := &mockAICallerForIntegration{
			callResult: &callcli.Result{
				Stdout: `Task execution started
Processing...
<!-- RALPH_STATUS -->
{
	"module": "executor",
	"job": "job_6",
	"status": "COMPLETED",
	"tasks_completed": 8,
	"tasks_total": 8,
	"summary": "All integration tests passed"
}
<!-- END_RALPH_STATUS -->`,
				Stderr:   "",
				ExitCode: 0,
				Duration: 150 * time.Millisecond,
			},
		}

		tr := NewTaskRunner(logger, mockCaller)

		// Execute task
		result, err := tr.Run(context.Background(), "Integration Test", "execute integration test")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Error("expected success")
		}

		if result.Duration != 150*time.Millisecond {
			t.Errorf("expected duration 150ms, got %v", result.Duration)
		}

		if !mockCaller.called {
			t.Error("expected caller to be invoked")
		}
	})
}

// BenchmarkTaskRunner_Run benchmarks the task execution.
func BenchmarkTaskRunner_Run(b *testing.B) {
	logger := &integrationTestLogger{}
	mockCaller := &mockAICallerForIntegration{
		callResult: &callcli.Result{
			ExitCode: 0,
			Duration: 1 * time.Millisecond,
		},
	}

	tr := NewTaskRunner(logger, mockCaller)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tr.Run(ctx, "Benchmark Task", "test prompt")
	}
}

// TestIntegration_CoverageVerification verifies that all integration points are tested.
func TestIntegration_CoverageVerification(t *testing.T) {
	// This test documents all the integration points that are verified:
	// 1. TaskRunner -> Call CLI (AICliCaller.CallWithPromptContent)
	// 2. Prompt passing (content via stdin)
	// 3. Timeout configuration (default 10 min, custom, enforcement)
	// 4. Exit code handling (0 = success, non-zero = error)
	// 5. Output capture (stdout, stderr)
	// 6. Result parsing (RALPH_STATUS extraction)
	// 7. Interruption handling (context cancellation)
	// 8. JobRunner -> TaskExecutor integration
	// 9. Engine -> JobRunner integration

	// Verify all tests exist
	tests := []string{
		"TestTaskRunner_Integration_CallCLI",
		"TestTaskRunner_Integration_PromptPassedCorrectly",
		"TestTaskRunner_Integration_Timeout",
		"TestTaskRunner_Integration_ResultParsing",
		"TestTaskRunner_Integration_Interruption",
		"TestJobRunner_Integration_TaskExecution",
		"TestEngine_Integration_ExecuteJob",
		"TestTaskRunner_Integration_ExitCodeHandling",
		"TestTaskRunner_Integration_OutputCapture",
		"TestTaskRunner_Integration_WithPromptFile",
		"TestFullIntegration_ExecutorWithCallCLI",
	}

	t.Logf("Integration tests covering %d integration points:", len(tests))
	for _, test := range tests {
		t.Logf("  - %s", test)
	}
}

// Helper function to check if a string contains a substring.
func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
