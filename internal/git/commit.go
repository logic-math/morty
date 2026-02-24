// Package git provides Git repository operations for Morty.
package git

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Committer defines the interface for git commit operations.
type Committer interface {
	// CreateLoopCommit creates a commit with loop number and status in the message.
	// It automatically stages all changes before committing.
	// Returns the commit hash and any error encountered.
	CreateLoopCommit(loopNumber int, status string, dir string) (string, error)

	// GetCurrentLoopNumber returns the next loop number based on commit history.
	// It parses commit messages to find the highest loop number used.
	GetCurrentLoopNumber(dir string) (int, error)

	// CreateBackupBranch creates a backup branch with timestamp.
	// Returns the branch name and any error encountered.
	CreateBackupBranch(dir string) (string, error)
}

// Ensure Manager implements Committer interface.
var _ Committer = (*Manager)(nil)

// CreateLoopCommit creates a commit with the loop number and status.
// It automatically stages all changes and creates a commit with a formatted message.
// The commit message format is: "morty: loop [number] - [status]"
// It also includes change statistics in the commit body.
func (m *Manager) CreateLoopCommit(loopNumber int, status string, dir string) (string, error) {
	// Check if it's a git repo first
	if !m.isGitRepo(dir) {
		return "", fmt.Errorf("directory %s is not a git repository", dir)
	}

	// Stage all changes
	_, err := m.run(dir, "add", "-A")
	if err != nil {
		return "", fmt.Errorf("failed to stage changes: %w", err)
	}

	// Check if there are changes to commit
	hasChanges, err := m.HasUncommittedChanges(dir)
	if err != nil {
		return "", fmt.Errorf("failed to check for changes: %w", err)
	}

	if !hasChanges {
		return "", fmt.Errorf("no changes to commit")
	}

	// Get change statistics
	stats, err := m.GetChangeStats(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get change statistics: %w", err)
	}

	// Build commit message
	commitMsg := buildCommitMessage(loopNumber, status, stats)

	// Create commit
	_, err = m.run(dir, "commit", "-m", commitMsg)
	if err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}

	// Extract commit hash
	commitHash, err := m.run(dir, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return commitHash, nil
}

// buildCommitMessage builds a formatted commit message with loop number, status, and stats.
func buildCommitMessage(loopNumber int, status string, stats *ChangeStats) string {
	var sb strings.Builder

	// Subject line: morty: loop [number] - [status]
	sb.WriteString(fmt.Sprintf("morty: loop %d - %s", loopNumber, status))

	// Body: empty line then stats
	sb.WriteString("\n\n")
	sb.WriteString("Change Statistics:\n")
	sb.WriteString(fmt.Sprintf("- Files added: %d\n", stats.FilesAdded))
	sb.WriteString(fmt.Sprintf("- Files modified: %d\n", stats.FilesModified))
	sb.WriteString(fmt.Sprintf("- Files deleted: %d\n", stats.FilesDeleted))
	sb.WriteString(fmt.Sprintf("- Lines added: %d\n", stats.LinesAdded))
	sb.WriteString(fmt.Sprintf("- Lines deleted: %d", stats.LinesDeleted))

	return sb.String()
}

// GetCurrentLoopNumber returns the next loop number based on existing commits.
// It searches commit history for morty loop commits and returns the highest number + 1.
func (m *Manager) GetCurrentLoopNumber(dir string) (int, error) {
	// Check if it's a git repo first
	if !m.isGitRepo(dir) {
		return 0, fmt.Errorf("directory %s is not a git repository", dir)
	}

	// Get commit log with subject lines only
	output, err := m.run(dir, "log", "--pretty=format:%s")
	if err != nil {
		// If there's no commit history yet, start from 1
		if strings.Contains(err.Error(), "does not have any commits yet") {
			return 1, nil
		}
		return 0, fmt.Errorf("failed to get commit log: %w", err)
	}

	// If no commits, start from 1
	if strings.TrimSpace(output) == "" {
		return 1, nil
	}

	// Parse commit messages to find the highest loop number
	maxLoop := 0
	lines := strings.Split(output, "\n")

	// Regex to match "morty: loop [number]" or similar patterns
	// Matches patterns like: "morty: loop 1 - COMPLETED" or "morty: loop 5 - RUNNING"
	re := regexp.MustCompile(`(?i)morty:\s*loop\s+(\d+)`)

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			num, err := strconv.Atoi(matches[1])
			if err == nil && num > maxLoop {
				maxLoop = num
			}
		}
	}

	// Return next loop number (max + 1), or 1 if no morty commits found
	if maxLoop == 0 {
		return 1, nil
	}
	return maxLoop + 1, nil
}

// CreateBackupBranch creates a backup branch with a timestamp.
// The branch name format is: morty/backup-[timestamp]
// Returns the branch name created.
func (m *Manager) CreateBackupBranch(dir string) (string, error) {
	// Check if it's a git repo first
	if !m.isGitRepo(dir) {
		return "", fmt.Errorf("directory %s is not a git repository", dir)
	}

	// Generate branch name with timestamp
	timestamp := time.Now().Format("20060102-150405")
	branchName := fmt.Sprintf("morty/backup-%s", timestamp)

	// Create the branch
	_, err := m.run(dir, "branch", branchName)
	if err != nil {
		return "", fmt.Errorf("failed to create backup branch: %w", err)
	}

	return branchName, nil
}

// CreateBackupBranchWithName creates a backup branch with a specific name.
// This is a helper function for testing and custom branch naming.
func (m *Manager) CreateBackupBranchWithName(dir string, branchName string) (string, error) {
	// Check if it's a git repo first
	if !m.isGitRepo(dir) {
		return "", fmt.Errorf("directory %s is not a git repository", dir)
	}

	// Create the branch
	_, err := m.run(dir, "branch", branchName)
	if err != nil {
		return "", fmt.Errorf("failed to create backup branch: %w", err)
	}

	return branchName, nil
}
