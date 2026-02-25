// Package logging provides a structured logging interface for Morty.
package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
)

// TestNewSlogAdapter tests creating a new SlogAdapter.
func TestNewSlogAdapter(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		output  string
		level   Level
		wantErr bool
	}{
		{
			name:    "JSON format stdout",
			format:  "json",
			output:  "stdout",
			level:   InfoLevel,
			wantErr: false,
		},
		{
			name:    "Text format stdout",
			format:  "text",
			output:  "stdout",
			level:   DebugLevel,
			wantErr: false,
		},
		{
			name:    "Invalid format defaults to JSON",
			format:  "invalid",
			output:  "stdout",
			level:   WarnLevel,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewSlogAdapter(tt.format, tt.output, tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSlogAdapter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if adapter == nil {
				t.Error("NewSlogAdapter() returned nil adapter")
				return
			}
			if adapter.GetLevel() != tt.level {
				t.Errorf("Expected level %v, got %v", tt.level, adapter.GetLevel())
			}
		})
	}
}

// TestSlogAdapterLogLevels tests all log level methods.
func TestSlogAdapterLogLevels(t *testing.T) {
	var buf bytes.Buffer

	// Create a custom handler that writes to buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	adapter := &SlogAdapter{
		logger: logger,
		level:  &slog.LevelVar{},
	}
	adapter.level.Set(slog.LevelDebug)

	tests := []struct {
		name     string
		logFunc  func(string, ...Attr)
		msg      string
		attrs    []Attr
		wantLvl  string
		wantMsg  string
	}{
		{
			name:     "Debug log",
			logFunc:  adapter.Debug,
			msg:      "debug message",
			attrs:    []Attr{String("key", "value")},
			wantLvl:  "DEBUG",
			wantMsg:  "debug message",
		},
		{
			name:     "Info log",
			logFunc:  adapter.Info,
			msg:      "info message",
			attrs:    []Attr{Int("count", 42)},
			wantLvl:  "INFO",
			wantMsg:  "info message",
		},
		{
			name:     "Warn log",
			logFunc:  adapter.Warn,
			msg:      "warn message",
			attrs:    []Attr{Bool("flag", true)},
			wantLvl:  "WARN",
			wantMsg:  "warn message",
		},
		{
			name:     "Error log",
			logFunc:  adapter.Error,
			msg:      "error message",
			attrs:    []Attr{String("error", "test error")},
			wantLvl:  "ERROR",
			wantMsg:  "error message",
		},
		{
			name:     "Success log",
			logFunc:  adapter.Success,
			msg:      "success message",
			attrs:    []Attr{String("task", "completed")},
			wantLvl:  "SUCCESS",
			wantMsg:  "success message",
		},
		{
			name:     "Loop log",
			logFunc:  adapter.Loop,
			msg:      "loop message",
			attrs:    []Attr{Int("iteration", 1)},
			wantLvl:  "LOOP",
			wantMsg:  "loop message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.msg, tt.attrs...)

			var result map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			if result["msg"] != tt.wantMsg {
				t.Errorf("msg = %v, want %v", result["msg"], tt.wantMsg)
			}
			if result["level"] != tt.wantLvl {
				t.Errorf("level = %v, want %v", result["level"], tt.wantLvl)
			}
		})
	}
}

// TestSlogAdapterWithAttrs tests adding attributes to logger.
func TestSlogAdapterWithAttrs(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	adapter := &SlogAdapter{
		logger: logger,
		level:  &slog.LevelVar{},
	}
	adapter.level.Set(slog.LevelDebug)

	// Create logger with attributes
	loggerWithAttrs := adapter.WithAttrs(
		String("service", "test"),
		Int("version", 1),
	)

	loggerWithAttrs.Info("test message")

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result["service"] != "test" {
		t.Errorf("service = %v, want test", result["service"])
	}

	// JSON numbers are float64 when unmarshaled
	if v, ok := result["version"].(float64); !ok || v != 1 {
		t.Errorf("version = %v, want 1", result["version"])
	}
}

// TestSlogAdapterWithJob tests adding job context to logger.
func TestSlogAdapterWithJob(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	adapter := &SlogAdapter{
		logger: logger,
		level:  &slog.LevelVar{},
	}
	adapter.level.Set(slog.LevelDebug)

	// Create logger with job context
	loggerWithJob := adapter.WithJob("test-module", "test-job")

	loggerWithJob.Info("test message")

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result["module"] != "test-module" {
		t.Errorf("module = %v, want test-module", result["module"])
	}
	if result["job"] != "test-job" {
		t.Errorf("job = %v, want test-job", result["job"])
	}
}

// TestSlogAdapterWithContext tests adding context to logger.
func TestSlogAdapterWithContext(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	adapter := &SlogAdapter{
		logger: logger,
		level:  &slog.LevelVar{},
	}
	adapter.level.Set(slog.LevelDebug)

	// Create context with values
	ctx := context.Background()
	ctx = ContextWithModule(ctx, "ctx-module")
	ctx = ContextWithJob(ctx, "ctx-job")
	ctx = ContextWithLoop(ctx, 5)

	// Create logger with context
	loggerWithCtx := adapter.WithContext(ctx)

	loggerWithCtx.Info("test message")

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result["module"] != "ctx-module" {
		t.Errorf("module = %v, want ctx-module", result["module"])
	}
	if result["job"] != "ctx-job" {
		t.Errorf("job = %v, want ctx-job", result["job"])
	}

	// JSON numbers are float64 when unmarshaled
	if v, ok := result["loop"].(float64); !ok || v != 5 {
		t.Errorf("loop = %v, want 5", result["loop"])
	}
}

// TestSlogAdapterSetLevel tests setting log level.
func TestSlogAdapterSetLevel(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	adapter := &SlogAdapter{
		logger: logger,
		level:  &slog.LevelVar{},
	}
	adapter.level.Set(slog.LevelDebug)

	// Set to Info level
	adapter.SetLevel(InfoLevel)
	if adapter.GetLevel() != InfoLevel {
		t.Errorf("GetLevel() = %v, want InfoLevel", adapter.GetLevel())
	}

	buf.Reset()
	adapter.Debug("debug should not appear")
	if buf.Len() > 0 {
		t.Error("Debug message should not be logged at Info level")
	}

	buf.Reset()
	adapter.Info("info should appear")
	if buf.Len() == 0 {
		t.Error("Info message should be logged at Info level")
	}

	// Set to Error level
	adapter.SetLevel(ErrorLevel)
	if adapter.GetLevel() != ErrorLevel {
		t.Errorf("GetLevel() = %v, want ErrorLevel", adapter.GetLevel())
	}

	buf.Reset()
	adapter.Info("info should not appear at error level")
	if buf.Len() > 0 {
		t.Error("Info message should not be logged at Error level")
	}

	buf.Reset()
	adapter.Error("error should appear")
	if buf.Len() == 0 {
		t.Error("Error message should be logged at Error level")
	}
}

// TestSlogAdapterIsEnabled tests IsEnabled method.
func TestSlogAdapterIsEnabled(t *testing.T) {
	adapter := &SlogAdapter{
		level: &slog.LevelVar{},
	}
	adapter.level.Set(slog.LevelInfo)

	if !adapter.IsEnabled(InfoLevel) {
		t.Error("IsEnabled(InfoLevel) should be true at Info level")
	}
	if !adapter.IsEnabled(ErrorLevel) {
		t.Error("IsEnabled(ErrorLevel) should be true at Info level")
	}
	if adapter.IsEnabled(DebugLevel) {
		t.Error("IsEnabled(DebugLevel) should be false at Info level")
	}
}

// TestLevelString tests Level.String().
func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level.String() = %v, want %v", got, tt.want)
		}
	}
}

// TestParseLevel tests ParseLevel.
func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"warn", WarnLevel},
		{"WARN", WarnLevel},
		{"warning", WarnLevel},
		{"error", ErrorLevel},
		{"ERROR", ErrorLevel},
		{"invalid", InfoLevel}, // default
	}

	for _, tt := range tests {
		if got := ParseLevel(tt.input); got != tt.want {
			t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// TestAttrHelpers tests attribute helper functions.
func TestAttrHelpers(t *testing.T) {
	tests := []struct {
		name     string
		attr     Attr
		wantKey  string
		wantVal  interface{}
	}{
		{"String", String("key", "value"), "key", "value"},
		{"Int", Int("count", 42), "count", 42},
		{"Bool", Bool("enabled", true), "enabled", true},
		{"Any", Any("data", "any_value"), "data", "any_value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.Key != tt.wantKey {
				t.Errorf("Key = %v, want %v", tt.attr.Key, tt.wantKey)
			}
			if tt.attr.Value != tt.wantVal {
				t.Errorf("Value = %v, want %v", tt.attr.Value, tt.wantVal)
			}
		})
	}
}

// TestContextHelpers tests context helper functions.
func TestContextHelpers(t *testing.T) {
	ctx := context.Background()

	// Test empty context
	if ModuleFromContext(ctx) != "" {
		t.Error("ModuleFromContext should return empty string for empty context")
	}
	if JobFromContext(ctx) != "" {
		t.Error("JobFromContext should return empty string for empty context")
	}
	if LoopFromContext(ctx) != 0 {
		t.Error("LoopFromContext should return 0 for empty context")
	}

	// Test with values
	ctx = ContextWithModule(ctx, "test-module")
	ctx = ContextWithJob(ctx, "test-job")
	ctx = ContextWithLoop(ctx, 3)

	if ModuleFromContext(ctx) != "test-module" {
		t.Errorf("ModuleFromContext = %v, want test-module", ModuleFromContext(ctx))
	}
	if JobFromContext(ctx) != "test-job" {
		t.Errorf("JobFromContext = %v, want test-job", JobFromContext(ctx))
	}
	if LoopFromContext(ctx) != 3 {
		t.Errorf("LoopFromContext = %v, want 3", LoopFromContext(ctx))
	}
}

// TestSlogAdapterConcurrency tests thread safety.
func TestSlogAdapterConcurrency(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	adapter := &SlogAdapter{
		logger: logger,
		level:  &slog.LevelVar{},
	}
	adapter.level.Set(slog.LevelDebug)

	var wg sync.WaitGroup

	// Concurrent logging
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			adapter.Info("concurrent message", Int("n", n))
		}(i)
	}

	// Concurrent level changes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if n%2 == 0 {
				adapter.SetLevel(DebugLevel)
			} else {
				adapter.SetLevel(InfoLevel)
			}
		}(i)
	}

	wg.Wait()

	// Count logged messages
	lines := strings.Split(buf.String(), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}

	if count != 100 {
		t.Errorf("Expected 100 log messages, got %d", count)
	}
}

// TestToSlogAttrs tests conversion to slog.Attr.
func TestToSlogAttrs(t *testing.T) {
	attrs := []Attr{
		String("string", "value"),
		Int("int", 42),
		Bool("bool", true),
		Any("any", []string{"a", "b"}),
	}

	slogAttrs := toSlogAttrs(attrs)

	if len(slogAttrs) != len(attrs) {
		t.Errorf("Expected %d slog attrs, got %d", len(attrs), len(slogAttrs))
	}

	for i, attr := range attrs {
		if slogAttrs[i].Key != attr.Key {
			t.Errorf("slogAttrs[%d].Key = %v, want %v", i, slogAttrs[i].Key, attr.Key)
		}
	}
}

// TestNewSlogAdapterWithConfig tests creating adapter with full config.
func TestNewSlogAdapterWithConfig(t *testing.T) {
	// Create temporary directory for log file
	tmpDir := t.TempDir()
	logPath := tmpDir + "/test.log"

	adapter, err := NewSlogAdapterWithConfig("json", "file", logPath, InfoLevel, true)
	if err != nil {
		t.Fatalf("NewSlogAdapterWithConfig() error = %v", err)
	}
	defer adapter.Close()

	if adapter == nil {
		t.Fatal("NewSlogAdapterWithConfig() returned nil adapter")
	}

	// Write a log message
	adapter.Info("test message")

	// Verify file was created and contains the message
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test message") {
		t.Error("Log file does not contain expected message")
	}
}

// TestSlogAdapterClone tests cloning the adapter.
func TestSlogAdapterClone(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	adapter := &SlogAdapter{
		logger: logger,
		level:  &slog.LevelVar{},
		attrs:  []Attr{String("base", "value")},
		module: "original-module",
		job:    "original-job",
	}
	adapter.level.Set(slog.LevelDebug)

	// Clone the adapter
	cloned := adapter.clone()

	// Modify original
	adapter.attrs = append(adapter.attrs, String("extra", "data"))
	adapter.module = "modified-module"

	// Verify clone is not affected
	if cloned.module != "original-module" {
		t.Error("Clone was affected by original modification")
	}

	if len(cloned.attrs) != 1 {
		t.Errorf("Clone attrs length = %d, want 1", len(cloned.attrs))
	}
}

// TestSlogAdapterChaining tests method chaining.
func TestSlogAdapterChaining(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	adapter := &SlogAdapter{
		logger: logger,
		level:  &slog.LevelVar{},
	}
	adapter.level.Set(slog.LevelDebug)

	// Chain multiple With* methods
	chainedLogger := adapter.
		WithAttrs(String("attr1", "value1")).
		WithJob("module1", "job1").
		WithAttrs(Int("attr2", 2))

	chainedLogger.Info("chained message")

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Check all attributes are present
	if result["attr1"] != "value1" {
		t.Errorf("attr1 = %v, want value1", result["attr1"])
	}
	if result["module"] != "module1" {
		t.Errorf("module = %v, want module1", result["module"])
	}
	if result["job"] != "job1" {
		t.Errorf("job = %v, want job1", result["job"])
	}

	// JSON numbers are float64 when unmarshaled
	if v, ok := result["attr2"].(float64); !ok || v != 2 {
		t.Errorf("attr2 = %v, want 2", result["attr2"])
	}
}

// BenchmarkSlogAdapterInfo benchmarks the Info method.
func BenchmarkSlogAdapterInfo(b *testing.B) {
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	adapter := &SlogAdapter{
		logger: logger,
		level:  &slog.LevelVar{},
	}
	adapter.level.Set(slog.LevelDebug)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.Info("benchmark message", String("key", "value"), Int("count", i))
	}
}

// BenchmarkSlogAdapterWithAttrs benchmarks creating loggers with attributes.
func BenchmarkSlogAdapterWithAttrs(b *testing.B) {
	var buf bytes.Buffer

	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	adapter := &SlogAdapter{
		logger: logger,
		level:  &slog.LevelVar{},
	}
	adapter.level.Set(slog.LevelDebug)

	attrs := []Attr{
		String("service", "test"),
		Int("version", 1),
		String("env", "prod"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = adapter.WithAttrs(attrs...)
	}
}
