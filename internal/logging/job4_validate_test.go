// Package logging provides validation tests for job_4 requirements.
package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/morty/morty/internal/config"
)

// TestValidation_TextFormatContainsRequiredFields 验证文本格式输出包含时间、级别、消息、属性
func TestValidation_TextFormatContainsRequiredFields(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewTextFormatter(false, time.RFC3339)
	entry := &LogEntry{
		Time:       time.Now(),
		Level:      InfoLevel,
		Message:    "test message",
		Module:     "test-module",
		Job:        "test-job",
		Attributes: []Attr{String("key1", "value1")},
	}
	formatter.Format(&buf, entry)
	output := buf.String()

	checks := map[string]bool{
		"时间":   strings.Contains(output, "T") || strings.Contains(output, "202"),
		"级别":   strings.Contains(output, "INFO"),
		"消息":   strings.Contains(output, "test message"),
		"模块":   strings.Contains(output, "test-module"),
		"属性":   strings.Contains(output, "key1=value1"),
	}

	for name, found := range checks {
		if !found {
			t.Errorf("文本格式输出缺少 %s: %s", name, output)
		}
	}
}

// TestValidation_JSONFormatIsValidJSON 验证 JSON 格式输出是有效的 JSON 对象
func TestValidation_JSONFormatIsValidJSON(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewJSONFormatter()
	entry := &LogEntry{
		Time:       time.Now(),
		Level:      InfoLevel,
		Message:    "json test",
		Module:     "test-module",
		Job:        "test-job",
		Attributes: []Attr{String("key", "value"), Int("count", 42)},
	}
	formatter.Format(&buf, entry)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("JSON 格式输出不是有效的 JSON: %v\nOutput: %s", err, buf.String())
	}

	// 验证必需字段
	if result["level"] != "INFO" {
		t.Errorf("JSON 中 level 字段不正确: %v", result["level"])
	}
	if result["msg"] != "json test" {
		t.Errorf("JSON 中 msg 字段不正确: %v", result["msg"])
	}
	if result["module"] != "test-module" {
		t.Errorf("JSON 中 module 字段不正确: %v", result["module"])
	}
	if result["key"] != "value" {
		t.Errorf("JSON 中 key 字段不正确: %v", result["key"])
	}
	if result["count"] != float64(42) {
		t.Errorf("JSON 中 count 字段不正确: %v", result["count"])
	}
}

// TestValidation_MultiOutputTarget 验证同时输出到控制台和文件
func TestValidation_MultiOutputTarget(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := tmpDir + "/test.log"

	cfg := &config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "both",
		File: config.FileConfig{
			Enabled: true,
			Path:    logFile,
		},
	}

	logger, closer, err := NewLoggerFromConfig(cfg)
	if err != nil {
		t.Fatalf("创建 logger 失败: %v", err)
	}
	defer closer.Close()

	logger.Info("test message for multi output")
	closer.Close()

	// 检查文件是否创建并包含日志
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}

	if !strings.Contains(string(data), "test message for multi output") {
		t.Errorf("文件中没有找到日志内容: %s", string(data))
	}
}

// TestValidation_ConfigFileControlsLogging 验证配置文件正确控制日志行为
func TestValidation_ConfigFileControlsLogging(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.LoggingConfig
		wantLevel    Level
		wantFormat   Format
		wantOutput   OutputTarget
	}{
		{
			name: "warn level json format",
			cfg: &config.LoggingConfig{
				Level:  "warn",
				Format: "json",
				Output: "stdout",
			},
			wantLevel:  WarnLevel,
			wantFormat: FormatJSON,
			wantOutput: OutputStdout,
		},
		{
			name: "debug level text format file output",
			cfg: &config.LoggingConfig{
				Level:  "debug",
				Format: "text",
				Output: "file",
				File:   config.FileConfig{Enabled: true},
			},
			wantLevel:  DebugLevel,
			wantFormat: FormatText,
			wantOutput: OutputFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatConfig := FormatConfigFromLoggingConfig(tt.cfg)

			if formatConfig.Level != tt.wantLevel {
				t.Errorf("Level = %v, want %v", formatConfig.Level, tt.wantLevel)
			}
			if formatConfig.Format != tt.wantFormat {
				t.Errorf("Format = %v, want %v", formatConfig.Format, tt.wantFormat)
			}
			if formatConfig.Output != tt.wantOutput {
				t.Errorf("Output = %v, want %v", formatConfig.Output, tt.wantOutput)
			}
		})
	}
}

// TestValidation_EnvironmentDefaultFormat 验证开发环境默认文本格式，生产环境默认 JSON 格式
func TestValidation_EnvironmentDefaultFormat(t *testing.T) {
	tests := []struct {
		env      Environment
		expected Format
	}{
		{EnvDevelopment, FormatText},
		{EnvProduction, FormatJSON},
		{EnvTesting, FormatText},
	}

	for _, tt := range tests {
		t.Run(tt.env.String(), func(t *testing.T) {
			format := tt.env.DefaultFormat()
			if format != tt.expected {
				t.Errorf("%s 环境默认格式 = %v, want %v", tt.env.String(), format, tt.expected)
			}
		})
	}
}

// TestValidation_FormatterLoggerImplementsInterface 验证 FormatterLogger 实现 Logger 接口
func TestValidation_FormatterLoggerImplementsInterface(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewTextFormatter(false, time.RFC3339)
	logger := NewFormatterLogger(formatter, &buf, DebugLevel)

	// 测试所有日志级别
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
	logger.Success("success message")
	logger.Loop("loop message")

	output := buf.String()
	expectedLevels := []string{"DEBUG", "INFO", "WARN", "ERROR", "INFO", "DEBUG"}
	messages := []string{"debug message", "info message", "warn message", "error message", "success message", "loop message"}

	for i, level := range expectedLevels {
		if !strings.Contains(output, level) {
			t.Errorf("输出缺少级别 %s", level)
		}
		if !strings.Contains(output, messages[i]) {
			t.Errorf("输出缺少消息 %s", messages[i])
		}
	}
}

// TestValidation_LevelFiltering 验证日志级别过滤功能
func TestValidation_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewTextFormatter(false, time.RFC3339)
	logger := NewFormatterLogger(formatter, &buf, WarnLevel)

	// Info 应该被过滤
	logger.Info("info message")
	if buf.Len() > 0 {
		t.Error("Info 消息应该被过滤掉")
	}

	// Warn 和 Error 应该通过
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	if !strings.Contains(output, "warn message") {
		t.Error("Warn 消息应该被记录")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error 消息应该被记录")
	}
}

// TestValidation_EnvironmentDetection 验证环境检测功能
func TestValidation_EnvironmentDetection(t *testing.T) {
	// 验证环境类型存在
	envs := []Environment{EnvDevelopment, EnvProduction, EnvTesting}
	for _, env := range envs {
		if env.String() == "" {
			t.Errorf("环境 %v 的 String() 返回空", env)
		}
	}

	// 验证检测函数
	env := DetectEnvironment()
	if env != EnvDevelopment && env != EnvProduction && env != EnvTesting {
		t.Errorf("DetectEnvironment() 返回了未知环境: %v", env)
	}

	// 验证环境信息
	info := GetEnvironmentInfo()
	if info["environment"] == nil {
		t.Error("GetEnvironmentInfo() 缺少 environment 字段")
	}
}

// TestValidation_WithJobAndContext 验证 WithJob 和 WithContext 功能
func TestValidation_WithJobAndContext(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewTextFormatter(false, time.RFC3339)
	logger := NewFormatterLogger(formatter, &buf, DebugLevel)

	// 测试 WithJob
	jobLogger := logger.WithJob("my-module", "my-job")
	jobLogger.Info("job message")

	output := buf.String()
	if !strings.Contains(output, "my-module/my-job") {
		t.Errorf("WithJob 输出应该包含模块/任务信息: %s", output)
	}

	// 测试 WithContext
	buf.Reset()
	baseCtx := context.Background()
	ctx := ContextWithModule(ContextWithJob(ContextWithLoop(baseCtx, 1), "ctx-job"), "ctx-module")
	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info("context message")

	output = buf.String()
	if !strings.Contains(output, "ctx-module/ctx-job") {
		t.Errorf("WithContext 输出应该包含模块/任务信息: %s", output)
	}
}

// TestValidation_AutoSelectFunctions 验证自动选择函数
func TestValidation_AutoSelectFunctions(t *testing.T) {
	// 测试 AutoSelectFormat
	if f := AutoSelectFormat("json"); f != FormatJSON {
		t.Errorf("AutoSelectFormat('json') = %v, want %v", f, FormatJSON)
	}
	if f := AutoSelectFormat("text"); f != FormatText {
		t.Errorf("AutoSelectFormat('text') = %v, want %v", f, FormatText)
	}

	// 测试 AutoSelectOutput
	if o := AutoSelectOutput("stdout", true); o != OutputStdout {
		t.Errorf("AutoSelectOutput('stdout') = %v, want %v", o, OutputStdout)
	}
	if o := AutoSelectOutput("file", true); o != OutputFile {
		t.Errorf("AutoSelectOutput('file') = %v, want %v", o, OutputFile)
	}
	if o := AutoSelectOutput("both", true); o != OutputBoth {
		t.Errorf("AutoSelectOutput('both') = %v, want %v", o, OutputBoth)
	}
}
