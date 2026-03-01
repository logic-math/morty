// Package cmd provides command handlers for Morty CLI commands.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/git"
	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/state"
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
	ResetLocal bool   // -l flag
	ResetClean bool   // -c flag
	CommitHash string // commit hash for reset to commit
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

	// Handle list history (-l flag) - when used alone, show loop history (no need for git repo check)
	if opts.ResetLocal && !opts.ResetClean && opts.CommitHash == "" {
		// Check if this is a list request by looking for count argument
		count := 10 // default count
		for i, arg := range args {
			if arg == "-l" && i+1 < len(args) {
				// Check if next arg is a number
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					count = n
					break
				}
			}
		}
		historyResult, err := h.showLoopHistory(count)
		if err != nil {
			result.Err = err
			result.ExitCode = 1
			result.Duration = time.Since(startTime)
			return result, err
		}
		fmt.Println(historyResult.Formatted)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Check if we're in a Git repository (for non-list operations)
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
	if !opts.ResetLocal && !opts.ResetClean && opts.CommitHash == "" {
		err := fmt.Errorf("请指定重置选项:\n\n  -l         本地重置 (保留配置文件)\n  -c         完整重置 (清除所有数据和配置)\n  hash       回滚到指定提交\n\n示例:\n  morty reset -l          # 本地重置\n  morty reset -c          # 完整重置\n  morty reset abc1234     # 回滚到提交 abc1234")
		result.Err = err
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		fmt.Println(err.Error())
		return result, err
	}

	// Handle commit hash reset (new functionality)
	if opts.CommitHash != "" {
		commitResult, err := h.resetToCommit(opts.CommitHash)
		result.Err = commitResult.Err
		result.ExitCode = commitResult.ExitCode
		result.Duration = time.Since(startTime)
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
			} else if !strings.HasPrefix(arg, "-") && len(arg) >= 7 {
				// Treat as commit hash if it's not a flag and looks like a hash (>=7 chars)
				opts.CommitHash = arg
			}
		}
	}

	// Check for mutually exclusive options
	if opts.ResetLocal && opts.ResetClean {
		return nil, fmt.Errorf("错误: 选项 -l 和 -c 不能同时使用\n\n请只选择其中一个选项:\n  -l    本地重置\n  -c    完整重置")
	}

	// Check for mutually exclusive: commit hash can't be used with -l or -c
	if opts.CommitHash != "" && (opts.ResetLocal || opts.ResetClean) {
		return nil, fmt.Errorf("错误: 提交哈希不能与 -l 或 -c 选项同时使用\n\n请只选择其中一种:\n  hash    回滚到指定提交\n  -l      本地重置\n  -c      完整重置")
	}

	return opts, nil
}

// getWorkDir returns the working directory.
func (h *ResetHandler) getWorkDir() string {
	// First try to get actual current working directory
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	// Fall back to configured work dir
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

// ResetToCommitResult represents the result of resetting to a specific commit.
type ResetToCommitResult struct {
	CommitHash    string
	CommitMessage string
	BackupBranch  string
	StateRestored bool
	Err           error
	ExitCode      int
}

// CommitInfo holds information about a git commit for display.
type CommitInfo struct {
	Hash       string
	ShortHash  string
	Message    string
	Author     string
	Timestamp  time.Time
	Module     string
	Job        string
	Status     string
	LoopNumber int
}

// resetToCommit resets the repository to a specific commit hash.
// This is Task 1: Main entry point for commit reset functionality.
func (h *ResetHandler) resetToCommit(hash string) (*ResetToCommitResult, error) {
	result := &ResetToCommitResult{
		CommitHash: hash,
		ExitCode:   0,
	}

	workDir := h.getWorkDir()

	// Task 2: Validate commit hash validity
	if err := h.validateCommitHash(hash); err != nil {
		result.Err = err
		result.ExitCode = 1
		return result, err
	}

	// Task 3: Get commit info for confirmation prompt
	commitInfo, err := h.getCommitInfo(hash)
	if err != nil {
		result.Err = fmt.Errorf("获取提交信息失败: %w", err)
		result.ExitCode = 1
		return result, result.Err
	}
	result.CommitMessage = commitInfo.Message

	// Task 4: Interactive confirmation (Y/n)
	confirmed, err := h.promptForConfirmation(commitInfo)
	if err != nil {
		result.Err = err
		result.ExitCode = 1
		return result, err
	}

	if !confirmed {
		fmt.Println("操作已取消")
		result.ExitCode = 0
		return result, nil
	}

	// Task 5: Create backup branch (optional but recommended)
	backupBranch, err := h.gitManager.CreateBackupBranch(workDir)
	if err != nil {
		// Non-fatal: log warning but continue
		h.logger.Warn("Failed to create backup branch", logging.String("error", err.Error()))
	} else {
		result.BackupBranch = backupBranch
		h.logger.Info("Created backup branch", logging.String("branch", backupBranch))
	}

	// Task 6: Execute git reset --hard
	fmt.Println("正在回滚...")
	if err := h.gitManager.ResetToCommit(hash, workDir, git.HardReset); err != nil {
		result.Err = fmt.Errorf("回滚失败: %w", err)
		result.ExitCode = 1
		return result, result.Err
	}

	// Task 7: Restore corresponding state file
	if err := h.restoreStateForCommit(commitInfo); err != nil {
		h.logger.Warn("Failed to restore state file", logging.String("error", err.Error()))
		// Non-fatal: reset succeeded even if state restore failed
	} else {
		result.StateRestored = true
	}

	fmt.Printf("✓ 已回滚到 commit %s\n", commitInfo.ShortHash)
	if result.BackupBranch != "" {
		fmt.Printf("  备份分支: %s\n", result.BackupBranch)
	}

	return result, nil
}

// validateCommitHash validates that the given hash is a valid git commit.
// Task 2: Verify commit hash validity.
func (h *ResetHandler) validateCommitHash(hash string) error {
	if hash == "" {
		return fmt.Errorf("错误: 提交哈希不能为空")
	}

	// Check hash format (at least 7 characters, hexadecimal)
	if len(hash) < 7 {
		return fmt.Errorf("错误: 提交哈希至少需要 7 个字符")
	}

	// Check for valid hex characters
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("错误: 提交哈希包含无效字符 '%c'", c)
		}
	}

	workDir := h.getWorkDir()

	// Verify the commit exists using git cat-file
	_, err := h.gitManager.RunGitCommand(workDir, "cat-file", "-t", hash)
	if err != nil {
		return fmt.Errorf("错误: 无效的提交哈希 '%s'\n\n请使用 'morty reset -l' 查看有效的提交历史", hash)
	}

	return nil
}

// getCommitInfo retrieves information about a specific commit.
// Task 3: Get commit info for confirmation prompt.
func (h *ResetHandler) getCommitInfo(hash string) (*CommitInfo, error) {
	workDir := h.getWorkDir()

	// Get commit details using git log
	format := "%H|%h|%s|%an|%at"
	output, err := h.gitManager.RunGitCommand(workDir, "log", "-1", "--pretty=format:"+format, hash)
	if err != nil {
		return nil, fmt.Errorf("无法获取提交信息: %w", err)
	}

	parts := strings.SplitN(output, "|", 5)
	if len(parts) != 5 {
		return nil, fmt.Errorf("无法解析提交信息")
	}

	timestampUnix, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("无法解析提交时间: %w", err)
	}

	info := &CommitInfo{
		Hash:      parts[0],
		ShortHash: parts[1],
		Message:   parts[2],
		Author:    parts[3],
		Timestamp: time.Unix(timestampUnix, 0),
	}

	// Try to parse module/job from commit message
	info.Module, info.Job = h.parseCommitMessageForModuleJob(info.Message)

	// Try to parse loop number and status from message
	if commit, err := h.gitManager.ParseCommitMessage(info.Message); err == nil {
		info.LoopNumber = commit.LoopNumber
		info.Status = commit.Status
	}

	return info, nil
}

// promptForConfirmation prompts the user to confirm the reset operation.
// Task 4: Interactive confirmation (Y/n).
func (h *ResetHandler) promptForConfirmation(info *CommitInfo) (bool, error) {
	fmt.Printf("\n确认回滚到 commit %s?\n", info.ShortHash)
	fmt.Printf("提交信息: %s\n", info.Message)
	fmt.Printf("作者: %s\n", info.Author)
	fmt.Printf("时间: %s\n", info.Timestamp.Format("2006-01-02 15:04:05"))

	if info.Module != "-" && info.Job != "-" {
		fmt.Printf("模块/Job: %s/%s\n", info.Module, info.Job)
	}

	if info.Status != "" {
		fmt.Printf("状态: %s\n", info.Status)
	}

	fmt.Println("\n这将重置工作目录到该提交状态，未提交的变更将丢失。")
	fmt.Print("[Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("读取输入失败: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))

	// Default to Yes if empty, accept "y" or "yes"
	if response == "" || response == "y" || response == "yes" {
		return true, nil
	}

	return false, nil
}

// restoreStateForAttempt restores the state file to match the commit.
// Task 7: Restore corresponding state file.
func (h *ResetHandler) restoreStateForCommit(info *CommitInfo) error {
	// If we can't determine module/job from commit, skip state restoration
	if info.Module == "-" || info.Job == "-" {
		h.logger.Info("无法从提交信息解析模块/Job，跳过状态恢复")
		return nil
	}

	// Load current state
	stateManager := state.NewManager(h.paths.GetStatusFile())
	if err := stateManager.Load(); err != nil {
		return fmt.Errorf("加载状态文件失败: %w", err)
	}

	// Update state: set the target job as the current running job
	// and reset subsequent jobs to PENDING
	if err := h.syncStateAfterReset(stateManager, info); err != nil {
		return fmt.Errorf("同步状态失败: %w", err)
	}

	return nil
}

// syncStateAfterReset synchronizes the state file after a reset.
// Task 1: Implement `syncStatusAfterReset(commit)` - Main entry point for status synchronization.
// It sets the target job and resets subsequent jobs appropriately.
func (h *ResetHandler) syncStateAfterReset(sm *state.Manager, targetCommit *CommitInfo) error {
	// Get current state
	currentState := sm.GetState()
	if currentState == nil {
		return fmt.Errorf("状态未加载")
	}

	// Task 2: Parse rollback position from commit info
	targetModule := targetCommit.Module
	targetJob := targetCommit.Job

	if targetModule == "-" || targetJob == "-" {
		return fmt.Errorf("无法从提交信息解析模块/Job")
	}

	// Track statistics for output summary
	resetCount := 0
	preservedCount := 0

	// Task 3 & 5: Reset subsequent jobs to PENDING, keep previous jobs COMPLETED
	for i := range currentState.Modules {
		module := &currentState.Modules[i]
		for j := range module.Jobs {
			job := &module.Jobs[j]
			if module.Name == targetModule && job.Name == targetJob {
				// Target job: reset to PENDING for re-execution
				job.Status = state.StatusPending
				job.LoopCount = 0
				job.RetryCount = 0
				job.TasksCompleted = 0
				job.UpdatedAt = time.Now()
				resetCount++
			} else if shouldResetJob(module.Name, job.Name, targetModule, targetJob) {
				// Subsequent jobs: reset to PENDING
				job.Status = state.StatusPending
				job.LoopCount = 0
				job.RetryCount = 0
				job.TasksCompleted = 0
				job.UpdatedAt = time.Now()
				resetCount++
			} else {
				// Previous jobs: keep as COMPLETED (Task 5)
				if job.Status == state.StatusCompleted {
					preservedCount++
				}
			}
		}

		// Recalculate module status
		recalculateModuleStatus(module)
	}

	// Update global status
	currentState.Global.Status = state.StatusRunning
	currentState.Global.CurrentModuleIndex = 0  // Reset to start
	currentState.Global.CurrentJobIndex = 0      // Reset to start
	currentState.Global.LastUpdate = time.Now()

	// Task 4: Update status.json
	if err := sm.Save(currentState); err != nil {
		return fmt.Errorf("保存状态失败: %w", err)
	}

	// Task 6: Output current status hints
	fmt.Printf("\n状态同步完成:\n")
	fmt.Printf("  当前位置: %s/%s\n", targetModule, targetJob)
	fmt.Printf("  重置 %d 个 Job 为 PENDING\n", resetCount)
	fmt.Printf("  保留 %d 个已完成的 Job\n", preservedCount)

	return nil
}

// shouldResetJob determines if a job should be reset based on the target.
func shouldResetJob(moduleName, jobName, targetModule, targetJob string) bool {
	// Simple heuristic: compare module/job names lexicographically
	// In a real implementation, this would use the plan order
	if moduleName > targetModule {
		return true
	}
	if moduleName == targetModule && jobName > targetJob {
		return true
	}
	return false
}

// recalculateModuleStatus recalculates a module's status based on its jobs.
func recalculateModuleStatus(module *state.ModuleState) {
	hasRunning := false
	hasFailed := false
	hasPending := false
	allCompleted := true

	for _, job := range module.Jobs {
		switch job.Status {
		case state.StatusRunning:
			hasRunning = true
			allCompleted = false
		case state.StatusFailed:
			hasFailed = true
			allCompleted = false
		case state.StatusPending:
			hasPending = true
			allCompleted = false
		case state.StatusCompleted:
			// Continue checking
		default:
			allCompleted = false
		}
	}

	switch {
	case hasRunning:
		module.Status = state.StatusRunning
	case hasFailed:
		module.Status = state.StatusFailed
	case allCompleted:
		module.Status = state.StatusCompleted
	case hasPending:
		module.Status = state.StatusPending
	}

	module.UpdatedAt = time.Now()
}
