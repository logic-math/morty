// Package validator provides plan file format validation.
package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/morty/morty/internal/parser/plan"
)

// ValidationError represents a format validation error.
type ValidationError struct {
	Code     string // Error code (E001-E012)
	File     string // File path
	Line     int    // Line number (0 if not applicable)
	Message  string // Error message
	Found    string // What was found
	Expected string // What was expected
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: %s [%s]", e.File, e.Line, e.Message, e.Code)
	}
	return fmt.Sprintf("%s: %s [%s]", e.File, e.Message, e.Code)
}

// ValidationResult represents the result of validation.
type ValidationResult struct {
	File   string
	Passed bool
	Errors []*ValidationError
}

// PlanValidator validates plan files against the format specification.
type PlanValidator struct {
	planDir string
	verbose bool
}

// NewPlanValidator creates a new plan validator.
func NewPlanValidator(planDir string, verbose bool) *PlanValidator {
	return &PlanValidator{
		planDir: planDir,
		verbose: verbose,
	}
}

// ValidateAll validates all plan files in the plan directory.
func (v *PlanValidator) ValidateAll() ([]*ValidationResult, error) {
	// Find all .md files
	files, err := filepath.Glob(filepath.Join(v.planDir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to list plan files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no plan files found in %s", v.planDir)
	}

	// Check if e2e_test.md exists
	e2eTestExists := false
	for _, file := range files {
		if filepath.Base(file) == "e2e_test.md" {
			e2eTestExists = true
			break
		}
	}

	results := make([]*ValidationResult, 0, len(files))

	// If e2e_test.md is missing, add a validation error
	if !e2eTestExists {
		results = append(results, &ValidationResult{
			File:   filepath.Join(v.planDir, "e2e_test.md"),
			Passed: false,
			Errors: []*ValidationError{
				{
					Code:     "E003",
					File:     filepath.Join(v.planDir, "e2e_test.md"),
					Message:  "缺少必需的 e2e_test.md 文件",
					Expected: "每个 plan 目录必须包含 e2e_test.md 作为端到端测试模块",
				},
			},
		})
	}

	for _, file := range files {
		result := v.ValidateFile(file)
		results = append(results, result)
	}

	return results, nil
}

// ValidateFile validates a single plan file.
func (v *PlanValidator) ValidateFile(filePath string) *ValidationResult {
	result := &ValidationResult{
		File:   filePath,
		Passed: true,
		Errors: make([]*ValidationError, 0),
	}

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, &ValidationError{
			Code:    "E000",
			File:    filePath,
			Message: fmt.Sprintf("failed to read file: %v", err),
		})
		return result
	}

	fileName := filepath.Base(filePath)

	// Validate filename
	if err := v.validateFilename(fileName); err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, &ValidationError{
			Code:     "E001",
			File:     filePath,
			Message:  "文件名不符合规范",
			Found:    fileName,
			Expected: "小写字母、数字、下划线组成的文件名",
		})
	}

	// Skip README.md from content validation
	if fileName == "README.md" {
		return v.validateREADME(filePath, string(content), result)
	}

	// Parse plan file
	planData, err := plan.ParsePlan(string(content))
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, &ValidationError{
			Code:    "E000",
			File:    filePath,
			Message: fmt.Sprintf("failed to parse plan file: %v", err),
		})
		return result
	}

	// Validate structure
	v.validatePlanStructure(filePath, planData, string(content), result)

	return result
}

// validateFilename validates the file name format.
func (v *PlanValidator) validateFilename(fileName string) error {
	// Special cases: e2e_test.md and README.md
	if fileName == "e2e_test.md" || fileName == "README.md" {
		return nil
	}

	// Regular files: lowercase, numbers, underscores only
	matched, _ := regexp.MatchString(`^[a-z0-9_]+\.md$`, fileName)
	if !matched {
		return fmt.Errorf("invalid filename format")
	}

	return nil
}

// validatePlanStructure validates the plan file structure.
func (v *PlanValidator) validatePlanStructure(filePath string, planData *plan.Plan, content string, result *ValidationResult) {
	lines := strings.Split(content, "\n")

	// Validate module overview
	v.validateModuleOverview(filePath, planData, lines, result)

	// Validate jobs
	v.validateJobs(filePath, planData, lines, result)
}

// validateModuleOverview validates the module overview section.
func (v *PlanValidator) validateModuleOverview(filePath string, planData *plan.Plan, lines []string, result *ValidationResult) {
	// Check required fields
	if planData.Responsibility == "" {
		result.Passed = false
		result.Errors = append(result.Errors, &ValidationError{
			Code:     "E002",
			File:     filePath,
			Message:  "缺少必需字段: 模块职责",
			Expected: "**模块职责**: [描述]",
		})
	}

	// Check dependencies format
	if !v.isValidDependencyFormat(planData.Dependencies) {
		result.Passed = false
		depStr := strings.Join(planData.Dependencies, ", ")
		if depStr == "" {
			depStr = "(空)"
		}
		result.Errors = append(result.Errors, &ValidationError{
			Code:     "E005",
			File:     filePath,
			Message:  "依赖模块格式错误",
			Found:    depStr,
			Expected: "无 或 module1, module2 或 __ALL__",
		})
	}

	// Validate module name format in dependencies
	for _, dep := range planData.Dependencies {
		if dep != "无" && dep != "__ALL__" && !v.isValidModuleName(dep) {
			result.Passed = false
			result.Errors = append(result.Errors, &ValidationError{
				Code:     "E005",
				File:     filePath,
				Message:  "依赖模块名称格式错误",
				Found:    dep,
				Expected: "小写字母、数字、下划线组成",
			})
		}
	}
}

// validateJobs validates all jobs in the plan.
func (v *PlanValidator) validateJobs(filePath string, planData *plan.Plan, lines []string, result *ValidationResult) {
	if len(planData.Jobs) == 0 {
		result.Passed = false
		result.Errors = append(result.Errors, &ValidationError{
			Code:    "E002",
			File:    filePath,
			Message: "模块中没有定义任何 Job",
		})
		return
	}

	// Check job numbering
	for i, job := range planData.Jobs {
		expectedIndex := i + 1
		if job.Index != expectedIndex {
			lineNum := v.findJobLine(lines, job.Name)
			result.Passed = false
			result.Errors = append(result.Errors, &ValidationError{
				Code:     "E004",
				File:     filePath,
				Line:     lineNum,
				Message:  "Job 编号不连续",
				Found:    fmt.Sprintf("Job %d", job.Index),
				Expected: fmt.Sprintf("Job %d", expectedIndex),
			})
		}

		// Validate job structure
		v.validateJob(filePath, &job, lines, result)
	}
}

// validateJob validates a single job.
func (v *PlanValidator) validateJob(filePath string, job *plan.Job, lines []string, result *ValidationResult) {
	lineNum := v.findJobLine(lines, job.Name)

	// Validate goal
	if job.Goal == "" {
		result.Passed = false
		result.Errors = append(result.Errors, &ValidationError{
			Code:    "E002",
			File:    filePath,
			Line:    lineNum,
			Message: fmt.Sprintf("Job %d 缺少目标描述", job.Index),
		})
	}

	// Validate tasks format
	for i, task := range job.Tasks {
		expectedIndex := i + 1
		if task.Index != expectedIndex {
			result.Passed = false
			result.Errors = append(result.Errors, &ValidationError{
				Code:     "E006",
				File:     filePath,
				Line:     lineNum,
				Message:  fmt.Sprintf("Job %d Task 编号不连续", job.Index),
				Found:    fmt.Sprintf("Task %d", task.Index),
				Expected: fmt.Sprintf("Task %d", expectedIndex),
			})
		}

		// Validate task description
		if task.Description == "" {
			result.Passed = false
			result.Errors = append(result.Errors, &ValidationError{
				Code:    "E006",
				File:    filePath,
				Line:    lineNum,
				Message: fmt.Sprintf("Job %d Task %d 缺少描述", job.Index, task.Index),
			})
		}
	}

	// Validate prerequisites format
	for _, prereq := range job.Prerequisites {
		if !v.isValidPrerequisiteFormat(prereq) {
			result.Passed = false
			result.Errors = append(result.Errors, &ValidationError{
				Code:     "E007",
				File:     filePath,
				Line:     lineNum,
				Message:  fmt.Sprintf("Job %d 前置条件格式错误", job.Index),
				Found:    prereq,
				Expected: "job_N 或 module:job_N 或 job_N - 描述",
			})
		}
	}

	// Validate validators
	if len(job.Validators) == 0 {
		result.Passed = false
		result.Errors = append(result.Errors, &ValidationError{
			Code:    "E002",
			File:    filePath,
			Line:    lineNum,
			Message: fmt.Sprintf("Job %d 缺少验证器", job.Index),
		})
	}

	// Validate completion status
	if job.CompletionStatus == "" {
		result.Passed = false
		result.Errors = append(result.Errors, &ValidationError{
			Code:    "E002",
			File:    filePath,
			Line:    lineNum,
			Message: fmt.Sprintf("Job %d 缺少完成状态标记", job.Index),
		})
	} else if !v.isValidCompletionStatus(job.CompletionStatus) {
		result.Passed = false
		result.Errors = append(result.Errors, &ValidationError{
			Code:     "E008",
			File:     filePath,
			Line:     lineNum,
			Message:  fmt.Sprintf("Job %d 完成状态标记无效", job.Index),
			Found:    job.CompletionStatus,
			Expected: "✅ 已完成 | 🚧 进行中 | ⏸️ 暂停 | ❌ 失败 | ⏳ 待开始",
		})
	}

	// Validate debug logs format
	for _, log := range job.DebugLogs {
		if !v.isValidDebugLogFormat(log) {
			result.Passed = false
			result.Errors = append(result.Errors, &ValidationError{
				Code:     "E009",
				File:     filePath,
				Line:     lineNum,
				Message:  fmt.Sprintf("Job %d 调试日志格式错误", job.Index),
				Found:    log.ID,
				Expected: "debug日志应包含6个字段: 现象, 复现, 猜想, 验证, 修复, 进展",
			})
		}
	}
}

// validateREADME validates the README.md file.
func (v *PlanValidator) validateREADME(filePath, content string, result *ValidationResult) *ValidationResult {
	lines := strings.Split(content, "\n")

	// Check required sections
	requiredSections := []string{
		"# Plan 索引",
		"## 模块列表",
		"## 依赖关系图",
		"## 执行顺序",
		"## 统计信息",
	}

	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			result.Passed = false
			result.Errors = append(result.Errors, &ValidationError{
				Code:     "E002",
				File:     filePath,
				Message:  "README 缺少必需 section",
				Expected: section,
			})
		}
	}

	// Validate module list table
	v.validateModuleTable(filePath, lines, result)

	return result
}

// validateModuleTable validates the module list table in README.
func (v *PlanValidator) validateModuleTable(filePath string, lines []string, result *ValidationResult) {
	// Find table
	inTable := false
	headerFound := false
	rowCount := 0

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Check for table header
		if strings.Contains(line, "模块名称") && strings.Contains(line, "依赖模块") {
			headerFound = true
			inTable = true
			continue
		}

		// Skip separator line
		if inTable && strings.HasPrefix(line, "|---") {
			continue
		}

		// Parse table row
		if inTable && strings.HasPrefix(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) < 6 {
				result.Passed = false
				result.Errors = append(result.Errors, &ValidationError{
					Code:    "E010",
					File:    filePath,
					Line:    i + 1,
					Message: "模块列表表格列数不正确",
					Found:   fmt.Sprintf("%d 列", len(parts)-2),
					Expected: "5 列 (模块名称, 文件, Jobs 数量, 依赖模块, 状态)",
				})
			}
			rowCount++
		}

		// Stop at next section
		if inTable && line != "" && !strings.HasPrefix(line, "|") {
			break
		}
	}

	if !headerFound {
		result.Passed = false
		result.Errors = append(result.Errors, &ValidationError{
			Code:    "E010",
			File:    filePath,
			Message: "README 中未找到模块列表表格",
		})
	}
}

// Helper functions

func (v *PlanValidator) isValidDependencyFormat(deps []string) bool {
	if len(deps) == 0 {
		return true
	}

	// Check if it's "无" or "__ALL__"
	if len(deps) == 1 && (deps[0] == "无" || deps[0] == "__ALL__") {
		return true
	}

	// Check if all dependencies are valid module names
	for _, dep := range deps {
		if !v.isValidModuleName(dep) {
			return false
		}
	}

	return true
}

func (v *PlanValidator) isValidModuleName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-z0-9_]+$`, name)
	return matched
}

func (v *PlanValidator) isValidPrerequisiteFormat(prereq string) bool {
	// Format: job_N or module:job_N or job_N - description
	matched, _ := regexp.MatchString(`^(job_\d+|[a-z0-9_]+:job_\d+)(\s+-\s+.+)?$`, prereq)
	if matched {
		return true
	}

	// Allow natural language prerequisites (not job references)
	// If it doesn't start with "job_" or contain ":", it's a natural language prereq
	if !strings.HasPrefix(prereq, "job_") && !strings.Contains(prereq, ":job_") {
		return true
	}

	return false
}

func (v *PlanValidator) isValidCompletionStatus(status string) bool {
	validStatuses := []string{
		"✅ 已完成",
		"🚧 进行中",
		"⏸️ 暂停",
		"❌ 失败",
		"⏳ 待开始",
		"无", // Allow "无" for backward compatibility
	}

	for _, valid := range validStatuses {
		if strings.HasPrefix(status, valid) {
			return true
		}
	}

	return false
}

func (v *PlanValidator) isValidDebugLogFormat(log plan.DebugLog) bool {
	// Check that all required fields are non-empty
	return log.Phenomenon != "" &&
		log.Hypothesis != "" &&
		log.Fix != "" &&
		log.Progress != ""
}

func (v *PlanValidator) findJobLine(lines []string, jobName string) int {
	for i, line := range lines {
		if strings.Contains(line, "### Job") && strings.Contains(line, jobName) {
			return i + 1
		}
	}
	return 0
}

// FormatResults formats validation results for display.
func FormatResults(results []*ValidationResult, verbose bool) string {
	var sb strings.Builder

	totalErrors := 0
	passedCount := 0

	for _, result := range results {
		if result.Passed {
			passedCount++
		}
		totalErrors += len(result.Errors)
	}

	if totalErrors == 0 {
		sb.WriteString("✅ Plan 格式验证通过\n\n")
		sb.WriteString(fmt.Sprintf("检查的文件: %d\n", len(results)))
		for _, result := range results {
			fileName := filepath.Base(result.File)
			sb.WriteString(fmt.Sprintf("  - %s: ✅ 通过\n", fileName))
		}
		return sb.String()
	}

	sb.WriteString("❌ Plan 格式验证失败\n\n")

	for _, result := range results {
		if len(result.Errors) == 0 {
			continue
		}

		fileName := filepath.Base(result.File)
		sb.WriteString(fmt.Sprintf("%s:\n", fileName))

		for _, err := range result.Errors {
			if err.Line > 0 {
				sb.WriteString(fmt.Sprintf("  ❌ %s: %s (第 %d 行)\n", err.Code, err.Message, err.Line))
			} else {
				sb.WriteString(fmt.Sprintf("  ❌ %s: %s\n", err.Code, err.Message))
			}

			if verbose && err.Found != "" {
				sb.WriteString(fmt.Sprintf("     发现: %s\n", err.Found))
			}
			if verbose && err.Expected != "" {
				sb.WriteString(fmt.Sprintf("     期望: %s\n", err.Expected))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("总计: %d 个错误\n", totalErrors))

	return sb.String()
}
