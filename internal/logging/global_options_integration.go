// Package logging provides integration between global CLI options and logging
package logging

import (
	"github.com/morty/morty/internal/cli"
)

// ConfigureLoggerFromGlobalOptions configures a logger based on global CLI options
// - If --verbose is set, sets log level to InfoLevel (or keeps current if more verbose)
// - If --debug is set, sets log level to DebugLevel
func ConfigureLoggerFromGlobalOptions(logger Logger) {
	opts := cli.GetGlobalOptions()

	if opts.Debug {
		// Debug mode enables all logs including debug
		logger.SetLevel(DebugLevel)
	} else if opts.Verbose {
		// Verbose mode ensures at least Info level is enabled
		if logger.GetLevel() > InfoLevel {
			logger.SetLevel(InfoLevel)
		}
	}
}

// GetLogLevelFromGlobalOptions returns the appropriate log level based on global options
func GetLogLevelFromGlobalOptions() Level {
	opts := cli.GetGlobalOptions()

	if opts.Debug {
		return DebugLevel
	}
	if opts.Verbose {
		return InfoLevel
	}
	return InfoLevel
}

// CreateLoggerWithGlobalOptions creates a new logger configured with global options
func CreateLoggerWithGlobalOptions(config *FormatConfig) (Logger, error) {
	logger, err := CreateLogger(config)
	if err != nil {
		return nil, err
	}

	ConfigureLoggerFromGlobalOptions(logger)
	return logger, nil
}

// IsVerboseMode returns true if verbose mode is enabled via global options
func IsVerboseMode() bool {
	return cli.GetGlobalOptions().Verbose
}

// IsDebugMode returns true if debug mode is enabled via global options
func IsDebugMode() bool {
	return cli.GetGlobalOptions().Debug
}

// ApplyGlobalOptionsToConfig applies global options to a FormatConfig
func ApplyGlobalOptionsToConfig(config *FormatConfig) {
	opts := cli.GetGlobalOptions()

	if opts.Debug {
		config.Level = DebugLevel
	} else if opts.Verbose {
		// Only set to Info if current level is less verbose
		if config.Level > InfoLevel {
			config.Level = InfoLevel
		}
	}
}
