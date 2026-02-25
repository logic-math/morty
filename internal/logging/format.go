// Package logging provides structured logging with multiple format support.
// This file defines format types, formatters, and environment-based format selection.
package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Format represents the log output format type.
type Format string

const (
	// FormatJSON outputs logs as structured JSON.
	FormatJSON Format = "json"
	// FormatText outputs logs as human-readable text.
	FormatText Format = "text"
)

// FormatFromString converts a string to Format type.
func FormatFromString(s string) Format {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON
	case "text":
		return FormatText
	default:
		return FormatJSON
	}
}

// String returns the string representation of Format.
func (f Format) String() string {
	return string(f)
}

// IsValid returns true if the format is valid.
func (f Format) IsValid() bool {
	switch f {
	case FormatJSON, FormatText:
		return true
	default:
		return false
	}
}

// OutputTarget represents the log output destination.
type OutputTarget string

const (
	// OutputStdout writes logs to standard output only.
	OutputStdout OutputTarget = "stdout"
	// OutputFile writes logs to file only.
	OutputFile OutputTarget = "file"
	// OutputBoth writes logs to both stdout and file.
	OutputBoth OutputTarget = "both"
)

// OutputTargetFromString converts a string to OutputTarget type.
func OutputTargetFromString(s string) OutputTarget {
	switch strings.ToLower(s) {
	case "stdout":
		return OutputStdout
	case "file":
		return OutputFile
	case "both":
		return OutputBoth
	default:
		return OutputStdout
	}
}

// String returns the string representation of OutputTarget.
func (o OutputTarget) String() string {
	return string(o)
}

// IsValid returns true if the output target is valid.
func (o OutputTarget) IsValid() bool {
	switch o {
	case OutputStdout, OutputFile, OutputBoth:
		return true
	default:
		return false
	}
}

// Environment represents the runtime environment type.
type Environment string

const (
	// EnvDevelopment is the development environment.
	EnvDevelopment Environment = "development"
	// EnvProduction is the production environment.
	EnvProduction Environment = "production"
	// EnvTesting is the testing environment.
	EnvTesting Environment = "testing"
)

// EnvironmentFromString converts a string to Environment type.
func EnvironmentFromString(s string) Environment {
	switch strings.ToLower(s) {
	case "dev", "development":
		return EnvDevelopment
	case "prod", "production":
		return EnvProduction
	case "test", "testing":
		return EnvTesting
	default:
		return EnvDevelopment
	}
}

// String returns the string representation of Environment.
func (e Environment) String() string {
	return string(e)
}

// DefaultFormat returns the default log format for the environment.
func (e Environment) DefaultFormat() Format {
	switch e {
	case EnvProduction:
		return FormatJSON
	case EnvDevelopment, EnvTesting:
		return FormatText
	default:
		return FormatText
	}
}

// DefaultLevel returns the default log level for the environment.
func (e Environment) DefaultLevel() Level {
	switch e {
	case EnvProduction:
		return InfoLevel
	case EnvDevelopment:
		return DebugLevel
	case EnvTesting:
		return WarnLevel
	default:
		return InfoLevel
	}
}

// DetectEnvironment detects the current environment from environment variables.
// It checks MORTY_ENV, NODE_ENV, GO_ENV, and ENV in that order.
func DetectEnvironment() Environment {
	envVars := []string{"MORTY_ENV", "NODE_ENV", "GO_ENV", "ENV"}
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			return EnvironmentFromString(value)
		}
	}
	return EnvDevelopment
}

// FormatConfig contains configuration for log formatting.
type FormatConfig struct {
	Format       Format
	Output       OutputTarget
	Level        Level
	Environment  Environment
	TimeFormat   string
	EnableColors bool
	EnableSource bool
}

// DefaultFormatConfig returns a FormatConfig with default values.
func DefaultFormatConfig() *FormatConfig {
	env := DetectEnvironment()
	return &FormatConfig{
		Format:       env.DefaultFormat(),
		Output:       OutputStdout,
		Level:        env.DefaultLevel(),
		Environment:  env,
		TimeFormat:   time.RFC3339,
		EnableColors: env == EnvDevelopment,
		EnableSource: env == EnvDevelopment,
	}
}

// Formatter is the interface for log formatters.
type Formatter interface {
	// Format formats a log entry and writes it to the writer.
	Format(w io.Writer, entry *LogEntry) error
}

// LogEntry represents a single log entry.
type LogEntry struct {
	Time       time.Time
	Level      Level
	Message    string
	Module     string
	Job        string
	Attributes []Attr
	Source     string // Optional source file:line
}

// ToMap converts the log entry to a map for JSON serialization.
func (e *LogEntry) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	m["time"] = e.Time.Format(time.RFC3339Nano)
	m["level"] = e.Level.String()
	m["msg"] = e.Message
	if e.Module != "" {
		m["module"] = e.Module
	}
	if e.Job != "" {
		m["job"] = e.Job
	}
	if e.Source != "" {
		m["source"] = e.Source
	}
	for _, attr := range e.Attributes {
		m[attr.Key] = attr.Value
	}
	return m
}

// JSONFormatter formats log entries as JSON.
type JSONFormatter struct {
	mu sync.Mutex
}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Format formats a log entry as JSON.
func (f *JSONFormatter) Format(w io.Writer, entry *LogEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	m := entry.ToMap()
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(m)
}

// TextFormatter formats log entries as human-readable text.
type TextFormatter struct {
	EnableColors bool
	TimeFormat   string
	mu           sync.Mutex
}

// NewTextFormatter creates a new text formatter.
func NewTextFormatter(enableColors bool, timeFormat string) *TextFormatter {
	if timeFormat == "" {
		timeFormat = time.RFC3339
	}
	return &TextFormatter{
		EnableColors: enableColors,
		TimeFormat:   timeFormat,
	}
}

// Color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
	colorCyan   = "\033[36m"
)

// levelColor returns the color code for a log level.
func levelColor(level Level) string {
	switch level {
	case DebugLevel:
		return colorGray
	case InfoLevel:
		return colorBlue
	case WarnLevel:
		return colorYellow
	case ErrorLevel:
		return colorRed
	default:
		return colorReset
	}
}

// Format formats a log entry as human-readable text.
func (f *TextFormatter) Format(w io.Writer, entry *LogEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var parts []string

	// Time
	timeStr := entry.Time.Format(f.TimeFormat)
	if f.EnableColors {
		parts = append(parts, fmt.Sprintf("%s[%s]%s", colorGray, timeStr, colorReset))
	} else {
		parts = append(parts, fmt.Sprintf("[%s]", timeStr))
	}

	// Level
	levelStr := entry.Level.String()
	if f.EnableColors {
		color := levelColor(entry.Level)
		parts = append(parts, fmt.Sprintf("%s%-5s%s", color, levelStr, colorReset))
	} else {
		parts = append(parts, fmt.Sprintf("%-5s", levelStr))
	}

	// Module and Job
	contextParts := []string{}
	if entry.Module != "" {
		contextParts = append(contextParts, entry.Module)
	}
	if entry.Job != "" {
		contextParts = append(contextParts, entry.Job)
	}
	if len(contextParts) > 0 {
		if f.EnableColors {
			parts = append(parts, fmt.Sprintf("%s(%s)%s", colorCyan, strings.Join(contextParts, "/"), colorReset))
		} else {
			parts = append(parts, fmt.Sprintf("(%s)", strings.Join(contextParts, "/")))
		}
	}

	// Message
	parts = append(parts, entry.Message)

	// Attributes
	if len(entry.Attributes) > 0 {
		attrParts := []string{}
		for _, attr := range entry.Attributes {
			attrParts = append(attrParts, fmt.Sprintf("%s=%v", attr.Key, attr.Value))
		}
		if f.EnableColors {
			parts = append(parts, fmt.Sprintf("%s%s%s", colorGray, strings.Join(attrParts, " "), colorReset))
		} else {
			parts = append(parts, strings.Join(attrParts, " "))
		}
	}

	_, err := fmt.Fprintln(w, strings.Join(parts, " "))
	return err
}

// MultiWriter wraps multiple writers for simultaneous output.
type MultiWriter struct {
	writers []io.Writer
	mu      sync.RWMutex
}

// NewMultiWriter creates a new MultiWriter.
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	// Filter out nil writers
	var validWriters []io.Writer
	for _, w := range writers {
		if w != nil {
			validWriters = append(validWriters, w)
		}
	}
	return &MultiWriter{writers: validWriters}
}

// Write implements io.Writer by writing to all contained writers.
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	mw.mu.RLock()
	defer mw.mu.RUnlock()

	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
		if n != len(p) {
			return n, io.ErrShortWrite
		}
	}
	return len(p), nil
}

// AddWriter adds a writer to the multi-writer.
func (mw *MultiWriter) AddWriter(w io.Writer) {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	if w != nil {
		mw.writers = append(mw.writers, w)
	}
}

// Close closes all writers that implement io.Closer.
func (mw *MultiWriter) Close() error {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	var errs []error
	for _, w := range mw.writers {
		if closer, ok := w.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing writers: %v", errs)
	}
	return nil
}

// FormatterLogger is a Logger implementation that uses custom formatters.
type FormatterLogger struct {
	formatter Formatter
	writer    io.Writer
	level     Level
	mu        sync.RWMutex
	module    string
	job       string
	attrs     []Attr
}

// NewFormatterLogger creates a new FormatterLogger.
func NewFormatterLogger(formatter Formatter, writer io.Writer, level Level) *FormatterLogger {
	return &FormatterLogger{
		formatter: formatter,
		writer:    writer,
		level:     level,
	}
}

// log is the internal logging method.
func (l *FormatterLogger) log(level Level, msg string, attrs ...Attr) {
	if level < l.level {
		return
	}

	l.mu.RLock()
	module := l.module
	job := l.job
	baseAttrs := make([]Attr, len(l.attrs))
	copy(baseAttrs, l.attrs)
	l.mu.RUnlock()

	// Combine base attrs with new attrs
	allAttrs := append(baseAttrs, attrs...)

	entry := &LogEntry{
		Time:       time.Now(),
		Level:      level,
		Message:    msg,
		Module:     module,
		Job:        job,
		Attributes: allAttrs,
	}

	_ = l.formatter.Format(l.writer, entry)
}

// Debug logs a debug message.
func (l *FormatterLogger) Debug(msg string, attrs ...Attr) {
	l.log(DebugLevel, msg, attrs...)
}

// Info logs an info message.
func (l *FormatterLogger) Info(msg string, attrs ...Attr) {
	l.log(InfoLevel, msg, attrs...)
}

// Warn logs a warning message.
func (l *FormatterLogger) Warn(msg string, attrs ...Attr) {
	l.log(WarnLevel, msg, attrs...)
}

// Error logs an error message.
func (l *FormatterLogger) Error(msg string, attrs ...Attr) {
	l.log(ErrorLevel, msg, attrs...)
}

// Success logs a success message.
func (l *FormatterLogger) Success(msg string, attrs ...Attr) {
	l.log(InfoLevel, msg, append([]Attr{{Key: "status", Value: "success"}}, attrs...)...)
}

// Loop logs a loop iteration message.
func (l *FormatterLogger) Loop(msg string, attrs ...Attr) {
	l.log(DebugLevel, msg, append([]Attr{{Key: "type", Value: "loop"}}, attrs...)...)
}

// WithContext returns a new Logger with context information.
func (l *FormatterLogger) WithContext(ctx context.Context) Logger {
	newLogger := &FormatterLogger{
		formatter: l.formatter,
		writer:    l.writer,
		level:     l.level,
		attrs:     make([]Attr, len(l.attrs)),
	}
	copy(newLogger.attrs, l.attrs)

	if module := ModuleFromContext(ctx); module != "" {
		newLogger.module = module
	}
	if job := JobFromContext(ctx); job != "" {
		newLogger.job = job
	}
	if loop := LoopFromContext(ctx); loop > 0 {
		newLogger.attrs = append(newLogger.attrs, Int("loop", loop))
	}

	return newLogger
}

// WithJob returns a new Logger with job information.
func (l *FormatterLogger) WithJob(module, job string) Logger {
	newLogger := &FormatterLogger{
		formatter: l.formatter,
		writer:    l.writer,
		level:     l.level,
		module:    module,
		job:       job,
		attrs:     make([]Attr, len(l.attrs)),
	}
	copy(newLogger.attrs, l.attrs)
	return newLogger
}

// WithAttrs returns a new Logger with additional attributes.
func (l *FormatterLogger) WithAttrs(attrs ...Attr) Logger {
	newLogger := &FormatterLogger{
		formatter: l.formatter,
		writer:    l.writer,
		level:     l.level,
		module:    l.module,
		job:       l.job,
		attrs:     make([]Attr, len(l.attrs)+len(attrs)),
	}
	copy(newLogger.attrs, l.attrs)
	copy(newLogger.attrs[len(l.attrs):], attrs)
	return newLogger
}

// SetLevel sets the logging level.
func (l *FormatterLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current logging level.
func (l *FormatterLogger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// IsEnabled returns true if the given level is enabled.
func (l *FormatterLogger) IsEnabled(level Level) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

// Ensure FormatterLogger implements Logger interface
var _ Logger = (*FormatterLogger)(nil)

// CreateLogger creates a Logger based on FormatConfig.
func CreateLogger(config *FormatConfig) (Logger, error) {
	var formatter Formatter
	switch config.Format {
	case FormatText:
		formatter = NewTextFormatter(config.EnableColors, config.TimeFormat)
	case FormatJSON:
		fallthrough
	default:
		formatter = NewJSONFormatter()
	}

	var writers []io.Writer
	if config.Output == OutputStdout || config.Output == OutputBoth {
		writers = append(writers, os.Stdout)
	}
	// Note: File output would need to be handled separately with proper file path

	var writer io.Writer
	if len(writers) == 1 {
		writer = writers[0]
	} else if len(writers) > 1 {
		writer = NewMultiWriter(writers...)
	} else {
		writer = os.Stdout
	}

	return NewFormatterLogger(formatter, writer, config.Level), nil
}

// GetDefaultFormatForEnvironment returns the default format for the given environment.
func GetDefaultFormatForEnvironment(env Environment) Format {
	return env.DefaultFormat()
}

// IsDevelopment returns true if running in development environment.
func IsDevelopment() bool {
	return DetectEnvironment() == EnvDevelopment
}

// IsProduction returns true if running in production environment.
func IsProduction() bool {
	return DetectEnvironment() == EnvProduction
}

// IsTesting returns true if running in testing environment.
func IsTesting() bool {
	return DetectEnvironment() == EnvTesting
}
