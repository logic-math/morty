// Package git provides Git repository operations for Morty.
package git

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ResetMode represents the type of git reset to perform.
type ResetMode string

const (
	// HardReset resets the index and working tree, discarding all changes.
	HardReset ResetMode = "hard"
	// SoftReset resets only the HEAD to the specified commit, keeping changes staged.
	SoftReset ResetMode = "soft"
	// MixedReset resets the index but not the working tree (default git behavior).
	MixedReset ResetMode = "mixed"
)

// LoopCommit represents a single loop commit entry in the history.
type LoopCommit struct {
	// CommitHash is the full SHA-1 hash of the commit.
	CommitHash string
	// ShortHash is the abbreviated SHA-1 hash (first 7 characters).
	ShortHash string
	// LoopNumber is the loop number extracted from the commit message.
	LoopNumber int
	// Status is the status extracted from the commit message (e.g., COMPLETED, RUNNING).
	Status string
	// Message is the full commit message subject line.
	Message string
	// Author is the commit author name.
	Author string
	// Timestamp is the commit timestamp.
	Timestamp time.Time
	// Stats contains change statistics if available in the commit body.
	Stats *ChangeStats
}

// VersionController defines the interface for git version control operations.
type VersionController interface {
	// ResetToCommit resets the repository to a specific commit.
	// It creates a backup branch before performing the reset.
	// The mode parameter specifies whether to perform a hard or soft reset.
	//
	// Example:
	//   err := git.ResetToCommit("abc123", "/path/to/repo", git.HardReset)
	ResetToCommit(commitHash string, dir string, mode ResetMode) error

	// ShowLoopHistory returns the recent N loop commits from history.
	// The commits are returned in reverse chronological order (newest first).
	//
	// Example:
	//   history, err := git.ShowLoopHistory(10, "/path/to/repo")
	ShowLoopHistory(n int, dir string) ([]LoopCommit, error)

	// ParseCommitMessage parses a commit message and extracts loop number and status.
	//
	// Example:
	//   commit, err := git.ParseCommitMessage("morty: loop 5 - COMPLETED")
	ParseCommitMessage(message string) (*LoopCommit, error)

	// FormatLoopHistory formats loop history entries for display.
	//
	// Example:
	//   formatted := git.FormatLoopHistory(history)
	FormatLoopHistory(history []LoopCommit) string
}

// Ensure Manager implements VersionController interface.
var _ VersionController = (*Manager)(nil)

// ResetToCommit resets the repository to a specific commit.
// It automatically creates a backup branch before resetting.
func (m *Manager) ResetToCommit(commitHash string, dir string, mode ResetMode) error {
	// Check if it's a git repo first
	if !m.isGitRepo(dir) {
		return fmt.Errorf("directory %s is not a git repository", dir)
	}

	// Validate the commit hash exists
	_, err := m.run(dir, "cat-file", "-t", commitHash)
	if err != nil {
		return fmt.Errorf("invalid commit hash %s: %w", commitHash, err)
	}

	// Create backup branch before reset
	backupBranch, err := m.CreateBackupBranch(dir)
	if err != nil {
		return fmt.Errorf("failed to create backup branch: %w", err)
	}

	// Store backup branch info for potential rollback of rollback
	_ = backupBranch

	// Perform the reset with specified mode
	resetArgs := []string{"reset", "--" + string(mode), commitHash}
	_, err = m.run(dir, resetArgs...)
	if err != nil {
		return fmt.Errorf("failed to reset to commit %s with mode %s: %w", commitHash, mode, err)
	}

	return nil
}

// ResetToCommitWithBackup creates a named backup before reset.
// This allows for custom backup branch naming.
func (m *Manager) ResetToCommitWithBackup(commitHash string, dir string, mode ResetMode, backupName string) error {
	// Check if it's a git repo first
	if !m.isGitRepo(dir) {
		return fmt.Errorf("directory %s is not a git repository", dir)
	}

	// Validate the commit hash exists
	_, err := m.run(dir, "cat-file", "-t", commitHash)
	if err != nil {
		return fmt.Errorf("invalid commit hash %s: %w", commitHash, err)
	}

	// Create backup branch with custom name
	_, err = m.CreateBackupBranchWithName(dir, backupName)
	if err != nil {
		return fmt.Errorf("failed to create backup branch %s: %w", backupName, err)
	}

	// Perform the reset with specified mode
	resetArgs := []string{"reset", "--" + string(mode), commitHash}
	_, err = m.run(dir, resetArgs...)
	if err != nil {
		return fmt.Errorf("failed to reset to commit %s with mode %s: %w", commitHash, mode, err)
	}

	return nil
}

// ShowLoopHistory returns the recent N loop commits from history.
// Commits are returned in reverse chronological order (newest first).
func (m *Manager) ShowLoopHistory(n int, dir string) ([]LoopCommit, error) {
	// Check if it's a git repo first
	if !m.isGitRepo(dir) {
		return nil, fmt.Errorf("directory %s is not a git repository", dir)
	}

	if n <= 0 {
		return []LoopCommit{}, nil
	}

	// Get commit log with full format including hash, author, date, and message
	// Format: hash|author|date|subject
	format := "%H|%an|%at|%s"
	output, err := m.run(dir, "log", "--pretty=format:"+format, "-n", strconv.Itoa(n*3))
	if err != nil {
		// If there's no commit history yet, return empty slice
		if strings.Contains(err.Error(), "does not have any commits yet") {
			return []LoopCommit{}, nil
		}
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return []LoopCommit{}, nil
	}

	// Parse commits and filter for loop commits
	var loopCommits []LoopCommit
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		commit, err := m.parseLogLine(line)
		if err != nil {
			continue // Skip non-loop commits
		}

		// Only include commits that match morty loop pattern
		if commit.LoopNumber > 0 {
			loopCommits = append(loopCommits, *commit)
		}

		// Stop once we have N loop commits
		if len(loopCommits) >= n {
			break
		}
	}

	return loopCommits, nil
}

// parseLogLine parses a single log line in the format: hash|author|date|subject
func (m *Manager) parseLogLine(line string) (*LoopCommit, error) {
	parts := strings.SplitN(line, "|", 4)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid log line format")
	}

	commitHash := parts[0]
	author := parts[1]
	timestampStr := parts[2]
	subject := parts[3]

	// Parse timestamp
	timestampUnix, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}
	timestamp := time.Unix(timestampUnix, 0)

	// Extract loop number and status from message
	loopCommit, err := m.ParseCommitMessage(subject)
	if err != nil {
		// Still create a commit entry even if it's not a loop commit
		loopCommit = &LoopCommit{
			Message: subject,
		}
	}

	loopCommit.CommitHash = commitHash
	loopCommit.ShortHash = commitHash[:7]
	loopCommit.Author = author
	loopCommit.Timestamp = timestamp

	return loopCommit, nil
}

// ParseCommitMessage parses a commit message subject and extracts loop number and status.
// Expected format: "morty: loop [number] - [status]" or "morty: loop [number] - module/job - [status]"
func (m *Manager) ParseCommitMessage(message string) (*LoopCommit, error) {
	commit := &LoopCommit{
		Message: message,
	}

	// Try to match the extended format first: "morty: loop N - module/job - STATUS"
	// This handles commit messages like "morty: loop 3 - sudoku/job_3 - COMPLETED"
	reExtended := regexp.MustCompile(`(?i)morty:\s*loop\s+(\d+)\s*-\s*[\w/]+\s*-\s*(\w+)`)
	matches := reExtended.FindStringSubmatch(message)

	if len(matches) >= 3 {
		loopNum, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse loop number: %w", err)
		}
		commit.LoopNumber = loopNum
		commit.Status = strings.ToUpper(matches[2])
	} else {
		// Try the simple format: "morty: loop [number] - [status]"
		re := regexp.MustCompile(`(?i)morty:\s*loop\s+(\d+)\s*-\s*(\w+)`)
		matches := re.FindStringSubmatch(message)

		if len(matches) >= 3 {
			loopNum, err := strconv.Atoi(matches[1])
			if err != nil {
				return nil, fmt.Errorf("failed to parse loop number: %w", err)
			}
			commit.LoopNumber = loopNum
			commit.Status = strings.ToUpper(matches[2])
		} else {
			// Try alternative pattern: "morty: loop [number]" without status
			reSimple := regexp.MustCompile(`(?i)morty:\s*loop\s+(\d+)`)
			matchesSimple := reSimple.FindStringSubmatch(message)
			if len(matchesSimple) >= 2 {
				loopNum, err := strconv.Atoi(matchesSimple[1])
				if err != nil {
					return nil, fmt.Errorf("failed to parse loop number: %w", err)
				}
				commit.LoopNumber = loopNum
				commit.Status = "UNKNOWN"
			} else {
				return nil, fmt.Errorf("message does not match morty loop pattern")
			}
		}
	}

	return commit, nil
}

// FormatLoopHistory formats loop history entries for human-readable display.
// Returns a formatted string with columns for loop number, status, hash, and timestamp.
func (m *Manager) FormatLoopHistory(history []LoopCommit) string {
	if len(history) == 0 {
		return "No loop history found."
	}

	var sb strings.Builder

	// Header
	sb.WriteString("Loop History:\n")
	sb.WriteString(strings.Repeat("-", 60) + "\n")
	sb.WriteString(fmt.Sprintf("%-6s %-12s %-8s %-20s %s\n", "LOOP", "STATUS", "HASH", "TIME", "AUTHOR"))
	sb.WriteString(strings.Repeat("-", 60) + "\n")

	// Entries
	for _, commit := range history {
		timeStr := commit.Timestamp.Format("2006-01-02 15:04")
		sb.WriteString(fmt.Sprintf("%-6d %-12s %-8s %-20s %s\n",
			commit.LoopNumber,
			commit.Status,
			commit.ShortHash,
			timeStr,
			commit.Author,
		))
	}

	sb.WriteString(strings.Repeat("-", 60))
	return sb.String()
}

// FormatLoopHistoryDetailed returns a detailed formatted string with change statistics.
func (m *Manager) FormatLoopHistoryDetailed(history []LoopCommit) string {
	if len(history) == 0 {
		return "No loop history found."
	}

	var sb strings.Builder

	sb.WriteString("Detailed Loop History:\n")
	sb.WriteString(strings.Repeat("=", 70) + "\n\n")

	for i, commit := range history {
		sb.WriteString(fmt.Sprintf("[%d] Loop %d - %s\n", i+1, commit.LoopNumber, commit.Status))
		sb.WriteString(fmt.Sprintf("    Hash:    %s\n", commit.CommitHash))
		sb.WriteString(fmt.Sprintf("    Author:  %s\n", commit.Author))
		sb.WriteString(fmt.Sprintf("    Time:    %s\n", commit.Timestamp.Format("2006-01-02 15:04:05")))

		if commit.Stats != nil {
			sb.WriteString(fmt.Sprintf("    Changes: +%d/-%d lines, %d/%d/%d files (A/M/D)\n",
				commit.Stats.LinesAdded,
				commit.Stats.LinesDeleted,
				commit.Stats.FilesAdded,
				commit.Stats.FilesModified,
				commit.Stats.FilesDeleted,
			))
		}

		if i < len(history)-1 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString(strings.Repeat("=", 70))
	return sb.String()
}

// GetCommitAtLoop returns the commit hash for a specific loop number.
// Returns empty string if no commit found for that loop number.
func (m *Manager) GetCommitAtLoop(loopNumber int, dir string) (string, error) {
	history, err := m.ShowLoopHistory(100, dir) // Get enough history
	if err != nil {
		return "", err
	}

	for _, commit := range history {
		if commit.LoopNumber == loopNumber {
			return commit.CommitHash, nil
		}
	}

	return "", fmt.Errorf("no commit found for loop number %d", loopNumber)
}

// RollbackToLoop resets to the commit at a specific loop number.
// This is a convenience method that finds the commit and resets to it.
func (m *Manager) RollbackToLoop(loopNumber int, dir string, mode ResetMode) error {
	commitHash, err := m.GetCommitAtLoop(loopNumber, dir)
	if err != nil {
		return err
	}

	return m.ResetToCommit(commitHash, dir, mode)
}
