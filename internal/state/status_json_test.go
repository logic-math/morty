// Package state provides state management for Morty.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/morty/morty/pkg/errors"
)

// TestStatusConstants verifies that Status constants are defined correctly.
func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{"PENDING", StatusPending, "PENDING"},
		{"RUNNING", StatusRunning, "RUNNING"},
		{"COMPLETED", StatusCompleted, "COMPLETED"},
		{"FAILED", StatusFailed, "FAILED"},
		{"BLOCKED", StatusBlocked, "BLOCKED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Status %s = %s, expected %s", tt.name, tt.status, tt.expected)
			}
		})
	}
}

// TestStatusIsValid verifies the IsValid method.
func TestStatusIsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"PENDING", StatusPending, true},
		{"RUNNING", StatusRunning, true},
		{"COMPLETED", StatusCompleted, true},
		{"FAILED", StatusFailed, true},
		{"BLOCKED", StatusBlocked, true},
		{"INVALID", Status("INVALID"), false},
		{"empty", Status(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

// TestNewManager verifies manager creation.
func TestNewManager(t *testing.T) {
	path := "/tmp/test_state.json"
	m := NewManager(path)

	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.filePath != path {
		t.Errorf("filePath = %s, expected %s", m.filePath, path)
	}
	if m.state != nil {
		t.Error("initial state should be nil")
	}
}

// TestLoadFileNotExist verifies loading when file doesn't exist creates default state.
func TestLoadFileNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "nonexistent", "status.json")

	m := NewManager(statePath)
	err := m.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	state := m.GetState()
	if state == nil {
		t.Fatal("GetState() returned nil after Load()")
	}

	// Verify default state
	if state.Version != DefaultStateVersion {
		t.Errorf("Version = %s, expected %s", state.Version, DefaultStateVersion)
	}
	if state.Global.Status != StatusPending {
		t.Errorf("Global.Status = %s, expected %s", state.Global.Status, StatusPending)
	}
	if len(state.Modules) != 0 {
		t.Errorf("len(Modules) = %d, expected 0", len(state.Modules))
	}
}

// TestLoadExistingFile verifies loading an existing state file.
func TestLoadExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	// Create a test state file
	testState := &StatusJSON{
		Version: "1.0",
		Global: GlobalState{
			Status:     StatusRunning,
			StartTime:  time.Now(),
			LastUpdate: time.Now(),
			TotalLoops: 5,
		},
		Modules: map[string]*ModuleState{
			"test_module": {
				Name:      "test_module",
				Status:    StatusRunning,
				Jobs:      make(map[string]*JobState),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	data, _ := json.MarshalIndent(testState, "", "  ")
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load the state
	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	state := m.GetState()
	if state == nil {
		t.Fatal("GetState() returned nil")
	}

	if state.Global.Status != StatusRunning {
		t.Errorf("Global.Status = %s, expected %s", state.Global.Status, StatusRunning)
	}
	if state.Global.TotalLoops != 5 {
		t.Errorf("Global.TotalLoops = %d, expected 5", state.Global.TotalLoops)
	}
	if _, ok := state.Modules["test_module"]; !ok {
		t.Error("test_module not found in modules")
	}
}

// TestLoadCorruptedFile verifies loading a corrupted state file returns error.
func TestLoadCorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	// Create a corrupted file
	if err := os.WriteFile(statePath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	m := NewManager(statePath)
	err := m.Load()
	if err == nil {
		t.Fatal("Load() expected error for corrupted file")
	}

	// Check that it's a MortyError with correct code
	if me, ok := errors.AsMortyError(err); ok {
		if me.Code != "M2002" {
			t.Errorf("Error code = %s, expected M2002", me.Code)
		}
	} else {
		t.Error("Expected MortyError for corrupted file")
	}
}

// TestLoadInvalidState verifies loading a state with invalid status returns error.
func TestLoadInvalidState(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	// Create a state with invalid status
	testState := &StatusJSON{
		Version: "1.0",
		Global: GlobalState{
			Status:     Status("INVALID_STATUS"),
			StartTime:  time.Now(),
			LastUpdate: time.Now(),
		},
		Modules: make(map[string]*ModuleState),
	}

	data, _ := json.MarshalIndent(testState, "", "  ")
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	m := NewManager(statePath)
	err := m.Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid status")
	}

	if me, ok := errors.AsMortyError(err); ok {
		if me.Code != "M2003" {
			t.Errorf("Error code = %s, expected M2003", me.Code)
		}
	} else {
		t.Error("Expected MortyError for invalid state")
	}
}

// TestSave verifies saving state to file.
func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)

	// Create and set a state
	state := &StatusJSON{
		Version: "1.0",
		Global: GlobalState{
			Status:     StatusRunning,
			StartTime:  time.Now(),
			LastUpdate: time.Now(),
			TotalLoops: 3,
		},
		Modules: make(map[string]*ModuleState),
	}
	m.SetState(state)

	// Save
	if err := m.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists and has correct content
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	var savedState StatusJSON
	if err := json.Unmarshal(data, &savedState); err != nil {
		t.Fatalf("Failed to parse saved file: %v", err)
	}

	if savedState.Global.Status != StatusRunning {
		t.Errorf("Saved Global.Status = %s, expected %s", savedState.Global.Status, StatusRunning)
	}
	if savedState.Global.TotalLoops != 3 {
		t.Errorf("Saved Global.TotalLoops = %d, expected 3", savedState.Global.TotalLoops)
	}

	// Verify JSON is indented
	content := string(data)
	if content[0] != '{' || content[len(content)-1] != '}' {
		t.Error("Saved JSON should be an object")
	}
}

// TestSaveCreatesDirectory verifies that Save creates parent directories.
func TestSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "nested", "deep", "status.json")

	m := NewManager(statePath)

	// Load to initialize default state
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Save should create directories
	if err := m.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(statePath); err != nil {
		t.Errorf("State file does not exist after Save: %v", err)
	}
}

// TestBackup verifies creating a backup of the state file.
func TestBackup(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	// Create initial state file
	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := m.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Create backup
	backupPath, err := m.Backup()
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("Backup file does not exist: %v", err)
	}

	// Verify backup contains valid JSON
	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}

	var backupState StatusJSON
	if err := json.Unmarshal(data, &backupState); err != nil {
		t.Errorf("Backup file contains invalid JSON: %v", err)
	}
}

// TestBackupNoFile verifies backup returns error when state file doesn't exist.
func TestBackupNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	_, err := m.Backup()
	if err == nil {
		t.Fatal("Backup() expected error when file doesn't exist")
	}
}

// TestListBackups verifies listing backup files.
func TestListBackups(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	// Create initial state and save
	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := m.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Create multiple backups
	if _, err := m.Backup(); err != nil {
		t.Fatalf("Backup() error = %v", err)
	}
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	if _, err := m.Backup(); err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	// List backups
	backups, err := m.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("len(backups) = %d, expected 2", len(backups))
	}
}

// TestRestoreFromBackup verifies restoring state from a backup.
func TestRestoreFromBackup(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	// Create and save initial state
	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Set specific state
	state := m.GetState()
	state.Global.TotalLoops = 42
	state.Global.Status = StatusCompleted
	if err := m.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Create backup
	backupPath, err := m.Backup()
	if err != nil {
		t.Fatalf("Backup() error = %v", err)
	}

	// Modify state
	state.Global.TotalLoops = 100
	state.Global.Status = StatusFailed
	if err := m.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Create new manager and restore from backup
	m2 := NewManager(statePath)
	if err := m2.RestoreFromBackup(backupPath); err != nil {
		t.Fatalf("RestoreFromBackup() error = %v", err)
	}

	// Verify restored state
	restoredState := m2.GetState()
	if restoredState.Global.TotalLoops != 42 {
		t.Errorf("TotalLoops = %d, expected 42", restoredState.Global.TotalLoops)
	}
	if restoredState.Global.Status != StatusCompleted {
		t.Errorf("Status = %s, expected %s", restoredState.Global.Status, StatusCompleted)
	}
}

// TestGetModuleAndJob verifies getting module and job states.
func TestGetModuleAndJob(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Initially should return nil
	if m.GetModule("test") != nil {
		t.Error("GetModule should return nil for non-existent module")
	}
	if m.GetJob("test", "job") != nil {
		t.Error("GetJob should return nil for non-existent job")
	}

	// Add module and job
	module := &ModuleState{
		Name:      "test_module",
		Status:    StatusRunning,
		Jobs:      make(map[string]*JobState),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.SetModule(module)

	job := &JobState{
		Name:           "test_job",
		Status:         StatusCompleted,
		TasksTotal:     5,
		TasksCompleted: 5,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	m.SetJob("test_module", job)

	// Now should be able to retrieve
	gotModule := m.GetModule("test_module")
	if gotModule == nil {
		t.Fatal("GetModule returned nil for existing module")
	}
	if gotModule.Status != StatusRunning {
		t.Errorf("Module.Status = %s, expected %s", gotModule.Status, StatusRunning)
	}

	gotJob := m.GetJob("test_module", "test_job")
	if gotJob == nil {
		t.Fatal("GetJob returned nil for existing job")
	}
	if gotJob.Status != StatusCompleted {
		t.Errorf("Job.Status = %s, expected %s", gotJob.Status, StatusCompleted)
	}
}

// TestSetJobCreatesModule verifies SetJob creates module if it doesn't exist.
func TestSetJobCreatesModule(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	job := &JobState{
		Name:   "test_job",
		Status: StatusPending,
	}
	m.SetJob("new_module", job)

	// Module should be created automatically
	if m.GetModule("new_module") == nil {
		t.Error("SetJob should create module if it doesn't exist")
	}
}

// TestDebugLogEntry verifies debug log entry structure.
func TestDebugLogEntry(t *testing.T) {
	entry := DebugLogEntry{
		ID:           "debug1",
		Timestamp:    time.Now(),
		Phenomenon:   "Test issue",
		Reproduction: "Run test",
		Hypothesis:   "Guess 1) Guess 2",
		Verification: "Check X",
		Fix:          "Fix Y",
		Progress:     "fixed",
	}

	// Verify JSON marshaling
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal DebugLogEntry: %v", err)
	}

	var parsed DebugLogEntry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal DebugLogEntry: %v", err)
	}

	if parsed.ID != entry.ID {
		t.Errorf("ID = %s, expected %s", parsed.ID, entry.ID)
	}
	if parsed.Phenomenon != entry.Phenomenon {
		t.Errorf("Phenomenon = %s, expected %s", parsed.Phenomenon, entry.Phenomenon)
	}
}

// TestTaskState verifies task state structure.
func TestTaskState(t *testing.T) {
	task := TaskState{
		Index:       1,
		Status:      StatusCompleted,
		Description: "Test task",
		UpdatedAt:   time.Now(),
	}

	if task.Index != 1 {
		t.Errorf("Index = %d, expected 1", task.Index)
	}
	if task.Status != StatusCompleted {
		t.Errorf("Status = %s, expected %s", task.Status, StatusCompleted)
	}
}

// TestSaveNilState verifies saving when state is nil initializes default state.
func TestSaveNilState(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	m := NewManager(statePath)
	// Don't call Load, so state is nil

	// Save should initialize default state
	if err := m.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(statePath); err != nil {
		t.Errorf("State file was not created: %v", err)
	}

	// Verify state was initialized
	state := m.GetState()
	if state == nil {
		t.Fatal("State should be initialized after Save")
	}
	if state.Version != DefaultStateVersion {
		t.Errorf("Version = %s, expected %s", state.Version, DefaultStateVersion)
	}
}

// TestBackupWithCustomPath verifies creating a backup at a custom path.
func TestBackupWithCustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")
	backupPath := filepath.Join(tmpDir, "backups", "custom_backup.json")

	m := NewManager(statePath)
	if err := m.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Set some state
	state := m.GetState()
	state.Global.TotalLoops = 99
	state.Global.Status = StatusRunning

	// Backup to custom path
	if err := m.BackupWithCustomPath(backupPath); err != nil {
		t.Fatalf("BackupWithCustomPath() error = %v", err)
	}

	// Verify backup file exists
	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	var backupState StatusJSON
	if err := json.Unmarshal(data, &backupState); err != nil {
		t.Fatalf("Failed to parse backup: %v", err)
	}

	if backupState.Global.TotalLoops != 99 {
		t.Errorf("Backup TotalLoops = %d, expected 99", backupState.Global.TotalLoops)
	}
}

// TestBackupWithCustomPathNilState verifies backup with nil state returns error.
func TestBackupWithCustomPathNilState(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")
	backupPath := filepath.Join(tmpDir, "backup.json")

	m := NewManager(statePath)
	// Don't load state, so it's nil

	err := m.BackupWithCustomPath(backupPath)
	if err == nil {
		t.Fatal("BackupWithCustomPath() expected error when state is nil")
	}
}

// TestRestoreFromBackupErrors verifies error cases for RestoreFromBackup.
func TestRestoreFromBackupErrors(t *testing.T) {
	t.Run("NonExistentFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, "status.json")
		backupPath := filepath.Join(tmpDir, "nonexistent.json")

		m := NewManager(statePath)
		err := m.RestoreFromBackup(backupPath)
		if err == nil {
			t.Fatal("RestoreFromBackup() expected error for non-existent file")
		}
	})

	t.Run("CorruptedBackup", func(t *testing.T) {
		tmpDir := t.TempDir()
		statePath := filepath.Join(tmpDir, "status.json")
		backupPath := filepath.Join(tmpDir, "corrupted.json")

		// Create corrupted backup
		if err := os.WriteFile(backupPath, []byte("not json"), 0644); err != nil {
			t.Fatalf("Failed to create corrupted file: %v", err)
		}

		m := NewManager(statePath)
		err := m.RestoreFromBackup(backupPath)
		if err == nil {
			t.Fatal("RestoreFromBackup() expected error for corrupted file")
		}

		if me, ok := errors.AsMortyError(err); ok {
			if me.Code != "M2002" {
				t.Errorf("Error code = %s, expected M2002", me.Code)
			}
		}
	})
}

// TestListBackupsErrors verifies ListBackups error handling.
func TestListBackupsErrors(t *testing.T) {
	// Test with non-existent directory - should return empty list
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "nonexistent_dir", "status.json")

	m := NewManager(statePath)
	backups, err := m.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("len(backups) = %d, expected 0 for non-existent dir", len(backups))
	}
}

// TestValidateStateMissingVersion verifies validation of state with missing version.
func TestValidateStateMissingVersion(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	// Create a state without version
	testState := &StatusJSON{
		Version: "",
		Global: GlobalState{
			Status:     StatusRunning,
			StartTime:  time.Now(),
			LastUpdate: time.Now(),
		},
		Modules: make(map[string]*ModuleState),
	}

	data, _ := json.MarshalIndent(testState, "", "  ")
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	m := NewManager(statePath)
	err := m.Load()
	if err == nil {
		t.Fatal("Load() expected error for missing version")
	}

	if me, ok := errors.AsMortyError(err); ok {
		if me.Code != "M2003" {
			t.Errorf("Error code = %s, expected M2003", me.Code)
		}
	}
}

// TestValidateStateNilModule verifies validation of state with nil module.
func TestValidateStateNilModule(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	// Create a state with nil module
	testState := &StatusJSON{
		Version: "1.0",
		Global: GlobalState{
			Status:     StatusRunning,
			StartTime:  time.Now(),
			LastUpdate: time.Now(),
		},
		Modules: map[string]*ModuleState{
			"test_module": nil,
		},
	}

	data, _ := json.MarshalIndent(testState, "", "  ")
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	m := NewManager(statePath)
	err := m.Load()
	if err == nil {
		t.Fatal("Load() expected error for nil module")
	}

	if me, ok := errors.AsMortyError(err); ok {
		if me.Code != "M2003" {
			t.Errorf("Error code = %s, expected M2003", me.Code)
		}
	}
}

// TestValidateStateNilJob verifies validation of state with nil job.
func TestValidateStateNilJob(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	// Create a state with nil job
	testState := &StatusJSON{
		Version: "1.0",
		Global: GlobalState{
			Status:     StatusRunning,
			StartTime:  time.Now(),
			LastUpdate: time.Now(),
		},
		Modules: map[string]*ModuleState{
			"test_module": {
				Name:      "test_module",
				Status:    StatusRunning,
				Jobs:      map[string]*JobState{
					"test_job": nil,
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	data, _ := json.MarshalIndent(testState, "", "  ")
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	m := NewManager(statePath)
	err := m.Load()
	if err == nil {
		t.Fatal("Load() expected error for nil job")
	}

	if me, ok := errors.AsMortyError(err); ok {
		if me.Code != "M2003" {
			t.Errorf("Error code = %s, expected M2003", me.Code)
		}
	}
}

// TestValidateStateInvalidJobStatus verifies validation of state with invalid job status.
func TestValidateStateInvalidJobStatus(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "status.json")

	// Create a state with invalid job status
	testState := &StatusJSON{
		Version: "1.0",
		Global: GlobalState{
			Status:     StatusRunning,
			StartTime:  time.Now(),
			LastUpdate: time.Now(),
		},
		Modules: map[string]*ModuleState{
			"test_module": {
				Name:      "test_module",
				Status:    StatusRunning,
				Jobs:      map[string]*JobState{
					"test_job": {
						Name:      "test_job",
						Status:    Status("INVALID"),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}

	data, _ := json.MarshalIndent(testState, "", "  ")
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	m := NewManager(statePath)
	err := m.Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid job status")
	}

	if me, ok := errors.AsMortyError(err); ok {
		if me.Code != "M2003" {
			t.Errorf("Error code = %s, expected M2003", me.Code)
		}
	}
}

// TestLoadStatError verifies Load handles stat errors other than NotExist.
func TestLoadStatError(t *testing.T) {
	// Create a file that cannot be read as a directory
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "not_a_dir")

	// Create a file at the path
	if err := os.WriteFile(statePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Now try to use a path inside that "file"
	invalidPath := filepath.Join(statePath, "status.json")

	m := NewManager(invalidPath)
	err := m.Load()
	// This should fail because statePath is a file, not a directory
	if err == nil {
		t.Fatal("Load() expected error for invalid path")
	}
}

// TestStatusString verifies the String method.
func TestStatusString(t *testing.T) {
	if StatusRunning.String() != "RUNNING" {
		t.Errorf("String() = %s, expected RUNNING", StatusRunning.String())
	}
}
