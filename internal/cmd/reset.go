// Package cmd provides command handlers for Morty CLI commands.
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/git"
	"github.com/morty/morty/internal/logging"
)

// ResetResult represents the result of a reset operation.
type ResetResult struct {
	ResetLevel string
	Err        error
	ExitCode   int
	Duration   time.Duration
}

// ResetHandler handles the reset command.
type ResetHandler struct {
	cfg        config.Manager
	logger     logging.Logger
	paths      *config.Paths
	gitChecker GitChecker
	gitManager *git.Manager
}

// GitChecker defines the interface for Git repository checking.
type GitChecker interface {
	IsGitRepo(path string) bool
	GetRepoRoot(path string) (string, error)
}

// defaultGitChecker is the default implementation of GitChecker.
type defaultGitChecker struct{}

// IsGitRepo checks if the given path is a Git repository.
func (d *defaultGitChecker) IsGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetRepoRoot returns the root path of the Git repository.
func (d *defaultGitChecker) GetRepoRoot(path string) (string, error) {
	return git.GetRepoRoot(path)
}

// NewResetHandler creates a new ResetHandler instance.
func NewResetHandler(cfg config.Manager, logger logging.Logger) *ResetHandler {
	return &ResetHandler{
		cfg:        cfg,
		logger:     logger,
		paths:      config.NewPaths(),
		gitChecker: &defaultGitChecker{},
		gitManager: git.NewManager(),
	}
}

// SetGitChecker sets the Git checker for testing purposes.
func (h *ResetHandler) SetGitChecker(checker GitChecker) {
	h.gitChecker = checker
}

// ResetOptions holds the parsed command options.
type ResetOptions struct {
	ResetLocal bool // -l flag
	ResetClean bool // -c flag
}

// Execute executes the reset command.
func (h *ResetHandler) Execute(ctx context.Context, args []string) (*ResetResult, error) {
	logger := h.logger.WithContext(ctx)
	startTime := time.Now()

	result := &ResetResult{
		ExitCode: 0,
	}

	// Parse options
	opts, err := h.parseOptions(args)
	if err != nil {
		result.Err = err
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		fmt.Println(err.Error())
		return result, err
	}

	// Check if we're in a Git repository
	workDir := h.getWorkDir()
	if !h.gitChecker.IsGitRepo(workDir) {
		err := fmt.Errorf("错误: 当前目录不是 Git 仓库\n\n请在 Git 仓库目录下运行此命令")
		result.Err = err
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		fmt.Println(err.Error())
		return result, err
	}

	// No options provided - show friendly help
	if !opts.ResetLocal && !opts.ResetClean {
		err := fmt.Errorf("请指定重置选项:\n\n  -l    本地重置 (保留配置文件)\n  -c    完整重置 (清除所有数据和配置)\n\n示例:\n  morty reset -l    # 本地重置\n  morty reset -c    # 完整重置")
		result.Err = err
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		fmt.Println(err.Error())
		return result, err
	}

	// Perform the reset
	if opts.ResetLocal {
		result.ResetLevel = "local"
		logger.Info("Performing local reset")
		if err := h.performLocalReset(); err != nil {
			result.Err = err
			result.ExitCode = 1
			result.Duration = time.Since(startTime)
			logger.Error("Local reset failed", logging.String("error", err.Error()))
			return result, err
		}
		fmt.Println("本地重置完成")
	} else if opts.ResetClean {
		result.ResetLevel = "clean"
		logger.Info("Performing clean reset")
		if err := h.performCleanReset(); err != nil {
			result.Err = err
			result.ExitCode = 1
			result.Duration = time.Since(startTime)
			logger.Error("Clean reset failed", logging.String("error", err.Error()))
			return result, err
		}
		fmt.Println("完整重置完成")
	}

	result.Duration = time.Since(startTime)
	logger.Info("Reset completed", logging.String("level", result.ResetLevel))
	return result, nil
}

// parseOptions parses command line options.
// Returns error if -l and -c are both specified.
func (h *ResetHandler) parseOptions(args []string) (*ResetOptions, error) {
	opts := &ResetOptions{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "-l":
			opts.ResetLocal = true
		case "-c":
			opts.ResetClean = true
		default:
			// Handle --flag=value format
			if strings.HasPrefix(arg, "-l=") {
				val := strings.TrimPrefix(arg, "-l=")
				opts.ResetLocal = val == "true" || val == "1"
			} else if strings.HasPrefix(arg, "-c=") {
				val := strings.TrimPrefix(arg, "-c=")
				opts.ResetClean = val == "true" || val == "1"
			}
		}
	}

	// Check for mutually exclusive options
	if opts.ResetLocal && opts.ResetClean {
		return nil, fmt.Errorf("错误: 选项 -l 和 -c 不能同时使用\n\n请只选择其中一个选项:\n  -l    本地重置\n  -c    完整重置")
	}

	return opts, nil
}

// getWorkDir returns the working directory.
func (h *ResetHandler) getWorkDir() string {
	if h.cfg != nil {
		return h.cfg.GetWorkDir()
	}
	return h.paths.GetWorkDir()
}

// performLocalReset performs a local reset (keeps configuration).
func (h *ResetHandler) performLocalReset() error {
	// Remove state and logs but keep config
	workDir := h.getWorkDir()
	mortyDir := filepath.Join(workDir, ".morty")

	// Remove status.json
	statusFile := filepath.Join(mortyDir, "status.json")
	if err := os.RemoveAll(statusFile); err != nil {
		return fmt.Errorf("failed to remove status file: %w", err)
	}

	// Remove doing logs
	doingDir := filepath.Join(mortyDir, "doing")
	if err := os.RemoveAll(doingDir); err != nil {
		return fmt.Errorf("failed to remove doing directory: %w", err)
	}

	return nil
}

// performCleanReset performs a clean reset (removes everything).
func (h *ResetHandler) performCleanReset() error {
	// Remove entire .morty directory
	workDir := h.getWorkDir()
	mortyDir := filepath.Join(workDir, ".morty")

	if err := os.RemoveAll(mortyDir); err != nil {
		return fmt.Errorf("failed to remove morty directory: %w", err)
	}

	return nil
}

// LoopHistoryEntry represents a single loop history entry with parsed information.
type LoopHistoryEntry struct {
	LoopNumber int
	Status     string
	Module     string
	Job        string
	CommitHash string
	ShortHash  string
	Author     string
	Timestamp  time.Time
	Message    string
}

// ShowLoopHistoryResult represents the result of showing loop history.
type ShowLoopHistoryResult struct {
	History    []LoopHistoryEntry
	Formatted  string
	Err        error
	ExitCode   int
}

// showLoopHistory retrieves and displays the loop commit history.
// It returns the history entries and a formatted string for display.
// The count parameter limits the number of entries (default 10).
func (h *ResetHandler) showLoopHistory(count int) (*ShowLoopHistoryResult, error) {
	result := &ShowLoopHistoryResult{
		ExitCode: 0,
	}

	// Use default count if not specified or invalid
	if count <= 0 {
		count = 10
	}

	// Get working directory
	workDir := h.getWorkDir()

	// Check if we're in a git repository
	if !h.gitChecker.IsGitRepo(workDir) {
		err := fmt.Errorf("当前目录不是 Git 仓库，无法获取循环历史")
		result.Err = err
		result.ExitCode = 1
		return result, err
	}

	// Get loop history from git
	loopCommits, err := h.gitManager.ShowLoopHistory(count, workDir)
	if err != nil {
		err := fmt.Errorf("获取循环历史失败: %w", err)
		result.Err = err
		result.ExitCode = 1
		return result, err
	}

	// Handle no loop commits case
	if len(loopCommits) == 0 {
		result.Formatted = "未找到 morty 循环提交记录。\n\n提示: 循环提交是由 morty doing 命令自动创建的，\n格式为 'morty: loop N - STATUS'"
		return result, nil
	}

	// Convert LoopCommit to LoopHistoryEntry and parse module/job info
	entries := make([]LoopHistoryEntry, 0, len(loopCommits))
	for _, commit := range loopCommits {
		entry := LoopHistoryEntry{
			LoopNumber: commit.LoopNumber,
			Status:     commit.Status,
			CommitHash: commit.CommitHash,
			ShortHash:  commit.ShortHash,
			Author:     commit.Author,
			Timestamp:  commit.Timestamp,
			Message:    commit.Message,
		}

		// Parse module and job from commit message
		module, job := h.parseCommitMessageForModuleJob(commit.Message)
		entry.Module = module
		entry.Job = job

		entries = append(entries, entry)
	}

	result.History = entries
	result.Formatted = h.formatLoopHistory(entries)

	return result, nil
}

// parseCommitMessageForModuleJob parses the commit message to extract module and job info.
// Expected format variations:
//   - "morty: loop N - STATUS"
//   - "morty: module/job_name - STATUS"
//   - "morty: loop N - module/job_name - STATUS"
func (h *ResetHandler) parseCommitMessageForModuleJob(message string) (module, job string) {
	// Try to extract module/job from patterns like "module/job_name"
	// This is a simple parser that looks for patterns in the message
	parts := strings.Split(message, " - ")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Check for module/job pattern (contains /)
		if strings.Contains(part, "/") && !strings.Contains(part, " ") {
			subParts := strings.SplitN(part, "/", 2)
			if len(subParts) == 2 {
				return subParts[0], subParts[1]
			}
		}
	}

	// Default values if no module/job found
	return "-", "-"
}

// formatLoopHistory formats the loop history entries for display.
// This is a legacy method kept for backward compatibility.
// Use formatHistoryTable() for new code.
func (h *ResetHandler) formatLoopHistory(entries []LoopHistoryEntry) string {
	return h.formatHistoryTable(entries)
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// HistoryTableFormatter handles history table formatting with box-drawing characters
type HistoryTableFormatter struct {
	useColor bool
}

// NewHistoryTableFormatter creates a new HistoryTableFormatter
func NewHistoryTableFormatter(useColor bool) *HistoryTableFormatter {
	return &HistoryTableFormatter{
		useColor: useColor,
	}
}

// formatHistoryTable formats the loop history entries as a table with adaptive column widths.
// This is the main entry point for table formatting.
func (h *ResetHandler) formatHistoryTable(entries []LoopHistoryEntry) string {
	if len(entries) == 0 {
		return "未找到 morty 循环提交记录。"
	}

	formatter := NewHistoryTableFormatter(h.supportsColor())
	return formatter.formatTable(entries)
}

// supportsColor checks if the terminal supports ANSI colors
func (h *ResetHandler) supportsColor() bool {
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

// formatTable formats the history entries as a table
func (f *HistoryTableFormatter) formatTable(entries []LoopHistoryEntry) string {
	var sb strings.Builder

	// Calculate column widths based on content
	colWidths := f.calculateColumnWidths(entries)

	// Build table
	totalWidth := colWidths.CommitID + colWidths.Module + colWidths.Job + colWidths.Status + colWidths.Hash + colWidths.Time + 25 // borders and spacing

	// Top border
	sb.WriteString(f.topBorder(totalWidth))
	sb.WriteString("\n")

	// Header row
	sb.WriteString(f.formatHeader(colWidths))
	sb.WriteString("\n")

	// Separator
	sb.WriteString(f.separator(colWidths))
	sb.WriteString("\n")

	// Data rows
	for i, entry := range entries {
		sb.WriteString(f.formatRow(entry, colWidths))
		if i < len(entries)-1 {
			sb.WriteString("\n")
		}
	}

	// Bottom border
	sb.WriteString("\n")
	sb.WriteString(f.bottomBorder(totalWidth))

	return sb.String()
}

// ColumnWidths holds the calculated widths for each column
type ColumnWidths struct {
	CommitID int
	Module   int
	Job      int
	Status   int
	Hash     int
	Time     int
}

// calculateColumnWidths calculates adaptive column widths based on content
func (f *HistoryTableFormatter) calculateColumnWidths(entries []LoopHistoryEntry) ColumnWidths {
	// Minimum widths for headers
	widths := ColumnWidths{
		CommitID: 8,  // "CommitID"
		Module:   6,  // "Module"
		Job:      3,  // "Job"
		Status:   6,  // "Status"
		Hash:     8,  // "Hash"
		Time:     19, // "Time" (YYYY-MM-DD HH:MM:SS)
	}

	// Calculate maximum widths based on content
	for _, entry := range entries {
		// Short hash is always 7 chars + padding
		if len(entry.ShortHash) > widths.CommitID {
			widths.CommitID = len(entry.ShortHash)
		}

		// Module name
		if len(entry.Module) > widths.Module {
			widths.Module = len(entry.Module)
		}

		// Job name
		if len(entry.Job) > widths.Job {
			widths.Job = len(entry.Job)
		}

		// Status
		if len(entry.Status) > widths.Status {
			widths.Status = len(entry.Status)
		}

		// Hash (same as CommitID)
		if len(entry.ShortHash) > widths.Hash {
			widths.Hash = len(entry.ShortHash)
		}
	}

	// Ensure minimum practical widths
	if widths.Module < 10 {
		widths.Module = 10
	}
	if widths.Job < 12 {
		widths.Job = 12
	}
	if widths.Status < 8 {
		widths.Status = 8
	}

	return widths
}

// formatShortHash formats a commit hash as a 7-character short hash
func (f *HistoryTableFormatter) formatShortHash(hash string) string {
	if len(hash) <= 7 {
		return hash
	}
	return hash[:7]
}

// formatTime formats the timestamp as YYYY-MM-DD HH:MM:SS
func (f *HistoryTableFormatter) formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// colorize applies color to text if color support is enabled
func (f *HistoryTableFormatter) colorize(text string, colorCode string) string {
	if !f.useColor {
		return text
	}
	return colorCode + text + "\033[0m"
}

// getStatusColor returns the color code for a status
func (f *HistoryTableFormatter) getStatusColor(status string) string {
	if !f.useColor {
		return ""
	}
	switch status {
	case "COMPLETED":
		return "\033[32m" // Green
	case "FAILED":
		return "\033[31m" // Red
	case "RUNNING":
		return "\033[36m" // Cyan
	case "PENDING":
		return "\033[33m" // Yellow
	default:
		return "\033[90m" // Gray
	}
}

// formatStatus formats the status with color
func (f *HistoryTableFormatter) formatStatus(status string) string {
	colorCode := f.getStatusColor(status)
	return f.colorize(status, colorCode)
}

// topBorder returns the top border of the table
func (f *HistoryTableFormatter) topBorder(width int) string {
	return "┌" + strings.Repeat("─", width-2) + "┐"
}

// bottomBorder returns the bottom border of the table
func (f *HistoryTableFormatter) bottomBorder(width int) string {
	return "└" + strings.Repeat("─", width-2) + "┘"
}

// separator returns the separator line between header and data
func (f *HistoryTableFormatter) separator(widths ColumnWidths) string {
	var parts []string
	parts = append(parts, strings.Repeat("─", widths.CommitID+2))
	parts = append(parts, strings.Repeat("─", widths.Module+2))
	parts = append(parts, strings.Repeat("─", widths.Job+2))
	parts = append(parts, strings.Repeat("─", widths.Status+2))
	parts = append(parts, strings.Repeat("─", widths.Hash+2))
	parts = append(parts, strings.Repeat("─", widths.Time+2))
	return "├" + strings.Join(parts, "┼") + "┤"
}

// formatHeader formats the header row
func (f *HistoryTableFormatter) formatHeader(widths ColumnWidths) string {
	headers := []struct {
		name  string
		width int
	}{
		{"CommitID", widths.CommitID},
		{"Module", widths.Module},
		{"Job", widths.Job},
		{"Status", widths.Status},
		{"Hash", widths.Hash},
		{"Time", widths.Time},
	}

	var parts []string
	for _, h := range headers {
		header := f.padCenter(h.name, h.width)
		if f.useColor {
			header = f.colorize(header, "\033[1m\033[37m") // Bold white
		}
		parts = append(parts, " "+header+" ")
	}

	return "│" + strings.Join(parts, "│") + "│"
}

// formatRow formats a single data row
func (f *HistoryTableFormatter) formatRow(entry LoopHistoryEntry, widths ColumnWidths) string {
	shortHash := f.formatShortHash(entry.ShortHash)
	module := f.padRight(entry.Module, widths.Module)
	job := f.padRight(entry.Job, widths.Job)
	hash := f.padCenter(shortHash, widths.Hash)
	timeStr := f.padCenter(f.formatTime(entry.Timestamp), widths.Time)

	// For alignment without color codes in the padding calculation, we need special handling
	statusPadded := f.padRight(entry.Status, widths.Status)

	var parts []string
	parts = append(parts, " "+f.padCenter(shortHash, widths.CommitID)+" ")
	parts = append(parts, " "+module+" ")
	parts = append(parts, " "+job+" ")
	parts = append(parts, " "+f.colorize(statusPadded, f.getStatusColor(entry.Status))+" ")
	parts = append(parts, " "+hash+" ")
	parts = append(parts, " "+timeStr+" ")

	return "│" + strings.Join(parts, "│") + "│"
}

// padRight pads a string to the right to reach the specified width
func (f *HistoryTableFormatter) padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// padCenter pads a string to center it within the specified width
func (f *HistoryTableFormatter) padCenter(s string, width int) string {
	if len(s) >= width {
		return s
	}
	totalPad := width - len(s)
	leftPad := totalPad / 2
	rightPad := totalPad - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}
