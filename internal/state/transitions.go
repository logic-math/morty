// Package state provides state management for Morty.
package state

import (
	"fmt"
	"time"

	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/pkg/errors"
)

// TransitionRules defines the valid state transitions.
// The key is the source status, and the value is a slice of valid destination statuses.
var TransitionRules = map[Status][]Status{
	StatusPending:   {StatusRunning, StatusBlocked},
	StatusRunning:   {StatusCompleted, StatusFailed, StatusBlocked},
	StatusCompleted: {}, // Terminal state - no outgoing transitions
	StatusFailed:    {StatusPending}, // Retry flow
	StatusBlocked:   {StatusPending}, // Unblock flow
}

// IsValidTransition checks if a transition from one status to another is valid.
// Returns true if the transition is allowed according to the TransitionRules.
func IsValidTransition(from, to Status) bool {
	// Validate that both statuses are valid
	if !from.IsValid() || !to.IsValid() {
		return false
	}

	// Get valid transitions for the source status
	validTransitions, exists := TransitionRules[from]
	if !exists {
		return false
	}

	// Check if the destination status is in the list of valid transitions
	for _, validStatus := range validTransitions {
		if validStatus == to {
			return true
		}
	}

	return false
}

// GetValidTransitions returns all valid destination statuses for a given source status.
// Returns an empty slice if the source status is invalid or has no valid outgoing transitions.
func GetValidTransitions(from Status) []Status {
	if !from.IsValid() {
		return []Status{}
	}

	transitions, exists := TransitionRules[from]
	if !exists {
		return []Status{}
	}

	// Return a copy to prevent external modification
	result := make([]Status, len(transitions))
	copy(result, transitions)
	return result
}

// TransitionError represents an invalid state transition error.
type TransitionError struct {
	From   Status
	To     Status
	Reason string
}

// Error implements the error interface.
func (e *TransitionError) Error() string {
	return fmt.Sprintf("invalid transition from %s to %s: %s", e.From, e.To, e.Reason)
}

// TransitionJobStatus performs a validated status transition for a job.
// It checks if the transition is valid before updating the status.
// Returns an error if the transition is invalid or if the update fails.
func (m *Manager) TransitionJobStatus(module, job string, toStatus Status, logger logging.Logger) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// V2 Compatible: If V1 state is nil, try V2
	if m.state == nil {
		// Find module and job indices in V2 status
		statusV2Mu.RLock()
		if statusV2 != nil {
			moduleIndex := -1
			jobIndex := -1
			for i, mod := range statusV2.Modules {
				if mod.Name == module {
					moduleIndex = i
					for j, j2 := range mod.Jobs {
						if j2.Name == job {
							jobIndex = j
							break
						}
					}
					break
				}
			}
			statusV2Mu.RUnlock()

			if moduleIndex >= 0 && jobIndex >= 0 {
				// For V2, we don't validate transitions - just update status
				// Note: UpdateJobStatusV2 will handle the lock itself
				m.mu.Unlock() // Release the lock before calling UpdateJobStatusV2
				defer m.mu.Lock() // Re-acquire after return
				return m.UpdateJobStatusV2(moduleIndex, jobIndex, toStatus)
			}
		} else {
			statusV2Mu.RUnlock()
		}
		return errors.New("M2003", "state not loaded")
	}

	// Validate destination status
	if !toStatus.IsValid() {
		return errors.New("M2003", "invalid target status: "+string(toStatus))
	}

	// Get the module state
	moduleState, ok := m.state.Modules[module]
	if !ok {
		return errors.New("M2003", "module not found: "+module)
	}

	// Get the job state
	jobState, ok := moduleState.Jobs[job]
	if !ok {
		return errors.New("M2003", "job not found: "+job+" in module "+module)
	}

	fromStatus := jobState.Status

	// Check if the transition is valid
	if !IsValidTransition(fromStatus, toStatus) {
		reason := fmt.Sprintf("transition from %s to %s is not allowed", fromStatus, toStatus)
		if logger != nil {
			logger.Error("Invalid state transition",
				logging.String("module", module),
				logging.String("job", job),
				logging.String("from", string(fromStatus)),
				logging.String("to", string(toStatus)),
				logging.String("reason", reason),
			)
		}
		return &TransitionError{
			From:   fromStatus,
			To:     toStatus,
			Reason: reason,
		}
	}

	// Perform the transition
	return m.updateJobStatusInternal(moduleState, jobState, toStatus)
}

// updateJobStatusInternal updates the job status without lock (caller must hold lock).
func (m *Manager) updateJobStatusInternal(moduleState *ModuleState, jobState *JobState, status Status) error {
	now := time.Now()

	// Update job status
	oldStatus := jobState.Status
	jobState.Status = status
	jobState.UpdatedAt = now

	// Update retry count for retry transitions (FAILED -> PENDING)
	if oldStatus == StatusFailed && status == StatusPending {
		// This is a retry transition
		jobState.RetryCount++
	}

	// Update module timestamp
	moduleState.UpdatedAt = now

	// Update global timestamp
	m.state.Global.LastUpdate = now

	// Save state to file
	m.mu.Unlock()
	err := m.Save()
	m.mu.Lock()

	return err
}

// CanTransition checks if a job can transition to the specified status.
// Returns nil if the transition is valid, otherwise returns an error with details.
func (m *Manager) CanTransition(module, job string, toStatus Status) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.state == nil {
		return errors.New("M2003", "state not loaded")
	}

	// Validate destination status
	if !toStatus.IsValid() {
		return errors.New("M2003", "invalid target status: "+string(toStatus))
	}

	// Get the module state
	moduleState, ok := m.state.Modules[module]
	if !ok {
		return errors.New("M2003", "module not found: "+module)
	}

	// Get the job state
	jobState, ok := moduleState.Jobs[job]
	if !ok {
		return errors.New("M2003", "job not found: "+job+" in module "+module)
	}

	fromStatus := jobState.Status

	// Check if the transition is valid
	if !IsValidTransition(fromStatus, toStatus) {
		reason := fmt.Sprintf("transition from %s to %s is not allowed", fromStatus, toStatus)
		return &TransitionError{
			From:   fromStatus,
			To:     toStatus,
			Reason: reason,
		}
	}

	return nil
}

// GetJobValidTransitions returns all valid destination statuses for a specific job.
// Returns an error if the module or job doesn't exist.
func (m *Manager) GetJobValidTransitions(module, job string) ([]Status, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.state == nil {
		return nil, errors.New("M2003", "state not loaded")
	}

	// Get the module state
	moduleState, ok := m.state.Modules[module]
	if !ok {
		return nil, errors.New("M2003", "module not found: "+module)
	}

	// Get the job state
	jobState, ok := moduleState.Jobs[job]
	if !ok {
		return nil, errors.New("M2003", "job not found: "+job+" in module "+module)
	}

	return GetValidTransitions(jobState.Status), nil
}
