package executor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Event represents a Claude Code event from the JSON stream
type Event struct {
	Type       string          `json:"type"`
	Subtype    string          `json:"subtype,omitempty"`
	Timestamp  string          `json:"timestamp,omitempty"`
	Message    *EventMessage   `json:"message,omitempty"`
	RawMessage json.RawMessage `json:"-"` // Store raw for debugging
}

// EventMessage represents the message field in events
type EventMessage struct {
	Role     string         `json:"role,omitempty"`
	Content  []ContentBlock `json:"content,omitempty"`
	StopReason string       `json:"stop_reason,omitempty"`
	Usage    *TokenUsage    `json:"usage,omitempty"`
}

// ContentBlock represents a content block in a message
type ContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Name  string `json:"name,omitempty"`
	Input interface{} `json:"input,omitempty"`
}

// TokenUsage represents token usage statistics
type TokenUsage struct {
	InputTokens            int `json:"input_tokens"`
	OutputTokens           int `json:"output_tokens"`
	CacheReadInputTokens   int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// EventFormatter formats Claude Code events into human-readable text
type EventFormatter struct {
	writer io.Writer
	eventCount int
}

// NewEventFormatter creates a new event formatter
func NewEventFormatter(w io.Writer) *EventFormatter {
	return &EventFormatter{
		writer: w,
		eventCount: 0,
	}
}

// FormatEventStream parses and formats a JSON event stream
func (f *EventFormatter) FormatEventStream(jsonStream string) error {
	// Write header
	fmt.Fprintf(f.writer, "=== Claude Code Event Stream ===\n")
	fmt.Fprintf(f.writer, "Timestamp: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// Parse JSON array
	var events []json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(jsonStream))
	if err := decoder.Decode(&events); err != nil {
		// If not a JSON array, try line-by-line parsing
		return f.formatLineByLine(jsonStream)
	}

	// Process each event
	for i, rawEvent := range events {
		var event Event
		if err := json.Unmarshal(rawEvent, &event); err != nil {
			fmt.Fprintf(f.writer, "[%04d] ERROR: Failed to parse event: %v\n", i+1, err)
			continue
		}

		f.formatEvent(i+1, &event)
		f.eventCount++
	}

	// Write footer
	fmt.Fprintf(f.writer, "\n=== Total Events: %d ===\n", f.eventCount)
	return nil
}

// formatLineByLine handles line-by-line JSON parsing
func (f *EventFormatter) formatLineByLine(content string) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		lineNum++
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Not a JSON line, skip
			continue
		}

		f.formatEvent(lineNum, &event)
		f.eventCount++
	}

	return scanner.Err()
}

// formatEvent formats a single event
func (f *EventFormatter) formatEvent(num int, event *Event) {
	timestamp := f.extractTimestamp(event)
	eventType := f.formatEventType(event)
	summary := f.extractSummary(event)

	// Format: [NNNN] YYYY-MM-DD HH:MM:SS | TYPE | Summary
	fmt.Fprintf(f.writer, "[%04d] %s | %-20s | %s\n", num, timestamp, eventType, summary)
}

// extractTimestamp extracts or generates a timestamp for the event
func (f *EventFormatter) extractTimestamp(event *Event) string {
	if event.Timestamp != "" {
		// Try to parse and format
		if t, err := time.Parse(time.RFC3339, event.Timestamp); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
		return event.Timestamp
	}
	return time.Now().Format("2006-01-02 15:04:05")
}

// formatEventType formats the event type
func (f *EventFormatter) formatEventType(event *Event) string {
	if event.Subtype != "" {
		return fmt.Sprintf("%s/%s", event.Type, event.Subtype)
	}
	return event.Type
}

// extractSummary extracts a summary from the event
func (f *EventFormatter) extractSummary(event *Event) string {
	switch event.Type {
	case "system":
		return f.formatSystemEvent(event)
	case "assistant":
		return f.formatAssistantEvent(event)
	case "user":
		return f.formatUserEvent(event)
	case "result":
		return f.formatResultEvent(event)
	default:
		return fmt.Sprintf("Unknown event type: %s", event.Type)
	}
}

// formatSystemEvent formats system events
func (f *EventFormatter) formatSystemEvent(event *Event) string {
	switch event.Subtype {
	case "init":
		return "Session initialized"
	case "error":
		return "System error occurred"
	default:
		return fmt.Sprintf("System event: %s", event.Subtype)
	}
}

// formatAssistantEvent formats assistant events
func (f *EventFormatter) formatAssistantEvent(event *Event) string {
	if event.Message == nil {
		return "Assistant message (no content)"
	}

	// Extract text content
	var texts []string
	var tools []string

	for _, block := range event.Message.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				// Truncate long text
				text := block.Text
				if len(text) > 100 {
					text = text[:97] + "..."
				}
				texts = append(texts, text)
			}
		case "tool_use":
			tools = append(tools, block.Name)
		}
	}

	// Build summary
	parts := []string{}
	if len(texts) > 0 {
		parts = append(parts, fmt.Sprintf("Text: %s", strings.Join(texts, " | ")))
	}
	if len(tools) > 0 {
		parts = append(parts, fmt.Sprintf("Tools: [%s]", strings.Join(tools, ", ")))
	}

	// Add token usage if available
	if event.Message.Usage != nil {
		parts = append(parts, fmt.Sprintf("Tokens(in:%d out:%d cache:%d)",
			event.Message.Usage.InputTokens,
			event.Message.Usage.OutputTokens,
			event.Message.Usage.CacheReadInputTokens))
	}

	if len(parts) == 0 {
		return "Assistant message (empty)"
	}

	return strings.Join(parts, " | ")
}

// formatUserEvent formats user events (tool results)
func (f *EventFormatter) formatUserEvent(event *Event) string {
	if event.Message == nil {
		return "User message (no content)"
	}

	var toolResults []string
	for _, block := range event.Message.Content {
		if block.Type == "tool_result" {
			toolName := "unknown"
			if block.Name != "" {
				toolName = block.Name
			}
			toolResults = append(toolResults, toolName)
		}
	}

	if len(toolResults) > 0 {
		return fmt.Sprintf("Tool results: [%s]", strings.Join(toolResults, ", "))
	}

	return "User message"
}

// formatResultEvent formats result events
func (f *EventFormatter) formatResultEvent(event *Event) string {
	// Try to extract stop_reason from message
	if event.Message != nil && event.Message.StopReason != "" {
		return fmt.Sprintf("Execution completed: %s", event.Message.StopReason)
	}
	return "Execution result"
}
