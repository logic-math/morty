package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitInterface verifies that Manager implements Git interface.
func TestGitInterface(t *testing.T) {
	var _ Git = (*Manager)(nil)
	mgr := NewManager()
	var git Git = mgr
	if git == nil {
		t.Fatal("Manager does not implement Git interface")
	}
}

// TestNewManager tests the constructor.
func TestNewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
	if mgr.gitPath != "git" {
		t.Errorf("expected gitPath to be 'git', got %s", mgr.gitPath)
	}
}

// TestNewManagerWithPath tests the constructor with custom path.
func TestNewManagerWithPath(t *testing.T) {
	mgr := NewManagerWithPath("/usr/local/bin/git")
	if mgr == nil {
		t.Fatal("NewManagerWithPath() returned nil")
	}
	if mgr.gitPath != "/usr/local/bin/git" {
		t.Errorf("expected gitPath to be '/usr/local/bin/git', got %s", mgr.gitPath)
	}
}

// TestInitIfNeeded_NotARepo tests initializing a new repository.
func TestInitIfNeeded_NotARepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Verify it's not a git repo initially
	if mgr.isGitRepo(tempDir) {
		t.Fatal("tempDir should not be a git repo initially")
	}

	// Initialize the repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Verify it's now a git repo
	if !mgr.isGitRepo(tempDir) {
		t.Fatal("tempDir should be a git repo after InitIfNeeded")
	}

	// Verify .git directory exists
	gitDir := filepath.Join(tempDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Fatal(".git directory should exist after InitIfNeeded")
	}
}

// TestInitIfNeeded_AlreadyARepo tests InitIfNeeded on an existing repo.
func TestInitIfNeeded_AlreadyARepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize once
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("First InitIfNeeded failed: %v", err)
	}

	// Create a file and commit to verify repo state is preserved
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Stage the file
	_, err = mgr.run(tempDir, "add", "test.txt")
	if err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// Configure git user for commit
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Commit
	_, err = mgr.run(tempDir, "commit", "-m", "initial commit")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Call InitIfNeeded again - should not fail or change anything
	err = mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("Second InitIfNeeded failed: %v", err)
	}

	// Verify the commit still exists
	output, err := mgr.run(tempDir, "log", "--oneline")
	if err != nil {
		t.Fatalf("Failed to check git log: %v", err)
	}
	if !strings.Contains(output, "initial commit") {
		t.Fatal("Repository state was modified by second InitIfNeeded")
	}
}

// TestHasUncommittedChanges_NoChanges tests with a clean repo.
func TestHasUncommittedChanges_NoChanges(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize and setup repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	mgr.run(tempDir, "add", "test.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Check for uncommitted changes - should be false
	hasChanges, err := mgr.HasUncommittedChanges(tempDir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("Expected no uncommitted changes, got true")
	}
}

// TestHasUncommittedChanges_WithChanges tests with uncommitted changes.
func TestHasUncommittedChanges_WithChanges(t *testing.T) {
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

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	mgr.run(tempDir, "add", "test.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Modify the file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Check for uncommitted changes - should be true
	hasChanges, err := mgr.HasUncommittedChanges(tempDir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges failed: %v", err)
	}
	if !hasChanges {
		t.Error("Expected uncommitted changes, got false")
	}
}

// TestHasUncommittedChanges_StagedChanges tests with staged changes.
func TestHasUncommittedChanges_StagedChanges(t *testing.T) {
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

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	mgr.run(tempDir, "add", "test.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create a new file and stage it
	newFile := filepath.Join(tempDir, "newfile.txt")
	err = os.WriteFile(newFile, []byte("new content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}
	mgr.run(tempDir, "add", "newfile.txt")

	// Check for uncommitted changes - should be true (staged but not committed)
	hasChanges, err := mgr.HasUncommittedChanges(tempDir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges failed: %v", err)
	}
	if !hasChanges {
		t.Error("Expected uncommitted changes for staged file, got false")
	}
}

// TestHasUncommittedChanges_NotARepo tests error handling for non-repo.
func TestHasUncommittedChanges_NotARepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	_, err := mgr.HasUncommittedChanges(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}
}

// TestGetRepoRoot_CurrentDir tests getting repo root from the root itself.
func TestGetRepoRoot_CurrentDir(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Get repo root
	root, err := mgr.GetRepoRoot(tempDir)
	if err != nil {
		t.Fatalf("GetRepoRoot failed: %v", err)
	}

	// Should match tempDir
	expected, _ := filepath.Abs(tempDir)
	if root != expected {
		t.Errorf("Expected root %s, got %s", expected, root)
	}
}

// TestGetRepoRoot_SubDir tests getting repo root from a subdirectory.
func TestGetRepoRoot_SubDir(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir", "nested")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Get repo root from subdirectory
	root, err := mgr.GetRepoRoot(subDir)
	if err != nil {
		t.Fatalf("GetRepoRoot failed: %v", err)
	}

	// Should match tempDir
	expected, _ := filepath.Abs(tempDir)
	if root != expected {
		t.Errorf("Expected root %s, got %s", expected, root)
	}
}

// TestGetRepoRoot_NotARepo tests error handling for non-repo.
func TestGetRepoRoot_NotARepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	_, err := mgr.GetRepoRoot(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}
}

// TestGetChangeStats_NoChanges tests stats with no changes.
func TestGetChangeStats_NoChanges(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize and setup repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content\nline2\nline3\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	mgr.run(tempDir, "add", "test.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Get stats
	stats, err := mgr.GetChangeStats(tempDir)
	if err != nil {
		t.Fatalf("GetChangeStats failed: %v", err)
	}

	if stats.FilesAdded != 0 {
		t.Errorf("Expected 0 files added, got %d", stats.FilesAdded)
	}
	if stats.FilesModified != 0 {
		t.Errorf("Expected 0 files modified, got %d", stats.FilesModified)
	}
	if stats.FilesDeleted != 0 {
		t.Errorf("Expected 0 files deleted, got %d", stats.FilesDeleted)
	}
}

// TestGetChangeStats_WithUnstagedChanges tests stats with unstaged changes.
func TestGetChangeStats_WithUnstagedChanges(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize and setup repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	mgr.run(tempDir, "add", "test.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Modify the file (unstaged)
	err = os.WriteFile(testFile, []byte("line1\nmodified line2\nline3\nnew line4\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Get stats
	stats, err := mgr.GetChangeStats(tempDir)
	if err != nil {
		t.Fatalf("GetChangeStats failed: %v", err)
	}

	if stats.FilesModified != 1 {
		t.Errorf("Expected 1 file modified, got %d", stats.FilesModified)
	}

	// Should detect line changes (1 line modified + 1 line added = 2 insertions, 1 deletion)
	if stats.LinesAdded == 0 {
		t.Error("Expected non-zero lines added")
	}
}

// TestGetChangeStats_WithStagedChanges tests stats with staged changes.
func TestGetChangeStats_WithStagedChanges(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize and setup repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("line1\nline2\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	mgr.run(tempDir, "add", "test.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Create a new file and stage it
	newFile := filepath.Join(tempDir, "newfile.txt")
	err = os.WriteFile(newFile, []byte("new content\nline2\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}
	mgr.run(tempDir, "add", "newfile.txt")

	// Get stats
	stats, err := mgr.GetChangeStats(tempDir)
	if err != nil {
		t.Fatalf("GetChangeStats failed: %v", err)
	}

	if stats.FilesAdded != 1 {
		t.Errorf("Expected 1 file added, got %d", stats.FilesAdded)
	}

	// Should detect line changes
	if stats.LinesAdded == 0 {
		t.Error("Expected non-zero lines added for new file")
	}
}

// TestGetChangeStats_WithDeletedFile tests stats with deleted files.
func TestGetChangeStats_WithDeletedFile(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize and setup repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Configure git user
	mgr.run(tempDir, "config", "user.email", "test@test.com")
	mgr.run(tempDir, "config", "user.name", "Test User")

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("line1\nline2\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	mgr.run(tempDir, "add", "test.txt")
	mgr.run(tempDir, "commit", "-m", "initial commit")

	// Delete the file
	err = os.Remove(testFile)
	if err != nil {
		t.Fatalf("Failed to delete test file: %v", err)
	}

	// Get stats
	stats, err := mgr.GetChangeStats(tempDir)
	if err != nil {
		t.Fatalf("GetChangeStats failed: %v", err)
	}

	if stats.FilesDeleted != 1 {
		t.Errorf("Expected 1 file deleted, got %d", stats.FilesDeleted)
	}
}

// TestGetChangeStats_NotARepo tests error handling for non-repo.
func TestGetChangeStats_NotARepo(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	_, err := mgr.GetChangeStats(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}
}

// TestParseDiffStat tests the diff stat parsing function.
func TestParseDiffStat(t *testing.T) {
	mgr := NewManager()

	tests := []struct {
		name       string
		input      string
		insertions int
		deletions  int
	}{
		{
			name:       "empty output",
			input:      "",
			insertions: 0,
			deletions:  0,
		},
		{
			name:       "single file with insertions",
			input:      " file.go | 5 +++++\n 1 file changed, 5 insertions(+)",
			insertions: 5,
			deletions:  0,
		},
		{
			name:       "single file with deletions",
			input:      " file.go | 3 ---\n 1 file changed, 3 deletions(-)",
			insertions: 0,
			deletions:  3,
		},
		{
			name:       "file with both insertions and deletions",
			input:      " file.go | 10 +++++-----\n 1 file changed, 5 insertions(+), 5 deletions(-)",
			insertions: 5,
			deletions:  5,
		},
		{
			name:       "multiple files",
			input:      " file1.go | 10 ++++++++++\n file2.go | 5 -----\n 2 files changed, 10 insertions(+), 5 deletions(-)",
			insertions: 10,
			deletions:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insertions, deletions := mgr.parseDiffStat(tt.input)
			if insertions != tt.insertions {
				t.Errorf("Expected %d insertions, got %d", tt.insertions, insertions)
			}
			if deletions != tt.deletions {
				t.Errorf("Expected %d deletions, got %d", tt.deletions, deletions)
			}
		})
	}
}

// TestRun tests the internal run method.
func TestRun(t *testing.T) {
	mgr := NewManager()

	// Test successful command
	output, err := mgr.run("", "version")
	if err != nil {
		t.Fatalf("run(git version) failed: %v", err)
	}
	if output == "" {
		t.Error("Expected non-empty output for git version")
	}
	if !strings.Contains(output, "git version") {
		t.Errorf("Expected 'git version' in output, got: %s", output)
	}
}

// TestRun_InvalidCommand tests error handling for invalid commands.
func TestRun_InvalidCommand(t *testing.T) {
	mgr := NewManager()

	// Test invalid command
	_, err := mgr.run("", "invalid-command-xyz")
	if err == nil {
		t.Error("Expected error for invalid command, got nil")
	}
}

// TestRun_WithDir tests running command in specific directory.
func TestRun_WithDir(t *testing.T) {
	mgr := NewManager()
	tempDir := t.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		t.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Run command in tempDir
	output, err := mgr.run(tempDir, "rev-parse", "--git-dir")
	if err != nil {
		t.Fatalf("run in tempDir failed: %v", err)
	}
	if output != ".git" {
		t.Errorf("Expected '.git', got: %s", output)
	}
}

// TestIntegration_FullWorkflow tests a complete workflow.
func TestIntegration_FullWorkflow(t *testing.T) {
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

	// Step 3: Check no changes initially
	hasChanges, err := mgr.HasUncommittedChanges(tempDir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("Expected no changes initially")
	}

	// Step 4: Get repo root
	root, err := mgr.GetRepoRoot(tempDir)
	if err != nil {
		t.Fatalf("GetRepoRoot failed: %v", err)
	}
	expectedRoot, _ := filepath.Abs(tempDir)
	if root != expectedRoot {
		t.Errorf("Expected root %s, got %s", expectedRoot, root)
	}

	// Step 5: Create a file
	testFile := filepath.Join(tempDir, "main.go")
	content := `package main

func main() {
	println("Hello, World!")
}
`
	err = os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Step 6: Check for changes
	hasChanges, err = mgr.HasUncommittedChanges(tempDir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges failed: %v", err)
	}
	if !hasChanges {
		t.Error("Expected changes after creating file")
	}

	// Step 7: Get change stats
	stats, err := mgr.GetChangeStats(tempDir)
	if err != nil {
		t.Fatalf("GetChangeStats failed: %v", err)
	}
	if stats.FilesAdded != 1 {
		t.Errorf("Expected 1 file added, got %d", stats.FilesAdded)
	}

	// Step 8: Stage and commit
	_, err = mgr.run(tempDir, "add", "main.go")
	if err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}
	_, err = mgr.run(tempDir, "commit", "-m", "Add main.go")
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Step 9: Verify no changes after commit
	hasChanges, err = mgr.HasUncommittedChanges(tempDir)
	if err != nil {
		t.Fatalf("HasUncommittedChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("Expected no changes after commit")
	}

	// Step 10: Modify the file
	newContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	err = os.WriteFile(testFile, []byte(newContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Step 11: Get stats for modified file
	stats, err = mgr.GetChangeStats(tempDir)
	if err != nil {
		t.Fatalf("GetChangeStats failed: %v", err)
	}
	if stats.FilesModified != 1 {
		t.Errorf("Expected 1 file modified, got %d", stats.FilesModified)
	}
	if stats.LinesAdded == 0 {
		t.Error("Expected non-zero lines added")
	}
	if stats.LinesDeleted == 0 {
		t.Error("Expected non-zero lines deleted")
	}
}

// BenchmarkInitIfNeeded benchmarks repository initialization.
func BenchmarkInitIfNeeded(b *testing.B) {
	mgr := NewManager()
	tempDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use a subdirectory for each iteration
		subDir := filepath.Join(tempDir, fmt.Sprintf("repo%d", i))
		os.MkdirAll(subDir, 0755)
		err := mgr.InitIfNeeded(subDir)
		if err != nil {
			b.Fatalf("InitIfNeeded failed: %v", err)
		}
	}
}

// BenchmarkHasUncommittedChanges benchmarks change detection.
func BenchmarkHasUncommittedChanges(b *testing.B) {
	mgr := NewManager()
	tempDir := b.TempDir()

	// Initialize repo
	err := mgr.InitIfNeeded(tempDir)
	if err != nil {
		b.Fatalf("InitIfNeeded failed: %v", err)
	}

	// Create and modify a file
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mgr.HasUncommittedChanges(tempDir)
		if err != nil {
			b.Fatalf("HasUncommittedChanges failed: %v", err)
		}
	}
}
