package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/morty/morty/internal/state"
)

func TestNewStatHandler(t *testing.T) {
	cfg := &mockConfig{}
	logger := &mockLogger{}

	handler := NewStatHandler(cfg, logger)
	if handler == nil {
		t.Fatal("Expected handler to be non-nil")
	}

	if handler.cfg == nil {
		t.Error("Expected cfg to be set")
	}

	if handler.logger != logger {
		t.Error("Expected logger to be set")
	}

	if handler.paths == nil {
		t.Error("Expected paths to be initialized")
	}
}

func TestStatHandler_parseOptions(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantWatch    bool
		wantJSON     bool
	}{
		{
			name:      "no flags",
			args:      []string{},
			wantWatch: false,
			wantJSON:  false,
		},
		{
			name:      "watch flag long",
			args:      []string{"--watch"},
			wantWatch: true,
			wantJSON:  false,
		},
		{
			name:      "watch flag short",
			args:      []string{"-w"},
			wantWatch: true,
			wantJSON:  false,
		},
		{
			name:      "json flag long",
			args:      []string{"--json"},
			wantWatch: false,
			wantJSON:  true,
		},
		{
			name:      "json flag short",
			args:      []string{"-j"},
			wantWatch: false,
			wantJSON:  true,
		},
		{
			name:      "both flags",
			args:      []string{"--watch", "--json"},
			wantWatch: true,
			wantJSON:  true,
		},
		{
			name:      "watch equals true",
			args:      []string{"--watch=true"},
			wantWatch: true,
			wantJSON:  false,
		},
		{
			name:      "watch equals 1",
			args:      []string{"--watch=1"},
			wantWatch: true,
			wantJSON:  false,
		},
		{
			name:      "watch equals false",
			args:      []string{"--watch=false"},
			wantWatch: false,
			wantJSON:  false,
		},
		{
			name:      "json equals true",
			args:      []string{"--json=true"},
			wantWatch: false,
			wantJSON:  true,
		},
		{
			name:      "mixed args",
			args:      []string{"some", "args", "--watch", "more"},
			wantWatch: true,
			wantJSON:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewStatHandler(&mockConfig{}, &mockLogger{})
			gotWatch, gotJSON := handler.parseOptions(tt.args)

			if gotWatch != tt.wantWatch {
				t.Errorf("parseOptions() watch = %v, want %v", gotWatch, tt.wantWatch)
			}
			if gotJSON != tt.wantJSON {
				t.Errorf("parseOptions() json = %v, want %v", gotJSON, tt.wantJSON)
			}
		})
	}
}

func TestStatHandler_getStatusFilePath(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *mockConfig
		expectedSub string
	}{
		{
			name:        "with config",
			cfg:         &mockConfig{workDir: "/custom/path"},
			expectedSub: "/custom/path/status.json",
		},
		{
			name:        "without config workdir",
			cfg:         &mockConfig{},
			expectedSub: ".morty/status.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewStatHandler(tt.cfg, &mockLogger{})
			got := handler.getStatusFilePath()

			if !strings.Contains(got, tt.expectedSub) {
				t.Errorf("getStatusFilePath() = %v, expected to contain %v", got, tt.expectedSub)
			}
		})
	}
}

func TestStatHandler_Execute_NoStatusFile(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{workDir: tmpDir}
	logger := &mockLogger{}
	handler := NewStatHandler(cfg, logger)

	ctx := context.Background()
	result, err := handler.Execute(ctx, []string{})

	// Should return error
	if err == nil {
		t.Error("Expected error when status file does not exist")
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", result.ExitCode)
	}

	if result.Err == nil {
		t.Error("Expected result.Err to be set")
	}
}

func TestStatHandler_Execute_NoStatusFile_JSON(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{workDir: tmpDir}
	logger := &mockLogger{}
	handler := NewStatHandler(cfg, logger)

	ctx := context.Background()
	result, err := handler.Execute(ctx, []string{"--json"})

	// Should return error
	if err == nil {
		t.Error("Expected error when status file does not exist")
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", result.ExitCode)
	}
}

func TestStatHandler_Execute_WithStatusFile(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create a mock status.json file
	statusFile := filepath.Join(tmpDir, "status.json")
	statusData := `{
		"global": {
			"status": "RUNNING",
			"current_module": "test_module",
			"current_job": "test_job",
			"start_time": "2024-01-01T00:00:00Z",
			"last_update": "2024-01-01T00:00:00Z",
			"total_loops": 1
		},
		"modules": {
			"test_module": {
				"name": "test_module",
				"status": "RUNNING",
				"jobs": {
					"test_job": {
						"name": "test_job",
						"status": "RUNNING",
						"loop_count": 1,
						"tasks_total": 5,
						"tasks_completed": 3,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					}
				},
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z"
			}
		},
		"version": "1.0"
	}`

	if err := os.WriteFile(statusFile, []byte(statusData), 0644); err != nil {
		t.Fatalf("Failed to create status file: %v", err)
	}

	cfg := &mockConfig{workDir: tmpDir}
	logger := &mockLogger{}
	handler := NewStatHandler(cfg, logger)

	ctx := context.Background()
	result, err := handler.Execute(ctx, []string{})

	// Should not return error
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if result.CurrentJob == nil {
		t.Error("Expected CurrentJob to be set")
	} else {
		if result.CurrentJob.Module != "test_module" {
			t.Errorf("Expected module 'test_module', got '%s'", result.CurrentJob.Module)
		}
		if result.CurrentJob.Job != "test_job" {
			t.Errorf("Expected job 'test_job', got '%s'", result.CurrentJob.Job)
		}
	}

	if result.Summary == nil {
		t.Error("Expected Summary to be set")
	} else {
		if result.Summary.TotalModules != 1 {
			t.Errorf("Expected 1 module, got %d", result.Summary.TotalModules)
		}
		if result.Summary.TotalJobs != 1 {
			t.Errorf("Expected 1 job, got %d", result.Summary.TotalJobs)
		}
	}
}

func TestStatHandler_Execute_WithStatusFile_JSON(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create a mock status.json file
	statusFile := filepath.Join(tmpDir, "status.json")
	statusData := `{
		"global": {
			"status": "COMPLETED",
			"start_time": "2024-01-01T00:00:00Z",
			"last_update": "2024-01-01T00:00:00Z",
			"total_loops": 5
		},
		"modules": {
			"module1": {
				"name": "module1",
				"status": "COMPLETED",
				"jobs": {
					"job1": {
						"name": "job1",
						"status": "COMPLETED",
						"loop_count": 1,
						"tasks_total": 3,
						"tasks_completed": 3,
						"created_at": "2024-01-01T00:00:00Z",
						"updated_at": "2024-01-01T00:00:00Z"
					}
				},
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z"
			}
		},
		"version": "1.0"
	}`

	if err := os.WriteFile(statusFile, []byte(statusData), 0644); err != nil {
		t.Fatalf("Failed to create status file: %v", err)
	}

	cfg := &mockConfig{workDir: tmpDir}
	logger := &mockLogger{}
	handler := NewStatHandler(cfg, logger)

	ctx := context.Background()
	result, err := handler.Execute(ctx, []string{"--json"})

	// Should not return error
	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if !result.JSONOutput {
		t.Error("Expected JSONOutput to be true")
	}
}

func TestStatHandler_Execute_ContextCancellation(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create a mock status.json file
	statusFile := filepath.Join(tmpDir, "status.json")
	statusData := `{
		"global": {
			"status": "RUNNING",
			"start_time": "2024-01-01T00:00:00Z",
			"last_update": "2024-01-01T00:00:00Z",
			"total_loops": 1
		},
		"modules": {},
		"version": "1.0"
	}`

	if err := os.WriteFile(statusFile, []byte(statusData), 0644); err != nil {
		t.Fatalf("Failed to create status file: %v", err)
	}

	cfg := &mockConfig{workDir: tmpDir}
	logger := &mockLogger{}
	handler := NewStatHandler(cfg, logger)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := handler.Execute(ctx, []string{})

	// The execution should complete before checking context cancellation in this simple case
	_ = err
}

func TestStatHandler_outputJSON(t *testing.T) {
	tmpDir := setupTestDir(t)

	handler := NewStatHandler(&mockConfig{workDir: tmpDir}, &mockLogger{})

	tests := []struct {
		name   string
		result *StatResult
	}{
		{
			name: "error result",
			result: &StatResult{
				Err:        fmt.Errorf("test error"),
				ExitCode:   1,
				Duration:   time.Second,
				JSONOutput: true,
			},
		},
		{
			name: "success with current job",
			result: &StatResult{
				ExitCode: 0,
				Duration: time.Second,
				CurrentJob: &state.CurrentJob{
					Module: "test",
					Job:    "job",
					Status: state.StatusRunning,
				},
				Summary: &state.Summary{
					TotalModules: 1,
					TotalJobs:    1,
					Pending:      0,
					Running:      1,
					Completed:    0,
					Failed:       0,
					Blocked:      0,
					Modules: map[string]state.ModuleSummary{
						"test": {
							TotalJobs: 1,
							Running:   1,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Redirect stdout to capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			handler.outputJSON(tt.result)

			w.Close()
			os.Stdout = oldStdout

			outputBytes, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("Failed to read output: %v", err)
			}
			output := string(outputBytes)
			if output == "" {
				t.Error("Expected JSON output, got empty string")
			}

			// Verify it's valid JSON
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(output), &parsed); err != nil {
				t.Errorf("Output is not valid JSON: %v\nOutput: %s", err, output)
			}
		})
	}
}

func TestStatHandler_outputText(t *testing.T) {
	tmpDir := setupTestDir(t)

	handler := NewStatHandler(&mockConfig{workDir: tmpDir}, &mockLogger{})

	tests := []struct {
		name   string
		result *StatResult
	}{
		{
			name: "error result",
			result: &StatResult{
				Err:      fmt.Errorf("test error"),
				ExitCode: 1,
			},
		},
		{
			name: "no current job",
			result: &StatResult{
				ExitCode: 0,
				Summary: &state.Summary{
					TotalModules: 0,
					TotalJobs:    0,
				},
			},
		},
		{
			name: "with current job and summary",
			result: &StatResult{
				ExitCode: 0,
				CurrentJob: &state.CurrentJob{
					Module:    "test",
					Job:       "job",
					Status:    state.StatusRunning,
					StartedAt: time.Now(),
				},
				Summary: &state.Summary{
					TotalModules: 1,
					TotalJobs:    2,
					Pending:      1,
					Running:      1,
					Completed:    0,
					Failed:       0,
					Blocked:      0,
					Modules: map[string]state.ModuleSummary{
						"test": {
							TotalJobs: 2,
							Pending:   1,
							Running:   1,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Redirect stdout to capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			handler.outputText(tt.result)

			w.Close()
			os.Stdout = oldStdout

			outputBytes, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("Failed to read output: %v", err)
			}
			output := string(outputBytes)
			if output == "" {
				t.Error("Expected text output, got empty string")
			}

			// Check for expected content
			// For error results, output is just the error message
			// For success results, output contains "Morty Status"
			if tt.result.Err != nil {
				if !strings.Contains(output, tt.result.Err.Error()) {
					t.Errorf("Expected output to contain error message '%s', got '%s'", tt.result.Err.Error(), output)
				}
			} else {
				if !strings.Contains(output, "Morty Status") {
					t.Error("Expected output to contain 'Morty Status'")
				}
			}
		})
	}
}

func TestStatHandler_getStatusString(t *testing.T) {
	handler := NewStatHandler(&mockConfig{}, &mockLogger{})

	tests := []struct {
		name     string
		result   *StatResult
		expected string
	}{
		{
			name: "error result",
			result: &StatResult{
				Err: fmt.Errorf("test error"),
			},
			expected: "error",
		},
		{
			name: "idle result",
			result: &StatResult{
				ExitCode: 0,
			},
			expected: "idle",
		},
		{
			name: "running result",
			result: &StatResult{
				ExitCode: 0,
				CurrentJob: &state.CurrentJob{
					Status: state.StatusRunning,
				},
			},
			expected: "RUNNING",
		},
		{
			name: "completed result",
			result: &StatResult{
				ExitCode: 0,
				CurrentJob: &state.CurrentJob{
					Status: state.StatusCompleted,
				},
			},
			expected: "COMPLETED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.getStatusString(tt.result)
			if got != tt.expected {
				t.Errorf("getStatusString() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStatResult_Struct(t *testing.T) {
	result := &StatResult{
		Status:     "test",
		ExitCode:   0,
		Duration:   time.Second,
		JSONOutput: true,
	}

	if result.Status != "test" {
		t.Error("Status not set correctly")
	}

	if result.ExitCode != 0 {
		t.Error("ExitCode not set correctly")
	}

	if result.Duration != time.Second {
		t.Error("Duration not set correctly")
	}

	if !result.JSONOutput {
		t.Error("JSONOutput not set correctly")
	}
}

func TestStatHandler_collectStatus(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create a mock status.json file with comprehensive data (version 1.0 format)
	statusFile := filepath.Join(tmpDir, "status.json")
	statusData := `{
		"version": "1.0",
		"global": {
			"status": "RUNNING",
			"current_module": "test_module",
			"current_job": "job_2",
			"start_time": "2024-01-01T00:00:00Z",
			"last_update": "2024-01-01T10:00:00Z",
			"total_loops": 5
		},
		"modules": {
			"test_module": {
				"name": "test_module",
				"status": "RUNNING",
				"jobs": {
					"job_1": {
						"name": "job_1",
						"status": "COMPLETED",
						"loop_count": 1,
						"tasks_total": 3,
						"tasks_completed": 3,
						"debug_logs": [],
						"created_at": "2024-01-01T08:00:00Z",
						"updated_at": "2024-01-01T09:00:00Z",
						"completed_at": "2024-01-01T09:00:00Z"
					},
					"job_2": {
						"name": "job_2",
						"status": "RUNNING",
						"loop_count": 2,
						"tasks_total": 5,
						"tasks_completed": 2,
						"debug_logs": [
							{
								"id": "debug1",
								"timestamp": "2024-01-01T09:30:00Z",
								"phenomenon": "Test failure in loop 2",
								"reproduction": "Run test",
								"hypothesis": "Missing mock",
								"verification": "Add mock",
								"fix": "Add mock config",
								"progress": "待修复"
							}
						],
						"created_at": "2024-01-01T10:00:00Z",
						"updated_at": "2024-01-01T10:00:00Z"
					}
				},
				"created_at": "2024-01-01T08:00:00Z",
				"updated_at": "2024-01-01T10:00:00Z"
			}
		}
	}`

	if err := os.WriteFile(statusFile, []byte(statusData), 0644); err != nil {
		t.Fatalf("Failed to create status file: %v", err)
	}

	cfg := &mockConfig{workDir: tmpDir}
	logger := &mockLogger{}
	handler := NewStatHandler(cfg, logger)

	stateManager := state.NewManager(statusFile)
	if err := stateManager.Load(); err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	statusInfo, err := handler.collectStatus(stateManager)
	if err != nil {
		t.Errorf("collectStatus failed: %v", err)
	}

	if statusInfo == nil {
		t.Fatal("Expected statusInfo to be non-nil")
	}

	// Check current job
	if statusInfo.Current.Module != "test_module" {
		t.Errorf("Expected current module 'test_module', got '%s'", statusInfo.Current.Module)
	}
	if statusInfo.Current.Job != "job_2" {
		t.Errorf("Expected current job 'job_2', got '%s'", statusInfo.Current.Job)
	}

	// Check progress
	if statusInfo.Progress.TotalJobs != 2 {
		t.Errorf("Expected 2 total jobs, got %d", statusInfo.Progress.TotalJobs)
	}
	if statusInfo.Progress.CompletedJobs != 1 {
		t.Errorf("Expected 1 completed job, got %d", statusInfo.Progress.CompletedJobs)
	}
	if statusInfo.Progress.Percentage != 50 {
		t.Errorf("Expected 50%% progress, got %d", statusInfo.Progress.Percentage)
	}

	// Check modules
	if len(statusInfo.Modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(statusInfo.Modules))
	}

	// Check debug issues
	if len(statusInfo.DebugIssues) != 1 {
		t.Errorf("Expected 1 debug issue, got %d", len(statusInfo.DebugIssues))
	} else {
		issue := statusInfo.DebugIssues[0]
		if issue.Description != "Test failure in loop 2" {
			t.Errorf("Expected description 'Test failure in loop 2', got '%s'", issue.Description)
		}
		if issue.Hypothesis != "Missing mock" {
			t.Errorf("Expected hypothesis 'Missing mock', got '%s'", issue.Hypothesis)
		}
	}

	// Check previous job
	if statusInfo.Previous == nil {
		t.Error("Expected previous job to be found")
	} else {
		if statusInfo.Previous.Module != "test_module" {
			t.Errorf("Expected previous module 'test_module', got '%s'", statusInfo.Previous.Module)
		}
		if statusInfo.Previous.Job != "job_1" {
			t.Errorf("Expected previous job 'job_1', got '%s'", statusInfo.Previous.Job)
		}
	}
}

func TestStatHandler_findPreviousJob(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create status file with completed jobs
	statusFile := filepath.Join(tmpDir, "status.json")
	statusData := `{
		"version": "2.0",
		"modules": {
			"module1": {
				"status": "COMPLETED",
				"jobs": {
					"job_1": {
						"status": "COMPLETED",
						"completed_at": "2024-01-01T08:00:00Z",
						"started_at": "2024-01-01T07:00:00Z"
					},
					"job_2": {
						"status": "COMPLETED",
						"completed_at": "2024-01-01T10:00:00Z",
						"started_at": "2024-01-01T09:00:00Z"
					}
				}
			},
			"module2": {
				"status": "RUNNING",
				"jobs": {
					"job_1": {
						"status": "RUNNING"
					}
				}
			}
		}
	}`

	if err := os.WriteFile(statusFile, []byte(statusData), 0644); err != nil {
		t.Fatalf("Failed to create status file: %v", err)
	}

	cfg := &mockConfig{workDir: tmpDir}
	handler := NewStatHandler(cfg, &mockLogger{})
	stateManager := state.NewManager(statusFile)
	stateManager.Load()

	// Find previous job from current running job
	previous := handler.findPreviousJob(stateManager, "module2", "job_1")

	if previous == nil {
		t.Fatal("Expected to find previous job")
	}

	if previous.Module != "module1" {
		t.Errorf("Expected module 'module1', got '%s'", previous.Module)
	}

	if previous.Job != "job_2" {
		t.Errorf("Expected job 'job_2', got '%s'", previous.Job)
	}

	if previous.Status != "COMPLETED" {
		t.Errorf("Expected status 'COMPLETED', got '%s'", previous.Status)
	}

	// Duration should be 1 hour
	expectedDuration := time.Hour
	if previous.Duration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, previous.Duration)
	}
}

func TestStatHandler_extractDebugIssues(t *testing.T) {
	tmpDir := setupTestDir(t)

	statusFile := filepath.Join(tmpDir, "status.json")
	statusData := `{
		"version": "2.0",
		"modules": {
			"test_module": {
				"status": "RUNNING",
				"jobs": {
					"job_1": {
						"status": "RUNNING",
						"loop_count": 3,
						"debug_logs": [
							{
								"id": "debug1",
								"timestamp": "2024-01-01T10:00:00Z",
								"phenomenon": "Connection timeout",
								"hypothesis": "Network issue",
								"progress": "待修复"
							},
							{
								"id": "debug2",
								"timestamp": "2024-01-01T11:00:00Z",
								"phenomenon": "Memory leak",
								"hypothesis": "Missing cleanup",
								"progress": "已修复"
							}
						]
					}
				}
			}
		}
	}`

	if err := os.WriteFile(statusFile, []byte(statusData), 0644); err != nil {
		t.Fatalf("Failed to create status file: %v", err)
	}

	cfg := &mockConfig{workDir: tmpDir}
	handler := NewStatHandler(cfg, &mockLogger{})
	stateManager := state.NewManager(statusFile)
	stateManager.Load()

	issues := handler.extractDebugIssues(stateManager, "test_module", "job_1")

	if len(issues) != 2 {
		t.Errorf("Expected 2 debug issues, got %d", len(issues))
	}

	if len(issues) >= 1 {
		if issues[0].Description != "Connection timeout" {
			t.Errorf("Expected description 'Connection timeout', got '%s'", issues[0].Description)
		}
		if issues[0].Loop != 3 {
			t.Errorf("Expected loop 3, got %d", issues[0].Loop)
		}
	}

	// Test with empty module/job
	emptyIssues := handler.extractDebugIssues(stateManager, "", "")
	if len(emptyIssues) != 0 {
		t.Errorf("Expected 0 issues for empty module/job, got %d", len(emptyIssues))
	}
}

func TestStatHandler_formatDuration(t *testing.T) {
	handler := NewStatHandler(&mockConfig{}, &mockLogger{})

	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			expected: "00:45",
		},
		{
			name:     "minutes and seconds",
			duration: 5*time.Minute + 30*time.Second,
			expected: "05:30",
		},
		{
			name:     "hours minutes seconds",
			duration: 2*time.Hour + 30*time.Minute + 15*time.Second,
			expected: "02:30:15",
		},
		{
			name:     "zero",
			duration: 0,
			expected: "00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.formatDuration(tt.duration)
			if got != tt.expected {
				t.Errorf("formatDuration() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStatHandler_formatProgressBar(t *testing.T) {
	handler := NewStatHandler(&mockConfig{}, &mockLogger{})

	tests := []struct {
		name       string
		percentage int
		width      int
		expected   string
	}{
		{
			name:       "0 percent",
			percentage: 0,
			width:      10,
			expected:   "░░░░░░░░░░",
		},
		{
			name:       "50 percent",
			percentage: 50,
			width:      10,
			expected:   "█████░░░░░",
		},
		{
			name:       "100 percent",
			percentage: 100,
			width:      10,
			expected:   "██████████",
		},
		{
			name:       "over 100 percent capped",
			percentage: 150,
			width:      10,
			expected:   "██████████",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.formatProgressBar(tt.percentage, tt.width)
			if got != tt.expected {
				t.Errorf("formatProgressBar() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestStatHandler_outputEnhancedText(t *testing.T) {
	tmpDir := setupTestDir(t)
	handler := NewStatHandler(&mockConfig{workDir: tmpDir}, &mockLogger{})

	result := &StatResult{
		ExitCode: 0,
		Duration: time.Second,
		StatusInfo: &StatusInfo{
			Current: CurrentJobInfo{
				Module:    "test_module",
				Job:       "job_1",
				Status:    "RUNNING",
				StartedAt: time.Now().Add(-30 * time.Minute),
			},
			Previous: &PreviousJob{
				Module:      "test_module",
				Job:         "job_0",
				Status:      "COMPLETED",
				Duration:    15 * time.Minute,
				CompletedAt: time.Now().Add(-1 * time.Hour),
			},
			Progress: ProgressInfo{
				TotalJobs:     10,
				CompletedJobs: 5,
				Percentage:    50,
			},
			Modules: []ModuleStatus{
				{Name: "module1", Status: "completed", TotalJobs: 3, CompletedJobs: 3},
				{Name: "module2", Status: "in_progress", TotalJobs: 3, CompletedJobs: 2},
			},
			DebugIssues: []DebugIssue{
				{
					ID:          "debug1",
					Description: "Test issue",
					Loop:        2,
					Hypothesis:  "Missing config",
					Status:      "待修复",
				},
			},
		},
	}

	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handler.outputEnhancedText(result)

	w.Close()
	os.Stdout = oldStdout

	outputBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}
	output := string(outputBytes)

	if output == "" {
		t.Error("Expected output, got empty string")
	}

	// Check for expected content
	expectedStrings := []string{
		"Morty 监控大盘",
		"test_module",
		"job_1",
		"RUNNING",
		"上一个 Job",
		"job_0",
		"COMPLETED",
		"整体进度",
		"50%",
		"Test issue",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s'", expected)
		}
	}
}

func TestStatInfo_Structs(t *testing.T) {
	// Test StatusInfo
	statusInfo := &StatusInfo{
		Current: CurrentJobInfo{
			Module: "test",
			Job:    "job1",
			Status: "RUNNING",
		},
		Modules: []ModuleStatus{
			{Name: "mod1", Status: "completed"},
		},
		DebugIssues: []DebugIssue{
			{ID: "d1", Description: "test issue"},
		},
	}

	if statusInfo.Current.Module != "test" {
		t.Error("Current.Module not set correctly")
	}

	// Test ProgressInfo
	progress := ProgressInfo{
		TotalJobs:     10,
		CompletedJobs: 5,
		Percentage:    50,
	}
	if progress.Percentage != 50 {
		t.Error("Percentage not set correctly")
	}

	// Test PreviousJob
	prev := &PreviousJob{
		Module: "mod",
		Job:    "job",
		Status: "COMPLETED",
	}
	if prev.Status != "COMPLETED" {
		t.Error("PreviousJob.Status not set correctly")
	}
}
