// Package cmd provides command handlers for Morty CLI commands.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Description string    `json:"description"`
	Status      string    `json:"status"`
	LoopCount   int       `json:"loop_count"`
	StartedAt   time.Time `json:"started_at"`
}

// PreviousJob represents information about the previous completed job.
type PreviousJob struct {
	Module      string        `json:"module"`
	Job         string        `json:"job"`
	Status      string        `json:"status"`
	Duration    time.Duration `json:"duration"`
	CompletedAt time.Time     `json:"completed_at"`
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

	// Load state
	stateManager := state.NewManager(statusFile)
	if err := stateManager.Load(); err != nil {
		logger.Error("Failed to load state", logging.String("error", err.Error()))
		result.Err = fmt.Errorf("failed to load state: %w", err)
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		return result, result.Err
	}

	// Get current job
	currentJob, err := stateManager.GetCurrent()
	if err != nil {
		logger.Warn("Failed to get current job", logging.String("error", err.Error()))
	}
	result.CurrentJob = currentJob

	// Get summary
	summary, err := stateManager.GetSummary()
	if err != nil {
		logger.Warn("Failed to get summary", logging.String("error", err.Error()))
	} else {
		result.Summary = summary
	}

	// Collect comprehensive status info
	statusInfo, err := h.collectStatus(stateManager)
	if err != nil {
		logger.Warn("Failed to collect status info", logging.String("error", err.Error()))
	} else {
		result.StatusInfo = statusInfo
	}

	result.Duration = time.Since(startTime)

	// Output results
	if jsonOutput {
		h.outputJSON(result)
	} else {
		h.outputText(result)
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
						mostRecent.Duration = job.CompletedAt.Sub(job.StartedAt)
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

// outputJSON outputs the result in JSON format.
func (h *StatHandler) outputJSON(result *StatResult) {
	// Use StatusInfo if available for enhanced output
	if result.StatusInfo != nil {
		output := struct {
			Status     string      `json:"status"`
			Current    CurrentJobInfo `json:"current"`
			Previous   *PreviousJob   `json:"previous,omitempty"`
			Progress   ProgressInfo   `json:"progress"`
			Modules    []ModuleStatus `json:"modules"`
			DebugIssues []DebugIssue  `json:"debug_issues"`
			Duration   string        `json:"duration"`
			Error      string        `json:"error,omitempty"`
		}{
			Status:      h.getStatusString(result),
			Current:     result.StatusInfo.Current,
			Previous:    result.StatusInfo.Previous,
			Progress:    result.StatusInfo.Progress,
			Modules:     result.StatusInfo.Modules,
			DebugIssues: result.StatusInfo.DebugIssues,
			Duration:    result.Duration.String(),
		}

		if result.Err != nil {
			output.Error = result.Err.Error()
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.Encode(output)
		return
	}

	// Fallback to basic output
	output := struct {
		Status     string             `json:"status"`
		CurrentJob *state.CurrentJob  `json:"current_job,omitempty"`
		Summary    *state.Summary     `json:"summary,omitempty"`
		Duration   string             `json:"duration"`
		Error      string             `json:"error,omitempty"`
	}{
		Status:     h.getStatusString(result),
		CurrentJob: result.CurrentJob,
		Summary:    result.Summary,
		Duration:   result.Duration.String(),
	}

	if result.Err != nil {
		output.Error = result.Err.Error()
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(output)
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
	info := result.StatusInfo

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("  Morty 监控大盘")
	fmt.Println("=" + strings.Repeat("=", 60))

	// Current job section
	fmt.Println()
	fmt.Println("  当前执行")
	if info.Current.Module != "" {
		fmt.Printf("    模块: %s\n", info.Current.Module)
		fmt.Printf("    Job:  %s\n", info.Current.Job)
		fmt.Printf("    状态: %s", info.Current.Status)
		if info.Current.LoopCount > 0 {
			fmt.Printf(" (第%d次循环)", info.Current.LoopCount)
		}
		fmt.Println()
		if !info.Current.StartedAt.IsZero() {
			elapsed := time.Since(info.Current.StartedAt)
			fmt.Printf("    累计时间: %s\n", h.formatDuration(elapsed))
		}
	} else {
		fmt.Println("    无")
	}

	// Previous job section
	if info.Previous != nil {
		fmt.Println()
		fmt.Println("  上一个 Job")
		fmt.Printf("    %s/%s: %s", info.Previous.Module, info.Previous.Job, info.Previous.Status)
		if info.Previous.Duration > 0 {
			fmt.Printf(" (耗时 %s)", h.formatDuration(info.Previous.Duration))
		}
		fmt.Println()
	}

	// Debug issues section
	if len(info.DebugIssues) > 0 {
		fmt.Println()
		fmt.Println("  Debug 问题 (当前 Job)")
		for _, issue := range info.DebugIssues {
			fmt.Printf("    • %s (loop %d)\n", issue.Description, issue.Loop)
			if issue.Hypothesis != "" {
				fmt.Printf("      猜想: %s\n", issue.Hypothesis)
			}
			if issue.Status != "" {
				fmt.Printf("      状态: %s\n", issue.Status)
			}
		}
	}

	// Progress section
	fmt.Println()
	fmt.Println("  整体进度")
	progressBar := h.formatProgressBar(info.Progress.Percentage, 40)
	fmt.Printf("    [%s] %d%% (%d/%d Jobs)\n",
		progressBar, info.Progress.Percentage,
		info.Progress.CompletedJobs, info.Progress.TotalJobs)

	// Module status
	if len(info.Modules) > 0 {
		fmt.Println()
		fmt.Println("  模块状态:")
		for _, mod := range info.Modules {
			statusStr := ""
			switch mod.Status {
			case "completed":
				statusStr = "已完成"
			case "in_progress":
				statusStr = "进行中"
			case "pending":
				statusStr = "待开始"
			case "failed":
				statusStr = "失败"
			default:
				statusStr = mod.Status
			}
			fmt.Printf("    %s: %s (%d/%d)\n", mod.Name, statusStr, mod.CompletedJobs, mod.TotalJobs)
		}
	}

	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Printf("  Duration: %s\n", result.Duration)
	fmt.Println("=" + strings.Repeat("=", 60))
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

// runWatchMode runs the stat command in watch mode.
func (h *StatHandler) runWatchMode(ctx context.Context, initialResult *StatResult) (*StatResult, error) {
	fmt.Println("\n  Watch mode enabled. Press Ctrl+C to exit.")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return initialResult, ctx.Err()
		case <-ticker.C:
			// Clear screen (ANSI escape sequence)
			fmt.Print("\033[H\033[2J")

			// Re-execute stat
			result, err := h.Execute(ctx, []string{})
			if err != nil {
				h.logger.Error("Watch mode error", logging.String("error", err.Error()))
			}
			initialResult = result
		}
	}
}
