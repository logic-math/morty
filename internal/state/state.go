// Package state provides state management for Morty.
// Uses array-based structure with topological ordering for efficient execution.
package state

import (
	"time"
)

// Status represents the execution status of a job or module.
type Status string

// Status constants for job and module states.
const (
	// StatusPending indicates the job/module is waiting to start.
	StatusPending Status = "PENDING"
	// StatusRunning indicates the job/module is currently executing.
	StatusRunning Status = "RUNNING"
	// StatusCompleted indicates the job/module completed successfully.
	StatusCompleted Status = "COMPLETED"
	// StatusFailed indicates the job/module failed.
	StatusFailed Status = "FAILED"
	// StatusBlocked indicates the job/module is blocked by dependencies.
	StatusBlocked Status = "BLOCKED"
)

// IsValid checks if the status is a valid value.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusRunning, StatusCompleted, StatusFailed, StatusBlocked:
		return true
	default:
		return false
	}
}

// TaskState represents the state of an individual task.
type TaskState struct {
	// Index is the task's position in the job's task list (1-based)
	Index int `json:"index"`
	// Status is the current status of the task
	Status Status `json:"status"`
	// Description is the task description
	Description string `json:"description"`
	// UpdatedAt is the timestamp of the last update
	UpdatedAt time.Time `json:"updated_at"`
}

// DebugLogEntry represents a debug log entry for a job.
type DebugLogEntry struct {
	// ID is a unique identifier for this debug entry
	ID string `json:"id"`
	// Timestamp is when the entry was created
	Timestamp time.Time `json:"timestamp"`
	// Phenomenon describes the issue observed
	Phenomenon string `json:"phenomenon"`
	// Reproduction describes how to reproduce the issue
	Reproduction string `json:"reproduction"`
	// Hypothesis lists possible causes
	Hypothesis string `json:"hypothesis"`
}

// ExecutionStatus represents the overall execution status format.
// Modules and jobs are stored in arrays, topologically sorted at generation time.
type ExecutionStatus struct {
	// Version is the format version ("2.0")
	Version string `json:"version"`
	// Global contains global state information
	Global GlobalState `json:"global"`
	// Modules is an ordered array of modules (topologically sorted)
	Modules []ModuleState `json:"modules"`
}

// GlobalState represents global execution state.
type GlobalState struct {
	// Status is the overall execution status
	Status Status `json:"status"`
	// StartTime is when execution started
	StartTime time.Time `json:"start_time"`
	// LastUpdate is the last update timestamp
	LastUpdate time.Time `json:"last_update"`
	// CurrentModuleIndex is the index of currently executing module
	CurrentModuleIndex int `json:"current_module_index"`
	// CurrentJobIndex is the global index of currently executing job
	CurrentJobIndex int `json:"current_job_index"`
	// TotalModules is the total number of modules
	TotalModules int `json:"total_modules"`
	// TotalJobs is the total number of jobs across all modules
	TotalJobs int `json:"total_jobs"`
}

// ModuleState represents a module in V2 format.
type ModuleState struct {
	// Index is the module's position in topological order
	Index int `json:"index"`
	// Name is the module identifier (filename without .md)
	Name string `json:"name"`
	// DisplayName is the human-readable name (may be Chinese)
	DisplayName string `json:"display_name"`
	// PlanFile is the actual plan filename
	PlanFile string `json:"plan_file"`
	// Status is the module status
	Status Status `json:"status"`
	// Dependencies is the list of module names this module depends on
	Dependencies []string `json:"dependencies"`
	// Jobs is an ordered array of jobs (topologically sorted)
	Jobs []JobState `json:"jobs"`
	// CreatedAt is when the module was added
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the last update timestamp
	UpdatedAt time.Time `json:"updated_at"`
}

// JobState represents a job in V2 format.
type JobState struct {
	// Index is the job's position within the module (topologically sorted)
	Index int `json:"index"`
	// GlobalIndex is the job's position in the global job array
	GlobalIndex int `json:"global_index"`
	// Name is the job name
	Name string `json:"name"`
	// Status is the job status
	Status Status `json:"status"`
	// Prerequisites is the original prerequisite list (for display only)
	Prerequisites []string `json:"prerequisites,omitempty"`
	// TasksTotal is the total number of tasks
	TasksTotal int `json:"tasks_total"`
	// TasksCompleted is the number of completed tasks
	TasksCompleted int `json:"tasks_completed"`
	// LoopCount is the number of execution loops
	LoopCount int `json:"loop_count"`
	// RetryCount is the number of retries
	RetryCount int `json:"retry_count"`
	// FailureReason contains error message if failed
	FailureReason string `json:"failure_reason,omitempty"`
	// Tasks contains the task states
	Tasks []TaskState `json:"tasks,omitempty"`
	// DebugLogs contains debug entries
	DebugLogs []DebugLogEntry `json:"debug_logs,omitempty"`
	// CreatedAt is when the job was added
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the last update timestamp
	UpdatedAt time.Time `json:"updated_at"`
}

// GetModuleByName finds a module by name in V2 status.
func (s *ExecutionStatus) GetModuleByName(name string) *ModuleState {
	for i := range s.Modules {
		if s.Modules[i].Name == name || s.Modules[i].DisplayName == name {
			return &s.Modules[i]
		}
	}
	return nil
}

// GetJobByName finds a job by name within a module in V2 status.
func (m *ModuleState) GetJobByName(name string) *JobState {
	for i := range m.Jobs {
		if m.Jobs[i].Name == name {
			return &m.Jobs[i]
		}
	}
	return nil
}

// GetNextPendingJob finds the next pending job in V2 status.
// Returns module index, job index, or -1, -1 if no pending job found.
func (s *ExecutionStatus) GetNextPendingJob() (int, int) {
	for mi, module := range s.Modules {
		for ji, job := range module.Jobs {
			if job.Status == StatusPending {
				return mi, ji
			}
		}
	}
	return -1, -1
}

// CountCompletedJobs counts the number of completed jobs in V2 status.
func (s *ExecutionStatus) CountCompletedJobs() int {
	count := 0
	for _, module := range s.Modules {
		for _, job := range module.Jobs {
			if job.Status == StatusCompleted {
				count++
			}
		}
	}
	return count
}

// CountCompletedModules counts the number of fully completed modules in V2 status.
func (s *ExecutionStatus) CountCompletedModules() int {
	count := 0
	for _, module := range s.Modules {
		allCompleted := true
		for _, job := range module.Jobs {
			if job.Status != StatusCompleted {
				allCompleted = false
				break
			}
		}
		if allCompleted && len(module.Jobs) > 0 {
			count++
		}
	}
	return count
}
