// Package cli provides command-line interface functionality including routing
package cli

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// CommandHandler is the function signature for command handlers
type CommandHandler func(ctx context.Context, args []string, opts ParseResult) error

// Option represents a command-line option
type Option struct {
	Name        string
	Short       string
	Description string
	HasValue    bool
	Required    bool
}

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Handler     CommandHandler
	Options     []Option
}

// Router handles command registration and routing
type Router struct {
	commands map[string]Command
	mu       sync.RWMutex
}

// NewRouter creates a new Router instance
func NewRouter() *Router {
	return &Router{
		commands: make(map[string]Command),
	}
}

// Register registers a command with the router
// Returns an error if a command with the same name is already registered
func (r *Router) Register(cmd Command) error {
	if cmd.Name == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	if cmd.Handler == nil {
		return fmt.Errorf("command '%s' must have a handler", cmd.Name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Normalize command name to lowercase for case-insensitive lookup
	normalizedName := strings.ToLower(cmd.Name)

	if _, exists := r.commands[normalizedName]; exists {
		return fmt.Errorf("command '%s' is already registered", cmd.Name)
	}

	r.commands[normalizedName] = cmd
	return nil
}

// Execute routes and executes a command based on the provided arguments
// The first argument is treated as the command name
func (r *Router) Execute(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified\n可用命令: %s", r.getAvailableCommands())
	}

	// First argument is the command name
	cmdName := strings.ToLower(args[0])

	// Get the command handler
	handler, exists := r.GetHandler(cmdName)
	if !exists {
		return fmt.Errorf("未知命令: %s\n可用命令: %s", args[0], r.getAvailableCommands())
	}

	// Parse the remaining arguments
	// Build known options from command's Options and global options
	cmd, _ := r.getCommand(cmdName)
	knownOptions := GetKnownGlobalOptions()

	// Create a mapping from short option names to long option names
	shortToLongMap := make(map[string]string)

	// Merge command-specific options
	for _, opt := range cmd.Options {
		if opt.HasValue {
			knownOptions[opt.Name] = OptionTypeString
		} else {
			knownOptions[opt.Name] = OptionTypeBool
		}
		// Also register short options
		if opt.Short != "" {
			shortName := strings.TrimPrefix(opt.Short, "-")
			if opt.HasValue {
				knownOptions[shortName] = OptionTypeString
			} else {
				knownOptions[shortName] = OptionTypeBool
			}
			// Map short name to long name for option normalization
			shortToLongMap[shortName] = opt.Name
		}
	}

	parser := NewParser(knownOptions)
	parseResult, err := parser.Parse(args)
	if err != nil {
		return fmt.Errorf("parsing error: %w", err)
	}

	// Normalize short option names to long option names in the parse result
	for shortName, longName := range shortToLongMap {
		if val, exists := parseResult.Options[shortName]; exists {
			parseResult.Options[longName] = val
			delete(parseResult.Options, shortName)
		}
	}

	// Parse global options from the result
	ParseGlobalOptions(parseResult)

	// Execute the handler
	return handler(ctx, parseResult.PositionalArgs, *parseResult)
}

// GetHandler retrieves a handler for a command by name
// The lookup is case-insensitive
// Returns the handler and a boolean indicating if the command was found
func (r *Router) GetHandler(name string) (CommandHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalizedName := strings.ToLower(name)
	cmd, exists := r.commands[normalizedName]
	if !exists {
		return nil, false
	}
	return cmd.Handler, true
}

// getCommand retrieves a command by name (internal use)
func (r *Router) getCommand(name string) (Command, bool) {
	normalizedName := strings.ToLower(name)
	cmd, exists := r.commands[normalizedName]
	return cmd, exists
}

// ListCommands returns a list of all registered commands
func (r *Router) ListCommands() []Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		result = append(result, cmd)
	}
	return result
}

// getAvailableCommands returns a comma-separated list of available command names
func (r *Router) getAvailableCommands() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.commands) == 0 {
		return "(none)"
	}

	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
