// Package state provides state management for Morty.
package state

import (
	"time"

	"github.com/morty/morty/pkg/errors"
)

// CurrentJob represents the currently executing job.
type CurrentJob struct {
	// Module is the name of the current module.
	Module string `json:"module"`
	// Job is the name of the current job.
	Job string `json:"job"`
	// Status is the current job status.
	Status Status `json:"status"`
	// StartedAt is when the job started execution.
	StartedAt time.Time `json:"started_at"`
}

// JobRef represents a reference to a job.
type JobRef struct {
	// Module is the module name.
	Module string `json:"module"`
	// Job is the job name.
	Job string `json:"job"`
	// Status is the job status.
	Status Status `json:"status"`
}

// Summary represents the statistics summary of all jobs.
type Summary struct {
	// TotalModules is the total number of modules.
	TotalModules int `json:"total_modules"`
	// TotalJobs is the total number of jobs across all modules.
	TotalJobs int `json:"total_jobs"`
	// Pending is the number of pending jobs.
	Pending int `json:"pending"`
	// Running is the number of running jobs.
	Running int `json:"running"`
	// Completed is the number of completed jobs.
	Completed int `json:"completed"`
	// Failed is the number of failed jobs.
	Failed int `json:"failed"`
	// Blocked is the number of blocked jobs.
	Blocked int `json:"blocked"`
	// Modules contains per-module statistics.
	Modules map[string]ModuleSummary `json:"modules"`
}

// ModuleSummary represents statistics for a specific module.
type ModuleSummary struct {
	// TotalJobs is the total number of jobs in the module.
	TotalJobs int `json:"total_jobs"`
	// Pending is the number of pending jobs.
	Pending int `json:"pending"`
	// Running is the number of running jobs.
	Running int `json:"running"`
	// Completed is the number of completed jobs.
	Completed int `json:"completed"`
	// Failed is the number of failed jobs.
	Failed int `json:"failed"`
	// Blocked is the number of blocked jobs.
	Blocked int `json:"blocked"`
}

// GetJobStatus returns the status of a specific job.
// Returns an error if the job does not exist.
func (m *Manager) GetJobStatus(module, job string) (Status, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.state == nil {
		return "", errors.New("M2003", "state not loaded")
	}

	moduleState, ok := m.state.Modules[module]
	if !ok {
		return "", errors.New("M2003", "module not found: "+module)
	}

	jobState, ok := moduleState.Jobs[job]
	if !ok {
		return "", errors.New("M2003", "job not found: "+job+" in module "+module)
	}

	return jobState.Status, nil
}

// UpdateJobStatus updates the status of a specific job.
// Also updates the module status and saves the state to file.
func (m *Manager) UpdateJobStatus(module, job string, status Status) error {
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
				m.mu.Unlock()
				err := m.UpdateJobStatusV2(moduleIndex, jobIndex, status)
				m.mu.Lock()
				return err
			}
		} else {
			statusV2Mu.RUnlock()
		}
		return errors.New("M2003", "state not loaded")
	}

	if !status.IsValid() {
		return errors.New("M2003", "invalid status: "+string(status))
	}

	moduleState, ok := m.state.Modules[module]
	if !ok {
		return errors.New("M2003", "module not found: "+module)
	}

	jobState, ok := moduleState.Jobs[job]
	if !ok {
		return errors.New("M2003", "job not found: "+job+" in module "+module)
	}

	// Update job status
	now := time.Now()
	jobState.Status = status
	jobState.UpdatedAt = now

	// Update module status and timestamp
	moduleState.UpdatedAt = now
	m.state.Global.LastUpdate = now

	// Save state to file
	m.mu.Unlock()
	err := m.Save()
	m.mu.Lock()

	return err
}

// GetCurrent returns the currently executing job.
// Returns nil if no job is currently running.
func (m *Manager) GetCurrent() (*CurrentJob, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.state == nil {
		return nil, errors.New("M2003", "state not loaded")
	}

	// If global state indicates a current module and job, return it
	if m.state.Global.CurrentModule != "" && m.state.Global.CurrentJob != "" {
		return &CurrentJob{
			Module: m.state.Global.CurrentModule,
			Job:    m.state.Global.CurrentJob,
			Status: m.state.Global.Status,
		}, nil
	}

	// Otherwise, search for a running job
	for moduleName, module := range m.state.Modules {
		for jobName, job := range module.Jobs {
			if job.Status == StatusRunning {
				return &CurrentJob{
					Module: moduleName,
					Job:    jobName,
					Status: StatusRunning,
				}, nil
			}
		}
	}

	return nil, nil
}

// SetCurrent sets the currently executing job.
// Updates the global state with the current module and job information.
// Pass empty strings for module and job to clear the current job.
func (m *Manager) SetCurrent(module, job string, status Status) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// V2 Compatible: If V1 state is nil, try V2
	if m.state == nil {
		// For V2, we update the global state in the status file
		statusV2Mu.Lock()
		if statusV2 != nil {
			// V2 doesn't have CurrentModule/CurrentJob in global state
			// The current job is tracked by the job with StatusRunning
			// So we don't need to do anything here for V2
			statusV2Mu.Unlock()
			// Just return success - V2 tracks current job via status
			return nil
		}
		statusV2Mu.Unlock()
		return errors.New("M2003", "state not loaded")
	}

	if !status.IsValid() {
		return errors.New("M2003", "invalid status: "+string(status))
	}

	// If both module and job are empty, this is a clear operation - skip validation
	if module != "" || job != "" {
		// Validate that the module and job exist
		moduleState, ok := m.state.Modules[module]
		if !ok {
			return errors.New("M2003", "module not found: "+module)
		}

		_, ok = moduleState.Jobs[job]
		if !ok {
			return errors.New("M2003", "job not found: "+job+" in module "+module)
		}
	}

	// Update global state
	now := time.Now()
	m.state.Global.CurrentModule = module
	m.state.Global.CurrentJob = job
	m.state.Global.Status = status
	m.state.Global.LastUpdate = now

	// Save state to file
	m.mu.Unlock()
	err := m.Save()
	m.mu.Lock()

	return err
}

// GetSummary returns statistics about all jobs.
func (m *Manager) GetSummary() (*Summary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.state == nil {
		return nil, errors.New("M2003", "state not loaded")
	}

	summary := &Summary{
		TotalModules: len(m.state.Modules),
		Modules:      make(map[string]ModuleSummary),
	}

	for moduleName, module := range m.state.Modules {
		moduleSummary := ModuleSummary{
			TotalJobs: len(module.Jobs),
		}

		for _, job := range module.Jobs {
			summary.TotalJobs++

			switch job.Status {
			case StatusPending:
				summary.Pending++
				moduleSummary.Pending++
			case StatusRunning:
				summary.Running++
				moduleSummary.Running++
			case StatusCompleted:
				summary.Completed++
				moduleSummary.Completed++
			case StatusFailed:
				summary.Failed++
				moduleSummary.Failed++
			case StatusBlocked:
				summary.Blocked++
				moduleSummary.Blocked++
			}
		}

		summary.Modules[moduleName] = moduleSummary
	}

	return summary, nil
}

// GetPendingJobs returns a list of all pending jobs.
func (m *Manager) GetPendingJobs() []JobRef {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.state == nil {
		return []JobRef{}
	}

	var pending []JobRef

	for moduleName, module := range m.state.Modules {
		for jobName, job := range module.Jobs {
			if job.Status == StatusPending {
				pending = append(pending, JobRef{
					Module: moduleName,
					Job:    jobName,
					Status: StatusPending,
				})
			}
		}
	}

	return pending
}

// ClearCurrent clears the currently executing job.
// This is a convenience method that calls SetCurrent with empty strings.
func (m *Manager) ClearCurrent() error {
	return m.SetCurrent("", "", StatusPending)
}

// UpdateTaskStatus updates the status of a specific task within a job.
func (m *Manager) UpdateTaskStatus(module, job string, taskIndex int, status Status) error {
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
				m.mu.Unlock()
				err := m.UpdateTaskStatusV2(moduleIndex, jobIndex, taskIndex, status)
				m.mu.Lock()
				return err
			}
		} else {
			statusV2Mu.RUnlock()
		}
		return errors.New("M2001", "state not loaded")
	}

	// Get the module
	moduleState, ok := m.state.Modules[module]
	if !ok {
		return errors.New("M2002", "module not found").
			WithDetail("module", module)
	}

	// Get the job
	jobState, ok := moduleState.Jobs[job]
	if !ok {
		return errors.New("M2003", "job not found").
			WithDetail("module", module).
			WithDetail("job", job)
	}

	// Validate task index (array index, 0-based)
	if taskIndex < 0 || taskIndex >= len(jobState.Tasks) {
		return errors.New("M2004", "task index out of range").
			WithDetail("module", module).
			WithDetail("job", job).
			WithDetail("task_index", taskIndex).
			WithDetail("tasks_total", len(jobState.Tasks))
	}

	// Update the task (taskIndex is array index, 0-based)
	jobState.Tasks[taskIndex].Status = status
	jobState.Tasks[taskIndex].UpdatedAt = time.Now()

	// Update job's updated_at timestamp
	jobState.UpdatedAt = time.Now()

	// Save the state
	m.mu.Unlock()
	err := m.Save()
	m.mu.Lock()

	return err
}

// UpdateTasksCompleted updates the tasks_completed count for a job.
func (m *Manager) UpdateTasksCompleted(module, job string, count int) error {
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
				// For V2, TasksCompleted is automatically calculated in UpdateTaskStatusV2
				// So we don't need to update it separately - just return success
				m.mu.Unlock()
				// Update the job status to trigger recalculation
				statusV2Mu.Lock()
				if moduleIndex < len(statusV2.Modules) && jobIndex < len(statusV2.Modules[moduleIndex].Jobs) {
					statusV2.Modules[moduleIndex].Jobs[jobIndex].TasksCompleted = count
				}
				statusV2Mu.Unlock()
				err := m.SaveV2(statusV2)
				m.mu.Lock()
				return err
			}
		} else {
			statusV2Mu.RUnlock()
		}
		return errors.New("M2001", "state not loaded")
	}

	// Get the module
	moduleState, ok := m.state.Modules[module]
	if !ok {
		return errors.New("M2002", "module not found").
			WithDetail("module", module)
	}

	// Get the job
	jobState, ok := moduleState.Jobs[job]
	if !ok {
		return errors.New("M2003", "job not found").
			WithDetail("module", module).
			WithDetail("job", job)
	}

	// Update tasks_completed count
	jobState.TasksCompleted = count
	jobState.UpdatedAt = time.Now()

	// Save the state
	m.mu.Unlock()
	err := m.Save()
	m.mu.Lock()

	return err
}
