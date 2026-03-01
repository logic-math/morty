package state

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Manager V2 methods for handling V2 status format.

// statusV2 holds the V2 status (protected by mu)
var (
	statusV2   *StatusV2
	statusV2Mu sync.RWMutex
)

// GetStatusV2 returns the current V2 status.
func (m *Manager) GetStatusV2() *StatusV2 {
	statusV2Mu.RLock()
	defer statusV2Mu.RUnlock()
	return statusV2
}

// LoadV2 loads V2 status from file.
func (m *Manager) LoadV2() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	content, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty state
			statusV2Mu.Lock()
			statusV2 = nil
			statusV2Mu.Unlock()
			return nil
		}
		return fmt.Errorf("failed to read status file: %w", err)
	}

	var status StatusV2
	if err := json.Unmarshal(content, &status); err != nil {
		return fmt.Errorf("failed to parse status JSON: %w", err)
	}

	statusV2Mu.Lock()
	statusV2 = &status
	statusV2Mu.Unlock()

	return nil
}

// SaveV2 saves V2 status to file.
func (m *Manager) SaveV2(status *StatusV2) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update in-memory state
	statusV2Mu.Lock()
	statusV2 = status
	statusV2Mu.Unlock()

	// Marshal to JSON
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	// Write to file
	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write status file: %w", err)
	}

	return nil
}

// InitializeV2 initializes V2 status from plan files.
func (m *Manager) InitializeV2(planDir string) error {
	// Generate V2 status
	status, err := GenerateStatusV2(planDir)
	if err != nil {
		return fmt.Errorf("failed to generate status: %w", err)
	}

	// Save to file
	return m.SaveV2(status)
}

// UpdateJobStatusV2 updates a job's status in V2 format.
func (m *Manager) UpdateJobStatusV2(moduleIndex, jobIndex int, newStatus Status) error {
	statusV2Mu.Lock()
	defer statusV2Mu.Unlock()

	if statusV2 == nil {
		return fmt.Errorf("status not loaded")
	}

	if moduleIndex < 0 || moduleIndex >= len(statusV2.Modules) {
		return fmt.Errorf("invalid module index: %d", moduleIndex)
	}

	module := &statusV2.Modules[moduleIndex]
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
	statusV2.Global.LastUpdate = now
	if newStatus == StatusRunning {
		statusV2.Global.Status = StatusRunning
		statusV2.Global.CurrentModuleIndex = moduleIndex
		statusV2.Global.CurrentJobIndex = job.GlobalIndex
	} else if newStatus == StatusCompleted {
		// Check if all jobs are completed
		allCompleted := true
		for _, mod := range statusV2.Modules {
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
			statusV2.Global.Status = StatusCompleted
		}
	} else if newStatus == StatusFailed {
		statusV2.Global.Status = StatusFailed
	}

	// Save to file
	// Note: SaveV2 will handle its own locking
	statusV2Mu.Unlock()
	err := m.SaveV2(statusV2)
	statusV2Mu.Lock()

	return err
}

// UpdateTaskStatusV2 updates a task's status in V2 format.
func (m *Manager) UpdateTaskStatusV2(moduleIndex, jobIndex, taskIndex int, newStatus Status) error {
	statusV2Mu.Lock()
	defer statusV2Mu.Unlock()

	if statusV2 == nil {
		return fmt.Errorf("status not loaded")
	}

	if moduleIndex < 0 || moduleIndex >= len(statusV2.Modules) {
		return fmt.Errorf("invalid module index: %d", moduleIndex)
	}

	module := &statusV2.Modules[moduleIndex]
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
	statusV2.Global.LastUpdate = now

	// Save to file
	// Note: SaveV2 will handle its own locking
	statusV2Mu.Unlock()
	err := m.SaveV2(statusV2)
	statusV2Mu.Lock()

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

// GetJobV2Compatible gets a job from V2 status and converts it to V1 JobState format.
// This is for backward compatibility with executor code that expects V1 format.
func (m *Manager) GetJobV2Compatible(moduleName, jobName string) *JobState {
	statusV2Mu.RLock()
	defer statusV2Mu.RUnlock()

	if statusV2 == nil {
		return nil
	}

	// Find module by name
	var moduleV2 *ModuleStateV2
	for i := range statusV2.Modules {
		if statusV2.Modules[i].Name == moduleName {
			moduleV2 = &statusV2.Modules[i]
			break
		}
	}

	if moduleV2 == nil {
		return nil
	}

	// Find job by name
	var jobV2 *JobStateV2
	for i := range moduleV2.Jobs {
		if moduleV2.Jobs[i].Name == jobName {
			jobV2 = &moduleV2.Jobs[i]
			break
		}
	}

	if jobV2 == nil {
		return nil
	}

	// Convert V2 JobStateV2 to V1 JobState
	// Note: V1 JobState has fewer fields than V2, so we only map the common ones
	jobV1 := &JobState{
		Name:           jobV2.Name,
		Status:         jobV2.Status,
		TasksTotal:     jobV2.TasksTotal,
		TasksCompleted: jobV2.TasksCompleted,
		LoopCount:      jobV2.LoopCount,
		RetryCount:     jobV2.RetryCount,
		CreatedAt:      jobV2.CreatedAt,
		UpdatedAt:      jobV2.UpdatedAt,
	}

	// Convert tasks
	jobV1.Tasks = make([]TaskState, len(jobV2.Tasks))
	for i, taskV2 := range jobV2.Tasks {
		jobV1.Tasks[i] = TaskState{
			Description: taskV2.Description,
			Status:      taskV2.Status,
			UpdatedAt:   taskV2.UpdatedAt,
		}
	}

	return jobV1
}
