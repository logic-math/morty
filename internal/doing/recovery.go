// Package doing provides job execution functionality with error handling and retry mechanisms.
package doing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/state"
)

// RecoveryPoint represents a point-in-time state snapshot for recovery.
type RecoveryPoint struct {
	Timestamp   time.Time              `json:"timestamp"`
	Module      string                 `json:"module"`
	Job         string                 `json:"job"`
	JobStatus   state.Status           `json:"job_status"`
	LoopCount   int                    `json:"loop_count"`
	RetryCount  int                    `json:"retry_count"`
	TasksDone   int                    `json:"tasks_done"`
	TasksTotal  int                    `json:"tasks_total"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// StateRecovery manages state recovery points and restoration.
// Task 6: Implement state recovery mechanism
type StateRecovery struct {
	logger         logging.Logger
	recoveryDir    string
	maxRecoveryPoints int
	stateManager   *state.Manager
}

// NewStateRecovery creates a new state recovery manager.
func NewStateRecovery(logger logging.Logger, workDir string, stateManager *state.Manager) *StateRecovery {
	if workDir == "" {
		workDir = ".morty"
	}

	return &StateRecovery{
		logger:            logger,
		recoveryDir:       filepath.Join(workDir, "recovery"),
		maxRecoveryPoints: 10,
		stateManager:      stateManager,
	}
}

// CreateRecoveryPoint creates a recovery point for the current state.
func (sr *StateRecovery) CreateRecoveryPoint(module, job string) (*RecoveryPoint, error) {
	if sr.stateManager == nil {
		return nil, fmt.Errorf("state manager not initialized")
	}

	jobState := sr.stateManager.GetJob(module, job)
	if jobState == nil {
		return nil, fmt.Errorf("job state not found: %s/%s", module, job)
	}

	recoveryPoint := &RecoveryPoint{
		Timestamp:  time.Now(),
		Module:     module,
		Job:        job,
		JobStatus:  jobState.Status,
		LoopCount:  jobState.LoopCount,
		RetryCount: jobState.RetryCount,
		TasksDone:  jobState.TasksCompleted,
		TasksTotal: jobState.TasksTotal,
		Metadata: map[string]interface{}{
			"created_by": "state_recovery",
			"version":    "1.0",
		},
	}

	// Save to file
	if err := sr.saveRecoveryPoint(recoveryPoint); err != nil {
		return nil, fmt.Errorf("failed to save recovery point: %w", err)
	}

	// Cleanup old recovery points
	sr.cleanupOldRecoveryPoints(module, job)

	sr.logger.Info("Created recovery point",
		logging.String("module", module),
		logging.String("job", job),
		logging.String("timestamp", recoveryPoint.Timestamp.Format(time.RFC3339)),
	)

	return recoveryPoint, nil
}

// saveRecoveryPoint saves a recovery point to file.
func (sr *StateRecovery) saveRecoveryPoint(rp *RecoveryPoint) error {
	if err := os.MkdirAll(sr.recoveryDir, 0755); err != nil {
		return fmt.Errorf("failed to create recovery directory: %w", err)
	}

	filename := fmt.Sprintf("%s_%s_%s.json",
		rp.Module,
		rp.Job,
		rp.Timestamp.Format("20060102_150405"),
	)
	filepath := filepath.Join(sr.recoveryDir, filename)

	data, err := json.MarshalIndent(rp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal recovery point: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write recovery point: %w", err)
	}

	return nil
}

// RestoreFromRecovery restores job state from a recovery point.
func (sr *StateRecovery) RestoreFromRecovery(recoveryPoint *RecoveryPoint) error {
	if sr.stateManager == nil {
		return fmt.Errorf("state manager not initialized")
	}

	// Update the job state
	err := sr.stateManager.UpdateJobStatus(recoveryPoint.Module, recoveryPoint.Job, recoveryPoint.JobStatus)
	if err != nil {
		return fmt.Errorf("failed to restore job status: %w", err)
	}

	// Get the job state to update additional fields
	jobState := sr.stateManager.GetJob(recoveryPoint.Module, recoveryPoint.Job)
	if jobState == nil {
		return fmt.Errorf("job not found after restore: %s/%s", recoveryPoint.Module, recoveryPoint.Job)
	}

	// Restore additional fields
	jobState.LoopCount = recoveryPoint.LoopCount
	jobState.RetryCount = recoveryPoint.RetryCount
	jobState.TasksCompleted = recoveryPoint.TasksDone
	jobState.TasksTotal = recoveryPoint.TasksTotal
	jobState.UpdatedAt = time.Now()

	// Save the state
	if err := sr.stateManager.Save(); err != nil {
		return fmt.Errorf("failed to save restored state: %w", err)
	}

	sr.logger.Info("Restored from recovery point",
		logging.String("module", recoveryPoint.Module),
		logging.String("job", recoveryPoint.Job),
		logging.String("status", string(recoveryPoint.JobStatus)),
	)

	return nil
}

// ListRecoveryPoints lists available recovery points for a job.
func (sr *StateRecovery) ListRecoveryPoints(module, job string) ([]*RecoveryPoint, error) {
	pattern := filepath.Join(sr.recoveryDir, fmt.Sprintf("%s_%s_*.json", module, job))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list recovery points: %w", err)
	}

	var points []*RecoveryPoint
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			sr.logger.Warn("Failed to read recovery point",
				logging.String("file", match),
				logging.String("error", err.Error()),
			)
			continue
		}

		var rp RecoveryPoint
		if err := json.Unmarshal(data, &rp); err != nil {
			sr.logger.Warn("Failed to unmarshal recovery point",
				logging.String("file", match),
				logging.String("error", err.Error()),
			)
			continue
		}

		points = append(points, &rp)
	}

	// Sort by timestamp (newest first)
	for i := 0; i < len(points)-1; i++ {
		for j := i + 1; j < len(points); j++ {
			if points[j].Timestamp.After(points[i].Timestamp) {
				points[i], points[j] = points[j], points[i]
			}
		}
	}

	return points, nil
}

// GetLatestRecoveryPoint gets the most recent recovery point for a job.
func (sr *StateRecovery) GetLatestRecoveryPoint(module, job string) (*RecoveryPoint, error) {
	points, err := sr.ListRecoveryPoints(module, job)
	if err != nil {
		return nil, err
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("no recovery points found for %s/%s", module, job)
	}

	return points[0], nil
}

// cleanupOldRecoveryPoints removes old recovery points, keeping only the most recent ones.
func (sr *StateRecovery) cleanupOldRecoveryPoints(module, job string) {
	points, err := sr.ListRecoveryPoints(module, job)
	if err != nil {
		sr.logger.Warn("Failed to list recovery points for cleanup", logging.String("error", err.Error()))
		return
	}

	if len(points) <= sr.maxRecoveryPoints {
		return
	}

	// Remove older recovery points
	for i := sr.maxRecoveryPoints; i < len(points); i++ {
		filename := fmt.Sprintf("%s_%s_%s.json",
			points[i].Module,
			points[i].Job,
			points[i].Timestamp.Format("20060102_150405"),
		)
		filepath := filepath.Join(sr.recoveryDir, filename)

		if err := os.Remove(filepath); err != nil {
			sr.logger.Warn("Failed to remove old recovery point",
				logging.String("file", filepath),
				logging.String("error", err.Error()),
			)
		}
	}
}

// DeleteRecoveryPoint deletes a specific recovery point.
func (sr *StateRecovery) DeleteRecoveryPoint(rp *RecoveryPoint) error {
	filename := fmt.Sprintf("%s_%s_%s.json",
		rp.Module,
		rp.Job,
		rp.Timestamp.Format("20060102_150405"),
	)
	filepath := filepath.Join(sr.recoveryDir, filename)

	if err := os.Remove(filepath); err != nil {
		return fmt.Errorf("failed to delete recovery point: %w", err)
	}

	return nil
}

// ClearAllRecoveryPoints clears all recovery points for a job.
func (sr *StateRecovery) ClearAllRecoveryPoints(module, job string) error {
	points, err := sr.ListRecoveryPoints(module, job)
	if err != nil {
		return err
	}

	for _, rp := range points {
		if err := sr.DeleteRecoveryPoint(rp); err != nil {
			sr.logger.Warn("Failed to delete recovery point",
				logging.String("timestamp", rp.Timestamp.Format(time.RFC3339)),
				logging.String("error", err.Error()),
			)
		}
	}

	return nil
}

// AutoRecover attempts to automatically recover from an error.
func (sr *StateRecovery) AutoRecover(module, job string, err error) (bool, error) {
	// Classify the error
	doingErr := ClassifyError(err)

	sr.logger.Info("Attempting auto-recovery",
		logging.String("module", module),
		logging.String("job", job),
		logging.String("category", doingErr.Category.String()),
	)

	switch doingErr.Category {
	case ErrorCategoryState:
		// Try to restore from the latest recovery point
		rp, err := sr.GetLatestRecoveryPoint(module, job)
		if err != nil {
			sr.logger.Warn("No recovery point available", logging.String("error", err.Error()))
			return false, nil
		}

		sr.logger.Info("Found recovery point, attempting restore",
			logging.String("timestamp", rp.Timestamp.Format(time.RFC3339)),
		)

		if err := sr.RestoreFromRecovery(rp); err != nil {
			return false, fmt.Errorf("failed to restore from recovery point: %w", err)
		}

		return true, nil

	case ErrorCategoryTransient:
		// Transient errors are handled by retry, no state recovery needed
		sr.logger.Info("Transient error detected, retry will handle it")
		return true, nil

	default:
		sr.logger.Info("Auto-recovery not available for this error category")
		return false, nil
	}
}

// FormatRecoveryReport formats a recovery report for display.
func (sr *StateRecovery) FormatRecoveryReport(module, job string) string {
	points, err := sr.ListRecoveryPoints(module, job)
	if err != nil {
		return fmt.Sprintf("Failed to list recovery points: %v", err)
	}

	if len(points) == 0 {
		return "No recovery points available for this job."
	}

	report := fmt.Sprintf("\n📋 Recovery Points for %s/%s\n", module, job)
	report += "═══════════════════════════════════════\n"

	for i, rp := range points {
		if i >= 5 {
			report += fmt.Sprintf("\n... and %d more\n", len(points)-5)
			break
		}

		report += fmt.Sprintf("\n[%d] %s\n", i+1, rp.Timestamp.Format("2006-01-02 15:04:05"))
		report += fmt.Sprintf("    Status: %s\n", rp.JobStatus)
		report += fmt.Sprintf("    Loop: %d, Retry: %d\n", rp.LoopCount, rp.RetryCount)
		report += fmt.Sprintf("    Tasks: %d/%d\n", rp.TasksDone, rp.TasksTotal)
	}

	return report
}
