package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Manager provides Git operations implementation.
// It uses the git command-line tool to perform operations.
type Manager struct {
	// gitPath is the path to the git executable.
	// If empty, "git" is used (assumes it's in PATH).
	gitPath string
}

// NewManager creates a new Git Manager instance.
// It uses "git" from PATH by default.
func NewManager() *Manager {
	return &Manager{
		gitPath: "git",
	}
}

// NewManagerWithPath creates a new Git Manager with a specific git executable path.
func NewManagerWithPath(gitPath string) *Manager {
	return &Manager{
		gitPath: gitPath,
	}
}

// run executes a git command in the specified directory and returns the output.
func (m *Manager) run(dir string, args ...string) (string, error) {
	cmd := exec.Command(m.gitPath, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", fmt.Errorf("git %s: %w (stderr: %s)",
				strings.Join(args, " "), err, stderrStr)
		}
		return "", fmt.Errorf("git %s: %w",
			strings.Join(args, " "), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// isGitRepo checks if the specified directory is inside a Git repository.
func (m *Manager) isGitRepo(dir string) bool {
	_, err := m.run(dir, "rev-parse", "--git-dir")
	return err == nil
}

// InitIfNeeded initializes a Git repository if one doesn't exist.
// If the directory is already a Git repository, it does nothing.
func (m *Manager) InitIfNeeded(dir string) error {
	// Check if already a git repository
	if m.isGitRepo(dir) {
		return nil
	}

	// Initialize new repository
	_, err := m.run(dir, "init")
	if err != nil {
		return fmt.Errorf("failed to initialize git repository in %s: %w", dir, err)
	}

	return nil
}

// HasUncommittedChanges checks if there are uncommitted changes.
// Returns true if there are staged or unstaged changes.
func (m *Manager) HasUncommittedChanges(dir string) (bool, error) {
	// Check if it's a git repo first
	if !m.isGitRepo(dir) {
		return false, fmt.Errorf("directory %s is not a git repository", dir)
	}

	// Check for any changes (staged or unstaged)
	// --porcelain gives machine-readable output
	output, err := m.run(dir, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	// If output is empty, no changes
	return output != "", nil
}

// GetRepoRoot returns the root directory of the git repository.
func (m *Manager) GetRepoRoot(dir string) (string, error) {
	// Check if it's a git repo first
	if !m.isGitRepo(dir) {
		return "", fmt.Errorf("directory %s is not inside a git repository", dir)
	}

	// Get the top-level directory
	output, err := m.run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	// Clean up the path
	root, err := filepath.Abs(output)
	if err != nil {
		return output, nil // Return original if Abs fails
	}

	return root, nil
}

// GetChangeStats returns statistics about uncommitted changes.
func (m *Manager) GetChangeStats(dir string) (*ChangeStats, error) {
	// Check if it's a git repo first
	if !m.isGitRepo(dir) {
		return nil, fmt.Errorf("directory %s is not a git repository", dir)
	}

	stats := &ChangeStats{}

	// Get file status counts using --porcelain
	output, err := m.run(dir, "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to get change status: %w", err)
	}

	// Parse porcelain output
	// Format: XY filename or XY filename -> newname for renames
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		// X is index status, Y is working tree status
		x := line[0]
		y := line[1]

		// Handle untracked files (??)
		if x == '?' && y == '?' {
			stats.FilesAdded++
			continue
		}

		// Use the most significant status
		status := x
		if x == ' ' {
			status = y
		}

		switch status {
		case 'A':
			stats.FilesAdded++
		case 'M':
			stats.FilesModified++
		case 'D':
			stats.FilesDeleted++
		case 'R':
			stats.FilesModified++ // Renames count as modified for simplicity
		case 'C':
			stats.FilesAdded++ // Copies count as added for simplicity
		}
	}

	// Get line statistics using diff --stat
	// First try cached (staged) changes
	stagedOutput, stagedErr := m.run(dir, "diff", "--cached", "--stat")
	// Then try unstaged changes
	unstagedOutput, unstagedErr := m.run(dir, "diff", "--stat")

	// Parse staged diff stats
	if stagedErr == nil && stagedOutput != "" {
		add, del := m.parseDiffStat(stagedOutput)
		stats.LinesAdded += add
		stats.LinesDeleted += del
	}

	// Parse unstaged diff stats
	if unstagedErr == nil && unstagedOutput != "" {
		add, del := m.parseDiffStat(unstagedOutput)
		stats.LinesAdded += add
		stats.LinesDeleted += del
	}

	return stats, nil
}

// RunGitCommand runs an arbitrary git command in the specified directory.
// It returns the command output and any error encountered.
// This is a low-level method for executing custom git commands.
func (m *Manager) RunGitCommand(dir string, args ...string) (string, error) {
	return m.run(dir, args...)
}

// parseDiffStat parses the output of git diff --stat to extract line counts.
// The format typically ends with "X insertions(+), Y deletions(-)" on the summary line.
func (m *Manager) parseDiffStat(output string) (insertions, deletions int) {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return 0, 0
	}

	// Find the summary line (usually the last non-empty line)
	var summary string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			summary = line
			break
		}
	}

	if summary == "" {
		return 0, 0
	}

	// Parse insertions
	if idx := strings.Index(summary, "insertion"); idx != -1 {
		// Find the number before "insertion"
		before := summary[:idx]
		parts := strings.Fields(before)
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			// Remove any trailing non-digit characters (like commas)
			last = strings.TrimRight(last, ",")
			if n, err := strconv.Atoi(last); err == nil {
				insertions = n
			}
		}
	}

	// Parse deletions
	if idx := strings.Index(summary, "deletion"); idx != -1 {
		// Find the number before "deletion"
		before := summary[:idx]
		parts := strings.Fields(before)
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			// Remove any trailing non-digit characters (like commas)
			last = strings.TrimRight(last, ",")
			if n, err := strconv.Atoi(last); err == nil {
				deletions = n
			}
		}
	}

	return insertions, deletions
}
