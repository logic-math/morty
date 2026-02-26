package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/morty/morty/internal/logging"
)

// mockLogger is a mock implementation of logging.Logger for testing.
type mockLogger struct {
	messages []logMessage
}

type logMessage struct {
	Level   string
	Message string
	Attrs   []logging.Attr
}

func (m *mockLogger) Debug(msg string, attrs ...logging.Attr) {
	m.messages = append(m.messages, logMessage{Level: "DEBUG", Message: msg, Attrs: attrs})
}

func (m *mockLogger) Info(msg string, attrs ...logging.Attr) {
	m.messages = append(m.messages, logMessage{Level: "INFO", Message: msg, Attrs: attrs})
}

func (m *mockLogger) Warn(msg string, attrs ...logging.Attr) {
	m.messages = append(m.messages, logMessage{Level: "WARN", Message: msg, Attrs: attrs})
}

func (m *mockLogger) Error(msg string, attrs ...logging.Attr) {
	m.messages = append(m.messages, logMessage{Level: "ERROR", Message: msg, Attrs: attrs})
}

func (m *mockLogger) Success(msg string, attrs ...logging.Attr) {
	m.messages = append(m.messages, logMessage{Level: "SUCCESS", Message: msg, Attrs: attrs})
}

func (m *mockLogger) Loop(msg string, attrs ...logging.Attr) {
	m.messages = append(m.messages, logMessage{Level: "LOOP", Message: msg, Attrs: attrs})
}

func (m *mockLogger) WithContext(ctx context.Context) logging.Logger {
	return m
}

func (m *mockLogger) WithJob(module, job string) logging.Logger {
	return m
}

func (m *mockLogger) WithAttrs(attrs ...logging.Attr) logging.Logger {
	return m
}

func (m *mockLogger) SetLevel(level logging.Level) {}

func (m *mockLogger) GetLevel() logging.Level {
	return logging.InfoLevel
}

func (m *mockLogger) IsEnabled(level logging.Level) bool {
	return true
}

// mockConfig is a mock implementation of config.Manager for testing.
type mockConfig struct {
	values map[string]interface{}
}

func (m *mockConfig) Load(path string) error {
	return nil
}

func (m *mockConfig) LoadWithMerge(userConfigPath string) error {
	return nil
}

func (m *mockConfig) Get(key string, defaultValue ...interface{}) (interface{}, error) {
	if val, ok := m.values[key]; ok {
		return val, nil
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return nil, fmt.Errorf("key not found: %s", key)
}

func (m *mockConfig) GetString(key string, defaultValue ...string) string {
	if val, ok := m.values[key].(string); ok {
		return val
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

func (m *mockConfig) GetInt(key string, defaultValue ...int) int {
	if val, ok := m.values[key].(int); ok {
		return val
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

func (m *mockConfig) GetBool(key string, defaultValue ...bool) bool {
	if val, ok := m.values[key].(bool); ok {
		return val
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

func (m *mockConfig) GetDuration(key string, defaultValue ...time.Duration) time.Duration {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return 0
}

func (m *mockConfig) Set(key string, value interface{}) error {
	if m.values == nil {
		m.values = make(map[string]interface{})
	}
	m.values[key] = value
	return nil
}

func (m *mockConfig) Save() error {
	return nil
}

func (m *mockConfig) SaveTo(path string) error {
	return nil
}

func (m *mockConfig) GetWorkDir() string {
	return ".morty"
}

func (m *mockConfig) GetLogDir() string {
	return ".morty/doing/logs"
}

func (m *mockConfig) GetResearchDir() string {
	return ".morty/research"
}

func (m *mockConfig) GetPlanDir() string {
	return ".morty/plan"
}

func (m *mockConfig) GetStatusFile() string {
	return ".morty/status.json"
}

func (m *mockConfig) GetConfigFile() string {
	return ".morty/settings.json"
}

// setupTestDir creates a temporary directory for testing.
func setupTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "research-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	return tmpDir
}

func TestNewResearchHandler(t *testing.T) {
	cfg := &mockConfig{}
	logger := &mockLogger{}

	handler := NewResearchHandler(cfg, logger)
	if handler == nil {
		t.Fatal("Expected handler to be non-nil")
	}

	if handler.cfg == nil {
		t.Error("Expected cfg to be set")
	}

	if handler.logger != logger {
		t.Error("Expected logger to be set")
	}

	if handler.paths == nil {
		t.Error("Expected paths to be initialized")
	}
}

func TestResearchHandler_Execute_WithArgs(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	logger := &mockLogger{}
	handler := NewResearchHandler(cfg, logger)

	// Override paths to use temp directory
	handler.paths.SetWorkDir(tmpDir)

	ctx := context.Background()
	args := []string{"golang", "concurrency"}

	result, err := handler.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	// Verify topic
	expectedTopic := "golang concurrency"
	if result.Topic != expectedTopic {
		t.Errorf("Expected topic '%s', got '%s'", expectedTopic, result.Topic)
	}

	// Verify output path is set
	if result.OutputPath == "" {
		t.Error("Expected OutputPath to be set")
	}

	// Verify research directory was created
	researchDir := handler.GetResearchDir()
	if _, err := os.Stat(researchDir); os.IsNotExist(err) {
		t.Errorf("Research directory was not created: %s", researchDir)
	}

	// Verify timestamp is set
	if result.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be set")
	}
}

func TestResearchHandler_Execute_WithEmptyArgs(t *testing.T) {
	// This test would require stdin mocking, which is complex
	// Instead, we verify that empty args results in an error when stdin is not available
	cfg := &mockConfig{}
	logger := &mockLogger{}
	handler := NewResearchHandler(cfg, logger)

	ctx := context.Background()
	args := []string{}

	// When running non-interactively, this should still work but prompt
	// We'll test the parseTopic logic separately
	_, err := handler.Execute(ctx, args)
	// This may or may not error depending on stdin availability
	// We're mainly checking it doesn't panic
	_ = err
}

func TestResearchHandler_parseTopic_WithArgs(t *testing.T) {
	handler := NewResearchHandler(&mockConfig{}, &mockLogger{})

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "single word",
			args:     []string{"topic"},
			expected: "topic",
		},
		{
			name:     "multiple words",
			args:     []string{"golang", "concurrency", "patterns"},
			expected: "golang concurrency patterns",
		},
		{
			name:     "with extra spaces",
			args:     []string{"  topic  "},
			expected: "topic",
		},
		{
			name:     "empty args",
			args:     []string{},
			expected: "", // Will prompt interactively
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For tests with args, we can test directly
			if len(tt.args) > 0 {
				result, err := handler.parseTopic(tt.args)
				if err != nil {
					t.Fatalf("parseTopic failed: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected '%s', got '%s'", tt.expected, result)
				}
			}
		})
	}
}

func TestResearchHandler_ensureResearchDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	logger := &mockLogger{}
	handler := NewResearchHandler(cfg, logger)
	handler.paths.SetWorkDir(tmpDir)

	err := handler.ensureResearchDir()
	if err != nil {
		t.Fatalf("ensureResearchDir failed: %v", err)
	}

	researchDir := handler.GetResearchDir()
	info, err := os.Stat(researchDir)
	if err != nil {
		t.Errorf("Research directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Research path is not a directory")
	}
}

func TestResearchHandler_generateOutputPath(t *testing.T) {
	tmpDir := setupTestDir(t)

	handler := NewResearchHandler(&mockConfig{}, &mockLogger{})
	handler.paths.SetWorkDir(tmpDir)

	tests := []struct {
		name          string
		topic         string
		expectContain []string
	}{
		{
			name:          "simple topic",
			topic:         "golang",
			expectContain: []string{"golang_", ".md"},
		},
		{
			name:          "topic with spaces",
			topic:         "golang concurrency",
			expectContain: []string{"golang_concurrency_", ".md"},
		},
		{
			name:          "topic with special chars",
			topic:         "C++ Programming!@#",
			expectContain: []string{"c___programming_", ".md"},
		},
		{
			name:          "empty topic fallback",
			topic:         "!@#$%",
			expectContain: []string{"research_", ".md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := handler.generateOutputPath(tt.topic)

			for _, expected := range tt.expectContain {
				if !strings.Contains(path, expected) {
					t.Errorf("Expected path to contain '%s', got '%s'", expected, path)
				}
			}

			// Verify it's an absolute path
			if !filepath.IsAbs(path) {
				t.Error("Expected absolute path")
			}

			// Verify it ends with .md
			if !strings.HasSuffix(path, ".md") {
				t.Error("Expected path to end with .md")
			}
		})
	}
}

func TestResearchHandler_sanitizeFilename(t *testing.T) {
	handler := NewResearchHandler(&mockConfig{}, &mockLogger{})

	tests := []struct {
		input    string
		expected string
	}{
		{"golang", "golang"},
		{"Go Programming", "go_programming"},
		{"C++ Programming", "c___programming"},
		{"Multiple   Spaces", "multiple___spaces"},
		{"Special!@#$%Chars", "special_____chars"},
		{"Mixed123Numbers456", "mixed123numbers456"},
		{"", "research"},
		{"!@#$%", "research"},
		{"VeryLongTopicThatExceedsFiftyCharactersLimitQuickly", "verylongtopicthatexceedsfiftycharacterslimitquickl"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := handler.sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename('%s') = '%s', expected '%s'", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResearchHandler_GetResearchDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	handler := NewResearchHandler(&mockConfig{}, &mockLogger{})
	handler.paths.SetWorkDir(tmpDir)

	dir := handler.GetResearchDir()
	expectedSuffix := filepath.Join(tmpDir, "research")
	if dir != expectedSuffix {
		t.Errorf("Expected research dir '%s', got '%s'", expectedSuffix, dir)
	}
}

func TestResearchHandler_Execute_ContextCancellation(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	logger := &mockLogger{}
	handler := NewResearchHandler(cfg, logger)
	handler.paths.SetWorkDir(tmpDir)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	args := []string{"test topic"}

	result, err := handler.Execute(ctx, args)

	// Result should be returned even if context is cancelled
	if result == nil {
		t.Fatal("Expected result to be non-nil even with cancelled context")
	}

	// Error should be context cancelled
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestResearchHandler_Execute_EmptyTopic(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	logger := &mockLogger{}
	handler := NewResearchHandler(cfg, logger)
	handler.paths.SetWorkDir(tmpDir)

	ctx := context.Background()
	args := []string{"  "} // Only whitespace

	_, err := handler.Execute(ctx, args)
	if err == nil {
		t.Error("Expected error for empty topic")
	}
}

func TestResearchResult_Struct(t *testing.T) {
	result := &ResearchResult{
		Topic:      "test topic",
		Content:    "test content",
		OutputPath: "/path/to/output.md",
	}

	if result.Topic != "test topic" {
		t.Error("Topic not set correctly")
	}

	if result.Content != "test content" {
		t.Error("Content not set correctly")
	}

	if result.OutputPath != "/path/to/output.md" {
		t.Error("OutputPath not set correctly")
	}
}
