package cli

import (
	"reflect"
	"testing"
)

func TestParse_Command(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantCommand string
	}{
		{
			name:        "simple command",
			args:        []string{"morty"},
			wantCommand: "morty",
		},
		{
			name:        "command with path",
			args:        []string{"/usr/bin/morty"},
			wantCommand: "/usr/bin/morty",
		},
		{
			name:        "command with args",
			args:        []string{"morty", "run"},
			wantCommand: "morty",
		},
		{
			name:        "empty args",
			args:        []string{},
			wantCommand: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if result.Command != tt.wantCommand {
				t.Errorf("Command = %v, want %v", result.Command, tt.wantCommand)
			}
		})
	}
}

func TestParse_LongOptions(t *testing.T) {
	// Create a parser with known options for tests that require specific option types
	parser := NewParser(map[string]OptionType{
		"verbose": OptionTypeBool,
		"output":  OptionTypeString,
		"format":  OptionTypeString,
	})

	tests := []struct {
		name         string
		args         []string
		wantOptions  map[string]string
		wantPosArgs  []string
		wantErr      bool
		errContains  string
		useParser    bool // whether to use the parser with known options
	}{
		{
			name:        "long flag only",
			args:        []string{"cmd", "--verbose"},
			wantOptions: map[string]string{"verbose": "true"},
			wantPosArgs: []string{},
			useParser:   true,
		},
		{
			name:        "long option with equals",
			args:        []string{"cmd", "--output=file.txt"},
			wantOptions: map[string]string{"output": "file.txt"},
			wantPosArgs: []string{},
			useParser:   true,
		},
		{
			name:        "long option with space",
			args:        []string{"cmd", "--output", "file.txt"},
			wantOptions: map[string]string{"output": "file.txt"},
			wantPosArgs: []string{},
			useParser:   true,
		},
		{
			name:        "multiple long options",
			args:        []string{"cmd", "--verbose", "--output=file.txt", "--format", "json"},
			wantOptions: map[string]string{"verbose": "true", "output": "file.txt", "format": "json"},
			wantPosArgs: []string{},
			useParser:   true,
		},
		{
			name:        "long option with positional args",
			args:        []string{"cmd", "--verbose", "run", "test"},
			wantOptions: map[string]string{"verbose": "true"},
			wantPosArgs: []string{"run", "test"},
			useParser:   true,
		},
		{
			name:        "double dash at end consumes no args",
			args:        []string{"cmd", "run", "--"},
			wantOptions: map[string]string{},
			wantPosArgs: []string{"run"},
			useParser:   false, // Uses default Parse
		},
		{
			name:        "long option with empty name",
			args:        []string{"cmd", "--=value"},
			wantOptions: map[string]string{},
			wantPosArgs: []string{},
			wantErr:     true,
			errContains: "invalid option",
			useParser:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result *ParseResult
			var err error
			if tt.useParser {
				result, err = parser.Parse(tt.args)
			} else {
				result, err = Parse(tt.args)
			}
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse() expected error, got nil")
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, should contain %v", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(result.Options, tt.wantOptions) {
				t.Errorf("Options = %v, want %v", result.Options, tt.wantOptions)
			}
			if !reflect.DeepEqual(result.PositionalArgs, tt.wantPosArgs) {
				t.Errorf("PositionalArgs = %v, want %v", result.PositionalArgs, tt.wantPosArgs)
			}
		})
	}
}

func TestParse_ShortOptions(t *testing.T) {
	// Create a parser with known options for tests that require specific option types
	parser := NewParser(map[string]OptionType{
		"v": OptionTypeBool,
		"f": OptionTypeBool,
		"n": OptionTypeBool,
		"o": OptionTypeString,
	})

	tests := []struct {
		name        string
		args         []string
		wantOptions  map[string]string
		wantPosArgs  []string
		wantErr      bool
		errContains  string
		useParser    bool
	}{
		{
			name:        "single short flag",
			args:        []string{"cmd", "-v"},
			wantOptions: map[string]string{"v": "true"},
			wantPosArgs: []string{},
			useParser:   true,
		},
		{
			name:        "multiple short flags",
			args:        []string{"cmd", "-v", "-f"},
			wantOptions: map[string]string{"v": "true", "f": "true"},
			wantPosArgs: []string{},
			useParser:   true,
		},
		{
			name:        "combined short flags",
			args:        []string{"cmd", "-vfn"},
			wantOptions: map[string]string{"v": "true", "f": "true", "n": "true"},
			wantPosArgs: []string{},
			useParser:   true,
		},
		{
			name:        "short option with value (concatenated)",
			args:        []string{"cmd", "-ofile.txt"},
			wantOptions: map[string]string{"o": "file.txt"},
			wantPosArgs: []string{},
			useParser:   true,
		},
		{
			name:        "short option with value (separate)",
			args:        []string{"cmd", "-o", "file.txt"},
			wantOptions: map[string]string{"o": "file.txt"},
			wantPosArgs: []string{},
			useParser:   true,
		},
		{
			name:        "combined flags with value at end",
			args:        []string{"cmd", "-vfofile.txt"},
			wantOptions: map[string]string{"v": "true", "f": "true", "o": "file.txt"},
			wantPosArgs: []string{},
			useParser:   true,
		},
		{
			name:        "short flags with positional args",
			args:        []string{"cmd", "-v", "run"},
			wantOptions: map[string]string{"v": "true"},
			wantPosArgs: []string{"run"},
			useParser:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result *ParseResult
			var err error
			if tt.useParser {
				result, err = parser.Parse(tt.args)
			} else {
				result, err = Parse(tt.args)
			}
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(result.Options, tt.wantOptions) {
				t.Errorf("Options = %v, want %v", result.Options, tt.wantOptions)
			}
			if !reflect.DeepEqual(result.PositionalArgs, tt.wantPosArgs) {
				t.Errorf("PositionalArgs = %v, want %v", result.PositionalArgs, tt.wantPosArgs)
			}
		})
	}
}

func TestParse_MixedOptions(t *testing.T) {
	// Create a parser with known options
	parser := NewParser(map[string]OptionType{
		"v":       OptionTypeBool,
		"f":       OptionTypeBool,
		"output":  OptionTypeString,
		"format":  OptionTypeString,
	})

	tests := []struct {
		name        string
		args         []string
		wantOptions  map[string]string
		wantPosArgs  []string
	}{
		{
			name:        "short and long options",
			args:        []string{"cmd", "-v", "--output", "file.txt", "-f"},
			wantOptions: map[string]string{"v": "true", "output": "file.txt", "f": "true"},
			wantPosArgs: []string{},
		},
		{
			name:        "mixed with positional args",
			args:        []string{"cmd", "-v", "run", "--format", "json", "test"},
			wantOptions: map[string]string{"v": "true", "format": "json"},
			wantPosArgs: []string{"run", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(result.Options, tt.wantOptions) {
				t.Errorf("Options = %v, want %v", result.Options, tt.wantOptions)
			}
			if !reflect.DeepEqual(result.PositionalArgs, tt.wantPosArgs) {
				t.Errorf("PositionalArgs = %v, want %v", result.PositionalArgs, tt.wantPosArgs)
			}
		})
	}
}

func TestParse_DoubleDash(t *testing.T) {
	// Create a parser with known options
	parser := NewParser(map[string]OptionType{
		"v": OptionTypeBool,
	})

	tests := []struct {
		name        string
		args         []string
		wantOptions  map[string]string
		wantPosArgs  []string
		useParser    bool
	}{
		{
			name:        "double dash stops parsing",
			args:        []string{"cmd", "-v", "--", "-f", "--option", "value"},
			wantOptions: map[string]string{"v": "true"},
			wantPosArgs: []string{"-f", "--option", "value"},
			useParser:   true,
		},
		{
			name:        "double dash at end",
			args:        []string{"cmd", "run", "--"},
			wantOptions: map[string]string{},
			wantPosArgs: []string{"run"},
			useParser:   false, // Uses default Parse
		},
		{
			name:        "double dash at start",
			args:        []string{"cmd", "--", "-v", "run"},
			wantOptions: map[string]string{},
			wantPosArgs: []string{"-v", "run"},
			useParser:   false, // Uses default Parse
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result *ParseResult
			var err error
			if tt.useParser {
				result, err = parser.Parse(tt.args)
			} else {
				result, err = Parse(tt.args)
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(result.Options, tt.wantOptions) {
				t.Errorf("Options = %v, want %v", result.Options, tt.wantOptions)
			}
			if !reflect.DeepEqual(result.PositionalArgs, tt.wantPosArgs) {
				t.Errorf("PositionalArgs = %v, want %v", result.PositionalArgs, tt.wantPosArgs)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	// Create a parser with known options that require values
	parser := NewParser(map[string]OptionType{
		"output": OptionTypeString,
		"o":      OptionTypeString,
	})

	tests := []struct {
		name        string
		args        []string
		errContains string
	}{
		{
			name:        "long option missing value",
			args:        []string{"cmd", "--output"},
			errContains: "requires a value",
		},
		{
			name:        "short option missing value",
			args:        []string{"cmd", "-o"},
			errContains: "requires a value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.args)
			if err == nil {
				t.Fatalf("Parse() expected error, got nil")
			}
			if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("Error = %v, should contain %v", err, tt.errContains)
			}
		})
	}
}

func TestParser_WithKnownOptions(t *testing.T) {
	knownOptions := map[string]OptionType{
		"verbose": OptionTypeBool,
		"v":       OptionTypeBool,
		"output":  OptionTypeString,
		"o":       OptionTypeString,
		"format":  OptionTypeString,
		"f":       OptionTypeString, // -f takes a value here
	}

	parser := NewParser(knownOptions)

	tests := []struct {
		name        string
		args         []string
		wantOptions  map[string]string
		wantPosArgs  []string
		wantErr      bool
	}{
		{
			name:        "known bool option",
			args:        []string{"cmd", "--verbose"},
			wantOptions: map[string]string{"verbose": "true"},
			wantPosArgs: []string{},
		},
		{
			name:        "known string option with equals",
			args:        []string{"cmd", "--output=file.txt"},
			wantOptions: map[string]string{"output": "file.txt"},
			wantPosArgs: []string{},
		},
		{
			name:        "known string option with space",
			args:        []string{"cmd", "--format", "json"},
			wantOptions: map[string]string{"format": "json"},
			wantPosArgs: []string{},
		},
		{
			name:        "missing value for known string option",
			args:        []string{"cmd", "--output"},
			wantErr:     true,
		},
		{
			name:        "short option with value",
			args:        []string{"cmd", "-f", "yaml"},
			wantOptions: map[string]string{"f": "yaml"},
			wantPosArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(result.Options, tt.wantOptions) {
				t.Errorf("Options = %v, want %v", result.Options, tt.wantOptions)
			}
			if !reflect.DeepEqual(result.PositionalArgs, tt.wantPosArgs) {
				t.Errorf("PositionalArgs = %v, want %v", result.PositionalArgs, tt.wantPosArgs)
			}
		})
	}
}

func TestParseResult_HasOption(t *testing.T) {
	result := &ParseResult{
		Options: map[string]string{
			"verbose": "true",
			"output":  "file.txt",
		},
	}

	if !result.HasOption("verbose") {
		t.Error("HasOption('verbose') should return true")
	}
	if !result.HasOption("output") {
		t.Error("HasOption('output') should return true")
	}
	if result.HasOption("missing") {
		t.Error("HasOption('missing') should return false")
	}
}

func TestParseResult_GetOption(t *testing.T) {
	result := &ParseResult{
		Options: map[string]string{
			"verbose": "true",
			"output":  "file.txt",
		},
	}

	val, exists := result.GetOption("verbose")
	if !exists || val != "true" {
		t.Errorf("GetOption('verbose') = (%v, %v), want (true, true)", val, exists)
	}

	val, exists = result.GetOption("missing")
	if exists {
		t.Errorf("GetOption('missing') = (%v, %v), want (_, false)", val, exists)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(substr) <= len(s) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
