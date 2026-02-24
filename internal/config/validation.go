package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Error returns the error message.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// ValidationErrors holds multiple validation errors.
type ValidationErrors struct {
	Errors []error
}

// Error returns the combined error message.
func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "no validation errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	var msgs []string
	for _, err := range e.Errors {
		msgs = append(msgs, err.Error())
	}
	return fmt.Sprintf("%d validation errors:\n- %s", len(e.Errors), strings.Join(msgs, "\n- "))
}

// HasErrors returns true if there are validation errors.
func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// Validator defines the interface for configuration validators.
type Validator interface {
	// Validate checks if the configuration is valid.
	// Returns nil if valid, otherwise returns a ValidationErrors containing all errors.
	Validate(cfg *Config) error
}

// ConfigValidator implements comprehensive configuration validation.
type ConfigValidator struct {
	validators []func(*Config) error
}

// NewConfigValidator creates a new ConfigValidator with all default validators.
func NewConfigValidator() *ConfigValidator {
	v := &ConfigValidator{}
	v.validators = []func(*Config) error{
		v.validateVersion,
		v.validateAICli,
		v.validateExecution,
		v.validateLogging,
		v.validateState,
		v.validateGit,
		v.validatePlan,
		v.validatePrompts,
	}
	return v
}

// Validate runs all validators and returns any errors found.
func (v *ConfigValidator) Validate(cfg *Config) error {
	if cfg == nil {
		return &ValidationError{Field: "config", Message: "config is nil"}
	}

	var errs ValidationErrors
	for _, validator := range v.validators {
		if err := validator(cfg); err != nil {
			errs.Errors = append(errs.Errors, err)
		}
	}

	if errs.HasErrors() {
		return &errs
	}
	return nil
}

// validateVersion checks the version field.
func (v *ConfigValidator) validateVersion(cfg *Config) error {
	if cfg.Version == "" {
		return &ValidationError{Field: "version", Message: "version is required"}
	}
	if cfg.Version != "2.0" {
		return &ValidationError{Field: "version", Message: fmt.Sprintf("unsupported version: %s (expected 2.0)", cfg.Version)}
	}
	return nil
}

// validateAICli validates AI CLI configuration.
func (v *ConfigValidator) validateAICli(cfg *Config) error {
	aiCli := cfg.AICli

	if aiCli.Command == "" {
		return &ValidationError{Field: "ai_cli.command", Message: "command is required"}
	}

	if aiCli.EnvVar == "" {
		return &ValidationError{Field: "ai_cli.env_var", Message: "env_var is required"}
	}

	// Validate timeout format
	if aiCli.DefaultTimeout != "" {
		if _, err := time.ParseDuration(aiCli.DefaultTimeout); err != nil {
			return &ValidationError{Field: "ai_cli.default_timeout", Message: fmt.Sprintf("invalid duration format: %s", aiCli.DefaultTimeout)}
		}
	}

	if aiCli.MaxTimeout != "" {
		if _, err := time.ParseDuration(aiCli.MaxTimeout); err != nil {
			return &ValidationError{Field: "ai_cli.max_timeout", Message: fmt.Sprintf("invalid duration format: %s", aiCli.MaxTimeout)}
		}
	}

	// Validate output format
	validFormats := map[string]bool{"json": true, "text": true}
	if aiCli.OutputFormat != "" && !validFormats[aiCli.OutputFormat] {
		return &ValidationError{Field: "ai_cli.output_format", Message: fmt.Sprintf("invalid output format: %s (must be 'json' or 'text')", aiCli.OutputFormat)}
	}

	return nil
}

// validateExecution validates execution configuration.
func (v *ConfigValidator) validateExecution(cfg *Config) error {
	exec := cfg.Execution

	if exec.MaxRetryCount < 0 {
		return &ValidationError{Field: "execution.max_retry_count", Message: "max_retry_count must be >= 0"}
	}

	if exec.ParallelJobs < 1 {
		return &ValidationError{Field: "execution.parallel_jobs", Message: "parallel_jobs must be >= 1"}
	}

	return nil
}

// validateLogging validates logging configuration.
func (v *ConfigValidator) validateLogging(cfg *Config) error {
	logging := cfg.Logging

	// Validate log level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if logging.Level != "" && !validLevels[strings.ToLower(logging.Level)] {
		return &ValidationError{Field: "logging.level", Message: fmt.Sprintf("invalid log level: %s (must be 'debug', 'info', 'warn', or 'error')", logging.Level)}
	}

	// Validate log format
	validFormats := map[string]bool{"json": true, "text": true}
	if logging.Format != "" && !validFormats[logging.Format] {
		return &ValidationError{Field: "logging.format", Message: fmt.Sprintf("invalid log format: %s (must be 'json' or 'text')", logging.Format)}
	}

	// Validate output destination
	validOutputs := map[string]bool{"stdout": true, "file": true, "both": true}
	if logging.Output != "" && !validOutputs[logging.Output] {
		return &ValidationError{Field: "logging.output", Message: fmt.Sprintf("invalid output destination: %s (must be 'stdout', 'file', or 'both')", logging.Output)}
	}

	// Validate file logging config
	if logging.File.Enabled {
		if logging.File.Path == "" {
			return &ValidationError{Field: "logging.file.path", Message: "path is required when file logging is enabled"}
		}

		if logging.File.MaxSize != "" {
			if err := validateSizeString(logging.File.MaxSize); err != nil {
				return &ValidationError{Field: "logging.file.max_size", Message: err.Error()}
			}
		}

		if logging.File.MaxBackups < 0 {
			return &ValidationError{Field: "logging.file.max_backups", Message: "max_backups must be >= 0"}
		}

		if logging.File.MaxAge < 0 {
			return &ValidationError{Field: "logging.file.max_age", Message: "max_age must be >= 0"}
		}
	}

	return nil
}

// validateState validates state configuration.
func (v *ConfigValidator) validateState(cfg *Config) error {
	state := cfg.State

	if state.File == "" {
		return &ValidationError{Field: "state.file", Message: "file is required"}
	}

	if state.SaveInterval != "" {
		if _, err := time.ParseDuration(state.SaveInterval); err != nil {
			return &ValidationError{Field: "state.save_interval", Message: fmt.Sprintf("invalid duration format: %s", state.SaveInterval)}
		}
	}

	return nil
}

// validateGit validates Git configuration.
func (v *ConfigValidator) validateGit(cfg *Config) error {
	git := cfg.Git

	if git.CommitPrefix == "" {
		return &ValidationError{Field: "git.commit_prefix", Message: "commit_prefix is required"}
	}

	return nil
}

// validatePlan validates plan configuration.
func (v *ConfigValidator) validatePlan(cfg *Config) error {
	plan := cfg.Plan

	if plan.Dir == "" {
		return &ValidationError{Field: "plan.dir", Message: "dir is required"}
	}

	if plan.FileExtension == "" {
		return &ValidationError{Field: "plan.file_extension", Message: "file_extension is required"}
	}

	return nil
}

// validatePrompts validates prompts configuration.
func (v *ConfigValidator) validatePrompts(cfg *Config) error {
	prompts := cfg.Prompts

	if prompts.Dir == "" {
		return &ValidationError{Field: "prompts.dir", Message: "dir is required"}
	}

	if prompts.Research == "" {
		return &ValidationError{Field: "prompts.research", Message: "research is required"}
	}

	if prompts.Plan == "" {
		return &ValidationError{Field: "prompts.plan", Message: "plan is required"}
	}

	if prompts.Doing == "" {
		return &ValidationError{Field: "prompts.doing", Message: "doing is required"}
	}

	return nil
}

// sizePattern matches size strings like "10MB", "1GB", "100KB", etc.
var sizePattern = regexp.MustCompile(`^(\d+)(B|KB|MB|GB|TB)$`)

// validateSizeString validates a size string (e.g., "10MB", "1GB").
func validateSizeString(s string) error {
	if s == "" {
		return fmt.Errorf("size string is empty")
	}

	matches := sizePattern.FindStringSubmatch(strings.ToUpper(s))
	if matches == nil {
		return fmt.Errorf("invalid size format: %s (expected format like '10MB', '1GB')", s)
	}

	size, err := strconv.Atoi(matches[1])
	if err != nil || size <= 0 {
		return fmt.Errorf("size must be a positive number: %s", s)
	}

	return nil
}

// IsValid checks if a configuration is valid without returning detailed errors.
func IsValid(cfg *Config) bool {
	validator := NewConfigValidator()
	return validator.Validate(cfg) == nil
}

// ValidateField validates a single configuration field by key using dot notation.
// Returns nil if the field is valid, otherwise returns a ValidationError.
func ValidateField(cfg *Config, key string, value interface{}) error {
	// This is a simplified implementation that validates common fields
	// Full implementation would use reflection to validate any field

	switch key {
	case "version":
		if v, ok := value.(string); !ok || v == "" {
			return &ValidationError{Field: key, Message: "version must be a non-empty string"}
		}
	case "ai_cli.command":
		if v, ok := value.(string); !ok || v == "" {
			return &ValidationError{Field: key, Message: "command must be a non-empty string"}
		}
	case "ai_cli.default_timeout", "ai_cli.max_timeout", "state.save_interval":
		if v, ok := value.(string); ok && v != "" {
			if _, err := time.ParseDuration(v); err != nil {
				return &ValidationError{Field: key, Message: fmt.Sprintf("invalid duration format: %v", value)}
			}
		}
	case "logging.level":
		if v, ok := value.(string); ok && v != "" {
			validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
			if !validLevels[strings.ToLower(v)] {
				return &ValidationError{Field: key, Message: fmt.Sprintf("invalid log level: %s", v)}
			}
		}
	case "execution.max_retry_count", "execution.parallel_jobs", "logging.file.max_backups", "logging.file.max_age":
		if v, ok := value.(int); ok && v < 0 {
			return &ValidationError{Field: key, Message: "value must be >= 0"}
		}
	}

	return nil
}
