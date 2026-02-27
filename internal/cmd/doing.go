// Package cmd provides command handlers for Morty CLI commands.
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/morty/morty/internal/callcli"
	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/executor"
	"github.com/morty/morty/internal/git"
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
	executor     executor.Engine
	gitManager   *git.Manager
}

// NewDoingHandler creates a new DoingHandler instance.
func NewDoingHandler(cfg config.Manager, logger logging.Logger) *DoingHandler {
	// Create paths with config loader if available
	var paths *config.Paths
	if loader, ok := cfg.(*config.Loader); ok {
		paths = config.NewPathsWithLoader(loader)
		if os.Getenv("MORTY_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "DEBUG: NewDoingHandler using Loader\n")
		}
	} else {
		paths = config.NewPaths()
		if os.Getenv("MORTY_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "DEBUG: NewDoingHandler using NewPaths (cfg type: %T)\n", cfg)
		}
	}

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
		// If module not found, try to sync from plan directory
		if strings.Contains(err.Error(), "模块不存在") && moduleName != "" {
			logger.Info("Module not found in state, attempting to sync from plan",
				logging.String("module", moduleName),
			)

			// Try to sync this specific module from plan
			if syncErr := h.stateManager.SyncModuleFromPlan(planDir, moduleName); syncErr != nil {
				logger.Warn("Failed to sync module from plan",
					logging.String("module", moduleName),
					logging.String("error", syncErr.Error()),
				)
			} else {
				logger.Info("Successfully synced module from plan",
					logging.String("module", moduleName),
				)

				// Retry selecting target job
				targetModule, targetJob, err = h.selectTargetJob(moduleName, jobName)
				if err != nil {
					result.Err = err
					result.ExitCode = 1
					result.Duration = time.Since(startTime)
					logger.Error("Failed to select target job after sync", logging.String("error", err.Error()))
					return result, result.Err
				}
				// Success! Continue with the synced module
				goto jobSelected
			}
		}

		// If no module specified, try to sync all modules from plan
		if strings.Contains(err.Error(), "没有待执行的 Job") {
			logger.Info("No pending jobs found, attempting to sync all modules from plan")

			if syncErr := h.stateManager.SyncFromPlanDir(planDir); syncErr != nil {
				logger.Warn("Failed to sync from plan directory",
					logging.String("error", syncErr.Error()),
				)
			} else {
				logger.Info("Successfully synced modules from plan directory")

				// Retry selecting target job
				targetModule, targetJob, err = h.selectTargetJob(moduleName, jobName)
				if err != nil {
					result.Err = err
					result.ExitCode = 1
					result.Duration = time.Since(startTime)
					logger.Error("Failed to select target job after sync", logging.String("error", err.Error()))
					return result, result.Err
				}
				// Success! Continue
				goto jobSelected
			}
		}

		result.Err = err
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		logger.Error("Failed to select target job", logging.String("error", err.Error()))
		return result, result.Err
	}

jobSelected:

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

	// Step 5: Initialize Executor and execute the job
	if err := h.initializeExecutor(); err != nil {
		result.Err = fmt.Errorf("初始化执行器失败: %w", err)
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		logger.Error("Failed to initialize executor", logging.String("error", err.Error()))
		return result, result.Err
	}

	// Step 6: Execute the job with timeout control
	execResult, err := h.executeJob(ctx, targetModule, targetJob)
	if err != nil {
		result.Err = err
		result.ExitCode = 1
		result.Duration = time.Since(startTime)
		logger.Error("Job execution failed",
			logging.String("module", targetModule),
			logging.String("job", targetJob),
			logging.String("error", err.Error()),
		)
		return result, result.Err
	}

	result.Duration = time.Since(startTime)
	result.ExitCode = 0

	logger.Info("Doing command completed",
		logging.String("module", targetModule),
		logging.String("job", targetJob),
		logging.Int("exit_code", result.ExitCode),
		logging.Any("duration", result.Duration),
		logging.String("exec_status", string(execResult.Status)),
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
// - If no params: find first executable PENDING job (no unmet prerequisites) across all modules
// - If module specified: find first executable PENDING job in that module
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

	// Case 2: Only module specified - find first executable PENDING job in that module
	if moduleName != "" {
		module, ok := stateData.Modules[moduleName]
		if !ok {
			return "", "", fmt.Errorf("模块不存在: %s", moduleName)
		}

		// Find first PENDING job that has no unmet prerequisites
		targetJob := h.findExecutableJob(moduleName, module)
		if targetJob != "" {
			return moduleName, targetJob, nil
		}

		return "", "", fmt.Errorf("模块 %s 没有待执行的 Job", moduleName)
	}

	// Case 3: No params - find first executable PENDING job across all modules
	// Sort module names for consistent ordering
	var moduleNames []string
	for name := range stateData.Modules {
		if name != "" { // Skip empty module name
			moduleNames = append(moduleNames, name)
		}
	}
	sort.Strings(moduleNames)

	for _, moduleName := range moduleNames {
		module := stateData.Modules[moduleName]
		targetJob := h.findExecutableJob(moduleName, module)
		if targetJob != "" {
			return moduleName, targetJob, nil
		}
	}

	return "", "", fmt.Errorf("没有待执行的 Job")
}

// findExecutableJob finds the first PENDING job in a module that has all prerequisites met.
// Returns empty string if no executable job is found.
func (h *DoingHandler) findExecutableJob(moduleName string, module *state.ModuleState) string {
	logger := h.logger

	// Collect all PENDING jobs with their job index from plan
	type jobWithIndex struct {
		name  string
		index int
	}
	var pendingJobs []jobWithIndex

	// Load plan to get job indices for proper ordering
	planFile := filepath.Join(h.getPlanDir(), moduleName+".md")
	content, err := os.ReadFile(planFile)
	jobIndexMap := make(map[string]int)

	if err == nil {
		// Parse plan to extract job indices
		lines := strings.Split(string(content), "\n")
		jobPattern := regexp.MustCompile(`(?i)^###\s*job\s*(\d+)[:：]\s*(.+)$`)
		for _, line := range lines {
			if matches := jobPattern.FindStringSubmatch(line); matches != nil && len(matches) >= 3 {
				var index int
				fmt.Sscanf(matches[1], "%d", &index)
				jobName := strings.TrimSpace(matches[2])
				jobIndexMap[jobName] = index
			}
		}
	}

	// Collect PENDING jobs
	for jobName, job := range module.Jobs {
		if job.Status == state.StatusPending {
			index := jobIndexMap[jobName]
			if index == 0 {
				// If not found in plan, use a large number to put it at the end
				index = 9999
			}
			pendingJobs = append(pendingJobs, jobWithIndex{name: jobName, index: index})
		}
	}

	if os.Getenv("MORTY_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "DEBUG: Finding executable job in module '%s', pending count: %d\n", moduleName, len(pendingJobs))
		fmt.Fprintf(os.Stderr, "DEBUG: Pending jobs: %v\n", pendingJobs)
	}

	logger.Debug("Finding executable job",
		logging.String("module", moduleName),
		logging.Int("pending_count", len(pendingJobs)),
	)

	// Sort by job index (topological order)
	sort.Slice(pendingJobs, func(i, j int) bool {
		return pendingJobs[i].index < pendingJobs[j].index
	})

	// Find first job with all prerequisites met (in topological order)
	for _, jobInfo := range pendingJobs {
		jobName := jobInfo.name
		// Check if this job's prerequisites are met
		err := h.checkPrerequisites(moduleName, jobName)
		if err == nil {
			if os.Getenv("MORTY_DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "DEBUG: Found executable job: %s/%s (index: %d)\n", moduleName, jobName, jobInfo.index)
			}
			logger.Info("Found executable job",
				logging.String("module", moduleName),
				logging.String("job", jobName),
				logging.Int("job_index", jobInfo.index),
			)
			return jobName
		} else {
			if os.Getenv("MORTY_DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "DEBUG: Job '%s/%s' (index: %d) has unmet prerequisites: %v\n", moduleName, jobName, jobInfo.index, err)
			}
			logger.Debug("Job has unmet prerequisites",
				logging.String("module", moduleName),
				logging.String("job", jobName),
				logging.Int("job_index", jobInfo.index),
				logging.String("error", err.Error()),
			)
		}
	}

	if os.Getenv("MORTY_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "DEBUG: No executable job found in module '%s'\n", moduleName)
	}

	logger.Debug("No executable job found in module",
		logging.String("module", moduleName),
	)

	return ""
}

// checkPrerequisites checks if all prerequisite jobs are completed.
// Task 4: Implement prerequisite checking
// It reads the plan file for the module and checks if all jobs listed
// in the job's Prerequisites are in COMPLETED status.
func (h *DoingHandler) checkPrerequisites(moduleName, jobName string) error {
	// Load the plan file for this module
	// First try the exact module name
	planFile := filepath.Join(h.getPlanDir(), moduleName+".md")
	content, err := os.ReadFile(planFile)

	// If not found, search for any plan file containing this module
	if err != nil && os.IsNotExist(err) {
		planFile, content, err = h.findPlanFileForModule(moduleName)
	}

	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("计划文件不存在: module=%s", moduleName)
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
		// If it doesn't match a known job, it's a descriptive prerequisite (e.g., "file exists")
		// and we skip it (assume user will verify manually)

		var prereqModule, prereqJob string
		if strings.Contains(prereq, "/") {
			parts := strings.SplitN(prereq, "/", 2)
			prereqModule = parts[0]
			prereqJob = parts[1]
		} else {
			prereqModule = moduleName
			prereqJob = prereq
		}

		// Check if prerequisite job exists and is completed
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

		// If job doesn't exist, it's likely a descriptive prerequisite (not a job reference)
		// Skip it and assume it will be verified manually
		if jobState == nil {
			if os.Getenv("MORTY_DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "DEBUG: Skipping non-job prerequisite: '%s'\n", prereq)
			}
			continue
		}

		// Job exists, check if it's completed
		if jobState.Status != state.StatusCompleted {
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

// initializeExecutor initializes the executor with necessary dependencies.
// Task 1: Initialize Executor
func (h *DoingHandler) initializeExecutor() error {
	if h.stateManager == nil {
		return fmt.Errorf("state manager not initialized")
	}

	// Initialize git manager if not already set
	if h.gitManager == nil {
		h.gitManager = git.NewManager()
	}

	// Create executor configuration
	execConfig := &executor.Config{
		MaxRetries:   3,
		AutoCommit:   true,
		CommitPrefix: "morty:",
		WorkingDir:   h.getWorkDir(),
		PromptsDir:   h.paths.GetPromptsDir(),
		PlanDir:      h.getPlanDir(),
	}

	// Create the executor engine with CLI caller
	h.executor = executor.NewEngine(h.stateManager, h.gitManager, h.logger, execConfig, h.cliCaller)

	return nil
}

// executeJob executes the specified job using the executor.
// Task 2: Implement executeJob(module, job)
// Task 3: Build execution context
// Task 4: Call Executor to execute
// Task 5: Handle execution results
// Task 6: Timeout control
func (h *DoingHandler) executeJob(ctx context.Context, module, job string) (*executor.ExecutionResult, error) {
	if h.executor == nil {
		return nil, fmt.Errorf("executor not initialized")
	}

	logger := h.logger.WithContext(ctx)
	logger.Info("Starting job execution",
		logging.String("module", module),
		logging.String("job", job),
	)

	// Task 6: Create timeout context (30 minutes default)
	timeout := 30 * time.Minute

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Task 4: Call Executor to execute the job
	err := h.executor.ExecuteJob(execCtx, module, job)

	// Task 5: Handle execution results
	result := &executor.ExecutionResult{
		Module:   module,
		Job:      job,
		Status:   state.StatusCompleted,
		Summary:  "Job completed successfully",
	}

	if err != nil {
		// Check it was a timeout
		if execCtx.Err() == context.DeadlineExceeded {
			result.Status = state.StatusFailed
			result.Summary = fmt.Sprintf("Job execution timed out after %v", timeout)
			logger.Error("Job execution timed out",
				logging.String("module", module),
				logging.String("job", job),
				logging.Any("timeout", timeout),
			)
			return result, fmt.Errorf("job execution timed out after %v: %w", timeout, err)
		}

		result.Status = state.StatusFailed
		result.Summary = fmt.Sprintf("Job execution failed: %v", err)
		logger.Error("Job execution failed",
			logging.String("module", module),
			logging.String("job", job),
			logging.String("error", err.Error()),
		)
		return result, err
	}

	// Get the final job state
	jobState := h.stateManager.GetJob(module, job)
	if jobState != nil {
		result.Status = jobState.Status
		result.TasksCompleted = jobState.TasksCompleted
		result.TasksTotal = jobState.TasksTotal
		result.RetryCount = jobState.RetryCount
	}

	logger.Success("Job execution completed",
		logging.String("module", module),
		logging.String("job", job),
		logging.String("status", string(result.Status)),
	)

	return result, nil
}

// SetExecutor sets a custom executor (useful for testing).
func (h *DoingHandler) SetExecutor(exec executor.Engine) {
	h.executor = exec
}

// SetGitManager sets a custom git manager (useful for testing).
func (h *DoingHandler) SetGitManager(gitMgr *git.Manager) {
	h.gitManager = gitMgr
}

// CommitSummary represents the summary information for creating a git commit.
type CommitSummary struct {
	Module    string
	Job       string
	Status    string
	LoopCount int
}

// ExecutionSummary represents the execution result summary for display.
type ExecutionSummary struct {
	Module         string
	Job            string
	Status         string
	Duration       time.Duration
	TasksTotal     int
	TasksCompleted int
	NextAction     string
}

// generateSummary generates an execution summary from the doing result and execution result.
// Task 1: Implement generateSummary()
// Task 2: Calculate execution duration
// Task 3: Count tasks completion status
// Task 4: Determine next action hint
func (h *DoingHandler) generateSummary(result *DoingResult, execResult *executor.ExecutionResult) *ExecutionSummary {
	summary := &ExecutionSummary{
		Module:   result.ModuleName,
		Job:      result.JobName,
		Status:   string(execResult.Status),
		Duration: result.Duration,
	}

	// Task 3: Get tasks statistics from execution result or state
	if execResult != nil {
		summary.TasksTotal = execResult.TasksTotal
		summary.TasksCompleted = execResult.TasksCompleted
	}

	// If execution result doesn't have task info, try getting from state
	if summary.TasksTotal == 0 && h.stateManager != nil {
		if jobState := h.stateManager.GetJob(result.ModuleName, result.JobName); jobState != nil {
			summary.TasksTotal = jobState.TasksTotal
			summary.TasksCompleted = jobState.TasksCompleted
		}
	}

	// Task 4: Determine next action based on status
	summary.NextAction = h.determineNextAction(summary.Status)

	return summary
}

// determineNextAction determines the next action based on job status.
func (h *DoingHandler) determineNextAction(status string) string {
	switch state.Status(status) {
	case state.StatusCompleted:
		return "运行 `morty doing` 继续执行下一个 Job"
	case state.StatusFailed:
		return "检查错误后运行 `morty doing --restart` 重试"
	case state.StatusBlocked:
		return "等待依赖项完成后重试"
	case state.StatusRunning:
		return "Job 正在执行中，请等待完成"
	default:
		return "运行 `morty doing` 开始执行"
	}
}

// formatSummary formats the execution summary for display.
// Task 5: Format output summary
func (h *DoingHandler) formatSummary(summary *ExecutionSummary) string {
	if summary == nil {
		return "无执行摘要"
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("╔════════════════════════════════════════════════╗\n")
	b.WriteString("║            📋 执行摘要                         ║\n")
	b.WriteString("╚════════════════════════════════════════════════╝\n")

	// Module and Job info
	b.WriteString(fmt.Sprintf("📦 模块: %s\n", summary.Module))
	b.WriteString(fmt.Sprintf("🔧 Job:  %s\n", summary.Job))

	// Status with color indicator (plain text for now, color added in print method)
	statusIcon := h.getStatusIcon(summary.Status)
	b.WriteString(fmt.Sprintf("📊 状态: %s %s\n", statusIcon, summary.Status))

	// Duration
	b.WriteString(fmt.Sprintf("⏱️  耗时: %s\n", h.formatDuration(summary.Duration)))

	// Tasks progress
	if summary.TasksTotal > 0 {
		progress := fmt.Sprintf("📝 任务: %d/%d 完成", summary.TasksCompleted, summary.TasksTotal)
		if summary.TasksCompleted == summary.TasksTotal && summary.TasksTotal > 0 {
			progress += " ✓"
		}
		b.WriteString(progress + "\n")
	}

	// Next action
	b.WriteString("\n👉 下一步: " + summary.NextAction + "\n")

	return b.String()
}

// getStatusIcon returns an icon for the given status.
func (h *DoingHandler) getStatusIcon(status string) string {
	switch state.Status(status) {
	case state.StatusCompleted:
		return "✅"
	case state.StatusFailed:
		return "❌"
	case state.StatusBlocked:
		return "🚫"
	case state.StatusRunning:
		return "🔄"
	default:
		return "⏳"
	}
}

// formatDuration formats a duration in a human-readable format.
func (h *DoingHandler) formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// printSummary prints the execution summary to stdout.
// Task 6: Colorful output support
func (h *DoingHandler) printSummary(summary *ExecutionSummary) {
	if summary == nil {
		return
	}

	// Print header
	fmt.Println()
	fmt.Println(h.colorize("╔════════════════════════════════════════════════╗", ColorCyan))
	fmt.Println(h.colorize("║            📋 执行摘要                         ║", ColorCyan))
	fmt.Println(h.colorize("╚════════════════════════════════════════════════╝", ColorCyan))

	// Print module and job
	fmt.Printf("📦 模块: %s\n", summary.Module)
	fmt.Printf("🔧 Job:  %s\n", summary.Job)

	// Print status with color
	statusColor := h.getStatusColor(summary.Status)
	statusIcon := h.getStatusIcon(summary.Status)
	fmt.Printf("📊 状态: %s %s\n", statusIcon, h.colorize(summary.Status, statusColor))

	// Print duration
	fmt.Printf("⏱️  耗时: %s\n", h.formatDuration(summary.Duration))

	// Print tasks progress
	if summary.TasksTotal > 0 {
		progress := fmt.Sprintf("📝 任务: %d/%d 完成", summary.TasksCompleted, summary.TasksTotal)
		if summary.TasksCompleted == summary.TasksTotal && summary.TasksTotal > 0 {
			progress += h.colorize(" ✓", ColorGreen)
		}
		fmt.Println(progress)
	}

	// Print next action
	fmt.Println()
	fmt.Printf("👉 下一步: %s\n", h.colorize(summary.NextAction, ColorYellow))
}

// Color constants for terminal output.
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

// colorize adds color to text if the terminal supports it.
func (h *DoingHandler) colorize(text, color string) string {
	// Check if terminal supports color
	if !h.isTerminal() {
		return text
	}
	return color + text + ColorReset
}

// isTerminal checks if stdout is a terminal.
func (h *DoingHandler) isTerminal() bool {
	// Check if stdout is a terminal
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

// getStatusColor returns the color for a given status.
func (h *DoingHandler) getStatusColor(status string) string {
	switch state.Status(status) {
	case state.StatusCompleted:
		return ColorGreen
	case state.StatusFailed:
		return ColorRed
	case state.StatusBlocked:
		return ColorYellow
	case state.StatusRunning:
		return ColorBlue
	default:
		return ColorReset
	}
}

// createGitCommit creates a git commit with the job execution summary.
// Task 1: Implement createGitCommit(summary)
// It generates a commit message, stages all changes, and creates a commit.
// The commit message format is: "morty: [module]/[job] - [STATUS]"
func (h *DoingHandler) createGitCommit(summary *CommitSummary) (string, error) {
	// Task 2: Generate commit message
	commitMsg := h.generateCommitMessage(summary)

	// Initialize git manager if needed
	if h.gitManager == nil {
		h.gitManager = git.NewManager()
	}

	workDir := h.getWorkDir()

	// Task 3: Check if there are changes to commit
	hasChanges, err := h.gitManager.HasUncommittedChanges(workDir)
	if err != nil {
		return "", fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}

	// If no changes, return empty hash without error (no commit needed)
	if !hasChanges {
		return "", nil
	}

	// Task 4: Stage all changes
	if _, err := h.gitManager.RunGitCommand(workDir, "add", "-A"); err != nil {
		return "", fmt.Errorf("failed to stage changes: %w", err)
	}

	// Task 5: Create commit
	if _, err := h.gitManager.RunGitCommand(workDir, "commit", "-m", commitMsg); err != nil {
		// Task 6: Handle commit errors
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	// Get the commit hash
	commitHash, err := h.gitManager.RunGitCommand(workDir, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return commitHash, nil
}

// generateCommitMessage generates a commit message from the summary.
// Format: "morty: [module]/[job] - [STATUS]"
// Includes loop count in the message if available.
func (h *DoingHandler) generateCommitMessage(summary *CommitSummary) string {
	if summary == nil {
		return "morty: unknown - UNKNOWN"
	}

	module := summary.Module
	if module == "" {
		module = "unknown"
	}

	job := summary.Job
	if job == "" {
		job = "unknown"
	}

	status := summary.Status
	if status == "" {
		status = "UNKNOWN"
	}

	// Format: morty: module/job - STATUS
	msg := fmt.Sprintf("morty: %s/%s - %s", module, job, status)

	// Include loop count if greater than 0
	if summary.LoopCount > 0 {
		msg = fmt.Sprintf("morty: %s/%s - %s (loop %d)", module, job, status, summary.LoopCount)
	}

	return msg
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

// findPlanFileForModule searches all plan files to find one containing the specified module.
func (h *DoingHandler) findPlanFileForModule(moduleName string) (string, []byte, error) {
	planDir := h.getPlanDir()

	// Read all files in plan directory
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read plan directory: %w", err)
	}

	// Search for plan files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, "README") {
			continue
		}

		// Read and parse the file
		planPath := filepath.Join(planDir, name)
		content, err := os.ReadFile(planPath)
		if err != nil {
			continue
		}

		// Parse to check if it contains the module
		parsedPlan, err := plan.ParsePlan(string(content))
		if err != nil {
			continue
		}

		// Check if this plan is for the requested module
		if parsedPlan.Name == moduleName {
			return planPath, content, nil
		}
	}

	return "", nil, os.ErrNotExist
}
