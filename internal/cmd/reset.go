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
func (h *ResetHandler) formatLoopHistory(entries []LoopHistoryEntry) string {
	if len(entries) == 0 {
		return "未找到 morty 循环提交记录。"
	}

	var sb strings.Builder

	// Header
	sb.WriteString("循环历史 (Loop History):\n")
	sb.WriteString(strings.Repeat("-", 70) + "\n")
	sb.WriteString(fmt.Sprintf("%-6s %-12s %-12s %-20s %-8s %s\n", "LOOP", "MODULE", "JOB", "STATUS", "HASH", "TIME"))
	sb.WriteString(strings.Repeat("-", 70) + "\n")

	// Entries
	for _, entry := range entries {
		timeStr := entry.Timestamp.Format("01-02 15:04")
		module := truncateString(entry.Module, 12)
		job := truncateString(entry.Job, 20)
		sb.WriteString(fmt.Sprintf("%-6d %-12s %-20s %-8s %-8s %s\n",
			entry.LoopNumber,
			module,
			job,
			entry.Status,
			entry.ShortHash,
			timeStr,
		))
	}

	sb.WriteString(strings.Repeat("-", 70))
	return sb.String()
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
