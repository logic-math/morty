// Package executor provides job execution engine for Morty.
package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/morty/morty/internal/parser/plan"
)

// setupResultParserTest creates a temporary test environment.
func setupResultParserTest(t *testing.T) (string, *mockLogger, func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "result_parser_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create directory structure
	planDir := filepath.Join(tempDir, ".morty", "plan")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create plan dir: %v", err)
	}

	// Create test plan file
	planContent := `# Plan: test

## 模块概述

**模块职责**: 测试模块

## Jobs

### Job 1: 测试任务

**目标**: 实现测试功能

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建测试文件
- [ ] Task 2: 实现测试逻辑

**验证器**:
- [ ] 测试通过

**调试日志**:
- 无

### Job 2: 另一个任务

**目标**: 实现其他功能

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建文件

**验证器**:
- [ ] 验证通过

**调试日志**:
- debug1: 旧问题, 复现步骤, 猜想, 验证, 修复, 已修复
`

	planPath := filepath.Join(planDir, "test.md")
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write plan file: %v", err)
	}

	logger := &mockLogger{}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, logger, cleanup
}

// TestNewResultParser tests the NewResultParser constructor.
func TestNewResultParser(t *testing.T) {
	logger := &mockLogger{}

	// Test with nil config
	rp := NewResultParser(logger, nil)
	if rp == nil {
		t.Error("NewResultParser with nil config should return non-nil parser")
	}

	// Test with custom config
	config := &ResultParserConfig{
		PlanDir: "custom/plan/dir",
	}
	rp = NewResultParser(logger, config)
	if rp == nil {
		t.Error("NewResultParser with custom config should return non-nil parser")
	}
}

// TestParse_NestedFormat tests parsing nested RALPH_STATUS format.
func TestParse_NestedFormat(t *testing.T) {
	tempDir, logger, cleanup := setupResultParserTest(t)
	defer cleanup()

	config := &ResultParserConfig{
		PlanDir: filepath.Join(tempDir, ".morty", "plan"),
	}
	rp := NewResultParser(logger, config)

	// Create test output file with nested format
	outputContent := `Some output here

<!-- RALPH_STATUS -->
{
  "ralph_status": {
    "module": "test",
    "job": "job_1",
    "status": "COMPLETED",
    "tasks_completed": 3,
    "tasks_total": 3,
    "loop_count": 2,
    "debug_issues": 0,
    "debug_logs_in_plan": true,
    "explore_subagent_used": false,
    "summary": "All tasks completed successfully"
  }
}
<!-- END_RALPH_STATUS -->
`

	outputFile := filepath.Join(tempDir, "output.txt")
	if err := os.WriteFile(outputFile, []byte(outputContent), 0644); err != nil {
		t.Fatalf("Failed to write output file: %v", err)
	}

	result, err := rp.Parse(outputFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify result
	if result.Module != "test" {
		t.Errorf("Expected module 'test', got '%s'", result.Module)
	}
	if result.Job != "job_1" {
		t.Errorf("Expected job 'job_1', got '%s'", result.Job)
	}
	if result.Status != "COMPLETED" {
		t.Errorf("Expected status 'COMPLETED', got '%s'", result.Status)
	}
	if result.TasksCompleted != 3 {
		t.Errorf("Expected tasks_completed 3, got %d", result.TasksCompleted)
	}
	if result.TasksTotal != 3 {
		t.Errorf("Expected tasks_total 3, got %d", result.TasksTotal)
	}
	if result.LoopCount != 2 {
		t.Errorf("Expected loop_count 2, got %d", result.LoopCount)
	}
	if !result.IsSuccess() {
		t.Error("Expected IsSuccess() to be true")
	}
}

// TestParse_FlatFormat tests parsing flat RALPH_STATUS format.
func TestParse_FlatFormat(t *testing.T) {
	tempDir, logger, cleanup := setupResultParserTest(t)
	defer cleanup()

	config := &ResultParserConfig{
		PlanDir: filepath.Join(tempDir, ".morty", "plan"),
	}
	rp := NewResultParser(logger, config)

	// Create test output file with flat format
	outputContent := `Some output here

<!-- RALPH_STATUS -->
{
  "module": "test",
  "job": "job_2",
  "status": "FAILED",
  "tasks_completed": 1,
  "tasks_total": 3,
  "loop_count": 1,
  "debug_issues": 1,
  "summary": "Task execution failed"
}
<!-- END_RALPH_STATUS -->
`

	outputFile := filepath.Join(tempDir, "output.txt")
	if err := os.WriteFile(outputFile, []byte(outputContent), 0644); err != nil {
		t.Fatalf("Failed to write output file: %v", err)
	}

	result, err := rp.Parse(outputFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify result
	if result.Module != "test" {
		t.Errorf("Expected module 'test', got '%s'", result.Module)
	}
	if result.Job != "job_2" {
		t.Errorf("Expected job 'job_2', got '%s'", result.Job)
	}
	if result.Status != "FAILED" {
		t.Errorf("Expected status 'FAILED', got '%s'", result.Status)
	}
	if result.TasksCompleted != 1 {
		t.Errorf("Expected tasks_completed 1, got %d", result.TasksCompleted)
	}
	if result.TasksTotal != 3 {
		t.Errorf("Expected tasks_total 3, got %d", result.TasksTotal)
	}
	if !result.IsFailed() {
		t.Error("Expected IsFailed() to be true")
	}
}

// TestParse_WithoutMarkers tests parsing RALPH_STATUS without markers.
func TestParse_WithoutMarkers(t *testing.T) {
	tempDir, logger, cleanup := setupResultParserTest(t)
	defer cleanup()

	config := &ResultParserConfig{
		PlanDir: filepath.Join(tempDir, ".morty", "plan"),
	}
	rp := NewResultParser(logger, config)

	// Create test output file without markers but with JSON
	outputContent := `Some output here

{
  "module": "test",
  "job": "job_1",
  "status": "COMPLETED",
  "tasks_completed": 2,
  "tasks_total": 2,
  "summary": "Done"
}
`

	outputFile := filepath.Join(tempDir, "output.txt")
	if err := os.WriteFile(outputFile, []byte(outputContent), 0644); err != nil {
		t.Fatalf("Failed to write output file: %v", err)
	}

	result, err := rp.Parse(outputFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Status != "COMPLETED" {
		t.Errorf("Expected status 'COMPLETED', got '%s'", result.Status)
	}
}

// TestParse_FileNotFound tests parsing when file doesn't exist.
func TestParse_FileNotFound(t *testing.T) {
	logger := &mockLogger{}
	rp := NewResultParser(logger, nil)

	_, err := rp.Parse("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "failed to read output file") {
		t.Errorf("Expected 'failed to read output file' error, got: %v", err)
	}
}

// TestParse_InvalidJSON tests parsing with invalid JSON.
func TestParse_InvalidJSON(t *testing.T) {
	tempDir, logger, cleanup := setupResultParserTest(t)
	defer cleanup()

	config := &ResultParserConfig{
		PlanDir: filepath.Join(tempDir, ".morty", "plan"),
	}
	rp := NewResultParser(logger, config)

	// Create test output file with invalid JSON
	outputContent := `Some output here

<!-- RALPH_STATUS -->
{
  "invalid json here
}
<!-- END_RALPH_STATUS -->
`

	outputFile := filepath.Join(tempDir, "output.txt")
	if err := os.WriteFile(outputFile, []byte(outputContent), 0644); err != nil {
		t.Fatalf("Failed to write output file: %v", err)
	}

	_, err := rp.Parse(outputFile)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// TestParse_MissingStatus tests parsing with missing status field.
func TestParse_MissingStatus(t *testing.T) {
	tempDir, logger, cleanup := setupResultParserTest(t)
	defer cleanup()

	config := &ResultParserConfig{
		PlanDir: filepath.Join(tempDir, ".morty", "plan"),
	}
	rp := NewResultParser(logger, config)

	// Create test output file without status field
	outputContent := `Some output here

<!-- RALPH_STATUS -->
{
  "module": "test",
  "job": "job_1",
  "tasks_completed": 2,
  "tasks_total": 2
}
<!-- END_RALPH_STATUS -->
`

	outputFile := filepath.Join(tempDir, "output.txt")
	if err := os.WriteFile(outputFile, []byte(outputContent), 0644); err != nil {
		t.Fatalf("Failed to write output file: %v", err)
	}

	_, err := rp.Parse(outputFile)
	if err == nil {
		t.Error("Expected error for missing status field")
	}
	if !strings.Contains(err.Error(), "missing required field: status") {
		t.Errorf("Expected 'missing required field: status' error, got: %v", err)
	}
}

// TestRALPHExecutionResult_IsSuccess tests the IsSuccess method.
func TestRALPHExecutionResult_IsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"completed uppercase", "COMPLETED", true},
		{"completed lowercase", "completed", true},
		{"completed mixed", "Completed", true},
		{"failed", "FAILED", false},
		{"running", "RUNNING", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &RALPHExecutionResult{Status: tt.status}
			if result.IsSuccess() != tt.expected {
				t.Errorf("IsSuccess() = %v, expected %v", result.IsSuccess(), tt.expected)
			}
		})
	}
}

// TestRALPHExecutionResult_IsFailed tests the IsFailed method.
func TestRALPHExecutionResult_IsFailed(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"failed uppercase", "FAILED", true},
		{"failed lowercase", "failed", true},
		{"completed", "COMPLETED", false},
		{"running", "RUNNING", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &RALPHExecutionResult{Status: tt.status}
			if result.IsFailed() != tt.expected {
				t.Errorf("IsFailed() = %v, expected %v", result.IsFailed(), tt.expected)
			}
		})
	}
}

// TestRALPHExecutionResult_IsRunning tests the IsRunning method.
func TestRALPHExecutionResult_IsRunning(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"running uppercase", "RUNNING", true},
		{"running lowercase", "running", true},
		{"completed", "COMPLETED", false},
		{"failed", "FAILED", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &RALPHExecutionResult{Status: tt.status}
			if result.IsRunning() != tt.expected {
				t.Errorf("IsRunning() = %v, expected %v", result.IsRunning(), tt.expected)
			}
		})
	}
}

// TestExtractErrors tests the extractErrors method.
func TestExtractErrors(t *testing.T) {
	rp := &resultParser{}

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "no errors",
			content:  "Some normal output\nNo errors here",
			expected: 0,
		},
		{
			name:     "single error",
			content:  "Error: something went wrong. Continuing...",
			expected: 1,
		},
		{
			name:     "multiple errors",
			content:  "Error: first error. Error: second error.",
			expected: 2,
		},
		{
			name:     "failed pattern",
			content:  "Failed: operation timed out. Please retry.",
			expected: 1,
		},
		{
			name:     "exception pattern",
			content:  "Exception: null pointer exception occurred.",
			expected: 1,
		},
		{
			name:     "panic pattern",
			content:  "Panic: runtime error.",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := rp.extractErrors(tt.content)
			if len(errors) != tt.expected {
				t.Errorf("Expected %d errors, got %d: %v", tt.expected, len(errors), errors)
			}
		})
	}
}

// TestCreateDebugLog tests the CreateDebugLog function.
func TestCreateDebugLog(t *testing.T) {
	log := CreateDebugLog(
		"debug1",
		"Something broke",
		"Run the test",
		"Bad code",
		"Add logging",
		"Refactor",
		"In progress",
	)

	if log.ID != "debug1" {
		t.Errorf("Expected ID 'debug1', got '%s'", log.ID)
	}
	if log.Phenomenon != "Something broke" {
		t.Errorf("Expected Phenomenon 'Something broke', got '%s'", log.Phenomenon)
	}
	if log.Reproduction != "Run the test" {
		t.Errorf("Expected Reproduction 'Run the test', got '%s'", log.Reproduction)
	}
	if log.Hypothesis != "Bad code" {
		t.Errorf("Expected Hypothesis 'Bad code', got '%s'", log.Hypothesis)
	}
	if log.Verification != "Add logging" {
		t.Errorf("Expected Verification 'Add logging', got '%s'", log.Verification)
	}
	if log.Fix != "Refactor" {
		t.Errorf("Expected Fix 'Refactor', got '%s'", log.Fix)
	}
	if log.Progress != "In progress" {
		t.Errorf("Expected Progress 'In progress', got '%s'", log.Progress)
	}
}

// TestUpdatePlanDebugLogs tests updating plan debug logs.
func TestUpdatePlanDebugLogs(t *testing.T) {
	tempDir, logger, cleanup := setupResultParserTest(t)
	defer cleanup()

	config := &ResultParserConfig{
		PlanDir: filepath.Join(tempDir, ".morty", "plan"),
	}
	rp := NewResultParser(logger, config).(*resultParser)

	// Create new debug logs
	newLogs := []plan.DebugLog{
		{
			ID:           "debug2",
			Phenomenon:   "New issue",
			Reproduction: "Run test",
			Hypothesis:   "Bad config",
			Verification: "Check config",
			Fix:          "Fix config",
			Progress:     "Fixed",
		},
	}

	err := rp.UpdatePlanDebugLogs("test", "job_2", newLogs)
	if err != nil {
		t.Fatalf("UpdatePlanDebugLogs failed: %v", err)
	}

	// Read updated plan file
	planPath := filepath.Join(tempDir, ".morty", "plan", "test.md")
	content, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("Failed to read updated plan: %v", err)
	}

	// Verify new debug log was added
	if !strings.Contains(string(content), "debug2") {
		t.Error("Updated plan should contain debug2")
	}
	if !strings.Contains(string(content), "New issue") {
		t.Error("Updated plan should contain 'New issue'")
	}
}

// TestUpdatePlanDebugLogs_JobNotFound tests updating debug logs for non-existent job.
func TestUpdatePlanDebugLogs_JobNotFound(t *testing.T) {
	tempDir, logger, cleanup := setupResultParserTest(t)
	defer cleanup()

	config := &ResultParserConfig{
		PlanDir: filepath.Join(tempDir, ".morty", "plan"),
	}
	rp := NewResultParser(logger, config).(*resultParser)

	newLogs := []plan.DebugLog{
		{ID: "debug1", Phenomenon: "Test"},
	}

	err := rp.UpdatePlanDebugLogs("test", "job_99", newLogs)
	if err == nil {
		t.Error("Expected error for non-existent job")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// TestUpdatePlanDebugLogs_EmptyLogs tests updating with empty debug logs.
func TestUpdatePlanDebugLogs_EmptyLogs(t *testing.T) {
	tempDir, logger, cleanup := setupResultParserTest(t)
	defer cleanup()

	config := &ResultParserConfig{
		PlanDir: filepath.Join(tempDir, ".morty", "plan"),
	}
	rp := NewResultParser(logger, config).(*resultParser)

	err := rp.UpdatePlanDebugLogs("test", "job_1", []plan.DebugLog{})
	if err != nil {
		t.Error("UpdatePlanDebugLogs with empty logs should not error")
	}
}

// TestDefaultResultParserConfig tests the default config.
func TestDefaultResultParserConfig(t *testing.T) {
	config := DefaultResultParserConfig()
	if config == nil {
		t.Fatal("DefaultResultParserConfig should return non-nil config")
	}
	if config.PlanDir != ".morty/plan" {
		t.Errorf("Expected PlanDir '.morty/plan', got '%s'", config.PlanDir)
	}
}

// TestParseErrorOutput tests parsing error output.
func TestParseErrorOutput(t *testing.T) {
	rp := &resultParser{}

	output := `Some normal output
Error: connection refused
	at /path/to/file.go:123
	at /path/to/other.go:456
More output
panic: runtime error
	at main.go:10
`

	errors := rp.ParseErrorOutput(output)

	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}

	if len(errors) > 0 {
		if errors[0].Type != "Error" {
			t.Errorf("Expected first error type 'Error', got '%s'", errors[0].Type)
		}
		if errors[0].Message != "connection refused" {
			t.Errorf("Expected message 'connection refused', got '%s'", errors[0].Message)
		}
	}

	if len(errors) > 1 {
		if errors[1].Type != "Panic" {
			t.Errorf("Expected second error type 'Panic', got '%s'", errors[1].Type)
		}
	}
}

// TestResultParser_ExtractStderr tests extracting stderr from output.
func TestResultParser_ExtractStderr(t *testing.T) {
	rp := &resultParser{}

	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "no stderr",
			content:  "Some normal output",
			expected: "",
		},
		{
			name:     "stderr section",
			content:  "Output\nStderr: some error message\nMore output",
			expected: "some error message",
		},
		{
			name:     "standard error section",
			content:  "Output\nStandard Error: another error\n---",
			expected: "another error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rp.extractStderr(tt.content)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestResultParser_Interface ensures resultParser implements ResultParser interface.
func TestResultParser_Interface(t *testing.T) {
	logger := &mockLogger{}
	var _ ResultParser = NewResultParser(logger, nil)
}
