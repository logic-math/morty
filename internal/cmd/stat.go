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
	Err         error
	ExitCode    int
	Duration    time.Duration
	JSONOutput  bool
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

// outputJSON outputs the result in JSON format.
func (h *StatHandler) outputJSON(result *StatResult) {
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
