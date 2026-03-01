// Package state provides state management for Morty.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/morty/morty/pkg/errors"
)

// backupSuffix is the suffix appended to backup files.
const backupSuffix = ".backup"

// backupTimeFormat is the format for backup file timestamps.
const backupTimeFormat = "20060102_150405"

// Load loads the state from the state file.
// V2 format is now the default. V1 format is no longer supported.
// If the file does not exist, returns an error asking to run init-status.
func (m *Manager) Load() error {
	// Use V2 load directly
	return m.LoadV2()
}

// Save saves the current state to the state file.
// It creates the parent directory if it doesn't exist and formats JSON with indentation.
func (m *Manager) Save() error {
	m.mu.RLock()
	state := m.state
	m.mu.RUnlock()

	if state == nil {
		// Initialize default state if none exists
		m.mu.Lock()
		m.state = m.createDefaultState()
		state = m.state
		m.mu.Unlock()
	}

	// Ensure directory exists
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, "M2001", "failed to create state directory")
	}

	// Marshal JSON with indentation for readability
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.Wrap(err, "M2002", "failed to marshal state")
	}

	// Write to temporary file first for atomic operation
	tempFile := m.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return errors.Wrap(err, "M2001", "failed to write state file")
	}

	// Atomic rename
	if err := os.Rename(tempFile, m.filePath); err != nil {
		// Clean up temp file on error
		os.Remove(tempFile)
		return errors.Wrap(err, "M2001", "failed to rename state file")
	}

	return nil
}

// GetState returns the current state.
// Returns nil if Load has not been called.
func (m *Manager) GetState() *StatusJSON {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

// SetState updates the current state.
func (m *Manager) SetState(state *StatusJSON) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = state
}

// createDefaultState creates a new default state structure.
func (m *Manager) createDefaultState() *StatusJSON {
	now := time.Now()
	return &StatusJSON{
		Version: DefaultStateVersion,
		Global: GlobalState{
			Status:     StatusPending,
			StartTime:  now,
			LastUpdate: now,
			TotalLoops: 0,
		},
		Modules: make(map[string]*ModuleState),
	}
}

// validateState validates the state structure.
// Returns an error if the state is corrupted or invalid.
func validateState(state *StatusJSON) error {
	// Check version
	if state.Version == "" {
		return errors.New("M2003", "state version is missing")
	}

	// Validate global state
	if !state.Global.Status.IsValid() {
		return errors.New("M2003", fmt.Sprintf("invalid global status: %s", state.Global.Status))
	}

	// Validate modules
	for name, module := range state.Modules {
		if module == nil {
			return errors.New("M2003", fmt.Sprintf("module %s is nil", name))
		}
		if module.Name == "" {
			module.Name = name
		}
		if !module.Status.IsValid() {
			return errors.New("M2003", fmt.Sprintf("invalid status for module %s: %s", name, module.Status))
		}
		// Validate jobs
		for jobName, job := range module.Jobs {
			if job == nil {
				return errors.New("M2003", fmt.Sprintf("job %s in module %s is nil", jobName, name))
			}
			if job.Name == "" {
				job.Name = jobName
			}
			if !job.Status.IsValid() {
				return errors.New("M2003", fmt.Sprintf("invalid status for job %s in module %s: %s", jobName, name, job.Status))
			}
		}
	}

	return nil
}

// Backup creates a backup of the current state file.
// The backup file is named with a timestamp suffix.
// Returns the path to the backup file.
func (m *Manager) Backup() (string, error) {
	// Check if state file exists
	_, err := os.Stat(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("M2001", "state file does not exist, cannot backup")
		}
		return "", errors.Wrap(err, "M2001", "failed to stat state file")
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format(backupTimeFormat)
	ext := filepath.Ext(m.filePath)
	base := m.filePath[:len(m.filePath)-len(ext)]
	backupPath := fmt.Sprintf("%s_%s%s%s", base, timestamp, backupSuffix, ext)

	// Handle filename collision by adding a counter
	counter := 1
	for {
		_, err := os.Stat(backupPath)
		if err != nil {
			if os.IsNotExist(err) {
				break // File doesn't exist, we can use this name
			}
			return "", errors.Wrap(err, "M2001", "failed to check backup file")
		}
		// File exists, try next counter
		backupPath = fmt.Sprintf("%s_%s_%d%s%s", base, timestamp, counter, backupSuffix, ext)
		counter++
		if counter > 1000 {
			return "", errors.New("M2001", "too many backup file collisions")
		}
	}

	// Read current state file
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return "", errors.Wrap(err, "M2001", "failed to read state file for backup")
	}

	// Write backup file
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", errors.Wrap(err, "M2001", "failed to write backup file")
	}

	return backupPath, nil
}

// BackupWithCustomPath creates a backup at the specified path.
func (m *Manager) BackupWithCustomPath(backupPath string) error {
	m.mu.RLock()
	state := m.state
	m.mu.RUnlock()

	if state == nil {
		return errors.New("M2001", "no state loaded, cannot backup")
	}

	// Ensure directory exists
	dir := filepath.Dir(backupPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.Wrap(err, "M2001", "failed to create backup directory")
	}

	// Marshal and write
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.Wrap(err, "M2002", "failed to marshal state for backup")
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return errors.Wrap(err, "M2001", "failed to write backup file")
	}

	return nil
}

// ListBackups returns a list of backup files for the current state file.
func (m *Manager) ListBackups() ([]string, error) {
	dir := filepath.Dir(m.filePath)
	base := filepath.Base(m.filePath)
	ext := filepath.Ext(base)
	prefix := base[:len(base)-len(ext)] + "_"
	suffix := backupSuffix + ext

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, errors.Wrap(err, "M2001", "failed to list backup directory")
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) > len(prefix)+len(suffix) &&
			name[:len(prefix)] == prefix &&
			name[len(name)-len(suffix):] == suffix {
			backups = append(backups, filepath.Join(dir, name))
		}
	}

	return backups, nil
}

// RestoreFromBackup restores state from a backup file.
func (m *Manager) RestoreFromBackup(backupPath string) error {
	// Read backup file
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return errors.Wrap(err, "M2001", "failed to read backup file")
	}

	// Parse JSON
	var state StatusJSON
	if err := json.Unmarshal(data, &state); err != nil {
		return errors.Wrap(err, "M2002", "failed to parse backup file")
	}

	// Validate
	if err := validateState(&state); err != nil {
		return err
	}

	// Update state
	m.mu.Lock()
	m.state = &state
	m.mu.Unlock()

	// Save to current state file
	return m.Save()
}

// GetModule returns the state for a specific module.
// Returns nil if the module does not exist.
func (m *Manager) GetModule(name string) *ModuleState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state == nil {
		return nil
	}
	return m.state.Modules[name]
}

// GetJob returns the state for a specific job within a module.
// Returns nil if the module or job does not exist.
// V2 Compatible: If V1 state is nil, tries to get from V2 status.
func (m *Manager) GetJob(moduleName, jobName string) *JobState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try V1 format first
	if m.state != nil {
		module, ok := m.state.Modules[moduleName]
		if ok {
			return module.Jobs[jobName]
		}
		return nil
	}

	// Fall back to V2 format
	return m.GetJobV2Compatible(moduleName, jobName)
}

// SetModule creates or updates a module state.
func (m *Manager) SetModule(module *ModuleState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == nil {
		return
	}
	module.UpdatedAt = time.Now()
	m.state.Modules[module.Name] = module
	m.state.Global.LastUpdate = time.Now()
}

// SetJob creates or updates a job state within a module.
func (m *Manager) SetJob(moduleName string, job *JobState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state == nil {
		return
	}
	module, ok := m.state.Modules[moduleName]
	if !ok {
		now := time.Now()
		module = &ModuleState{
			Name:      moduleName,
			Status:    StatusPending,
			Jobs:      make(map[string]*JobState),
			CreatedAt: now,
			UpdatedAt: now,
		}
		m.state.Modules[moduleName] = module
	}
	job.UpdatedAt = time.Now()
	module.Jobs[job.Name] = job
	module.UpdatedAt = time.Now()
	m.state.Global.LastUpdate = time.Now()
}
