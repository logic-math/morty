package cmd

import (
	"fmt"

	"github.com/morty/morty/internal/logging"
	"github.com/morty/morty/internal/state"
)

// selectNextJobV2 selects the next pending job from V2 status.
// Returns module index, job index, module name, job name, or error.
func (h *DoingHandler) selectNextJobV2() (int, int, string, string, error) {
	status := h.stateManager.GetStatusV2()
	if status == nil {
		return -1, -1, "", "", fmt.Errorf("status not loaded")
	}

	// Simply find the first PENDING job in the array
	moduleIndex, jobIndex := status.GetNextPendingJob()
	if moduleIndex == -1 {
		return -1, -1, "", "", fmt.Errorf("no pending jobs found")
	}

	module := &status.Modules[moduleIndex]
	job := &module.Jobs[jobIndex]

	h.logger.Info("Next job selected from V2 status",
		logging.Int("module_index", moduleIndex),
		logging.Int("job_index", jobIndex),
		logging.String("module", module.DisplayName),
		logging.String("job", job.Name),
	)

	return moduleIndex, jobIndex, module.Name, job.Name, nil
}

// updateJobStatusV2 updates job status in V2 format.
func (h *DoingHandler) updateJobStatusV2(moduleIndex, jobIndex int, newStatus state.Status) error {
	return h.stateManager.UpdateJobStatusV2(moduleIndex, jobIndex, newStatus)
}

// updateTaskStatusV2 updates task status in V2 format.
func (h *DoingHandler) updateTaskStatusV2(moduleIndex, jobIndex, taskIndex int, newStatus state.Status) error {
	return h.stateManager.UpdateTaskStatusV2(moduleIndex, jobIndex, taskIndex, newStatus)
}

// getJobStateV2 gets job state from V2 status.
func (h *DoingHandler) getJobStateV2(moduleIndex, jobIndex int) (*state.JobStateV2, error) {
	status := h.stateManager.GetStatusV2()
	if status == nil {
		return nil, fmt.Errorf("status not loaded")
	}

	if moduleIndex < 0 || moduleIndex >= len(status.Modules) {
		return nil, fmt.Errorf("invalid module index: %d", moduleIndex)
	}

	module := &status.Modules[moduleIndex]
	if jobIndex < 0 || jobIndex >= len(module.Jobs) {
		return nil, fmt.Errorf("invalid job index: %d", jobIndex)
	}

	return &module.Jobs[jobIndex], nil
}

// getModuleNameV2 gets module name from V2 status.
func (h *DoingHandler) getModuleNameV2(moduleIndex int) (string, error) {
	status := h.stateManager.GetStatusV2()
	if status == nil {
		return "", fmt.Errorf("status not loaded")
	}

	if moduleIndex < 0 || moduleIndex >= len(status.Modules) {
		return "", fmt.Errorf("invalid module index: %d", moduleIndex)
	}

	return status.Modules[moduleIndex].Name, nil
}

// getPlanFileV2 gets plan file name from V2 status.
func (h *DoingHandler) getPlanFileV2(moduleIndex int) (string, error) {
	status := h.stateManager.GetStatusV2()
	if status == nil {
		return "", fmt.Errorf("status not loaded")
	}

	if moduleIndex < 0 || moduleIndex >= len(status.Modules) {
		return "", fmt.Errorf("invalid module index: %d", moduleIndex)
	}

	return status.Modules[moduleIndex].PlanFile, nil
}
