// Package logging provides a structured logging interface for Morty.
// It wraps Go's standard slog package to provide a unified logging API
// with support for structured attributes, context, and job tracking.
package logging

import (
	"context"
)

// Level represents the logging level.
type Level int

const (
	// DebugLevel is the debug logging level.
	DebugLevel Level = iota
	// InfoLevel is the info logging level.
	InfoLevel
	// WarnLevel is the warn logging level.
	WarnLevel
	// ErrorLevel is the error logging level.
	ErrorLevel
)

// String returns the string representation of the log level.
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a log level string into a Level.
func ParseLevel(s string) Level {
	switch s {
	case "debug", "DEBUG":
		return DebugLevel
	case "info", "INFO":
		return InfoLevel
	case "warn", "WARN", "warning", "WARNING":
		return WarnLevel
	case "error", "ERROR":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// Attr represents a structured logging attribute.
type Attr struct {
	Key   string
	Value interface{}
}

// String returns a string Attr.
func String(key, value string) Attr {
	return Attr{Key: key, Value: value}
}

// Int returns an int Attr.
func Int(key string, value int) Attr {
	return Attr{Key: key, Value: value}
}

// Bool returns a bool Attr.
func Bool(key string, value bool) Attr {
	return Attr{Key: key, Value: value}
}

// Any returns an Attr with any value.
func Any(key string, value interface{}) Attr {
	return Attr{Key: key, Value: value}
}

// Logger is the interface for structured logging.
// It provides methods for different log levels and supports
// structured attributes and context.
type Logger interface {
	// Debug logs a debug message.
	Debug(msg string, attrs ...Attr)

	// Info logs an info message.
	Info(msg string, attrs ...Attr)

	// Warn logs a warning message.
	Warn(msg string, attrs ...Attr)

	// Error logs an error message.
	Error(msg string, attrs ...Attr)

	// Success logs a success message.
	Success(msg string, attrs ...Attr)

	// Loop logs a loop iteration message.
	Loop(msg string, attrs ...Attr)

	// WithContext returns a new Logger with context information.
	WithContext(ctx context.Context) Logger

	// WithJob returns a new Logger with job information.
	WithJob(module, job string) Logger

	// WithAttrs returns a new Logger with additional attributes.
	WithAttrs(attrs ...Attr) Logger

	// SetLevel sets the logging level.
	SetLevel(level Level)

	// GetLevel returns the current logging level.
	GetLevel() Level

	// IsEnabled returns true if the given level is enabled.
	IsEnabled(level Level) bool
}

// contextKey is the type for context keys.
type contextKey string

// context keys for storing logger context.
const (
	contextKeyModule contextKey = "module"
	contextKeyJob    contextKey = "job"
	contextKeyLoop   contextKey = "loop"
)

// ContextWithModule returns a context with the module name.
func ContextWithModule(ctx context.Context, module string) context.Context {
	return context.WithValue(ctx, contextKeyModule, module)
}

// ContextWithJob returns a context with the job name.
func ContextWithJob(ctx context.Context, job string) context.Context {
	return context.WithValue(ctx, contextKeyJob, job)
}

// ContextWithLoop returns a context with the loop count.
func ContextWithLoop(ctx context.Context, loop int) context.Context {
	return context.WithValue(ctx, contextKeyLoop, loop)
}

// ModuleFromContext returns the module from the context.
func ModuleFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyModule).(string); ok {
		return v
	}
	return ""
}

// JobFromContext returns the job from the context.
func JobFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyJob).(string); ok {
		return v
	}
	return ""
}

// LoopFromContext returns the loop count from the context.
func LoopFromContext(ctx context.Context) int {
	if v, ok := ctx.Value(contextKeyLoop).(int); ok {
		return v
	}
	return 0
}
