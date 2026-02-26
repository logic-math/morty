// Package config provides configuration management for Morty.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Paths provides centralized path management for Morty.
// It handles path resolution, expansion, and directory creation.
type Paths struct {
	loader     *Loader
	workDir    string
	configFile string
	promptsDir string
}

// NewPaths creates a new Paths instance with default values.
func NewPaths() *Paths {
	return &Paths{
		workDir: DefaultWorkDir,
	}
}

// NewPathsWithLoader creates a new Paths instance with a config loader.
func NewPathsWithLoader(loader *Loader) *Paths {
	return &Paths{
		loader:     loader,
		workDir:    DefaultWorkDir,
		configFile: loader.GetConfigFile(),
	}
}

// SetWorkDir sets a custom working directory.
func (p *Paths) SetWorkDir(dir string) {
	p.workDir = dir
}

// GetWorkDir returns the Morty working directory path.
// Returns absolute path with special characters properly handled.
func (p *Paths) GetWorkDir() string {
	// If custom workdir is set, use it
	if p.workDir != "" {
		return p.resolvePath(p.workDir)
	}
	return p.resolvePath(DefaultWorkDir)
}

// GetLogDir returns the log directory path.
// Returns: .morty/doing/logs (or equivalent based on workDir)
func (p *Paths) GetLogDir() string {
	return filepath.Join(p.GetWorkDir(), "doing", "logs")
}

// GetResearchDir returns the research directory path.
// Returns: .morty/research (or equivalent based on workDir)
func (p *Paths) GetResearchDir() string {
	return filepath.Join(p.GetWorkDir(), "research")
}

// GetPlanDir returns the plan directory path.
// Respects configuration value if loader is available.
func (p *Paths) GetPlanDir() string {
	if p.loader != nil && p.loader.config != nil && p.loader.config.Plan.Dir != "" {
		return p.resolvePath(p.loader.config.Plan.Dir)
	}
	return p.resolvePath(DefaultPlanDir)
}

// GetStatusFile returns the status file path.
// Respects configuration value if loader is available.
func (p *Paths) GetStatusFile() string {
	if p.loader != nil && p.loader.config != nil && p.loader.config.State.File != "" {
		return p.resolvePath(p.loader.config.State.File)
	}
	return p.resolvePath(DefaultStateFile)
}

// GetConfigFile returns the configuration file path.
func (p *Paths) GetConfigFile() string {
	if p.configFile != "" {
		return p.resolvePath(p.configFile)
	}
	// Default to project config file
	return p.resolvePath(DefaultProjectConfigFile)
}

// SetConfigFile sets the configuration file path.
func (p *Paths) SetConfigFile(path string) {
	p.configFile = path
}

// EnsureDir ensures a directory exists, creating it if necessary.
// Returns an error if the directory cannot be created.
func (p *Paths) EnsureDir(path string) error {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	// Check if directory already exists
	info, err := os.Stat(absPath)
	if err == nil && info.IsDir() {
		return nil // Directory already exists
	}

	// Create directory with parents
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", absPath, err)
	}

	return nil
}

// EnsureWorkDir ensures the working directory exists.
func (p *Paths) EnsureWorkDir() error {
	return p.EnsureDir(p.GetWorkDir())
}

// EnsureLogDir ensures the log directory exists.
func (p *Paths) EnsureLogDir() error {
	return p.EnsureDir(p.GetLogDir())
}

// EnsureResearchDir ensures the research directory exists.
func (p *Paths) EnsureResearchDir() error {
	return p.EnsureDir(p.GetResearchDir())
}

// EnsurePlanDir ensures the plan directory exists.
func (p *Paths) EnsurePlanDir() error {
	return p.EnsureDir(p.GetPlanDir())
}

// EnsureAllDirs ensures all standard directories exist.
func (p *Paths) EnsureAllDirs() error {
	if err := p.EnsureWorkDir(); err != nil {
		return fmt.Errorf("failed to ensure work directory: %w", err)
	}
	if err := p.EnsureLogDir(); err != nil {
		return fmt.Errorf("failed to ensure log directory: %w", err)
	}
	if err := p.EnsureResearchDir(); err != nil {
		return fmt.Errorf("failed to ensure research directory: %w", err)
	}
	if err := p.EnsurePlanDir(); err != nil {
		return fmt.Errorf("failed to ensure plan directory: %w", err)
	}
	return nil
}

// PathExists checks if a path exists.
func (p *Paths) PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir checks if a path is a directory.
func (p *Paths) IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// resolvePath resolves a path to an absolute path.
// Handles home directory expansion (~) and converts to absolute path.
func (p *Paths) resolvePath(path string) string {
	if path == "" {
		return ""
	}

	// Handle ~ at the start for home directory expansion
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[1:])
		}
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path // Return original if can't make absolute
	}

	return absPath
}

// GetAbsolutePath converts any path to an absolute path.
func (p *Paths) GetAbsolutePath(path string) string {
	return p.resolvePath(path)
}

// Join joins path elements with the OS-specific separator.
func (p *Paths) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// SanitizePath sanitizes a path by cleaning it and removing dangerous elements.
func (p *Paths) SanitizePath(path string) string {
	// Clean the path (remove .., ., etc.)
	cleaned := filepath.Clean(path)

	// Ensure it doesn't escape the working directory
	workDir := p.GetWorkDir()
	absCleaned, _ := filepath.Abs(cleaned)
	absWorkDir, _ := filepath.Abs(workDir)

	// If cleaned path is outside workDir, return workDir
	if !strings.HasPrefix(absCleaned, absWorkDir) {
		return absWorkDir
	}

	return absCleaned
}

// GetPromptsDir returns the prompts directory path.
func (p *Paths) GetPromptsDir() string {
	// If custom prompts dir is set, use it
	if p.promptsDir != "" {
		return p.resolvePath(p.promptsDir)
	}
	if p.loader != nil && p.loader.config != nil && p.loader.config.Prompts.Dir != "" {
		return p.resolvePath(p.loader.config.Prompts.Dir)
	}
	return p.resolvePath(DefaultPromptsDir)
}

// SetPromptsDir sets a custom prompts directory.
func (p *Paths) SetPromptsDir(dir string) {
	p.promptsDir = dir
}

// EnsurePromptsDir ensures the prompts directory exists.
func (p *Paths) EnsurePromptsDir() error {
	return p.EnsureDir(p.GetPromptsDir())
}
