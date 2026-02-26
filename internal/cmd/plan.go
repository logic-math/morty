// Package cmd provides command handlers for Morty CLI commands.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/morty/morty/internal/callcli"
	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/logging"
)

// PlanResult represents the result of a plan operation.
type PlanResult struct {
	PlanPath    string
	ModuleName  string
	CreatedAt   time.Time
	Err         error
	ExitCode    int
	Duration    time.Duration
	Overwritten bool
}

// PlanHandler handles the plan command.
type PlanHandler struct {
	cfg       config.Manager
	logger    logging.Logger
	paths     *config.Paths
	cliCaller callcli.AICliCaller
}

// NewPlanHandler creates a new PlanHandler instance.
func NewPlanHandler(cfg config.Manager, logger logging.Logger, parser interface{}) *PlanHandler {
	paths := config.NewPaths()
	// Set workDir from config if available
	if cfg != nil && cfg.GetWorkDir() != "" {
		paths.SetWorkDir(cfg.GetWorkDir())
	}
	return &PlanHandler{
		cfg:       cfg,
		logger:    logger,
		paths:     paths,
		cliCaller: callcli.NewAICliCallerWithLoader(cfg),
	}
}

// Execute executes the plan command.
// It creates a new plan file in the .morty/plan/ directory.
// If --force is not provided and a plan file already exists, it prompts for confirmation.
func (h *PlanHandler) Execute(ctx context.Context, args []string) (*PlanResult, error) {
	logger := h.logger.WithContext(ctx)
	startTime := time.Now()

	result := &PlanResult{
		CreatedAt: startTime,
	}

	// Parse options from args
	force, moduleName, remainingArgs := h.parseOptions(args)

	// Determine module name
	if moduleName == "" {
		moduleName = h.inferModuleName()
	}
	result.ModuleName = moduleName

	logger.Info("Starting plan creation",
		logging.String("module", moduleName),
		logging.Bool("force", force),
	)

	// Ensure plan directory exists
	if err := h.ensurePlanDir(); err != nil {
		logger.Error("Failed to create plan directory", logging.String("error", err.Error()))
		result.Err = err
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("failed to create plan directory: %w", err)
	}

	// Generate plan file path
	planPath := h.generatePlanPath(moduleName)
	result.PlanPath = planPath

	// Check if plan file already exists
	exists := h.planFileExists(planPath)

	if exists && !force {
		// Prompt for confirmation
		confirmed, err := h.promptForOverwrite(moduleName)
		if err != nil {
			logger.Error("Failed to prompt for confirmation", logging.String("error", err.Error()))
			result.Err = err
			result.Duration = time.Since(startTime)
			return result, err
		}
		if !confirmed {
			logger.Info("Plan creation cancelled by user")
			result.Duration = time.Since(startTime)
			return result, nil
		}
		result.Overwritten = true
	} else if exists && force {
		result.Overwritten = true
	}

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		result.Err = ctx.Err()
		result.Duration = time.Since(startTime)
		return result, ctx.Err()
	default:
	}

	// Create the plan file
	if err := h.createPlanFile(planPath, moduleName, remainingArgs); err != nil {
		logger.Error("Failed to create plan file", logging.String("error", err.Error()))
		result.Err = err
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("failed to create plan file: %w", err)
	}

	result.Duration = time.Since(startTime)
	result.ExitCode = 0

	logger.Info("Plan creation completed",
		logging.String("module", moduleName),
		logging.String("plan_path", planPath),
		logging.Bool("overwritten", result.Overwritten),
		logging.Any("duration", result.Duration),
	)

	return result, nil
}

// parseOptions parses command-line options from args.
// Returns (force flag, module name, remaining args)
func (h *PlanHandler) parseOptions(args []string) (bool, string, []string) {
	force := false
	var moduleName string
	var remaining []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check for --force or -f
		if arg == "--force" || arg == "-f" {
			force = true
			continue
		}

		// Check for --force=value format
		if strings.HasPrefix(arg, "--force=") {
			val := strings.TrimPrefix(arg, "--force=")
			force = val == "true" || val == "1"
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

		// Collect remaining args
		remaining = append(remaining, arg)
	}

	return force, moduleName, remaining
}

// inferModuleName attempts to infer the module name from the current directory.
func (h *PlanHandler) inferModuleName() string {
	// Get current directory name
	cwd, err := os.Getwd()
	if err != nil {
		return "default"
	}

	// Use the directory name as module name
	base := filepath.Base(cwd)
	if base == "" || base == "." {
		return "default"
	}

	return sanitizeModuleName(base)
}

// sanitizeModuleName converts a string into a safe module name.
func sanitizeModuleName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace spaces and special characters with underscores
	var sb strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			sb.WriteRune(r)
		case r >= '0' && r <= '9':
			sb.WriteRune(r)
		default:
			sb.WriteRune('_')
		}
	}

	result := sb.String()

	// Remove trailing underscores
	result = strings.Trim(result, "_")

	// Ensure not empty
	if result == "" {
		result = "default"
	}

	return result
}

// ensurePlanDir ensures the .morty/plan/ directory exists.
func (h *PlanHandler) ensurePlanDir() error {
	// Use config's GetPlanDir if available, otherwise fall back to paths
	if h.cfg != nil {
		return h.paths.EnsureDir(h.cfg.GetPlanDir())
	}
	return h.paths.EnsurePlanDir()
}

// getPlanDir returns the plan directory, preferring config if available.
func (h *PlanHandler) getPlanDir() string {
	if h.cfg != nil {
		return h.cfg.GetPlanDir()
	}
	return h.paths.GetPlanDir()
}

// generatePlanPath generates a plan file path for the given module.
func (h *PlanHandler) generatePlanPath(moduleName string) string {
	sanitized := sanitizeModuleName(moduleName)
	return filepath.Join(h.getPlanDir(), sanitized+".md")
}

// planFileExists checks if a plan file already exists.
func (h *PlanHandler) planFileExists(planPath string) bool {
	_, err := os.Stat(planPath)
	return err == nil
}

// promptForOverwrite prompts the user to confirm overwriting an existing plan.
func (h *PlanHandler) promptForOverwrite(moduleName string) (bool, error) {
	fmt.Printf("A plan file for module '%s' already exists. Overwrite? [y/N]: ", moduleName)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// createPlanFile creates a new plan file with the given content.
func (h *PlanHandler) createPlanFile(planPath, moduleName string, args []string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(planPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Build plan content
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# Plan: %s\n\n", moduleName))
	content.WriteString(fmt.Sprintf("Module: %s\n", moduleName))
	content.WriteString(fmt.Sprintf("Created: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	content.WriteString("## Overview\n\n")
	content.WriteString("<!-- Add plan overview here -->\n\n")

	content.WriteString("## Jobs\n\n")

	// If args are provided, use them as job names
	if len(args) > 0 {
		for i, arg := range args {
			content.WriteString(fmt.Sprintf("### Job %d: %s\n\n", i+1, arg))
			content.WriteString("**Goal**: ...\n\n")
			content.WriteString("**Tasks**:\n")
			content.WriteString("- [ ] Task 1\n")
			content.WriteString("- [ ] Task 2\n")
			content.WriteString("- [ ] Task 3\n\n")
			content.WriteString("**Validator**: ...\n\n")
		}
	} else {
		// Default job template
		content.WriteString("### Job 1: Initial Setup\n\n")
		content.WriteString("**Goal**: Setup the initial structure\n\n")
		content.WriteString("**Tasks**:\n")
		content.WriteString("- [ ] Task 1\n")
		content.WriteString("- [ ] Task 2\n")
		content.WriteString("- [ ] Task 3\n\n")
		content.WriteString("**Validator**: All tasks completed\n\n")
	}

	content.WriteString("## Notes\n\n")
	content.WriteString("<!-- Add additional notes here -->\n")

	// Write to file
	if err := os.WriteFile(planPath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	return nil
}

// GetPlanDir returns the plan directory path.
func (h *PlanHandler) GetPlanDir() string {
	return h.getPlanDir()
}

// SetPlanDir sets a custom plan directory (useful for testing).
func (h *PlanHandler) SetPlanDir(dir string) {
	h.paths.SetWorkDir(dir)
}

// getResearchDir returns the research directory path.
func (h *PlanHandler) getResearchDir() string {
	if h.cfg != nil {
		return h.cfg.GetResearchDir()
	}
	return h.paths.GetResearchDir()
}

// loadResearchFacts scans the .morty/research/ directory, reads all .md files,
// sorts them by filename, and formats them for prompt input.
// Returns an empty slice if no research files exist.
func (h *PlanHandler) loadResearchFacts() ([]string, error) {
	researchDir := h.getResearchDir()

	// Check if directory exists
	if _, err := os.Stat(researchDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	// Read directory contents
	entries, err := os.ReadDir(researchDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read research directory: %w", err)
	}

	// Collect all .md files
	var mdFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".md") {
			mdFiles = append(mdFiles, name)
		}
	}

	// Sort by filename
	sort.Strings(mdFiles)

	// Read and format each file
	// Initialize as empty slice to avoid returning nil
	facts := make([]string, 0, len(mdFiles))
	for _, filename := range mdFiles {
		filePath := filepath.Join(researchDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read research file %s: %w", filename, err)
		}

		// Format as: --- [filename] ---\n[content]
		formatted := fmt.Sprintf("--- %s ---\n%s", filename, string(content))
		facts = append(facts, formatted)
	}

	return facts, nil
}

// SetCLICaller sets a custom CLI caller (useful for testing).
func (h *PlanHandler) SetCLICaller(caller callcli.AICliCaller) {
	h.cliCaller = caller
}

// SetPromptsDir sets a custom prompts directory (useful for testing).
func (h *PlanHandler) SetPromptsDir(dir string) {
	h.paths.SetPromptsDir(dir)
}

// loadPlanPrompt loads the plan prompt from prompts/plan.md.
func (h *PlanHandler) loadPlanPrompt() (string, error) {
	promptPath := h.getPlanPromptPath()

	// Read the prompt file
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read plan prompt file %s: %w", promptPath, err)
	}

	return string(content), nil
}

// getPlanPromptPath returns the path to the plan prompt file.
func (h *PlanHandler) getPlanPromptPath() string {
	// First check if there's a config override
	if h.cfg != nil {
		if promptPath := h.cfg.GetString("prompts.plan"); promptPath != "" {
			return h.paths.GetAbsolutePath(promptPath)
		}
	}

	// Default to prompts/plan.md relative to prompts dir
	return filepath.Join(h.paths.GetPromptsDir(), "plan.md")
}

// buildClaudeCommand builds the Claude Code command arguments.
func (h *PlanHandler) buildClaudeCommand(prompt string, facts []string) []string {
	var args []string

	// Add permission mode plan
	args = append(args, "--permission-mode", "plan")

	// Build full prompt with research facts if available
	var fullPrompt strings.Builder

	// Add research facts section if there are any
	if len(facts) > 0 {
		fullPrompt.WriteString("# Research Facts\n\n")
		for i, fact := range facts {
			fullPrompt.WriteString(fmt.Sprintf("## Fact %d\n%s\n\n", i+1, fact))
		}
		fullPrompt.WriteString("---\n\n")
	}

	// Add the main prompt content
	fullPrompt.WriteString(prompt)

	// Add the prompt content via -p flag
	args = append(args, "-p", fullPrompt.String())

	return args
}

// executeClaudeCode executes Claude Code with the given prompt and research facts.
// Returns the exit code and any error that occurred.
func (h *PlanHandler) executeClaudeCode(ctx context.Context, prompt string, facts []string) (int, error) {
	logger := h.logger.WithContext(ctx)

	// Build full prompt with research facts
	var fullPrompt strings.Builder

	// Add research facts section if there are any
	if len(facts) > 0 {
		fullPrompt.WriteString("# Research Facts\n\n")
		for i, fact := range facts {
			fullPrompt.WriteString(fmt.Sprintf("## Fact %d\n%s\n\n", i+1, fact))
		}
		fullPrompt.WriteString("---\n\n")
	}

	// Add the main prompt content
	fullPrompt.WriteString(prompt)

	logger.Info("Executing Claude Code for plan mode",
		logging.String("cli_path", h.cliCaller.GetCLIPath()),
		logging.Int("facts_count", len(facts)),
	)

	// Create options for the call
	opts := callcli.Options{
		Timeout: 0, // No timeout for interactive plan mode
		Output: callcli.OutputConfig{
			Mode: callcli.OutputStream, // Stream output for interactive mode
		},
	}

	// Build base args
	baseArgs := h.cliCaller.BuildArgs()

	// Add permission mode plan
	args := append([]string{"--permission-mode", "plan"}, baseArgs...)

	// Add the prompt content
	args = append(args, "-p", fullPrompt.String())

	// Execute the command using the base caller
	result, err := h.cliCaller.GetBaseCaller().CallWithOptions(ctx, h.cliCaller.GetCLIPath(), args, opts)

	if err != nil {
		return result.ExitCode, err
	}

	if result.ExitCode != 0 {
		return result.ExitCode, fmt.Errorf("claude code exited with code %d: %s", result.ExitCode, result.Stderr)
	}

	return result.ExitCode, nil
}
