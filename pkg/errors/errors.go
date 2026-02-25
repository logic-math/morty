// Package errors provides Morty's unified error code system and error handling utilities.
// It defines structured errors with codes, messages, modules, and chain support.
package errors

import (
	"errors"
	"fmt"
)

// MortyError is the unified error structure for Morty.
// It provides error codes, messages, module context, cause chains, and detailed information.
type MortyError struct {
	Code    string
	Message string
	Module  string
	Cause   error
	Details map[string]interface{}
}

// Error returns the error message with code and optional cause.
// This implements the error interface.
func (e *MortyError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error (cause), enabling error chain traversal.
// This implements the errors.Unwrap interface.
func (e *MortyError) Unwrap() error {
	return e.Cause
}

// WithDetail adds a key-value detail to the error and returns the error for chaining.
func (e *MortyError) WithDetail(key string, value interface{}) *MortyError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithModule sets the module for the error and returns the error for chaining.
func (e *MortyError) WithModule(module string) *MortyError {
	e.Module = module
	return e
}

// New creates a new MortyError with the given code, message, and optional module.
func New(code, message string) *MortyError {
	return &MortyError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with a MortyError.
// The original error becomes the Cause of the returned MortyError.
func Wrap(err error, code, message string) *MortyError {
	return &MortyError{
		Code:    code,
		Message: message,
		Cause:   err,
		Details: make(map[string]interface{}),
	}
}

// Is checks if an error matches the target MortyError by code.
// It traverses the error chain and checks each MortyError's code.
// If target is nil, it checks if err is nil.
func Is(err error, target *MortyError) bool {
	// Handle nil cases
	if target == nil {
		return err == nil
	}
	if err == nil {
		return false
	}

	// Traverse the error chain, checking each MortyError
	for err != nil {
		if me, ok := err.(*MortyError); ok {
			if me.Code == target.Code {
				return true
			}
		}
		// Unwrap to next error in chain
		err = errors.Unwrap(err)
	}
	return false
}

// AsMortyError attempts to extract a MortyError from the error chain.
func AsMortyError(err error) (*MortyError, bool) {
	var me *MortyError
	if errors.As(err, &me) {
		return me, true
	}
	return nil, false
}

// ============================================================================
// General Errors (M0xxx)
// ============================================================================

var (
	// ErrSuccess indicates a successful operation
	ErrSuccess = &MortyError{Code: "M0000", Message: "success"}
	// ErrGeneral is a general/unspecified error
	ErrGeneral = &MortyError{Code: "M0001", Message: "general error"}
	// ErrInvalidArgs indicates invalid arguments were provided
	ErrInvalidArgs = &MortyError{Code: "M0002", Message: "invalid arguments"}
	// ErrNotFound indicates a resource was not found
	ErrNotFound = &MortyError{Code: "M0003", Message: "not found"}
	// ErrAlreadyExists indicates a resource already exists
	ErrAlreadyExists = &MortyError{Code: "M0004", Message: "already exists"}
	// ErrPermission indicates insufficient permissions
	ErrPermission = &MortyError{Code: "M0005", Message: "permission denied"}
	// ErrTimeout indicates an operation timed out
	ErrTimeout = &MortyError{Code: "M0006", Message: "timeout"}
	// ErrCancelled indicates an operation was cancelled
	ErrCancelled = &MortyError{Code: "M0007", Message: "cancelled"}
	// ErrInterrupted indicates an operation was interrupted
	ErrInterrupted = &MortyError{Code: "M0008", Message: "interrupted"}
	// ErrNotImplemented indicates a feature is not implemented
	ErrNotImplemented = &MortyError{Code: "M0009", Message: "not implemented"}
)

// ============================================================================
// Config Errors (M1xxx)
// ============================================================================

var (
	// ErrConfigNotFound indicates the config file was not found
	ErrConfigNotFound = &MortyError{Code: "M1001", Message: "config file not found", Module: "config"}
	// ErrConfigParse indicates config file parsing failed
	ErrConfigParse = &MortyError{Code: "M1002", Message: "config parse failed", Module: "config"}
	// ErrConfigInvalid indicates an invalid config value
	ErrConfigInvalid = &MortyError{Code: "M1003", Message: "invalid config value", Module: "config"}
	// ErrConfigVersion indicates a config version mismatch
	ErrConfigVersion = &MortyError{Code: "M1004", Message: "config version mismatch", Module: "config"}
	// ErrConfigRequired indicates a required config is missing
	ErrConfigRequired = &MortyError{Code: "M1005", Message: "required config missing", Module: "config"}
)

// ============================================================================
// State Errors (M2xxx)
// ============================================================================

var (
	// ErrStateNotFound indicates the state file was not found
	ErrStateNotFound = &MortyError{Code: "M2001", Message: "state file not found", Module: "state"}
	// ErrStateParse indicates state file parsing failed
	ErrStateParse = &MortyError{Code: "M2002", Message: "state parse failed", Module: "state"}
	// ErrStateCorrupted indicates the state file is corrupted
	ErrStateCorrupted = &MortyError{Code: "M2003", Message: "state file corrupted", Module: "state"}
	// ErrStateTransition indicates an invalid state transition
	ErrStateTransition = &MortyError{Code: "M2004", Message: "invalid state transition", Module: "state"}
	// ErrStateModuleNotFound indicates a module was not found in state
	ErrStateModuleNotFound = &MortyError{Code: "M2005", Message: "module not found in state", Module: "state"}
	// ErrStateJobNotFound indicates a job was not found in state
	ErrStateJobNotFound = &MortyError{Code: "M2006", Message: "job not found in state", Module: "state"}
)

// ============================================================================
// Git Errors (M3xxx)
// ============================================================================

var (
	// ErrGitNotRepo indicates the path is not a git repository
	ErrGitNotRepo = &MortyError{Code: "M3001", Message: "not a git repository", Module: "git"}
	// ErrGitCommit indicates git commit failed
	ErrGitCommit = &MortyError{Code: "M3002", Message: "git commit failed", Module: "git"}
	// ErrGitStatus indicates getting git status failed
	ErrGitStatus = &MortyError{Code: "M3003", Message: "git status failed", Module: "git"}
	// ErrGitDirtyWorktree indicates the worktree is dirty
	ErrGitDirtyWorktree = &MortyError{Code: "M3004", Message: "worktree is dirty", Module: "git"}
	// ErrGitNoCommits indicates no commits were found
	ErrGitNoCommits = &MortyError{Code: "M3005", Message: "no commits found", Module: "git"}
)

// ============================================================================
// Parser Errors (M4xxx)
// ============================================================================

var (
	// ErrParserNotFound indicates no parser was found for the file type
	ErrParserNotFound = &MortyError{Code: "M4001", Message: "parser not found", Module: "parser"}
	// ErrParserFileNotFound indicates the plan file was not found
	ErrParserFileNotFound = &MortyError{Code: "M4002", Message: "plan file not found", Module: "parser"}
	// ErrParserParse indicates parsing failed
	ErrParserParse = &MortyError{Code: "M4003", Message: "parse failed", Module: "parser"}
	// ErrParserInvalidFormat indicates an invalid file format
	ErrParserInvalidFormat = &MortyError{Code: "M4004", Message: "invalid format", Module: "parser"}
	// ErrParserNoJobs indicates no jobs were found in the plan
	ErrParserNoJobs = &MortyError{Code: "M4005", Message: "no jobs found", Module: "parser"}
)

// ============================================================================
// Call CLI Errors (M5xxx)
// ============================================================================

var (
	// ErrCallCLINotFound indicates the AI CLI command was not found
	ErrCallCLINotFound = &MortyError{Code: "M5001", Message: "AI CLI command not found", Module: "callcli"}
	// ErrCallCLIExec indicates execution failed
	ErrCallCLIExec = &MortyError{Code: "M5002", Message: "execution failed", Module: "callcli"}
	// ErrCallCLITimeout indicates the execution timed out
	ErrCallCLITimeout = &MortyError{Code: "M5003", Message: "execution timeout", Module: "callcli"}
	// ErrCallCLIKilled indicates the process was killed
	ErrCallCLIKilled = &MortyError{Code: "M5004", Message: "process killed", Module: "callcli"}
	// ErrCallCLIOutput indicates reading output failed
	ErrCallCLIOutput = &MortyError{Code: "M5005", Message: "output read failed", Module: "callcli"}
	// ErrCallCLISignal indicates signal handling failed
	ErrCallCLISignal = &MortyError{Code: "M5006", Message: "signal handling failed", Module: "callcli"}
)

// ============================================================================
// CLI Errors (M6xxx)
// ============================================================================

var (
	// ErrCLIUnknownCommand indicates an unknown command
	ErrCLIUnknownCommand = &MortyError{Code: "M6001", Message: "unknown command", Module: "cli"}
	// ErrCLIInvalidFlag indicates an invalid flag/option
	ErrCLIInvalidFlag = &MortyError{Code: "M6002", Message: "invalid flag", Module: "cli"}
	// ErrCLIMissingArg indicates a missing required argument
	ErrCLIMissingArg = &MortyError{Code: "M6003", Message: "missing argument", Module: "cli"}
	// ErrCLIFlagConflict indicates conflicting flags
	ErrCLIFlagConflict = &MortyError{Code: "M6004", Message: "flag conflict", Module: "cli"}
)

// ============================================================================
// Executor Errors (M7xxx)
// ============================================================================

var (
	// ErrExecutorPrecondition indicates a precondition was not met
	ErrExecutorPrecondition = &MortyError{Code: "M7001", Message: "precondition not met", Module: "executor"}
	// ErrExecutorJobFailed indicates a job execution failed
	ErrExecutorJobFailed = &MortyError{Code: "M7002", Message: "job execution failed", Module: "executor"}
	// ErrExecutorMaxRetry indicates max retry attempts were exceeded
	ErrExecutorMaxRetry = &MortyError{Code: "M7003", Message: "max retry exceeded", Module: "executor"}
	// ErrExecutorPromptBuild indicates prompt building failed
	ErrExecutorPromptBuild = &MortyError{Code: "M7004", Message: "prompt build failed", Module: "executor"}
	// ErrExecutorResultParse indicates result parsing failed
	ErrExecutorResultParse = &MortyError{Code: "M7005", Message: "result parse failed", Module: "executor"}
	// ErrExecutorBlocked indicates the job is blocked
	ErrExecutorBlocked = &MortyError{Code: "M7006", Message: "job blocked", Module: "executor"}
)

// ============================================================================
// Cmd Errors (M8xxx)
// ============================================================================

var (
	// ErrCmdPlanNotFound indicates the plan directory was not found
	ErrCmdPlanNotFound = &MortyError{Code: "M8001", Message: "plan directory not found", Module: "cmd"}
	// ErrCmdResearchNotFound indicates the research directory was not found
	ErrCmdResearchNotFound = &MortyError{Code: "M8002", Message: "research directory not found", Module: "cmd"}
	// ErrCmdDoingRunning indicates a doing process is already running
	ErrCmdDoingRunning = &MortyError{Code: "M8003", Message: "doing already running", Module: "cmd"}
	// ErrCmdNoPendingJobs indicates there are no pending jobs
	ErrCmdNoPendingJobs = &MortyError{Code: "M8004", Message: "no pending jobs", Module: "cmd"}
	// ErrCmdModuleNotFound indicates the specified module was not found
	ErrCmdModuleNotFound = &MortyError{Code: "M8005", Message: "module not found", Module: "cmd"}
	// ErrCmdJobNotFound indicates the specified job was not found
	ErrCmdJobNotFound = &MortyError{Code: "M8006", Message: "job not found", Module: "cmd"}
	// ErrCmdResetFailed indicates a reset/rollback operation failed
	ErrCmdResetFailed = &MortyError{Code: "M8007", Message: "reset failed", Module: "cmd"}
	// ErrCmdStatFailed indicates a status query failed
	ErrCmdStatFailed = &MortyError{Code: "M8008", Message: "status query failed", Module: "cmd"}
)

// ============================================================================
// Deploy Errors (M9xxx)
// ============================================================================

var (
	// ErrDeployBuild indicates a build failure
	ErrDeployBuild = &MortyError{Code: "M9001", Message: "build failed", Module: "deploy"}
	// ErrDeployInstall indicates an installation failure
	ErrDeployInstall = &MortyError{Code: "M9002", Message: "install failed", Module: "deploy"}
	// ErrDeployUninstall indicates an uninstallation failure
	ErrDeployUninstall = &MortyError{Code: "M9003", Message: "uninstall failed", Module: "deploy"}
	// ErrDeployUpgrade indicates an upgrade failure
	ErrDeployUpgrade = &MortyError{Code: "M9004", Message: "upgrade failed", Module: "deploy"}
	// ErrDeployVersion indicates a version check failure
	ErrDeployVersion = &MortyError{Code: "M9005", Message: "version check failed", Module: "deploy"}
)
