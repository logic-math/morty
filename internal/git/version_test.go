// Package git provides Git repository operations for Morty.
package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestVersionControllerInterface verifies that Manager implements VersionController interface.
func TestVersionControllerInterface(t *testing.T) {
	var _ VersionController = (*Manager)(nil)
	mgr := NewManager()
	var vc VersionController = mgr
	if vc == nil {
		t.Fatal("Manager does not implement VersionController interface")
	}
}

// TestResetToCommitSuccess tests successful reset to a commit.
func TestResetToCommitSuccess(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create file1 and commit
	file1 := filepath.Join(tempDir, "file1.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	mgr.run(tempDir, "add", "-A")
	mgr.run(tempDir, "commit", "-m", "morty: loop 1 - COMPLETED")

	// Get commit hash of first morty commit
	commitHash, _ := mgr.run(tempDir, "rev-parse", "HEAD~1")
	if commitHash == "" {
		t.Fatal("Failed to get commit hash")
	}

	// Create file2 and commit
	file2 := filepath.Join(tempDir, "file2.txt")
	os.WriteFile(file2, []byte("content2"), 0644)
	mgr.run(tempDir, "add", "-A")
	mgr.run(tempDir, "commit", "-m", "morty: loop 2 - COMPLETED")

	// Verify file2 exists
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Fatal("file2 should exist before reset")
	}

	// Reset to first morty commit with hard reset
	err = mgr.ResetToCommit(commitHash, tempDir, HardReset)
	if err != nil {
		t.Fatalf("ResetToCommit failed: %v", err)
	}

	// Verify file2 no longer exists (hard reset removed it)
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Error("file2 should not exist after hard reset")
	}

	// Verify backup branch was created
	branches, _ := mgr.run(tempDir, "branch", "--list", "morty/backup-*")
	if !strings.Contains(branches, "morty/backup-") {
		t.Error("Backup branch should have been created")
	}
}

// TestResetToCommitSoftReset tests soft reset functionality.
func TestResetToCommitSoftReset(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial content"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create file1 and commit
	file1 := filepath.Join(tempDir, "file1.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	mgr.run(tempDir, "add", "-A")
	mgr.run(tempDir, "commit", "-m", "morty: loop 1 - COMPLETED")

	// Get commit hash of first morty commit
	commitHash, _ := mgr.run(tempDir, "rev-parse", "HEAD")

	// Create file2 and commit
	file2 := filepath.Join(tempDir, "file2.txt")
	os.WriteFile(file2, []byte("content2"), 0644)
	mgr.run(tempDir, "add", "-A")
	mgr.run(tempDir, "commit", "-m", "morty: loop 2 - COMPLETED")

	// Soft reset to first morty commit
	err = mgr.ResetToCommit(commitHash, tempDir, SoftReset)
	if err != nil {
		t.Fatalf("ResetToCommit with soft reset failed: %v", err)
	}

	// Verify file2 still exists (soft reset keeps changes)
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Error("file2 should still exist after soft reset")
	}

	// Verify changes are staged
	status, _ := mgr.run(tempDir, "status", "--porcelain")
	if !strings.Contains(status, "file2.txt") {
		t.Error("file2 should be staged after soft reset")
	}
}

// TestShowLoopHistorySuccess tests successful history retrieval.
func TestShowLoopHistorySuccess(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create 5 loop commits
	for i := 1; i <= 5; i++ {
		file := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(file, []byte(fmt.Sprintf("content%d", i)), 0644)
		mgr.run(tempDir, "add", "-A")
		status := "COMPLETED"
		if i == 5 {
			status = "RUNNING"
		}
		mgr.run(tempDir, "commit", "-m", fmt.Sprintf("morty: loop %d - %s", i, status))
	}

	// Get loop history (last 3)
	history, err := mgr.ShowLoopHistory(3, tempDir)
	if err != nil {
		t.Fatalf("ShowLoopHistory failed: %v", err)
	}

	// Verify we got 3 commits
	if len(history) != 3 {
		t.Errorf("Expected 3 history entries, got %d", len(history))
	}

	// Verify order (newest first) and content
	if len(history) >= 3 {
		// First entry should be loop 5 (newest)
		if history[0].LoopNumber != 5 {
			t.Errorf("Expected first entry to be loop 5, got %d", history[0].LoopNumber)
		}
		if history[0].Status != "RUNNING" {
			t.Errorf("Expected first entry status to be RUNNING, got %s", history[0].Status)
		}
	}
}

// TestParseCommitMessageSuccess tests successful message parsing.
func TestParseCommitMessageSuccess(t *testing.T) {
	mgr := NewManager()

	tests := []struct {
		message        string
		expectedLoop   int
		expectedStatus string
		expectError    bool
	}{
		{"morty: loop 1 - COMPLETED", 1, "COMPLETED", false},
		{"morty: loop 5 - RUNNING", 5, "RUNNING", false},
		{"MORTY: LOOP 3 - COMPLETED", 3, "COMPLETED", false},
		{"regular commit", 0, "", true},
	}

	for _, tc := range tests {
		commit, err := mgr.ParseCommitMessage(tc.message)

		if tc.expectError {
			if err == nil {
				t.Errorf("Expected error for message '%s', got nil", tc.message)
			}
			continue
		}

		if err != nil {
			t.Errorf("Unexpected error for message '%s': %v", tc.message, err)
			continue
		}

		if commit.LoopNumber != tc.expectedLoop {
			t.Errorf("Expected loop %d, got %d for message '%s'", tc.expectedLoop, commit.LoopNumber, tc.message)
		}

		if commit.Status != tc.expectedStatus {
			t.Errorf("Expected status '%s', got '%s' for message '%s'", tc.expectedStatus, commit.Status, tc.message)
		}
	}
}

// TestFormatLoopHistorySuccess tests history formatting.
func TestFormatLoopHistorySuccess(t *testing.T) {
	mgr := NewManager()

	history := []LoopCommit{
		{
			LoopNumber: 5,
			Status:     "COMPLETED",
			ShortHash:  "abc1234",
			Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Author:     "Test User",
		},
		{
			LoopNumber: 4,
			Status:     "RUNNING",
			ShortHash:  "def5678",
			Timestamp:  time.Date(2024, 1, 15, 10, 25, 0, 0, time.UTC),
			Author:     "Test User",
		},
	}

	formatted := mgr.FormatLoopHistory(history)

	// Verify header is present
	if !strings.Contains(formatted, "Loop History:") {
		t.Error("Expected 'Loop History:' header")
	}

	// Verify entries are present
	if !strings.Contains(formatted, "5") {
		t.Error("Expected loop 5 in output")
	}
	if !strings.Contains(formatted, "COMPLETED") {
		t.Error("Expected COMPLETED status in output")
	}
}

// TestShowLoopHistoryCorrectOrder verifies reverse chronological order.
func TestShowLoopHistoryCorrectOrder(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create 5 loop commits
	for i := 1; i <= 5; i++ {
		file := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(file, []byte("content"), 0644)
		mgr.run(tempDir, "add", "-A")
		mgr.run(tempDir, "commit", "-m", fmt.Sprintf("morty: loop %d - COMPLETED", i))
	}

	// Get all history
	history, err := mgr.ShowLoopHistory(10, tempDir)
	if err != nil {
		t.Fatalf("ShowLoopHistory failed: %v", err)
	}

	if len(history) != 5 {
		t.Fatalf("Expected 5 entries, got %d", len(history))
	}

	// Verify reverse chronological order (newest first)
	for i := 0; i < len(history)-1; i++ {
		if history[i].LoopNumber <= history[i+1].LoopNumber {
			t.Errorf("History not in reverse chronological order at index %d: %d vs %d",
				i, history[i].LoopNumber, history[i+1].LoopNumber)
		}
	}
}

// TestShowLoopHistoryEmptyRepo tests empty repo returns empty history.
func TestShowLoopHistoryEmptyRepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Get loop history from empty repo
	history, err := mgr.ShowLoopHistory(10, tempDir)
	if err != nil {
		t.Fatalf("ShowLoopHistory failed: %v", err)
	}

	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d entries", len(history))
	}
}

// TestShowLoopHistoryNotARepo tests error for non-git directory.
func TestShowLoopHistoryNotARepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	_, err := mgr.ShowLoopHistory(10, tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory")
	}
}

// TestResetToCommitWithBackup tests custom backup branch creation.
func TestResetToCommitWithBackup(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create first loop commit
	file1 := filepath.Join(tempDir, "file1.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	mgr.run(tempDir, "add", "-A")
	mgr.run(tempDir, "commit", "-m", "morty: loop 1 - COMPLETED")

	commitHash, _ := mgr.run(tempDir, "rev-parse", "HEAD")

	// Create second loop commit
	file2 := filepath.Join(tempDir, "file2.txt")
	os.WriteFile(file2, []byte("content2"), 0644)
	mgr.run(tempDir, "add", "-A")
	mgr.run(tempDir, "commit", "-m", "morty: loop 2 - COMPLETED")

	// Reset with custom backup name
	customBackup := "morty/custom-backup-test"
	err = mgr.ResetToCommitWithBackup(commitHash, tempDir, HardReset, customBackup)
	if err != nil {
		t.Fatalf("ResetToCommitWithBackup failed: %v", err)
	}

	// Verify custom backup branch exists
	branches, _ := mgr.run(tempDir, "branch", "--list", customBackup)
	if !strings.Contains(branches, customBackup) {
		t.Errorf("Custom backup branch '%s' should exist", customBackup)
	}
}

// TestGetCommitAtLoop tests finding commit by loop number.
func TestGetCommitAtLoop(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create loop commits
	var commitHashes []string
	for i := 1; i <= 3; i++ {
		file := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(file, []byte("content"), 0644)
		mgr.run(tempDir, "add", "-A")
		mgr.run(tempDir, "commit", "-m", fmt.Sprintf("morty: loop %d - COMPLETED", i))
		hash, _ := mgr.run(tempDir, "rev-parse", "HEAD")
		commitHashes = append(commitHashes, hash)
	}

	// Test finding each loop
	for i := 1; i <= 3; i++ {
		hash, err := mgr.GetCommitAtLoop(i, tempDir)
		if err != nil {
			t.Errorf("GetCommitAtLoop(%d) failed: %v", i, err)
			continue
		}
		// The commits are in reverse order in history (newest first)
		// commitHashes[0] is loop 1, commitHashes[1] is loop 2, etc.
		expectedHash := commitHashes[i-1]
		if hash != expectedHash {
			t.Errorf("GetCommitAtLoop(%d): expected %s, got %s", i, expectedHash, hash)
		}
	}
}

// TestGetCommitAtLoopNotFound tests finding non-existent loop.
func TestGetCommitAtLoopNotFound(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create only loop 1 and 2
	for i := 1; i <= 2; i++ {
		file := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(file, []byte("content"), 0644)
		mgr.run(tempDir, "add", "-A")
		mgr.run(tempDir, "commit", "-m", fmt.Sprintf("morty: loop %d - COMPLETED", i))
	}

	// Try to find loop 99
	_, err = mgr.GetCommitAtLoop(99, tempDir)
	if err == nil {
		t.Error("Expected error for non-existent loop")
	}
}

// TestRollbackToLoop tests rollback to a specific loop number.
func TestRollbackToLoop(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create loop commits with files
	for i := 1; i <= 3; i++ {
		file := filepath.Join(tempDir, fmt.Sprintf("loop%d.txt", i))
		os.WriteFile(file, []byte(fmt.Sprintf("content version %d", i)), 0644)
		mgr.run(tempDir, "add", "-A")
		mgr.run(tempDir, "commit", "-m", fmt.Sprintf("morty: loop %d - COMPLETED", i))
	}

	// Verify file3 exists
	file3 := filepath.Join(tempDir, "loop3.txt")
	if _, err := os.Stat(file3); os.IsNotExist(err) {
		t.Fatal("loop3.txt should exist before rollback")
	}

	// Rollback to loop 1
	err = mgr.RollbackToLoop(1, tempDir, HardReset)
	if err != nil {
		t.Fatalf("RollbackToLoop failed: %v", err)
	}

	// Verify file3 and file2 no longer exist, but file1 still does
	if _, err := os.Stat(file3); !os.IsNotExist(err) {
		t.Error("loop3.txt should not exist after rollback to loop 1")
	}
	file2 := filepath.Join(tempDir, "loop2.txt")
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Error("loop2.txt should not exist after rollback to loop 1")
	}
	file1 := filepath.Join(tempDir, "loop1.txt")
	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Error("loop1.txt should still exist after rollback to loop 1")
	}
}

// TestFormatLoopHistoryDetailed tests detailed formatting.
func TestFormatLoopHistoryDetailed(t *testing.T) {
	mgr := NewManager()

	history := []LoopCommit{
		{
			CommitHash: "abc123def456789",
			LoopNumber: 1,
			Status:     "COMPLETED",
			Author:     "Test User",
			Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Stats: &ChangeStats{
				FilesAdded:    2,
				FilesModified: 1,
				FilesDeleted:  0,
				LinesAdded:    50,
				LinesDeleted:  10,
			},
		},
	}

	formatted := mgr.FormatLoopHistoryDetailed(history)

	// Verify detailed header
	if !strings.Contains(formatted, "Detailed Loop History:") {
		t.Error("Expected 'Detailed Loop History:' header")
	}

	// Verify commit hash
	if !strings.Contains(formatted, "abc123def456789") {
		t.Error("Expected full commit hash")
	}

	// Verify stats
	if !strings.Contains(formatted, "Changes:") {
		t.Error("Expected 'Changes:' with stats")
	}
}

// TestFormatLoopHistoryEmpty tests formatting empty history.
func TestFormatLoopHistoryEmpty(t *testing.T) {
	mgr := NewManager()

	formatted := mgr.FormatLoopHistory([]LoopCommit{})
	if !strings.Contains(formatted, "No loop history found") {
		t.Error("Expected 'No loop history found' message")
	}

	formattedDetailed := mgr.FormatLoopHistoryDetailed([]LoopCommit{})
	if !strings.Contains(formattedDetailed, "No loop history found") {
		t.Error("Expected 'No loop history found' message in detailed format")
	}
}

// TestResetToCommitInvalidHash tests error for invalid commit hash.
func TestResetToCommitInvalidHash(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Try to reset to invalid hash
	err = mgr.ResetToCommit("invalidhash123", tempDir, HardReset)
	if err == nil {
		t.Error("Expected error for invalid commit hash")
	}
}

// TestResetToCommitNotARepo tests error for non-git directory.
func TestResetToCommitNotARepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	err := mgr.ResetToCommit("abc123", tempDir, HardReset)
	if err == nil {
		t.Error("Expected error for non-git directory")
	}
}

// TestShowLoopHistoryZeroCount tests with n=0.
func TestShowLoopHistoryZeroCount(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create a loop commit
	file1 := filepath.Join(tempDir, "file1.txt")
	os.WriteFile(file1, []byte("content"), 0644)
	mgr.run(tempDir, "add", "-A")
	mgr.run(tempDir, "commit", "-m", "morty: loop 1 - COMPLETED")

	// Get loop history with n=0
	history, err := mgr.ShowLoopHistory(0, tempDir)
	if err != nil {
		t.Fatalf("ShowLoopHistory failed: %v", err)
	}

	if len(history) != 0 {
		t.Errorf("Expected empty history for n=0, got %d entries", len(history))
	}
}

// TestParseCommitMessageVariousFormats tests various message formats.
func TestParseCommitMessageVariousFormats(t *testing.T) {
	mgr := NewManager()

	tests := []struct {
		message        string
		expectedLoop   int
		expectedStatus string
	}{
		{"morty: loop 1 - COMPLETED", 1, "COMPLETED"},
		{"morty: loop 42 - RUNNING", 42, "RUNNING"},
		{"morty: loop 999 - FAILED", 999, "FAILED"},
		{"MORTY: LOOP 1 - COMPLETED", 1, "COMPLETED"},
		{"Morty: Loop 5 - Running", 5, "RUNNING"},
		{"morty:loop 3 - STATUS", 3, "STATUS"},
		{"morty: loop 2", 2, "UNKNOWN"},
	}

	for _, tc := range tests {
		commit, err := mgr.ParseCommitMessage(tc.message)
		if err != nil {
			t.Errorf("Failed to parse '%s': %v", tc.message, err)
			continue
		}

		if commit.LoopNumber != tc.expectedLoop {
			t.Errorf("For '%s': expected loop %d, got %d",
				tc.message, tc.expectedLoop, commit.LoopNumber)
		}

		if commit.Status != tc.expectedStatus {
			t.Errorf("For '%s': expected status '%s', got '%s'",
				tc.message, tc.expectedStatus, commit.Status)
		}
	}
}

// TestShowLoopHistoryMixedCommits tests history with mixed commit types.
func TestShowLoopHistoryMixedCommits(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create mix of regular and loop commits
	commits := []struct {
		filename string
		message  string
		isLoop   bool
	}{
		{"file1.txt", "regular commit 1", false},
		{"file2.txt", "morty: loop 1 - COMPLETED", true},
		{"file3.txt", "regular commit 2", false},
		{"file4.txt", "morty: loop 2 - RUNNING", true},
		{"file5.txt", "regular commit 3", false},
		{"file6.txt", "morty: loop 3 - PENDING", true},
	}

	for _, tc := range commits {
		file := filepath.Join(tempDir, tc.filename)
		os.WriteFile(file, []byte("content"), 0644)
		mgr.run(tempDir, "add", "-A")
		mgr.run(tempDir, "commit", "-m", tc.message)
	}

	// Get loop history
	history, err := mgr.ShowLoopHistory(10, tempDir)
	if err != nil {
		t.Fatalf("ShowLoopHistory failed: %v", err)
	}

	// Should only get loop commits (3), not regular ones
	if len(history) != 3 {
		t.Errorf("Expected 3 loop commits, got %d", len(history))
	}
}
