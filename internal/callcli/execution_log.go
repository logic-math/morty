// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ExecutionLog represents a single execution record.
// It contains all information about a command execution including
// command details, execution time, exit code, and output size.
type ExecutionLog struct {
	// ID is a unique identifier for this execution
	ID string `json:"id"`
	// Timestamp is when the execution started
	Timestamp time.Time `json:"timestamp"`
	// Command is the executed command name
	Command string `json:"command"`
	// Args contains all command arguments
	Args []string `json:"args"`
	// FullCommand is the complete command string with arguments
	FullCommand string `json:"full_command"`
	// WorkingDir is the working directory for execution
	WorkingDir string `json:"working_dir,omitempty"`
	// Env contains relevant environment variables
	Env map[string]string `json:"env,omitempty"`
	// Duration is the execution time
	Duration time.Duration `json:"duration"`
	// DurationMs is the execution time in milliseconds
	DurationMs int64 `json:"duration_ms"`
	// ExitCode is the process exit code
	ExitCode int `json:"exit_code"`
	// Success indicates if the execution succeeded (exit code 0)
	Success bool `json:"success"`
	// TimedOut indicates if the execution timed out
	TimedOut bool `json:"timed_out"`
	// Interrupted indicates if the execution was interrupted by a signal
	Interrupted bool `json:"interrupted"`
	// StdoutSize is the size of stdout output in bytes
	StdoutSize int64 `json:"stdout_size"`
	// StderrSize is the size of stderr output in bytes
	StderrSize int64 `json:"stderr_size"`
	// TotalOutputSize is the total output size (stdout + stderr)
	TotalOutputSize int64 `json:"total_output_size"`
	// Error message if execution failed
	Error string `json:"error,omitempty"`
	// Timeout is the configured timeout duration
	Timeout time.Duration `json:"timeout,omitempty"`
}

// ExecutionLogger provides logging for command executions with
// support for log rotation and execution statistics.
type ExecutionLogger struct {
	mu sync.RWMutex

	// logDir is the directory for log files
	logDir string
	// currentFile is the current log file path
	currentFile string
	// file is the current log file handle
	file *os.File

	// maxSize is the maximum size of a log file before rotation (bytes)
	maxSize int64
	// maxBackups is the maximum number of backup files to keep
	maxBackups int
	// maxAge is the maximum age of log files before cleanup (days)
	maxAge int

	// stats tracks execution statistics
	stats *ExecutionStats

	// closed indicates if the logger is closed
	closed bool
}

// ExecutionStats tracks aggregate statistics for command executions.
type ExecutionStats struct {
	mu sync.RWMutex

	// TotalExecutions is the total number of executions
	TotalExecutions int64 `json:"total_executions"`
	// SuccessfulExecutions is the number of successful executions
	SuccessfulExecutions int64 `json:"successful_executions"`
	// FailedExecutions is the number of failed executions
	FailedExecutions int64 `json:"failed_executions"`
	// TimeoutExecutions is the number of timed out executions
	TimeoutExecutions int64 `json:"timeout_executions"`
	// InterruptedExecutions is the number of interrupted executions
	InterruptedExecutions int64 `json:"interrupted_executions"`

	// TotalDuration is the sum of all execution durations
	TotalDuration time.Duration `json:"total_duration"`
	// AverageDuration is the average execution duration
	AverageDuration time.Duration `json:"average_duration"`
	// MinDuration is the minimum execution duration
	MinDuration time.Duration `json:"min_duration"`
	// MaxDuration is the maximum execution duration
	MaxDuration time.Duration `json:"max_duration"`

	// CommandStats tracks stats per command
	CommandStats map[string]*CommandStats `json:"command_stats"`

	// LastExecution is the timestamp of the last execution
	LastExecution time.Time `json:"last_execution"`
}

// CommandStats tracks statistics for a specific command.
type CommandStats struct {
	// Command name
	Command string `json:"command"`
	// TotalExecutions is the total number of executions for this command
	TotalExecutions int64 `json:"total_executions"`
	// SuccessfulExecutions is the number of successful executions
	SuccessfulExecutions int64 `json:"successful_executions"`
	// FailedExecutions is the number of failed executions
	FailedExecutions int64 `json:"failed_executions"`
	// TotalDuration is the sum of all execution durations
	TotalDuration time.Duration `json:"total_duration"`
	// AverageDuration is the average execution duration
	AverageDuration time.Duration `json:"average_duration"`
}

// NewExecutionLogger creates a new ExecutionLogger.
//
// logDir: the directory for log files
// maxSize: maximum size of a log file before rotation (0 = no rotation)
// maxBackups: maximum number of backup files to keep (0 = unlimited)
// maxAge: maximum age of log files in days (0 = no cleanup)
func NewExecutionLogger(logDir string, maxSize int64, maxBackups, maxAge int) (*ExecutionLogger, error) {
	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logger := &ExecutionLogger{
		logDir:     logDir,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAge:     maxAge,
		stats: &ExecutionStats{
			CommandStats: make(map[string]*CommandStats),
			MinDuration:  -1, // Indicates not set
		},
	}

	// Open initial log file
	if err := logger.openNewFile(); err != nil {
		return nil, err
	}

	return logger, nil
}

// openNewFile opens a new log file with timestamp in name.
func (el *ExecutionLogger) openNewFile() error {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("execution_%s.log", timestamp)
	filepath := filepath.Join(el.logDir, filename)

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	el.currentFile = filepath
	el.file = file
	return nil
}

// rotate rotates the log file if it exceeds maxSize.
func (el *ExecutionLogger) rotate() error {
	if el.maxSize <= 0 {
		return nil // Rotation disabled
	}

	// Get current file size
	info, err := el.file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	if info.Size() < el.maxSize {
		return nil // No rotation needed
	}

	// Close current file
	if err := el.file.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	// Clean up old files
	if err := el.cleanup(); err != nil {
		// Log error but continue
		fmt.Fprintf(os.Stderr, "Failed to cleanup old logs: %v\n", err)
	}

	// Open new file
	return el.openNewFile()
}

// cleanup removes old log files based on maxBackups and maxAge.
func (el *ExecutionLogger) cleanup() error {
	files, err := filepath.Glob(filepath.Join(el.logDir, "execution_*.log"))
	if err != nil {
		return err
	}

	// Remove files older than maxAge
	if el.maxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -el.maxAge)
		for _, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(file)
			}
		}

		// Refresh file list
		files, err = filepath.Glob(filepath.Join(el.logDir, "execution_*.log"))
		if err != nil {
			return err
		}
	}

	// Remove excess backups (keep most recent)
	if el.maxBackups > 0 && len(files) > el.maxBackups {
		// Sort by modification time (most recent first)
		type fileInfo struct {
			path    string
			modTime time.Time
		}
		fileInfos := make([]fileInfo, len(files))
		for i, file := range files {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			fileInfos[i] = fileInfo{path: file, modTime: info.ModTime()}
		}

		// Simple bubble sort by modification time (newest first)
		for i := 0; i < len(fileInfos); i++ {
			for j := i + 1; j < len(fileInfos); j++ {
				if fileInfos[j].modTime.After(fileInfos[i].modTime) {
					fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
				}
			}
		}

		// Remove excess files
		for i := el.maxBackups; i < len(fileInfos); i++ {
			os.Remove(fileInfos[i].path)
		}
	}

	return nil
}

// LogExecution logs a single execution record.
// This method is thread-safe.
func (el *ExecutionLogger) LogExecution(log *ExecutionLog) error {
	el.mu.Lock()
	defer el.mu.Unlock()

	if el.closed {
		return fmt.Errorf("logger is closed")
	}

	// Rotate if needed
	if err := el.rotate(); err != nil {
		return fmt.Errorf("failed to rotate log: %w", err)
	}

	// Write log entry as JSON
	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	if _, err := el.file.Write(data); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	if _, err := el.file.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	// Update statistics
	el.updateStats(log)

	return nil
}

// updateStats updates the execution statistics.
// Must be called with el.mu held.
func (el *ExecutionLogger) updateStats(log *ExecutionLog) {
	el.stats.mu.Lock()
	defer el.stats.mu.Unlock()

	el.stats.TotalExecutions++
	el.stats.LastExecution = log.Timestamp

	if log.Success {
		el.stats.SuccessfulExecutions++
	} else {
		el.stats.FailedExecutions++
	}

	if log.TimedOut {
		el.stats.TimeoutExecutions++
	}

	if log.Interrupted {
		el.stats.InterruptedExecutions++
	}

	el.stats.TotalDuration += log.Duration
	el.stats.AverageDuration = el.stats.TotalDuration / time.Duration(el.stats.TotalExecutions)

	if el.stats.MinDuration < 0 || log.Duration < el.stats.MinDuration {
		el.stats.MinDuration = log.Duration
	}

	if log.Duration > el.stats.MaxDuration {
		el.stats.MaxDuration = log.Duration
	}

	// Update per-command stats
	cmdStats, exists := el.stats.CommandStats[log.Command]
	if !exists {
		cmdStats = &CommandStats{
			Command: log.Command,
		}
		el.stats.CommandStats[log.Command] = cmdStats
	}

	cmdStats.TotalExecutions++
	if log.Success {
		cmdStats.SuccessfulExecutions++
	} else {
		cmdStats.FailedExecutions++
	}
	cmdStats.TotalDuration += log.Duration
	cmdStats.AverageDuration = cmdStats.TotalDuration / time.Duration(cmdStats.TotalExecutions)
}

// GetStats returns a copy of the current execution statistics.
func (el *ExecutionLogger) GetStats() ExecutionStats {
	el.stats.mu.RLock()
	defer el.stats.mu.RUnlock()

	// Deep copy
	statsCopy := *el.stats
	statsCopy.CommandStats = make(map[string]*CommandStats)
	for k, v := range el.stats.CommandStats {
		cmdCopy := *v
		statsCopy.CommandStats[k] = &cmdCopy
	}

	return statsCopy
}

// GetSuccessRate returns the success rate as a percentage (0-100).
func (es *ExecutionStats) GetSuccessRate() float64 {
	es.mu.RLock()
	defer es.mu.RUnlock()

	if es.TotalExecutions == 0 {
		return 0
	}
	return float64(es.SuccessfulExecutions) * 100.0 / float64(es.TotalExecutions)
}

// GetFailureRate returns the failure rate as a percentage (0-100).
func (es *ExecutionStats) GetFailureRate() float64 {
	es.mu.RLock()
	defer es.mu.RUnlock()

	if es.TotalExecutions == 0 {
		return 0
	}
	return float64(es.FailedExecutions) * 100.0 / float64(es.TotalExecutions)
}

// GetTimeoutRate returns the timeout rate as a percentage (0-100).
func (es *ExecutionStats) GetTimeoutRate() float64 {
	es.mu.RLock()
	defer es.mu.RUnlock()

	if es.TotalExecutions == 0 {
		return 0
	}
	return float64(es.TimeoutExecutions) * 100.0 / float64(es.TotalExecutions)
}

// Close closes the execution logger.
func (el *ExecutionLogger) Close() error {
	el.mu.Lock()
	defer el.mu.Unlock()

	if el.closed {
		return nil
	}

	el.closed = true

	if el.file != nil {
		return el.file.Close()
	}

	return nil
}

// NewExecutionLogFromResult creates an ExecutionLog from a Result.
// This is a convenience function for creating execution logs from command results.
func NewExecutionLogFromResult(result *Result, command string, args []string, workingDir string, timeout time.Duration) *ExecutionLog {
	return &ExecutionLog{
		ID:              generateID(),
		Timestamp:       time.Now(),
		Command:         command,
		Args:            args,
		FullCommand:     result.Command,
		WorkingDir:      workingDir,
		Duration:        result.Duration,
		DurationMs:      result.Duration.Milliseconds(),
		ExitCode:        result.ExitCode,
		Success:         result.ExitCode == 0 && !result.TimedOut && !result.Interrupted,
		TimedOut:        result.TimedOut,
		Interrupted:     result.Interrupted,
		StdoutSize:      int64(len(result.Stdout)),
		StderrSize:      int64(len(result.Stderr)),
		TotalOutputSize: int64(len(result.Stdout) + len(result.Stderr)),
		Timeout:         timeout,
	}
}

// generateID generates a unique ID for an execution log.
func generateID() string {
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), os.Getpid())
}

// ReadLogs reads all execution logs from the specified directory.
// Returns a slice of ExecutionLog entries.
func ReadLogs(logDir string) ([]ExecutionLog, error) {
	files, err := filepath.Glob(filepath.Join(logDir, "execution_*.log"))
	if err != nil {
		return nil, err
	}

	var logs []ExecutionLog
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files we can't read
		}

		lines := splitLines(string(data))
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}

			var log ExecutionLog
			if err := json.Unmarshal([]byte(line), &log); err != nil {
				continue // Skip invalid lines
			}
			logs = append(logs, log)
		}
	}

	return logs, nil
}

// splitLines splits a string into lines.
func splitLines(s string) []string {
	var lines []string
	var start int
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
