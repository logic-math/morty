#!/bin/bash
#
# Format existing JSON log files to human-readable text format
#

set -e

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check if log directory exists
LOG_DIR="${1:-.morty/logs}"

if [ ! -d "$LOG_DIR" ]; then
    echo "Error: Log directory not found: $LOG_DIR"
    exit 1
fi

print_info "Processing logs in: $LOG_DIR"

# Create a Go program to format logs
cat > /tmp/format_log.go << 'EOF'
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Event struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
	Message   *EventMessage   `json:"message,omitempty"`
}

type EventMessage struct {
	Role       string         `json:"role,omitempty"`
	Content    []ContentBlock `json:"content,omitempty"`
	StopReason string         `json:"stop_reason,omitempty"`
	Usage      *TokenUsage    `json:"usage,omitempty"`
}

type ContentBlock struct {
	Type  string      `json:"type"`
	Text  string      `json:"text,omitempty"`
	Name  string      `json:"name,omitempty"`
	Input interface{} `json:"input,omitempty"`
}

type TokenUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: format_log <log_file>")
		os.Exit(1)
	}

	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	formatEventStream(string(content))
}

func formatEventStream(jsonStream string) {
	fmt.Println("=== Claude Code Event Stream ===")
	fmt.Printf("Timestamp: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	var events []json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(jsonStream))
	if err := decoder.Decode(&events); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		return
	}

	for i, rawEvent := range events {
		var event Event
		if err := json.Unmarshal(rawEvent, &event); err != nil {
			fmt.Printf("[%04d] ERROR: Failed to parse event: %v\n", i+1, err)
			continue
		}

		formatEvent(i+1, &event)
	}

	fmt.Printf("\n=== Total Events: %d ===\n", len(events))
}

func formatEvent(num int, event *Event) {
	timestamp := extractTimestamp(event)
	eventType := formatEventType(event)
	summary := extractSummary(event)

	fmt.Printf("[%04d] %s | %-20s | %s\n", num, timestamp, eventType, summary)
}

func extractTimestamp(event *Event) string {
	if event.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, event.Timestamp); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
		return event.Timestamp
	}
	return time.Now().Format("2006-01-02 15:04:05")
}

func formatEventType(event *Event) string {
	if event.Subtype != "" {
		return fmt.Sprintf("%s/%s", event.Type, event.Subtype)
	}
	return event.Type
}

func extractSummary(event *Event) string {
	switch event.Type {
	case "system":
		return formatSystemEvent(event)
	case "assistant":
		return formatAssistantEvent(event)
	case "user":
		return "Tool results"
	case "result":
		return "Execution completed"
	default:
		return fmt.Sprintf("Unknown: %s", event.Type)
	}
}

func formatSystemEvent(event *Event) string {
	switch event.Subtype {
	case "init":
		return "Session initialized"
	case "error":
		return "System error"
	default:
		return fmt.Sprintf("System: %s", event.Subtype)
	}
}

func formatAssistantEvent(event *Event) string {
	if event.Message == nil {
		return "Assistant message (no content)"
	}

	var texts []string
	var tools []string

	for _, block := range event.Message.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				text := block.Text
				if len(text) > 80 {
					text = text[:77] + "..."
				}
				texts = append(texts, text)
			}
		case "tool_use":
			tools = append(tools, block.Name)
		}
	}

	parts := []string{}
	if len(texts) > 0 {
		parts = append(parts, strings.Join(texts, " | "))
	}
	if len(tools) > 0 {
		parts = append(parts, fmt.Sprintf("Tools: [%s]", strings.Join(tools, ", ")))
	}

	if event.Message.Usage != nil {
		parts = append(parts, fmt.Sprintf("Tokens(in:%d out:%d)",
			event.Message.Usage.InputTokens,
			event.Message.Usage.OutputTokens))
	}

	if len(parts) == 0 {
		return "Assistant message (empty)"
	}

	return strings.Join(parts, " | ")
}
EOF

# Compile the formatter
print_info "Compiling log formatter..."
go build -o /tmp/format_log /tmp/format_log.go

# Process each log file
count=0
for logfile in "$LOG_DIR"/*.log; do
    if [ -f "$logfile" ]; then
        basename=$(basename "$logfile")
        output="${logfile}.formatted"

        print_info "Formatting: $basename"
        /tmp/format_log "$logfile" > "$output" 2>/dev/null || {
            echo "  Skipped (not JSON format)"
            rm -f "$output"
            continue
        }

        print_success "Created: ${output}"
        ((count++))
    fi
done

print_success "Formatted $count log files"

# Cleanup
rm -f /tmp/format_log.go /tmp/format_log
