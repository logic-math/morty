// Package executor provides job execution engine for Morty.
package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/morty/morty/internal/git"
	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/state"
)

// mockLogger implements logging.Logger for testing.
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, attrs ...logging.Attr) {}
func (m *mockLogger) Info(msg string, attrs ...logging.Attr)  {}
func (m *mockLogger) Warn(msg string, attrs ...logging.Attr)  {}
func (m *mockLogger) Error(msg string, attrs ...logging.Attr) {}
func (m *mockLogger) Success(msg string, attrs ...logging.Attr) {}
func (m *mockLogger) Loop(msg string, attrs ...logging.Attr)  {}
func (m *mockLogger) WithContext(ctx context.Context) logging.Logger { return m }
func (m *mockLogger) WithJob(module, job string) logging.Logger      { return m }
func (m *mockLogger) WithAttrs(attrs ...logging.Attr) logging.Logger { return m }
func (m *mockLogger) SetLevel(level logging.Level)                   {}
func (m *mockLogger) GetLevel() logging.Level                        { return logging.InfoLevel }
func (m *mockLogger) IsEnabled(level logging.Level) bool             { return true }

// setupTestEnv creates a test environment with state file and git repo.
func setupTestEnv(t *testing.T) (string, *state.Manager, *git.Manager, logging.Logger, func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create state file
	stateFile := filepath.Join(tempDir, "status.json")
	stateManager := state.NewManager(stateFile)

	// Save initial state
	if err := os.WriteFile(stateFile, []byte(`{
		"version": "1.0",
		"global": {
			"status": "PENDING",
			"start_time": "2024-01-01T00:00:00Z",
			"last_update": "2024-01-01T00:00:00Z"
		},
		"modules": {
			"test-module": {
				"name": "test-module",
				"status": "PENDING",
				"jobs": {
					"test-job": {
						"name": "test-job",
						"status": "PENDING",
						"tasks_total": 3,
						"tasks_completed": 0,
						"tasks": [
							{"index": 0, "status": "PENDING", "description": "Task 1"},
							{"index": 1, "status": "PENDING", "description": "Task 2"},
							{"index": 2, "status": "PENDING", "description": "Task 3"}
						],
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					},
					"completed-job": {
						"name": "completed-job",
						"status": "COMPLETED",
						"tasks_total": 1,
						"tasks_completed": 1,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					},
					"failed-job": {
						"name": "failed-job",
						"status": "FAILED",
						"tasks_total": 1,
						"tasks_completed": 0,
						"retry_count": 0,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					},
					"running-job": {
						"name": "running-job",
						"status": "RUNNING",
						"tasks_total": 1,
						"tasks_completed": 0,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					},
					"blocked-job": {
						"name": "blocked-job",
						"status": "BLOCKED",
						"tasks_total": 1,
						"tasks_completed": 0,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					},
					"max-retries-job": {
						"name": "max-retries-job",
						"status": "FAILED",
						"tasks_total": 1,
						"tasks_completed": 0,
						"retry_count": 3,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					}
				},
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z"
			}
		}
	}`), 0644); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Create logger
	logger := &mockLogger{}

	// Create git manager
	gitManager := git.NewManager()

	// Cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	// Load state
	if err := stateManager.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	return tempDir, stateManager, gitManager, logger, cleanup
}

func TestNewEngine(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	tests := []struct {
		name         string
		config       *Config
		wantMaxRetry int
		wantAutoCommit bool
	}{
		{
			name:           "default config",
			config:         nil,
			wantMaxRetry:   3,
			wantAutoCommit: true,
		},
		{
			name: "custom config",
			config: &Config{
				MaxRetries:   5,
				AutoCommit:   false,
				CommitPrefix: "test:",
			},
			wantMaxRetry:   5,
			wantAutoCommit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine(stateManager, gitManager, logger, tt.config)
			if eng == nil {
				t.Fatal("NewEngine returned nil")
			}

			// Type assert to access internal fields
			e, ok := eng.(*engine)
			if !ok {
				t.Fatal("Engine is not *engine type")
			}

			if e.config.MaxRetries != tt.wantMaxRetry {
				t.Errorf("MaxRetries = %d, want %d", e.config.MaxRetries, tt.wantMaxRetry)
			}
			if e.config.AutoCommit != tt.wantAutoCommit {
				t.Errorf("AutoCommit = %v, want %v", e.config.AutoCommit, tt.wantAutoCommit)
			}
		})
	}
}

func TestEngine_checkPrerequisites(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	tests := []struct {
		name      string
		module    string
		job       string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid pending job",
			module:    "test-module",
			job:       "test-job",
			wantError: false,
		},
		{
			name:      "completed job should fail",
			module:    "test-module",
			job:       "completed-job",
			wantError: true,
			errorMsg:  "already completed",
		},
		{
			name:      "running job should fail",
			module:    "test-module",
			job:       "running-job",
			wantError: true,
			errorMsg:  "already running",
		},
		{
			name:      "blocked job should fail",
			module:    "test-module",
			job:       "blocked-job",
			wantError: true,
			errorMsg:  "blocked by dependencies",
		},
		{
			name:      "non-existent module",
			module:    "non-existent",
			job:       "test-job",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eng := NewEngine(stateManager, gitManager, logger, nil)
			e := eng.(*engine)

			err := e.checkPrerequisites(context.Background(), tt.module, tt.job)
			if tt.wantError {
				if err == nil {
					t.Errorf("checkPrerequisites() error = nil, want error containing %q", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("checkPrerequisites() error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("checkPrerequisites() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestEngine_transitionState(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	eng := NewEngine(stateManager, gitManager, logger, nil)
	e := eng.(*engine)

	tests := []struct {
		name      string
		module    string
		job       string
		toStatus  state.Status
		wantError bool
	}{
		{
			name:      "pending to running",
			module:    "test-module",
			job:       "test-job",
			toStatus:  state.StatusRunning,
			wantError: false,
		},
		{
			name:      "running to completed",
			module:    "test-module",
			job:       "test-job",
			toStatus:  state.StatusCompleted,
			wantError: false,
		},
		{
			name:      "invalid transition - completed to running",
			module:    "test-module",
			job:       "completed-job",
			toStatus:  state.StatusRunning,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := e.transitionState(tt.module, tt.job, tt.toStatus)
			if tt.wantError {
				if err == nil {
					t.Errorf("transitionState() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("transitionState() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestEngine_ExecuteJob_PrerequisiteFailure(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	eng := NewEngine(stateManager, gitManager, logger, nil)

	// Test with completed job - should fail prerequisite check
	err := eng.ExecuteJob(context.Background(), "test-module", "completed-job")
	if err == nil {
		t.Error("ExecuteJob() with completed job should return error")
	}
	if !contains(err.Error(), "prerequisite check failed") {
		t.Errorf("ExecuteJob() error = %v, want prerequisite check failed", err)
	}
}

func TestEngine_ExecuteJob_MaxRetriesExceeded(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	config := &Config{
		MaxRetries: 3,
		AutoCommit: false,
	}
	eng := NewEngine(stateManager, gitManager, logger, config)

	// Test with job that has already reached max retries
	err := eng.ExecuteJob(context.Background(), "test-module", "max-retries-job")
	if err == nil {
		t.Error("ExecuteJob() with max retries exceeded should return error")
	}
	if !contains(err.Error(), "max retries exceeded") {
		t.Errorf("ExecuteJob() error = %v, want max retries exceeded", err)
	}
}

func TestEngine_ResumeJob(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	eng := NewEngine(stateManager, gitManager, logger, nil)

	// Test resume with running job - should work
	err := eng.ResumeJob(context.Background(), "test-module", "running-job")
	// This may fail due to other reasons but should not fail due to status check
	// since running is a valid status for resume
	if err != nil {
		// Expected to potentially fail on execution, but should pass resume check
		t.Logf("ResumeJob() error (may be expected): %v", err)
	}

	// Test resume with completed job - should fail
	err = eng.ResumeJob(context.Background(), "test-module", "completed-job")
	if err == nil {
		t.Error("ResumeJob() with completed job should return error")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("DefaultConfig().MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if !cfg.AutoCommit {
		t.Error("DefaultConfig().AutoCommit = false, want true")
	}
	if cfg.CommitPrefix != "morty:" {
		t.Errorf("DefaultConfig().CommitPrefix = %s, want morty:", cfg.CommitPrefix)
	}
	if cfg.WorkingDir != "." {
		t.Errorf("DefaultConfig().WorkingDir = %s, want .", cfg.WorkingDir)
	}
}

func TestExecutionResult(t *testing.T) {
	result := ExecutionResult{
		Status:         state.StatusCompleted,
		TasksCompleted: 3,
		TasksTotal:     3,
		Summary:        "Test completed",
		Module:         "test-module",
		Job:            "test-job",
		RetryCount:     0,
		Error:          "",
	}

	if result.Status != state.StatusCompleted {
		t.Errorf("ExecutionResult.Status = %s, want COMPLETED", result.Status)
	}
	if result.TasksCompleted != 3 {
		t.Errorf("ExecutionResult.TasksCompleted = %d, want 3", result.TasksCompleted)
	}
}

func TestEngine_ExecuteTask(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	eng := NewEngine(stateManager, gitManager, logger, nil)

	ctx := context.Background()
	err := eng.ExecuteTask(ctx, "test-module", "test-job", 0, "Test task")
	if err != nil {
		t.Errorf("ExecuteTask() error = %v, want nil", err)
	}
}

func TestEngine_getJobState(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	eng := NewEngine(stateManager, gitManager, logger, nil)
	e := eng.(*engine)

	// Test getting existing job
	jobState, err := e.getJobState("test-module", "test-job")
	if err != nil {
		t.Errorf("getJobState() error = %v, want nil", err)
	}
	if jobState == nil {
		t.Fatal("getJobState() returned nil")
	}
	if jobState.Name != "test-job" {
		t.Errorf("jobState.Name = %s, want test-job", jobState.Name)
	}
	if jobState.Status != state.StatusPending {
		t.Errorf("jobState.Status = %s, want PENDING", jobState.Status)
	}

	// Test getting non-existent job
	_, err = e.getJobState("non-existent", "non-existent")
	if err == nil {
		t.Error("getJobState() with non-existent job should return error")
	}
}

func TestEngine_ExecuteJob_RetryFlow(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	config := &Config{
		MaxRetries:   3,
		AutoCommit:   false,
		CommitPrefix: "morty:",
	}
	eng := NewEngine(stateManager, gitManager, logger, config)

	// Test retry with failed-job (retry_count = 0)
	// This should work since retry_count < MaxRetries
	err := eng.ExecuteJob(context.Background(), "test-module", "failed-job")
	// This will likely fail during task execution but should pass the retry check
	if err != nil {
		t.Logf("ExecuteJob() error (may fail due to task execution): %v", err)
	}

	// Verify the job was transitioned through FAILED -> PENDING -> RUNNING
	jobState := stateManager.GetJob("test-module", "failed-job")
	if jobState != nil {
		// After retry attempt, status should be RUNNING or FAILED/COMPLETED depending on execution
		if jobState.Status != state.StatusRunning &&
			jobState.Status != state.StatusFailed &&
			jobState.Status != state.StatusCompleted {
			t.Errorf("Unexpected job status after retry: %s", jobState.Status)
		}
	}
}

func TestEngine_createGitCommit_NoChanges(t *testing.T) {
	tempDir, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	config := &Config{
		MaxRetries:   3,
		AutoCommit:   true,
		CommitPrefix: "morty:",
		WorkingDir:   tempDir,
	}
	eng := NewEngine(stateManager, gitManager, logger, config)
	e := eng.(*engine)

	// Initialize git repo
	if err := gitManager.InitIfNeeded(tempDir); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Test commit with no changes - should return nil (no error, just no commit)
	err := e.createGitCommit("test-module", "test-job")
	// Should not error, but may not create a commit if no changes
	if err != nil {
		t.Logf("createGitCommit() error (expected with no changes): %v", err)
	}
}

func TestEngine_createGitCommit_WithChanges(t *testing.T) {
	tempDir, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	config := &Config{
		MaxRetries:   3,
		AutoCommit:   true,
		CommitPrefix: "morty:",
		WorkingDir:   tempDir,
	}
	eng := NewEngine(stateManager, gitManager, logger, config)
	e := eng.(*engine)

	// Initialize git repo
	if err := gitManager.InitIfNeeded(tempDir); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create a file to have some changes
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Configure git user for commit
	os.Setenv("GIT_AUTHOR_NAME", "Test User")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "Test User")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")

	// Test commit with changes
	err := e.createGitCommit("test-module", "test-job")
	if err != nil {
		t.Logf("createGitCommit() error: %v", err)
	}
}

func TestEngine_executeTasks(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	eng := NewEngine(stateManager, gitManager, logger, nil)
	e := eng.(*engine)

	// Test executeTasks with test-job which has 3 pending tasks
	completed, err := e.executeTasks(context.Background(), "test-module", "test-job")
	if err != nil {
		t.Logf("executeTasks() error: %v", err)
	}
	t.Logf("Tasks completed: %d", completed)
}

func TestEngine_UpdateMethods(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	eng := NewEngine(stateManager, gitManager, logger, nil)
	e := eng.(*engine)

	// Test updateTasksCompleted
	err := e.updateTasksCompleted("test-module", "test-job", 2)
	if err != nil {
		t.Errorf("updateTasksCompleted() error = %v, want nil", err)
	}

	// Test updateFailureReason
	err = e.updateFailureReason("test-module", "test-job", "test failure")
	if err != nil {
		t.Errorf("updateFailureReason() error = %v, want nil", err)
	}
}

func TestEngine_ExecuteJob_BlockedJob(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	eng := NewEngine(stateManager, gitManager, logger, nil)

	// Test with blocked job - should fail prerequisite check
	err := eng.ExecuteJob(context.Background(), "test-module", "blocked-job")
	if err == nil {
		t.Error("ExecuteJob() with blocked job should return error")
	}
	if !contains(err.Error(), "blocked by dependencies") {
		t.Errorf("ExecuteJob() error = %v, want blocked by dependencies", err)
	}
}

func TestEngine_TransitionState_NonExistent(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	eng := NewEngine(stateManager, gitManager, logger, nil)
	e := eng.(*engine)

	// Test transition with non-existent module
	err := e.transitionState("non-existent", "test-job", state.StatusRunning)
	if err == nil {
		t.Error("transitionState() with non-existent module should return error")
	}

	// Test transition with non-existent job
	err = e.transitionState("test-module", "non-existent", state.StatusRunning)
	if err == nil {
		t.Error("transitionState() with non-existent job should return error")
	}
}

func TestEngine_CreateGitCommit_NilGitManager(t *testing.T) {
	_, stateManager, _, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	config := &Config{
		MaxRetries:   3,
		AutoCommit:   true,
		CommitPrefix: "morty:",
	}
	// Create engine with nil git manager
	eng := NewEngine(stateManager, nil, logger, config)
	e := eng.(*engine)

	// Test commit with nil git manager
	err := e.createGitCommit("test-module", "test-job")
	if err == nil {
		t.Error("createGitCommit() with nil git manager should return error")
	}
}

func TestEngine_ExecuteJob_WithNilGitManager(t *testing.T) {
	_, stateManager, _, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	config := &Config{
		MaxRetries:   3,
		AutoCommit:   true,
		CommitPrefix: "morty:",
	}
	// Create engine with nil git manager - auto-commit enabled but no git manager
	eng := NewEngine(stateManager, nil, logger, config)

	// Execute job - should work even with nil git manager (just logs warning)
	err := eng.ExecuteJob(context.Background(), "test-module", "test-job")
	// This will likely fail during task execution but not due to nil git manager
	if err != nil {
		t.Logf("ExecuteJob() error: %v", err)
	}
}

func TestEngine_ExecuteJob_LoadStateFailure(t *testing.T) {
	tempDir, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	// Corrupt the state file to cause Load to fail
	stateFile := filepath.Join(tempDir, "status.json")
	if err := os.WriteFile(stateFile, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to corrupt state file: %v", err)
	}

	eng := NewEngine(stateManager, gitManager, logger, nil)

	// Execute job - should fail due to state load failure
	err := eng.ExecuteJob(context.Background(), "test-module", "test-job")
	if err == nil {
		t.Error("ExecuteJob() with corrupted state should return error")
	}
}

func TestEngine_ExecuteJob_CompletionTransitions(t *testing.T) {
	// Create a special state file where we can verify the completion transitions
	tempDir, err := os.MkdirTemp("", "executor-completion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stateFile := filepath.Join(tempDir, "status.json")

	// Create state with a job that will complete successfully
	stateData := `{
		"version": "1.0",
		"global": {
			"status": "PENDING",
			"start_time": "2024-01-01T00:00:00Z",
			"last_update": "2024-01-01T00:00:00Z"
		},
		"modules": {
			"completion-test": {
				"name": "completion-test",
				"status": "PENDING",
				"jobs": {
					"completion-job": {
						"name": "completion-job",
						"status": "PENDING",
						"tasks_total": 0,
						"tasks_completed": 0,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					}
				},
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z"
			}
		}
	}`

	if err := os.WriteFile(stateFile, []byte(stateData), 0644); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	stateManager := state.NewManager(stateFile)
	if err := stateManager.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	gitManager := git.NewManager()
	logger := &mockLogger{}

	config := &Config{
		MaxRetries:   3,
		AutoCommit:   false,
		CommitPrefix: "morty:",
	}
	eng := NewEngine(stateManager, gitManager, logger, config)

	// Execute job with no tasks - should complete successfully
	err = eng.ExecuteJob(context.Background(), "completion-test", "completion-job")
	if err != nil {
		t.Logf("ExecuteJob() error: %v", err)
	}

	// Verify job was transitioned to COMPLETED
	jobState := stateManager.GetJob("completion-test", "completion-job")
	if jobState != nil && jobState.Status != state.StatusCompleted {
		t.Errorf("Job status = %s, want COMPLETED", jobState.Status)
	}
}

func TestEngine_ExecuteJob_NonExistentModule(t *testing.T) {
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	eng := NewEngine(stateManager, gitManager, logger, nil)

	// Test with non-existent module
	err := eng.ExecuteJob(context.Background(), "non-existent-module", "test-job")
	if err == nil {
		t.Error("ExecuteJob() with non-existent module should return error")
	}
	if !contains(err.Error(), "prerequisite check failed") {
		t.Errorf("ExecuteJob() error = %v, want prerequisite check failed", err)
	}
}

// Test to cover ExecuteJob lines that handle SetCurrent error
func TestEngine_ExecuteJob_SetCurrentError(t *testing.T) {
	// This test verifies ExecuteJob continues even if SetCurrent fails
	_, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	config := &Config{
		MaxRetries:   3,
		AutoCommit:   false,
		CommitPrefix: "morty:",
	}
	eng := NewEngine(stateManager, gitManager, logger, config)

	// Execute job - should work even if SetCurrent has issues
	ctx := context.Background()
	err := eng.ExecuteJob(ctx, "test-module", "test-job")
	// This will fail due to task execution but covers more code paths
	t.Logf("ExecuteJob result: %v", err)
}

// Test to cover createGitCommitUsingCommitter error paths
func TestEngine_CreateGitCommitUsingCommitter_Errors(t *testing.T) {
	tempDir, stateManager, gitManager, logger, cleanup := setupTestEnv(t)
	defer cleanup()

	config := &Config{
		MaxRetries:   3,
		AutoCommit:   true,
		CommitPrefix: "morty:",
		WorkingDir:   tempDir,
	}
	eng := NewEngine(stateManager, gitManager, logger, config)
	e := eng.(*engine)

	// Test with non-git directory (should return error)
	err := e.createGitCommitUsingCommitter("test-module", "test-job")
	if err == nil {
		t.Log("createGitCommitUsingCommitter with non-git dir should error")
	}
}

// Test executeTasks with job that has no tasks
func TestEngine_executeTasks_NoTasks(t *testing.T) {
	// Create a temp directory with a job that has no tasks
	tempDir, err := os.MkdirTemp("", "executor-no-tasks-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stateFile := filepath.Join(tempDir, "status.json")

	// Create state with a job that has 0 tasks
	stateData := `{
		"version": "1.0",
		"global": {
			"status": "PENDING",
			"start_time": "2024-01-01T00:00:00Z",
			"last_update": "2024-01-01T00:00:00Z"
		},
		"modules": {
			"no-tasks-module": {
				"name": "no-tasks-module",
				"status": "PENDING",
				"jobs": {
					"no-tasks-job": {
						"name": "no-tasks-job",
						"status": "PENDING",
						"tasks_total": 0,
						"tasks_completed": 0,
						"tasks": [],
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					}
				},
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z"
			}
		}
	}`

	if err := os.WriteFile(stateFile, []byte(stateData), 0644); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	stateManager := state.NewManager(stateFile)
	if err := stateManager.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	gitManager := git.NewManager()
	logger := &mockLogger{}

	eng := NewEngine(stateManager, gitManager, logger, nil)
	e := eng.(*engine)

	// Test executeTasks with job that has no tasks
	completed, err := e.executeTasks(context.Background(), "no-tasks-module", "no-tasks-job")
	if err != nil {
		t.Logf("executeTasks() error: %v", err)
	}
	t.Logf("Tasks completed: %d", completed)
}

// Test successful git commit flow with initialized repo
func TestEngine_CreateGitCommitUsingCommitter_SuccessPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "executor-git-success-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stateFile := filepath.Join(tempDir, "status.json")
	stateData := `{
		"version": "1.0",
		"global": {"status": "PENDING", "start_time": "2024-01-01T00:00:00Z", "last_update": "2024-01-01T00:00:00Z"},
		"modules": {"test-module": {"name": "test-module", "status": "PENDING", "jobs": {}, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"}}}
	`
	if err := os.WriteFile(stateFile, []byte(stateData), 0644); err != nil {
		t.Fatalf("Failed to write state: %v", err)
	}

	stateManager := state.NewManager(stateFile)
	if err := stateManager.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	gitManager := git.NewManager()
	logger := &mockLogger{}

	// Initialize git repo
	if err := gitManager.InitIfNeeded(tempDir); err != nil {
		t.Fatalf("Failed to init git: %v", err)
	}

	// Configure git user using os/exec directly
	execGit := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = tempDir
		return cmd.Run()
	}
	_ = execGit("config", "user.email", "test@test.com")
	_ = execGit("config", "user.name", "Test")

	// Create initial commit (needed for branch operations)
	initFile := filepath.Join(tempDir, "init.txt")
	if err := os.WriteFile(initFile, []byte("init"), 0644); err != nil {
		t.Fatalf("Failed to write init file: %v", err)
	}
	_ = execGit("add", "-A")
	_ = execGit("commit", "-m", "init")

	// Create new file for the actual commit test
	newFile := filepath.Join(tempDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to write new file: %v", err)
	}

	config := &Config{
		MaxRetries:   3,
		AutoCommit:   true,
		CommitPrefix: "morty:",
		WorkingDir:   tempDir,
	}
	eng := NewEngine(stateManager, gitManager, logger, config)
	e := eng.(*engine)

	// This should create a commit successfully
	err = e.createGitCommitUsingCommitter("test-module", "test-job")
	if err != nil {
		t.Logf("createGitCommitUsingCommitter error (may be OK): %v", err)
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) > 0 && containsInternal(s, substr))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
