// Package cli provides command-line argument parsing functionality
package cli

import (
	"fmt"
	"strings"
)

// ParseResult holds the parsed command-line arguments
type ParseResult struct {
	// Command is the first positional argument (the command name)
	Command string
	// Options maps option names to their values
	// For boolean flags, the value is "true"
	Options map[string]string
	// PositionalArgs contains non-option arguments (excluding command)
	PositionalArgs []string
	// RawArgs contains the original arguments
	RawArgs []string
}

// HasOption checks if an option was provided
func (r *ParseResult) HasOption(name string) bool {
	_, exists := r.Options[name]
	return exists
}

// GetOption returns the value of an option and a boolean indicating if it exists
func (r *ParseResult) GetOption(name string) (string, bool) {
	val, exists := r.Options[name]
	return val, exists
}

// Parser handles command-line argument parsing
type Parser struct {
	// KnownOptions defines known options with their types
	// If an option is not in this map, it's treated as a string option
	KnownOptions map[string]OptionType
}

// OptionType defines how an option should be parsed
type OptionType int

const (
	// OptionTypeBool is a flag without a value
	OptionTypeBool OptionType = iota
	// OptionTypeString requires a value
	OptionTypeString
)

// NewParser creates a new Parser with optional known options
func NewParser(knownOptions map[string]OptionType) *Parser {
	if knownOptions == nil {
		knownOptions = make(map[string]OptionType)
	}
	return &Parser{
		KnownOptions: knownOptions,
	}
}

// Parse parses command-line arguments and returns a ParseResult
// It supports:
//   - Long options: --option or --option=value
//   - Short options: -o or -ovalue
//   - Boolean flags: --flag or -f
//   - Positional arguments
//   - -- to stop option parsing
func (p *Parser) Parse(args []string) (*ParseResult, error) {
	result := &ParseResult{
		Options:        make(map[string]string),
		PositionalArgs: make([]string, 0),
		RawArgs:        args,
	}

	if len(args) == 0 {
		return result, nil
	}

	// The first argument is the command
	result.Command = args[0]

	// Parse remaining arguments
	i := 1
	for i < len(args) {
		arg := args[i]

		// Check for -- (end of options marker)
		if arg == "--" {
			// Collect all remaining arguments as positional
			i++
			for i < len(args) {
				result.PositionalArgs = append(result.PositionalArgs, args[i])
				i++
			}
			break
		}

		// Check for long option (--option)
		if strings.HasPrefix(arg, "--") {
			// Extract option name and value
			optName := arg[2:] // Remove "--"
			if optName == "" {
				// This is just "--" which is handled above
				result.PositionalArgs = append(result.PositionalArgs, arg)
				i++
				continue
			}

			// Check for --option=value format
			if idx := strings.Index(optName, "="); idx != -1 {
				optValue := optName[idx+1:]
				optName = optName[:idx]
				if optName == "" {
					return nil, fmt.Errorf("invalid option: '%s'", arg)
				}
				result.Options[optName] = optValue
			} else {
				// Check if this option requires a value
				optType, isKnown := p.getOptionType(optName)
				if isKnown && optType == OptionTypeString {
					// Known string option - next argument is the value
					if i+1 >= len(args) {
						return nil, fmt.Errorf("option '--%s' requires a value", optName)
					}
					i++
					result.Options[optName] = args[i]
				} else if isKnown && optType == OptionTypeBool {
					// Known boolean flag
					result.Options[optName] = "true"
				} else {
					// Unknown option - use heuristic:
					// If next arg exists and doesn't start with '-', treat as value
					// Otherwise, treat as boolean flag
					if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
						i++
						result.Options[optName] = args[i]
					} else {
						result.Options[optName] = "true"
					}
				}
			}
			i++
			continue
		}

		// Check for short option (-o)
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			shortOpts := arg[1:]
			isCombined := len(shortOpts) > 1

			for j := 0; j < len(shortOpts); j++ {
				optName := string(shortOpts[j])
				optType, isKnown := p.getOptionType(optName)

				if isKnown && optType == OptionTypeString {
					// This option requires a value
					if j < len(shortOpts)-1 {
						// Value is concatenated: -ovalue
						result.Options[optName] = shortOpts[j+1:]
					} else {
						// Value is the next argument: -o value
						if i+1 >= len(args) {
							return nil, fmt.Errorf("option '-%s' requires a value", optName)
						}
						i++
						result.Options[optName] = args[i]
					}
					break
				} else if !isKnown && !isCombined {
					// Unknown single option (not combined) - use heuristic
					// If next arg exists and doesn't start with '-', treat as value
					if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
						i++
						result.Options[optName] = args[i]
					} else {
						// Treat as boolean flag
						result.Options[optName] = "true"
					}
				} else {
					// Boolean flag (known bool, or unknown in combined form)
					result.Options[optName] = "true"
				}
			}
			i++
			continue
		}

		// Positional argument
		result.PositionalArgs = append(result.PositionalArgs, arg)
		i++
	}

	return result, nil
}

// getOptionType returns the type of an option and whether it's known
func (p *Parser) getOptionType(name string) (OptionType, bool) {
	if optType, exists := p.KnownOptions[name]; exists {
		return optType, true
	}
	// Unknown option
	return OptionTypeBool, false
}

// Parse is the standalone function to parse arguments
// It creates a default parser and parses the arguments
func Parse(args []string) (*ParseResult, error) {
	parser := NewParser(nil)
	return parser.Parse(args)
}
