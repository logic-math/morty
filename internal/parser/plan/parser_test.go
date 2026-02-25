package plan

import (
	"strings"
	"testing"

	"github.com/morty/morty/internal/parser/markdown"
)

// TestParsePlan_Basic tests basic plan parsing.
func TestParsePlan_Basic(t *testing.T) {
	content := `# Plan: TestModule

## 模块概述

**模块职责**: 测试模块职责

**对应 Research**:
- research1.md
- research2.md

**依赖模块**:
- module1
- module2

**被依赖模块**:
- module3

## Jobs

### Job 1: First Job

**目标**: 实现第一个功能

**前置条件**:
- 无

**Tasks (Todo 列表)**:
- [x] Task 1: 完成任务1
- [ ] Task 2: 完成任务2

**验证器**:
- [x] 验证器1通过
- [ ] 验证器2通过

**调试日志**:
- debug1: 问题1, 复现步骤1, 猜想1, 验证1, 修复1, 已修复

### Job 2: Second Job

**目标**: 实现第二个功能

**前置条件**:
- Job 1完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 开始任务1

**验证器**:
- [ ] 验证器通过
`

	plan, err := ParsePlan(content)
	if err != nil {
		t.Fatalf("ParsePlan() error = %v", err)
	}

	if plan.Name != "TestModule" {
		t.Errorf("plan.Name = %q, want %q", plan.Name, "TestModule")
	}

	if plan.Responsibility != "测试模块职责" {
		t.Errorf("plan.Responsibility = %q, want %q", plan.Responsibility, "测试模块职责")
	}

	// Check dependencies
	if len(plan.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(plan.Dependencies))
	}

	// Check jobs
	if len(plan.Jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(plan.Jobs))
	}
}

// TestExtractModuleName tests module name extraction.
func TestExtractModuleName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"# Plan: Parser", "Parser"},
		{"Plan: Config", "Config"},
		{"# Plan:  TestModule  ", "TestModule"},
		{"plan: lowercase", "lowercase"},
		{"## Plan: Git", "Git"},
		{"# Parser", "Parser"},
		{"TestModule", "TestModule"},
	}

	for _, tc := range tests {
		result := extractModuleName(tc.input)
		if result != tc.expected {
			t.Errorf("extractModuleName(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

// TestExtractField tests field extraction from content.
func TestExtractField(t *testing.T) {
	content := `**目标**: 实现功能
**模块职责**: 测试职责
其他内容
**前置条件**: 完成前置`

	tests := []struct {
		fieldName string
		expected  string
	}{
		{"目标", "实现功能"},
		{"模块职责", "测试职责"},
		{"前置条件", "完成前置"},
		{"不存在", ""},
	}

	for _, tc := range tests {
		result := extractField(content, tc.fieldName)
		if result != tc.expected {
			t.Errorf("extractField(content, %q) = %q, want %q", tc.fieldName, result, tc.expected)
		}
	}
}

// TestExtractListField tests list field extraction.
func TestExtractListField(t *testing.T) {
	content := `**依赖模块**:
- module1
- module2

**被依赖模块**:
- module3

**其他**: value`

	result := extractListField(content, "依赖模块")
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	if len(result) > 0 && result[0] != "module1" {
		t.Errorf("first item = %q, want %q", result[0], "module1")
	}
}

// TestExtractJobFromSection tests job extraction from section.
func TestExtractJobFromSection(t *testing.T) {
	content := `### Job 5: Test Job

**目标**: 测试目标

**前置条件**:
- 条件1
- 条件2

**Tasks (Todo 列表)**:
- [x] Task 1: 完成任务1
- [ ] Task 2: 未完成任务2

**验证器**:
- [ ] 验证器1
- [x] 验证器2

**调试日志**:
- debug1: 现象, 复现, 猜想, 验证, 修复, 进展
- explore1: 探索发现, 使用, 测试, 已记录
`

	// Create a mock section
	sec := mockSection{
		Title:   "Job 5: Test Job",
		Level:   3,
		Content: content,
	}

	job := extractJobFromSection(toMarkdownSection(sec))
	if job == nil {
		t.Fatal("expected job, got nil")
	}

	if job.Index != 5 {
		t.Errorf("job.Index = %d, want %d", job.Index, 5)
	}

	if job.Name != "Test Job" {
		t.Errorf("job.Name = %q, want %q", job.Name, "Test Job")
	}

	if job.Goal != "测试目标" {
		t.Errorf("job.Goal = %q, want %q", job.Goal, "测试目标")
	}

	if len(job.Prerequisites) != 2 {
		t.Errorf("expected 2 prerequisites, got %d", len(job.Prerequisites))
	}

	if len(job.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(job.Tasks))
	}

	// Check task completion status
	if !job.Tasks[0].Completed {
		t.Error("Task 1 should be completed")
	}
	if job.Tasks[1].Completed {
		t.Error("Task 2 should be pending")
	}

	if len(job.Validators) != 2 {
		t.Errorf("expected 2 validators, got %d", len(job.Validators))
	}

	if len(job.DebugLogs) != 2 {
		t.Errorf("expected 2 debug logs, got %d", len(job.DebugLogs))
	}
}

// TestExtractTasksFromContent tests task extraction.
func TestExtractTasksFromContent(t *testing.T) {
	content := `**Tasks (Todo 列表)**:
- [x] Task 1: 完成任务1
- [ ] Task 2: 未完成任务2
- [X] Task 3: 大写X完成

**验证器**:
- [ ] Validator 1`

	tasks := extractTasksFromContent(content)

	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	// Check task details
	if tasks[0].Index != 1 {
		t.Errorf("task[0].Index = %d, want %d", tasks[0].Index, 1)
	}
	if tasks[0].Description != "完成任务1" {
		t.Errorf("task[0].Description = %q, want %q", tasks[0].Description, "完成任务1")
	}
	if !tasks[0].Completed {
		t.Error("task[0] should be completed")
	}

	if tasks[1].Completed {
		t.Error("task[1] should be pending")
	}

	if !tasks[2].Completed {
		t.Error("task[2] should be completed (capital X)")
	}
}

// TestExtractValidators tests validator extraction.
func TestExtractValidators(t *testing.T) {
	content := `**验证器**:
- [x] 验证器1通过
- [ ] 验证器2未通过
- 验证器3无状态

**其他**: value`

	validators := extractValidators(content)

	if len(validators) != 3 {
		t.Errorf("expected 3 validators, got %d", len(validators))
	}

	expected := []string{"验证器1通过", "验证器2未通过", "验证器3无状态"}
	for i, v := range expected {
		if i < len(validators) && validators[i] != v {
			t.Errorf("validator[%d] = %q, want %q", i, validators[i], v)
		}
	}
}

// TestExtractDebugLogs tests debug log extraction.
func TestExtractDebugLogs(t *testing.T) {
	content := `**调试日志**:
- debug1: 问题描述, 复现步骤, 可能原因, 验证方法, 修复方案, 已修复
- explore1: 探索发现内容, 使用工具, 测试结果, 已记录

**其他**: value`

	logs := extractDebugLogs(content)

	if len(logs) != 2 {
		t.Fatalf("expected 2 debug logs, got %d", len(logs))
	}

	// Check first log
	if logs[0].ID != "debug1" {
		t.Errorf("log[0].ID = %q, want %q", logs[0].ID, "debug1")
	}
	if logs[0].Phenomenon != "问题描述" {
		t.Errorf("log[0].Phenomenon = %q, want %q", logs[0].Phenomenon, "问题描述")
	}
	if logs[0].Progress != "已修复" {
		t.Errorf("log[0].Progress = %q, want %q", logs[0].Progress, "已修复")
	}

	// Check second log
	if logs[1].ID != "explore1" {
		t.Errorf("log[1].ID = %q, want %q", logs[1].ID, "explore1")
	}
}

// TestPlanGetJobByIndex tests GetJobByIndex method.
func TestPlanGetJobByIndex(t *testing.T) {
	plan := &Plan{
		Jobs: []Job{
			{Index: 1, Name: "Job 1"},
			{Index: 5, Name: "Job 5"},
			{Index: 10, Name: "Job 10"},
		},
	}

	// Test finding existing job
	job, err := plan.GetJobByIndex(5)
	if err != nil {
		t.Errorf("GetJobByIndex(5) error = %v", err)
	}
	if job.Name != "Job 5" {
		t.Errorf("job.Name = %q, want %q", job.Name, "Job 5")
	}

	// Test finding non-existent job
	_, err = plan.GetJobByIndex(99)
	if err == nil {
		t.Error("GetJobByIndex(99) should return error")
	}
}

// TestPlanGetJobByName tests GetJobByName method.
func TestPlanGetJobByName(t *testing.T) {
	plan := &Plan{
		Jobs: []Job{
			{Index: 1, Name: "First Job"},
			{Index: 2, Name: "Second Job"},
		},
	}

	// Test finding existing job (case-insensitive)
	job, err := plan.GetJobByName("first job")
	if err != nil {
		t.Errorf("GetJobByName error = %v", err)
	}
	if job.Index != 1 {
		t.Errorf("job.Index = %d, want %d", job.Index, 1)
	}

	// Test finding non-existent job
	_, err = plan.GetJobByName("NonExistent")
	if err == nil {
		t.Error("GetJobByName should return error for non-existent job")
	}
}

// TestPlanCountTasks tests CountTasks method.
func TestPlanCountTasks(t *testing.T) {
	plan := &Plan{
		Jobs: []Job{
			{
				Tasks: []TaskItem{
					{Completed: true},
					{Completed: false},
				},
			},
			{
				Tasks: []TaskItem{
					{Completed: true},
					{Completed: true},
					{Completed: false},
				},
			},
		},
	}

	total, completed := plan.CountTasks()

	if total != 5 {
		t.Errorf("total = %d, want %d", total, 5)
	}
	if completed != 3 {
		t.Errorf("completed = %d, want %d", completed, 3)
	}
}

// TestPlanGetPendingTasks tests GetPendingTasks method.
func TestPlanGetPendingTasks(t *testing.T) {
	plan := &Plan{
		Jobs: []Job{
			{
				Tasks: []TaskItem{
					{Description: "Task 1", Completed: true},
					{Description: "Task 2", Completed: false},
				},
			},
			{
				Tasks: []TaskItem{
					{Description: "Task 3", Completed: false},
				},
			},
		},
	}

	pending := plan.GetPendingTasks()

	if len(pending) != 2 {
		t.Errorf("expected 2 pending tasks, got %d", len(pending))
	}

	// Verify we got the right tasks
	found := make(map[string]bool)
	for _, task := range pending {
		found[task.Description] = true
	}
	if !found["Task 2"] || !found["Task 3"] {
		t.Error("did not find expected pending tasks")
	}
}

// TestIsModuleOverviewTitle tests module overview title detection.
func TestIsModuleOverviewTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"模块概述", true},
		{"Module Overview", true},
		{"Overview", true},
		{"模块概述 (Module Overview)", true},
		{"Overview of Module", true},
		{"Jobs", false},
		{"Architecture", false},
	}

	for _, tc := range tests {
		result := isModuleOverviewTitle(tc.input)
		if result != tc.expected {
			t.Errorf("isModuleOverviewTitle(%q) = %v, want %v", tc.input, result, tc.expected)
		}
	}
}

// TestParsePlan_RealParserFile tests parsing with the actual parser.md file format.
func TestParsePlan_RealParserFile(t *testing.T) {
	content := `# Plan: Parser

## 模块概述

**模块职责**: 提供通用的文件解析框架，支持多种文件格式的解析。

**对应 Research**:
- plan-mode-design.md

**依赖模块**: 无

**被依赖模块**: plan_cmd, research_cmd, doing_cmd

## Jobs (Loop 块列表)

### Job 1: 解析器框架核心

**目标**: 实现 Parser 框架核心接口和工厂

**前置条件**:
- 无

**Tasks (Todo 列表)**:
- [x] Task 1: 创建 internal/parser/interface.go
- [ ] Task 2: 创建 internal/parser/factory.go

**验证器**:
- [x] 能根据扩展名正确检测文件类型
- [ ] 集成测试通过

**调试日志**:
- debug1: 初始文件创建路径错误, 文件写入位置问题, 工作目录不一致, 检查目录结构, 切换到正确目录, 已修复

### Job 2: Markdown 解析器基础

**目标**: 实现 Markdown 解析器

**前置条件**:
- Job 1 完成

**Tasks (Todo 列表)**:
- [ ] Task 1: 创建 internal/parser/markdown/parser.go
- [x] Task 2: 实现 Parse(content string)

**验证器**:
- [ ] 正确解析标题层级
`

	plan, err := ParsePlan(content)
	if err != nil {
		t.Fatalf("ParsePlan() error = %v", err)
	}

	if plan.Name != "Parser" {
		t.Errorf("plan.Name = %q, want %q", plan.Name, "Parser")
	}

	if plan.Responsibility != "提供通用的文件解析框架，支持多种文件格式的解析。" {
		t.Errorf("plan.Responsibility = %q", plan.Responsibility)
	}

	if len(plan.Jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(plan.Jobs))
	}

	// Check Job 1
	job1 := plan.Jobs[0]
	if job1.Index != 1 {
		t.Errorf("job1.Index = %d, want %d", job1.Index, 1)
	}
	if job1.Name != "解析器框架核心" {
		t.Errorf("job1.Name = %q, want %q", job1.Name, "解析器框架核心")
	}

	// Check Job 1 tasks
	if len(job1.Tasks) != 2 {
		t.Errorf("expected 2 tasks in job 1, got %d", len(job1.Tasks))
	}
	if !job1.Tasks[0].Completed {
		t.Error("Job 1 Task 1 should be completed")
	}
	if job1.Tasks[1].Completed {
		t.Error("Job 1 Task 2 should be pending")
	}

	// Check validators
	if len(job1.Validators) != 2 {
		t.Errorf("expected 2 validators in job 1, got %d", len(job1.Validators))
	}
}

// TestParsePlan_EmptyContent tests parsing empty content.
func TestParsePlan_EmptyContent(t *testing.T) {
	plan, err := ParsePlan("")
	if err != nil {
		t.Fatalf("ParsePlan() error = %v", err)
	}

	if plan.Name != "" {
		t.Errorf("plan.Name should be empty, got %q", plan.Name)
	}

	if len(plan.Jobs) != 0 {
		t.Errorf("expected 0 jobs for empty content, got %d", len(plan.Jobs))
	}
}

// TestParsePlan_NoJobs tests parsing content without jobs.
func TestParsePlan_NoJobs(t *testing.T) {
	content := `# Plan: SimpleModule

## 模块概述

**模块职责**: 简单模块

**依赖模块**: 无
`

	plan, err := ParsePlan(content)
	if err != nil {
		t.Fatalf("ParsePlan() error = %v", err)
	}

	if plan.Name != "SimpleModule" {
		t.Errorf("plan.Name = %q, want %q", plan.Name, "SimpleModule")
	}

	if len(plan.Jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(plan.Jobs))
	}
}

// TestExtractJobs_MultipleSections tests extracting jobs from multiple sections.
func TestExtractJobs_MultipleSections(t *testing.T) {
	content := `# Title

## Section 1
Content 1

### Job 1: First Job
Job 1 content

### Job 2: Second Job
Job 2 content

## Section 2
Content 2

### Job 3: Third Job
Job 3 content
`

	// Parse and extract sections
	parser := NewParser()
	doc, _ := parser.mdParser.ParseDocument(content)
	sections, _ := markdown.ExtractSections(doc)

	jobs := extractJobsFromAllSections(sections)

	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}

	// Verify job indices
	for i, job := range jobs {
		if job.Index != i+1 {
			t.Errorf("job[%d].Index = %d, want %d", i, job.Index, i+1)
		}
	}
}

// Helper types and functions for testing

type mockSection struct {
	Title    string
	Level    int
	Content  string
	Children []mockSection
}

func toMarkdownSection(m mockSection) markdown.Section {
	sec := markdown.Section{
		Title:   m.Title,
		Level:   m.Level,
		Content: m.Content,
	}
	for _, child := range m.Children {
		sec.Children = append(sec.Children, toMarkdownSection(child))
	}
	return sec
}

// BenchmarkParsePlan benchmarks plan parsing.
func BenchmarkParsePlan(b *testing.B) {
	content := `# Plan: BenchmarkModule

## 模块概述

**模块职责**: 测试职责

**依赖模块**:
- mod1
- mod2

## Jobs

### Job 1: Job One
**目标**: 目标1

**Tasks (Todo 列表)**:
- [x] Task 1: 完成任务1
- [ ] Task 2: 未完成任务2

**验证器**:
- [ ] 验证器1

### Job 2: Job Two
**目标**: 目标2

**Tasks (Todo 列表)**:
- [ ] Task 1: 任务1
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParsePlan(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestParsePlan_JobNameVariations tests various job name formats.
func TestParsePlan_JobNameVariations(t *testing.T) {
	tests := []struct {
		title         string
		expectedIndex int
		expectedName  string
	}{
		{"Job 1: Simple Name", 1, "Simple Name"},
		{"Job 5: Name With Spaces", 5, "Name With Spaces"},
		{"Job 10: Numbered Job", 10, "Numbered Job"},
		{"job 3: lowercase", 3, "lowercase"},
		{"JOB 7: UPPERCASE", 7, "UPPERCASE"},
		{"Job 1：Chinese Colon", 1, "Chinese Colon"},
	}

	for _, tc := range tests {
		sec := mockSection{
			Title:   tc.title,
			Level:   3,
			Content: "**目标**: test\n\n**Tasks (Todo 列表)**:",
		}

		job := extractJobFromSection(toMarkdownSection(sec))
		if job == nil {
			t.Errorf("%q: expected job, got nil", tc.title)
			continue
		}

		if job.Index != tc.expectedIndex {
			t.Errorf("%q: job.Index = %d, want %d", tc.title, job.Index, tc.expectedIndex)
		}

		if job.Name != tc.expectedName {
			t.Errorf("%q: job.Name = %q, want %q", tc.title, job.Name, tc.expectedName)
		}
	}
}

// TestExtractListField_EmptyContent tests list field extraction with empty content.
func TestExtractListField_EmptyContent(t *testing.T) {
	content := `**依赖模块**: 无

**被依赖模块**:
- module1
`

	// Test "无" (none) value
	result := extractListField(content, "依赖模块")
	if len(result) != 0 {
		t.Errorf("expected empty list for '无', got %v", result)
	}

	// Test normal list
	result = extractListField(content, "被依赖模块")
	if len(result) != 1 || result[0] != "module1" {
		t.Errorf("expected [module1], got %v", result)
	}
}

// TestExtractDebugLogs_IncompleteFields tests debug log with incomplete fields.
func TestExtractDebugLogs_IncompleteFields(t *testing.T) {
	content := `**调试日志**:
- debug1: 只有现象

**其他**: value`

	logs := extractDebugLogs(content)

	if len(logs) != 1 {
		t.Fatalf("expected 1 debug log, got %d", len(logs))
	}

	if logs[0].Phenomenon != "只有现象" {
		t.Errorf("Phenomenon = %q, want %q", logs[0].Phenomenon, "只有现象")
	}
}

// TestParsePlan_ComplexRealWorld tests with a complex real-world example.
func TestParsePlan_ComplexRealWorld(t *testing.T) {
	content := `# Plan: ComplexModule

## 模块概述

**模块职责**: 实现复杂的业务逻辑处理，包括数据验证、转换和存储。

**对应 Research**:
- architecture-design.md
- api-specification.md
- performance-requirements.md

**依赖模块**:
- config
- database
- logger
- cache

**被依赖模块**:
- api_gateway
- worker_service

## 架构设计

[ASCII diagram]

## 接口定义

[Interface definitions]

## Jobs (Loop 块列表)

---

### Job 1: 基础框架实现

**目标**: 搭建模块基础框架，实现核心接口

**前置条件**:
- config 模块已完成
- logger 模块已完成

**Tasks (Todo 列表)**:
- [x] Task 1: 创建模块目录结构
- [x] Task 2: 定义核心接口
- [x] Task 3: 实现配置加载
- [ ] Task 4: 实现依赖注入
- [ ] Task 5: 编写单元测试

**验证器**:
- [x] 接口定义完整
- [x] 配置加载正确
- [ ] 单元测试覆盖率 >= 80%
- [ ] 性能基准测试通过

**调试日志**:
- debug1: 配置加载失败, 缺少配置文件, 配置路径错误, 检查路径配置, 添加默认配置, 已修复
- explore1: 调研依赖注入框架, 使用 wire 和 dig 比较, wire 代码生成 dig 运行时, 选择 wire, 已决定

---

### Job 2: 核心业务逻辑

**目标**: 实现核心业务处理逻辑

**前置条件**:
- Job 1 完成
- database 模块已完成

**Tasks (Todo 列表)**:
- [x] Task 1: 实现数据验证器
- [ ] Task 2: 实现数据转换器
- [ ] Task 3: 实现存储逻辑

**验证器**:
- [ ] 数据验证正确
- [ ] 转换逻辑正确
- [ ] 存储操作正确

**调试日志**:
- 无
`

	plan, err := ParsePlan(content)
	if err != nil {
		t.Fatalf("ParsePlan() error = %v", err)
	}

	// Verify module overview
	if plan.Name != "ComplexModule" {
		t.Errorf("plan.Name = %q, want %q", plan.Name, "ComplexModule")
	}

	if len(plan.Research) != 3 {
		t.Errorf("expected 3 research items, got %d", len(plan.Research))
	}

	if len(plan.Dependencies) != 4 {
		t.Errorf("expected 4 dependencies, got %d", len(plan.Dependencies))
	}

	if len(plan.Dependents) != 2 {
		t.Errorf("expected 2 dependents, got %d", len(plan.Dependents))
	}

	// Verify jobs
	if len(plan.Jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(plan.Jobs))
	}

	// Job 1 details
	job1 := plan.Jobs[0]
	if job1.Index != 1 {
		t.Errorf("job1.Index = %d, want %d", job1.Index, 1)
	}

	if len(job1.Tasks) != 5 {
		t.Errorf("expected 5 tasks in job 1, got %d", len(job1.Tasks))
	}

	if len(job1.Validators) != 4 {
		t.Errorf("expected 4 validators in job 1, got %d", len(job1.Validators))
	}

	if len(job1.DebugLogs) != 2 {
		t.Errorf("expected 2 debug logs in job 1, got %d", len(job1.DebugLogs))
	}

	// Check task completion
	completed := 0
	for _, task := range job1.Tasks {
		if task.Completed {
			completed++
		}
	}
	if completed != 3 {
		t.Errorf("expected 3 completed tasks, got %d", completed)
	}

	// Job 2 details
	job2 := plan.Jobs[1]
	if job2.Index != 2 {
		t.Errorf("job2.Index = %d, want %d", job2.Index, 2)
	}

	if len(job2.DebugLogs) != 0 {
		t.Errorf("expected 0 debug logs in job 2 (marked '无'), got %d", len(job2.DebugLogs))
	}
}

// TestParsePlan_NonJobSections tests that non-Job sections are not parsed as jobs.
func TestParsePlan_NonJobSections(t *testing.T) {
	content := `# Plan: TestModule

## 模块概述
模块描述

### Regular Section
Regular content

### Job 1: Actual Job
Job content

## Another Section
More content
`

	plan, err := ParsePlan(content)
	if err != nil {
		t.Fatalf("ParsePlan() error = %v", err)
	}

	// Should only have 1 job (Job 1), not "Regular Section"
	if len(plan.Jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(plan.Jobs))
	}

	if len(plan.Jobs) > 0 && plan.Jobs[0].Name != "Actual Job" {
		t.Errorf("job.Name = %q, want %q", plan.Jobs[0].Name, "Actual Job")
	}
}

// TestParsePlan_TaskDescriptionVariations tests various task description formats.
func TestParsePlan_TaskDescriptionVariations(t *testing.T) {
	content := `### Job 1: Test Job

**Tasks (Todo 列表)**:
- [ ] Task 1: Simple description
- [x] Task 2: Description with: colons
- [ ] Task 3: Description with special chars: *bold* _italic_
- [x] Task 4: Description with code: ` + "`code`" + `
- [ ] Task 5: Description with [link](url)
`

	parser := NewParser()
	doc, _ := parser.mdParser.ParseDocument(content)
	sections, _ := markdown.ExtractSections(doc)

	if len(sections) == 0 {
		t.Fatal("no sections found")
	}

	job := extractJobFromSection(sections[0])
	if job == nil {
		t.Fatal("job should not be nil")
	}

	if len(job.Tasks) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(job.Tasks))
	}

	// Check specific descriptions
	expectedDescriptions := []string{
		"Simple description",
		"Description with: colons",
		"Description with special chars: *bold* _italic_",
		"Description with code: `code`",
		"Description with [link](url)",
	}

	for i, expected := range expectedDescriptions {
		if i < len(job.Tasks) {
			// Normalize whitespace for comparison
			got := strings.TrimSpace(job.Tasks[i].Description)
			expectedTrimmed := strings.TrimSpace(expected)
			if !strings.Contains(got, expectedTrimmed) && !strings.Contains(expectedTrimmed, got) {
				t.Errorf("task[%d].Description = %q, want %q", i, got, expectedTrimmed)
			}
		}
	}
}

// TestNewParser tests the parser constructor.
func TestNewParser(t *testing.T) {
	parser := NewParser()
	if parser == nil {
		t.Error("NewParser() returned nil")
	}
	if parser.mdParser == nil {
		t.Error("parser.mdParser is nil")
	}
}
