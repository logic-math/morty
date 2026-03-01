package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/state"
)

// StatHandler handles status display commands.
type StatHandler struct {
	configManager config.Manager
	logger        logging.Logger
	stateManager  *state.Manager
}

// StatResult represents the result of stat command execution.
type StatResult struct {
	Status *state.ExecutionStatus
}

// NewStatHandler creates a new StatHandler.
func NewStatHandler(configManager config.Manager, logger logging.Logger) *StatHandler {
	return &StatHandler{
		configManager: configManager,
		logger:        logger,
	}
}

// Execute executes the stat command.
func (h *StatHandler) Execute(ctx context.Context, args []string) (*StatResult, error) {
	// Initialize state manager if not already done
	if h.stateManager == nil {
		statusFile := h.configManager.GetStatusFile()
		h.stateManager = state.NewManager(statusFile)

		// Load status
		if err := h.stateManager.Load(); err != nil {
			return nil, fmt.Errorf("failed to load status: %w", err)
		}
	}

	// Get current status
	status := h.stateManager.GetStatus()
	if status == nil {
		return nil, fmt.Errorf("no status available")
	}

	// Display status
	h.DisplayStatus(status)

	return &StatResult{Status: status}, nil
}

// DisplayStatus displays status in human-readable format.
func (h *StatHandler) DisplayStatus(status *state.ExecutionStatus) {
	// Calculate statistics
	completedJobs := status.CountCompletedJobs()
	completedModules := status.CountCompletedModules()
	totalJobs := status.Global.TotalJobs
	totalModules := status.Global.TotalModules

	// Overall progress
	progressPercent := 0.0
	if totalJobs > 0 {
		progressPercent = float64(completedJobs) / float64(totalJobs) * 100
	}

	fmt.Printf("\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("  Morty Status\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("\n")

	// Global status
	fmt.Printf("Overall Status: %s\n", colorizeStatus(string(status.Global.Status)))
	fmt.Printf("Progress: %d/%d jobs completed (%.1f%%)\n", completedJobs, totalJobs, progressPercent)
	fmt.Printf("Modules: %d/%d completed\n", completedModules, totalModules)
	fmt.Printf("Last Update: %s\n", status.Global.LastUpdate.Format("2006-01-02 15:04:05"))
	fmt.Printf("\n")

	// Module progress
	fmt.Printf("Module Progress:\n")
	fmt.Printf("───────────────────────────────────────────────────────────────\n")

	for i, module := range status.Modules {
		completed := 0
		running := 0
		failed := 0
		for _, job := range module.Jobs {
			switch job.Status {
			case state.StatusCompleted:
				completed++
			case state.StatusRunning:
				running++
			case state.StatusFailed:
				failed++
			}
		}

		// Module icon
		icon := "⏳"
		if completed == len(module.Jobs) && len(module.Jobs) > 0 {
			icon = "✅"
		} else if running > 0 {
			icon = "▶️"
		} else if failed > 0 {
			icon = "❌"
		} else if completed > 0 {
			icon = "🔄"
		}

		// Display module
		fmt.Printf("  %s [%d] %s\n", icon, i+1, module.DisplayName)
		fmt.Printf("      Jobs: %d/%d", completed, len(module.Jobs))
		if running > 0 {
			fmt.Printf(" (running: %d)", running)
		}
		if failed > 0 {
			fmt.Printf(" (failed: %d)", failed)
		}
		fmt.Printf("\n")

		if len(module.Dependencies) > 0 {
			fmt.Printf("      Dependencies: %s\n", strings.Join(module.Dependencies, ", "))
		}

		// Show jobs if module is active
		if running > 0 || (completed > 0 && completed < len(module.Jobs)) {
			fmt.Printf("      Jobs:\n")
			for ji, job := range module.Jobs {
				if job.Status != state.StatusPending {
					jobIcon := getJobIcon(job.Status)
					fmt.Printf("        %s [%d.%d] %s", jobIcon, i+1, ji+1, job.Name)
					if job.Status == state.StatusRunning {
						fmt.Printf(" (%d/%d tasks)", job.TasksCompleted, job.TasksTotal)
					}
					fmt.Printf("\n")
				}
			}
		}
		fmt.Printf("\n")
	}

	// Current execution
	if status.Global.Status == state.StatusRunning {
		fmt.Printf("Current Execution:\n")
		fmt.Printf("───────────────────────────────────────────────────────────────\n")

		moduleIndex := status.Global.CurrentModuleIndex
		if moduleIndex >= 0 && moduleIndex < len(status.Modules) {
			currentModule := status.Modules[moduleIndex]
			fmt.Printf("  Module: %s\n", currentModule.DisplayName)

			// Find current job
			for _, job := range currentModule.Jobs {
				if job.GlobalIndex == status.Global.CurrentJobIndex {
					fmt.Printf("  Job: %s (job %d/%d in module)\n",
						job.Name, job.Index+1, len(currentModule.Jobs))
					fmt.Printf("  Progress: %d/%d tasks completed\n",
						job.TasksCompleted, job.TasksTotal)
					fmt.Printf("  Loop: %d, Retry: %d\n", job.LoopCount, job.RetryCount)
					break
				}
			}
		}
		fmt.Printf("\n")
	}

	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("\n")
}

// FormatStatusAsJSON formats status as JSON string.
func (h *StatHandler) FormatStatusAsJSON(status *state.ExecutionStatus) (string, error) {
	// The status is already in the correct format for JSON
	// We can use the standard JSON marshaling
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal status to JSON: %w", err)
	}
	return string(data), nil
}

// getJobIcon returns an icon for a job status.
func getJobIcon(status state.Status) string {
	switch status {
	case state.StatusCompleted:
		return "✅"
	case state.StatusRunning:
		return "▶️"
	case state.StatusFailed:
		return "❌"
	case state.StatusBlocked:
		return "🚫"
	default:
		return "⏳"
	}
}

// colorizeStatus adds color to status string (if terminal supports it).
func colorizeStatus(status string) string {
	// Simple colorization - can be enhanced with actual terminal colors
	switch status {
	case "COMPLETED":
		return "✅ " + status
	case "RUNNING":
		return "▶️ " + status
	case "FAILED":
		return "❌ " + status
	case "PENDING":
		return "⏳ " + status
	default:
		return status
	}
}
