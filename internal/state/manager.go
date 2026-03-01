package state

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Manager provides status management operations.
type Manager struct {
	// filePath is the path to the status file
	filePath string
	// mu protects state for thread-safe access
	mu sync.RWMutex
}

// NewManager creates a new state manager with the given file path.
func NewManager(filePath string) *Manager {
	return &Manager{
		filePath: filePath,
	}
}

// status holds the current status (protected by mu)
var (
	status   *ExecutionStatus
	statusMu sync.RWMutex
)

// GetStatus returns the current status.
func (m *Manager) GetStatus() *ExecutionStatus {
	statusMu.RLock()
	defer statusMu.RUnlock()
	return status
}

// GetState is an alias for GetStatus (for compatibility).
func (m *Manager) GetState() *ExecutionStatus {
	return m.GetStatus()
}

// Load loads status from file.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	content, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty state
			statusMu.Lock()
			status = nil
			statusMu.Unlock()
			return nil
		}
		return fmt.Errorf("failed to read status file: %w", err)
	}

	var loadedStatus ExecutionStatus
	if err := json.Unmarshal(content, &loadedStatus); err != nil {
		return fmt.Errorf("failed to parse status JSON: %w", err)
	}

	statusMu.Lock()
	status = &loadedStatus
	statusMu.Unlock()

	return nil
}

// Save saves status to file.
func (m *Manager) Save(newStatus *ExecutionStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update in-memory state
	statusMu.Lock()
	status = newStatus
	statusMu.Unlock()

	// Marshal to JSON
	data, err := json.MarshalIndent(newStatus, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	// Write to file
	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write status file: %w", err)
	}

	return nil
}

// Initialize initializes V2 status from plan files.
func (m *Manager) Initialize(planDir string) error {
	// Generate V2 status
	status, err := GenerateStatus(planDir)
	if err != nil {
		return fmt.Errorf("failed to generate status: %w", err)
	}

	// Save to file
	return m.Save(status)
}

// UpdateJobStatus updates a job's status in V2 format.
func (m *Manager) UpdateJobStatus(moduleIndex, jobIndex int, newStatus Status) error {
	statusMu.Lock()
	defer statusMu.Unlock()

	if status == nil {
		return fmt.Errorf("status not loaded")
	}

	if moduleIndex < 0 || moduleIndex >= len(status.Modules) {
		return fmt.Errorf("invalid module index: %d", moduleIndex)
	}

	module := &status.Modules[moduleIndex]
	if jobIndex < 0 || jobIndex >= len(module.Jobs) {
		return fmt.Errorf("invalid job index: %d", jobIndex)
	}

	job := &module.Jobs[jobIndex]
	now := time.Now()

	// Update job status
	job.Status = newStatus
	job.UpdatedAt = now

	// Update module status
	module.UpdatedAt = now

	// Update global status
	status.Global.LastUpdate = now
	if newStatus == StatusRunning {
		status.Global.Status = StatusRunning
		status.Global.CurrentModuleIndex = moduleIndex
		status.Global.CurrentJobIndex = job.GlobalIndex
	} else if newStatus == StatusCompleted {
		// Check if all jobs are completed
		allCompleted := true
		for _, mod := range status.Modules {
			for _, j := range mod.Jobs {
				if j.Status != StatusCompleted {
					allCompleted = false
					break
				}
			}
			if !allCompleted {
				break
			}
		}
		if allCompleted {
			status.Global.Status = StatusCompleted
		}
	} else if newStatus == StatusFailed {
		status.Global.Status = StatusFailed
	}

	// Save to file
	// Note: Save will handle its own locking
	statusMu.Unlock()
	err := m.Save(status)
	statusMu.Lock()

	return err
}

// UpdateTaskStatus updates a task's status in V2 format.
func (m *Manager) UpdateTaskStatus(moduleIndex, jobIndex, taskIndex int, newStatus Status) error {
	statusMu.Lock()
	defer statusMu.Unlock()

	if status == nil {
		return fmt.Errorf("status not loaded")
	}

	if moduleIndex < 0 || moduleIndex >= len(status.Modules) {
		return fmt.Errorf("invalid module index: %d", moduleIndex)
	}

	module := &status.Modules[moduleIndex]
	if jobIndex < 0 || jobIndex >= len(module.Jobs) {
		return fmt.Errorf("invalid job index: %d", jobIndex)
	}

	job := &module.Jobs[jobIndex]
	if taskIndex < 0 || taskIndex >= len(job.Tasks) {
		return fmt.Errorf("invalid task index: %d", taskIndex)
	}

	task := &job.Tasks[taskIndex]
	now := time.Now()

	// Update task status
	task.Status = newStatus
	task.UpdatedAt = now

	// Update job's completed count
	if newStatus == StatusCompleted {
		// Recalculate completed tasks
		completed := 0
		for _, t := range job.Tasks {
			if t.Status == StatusCompleted {
				completed++
			}
		}
		job.TasksCompleted = completed
	}

	// Update timestamps
	job.UpdatedAt = now
	module.UpdatedAt = now
	status.Global.LastUpdate = now

	// Save to file
	// Note: Save will handle its own locking
	statusMu.Unlock()
	err := m.Save(status)
	statusMu.Lock()

	return err
}

// DetectVersion detects the status file version.
func DetectVersion(statusFile string) (string, error) {
	content, err := os.ReadFile(statusFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No file, no version
		}
		return "", err
	}

	var versionCheck struct {
		Version string `json:"version"`
	}

	if err := json.Unmarshal(content, &versionCheck); err != nil {
		return "", err
	}

	if versionCheck.Version == "2.0" {
		return "2.0", nil
	}

	// Check if it's V1 format (has "modules" as object)
	var v1Check struct {
		Modules map[string]interface{} `json:"modules"`
	}
	if err := json.Unmarshal(content, &v1Check); err == nil && v1Check.Modules != nil {
		return "1.0", nil
	}

	return "", fmt.Errorf("unknown status file format")
}

// GetJob gets a job from status by module and job name.
func (m *Manager) GetJob(moduleName, jobName string) *JobState {
	statusMu.RLock()
	defer statusMu.RUnlock()

	if status == nil {
		return nil
	}

	// Find module by name
	var module *ModuleState
	for i := range status.Modules {
		if status.Modules[i].Name == moduleName {
			module = &status.Modules[i]
			break
		}
	}

	if module == nil {
		return nil
	}

	// Find job by name
	for i := range module.Jobs {
		if module.Jobs[i].Name == jobName {
			return &module.Jobs[i]
		}
	}

	return nil
}

// GetJobStatus gets the status of a job by module and job name.
func (m *Manager) GetJobStatus(moduleName, jobName string) (Status, error) {
	job := m.GetJob(moduleName, jobName)
	if job == nil {
		return "", fmt.Errorf("job not found: %s/%s", moduleName, jobName)
	}
	return job.Status, nil
}

// findJobIndices finds module and job indices by name.
func (m *Manager) findJobIndices(moduleName, jobName string) (int, int, error) {
	statusMu.RLock()
	defer statusMu.RUnlock()

	if status == nil {
		return -1, -1, fmt.Errorf("status not loaded")
	}

	// Find module by name
	moduleIndex := -1
	for i := range status.Modules {
		if status.Modules[i].Name == moduleName {
			moduleIndex = i
			break
		}
	}

	if moduleIndex == -1 {
		return -1, -1, fmt.Errorf("module not found: %s", moduleName)
	}

	// Find job by name
	jobIndex := -1
	for i := range status.Modules[moduleIndex].Jobs {
		if status.Modules[moduleIndex].Jobs[i].Name == jobName {
			jobIndex = i
			break
		}
	}

	if jobIndex == -1 {
		return -1, -1, fmt.Errorf("job not found: %s in module %s", jobName, moduleName)
	}

	return moduleIndex, jobIndex, nil
}

// UpdateJobStatusByName updates a job's status by module and job name.
func (m *Manager) UpdateJobStatusByName(moduleName, jobName string, newStatus Status) error {
	moduleIndex, jobIndex, err := m.findJobIndices(moduleName, jobName)
	if err != nil {
		return err
	}
	return m.UpdateJobStatus(moduleIndex, jobIndex, newStatus)
}

// UpdateTaskStatusByName updates a task's status by module name, job name, and task index.
func (m *Manager) UpdateTaskStatusByName(moduleName, jobName string, taskIndex int, newStatus Status) error {
	moduleIndex, jobIndex, err := m.findJobIndices(moduleName, jobName)
	if err != nil {
		return err
	}
	return m.UpdateTaskStatus(moduleIndex, jobIndex, taskIndex, newStatus)
}

// UpdateTasksCompleted updates the completed task count for a job.
func (m *Manager) UpdateTasksCompleted(moduleName, jobName string, count int) error {
	statusMu.Lock()
	defer statusMu.Unlock()

	if status == nil {
		return fmt.Errorf("status not loaded")
	}

	// Find module by name
	var module *ModuleState
	for i := range status.Modules {
		if status.Modules[i].Name == moduleName {
			module = &status.Modules[i]
			break
		}
	}

	if module == nil {
		return fmt.Errorf("module not found: %s", moduleName)
	}

	// Find job by name
	var job *JobState
	for i := range module.Jobs {
		if module.Jobs[i].Name == jobName {
			job = &module.Jobs[i]
			break
		}
	}

	if job == nil {
		return fmt.Errorf("job not found: %s in module %s", jobName, moduleName)
	}

	// Update count
	job.TasksCompleted = count
	job.UpdatedAt = time.Now()

	// Save to file
	statusMu.Unlock()
	err := m.Save(status)
	statusMu.Lock()

	return err
}

// SetCurrent sets the current executing job (for compatibility).
// In V2, this is tracked via Global.CurrentModuleIndex and Global.CurrentJobIndex.
func (m *Manager) SetCurrent(moduleName, jobName string, newStatus Status) error {
	// This is now handled by UpdateJobStatus which updates Global state
	return m.UpdateJobStatusByName(moduleName, jobName, newStatus)
}

// ClearCurrent clears the current job marker (for compatibility).
// In V2, we don't need to explicitly clear - the Global state tracks progress.
func (m *Manager) ClearCurrent() error {
	// No-op in V2 - current job is tracked via indices
	return nil
}

// TransitionJobStatus transitions a job to a new status with validation.
func (m *Manager) TransitionJobStatus(moduleName, jobName string, toStatus Status, logger interface{}) error {
	// For now, just update the status
	// TODO: Add validation logic if needed
	return m.UpdateJobStatusByName(moduleName, jobName, toStatus)
}

// UpdateFailureReason updates the failure reason for a job.
func (m *Manager) UpdateFailureReason(moduleName, jobName, reason string) error {
	statusMu.Lock()
	defer statusMu.Unlock()

	if status == nil {
		return fmt.Errorf("status not loaded")
	}

	// Find module by name
	var module *ModuleState
	for i := range status.Modules {
		if status.Modules[i].Name == moduleName {
			module = &status.Modules[i]
			break
		}
	}

	if module == nil {
		return fmt.Errorf("module not found: %s", moduleName)
	}

	// Find job by name
	var job *JobState
	for i := range module.Jobs {
		if module.Jobs[i].Name == jobName {
			job = &module.Jobs[i]
			break
		}
	}

	if job == nil {
		return fmt.Errorf("job not found: %s in module %s", jobName, moduleName)
	}

	// Update failure reason
	job.FailureReason = reason
	job.UpdatedAt = time.Now()

	// Save to file
	statusMu.Unlock()
	err := m.Save(status)
	statusMu.Lock()

	return err
}
