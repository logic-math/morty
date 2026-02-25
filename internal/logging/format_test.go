// Package logging provides tests for log formatting functionality.
package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/morty/morty/internal/config"
)

// TestFormatTypes tests Format type and related functions.
func TestFormatTypes(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       Format
		wantValid  bool
		wantString string
	}{
		{"json lowercase", "json", FormatJSON, true, "json"},
		{"JSON uppercase", "JSON", FormatJSON, true, "json"},
		{"text lowercase", "text", FormatText, true, "text"},
		{"TEXT uppercase", "TEXT", FormatText, true, "text"},
		{"invalid format", "yaml", FormatJSON, true, "json"},
		{"empty format", "", FormatJSON, true, "json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFromString(tt.input)
			if got != tt.want {
				t.Errorf("FormatFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
			if got.IsValid() != tt.wantValid {
				t.Errorf("Format(%v).IsValid() = %v, want %v", got, got.IsValid(), tt.wantValid)
			}
			if got.String() != tt.wantString {
				t.Errorf("Format(%v).String() = %v, want %v", got, got.String(), tt.wantString)
			}
		})
	}
}

// TestOutputTargetTypes tests OutputTarget type and related functions.
func TestOutputTargetTypes(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		want       OutputTarget
		wantValid  bool
		wantString string
	}{
		{"stdout lowercase", "stdout", OutputStdout, true, "stdout"},
		{"STDOUT uppercase", "STDOUT", OutputStdout, true, "stdout"},
		{"file lowercase", "file", OutputFile, true, "file"},
		{"FILE uppercase", "FILE", OutputFile, true, "file"},
		{"both lowercase", "both", OutputBoth, true, "both"},
		{"BOTH uppercase", "BOTH", OutputBoth, true, "both"},
		{"invalid output", "stderr", OutputStdout, true, "stdout"},
		{"empty output", "", OutputStdout, true, "stdout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OutputTargetFromString(tt.input)
			if got != tt.want {
				t.Errorf("OutputTargetFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
			if got.IsValid() != tt.wantValid {
				t.Errorf("OutputTarget(%v).IsValid() = %v, want %v", got, got.IsValid(), tt.wantValid)
			}
			if got.String() != tt.wantString {
				t.Errorf("OutputTarget(%v).String() = %v, want %v", got, got.String(), tt.wantString)
			}
		})
	}
}

// TestEnvironmentTypes tests Environment type and related functions.
func TestEnvironmentTypes(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		want            Environment
		wantFormat      Format
		wantDevLevel    Level
		wantString      string
	}{
		{"development", "development", EnvDevelopment, FormatText, DebugLevel, "development"},
		{"dev", "dev", EnvDevelopment, FormatText, DebugLevel, "development"},
		{"production", "production", EnvProduction, FormatJSON, InfoLevel, "production"},
		{"prod", "prod", EnvProduction, FormatJSON, InfoLevel, "production"},
		{"testing", "testing", EnvTesting, FormatText, WarnLevel, "testing"},
		{"test", "test", EnvTesting, FormatText, WarnLevel, "testing"},
		{"invalid", "invalid", EnvDevelopment, FormatText, DebugLevel, "development"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnvironmentFromString(tt.input)
			if got != tt.want {
				t.Errorf("EnvironmentFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
			if got.DefaultFormat() != tt.wantFormat {
				t.Errorf("Environment(%v).DefaultFormat() = %v, want %v", got, got.DefaultFormat(), tt.wantFormat)
			}
			if got.String() != tt.wantString {
				t.Errorf("Environment(%v).String() = %v, want %v", got, got.String(), tt.wantString)
			}
		})
	}
}

// TestDetectEnvironment tests environment detection from environment variables.
func TestDetectEnvironment(t *testing.T) {
	// Save and restore environment variables
	originalVars := map[string]string{
		"MORTY_ENV": os.Getenv("MORTY_ENV"),
		"NODE_ENV":  os.Getenv("NODE_ENV"),
		"GO_ENV":    os.Getenv("GO_ENV"),
		"ENV":       os.Getenv("ENV"),
	}
	defer func() {
		for k, v := range originalVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected Environment
	}{
		{
			name:     "MORTY_ENV takes precedence",
			envVars:  map[string]string{"MORTY_ENV": "production", "NODE_ENV": "development"},
			expected: EnvProduction,
		},
		{
			name:     "NODE_ENV used when MORTY_ENV not set",
			envVars:  map[string]string{"MORTY_ENV": "", "NODE_ENV": "production"},
			expected: EnvProduction,
		},
		{
			name:     "GO_ENV used when others not set",
			envVars:  map[string]string{"MORTY_ENV": "", "NODE_ENV": "", "GO_ENV": "testing"},
			expected: EnvTesting,
		},
		{
			name:     "ENV used when others not set",
			envVars:  map[string]string{"MORTY_ENV": "", "NODE_ENV": "", "GO_ENV": "", "ENV": "production"},
			expected: EnvProduction,
		},
		{
			name:     "default to development",
			envVars:  map[string]string{"MORTY_ENV": "", "NODE_ENV": "", "GO_ENV": "", "ENV": ""},
			expected: EnvDevelopment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			for k := range originalVars {
				os.Unsetenv(k)
			}
			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			got := DetectEnvironment()
			if got != tt.expected {
				t.Errorf("DetectEnvironment() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestDefaultFormatConfig tests the default format configuration.
func TestDefaultFormatConfig(t *testing.T) {
	config := DefaultFormatConfig()

	if config.Format != FormatText && config.Format != FormatJSON {
		t.Errorf("DefaultFormatConfig().Format = %v, want text or json", config.Format)
	}
	if config.Output != OutputStdout {
		t.Errorf("DefaultFormatConfig().Output = %v, want stdout", config.Output)
	}
	if config.TimeFormat == "" {
		t.Error("DefaultFormatConfig().TimeFormat should not be empty")
	}
	if config.Environment == "" {
		t.Error("DefaultFormatConfig().Environment should not be empty")
	}
}

// TestJSONFormatter tests JSON formatting.
func TestJSONFormatter(t *testing.T) {
	formatter := NewJSONFormatter()
	var buf bytes.Buffer

	entry := &LogEntry{
		Time:       time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:      InfoLevel,
		Message:    "test message",
		Module:     "test-module",
		Job:        "test-job",
		Attributes: []Attr{String("key1", "value1"), Int("key2", 42)},
	}

	err := formatter.Format(&buf, entry)
	if err != nil {
		t.Fatalf("JSONFormatter.Format() error = %v", err)
	}

	// Parse the output as JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, buf.String())
	}

	// Verify fields
	if result["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", result["level"])
	}
	if result["msg"] != "test message" {
		t.Errorf("msg = %v, want 'test message'", result["msg"])
	}
	if result["module"] != "test-module" {
		t.Errorf("module = %v, want 'test-module'", result["module"])
	}
	if result["job"] != "test-job" {
		t.Errorf("job = %v, want 'test-job'", result["job"])
	}
	if result["key1"] != "value1" {
		t.Errorf("key1 = %v, want 'value1'", result["key1"])
	}
	if result["key2"] != float64(42) {
		t.Errorf("key2 = %v, want 42", result["key2"])
	}
	if result["time"] == "" {
		t.Error("time field is missing")
	}
}

// TestTextFormatter tests text formatting.
func TestTextFormatter(t *testing.T) {
	tests := []struct {
		name         string
		enableColors bool
		checkFunc    func(t *testing.T, output string)
	}{
		{
			name:         "text format without colors",
			enableColors: false,
			checkFunc: func(t *testing.T, output string) {
				// Verify time is present
				if !strings.Contains(output, "2024-01-15") {
					t.Error("Output should contain date")
				}
				// Verify level is present
				if !strings.Contains(output, "INFO") {
					t.Error("Output should contain level INFO")
				}
				// Verify message is present
				if !strings.Contains(output, "test message") {
					t.Error("Output should contain message")
				}
				// Verify module/job context is present
				if !strings.Contains(output, "test-module/test-job") {
					t.Error("Output should contain module/job context")
				}
				// Verify attributes are present
				if !strings.Contains(output, "key1=value1") {
					t.Error("Output should contain attribute key1=value1")
				}
			},
		},
		{
			name:         "text format with colors",
			enableColors: true,
			checkFunc: func(t *testing.T, output string) {
				// Verify ANSI color codes are present
				if !strings.Contains(output, "\033[") {
					t.Error("Output should contain ANSI color codes")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewTextFormatter(tt.enableColors, time.RFC3339)
			var buf bytes.Buffer

			entry := &LogEntry{
				Time:       time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				Level:      InfoLevel,
				Message:    "test message",
				Module:     "test-module",
				Job:        "test-job",
				Attributes: []Attr{String("key1", "value1")},
			}

			err := formatter.Format(&buf, entry)
			if err != nil {
				t.Fatalf("TextFormatter.Format() error = %v", err)
			}

			tt.checkFunc(t, buf.String())
		})
	}
}

// TestTextFormatterAllLevels tests text formatting for all log levels.
func TestTextFormatterAllLevels(t *testing.T) {
	formatter := NewTextFormatter(false, time.RFC3339)
	levels := []Level{DebugLevel, InfoLevel, WarnLevel, ErrorLevel}

	for _, level := range levels {
		t.Run(level.String(), func(t *testing.T) {
			var buf bytes.Buffer
			entry := &LogEntry{
				Time:    time.Now(),
				Level:   level,
				Message: "test",
			}

			err := formatter.Format(&buf, entry)
			if err != nil {
				t.Fatalf("TextFormatter.Format() error = %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, level.String()) {
				t.Errorf("Output should contain level %s", level.String())
			}
		})
	}
}

// TestMultiWriter tests the MultiWriter functionality.
func TestMultiWriter(t *testing.T) {
	var buf1 bytes.Buffer
	var buf2 bytes.Buffer
	var buf3 bytes.Buffer

	mw := NewMultiWriter(&buf1, &buf2, nil, &buf3) // nil should be filtered out

	testData := []byte("hello world\n")
	n, err := mw.Write(testData)
	if err != nil {
		t.Fatalf("MultiWriter.Write() error = %v", err)
	}
	if n != len(testData) {
		t.Errorf("MultiWriter.Write() wrote %d bytes, want %d", n, len(testData))
	}

	// Verify all non-nil writers received the data
	if buf1.String() != string(testData) {
		t.Errorf("buf1 = %q, want %q", buf1.String(), testData)
	}
	if buf2.String() != string(testData) {
		t.Errorf("buf2 = %q, want %q", buf2.String(), testData)
	}
	if buf3.String() != string(testData) {
		t.Errorf("buf3 = %q, want %q", buf3.String(), testData)
	}
}

// TestMultiWriterAddWriter tests adding writers dynamically.
func TestMultiWriterAddWriter(t *testing.T) {
	var buf1 bytes.Buffer
	var buf2 bytes.Buffer

	mw := NewMultiWriter(&buf1)
	mw.AddWriter(&buf2)

	testData := []byte("test")
	mw.Write(testData)

	if buf1.String() != string(testData) {
		t.Errorf("buf1 = %q, want %q", buf1.String(), testData)
	}
	if buf2.String() != string(testData) {
		t.Errorf("buf2 = %q, want %q", buf2.String(), testData)
	}
}

// TestFormatterLogger tests the FormatterLogger implementation.
func TestFormatterLogger(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewTextFormatter(false, time.RFC3339)
	logger := NewFormatterLogger(formatter, &buf, DebugLevel)

	tests := []struct {
		name      string
		logFunc   func()
		wantLevel string
		wantMsg   string
	}{
		{
			name:      "Debug log",
			logFunc:   func() { logger.Debug("debug message", String("key", "value")) },
			wantLevel: "DEBUG",
			wantMsg:   "debug message",
		},
		{
			name:      "Info log",
			logFunc:   func() { logger.Info("info message") },
			wantLevel: "INFO",
			wantMsg:   "info message",
		},
		{
			name:      "Warn log",
			logFunc:   func() { logger.Warn("warn message") },
			wantLevel: "WARN",
			wantMsg:   "warn message",
		},
		{
			name:      "Error log",
			logFunc:   func() { logger.Error("error message") },
			wantLevel: "ERROR",
			wantMsg:   "error message",
		},
		{
			name:      "Success log",
			logFunc:   func() { logger.Success("success message") },
			wantLevel: "INFO",
			wantMsg:   "success message",
		},
		{
			name:      "Loop log",
			logFunc:   func() { logger.Loop("loop message") },
			wantLevel: "DEBUG",
			wantMsg:   "loop message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()
			output := buf.String()

			if !strings.Contains(output, tt.wantLevel) {
				t.Errorf("Output should contain level %s, got: %s", tt.wantLevel, output)
			}
			if !strings.Contains(output, tt.wantMsg) {
				t.Errorf("Output should contain message %q, got: %s", tt.wantMsg, output)
			}
		})
	}
}

// TestFormatterLoggerLevelFiltering tests level filtering.
func TestFormatterLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewTextFormatter(false, time.RFC3339)
	logger := NewFormatterLogger(formatter, &buf, InfoLevel)

	// Debug should be filtered out
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug message should be filtered out when level is Info")
	}

	// Info should pass through
	logger.Info("info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Error("Info message should be logged")
	}
}

// TestFormatterLoggerWithJob tests the WithJob method.
func TestFormatterLoggerWithJob(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewTextFormatter(false, time.RFC3339)
	logger := NewFormatterLogger(formatter, &buf, DebugLevel)

	jobLogger := logger.WithJob("test-module", "test-job")
	jobLogger.Info("job message")

	output := buf.String()
	if !strings.Contains(output, "test-module/test-job") {
		t.Errorf("Output should contain job context, got: %s", output)
	}
}

// TestFormatterLoggerWithAttrs tests the WithAttrs method.
func TestFormatterLoggerWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewTextFormatter(false, time.RFC3339)
	logger := NewFormatterLogger(formatter, &buf, DebugLevel)

	attrLogger := logger.WithAttrs(String("base", "value"))
	attrLogger.Info("message", String("extra", "data"))

	output := buf.String()
	if !strings.Contains(output, "base=value") {
		t.Errorf("Output should contain base attribute, got: %s", output)
	}
	if !strings.Contains(output, "extra=data") {
		t.Errorf("Output should contain extra attribute, got: %s", output)
	}
}

// TestFormatterLoggerSetLevel tests the SetLevel method.
func TestFormatterLoggerSetLevel(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewTextFormatter(false, time.RFC3339)
	logger := NewFormatterLogger(formatter, &buf, DebugLevel)

	if logger.GetLevel() != DebugLevel {
		t.Errorf("GetLevel() = %v, want DebugLevel", logger.GetLevel())
	}

	logger.SetLevel(ErrorLevel)
	if logger.GetLevel() != ErrorLevel {
		t.Errorf("After SetLevel(ErrorLevel), GetLevel() = %v, want ErrorLevel", logger.GetLevel())
	}

	// Info should now be filtered
	buf.Reset()
	logger.Info("info message")
	if buf.Len() > 0 {
		t.Error("Info message should be filtered after setting level to Error")
	}
}

// TestFormatterLoggerIsEnabled tests the IsEnabled method.
func TestFormatterLoggerIsEnabled(t *testing.T) {
	formatter := NewTextFormatter(false, time.RFC3339)
	logger := NewFormatterLogger(formatter, io.Discard, WarnLevel)

	if logger.IsEnabled(DebugLevel) {
		t.Error("DebugLevel should not be enabled when level is WarnLevel")
	}
	if logger.IsEnabled(InfoLevel) {
		t.Error("InfoLevel should not be enabled when level is WarnLevel")
	}
	if !logger.IsEnabled(WarnLevel) {
		t.Error("WarnLevel should be enabled when level is WarnLevel")
	}
	if !logger.IsEnabled(ErrorLevel) {
		t.Error("ErrorLevel should be enabled when level is WarnLevel")
	}
}

// TestCreateLogger tests the CreateLogger function.
func TestCreateLogger(t *testing.T) {
	config := &FormatConfig{
		Format:       FormatJSON,
		Output:       OutputStdout,
		Level:        InfoLevel,
		TimeFormat:   time.RFC3339,
		EnableColors: false,
	}

	logger, err := CreateLogger(config)
	if err != nil {
		t.Fatalf("CreateLogger() error = %v", err)
	}
	if logger == nil {
		t.Fatal("CreateLogger() returned nil logger")
	}
}

// TestFormatConfigFromLoggingConfig tests configuration conversion.
func TestFormatConfigFromLoggingConfig(t *testing.T) {
	tests := []struct {
		name         string
		loggingCfg   *config.LoggingConfig
		wantFormat   Format
		wantOutput   OutputTarget
		wantLevel    Level
		wantColors   bool
	}{
		{
			name: "JSON format with stdout",
			loggingCfg: &config.LoggingConfig{
				Level:  "info",
				Format: "json",
				Output: "stdout",
			},
			wantFormat: FormatJSON,
			wantOutput: OutputStdout,
			wantLevel:  InfoLevel,
			wantColors: true, // development env uses colors
		},
		{
			name: "Text format with file",
			loggingCfg: &config.LoggingConfig{
				Level:  "debug",
				Format: "text",
				Output: "file",
				File:   config.FileConfig{Enabled: true},
			},
			wantFormat: FormatText,
			wantOutput: OutputFile,
			wantLevel:  DebugLevel,
			wantColors: false, // file output doesn't use colors
		},
		{
			name: "Both output",
			loggingCfg: &config.LoggingConfig{
				Level:  "warn",
				Format: "json",
				Output: "both",
				File:   config.FileConfig{Enabled: true},
			},
			wantFormat: FormatJSON,
			wantOutput: OutputBoth,
			wantLevel:  WarnLevel,
			wantColors: false, // both includes file, no colors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatConfigFromLoggingConfig(tt.loggingCfg)

			if got.Format != tt.wantFormat {
				t.Errorf("Format = %v, want %v", got.Format, tt.wantFormat)
			}
			if got.Output != tt.wantOutput {
				t.Errorf("Output = %v, want %v", got.Output, tt.wantOutput)
			}
			if got.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", got.Level, tt.wantLevel)
			}
		})
	}
}

// TestNewLoggerFromConfig tests creating logger from config.
func TestNewLoggerFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	tests := []struct {
		name       string
		cfg        *config.LoggingConfig
		wantErr    bool
		checkFunc  func(t *testing.T, logger Logger, closer io.Closer)
	}{
		{
			name: "JSON stdout only",
			cfg: &config.LoggingConfig{
				Level:  "info",
				Format: "json",
				Output: "stdout",
				File:   config.FileConfig{Enabled: false},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, logger Logger, closer io.Closer) {
				if logger == nil {
					t.Error("Expected non-nil logger")
				}
			},
		},
		{
			name: "Text with file output",
			cfg: &config.LoggingConfig{
				Level:  "debug",
				Format: "text",
				Output: "file",
				File: config.FileConfig{
					Enabled: true,
					Path:    logFile,
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, logger Logger, closer io.Closer) {
				if closer == nil {
					t.Error("Expected non-nil closer for file output")
				}
			},
		},
		{
			name: "Both output",
			cfg: &config.LoggingConfig{
				Level:  "info",
				Format: "json",
				Output: "both",
				File: config.FileConfig{
					Enabled: true,
					Path:    logFile + ".both",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, closer, err := NewLoggerFromConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLoggerFromConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, logger, closer)
			}
			if closer != nil {
				closer.Close()
			}
		})
	}
}

// TestAutoSelectFormat tests automatic format selection.
func TestAutoSelectFormat(t *testing.T) {
	tests := []struct {
		name         string
		configFormat string
		expected     Format
	}{
		{"valid json", "json", FormatJSON},
		{"valid text", "text", FormatText},
		{"invalid format", "invalid", FormatJSON},
		{"empty format", "", FormatJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AutoSelectFormat(tt.configFormat)
			if got != tt.expected {
				t.Errorf("AutoSelectFormat(%q) = %v, want %v", tt.configFormat, got, tt.expected)
			}
		})
	}
}

// TestAutoSelectOutput tests automatic output selection.
func TestAutoSelectOutput(t *testing.T) {
	tests := []struct {
		name        string
		configOut   string
		fileEnabled bool
		expected    OutputTarget
	}{
		{"stdout valid", "stdout", true, OutputStdout},
		{"file valid", "file", true, OutputFile},
		{"file disabled", "file", false, OutputStdout},
		{"both valid", "both", true, OutputBoth},
		{"both file disabled", "both", false, OutputStdout},
		{"invalid fallback", "invalid", true, OutputStdout},
		{"empty file enabled", "", true, OutputStdout},
		{"empty file disabled", "", false, OutputStdout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AutoSelectOutput(tt.configOut, tt.fileEnabled)
			if got != tt.expected {
				t.Errorf("AutoSelectOutput(%q, %v) = %v, want %v", tt.configOut, tt.fileEnabled, got, tt.expected)
			}
		})
	}
}

// TestGetEnvironmentInfo tests environment info retrieval.
func TestGetEnvironmentInfo(t *testing.T) {
	info := GetEnvironmentInfo()

	requiredKeys := []string{"environment", "defaultFormat", "defaultLevel", "isDevelopment", "isProduction", "isTesting"}
	for _, key := range requiredKeys {
		if _, ok := info[key]; !ok {
			t.Errorf("GetEnvironmentInfo() missing key %q", key)
		}
	}
}

// TestLogEntryToMap tests LogEntry conversion to map.
func TestLogEntryToMap(t *testing.T) {
	entry := &LogEntry{
		Time:       time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Level:      InfoLevel,
		Message:    "test message",
		Module:     "test-module",
		Job:        "test-job",
		Source:     "file.go:42",
		Attributes: []Attr{String("key", "value")},
	}

	m := entry.ToMap()

	if m["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", m["level"])
	}
	if m["msg"] != "test message" {
		t.Errorf("msg = %v, want 'test message'", m["msg"])
	}
	if m["module"] != "test-module" {
		t.Errorf("module = %v, want 'test-module'", m["module"])
	}
	if m["job"] != "test-job" {
		t.Errorf("job = %v, want 'test-job'", m["job"])
	}
	if m["source"] != "file.go:42" {
		t.Errorf("source = %v, want 'file.go:42'", m["source"])
	}
	if m["key"] != "value" {
		t.Errorf("key = %v, want 'value'", m["key"])
	}
	if m["time"] == "" {
		t.Error("time should not be empty")
	}
}

// TestFormatterLoggerWithContext tests WithContext method.
func TestFormatterLoggerWithContext(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewTextFormatter(false, time.RFC3339)
	logger := NewFormatterLogger(formatter, &buf, DebugLevel)

	ctx := ContextWithModule(context.Background(), "my-module")
	ctx = ContextWithJob(ctx, "my-job")
	ctx = ContextWithLoop(ctx, 3)

	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info("context test")

	output := buf.String()
	if !strings.Contains(output, "my-module/my-job") {
		t.Errorf("Output should contain module/job from context, got: %s", output)
	}
	if !strings.Contains(output, "loop=3") {
		t.Errorf("Output should contain loop from context, got: %s", output)
	}
}

// TestMultiCloser tests the multiCloser implementation.
func TestMultiCloser(t *testing.T) {
	// Test with closers
	var buf1 bytes.Buffer
	var buf2 bytes.Buffer

	closer := &multiCloser{
		closers: []io.Closer{
			nopCloser{Writer: &buf1},
			nopCloser{Writer: &buf2},
		},
	}

	err := closer.Close()
	if err != nil {
		t.Errorf("multiCloser.Close() error = %v", err)
	}

	// Test with empty closers
	emptyCloser := &multiCloser{closers: []io.Closer{}}
	err = emptyCloser.Close()
	if err != nil {
		t.Errorf("empty multiCloser.Close() error = %v", err)
	}
}

// nopCloser is a WriteCloser that does nothing on Close.
type nopCloser struct {
	io.Writer
}

func (n nopCloser) Close() error {
	return nil
}

// TestFormatterInterface tests that formatters implement the interface correctly.
func TestFormatterInterface(t *testing.T) {
	tests := []struct {
		name      string
		formatter Formatter
	}{
		{"JSON", NewJSONFormatter()},
		{"Text", NewTextFormatter(false, time.RFC3339)},
	}

	entry := &LogEntry{
		Time:    time.Now(),
		Level:   InfoLevel,
		Message: "interface test",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.formatter.Format(&buf, entry)
			if err != nil {
				t.Errorf("Formatter.Format() error = %v", err)
			}
			if buf.Len() == 0 {
				t.Error("Formatter.Format() produced no output")
			}
		})
	}
}

// TestIsDevelopmentIsProductionIsTesting tests the environment check functions.
func TestIsDevelopmentIsProductionIsTesting(t *testing.T) {
	// Save and restore environment
	origEnv := os.Getenv("MORTY_ENV")
	defer os.Setenv("MORTY_ENV", origEnv)

	// Test development
	os.Setenv("MORTY_ENV", "development")
	if !IsDevelopment() {
		t.Error("IsDevelopment() should return true when MORTY_ENV=development")
	}
	if IsProduction() {
		t.Error("IsProduction() should return false when MORTY_ENV=development")
	}

	// Test production
	os.Setenv("MORTY_ENV", "production")
	if !IsProduction() {
		t.Error("IsProduction() should return true when MORTY_ENV=production")
	}
	if IsDevelopment() {
		t.Error("IsDevelopment() should return false when MORTY_ENV=production")
	}

	// Test testing
	os.Setenv("MORTY_ENV", "testing")
	if !IsTesting() {
		t.Error("IsTesting() should return true when MORTY_ENV=testing")
	}
}

// TestEnvironmentAwareLogger tests the EnvironmentAwareLogger function.
func TestEnvironmentAwareLogger(t *testing.T) {
	cfg := &config.LoggingConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}

	logger, closer, err := EnvironmentAwareLogger(cfg)
	if err != nil {
		t.Errorf("EnvironmentAwareLogger() error = %v", err)
	}
	if logger == nil {
		t.Error("EnvironmentAwareLogger() returned nil logger")
	}
	if closer != nil {
		closer.Close()
	}
}

// BenchmarkJSONFormatter benchmarks JSON formatting.
func BenchmarkJSONFormatter(b *testing.B) {
	formatter := NewJSONFormatter()
	entry := &LogEntry{
		Time:       time.Now(),
		Level:      InfoLevel,
		Message:    "benchmark message",
		Module:     "bench",
		Job:        "test",
		Attributes: []Attr{String("key1", "value1"), Int("key2", 42)},
	}
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		formatter.Format(&buf, entry)
	}
}

// BenchmarkTextFormatter benchmarks text formatting.
func BenchmarkTextFormatter(b *testing.B) {
	formatter := NewTextFormatter(false, time.RFC3339)
	entry := &LogEntry{
		Time:       time.Now(),
		Level:      InfoLevel,
		Message:    "benchmark message",
		Module:     "bench",
		Job:        "test",
		Attributes: []Attr{String("key1", "value1"), Int("key2", 42)},
	}
	var buf bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		formatter.Format(&buf, entry)
	}
}
