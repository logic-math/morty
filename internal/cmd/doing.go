// Package cmd provides command handlers for Morty CLI commands.
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/morty/morty/internal/callcli"
	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/logging"
)

// DoingResult represents the result of a doing operation.
type DoingResult struct {
	ModuleName string
	JobName    string
	PlanDir    string
	Err        error
	ExitCode   int
	Duration   time.Duration
	Restart    bool
}

// DoingHandler handles the doing command.
type DoingHandler struct {
	cfg       config.Manager
	logger    logging.Logger
	paths     *config.Paths
	cliCaller callcli.AICliCaller
}

// NewDoingHandler creates a new DoingHandler instance.
func NewDoingHandler(cfg config.Manager, logger logging.Logger) *DoingHandler {
	paths := config.NewPaths()
	// Set workDir from config if available
	if cfg != nil && cfg.GetWorkDir() != "" {
		paths.SetWorkDir(cfg.GetWorkDir())
	}
	return &DoingHandler{
		cfg:       cfg,
		logger:    logger,
		paths:     paths,
		cliCaller: callcli.NewAICliCallerWithLoader(cfg),
	}
}

// Execute executes the doing command.
// It validates plan directory exists and prepares for job execution.
func (h *DoingHandler) Execute(ctx context.Context, args []string) (*DoingResult, error) {
	logger := h.logger.WithContext(ctx)
	startTime := time.Now()

	result := &DoingResult{
		ExitCode: 0,
	}

	// Parse options from args
	restart, moduleName, jobName, remainingArgs := h.parseOptions(args)
	result.Restart = restart

	logger.Info("Starting doing command",
		logging.Bool("restart", restart),
		logging.String("module", moduleName),
		logging.String("job", jobName),
	)

	// Check if --job is used without --module
	if jobName != "" && moduleName == "" {
		result.Err = fmt.Errorf("--job 选项需要配合 --module 使用")
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		logger.Error("Job specified without module", logging.String("job", jobName))
		return result, result.Err
	}

	// Get plan directory
	planDir := h.getPlanDir()
	result.PlanDir = planDir

	// Check if plan directory exists
	if err := h.checkPlanDirExists(); err != nil {
		result.Err = err
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		logger.Error("Plan directory check failed", logging.String("error", err.Error()))
		return result, err
	}

	// Store module and job names
	result.ModuleName = moduleName
	result.JobName = jobName

	// Log remaining args (for future use)
	if len(remainingArgs) > 0 {
		logger.Info("Additional arguments", logging.Any("args", remainingArgs))
	}

	result.Duration = time.Since(startTime)
	result.ExitCode = 0

	logger.Info("Doing command completed",
		logging.String("module", moduleName),
		logging.String("job", jobName),
		logging.Int("exit_code", result.ExitCode),
		logging.Any("duration", result.Duration),
	)

	return result, nil
}

// parseOptions parses command-line options from args.
// Returns (restart flag, module name, job name, remaining args)
func (h *DoingHandler) parseOptions(args []string) (bool, string, string, []string) {
	restart := false
	var moduleName string
	var jobName string
	var remaining []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check for --restart flag
		if arg == "--restart" || arg == "-r" {
			restart = true
			continue
		}

		// Check for --restart=value format
		if strings.HasPrefix(arg, "--restart=") {
			val := strings.TrimPrefix(arg, "--restart=")
			restart = val == "true" || val == "1"
			continue
		}

		// Check for --module or -m
		if arg == "--module" || arg == "-m" {
			if i+1 < len(args) {
				i++
				moduleName = args[i]
			}
			continue
		}

		// Check for --module=value format
		if strings.HasPrefix(arg, "--module=") {
			moduleName = strings.TrimPrefix(arg, "--module=")
			continue
		}

		// Check for --job or -j
		if arg == "--job" || arg == "-j" {
			if i+1 < len(args) {
				i++
				jobName = args[i]
			}
			continue
		}

		// Check for --job=value format
		if strings.HasPrefix(arg, "--job=") {
			jobName = strings.TrimPrefix(arg, "--job=")
			continue
		}

		// Collect remaining args
		remaining = append(remaining, arg)
	}

	return restart, moduleName, jobName, remaining
}

// checkPlanDirExists checks if the plan directory exists.
// Returns a user-friendly error if it doesn't exist.
func (h *DoingHandler) checkPlanDirExists() error {
	planDir := h.getPlanDir()

	// Check if .morty directory exists
	workDir := h.getWorkDir()
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		return fmt.Errorf("请先运行 morty init")
	}

	// Check if plan directory exists
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		return fmt.Errorf("请先运行 morty plan")
	}

	// Check if plan directory is readable
	info, err := os.Stat(planDir)
	if err != nil {
		return fmt.Errorf("无法访问计划目录 %s: %w", planDir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("计划路径 %s 不是一个目录", planDir)
	}

	return nil
}

// getPlanDir returns the plan directory, preferring config if available.
func (h *DoingHandler) getPlanDir() string {
	if h.cfg != nil {
		return h.cfg.GetPlanDir()
	}
	return h.paths.GetPlanDir()
}

// getWorkDir returns the work directory, preferring config if available.
func (h *DoingHandler) getWorkDir() string {
	if h.cfg != nil {
		return h.cfg.GetWorkDir()
	}
	return h.paths.GetWorkDir()
}

// GetPlanDir returns the plan directory path.
func (h *DoingHandler) GetPlanDir() string {
	return h.getPlanDir()
}

// SetPlanDir sets a custom plan directory (useful for testing).
func (h *DoingHandler) SetPlanDir(dir string) {
	h.paths.SetWorkDir(dir)
}

// SetCLICaller sets a custom CLI caller (useful for testing).
func (h *DoingHandler) SetCLICaller(caller callcli.AICliCaller) {
	h.cliCaller = caller
}

// PrintDoingSummary prints a summary of the doing command result.
func (h *DoingHandler) PrintDoingSummary(result *DoingResult) {
	fmt.Println("\n🚀 Doing Command")
	fmt.Println(strings.Repeat("=", 50))

	if result.ModuleName != "" {
		fmt.Printf("📦 Module: %s\n", result.ModuleName)
	}
	if result.JobName != "" {
		fmt.Printf("🔧 Job: %s\n", result.JobName)
	}
	if result.Restart {
		fmt.Println("🔄 Restart mode: enabled")
	}
	fmt.Printf("📁 Plan Directory: %s\n", result.PlanDir)

	if result.Err != nil {
		fmt.Println()
		fmt.Println("❌ Error:")
		fmt.Printf("  %s\n", result.Err)
		fmt.Println(strings.Repeat("=", 50))
		return
	}

	fmt.Println()
	fmt.Println("✅ Ready to execute jobs")
	fmt.Println(strings.Repeat("=", 50))
}
