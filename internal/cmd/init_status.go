package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/state"
)

// InitStatusHandler handles the init-status command.
type InitStatusHandler struct {
	cfg          config.Manager
	paths        *config.Paths
	stateManager *state.Manager
	logger       logging.Logger
}

// NewInitStatusHandler creates a new init-status handler.
func NewInitStatusHandler(cfg config.Manager, logger logging.Logger) *InitStatusHandler {
	return &InitStatusHandler{
		cfg:    cfg,
		paths:  config.NewPaths(),
		logger: logger,
	}
}

// Execute executes the init-status command.
func (h *InitStatusHandler) Execute(ctx context.Context, args []string) error {
	logger := h.logger.WithContext(ctx)

	// Parse flags
	force := false
	for _, arg := range args {
		if arg == "--force" || arg == "-f" {
			force = true
		}
	}

	// Get plan directory
	planDir := h.paths.GetPlanDir()

	// Get status file path
	statusFile := h.paths.GetStatusFile()

	// Check if status file already exists
	if _, err := os.Stat(statusFile); err == nil && !force {
		logger.Warn("Status file already exists. Use --force to overwrite.",
			logging.String("file", statusFile),
		)
		return fmt.Errorf("status file already exists: %s (use --force to overwrite)", statusFile)
	}

	// Initialize state manager
	h.stateManager = state.NewManager(statusFile)

	logger.Info("Generating status.json from plan files",
		logging.String("plan_dir", planDir),
		logging.String("status_file", statusFile),
	)

	// Generate status
	if err := h.stateManager.Initialize(planDir); err != nil {
		logger.Error("Failed to generate status.json",
			logging.String("error", err.Error()),
		)
		return fmt.Errorf("failed to generate status.json: %w", err)
	}

	// Load and display summary
	if err := h.stateManager.Load(); err != nil {
		logger.Error("Failed to load generated status",
			logging.String("error", err.Error()),
		)
		return err
	}

	status := h.stateManager.GetStatus()
	if status == nil {
		return fmt.Errorf("status is nil after generation")
	}

	logger.Info("Status file generated successfully",
		logging.String("file", statusFile),
		logging.Int("modules", status.Global.TotalModules),
		logging.Int("jobs", status.Global.TotalJobs),
	)

	// Display summary
	fmt.Printf("\n✅ Status file generated successfully\n\n")
	fmt.Printf("File: %s\n", statusFile)
	fmt.Printf("Version: %s\n", status.Version)
	fmt.Printf("Modules: %d\n", status.Global.TotalModules)
	fmt.Printf("Jobs: %d\n\n", status.Global.TotalJobs)

	fmt.Printf("Module execution order:\n")
	for i, module := range status.Modules {
		depsStr := "none"
		if len(module.Dependencies) > 0 {
			depsStr = fmt.Sprintf("%v", module.Dependencies)
		}
		fmt.Printf("  %d. %s (%d jobs, deps: %s)\n",
			i+1, module.DisplayName, len(module.Jobs), depsStr)
	}

	fmt.Printf("\nNext step: Run 'morty doing' to start execution\n")

	return nil
}
