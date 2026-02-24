package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewPaths(t *testing.T) {
	p := NewPaths()
	if p == nil {
		t.Fatal("NewPaths() returned nil")
	}
	if p.workDir != DefaultWorkDir {
		t.Errorf("expected workDir to be %s, got %s", DefaultWorkDir, p.workDir)
	}
}

func TestNewPathsWithLoader(t *testing.T) {
	loader := NewLoader()
	p := NewPathsWithLoader(loader)
	if p == nil {
		t.Fatal("NewPathsWithLoader() returned nil")
	}
	if p.loader != loader {
		t.Error("expected loader to be set")
	}
}

func TestGetWorkDir(t *testing.T) {
	tests := []struct {
		name     string
		workDir  string
		expected string
	}{
		{
			name:     "default work dir",
			workDir:  "",
			expected: DefaultWorkDir,
		},
		{
			name:     "custom work dir",
			workDir:  ".custom",
			expected: ".custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPaths()
			if tt.workDir != "" {
				p.SetWorkDir(tt.workDir)
			}
			got := p.GetWorkDir()
			if !strings.Contains(got, tt.expected) {
				t.Errorf("GetWorkDir() = %v, should contain %v", got, tt.expected)
			}
			// Verify it's an absolute path
			if !filepath.IsAbs(got) {
				t.Errorf("GetWorkDir() = %v, should be absolute path", got)
			}
		})
	}
}

func TestGetLogDir(t *testing.T) {
	p := NewPaths()
	got := p.GetLogDir()

	// Should contain workDir/doing/logs
	if !strings.Contains(got, "doing") || !strings.Contains(got, "logs") {
		t.Errorf("GetLogDir() = %v, should contain 'doing' and 'logs'", got)
	}
	// Verify it's an absolute path
	if !filepath.IsAbs(got) {
		t.Errorf("GetLogDir() = %v, should be absolute path", got)
	}
}

func TestGetResearchDir(t *testing.T) {
	p := NewPaths()
	got := p.GetResearchDir()

	// Should contain workDir/research
	if !strings.Contains(got, "research") {
		t.Errorf("GetResearchDir() = %v, should contain 'research'", got)
	}
	// Verify it's an absolute path
	if !filepath.IsAbs(got) {
		t.Errorf("GetResearchDir() = %v, should be absolute path", got)
	}
}

func TestGetPlanDir(t *testing.T) {
	t.Run("without loader", func(t *testing.T) {
		p := NewPaths()
		got := p.GetPlanDir()

		// Should return default plan dir
		if !strings.Contains(got, "plan") {
			t.Errorf("GetPlanDir() = %v, should contain 'plan'", got)
		}
		// Verify it's an absolute path
		if !filepath.IsAbs(got) {
			t.Errorf("GetPlanDir() = %v, should be absolute path", got)
		}
	})

	t.Run("with loader and custom config", func(t *testing.T) {
		loader := NewLoader()
		loader.config.Plan.Dir = ".custom/plan"
		p := NewPathsWithLoader(loader)
		got := p.GetPlanDir()

		if !strings.Contains(got, ".custom") || !strings.Contains(got, "plan") {
			t.Errorf("GetPlanDir() = %v, should contain '.custom' and 'plan'", got)
		}
	})
}

func TestGetStatusFile(t *testing.T) {
	t.Run("without loader", func(t *testing.T) {
		p := NewPaths()
		got := p.GetStatusFile()

		// Should return default status file
		if !strings.Contains(got, "status.json") {
			t.Errorf("GetStatusFile() = %v, should contain 'status.json'", got)
		}
		// Verify it's an absolute path
		if !filepath.IsAbs(got) {
			t.Errorf("GetStatusFile() = %v, should be absolute path", got)
		}
	})

	t.Run("with loader and custom config", func(t *testing.T) {
		loader := NewLoader()
		loader.config.State.File = ".custom/status.json"
		p := NewPathsWithLoader(loader)
		got := p.GetStatusFile()

		if !strings.Contains(got, ".custom") || !strings.Contains(got, "status.json") {
			t.Errorf("GetStatusFile() = %v, should contain '.custom' and 'status.json'", got)
		}
	})
}

func TestGetConfigFile(t *testing.T) {
	t.Run("without config file set", func(t *testing.T) {
		p := NewPaths()
		got := p.GetConfigFile()

		// Should return default project config file
		if !strings.Contains(got, "settings.json") {
			t.Errorf("GetConfigFile() = %v, should contain 'settings.json'", got)
		}
		// Verify it's an absolute path
		if !filepath.IsAbs(got) {
			t.Errorf("GetConfigFile() = %v, should be absolute path", got)
		}
	})

	t.Run("with config file set", func(t *testing.T) {
		p := NewPaths()
		p.SetConfigFile("custom/config.json")
		got := p.GetConfigFile()

		if !strings.Contains(got, "custom") || !strings.Contains(got, "config.json") {
			t.Errorf("GetConfigFile() = %v, should contain 'custom' and 'config.json'", got)
		}
	})
}

func TestEnsureDir(t *testing.T) {
	t.Run("create new directory", func(t *testing.T) {
		p := NewPaths()
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "test", "nested", "dir")

		err := p.EnsureDir(testDir)
		if err != nil {
			t.Errorf("EnsureDir() error = %v", err)
		}

		// Verify directory exists
		if _, err := os.Stat(testDir); os.IsNotExist(err) {
			t.Error("EnsureDir() did not create directory")
		}
	})

	t.Run("directory already exists", func(t *testing.T) {
		p := NewPaths()
		tempDir := t.TempDir()

		// Create directory first
		os.MkdirAll(tempDir, 0755)

		err := p.EnsureDir(tempDir)
		if err != nil {
			t.Errorf("EnsureDir() error = %v", err)
		}
	})
}

func TestEnsureWorkDir(t *testing.T) {
	p := NewPaths()
	tempDir := t.TempDir()
	p.SetWorkDir(filepath.Join(tempDir, ".morty"))

	err := p.EnsureWorkDir()
	if err != nil {
		t.Errorf("EnsureWorkDir() error = %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(p.GetWorkDir()); os.IsNotExist(err) {
		t.Error("EnsureWorkDir() did not create directory")
	}
}

func TestEnsureLogDir(t *testing.T) {
	p := NewPaths()
	tempDir := t.TempDir()
	p.SetWorkDir(filepath.Join(tempDir, ".morty"))

	err := p.EnsureLogDir()
	if err != nil {
		t.Errorf("EnsureLogDir() error = %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(p.GetLogDir()); os.IsNotExist(err) {
		t.Error("EnsureLogDir() did not create directory")
	}
}

func TestEnsureResearchDir(t *testing.T) {
	p := NewPaths()
	tempDir := t.TempDir()
	p.SetWorkDir(filepath.Join(tempDir, ".morty"))

	err := p.EnsureResearchDir()
	if err != nil {
		t.Errorf("EnsureResearchDir() error = %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(p.GetResearchDir()); os.IsNotExist(err) {
		t.Error("EnsureResearchDir() did not create directory")
	}
}

func TestEnsurePlanDir(t *testing.T) {
	p := NewPaths()
	tempDir := t.TempDir()
	p.SetWorkDir(filepath.Join(tempDir, ".morty"))

	err := p.EnsurePlanDir()
	if err != nil {
		t.Errorf("EnsurePlanDir() error = %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(p.GetPlanDir()); os.IsNotExist(err) {
		t.Error("EnsurePlanDir() did not create directory")
	}
}

func TestEnsureAllDirs(t *testing.T) {
	p := NewPaths()
	tempDir := t.TempDir()
	p.SetWorkDir(filepath.Join(tempDir, ".morty-test"))

	err := p.EnsureAllDirs()
	if err != nil {
		t.Errorf("EnsureAllDirs() error = %v", err)
	}

	// Verify all directories exist
	dirs := []string{
		p.GetWorkDir(),
		p.GetLogDir(),
		p.GetResearchDir(),
		p.GetPlanDir(),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("EnsureAllDirs() did not create directory: %s", dir)
		}
	}
}

func TestPathExists(t *testing.T) {
	p := NewPaths()
	tempDir := t.TempDir()

	t.Run("existing path", func(t *testing.T) {
		if !p.PathExists(tempDir) {
			t.Error("PathExists() should return true for existing path")
		}
	})

	t.Run("non-existing path", func(t *testing.T) {
		nonExisting := filepath.Join(tempDir, "does-not-exist")
		if p.PathExists(nonExisting) {
			t.Error("PathExists() should return false for non-existing path")
		}
	})
}

func TestIsDir(t *testing.T) {
	p := NewPaths()
	tempDir := t.TempDir()

	t.Run("directory", func(t *testing.T) {
		if !p.IsDir(tempDir) {
			t.Error("IsDir() should return true for directory")
		}
	})

	t.Run("file", func(t *testing.T) {
		tempFile := filepath.Join(tempDir, "test.txt")
		os.WriteFile(tempFile, []byte("test"), 0644)
		if p.IsDir(tempFile) {
			t.Error("IsDir() should return false for file")
		}
	})

	t.Run("non-existing", func(t *testing.T) {
		nonExisting := filepath.Join(tempDir, "does-not-exist")
		if p.IsDir(nonExisting) {
			t.Error("IsDir() should return false for non-existing path")
		}
	})
}

func TestGetAbsolutePath(t *testing.T) {
	p := NewPaths()

	tests := []struct {
		name     string
		path     string
		isAbs    bool
	}{
		{
			name:     "relative path",
			path:     ".morty/test",
			isAbs:    true,
		},
		{
			name:     "absolute path",
			path:     "/tmp/.morty/test",
			isAbs:    true,
		},
		{
			name:     "empty path",
			path:     "",
			isAbs:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.GetAbsolutePath(tt.path)
			if tt.isAbs && got != "" && !filepath.IsAbs(got) {
				t.Errorf("GetAbsolutePath() = %v, should be absolute", got)
			}
		})
	}
}

func TestJoin(t *testing.T) {
	p := NewPaths()

	got := p.Join("a", "b", "c")
	expected := filepath.Join("a", "b", "c")

	if got != expected {
		t.Errorf("Join() = %v, want %v", got, expected)
	}
}

func TestSanitizePath(t *testing.T) {
	p := NewPaths()
	tempDir := t.TempDir()
	p.SetWorkDir(tempDir)
	workDir := p.GetWorkDir()

	tests := []struct {
		name         string
		path         string
		shouldBeWork bool // if true, should return workDir (outside attempt)
	}{
		{
			name:         "clean relative path within workDir",
			path:         "test/file.txt",
			shouldBeWork: false,
		},
		{
			name:         "path with dots within workDir",
			path:         "dir/../file.txt",
			shouldBeWork: false,
		},
		{
			name:         "path attempting to escape",
			path:         "../escape.txt",
			shouldBeWork: true, // Should return workDir when trying to escape
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.SanitizePath(tt.path)
			// Result should always be an absolute path
			if !filepath.IsAbs(got) {
				t.Errorf("SanitizePath() = %v, should be absolute", got)
			}
			// If trying to escape, should return workDir
			if tt.shouldBeWork && got != workDir {
				// Check that it's within workDir
				if !strings.HasPrefix(got, workDir) {
					t.Errorf("SanitizePath() = %v, should be within workDir %v", got, workDir)
				}
			}
		})
	}
}

func TestGetPromptsDir(t *testing.T) {
	t.Run("without loader", func(t *testing.T) {
		p := NewPaths()
		got := p.GetPromptsDir()

		// Should return default prompts dir
		if !strings.Contains(got, "prompts") {
			t.Errorf("GetPromptsDir() = %v, should contain 'prompts'", got)
		}
	})

	t.Run("with loader and custom config", func(t *testing.T) {
		loader := NewLoader()
		loader.config.Prompts.Dir = ".custom/prompts"
		p := NewPathsWithLoader(loader)
		got := p.GetPromptsDir()

		if !strings.Contains(got, ".custom") || !strings.Contains(got, "prompts") {
			t.Errorf("GetPromptsDir() = %v, should contain '.custom' and 'prompts'", got)
		}
	})
}

func TestEnsurePromptsDir(t *testing.T) {
	p := NewPaths()
	tempDir := t.TempDir()
	p.SetWorkDir(tempDir)

	err := p.EnsurePromptsDir()
	if err != nil {
		t.Errorf("EnsurePromptsDir() error = %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(p.GetPromptsDir()); os.IsNotExist(err) {
		t.Error("EnsurePromptsDir() did not create directory")
	}
}

func TestPathsWithSpecialCharacters(t *testing.T) {
	// Skip on Windows for certain special characters
	if runtime.GOOS == "windows" {
		t.Skip("Skipping special character tests on Windows")
	}

	p := NewPaths()
	tempDir := t.TempDir()

	// Test with paths containing spaces
	t.Run("spaces in path", func(t *testing.T) {
		specialDir := filepath.Join(tempDir, "path with spaces", ".morty")
		p.SetWorkDir(specialDir)

		got := p.GetWorkDir()
		if !strings.Contains(got, "path with spaces") {
			t.Errorf("GetWorkDir() = %v, should handle spaces", got)
		}
	})
}

func TestResolvePath(t *testing.T) {
	p := NewPaths()

	t.Run("empty path", func(t *testing.T) {
		got := p.resolvePath("")
		if got != "" {
			t.Errorf("resolvePath(\"\") = %v, want empty string", got)
		}
	})

	t.Run("path with tilde expansion", func(t *testing.T) {
		got := p.resolvePath("~/test")
		home, _ := os.UserHomeDir()
		if !strings.Contains(got, home) {
			t.Errorf("resolvePath(\"~/test\") = %v, should contain home dir", got)
		}
	})
}

func TestPathsIntegration(t *testing.T) {
	// Integration test to verify all paths work together
	p := NewPaths()
	tempDir := t.TempDir()
	p.SetWorkDir(filepath.Join(tempDir, ".morty-integration"))

	// Ensure all directories
	if err := p.EnsureAllDirs(); err != nil {
		t.Fatalf("EnsureAllDirs() failed: %v", err)
	}

	// Verify workDir-based paths are within workDir
	workDir := p.GetWorkDir()
	workDirBasedPaths := map[string]string{
		"log":      p.GetLogDir(),
		"research": p.GetResearchDir(),
	}

	for name, path := range workDirBasedPaths {
		if !strings.HasPrefix(path, workDir) {
			t.Errorf("%s dir (%s) should be within workDir (%s)", name, path, workDir)
		}
		if !p.PathExists(path) {
			t.Errorf("%s dir (%s) should exist", name, path)
		}
	}

	// Plan dir uses default when no loader config is set
	// It may not be within workDir, but should be absolute
	planDir := p.GetPlanDir()
	if !filepath.IsAbs(planDir) {
		t.Error("GetPlanDir() should return absolute path")
	}

	// Verify status file path
	statusFile := p.GetStatusFile()
	if !filepath.IsAbs(statusFile) {
		t.Error("GetStatusFile() should return absolute path")
	}

	// Verify config file path
	configFile := p.GetConfigFile()
	if !filepath.IsAbs(configFile) {
		t.Error("GetConfigFile() should return absolute path")
	}
}
