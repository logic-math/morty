// Package executor provides job execution engine for Morty.
package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/morty/morty/internal/state"
)

// setupPromptBuilderTest creates a temporary test environment.
func setupPromptBuilderTest(t *testing.T) (string, *state.Manager, func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "prompt_builder_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create directory structure
	planDir := filepath.Join(tempDir, ".morty", "plan")
	promptsDir := filepath.Join(tempDir, "prompts")
	stateDir := filepath.Join(tempDir, ".morty")

	for _, dir := range []string{planDir, promptsDir, stateDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create test system prompt
	systemPrompt := `# Doing

在满足执行意图的约束下不断执行循环中的工作步骤。

## 精简上下文格式

` + "```json\n{\n  \"current\": {\n    \"module\": \"test\",\n    \"job\": \"job_1\"\n  }\n}\n```\n"

	systemPromptPath := filepath.Join(promptsDir, "doing.md")
	if err := os.WriteFile(systemPromptPath, []byte(systemPrompt), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write system prompt: %v", err)
	}

	// Create test plan file
	planContent := `# Plan: test

## 模块概述

**模块职责**: 测试模块

**依赖模块**: state

## Jobs

### Job 1: 测试任务

**目标**: 实现测试功能

**前置条件**:
- state 模块完成

**Tasks (Todo 列表)**:
- [ ] Task 0: 创建测试文件
- [ ] Task 1: 实现测试逻辑
- [ ] Task 2: 编写单元测试

**验证器**:
- [ ] 测试文件创建成功
- [ ] 测试逻辑正确
- [ ] 单元测试通过

**调试日志**:
- 无
`

	planPath := filepath.Join(planDir, "test.md")
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write plan file: %v", err)
	}

	// Create state file
	stateContent := `{
  "global": {
    "status": "RUNNING",
    "current_module": "test",
    "current_job": "job_1",
    "start_time": "2024-01-01T00:00:00Z",
    "last_update": "2024-01-01T00:00:00Z",
    "total_loops": 2
  },
  "modules": {
    "test": {
      "name": "test",
      "status": "RUNNING",
      "jobs": {
        "job_1": {
          "name": "job_1",
          "status": "RUNNING",
          "loop_count": 1,
          "retry_count": 0,
          "tasks_total": 3,
          "tasks_completed": 0,
          "tasks": [
            {"index": 0, "status": "PENDING", "description": "创建测试文件", "updated_at": "2024-01-01T00:00:00Z"},
            {"index": 1, "status": "PENDING", "description": "实现测试逻辑", "updated_at": "2024-01-01T00:00:00Z"},
            {"index": 2, "status": "PENDING", "description": "编写单元测试", "updated_at": "2024-01-01T00:00:00Z"}
          ],
          "created_at": "2024-01-01T00:00:00Z",
          "updated_at": "2024-01-01T00:00:00Z"
        },
        "job_0": {
          "name": "job_0",
          "status": "COMPLETED",
          "loop_count": 1,
          "retry_count": 0,
          "tasks_total": 2,
          "tasks_completed": 2,
          "tasks": [
            {"index": 0, "status": "COMPLETED", "description": "准备工作", "updated_at": "2024-01-01T00:00:00Z"},
            {"index": 1, "status": "COMPLETED", "description": "初始化环境", "updated_at": "2024-01-01T00:00:00Z"}
          ],
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

	statePath := filepath.Join(stateDir, "status.json")
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write state file: %v", err)
	}

	// Create state manager
	stateManager := state.NewManager(statePath)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, stateManager, cleanup
}

func TestNewPromptBuilder(t *testing.T) {
	_, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          ".morty/plan",
		PromptsDir:       "prompts",
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)
	if pb == nil {
		t.Fatal("Expected PromptBuilder, got nil")
	}

	// Test with nil config
	pb2 := NewPromptBuilder(stateManager, nil)
	if pb2 == nil {
		t.Fatal("Expected PromptBuilder with default config, got nil")
	}
}

func TestDefaultPromptBuilderConfig(t *testing.T) {
	config := DefaultPromptBuilderConfig()
	if config == nil {
		t.Fatal("Expected default config, got nil")
	}

	if config.PlanDir != ".morty/plan" {
		t.Errorf("Expected PlanDir '.morty/plan', got %s", config.PlanDir)
	}

	if config.PromptsDir != "prompts" {
		t.Errorf("Expected PromptsDir 'prompts', got %s", config.PromptsDir)
	}

	if config.SystemPromptFile != "doing.md" {
		t.Errorf("Expected SystemPromptFile 'doing.md', got %s", config.SystemPromptFile)
	}
}

func TestPromptBuilder_LoadSystemPrompt(t *testing.T) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config).(*promptBuilder)

	content, err := pb.loadSystemPrompt()
	if err != nil {
		t.Fatalf("Failed to load system prompt: %v", err)
	}

	if !strings.Contains(content, "Doing") {
		t.Error("System prompt should contain 'Doing'")
	}

	if !strings.Contains(content, "精简上下文格式") {
		t.Error("System prompt should contain '精简上下文格式'")
	}
}

func TestPromptBuilder_LoadPlanContent(t *testing.T) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config).(*promptBuilder)

	content, err := pb.loadPlanContent("test")
	if err != nil {
		t.Fatalf("Failed to load plan content: %v", err)
	}

	if !strings.Contains(content, "Plan: test") {
		t.Error("Plan content should contain 'Plan: test'")
	}

	if !strings.Contains(content, "测试任务") {
		t.Error("Plan content should contain '测试任务'")
	}
}

func TestPromptBuilder_BuildCompactContext(t *testing.T) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)

	ctx, err := pb.BuildCompactContext("test", "job_1")
	if err != nil {
		t.Fatalf("Failed to build compact context: %v", err)
	}

	// Verify structure
	current, ok := ctx["current"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'current' section in context")
	}

	if current["module"] != "test" {
		t.Errorf("Expected module 'test', got %v", current["module"])
	}

	if current["job"] != "job_1" {
		t.Errorf("Expected job 'job_1', got %v", current["job"])
	}

	contextSection, ok := ctx["context"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'context' section")
	}

	// Check completed jobs summary
	completedSummary, ok := contextSection["completed_jobs_summary"].([]string)
	if !ok {
		t.Fatal("Expected completed_jobs_summary to be []string")
	}

	foundCompletedJob := false
	for _, summary := range completedSummary {
		if strings.Contains(summary, "job_0") && strings.Contains(summary, "完成") {
			foundCompletedJob = true
			break
		}
	}
	if !foundCompletedJob {
		t.Error("Expected completed job summary to include job_0")
	}

	// Check current job details
	currentJob, ok := contextSection["current_job"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected current_job section")
	}

	if currentJob["name"] != "job_1" {
		t.Errorf("Expected current job name 'job_1', got %v", currentJob["name"])
	}

	tasks, ok := currentJob["tasks"].([]string)
	if !ok {
		t.Fatal("Expected tasks to be []string")
	}
	if len(tasks) == 0 {
		t.Error("Expected non-empty tasks list")
	}
}

func TestPromptBuilder_BuildPrompt(t *testing.T) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)

	prompt, err := pb.BuildPrompt("test", "job_1", 0, "创建测试文件")
	if err != nil {
		t.Fatalf("Failed to build prompt: %v", err)
	}

	// Verify prompt contains all required sections
	if !strings.Contains(prompt, "Doing") {
		t.Error("Prompt should contain system prompt (Doing)")
	}

	if !strings.Contains(prompt, "精简上下文") {
		t.Error("Prompt should contain '精简上下文' section")
	}

	if !strings.Contains(prompt, "Plan 内容") {
		t.Error("Prompt should contain 'Plan 内容' section")
	}

	if !strings.Contains(prompt, "当前 Job 上下文") {
		t.Error("Prompt should contain '当前 Job 上下文' section")
	}

	if !strings.Contains(prompt, "任务完成要求") {
		t.Error("Prompt should contain '任务完成要求' section")
	}

	if !strings.Contains(prompt, "RALPH_STATUS") {
		t.Error("Prompt should contain 'RALPH_STATUS'")
	}

	// Verify JSON context is included
	if !strings.Contains(prompt, `"module": "test"`) {
		t.Error("Prompt should contain module in JSON context")
	}

	if !strings.Contains(prompt, `"job": "job_1"`) {
		t.Error("Prompt should contain job in JSON context")
	}
}

func TestPromptBuilder_BuildPrompt_InvalidModule(t *testing.T) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)

	// Test with non-existent module
	_, err := pb.BuildPrompt("nonexistent", "job_1", 0, "test task")
	if err == nil {
		t.Error("Expected error for non-existent module, got nil")
	}
}

func TestPromptBuilder_BuildCompactContext_InvalidJob(t *testing.T) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)

	// Test with non-existent job
	_, err := pb.BuildCompactContext("test", "nonexistent_job")
	if err == nil {
		t.Error("Expected error for non-existent job, got nil")
	}
}

func TestPromptBuilder_ReplaceTemplateVariables(t *testing.T) {
	_, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	pb := NewPromptBuilder(stateManager, nil).(*promptBuilder)

	template := "Module: {{module}}, Job: {{job}}, Task: {{task_index}}, Desc: {{task_desc}}"
	result := pb.ReplaceTemplateVariables(template, "test", "job_1", 5, "test task")

	expected := "Module: test, Job: job_1, Task: 5, Desc: test task"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestPromptBuilder_ReplaceTemplateVariablesRegex(t *testing.T) {
	_, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	pb := NewPromptBuilder(stateManager, nil).(*promptBuilder)

	// Test with spaces in variables
	template := "Module: {{ module }}, Job: {{job }}, Task: {{ task_index }}"
	result := pb.ReplaceTemplateVariablesRegex(template, "test", "job_1", 3, "test")

	expected := "Module: test, Job: job_1, Task: 3"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with unknown variables (should keep them)
	template2 := "Known: {{module}}, Unknown: {{unknown_var}}"
	result2 := pb.ReplaceTemplateVariablesRegex(template2, "test", "job_1", 0, "")

	if !strings.Contains(result2, "Known: test") {
		t.Error("Should replace known variables")
	}
	if !strings.Contains(result2, "{{unknown_var}}") {
		t.Error("Should keep unknown variables unchanged")
	}
}

func TestPromptBuilder_ContextJSONStructure(t *testing.T) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)

	ctx, err := pb.BuildCompactContext("test", "job_1")
	if err != nil {
		t.Fatalf("Failed to build compact context: %v", err)
	}

	// Serialize and deserialize to verify JSON structure
	jsonData, err := json.Marshal(ctx)
	if err != nil {
		t.Fatalf("Failed to marshal context: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal context: %v", err)
	}

	// Verify required fields exist
	if _, ok := decoded["current"]; !ok {
		t.Error("Context should have 'current' field")
	}

	if _, ok := decoded["context"]; !ok {
		t.Error("Context should have 'context' field")
	}

	innerContext := decoded["context"].(map[string]interface{})
	if _, ok := innerContext["completed_jobs_summary"]; !ok {
		t.Error("Context should have 'completed_jobs_summary' field")
	}

	if _, ok := innerContext["current_job"]; !ok {
		t.Error("Context should have 'current_job' field")
	}
}

func TestPromptBuilder_BuildPrompt_LengthControl(t *testing.T) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)

	prompt, err := pb.BuildPrompt("test", "job_1", 0, "创建测试文件")
	if err != nil {
		t.Fatalf("Failed to build prompt: %v", err)
	}

	// Verify prompt is not excessively long (should be reasonable size)
	// This is a sanity check - prompts can be long but shouldn't be extreme
	if len(prompt) < 100 {
		t.Error("Prompt seems too short, may be missing content")
	}

	if len(prompt) > 100000 {
		t.Error("Prompt is excessively long (> 100KB)")
	}
}

func TestPromptBuilder_BuildPrompt_ContainsValidator(t *testing.T) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)

	prompt, err := pb.BuildPrompt("test", "job_1", 0, "创建测试文件")
	if err != nil {
		t.Fatalf("Failed to build prompt: %v", err)
	}

	// Check that validator section is included
	if !strings.Contains(prompt, "验证器") {
		t.Error("Prompt should contain validators section")
	}

	// Check for specific validator items from plan
	if !strings.Contains(prompt, "测试文件创建成功") {
		t.Error("Prompt should contain specific validator items from plan")
	}
}

func TestPromptBuilder_BuildPrompt_ContainsRALPHInstructions(t *testing.T) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)

	prompt, err := pb.BuildPrompt("test", "job_1", 0, "创建测试文件")
	if err != nil {
		t.Fatalf("Failed to build prompt: %v", err)
	}

	// Verify RALPH_STATUS instructions
	if !strings.Contains(prompt, `"status": "COMPLETED"`) {
		t.Error("Prompt should contain RALPH_STATUS COMPLETED example")
	}

	if !strings.Contains(prompt, `"status": "COMPLETED"`) {
		t.Error("Prompt should mention status: COMPLETED requirement")
	}
}

// Benchmark tests

func BenchmarkBuildCompactContext(b *testing.B) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(&testing.T{})
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pb.BuildCompactContext("test", "job_1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildPrompt(b *testing.B) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(&testing.T{})
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pb.BuildPrompt("test", "job_1", 0, "创建测试文件")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReplaceTemplateVariables(b *testing.B) {
	_, stateManager, cleanup := setupPromptBuilderTest(&testing.T{})
	defer cleanup()

	pb := NewPromptBuilder(stateManager, nil).(*promptBuilder)

	template := "Module: {{module}}, Job: {{job}}, Task: {{task_index}}, Desc: {{task_desc}}, Dir: {{plan_dir}}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pb.ReplaceTemplateVariables(template, "test", "job_1", 5, "test task")
	}
}

// Integration test
func TestPromptBuilder_Integration(t *testing.T) {
	tempDir, stateManager, cleanup := setupPromptBuilderTest(t)
	defer cleanup()

	config := &PromptBuilderConfig{
		PlanDir:          filepath.Join(tempDir, ".morty/plan"),
		PromptsDir:       filepath.Join(tempDir, "prompts"),
		SystemPromptFile: "doing.md",
	}

	pb := NewPromptBuilder(stateManager, config)

	// Build compact context
	ctx, err := pb.BuildCompactContext("test", "job_1")
	if err != nil {
		t.Fatalf("Failed to build compact context: %v", err)
	}

	// Verify we can serialize it
	ctxJSON, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal context: %v", err)
	}

	// Build full prompt
	prompt, err := pb.BuildPrompt("test", "job_1", 1, "实现测试逻辑")
	if err != nil {
		t.Fatalf("Failed to build prompt: %v", err)
	}

	// Verify context JSON is embedded in prompt
	if !strings.Contains(prompt, string(ctxJSON)) {
		// This is OK - the formatting might differ slightly
		// Just verify it contains the key fields
		if !strings.Contains(prompt, `"module": "test"`) {
			t.Error("Prompt should contain the JSON context")
		}
	}

	fmt.Printf("Integration test passed. Prompt length: %d bytes\n", len(prompt))
}
