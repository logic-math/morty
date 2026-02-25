// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/morty/morty/internal/config"
)

// TestAICliCallerInterface verifies that AICliCallerImpl implements AICliCaller interface.
func TestAICliCallerInterface(t *testing.T) {
	var _ AICliCaller = (*AICliCallerImpl)(nil)
}

// TestNewAICliCaller tests the creation of a new AI CLI caller.
func TestNewAICliCaller(t *testing.T) {
	caller := NewAICliCaller()
	if caller == nil {
		t.Fatal("NewAICliCaller() returned nil")
	}

	if caller.config == nil {
		t.Error("config is nil")
	}

	if caller.baseCaller == nil {
		t.Error("baseCaller is nil")
	}
}

// TestNewAICliCallerWithLoader tests creation with config loader.
func TestNewAICliCallerWithLoader(t *testing.T) {
	loader := config.NewLoader()

	caller := NewAICliCallerWithLoader(loader)
	if caller == nil {
		t.Fatal("NewAICliCallerWithLoader() returned nil")
	}

	if caller.config == nil {
		t.Error("config is nil")
	}

	if caller.loader == nil {
		t.Error("loader is nil")
	}
}

// TestAICliCallerImpl_GetCLIPath tests CLI path resolution.
func TestAICliCallerImpl_GetCLIPath(t *testing.T) {
	tests := []struct {
		name       string
		envVarName string
		envValue   string
		configCmd  string
		want       string
	}{
		{
			name:       "from environment variable",
			envVarName: "CLAUDE_CODE_CLI",
			envValue:   "/custom/path/to/ai_cli",
			configCmd:  "ai_cli",
			want:       "/custom/path/to/ai_cli",
		},
		{
			name:       "from config when env not set",
			envVarName: "CLAUDE_CODE_CLI",
			envValue:   "",
			configCmd:  "my_ai_cli",
			want:       "my_ai_cli",
		},
		{
			name:       "custom env var name",
			envVarName: "MY_CUSTOM_CLI",
			envValue:   "/custom/cli",
			configCmd:  "ai_cli",
			want:       "/custom/cli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up env var after test
			if tt.envValue != "" {
				os.Setenv(tt.envVarName, tt.envValue)
				defer os.Unsetenv(tt.envVarName)
			} else {
				os.Unsetenv(tt.envVarName)
			}

			caller := NewAICliCaller()
			caller.config.EnvVar = tt.envVarName
			caller.config.Command = tt.configCmd

			got := caller.GetCLIPath()
			if got != tt.want {
				t.Errorf("GetCLIPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAICliCallerImpl_BuildArgs tests argument building.
func TestAICliCallerImpl_BuildArgs(t *testing.T) {
	tests := []struct {
		name                  string
		defaultArgs           []string
		outputFormat          string
		enableSkipPermissions bool
		want                  []string
	}{
		{
			name:                  "with all options",
			defaultArgs:           []string{"--verbose", "--debug"},
			outputFormat:          "json",
			enableSkipPermissions: true,
			want:                  []string{"--verbose", "--debug", "--output-format", "json", "--dangerously-skip-permissions"},
		},
		{
			name:                  "without skip permissions",
			defaultArgs:           []string{"--verbose"},
			outputFormat:          "text",
			enableSkipPermissions: false,
			want:                  []string{"--verbose", "--output-format", "text"},
		},
		{
			name:                  "no default args",
			defaultArgs:           []string{},
			outputFormat:          "json",
			enableSkipPermissions: true,
			want:                  []string{"--output-format", "json", "--dangerously-skip-permissions"},
		},
		{
			name:                  "empty output format",
			defaultArgs:           []string{"--verbose"},
			outputFormat:          "",
			enableSkipPermissions: false,
			want:                  []string{"--verbose"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caller := NewAICliCaller()
			caller.config.DefaultArgs = tt.defaultArgs
			caller.config.OutputFormat = tt.outputFormat
			caller.config.EnableSkipPermissions = tt.enableSkipPermissions

			got := caller.BuildArgs()

			if len(got) != len(tt.want) {
				t.Errorf("BuildArgs() = %v, want %v", got, tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("BuildArgs()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestAICliCallerImpl_CallWithPrompt tests calling with prompt file.
func TestAICliCallerImpl_CallWithPrompt(t *testing.T) {
	// Create a temporary prompt file
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "test_prompt.md")
	if err := os.WriteFile(promptFile, []byte("test prompt content"), 0644); err != nil {
		t.Fatalf("Failed to create test prompt file: %v", err)
	}

	// Create a mock caller
	mockCaller := &mockCallerImpl{
		result: &Result{
			Stdout:   "success",
			ExitCode: 0,
		},
	}

	caller := NewAICliCaller()
	caller.SetBaseCaller(mockCaller)
	caller.config.Command = "echo"
	caller.config.DefaultArgs = []string{}

	ctx := context.Background()
	result, err := caller.CallWithPrompt(ctx, promptFile)

	if err != nil {
		t.Errorf("CallWithPrompt() error = %v", err)
	}

	if result == nil {
		t.Error("CallWithPrompt() result is nil")
	} else if result.ExitCode != 0 {
		t.Errorf("CallWithPrompt() exit code = %v, want 0", result.ExitCode)
	}

	// Verify that the mock was called with correct parameters
	if mockCaller.lastName != "echo" {
		t.Errorf("Expected command 'echo', got '%s'", mockCaller.lastName)
	}

	// Check that prompt file is in args
	foundPrompt := false
	for _, arg := range mockCaller.lastArgs {
		if arg == promptFile || filepath.Base(arg) == "test_prompt.md" {
			foundPrompt = true
			break
		}
	}
	if !foundPrompt {
		t.Errorf("Prompt file not found in args: %v", mockCaller.lastArgs)
	}
}

// TestAICliCallerImpl_CallWithPromptContent tests calling with prompt content.
func TestAICliCallerImpl_CallWithPromptContent(t *testing.T) {
	// Create a mock caller
	mockCaller := &mockCallerImpl{
		result: &Result{
			Stdout:   "success",
			ExitCode: 0,
		},
	}

	caller := NewAICliCaller()
	caller.SetBaseCaller(mockCaller)
	caller.config.Command = "cat"
	caller.config.DefaultArgs = []string{}

	content := "test prompt content"
	ctx := context.Background()
	result, err := caller.CallWithPromptContent(ctx, content)

	if err != nil {
		t.Errorf("CallWithPromptContent() error = %v", err)
	}

	if result == nil {
		t.Error("CallWithPromptContent() result is nil")
	}

	// Verify that stdin was set correctly
	if mockCaller.lastOpts.Stdin != content {
		t.Errorf("Expected stdin '%s', got '%s'", content, mockCaller.lastOpts.Stdin)
	}
}

// TestAICliCallerImpl_GetConfig tests configuration retrieval.
func TestAICliCallerImpl_GetConfig(t *testing.T) {
	caller := NewAICliCaller()
	cfg := caller.GetConfig()

	if cfg == nil {
		t.Error("GetConfig() returned nil")
	}

	// Verify default values
	if cfg.Command == "" {
		t.Error("Config Command is empty")
	}
}

// TestAICliCallerImpl_SetCLITimeout tests timeout setting.
func TestAICliCallerImpl_SetCLITimeout(t *testing.T) {
	caller := NewAICliCaller()
	timeout := 30 * time.Second

	caller.SetCLITimeout(timeout)

	if caller.baseCaller.GetDefaultTimeout() != timeout {
		t.Errorf("SetCLITimeout() failed, expected %v, got %v",
			timeout, caller.baseCaller.GetDefaultTimeout())
	}
}

// TestNewAICliCallerWithLoader_ConfigValues tests that loader values override defaults.
func TestNewAICliCallerWithLoader_ConfigValues(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{
		"ai_cli": {
			"command": "custom_ai",
			"env_var": "MY_AI_VAR",
			"output_format": "text",
			"enable_skip_permissions": false
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	loader := config.NewLoader()
	if err := loader.Load(configPath); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	caller := NewAICliCallerWithLoader(loader)

	if caller.config.Command != "custom_ai" {
		t.Errorf("Expected command 'custom_ai', got '%s'", caller.config.Command)
	}

	if caller.config.EnvVar != "MY_AI_VAR" {
		t.Errorf("Expected env_var 'MY_AI_VAR', got '%s'", caller.config.EnvVar)
	}

	if caller.config.OutputFormat != "text" {
		t.Errorf("Expected output_format 'text', got '%s'", caller.config.OutputFormat)
	}

	if caller.config.EnableSkipPermissions != false {
		t.Errorf("Expected enable_skip_permissions false, got %v", caller.config.EnableSkipPermissions)
	}
}

// TestAICliCaller_EnvironmentVariablePriority tests that env var takes priority over config.
func TestAICliCaller_EnvironmentVariablePriority(t *testing.T) {
	// Set environment variable
	os.Setenv("CLAUDE_CODE_CLI", "/env/path/ai")
	defer os.Unsetenv("CLAUDE_CODE_CLI")

	// Create caller with custom config
	caller := NewAICliCaller()
	caller.config.Command = "/config/path/ai"
	caller.config.EnvVar = "CLAUDE_CODE_CLI"

	// Should return env var value
	if path := caller.GetCLIPath(); path != "/env/path/ai" {
		t.Errorf("Expected env var path '/env/path/ai', got '%s'", path)
	}
}

// TestAICliCaller_BuildArgsWithVerbose tests verbose flag building.
func TestAICliCaller_BuildArgsWithVerbose(t *testing.T) {
	caller := NewAICliCaller()
	caller.config.DefaultArgs = []string{"--verbose", "--debug"}
	caller.config.OutputFormat = "json"

	args := caller.BuildArgs()

	// Check that verbose is included
	hasVerbose := false
	hasDebug := false
	hasOutputFormat := false

	for i, arg := range args {
		if arg == "--verbose" {
			hasVerbose = true
		}
		if arg == "--debug" {
			hasDebug = true
		}
		if arg == "--output-format" && i+1 < len(args) && args[i+1] == "json" {
			hasOutputFormat = true
		}
	}

	if !hasVerbose {
		t.Error("--verbose flag not found in args")
	}
	if !hasDebug {
		t.Error("--debug flag not found in args")
	}
	if !hasOutputFormat {
		t.Error("--output-format json not found in args")
	}
}

// TestAICliCaller_BuildArgsWithSkipPermissions tests skip permissions flag.
func TestAICliCaller_BuildArgsWithSkipPermissions(t *testing.T) {
	caller := NewAICliCaller()
	caller.config.DefaultArgs = []string{}
	caller.config.EnableSkipPermissions = true

	args := caller.BuildArgs()

	hasSkipPermissions := false
	for _, arg := range args {
		if arg == "--dangerously-skip-permissions" {
			hasSkipPermissions = true
			break
		}
	}

	if !hasSkipPermissions {
		t.Error("--dangerously-skip-permissions flag not found in args")
	}
}

// TestAICliCallerImpl_CallWithPrompt_Timeout tests timeout handling.
func TestAICliCallerImpl_CallWithPrompt_Timeout(t *testing.T) {
	mockCaller := &mockCallerImpl{
		result: &Result{
			Stdout:   "success",
			ExitCode: 0,
		},
	}

	caller := NewAICliCaller()
	caller.SetBaseCaller(mockCaller)
	caller.config.DefaultTimeout = "5s"

	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "test.md")
	os.WriteFile(promptFile, []byte("test"), 0644)

	ctx := context.Background()
	_, err := caller.CallWithPrompt(ctx, promptFile)

	if err != nil {
		t.Errorf("CallWithPrompt() error = %v", err)
	}

	// Verify timeout was passed
	if mockCaller.lastOpts.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", mockCaller.lastOpts.Timeout)
	}
}

// TestAICliCallerImpl_CallWithPromptContent_Timeout tests timeout for content calls.
func TestAICliCallerImpl_CallWithPromptContent_Timeout(t *testing.T) {
	mockCaller := &mockCallerImpl{
		result: &Result{
			Stdout:   "success",
			ExitCode: 0,
		},
	}

	caller := NewAICliCaller()
	caller.SetBaseCaller(mockCaller)
	caller.config.DefaultTimeout = "10s"

	ctx := context.Background()
	_, err := caller.CallWithPromptContent(ctx, "test content")

	if err != nil {
		t.Errorf("CallWithPromptContent() error = %v", err)
	}

	// Verify timeout was passed
	if mockCaller.lastOpts.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", mockCaller.lastOpts.Timeout)
	}
}

// mockCallerImpl is a mock implementation of Caller for testing.
type mockCallerImpl struct {
	result   *Result
	err      error
	lastName string
	lastArgs []string
	lastOpts Options
}

func (m *mockCallerImpl) Call(ctx context.Context, name string, args ...string) (*Result, error) {
	m.lastName = name
	m.lastArgs = args
	return m.result, m.err
}

func (m *mockCallerImpl) CallWithOptions(ctx context.Context, name string, args []string, opts Options) (*Result, error) {
	m.lastName = name
	m.lastArgs = args
	m.lastOpts = opts
	return m.result, m.err
}

func (m *mockCallerImpl) CallWithCtx(ctx context.Context, name string, args []string, opts Options) (CallHandler, error) {
	m.lastName = name
	m.lastArgs = args
	m.lastOpts = opts
	return nil, nil
}

func (m *mockCallerImpl) CallAsync(ctx context.Context, name string, args ...string) (CallHandler, error) {
	m.lastName = name
	m.lastArgs = args
	return nil, nil
}

func (m *mockCallerImpl) CallAsyncWithOptions(ctx context.Context, name string, args []string, opts Options) (CallHandler, error) {
	m.lastName = name
	m.lastArgs = args
	m.lastOpts = opts
	return nil, nil
}

func (m *mockCallerImpl) SetDefaultTimeout(timeout time.Duration) {}
func (m *mockCallerImpl) GetDefaultTimeout() time.Duration        { return 0 }
