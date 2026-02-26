// Package cmd provides command handlers for Morty CLI commands.
package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/logging"
)

// mockResetLogger is a mock implementation of logging.Logger for testing.
type mockResetLogger struct{}

func (m *mockResetLogger) WithContext(ctx context.Context) logging.Logger { return m }
func (m *mockResetLogger) Debug(msg string, attrs ...logging.Attr)        {}
func (m *mockResetLogger) Info(msg string, attrs ...logging.Attr)         {}
func (m *mockResetLogger) Warn(msg string, attrs ...logging.Attr)         {}
func (m *mockResetLogger) Error(msg string, attrs ...logging.Attr)        {}
func (m *mockResetLogger) Success(msg string, attrs ...logging.Attr)      {}
func (m *mockResetLogger) Loop(msg string, attrs ...logging.Attr)         {}
func (m *mockResetLogger) WithJob(module, job string) logging.Logger      { return m }
func (m *mockResetLogger) WithAttrs(attrs ...logging.Attr) logging.Logger { return m }
func (m *mockResetLogger) SetLevel(level logging.Level)                   {}
func (m *mockResetLogger) GetLevel() logging.Level                        { return logging.InfoLevel }
func (m *mockResetLogger) IsEnabled(level logging.Level) bool             { return true }

// mockGitChecker is a mock implementation of GitChecker for testing.
type mockGitChecker struct {
	isGitRepo bool
	repoRoot  string
	err       error
}

func (m *mockGitChecker) IsGitRepo(path string) bool {
	return m.isGitRepo
}

func (m *mockGitChecker) GetRepoRoot(path string) (string, error) {
	return m.repoRoot, m.err
}

// mockResetConfig is a mock implementation of config.Manager for testing.
type mockResetConfig struct {
	workDir string
}

func (m *mockResetConfig) Load(path string) error                            { return nil }
func (m *mockResetConfig) LoadWithMerge(userConfigPath string) error         { return nil }
func (m *mockResetConfig) Get(key string, defaultValue ...interface{}) (interface{}, error) {
	return nil, nil
}
func (m *mockResetConfig) GetString(key string, defaultValue ...string) string { return "" }
func (m *mockResetConfig) GetInt(key string, defaultValue ...int) int          { return 0 }
func (m *mockResetConfig) GetBool(key string, defaultValue ...bool) bool       { return false }
func (m *mockResetConfig) GetDuration(key string, defaultValue ...time.Duration) time.Duration {
	return 0
}
func (m *mockResetConfig) Set(key string, value interface{}) error             { return nil }
func (m *mockResetConfig) Save() error                                         { return nil }
func (m *mockResetConfig) SaveTo(path string) error                            { return nil }
func (m *mockResetConfig) GetWorkDir() string                                  { return m.workDir }
func (m *mockResetConfig) GetLogDir() string                                   { return "" }
func (m *mockResetConfig) GetResearchDir() string                              { return "" }
func (m *mockResetConfig) GetPlanDir() string                                  { return "" }
func (m *mockResetConfig) GetStatusFile() string                               { return "" }
func (m *mockResetConfig) GetConfigFile() string                               { return "" }

func TestResetHandler_parseOptions(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantLocal bool
		wantClean bool
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "no options",
			args:      []string{},
			wantLocal: false,
			wantClean: false,
			wantErr:   false,
		},
		{
			name:      "-l flag",
			args:      []string{"-l"},
			wantLocal: true,
			wantClean: false,
			wantErr:   false,
		},
		{
			name:      "-c flag",
			args:      []string{"-c"},
			wantLocal: false,
			wantClean: true,
			wantErr:   false,
		},
		{
			name:      "both -l and -c flags",
			args:      []string{"-l", "-c"},
			wantLocal: true,
			wantClean: true,
			wantErr:   true,
			errMsg:    "错误: 选项 -l 和 -c 不能同时使用",
		},
		{
			name:      "-l=true format",
			args:      []string{"-l=true"},
			wantLocal: true,
			wantClean: false,
			wantErr:   false,
		},
		{
			name:      "-c=1 format",
			args:      []string{"-c=1"},
			wantLocal: false,
			wantClean: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewResetHandler(nil, &mockResetLogger{})
			opts, err := handler.parseOptions(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseOptions() expected error but got none")
					return
				}
				if tt.errMsg != "" && !containsReset(err.Error(), tt.errMsg) {
					t.Errorf("parseOptions() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("parseOptions() unexpected error = %v", err)
				return
			}

			if opts.ResetLocal != tt.wantLocal {
				t.Errorf("parseOptions() ResetLocal = %v, want %v", opts.ResetLocal, tt.wantLocal)
			}
			if opts.ResetClean != tt.wantClean {
				t.Errorf("parseOptions() ResetClean = %v, want %v", opts.ResetClean, tt.wantClean)
			}
		})
	}
}

func TestResetHandler_Execute_NoOptions(t *testing.T) {
	// Create a temporary directory that simulates a git repo
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	cfg := &mockResetConfig{workDir: tmpDir}
	logger := &mockResetLogger{}
	handler := NewResetHandler(cfg, logger)

	// Use mock git checker that reports it's a git repo
	mockChecker := &mockGitChecker{isGitRepo: true, repoRoot: tmpDir}
	handler.SetGitChecker(mockChecker)

	// Execute with no options
	result, err := handler.Execute(context.Background(), []string{})

	if err == nil {
		t.Error("Execute() expected error when no options provided, got nil")
	}

	if result == nil {
		t.Fatal("Execute() returned nil result")
	}

	if result.ExitCode != 1 {
		t.Errorf("Execute() ExitCode = %d, want 1", result.ExitCode)
	}

	// Check that the error message suggests using -l or -c
	if result.Err != nil && !containsReset(result.Err.Error(), "-l") {
		t.Errorf("Execute() error message should mention -l option")
	}
}

func TestResetHandler_Execute_MutualExclusion(t *testing.T) {
	// Create a temporary directory that simulates a git repo
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	cfg := &mockResetConfig{workDir: tmpDir}
	logger := &mockResetLogger{}
	handler := NewResetHandler(cfg, logger)

	// Use mock git checker that reports it's a git repo
	mockChecker := &mockGitChecker{isGitRepo: true, repoRoot: tmpDir}
	handler.SetGitChecker(mockChecker)

	// Execute with both -l and -c options
	result, err := handler.Execute(context.Background(), []string{"-l", "-c"})

	if err == nil {
		t.Error("Execute() expected error when both -l and -c provided, got nil")
	}

	if result == nil {
		t.Fatal("Execute() returned nil result")
	}

	if result.ExitCode != 1 {
		t.Errorf("Execute() ExitCode = %d, want 1", result.ExitCode)
	}

	// Check that the error message mentions mutual exclusion
	if result.Err != nil && !containsReset(result.Err.Error(), "不能同时使用") {
		t.Errorf("Execute() error message should mention mutual exclusion")
	}
}

func TestResetHandler_Execute_NotGitRepo(t *testing.T) {
	// Create a temporary directory that is NOT a git repo
	tmpDir := t.TempDir()

	cfg := &mockResetConfig{workDir: tmpDir}
	logger := &mockResetLogger{}
	handler := NewResetHandler(cfg, logger)

	// Use mock git checker that reports it's NOT a git repo
	mockChecker := &mockGitChecker{isGitRepo: false}
	handler.SetGitChecker(mockChecker)

	// Execute with -l option
	result, err := handler.Execute(context.Background(), []string{"-l"})

	if err == nil {
		t.Error("Execute() expected error when not in git repo, got nil")
	}

	if result == nil {
		t.Fatal("Execute() returned nil result")
	}

	if result.ExitCode != 1 {
		t.Errorf("Execute() ExitCode = %d, want 1", result.ExitCode)
	}

	// Check that the error message mentions Git repository
	if result.Err != nil && !containsReset(result.Err.Error(), "Git 仓库") {
		t.Errorf("Execute() error message should mention Git repository")
	}
}

func TestResetHandler_Execute_LocalReset(t *testing.T) {
	// Create a temporary directory that simulates a git repo with morty state
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	// Create some morty state files
	mortyDir := filepath.Join(tmpDir, ".morty")
	statusFile := filepath.Join(mortyDir, "status.json")
	doingDir := filepath.Join(mortyDir, "doing", "logs")

	if err := os.MkdirAll(doingDir, 0755); err != nil {
		t.Fatalf("Failed to create doing directory: %v", err)
	}

	if err := os.WriteFile(statusFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create status file: %v", err)
	}

	cfg := &mockResetConfig{workDir: tmpDir}
	logger := &mockResetLogger{}
	handler := NewResetHandler(cfg, logger)

	// Use mock git checker that reports it's a git repo
	mockChecker := &mockGitChecker{isGitRepo: true, repoRoot: tmpDir}
	handler.SetGitChecker(mockChecker)

	// Execute with -l option
	result, err := handler.Execute(context.Background(), []string{"-l"})

	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Execute() returned nil result")
	}

	if result.ExitCode != 0 {
		t.Errorf("Execute() ExitCode = %d, want 0", result.ExitCode)
	}

	if result.ResetLevel != "local" {
		t.Errorf("Execute() ResetLevel = %s, want local", result.ResetLevel)
	}

	// Verify that status.json and doing directory were removed
	if _, err := os.Stat(statusFile); !os.IsNotExist(err) {
		t.Error("status.json should have been removed")
	}

	if _, err := os.Stat(doingDir); !os.IsNotExist(err) {
		t.Error("doing directory should have been removed")
	}
}

func TestResetHandler_Execute_CleanReset(t *testing.T) {
	// Create a temporary directory that simulates a git repo with morty state
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	// Create some morty state files
	mortyDir := filepath.Join(tmpDir, ".morty")
	statusFile := filepath.Join(mortyDir, "status.json")
	planDir := filepath.Join(mortyDir, "plan")

	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("Failed to create plan directory: %v", err)
	}

	if err := os.WriteFile(statusFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create status file: %v", err)
	}

	cfg := &mockResetConfig{workDir: tmpDir}
	logger := &mockResetLogger{}
	handler := NewResetHandler(cfg, logger)

	// Use mock git checker that reports it's a git repo
	mockChecker := &mockGitChecker{isGitRepo: true, repoRoot: tmpDir}
	handler.SetGitChecker(mockChecker)

	// Execute with -c option
	result, err := handler.Execute(context.Background(), []string{"-c"})

	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Execute() returned nil result")
	}

	if result.ExitCode != 0 {
		t.Errorf("Execute() ExitCode = %d, want 0", result.ExitCode)
	}

	if result.ResetLevel != "clean" {
		t.Errorf("Execute() ResetLevel = %s, want clean", result.ResetLevel)
	}

	// Verify that entire .morty directory was removed
	if _, err := os.Stat(mortyDir); !os.IsNotExist(err) {
		t.Error(".morty directory should have been removed")
	}
}

// Helper function to check if a string contains a substring
func containsReset(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(s[:len(substr)] == substr) ||
		(s[len(s)-len(substr):] == substr) ||
		findInStringReset(s, substr))
}

func findInStringReset(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure mock implementations satisfy interfaces
var _ logging.Logger = (*mockResetLogger)(nil)
var _ config.Manager = (*mockResetConfig)(nil)
var _ GitChecker = (*mockGitChecker)(nil)
