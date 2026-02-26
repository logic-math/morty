// Package executor provides job execution engine for Morty.
package executor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/parser/plan"
)

// ResultParser defines the interface for parsing AI CLI execution results.
type ResultParser interface {
	// Parse reads and parses the output file from AI CLI execution.
	// It extracts RALPH_STATUS JSON and returns structured execution result.
	Parse(outputFile string) (*RALPHExecutionResult, error)
}

// RALPHExecutionResult represents the parsed result from AI CLI output (RALPH_STATUS format).
// This is the result returned by the ResultParser after parsing AI CLI output.
type RALPHExecutionResult struct {
	// Status indicates the execution status (COMPLETED, FAILED, RUNNING)
	Status string `json:"status"`
	// TasksCompleted is the number of tasks completed
	TasksCompleted int `json:"tasks_completed"`
	// TasksTotal is the total number of tasks
	TasksTotal int `json:"tasks_total"`
	// Summary is a brief description of the execution result
	Summary string `json:"summary"`
	// Module is the module name
	Module string `json:"module"`
	// Job is the job name
	Job string `json:"job"`
	// LoopCount is the current loop iteration count
	LoopCount int `json:"loop_count,omitempty"`
	// DebugIssues is the number of debug issues found
	DebugIssues int `json:"debug_issues,omitempty"`
	// DebugLogsInPlan indicates if debug logs were recorded in plan
	DebugLogsInPlan bool `json:"debug_logs_in_plan,omitempty"`
	// ExploreSubagentUsed indicates if explore subagent was used
	ExploreSubagentUsed bool `json:"explore_subagent_used,omitempty"`
	// RawRALPHStatus contains the raw RALPH_STATUS JSON block
	RawRALPHStatus string `json:"-"`
	// Errors contains any error messages extracted from output
	Errors []string `json:"errors,omitempty"`
	// Stderr contains the stderr output
	Stderr string `json:"stderr,omitempty"`
}

// IsSuccess returns true if the execution was successful.
func (r *RALPHExecutionResult) IsSuccess() bool {
	return strings.ToUpper(r.Status) == "COMPLETED"
}

// IsFailed returns true if the execution failed.
func (r *RALPHExecutionResult) IsFailed() bool {
	return strings.ToUpper(r.Status) == "FAILED"
}

// IsRunning returns true if the execution is still running.
func (r *RALPHExecutionResult) IsRunning() bool {
	return strings.ToUpper(r.Status) == "RUNNING"
}

// resultParser implements the ResultParser interface.
type resultParser struct {
	logger   logging.Logger
	planDir  string
}

// ResultParserConfig holds configuration for creating a ResultParser.
type ResultParserConfig struct {
	// PlanDir is the directory containing plan files (default: ".morty/plan")
	PlanDir string
}

// DefaultResultParserConfig returns the default configuration.
func DefaultResultParserConfig() *ResultParserConfig {
	return &ResultParserConfig{
		PlanDir: ".morty/plan",
	}
}

// NewResultParser creates a new ResultParser with the given dependencies.
//
// Parameters:
//   - logger: The logger for recording parsing progress
//   - config: Optional configuration. If nil, default config is used.
//
// Returns:
//   - A ResultParser implementation
func NewResultParser(logger logging.Logger, config *ResultParserConfig) ResultParser {
	if config == nil {
		config = DefaultResultParserConfig()
	}
	return &resultParser{
		logger:  logger,
		planDir: config.PlanDir,
	}
}

// Parse reads and parses the output file from AI CLI execution.
// It extracts RALPH_STATUS JSON block and returns structured execution result.
//
// Parameters:
//   - outputFile: Path to the output file containing AI CLI output
//
// Returns:
//   - Parsed RALPHExecutionResult
//   - An error if parsing fails
func (rp *resultParser) Parse(outputFile string) (*RALPHExecutionResult, error) {
	rp.logger.Info("Parsing execution result", logging.String("output_file", outputFile))

	// Read the output file
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read output file %s: %w", outputFile, err)
	}

	contentStr := string(content)

	// Extract RALPH_STATUS JSON block
	ralphJSON, err := rp.extractRALPHStatus(contentStr)
	if err != nil {
		rp.logger.Warn("Failed to extract RALPH_STATUS, trying fallback parsing",
			logging.String("error", err.Error()))
		// Try fallback parsing from entire content
		ralphJSON = rp.findJSONBlock(contentStr)
	}

	// Parse the JSON into ExecutionResult
	result, err := rp.parseRALPHStatus(ralphJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RALPH_STATUS: %w", err)
	}

	// Store raw RALPH status
	result.RawRALPHStatus = ralphJSON

	// Extract errors from stderr or error patterns in output
	result.Errors = rp.extractErrors(contentStr)
	result.Stderr = rp.extractStderr(contentStr)

	rp.logger.Info("Execution result parsed successfully",
		logging.String("module", result.Module),
		logging.String("job", result.Job),
		logging.String("status", result.Status),
		logging.Int("tasks_completed", result.TasksCompleted),
		logging.Int("tasks_total", result.TasksTotal))

	return result, nil
}

// extractRALPHStatus extracts the RALPH_STATUS JSON block from the output.
// It looks for the <!-- RALPH_STATUS --> ... <!-- END_RALPH_STATUS --> markers.
func (rp *resultParser) extractRALPHStatus(content string) (string, error) {
	// Try to find RALPH_STATUS markers
	startMarker := "<!-- RALPH_STATUS -->"
	endMarker := "<!-- END_RALPH_STATUS -->"

	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		// Try without HTML comment format (just JSON)
		return rp.findJSONBlock(content), nil
	}

	// Find end marker in the full content (not relative)
	endIdx := strings.Index(content, endMarker)
	if endIdx == -1 || endIdx <= startIdx {
		return "", fmt.Errorf("RALPH_STATUS start marker found but no end marker")
	}

	// Extract content between markers
	startIdx += len(startMarker)
	jsonContent := content[startIdx:endIdx]

	// Trim whitespace
	jsonContent = strings.TrimSpace(jsonContent)

	return jsonContent, nil
}

// findJSONBlock finds a JSON block in the content that looks like RALPH_STATUS.
// This is a fallback method when markers are not present.
func (rp *resultParser) findJSONBlock(content string) string {
	// Look for JSON blocks with status field
	jsonPattern := regexp.MustCompile(`(?s)\{[^{}]*"status"[^{}]*(?:\{[^{}]*\}[^{}]*)*\}`)
	matches := jsonPattern.FindAllString(content, -1)

	// Find the match that has RALPH_STATUS fields
	for _, match := range matches {
		matchLower := strings.ToLower(match)
		if strings.Contains(matchLower, "module") &&
			strings.Contains(matchLower, "job") &&
			(strings.Contains(matchLower, "tasks_completed") ||
				strings.Contains(matchLower, "tasks_total")) {
			return match
		}
	}

	// If no specific match found, return the last JSON block (most likely RALPH_STATUS)
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}

	return ""
}

// parseRALPHStatus parses the RALPH_STATUS JSON into RALPHExecutionResult.
// It supports both nested format (ralph_status: {...}) and flat format.
func (rp *resultParser) parseRALPHStatus(jsonContent string) (*RALPHExecutionResult, error) {
	if jsonContent == "" {
		return nil, fmt.Errorf("empty JSON content")
	}

	result := &RALPHExecutionResult{}

	// First, try to parse as nested format
	var nested struct {
		RALPHStatus *RALPHExecutionResult `json:"ralph_status"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &nested); err == nil && nested.RALPHStatus != nil {
		// Nested format detected
		result = nested.RALPHStatus
	} else {
		// Try flat format - parse directly into result
		if err := json.Unmarshal([]byte(jsonContent), result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	}

	// Validate required fields
	if result.Status == "" {
		return nil, fmt.Errorf("RALPH_STATUS missing required field: status")
	}

	// Normalize status to uppercase
	result.Status = strings.ToUpper(result.Status)

	return result, nil
}

// extractErrors extracts error messages from the output content.
func (rp *resultParser) extractErrors(content string) []string {
	var errors []string

	// Look for error patterns
	errorPatterns := []string{
		`(?i)error[\s:]+([^.]+\.)`,
		`(?i)failed[\s:]+([^.]+\.)`,
		`(?i)exception[\s:]+([^.]+\.)`,
		`(?i)pani(?:c|cked)[\s:]+([^.]+\.)`,
	}

	for _, pattern := range errorPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				err := strings.TrimSpace(match[1])
				if err != "" && !rp.containsError(errors, err) {
					errors = append(errors, err)
				}
			}
		}
	}

	return errors
}

// containsError checks if errors slice already contains a similar error.
func (rp *resultParser) containsError(errors []string, err string) bool {
	for _, e := range errors {
		if strings.Contains(e, err) || strings.Contains(err, e) {
			return true
		}
	}
	return false
}

// extractStderr extracts stderr content from the output.
func (rp *resultParser) extractStderr(content string) string {
	// Look for stderr section - supports formats like:
	// "Stderr: message" or "Standard Error: message" or "Stderr:\nmessage"
	stderrPattern := regexp.MustCompile(`(?i)(?:stderr|standard error)[\s:]*\s*(.+?)(?:\n\n|\n---|\n[A-Z]|$)`)
	matches := stderrPattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// UpdatePlanDebugLogs updates the Plan file with new debug log entries.
// This is called when errors are detected in the execution output.
//
// Parameters:
//   - module: The module name
//   - job: The job name
//   - debugLogs: The debug log entries to add
//
// Returns:
//   - An error if update fails
func (rp *resultParser) UpdatePlanDebugLogs(module, job string, debugLogs []plan.DebugLog) error {
	if len(debugLogs) == 0 {
		return nil
	}

	planPath := filepath.Join(rp.planDir, module+".md")

	// Read existing plan content
	content, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("failed to read plan file %s: %w", planPath, err)
	}

	// Parse the plan
	parsedPlan, err := plan.ParsePlan(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse plan: %w", err)
	}

	// Find the job
	var targetJob *plan.Job
	for i := range parsedPlan.Jobs {
		if strings.EqualFold(parsedPlan.Jobs[i].Name, job) ||
			fmt.Sprintf("job_%d", parsedPlan.Jobs[i].Index) == job {
			targetJob = &parsedPlan.Jobs[i]
			break
		}
	}

	if targetJob == nil {
		return fmt.Errorf("job %s not found in plan", job)
	}

	// Append new debug logs
	targetJob.DebugLogs = append(targetJob.DebugLogs, debugLogs...)

	// Write updated plan back
	updatedContent := rp.rebuildPlanContent(string(content), targetJob)

	if err := os.WriteFile(planPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated plan: %w", err)
	}

	rp.logger.Info("Updated plan debug logs",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("new_logs", len(debugLogs)))

	return nil
}

// rebuildPlanContent rebuilds the plan content with updated debug logs.
func (rp *resultParser) rebuildPlanContent(originalContent string, job *plan.Job) string {
	// Find the debug logs section for this job
	jobPattern := regexp.MustCompile(fmt.Sprintf(`(?i)### Job\s*%d[^#]*`, job.Index))

	// Find where this job section ends (next ### or end of file)
	jobMatch := jobPattern.FindStringIndex(originalContent)
	if jobMatch == nil {
		return originalContent
	}

	jobStart := jobMatch[0]
	jobEnd := len(originalContent)

	// Find next job section
	nextJobPattern := regexp.MustCompile(`\n### Job \d+`)
	nextJobMatch := nextJobPattern.FindStringIndex(originalContent[jobMatch[1]:])
	if nextJobMatch != nil {
		jobEnd = jobMatch[1] + nextJobMatch[0]
	}

	jobSection := originalContent[jobStart:jobEnd]

	// Find and replace debug logs section
	debugPattern := regexp.MustCompile(`(?i)(\*\*调试日志\*\*[:：]?\s*\n)(.*?)(\n\*\*|$)`)

	// Build new debug logs content
	var debugContent strings.Builder
	debugContent.WriteString("**调试日志**:\n")
	for _, log := range job.DebugLogs {
		debugContent.WriteString(fmt.Sprintf("- %s: %s, %s, %s, %s, %s, %s\n",
			log.ID,
			log.Phenomenon,
			log.Reproduction,
			log.Hypothesis,
			log.Verification,
			log.Fix,
			log.Progress))
	}

	// Replace or add debug logs section
	if debugPattern.MatchString(jobSection) {
		// Replace existing section
		newJobSection := debugPattern.ReplaceAllString(jobSection, "${1}"+debugContent.String()+"${3}")
		return originalContent[:jobStart] + newJobSection + originalContent[jobEnd:]
	}

	// Add new debug logs section before the end of job section
	// Find a good insertion point (after validators or tasks)
	insertPattern := regexp.MustCompile(`(?i)(\*\*验证器\*\*[:：]?\s*\n.*?)(\n\n|\n###|$)`)
	insertMatch := insertPattern.FindStringIndex(jobSection)
	if insertMatch != nil {
		insertPos := jobStart + insertMatch[1] - len("${2}")
		return originalContent[:insertPos] + "\n\n" + debugContent.String() + originalContent[insertPos:]
	}

	// Fallback: append to job section
	return originalContent[:jobEnd] + "\n\n" + debugContent.String() + originalContent[jobEnd:]
}

// CreateDebugLog creates a new debug log entry from error information.
//
// Parameters:
//   - id: Log ID (debug1, debug2, etc.)
//   - phenomenon: Issue description
//   - reproduction: How to reproduce
//   - hypothesis: Possible causes
//   - verification: Verification steps
//   - fix: Fix method
//   - progress: Fix progress
//
// Returns:
//   - A DebugLog entry
func CreateDebugLog(id, phenomenon, reproduction, hypothesis, verification, fix, progress string) plan.DebugLog {
	return plan.DebugLog{
		ID:           id,
		Phenomenon:   phenomenon,
		Reproduction: reproduction,
		Hypothesis:   hypothesis,
		Verification: verification,
		Fix:          fix,
		Progress:     progress,
	}
}

// ExecutionError represents a detailed execution error for debug logging.
type ExecutionError struct {
	Timestamp   time.Time
	Type        string
	Message     string
	StackTrace  string
	Source      string
	Recoverable bool
}

// ParseErrorOutput parses error output from AI CLI execution.
// It extracts structured error information for debug logging.
//
// Parameters:
//   - output: The AI CLI output string
//
// Returns:
//   - A slice of ExecutionError structs
func (rp *resultParser) ParseErrorOutput(output string) []ExecutionError {
	var errors []ExecutionError

	scanner := bufio.NewScanner(strings.NewReader(output))
	var currentError *ExecutionError

	for scanner.Scan() {
		line := scanner.Text()

		// Check for error indicators
		if strings.Contains(line, "Error:") || strings.Contains(line, "error:") {
			if currentError != nil {
				errors = append(errors, *currentError)
			}
			currentError = &ExecutionError{
				Timestamp: time.Now(),
				Type:      "Error",
				Message:   strings.TrimSpace(strings.SplitN(line, ":", 2)[1]),
			}
		} else if strings.Contains(line, "panic:") {
			if currentError != nil {
				errors = append(errors, *currentError)
			}
			currentError = &ExecutionError{
				Timestamp: time.Now(),
				Type:      "Panic",
				Message:   strings.TrimSpace(strings.SplitN(line, ":", 2)[1]),
			}
		} else if currentError != nil {
			// Accumulate stack trace or additional info
			if strings.HasPrefix(line, "\t") || strings.Contains(line, "/") {
				currentError.StackTrace += line + "\n"
			} else if strings.Contains(line, "at ") {
				currentError.Source = strings.TrimSpace(line)
			}
		}
	}

	if currentError != nil {
		errors = append(errors, *currentError)
	}

	return errors
}

// Ensure resultParser implements ResultParser interface
var _ ResultParser = (*resultParser)(nil)
