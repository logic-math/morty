// Package cmd provides command handlers for Morty CLI commands.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/state"
)

// StatResult represents the result of a stat operation.
type StatResult struct {
	Status      string
	CurrentJob  *state.CurrentJob
	Summary     *state.Summary
	StatusInfo  *StatusInfo
	Err         error
	ExitCode    int
	Duration    time.Duration
	JSONOutput  bool
}

// StatusInfo represents comprehensive status information.
type StatusInfo struct {
	Current     CurrentJobInfo   `json:"current"`
	Previous    *PreviousJob     `json:"previous,omitempty"`
	Progress    ProgressInfo     `json:"progress"`
	Modules     []ModuleStatus   `json:"modules"`
	DebugIssues []DebugIssue     `json:"debug_issues"`
}

// CurrentJobInfo represents information about the current job.
type CurrentJobInfo struct {
	Module      string    `json:"module"`
	Job         string    `json:"job"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	LoopCount   int       `json:"loop_count"`
	StartedAt   time.Time `json:"started_at"`
	ElapsedTime string    `json:"elapsed_time,omitempty"`
}

// PreviousJob represents information about the previous completed job.
type PreviousJob struct {
	Module       string    `json:"module"`
	Job          string    `json:"job"`
	Status       string    `json:"status"`
	Duration     string    `json:"duration"`
	CompletedAt  time.Time `json:"completed_at"`
}

// ProgressInfo represents progress information.
type ProgressInfo struct {
	TotalJobs     int `json:"total_jobs"`
	CompletedJobs int `json:"completed_jobs"`
	FailedJobs    int `json:"failed_jobs"`
	PendingJobs   int `json:"pending_jobs"`
	RunningJobs   int `json:"running_jobs"`
	Percentage    int `json:"percentage"`
}

// ModuleStatus represents status of a specific module.
type ModuleStatus struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	TotalJobs      int    `json:"total_jobs"`
	CompletedJobs  int    `json:"completed_jobs"`
}

// DebugIssue represents a debug issue extracted from logs.
type DebugIssue struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Loop        int       `json:"loop"`
	Hypothesis  string    `json:"hypothesis"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
}

// StatHandler handles the stat command.
type StatHandler struct {
	cfg       config.Manager
	logger    logging.Logger
	paths     *config.Paths
}

// NewStatHandler creates a new StatHandler instance.
func NewStatHandler(cfg config.Manager, logger logging.Logger) *StatHandler {
	return &StatHandler{
		cfg:    cfg,
		logger: logger,
		paths:  config.NewPaths(),
	}
}

// Execute executes the stat command.
func (h *StatHandler) Execute(ctx context.Context, args []string) (*StatResult, error) {
	logger := h.logger.WithContext(ctx)
	startTime := time.Now()

	// Parse options
	watchMode, jsonOutput := h.parseOptions(args)

	result := &StatResult{
		ExitCode:   0,
		JSONOutput: jsonOutput,
	}

	// Check if status file exists
	statusFile := h.getStatusFilePath()
	if _, err := os.Stat(statusFile); os.IsNotExist(err) {
		logger.Info("Status file does not exist", logging.String("path", statusFile))
		result.Err = fmt.Errorf("请先运行 morty doing")
		result.ExitCode = 1
		result.Duration = time.Since(startTime)

		if jsonOutput {
			h.outputJSON(result)
		} else {
			fmt.Println("请先运行 morty doing")
		}

		return result, result.Err
	}

	// Load state (V2)
	stateManager := state.NewManager(statusFile)
	if err := stateManager.Load(); err != nil {
		logger.Error("Failed to load state", logging.String("error", err.Error()))
		result.Err = fmt.Errorf("failed to load state: %w", err)
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		return result, result.Err
	}

	// Get V2 status
	statusV2 := stateManager.GetStatusV2()
	if statusV2 == nil {
		logger.Error("Failed to get V2 status")
		result.Err = fmt.Errorf("failed to get status")
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		return result, result.Err
	}

	result.Duration = time.Since(startTime)

	// Output results using V2 format
	if jsonOutput {
		jsonStr, err := h.FormatStatusV2AsJSON(statusV2)
		if err != nil {
			logger.Error("Failed to format JSON", logging.String("error", err.Error()))
			result.Err = err
			result.ExitCode = 1
			return result, result.Err
		}
		fmt.Println(jsonStr)
	} else {
		h.DisplayStatusV2(statusV2)
	}

	// Handle watch mode
	if watchMode {
		return h.runWatchMode(ctx, result)
	}

	return result, nil
}

// parseOptions parses command line options.
// Returns (watchMode, jsonOutput).
func (h *StatHandler) parseOptions(args []string) (bool, bool) {
	watchMode := false
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "--watch", "-w":
			watchMode = true
		case "--json", "-j":
			jsonOutput = true
		}

		// Handle --key=value format
		if strings.HasPrefix(arg, "--watch=") {
			val := strings.TrimPrefix(arg, "--watch=")
			watchMode = val == "true" || val == "1"
		}
		if strings.HasPrefix(arg, "--json=") {
			val := strings.TrimPrefix(arg, "--json=")
			jsonOutput = val == "true" || val == "1"
		}
	}

	return watchMode, jsonOutput
}

// getStatusFilePath returns the path to the status file.
func (h *StatHandler) getStatusFilePath() string {
	if h.cfg != nil {
		return h.cfg.GetStatusFile()
	}
	return filepath.Join(h.paths.GetWorkDir(), "status.json")
}

// collectStatus collects comprehensive status information from the state manager.
func (h *StatHandler) collectStatus(stateManager *state.Manager) (*StatusInfo, error) {
	info := &StatusInfo{
		Modules:     make([]ModuleStatus, 0),
		DebugIssues: make([]DebugIssue, 0),
	}

	// Get current job
	currentJob, err := stateManager.GetCurrent()
	if err != nil {
		return nil, fmt.Errorf("failed to get current job: %w", err)
	}

	if currentJob != nil {
		info.Current = CurrentJobInfo{
			Module:    currentJob.Module,
			Job:       currentJob.Job,
			Status:    string(currentJob.Status),
			StartedAt: currentJob.StartedAt,
		}

		// Calculate elapsed time if job has started
		if !currentJob.StartedAt.IsZero() {
			elapsed := time.Since(currentJob.StartedAt)
			info.Current.ElapsedTime = h.formatDuration(elapsed)
		}

		// Get loop count from raw state data
		statusFile := h.getStatusFilePath()
		if data, err := os.ReadFile(statusFile); err == nil {
			var rawState struct {
				Modules map[string]struct {
					Jobs map[string]struct {
						LoopCount   int    `json:"loop_count"`
						Description string `json:"description"`
					} `json:"jobs"`
				} `json:"modules"`
			}
			if err := json.Unmarshal(data, &rawState); err == nil {
				if module, ok := rawState.Modules[currentJob.Module]; ok {
					if job, ok := module.Jobs[currentJob.Job]; ok {
						info.Current.LoopCount = job.LoopCount
						info.Current.Description = job.Description
					}
				}
			}
		}
	}

	// Get summary for progress info
	summary, err := stateManager.GetSummary()
	if err != nil {
		return nil, fmt.Errorf("failed to get summary: %w", err)
	}

	if summary != nil {
		info.Progress = ProgressInfo{
			TotalJobs:     summary.TotalJobs,
			CompletedJobs: summary.Completed,
			FailedJobs:    summary.Failed,
			PendingJobs:   summary.Pending,
			RunningJobs:   summary.Running,
		}

		// Calculate percentage
		if summary.TotalJobs > 0 {
			info.Progress.Percentage = (summary.Completed * 100) / summary.TotalJobs
		}

		// Convert module summary to module status
		for name, mod := range summary.Modules {
			modStatus := ModuleStatus{
				Name:          name,
				TotalJobs:     mod.TotalJobs,
				CompletedJobs: mod.Completed,
			}

			// Determine module status
			if mod.Running > 0 {
				modStatus.Status = "in_progress"
			} else if mod.Pending == mod.TotalJobs {
				modStatus.Status = "pending"
			} else if mod.Completed == mod.TotalJobs {
				modStatus.Status = "completed"
			} else if mod.Failed > 0 {
				modStatus.Status = "failed"
			} else {
				modStatus.Status = "partial"
			}

			info.Modules = append(info.Modules, modStatus)
		}
	}

	// Find previous completed job
	previousJob := h.findPreviousJob(stateManager, info.Current.Module, info.Current.Job)
	if previousJob != nil {
		info.Previous = previousJob
	}

	// Extract debug issues from current job
	debugIssues := h.extractDebugIssues(stateManager, info.Current.Module, info.Current.Job)
	info.DebugIssues = debugIssues

	return info, nil
}

// findPreviousJob finds the most recently completed job before the current one.
func (h *StatHandler) findPreviousJob(stateManager *state.Manager, currentModule, currentJob string) *PreviousJob {
	// Access the internal state to find completed jobs
	// We need to get the raw state data
	statusFile := h.getStatusFilePath()
	data, err := os.ReadFile(statusFile)
	if err != nil {
		return nil
	}

	var status struct {
		Modules map[string]struct {
			Jobs map[string]struct {
				Status      string    `json:"status"`
				CompletedAt time.Time `json:"completed_at"`
				StartedAt   time.Time `json:"started_at"`
			} `json:"jobs"`
		} `json:"modules"`
	}

	if err := json.Unmarshal(data, &status); err != nil {
		return nil
	}

	var mostRecent *PreviousJob

	for moduleName, module := range status.Modules {
		for jobName, job := range module.Jobs {
			// Skip current job
			if moduleName == currentModule && jobName == currentJob {
				continue
			}

			// Only consider completed jobs
			if job.Status == "COMPLETED" && !job.CompletedAt.IsZero() {
				if mostRecent == nil || job.CompletedAt.After(mostRecent.CompletedAt) {
					mostRecent = &PreviousJob{
						Module:      moduleName,
						Job:         jobName,
						Status:      job.Status,
						CompletedAt: job.CompletedAt,
					}

					// Calculate duration if we have started_at
					if !job.StartedAt.IsZero() {
						duration := job.CompletedAt.Sub(job.StartedAt)
					mostRecent.Duration = h.formatDuration(duration)
					}
				}
			}
		}
	}

	return mostRecent
}

// extractDebugIssues extracts debug issues from the current job's debug logs.
func (h *StatHandler) extractDebugIssues(stateManager *state.Manager, currentModule, currentJob string) []DebugIssue {
	if currentModule == "" || currentJob == "" {
		return []DebugIssue{}
	}

	// Read the status file directly to access debug logs
	statusFile := h.getStatusFilePath()
	data, err := os.ReadFile(statusFile)
	if err != nil {
		return []DebugIssue{}
	}

	var status struct {
		Modules map[string]struct {
			Jobs map[string]struct {
				LoopCount  int `json:"loop_count"`
				DebugLogs []struct {
					ID           string    `json:"id"`
					Timestamp    time.Time `json:"timestamp"`
					Phenomenon   string    `json:"phenomenon"`
					Hypothesis   string    `json:"hypothesis"`
					Progress     string    `json:"progress"`
				} `json:"debug_logs"`
			} `json:"jobs"`
		} `json:"modules"`
	}

	if err := json.Unmarshal(data, &status); err != nil {
		return []DebugIssue{}
	}

	module, ok := status.Modules[currentModule]
	if !ok {
		return []DebugIssue{}
	}

	job, ok := module.Jobs[currentJob]
	if !ok {
		return []DebugIssue{}
	}

	issues := make([]DebugIssue, 0, len(job.DebugLogs))
	for _, log := range job.DebugLogs {
		issue := DebugIssue{
			ID:          log.ID,
			Description: log.Phenomenon,
			Loop:        job.LoopCount,
			Hypothesis:  log.Hypothesis,
			Status:      log.Progress,
			Timestamp:   log.Timestamp,
		}
		issues = append(issues, issue)
	}

	return issues
}

// JSONOutput represents the complete JSON output structure.
type JSONOutput struct {
	Status      string         `json:"status"`
	Current     CurrentJobInfo `json:"current"`
	Previous    *PreviousJob   `json:"previous,omitempty"`
	Progress    ProgressInfo   `json:"progress"`
	Modules     []ModuleStatus `json:"modules"`
	DebugIssues []DebugIssue   `json:"debug_issues"`
	Duration    string         `json:"duration"`
	Error       string         `json:"error,omitempty"`
}

// formatJSON formats the result as a JSON string with proper indentation.
// Returns the formatted JSON string and any error encountered during marshaling.
func (h *StatHandler) formatJSON(result *StatResult) (string, error) {
	var output JSONOutput

	// Set status
	output.Status = h.getStatusString(result)
	output.Duration = h.formatDuration(result.Duration)

	// Use StatusInfo if available for enhanced output
	if result.StatusInfo != nil {
		output.Current = result.StatusInfo.Current
		output.Previous = result.StatusInfo.Previous
		output.Progress = result.StatusInfo.Progress
		output.Modules = result.StatusInfo.Modules
		output.DebugIssues = result.StatusInfo.DebugIssues
	}

	// Add error if present
	if result.Err != nil {
		output.Error = result.Err.Error()
	}

	// Marshal to JSON with indentation
	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(bytes), nil
}

// outputJSON outputs the result in JSON format.
func (h *StatHandler) outputJSON(result *StatResult) {
	jsonStr, err := h.formatJSON(result)
	if err != nil {
		// Fallback to basic error output
		errorOutput := map[string]string{
			"status": "error",
			"error":  err.Error(),
		}
		bytes, _ := json.MarshalIndent(errorOutput, "", "  ")
		fmt.Println(string(bytes))
		return
	}
	fmt.Println(jsonStr)
}

// outputText outputs the result in human-readable format.
func (h *StatHandler) outputText(result *StatResult) {
	if result.Err != nil {
		fmt.Println(result.Err.Error())
		return
	}

	// Use enhanced StatusInfo if available
	if result.StatusInfo != nil {
		h.outputEnhancedText(result)
		return
	}

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("  Morty Status")
	fmt.Println("=" + strings.Repeat("=", 60))

	// Current job info
	if result.CurrentJob != nil {
		fmt.Println()
		fmt.Printf("  Current Job: %s/%s\n", result.CurrentJob.Module, result.CurrentJob.Job)
		fmt.Printf("  Status:      %s\n", result.CurrentJob.Status)
		if !result.CurrentJob.StartedAt.IsZero() {
			fmt.Printf("  Started:     %s\n", result.CurrentJob.StartedAt.Format("2006-01-02 15:04:05"))
		}
	} else {
		fmt.Println()
		fmt.Println("  Current Job: None")
	}

	// Summary
	if result.Summary != nil {
		fmt.Println()
		fmt.Println("  Summary:")
		fmt.Printf("    Total Modules: %d\n", result.Summary.TotalModules)
		fmt.Printf("    Total Jobs:    %d\n", result.Summary.TotalJobs)
		fmt.Println()
		fmt.Println("    Status Breakdown:")
		fmt.Printf("      Pending:   %d\n", result.Summary.Pending)
		fmt.Printf("      Running:   %d\n", result.Summary.Running)
		fmt.Printf("      Completed: %d\n", result.Summary.Completed)
		fmt.Printf("      Failed:    %d\n", result.Summary.Failed)
		fmt.Printf("      Blocked:   %d\n", result.Summary.Blocked)

		// Module details
		if len(result.Summary.Modules) > 0 {
			fmt.Println()
			fmt.Println("  Modules:")
			for name, mod := range result.Summary.Modules {
				fmt.Printf("    %s:\n", name)
				fmt.Printf("      Total: %d (Pending: %d, Running: %d, Completed: %d, Failed: %d, Blocked: %d)\n",
					mod.TotalJobs, mod.Pending, mod.Running, mod.Completed, mod.Failed, mod.Blocked)
			}
		}
	}

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Printf("  Duration: %s\n", result.Duration)
	fmt.Println("=" + strings.Repeat("=", 60))
}

// outputEnhancedText outputs enhanced text format using StatusInfo.
func (h *StatHandler) outputEnhancedText(result *StatResult) {
	if result.StatusInfo == nil {
		// Fallback to basic output if StatusInfo is not available
		h.outputText(result)
		return
	}

	// Use the new table formatting
	output := h.formatTable(result.StatusInfo, result.Duration)
	fmt.Print(output)
}

// formatDuration formats a duration in a human-readable way.
func (h *StatHandler) formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// formatProgressBar creates a text progress bar.
func (h *StatHandler) formatProgressBar(percentage, width int) string {
	filled := (percentage * width) / 100
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return bar
}

// Table formatting constants
const (
	tableWidth     = 61
	contentWidth   = 57 // tableWidth - 4 (for "│ " and " │")
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// TableFormatter handles table formatting with box-drawing characters
type TableFormatter struct {
	useColor   bool
	useUnicode bool
}

// NewTableFormatter creates a new TableFormatter
func NewTableFormatter(useColor, useUnicode bool) *TableFormatter {
	return &TableFormatter{
		useColor:   useColor,
		useUnicode: useUnicode,
	}
}

// formatTable formats the entire status table
func (h *StatHandler) formatTable(info *StatusInfo, duration time.Duration) string {
	// Check if terminal supports colors
	useColor := h.supportsColor()
	formatter := NewTableFormatter(useColor, true)

	var sb strings.Builder

	// Top border
	sb.WriteString(formatter.topBorder())
	sb.WriteString("\n")

	// Title
	sb.WriteString(formatter.formatTitle("Morty 监控大盘"))
	sb.WriteString("\n")

	// Title separator
	sb.WriteString(formatter.sectionSeparator())
	sb.WriteString("\n")

	// Current execution section
	sb.WriteString(formatter.formatSectionHeader("当前执行"))
	sb.WriteString("\n")
	sb.WriteString(h.formatCurrentJobSection(info.Current, formatter))
	sb.WriteString("\n")

	// Previous job section
	if info.Previous != nil {
		sb.WriteString(formatter.sectionSeparator())
		sb.WriteString("\n")
		sb.WriteString(formatter.formatSectionHeader("上一个 Job"))
		sb.WriteString("\n")
		sb.WriteString(h.formatPreviousJobSection(info.Previous, formatter))
		sb.WriteString("\n")
	}

	// Debug issues section
	if len(info.DebugIssues) > 0 {
		sb.WriteString(formatter.sectionSeparator())
		sb.WriteString("\n")
		sb.WriteString(formatter.formatSectionHeader("Debug 问题 (当前 Job)"))
		sb.WriteString("\n")
		sb.WriteString(h.formatDebugIssuesSection(info.DebugIssues, formatter))
		sb.WriteString("\n")
	}

	// Progress section
	sb.WriteString(formatter.sectionSeparator())
	sb.WriteString("\n")
	sb.WriteString(formatter.formatSectionHeader("整体进度"))
	sb.WriteString("\n")
	sb.WriteString(h.formatProgressSection(info.Progress, info.Modules, formatter))
	sb.WriteString("\n")

	// Bottom border
	sb.WriteString(formatter.bottomBorder())
	sb.WriteString("\n")

	// Duration line
	sb.WriteString(formatter.formatDurationLine(duration))
	sb.WriteString("\n")

	return sb.String()
}

// supportsColor checks if the terminal supports ANSI colors
func (h *StatHandler) supportsColor() bool {
	// Check if NO_COLOR is set
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Check if stdout is a terminal
	if fi, err := os.Stdout.Stat(); err == nil {
		return (fi.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

// topBorder returns the top border of the table
func (f *TableFormatter) topBorder() string {
	return "┌" + strings.Repeat("─", tableWidth-2) + "┐"
}

// bottomBorder returns the bottom border of the table
func (f *TableFormatter) bottomBorder() string {
	return "└" + strings.Repeat("─", tableWidth-2) + "┘"
}

// sectionSeparator returns a separator line between sections
func (f *TableFormatter) sectionSeparator() string {
	return "├" + strings.Repeat("─", tableWidth-2) + "┤"
}

// formatTitle formats the table title centered
func (f *TableFormatter) formatTitle(title string) string {
	padding := (contentWidth - len(title)) / 2
	leftPad := strings.Repeat(" ", padding)
	rightPad := strings.Repeat(" ", contentWidth-len(title)-padding)
	line := leftPad + title + rightPad
	if f.useColor {
		line = colorBold + colorCyan + line + colorReset
	}
	return "│ " + line + " │"
}

// formatSectionHeader formats a section header
func (f *TableFormatter) formatSectionHeader(header string) string {
	line := header + strings.Repeat(" ", contentWidth-len(header))
	if f.useColor {
		line = colorBold + line + colorReset
	}
	return "│ " + line + " │"
}

// formatContentLine formats a content line with proper indentation
func (f *TableFormatter) formatContentLine(content string, indent int) string {
	prefix := strings.Repeat("  ", indent)
	fullContent := prefix + content

	// Calculate display width considering CJK characters
	displayWidth := f.displayWidth(fullContent)

	if displayWidth > contentWidth {
		// Truncate to fit
		fullContent = f.truncateToWidth(fullContent, contentWidth-3) + "..."
		displayWidth = contentWidth
	}

	// Pad with spaces to fill the remaining width
	padding := contentWidth - displayWidth
	if padding < 0 {
		padding = 0
	}
	line := fullContent + strings.Repeat(" ", padding)
	return "│ " + line + " │"
}

// displayWidth calculates the display width of a string, accounting for CJK characters
func (f *TableFormatter) displayWidth(s string) int {
	width := 0
	inEscape := false

	for _, r := range s {
		// Skip ANSI escape sequences
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}

		// CJK characters (Chinese, Japanese, Korean) typically occupy 2 cells
		if r >= 0x4E00 && r <= 0x9FFF || // CJK Unified Ideographs
			r >= 0x3400 && r <= 0x4DBF || // CJK Extension A
			r >= 0xF900 && r <= 0xFAFF || // CJK Compatibility Ideographs
			r >= 0x3040 && r <= 0x309F || // Hiragana
			r >= 0x30A0 && r <= 0x30FF || // Katakana
			r >= 0xAC00 && r <= 0xD7AF { // Hangul Syllables
			width += 2
		} else {
			width += 1
		}
	}

	return width
}

// truncateToWidth truncates a string to fit within the specified display width
func (f *TableFormatter) truncateToWidth(s string, maxWidth int) string {
	width := 0
	result := []rune{}
	inEscape := false
	escapeSeq := []rune{}

	for _, r := range s {
		// Handle ANSI escape sequences
		if r == '\033' {
			inEscape = true
			escapeSeq = []rune{r}
			continue
		}
		if inEscape {
			escapeSeq = append(escapeSeq, r)
			if r == 'm' {
				inEscape = false
				result = append(result, escapeSeq...)
				escapeSeq = nil
			}
			continue
		}

		// Calculate character width
		charWidth := 1
		if r >= 0x4E00 && r <= 0x9FFF || r >= 0x3400 && r <= 0x4DBF ||
			r >= 0xF900 && r <= 0xFAFF || r >= 0x3040 && r <= 0x309F ||
			r >= 0x30A0 && r <= 0x30FF || r >= 0xAC00 && r <= 0xD7AF {
			charWidth = 2
		}

		if width+charWidth > maxWidth {
			break
		}

		result = append(result, r)
		width += charWidth
	}

	return string(result)
}

// formatDurationLine formats the duration line outside the table
func (f *TableFormatter) formatDurationLine(duration time.Duration) string {
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	var durationStr string
	if hours > 0 {
		durationStr = fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	} else {
		durationStr = fmt.Sprintf("%02d:%02d", minutes, seconds)
	}

	line := "Duration: " + durationStr
	padding := (tableWidth - len(line)) / 2
	return strings.Repeat(" ", padding) + line
}

// formatCurrentJobSection formats the current job section
func (h *StatHandler) formatCurrentJobSection(current CurrentJobInfo, f *TableFormatter) string {
	var sb strings.Builder

	if current.Module != "" {
		moduleLine := "模块: " + current.Module
		if f.useColor {
			moduleLine = "模块: " + colorBlue + current.Module + colorReset
		}
		sb.WriteString(f.formatContentLine(moduleLine, 1))
		sb.WriteString("\n")

		jobLine := "Job:  " + current.Job
		if f.useColor {
			jobLine = "Job:  " + colorYellow + current.Job + colorReset
		}
		sb.WriteString(f.formatContentLine(jobLine, 1))
		sb.WriteString("\n")

		statusStr := current.Status
		if f.useColor {
			switch statusStr {
			case "COMPLETED":
				statusStr = colorGreen + statusStr + colorReset
			case "RUNNING":
				statusStr = colorCyan + statusStr + colorReset
			case "FAILED":
				statusStr = "\033[31m" + statusStr + colorReset
			default:
				statusStr = colorYellow + statusStr + colorReset
			}
		}
		statusLine := "状态: " + statusStr
		sb.WriteString(f.formatContentLine(statusLine, 1))
		sb.WriteString("\n")

		if !current.StartedAt.IsZero() {
			elapsed := time.Since(current.StartedAt)
			timeLine := "累计时间: " + h.formatDuration(elapsed)
			sb.WriteString(f.formatContentLine(timeLine, 1))
		}
	} else {
		sb.WriteString(f.formatContentLine("无", 1))
	}

	return sb.String()
}

// formatPreviousJobSection formats the previous job section
func (h *StatHandler) formatPreviousJobSection(previous *PreviousJob, f *TableFormatter) string {
	var sb strings.Builder

	jobInfo := previous.Module + "/" + previous.Job
	if f.useColor {
		jobInfo = colorBlue + previous.Module + colorReset + "/" + colorYellow + previous.Job + colorReset
	}

	statusStr := previous.Status
	if f.useColor {
		switch statusStr {
		case "COMPLETED":
			statusStr = colorGreen + statusStr + colorReset
		default:
			statusStr = colorYellow + statusStr + colorReset
		}
	}

	line := jobInfo + ": " + statusStr
	if previous.Duration != "" {
		line += " (耗时 " + previous.Duration + ")"
	}
	sb.WriteString(f.formatContentLine(line, 1))

	return sb.String()
}

// formatDebugIssuesSection formats the debug issues section
func (h *StatHandler) formatDebugIssuesSection(issues []DebugIssue, f *TableFormatter) string {
	var sb strings.Builder

	for _, issue := range issues {
		bullet := "•"
		if f.useColor {
			bullet = colorYellow + "•" + colorReset
		}
		line := bullet + " " + issue.Description
		if issue.Loop > 0 {
			line += fmt.Sprintf(" (loop %d)", issue.Loop)
		}
		sb.WriteString(f.formatContentLine(line, 1))
		sb.WriteString("\n")

		if issue.Hypothesis != "" {
			hypoLine := "猜想: " + issue.Hypothesis
			if f.useColor {
				hypoLine = colorGray + hypoLine + colorReset
			}
			sb.WriteString(f.formatContentLine(hypoLine, 2))
			sb.WriteString("\n")
		}

		if issue.Status != "" {
			statusLine := "状态: " + issue.Status
			if f.useColor {
				statusLine = colorGray + statusLine + colorReset
			}
			sb.WriteString(f.formatContentLine(statusLine, 2))
		}
	}

	return sb.String()
}

// formatProgressSection formats the progress section
func (h *StatHandler) formatProgressSection(progress ProgressInfo, modules []ModuleStatus, f *TableFormatter) string {
	var sb strings.Builder

	// Progress bar - use a narrower bar to fit within content width
	// Content width is 57, with indent of 2 ("  "), we have 55 chars
	// "[xxxxxx] 100% (100/100 Jobs)" needs ~28 chars, so barWidth = 8
	barWidth := 10
	progressBar := h.formatProgressBar(progress.Percentage, barWidth)
	if f.useColor {
		// Colorize the progress bar
		filled := (progress.Percentage * barWidth) / 100
		if filled > barWidth {
			filled = barWidth
		}
		empty := barWidth - filled
		coloredBar := colorGreen + strings.Repeat("█", filled) + colorReset + strings.Repeat("░", empty)
		progressBar = coloredBar
	}

	progressLine := fmt.Sprintf("[%s] %d%% (%d/%d Jobs)",
		progressBar, progress.Percentage,
		progress.CompletedJobs, progress.TotalJobs)
	sb.WriteString(f.formatContentLine(progressLine, 1))
	sb.WriteString("\n")

	// Module status summary
	if len(modules) > 0 {
		sb.WriteString(f.formatContentLine("", 0))
		sb.WriteString("\n")

		// Group modules by status
		var completed, inProgress, pending, failed []ModuleStatus
		for _, mod := range modules {
			switch mod.Status {
			case "completed":
				completed = append(completed, mod)
			case "in_progress":
				inProgress = append(inProgress, mod)
			case "pending":
				pending = append(pending, mod)
			case "failed":
				failed = append(failed, mod)
			}
		}

		// Format each group
		if len(completed) > 0 {
			sb.WriteString(h.formatModuleGroup("已完成", completed, colorGreen, f))
		}
		if len(inProgress) > 0 {
			sb.WriteString(h.formatModuleGroup("进行中", inProgress, colorCyan, f))
		}
		if len(pending) > 0 {
			sb.WriteString(h.formatModuleGroup("待开始", pending, colorGray, f))
		}
		if len(failed) > 0 {
			sb.WriteString(h.formatModuleGroup("失败", failed, "\033[31m", f))
		}
	}

	return sb.String()
}

// formatModuleGroup formats a group of modules with the same status
func (h *StatHandler) formatModuleGroup(label string, modules []ModuleStatus, color string, f *TableFormatter) string {
	var sb strings.Builder

	labelPart := label + ": "
	if f.useColor {
		labelPart = color + label + colorReset + ": "
	}

	var moduleParts []string
	for _, mod := range modules {
		modStr := mod.Name + " (" + fmt.Sprintf("%d/%d", mod.CompletedJobs, mod.TotalJobs) + ")"
		if f.useColor {
			modStr = color + mod.Name + colorReset + " (" + fmt.Sprintf("%d/%d", mod.CompletedJobs, mod.TotalJobs) + ")"
		}
		moduleParts = append(moduleParts, modStr)
	}

	content := labelPart + strings.Join(moduleParts, ", ")
	sb.WriteString(f.formatContentLine(content, 1))
	sb.WriteString("\n")

	return sb.String()
}

// getStatusString returns a status string for the result.
func (h *StatHandler) getStatusString(result *StatResult) string {
	if result.Err != nil {
		return "error"
	}
	if result.CurrentJob != nil {
		return string(result.CurrentJob.Status)
	}
	return "idle"
}

// defaultRefreshInterval is the default interval for watch mode
const defaultRefreshInterval = 60 * time.Second

// clearScreen clears the terminal screen using ANSI escape sequences
func (h *StatHandler) clearScreen() {
	// ANSI escape sequences: clear screen and move cursor to top-left
	fmt.Print("\033[H\033[2J")
	// Alternative for better compatibility: clear screen and scrollback buffer
	fmt.Print("\033[2J\033[3J\033[H")
}

// runWatchMode runs the stat command in watch mode with 60s refresh interval.
func (h *StatHandler) runWatchMode(ctx context.Context, initialResult *StatResult) (*StatResult, error) {
	// Create a signal-aware context for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create a cancellable context
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle signals in a goroutine
	go func() {
		select {
		case <-sigChan:
			h.logger.Info("Watch mode: received interrupt signal, exiting gracefully")
			cancel()
		case <-watchCtx.Done():
		}
	}()

	// Track last refresh time
	lastRefresh := time.Now()

	// Initial screen clear and display
	h.clearScreen()
	h.displayWatchHeader(lastRefresh)

	ticker := time.NewTicker(defaultRefreshInterval)
	defer ticker.Stop()

	// Initial data display
	result := initialResult
	for {
		select {
		case <-watchCtx.Done():
			fmt.Println("\n\n  Watch mode exited.")
			return result, nil
		case <-ticker.C:
			// Update refresh time
			lastRefresh = time.Now()

			// Clear screen for in-place refresh
			h.clearScreen()

			// Display refresh header
			h.displayWatchHeader(lastRefresh)

			// Re-execute stat to collect fresh data (no json output in watch mode)
			freshResult, err := h.Execute(watchCtx, []string{})
			if err != nil {
				h.logger.Error("Watch mode error", logging.String("error", err.Error()))
				// Continue watching even on error
			} else {
				result = freshResult
			}
		}
	}
}

// displayWatchHeader displays the watch mode header with refresh time
func (h *StatHandler) displayWatchHeader(refreshTime time.Time) {
	fmt.Printf("\n  Watch mode enabled. Refresh every 60s. Press Ctrl+C to exit.\n")
	fmt.Printf("  Last refresh: %s\n\n", refreshTime.Format("2006-01-02 15:04:05"))
}
