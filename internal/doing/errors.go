// Package doing provides job execution functionality with error handling and retry mechanisms.
package doing

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCategory represents the category of an error.
type ErrorCategory int

const (
	// ErrorCategoryUnknown represents an unknown error category.
	ErrorCategoryUnknown ErrorCategory = iota
	// ErrorCategoryPrerequisite represents a prerequisite not met error.
	ErrorCategoryPrerequisite
	// ErrorCategoryPlan represents a plan file related error.
	ErrorCategoryPlan
	// ErrorCategoryExecution represents a job execution error.
	ErrorCategoryExecution
	// ErrorCategoryGit represents a git operation error.
	ErrorCategoryGit
	// ErrorCategoryState represents a state management error.
	ErrorCategoryState
	// ErrorCategoryConfig represents a configuration error.
	ErrorCategoryConfig
	// ErrorCategoryTransient represents a transient error that may succeed on retry.
	ErrorCategoryTransient
)

// String returns the string representation of the error category.
func (c ErrorCategory) String() string {
	switch c {
	case ErrorCategoryPrerequisite:
		return "Prerequisite"
	case ErrorCategoryPlan:
		return "Plan"
	case ErrorCategoryExecution:
		return "Execution"
	case ErrorCategoryGit:
		return "Git"
	case ErrorCategoryState:
		return "State"
	case ErrorCategoryConfig:
		return "Config"
	case ErrorCategoryTransient:
		return "Transient"
	default:
		return "Unknown"
	}
}

// ErrorSeverity represents the severity level of an error.
type ErrorSeverity int

const (
	// SeverityInfo represents an informational message.
	SeverityInfo ErrorSeverity = iota
	// SeverityWarning represents a warning that doesn't prevent operation.
	SeverityWarning
	// SeverityError represents a recoverable error.
	SeverityError
	// SeverityFatal represents a fatal error that stops execution.
	SeverityFatal
)

// String returns the string representation of the error severity.
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// DoingError represents a structured error in the doing package.
type DoingError struct {
	Category   ErrorCategory
	Severity   ErrorSeverity
	Message    string
	Cause      error
	Retryable  bool
	Context    map[string]interface{}
}

// Error implements the error interface.
func (e *DoingError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Category.String(), e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Category.String(), e.Message)
}

// Unwrap returns the underlying cause of the error.
func (e *DoingError) Unwrap() error {
	return e.Cause
}

// IsRetryable returns whether the error is retryable.
func (e *DoingError) IsRetryable() bool {
	return e.Retryable
}

// NewDoingError creates a new DoingError.
func NewDoingError(category ErrorCategory, message string, cause error) *DoingError {
	return &DoingError{
		Category:  category,
		Message:   message,
		Cause:     cause,
		Severity:  SeverityError,
		Retryable: isCategoryRetryable(category),
		Context:   make(map[string]interface{}),
	}
}

// NewDoingErrorWithSeverity creates a new DoingError with specified severity.
func NewDoingErrorWithSeverity(category ErrorCategory, severity ErrorSeverity, message string, cause error) *DoingError {
	return &DoingError{
		Category:  category,
		Severity:  severity,
		Message:   message,
		Cause:     cause,
		Retryable: isCategoryRetryable(category),
		Context:   make(map[string]interface{}),
	}
}

// WithContext adds context information to the error.
func (e *DoingError) WithContext(key string, value interface{}) *DoingError {
	e.Context[key] = value
	return e
}

// isCategoryRetryable determines if an error category is generally retryable.
func isCategoryRetryable(category ErrorCategory) bool {
	switch category {
	case ErrorCategoryTransient, ErrorCategoryExecution:
		return true
	case ErrorCategoryPrerequisite, ErrorCategoryPlan, ErrorCategoryConfig:
		return false
	case ErrorCategoryGit, ErrorCategoryState:
		return true // Some git/state errors might be transient
	default:
		return false
	}
}

// Common error variables for identification.
var (
	// ErrPrerequisiteNotMet is returned when prerequisites are not satisfied.
	ErrPrerequisiteNotMet = errors.New("prerequisites not met")

	// ErrPlanNotFound is returned when a plan file is not found.
	ErrPlanNotFound = errors.New("plan file not found")

	// ErrPlanInvalid is returned when a plan file is invalid.
	ErrPlanInvalid = errors.New("invalid plan file")

	// ErrJobNotFound is returned when a job is not found.
	ErrJobNotFound = errors.New("job not found")

	// ErrModuleNotFound is returned when a module is not found.
	ErrModuleNotFound = errors.New("module not found")

	// ErrExecutionTimeout is returned when job execution times out.
	ErrExecutionTimeout = errors.New("execution timeout")

	// ErrExecutionFailed is returned when job execution fails.
	ErrExecutionFailed = errors.New("execution failed")

	// ErrMaxRetriesExceeded is returned when max retries are exceeded.
	ErrMaxRetriesExceeded = errors.New("maximum retries exceeded")

	// ErrGitNotInitialized is returned when git is not initialized.
	ErrGitNotInitialized = errors.New("git not initialized")

	// ErrGitCommitFailed is returned when git commit fails.
	ErrGitCommitFailed = errors.New("git commit failed")

	// ErrStateCorrupted is returned when state file is corrupted.
	ErrStateCorrupted = errors.New("state file corrupted")

	// ErrStateNotFound is returned when state file is not found.
	ErrStateNotFound = errors.New("state file not found")
)

// ClassifyError classifies a standard error into a DoingError.
// Task 2: Implement error classification
func ClassifyError(err error) *DoingError {
	if err == nil {
		return nil
	}

	// Check if already a DoingError
	var doingErr *DoingError
	if errors.As(err, &doingErr) {
		return doingErr
	}

	errStr := err.Error()
	lowerErrStr := strings.ToLower(errStr)

	// Classify based on error message patterns
	switch {
	// Prerequisite errors
	case strings.Contains(lowerErrStr, "prerequisite") ||
		strings.Contains(lowerErrStr, "前置条件"):
		return NewDoingError(ErrorCategoryPrerequisite, "前置条件不满足", err).
			WithContext("error_type", "prerequisite")

	// Plan file errors
	case strings.Contains(lowerErrStr, "plan") &&
		(strings.Contains(lowerErrStr, "not found") ||
			strings.Contains(lowerErrStr, "不存在")):
		return NewDoingError(ErrorCategoryPlan, "计划文件不存在", err).
			WithContext("error_type", "plan_not_found")

	case strings.Contains(lowerErrStr, "plan") &&
		(strings.Contains(lowerErrStr, "invalid") ||
			strings.Contains(lowerErrStr, "解析失败")):
		return NewDoingError(ErrorCategoryPlan, "计划文件格式无效", err).
			WithContext("error_type", "plan_invalid")

	// Job/Module not found
	case strings.Contains(lowerErrStr, "job") &&
		(strings.Contains(lowerErrStr, "not found") ||
			strings.Contains(lowerErrStr, "不存在")):
		return NewDoingError(ErrorCategoryPlan, "Job 不存在", err).
			WithContext("error_type", "job_not_found")

	case strings.Contains(lowerErrStr, "module") &&
		(strings.Contains(lowerErrStr, "not found") ||
			strings.Contains(lowerErrStr, "不存在")):
		return NewDoingError(ErrorCategoryState, "模块不存在", err).
			WithContext("error_type", "module_not_found")

	// Execution errors
	case strings.Contains(lowerErrStr, "timeout") ||
		strings.Contains(lowerErrStr, "timed out") ||
		strings.Contains(lowerErrStr, "超时"):
		return NewDoingError(ErrorCategoryTransient, "执行超时", err).
			WithContext("error_type", "timeout")

	case strings.Contains(lowerErrStr, "execution") &&
		(strings.Contains(lowerErrStr, "fail") ||
			strings.Contains(lowerErrStr, "失败")):
		return NewDoingError(ErrorCategoryExecution, "执行失败", err).
			WithContext("error_type", "execution_failed")

	// Git errors
	case strings.Contains(lowerErrStr, "git"):
		return classifyGitError(err)

	// State errors
	case strings.Contains(lowerErrStr, "state") ||
		strings.Contains(lowerErrStr, "状态"):
		return classifyStateError(err)

	// Config errors
	case strings.Contains(lowerErrStr, "config") ||
		strings.Contains(lowerErrStr, "配置"):
		return NewDoingError(ErrorCategoryConfig, "配置错误", err).
			WithContext("error_type", "config_error")

	// Connection/transient errors
	case strings.Contains(lowerErrStr, "connection") ||
		strings.Contains(lowerErrStr, "network") ||
		strings.Contains(lowerErrStr, "temporary") ||
		strings.Contains(lowerErrStr, "连接"):
		return NewDoingError(ErrorCategoryTransient, "网络连接错误", err).
			WithContext("error_type", "network_error")

	// Default to execution error
	default:
		return NewDoingError(ErrorCategoryExecution, "执行过程中发生错误", err).
			WithContext("error_type", "unknown")
	}
}

// classifyGitError specifically classifies git-related errors.
func classifyGitError(err error) *DoingError {
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "not a git repo") ||
		strings.Contains(errStr, "not initialized"):
		return NewDoingError(ErrorCategoryGit, "Git 仓库未初始化", err).
			WithContext("error_type", "git_not_initialized").
			WithContext("retryable", false)

	case strings.Contains(errStr, "commit") &&
		(strings.Contains(errStr, "fail") || strings.Contains(errStr, "error")):
		return NewDoingError(ErrorCategoryGit, "Git 提交失败", err).
			WithContext("error_type", "git_commit_failed")

	case strings.Contains(errStr, "authentication") ||
		strings.Contains(errStr, "permission") ||
		strings.Contains(errStr, "权限"):
		return NewDoingError(ErrorCategoryGit, "Git 权限错误", err).
			WithContext("error_type", "git_permission").
			WithContext("retryable", false)

	default:
		return NewDoingError(ErrorCategoryGit, "Git 操作错误", err).
			WithContext("error_type", "git_error")
	}
}

// classifyStateError specifically classifies state-related errors.
func classifyStateError(err error) *DoingError {
	errStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errStr, "corrupt") ||
		strings.Contains(errStr, "invalid") ||
		strings.Contains(errStr, "解析"):
		return NewDoingError(ErrorCategoryState, "状态文件损坏", err).
			WithContext("error_type", "state_corrupted").
			WithContext("recovery_suggestion", "删除 .morty/status.json 后重试")

	case strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "不存在"):
		return NewDoingError(ErrorCategoryState, "状态文件不存在", err).
			WithContext("error_type", "state_not_found").
			WithContext("retryable", true)

	default:
		return NewDoingError(ErrorCategoryState, "状态管理错误", err).
			WithContext("error_type", "state_error")
	}
}

// IsRetryableError checks if an error is retryable.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var doingErr *DoingError
	if errors.As(err, &doingErr) {
		return doingErr.IsRetryable()
	}

	// Check for known transient errors
	if errors.Is(err, ErrExecutionTimeout) {
		return true
	}

	// Classify and check
	classified := ClassifyError(err)
	return classified.IsRetryable()
}

// GetErrorCategory extracts the error category from an error.
func GetErrorCategory(err error) ErrorCategory {
	if err == nil {
		return ErrorCategoryUnknown
	}

	var doingErr *DoingError
	if errors.As(err, &doingErr) {
		return doingErr.Category
	}

	classified := ClassifyError(err)
	return classified.Category
}
