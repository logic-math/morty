// Package cmd provides command handlers for Morty CLI commands.
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/morty/morty/internal/callcli"
	"github.com/morty/morty/internal/config"
	"github.com/morty/morty/internal/logging"
)

// ResearchResult represents the result of a research operation.
type ResearchResult struct {
	Topic       string
	Content     string
	OutputPath  string
	Timestamp   time.Time
	Err         error
	ExitCode    int
	Duration    time.Duration
}

// ResearchHandler handles the research command.
type ResearchHandler struct {
	cfg        config.Manager
	logger     logging.Logger
	paths      *config.Paths
	cliCaller  callcli.AICliCaller
}

// NewResearchHandler creates a new ResearchHandler instance.
func NewResearchHandler(cfg config.Manager, logger logging.Logger) *ResearchHandler {
	// Create paths with config loader if available
	var paths *config.Paths
	if loader, ok := cfg.(*config.Loader); ok {
		paths = config.NewPathsWithLoader(loader)
		if os.Getenv("MORTY_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "DEBUG: NewResearchHandler using Loader\n")
		}
	} else {
		paths = config.NewPaths()
		if os.Getenv("MORTY_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "DEBUG: NewResearchHandler using NewPaths (cfg type: %T)\n", cfg)
		}
	}

	return &ResearchHandler{
		cfg:       cfg,
		logger:    logger,
		paths:     paths,
		cliCaller: callcli.NewAICliCallerWithLoader(cfg),
	}
}

// SetCLICaller sets a custom CLI caller (useful for testing).
func (h *ResearchHandler) SetCLICaller(caller callcli.AICliCaller) {
	h.cliCaller = caller
}

// SetPromptsDir sets a custom prompts directory (useful for testing).
func (h *ResearchHandler) SetPromptsDir(dir string) {
	h.paths.SetPromptsDir(dir)
}

// Execute executes the research command.
// If no topic is provided in args, it prompts the user interactively.
func (h *ResearchHandler) Execute(ctx context.Context, args []string) (*ResearchResult, error) {
	logger := h.logger.WithContext(ctx)

	// Parse topic from args or prompt interactively
	topic, err := h.parseTopic(args)
	if err != nil {
		logger.Error("Failed to get research topic", logging.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get research topic: %w", err)
	}

	if topic == "" {
		return nil, fmt.Errorf("research topic cannot be empty")
	}

	logger.Info("Starting research", logging.String("topic", topic))

	// Ensure research directory exists
	if err := h.ensureResearchDir(); err != nil {
		logger.Error("Failed to create research directory", logging.String("error", err.Error()))
		return nil, fmt.Errorf("failed to create research directory: %w", err)
	}

	// Generate output path
	outputPath := h.generateOutputPath(topic)

	result := &ResearchResult{
		Topic:      topic,
		OutputPath: outputPath,
		Timestamp:  time.Now(),
	}

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		result.Err = ctx.Err()
		return result, ctx.Err()
	default:
	}

	// Load research prompt
	prompt, err := h.loadResearchPrompt()
	if err != nil {
		logger.Error("Failed to load research prompt", logging.String("error", err.Error()))
		result.Err = err
		return result, fmt.Errorf("failed to load research prompt: %w", err)
	}

	logger.Info("Loaded research prompt", logging.String("prompt_path", h.getResearchPromptPath()))

	// Build and execute Claude Code command
	startTime := time.Now()
	exitCode, err := h.executeClaudeCode(ctx, topic, prompt)
	result.Duration = time.Since(startTime)
	result.ExitCode = exitCode

	if err != nil {
		logger.Error("Claude Code execution failed",
			logging.String("error", err.Error()),
			logging.Int("exit_code", exitCode),
		)
		result.Err = err
		return result, fmt.Errorf("claude code execution failed: %w", err)
	}

	logger.Info("Research completed",
		logging.String("topic", topic),
		logging.String("output_path", outputPath),
		logging.Int("exit_code", exitCode),
		logging.Any("duration", result.Duration),
	)

	return result, nil
}

// loadResearchPrompt loads the research prompt from prompts/research.md.
func (h *ResearchHandler) loadResearchPrompt() (string, error) {
	if os.Getenv("MORTY_DEBUG") != "" {
		fmt.Printf("DEBUG: loadResearchPrompt called\n")
	}

	promptPath := h.getResearchPromptPath()

	if os.Getenv("MORTY_DEBUG") != "" {
		fmt.Printf("DEBUG: loadResearchPrompt got path: %s\n", promptPath)
	}

	// Read the prompt file
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read research prompt file %s: %w", promptPath, err)
	}

	return string(content), nil
}

// getResearchPromptPath returns the path to the research prompt file.
func (h *ResearchHandler) getResearchPromptPath() string {
	// Get prompts directory first
	promptsDir := h.paths.GetPromptsDir()

	// Check if there's a specific research prompt path configured
	if h.cfg != nil {
		if promptPath := h.cfg.GetString("prompts.research"); promptPath != "" {
			// If it's an absolute path, use it directly
			if filepath.IsAbs(promptPath) {
				return promptPath
			}
			// If it's a relative path, resolve it relative to prompts dir
			return filepath.Join(promptsDir, filepath.Base(promptPath))
		}
	}

	// Default to research.md in prompts directory
	return filepath.Join(promptsDir, "research.md")
}

// buildClaudeCommand builds the Claude Code command arguments.
func (h *ResearchHandler) buildClaudeCommand(topic, prompt string) []string {
	var args []string

	// Add permission mode plan
	args = append(args, "--permission-mode", "plan")

	// Add the prompt content via -p flag
	// The topic is prepended to the prompt for context
	fullPrompt := fmt.Sprintf("# Research Topic: %s\n\n%s", topic, prompt)
	args = append(args, "-p", fullPrompt)

	return args
}

// executeClaudeCode executes Claude Code with the given topic and prompt.
// Returns the exit code and any error that occurred.
func (h *ResearchHandler) executeClaudeCode(ctx context.Context, topic, prompt string) (int, error) {
	logger := h.logger.WithContext(ctx)

	// Build the full prompt with topic context
	fullPrompt := fmt.Sprintf("# Research Topic: %s\n\n%s", topic, prompt)

	logger.Info("Executing Claude Code in interactive mode",
		logging.String("topic", topic),
		logging.String("cli_path", h.cliCaller.GetCLIPath()),
	)

	// Research mode: use interactive mode (launch Claude Code UI)
	// Pass prompt via stdin to avoid shell parsing issues
	opts := callcli.Options{
		Timeout: 0,
		Stdin:   fullPrompt, // Pass via stdin
		Output: callcli.OutputConfig{
			Mode: callcli.OutputStream, // Stream to terminal for interactive mode
		},
	}

	// Build args for interactive mode: just --permission-mode plan
	// Don't use -p flag (that's for non-interactive mode)
	baseArgs := h.cliCaller.BuildArgs()
	args := append([]string{"--permission-mode", "plan"}, baseArgs...)

	// Execute the command using the base caller
	result, err := h.cliCaller.GetBaseCaller().CallWithOptions(ctx, h.cliCaller.GetCLIPath(), args, opts)

	if err != nil {
		// If result is nil, return error with exit code 1
		if result == nil {
			return 1, err
		}
		return result.ExitCode, err
	}

	if result.ExitCode != 0 {
		return result.ExitCode, fmt.Errorf("claude code exited with code %d: %s", result.ExitCode, result.Stderr)
	}

	return result.ExitCode, nil
}

// parseTopic extracts the topic from command arguments or prompts interactively.
func (h *ResearchHandler) parseTopic(args []string) (string, error) {
	// If arguments provided, use them as the topic
	if len(args) > 0 {
		// Join all arguments with spaces
		return strings.TrimSpace(strings.Join(args, " ")), nil
	}

	// No arguments provided, prompt interactively
	fmt.Print("Enter research topic: ")

	reader := bufio.NewReader(os.Stdin)
	topic, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	// Trim whitespace and newlines
	topic = strings.TrimSpace(topic)

	return topic, nil
}

// ensureResearchDir ensures the .morty/research/ directory exists.
func (h *ResearchHandler) ensureResearchDir() error {
	return h.paths.EnsureResearchDir()
}

// generateOutputPath generates an output file path for the research topic.
func (h *ResearchHandler) generateOutputPath(topic string) string {
	// Sanitize topic for use in filename
	sanitized := h.sanitizeFilename(topic)

	// Add timestamp to filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.md", sanitized, timestamp)

	return filepath.Join(h.paths.GetResearchDir(), filename)
}

// sanitizeFilename converts a topic into a safe filename.
func (h *ResearchHandler) sanitizeFilename(topic string) string {
	// Replace spaces and special characters with underscores
	result := strings.ToLower(topic)

	// Remove or replace unsafe characters
	var sb strings.Builder
	for _, r := range result {
		switch {
		case r >= 'a' && r <= 'z':
			sb.WriteRune(r)
		case r >= '0' && r <= '9':
			sb.WriteRune(r)
		default:
			// Replace spaces and special chars with underscore
			sb.WriteRune('_')
		}
	}

	// Limit length to avoid extremely long filenames
	result = sb.String()
	if len(result) > 50 {
		result = result[:50]
	}

	// Remove trailing underscores
	result = strings.Trim(result, "_")

	// Ensure not empty
	if result == "" {
		result = "research"
	}

	return result
}

// GetResearchDir returns the research directory path.
func (h *ResearchHandler) GetResearchDir() string {
	return h.paths.GetResearchDir()
}

// GetPromptsDir returns the prompts directory path.
func (h *ResearchHandler) GetPromptsDir() string {
	return h.paths.GetPromptsDir()
}

// ValidateResearchResult validates that a research result file exists and is valid.
// It checks if the file exists, is non-empty, and contains valid Markdown content.
func (h *ResearchHandler) ValidateResearchResult(topic string) error {
	// Task 2: Check if research file exists
	researchPath := h.getResearchFilePath(topic)

	info, err := os.Stat(researchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("research result not found for topic '%s': file does not exist at %s", topic, researchPath)
		}
		return fmt.Errorf("failed to access research result for topic '%s': %w", topic, err)
	}

	// Task 3: Verify file is not empty
	if info.Size() == 0 {
		return fmt.Errorf("research result file is empty for topic '%s'", topic)
	}

	// Read and validate content
	content, err := os.ReadFile(researchPath)
	if err != nil {
		return fmt.Errorf("failed to read research result for topic '%s': %w", topic, err)
	}

	// Task 3: Verify content is non-empty (after trimming whitespace)
	if len(strings.TrimSpace(string(content))) == 0 {
		return fmt.Errorf("research result file has no meaningful content for topic '%s'", topic)
	}

	// Task 4: Output research summary
	fmt.Println()
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("  Research Result Summary")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Printf("  Topic: %s\n", topic)
	fmt.Printf("  File:  %s\n", researchPath)
	fmt.Printf("  Size:  %d bytes\n", info.Size())
	fmt.Printf("  Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	fmt.Println("=" + strings.Repeat("=", 60))

	// Show first few lines of content
	lines := strings.Split(string(content), "\n")
	previewLines := 5
	if len(lines) < previewLines {
		previewLines = len(lines)
	}
	if previewLines > 0 {
		fmt.Println("  Preview:")
		for i := 0; i < previewLines; i++ {
			line := strings.TrimSpace(lines[i])
			if line != "" {
				fmt.Printf("    %s\n", line)
			}
		}
		if len(lines) > previewLines {
			fmt.Printf("    ... (%d more lines)\n", len(lines)-previewLines)
		}
	}
	fmt.Println("=" + strings.Repeat("=", 60))

	// Task 5: Prompt next step
	fmt.Println()
	fmt.Println("  Next step:")
	fmt.Println("    Run `morty plan` to create an execution plan")
	fmt.Println("    based on this research.")
	fmt.Println()

	return nil
}

// getResearchFilePath returns the path to a research file for a given topic.
func (h *ResearchHandler) getResearchFilePath(topic string) string {
	sanitized := h.sanitizeFilename(topic)
	return filepath.Join(h.paths.GetResearchDir(), sanitized+".md")
}
