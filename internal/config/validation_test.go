package config

import (
	"strings"
	"testing"
)

// TestValidationError tests the ValidationError type.
func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "test.field",
		Message: "test message",
	}

	want := "validation error for field 'test.field': test message"
	if got := err.Error(); got != want {
		t.Errorf("ValidationError.Error() = %q, want %q", got, want)
	}
}

// TestValidationErrors tests the ValidationErrors type.
func TestValidationErrors(t *testing.T) {
	t.Run("empty errors", func(t *testing.T) {
		errs := &ValidationErrors{Errors: []error{}}
		if errs.HasErrors() {
			t.Error("HasErrors() should be false for empty errors")
		}
		want := "no validation errors"
		if got := errs.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("single error", func(t *testing.T) {
		errs := &ValidationErrors{
			Errors: []error{
				&ValidationError{Field: "field1", Message: "error1"},
			},
		}
		if !errs.HasErrors() {
			t.Error("HasErrors() should be true for non-empty errors")
		}
		want := "validation error for field 'field1': error1"
		if got := errs.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		errs := &ValidationErrors{
			Errors: []error{
				&ValidationError{Field: "field1", Message: "error1"},
				&ValidationError{Field: "field2", Message: "error2"},
			},
		}
		if !errs.HasErrors() {
			t.Error("HasErrors() should be true for multiple errors")
		}
		got := errs.Error()
		if !strings.Contains(got, "2 validation errors") {
			t.Errorf("Error() should contain '2 validation errors', got: %q", got)
		}
		if !strings.Contains(got, "field1") {
			t.Errorf("Error() should contain 'field1', got: %q", got)
		}
		if !strings.Contains(got, "field2") {
			t.Errorf("Error() should contain 'field2', got: %q", got)
		}
	})
}

// TestConfigValidatorValidate tests the Validate method.
func TestConfigValidatorValidate(t *testing.T) {
	validator := NewConfigValidator()

	t.Run("nil config", func(t *testing.T) {
		err := validator.Validate(nil)
		if err == nil {
			t.Error("Validate(nil) should return error")
		}
	})

	t.Run("valid config", func(t *testing.T) {
		cfg := DefaultConfig()
		err := validator.Validate(cfg)
		if err != nil {
			t.Errorf("Validate(valid config) should not return error, got: %v", err)
		}
	})

	t.Run("invalid version", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Version = "1.0"
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("Validate() should return error for invalid version")
		}
	})

	t.Run("empty version", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Version = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("Validate() should return error for empty version")
		}
	})
}

// TestValidateVersion tests version validation.
func TestValidateVersion(t *testing.T) {
	validator := NewConfigValidator()

	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"valid 2.0", "2.0", false},
		{"invalid 1.0", "1.0", true},
		{"empty", "", true},
		{"invalid string", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Version = tt.version
			err := validator.Validate(cfg)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestValidateAICli tests AI CLI validation.
func TestValidateAICli(t *testing.T) {
	validator := NewConfigValidator()

	t.Run("empty command", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.AICli.Command = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for empty command")
		}
	})

	t.Run("empty env_var", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.AICli.EnvVar = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for empty env_var")
		}
	})

	t.Run("invalid default_timeout", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.AICli.DefaultTimeout = "invalid"
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for invalid default_timeout")
		}
	})

	t.Run("invalid max_timeout", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.AICli.MaxTimeout = "invalid"
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for invalid max_timeout")
		}
	})

	t.Run("invalid output_format", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.AICli.OutputFormat = "xml"
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for invalid output_format")
		}
	})

	t.Run("valid output formats", func(t *testing.T) {
		validFormats := []string{"json", "text", ""}
		for _, format := range validFormats {
			cfg := DefaultConfig()
			cfg.AICli.OutputFormat = format
			if err := validator.Validate(cfg); err != nil {
				t.Errorf("unexpected error for format %q: %v", format, err)
			}
		}
	})
}

// TestValidateExecution tests execution validation.
func TestValidateExecution(t *testing.T) {
	validator := NewConfigValidator()

	t.Run("negative max_retry_count", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Execution.MaxRetryCount = -1
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for negative max_retry_count")
		}
	})

	t.Run("zero max_retry_count", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Execution.MaxRetryCount = 0
		if err := validator.Validate(cfg); err != nil {
			t.Errorf("unexpected error for zero max_retry_count: %v", err)
		}
	})

	t.Run("zero parallel_jobs", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Execution.ParallelJobs = 0
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for zero parallel_jobs")
		}
	})

	t.Run("negative parallel_jobs", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Execution.ParallelJobs = -1
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for negative parallel_jobs")
		}
	})
}

// TestValidateLogging tests logging validation.
func TestValidateLogging(t *testing.T) {
	validator := NewConfigValidator()

	t.Run("invalid level", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Logging.Level = "invalid"
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for invalid level")
		}
	})

	t.Run("valid levels", func(t *testing.T) {
		validLevels := []string{"debug", "info", "warn", "error", "DEBUG", "INFO", ""}
		for _, level := range validLevels {
			cfg := DefaultConfig()
			cfg.Logging.Level = level
			if err := validator.Validate(cfg); err != nil {
				t.Errorf("unexpected error for level %q: %v", level, err)
			}
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Logging.Format = "yaml"
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for invalid format")
		}
	})

	t.Run("invalid output", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Logging.Output = "stderr"
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for invalid output")
		}
	})

	t.Run("file enabled without path", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Logging.File.Enabled = true
		cfg.Logging.File.Path = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for enabled file without path")
		}
	})

	t.Run("invalid max_size", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Logging.File.MaxSize = "100"
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for invalid max_size")
		}
	})

	t.Run("negative max_backups", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Logging.File.MaxBackups = -1
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for negative max_backups")
		}
	})

	t.Run("negative max_age", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Logging.File.MaxAge = -1
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for negative max_age")
		}
	})
}

// TestValidateState tests state validation.
func TestValidateState(t *testing.T) {
	validator := NewConfigValidator()

	t.Run("empty file", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.State.File = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for empty file")
		}
	})

	t.Run("invalid save_interval", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.State.SaveInterval = "invalid"
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for invalid save_interval")
		}
	})
}

// TestValidateGit tests git validation.
func TestValidateGit(t *testing.T) {
	validator := NewConfigValidator()

	t.Run("empty commit_prefix", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Git.CommitPrefix = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for empty commit_prefix")
		}
	})
}

// TestValidatePlan tests plan validation.
func TestValidatePlan(t *testing.T) {
	validator := NewConfigValidator()

	t.Run("empty dir", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Plan.Dir = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for empty dir")
		}
	})

	t.Run("empty file_extension", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Plan.FileExtension = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for empty file_extension")
		}
	})
}

// TestValidatePrompts tests prompts validation.
func TestValidatePrompts(t *testing.T) {
	validator := NewConfigValidator()

	t.Run("empty dir", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Prompts.Dir = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for empty dir")
		}
	})

	t.Run("empty research", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Prompts.Research = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for empty research")
		}
	})

	t.Run("empty plan", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Prompts.Plan = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for empty plan")
		}
	})

	t.Run("empty doing", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Prompts.Doing = ""
		err := validator.Validate(cfg)
		if err == nil {
			t.Error("expected error for empty doing")
		}
	})
}

// TestValidateSizeString tests the validateSizeString function.
func TestValidateSizeString(t *testing.T) {
	tests := []struct {
		name    string
		size    string
		wantErr bool
	}{
		{"valid KB", "100KB", false},
		{"valid MB", "10MB", false},
		{"valid GB", "1GB", false},
		{"valid TB", "1TB", false},
		{"valid B", "100B", false},
		{"valid lowercase", "10mb", false},
		{"invalid format", "100", true},
		{"invalid unit", "10XB", true},
		{"zero", "0MB", true},
		{"negative", "-10MB", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSizeString(tt.size)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestIsValid tests the IsValid helper function.
func TestIsValid(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := DefaultConfig()
		if !IsValid(cfg) {
			t.Error("IsValid(valid config) should return true")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Version = "invalid"
		if IsValid(cfg) {
			t.Error("IsValid(invalid config) should return false")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		if IsValid(nil) {
			t.Error("IsValid(nil) should return false")
		}
	})
}

// TestValidateField tests the ValidateField function.
func TestValidateField(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		value      interface{}
		wantErr    bool
		errContain string
	}{
		{"valid version", "version", "2.0", false, ""},
		{"empty version", "version", "", true, "non-empty"},
		{"invalid version type", "version", 123, true, "non-empty"},
		{"valid command", "ai_cli.command", "claude", false, ""},
		{"empty command", "ai_cli.command", "", true, "non-empty"},
		{"valid duration", "ai_cli.default_timeout", "10m", false, ""},
		{"invalid duration", "ai_cli.default_timeout", "invalid", true, "invalid duration"},
		{"valid log level", "logging.level", "info", false, ""},
		{"invalid log level", "logging.level", "trace", true, "invalid log level"},
		{"valid retry count", "execution.max_retry_count", 5, false, ""},
		{"negative retry count", "execution.max_retry_count", -1, true, ">= 0"},
		{"unknown field", "unknown.field", "value", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			err := ValidateField(cfg, tt.key, tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && tt.errContain != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error should contain %q, got: %q", tt.errContain, err.Error())
				}
			}
		})
	}
}

// BenchmarkValidate benchmarks validation performance.
func BenchmarkValidate(b *testing.B) {
	validator := NewConfigValidator()
	cfg := DefaultConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.Validate(cfg)
	}
}

// BenchmarkIsValid benchmarks the IsValid helper.
func BenchmarkIsValid(b *testing.B) {
	cfg := DefaultConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsValid(cfg)
	}
}
