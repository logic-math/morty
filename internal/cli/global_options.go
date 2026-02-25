// Package cli provides command-line interface functionality
package cli

import (
	"sync"
)

// GlobalOptions represents global CLI options that can be used with any command
type GlobalOptions struct {
	// Verbose enables verbose output (detailed logging)
	Verbose bool
	// Debug enables debug mode (includes debug-level logs)
	Debug bool
}

// globalOptionsStore is the singleton storage for global options
var (
	globalOptionsStore     *GlobalOptions
	globalOptionsStoreOnce sync.Once
	globalOptionsMu        sync.RWMutex
)

// GlobalOptionName represents the name of a global option
const (
	// GlobalOptionVerbose is the name of the verbose flag
	GlobalOptionVerbose = "verbose"
	// GlobalOptionDebug is the name of the debug flag
	GlobalOptionDebug = "debug"
	// GlobalOptionVerboseShort is the short name of the verbose flag
	GlobalOptionVerboseShort = "v"
	// GlobalOptionDebugShort is the short name of the debug flag
	GlobalOptionDebugShort = "d"
)

// GlobalOptionDefinitions returns the global option definitions for registration
func GlobalOptionDefinitions() []Option {
	return []Option{
		{
			Name:        GlobalOptionVerbose,
			Short:       GlobalOptionVerboseShort,
			Description: "Enable verbose output (detailed logging)",
			HasValue:    false,
			Required:    false,
		},
		{
			Name:        GlobalOptionDebug,
			Short:       GlobalOptionDebugShort,
			Description: "Enable debug mode (includes debug-level logs)",
			HasValue:    false,
			Required:    false,
		},
	}
}

// ParseGlobalOptions extracts global options from ParseResult and returns remaining args
func ParseGlobalOptions(result *ParseResult) []string {
	globalOptionsMu.Lock()
	defer globalOptionsMu.Unlock()

	if globalOptionsStore == nil {
		globalOptionsStore = &GlobalOptions{}
	}

	// Check for verbose flag (long or short form)
	if result.HasOption(GlobalOptionVerbose) {
		globalOptionsStore.Verbose = true
	}
	if result.HasOption(GlobalOptionVerboseShort) {
		globalOptionsStore.Verbose = true
	}

	// Check for debug flag (long or short form)
	if result.HasOption(GlobalOptionDebug) {
		globalOptionsStore.Debug = true
	}
	if result.HasOption(GlobalOptionDebugShort) {
		globalOptionsStore.Debug = true
	}

	// Return remaining args (excluding global options)
	return result.PositionalArgs
}

// GetGlobalOptions returns the current global options
// This is safe for concurrent access
func GetGlobalOptions() GlobalOptions {
	globalOptionsMu.RLock()
	defer globalOptionsMu.RUnlock()

	if globalOptionsStore == nil {
		return GlobalOptions{}
	}

	// Return a copy to prevent external modification
	return GlobalOptions{
		Verbose: globalOptionsStore.Verbose,
		Debug:   globalOptionsStore.Debug,
	}
}

// ResetGlobalOptions resets the global options to their default values
// This is primarily used for testing
func ResetGlobalOptions() {
	globalOptionsMu.Lock()
	defer globalOptionsMu.Unlock()

	globalOptionsStore = &GlobalOptions{}
}

// SetGlobalOptions sets the global options directly
// This is primarily used for testing or initialization
func SetGlobalOptions(opts GlobalOptions) {
	globalOptionsMu.Lock()
	defer globalOptionsMu.Unlock()

	globalOptionsStore = &GlobalOptions{
		Verbose: opts.Verbose,
		Debug:   opts.Debug,
	}
}

// IsVerboseEnabled returns true if verbose mode is enabled
func IsVerboseEnabled() bool {
	return GetGlobalOptions().Verbose
}

// IsDebugEnabled returns true if debug mode is enabled
func IsDebugEnabled() bool {
	return GetGlobalOptions().Debug
}

// GetKnownGlobalOptions returns a map of known global options for parser
// This is used by the router to register global options as known boolean flags
func GetKnownGlobalOptions() map[string]OptionType {
	return map[string]OptionType{
		GlobalOptionVerbose:      OptionTypeBool,
		GlobalOptionVerboseShort: OptionTypeBool,
		GlobalOptionDebug:        OptionTypeBool,
		GlobalOptionDebugShort:   OptionTypeBool,
	}
}
