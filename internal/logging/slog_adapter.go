// Package logging provides a structured logging interface for Morty.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// SlogAdapter is an adapter that implements the Logger interface
// using Go's standard slog package.
type SlogAdapter struct {
	logger    *slog.Logger
	handler   slog.Handler
	level     *slog.LevelVar
	attrs     []Attr
	module    string
	job       string
	mu        sync.RWMutex
	output    string // "stdout", "file", or "both"
	fileWriter io.WriteCloser
}

// NewSlogAdapter creates a new SlogAdapter with the specified configuration.
func NewSlogAdapter(format, output string, level Level) (*SlogAdapter, error) {
	levelVar := &slog.LevelVar{}
	levelVar.Set(slogLevel(level))

	adapter := &SlogAdapter{
		level:  levelVar,
		output: output,
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: levelVar,
	}

	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	adapter.handler = handler
	adapter.logger = slog.New(handler)

	return adapter, nil
}

// NewSlogAdapterWithConfig creates a new SlogAdapter using config.LoggingConfig.
func NewSlogAdapterWithConfig(format, output, filePath string, level Level, fileEnabled bool) (*SlogAdapter, error) {
	levelVar := &slog.LevelVar{}
	levelVar.Set(slogLevel(level))

	adapter := &SlogAdapter{
		level:  levelVar,
		output: output,
	}

	// Handle output destinations
	var writers []io.Writer

	if output == "stdout" || output == "both" {
		writers = append(writers, os.Stdout)
	}

	if (output == "file" || output == "both") && fileEnabled && filePath != "" {
		// Ensure log directory exists
		dir := filePath[:len(filePath)-len(getFileName(filePath))]
		if dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create log directory: %w", err)
			}
		}

		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		adapter.fileWriter = file
		writers = append(writers, file)
	}

	var outputWriter io.Writer
	if len(writers) == 1 {
		outputWriter = writers[0]
	} else if len(writers) > 1 {
		outputWriter = io.MultiWriter(writers...)
	} else {
		outputWriter = os.Stdout
	}

	// Create handler based on format
	opts := &slog.HandlerOptions{
		Level: levelVar,
	}

	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(outputWriter, opts)
	case "text":
		handler = slog.NewTextHandler(outputWriter, opts)
	default:
		handler = slog.NewJSONHandler(outputWriter, opts)
	}

	adapter.handler = handler
	adapter.logger = slog.New(handler)

	return adapter, nil
}

// getFileName extracts the file name from a path.
func getFileName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

// slogLevel converts our Level to slog.Level.
func slogLevel(level Level) slog.Level {
	switch level {
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// toSlogAttrs converts our Attr slice to slog.Attr slice.
func toSlogAttrs(attrs []Attr) []slog.Attr {
	result := make([]slog.Attr, len(attrs))
	for i, attr := range attrs {
		result[i] = slog.Any(attr.Key, attr.Value)
	}
	return result
}

// Debug logs a debug message.
func (l *SlogAdapter) Debug(msg string, attrs ...Attr) {
	if !l.IsEnabled(DebugLevel) {
		return
	}
	allAttrs := append(l.getBaseAttrs(), attrs...)
	l.logger.LogAttrs(context.Background(), slog.LevelDebug, msg, toSlogAttrs(allAttrs)...)
}

// Info logs an info message.
func (l *SlogAdapter) Info(msg string, attrs ...Attr) {
	if !l.IsEnabled(InfoLevel) {
		return
	}
	allAttrs := append(l.getBaseAttrs(), attrs...)
	l.logger.LogAttrs(context.Background(), slog.LevelInfo, msg, toSlogAttrs(allAttrs)...)
}

// Warn logs a warning message.
func (l *SlogAdapter) Warn(msg string, attrs ...Attr) {
	if !l.IsEnabled(WarnLevel) {
		return
	}
	allAttrs := append(l.getBaseAttrs(), attrs...)
	l.logger.LogAttrs(context.Background(), slog.LevelWarn, msg, toSlogAttrs(allAttrs)...)
}

// Error logs an error message.
func (l *SlogAdapter) Error(msg string, attrs ...Attr) {
	if !l.IsEnabled(ErrorLevel) {
		return
	}
	allAttrs := append(l.getBaseAttrs(), attrs...)
	l.logger.LogAttrs(context.Background(), slog.LevelError, msg, toSlogAttrs(allAttrs)...)
}

// Success logs a success message (mapped to Info with success marker).
func (l *SlogAdapter) Success(msg string, attrs ...Attr) {
	if !l.IsEnabled(InfoLevel) {
		return
	}
	allAttrs := append(l.getBaseAttrs(), Attr{Key: "level", Value: "SUCCESS"})
	allAttrs = append(allAttrs, attrs...)
	l.logger.LogAttrs(context.Background(), slog.LevelInfo, msg, toSlogAttrs(allAttrs)...)
}

// Loop logs a loop iteration message (mapped to Debug with loop context).
func (l *SlogAdapter) Loop(msg string, attrs ...Attr) {
	if !l.IsEnabled(DebugLevel) {
		return
	}
	allAttrs := append(l.getBaseAttrs(), Attr{Key: "level", Value: "LOOP"})
	allAttrs = append(allAttrs, attrs...)
	l.logger.LogAttrs(context.Background(), slog.LevelDebug, msg, toSlogAttrs(allAttrs)...)
}

// getBaseAttrs returns the base attributes including module and job.
func (l *SlogAdapter) getBaseAttrs() []Attr {
	l.mu.RLock()
	defer l.mu.RUnlock()

	baseAttrs := make([]Attr, 0, len(l.attrs)+2)
	baseAttrs = append(baseAttrs, l.attrs...)

	if l.module != "" {
		baseAttrs = append(baseAttrs, String("module", l.module))
	}
	if l.job != "" {
		baseAttrs = append(baseAttrs, String("job", l.job))
	}

	return baseAttrs
}

// WithContext returns a new Logger with context information.
func (l *SlogAdapter) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	newAdapter := l.clone()

	if module := ModuleFromContext(ctx); module != "" {
		newAdapter.module = module
	}
	if job := JobFromContext(ctx); job != "" {
		newAdapter.job = job
	}
	if loop := LoopFromContext(ctx); loop > 0 {
		newAdapter.attrs = append(newAdapter.attrs, Int("loop", loop))
	}

	return newAdapter
}

// WithJob returns a new Logger with job information.
func (l *SlogAdapter) WithJob(module, job string) Logger {
	newAdapter := l.clone()
	newAdapter.module = module
	newAdapter.job = job
	return newAdapter
}

// WithAttrs returns a new Logger with additional attributes.
func (l *SlogAdapter) WithAttrs(attrs ...Attr) Logger {
	newAdapter := l.clone()
	newAdapter.attrs = append(newAdapter.attrs, attrs...)
	return newAdapter
}

// clone creates a copy of the adapter.
func (l *SlogAdapter) clone() *SlogAdapter {
	l.mu.RLock()
	defer l.mu.RUnlock()

	attrsCopy := make([]Attr, len(l.attrs))
	copy(attrsCopy, l.attrs)

	return &SlogAdapter{
		logger:     l.logger,
		handler:    l.handler,
		level:      l.level,
		attrs:      attrsCopy,
		module:     l.module,
		job:        l.job,
		output:     l.output,
		fileWriter: l.fileWriter,
	}
}

// SetLevel sets the logging level.
func (l *SlogAdapter) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level.Set(slogLevel(level))
}

// GetLevel returns the current logging level.
func (l *SlogAdapter) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Convert slog.Level back to our Level
	sl := l.level.Level()
	switch {
	case sl <= slog.LevelDebug:
		return DebugLevel
	case sl <= slog.LevelInfo:
		return InfoLevel
	case sl <= slog.LevelWarn:
		return WarnLevel
	case sl <= slog.LevelError:
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// IsEnabled returns true if the given level is enabled.
func (l *SlogAdapter) IsEnabled(level Level) bool {
	return slogLevel(level) >= l.level.Level()
}

// Close closes the logger and any open file writers.
func (l *SlogAdapter) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.fileWriter != nil {
		return l.fileWriter.Close()
	}
	return nil
}
