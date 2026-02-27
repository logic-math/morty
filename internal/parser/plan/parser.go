// Package plan provides Plan file parsing functionality.
package plan

import (
	"fmt"
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
	// Find the "模块概述" (Module Overview) section
	for _, sec := range sections {
		if isModuleOverviewTitle(sec.Title) {
			content := sec.Content
			p.Responsibility = extractField(content, "模块职责")
			p.Research = extractListField(content, "对应 Research")
			p.Dependencies = extractListField(content, "依赖模块")
			p.Dependents = extractListField(content, "被依赖模块")
			return
		}
	}

	// If not found as a section, look in the first section content
	if len(sections) > 0 {
		content := sections[0].Content
		p.Responsibility = extractField(content, "模块职责")
		p.Research = extractListField(content, "对应 Research")
		p.Dependencies = extractListField(content, "依赖模块")
		p.Dependents = extractListField(content, "被依赖模块")
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

	// Extract goal
	job.Goal = extractField(content, "目标")

	// Extract prerequisites
	job.Prerequisites = extractListField(content, "前置条件")

	// Extract tasks
	job.Tasks = extractTasksFromContent(content)

	// Extract validators
	job.Validators = extractValidators(content)

	// Extract debug logs
	job.DebugLogs = extractDebugLogs(content)

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
