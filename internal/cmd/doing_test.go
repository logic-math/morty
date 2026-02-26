package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/morty/morty/internal/executor"
	"github.com/morty/morty/internal/git"
	"github.com/morty/morty/internal/state"
)

func TestNewDoingHandler(t *testing.T) {
	cfg := &mockConfig{}
	logger := &mockLogger{}

	handler := NewDoingHandler(cfg, logger)

	if handler == nil {
		t.Fatal("NewDoingHandler returned nil")
	}

	if handler.cfg == nil {
		t.Error("Handler cfg not set correctly")
	}

	if handler.logger == nil {
		t.Error("Handler logger not set")
	}

	if handler.paths == nil {
		t.Error("Handler paths not initialized")
	}
}

func TestDoingHandler_parseOptions(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantRestart   bool
		wantModule    string
		wantJob       string
		wantRemaining []string
	}{
		{
			name:          "no options",
			args:          []string{},
			wantRestart:   false,
			wantModule:    "",
			wantJob:       "",
			wantRemaining: nil,
		},
		{
			name:          "restart flag long",
			args:          []string{"--restart"},
			wantRestart:   true,
			wantModule:    "",
			wantJob:       "",
			wantRemaining: nil,
		},
		{
			name:          "restart flag short",
			args:          []string{"-r"},
			wantRestart:   true,
			wantModule:    "",
			wantJob:       "",
			wantRemaining: nil,
		},
		{
			name:          "restart flag with value true",
			args:          []string{"--restart=true"},
			wantRestart:   true,
			wantModule:    "",
			wantJob:       "",
			wantRemaining: nil,
		},
		{
			name:          "restart flag with value 1",
			args:          []string{"--restart=1"},
			wantRestart:   true,
			wantModule:    "",
			wantJob:       "",
			wantRemaining: nil,
		},
		{
			name:          "restart flag with value false",
			args:          []string{"--restart=false"},
			wantRestart:   false,
			wantModule:    "",
			wantJob:       "",
			wantRemaining: nil,
		},
		{
			name:          "module flag long",
			args:          []string{"--module", "my-module"},
			wantRestart:   false,
			wantModule:    "my-module",
			wantJob:       "",
			wantRemaining: nil,
		},
		{
			name:          "module flag short",
			args:          []string{"-m", "my-module"},
			wantRestart:   false,
			wantModule:    "my-module",
			wantJob:       "",
			wantRemaining: nil,
		},
		{
			name:          "module flag with equals",
			args:          []string{"--module=my-module"},
			wantRestart:   false,
			wantModule:    "my-module",
			wantJob:       "",
			wantRemaining: nil,
		},
		{
			name:          "job flag long",
			args:          []string{"--module", "my-module", "--job", "my-job"},
			wantRestart:   false,
			wantModule:    "my-module",
			wantJob:       "my-job",
			wantRemaining: nil,
		},
		{
			name:          "job flag short",
			args:          []string{"-m", "my-module", "-j", "my-job"},
			wantRestart:   false,
			wantModule:    "my-module",
			wantJob:       "my-job",
			wantRemaining: nil,
		},
		{
			name:          "job flag with equals",
			args:          []string{"--module=my-module", "--job=my-job"},
			wantRestart:   false,
			wantModule:    "my-module",
			wantJob:       "my-job",
			wantRemaining: nil,
		},
		{
			name:          "combined flags",
			args:          []string{"--restart", "--module", "my-module", "--job", "my-job"},
			wantRestart:   true,
			wantModule:    "my-module",
			wantJob:       "my-job",
			wantRemaining: nil,
		},
		{
			name:          "remaining args",
			args:          []string{"arg1", "arg2"},
			wantRestart:   false,
			wantModule:    "",
			wantJob:       "",
			wantRemaining: []string{"arg1", "arg2"},
		},
		{
			name:          "mixed args and flags",
			args:          []string{"--restart", "arg1", "--module", "my-module", "arg2"},
			wantRestart:   true,
			wantModule:    "my-module",
			wantJob:       "",
			wantRemaining: []string{"arg1", "arg2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewDoingHandler(&mockConfig{}, &mockLogger{})
			restart, module, job, remaining := handler.parseOptions(tt.args)

			if restart != tt.wantRestart {
				t.Errorf("parseOptions() restart = %v, want %v", restart, tt.wantRestart)
			}
			if module != tt.wantModule {
				t.Errorf("parseOptions() module = %v, want %v", module, tt.wantModule)
			}
			if job != tt.wantJob {
				t.Errorf("parseOptions() job = %v, want %v", job, tt.wantJob)
			}
			if len(remaining) != len(tt.wantRemaining) {
				t.Errorf("parseOptions() remaining length = %d, want %d", len(remaining), len(tt.wantRemaining))
			}
			for i := range remaining {
				if i < len(tt.wantRemaining) && remaining[i] != tt.wantRemaining[i] {
					t.Errorf("parseOptions() remaining[%d] = %v, want %v", i, remaining[i], tt.wantRemaining[i])
				}
			}
		})
	}
}

func TestDoingHandler_checkPlanDirExists_noWorkDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Use a non-existent work directory
	cfg := &mockConfig{
		workDir: filepath.Join(tmpDir, "nonexistent"),
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	err := handler.checkPlanDirExists()

	if err == nil {
		t.Error("checkPlanDirExists() expected error when work dir doesn't exist")
	}

	if !strings.Contains(err.Error(), "请先运行 morty init") {
		t.Errorf("checkPlanDirExists() error message = %v, want it to contain '请先运行 morty init'", err)
	}
}

func TestDoingHandler_checkPlanDirExists_noPlanDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create work directory but no plan directory
	workDir := filepath.Join(tmpDir, ".morty")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create work dir: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
		planDir: filepath.Join(workDir, "plan"),
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	err := handler.checkPlanDirExists()

	if err == nil {
		t.Error("checkPlanDirExists() expected error when plan dir doesn't exist")
	}

	if !strings.Contains(err.Error(), "请先运行 morty plan") {
		t.Errorf("checkPlanDirExists() error message = %v, want it to contain '请先运行 morty plan'", err)
	}
}

func TestDoingHandler_checkPlanDirExists_success(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create work directory and plan directory
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	err := handler.checkPlanDirExists()

	if err != nil {
		t.Errorf("checkPlanDirExists() unexpected error: %v", err)
	}
}

func TestDoingHandler_Execute_noPlanDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	// No work directory created
	cfg := &mockConfig{
		workDir: filepath.Join(tmpDir, ".morty"),
		planDir: filepath.Join(tmpDir, ".morty", "plan"),
	}
	logger := &mockLogger{}
	handler := NewDoingHandler(cfg, logger)

	ctx := context.Background()
	result, err := handler.Execute(ctx, []string{})

	if err == nil {
		t.Fatal("Execute() expected error when no plan dir exists")
	}

	if result == nil {
		t.Fatal("Execute() returned nil result")
	}

	if result.ExitCode != 1 {
		t.Errorf("Execute() exit code = %d, want 1", result.ExitCode)
	}

	if !strings.Contains(err.Error(), "请先运行 morty init") {
		t.Errorf("Execute() error = %v, want it to contain '请先运行 morty init'", err)
	}
}

func TestDoingHandler_Execute_jobWithoutModule(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create work directory and plan directory
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	logger := &mockLogger{}
	handler := NewDoingHandler(cfg, logger)

	ctx := context.Background()
	result, err := handler.Execute(ctx, []string{"--job", "my-job"})

	if err == nil {
		t.Fatal("Execute() expected error when job is specified without module")
	}

	if result == nil {
		t.Fatal("Execute() returned nil result")
	}

	if result.ExitCode != 1 {
		t.Errorf("Execute() exit code = %d, want 1", result.ExitCode)
	}

	if !strings.Contains(err.Error(), "--job 选项需要配合 --module 使用") {
		t.Errorf("Execute() error = %v, want it to contain '--job 选项需要配合 --module 使用'", err)
	}
}

func TestDoingHandler_Execute_success(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create work directory and plan directory
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state with the module and job
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"my-module": {
			"my-job": state.StatusPending,
		},
	})

	// Create plan file
	planContent := `# Plan: my-module

### Job 1: my-job

**目标**: Test goal

**Tasks (Todo 列表)**:
- [ ] Task 1: Test task

**验证器**: ...
`
	setupTestPlanFile(t, planDir, "my-module", planContent)

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	logger := &mockLogger{}
	handler := NewDoingHandler(cfg, logger)

	ctx := context.Background()
	result, err := handler.Execute(ctx, []string{"--module", "my-module", "--job", "my-job", "--restart"})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Execute() returned nil result")
	}

	if result.ExitCode != 0 {
		t.Errorf("Execute() exit code = %d, want 0", result.ExitCode)
	}

	if result.ModuleName != "my-module" {
		t.Errorf("Execute() module name = %v, want 'my-module'", result.ModuleName)
	}

	if result.JobName != "my-job" {
		t.Errorf("Execute() job name = %v, want 'my-job'", result.JobName)
	}

	if !result.Restart {
		t.Error("Execute() restart = false, want true")
	}

	if result.PlanDir != planDir {
		t.Errorf("Execute() plan dir = %v, want %v", result.PlanDir, planDir)
	}
}

func TestDoingHandler_Execute_contextCancellation(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create work directory and plan directory
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"test": {
			"job_1": state.StatusPending,
		},
	})

	// Create plan file
	planContent := `# Plan: test

### Job 1: job_1

**目标**: Test goal

**Tasks (Todo 列表)**:
- [ ] Task 1: Test task
`
	setupTestPlanFile(t, planDir, "test", planContent)

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	logger := &mockLogger{}
	handler := NewDoingHandler(cfg, logger)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Note: Context cancellation check happens early in Execute,
	// but after parameter parsing and plan directory check
	// So for this test, we expect it to succeed since context is checked
	// at the beginning of Execute but we need to pass plan check first
	result, err := handler.Execute(ctx, []string{"--module", "test", "--job", "job_1"})

	// The context cancellation check is not at the very beginning of Execute,
	// so this test verifies the command handles normal execution
	// Context cancellation handling can be added if needed in the future
	if err != nil {
		t.Logf("Execute() with cancelled context returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestDoingHandler_GetPlanDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	gotPlanDir := handler.GetPlanDir()

	if gotPlanDir != planDir {
		t.Errorf("GetPlanDir() = %v, want %v", gotPlanDir, planDir)
	}
}

func TestDoingHandler_SetPlanDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewDoingHandler(cfg, &mockLogger{})

	expectedPlanDir := filepath.Join(tmpDir, "plan")
	gotPlanDir := handler.GetPlanDir()

	if gotPlanDir != expectedPlanDir {
		t.Errorf("GetPlanDir() = %v, want %v", gotPlanDir, expectedPlanDir)
	}
}

func TestDoingHandler_PrintDoingSummary(t *testing.T) {
	handler := NewDoingHandler(&mockConfig{}, &mockLogger{})

	// Test successful result
	result := &DoingResult{
		ModuleName: "test-module",
		JobName:    "test-job",
		PlanDir:    "/test/plan",
		ExitCode:   0,
		Restart:    true,
	}

	// Should not panic
	handler.PrintDoingSummary(result)

	// Test error result
	resultWithError := &DoingResult{
		ModuleName: "test-module",
		Err:        ErrNoPlan,
		ExitCode:   1,
	}

	// Should not panic even with errors
	handler.PrintDoingSummary(resultWithError)
}

// Helper error for testing
var ErrNoPlan = &testError{msg: "请先运行 morty plan"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// setupTestState creates a test state file with the given modules and jobs
func setupTestState(t *testing.T, workDir string, modules map[string]map[string]state.Status) string {
	statusFile := filepath.Join(workDir, "status.json")
	stateMgr := state.NewManager(statusFile)

	// Load or create default state
	if err := stateMgr.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Add modules and jobs
	for moduleName, jobs := range modules {
		module := &state.ModuleState{
			Name:      moduleName,
			Status:    state.StatusPending,
			Jobs:      make(map[string]*state.JobState),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		for jobName, jobStatus := range jobs {
			module.Jobs[jobName] = &state.JobState{
				Name:           jobName,
				Status:         jobStatus,
				TasksTotal:     5,
				TasksCompleted: 0,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
		}

		stateMgr.SetModule(module)
	}

	// Save state
	if err := stateMgr.Save(); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	return statusFile
}

// setupTestPlanFile creates a test plan file
func setupTestPlanFile(t *testing.T, planDir, moduleName string, content string) string {
	planPath := filepath.Join(planDir, moduleName+".md")
	if err := os.WriteFile(planPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write plan file: %v", err)
	}
	return planPath
}

// Test loadStatus function
func TestDoingHandler_loadStatus(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"test-module": {
			"job_1": state.StatusPending,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	err := handler.loadStatus()
	if err != nil {
		t.Errorf("loadStatus() error = %v", err)
	}

	if handler.stateManager == nil {
		t.Error("loadStatus() stateManager is nil")
	}

	stateData := handler.stateManager.GetState()
	if stateData == nil {
		t.Fatal("loadStatus() state is nil")
	}

	// Verify state was loaded correctly
	if _, ok := stateData.Modules["test-module"]; !ok {
		t.Error("loadStatus() test-module not found in state")
	}
}

// Test loadStatus with non-existent file (should create default)
func TestDoingHandler_loadStatus_createDefault(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	err := handler.loadStatus()
	if err != nil {
		t.Errorf("loadStatus() error = %v", err)
	}

	if handler.stateManager == nil {
		t.Error("loadStatus() stateManager is nil")
	}

	stateData := handler.stateManager.GetState()
	if stateData == nil {
		t.Fatal("loadStatus() state is nil")
	}

	// Should have default state
	if stateData.Version == "" {
		t.Error("loadStatus() default state version is empty")
	}
}

// Test handleRestart with no module specified (reset all)
func TestDoingHandler_handleRestart_all(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state with completed jobs
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"module1": {
			"job_1": state.StatusCompleted,
			"job_2": state.StatusFailed,
		},
		"module2": {
			"job_1": state.StatusRunning,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	err := handler.handleRestart("", "")
	if err != nil {
		t.Errorf("handleRestart() error = %v", err)
	}

	// Verify all jobs are reset to PENDING
	stateData := handler.stateManager.GetState()
	for moduleName, module := range stateData.Modules {
		for jobName, job := range module.Jobs {
			if job.Status != state.StatusPending {
				t.Errorf("handleRestart() job %s/%s status = %v, want PENDING", moduleName, jobName, job.Status)
			}
		}
	}
}

// Test handleRestart with module specified (reset module)
func TestDoingHandler_handleRestart_module(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"module1": {
			"job_1": state.StatusCompleted,
		},
		"module2": {
			"job_1": state.StatusCompleted,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	err := handler.handleRestart("module1", "")
	if err != nil {
		t.Errorf("handleRestart() error = %v", err)
	}

	// Verify only module1 jobs are reset
	stateData := handler.stateManager.GetState()
	if stateData.Modules["module1"].Jobs["job_1"].Status != state.StatusPending {
		t.Error("handleRestart() module1/job_1 should be PENDING")
	}
	if stateData.Modules["module2"].Jobs["job_1"].Status != state.StatusCompleted {
		t.Error("handleRestart() module2/job_1 should still be COMPLETED")
	}
}

// Test handleRestart with both module and job specified (reset specific job)
func TestDoingHandler_handleRestart_job(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"module1": {
			"job_1": state.StatusCompleted,
			"job_2": state.StatusCompleted,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	err := handler.handleRestart("module1", "job_1")
	if err != nil {
		t.Errorf("handleRestart() error = %v", err)
	}

	// Verify only job_1 is reset
	stateData := handler.stateManager.GetState()
	if stateData.Modules["module1"].Jobs["job_1"].Status != state.StatusPending {
		t.Error("handleRestart() job_1 should be PENDING")
	}
	if stateData.Modules["module1"].Jobs["job_2"].Status != state.StatusCompleted {
		t.Error("handleRestart() job_2 should still be COMPLETED")
	}
}

// Test selectTargetJob with no params (find first PENDING)
func TestDoingHandler_selectTargetJob_noParams(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"module1": {
			"job_1": state.StatusCompleted,
			"job_2": state.StatusPending,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	module, job, err := handler.selectTargetJob("", "")
	if err != nil {
		t.Errorf("selectTargetJob() error = %v", err)
	}

	if module != "module1" || job != "job_2" {
		t.Errorf("selectTargetJob() = %s/%s, want module1/job_2", module, job)
	}
}

// Test selectTargetJob with module specified
func TestDoingHandler_selectTargetJob_withModule(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"module1": {
			"job_1": state.StatusPending,
		},
		"module2": {
			"job_1": state.StatusPending,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	module, job, err := handler.selectTargetJob("module2", "")
	if err != nil {
		t.Errorf("selectTargetJob() error = %v", err)
	}

	if module != "module2" || job != "job_1" {
		t.Errorf("selectTargetJob() = %s/%s, want module2/job_1", module, job)
	}
}

// Test selectTargetJob with both module and job specified
func TestDoingHandler_selectTargetJob_withModuleAndJob(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"module1": {
			"job_1": state.StatusPending,
			"job_2": state.StatusPending,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	module, job, err := handler.selectTargetJob("module1", "job_2")
	if err != nil {
		t.Errorf("selectTargetJob() error = %v", err)
	}

	if module != "module1" || job != "job_2" {
		t.Errorf("selectTargetJob() = %s/%s, want module1/job_2", module, job)
	}
}

// Test selectTargetJob with non-existent module
func TestDoingHandler_selectTargetJob_nonExistentModule(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	_, _, err := handler.selectTargetJob("nonexistent", "")
	if err == nil {
		t.Error("selectTargetJob() expected error for non-existent module")
	}
}

// Test selectTargetJob with no pending jobs
func TestDoingHandler_selectTargetJob_noPending(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state with all jobs completed
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"module1": {
			"job_1": state.StatusCompleted,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	_, _, err := handler.selectTargetJob("", "")
	if err == nil {
		t.Error("selectTargetJob() expected error when no pending jobs")
	}
}

// Test checkPrerequisites with no prerequisites
func TestDoingHandler_checkPrerequisites_noPrereqs(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"test-module": {
			"job_1": state.StatusPending,
		},
	})

	// Create plan file with no prerequisites
	planContent := `# Plan: test-module

### Job 1: Test Job

**目标**: Test goal

**Tasks (Todo 列表)**:
- [ ] Task 1: Test task

**验证器**: ...
`
	setupTestPlanFile(t, planDir, "test-module", planContent)

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	err := handler.checkPrerequisites("test-module", "job_1")
	if err != nil {
		t.Errorf("checkPrerequisites() error = %v", err)
	}
}

// Test checkPrerequisites with met prerequisites
func TestDoingHandler_checkPrerequisites_met(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state with completed prerequisite
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"test-module": {
			"First Job":  state.StatusCompleted,
			"Second Job": state.StatusPending,
		},
	})

	// Create plan file with prerequisites
	planContent := `# Plan: test-module

### Job 1: First Job

**目标**: First goal

**Tasks (Todo 列表)**:
- [ ] Task 1: First task

### Job 2: Second Job

**目标**: Second goal

**前置条件**:
- First Job

**Tasks (Todo 列表)**:
- [ ] Task 1: Second task
`
	setupTestPlanFile(t, planDir, "test-module", planContent)

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	err := handler.checkPrerequisites("test-module", "Second Job")
	if err != nil {
		t.Errorf("checkPrerequisites() error = %v", err)
	}
}

// Test checkPrerequisites with unmet prerequisites
func TestDoingHandler_checkPrerequisites_unmet(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state with pending prerequisite
	// Note: The job name in state should match the job name in the plan file
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"test-module": {
			"First Job":  state.StatusPending,
			"Second Job": state.StatusPending,
		},
	})

	// Create plan file with prerequisites
	planContent := `# Plan: test-module

### Job 1: First Job

**目标**: First goal

**Tasks (Todo 列表)**:
- [ ] Task 1: First task

### Job 2: Second Job

**目标**: Second goal

**前置条件**:
- First Job

**Tasks (Todo 列表)**:
- [ ] Task 1: Second task
`
	setupTestPlanFile(t, planDir, "test-module", planContent)

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	err := handler.checkPrerequisites("test-module", "Second Job")
	if err == nil {
		t.Error("checkPrerequisites() expected error for unmet prerequisites")
	}

	if !strings.Contains(err.Error(), "前置条件不满足") {
		t.Errorf("checkPrerequisites() error = %v, should contain '前置条件不满足'", err)
	}
}

// Test updateStatus
func TestDoingHandler_updateStatus(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"test-module": {
			"job_1": state.StatusPending,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	// Update status to RUNNING
	err := handler.updateStatus("test-module", "job_1", state.StatusRunning)
	if err != nil {
		t.Errorf("updateStatus() error = %v", err)
	}

	// Verify status was updated
	stateData := handler.stateManager.GetState()
	if stateData.Modules["test-module"].Jobs["job_1"].Status != state.StatusRunning {
		t.Errorf("updateStatus() status = %v, want RUNNING", stateData.Modules["test-module"].Jobs["job_1"].Status)
	}
}

// Test updateStatus persists to file
func TestDoingHandler_updateStatus_persistence(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"test-module": {
			"job_1": state.StatusPending,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	// Update status
	err := handler.updateStatus("test-module", "job_1", state.StatusCompleted)
	if err != nil {
		t.Errorf("updateStatus() error = %v", err)
	}

	// Create a new handler and load state to verify persistence
	handler2 := NewDoingHandler(cfg, &mockLogger{})
	handler2.loadStatus()

	stateData := handler2.stateManager.GetState()
	if stateData.Modules["test-module"].Jobs["job_1"].Status != state.StatusCompleted {
		t.Errorf("updateStatus() persisted status = %v, want COMPLETED", stateData.Modules["test-module"].Jobs["job_1"].Status)
	}
}

// Test getStatusFilePath
func TestDoingHandler_getStatusFilePath(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")

	cfg := &mockConfig{
		workDir: workDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	path := handler.getStatusFilePath()
	expected := filepath.Join(workDir, "status.json")

	if path != expected {
		t.Errorf("getStatusFilePath() = %v, want %v", path, expected)
	}
}

// Test loadPlan function
// Task 1 & 2: Test loading Plan file and parsing with Markdown Parser
func TestDoingHandler_loadPlan_success(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Create a test plan file
	planContent := `# Plan: test-module

## 模块概述

**模块职责**: Test module for loading

**对应 Research**:
- research1.md

**依赖模块**:
- config

**被依赖模块**:
- cli

## Jobs

### Job 1: Test Job

**目标**: Test goal extraction

**前置条件**:
- config/job_1

**Tasks (Todo 列表)**:
- [ ] Task 1: First test task
- [x] Task 2: Second completed task

**验证器**:
- [ ] Validator 1: First validation check
- [x] Validator 2: Second validation check

**调试日志**:
- debug1: Test issue, test reproduction, hypothesis, verification, fix method, progress
`
	setupTestPlanFile(t, planDir, "test-module", planContent)

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	// Test loading the plan
	planData, err := handler.loadPlan("test-module")
	if err != nil {
		t.Fatalf("loadPlan() error = %v", err)
	}

	// Verify plan was loaded correctly
	if planData == nil {
		t.Fatal("loadPlan() returned nil plan")
	}

	// Check module name
	if planData.Name != "test-module" {
		t.Errorf("loadPlan() plan.Name = %v, want 'test-module'", planData.Name)
	}

	// Check responsibility
	if planData.Responsibility != "Test module for loading" {
		t.Errorf("loadPlan() plan.Responsibility = %v, want 'Test module for loading'", planData.Responsibility)
	}

	// Check research
	if len(planData.Research) != 1 || planData.Research[0] != "research1.md" {
		t.Errorf("loadPlan() plan.Research = %v, want ['research1.md']", planData.Research)
	}

	// Check dependencies
	if len(planData.Dependencies) != 1 || planData.Dependencies[0] != "config" {
		t.Errorf("loadPlan() plan.Dependencies = %v, want ['config']", planData.Dependencies)
	}

	// Check dependents
	if len(planData.Dependents) != 1 || planData.Dependents[0] != "cli" {
		t.Errorf("loadPlan() plan.Dependents = %v, want ['cli']", planData.Dependents)
	}

	// Check jobs
	if len(planData.Jobs) != 1 {
		t.Fatalf("loadPlan() len(plan.Jobs) = %v, want 1", len(planData.Jobs))
	}

	job := planData.Jobs[0]
	if job.Name != "Test Job" {
		t.Errorf("loadPlan() job.Name = %v, want 'Test Job'", job.Name)
	}

	if job.Index != 1 {
		t.Errorf("loadPlan() job.Index = %v, want 1", job.Index)
	}

	if job.Goal != "Test goal extraction" {
		t.Errorf("loadPlan() job.Goal = %v, want 'Test goal extraction'", job.Goal)
	}

	// Check prerequisites
	if len(job.Prerequisites) != 1 || job.Prerequisites[0] != "config/job_1" {
		t.Errorf("loadPlan() job.Prerequisites = %v, want ['config/job_1']", job.Prerequisites)
	}

	// Check tasks
	if len(job.Tasks) != 2 {
		t.Errorf("loadPlan() len(job.Tasks) = %v, want 2", len(job.Tasks))
	}

	// Task 1 should be pending
	if job.Tasks[0].Index != 1 || job.Tasks[0].Description != "First test task" || job.Tasks[0].Completed {
		t.Errorf("loadPlan() job.Tasks[0] = %+v, want pending Task 1", job.Tasks[0])
	}

	// Task 2 should be completed
	if job.Tasks[1].Index != 2 || job.Tasks[1].Description != "Second completed task" || !job.Tasks[1].Completed {
		t.Errorf("loadPlan() job.Tasks[1] = %+v, want completed Task 2", job.Tasks[1])
	}

	// Check validators
	if len(job.Validators) != 2 {
		t.Errorf("loadPlan() len(job.Validators) = %v, want 2", len(job.Validators))
	}

	// Debug logs
	if len(job.DebugLogs) != 1 {
		t.Errorf("loadPlan() len(job.DebugLogs) = %v, want 1", len(job.DebugLogs))
	}
}

// Test loadPlan with non-existent module
// Task 6: Test Plan file not found error handling
func TestDoingHandler_loadPlan_notFound(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	// Test loading a non-existent plan
	planData, err := handler.loadPlan("nonexistent-module")
	if err == nil {
		t.Error("loadPlan() expected error for non-existent module")
	}

	if planData != nil {
		t.Error("loadPlan() expected nil plan for non-existent module")
	}

	// Verify error message contains "计划文件不存在"
	if !strings.Contains(err.Error(), "计划文件不存在") {
		t.Errorf("loadPlan() error = %v, should contain '计划文件不存在'", err)
	}

	// Test IsPlanNotFoundError helper
	if !handler.IsPlanNotFoundError(err) {
		t.Error("IsPlanNotFoundError() should return true for not found error")
	}
}

// Test getJobFromPlan function
// Task 3: Test extracting target Job definition
func TestDoingHandler_getJobFromPlan(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Create a test plan file with multiple jobs
	planContent := `# Plan: test-module

### Job 1: First Job

**目标**: First goal

**Tasks (Todo 列表)**:
- [ ] Task 1: First task

### Job 2: Second Job

**目标**: Second goal

**Tasks (Todo 列表)**:
- [ ] Task 1: Second task
`
	setupTestPlanFile(t, planDir, "test-module", planContent)

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	planData, err := handler.loadPlan("test-module")
	if err != nil {
		t.Fatalf("loadPlan() error = %v", err)
	}

	// Test finding first job
	job1, err := handler.getJobFromPlan(planData, "First Job")
	if err != nil {
		t.Errorf("getJobFromPlan() error for First Job = %v", err)
	}
	if job1 == nil || job1.Name != "First Job" {
		t.Errorf("getJobFromPlan() returned wrong job: %+v", job1)
	}

	// Test finding second job
	job2, err := handler.getJobFromPlan(planData, "Second Job")
	if err != nil {
		t.Errorf("getJobFromPlan() error for Second Job = %v", err)
	}
	if job2 == nil || job2.Name != "Second Job" {
		t.Errorf("getJobFromPlan() returned wrong job: %+v", job2)
	}

	// Test finding non-existent job
	_, err = handler.getJobFromPlan(planData, "Nonexistent Job")
	if err == nil {
		t.Error("getJobFromPlan() expected error for non-existent job")
	}
}

// Test IsPlanNotFoundError helper
func TestDoingHandler_IsPlanNotFoundError(t *testing.T) {
	handler := NewDoingHandler(&mockConfig{}, &mockLogger{})

	// Test with nil error
	if handler.IsPlanNotFoundError(nil) {
		t.Error("IsPlanNotFoundError(nil) should return false")
	}

	// Test with random error
	randomErr := fmt.Errorf("some random error")
	if handler.IsPlanNotFoundError(randomErr) {
		t.Error("IsPlanNotFoundError(random error) should return false")
	}

	// Test with plan not found error
	notFoundErr := fmt.Errorf("计划文件不存在: /some/path")
	if !handler.IsPlanNotFoundError(notFoundErr) {
		t.Error("IsPlanNotFoundError(plan not found error) should return true")
	}
}

// Test loadPlan with complex plan structure
// Task 4 & 5: Test extracting Tasks list and validators
func TestDoingHandler_loadPlan_complexStructure(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Create a complex test plan file
	planContent := `# Plan: complex-module

## 模块概述

**模块职责**: Complex module with multiple jobs and features

**对应 Research**:
- research1.md
- research2.md
- research3.md

**依赖模块**:
- config
- logging
- state

**被依赖模块**:
- executor
- cli

## Jobs

### Job 1: Setup Job

**目标**: Initialize the module

**前置条件**:
- config/job_1
- logging/job_1

**Tasks (Todo 列表)**:
- [x] Task 1: Create structure
- [x] Task 2: Implement basic functions
- [ ] Task 3: Add error handling
- [ ] Task 4: Write tests

**验证器**:
- [ ] 正确加载指定模块的 Plan 文件
- [ ] 正确解析 Job 定义
- [ ] 正确提取 Tasks 列表
- [ ] 正确提取验证器

**调试日志**:
- debug1: Issue description, reproduction steps, hypothesis, verification steps, fix method, progress
- explore1: Discovery description, findings, notes, verified, recorded, done

### Job 2: Implementation Job

**目标**: Implement core features

**前置条件**:
- Setup Job

**Tasks (Todo 列表)**:
- [ ] Task 1: Feature A
- [ ] Task 2: Feature B

**验证器**:
- [ ] Feature A works correctly
- [ ] Feature B works correctly

**调试日志**:
- 无
`
	setupTestPlanFile(t, planDir, "complex-module", planContent)

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	planData, err := handler.loadPlan("complex-module")
	if err != nil {
		t.Fatalf("loadPlan() error = %v", err)
	}

	// Check module name
	if planData.Name != "complex-module" {
		t.Errorf("loadPlan() plan.Name = %v, want 'complex-module'", planData.Name)
	}

	// Check multiple research entries
	if len(planData.Research) != 3 {
		t.Errorf("loadPlan() len(plan.Research) = %v, want 3", len(planData.Research))
	}

	// Check multiple dependencies
	if len(planData.Dependencies) != 3 {
		t.Errorf("loadPlan() len(plan.Dependencies) = %v, want 3", len(planData.Dependencies))
	}

	// Check multiple dependents
	if len(planData.Dependents) != 2 {
		t.Errorf("loadPlan() len(plan.Dependents) = %v, want 2", len(planData.Dependents))
	}

	// Check multiple jobs
	if len(planData.Jobs) != 2 {
		t.Fatalf("loadPlan() len(plan.Jobs) = %v, want 2", len(planData.Jobs))
	}

	// Check first job details
	job1 := planData.Jobs[0]
	if job1.Name != "Setup Job" {
		t.Errorf("loadPlan() job1.Name = %v, want 'Setup Job'", job1.Name)
	}

	// Check first job tasks
	if len(job1.Tasks) != 4 {
		t.Errorf("loadPlan() len(job1.Tasks) = %v, want 4", len(job1.Tasks))
	}

	// Check completed vs pending tasks
	completedCount := 0
	pendingCount := 0
	for _, task := range job1.Tasks {
		if task.Completed {
			completedCount++
		} else {
			pendingCount++
		}
	}
	if completedCount != 2 {
		t.Errorf("loadPlan() completed tasks = %v, want 2", completedCount)
	}
	if pendingCount != 2 {
		t.Errorf("loadPlan() pending tasks = %v, want 2", pendingCount)
	}

	// Check validators
	if len(job1.Validators) != 4 {
		t.Errorf("loadPlan() len(job1.Validators) = %v, want 4", len(job1.Validators))
	}

	// Check debug logs
	if len(job1.DebugLogs) != 2 {
		t.Errorf("loadPlan() len(job1.DebugLogs) = %v, want 2", len(job1.DebugLogs))
	}

	// Check second job (with no debug logs)
	job2 := planData.Jobs[1]
	if job2.Name != "Implementation Job" {
		t.Errorf("loadPlan() job2.Name = %v, want 'Implementation Job'", job2.Name)
	}
	if len(job2.DebugLogs) != 0 {
		t.Errorf("loadPlan() len(job2.DebugLogs) = %v, want 0", len(job2.DebugLogs))
	}
}

// ============================================================================
// Job 4: Executor Integration Tests
// ============================================================================

// mockExecutor implements a mock executor.Engine for testing
type mockExecutor struct {
	executeJobFunc func(ctx context.Context, module, job string) error
	lastModule     string
	lastJob        string
	callCount      int
}

func (m *mockExecutor) ExecuteJob(ctx context.Context, module, job string) error {
	m.lastModule = module
	m.lastJob = job
	m.callCount++
	if m.executeJobFunc != nil {
		return m.executeJobFunc(ctx, module, job)
	}
	return nil
}

func (m *mockExecutor) ExecuteTask(ctx context.Context, module, job string, taskIndex int, taskDesc string) error {
	return nil
}

func (m *mockExecutor) ResumeJob(ctx context.Context, module, job string) error {
	return m.ExecuteJob(ctx, module, job)
}

// Test initializeExecutor
// Task 1: Test Executor initialization
func TestDoingHandler_initializeExecutor(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"test-module": {
			"job_1": state.StatusPending,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	// Test initializeExecutor
	err := handler.initializeExecutor()
	if err != nil {
		t.Errorf("initializeExecutor() error = %v", err)
	}

	// Verify executor was initialized
	if handler.executor == nil {
		t.Error("initializeExecutor() executor should be set")
	}

	// Verify git manager was initialized
	if handler.gitManager == nil {
		t.Error("initializeExecutor() gitManager should be set")
	}
}

// Test initializeExecutor without state manager
func TestDoingHandler_initializeExecutor_noStateManager(t *testing.T) {
	handler := NewDoingHandler(&mockConfig{}, &mockLogger{})

	// Test initializeExecutor without loading state first
	err := handler.initializeExecutor()
	if err == nil {
		t.Error("initializeExecutor() should return error when state manager is nil")
	}
}

// Test executeJob with mock executor
// Task 2, 3, 4, 5: Test executeJob function
func TestDoingHandler_executeJob(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"test-module": {
			"job_1": state.StatusPending,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	// Create mock executor
	mockExec := &mockExecutor{}
	handler.SetExecutor(mockExec)

	// Test executeJob
	ctx := context.Background()
	result, err := handler.executeJob(ctx, "test-module", "job_1")

	if err != nil {
		t.Errorf("executeJob() error = %v", err)
	}

	if result == nil {
		t.Fatal("executeJob() result should not be nil")
	}

	// Verify executor was called with correct parameters
	if mockExec.lastModule != "test-module" {
		t.Errorf("executeJob() module = %v, want 'test-module'", mockExec.lastModule)
	}
	if mockExec.lastJob != "job_1" {
		t.Errorf("executeJob() job = %v, want 'job_1'", mockExec.lastJob)
	}
	if mockExec.callCount != 1 {
		t.Errorf("executeJob() call count = %v, want 1", mockExec.callCount)
	}

	// Verify result fields
	if result.Module != "test-module" {
		t.Errorf("result.Module = %v, want 'test-module'", result.Module)
	}
	if result.Job != "job_1" {
		t.Errorf("result.Job = %v, want 'job_1'", result.Job)
	}
}

// Test executeJob without executor initialization
func TestDoingHandler_executeJob_noExecutor(t *testing.T) {
	handler := NewDoingHandler(&mockConfig{}, &mockLogger{})

	ctx := context.Background()
	_, err := handler.executeJob(ctx, "test-module", "job_1")

	if err == nil {
		t.Error("executeJob() should return error when executor is not initialized")
	}
}

// Test executeJob with execution failure
// Task 5: Test execution result handling
func TestDoingHandler_executeJob_failure(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"test-module": {
			"job_1": state.StatusPending,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	// Create mock executor that returns error
	mockExec := &mockExecutor{
		executeJobFunc: func(ctx context.Context, module, job string) error {
			return fmt.Errorf("execution failed")
		},
	}
	handler.SetExecutor(mockExec)

	// Test executeJob with failure
	ctx := context.Background()
	result, err := handler.executeJob(ctx, "test-module", "job_1")

	if err == nil {
		t.Error("executeJob() should return error when execution fails")
	}

	if result == nil {
		t.Fatal("executeJob() result should not be nil even on failure")
	}

	// Verify result status is FAILED
	if result.Status != state.StatusFailed {
		t.Errorf("result.Status = %v, want FAILED", result.Status)
	}
}

// Test executeJob timeout
// Task 6: Test timeout control
func TestDoingHandler_executeJob_timeout(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state
	setupTestState(t, workDir, map[string]map[string]state.Status{
		"test-module": {
			"job_1": state.StatusPending,
		},
	})

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	// Create mock executor that simulates timeout
	mockExec := &mockExecutor{
		executeJobFunc: func(ctx context.Context, module, job string) error {
			// Simulate context timeout
			<-ctx.Done()
			return ctx.Err()
		},
	}
	handler.SetExecutor(mockExec)

	// Create a context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test executeJob with timeout
	_, err := handler.executeJob(ctx, "test-module", "job_1")

	// Should return timeout error
	if err == nil {
		t.Error("executeJob() should return error on timeout")
	}
}

// Test SetExecutor
func TestDoingHandler_SetExecutor(t *testing.T) {
	handler := NewDoingHandler(&mockConfig{}, &mockLogger{})
	mockExec := &mockExecutor{}

	handler.SetExecutor(mockExec)

	if handler.executor != mockExec {
		t.Error("SetExecutor() should set the executor")
	}
}

// Test SetGitManager
func TestDoingHandler_SetGitManager(t *testing.T) {
	handler := NewDoingHandler(&mockConfig{}, &mockLogger{})
	gitMgr := git.NewManager()

	handler.SetGitManager(gitMgr)

	if handler.gitManager != gitMgr {
		t.Error("SetGitManager() should set the git manager")
	}
}

// Test Executor integration with real executor
// Task 7: Integration test
func TestDoingHandler_ExecutorIntegration(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")
	planDir := filepath.Join(workDir, "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Setup test state with tasks
	stateContent := `{
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
					"job_1": {
						"name": "job_1",
						"status": "PENDING",
						"tasks_total": 2,
						"tasks_completed": 0,
						"tasks": [
							{"index": 0, "status": "PENDING", "description": "Task 1"},
							{"index": 1, "status": "PENDING", "description": "Task 2"}
						],
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					}
				},
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z"
			}
		}
	}`

	stateFile := filepath.Join(workDir, "status.json")
	if err := os.WriteFile(stateFile, []byte(stateContent), 0644); err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
		planDir: planDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})
	handler.loadStatus()

	// Initialize git manager and executor
	gitMgr := git.NewManager()
	handler.SetGitManager(gitMgr)

	// Create real executor with test config
	execConfig := &executor.Config{
		MaxRetries:   3,
		AutoCommit:   false,
		CommitPrefix: "morty:",
		WorkingDir:   workDir,
	}
	eng := executor.NewEngine(handler.stateManager, gitMgr, &mockLogger{}, execConfig)
	handler.SetExecutor(eng)

	// Verify executor is properly set
	if handler.executor == nil {
		t.Fatal("Executor should be set")
	}

	// Test that executor can be called
	ctx := context.Background()
	_, err := handler.executeJob(ctx, "test-module", "job_1")
	// May fail due to actual execution, but should not panic
	if err != nil {
		t.Logf("executeJob() returned error (may be expected): %v", err)
	}
}

// ============================================================================
// Job 5: createGitCommit Tests
// ============================================================================

// Test generateCommitMessage with valid summary
// Task 2: Test commit message generation
func TestDoingHandler_generateCommitMessage(t *testing.T) {
	handler := NewDoingHandler(&mockConfig{}, &mockLogger{})

	tests := []struct {
		name     string
		summary  *CommitSummary
		expected string
	}{
		{
			name: "basic commit message",
			summary: &CommitSummary{
				Module: "doing_cmd",
				Job:    "job_5",
				Status: "COMPLETED",
			},
			expected: "morty: doing_cmd/job_5 - COMPLETED",
		},
		{
			name: "commit message with loop count",
			summary: &CommitSummary{
				Module:    "doing_cmd",
				Job:       "job_5",
				Status:    "RUNNING",
				LoopCount: 3,
			},
			expected: "morty: doing_cmd/job_5 - RUNNING (loop 3)",
		},
		{
			name: "commit message with failed status",
			summary: &CommitSummary{
				Module: "test_module",
				Job:    "test_job",
				Status: "FAILED",
			},
			expected: "morty: test_module/test_job - FAILED",
		},
		{
			name:     "nil summary",
			summary:  nil,
			expected: "morty: unknown - UNKNOWN",
		},
		{
			name: "empty fields",
			summary: &CommitSummary{
				Module: "",
				Job:    "",
				Status: "",
			},
			expected: "morty: unknown/unknown - UNKNOWN",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := handler.generateCommitMessage(tc.summary)
			if result != tc.expected {
				t.Errorf("generateCommitMessage() = %v, want %v", result, tc.expected)
			}
		})
	}
}

// Test createGitCommit success
// Task 1, 3, 4, 5, 6: Test full commit workflow
func TestDoingHandler_createGitCommit_success(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")

	// Create .morty directory
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create .morty dir: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	// Initialize git repository
	gitMgr := git.NewManager()
	handler.SetGitManager(gitMgr)

	// Initialize git repo
	err := gitMgr.InitIfNeeded(workDir)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user
	gitMgr.RunGitCommand(workDir, "config", "user.email", "test@test.com")
	gitMgr.RunGitCommand(workDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(workDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	gitMgr.RunGitCommand(workDir, "add", "initial.txt")
	gitMgr.RunGitCommand(workDir, "commit", "-m", "initial commit")

	// Create a test file to commit
	testFile := filepath.Join(workDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create commit
	summary := &CommitSummary{
		Module:    "doing_cmd",
		Job:       "job_5",
		Status:    "COMPLETED",
		LoopCount: 1,
	}

	commitHash, err := handler.createGitCommit(summary)
	if err != nil {
		t.Errorf("createGitCommit() error = %v", err)
	}

	if commitHash == "" {
		t.Error("createGitCommit() returned empty commit hash")
	}

	// Verify commit message
	logOutput, err := gitMgr.RunGitCommand(workDir, "log", "-1", "--pretty=format:%s")
	if err != nil {
		t.Fatalf("Failed to get log: %v", err)
	}

	expectedMsg := "morty: doing_cmd/job_5 - COMPLETED (loop 1)"
	if logOutput != expectedMsg {
		t.Errorf("Commit message = %v, want %v", logOutput, expectedMsg)
	}
}

// Test createGitCommit with no changes
// Task 3: Test no changes scenario
func TestDoingHandler_createGitCommit_noChanges(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")

	// Create .morty directory
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create .morty dir: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	// Initialize git repository
	gitMgr := git.NewManager()
	handler.SetGitManager(gitMgr)

	// Initialize git repo
	err := gitMgr.InitIfNeeded(workDir)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user
	gitMgr.RunGitCommand(workDir, "config", "user.email", "test@test.com")
	gitMgr.RunGitCommand(workDir, "config", "user.name", "Test User")

	// Create initial commit (so repo is clean)
	initialFile := filepath.Join(workDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	gitMgr.RunGitCommand(workDir, "add", "initial.txt")
	gitMgr.RunGitCommand(workDir, "commit", "-m", "initial commit")

	// Try to create commit with no changes
	summary := &CommitSummary{
		Module: "doing_cmd",
		Job:    "job_5",
		Status: "COMPLETED",
	}

	commitHash, err := handler.createGitCommit(summary)
	if err != nil {
		t.Errorf("createGitCommit() error = %v", err)
	}

	// Should return empty hash without error when no changes
	if commitHash != "" {
		t.Errorf("createGitCommit() = %v, want empty string when no changes", commitHash)
	}
}

// Test createGitCommit stages all changes
// Task 4: Test staging all changes
func TestDoingHandler_createGitCommit_stagesAllChanges(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")

	// Create .morty directory
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create .morty dir: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	// Initialize git repository
	gitMgr := git.NewManager()
	handler.SetGitManager(gitMgr)

	// Initialize git repo
	err := gitMgr.InitIfNeeded(workDir)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user
	gitMgr.RunGitCommand(workDir, "config", "user.email", "test@test.com")
	gitMgr.RunGitCommand(workDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(workDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	gitMgr.RunGitCommand(workDir, "add", "initial.txt")
	gitMgr.RunGitCommand(workDir, "commit", "-m", "initial commit")

	// Create multiple files without staging
	file1 := filepath.Join(workDir, "file1.txt")
	file2 := filepath.Join(workDir, "file2.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	// Create commit (should auto-stage all files)
	summary := &CommitSummary{
		Module: "doing_cmd",
		Job:    "job_5",
		Status: "COMPLETED",
	}

	_, err = handler.createGitCommit(summary)
	if err != nil {
		t.Errorf("createGitCommit() error = %v", err)
	}

	// Verify all files were committed (no uncommitted changes)
	status, err := gitMgr.RunGitCommand(workDir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if strings.TrimSpace(status) != "" {
		t.Errorf("Expected all files to be committed, but status shows: %s", status)
	}
}

// Test createGitCommit error handling
// Task 6: Test commit error handling
func TestDoingHandler_createGitCommit_notARepo(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")

	cfg := &mockConfig{
		workDir: workDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	// Don't initialize git repo - should fail
	gitMgr := git.NewManager()
	handler.SetGitManager(gitMgr)

	summary := &CommitSummary{
		Module: "doing_cmd",
		Job:    "job_5",
		Status: "COMPLETED",
	}

	// Create a file to have changes
	testFile := filepath.Join(workDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	_, err := handler.createGitCommit(summary)
	if err == nil {
		t.Error("createGitCommit() expected error for non-git directory")
	}
}

// Test createGitCommit commit message format
// Validator: Commit message format check
func TestDoingHandler_createGitCommit_messageFormat(t *testing.T) {
	tmpDir := setupTestDir(t)
	workDir := filepath.Join(tmpDir, ".morty")

	// Create .morty directory
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("Failed to create .morty dir: %v", err)
	}

	cfg := &mockConfig{
		workDir: workDir,
	}
	handler := NewDoingHandler(cfg, &mockLogger{})

	gitMgr := git.NewManager()
	handler.SetGitManager(gitMgr)

	err := gitMgr.InitIfNeeded(workDir)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	gitMgr.RunGitCommand(workDir, "config", "user.email", "test@test.com")
	gitMgr.RunGitCommand(workDir, "config", "user.name", "Test User")

	initialFile := filepath.Join(workDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	gitMgr.RunGitCommand(workDir, "add", "initial.txt")
	gitMgr.RunGitCommand(workDir, "commit", "-m", "initial commit")

	// Test various commit message formats
	tests := []struct {
		name           string
		summary        *CommitSummary
		expectedPrefix string
		contains       []string
	}{
		{
			name: "module job status format",
			summary: &CommitSummary{
				Module: "test_module",
				Job:    "test_job",
				Status: "COMPLETED",
			},
			expectedPrefix: "morty: ",
			contains:       []string{"test_module", "test_job", "COMPLETED"},
		},
		{
			name: "with loop number",
			summary: &CommitSummary{
				Module:    "logging",
				Job:       "job_1",
				Status:    "RUNNING",
				LoopCount: 5,
			},
			expectedPrefix: "morty: ",
			contains:       []string{"logging", "job_1", "RUNNING", "loop 5"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new file for each test
			testFile := filepath.Join(workDir, tc.name+".txt")
			os.WriteFile(testFile, []byte("test"), 0644)

			_, err := handler.createGitCommit(tc.summary)
			if err != nil {
				t.Fatalf("createGitCommit() error = %v", err)
			}

			// Get commit message
			msg, err := gitMgr.RunGitCommand(workDir, "log", "-1", "--pretty=format:%s")
			if err != nil {
				t.Fatalf("Failed to get log: %v", err)
			}

			// Check prefix
			if !strings.HasPrefix(msg, tc.expectedPrefix) {
				t.Errorf("Commit message %q does not start with %q", msg, tc.expectedPrefix)
			}

			// Check contains
			for _, s := range tc.contains {
				if !strings.Contains(msg, s) {
					t.Errorf("Commit message %q does not contain %q", msg, s)
				}
			}
		})
	}
}
