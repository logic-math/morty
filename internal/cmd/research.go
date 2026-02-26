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
}

// ResearchHandler handles the research command.
type ResearchHandler struct {
	cfg    config.Manager
	logger logging.Logger
	paths  *config.Paths
}

// NewResearchHandler creates a new ResearchHandler instance.
func NewResearchHandler(cfg config.Manager, logger logging.Logger) *ResearchHandler {
	return &ResearchHandler{
		cfg:    cfg,
		logger: logger,
		paths:  config.NewPaths(),
	}
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

	logger.Info("Research completed",
		logging.String("topic", topic),
		logging.String("output_path", outputPath),
	)

	return result, nil
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
