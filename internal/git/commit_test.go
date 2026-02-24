package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommitterInterface verifies that Manager implements Committer interface.
func TestCommitterInterface(t *testing.T) {
	var _ Committer = (*Manager)(nil)
	mgr := NewManager()
	var committer Committer = mgr
	if committer == nil {
		t.Fatal("Manager does not implement Committer interface")
	}
}

// TestBuildCommitMessage tests the commit message building function.
func TestBuildCommitMessage(t *testing.T) {
	stats := &ChangeStats{
		FilesAdded:    2,
		FilesModified: 3,
		FilesDeleted:  1,
		LinesAdded:    50,
		LinesDeleted:  20,
	}

	msg := buildCommitMessage(5, "COMPLETED", stats)

	// Check subject line
	if !strings.Contains(msg, "morty: loop 5 - COMPLETED") {
		t.Errorf("Commit message subject incorrect, got:\n%s", msg)
	}

	// Check stats in body
	if !strings.Contains(msg, "Files added: 2") {
		t.Error("Expected 'Files added: 2' in message")
	}
	if !strings.Contains(msg, "Files modified: 3") {
		t.Error("Expected 'Files modified: 3' in message")
	}
	if !strings.Contains(msg, "Files deleted: 1") {
		t.Error("Expected 'Files deleted: 1' in message")
	}
	if !strings.Contains(msg, "Lines added: 50") {
		t.Error("Expected 'Lines added: 50' in message")
	}
	if !strings.Contains(msg, "Lines deleted: 20") {
		t.Error("Expected 'Lines deleted: 20' in message")
	}
}

// TestCreateLoopCommitSuccess tests successful commit creation.
func TestCreateLoopCommitSuccess(t *testing.T) {
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

	// Create an initial commit (required for branch operations)
	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create a new file
	testFile := filepath.Join(tempDir, "test.go")
	err = os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create loop commit
	commitHash, err := mgr.CreateLoopCommit(1, "COMPLETED", tempDir)
	if err != nil {
		t.Fatalf("CreateLoopCommit failed: %v", err)
	}

	// Verify commit hash is not empty
	if commitHash == "" {
		t.Error("Expected non-empty commit hash")
	}

	// Verify commit was created with correct message
	logOutput, err := mgr.run(tempDir, "log", "-1", "--pretty=format:%s")
	if err != nil {
		t.Fatalf("Failed to get log: %v", err)
	}

	expectedSubject := "morty: loop 1 - COMPLETED"
	if logOutput != expectedSubject {
		t.Errorf("Expected subject '%s', got '%s'", expectedSubject, logOutput)
	}

	// Verify commit body contains stats
	bodyOutput, err := mgr.run(tempDir, "log", "-1", "--pretty=format:%b")
	if err != nil {
		t.Fatalf("Failed to get commit body: %v", err)
	}

	if !strings.Contains(bodyOutput, "Change Statistics:") {
		t.Error("Expected 'Change Statistics:' in commit body")
	}
}

// TestCreateLoopCommitNoChanges tests commit creation with no changes.
func TestCreateLoopCommitNoChanges(t *testing.T) {
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

	// Create an initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Try to create loop commit with no changes
	_, err = mgr.CreateLoopCommit(1, "COMPLETED", tempDir)
	if err == nil {
		t.Error("Expected error when no changes to commit, got nil")
	}

	if !strings.Contains(err.Error(), "no changes to commit") {
		t.Errorf("Expected 'no changes to commit' error, got: %v", err)
	}
}

// TestCreateLoopCommitNotARepo tests commit creation in non-git directory.
func TestCreateLoopCommitNotARepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	_, err := mgr.CreateLoopCommit(1, "COMPLETED", tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}

	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("Expected 'not a git repository' error, got: %v", err)
	}
}

// TestCreateLoopCommitStagesAllChanges tests that all changes are staged.
func TestCreateLoopCommitStagesAllChanges(t *testing.T) {
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

	// Create an initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create multiple files without staging
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	// Create loop commit (should auto-stage)
	_, err = mgr.CreateLoopCommit(2, "RUNNING", tempDir)
	if err != nil {
		t.Fatalf("CreateLoopCommit failed: %v", err)
	}

	// Verify both files were committed
	status, err := mgr.run(tempDir, "status", "--porcelain")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if strings.TrimSpace(status) != "" {
		t.Errorf("Expected all files to be committed, but status shows: %s", status)
	}
}

// TestGetCurrentLoopNumberEmptyRepo tests getting loop number in empty repo.
func TestGetCurrentLoopNumberEmptyRepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Get loop number in empty repo (no commits yet)
	loopNum, err := mgr.GetCurrentLoopNumber(tempDir)
	if err != nil {
		t.Fatalf("GetCurrentLoopNumber failed: %v", err)
	}

	if loopNum != 1 {
		t.Errorf("Expected loop number 1 for empty repo, got %d", loopNum)
	}
}

// TestGetCurrentLoopNumberWithCommits tests getting loop number with existing commits.
func TestGetCurrentLoopNumberWithCommits(t *testing.T) {
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

	// Create an initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create some morty loop commits
	for i := 1; i <= 3; i++ {
		file := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(file, []byte("content"), 0644)
		mgr.run(tempDir, "add", "-A")
		mgr.run(tempDir, "commit", "-m", fmt.Sprintf("morty: loop %d - COMPLETED", i))
	}

	// Get loop number
	loopNum, err := mgr.GetCurrentLoopNumber(tempDir)
	if err != nil {
		t.Fatalf("GetCurrentLoopNumber failed: %v", err)
	}

	if loopNum != 4 {
		t.Errorf("Expected loop number 4 (3 existing + 1), got %d", loopNum)
	}
}

// TestGetCurrentLoopNumberNoMortyCommits tests getting loop number with no morty commits.
func TestGetCurrentLoopNumberNoMortyCommits(t *testing.T) {
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

	// Create some regular commits
	for i := 1; i <= 3; i++ {
		file := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(file, []byte("content"), 0644)
		mgr.run(tempDir, "add", "-A")
		mgr.run(tempDir, "commit", "-m", fmt.Sprintf("Regular commit %d", i))
	}

	// Get loop number
	loopNum, err := mgr.GetCurrentLoopNumber(tempDir)
	if err != nil {
		t.Fatalf("GetCurrentLoopNumber failed: %v", err)
	}

	if loopNum != 1 {
		t.Errorf("Expected loop number 1 (no morty commits), got %d", loopNum)
	}
}

// TestGetCurrentLoopNumberNotARepo tests error handling for non-repo.
func TestGetCurrentLoopNumberNotARepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	_, err := mgr.GetCurrentLoopNumber(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}

	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("Expected 'not a git repository' error, got: %v", err)
	}
}

// TestGetCurrentLoopNumberVariousFormats tests parsing various commit message formats.
func TestGetCurrentLoopNumberVariousFormats(t *testing.T) {
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

	// Create an initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create commits with various morty formats
	formats := []struct {
		msg      string
		expected int
	}{
		{"morty: loop 1 - COMPLETED", 2},
		{"MORTY: LOOP 10 - RUNNING", 11}, // Case insensitive
		{"morty: loop 5 - PENDING", 11},  // 10 is still higher
		{"morty:loop 3 - DONE", 11},      // No space after colon
	}

	for i, tc := range formats {
		file := filepath.Join(tempDir, fmt.Sprintf("format%d.txt", i))
		os.WriteFile(file, []byte("content"), 0644)
		mgr.run(tempDir, "add", "-A")
		mgr.run(tempDir, "commit", "-m", tc.msg)
	}

	loopNum, err := mgr.GetCurrentLoopNumber(tempDir)
	if err != nil {
		t.Fatalf("GetCurrentLoopNumber failed: %v", err)
	}

	if loopNum != 11 {
		t.Errorf("Expected loop number 11, got %d", loopNum)
	}
}

// TestCreateBackupBranchSuccess tests successful backup branch creation.
func TestCreateBackupBranchSuccess(t *testing.T) {
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

	// Create an initial commit (required for branch creation)
	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create backup branch
	branchName, err := mgr.CreateBackupBranch(tempDir)
	if err != nil {
		t.Fatalf("CreateBackupBranch failed: %v", err)
	}

	// Verify branch name format
	if !strings.HasPrefix(branchName, "morty/backup-") {
		t.Errorf("Expected branch name to start with 'morty/backup-', got: %s", branchName)
	}

	// Verify branch exists
	branches, err := mgr.run(tempDir, "branch", "--list", branchName)
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	if !strings.Contains(branches, branchName) {
		t.Errorf("Branch %s was not created", branchName)
	}
}

// TestCreateBackupBranchNotARepo tests error handling for non-repo.
func TestCreateBackupBranchNotARepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	_, err := mgr.CreateBackupBranch(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}

	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("Expected 'not a git repository' error, got: %v", err)
	}
}

// TestCreateBackupBranchWithName tests creating a branch with specific name.
func TestCreateBackupBranchWithName(t *testing.T) {
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

	// Create an initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create branch with specific name
	customName := "morty/custom-backup"
	branchName, err := mgr.CreateBackupBranchWithName(tempDir, customName)
	if err != nil {
		t.Fatalf("CreateBackupBranchWithName failed: %v", err)
	}

	if branchName != customName {
		t.Errorf("Expected branch name '%s', got '%s'", customName, branchName)
	}

	// Verify branch exists
	branches, err := mgr.run(tempDir, "branch", "--list", customName)
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	if !strings.Contains(branches, customName) {
		t.Errorf("Branch %s was not created", customName)
	}
}

// TestIntegrationCommitWorkflow tests a complete commit workflow.
func TestIntegrationCommitWorkflow(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Step 1: Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Step 2: Configure git
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Step 3: Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Step 4: Get current loop number (should be 1)
	loopNum, err := mgr.GetCurrentLoopNumber(tempDir)
	if err != nil {
		t.Fatalf("GetCurrentLoopNumber failed: %v", err)
	}
	if loopNum != 1 {
		t.Errorf("Expected loop number 1, got %d", loopNum)
	}

	// Step 5: Create some changes
	testFile := filepath.Join(tempDir, "main.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644)

	// Step 6: Create loop commit
	commitHash, err := mgr.CreateLoopCommit(loopNum, "RUNNING", tempDir)
	if err != nil {
		t.Fatalf("CreateLoopCommit failed: %v", err)
	}
	if commitHash == "" {
		t.Error("Expected non-empty commit hash")
	}

	// Step 7: Verify next loop number
	loopNum, err = mgr.GetCurrentLoopNumber(tempDir)
	if err != nil {
		t.Fatalf("GetCurrentLoopNumber failed: %v", err)
	}
	if loopNum != 2 {
		t.Errorf("Expected loop number 2, got %d", loopNum)
	}

	// Step 8: Create backup branch
	branchName, err := mgr.CreateBackupBranchWithName(tempDir, "morty/backup-test")
	if err != nil {
		t.Fatalf("CreateBackupBranchWithName failed: %v", err)
	}
	if branchName != "morty/backup-test" {
		t.Errorf("Expected branch name 'morty/backup-test', got '%s'", branchName)
	}

	// Step 9: Create another commit
	os.WriteFile(testFile, []byte("package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}"), 0644)
	_, err = mgr.CreateLoopCommit(loopNum, "COMPLETED", tempDir)
	if err != nil {
		t.Fatalf("CreateLoopCommit failed: %v", err)
	}

	// Step 10: Verify commit log
	logOutput, err := mgr.run(tempDir, "log", "--oneline")
	if err != nil {
		t.Fatalf("Failed to get log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 commits, got %d", len(lines))
	}
}

// BenchmarkCreateLoopCommit benchmarks commit creation.
func BenchmarkCreateLoopCommit(b *testing.B) {
	mgr := NewManager()
	tempDir := b.TempDir()

	// Initialize repo
	mgr.InitIfNeeded(tempDir)
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a unique file for each iteration
		testFile := filepath.Join(tempDir, fmt.Sprintf("bench%d.txt", i))
		os.WriteFile(testFile, []byte("benchmark content"), 0644)

		_, err := mgr.CreateLoopCommit(i+1, "BENCHMARK", tempDir)
		if err != nil {
			b.Fatalf("CreateLoopCommit failed: %v", err)
		}
	}
}

// BenchmarkGetCurrentLoopNumber benchmarks loop number detection.
func BenchmarkGetCurrentLoopNumber(b *testing.B) {
	mgr := NewManager()
	tempDir := b.TempDir()

	// Initialize repo
	mgr.InitIfNeeded(tempDir)
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create initial commit
	initialFile := filepath.Join(tempDir, "initial.txt")
	os.WriteFile(initialFile, []byte("initial"), 0644)
	mgr.run(tempDir, "add", "initial.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create some morty commits
	for i := 1; i <= 10; i++ {
		file := filepath.Join(tempDir, fmt.Sprintf("bench%d.txt", i))
		os.WriteFile(file, []byte("content"), 0644)
		mgr.run(tempDir, "add", "-A")
		mgr.run(tempDir, "commit", "-m", fmt.Sprintf("morty: loop %d - COMPLETED", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mgr.GetCurrentLoopNumber(tempDir)
		if err != nil {
			b.Fatalf("GetCurrentLoopNumber failed: %v", err)
		}
	}
}
