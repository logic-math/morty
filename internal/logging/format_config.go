// Package logging provides configuration integration for log formatting.
package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/morty/morty/internal/config"
)

// FormatConfigFromLoggingConfig creates a FormatConfig from config.LoggingConfig.
func FormatConfigFromLoggingConfig(cfg *config.LoggingConfig) *FormatConfig {
	env := DetectEnvironment()

	format := FormatFromString(cfg.Format)
	if format == "" {
		format = env.DefaultFormat()
	}

	output := OutputTargetFromString(cfg.Output)
	if output == "" {
		output = OutputStdout
	}

	level := ParseLevel(cfg.Level)

	return &FormatConfig{
		Format:       format,
		Output:       output,
		Level:        level,
		Environment:  env,
		TimeFormat:   "2006-01-02T15:04:05.000Z07:00",
		EnableColors: env == EnvDevelopment && output == OutputStdout,
		EnableSource: env == EnvDevelopment,
	}
}

// NewLoggerFromConfig creates a Logger from LoggingConfig.
// This supports multiple output targets (console + file) and respects configuration settings.
func NewLoggerFromConfig(cfg *config.LoggingConfig) (Logger, io.Closer, error) {
	formatConfig := FormatConfigFromLoggingConfig(cfg)

	var formatter Formatter
	switch formatConfig.Format {
	case FormatText:
		formatter = NewTextFormatter(formatConfig.EnableColors, formatConfig.TimeFormat)
	case FormatJSON:
		fallthrough
	default:
		formatter = NewJSONFormatter()
	}

	var writers []io.Writer
	var closers []io.Closer

	// Add stdout writer if needed
	if formatConfig.Output == OutputStdout || formatConfig.Output == OutputBoth {
		writers = append(writers, os.Stdout)
	}

	// Add file writer if needed
	if (formatConfig.Output == OutputFile || formatConfig.Output == OutputBoth) && cfg.File.Enabled {
		fileWriter, err := createFileWriter(cfg.File.Path)
		if err != nil {
			// If file creation fails, fall back to stdout
			fmt.Fprintf(os.Stderr, "Warning: failed to create log file: %v\n", err)
			if formatConfig.Output == OutputFile {
				// If only file output was requested, use stdout as fallback
				writers = append(writers, os.Stdout)
			}
		} else {
			writers = append(writers, fileWriter)
			closers = append(closers, fileWriter)
		}
	}

	// Ensure at least one writer
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	var writer io.Writer
	if len(writers) == 1 {
		writer = writers[0]
	} else {
		writer = NewMultiWriter(writers...)
	}

	logger := NewFormatterLogger(formatter, writer, formatConfig.Level)

	// Create a combined closer
	closer := &multiCloser{closers: closers}

	return logger, closer, nil
}

// multiCloser closes multiple io.Closer instances.
type multiCloser struct {
	closers []io.Closer
}

// Close implements io.Closer.
func (m *multiCloser) Close() error {
	var lastErr error
	for _, closer := range m.closers {
		if err := closer.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// createFileWriter creates a file writer with proper directory creation.
func createFileWriter(path string) (io.WriteCloser, error) {
	// Expand home directory if needed
	if path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
	}

	// Open file for append
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return file, nil
}

// AutoSelectFormat returns the appropriate format based on environment and config.
// If configFormat is not empty and valid, it uses that. Otherwise, it auto-detects.
func AutoSelectFormat(configFormat string) Format {
	if f := FormatFromString(configFormat); f.IsValid() {
		return f
	}
	return DetectEnvironment().DefaultFormat()
}

// AutoSelectOutput returns the appropriate output target based on config.
// If configOutput is not empty and valid, it uses that. Otherwise, defaults to stdout.
func AutoSelectOutput(configOutput string, fileEnabled bool) OutputTarget {
	if o := OutputTargetFromString(configOutput); o.IsValid() {
		// If file output is requested but file is disabled, use stdout
		if (o == OutputFile || o == OutputBoth) && !fileEnabled {
			return OutputStdout
		}
		return o
	}
	// Default to both if file is enabled, otherwise stdout
	if fileEnabled {
		return OutputBoth
	}
	return OutputStdout
}

// EnvironmentAwareLogger creates a logger that automatically configures itself
// based on the current environment and provided configuration.
func EnvironmentAwareLogger(cfg *config.LoggingConfig) (Logger, io.Closer, error) {
	return NewLoggerFromConfig(cfg)
}

// GetEnvironmentInfo returns information about the current environment.
func GetEnvironmentInfo() map[string]interface{} {
	env := DetectEnvironment()
	return map[string]interface{}{
		"environment":   env.String(),
		"defaultFormat": env.DefaultFormat().String(),
		"defaultLevel":  env.DefaultLevel().String(),
		"isDevelopment": env == EnvDevelopment,
		"isProduction":  env == EnvProduction,
		"isTesting":     env == EnvTesting,
	}
}
