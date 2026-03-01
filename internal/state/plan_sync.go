package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/morty/morty/internal/parser/plan"
)

// SyncFromPlanDir scans the plan directory and initializes/updates state
// for all modules found in plan files.
// This is used to bootstrap state when status.json doesn't exist or is empty.
func (m *Manager) SyncFromPlanDir(planDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure state is initialized
	if m.state == nil {
		m.state = m.createDefaultState()
	}

	// Ensure Modules map is initialized
	if m.state.Modules == nil {
		m.state.Modules = make(map[string]*ModuleState)
	}

	// Scan plan directory for .md files
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return fmt.Errorf("failed to read plan directory: %w", err)
	}

	now := time.Now()
	modulesAdded := 0
	modulesUpdated := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") || strings.HasPrefix(name, "README") {
			continue
		}

		// Read plan file
		planPath := filepath.Join(planDir, name)
		content, err := os.ReadFile(planPath)
		if err != nil {
			continue // Skip files that can't be read
		}

		// Parse plan
		parsedPlan, err := plan.ParsePlan(string(content))
		if err != nil {
			continue // Skip files that can't be parsed
		}

		// Check if module already exists
		if existingModule, exists := m.state.Modules[parsedPlan.Name]; exists {
			// Update PlanFile if not set
			if existingModule.PlanFile == "" {
				existingModule.PlanFile = name
				existingModule.UpdatedAt = now
				modulesUpdated++
			}
			continue
		}

		// Create module state
		moduleState := &ModuleState{
			Name:      parsedPlan.Name,
			PlanFile:  name, // Store the actual file name
			Status:    StatusPending,
			Jobs:      make(map[string]*JobState),
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Add jobs from plan
		for _, job := range parsedPlan.Jobs {
			// Create task states
			tasks := make([]TaskState, 0, len(job.Tasks))
			for _, task := range job.Tasks {
				taskState := TaskState{
					Index:       task.Index,
					Status:      StatusPending,
					Description: task.Description,
					UpdatedAt:   now,
				}
				tasks = append(tasks, taskState)
			}

			jobState := &JobState{
				Name:           job.Name,
				Status:         StatusPending,
				LoopCount:      0,
				RetryCount:     0,
				TasksTotal:     len(job.Tasks),
				TasksCompleted: 0,
				Tasks:          tasks,
				DebugLogs:      []DebugLogEntry{},
				CreatedAt:      now,
				UpdatedAt:      now,
			}

			moduleState.Jobs[job.Name] = jobState
		}

		m.state.Modules[parsedPlan.Name] = moduleState
		modulesAdded++
	}

	if modulesAdded > 0 || modulesUpdated > 0 {
		// Save the updated state
		m.mu.Unlock()
		err := m.Save()
		m.mu.Lock()
		return err
	}

	return nil
}

// SyncModuleFromPlan syncs a specific module from its plan file.
// This is useful when a module is requested but doesn't exist in state.
func (m *Manager) SyncModuleFromPlan(planDir, moduleName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if module already exists
	if _, exists := m.state.Modules[moduleName]; exists {
		return nil // Already exists, nothing to do
	}

	// Try to find the plan file for this module
	// Look for files matching the module name (sanitized)
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return fmt.Errorf("failed to read plan directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		// Read and parse plan file
		planPath := filepath.Join(planDir, name)
		content, err := os.ReadFile(planPath)
		if err != nil {
			continue
		}

		parsedPlan, err := plan.ParsePlan(string(content))
		if err != nil {
			continue
		}

		// Check if this is the module we're looking for
		if parsedPlan.Name != moduleName {
			continue
		}

		// Found the module! Initialize it
		now := time.Now()
		moduleState := &ModuleState{
			Name:      parsedPlan.Name,
			Status:    StatusPending,
			Jobs:      make(map[string]*JobState),
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Add jobs
		for _, job := range parsedPlan.Jobs {
			// Create task states
			tasks := make([]TaskState, 0, len(job.Tasks))
			for _, task := range job.Tasks {
				taskState := TaskState{
					Index:       task.Index,
					Status:      StatusPending,
					Description: task.Description,
					UpdatedAt:   now,
				}
				tasks = append(tasks, taskState)
			}

			jobState := &JobState{
				Name:           job.Name,
				Status:         StatusPending,
				LoopCount:      0,
				RetryCount:     0,
				TasksTotal:     len(job.Tasks),
				TasksCompleted: 0,
				Tasks:          tasks,
				DebugLogs:      []DebugLogEntry{},
				CreatedAt:      now,
				UpdatedAt:      now,
			}

			moduleState.Jobs[job.Name] = jobState
		}

		m.state.Modules[parsedPlan.Name] = moduleState

		// Save the updated state
		m.mu.Unlock()
		err = m.Save()
		m.mu.Lock()
		return err
	}

	return fmt.Errorf("plan file not found for module: %s", moduleName)
}
