package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPlanHandler(t *testing.T) {
	cfg := &mockConfig{}
	logger := &mockLogger{}

	handler := NewPlanHandler(cfg, logger, nil)

	if handler == nil {
		t.Fatal("NewPlanHandler returned nil")
	}

	if handler.cfg == nil {
		t.Error("Handler cfg not set correctly")
	}

	if handler.logger == nil {
		t.Error("Handler logger not set")
	}

	if handler.paths == nil {
		t.Error("Handler paths not initialized")
	}
}

func TestPlanHandler_parseOptions(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantForce     bool
		wantModule    string
		wantRemaining []string
	}{
		{
			name:          "no options",
			args:          []string{},
			wantForce:     false,
			wantModule:    "",
			wantRemaining: nil,
		},
		{
			name:          "force flag long",
			args:          []string{"--force"},
			wantForce:     true,
			wantModule:    "",
			wantRemaining: nil,
		},
		{
			name:          "force flag short",
			args:          []string{"-f"},
			wantForce:     true,
			wantModule:    "",
			wantRemaining: nil,
		},
		{
			name:          "force flag with value true",
			args:          []string{"--force=true"},
			wantForce:     true,
			wantModule:    "",
			wantRemaining: nil,
		},
		{
			name:          "force flag with value 1",
			args:          []string{"--force=1"},
			wantForce:     true,
			wantModule:    "",
			wantRemaining: nil,
		},
		{
			name:          "force flag with value false",
			args:          []string{"--force=false"},
			wantForce:     false,
			wantModule:    "",
			wantRemaining: nil,
		},
		{
			name:          "module flag long",
			args:          []string{"--module", "my-module"},
			wantForce:     false,
			wantModule:    "my-module",
			wantRemaining: nil,
		},
		{
			name:          "module flag short",
			args:          []string{"-m", "my-module"},
			wantForce:     false,
			wantModule:    "my-module",
			wantRemaining: nil,
		},
		{
			name:          "module flag with equals",
			args:          []string{"--module=my-module"},
			wantForce:     false,
			wantModule:    "my-module",
			wantRemaining: nil,
		},
		{
			name:          "combined flags",
			args:          []string{"--force", "--module", "my-module"},
			wantForce:     true,
			wantModule:    "my-module",
			wantRemaining: nil,
		},
		{
			name:          "remaining args",
			args:          []string{"job1", "job2", "job3"},
			wantForce:     false,
			wantModule:    "",
			wantRemaining: []string{"job1", "job2", "job3"},
		},
		{
			name:          "mixed args and flags",
			args:          []string{"--force", "job1", "--module", "my-module", "job2"},
			wantForce:     true,
			wantModule:    "my-module",
			wantRemaining: []string{"job1", "job2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewPlanHandler(&mockConfig{}, &mockLogger{}, nil)
			force, module, remaining := handler.parseOptions(tt.args)

			if force != tt.wantForce {
				t.Errorf("parseOptions() force = %v, want %v", force, tt.wantForce)
			}
			if module != tt.wantModule {
				t.Errorf("parseOptions() module = %v, want %v", module, tt.wantModule)
			}
			if len(remaining) != len(tt.wantRemaining) {
				t.Errorf("parseOptions() remaining length = %d, want %d", len(remaining), len(tt.wantRemaining))
			}
			for i := range remaining {
				if i < len(tt.wantRemaining) && remaining[i] != tt.wantRemaining[i] {
					t.Errorf("parseOptions() remaining[%d] = %v, want %v", i, remaining[i], tt.wantRemaining[i])
				}
			}
		})
	}
}

func TestPlanHandler_inferModuleName(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	tests := []struct {
		name     string
		dirName  string
		expected string
	}{
		{
			name:     "normal directory",
			dirName:  "my-project",
			expected: "my_project",
		},
		{
			name:     "directory with spaces",
			dirName:  "my project",
			expected: "my_project",
		},
		{
			name:     "directory with special chars",
			dirName:  "my-project-v1.0",
			expected: "my_project_v1_0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupTestDir(t)
			testDir := filepath.Join(tmpDir, tt.dirName)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test dir: %v", err)
			}
			if err := os.Chdir(testDir); err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}

			handler := NewPlanHandler(&mockConfig{}, &mockLogger{}, nil)
			moduleName := handler.inferModuleName()

			if moduleName != tt.expected {
				t.Errorf("inferModuleName() = %v, want %v", moduleName, tt.expected)
			}
		})
	}
}

func TestSanitizeModuleName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-project", "my_project"},
		{"my project", "my_project"},
		{"my-project-v1.0", "my_project_v1_0"},
		{"MyProject", "myproject"},
		{"123-project", "123_project"},
		{"project_with_underscores", "project_with_underscores"},
		{"", "default"},
		{"___", "default"},
		{"a", "a"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeModuleName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeModuleName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPlanHandler_ensurePlanDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewPlanHandler(cfg, &mockLogger{}, nil)

	err := handler.ensurePlanDir()
	if err != nil {
		t.Errorf("ensurePlanDir() error = %v", err)
	}

	// Check that the directory was created
	planDir := filepath.Join(tmpDir, "plan")
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		t.Error("Plan directory was not created")
	}
}

func TestPlanHandler_generatePlanPath(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewPlanHandler(cfg, &mockLogger{}, nil)

	planPath := handler.generatePlanPath("my-module")
	expectedPath := filepath.Join(tmpDir, "plan", "my_module.md")

	if planPath != expectedPath {
		t.Errorf("generatePlanPath() = %v, want %v", planPath, expectedPath)
	}
}

func TestPlanHandler_planFileExists(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewPlanHandler(cfg, &mockLogger{}, nil)

	planPath := filepath.Join(tmpDir, "plan", "test.md")

	// Initially should not exist
	if handler.planFileExists(planPath) {
		t.Error("planFileExists() should return false for non-existent file")
	}

	// Create the directory and file
	os.MkdirAll(filepath.Dir(planPath), 0755)
	os.WriteFile(planPath, []byte("test"), 0644)

	// Now should exist
	if !handler.planFileExists(planPath) {
		t.Error("planFileExists() should return true for existing file")
	}
}

func TestPlanHandler_createPlanFile(t *testing.T) {
	tmpDir := setupTestDir(t)

	handler := NewPlanHandler(&mockConfig{}, &mockLogger{}, nil)
	planPath := filepath.Join(tmpDir, "test.md")

	err := handler.createPlanFile(planPath, "test-module", []string{"job1", "job2"})
	if err != nil {
		t.Fatalf("createPlanFile() error = %v", err)
	}

	// Check file exists
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		t.Fatal("Plan file was not created")
	}

	// Check content
	content, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("Failed to read plan file: %v", err)
	}

	contentStr := string(content)

	// Verify expected content
	if !strings.Contains(contentStr, "# Plan: test-module") {
		t.Error("Plan file missing title")
	}
	if !strings.Contains(contentStr, "Module: test-module") {
		t.Error("Plan file missing module name")
	}
	if !strings.Contains(contentStr, "### Job 1: job1") {
		t.Error("Plan file missing job1")
	}
	if !strings.Contains(contentStr, "### Job 2: job2") {
		t.Error("Plan file missing job2")
	}
}

func TestPlanHandler_createPlanFile_defaultJobs(t *testing.T) {
	tmpDir := setupTestDir(t)

	handler := NewPlanHandler(&mockConfig{}, &mockLogger{}, nil)
	planPath := filepath.Join(tmpDir, "test.md")

	err := handler.createPlanFile(planPath, "test-module", nil)
	if err != nil {
		t.Fatalf("createPlanFile() error = %v", err)
	}

	content, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("Failed to read plan file: %v", err)
	}

	contentStr := string(content)

	// Should have default job template
	if !strings.Contains(contentStr, "### Job 1: Initial Setup") {
		t.Error("Plan file missing default job")
	}
}

func TestPlanHandler_Execute_newPlan(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	logger := &mockLogger{}
	handler := NewPlanHandler(cfg, logger, nil)

	ctx := context.Background()
	result, err := handler.Execute(ctx, []string{"--module", "test-module"})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if result == nil {
		t.Fatal("Execute() returned nil result")
	}

	if result.Err != nil {
		t.Errorf("Execute() result.Err = %v", result.Err)
	}

	if result.ModuleName != "test-module" {
		t.Errorf("Execute() module name = %v, want test-module", result.ModuleName)
	}

	if result.Overwritten {
		t.Error("Execute() should not have overwritten for new plan")
	}

	// Check file was created
	if _, err := os.Stat(result.PlanPath); os.IsNotExist(err) {
		t.Error("Plan file was not created")
	}
}

func TestPlanHandler_Execute_withForceFlag(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	logger := &mockLogger{}
	handler := NewPlanHandler(cfg, logger, nil)

	ctx := context.Background()

	// First create a plan
	_, err := handler.Execute(ctx, []string{"--module", "test-module"})
	if err != nil {
		t.Fatalf("First Execute() error = %v", err)
	}

	// Now overwrite with force flag
	result, err := handler.Execute(ctx, []string{"--force", "--module", "test-module"})
	if err != nil {
		t.Errorf("Execute() with force error = %v", err)
	}

	if !result.Overwritten {
		t.Error("Execute() with force should mark as overwritten")
	}
}

func TestPlanHandler_Execute_withShortForceFlag(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	logger := &mockLogger{}
	handler := NewPlanHandler(cfg, logger, nil)

	ctx := context.Background()

	// Create a plan first
	_, err := handler.Execute(ctx, []string{"--module", "test-module"})
	if err != nil {
		t.Fatalf("First Execute() error = %v", err)
	}

	// Overwrite with short force flag
	result, err := handler.Execute(ctx, []string{"-f", "--module", "test-module"})
	if err != nil {
		t.Errorf("Execute() with -f error = %v", err)
	}

	if !result.Overwritten {
		t.Error("Execute() with -f should mark as overwritten")
	}
}

func TestPlanHandler_Execute_inferModuleName(t *testing.T) {
	tmpDir := setupTestDir(t)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to temp directory so module name is inferred from it
	testDir := filepath.Join(tmpDir, "my-project")
	os.MkdirAll(testDir, 0755)
	os.Chdir(testDir)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	logger := &mockLogger{}
	handler := NewPlanHandler(cfg, logger, nil)

	ctx := context.Background()
	result, err := handler.Execute(ctx, nil)

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if result.ModuleName != "my_project" {
		t.Errorf("Execute() inferred module name = %v, want my_project", result.ModuleName)
	}
}

func TestPlanHandler_Execute_createsPlanDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	logger := &mockLogger{}
	handler := NewPlanHandler(cfg, logger, nil)

	ctx := context.Background()
	_, err := handler.Execute(ctx, []string{"--module", "test"})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	// Check plan directory was created
	planDir := filepath.Join(tmpDir, "plan")
	if _, err := os.Stat(planDir); os.IsNotExist(err) {
		t.Error("Plan directory was not created")
	}
}

func TestPlanHandler_Execute_returnsCorrectPlanResult(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	logger := &mockLogger{}
	handler := NewPlanHandler(cfg, logger, nil)

	ctx := context.Background()
	result, err := handler.Execute(ctx, []string{"--module", "test-module"})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.PlanPath == "" {
		t.Error("Execute() returned empty PlanPath")
	}

	if result.ModuleName != "test-module" {
		t.Errorf("Execute() returned wrong ModuleName: %v", result.ModuleName)
	}

	if result.ExitCode != 0 {
		t.Errorf("Execute() returned non-zero ExitCode: %v", result.ExitCode)
	}

	if result.CreatedAt.IsZero() {
		t.Error("Execute() returned zero CreatedAt")
	}

	if result.Duration < 0 {
		t.Error("Execute() returned negative Duration")
	}
}

func TestPlanHandler_Execute_contextCancellation(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	logger := &mockLogger{}
	handler := NewPlanHandler(cfg, logger, nil)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := handler.Execute(ctx, []string{"--module", "test"})

	// Should return context error
	if err != context.Canceled {
		t.Errorf("Execute() with cancelled context error = %v, want context.Canceled", err)
	}

	if result == nil {
		t.Fatal("Execute() returned nil result with cancelled context")
	}

	if result.Err != context.Canceled {
		t.Errorf("Execute() result.Err = %v, want context.Canceled", result.Err)
	}
}

func TestPlanHandler_GetPlanDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewPlanHandler(cfg, &mockLogger{}, nil)

	planDir := handler.GetPlanDir()
	expected := filepath.Join(tmpDir, "plan")

	if planDir != expected {
		t.Errorf("GetPlanDir() = %v, want %v", planDir, expected)
	}
}

func TestPlanHandler_Execute_withJobArgs(t *testing.T) {
	tmpDir := setupTestDir(t)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	logger := &mockLogger{}
	handler := NewPlanHandler(cfg, logger, nil)

	ctx := context.Background()
	result, err := handler.Execute(ctx, []string{"--module", "test-module", "setup", "build", "test"})

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Check file content
	content, err := os.ReadFile(result.PlanPath)
	if err != nil {
		t.Fatalf("Failed to read plan file: %v", err)
	}

	contentStr := string(content)

	// Should have jobs from args
	if !strings.Contains(contentStr, "### Job 1: setup") {
		t.Error("Plan file missing setup job")
	}
	if !strings.Contains(contentStr, "### Job 2: build") {
		t.Error("Plan file missing build job")
	}
	if !strings.Contains(contentStr, "### Job 3: test") {
		t.Error("Plan file missing test job")
	}
}

func TestPlanHandler_loadResearchFacts_emptyDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create empty research directory
	researchDir := filepath.Join(tmpDir, "research")
	os.MkdirAll(researchDir, 0755)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewPlanHandler(cfg, &mockLogger{}, nil)

	facts, err := handler.loadResearchFacts()

	if err != nil {
		t.Errorf("loadResearchFacts() error = %v", err)
	}

	if facts == nil {
		t.Error("loadResearchFacts() returned nil, expected empty slice")
	}

	if len(facts) != 0 {
		t.Errorf("loadResearchFacts() returned %d facts, expected 0", len(facts))
	}
}

func TestPlanHandler_loadResearchFacts_noResearchDir(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Remove the research directory
	researchDir := filepath.Join(tmpDir, "research")
	os.RemoveAll(researchDir)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewPlanHandler(cfg, &mockLogger{}, nil)

	facts, err := handler.loadResearchFacts()

	if err != nil {
		t.Errorf("loadResearchFacts() error = %v", err)
	}

	if facts == nil {
		t.Error("loadResearchFacts() returned nil, expected empty slice")
	}

	if len(facts) != 0 {
		t.Errorf("loadResearchFacts() returned %d facts, expected 0", len(facts))
	}
}

func TestPlanHandler_loadResearchFacts_singleFile(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create research directory and a research file
	researchDir := filepath.Join(tmpDir, "research")
	os.MkdirAll(researchDir, 0755)
	content := "# Research Content\nThis is test research content."
	os.WriteFile(filepath.Join(researchDir, "test-research.md"), []byte(content), 0644)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewPlanHandler(cfg, &mockLogger{}, nil)

	facts, err := handler.loadResearchFacts()

	if err != nil {
		t.Fatalf("loadResearchFacts() error = %v", err)
	}

	if len(facts) != 1 {
		t.Fatalf("loadResearchFacts() returned %d facts, expected 1", len(facts))
	}

	// Check formatted content
	expected := "--- test-research.md ---\n" + content
	if facts[0] != expected {
		t.Errorf("loadResearchFacts() fact = %v, want %v", facts[0], expected)
	}
}

func TestPlanHandler_loadResearchFacts_multipleFilesSorted(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create research directory and multiple research files in non-alphabetical order
	researchDir := filepath.Join(tmpDir, "research")
	os.MkdirAll(researchDir, 0755)
	os.WriteFile(filepath.Join(researchDir, "zebra.md"), []byte("Zebra content"), 0644)
	os.WriteFile(filepath.Join(researchDir, "alpha.md"), []byte("Alpha content"), 0644)
	os.WriteFile(filepath.Join(researchDir, "beta.md"), []byte("Beta content"), 0644)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewPlanHandler(cfg, &mockLogger{}, nil)

	facts, err := handler.loadResearchFacts()

	if err != nil {
		t.Fatalf("loadResearchFacts() error = %v", err)
	}

	if len(facts) != 3 {
		t.Fatalf("loadResearchFacts() returned %d facts, expected 3", len(facts))
	}

	// Check that files are sorted alphabetically
	expectedOrder := []string{
		"--- alpha.md ---\nAlpha content",
		"--- beta.md ---\nBeta content",
		"--- zebra.md ---\nZebra content",
	}

	for i, expected := range expectedOrder {
		if facts[i] != expected {
			t.Errorf("loadResearchFacts() fact[%d] = %v, want %v", i, facts[i], expected)
		}
	}
}

func TestPlanHandler_loadResearchFacts_ignoresNonMdFiles(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create research directory and files
	researchDir := filepath.Join(tmpDir, "research")
	os.MkdirAll(researchDir, 0755)
	os.WriteFile(filepath.Join(researchDir, "valid.md"), []byte("Valid content"), 0644)
	os.WriteFile(filepath.Join(researchDir, "invalid.txt"), []byte("Invalid content"), 0644)
	os.WriteFile(filepath.Join(researchDir, "another.json"), []byte(`{"key": "value"}`), 0644)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewPlanHandler(cfg, &mockLogger{}, nil)

	facts, err := handler.loadResearchFacts()

	if err != nil {
		t.Fatalf("loadResearchFacts() error = %v", err)
	}

	if len(facts) != 1 {
		t.Fatalf("loadResearchFacts() returned %d facts, expected 1", len(facts))
	}

	// Should only contain the .md file
	expected := "--- valid.md ---\nValid content"
	if facts[0] != expected {
		t.Errorf("loadResearchFacts() fact = %v, want %v", facts[0], expected)
	}
}

func TestPlanHandler_loadResearchFacts_ignoresSubdirectories(t *testing.T) {
	tmpDir := setupTestDir(t)

	// Create research directory with subdirectory
	researchDir := filepath.Join(tmpDir, "research")
	os.MkdirAll(researchDir, 0755)
	os.WriteFile(filepath.Join(researchDir, "valid.md"), []byte("Valid content"), 0644)
	os.MkdirAll(filepath.Join(researchDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(researchDir, "subdir", "nested.md"), []byte("Nested content"), 0644)

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewPlanHandler(cfg, &mockLogger{}, nil)

	facts, err := handler.loadResearchFacts()

	if err != nil {
		t.Fatalf("loadResearchFacts() error = %v", err)
	}

	if len(facts) != 1 {
		t.Fatalf("loadResearchFacts() returned %d facts, expected 1", len(facts))
	}

	// Should only contain files in root, not subdirectories
	expected := "--- valid.md ---\nValid content"
	if facts[0] != expected {
		t.Errorf("loadResearchFacts() fact = %v, want %v", facts[0], expected)
	}
}

func TestPlanHandler_loadResearchFacts_returnsErrorForUnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root (can read unreadable files)")
	}

	tmpDir := setupTestDir(t)

	// Create unreadable research file
	researchDir := filepath.Join(tmpDir, "research")
	os.MkdirAll(researchDir, 0755)
	filePath := filepath.Join(researchDir, "unreadable.md")
	os.WriteFile(filePath, []byte("Content"), 0644)
	os.Chmod(filePath, 0000)
	defer os.Chmod(filePath, 0644) // Restore permissions for cleanup

	cfg := &mockConfig{}
	cfg.SetWorkDir(tmpDir)
	handler := NewPlanHandler(cfg, &mockLogger{}, nil)

	_, err := handler.loadResearchFacts()

	if err == nil {
		t.Error("loadResearchFacts() expected error for unreadable file, got nil")
	}
}
