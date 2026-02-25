// Package cli provides command-line interface functionality
package cli

import (
	"testing"
)

func TestGlobalOptionDefinitions(t *testing.T) {
	definitions := GlobalOptionDefinitions()

	if len(definitions) != 2 {
		t.Fatalf("expected 2 global option definitions, got %d", len(definitions))
	}

	// Check verbose option
	var verboseFound bool
	var debugFound bool

	for _, opt := range definitions {
		switch opt.Name {
		case GlobalOptionVerbose:
			verboseFound = true
			if opt.Short != GlobalOptionVerboseShort {
				t.Errorf("expected verbose short option '%s', got '%s'", GlobalOptionVerboseShort, opt.Short)
			}
			if opt.HasValue {
				t.Error("verbose option should not have a value")
			}
		case GlobalOptionDebug:
			debugFound = true
			if opt.Short != GlobalOptionDebugShort {
				t.Errorf("expected debug short option '%s', got '%s'", GlobalOptionDebugShort, opt.Short)
			}
			if opt.HasValue {
				t.Error("debug option should not have a value")
			}
		}
	}

	if !verboseFound {
		t.Error("verbose option not found in definitions")
	}
	if !debugFound {
		t.Error("debug option not found in definitions")
	}
}

func TestParseGlobalOptions(t *testing.T) {
	// Reset global options before each test
	ResetGlobalOptions()

	tests := []struct {
		name            string
		options         map[string]string
		expectedVerbose bool
		expectedDebug   bool
	}{
		{
			name:            "no options",
			options:         map[string]string{},
			expectedVerbose: false,
			expectedDebug:   false,
		},
		{
			name: "verbose long option",
			options: map[string]string{
				"verbose": "true",
			},
			expectedVerbose: true,
			expectedDebug:   false,
		},
		{
			name: "verbose short option",
			options: map[string]string{
				"v": "true",
			},
			expectedVerbose: true,
			expectedDebug:   false,
		},
		{
			name: "debug long option",
			options: map[string]string{
				"debug": "true",
			},
			expectedVerbose: false,
			expectedDebug:   true,
		},
		{
			name: "debug short option",
			options: map[string]string{
				"d": "true",
			},
			expectedVerbose: false,
			expectedDebug:   true,
		},
		{
			name: "both options",
			options: map[string]string{
				"verbose": "true",
				"debug":   "true",
			},
			expectedVerbose: true,
			expectedDebug:   true,
		},
		{
			name: "both short options",
			options: map[string]string{
				"v": "true",
				"d": "true",
			},
			expectedVerbose: true,
			expectedDebug:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global options
			ResetGlobalOptions()

			// Create parse result with options
			result := &ParseResult{
				Options:        tt.options,
				PositionalArgs: []string{},
			}

			// Parse global options
			ParseGlobalOptions(result)

			// Check results
			opts := GetGlobalOptions()
			if opts.Verbose != tt.expectedVerbose {
				t.Errorf("expected Verbose=%v, got %v", tt.expectedVerbose, opts.Verbose)
			}
			if opts.Debug != tt.expectedDebug {
				t.Errorf("expected Debug=%v, got %v", tt.expectedDebug, opts.Debug)
			}
		})
	}
}

func TestGetGlobalOptions(t *testing.T) {
	// Reset before test
	ResetGlobalOptions()

	// Default should be false, false
	opts := GetGlobalOptions()
	if opts.Verbose {
		t.Error("expected Verbose to be false by default")
	}
	if opts.Debug {
		t.Error("expected Debug to be false by default")
	}

	// Set options and verify
	SetGlobalOptions(GlobalOptions{Verbose: true, Debug: true})

	opts = GetGlobalOptions()
	if !opts.Verbose {
		t.Error("expected Verbose to be true")
	}
	if !opts.Debug {
		t.Error("expected Debug to be true")
	}

	// Verify it returns a copy (modifying returned struct shouldn't affect original)
	opts.Verbose = false
	opts.Debug = false

	opts2 := GetGlobalOptions()
	if !opts2.Verbose {
		t.Error("GetGlobalOptions should return a copy, original should not be affected")
	}
	if !opts2.Debug {
		t.Error("GetGlobalOptions should return a copy, original should not be affected")
	}

	// Reset after test
	ResetGlobalOptions()
}

func TestSetGlobalOptions(t *testing.T) {
	ResetGlobalOptions()

	SetGlobalOptions(GlobalOptions{Verbose: true, Debug: false})
	opts := GetGlobalOptions()
	if !opts.Verbose {
		t.Error("expected Verbose to be true after SetGlobalOptions")
	}
	if opts.Debug {
		t.Error("expected Debug to be false after SetGlobalOptions")
	}

	SetGlobalOptions(GlobalOptions{Verbose: false, Debug: true})
	opts = GetGlobalOptions()
	if opts.Verbose {
		t.Error("expected Verbose to be false after second SetGlobalOptions")
	}
	if !opts.Debug {
		t.Error("expected Debug to be true after second SetGlobalOptions")
	}

	ResetGlobalOptions()
}

func TestResetGlobalOptions(t *testing.T) {
	// Set some options
	SetGlobalOptions(GlobalOptions{Verbose: true, Debug: true})

	// Reset
	ResetGlobalOptions()

	// Verify reset
	opts := GetGlobalOptions()
	if opts.Verbose {
		t.Error("expected Verbose to be false after ResetGlobalOptions")
	}
	if opts.Debug {
		t.Error("expected Debug to be false after ResetGlobalOptions")
	}
}

func TestIsVerboseEnabled(t *testing.T) {
	ResetGlobalOptions()

	if IsVerboseEnabled() {
		t.Error("expected IsVerboseEnabled to return false by default")
	}

	SetGlobalOptions(GlobalOptions{Verbose: true})
	if !IsVerboseEnabled() {
		t.Error("expected IsVerboseEnabled to return true after setting")
	}

	ResetGlobalOptions()
}

func TestIsDebugEnabled(t *testing.T) {
	ResetGlobalOptions()

	if IsDebugEnabled() {
		t.Error("expected IsDebugEnabled to return false by default")
	}

	SetGlobalOptions(GlobalOptions{Debug: true})
	if !IsDebugEnabled() {
		t.Error("expected IsDebugEnabled to return true after setting")
	}

	ResetGlobalOptions()
}

func TestGetKnownGlobalOptions(t *testing.T) {
	knownOpts := GetKnownGlobalOptions()

	// Should have 4 entries: verbose, v, debug, d
	expectedOpts := []string{GlobalOptionVerbose, GlobalOptionVerboseShort, GlobalOptionDebug, GlobalOptionDebugShort}

	for _, opt := range expectedOpts {
		if optType, exists := knownOpts[opt]; !exists {
			t.Errorf("expected option '%s' to be in known options", opt)
		} else if optType != OptionTypeBool {
			t.Errorf("expected option '%s' to be OptionTypeBool, got %v", opt, optType)
		}
	}
}

func TestParseGlobalOptionsWithParser(t *testing.T) {
	ResetGlobalOptions()

	// Test parsing through the actual parser
	parser := NewParser(GetKnownGlobalOptions())

	args := []string{"test-cmd", "--verbose", "-d", "arg1", "arg2"}
	result, err := parser.Parse(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse global options
	remainingArgs := ParseGlobalOptions(result)

	// Check that global options were extracted
	opts := GetGlobalOptions()
	if !opts.Verbose {
		t.Error("expected Verbose to be true after parsing")
	}
	if !opts.Debug {
		t.Error("expected Debug to be true after parsing")
	}

	// Check that positional args remain
	if len(remainingArgs) != 2 {
		t.Errorf("expected 2 remaining args, got %d: %v", len(remainingArgs), remainingArgs)
	}

	ResetGlobalOptions()
}

func TestGlobalOptionsConcurrency(t *testing.T) {
	ResetGlobalOptions()

	// Run concurrent reads and writes
	done := make(chan bool, 100)

	// 50 goroutines setting options
	for i := 0; i < 50; i++ {
		go func(verbose bool) {
			SetGlobalOptions(GlobalOptions{Verbose: verbose})
			done <- true
		}(i%2 == 0)
	}

	// 50 goroutines reading options
	for i := 0; i < 50; i++ {
		go func() {
			_ = GetGlobalOptions()
			_ = IsVerboseEnabled()
			_ = IsDebugEnabled()
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 100; i++ {
		<-done
	}

	// If we got here without panic or deadlock, the test passed
	ResetGlobalOptions()
}
