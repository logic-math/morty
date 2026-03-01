// Package executor provides job execution engine for Morty.
package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/morty/morty/internal/parser/plan"
	"github.com/morty/morty/internal/state"
)

// PromptBuilder defines the interface for building prompts for AI task execution.
type PromptBuilder interface {
	// BuildPrompt constructs a complete prompt for a specific task.
	// It combines the system prompt, plan content, compact context, and task details.
	BuildPrompt(module, job string, taskIndex int, taskDesc string) (string, error)

	// BuildCompactContext creates a compact context with current job info and completed jobs summary.
	// This helps keep the context window efficient by excluding full historical details.
	BuildCompactContext(module, job string) (map[string]interface{}, error)
}

// promptBuilder implements the PromptBuilder interface.
type promptBuilder struct {
	stateManager      *state.Manager
	planDir           string
	promptsDir        string
	systemPromptFile  string
}

// PromptBuilderConfig holds configuration for creating a PromptBuilder.
type PromptBuilderConfig struct {
	// PlanDir is the directory containing plan files (default: ".morty/plan")
	PlanDir string
	// PromptsDir is the directory containing prompt files (default: "prompts")
	PromptsDir string
	// SystemPromptFile is the name of the system prompt file (default: "doing.md")
	SystemPromptFile string
}

// DefaultPromptBuilderConfig returns the default configuration.
func DefaultPromptBuilderConfig() *PromptBuilderConfig {
	return &PromptBuilderConfig{
		PlanDir:          ".morty/plan",
		PromptsDir:       "prompts",
		SystemPromptFile: "doing.md",
	}
}

// NewPromptBuilder creates a new PromptBuilder with the given dependencies.
//
// Parameters:
//   - stateManager: The state manager for accessing job state
//   - config: Optional configuration. If nil, default config is used.
//
// Returns:
//   - A PromptBuilder implementation
func NewPromptBuilder(
	stateManager *state.Manager,
	config *PromptBuilderConfig,
) PromptBuilder {
	if config == nil {
		config = DefaultPromptBuilderConfig()
	}
	return &promptBuilder{
		stateManager:     stateManager,
		planDir:          config.PlanDir,
		promptsDir:       config.PromptsDir,
		systemPromptFile: config.SystemPromptFile,
	}
}

// BuildPrompt constructs a complete prompt for a specific task.
// It combines:
// 1. System prompt (from prompts/doing.md)
// 2. Plan file content
// 3. Compact context (current job + completed jobs summary)
// 4. Current task details
// 5. Validator requirements
//
// Parameters:
//   - module: The module name
//   - job: The job name
//   - taskIndex: The index of the current task
//   - taskDesc: The description of the current task
//
// Returns:
//   - The complete prompt string
//   - An error if any step fails
func (pb *promptBuilder) BuildPrompt(module, job string, taskIndex int, taskDesc string) (string, error) {
	var parts []string

	// Step 1: Add system prompt
	systemPrompt, err := pb.loadSystemPrompt()
	if err != nil {
		return "", fmt.Errorf("failed to load system prompt: %w", err)
	}
	parts = append(parts, systemPrompt)

	// Step 2: Build and add compact context
	compactContext, err := pb.BuildCompactContext(module, job)
	if err != nil {
		return "", fmt.Errorf("failed to build compact context: %w", err)
	}

	contextJSON, err := json.MarshalIndent(compactContext, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal compact context: %w", err)
	}

	parts = append(parts, "---\n\n# 精简上下文\n\n```json\n"+string(contextJSON)+"\n```\n")

	// Step 3: Add Plan content
	planContent, err := pb.loadPlanContent(module)
	if err != nil {
		return "", fmt.Errorf("failed to load plan content: %w", err)
	}
	parts = append(parts, "---\n\n# Plan 内容\n\n"+planContent)

	// Step 4: Add current task context
	taskContext, err := pb.buildTaskContext(module, job, taskIndex, taskDesc)
	if err != nil {
		return "", fmt.Errorf("failed to build task context: %w", err)
	}
	parts = append(parts, taskContext)

	// Step 5: Add execution instructions
	parts = append(parts, pb.buildExecutionInstructions())

	return strings.Join(parts, "\n\n"), nil
}

// BuildCompactContext creates a compact context for efficient token usage.
// It includes:
// - Current job info (module, job, status, loop_count)
// - Completed jobs summary (brief descriptions)
// - Current job details (name, description, tasks, dependencies, validator)
//
// Parameters:
//   - module: The module name
//   - job: The job name
//
// Returns:
//   - A map containing the compact context
//   - An error if state cannot be loaded
func (pb *promptBuilder) BuildCompactContext(module, job string) (map[string]interface{}, error) {
	// Load state
	if err := pb.stateManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Get job state
	jobState := pb.stateManager.GetJob(module, job)
	if jobState == nil {
		return nil, fmt.Errorf("job not found: %s in module %s", job, module)
	}

	// Get loop count from job state
	loopCount := jobState.LoopCount
	if loopCount == 0 {
		loopCount = 1
	}

	// Build current section
	current := map[string]interface{}{
		"module":     module,
		"job":        job,
		"status":     string(jobState.Status),
		"loop_count": loopCount,
	}

	// Build completed jobs summary
	completedSummary := pb.buildCompletedJobsSummary(module)

	// Build current job details
	currentJobDetails, err := pb.buildCurrentJobDetails(module, job, jobState)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"current": current,
		"context": map[string]interface{}{
			"completed_jobs_summary": completedSummary,
			"current_job":            currentJobDetails,
		},
	}, nil
}

// loadSystemPrompt reads the system prompt from the prompts directory.
func (pb *promptBuilder) loadSystemPrompt() (string, error) {
	promptPath := filepath.Join(pb.promptsDir, pb.systemPromptFile)
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read system prompt from %s: %w", promptPath, err)
	}
	return string(content), nil
}

// loadPlanContent reads and returns the Plan file content for the given module.
func (pb *promptBuilder) loadPlanContent(module string) (string, error) {
	planPath := filepath.Join(pb.planDir, module+".md")
	content, err := os.ReadFile(planPath)
	if err != nil {
		return "", fmt.Errorf("failed to read plan file from %s: %w", planPath, err)
	}
	return string(content), nil
}

// buildCompletedJobsSummary creates a summary of completed jobs for the given module.
func (pb *promptBuilder) buildCompletedJobsSummary(module string) []string {
	var summaries []string

	statePtr := pb.stateManager.GetState()
	if statePtr == nil {
		return summaries
	}

	// Find module by name
	var moduleState *state.ModuleState
	for i := range statePtr.Modules {
		if statePtr.Modules[i].Name == module {
			moduleState = &statePtr.Modules[i]
			break
		}
	}

	if moduleState == nil {
		return summaries
	}

	for _, jobState := range moduleState.Jobs {
		if jobState.Status == state.StatusCompleted {
			summary := fmt.Sprintf("%s/%s: 完成 (%d tasks)",
				module, jobState.Name, len(jobState.Tasks))
			summaries = append(summaries, summary)
		}
	}

	return summaries
}

// buildCurrentJobDetails extracts current job details from the plan and state.
func (pb *promptBuilder) buildCurrentJobDetails(module, job string, jobState *state.JobState) (map[string]interface{}, error) {
	// Load plan to get job details
	planContent, err := pb.loadPlanContent(module)
	if err != nil {
		return nil, err
	}

	// Parse plan
	parsedPlan, err := plan.ParsePlan(planContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	// Find the job
	var planJob *plan.Job
	for i := range parsedPlan.Jobs {
		if strings.EqualFold(parsedPlan.Jobs[i].Name, job) ||
			fmt.Sprintf("job_%d", parsedPlan.Jobs[i].Index) == job {
			planJob = &parsedPlan.Jobs[i]
			break
		}
	}

	if planJob == nil {
		// Fallback: build from state only
		return pb.buildJobDetailsFromState(job, jobState), nil
	}

	// Build tasks list
	var tasks []string
	for _, task := range planJob.Tasks {
		status := "pending"
		if task.Completed {
			status = "completed"
		} else if task.Index < jobState.TasksCompleted {
			status = "completed"
		}
		tasks = append(tasks, fmt.Sprintf("Task %d: %s (%s)",
			task.Index, task.Description, status))
	}

	// If plan parsing didn't find tasks but state has them
	if len(tasks) == 0 && len(jobState.Tasks) > 0 {
		for _, task := range jobState.Tasks {
			status := string(task.Status)
			tasks = append(tasks, fmt.Sprintf("Task %d: %s (%s)",
				task.Index, task.Description, status))
		}
	}

	return map[string]interface{}{
		"name":         job,
		"description":  planJob.Goal,
		"tasks":        tasks,
		"dependencies": planJob.Prerequisites,
		"validator":    strings.Join(planJob.Validators, ", "),
	}, nil
}

// buildJobDetailsFromState builds job details from state when plan parsing fails.
func (pb *promptBuilder) buildJobDetailsFromState(job string, jobState *state.JobState) map[string]interface{} {
	var tasks []string
	for _, task := range jobState.Tasks {
		tasks = append(tasks, fmt.Sprintf("Task %d: %s (%s)",
			task.Index, task.Description, task.Status))
	}

	return map[string]interface{}{
		"name":         job,
		"description":  "",
		"tasks":        tasks,
		"dependencies": []string{},
		"validator":    "",
	}
}

// buildTaskContext builds the context for the current task.
func (pb *promptBuilder) buildTaskContext(module, job string, taskIndex int, taskDesc string) (string, error) {
	var builder strings.Builder

	builder.WriteString("---\n\n# 当前 Job 上下文\n\n")
	builder.WriteString(fmt.Sprintf("**模块**: %s\n", module))
	builder.WriteString(fmt.Sprintf("**Job**: %s\n", job))

	// Get job state for total tasks
	jobState := pb.stateManager.GetJob(module, job)
	if jobState != nil {
		builder.WriteString(fmt.Sprintf("**总 Tasks**: %d\n", len(jobState.Tasks)))
	}

	builder.WriteString("\n## 任务列表\n\n")
	builder.WriteString("你需要按顺序完成以下所有 tasks：\n\n")

	// List all tasks
	if jobState != nil {
		for i, task := range jobState.Tasks {
			status := "[ ]"
			if task.Status == state.StatusCompleted {
				status = "[x]"
			}
			builder.WriteString(fmt.Sprintf("- %s Task %d: %s\n",
				status, i, task.Description))
		}
	}

	builder.WriteString("\n## 验证器\n\n")

	// Add validators from plan
	planContent, err := pb.loadPlanContent(module)
	if err == nil {
		parsedPlan, err := plan.ParsePlan(planContent)
		if err == nil {
			for _, planJob := range parsedPlan.Jobs {
				if strings.EqualFold(planJob.Name, job) ||
					fmt.Sprintf("job_%d", planJob.Index) == job {
					for _, validator := range planJob.Validators {
						if strings.TrimSpace(validator) != "" {
							builder.WriteString(fmt.Sprintf("- %s\n", validator))
						}
					}
					break
				}
			}
		}
	}

	builder.WriteString("\n## 执行指令\n\n")
	builder.WriteString("请按照 Doing 模式的循环步骤执行：\n")
	builder.WriteString("1. 读取精简上下文了解当前状态\n")
	builder.WriteString("2. **按顺序执行所有 Tasks**，完成一个后再进行下一个\n")
	builder.WriteString("3. 每个 Task 完成后在内部标记进度\n")
	builder.WriteString("4. 所有 Tasks 完成后，运行所有验证器检查\n")
	builder.WriteString("5. 如有问题，记录 debug_log\n")

	return builder.String(), nil
}

// buildExecutionInstructions builds the final execution instructions.
func (pb *promptBuilder) buildExecutionInstructions() string {
	return `---

# 任务完成要求（必须执行）

**所有 Tasks 执行完毕后**，你必须在输出中返回 JSON 格式的执行结果（RALPH_STATUS）：

` + "```json\n" + `{
  "module": "[模块名]",
  "job": "[Job 名]",
  "status": "COMPLETED",
  "tasks_completed": 8,
  "tasks_total": 8,
  "summary": "执行摘要"
}` + "\n```\n\n" + `### 重要规则：
- **成功时**: status 必须是 "COMPLETED"（全部大写）
- **失败时**: status 可以是 "FAILED"
- 系统会检查输出内容中是否包含 "status": "COMPLETED" 来判断任务是否成功
- **不需要写入任何文件**，只需要在输出中包含上述 JSON

### 验证器自检清单
在输出结果前，请确认：
- [ ] 我已执行完当前 Job 的所有 Tasks
- [ ] 我已运行所有验证器检查
- [ ] 验证器全部通过（或在失败情况下明确记录原因）
- [ ] 我已输出 RALPH_STATUS JSON 且 status 为 "COMPLETED"

**注意**: 系统通过检测输出中的 "status": "COMPLETED" 来判断任务成功，未检测到则标记为失败。

开始执行!`
}

// ReplaceTemplateVariables replaces template variables in the prompt.
// Supported variables:
// - {{module}}: Module name
// - {{job}}: Job name
// - {{task_index}}: Current task index
// - {{task_desc}}: Current task description
// - {{plan_dir}}: Plan directory path
// - {{prompts_dir}}: Prompts directory path
func (pb *promptBuilder) ReplaceTemplateVariables(prompt, module, job string, taskIndex int, taskDesc string) string {
	replacements := map[string]string{
		"{{module}}":      module,
		"{{job}}":         job,
		"{{task_index}}":  fmt.Sprintf("%d", taskIndex),
		"{{task_desc}}":   taskDesc,
		"{{plan_dir}}":    pb.planDir,
		"{{prompts_dir}}": pb.promptsDir,
	}

	result := prompt
	for key, value := range replacements {
		result = strings.ReplaceAll(result, key, value)
	}

	return result
}

// ReplaceTemplateVariablesRegex replaces template variables using regex.
// This supports variables with optional spaces: {{ module }} or {{module}}
func (pb *promptBuilder) ReplaceTemplateVariablesRegex(prompt, module, job string, taskIndex int, taskDesc string) string {
	replacements := map[string]string{
		"module":      module,
		"job":         job,
		"task_index":  fmt.Sprintf("%d", taskIndex),
		"task_desc":   taskDesc,
		"plan_dir":    pb.planDir,
		"prompts_dir": pb.promptsDir,
	}

	// Regex to match {{variable}} or {{ variable }}
	re := regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_-]+)\s*\}\}`)

	return re.ReplaceAllStringFunc(prompt, func(match string) string {
		// Extract variable name
		matches := re.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}
		varName := matches[1]
		if value, ok := replacements[varName]; ok {
			return value
		}
		return match
	})
}

// Ensure promptBuilder implements PromptBuilder interface
var _ PromptBuilder = (*promptBuilder)(nil)
