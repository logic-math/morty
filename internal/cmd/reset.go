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
