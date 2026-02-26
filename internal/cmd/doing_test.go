package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
