// Package logging provides integration between global CLI options and logging
package logging

import (
	"testing"

	"github.com/morty/morty/internal/cli"
)

func TestConfigureLoggerFromGlobalOptions(t *testing.T) {
	// Reset global options before test
	cli.ResetGlobalOptions()

	// Create a test logger with default level
	logger := NewFormatterLogger(NewTextFormatter(false, ""), nil, InfoLevel)

	// Test default (no options)
	ConfigureLoggerFromGlobalOptions(logger)
	if logger.GetLevel() != InfoLevel {
		t.Errorf("expected level to remain InfoLevel, got %v", logger.GetLevel())
	}

	// Reset and test verbose
	cli.ResetGlobalOptions()
	cli.SetGlobalOptions(cli.GlobalOptions{Verbose: true})

	logger = NewFormatterLogger(NewTextFormatter(false, ""), nil, WarnLevel)
	ConfigureLoggerFromGlobalOptions(logger)
	if logger.GetLevel() != InfoLevel {
		t.Errorf("expected level to be InfoLevel with verbose, got %v", logger.GetLevel())
	}

	// Reset and test debug
	cli.ResetGlobalOptions()
	cli.SetGlobalOptions(cli.GlobalOptions{Debug: true})

	logger = NewFormatterLogger(NewTextFormatter(false, ""), nil, InfoLevel)
	ConfigureLoggerFromGlobalOptions(logger)
	if logger.GetLevel() != DebugLevel {
		t.Errorf("expected level to be DebugLevel with debug, got %v", logger.GetLevel())
	}

	// Reset and test both
	cli.ResetGlobalOptions()
	cli.SetGlobalOptions(cli.GlobalOptions{Verbose: true, Debug: true})

	logger = NewFormatterLogger(NewTextFormatter(false, ""), nil, InfoLevel)
	ConfigureLoggerFromGlobalOptions(logger)
	if logger.GetLevel() != DebugLevel {
		t.Errorf("expected level to be DebugLevel with both flags, got %v", logger.GetLevel())
	}

	// Cleanup
	cli.ResetGlobalOptions()
}

func TestGetLogLevelFromGlobalOptions(t *testing.T) {
	// Reset global options
	cli.ResetGlobalOptions()

	// Test default
	if level := GetLogLevelFromGlobalOptions(); level != InfoLevel {
		t.Errorf("expected default level InfoLevel, got %v", level)
	}

	// Test verbose
	cli.SetGlobalOptions(cli.GlobalOptions{Verbose: true})
	if level := GetLogLevelFromGlobalOptions(); level != InfoLevel {
		t.Errorf("expected level InfoLevel with verbose, got %v", level)
	}

	// Test debug
	cli.SetGlobalOptions(cli.GlobalOptions{Debug: true})
	if level := GetLogLevelFromGlobalOptions(); level != DebugLevel {
		t.Errorf("expected level DebugLevel with debug, got %v", level)
	}

	// Cleanup
	cli.ResetGlobalOptions()
}

func TestIsVerboseMode(t *testing.T) {
	cli.ResetGlobalOptions()

	if IsVerboseMode() {
		t.Error("expected IsVerboseMode to return false by default")
	}

	cli.SetGlobalOptions(cli.GlobalOptions{Verbose: true})
	if !IsVerboseMode() {
		t.Error("expected IsVerboseMode to return true after setting")
	}

	cli.ResetGlobalOptions()
}

func TestIsDebugMode(t *testing.T) {
	cli.ResetGlobalOptions()

	if IsDebugMode() {
		t.Error("expected IsDebugMode to return false by default")
	}

	cli.SetGlobalOptions(cli.GlobalOptions{Debug: true})
	if !IsDebugMode() {
		t.Error("expected IsDebugMode to return true after setting")
	}

	cli.ResetGlobalOptions()
}

func TestApplyGlobalOptionsToConfig(t *testing.T) {
	cli.ResetGlobalOptions()

	// Test with default options
	config := DefaultFormatConfig()
	config.Level = WarnLevel
	ApplyGlobalOptionsToConfig(config)
	if config.Level != WarnLevel {
		t.Errorf("expected level to remain WarnLevel, got %v", config.Level)
	}

	// Test with verbose
	cli.ResetGlobalOptions()
	cli.SetGlobalOptions(cli.GlobalOptions{Verbose: true})
	config = DefaultFormatConfig()
	config.Level = WarnLevel
	ApplyGlobalOptionsToConfig(config)
	if config.Level != InfoLevel {
		t.Errorf("expected level to be InfoLevel with verbose, got %v", config.Level)
	}

	// Test with debug
	cli.ResetGlobalOptions()
	cli.SetGlobalOptions(cli.GlobalOptions{Debug: true})
	config = DefaultFormatConfig()
	config.Level = InfoLevel
	ApplyGlobalOptionsToConfig(config)
	if config.Level != DebugLevel {
		t.Errorf("expected level to be DebugLevel with debug, got %v", config.Level)
	}

	// Cleanup
	cli.ResetGlobalOptions()
}

func TestCreateLoggerWithGlobalOptions(t *testing.T) {
	cli.ResetGlobalOptions()

	// Test with debug flag
	cli.SetGlobalOptions(cli.GlobalOptions{Debug: true})

	config := DefaultFormatConfig()
	logger, err := CreateLoggerWithGlobalOptions(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if logger.GetLevel() != DebugLevel {
		t.Errorf("expected logger level to be DebugLevel, got %v", logger.GetLevel())
	}

	// Cleanup
	cli.ResetGlobalOptions()
}
