package doing

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestErrorCategory_String(t *testing.T) {
	tests := []struct {
		category ErrorCategory
		expected string
	}{
		{ErrorCategoryUnknown, "Unknown"},
		{ErrorCategoryPrerequisite, "Prerequisite"},
		{ErrorCategoryPlan, "Plan"},
		{ErrorCategoryExecution, "Execution"},
		{ErrorCategoryGit, "Git"},
		{ErrorCategoryState, "State"},
		{ErrorCategoryConfig, "Config"},
		{ErrorCategoryTransient, "Transient"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.category.String(); got != tt.expected {
				t.Errorf("ErrorCategory.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorSeverity_String(t *testing.T) {
	tests := []struct {
		severity ErrorSeverity
		expected string
	}{
		{SeverityInfo, "INFO"},
		{SeverityWarning, "WARNING"},
		{SeverityError, "ERROR"},
		{SeverityFatal, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.severity.String(); got != tt.expected {
				t.Errorf("ErrorSeverity.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDoingError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *DoingError
		expected string
	}{
		{
			name:     "with cause",
			err:      NewDoingError(ErrorCategoryExecution, "execution failed", errors.New("timeout")),
			expected: "[Execution] execution failed: timeout",
		},
		{
			name:     "without cause",
			err:      NewDoingError(ErrorCategoryPlan, "plan not found", nil),
			expected: "[Plan] plan not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("DoingError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDoingError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := NewDoingError(ErrorCategoryExecution, "execution failed", cause)

	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("DoingError.Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestDoingError_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		category ErrorCategory
		expected bool
	}{
		{"Prerequisite", ErrorCategoryPrerequisite, false},
		{"Plan", ErrorCategoryPlan, false},
		{"Config", ErrorCategoryConfig, false},
		{"Transient", ErrorCategoryTransient, true},
		{"Execution", ErrorCategoryExecution, true},
		{"Git", ErrorCategoryGit, true},
		{"State", ErrorCategoryState, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewDoingError(tt.category, "test", nil)
			if got := err.IsRetryable(); got != tt.expected {
				t.Errorf("DoingError.IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDoingError_WithContext(t *testing.T) {
	err := NewDoingError(ErrorCategoryExecution, "test", nil).
		WithContext("key1", "value1").
		WithContext("key2", 42)

	if err.Context["key1"] != "value1" {
		t.Errorf("Context key1 = %v, want value1", err.Context["key1"])
	}
	if err.Context["key2"] != 42 {
		t.Errorf("Context key2 = %v, want 42", err.Context["key2"])
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name             string
		err              error
		expectedCategory ErrorCategory
	}{
		{
			name:             "nil error",
			err:              nil,
			expectedCategory: ErrorCategoryUnknown,
		},
		{
			name:             "already DoingError",
			err:              NewDoingError(ErrorCategoryPlan, "test", nil),
			expectedCategory: ErrorCategoryPlan,
		},
		{
			name:             "prerequisite error",
			err:              errors.New("prerequisites not met"),
			expectedCategory: ErrorCategoryPrerequisite,
		},
		{
			name:             "plan not found error",
			err:              errors.New("plan file not found"),
			expectedCategory: ErrorCategoryPlan,
		},
		{
			name:             "job not found error",
			err:              errors.New("job not found in plan"),
			expectedCategory: ErrorCategoryPlan,
		},
		{
			name:             "module not found error",
			err:              errors.New("module not found"),
			expectedCategory: ErrorCategoryState,
		},
		{
			name:             "timeout error",
			err:              errors.New("execution timeout"),
			expectedCategory: ErrorCategoryTransient,
		},
		{
			name:             "execution failed error",
			err:              errors.New("execution failed"),
			expectedCategory: ErrorCategoryExecution,
		},
		{
			name:             "git error",
			err:              errors.New("git commit failed"),
			expectedCategory: ErrorCategoryGit,
		},
		{
			name:             "state error",
			err:              errors.New("state file corrupted"),
			expectedCategory: ErrorCategoryState,
		},
		{
			name:             "config error",
			err:              errors.New("config error"),
			expectedCategory: ErrorCategoryConfig,
		},
		{
			name:             "network error",
			err:              errors.New("connection failed"),
			expectedCategory: ErrorCategoryTransient,
		},
		{
			name:             "unknown error",
			err:              errors.New("something weird happened"),
			expectedCategory: ErrorCategoryExecution,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyError(tt.err)
			if result == nil {
				if tt.expectedCategory != ErrorCategoryUnknown {
					t.Errorf("ClassifyError() = nil, want category %v", tt.expectedCategory)
				}
				return
			}
			if result.Category != tt.expectedCategory {
				t.Errorf("ClassifyError() category = %v, want %v", result.Category, tt.expectedCategory)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "retryable DoingError",
			err:      NewDoingError(ErrorCategoryTransient, "timeout", nil),
			expected: true,
		},
		{
			name:     "non-retryable DoingError",
			err:      NewDoingError(ErrorCategoryPrerequisite, "missing prereq", nil),
			expected: false,
		},
		{
			name:     "timeout error",
			err:      ErrExecutionTimeout,
			expected: true,
		},
		{
			name:     "execution timeout error",
			err:      errors.New("execution timed out"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryableError(tt.err); got != tt.expected {
				t.Errorf("IsRetryableError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetErrorCategory(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCategory
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: ErrorCategoryUnknown,
		},
		{
			name:     "DoingError",
			err:      NewDoingError(ErrorCategoryGit, "test", nil),
			expected: ErrorCategoryGit,
		},
		{
			name:     "regular error",
			err:      errors.New("plan file not found"),
			expected: ErrorCategoryPlan,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorCategory(tt.err); got != tt.expected {
				t.Errorf("GetErrorCategory() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClassifyGitError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType string
		retryable    bool
	}{
		{
			name:         "not initialized",
			err:          errors.New("not a git repo"),
			expectedType: "git_not_initialized",
			retryable:    false,
		},
		{
			name:         "commit failed",
			err:          errors.New("git commit failed"),
			expectedType: "git_commit_failed",
			retryable:    false, // default for git
		},
		{
			name:         "permission error",
			err:          errors.New("authentication failed"),
			expectedType: "git_permission",
			retryable:    false,
		},
		{
			name:         "generic git error",
			err:          errors.New("git error"),
			expectedType: "git_error",
			retryable:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyGitError(tt.err)
			errorType, _ := result.Context["error_type"].(string)
			if errorType != tt.expectedType {
				t.Errorf("classifyGitError() error_type = %v, want %v", errorType, tt.expectedType)
			}
		})
	}
}

func TestClassifyStateError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType string
	}{
		{
			name:         "corrupted",
			err:          errors.New("state file corrupted"),
			expectedType: "state_corrupted",
		},
		{
			name:         "not found",
			err:          errors.New("state not found"),
			expectedType: "state_not_found",
		},
		{
			name:         "generic state error",
			err:          errors.New("state error"),
			expectedType: "state_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyStateError(tt.err)
			errorType, _ := result.Context["error_type"].(string)
			if errorType != tt.expectedType {
				t.Errorf("classifyStateError() error_type = %v, want %v", errorType, tt.expectedType)
			}
		})
	}
}

func TestAsDoingError(t *testing.T) {
	t.Run("direct DoingError", func(t *testing.T) {
		original := NewDoingError(ErrorCategoryExecution, "test", nil)
		var result *DoingError
		if !AsDoingError(original, &result) {
			t.Error("AsDoingError() should return true for direct DoingError")
		}
		if result != original {
			t.Error("AsDoingError() result should match original")
		}
	})

	t.Run("wrapped DoingError", func(t *testing.T) {
		inner := NewDoingError(ErrorCategoryExecution, "inner", nil)
		outer := fmt.Errorf("outer: %w", inner)
		var result *DoingError
		if !AsDoingError(outer, &result) {
			t.Error("AsDoingError() should return true for wrapped DoingError")
		}
		if result != inner {
			t.Error("AsDoingError() result should match inner error")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		var result *DoingError
		if AsDoingError(nil, &result) {
			t.Error("AsDoingError() should return false for nil error")
		}
	})

	t.Run("non-DoingError", func(t *testing.T) {
		var result *DoingError
		if AsDoingError(errors.New("regular error"), &result) {
			t.Error("AsDoingError() should return false for regular error")
		}
	})
}

func TestNewDoingErrorWithSeverity(t *testing.T) {
	err := NewDoingErrorWithSeverity(ErrorCategoryExecution, SeverityFatal, "critical failure", nil)

	if err.Severity != SeverityFatal {
		t.Errorf("Severity = %v, want %v", err.Severity, SeverityFatal)
	}
	if err.Category != ErrorCategoryExecution {
		t.Errorf("Category = %v, want %v", err.Category, ErrorCategoryExecution)
	}
	if err.Message != "critical failure" {
		t.Errorf("Message = %v, want critical failure", err.Message)
	}
}

func TestErrorMessageContainsCategory(t *testing.T) {
	err := NewDoingError(ErrorCategoryPlan, "file not found", nil)
	if !strings.Contains(err.Error(), "Plan") {
		t.Error("Error message should contain category name")
	}
}

func TestErrorMessageContainsCause(t *testing.T) {
	cause := errors.New("underlying issue")
	err := NewDoingError(ErrorCategoryExecution, "operation failed", cause)

	if !strings.Contains(err.Error(), "underlying issue") {
		t.Error("Error message should contain cause")
	}
}
