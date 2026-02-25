// Package logging provides a structured logging interface for Morty.
package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// JobLogger provides structured logging for job execution with automatic
// start/end logging, task tracking, and consistent context attributes.
type JobLogger struct {
	logger     Logger
	module     string
	job        string
	startTime  time.Time
	taskCount  int32 // atomic counter
	mu         sync.RWMutex
	fileWriter *os.File
}

// NewJobLogger creates a new JobLogger for the specified module and job.
// It automatically logs the job start event.
func NewJobLogger(module, job string, baseLogger Logger) *JobLogger {
	jl := &JobLogger{
		logger: baseLogger.WithJob(module, job),
		module: module,
		job:    job,
	}

	jl.logJobStart()
	return jl
}

// NewJobLoggerWithFile creates a new JobLogger with a separate log file for this job.
// It creates a job-specific log file in the specified directory.
func NewJobLoggerWithFile(module, job string, baseLogger Logger, logDir string) (*JobLogger, error) {
	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create job-specific log file
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s_%s.log", module, job, timestamp)
	filepath := filepath.Join(logDir, filename)

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create job log file: %w", err)
	}

	// Create a new logger that writes to the file
	adapter, err := NewSlogAdapterWithConfig("json", "file", filepath, InfoLevel, true)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create job logger: %w", err)
	}

	jl := &JobLogger{
		logger:     adapter.WithJob(module, job),
		module:     module,
		job:        job,
		fileWriter: file,
	}

	jl.logJobStart()
	return jl, nil
}

// logJobStart logs the job start event with module, job name, and start time.
func (jl *JobLogger) logJobStart() {
	jl.mu.Lock()
	jl.startTime = time.Now()
	jl.mu.Unlock()

	jl.logger.Info("Job started",
		String("event", "job_start"),
		String("module", jl.module),
		String("job", jl.job),
		String("timestamp", jl.startTime.Format(time.RFC3339)),
	)
}

// LogJobEnd logs the job completion event with execution result and duration.
// This should be called when the job completes, typically with defer.
func (jl *JobLogger) LogJobEnd(result string) {
	jl.mu.RLock()
	startTime := jl.startTime
	jl.mu.RUnlock()

	taskCount := atomic.LoadInt32(&jl.taskCount)
	duration := time.Since(startTime)

	jl.logger.Info("Job completed",
		String("event", "job_end"),
		String("module", jl.module),
		String("job", jl.job),
		String("result", result),
		String("duration", duration.String()),
		Int("duration_ms", int(duration.Milliseconds())),
		Int("tasks_completed", int(taskCount)),
		String("timestamp", time.Now().Format(time.RFC3339)),
	)

	// Close file writer if exists
	jl.mu.Lock()
	if jl.fileWriter != nil {
		jl.fileWriter.Close()
		jl.fileWriter = nil
	}
	jl.mu.Unlock()
}

// LogJobEndWithError logs the job completion with an error result.
func (jl *JobLogger) LogJobEndWithError(err error) {
	jl.mu.RLock()
	startTime := jl.startTime
	jl.mu.RUnlock()

	taskCount := atomic.LoadInt32(&jl.taskCount)
	duration := time.Since(startTime)

	jl.logger.Error("Job failed",
		String("event", "job_end"),
		String("module", jl.module),
		String("job", jl.job),
		String("result", "failed"),
		String("error", err.Error()),
		String("duration", duration.String()),
		Int("duration_ms", int(duration.Milliseconds())),
		Int("tasks_completed", int(taskCount)),
		String("timestamp", time.Now().Format(time.RFC3339)),
	)

	// Close file writer if exists
	jl.mu.Lock()
	if jl.fileWriter != nil {
		jl.fileWriter.Close()
		jl.fileWriter = nil
	}
	jl.mu.Unlock()
}

// LogTaskStart logs the start of a task with task number and description.
func (jl *JobLogger) LogTaskStart(taskNum int, taskDesc string) {
	// Atomically increment task count
	atomic.AddInt32(&jl.taskCount, 1)

	jl.logger.Info("Task started",
		String("event", "task_start"),
		String("module", jl.module),
		String("job", jl.job),
		Int("task_num", taskNum),
		String("task_desc", taskDesc),
		String("timestamp", time.Now().Format(time.RFC3339)),
	)
}

// LogTaskEnd logs the completion of a task with result.
func (jl *JobLogger) LogTaskEnd(taskNum int, taskDesc string, result string) {
	jl.logger.Info("Task completed",
		String("event", "task_end"),
		String("module", jl.module),
		String("job", jl.job),
		Int("task_num", taskNum),
		String("task_desc", taskDesc),
		String("result", result),
		String("timestamp", time.Now().Format(time.RFC3339)),
	)
}

// LogTaskEndWithError logs the completion of a task with an error.
func (jl *JobLogger) LogTaskEndWithError(taskNum int, taskDesc string, err error) {
	jl.logger.Error("Task failed",
		String("event", "task_end"),
		String("module", jl.module),
		String("job", jl.job),
		Int("task_num", taskNum),
		String("task_desc", taskDesc),
		String("result", "failed"),
		String("error", err.Error()),
		String("timestamp", time.Now().Format(time.RFC3339)),
	)
}

// Logger returns the underlying Logger interface.
// This allows using JobLogger wherever a Logger is expected.
func (jl *JobLogger) Logger() Logger {
	return jl.logger
}

// GetModule returns the module name.
func (jl *JobLogger) GetModule() string {
	jl.mu.RLock()
	defer jl.mu.RUnlock()
	return jl.module
}

// GetJob returns the job name.
func (jl *JobLogger) GetJob() string {
	jl.mu.RLock()
	defer jl.mu.RUnlock()
	return jl.job
}

// GetTaskCount returns the number of tasks logged.
func (jl *JobLogger) GetTaskCount() int {
	return int(atomic.LoadInt32(&jl.taskCount))
}

// GetStartTime returns the job start time.
func (jl *JobLogger) GetStartTime() time.Time {
	jl.mu.RLock()
	defer jl.mu.RUnlock()
	return jl.startTime
}

// GetDuration returns the elapsed time since job start.
func (jl *JobLogger) GetDuration() time.Duration {
	jl.mu.RLock()
	startTime := jl.startTime
	jl.mu.RUnlock()
	return time.Since(startTime)
}

// Debug logs a debug message with job context.
func (jl *JobLogger) Debug(msg string, attrs ...Attr) {
	jl.logger.Debug(msg, attrs...)
}

// Info logs an info message with job context.
func (jl *JobLogger) Info(msg string, attrs ...Attr) {
	jl.logger.Info(msg, attrs...)
}

// Warn logs a warning message with job context.
func (jl *JobLogger) Warn(msg string, attrs ...Attr) {
	jl.logger.Warn(msg, attrs...)
}

// Error logs an error message with job context.
func (jl *JobLogger) Error(msg string, attrs ...Attr) {
	jl.logger.Error(msg, attrs...)
}

// Success logs a success message with job context.
func (jl *JobLogger) Success(msg string, attrs ...Attr) {
	jl.logger.Success(msg, attrs...)
}

// Loop logs a loop iteration message with job context.
func (jl *JobLogger) Loop(msg string, attrs ...Attr) {
	jl.logger.Loop(msg, attrs...)
}

// WithAttrs returns a new JobLogger with additional attributes.
func (jl *JobLogger) WithAttrs(attrs ...Attr) *JobLogger {
	return &JobLogger{
		logger:    jl.logger.WithAttrs(attrs...),
		module:    jl.module,
		job:       jl.job,
		startTime: jl.startTime,
	}
}

// Close closes the job logger and any open resources.
func (jl *JobLogger) Close() error {
	jl.mu.Lock()
	defer jl.mu.Unlock()
	if jl.fileWriter != nil {
		err := jl.fileWriter.Close()
		jl.fileWriter = nil
		return err
	}
	return nil
}
