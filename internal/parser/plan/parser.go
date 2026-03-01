// Package plan provides Plan file parsing functionality.
package plan

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/morty/morty/internal/parser/markdown"
)

// Plan represents a parsed Plan document for a module.
type Plan struct {
	Name           string       `json:"name"`            // Module name
	Responsibility string       `json:"responsibility"`  // Module responsibilities
	Research       []string     `json:"research"`        // Related research documents
	Dependencies   []string     `json:"dependencies"`    // Modules this module depends on
	Dependents     []string     `json:"dependents"`      // Modules that depend on this module
	Jobs           []Job        `json:"jobs"`            // List of jobs in the plan
	RawContent     string       `json:"raw_content"`     // Original markdown content
}

// Job represents a single job in a Plan.
type Job struct {
	Name         string       `json:"name"`          // Job name
	Index        int          `json:"index"`         // Job index number
	Goal         string       `json:"goal"`          // Job objective/goal
	Prerequisites []string    `json:"prerequisites"` // Prerequisites for this job
	Tasks        []TaskItem   `json:"tasks"`         // List of tasks
	Validators   []string     `json:"validators"`    // Validation criteria
	DebugLogs    []DebugLog   `json:"debug_logs"`    // Debug log entries
	CompletionStatus string   `json:"completion_status"` // Completion status marker from plan file
	IsCompleted  bool         `json:"is_completed"`  // Whether job is marked as completed in plan
}

// TaskItem represents a single task within a Job.
type TaskItem struct {
	Index       int    `json:"index"`        // Task number/index
	Description string `json:"description"`  // Task description
	Completed   bool   `json:"completed"`    // Whether task is completed
}

// DebugLog represents a debug log entry.
type DebugLog struct {
	ID       string `json:"id"`       // Log ID (debug1, explore1, etc.)
	Phenomenon string `json:"phenomenon"` // Issue description
	Reproduction string `json:"reproduction"` // How to reproduce
	Hypothesis string `json:"hypothesis"` // Possible causes
	Verification string `json:"verification"` // Verification steps
	Fix string `json:"fix"` // Fix method
	Progress string `json:"progress"` // Fix progress
}

// Parser provides functionality to parse Plan markdown files.
type Parser struct {
	mdParser *markdown.Parser
}

// NewParser creates a new Plan parser instance.
func NewParser() *Parser {
	return &Parser{
		mdParser: markdown.NewParser(),
	}
}

// ParsePlan parses Plan content and returns a structured Plan.
func ParsePlan(content string) (*Plan, error) {
	parser := NewParser()
	return parser.parsePlanContent(content)
}

// parsePlanContent parses the plan content into a Plan struct.
func (p *Parser) parsePlanContent(content string) (*Plan, error) {
	// Parse the markdown document
	doc, err := p.mdParser.ParseDocument(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown: %w", err)
	}

	plan := &Plan{
		RawContent: content,
	}

	// Extract sections
	sections, err := markdown.ExtractSections(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to extract sections: %w", err)
	}

	// Find the main title (H1)
	for _, sec := range sections {
		if sec.Level == 1 {
			plan.Name = extractModuleName(sec.Title)
			break
		}
	}

	// Extract module overview
	plan.extractModuleOverview(sections)

	// Extract Jobs - look at all sections recursively
	plan.Jobs = extractJobsFromAllSections(sections)

	return plan, nil
}

// extractModuleName extracts the module name from the title.
// Title format: "# Plan: ModuleName" or "Plan: ModuleName"
func extractModuleName(title string) string {
	title = strings.TrimSpace(title)
	// Remove leading # if present
	title = regexp.MustCompile(`^#+\s*`).ReplaceAllString(title, "")
	// Remove "Plan:" prefix if present
	title = regexp.MustCompile(`(?i)^plan:\s*`).ReplaceAllString(title, "")
	return strings.TrimSpace(title)
}

// extractModuleOverview extracts module overview information.
func (p *Plan) extractModuleOverview(sections []markdown.Section) {
	debug := os.Getenv("MORTY_DEBUG") != ""

	if debug {
		fmt.Fprintf(os.Stderr, "DEBUG: extractModuleOverview called for module: %s\n", p.Name)
		fmt.Fprintf(os.Stderr, "DEBUG: Total sections: %d\n", len(sections))
	}

	// Helper function to search recursively
	var findOverviewSection func(secs []markdown.Section) *markdown.Section
	findOverviewSection = func(secs []markdown.Section) *markdown.Section {
		for _, sec := range secs {
			if debug {
				fmt.Fprintf(os.Stderr, "DEBUG:   Checking section level=%d title=%s\n", sec.Level, sec.Title)
			}
			if isModuleOverviewTitle(sec.Title) {
				return &sec
			}
			// Search in children
			if len(sec.Children) > 0 {
				if found := findOverviewSection(sec.Children); found != nil {
					return found
				}
			}
		}
		return nil
	}

	// Find the "模块概述" (Module Overview) section
	overviewSec := findOverviewSection(sections)
	if overviewSec != nil {
		content := overviewSec.Content
		if debug {
			fmt.Fprintf(os.Stderr, "DEBUG: Found module overview section: %s\n", overviewSec.Title)
			fmt.Fprintf(os.Stderr, "DEBUG: Content length: %d bytes\n", len(content))
		}
		p.Responsibility = extractField(content, "模块职责")
		p.Research = extractListField(content, "对应 Research")
		// Extract dependencies from module overview content
		p.Dependencies = extractListField(content, "依赖模块")
		p.Dependents = extractListField(content, "被依赖模块")
		if debug {
			fmt.Fprintf(os.Stderr, "DEBUG: Extracted dependencies: %v\n", p.Dependencies)
			fmt.Fprintf(os.Stderr, "DEBUG: Extracted dependents: %v\n", p.Dependents)
		}
		return // Found and extracted, we're done
	}

	// Find the "依赖模块" (Dependencies) section (if exists as separate section)
	var findDepsSection func(secs []markdown.Section) *markdown.Section
	findDepsSection = func(secs []markdown.Section) *markdown.Section {
		for _, sec := range secs {
			if isMatchingTitle(sec.Title, "依赖模块", "Dependencies") {
				return &sec
			}
			if len(sec.Children) > 0 {
				if found := findDepsSection(sec.Children); found != nil {
					return found
				}
			}
		}
		return nil
	}

	depsSec := findDepsSection(sections)
	if depsSec != nil {
		content := depsSec.Content
		p.Dependencies = extractListField(content, "依赖模块")
		p.Dependents = extractListField(content, "被依赖模块")
		if debug {
			fmt.Fprintf(os.Stderr, "DEBUG: Found separate dependencies section\n")
			fmt.Fprintf(os.Stderr, "DEBUG: Dependencies: %v\n", p.Dependencies)
		}
		return
	}

	// If dependencies still not found, search in all section contents
	if len(p.Dependencies) == 0 {
		for _, sec := range sections {
			deps := extractListField(sec.Content, "依赖模块")
			if len(deps) > 0 {
				p.Dependencies = deps
				p.Dependents = extractListField(sec.Content, "被依赖模块")
				if debug {
					fmt.Fprintf(os.Stderr, "DEBUG: Extracted from section '%s': deps=%v\n", sec.Title, deps)
				}
				break
			}
			// Also search in children
			var searchChildren func(children []markdown.Section) bool
			searchChildren = func(children []markdown.Section) bool {
				for _, child := range children {
					deps := extractListField(child.Content, "依赖模块")
					if len(deps) > 0 {
						p.Dependencies = deps
						p.Dependents = extractListField(child.Content, "被依赖模块")
						if debug {
							fmt.Fprintf(os.Stderr, "DEBUG: Extracted from child section '%s': deps=%v\n", child.Title, deps)
						}
						return true
					}
					if len(child.Children) > 0 {
						if searchChildren(child.Children) {
							return true
						}
					}
				}
				return false
			}
			if len(sec.Children) > 0 && searchChildren(sec.Children) {
				break
			}
		}
	}

	// Last resort: search in raw content
	if len(p.Dependencies) == 0 && p.RawContent != "" {
		p.Dependencies = extractListField(p.RawContent, "依赖模块")
		p.Dependents = extractListField(p.RawContent, "被依赖模块")
		if debug && len(p.Dependencies) > 0 {
			fmt.Fprintf(os.Stderr, "DEBUG: Extracted from raw content: deps=%v\n", p.Dependencies)
		}
	}

	if debug {
		fmt.Fprintf(os.Stderr, "DEBUG: extractModuleOverview finished for %s: deps=%v\n", p.Name, p.Dependencies)
	}
}

// isModuleOverviewTitle checks if the title indicates module overview section.
func isModuleOverviewTitle(title string) bool {
	lower := strings.ToLower(title)
	return strings.Contains(lower, "模块概述") ||
		strings.Contains(lower, "module overview") ||
		strings.Contains(lower, "overview")
}

// extractField extracts a field value from content.
// Format: **Field**: value or **Field**: value
func extractField(content, fieldName string) string {
	// Match patterns like "**模块职责**: value" or "**模块职责**: value"
	// Stop at newline or next ** field
	pattern := regexp.MustCompile(`\*\*` + regexp.QuoteMeta(fieldName) + `\*\*[:：]\s*([^\n]+?)(?:\n|$)`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// extractListField extracts a list field from content.
// Format: **Field**:
// - item1
// - item2
func extractListField(content, fieldName string) []string {
	var result []string

	// First try to find list items after the field
	// Match the field line and capture following list items
	lines := strings.Split(content, "\n")
	inField := false
	fieldPattern := regexp.MustCompile(`^\s*\*\*` + regexp.QuoteMeta(fieldName) + `\*\*[:：]`)
	listItemPattern := regexp.MustCompile(`^\s*[-*]\s*(.+)$`)

	for _, line := range lines {
		if fieldPattern.MatchString(line) {
			inField = true
			// Check if there's an inline value on the same line
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				if value != "" && value != "无" && !strings.HasPrefix(value, "-") && !strings.HasPrefix(value, "*") {
					// Inline list like: item1, item2
					items := strings.Split(value, ",")
					for _, item := range items {
						trimmed := strings.TrimSpace(item)
						if trimmed != "" && trimmed != "无" {
							result = append(result, trimmed)
						}
					}
					return result
				}
			}
			continue
		}

		if inField {
			// Check if we hit another field
			if strings.HasPrefix(line, "**") && strings.Contains(line, "**:") {
				break
			}
			// Check if this is a list item
			if matches := listItemPattern.FindStringSubmatch(line); matches != nil {
				result = append(result, strings.TrimSpace(matches[1]))
			} else if strings.TrimSpace(line) == "" {
				// Empty line - continue in case there are more items
				continue
			} else if !strings.HasPrefix(strings.TrimSpace(line), "-") &&
				!strings.HasPrefix(strings.TrimSpace(line), "*") &&
				len(strings.TrimSpace(line)) > 0 {
				// Non-list content after field, stop
				break
			}
		}
	}

	// If no list items found, try inline format
	if len(result) == 0 {
		fieldValue := extractField(content, fieldName)
		if fieldValue != "" && fieldValue != "无" {
			// Split by comma if there are multiple items inline
			items := strings.Split(fieldValue, ",")
			for _, item := range items {
				trimmed := strings.TrimSpace(item)
				if trimmed != "" && trimmed != "无" {
					result = append(result, trimmed)
				}
			}
		}
	}

	return result
}

// extractJobsFromAllSections extracts jobs from all sections recursively.
func extractJobsFromAllSections(sections []markdown.Section) []Job {
	var jobs []Job

	for _, sec := range sections {
		if job := extractJobFromSection(sec); job != nil {
			jobs = append(jobs, *job)
		}
		// Also check children recursively
		if len(sec.Children) > 0 {
			childJobs := extractJobsFromAllSections(sec.Children)
			jobs = append(jobs, childJobs...)
		}
	}

	return jobs
}

// extractJobFromSection extracts a Job from a section if it matches Job format.
// Job format: "### Job N: JobName"
func extractJobFromSection(sec markdown.Section) *Job {
	// Check if this is a Job section (level 3 heading starting with "Job")
	if sec.Level != 3 {
		return nil
	}

	title := sec.Title

	// Match "Job N: Name" or "JobN: Name" pattern (case insensitive)
	jobPattern := regexp.MustCompile(`(?i)^job\s*(\d+)[:：]\s*(.+)$`)
	matches := jobPattern.FindStringSubmatch(title)

	if len(matches) < 3 {
		return nil
	}

	var jobIndex int
	fmt.Sscanf(matches[1], "%d", &jobIndex)

	job := &Job{
		Name:  strings.TrimSpace(matches[2]),
		Index: jobIndex,
	}

	content := sec.Content

	// Try to extract from #### subsections first (new format)
	// If not found, fall back to ** field format (old format)
	job.Goal = extractFromSubsectionOrField(sec, "目标", "Goal")
	job.Prerequisites = extractListFromSubsectionOrField(sec, "前置条件", "Prerequisites")
	job.Tasks = extractTasksFromSubsectionOrContent(sec, content)
	job.Validators = extractValidatorsFromSubsectionOrContent(sec, content)
	job.DebugLogs = extractDebugLogsFromSubsectionOrContent(sec, content)

	// Extract completion status
	job.CompletionStatus = extractFromSubsectionOrField(sec, "完成状态", "Completion Status")
	job.IsCompleted = isJobMarkedCompleted(job.CompletionStatus)

	return job
}

// extractTasksFromContent extracts tasks from job content.
func extractTasksFromContent(content string) []TaskItem {
	var tasks []TaskItem

	// Find the Tasks section - look for "**Tasks" or "**Tasks (Todo 列表)**"
	// Match up to the colon/newline and capture rest
	lines := strings.Split(content, "\n")
	inTasksSection := false
	var taskLines []string

	for i, line := range lines {
		// Check if this is the Tasks section header
		if regexp.MustCompile(`(?i)^\s*\*\*tasks?`).MatchString(line) {
			inTasksSection = true
			continue
		}

		if inTasksSection {
			// Check if we hit another section header
			if regexp.MustCompile(`^\s*\*\*[^*]+\*\*`).MatchString(line) {
				break
			}
			// Stop at separator line (---)
			if strings.TrimSpace(line) == "---" {
				break
			}
			// Collect task lines
			if strings.TrimSpace(line) != "" {
				taskLines = append(taskLines, line)
			}
		} else if i < len(lines)-1 {
			// Check if next line is a task (Tasks without header)
			nextLine := lines[i+1]
			if regexp.MustCompile(`^\s*[-*]\s*\[[ xX]\]\s*task\s*\d+`).MatchString(strings.ToLower(nextLine)) {
				taskLines = append(taskLines, line)
			}
		}
	}

	taskContent := strings.Join(taskLines, "\n")
	if taskContent == "" {
		taskContent = content
	}

	// Extract task items in two formats:
	// 1. "- [ ] Task N: description" or "- [x] Task N: description" (with explicit index)
	// 2. "- [ ] description" or "- [x] description" (auto-assign index)

	// First try to match tasks with explicit index
	taskWithIndexPattern := regexp.MustCompile(`(?im)^\s*[-*]\s*\[([ xX])\]\s*task\s*(\d+)[:：]\s*(.+)$`)
	indexedMatches := taskWithIndexPattern.FindAllStringSubmatch(taskContent, -1)

	if len(indexedMatches) > 0 {
		// Use explicitly indexed tasks
		for _, match := range indexedMatches {
			if len(match) >= 4 {
				var index int
				fmt.Sscanf(match[2], "%d", &index)
				tasks = append(tasks, TaskItem{
					Index:       index,
					Description: strings.TrimSpace(match[3]),
					Completed:   strings.ToLower(match[1]) == "x",
				})
			}
		}
	} else {
		// No explicitly indexed tasks found, try simple checkbox format
		simpleTaskPattern := regexp.MustCompile(`(?im)^\s*[-*]\s*\[([ xX])\]\s*(.+)$`)
		simpleMatches := simpleTaskPattern.FindAllStringSubmatch(taskContent, -1)

		for i, match := range simpleMatches {
			if len(match) >= 3 {
				tasks = append(tasks, TaskItem{
					Index:       i + 1, // Auto-assign 1-based index
					Description: strings.TrimSpace(match[2]),
					Completed:   strings.ToLower(match[1]) == "x",
				})
			}
		}
	}

	return tasks
}

// extractValidators extracts validator items from content.
func extractValidators(content string) []string {
	var validators []string

	// Find the Validators section
	validatorPattern := regexp.MustCompile(`(?i)\*\*验证器\*\*[:：]?\s*\n`)
	validatorLoc := validatorPattern.FindStringIndex(content)

	if validatorLoc == nil {
		return validators
	}

	// Extract content from after Validators header to next ** header or end
	start := validatorLoc[1]
	end := len(content)
	nextHeader := regexp.MustCompile(`\n\s*\*\*[^*]+\*\*[:：]?`).FindStringIndex(content[start:])
	if nextHeader != nil {
		end = start + nextHeader[0]
	}
	validatorContent := content[start:end]

	// Extract validator items: "- [ ] description" or "- [x] description" or "- description"
	itemPattern := regexp.MustCompile(`(?m)^\s*[-*]\s*(?:\[[ xX]\]\s*)?(.+)$`)
	items := itemPattern.FindAllStringSubmatch(validatorContent, -1)

	for _, item := range items {
		if len(item) > 1 {
			desc := strings.TrimSpace(item[1])
			// Skip debug log entries
			if desc != "" && !regexp.MustCompile(`^(debug|explore)\d+[:：]`).MatchString(desc) {
				validators = append(validators, desc)
			}
		}
	}

	return validators
}

// extractDebugLogs extracts debug log entries from content.
func extractDebugLogs(content string) []DebugLog {
	var logs []DebugLog

	// Find the Debug Logs section
	debugPattern := regexp.MustCompile(`(?i)\*\*调试日志\*\*[:：]?\s*\n`)
	debugLoc := debugPattern.FindStringIndex(content)

	if debugLoc == nil {
		return logs
	}

	// Extract content from after Debug Logs header to next ** header or end
	start := debugLoc[1]
	end := len(content)
	nextHeader := regexp.MustCompile(`\n\s*\*\*[^*]+\*\*[:：]?`).FindStringIndex(content[start:])
	if nextHeader != nil {
		end = start + nextHeader[0]
	}
	debugContent := content[start:end]

	// Check if it says "无" (none) or is empty
	if strings.TrimSpace(debugContent) == "无" || strings.TrimSpace(debugContent) == "" ||
		strings.TrimSpace(debugContent) == "- 无" {
		return logs
	}

	// Extract debug entries: "- debug1: phenomenon, reproduction, hypothesis, verification, fix, progress"
	entryPattern := regexp.MustCompile(`(?m)^\s*[-*]\s*(debug\d+|explore\d+)[:：]\s*(.+)$`)
	entries := entryPattern.FindAllStringSubmatch(debugContent, -1)

	for _, entry := range entries {
		if len(entry) > 2 {
			logID := entry[1]
			fields := strings.Split(entry[2], ",")

			// Pad fields to ensure we have all 6 fields
			for len(fields) < 6 {
				fields = append(fields, "")
			}

			logs = append(logs, DebugLog{
				ID:           logID,
				Phenomenon:   strings.TrimSpace(fields[0]),
				Reproduction: strings.TrimSpace(fields[1]),
				Hypothesis:   strings.TrimSpace(fields[2]),
				Verification: strings.TrimSpace(fields[3]),
				Fix:          strings.TrimSpace(fields[4]),
				Progress:     strings.TrimSpace(fields[5]),
			})
		}
	}

	return logs
}

// GetJobByIndex returns a job by its index number.
func (p *Plan) GetJobByIndex(index int) (*Job, error) {
	for _, job := range p.Jobs {
		if job.Index == index {
			return &job, nil
		}
	}
	return nil, fmt.Errorf("job with index %d not found", index)
}

// GetJobByName returns a job by its name (case-insensitive).
func (p *Plan) GetJobByName(name string) (*Job, error) {
	searchName := strings.ToLower(strings.TrimSpace(name))
	for _, job := range p.Jobs {
		if strings.ToLower(job.Name) == searchName {
			return &job, nil
		}
	}
	return nil, fmt.Errorf("job with name %q not found", name)
}

// CountTasks returns the total number of tasks across all jobs.
func (p *Plan) CountTasks() (total, completed int) {
	for _, job := range p.Jobs {
		for _, task := range job.Tasks {
			total++
			if task.Completed {
				completed++
			}
		}
	}
	return total, completed
}

// GetPendingTasks returns all pending (incomplete) tasks.
func (p *Plan) GetPendingTasks() []TaskItem {
	var pending []TaskItem
	for _, job := range p.Jobs {
		for _, task := range job.Tasks {
			if !task.Completed {
				pending = append(pending, task)
			}
		}
	}
	return pending
}

// extractFromSubsectionOrField extracts content from #### subsection or ** field.
// Tries subsection first (new format), falls back to field (old format).
func extractFromSubsectionOrField(sec markdown.Section, subsectionTitle, fieldName string) string {
	// Try to find #### subsection first
	for _, child := range sec.Children {
		if child.Level == 4 && isMatchingTitle(child.Title, subsectionTitle) {
			// Found subsection, return its content
			return strings.TrimSpace(child.Content)
		}
	}
	// Fall back to ** field format
	return extractField(sec.Content, subsectionTitle)
}

// extractListFromSubsectionOrField extracts list from #### subsection or ** field.
func extractListFromSubsectionOrField(sec markdown.Section, subsectionTitle, fieldName string) []string {
	// Try to find #### subsection first
	for _, child := range sec.Children {
		if child.Level == 4 && isMatchingTitle(child.Title, subsectionTitle) {
			// Found subsection, parse its content as list
			return parseListContent(child.Content)
		}
	}
	// Fall back to ** field format
	return extractListField(sec.Content, subsectionTitle)
}

// extractTasksFromSubsectionOrContent extracts tasks from #### subsection or content.
func extractTasksFromSubsectionOrContent(sec markdown.Section, content string) []TaskItem {
	// Try to find #### Tasks subsection first
	for _, child := range sec.Children {
		if child.Level == 4 && isMatchingTitle(child.Title, "Tasks", "Todo 列表", "任务列表") {
			// Found Tasks subsection, extract tasks from its content
			return extractTasksFromContent(child.Content)
		}
	}
	// Fall back to extracting from full content
	return extractTasksFromContent(content)
}

// extractValidatorsFromSubsectionOrContent extracts validators from #### subsection or content.
func extractValidatorsFromSubsectionOrContent(sec markdown.Section, content string) []string {
	// Try to find #### Validators subsection first
	for _, child := range sec.Children {
		if child.Level == 4 && isMatchingTitle(child.Title, "验证器", "Validator") {
			// Found Validators subsection, parse its content
			return parseValidatorContent(child.Content)
		}
	}
	// Fall back to extracting from full content
	return extractValidators(content)
}

// extractDebugLogsFromSubsectionOrContent extracts debug logs from #### subsection or content.
func extractDebugLogsFromSubsectionOrContent(sec markdown.Section, content string) []DebugLog {
	// Try to find #### Debug Logs subsection first
	for _, child := range sec.Children {
		if child.Level == 4 && isMatchingTitle(child.Title, "调试日志", "Debug", "Debug Logs") {
			// Found Debug Logs subsection, parse its content
			return parseDebugLogContent(child.Content)
		}
	}
	// Fall back to extracting from full content
	return extractDebugLogs(content)
}

// isMatchingTitle checks if a title matches any of the given keywords (case-insensitive).
func isMatchingTitle(title string, keywords ...string) bool {
	lower := strings.ToLower(strings.TrimSpace(title))
	for _, keyword := range keywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

// parseListContent parses list content into a string slice.
// Supports both:
// - "- item1\n- item2" format
// - "job_1, job_2" comma-separated format
// - "job_1 - description" format (extracts job_N)
func parseListContent(content string) []string {
	var result []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "无" {
			continue
		}

		// Check if it's a list item
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			// Extract content after "-" or "*"
			item := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "-"), "*"))
			item = strings.TrimSpace(strings.TrimPrefix(item, "*"))

			// Extract job_N from "job_1 - description" format
			if strings.Contains(item, " - ") {
				parts := strings.SplitN(item, " - ", 2)
				item = strings.TrimSpace(parts[0])
			}

			if item != "" && item != "无" {
				result = append(result, item)
			}
		} else if strings.Contains(line, ",") {
			// Comma-separated format
			items := strings.Split(line, ",")
			for _, item := range items {
				item = strings.TrimSpace(item)
				if item != "" && item != "无" {
					result = append(result, item)
				}
			}
		} else if line != "" {
			// Single item
			result = append(result, line)
		}
	}

	return result
}

// parseValidatorContent parses validator content from subsection.
func parseValidatorContent(content string) []string {
	var validators []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip debug log entries
		if regexp.MustCompile(`^(debug|explore)\d+[:：]`).MatchString(line) {
			continue
		}

		// Check if it's a list item
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			item := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "-"), "*"))
			item = strings.TrimSpace(strings.TrimPrefix(item, "*"))
			if item != "" {
				validators = append(validators, item)
			}
		} else if line != "" {
			// Non-list item, add as-is
			validators = append(validators, line)
		}
	}

	return validators
}

// parseDebugLogContent parses debug log content from subsection.
func parseDebugLogContent(content string) []DebugLog {
	var logs []DebugLog

	// Check if it says "无" (none) or is empty
	trimmed := strings.TrimSpace(content)
	if trimmed == "无" || trimmed == "" || trimmed == "- 无" {
		return logs
	}

	// Extract debug entries: "- debug1: phenomenon, reproduction, hypothesis, verification, fix, progress"
	entryPattern := regexp.MustCompile(`(?m)^\s*[-*]\s*(debug\d+|explore\d+)[:：]\s*(.+)$`)
	entries := entryPattern.FindAllStringSubmatch(content, -1)

	for _, entry := range entries {
		if len(entry) > 2 {
			logID := entry[1]
			fields := strings.Split(entry[2], ",")

			// Pad fields to ensure we have all 6 fields
			for len(fields) < 6 {
				fields = append(fields, "")
			}

			logs = append(logs, DebugLog{
				ID:           logID,
				Phenomenon:   strings.TrimSpace(fields[0]),
				Reproduction: strings.TrimSpace(fields[1]),
				Hypothesis:   strings.TrimSpace(fields[2]),
				Verification: strings.TrimSpace(fields[3]),
				Fix:          strings.TrimSpace(fields[4]),
				Progress:     strings.TrimSpace(fields[5]),
			})
		}
	}

	return logs
}

// isJobMarkedCompleted checks if the completion status indicates the job is completed.
// Returns true if the status contains completion markers like "✅", "已完成", "completed", etc.
func isJobMarkedCompleted(completionStatus string) bool {
	if completionStatus == "" {
		return false
	}

	lower := strings.ToLower(completionStatus)

	// Check for completion markers
	markers := []string{
		"✅",
		"已完成",
		"完成",
		"completed",
		"done",
		"finished",
	}

	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}

	return false
}
