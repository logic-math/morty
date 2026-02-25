// Package logging provides a structured logging interface for Morty.
package logging

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Rotator handles log file rotation based on size and retention policies.
type Rotator struct {
	maxSize    int64  // Maximum file size in bytes before rotation
	maxBackups int    // Maximum number of backup files to retain
	maxAge     int    // Maximum age in days for log files
	mu         sync.Mutex
}

// NewRotator creates a new Rotator with the specified configuration.
// maxSize is the maximum file size (e.g., "10MB", "1GB").
// maxBackups is the number of backup files to retain.
// maxAge is the maximum age in days for log files.
func NewRotator(maxSize string, maxBackups, maxAge int) (*Rotator, error) {
	sizeBytes, err := parseSize(maxSize)
	if err != nil {
		return nil, fmt.Errorf("invalid max size: %w", err)
	}

	return &Rotator{
		maxSize:    sizeBytes,
		maxBackups: maxBackups,
		maxAge:     maxAge,
	}, nil
}

// parseSize converts a size string (e.g., "10MB", "1GB") to bytes.
func parseSize(size string) (int64, error) {
	size = strings.TrimSpace(strings.ToUpper(size))
	if size == "" {
		return 0, fmt.Errorf("size cannot be empty")
	}

	// Extract numeric part and unit
	var numStr string
	var unit string
	for i, c := range size {
		if c >= '0' && c <= '9' || c == '.' {
			numStr += string(c)
		} else {
			unit = size[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("no numeric value found in size: %s", size)
	}

	value, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %s", numStr)
	}

	// Convert to bytes based on unit
	multiplier := float64(1)
	switch strings.TrimSpace(unit) {
	case "", "B", "BYTES":
		multiplier = 1
	case "KB", "K":
		multiplier = 1024
	case "MB", "M":
		multiplier = 1024 * 1024
	case "GB", "G":
		multiplier = 1024 * 1024 * 1024
	case "TB", "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown size unit: %s", unit)
	}

	return int64(value * multiplier), nil
}

// ShouldRotate checks if the log file should be rotated based on size.
// Returns true if the file exists and its size exceeds maxSize.
func (r *Rotator) ShouldRotate(logFile string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, err := os.Stat(logFile)
	if err != nil {
		// File doesn't exist or can't be accessed, no need to rotate
		return false
	}

	return info.Size() > r.maxSize
}

// Rotate performs log file rotation.
// The current log file is moved to logFile.1, and existing backups are shifted.
// Files at index 2 and higher are gzip compressed.
func (r *Rotator) Rotate(logFile string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		// No file to rotate
		return nil
	}

	// Perform rotation: shift existing backup files
	if err := r.shiftFiles(logFile); err != nil {
		return fmt.Errorf("failed to shift files: %w", err)
	}

	// Move current log file to .1
	backupFile := logFile + ".1"
	if err := os.Rename(logFile, backupFile); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Clean up old files
	if err := r.cleanup(logFile); err != nil {
		return fmt.Errorf("failed to cleanup old files: %w", err)
	}

	return nil
}

// shiftFiles shifts existing backup files to make room for the new backup.
// logFile.1 -> logFile.2.gz, logFile.2.gz -> logFile.3.gz, etc.
func (r *Rotator) shiftFiles(logFile string) error {
	dir := filepath.Dir(logFile)
	base := filepath.Base(logFile)

	// Start from the highest index and work backwards to avoid overwriting
	for i := r.maxBackups - 1; i >= 1; i-- {
		var currentPath string
		if i == 1 {
			currentPath = filepath.Join(dir, base+"."+strconv.Itoa(i))
		} else {
			currentPath = filepath.Join(dir, base+"."+strconv.Itoa(i)+".gz")
		}

		nextPath := filepath.Join(dir, base+"."+strconv.Itoa(i+1)+".gz")

		// Check if current file exists
		if _, err := os.Stat(currentPath); os.IsNotExist(err) {
			continue
		}

		// If this is the last backup slot, just delete the file
		if i >= r.maxBackups {
			os.Remove(currentPath)
			continue
		}

		// Remove the destination if it exists
		os.Remove(nextPath)

		if i == 1 {
			// Compress logFile.1 to logFile.2.gz
			if err := r.compressFile(currentPath, nextPath); err != nil {
				return fmt.Errorf("failed to compress %s: %w", currentPath, err)
			}
			os.Remove(currentPath)
		} else {
			// Just rename .gz files
			if err := os.Rename(currentPath, nextPath); err != nil {
				return fmt.Errorf("failed to rename %s to %s: %w", currentPath, nextPath, err)
			}
		}
	}

	return nil
}

// compressFile compresses a file using gzip.
func (r *Rotator) compressFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	gzipWriter := gzip.NewWriter(dstFile)
	defer gzipWriter.Close()

	if _, err := io.Copy(gzipWriter, srcFile); err != nil {
		return fmt.Errorf("failed to compress file: %w", err)
	}

	return nil
}

// cleanup removes old log files that exceed the retention policy.
func (r *Rotator) cleanup(logFile string) error {
	// For now, we handle cleanup through maxBackups during rotation
	// Additional age-based cleanup could be implemented here
	return nil
}

// GetMaxSize returns the maximum file size in bytes.
func (r *Rotator) GetMaxSize() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.maxSize
}

// GetMaxBackups returns the maximum number of backups.
func (r *Rotator) GetMaxBackups() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.maxBackups
}

// GetMaxAge returns the maximum age in days.
func (r *Rotator) GetMaxAge() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.maxAge
}
