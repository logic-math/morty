// Package git provides Git repository operations for Morty.
// It abstracts common Git commands needed for the AI coding agent workflow.
package git

// Git defines the interface for Git operations.
// It provides methods for repository initialization, change detection,
// and repository metadata queries.
type Git interface {
	// InitIfNeeded initializes a Git repository in the specified directory
	// if one doesn't already exist. If the directory is already a Git repository,
	// no action is taken and no error is returned.
	//
	// Example:
	//   err := git.InitIfNeeded("/path/to/project")
	InitIfNeeded(dir string) error

	// HasUncommittedChanges checks if there are any uncommitted changes
	// in the Git repository at the specified directory.
	// Returns true if there are staged or unstaged changes.
	//
	// Example:
	//   hasChanges, err := git.HasUncommittedChanges("/path/to/project")
	HasUncommittedChanges(dir string) (bool, error)

	// GetRepoRoot returns the root directory of the Git repository
	// that contains the specified directory. Returns an error if
	// the directory is not inside a Git repository.
	//
	// Example:
	//   root, err := git.GetRepoRoot("/path/to/project/subdir")
	GetRepoRoot(dir string) (string, error)

	// GetChangeStats returns statistics about uncommitted changes
	// in the repository, including files added, modified, deleted,
	// and lines changed.
	//
	// Example:
	//   stats, err := git.GetChangeStats("/path/to/project")
	GetChangeStats(dir string) (*ChangeStats, error)
}

// ChangeStats holds statistics about repository changes.
type ChangeStats struct {
	// FilesAdded is the number of new files staged or unstaged
	FilesAdded int
	// FilesModified is the number of modified files
	FilesModified int
	// FilesDeleted is the number of deleted files
	FilesDeleted int
	// LinesAdded is the number of lines added (approximate)
	LinesAdded int
	// LinesDeleted is the number of lines deleted (approximate)
	LinesDeleted int
}

// ChangeType represents the type of change in a file.
type ChangeType string

const (
	// Added represents a newly added file.
	Added ChangeType = "A"
	// Modified represents a modified file.
	Modified ChangeType = "M"
	// Deleted represents a deleted file.
	Deleted ChangeType = "D"
	// Renamed represents a renamed file.
	Renamed ChangeType = "R"
	// Copied represents a copied file.
	Copied ChangeType = "C"
)

// Ensure Manager implements Git interface.
var _ Git = (*Manager)(nil)
