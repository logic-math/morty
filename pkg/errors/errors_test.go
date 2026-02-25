package errors

import (
	"errors"
	"fmt"
	"testing"
)

// TestMortyError_Error tests the Error() method
func TestMortyError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *MortyError
		expected string
	}{
		{
			name:     "simple error without cause",
			err:      &MortyError{Code: "M0001", Message: "general error"},
			expected: "[M0001] general error",
		},
		{
			name:     "error with cause",
			err:      &MortyError{Code: "M1001", Message: "config file not found", Cause: fmt.Errorf("file not found")},
			expected: "[M1001] config file not found: file not found",
		},
		{
			name:     "error with module",
			err:      &MortyError{Code: "M2001", Message: "state file not found", Module: "state"},
			expected: "[M2001] state file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestMortyError_Unwrap tests the Unwrap() method
func TestMortyError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := &MortyError{Code: "M0001", Message: "general error", Cause: cause}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Test nil cause
	errNoCause := &MortyError{Code: "M0001", Message: "general error"}
	if errNoCause.Unwrap() != nil {
		t.Errorf("Unwrap() with nil cause should return nil")
	}
}

// TestMortyError_WithDetail tests the WithDetail() method
func TestMortyError_WithDetail(t *testing.T) {
	err := New("M1001", "config not found").
		WithDetail("file", "config.yaml").
		WithDetail("line", 42)

	if err.Details["file"] != "config.yaml" {
		t.Errorf("WithDetail() did not set file correctly")
	}
	if err.Details["line"] != 42 {
		t.Errorf("WithDetail() did not set line correctly")
	}

	// Test chaining returns same error
	err2 := New("M1001", "config not found")
	err3 := err2.WithDetail("key", "value")
	if err2 != err3 {
		t.Errorf("WithDetail() should return the same error for chaining")
	}
}

// TestMortyError_WithModule tests the WithModule() method
func TestMortyError_WithModule(t *testing.T) {
	err := New("M1001", "config not found").WithModule("config")
	if err.Module != "config" {
		t.Errorf("WithModule() = %q, want %q", err.Module, "config")
	}
}

// TestNew tests the New() constructor
func TestNew(t *testing.T) {
	err := New("M9999", "test error")
	if err.Code != "M9999" {
		t.Errorf("New() Code = %q, want %q", err.Code, "M9999")
	}
	if err.Message != "test error" {
		t.Errorf("New() Message = %q, want %q", err.Message, "test error")
	}
	if err.Details == nil {
		t.Errorf("New() should initialize Details map")
	}
	if err.Cause != nil {
		t.Errorf("New() should have nil Cause")
	}
}

// TestWrap tests the Wrap() constructor
func TestWrap(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := Wrap(cause, "M1001", "config not found")

	if err.Code != "M1001" {
		t.Errorf("Wrap() Code = %q, want %q", err.Code, "M1001")
	}
	if err.Message != "config not found" {
		t.Errorf("Wrap() Message = %q, want %q", err.Message, "config not found")
	}
	if err.Cause != cause {
		t.Errorf("Wrap() Cause should be set correctly")
	}
	if err.Details == nil {
		t.Errorf("Wrap() should initialize Details map")
	}
}

// TestIs tests the Is() function
func TestIs(t *testing.T) {
	// Test exact match
	err := ErrNotFound
	if !Is(err, ErrNotFound) {
		t.Errorf("Is() should return true for exact match")
	}

	// Test different error codes
	if Is(ErrGeneral, ErrNotFound) {
		t.Errorf("Is() should return false for different error codes")
	}

	// Test wrapped error
	wrapped := Wrap(ErrNotFound, "M1001", "config not found")
	if !Is(wrapped, ErrNotFound) {
		t.Errorf("Is() should return true for wrapped error with matching code in chain")
	}

	// Test deeply wrapped error
	deepWrapped := Wrap(wrapped, "M8001", "plan not found")
	if !Is(deepWrapped, ErrNotFound) {
		t.Errorf("Is() should return true for deeply wrapped error")
	}

	// Test nil handling
	if Is(nil, nil) != true {
		t.Errorf("Is(nil, nil) should return true")
	}
	if Is(ErrGeneral, nil) != false {
		t.Errorf("Is(err, nil) should return false")
	}
	if Is(nil, ErrGeneral) != false {
		t.Errorf("Is(nil, target) should return false")
	}

	// Test non-MortyError
	stdErr := fmt.Errorf("standard error")
	if Is(stdErr, ErrNotFound) {
		t.Errorf("Is() should return false for non-MortyError")
	}
}

// TestAsMortyError tests the AsMortyError function
func TestAsMortyError(t *testing.T) {
	// Test direct MortyError
	me, ok := AsMortyError(ErrNotFound)
	if !ok || me == nil {
		t.Errorf("AsMortyError() should extract MortyError")
	}
	if me.Code != "M0003" {
		t.Errorf("AsMortyError() extracted wrong code: %s", me.Code)
	}

	// Test wrapped error
	wrapped := Wrap(ErrNotFound, "M1001", "config not found")
	me2, ok2 := AsMortyError(wrapped)
	if !ok2 || me2 == nil {
		t.Errorf("AsMortyError() should extract from wrapped error")
	}
	if me2.Code != "M1001" {
		t.Errorf("AsMortyError() should return outermost MortyError")
	}

	// Test non-MortyError
	stdErr := fmt.Errorf("standard error")
	me3, ok3 := AsMortyError(stdErr)
	if ok3 || me3 != nil {
		t.Errorf("AsMortyError() should return false for standard error")
	}

	// Test nil
	me4, ok4 := AsMortyError(nil)
	if ok4 || me4 != nil {
		t.Errorf("AsMortyError(nil) should return false, nil")
	}
}

// TestErrorChain tests error chain functionality with standard library
func TestErrorChain(t *testing.T) {
	// Create a chain: root -> middle -> top
	root := fmt.Errorf("root cause")
	middle := Wrap(root, "M1001", "middle error")
	top := Wrap(middle, "M2001", "top error")

	// Test Unwrap chain
	if !errors.Is(top, root) {
		t.Errorf("errors.Is should traverse chain to find root")
	}

	// Test finding MortyError in chain
	var me *MortyError
	if !errors.As(top, &me) {
		t.Errorf("errors.As should extract MortyError from chain")
	}
	if me.Code != "M2001" {
		t.Errorf("errors.As should return outermost MortyError")
	}
}

// TestAllErrorCodes tests that all predefined errors have correct codes
func TestAllErrorCodes(t *testing.T) {
	testCases := []struct {
		err  *MortyError
		code string
	}{
		// General errors
		{ErrSuccess, "M0000"},
		{ErrGeneral, "M0001"},
		{ErrInvalidArgs, "M0002"},
		{ErrNotFound, "M0003"},
		{ErrAlreadyExists, "M0004"},
		{ErrPermission, "M0005"},
		{ErrTimeout, "M0006"},
		{ErrCancelled, "M0007"},
		{ErrInterrupted, "M0008"},
		{ErrNotImplemented, "M0009"},
		// Config errors
		{ErrConfigNotFound, "M1001"},
		{ErrConfigParse, "M1002"},
		{ErrConfigInvalid, "M1003"},
		{ErrConfigVersion, "M1004"},
		{ErrConfigRequired, "M1005"},
		// State errors
		{ErrStateNotFound, "M2001"},
		{ErrStateParse, "M2002"},
		{ErrStateCorrupted, "M2003"},
		{ErrStateTransition, "M2004"},
		{ErrStateModuleNotFound, "M2005"},
		{ErrStateJobNotFound, "M2006"},
		// Git errors
		{ErrGitNotRepo, "M3001"},
		{ErrGitCommit, "M3002"},
		{ErrGitStatus, "M3003"},
		{ErrGitDirtyWorktree, "M3004"},
		{ErrGitNoCommits, "M3005"},
		// Parser errors
		{ErrParserNotFound, "M4001"},
		{ErrParserFileNotFound, "M4002"},
		{ErrParserParse, "M4003"},
		{ErrParserInvalidFormat, "M4004"},
		{ErrParserNoJobs, "M4005"},
		// Call CLI errors
		{ErrCallCLINotFound, "M5001"},
		{ErrCallCLIExec, "M5002"},
		{ErrCallCLITimeout, "M5003"},
		{ErrCallCLIKilled, "M5004"},
		{ErrCallCLIOutput, "M5005"},
		{ErrCallCLISignal, "M5006"},
		// CLI errors
		{ErrCLIUnknownCommand, "M6001"},
		{ErrCLIInvalidFlag, "M6002"},
		{ErrCLIMissingArg, "M6003"},
		{ErrCLIFlagConflict, "M6004"},
		// Executor errors
		{ErrExecutorPrecondition, "M7001"},
		{ErrExecutorJobFailed, "M7002"},
		{ErrExecutorMaxRetry, "M7003"},
		{ErrExecutorPromptBuild, "M7004"},
		{ErrExecutorResultParse, "M7005"},
		{ErrExecutorBlocked, "M7006"},
		// Cmd errors
		{ErrCmdPlanNotFound, "M8001"},
		{ErrCmdResearchNotFound, "M8002"},
		{ErrCmdDoingRunning, "M8003"},
		{ErrCmdNoPendingJobs, "M8004"},
		{ErrCmdModuleNotFound, "M8005"},
		{ErrCmdJobNotFound, "M8006"},
		{ErrCmdResetFailed, "M8007"},
		{ErrCmdStatFailed, "M8008"},
		// Deploy errors
		{ErrDeployBuild, "M9001"},
		{ErrDeployInstall, "M9002"},
		{ErrDeployUninstall, "M9003"},
		{ErrDeployUpgrade, "M9004"},
		{ErrDeployVersion, "M9005"},
	}

	for _, tc := range testCases {
		t.Run(tc.code, func(t *testing.T) {
			if tc.err.Code != tc.code {
				t.Errorf("%s.Code = %q, want %q", tc.code, tc.err.Code, tc.code)
			}
		})
	}
}

// TestErrorModules tests that errors have correct module assignments
func TestErrorModules(t *testing.T) {
	testCases := []struct {
		err    *MortyError
		module string
	}{
		// Config errors
		{ErrConfigNotFound, "config"},
		{ErrConfigParse, "config"},
		// State errors
		{ErrStateNotFound, "state"},
		{ErrStateParse, "state"},
		// Git errors
		{ErrGitNotRepo, "git"},
		{ErrGitCommit, "git"},
		// Parser errors
		{ErrParserNotFound, "parser"},
		{ErrParserParse, "parser"},
		// Call CLI errors
		{ErrCallCLINotFound, "callcli"},
		{ErrCallCLIExec, "callcli"},
		// CLI errors
		{ErrCLIUnknownCommand, "cli"},
		{ErrCLIMissingArg, "cli"},
		// Executor errors
		{ErrExecutorJobFailed, "executor"},
		{ErrExecutorBlocked, "executor"},
		// Cmd errors
		{ErrCmdPlanNotFound, "cmd"},
		{ErrCmdJobNotFound, "cmd"},
		// Deploy errors
		{ErrDeployBuild, "deploy"},
		{ErrDeployInstall, "deploy"},
	}

	for _, tc := range testCases {
		t.Run(tc.err.Code, func(t *testing.T) {
			if tc.err.Module != tc.module {
				t.Errorf("%s.Module = %q, want %q", tc.err.Code, tc.err.Module, tc.module)
			}
		})
	}
}

// BenchmarkErrorCreation benchmarks creating new errors
func BenchmarkErrorCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New("M9999", "benchmark error")
	}
}

// BenchmarkErrorWrapping benchmarks wrapping errors
func BenchmarkErrorWrapping(b *testing.B) {
	cause := fmt.Errorf("cause")
	for i := 0; i < b.N; i++ {
		_ = Wrap(cause, "M9999", "wrapped")
	}
}

// BenchmarkIs benchmarks the Is function
func BenchmarkIs(b *testing.B) {
	err := Wrap(ErrNotFound, "M1001", "config not found")
	for i := 0; i < b.N; i++ {
		_ = Is(err, ErrNotFound)
	}
}
