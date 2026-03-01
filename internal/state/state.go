// Package state provides state management for Morty.
// It handles the persistence and retrieval of execution state,
// including module and job statuses, task completion tracking,
// and debug logs.
package state

import (
	"sync"
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

// TaskState represents the state of an individual task.
type TaskState struct {
	// Index is the task's position in the job's task list.
	Index int `json:"index"`
	// Status is the current status of the task.
	Status Status `json:"status"`
	// Description is the task description.
	Description string `json:"description"`
	// UpdatedAt is the timestamp of the last update.
	UpdatedAt time.Time `json:"updated_at"`
}

// JobState represents the state of a job within a module.
type JobState struct {
	// Name is the job identifier.
	Name string `json:"name"`
	// Status is the current execution status.
	Status Status `json:"status"`
	// LoopCount tracks how many execution loops have occurred.
	LoopCount int `json:"loop_count"`
	// RetryCount tracks retry attempts for failed jobs.
	RetryCount int `json:"retry_count"`
	// TasksTotal is the total number of tasks in the job.
	TasksTotal int `json:"tasks_total"`
	// TasksCompleted is the number of completed tasks.
	TasksCompleted int `json:"tasks_completed"`
	// FailureReason contains the error message if the job failed.
	FailureReason string `json:"failure_reason,omitempty"`
	// Interrupted indicates if the job was interrupted.
	Interrupted bool `json:"interrupted,omitempty"`
	// Tasks contains the state of individual tasks.
	Tasks []TaskState `json:"tasks,omitempty"`
	// DebugLogs contains debug entries for the job.
	DebugLogs []DebugLogEntry `json:"debug_logs,omitempty"`
	// CreatedAt is when the job was first added.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the timestamp of the last update.
	UpdatedAt time.Time `json:"updated_at"`
}

// DebugLogEntry represents a single debug log entry for a job.
type DebugLogEntry struct {
	// ID is a unique identifier for this debug entry.
	ID string `json:"id"`
	// Timestamp is when the entry was created.
	Timestamp time.Time `json:"timestamp"`
	// Phenomenon describes the issue observed.
	Phenomenon string `json:"phenomenon"`
	// Reproduction describes how to reproduce the issue.
	Reproduction string `json:"reproduction"`
	// Hypothesis lists possible causes.
	Hypothesis string `json:"hypothesis"`
	// Verification describes how to verify the hypothesis.
	Verification string `json:"verification"`
	// Fix describes the fix applied or planned.
	Fix string `json:"fix"`
	// Progress indicates the fix status.
	Progress string `json:"progress"`
}

// ModuleState represents the state of a module.
type ModuleState struct {
	// Name is the module identifier.
	Name string `json:"name"`
	// PlanFile is the actual plan file name (e.g., "test_hello_world.md").
	// This may differ from Name when the module has a Chinese name.
	PlanFile string `json:"plan_file,omitempty"`
	// Status is the overall module status.
	Status Status `json:"status"`
	// Jobs contains the states of jobs within this module.
	Jobs map[string]*JobState `json:"jobs"`
	// CreatedAt is when the module was first added.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the timestamp of the last update.
	UpdatedAt time.Time `json:"updated_at"`
}

// GlobalState represents the global execution state.
type GlobalState struct {
	// Status is the overall system status.
	Status Status `json:"status"`
	// CurrentModule is the currently executing module.
	CurrentModule string `json:"current_module,omitempty"`
	// CurrentJob is the currently executing job.
	CurrentJob string `json:"current_job,omitempty"`
	// StartTime is when the session started.
	StartTime time.Time `json:"start_time"`
	// LastUpdate is the timestamp of the last state update.
	LastUpdate time.Time `json:"last_update"`
	// TotalLoops tracks the total number of execution loops.
	TotalLoops int `json:"total_loops"`
}

// StatusJSON represents the complete state stored in status.json.
type StatusJSON struct {
	// Global contains the global execution state.
	Global GlobalState `json:"global"`
	// Modules contains states for all modules.
	Modules map[string]*ModuleState `json:"modules"`
	// Version is the state file format version.
	Version string `json:"version"`
}

const (
	// DefaultStateVersion is the current state file format version.
	DefaultStateVersion = "1.0"
)

// Manager handles state persistence and retrieval.
type Manager struct {
	// filePath is the path to the state file.
	filePath string
	// state is the current in-memory state.
	state *StatusJSON
	// mu protects state for thread-safe access.
	mu sync.RWMutex
}

// NewManager creates a new state manager with the given file path.
func NewManager(filePath string) *Manager {
	return &Manager{
		filePath: filePath,
		state:    nil,
	}
}

// String returns the string representation of a Status.
func (s Status) String() string {
	return string(s)
}

// IsValid checks if the status is a valid status value.
func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusRunning, StatusCompleted, StatusFailed, StatusBlocked:
		return true
	}
	return false
}
