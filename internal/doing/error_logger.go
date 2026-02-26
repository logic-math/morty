// Package doing provides job execution functionality with error handling and retry mechanisms.
package doing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/morty/morty/internal/logging"
)

// ErrorLogEntry represents a single error log entry.
type ErrorLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Category    string                 `json:"category"`
	Message     string                 `json:"message"`
	Details     string                 `json:"details,omitempty"`
	Module      string                 `json:"module,omitempty"`
	Job         string                 `json:"job,omitempty"`
	LoopCount   int                    `json:"loop_count,omitempty"`
	RetryCount  int                    `json:"retry_count,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
}

// ErrorLogger handles structured error logging.
// Task 5: Implement error logging
type ErrorLogger struct {
	logger   logging.Logger
	logDir   string
	logFile  string
	entries  []ErrorLogEntry
	maxEntries int
}

// NewErrorLogger creates a new error logger.
func NewErrorLogger(logger logging.Logger, logDir string) *ErrorLogger {
	if logDir == "" {
		logDir = ".morty/logs"
	}

	return &ErrorLogger{
		logger:     logger,
		logDir:     logDir,
		logFile:    filepath.Join(logDir, "errors.json"),
		entries:    make([]ErrorLogEntry, 0),
		maxEntries: 1000,
	}
}

// LogError logs an error with context.
func (el *ErrorLogger) LogError(err error, module, job string, loopCount, retryCount int) {
	if err == nil {
		return
	}

	// Classify the error
	doingErr := ClassifyError(err)

	entry := ErrorLogEntry{
		Timestamp:  time.Now(),
		Level:      "ERROR",
		Category:   doingErr.Category.String(),
		Message:    doingErr.Message,
		Details:    err.Error(),
		Module:     module,
		Job:        job,
		LoopCount:  loopCount,
		RetryCount: retryCount,
		Context:    doingErr.Context,
	}

	el.entries = append(el.entries, entry)

	// Trim if exceeds max
	if len(el.entries) > el.maxEntries {
		el.entries = el.entries[len(el.entries)-el.maxEntries:]
	}

	// Log to the logger
	el.logger.Error("Job execution error",
		logging.String("category", entry.Category),
		logging.String("message", entry.Message),
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("loop", loopCount),
		logging.Int("retry", retryCount),
	)

	// Persist to file
	el.persist()
}

// LogWarning logs a warning message.
func (el *ErrorLogger) LogWarning(message, module, job string, context map[string]interface{}) {
	entry := ErrorLogEntry{
		Timestamp: time.Now(),
		Level:     "WARNING",
		Category:  "Warning",
		Message:   message,
		Module:    module,
		Job:       job,
		Context:   context,
	}

	el.entries = append(el.entries, entry)

	if len(el.entries) > el.maxEntries {
		el.entries = el.entries[len(el.entries)-el.maxEntries:]
	}

	el.logger.Warn(message,
		logging.String("module", module),
		logging.String("job", job),
	)

	el.persist()
}

// LogRetry logs a retry attempt.
func (el *ErrorLogger) LogRetry(module, job string, attempt int, maxRetries int, err error) {
	entry := ErrorLogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Category:   "Retry",
		Message:    fmt.Sprintf("Retrying job execution (%d/%d)", attempt, maxRetries),
		Module:     module,
		Job:        job,
		RetryCount: attempt,
		Context: map[string]interface{}{
			"max_retries": maxRetries,
			"error":       err.Error(),
		},
	}

	el.entries = append(el.entries, entry)

	el.logger.Info("Retrying job execution",
		logging.String("module", module),
		logging.String("job", job),
		logging.Int("attempt", attempt),
		logging.Int("max_retries", maxRetries),
	)

	el.persist()
}

// persist saves the error entries to the log file.
func (el *ErrorLogger) persist() error {
	// Ensure log directory exists
	if err := os.MkdirAll(el.logDir, 0755); err != nil {
		el.logger.Error("Failed to create log directory", logging.String("error", err.Error()))
		return err
	}

	// Write to file
	data, err := json.MarshalIndent(el.entries, "", "  ")
	if err != nil {
		el.logger.Error("Failed to marshal error log", logging.String("error", err.Error()))
		return err
	}

	if err := os.WriteFile(el.logFile, data, 0644); err != nil {
		el.logger.Error("Failed to write error log", logging.String("error", err.Error()))
		return err
	}

	return nil
}

// GetRecentErrors returns recent error entries.
func (el *ErrorLogger) GetRecentErrors(count int) []ErrorLogEntry {
	if count <= 0 || count > len(el.entries) {
		count = len(el.entries)
	}

	start := len(el.entries) - count
	if start < 0 {
		start = 0
	}

	result := make([]ErrorLogEntry, count)
	copy(result, el.entries[start:])
	return result
}

// GetErrorsByModule returns errors filtered by module.
func (el *ErrorLogger) GetErrorsByModule(module string) []ErrorLogEntry {
	var result []ErrorLogEntry
	for _, entry := range el.entries {
		if entry.Module == module {
			result = append(result, entry)
		}
	}
	return result
}

// GetErrorsByJob returns errors filtered by job.
func (el *ErrorLogger) GetErrorsByJob(module, job string) []ErrorLogEntry {
	var result []ErrorLogEntry
	for _, entry := range el.entries {
		if entry.Module == module && entry.Job == job {
			result = append(result, entry)
		}
	}
	return result
}

// LoadErrorLog loads the error log from file.
func (el *ErrorLogger) LoadErrorLog() error {
	data, err := os.ReadFile(el.logFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No existing log, that's ok
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &el.entries)
}

// Clear clears all error entries.
func (el *ErrorLogger) Clear() {
	el.entries = make([]ErrorLogEntry, 0)
	el.persist()
}

// FormatErrorReport formats an error report for display.
func (el *ErrorLogger) FormatErrorReport(module, job string) string {
	errors := el.GetErrorsByJob(module, job)
	if len(errors) == 0 {
		return "No errors recorded for this job."
	}

	report := fmt.Sprintf("\n📋 Error Report for %s/%s\n", module, job)
	report += "═══════════════════════════════════════\n"

	for i, entry := range errors {
		report += fmt.Sprintf("\n[%d] %s | %s\n", i+1, entry.Timestamp.Format("15:04:05"), entry.Category)
		report += fmt.Sprintf("    Level: %s\n", entry.Level)
		report += fmt.Sprintf("    Message: %s\n", entry.Message)
		if entry.Details != "" {
			report += fmt.Sprintf("    Details: %s\n", entry.Details)
		}
		if entry.RetryCount > 0 {
			report += fmt.Sprintf("    Retry: %d\n", entry.RetryCount)
		}
	}

	return report
}
