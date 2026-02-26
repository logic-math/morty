// Package cmd provides command handlers for Morty CLI commands.
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/morty/morty/internal/callcli"
	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/parser/plan"
	"github.com/morty/morty/internal/state"
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
	cfg          config.Manager
	logger       logging.Logger
	paths        *config.Paths
	cliCaller    callcli.AICliCaller
	stateManager *state.Manager
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

	// Step 1: Load status
	if err := h.loadStatus(); err != nil {
		result.Err = fmt.Errorf("加载状态失败: %w", err)
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		logger.Error("Failed to load status", logging.String("error", err.Error()))
		return result, result.Err
	}

	// Step 2: Handle --restart flag
	if restart {
		if err := h.handleRestart(moduleName, jobName); err != nil {
			result.Err = fmt.Errorf("重置状态失败: %w", err)
			result.ExitCode = 1
			result.Duration = time.Since(startTime)
			logger.Error("Failed to handle restart", logging.String("error", err.Error()))
			return result, result.Err
		}
		logger.Info("State reset completed",
			logging.String("module", moduleName),
			logging.String("job", jobName),
		)
	}

	// Step 3: Select target job
	targetModule, targetJob, err := h.selectTargetJob(moduleName, jobName)
	if err != nil {
		result.Err = err
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		logger.Error("Failed to select target job", logging.String("error", err.Error()))
		return result, result.Err
	}

	result.ModuleName = targetModule
	result.JobName = targetJob

	logger.Info("Target job selected",
		logging.String("module", targetModule),
		logging.String("job", targetJob),
	)

	// Step 4: Check prerequisites
	if err := h.checkPrerequisites(targetModule, targetJob); err != nil {
		result.Err = err
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		logger.Error("Prerequisites check failed", logging.String("error", err.Error()))
		return result, result.Err
	}

	// Log remaining args (for future use)
	if len(remainingArgs) > 0 {
		logger.Info("Additional arguments", logging.Any("args", remainingArgs))
	}

	result.Duration = time.Since(startTime)
	result.ExitCode = 0

	logger.Info("Doing command completed",
		logging.String("module", targetModule),
		logging.String("job", targetJob),
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

// loadStatus loads the state from the status file.
// Task 1: Implement loadStatus() to load state from file
func (h *DoingHandler) loadStatus() error {
	statusFile := h.getStatusFilePath()
	h.stateManager = state.NewManager(statusFile)

	if err := h.stateManager.Load(); err != nil {
		return fmt.Errorf("failed to load state from %s: %w", statusFile, err)
	}

	return nil
}

// getStatusFilePath returns the path to the status file.
func (h *DoingHandler) getStatusFilePath() string {
	if h.cfg != nil {
		return h.cfg.GetStatusFile()
	}
	return filepath.Join(h.getWorkDir(), "status.json")
}

// handleRestart resets the state for the specified range.
// Task 2: Implement --restart status reset logic
// - If no module specified: reset all jobs to PENDING
// - If module specified but no job: reset all jobs in that module to PENDING
// - If both module and job specified: reset only that job to PENDING
func (h *DoingHandler) handleRestart(moduleName, jobName string) error {
	if h.stateManager == nil {
		return fmt.Errorf("state manager not initialized")
	}

	stateData := h.stateManager.GetState()
	if stateData == nil {
		return fmt.Errorf("state not loaded")
	}

	now := time.Now()

	// Case 1: No module specified - reset all jobs
	if moduleName == "" {
		for _, module := range stateData.Modules {
			for _, job := range module.Jobs {
				job.Status = state.StatusPending
				job.LoopCount = 0
				job.RetryCount = 0
				job.TasksCompleted = 0
				job.UpdatedAt = now
			}
			module.Status = state.StatusPending
			module.UpdatedAt = now
		}
		stateData.Global.Status = state.StatusPending
		stateData.Global.CurrentModule = ""
		stateData.Global.CurrentJob = ""
		stateData.Global.LastUpdate = now
		return h.stateManager.Save()
	}

	// Case 2: Module specified
	module, ok := stateData.Modules[moduleName]
	if !ok {
		// Module doesn't exist yet, nothing to reset
		return nil
	}

	// Case 2a: Only module specified - reset all jobs in this module
	if jobName == "" {
		for _, job := range module.Jobs {
			job.Status = state.StatusPending
			job.LoopCount = 0
			job.RetryCount = 0
			job.TasksCompleted = 0
			job.UpdatedAt = now
		}
		module.Status = state.StatusPending
		module.UpdatedAt = now
	} else {
		// Case 2b: Both module and job specified - reset only this job
		job, ok := module.Jobs[jobName]
		if !ok {
			// Job doesn't exist yet, nothing to reset
			return nil
		}
		job.Status = state.StatusPending
		job.LoopCount = 0
		job.RetryCount = 0
		job.TasksCompleted = 0
		job.UpdatedAt = now

		// Recalculate module status
		module.Status = h.calculateModuleStatus(module)
		module.UpdatedAt = now
	}

	stateData.Global.LastUpdate = now
	return h.stateManager.Save()
}

// calculateModuleStatus calculates the overall module status based on its jobs.
func (h *DoingHandler) calculateModuleStatus(module *state.ModuleState) state.Status {
	if len(module.Jobs) == 0 {
		return state.StatusPending
	}

	hasRunning := false
	hasFailed := false
	hasBlocked := false
	allCompleted := true

	for _, job := range module.Jobs {
		switch job.Status {
		case state.StatusRunning:
			hasRunning = true
			allCompleted = false
		case state.StatusFailed:
			hasFailed = true
			allCompleted = false
		case state.StatusBlocked:
			hasBlocked = true
			allCompleted = false
		case state.StatusPending:
			allCompleted = false
		case state.StatusCompleted:
			// Continue checking
		}
	}

	if allCompleted {
		return state.StatusCompleted
	}
	if hasRunning {
		return state.StatusRunning
	}
	if hasFailed {
		return state.StatusFailed
	}
	if hasBlocked {
		return state.StatusBlocked
	}
	return state.StatusPending
}

// selectTargetJob selects the target job to execute.
// Task 3: Implement selectTargetJob() to select target Job
// - If no params: find first PENDING job across all modules
// - If module specified: find first PENDING job in that module
// - If both module and job specified: use the specified job
func (h *DoingHandler) selectTargetJob(moduleName, jobName string) (string, string, error) {
	if h.stateManager == nil {
		return "", "", fmt.Errorf("state manager not initialized")
	}

	stateData := h.stateManager.GetState()
	if stateData == nil {
		return "", "", fmt.Errorf("state not loaded")
	}

	// Case 1: Both module and job specified - use them directly
	if moduleName != "" && jobName != "" {
		// Validate that the module and job exist
		module, ok := stateData.Modules[moduleName]
		if !ok {
			return "", "", fmt.Errorf("模块不存在: %s", moduleName)
		}
		if _, ok := module.Jobs[jobName]; !ok {
			return "", "", fmt.Errorf("Job 不存在: %s/%s", moduleName, jobName)
		}
		return moduleName, jobName, nil
	}

	// Case 2: Only module specified - find first PENDING job in that module
	if moduleName != "" {
		module, ok := stateData.Modules[moduleName]
		if !ok {
			return "", "", fmt.Errorf("模块不存在: %s", moduleName)
		}

		// Find first PENDING job in the module
		for jobName, job := range module.Jobs {
			if job.Status == state.StatusPending {
				return moduleName, jobName, nil
			}
		}

		return "", "", fmt.Errorf("模块 %s 没有待执行的 Job", moduleName)
	}

	// Case 3: No params - find first PENDING job across all modules
	for moduleName, module := range stateData.Modules {
		for jobName, job := range module.Jobs {
			if job.Status == state.StatusPending {
				return moduleName, jobName, nil
			}
		}
	}

	return "", "", fmt.Errorf("没有待执行的 Job")
}

// checkPrerequisites checks if all prerequisite jobs are completed.
// Task 4: Implement prerequisite checking
// It reads the plan file for the module and checks if all jobs listed
// in the job's Prerequisites are in COMPLETED status.
func (h *DoingHandler) checkPrerequisites(moduleName, jobName string) error {
	// Load the plan file for this module
	planFile := filepath.Join(h.getPlanDir(), moduleName+".md")
	content, err := os.ReadFile(planFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("计划文件不存在: %s", planFile)
		}
		return fmt.Errorf("读取计划文件失败: %w", err)
	}

	// Parse the plan file
	planData, err := plan.ParsePlan(string(content))
	if err != nil {
		return fmt.Errorf("解析计划文件失败: %w", err)
	}

	// Find the job definition
	var targetJob *plan.Job
	for i := range planData.Jobs {
		if planData.Jobs[i].Name == jobName {
			targetJob = &planData.Jobs[i]
			break
		}
	}

	if targetJob == nil {
		// Job not found in plan, but this might be a dynamic job
		// Return success to allow execution
		return nil
	}

	// Check prerequisites
	if len(targetJob.Prerequisites) == 0 {
		return nil
	}

	stateData := h.stateManager.GetState()
	module, ok := stateData.Modules[moduleName]
	if !ok {
		return fmt.Errorf("模块不存在: %s", moduleName)
	}

	var unmetPrereqs []string
	for _, prereq := range targetJob.Prerequisites {
		// Parse prerequisite format: "module/job" or just "job" (same module)
		var prereqModule, prereqJob string
		if strings.Contains(prereq, "/") {
			parts := strings.SplitN(prereq, "/", 2)
			prereqModule = parts[0]
			prereqJob = parts[1]
		} else {
			prereqModule = moduleName
			prereqJob = prereq
		}

		// Check if prerequisite job is completed
		var jobState *state.JobState
		if prereqModule == moduleName {
			jobState = module.Jobs[prereqJob]
		} else {
			// Check in another module
			otherModule, ok := stateData.Modules[prereqModule]
			if ok {
				jobState = otherModule.Jobs[prereqJob]
			}
		}

		if jobState == nil || jobState.Status != state.StatusCompleted {
			unmetPrereqs = append(unmetPrereqs, prereq)
		}
	}

	if len(unmetPrereqs) > 0 {
		return fmt.Errorf("前置条件不满足: %s", strings.Join(unmetPrereqs, ", "))
	}

	return nil
}

// updateStatus updates the status of a job and persists it to file.
// Task 5 & 6: Implement updateStatus() and state persistence
func (h *DoingHandler) updateStatus(moduleName, jobName string, status state.Status) error {
	if h.stateManager == nil {
		return fmt.Errorf("state manager not initialized")
	}

	return h.stateManager.UpdateJobStatus(moduleName, jobName, status)
}

// GetStateManager returns the state manager (useful for testing).
func (h *DoingHandler) GetStateManager() *state.Manager {
	return h.stateManager
}

// loadPlan loads and parses a Plan file for the specified module.
// Task 1: Implement `loadPlan(module)` to load module Plan
// Task 2: Use Markdown Parser to parse Plan file
// It returns the parsed Plan struct or an error if the file doesn't exist or can't be parsed.
func (h *DoingHandler) loadPlan(module string) (*plan.Plan, error) {
	planFile := filepath.Join(h.getPlanDir(), module+".md")

	content, err := os.ReadFile(planFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("计划文件不存在: %s", planFile)
		}
		return nil, fmt.Errorf("读取计划文件失败: %w", err)
	}

	planData, err := plan.ParsePlan(string(content))
	if err != nil {
		return nil, fmt.Errorf("解析计划文件失败: %w", err)
	}

	return planData, nil
}

// getJobFromPlan retrieves a specific Job from the parsed Plan.
// Task 3: Extract target Job definition
// Task 4: Extract Job's Tasks list
// Task 5: Extract validator definition
func (h *DoingHandler) getJobFromPlan(planData *plan.Plan, jobName string) (*plan.Job, error) {
	for i := range planData.Jobs {
		if planData.Jobs[i].Name == jobName {
			return &planData.Jobs[i], nil
		}
	}
	return nil, fmt.Errorf("job %q not found in plan", jobName)
}

// PlanNotFoundError represents an error when a Plan file is not found.
// Task 6: Handle Plan file not found error
func (h *DoingHandler) IsPlanNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return os.IsNotExist(err) || strings.Contains(err.Error(), "计划文件不存在")
}
