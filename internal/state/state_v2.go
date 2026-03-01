// Package state provides state management for Morty V2.
// V2 uses array-based structure with topological ordering.
package state

import (
	"time"
)

// StatusV2 represents the V2 status format with topological ordering.
type StatusV2 struct {
	// Version is the format version ("2.0")
	Version string `json:"version"`
	// Global contains global state information
	Global GlobalStateV2 `json:"global"`
	// Modules is an ordered array of modules (topologically sorted)
	Modules []ModuleStateV2 `json:"modules"`
}

// GlobalStateV2 represents global execution state.
type GlobalStateV2 struct {
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

// ModuleStateV2 represents a module in V2 format.
type ModuleStateV2 struct {
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
	Jobs []JobStateV2 `json:"jobs"`
	// CreatedAt is when the module was added
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the last update timestamp
	UpdatedAt time.Time `json:"updated_at"`
}

// JobStateV2 represents a job in V2 format.
type JobStateV2 struct {
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
func (s *StatusV2) GetModuleByName(name string) *ModuleStateV2 {
	for i := range s.Modules {
		if s.Modules[i].Name == name || s.Modules[i].DisplayName == name {
			return &s.Modules[i]
		}
	}
	return nil
}

// GetJobByName finds a job by name within a module in V2 status.
func (m *ModuleStateV2) GetJobByName(name string) *JobStateV2 {
	for i := range m.Jobs {
		if m.Jobs[i].Name == name {
			return &m.Jobs[i]
		}
	}
	return nil
}

// GetNextPendingJob finds the next pending job in V2 status.
// Returns module index, job index, or -1, -1 if no pending job found.
func (s *StatusV2) GetNextPendingJob() (int, int) {
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
func (s *StatusV2) CountCompletedJobs() int {
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
func (s *StatusV2) CountCompletedModules() int {
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
