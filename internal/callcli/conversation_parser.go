// Package callcli provides functionality for executing external CLI commands.
package callcli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Event represents a single event in the Claude Code event stream.
type Event struct {
	Type              string                 `json:"type"`
	Subtype           string                 `json:"subtype,omitempty"`
	Message           *Message               `json:"message,omitempty"`
	SessionID         string                 `json:"session_id,omitempty"`
	UUID              string                 `json:"uuid,omitempty"`
	ParentToolUseID   string                 `json:"parent_tool_use_id,omitempty"`
	ToolUseResult     *ToolUseResultData     `json:"tool_use_result,omitempty"`
	Result            string                 `json:"result,omitempty"`
	DurationMs        int64                  `json:"duration_ms,omitempty"`
	NumTurns          int                    `json:"num_turns,omitempty"`
	Usage             *Usage                 `json:"usage,omitempty"`
	ModelUsage        map[string]*ModelUsage `json:"modelUsage,omitempty"`
	TotalCostUSD      float64                `json:"total_cost_usd,omitempty"`
	Tools             []string               `json:"tools,omitempty"`
	MCPServers        []string               `json:"mcp_servers,omitempty"`
	Model             string                 `json:"model,omitempty"`
	PermissionMode    string                 `json:"permissionMode,omitempty"`
	SlashCommands     []string               `json:"slash_commands,omitempty"`
	Agents            []string               `json:"agents,omitempty"`
	Skills            []string               `json:"skills,omitempty"`
	ClaudeCodeVersion string                 `json:"claude_code_version,omitempty"`
}

// Message represents a message within an event.
type Message struct {
	Role         string                   `json:"role"`
	Content      []ContentBlock           `json:"content"`
	StopReason   string                   `json:"stop_reason,omitempty"`
	StopSequence string                   `json:"stop_sequence,omitempty"`
	Usage        *Usage                   `json:"usage,omitempty"`
	ID           string                   `json:"id,omitempty"`
	Type         string                   `json:"type,omitempty"`
	Model        string                   `json:"model,omitempty"`
}

// ContentBlock represents a content block in a message.
type ContentBlock struct {
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
	Content   string                 `json:"content,omitempty"`
	IsError   bool                   `json:"is_error,omitempty"`
}

// Usage represents token usage information.
type Usage struct {
	InputTokens               int `json:"input_tokens"`
	OutputTokens              int `json:"output_tokens"`
	CacheCreationInputTokens  int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens      int `json:"cache_read_input_tokens"`
}

// ModelUsage represents usage statistics for a specific model.
type ModelUsage struct {
	InputTokens              int     `json:"inputTokens"`
	OutputTokens             int     `json:"outputTokens"`
	CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
	CostUSD                  float64 `json:"costUSD"`
}

// ToolUseResultData represents tool execution result data.
type ToolUseResultData struct {
	Type        string                 `json:"type,omitempty"`
	Stdout      string                 `json:"stdout,omitempty"`
	Stderr      string                 `json:"stderr,omitempty"`
	Interrupted bool                   `json:"interrupted,omitempty"`
	IsImage     bool                   `json:"isImage,omitempty"`
	File        *FileData              `json:"file,omitempty"`
	FilePath    string                 `json:"filePath,omitempty"`
	OldString   string                 `json:"oldString,omitempty"`
	NewString   string                 `json:"newString,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// FileData represents file information in tool results.
type FileData struct {
	FilePath   string `json:"filePath"`
	Content    string `json:"content"`
	NumLines   int    `json:"numLines"`
	StartLine  int    `json:"startLine"`
	TotalLines int    `json:"totalLines"`
}

// ConversationMessage represents a single message in the Claude Code conversation (legacy).
type ConversationMessage struct {
	Role      string                 `json:"role"`      // "user" or "assistant"
	Content   interface{}            `json:"content"`   // Can be string or array of content blocks
	Timestamp time.Time              `json:"timestamp,omitempty"`
	Type      string                 `json:"type,omitempty"`
	ToolUse   *ToolUseBlock          `json:"tool_use,omitempty"`
	ToolCalls []ToolCall             `json:"tool_calls,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCall represents a tool call made by the assistant.
type ToolCall struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Function   string                 `json:"function"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolUseBlock represents a tool use content block.
type ToolUseBlock struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// ConversationData represents the full Claude Code conversation (event stream format).
type ConversationData struct {
	Events           []Event                `json:"events"`
	SessionID        string                 `json:"session_id,omitempty"`
	Model            string                 `json:"model,omitempty"`
	TotalCostUSD     float64                `json:"total_cost_usd,omitempty"`
	TotalDurationMs  int64                  `json:"total_duration_ms,omitempty"`
	NumTurns         int                    `json:"num_turns,omitempty"`
	TotalInputTokens int                    `json:"total_input_tokens,omitempty"`
	TotalOutputTokens int                   `json:"total_output_tokens,omitempty"`
	ModelUsage       map[string]*ModelUsage `json:"model_usage,omitempty"`

	// Legacy fields for backward compatibility
	Messages    []ConversationMessage  `json:"messages,omitempty"`
	SystemInfo  map[string]interface{} `json:"system_info,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	StartTime   time.Time              `json:"start_time,omitempty"`
	EndTime     time.Time              `json:"end_time,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	TokensUsed  int                    `json:"tokens_used,omitempty"`
	TokensLimit int                    `json:"tokens_limit,omitempty"`
}

// FormattedLog represents a formatted log entry extracted from conversation.
type FormattedLog struct {
	Timestamp   time.Time              `json:"timestamp"`
	MessageType string                 `json:"message_type"` // "user_message", "assistant_text", "tool_call", "tool_result", "error"
	Role        string                 `json:"role"`
	Content     string                 `json:"content"`
	ToolName    string                 `json:"tool_name,omitempty"`
	ToolParams  map[string]interface{} `json:"tool_params,omitempty"`
	ToolResult  string                 `json:"tool_result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ConversationParser parses Claude Code conversation JSON and extracts useful logs.
type ConversationParser struct {
	logDir string
}

// NewConversationParser creates a new conversation parser.
func NewConversationParser(logDir string) *ConversationParser {
	return &ConversationParser{
		logDir: logDir,
	}
}

// ParseAndSave parses the conversation JSON and saves formatted logs to disk.
//
// Parameters:
//   - jsonData: The raw JSON string from Claude Code
//   - module: The module name (for organizing logs)
//   - job: The job name (for organizing logs)
//
// Returns:
//   - The path to the saved log file
//   - An error if parsing or saving fails
func (cp *ConversationParser) ParseAndSave(jsonData string, module, job string) (string, error) {
	// Parse the JSON
	conversation, err := cp.Parse(jsonData)
	if err != nil {
		return "", fmt.Errorf("failed to parse conversation JSON: %w", err)
	}

	// Extract formatted logs
	logs := cp.ExtractLogs(conversation)

	// Generate log filename
	timestamp := time.Now().Format("20060102_150405")
	sanitizedModule := sanitizeFilename(module)
	sanitizedJob := sanitizeFilename(job)
	logFilename := fmt.Sprintf("%s_%s_%s.log", sanitizedModule, sanitizedJob, timestamp)
	logPath := filepath.Join(cp.logDir, logFilename)

	// Ensure log directory exists
	if err := os.MkdirAll(cp.logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// Write formatted logs
	if err := cp.WriteLogs(logPath, logs, conversation); err != nil {
		return "", fmt.Errorf("failed to write logs: %w", err)
	}

	return logPath, nil
}

// Parse parses the raw JSON data into ConversationData structure.
// Supports both event stream format (array) and legacy format (object).
func (cp *ConversationParser) Parse(jsonData string) (*ConversationData, error) {
	// Trim whitespace
	jsonData = strings.TrimSpace(jsonData)

	// Check if it's an array (event stream format)
	if strings.HasPrefix(jsonData, "[") {
		return cp.parseEventStream(jsonData)
	}

	// Otherwise try legacy format (object with messages array)
	return cp.parseLegacyFormat(jsonData)
}

// parseEventStream parses Claude Code event stream format.
func (cp *ConversationParser) parseEventStream(jsonData string) (*ConversationData, error) {
	var events []Event
	if err := json.Unmarshal([]byte(jsonData), &events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event stream: %w", err)
	}

	conversation := &ConversationData{
		Events: events,
	}

	// Extract metadata from events
	for _, event := range events {
		if event.SessionID != "" && conversation.SessionID == "" {
			conversation.SessionID = event.SessionID
		}
		if event.Model != "" && conversation.Model == "" {
			conversation.Model = event.Model
		}
		if event.Type == "result" {
			conversation.TotalCostUSD = event.TotalCostUSD
			conversation.TotalDurationMs = event.DurationMs
			conversation.NumTurns = event.NumTurns
			if event.Usage != nil {
				conversation.TotalInputTokens = event.Usage.InputTokens
				conversation.TotalOutputTokens = event.Usage.OutputTokens
			}
			if event.ModelUsage != nil {
				conversation.ModelUsage = event.ModelUsage
			}
		}
	}

	return conversation, nil
}

// parseLegacyFormat parses legacy conversation format.
func (cp *ConversationParser) parseLegacyFormat(jsonData string) (*ConversationData, error) {
	var conversation ConversationData
	if err := json.Unmarshal([]byte(jsonData), &conversation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal legacy format: %w", err)
	}
	return &conversation, nil
}

// ExtractLogs extracts formatted log entries from the conversation.
func (cp *ConversationParser) ExtractLogs(conversation *ConversationData) []FormattedLog {
	var logs []FormattedLog
	timestamp := time.Now()

	// Handle event stream format
	if len(conversation.Events) > 0 {
		return cp.extractLogsFromEvents(conversation.Events, timestamp)
	}

	// Handle legacy format
	for i, msg := range conversation.Messages {
		// Generate timestamp for each message (if not provided)
		msgTimestamp := msg.Timestamp
		if msgTimestamp.IsZero() {
			msgTimestamp = timestamp.Add(time.Duration(i) * time.Second)
		}

		// Process based on role
		switch msg.Role {
		case "user":
			logs = append(logs, cp.extractUserMessage(msg, msgTimestamp)...)
		case "assistant":
			logs = append(logs, cp.extractAssistantMessage(msg, msgTimestamp)...)
		}
	}

	return logs
}

// extractLogsFromEvents extracts logs from event stream format.
func (cp *ConversationParser) extractLogsFromEvents(events []Event, baseTime time.Time) []FormattedLog {
	var logs []FormattedLog

	for i, event := range events {
		eventTime := baseTime.Add(time.Duration(i) * time.Second)

		switch event.Type {
		case "system":
			// System initialization event
			if event.Subtype == "init" {
				logs = append(logs, FormattedLog{
					Timestamp:   eventTime,
					MessageType: "system_init",
					Role:        "system",
					Content:     fmt.Sprintf("Session initialized: %s", event.SessionID),
					Metadata: map[string]interface{}{
						"model":              event.Model,
						"permission_mode":    event.PermissionMode,
						"claude_code_version": event.ClaudeCodeVersion,
						"tools":              event.Tools,
					},
				})
			}

		case "assistant":
			// Assistant message
			if event.Message != nil {
				logs = append(logs, cp.extractLogsFromMessage(event.Message, eventTime)...)
			}

		case "user":
			// User message (usually tool results)
			if event.Message != nil {
				for _, block := range event.Message.Content {
					if block.Type == "tool_result" {
						logs = append(logs, FormattedLog{
							Timestamp:   eventTime,
							MessageType: "tool_result",
							Role:        "user",
							Content:     truncateString(block.Content, 500),
							ToolName:    "", // Tool name is in the corresponding tool_use
							ToolResult:  block.Content,
							Metadata: map[string]interface{}{
								"tool_use_id": block.ToolUseID,
								"is_error":    block.IsError,
							},
						})
					}
				}
			}

		case "result":
			// Final result
			logs = append(logs, FormattedLog{
				Timestamp:   eventTime,
				MessageType: "session_result",
				Role:        "system",
				Content:     event.Result,
				Metadata: map[string]interface{}{
					"duration_ms":    event.DurationMs,
					"num_turns":      event.NumTurns,
					"total_cost_usd": event.TotalCostUSD,
					"usage":          event.Usage,
				},
			})
		}
	}

	return logs
}

// extractLogsFromMessage extracts logs from a Message structure.
func (cp *ConversationParser) extractLogsFromMessage(msg *Message, timestamp time.Time) []FormattedLog {
	var logs []FormattedLog

	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				logs = append(logs, FormattedLog{
					Timestamp:   timestamp,
					MessageType: "assistant_text",
					Role:        "assistant",
					Content:     block.Text,
				})
			}

		case "tool_use":
			logs = append(logs, FormattedLog{
				Timestamp:   timestamp,
				MessageType: "tool_call",
				Role:        "assistant",
				Content:     fmt.Sprintf("Tool call: %s", block.Name),
				ToolName:    block.Name,
				ToolParams:  block.Input,
				Metadata: map[string]interface{}{
					"tool_use_id": block.ID,
				},
			})
		}
	}

	return logs
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

// extractUserMessage extracts log entries from a user message.
func (cp *ConversationParser) extractUserMessage(msg ConversationMessage, timestamp time.Time) []FormattedLog {
	var logs []FormattedLog

	content := cp.extractContentString(msg.Content)
	if content != "" {
		logs = append(logs, FormattedLog{
			Timestamp:   timestamp,
			MessageType: "user_message",
			Role:        "user",
			Content:     content,
			Metadata:    msg.Metadata,
		})
	}

	return logs
}

// extractAssistantMessage extracts log entries from an assistant message.
func (cp *ConversationParser) extractAssistantMessage(msg ConversationMessage, timestamp time.Time) []FormattedLog {
	var logs []FormattedLog

	// Extract text content
	content := cp.extractContentString(msg.Content)
	if content != "" {
		logs = append(logs, FormattedLog{
			Timestamp:   timestamp,
			MessageType: "assistant_text",
			Role:        "assistant",
			Content:     content,
			Metadata:    msg.Metadata,
		})
	}

	// Extract tool calls
	if msg.ToolUse != nil {
		logs = append(logs, FormattedLog{
			Timestamp:   timestamp,
			MessageType: "tool_call",
			Role:        "assistant",
			Content:     fmt.Sprintf("Tool call: %s", msg.ToolUse.Name),
			ToolName:    msg.ToolUse.Name,
			ToolParams:  msg.ToolUse.Input,
			Metadata:    msg.Metadata,
		})
	}

	for _, toolCall := range msg.ToolCalls {
		logs = append(logs, FormattedLog{
			Timestamp:   timestamp,
			MessageType: "tool_call",
			Role:        "assistant",
			Content:     fmt.Sprintf("Tool call: %s", toolCall.Function),
			ToolName:    toolCall.Function,
			ToolParams:  toolCall.Parameters,
			Metadata:    msg.Metadata,
		})
	}

	return logs
}

// extractContentString extracts string content from various content formats.
func (cp *ConversationParser) extractContentString(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, item := range v {
			if block, ok := item.(map[string]interface{}); ok {
				if text, ok := block["text"].(string); ok {
					parts = append(parts, text)
				} else if typ, ok := block["type"].(string); ok && typ == "text" {
					if text, ok := block["content"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return fmt.Sprintf("%v", content)
	}
}

// WriteLogs writes the formatted logs to a file.
func (cp *ConversationParser) WriteLogs(logPath string, logs []FormattedLog, conversation *ConversationData) error {
	file, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "=== Claude Code Conversation Log ===\n")
	fmt.Fprintf(file, "Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	if conversation.SessionID != "" {
		fmt.Fprintf(file, "Session ID: %s\n", conversation.SessionID)
	}
	if conversation.Model != "" {
		fmt.Fprintf(file, "Model: %s\n", conversation.Model)
	}
	if conversation.TotalInputTokens > 0 || conversation.TotalOutputTokens > 0 {
		fmt.Fprintf(file, "Tokens: %d input + %d output = %d total\n",
			conversation.TotalInputTokens,
			conversation.TotalOutputTokens,
			conversation.TotalInputTokens+conversation.TotalOutputTokens)
	}
	if conversation.TotalCostUSD > 0 {
		fmt.Fprintf(file, "Total Cost: $%.4f USD\n", conversation.TotalCostUSD)
	}
	if conversation.TotalDurationMs > 0 {
		fmt.Fprintf(file, "Duration: %.2f seconds\n", float64(conversation.TotalDurationMs)/1000.0)
	}
	if conversation.NumTurns > 0 {
		fmt.Fprintf(file, "Turns: %d\n", conversation.NumTurns)
	}
	fmt.Fprintf(file, "Total Events: %d\n", len(logs))
	fmt.Fprintf(file, "=====================================\n\n")

	// Write each log entry
	for _, log := range logs {
		cp.writeLogEntry(file, log)
	}

	// Write summary statistics
	cp.writeStatistics(file, logs)

	// Write model usage breakdown if available
	if len(conversation.ModelUsage) > 0 {
		cp.writeModelUsage(file, conversation.ModelUsage)
	}

	return nil
}

// writeLogEntry writes a single log entry to the file.
func (cp *ConversationParser) writeLogEntry(file *os.File, log FormattedLog) {
	timestamp := log.Timestamp.Format("15:04:05")

	switch log.MessageType {
	case "system_init":
		fmt.Fprintf(file, "[%s] SYSTEM INIT:\n%s\n", timestamp, log.Content)
		if log.Metadata != nil {
			if model, ok := log.Metadata["model"].(string); ok {
				fmt.Fprintf(file, "  Model: %s\n", model)
			}
			if version, ok := log.Metadata["claude_code_version"].(string); ok {
				fmt.Fprintf(file, "  Claude Code Version: %s\n", version)
			}
			if tools, ok := log.Metadata["tools"].([]string); ok && len(tools) > 0 {
				fmt.Fprintf(file, "  Available Tools: %d\n", len(tools))
			}
		}
		fmt.Fprintf(file, "\n")

	case "user_message":
		fmt.Fprintf(file, "[%s] USER:\n%s\n\n", timestamp, log.Content)

	case "assistant_text":
		fmt.Fprintf(file, "[%s] ASSISTANT:\n%s\n\n", timestamp, log.Content)

	case "tool_call":
		fmt.Fprintf(file, "[%s] TOOL CALL: %s\n", timestamp, log.ToolName)
		if len(log.ToolParams) > 0 {
			paramsJSON, _ := json.MarshalIndent(log.ToolParams, "  ", "  ")
			fmt.Fprintf(file, "  Parameters:\n  %s\n", string(paramsJSON))
		}
		fmt.Fprintf(file, "\n")

	case "tool_result":
		fmt.Fprintf(file, "[%s] TOOL RESULT\n", timestamp)
		if log.ToolResult != "" {
			// Truncate long results
			result := log.ToolResult
			if len(result) > 500 {
				result = result[:500] + "... (truncated)"
			}
			fmt.Fprintf(file, "  %s\n", result)
		}
		if log.Metadata != nil {
			if isError, ok := log.Metadata["is_error"].(bool); ok && isError {
				fmt.Fprintf(file, "  [ERROR]\n")
			}
		}
		fmt.Fprintf(file, "\n")

	case "session_result":
		fmt.Fprintf(file, "[%s] SESSION COMPLETED\n", timestamp)
		if log.Content != "" {
			fmt.Fprintf(file, "%s\n", log.Content)
		}
		if log.Metadata != nil {
			if duration, ok := log.Metadata["duration_ms"].(int64); ok {
				fmt.Fprintf(file, "  Duration: %.2f seconds\n", float64(duration)/1000.0)
			}
			if turns, ok := log.Metadata["num_turns"].(int); ok {
				fmt.Fprintf(file, "  Turns: %d\n", turns)
			}
			if cost, ok := log.Metadata["total_cost_usd"].(float64); ok {
				fmt.Fprintf(file, "  Cost: $%.4f USD\n", cost)
			}
		}
		fmt.Fprintf(file, "\n")

	case "error":
		fmt.Fprintf(file, "[%s] ERROR:\n%s\n\n", timestamp, log.Error)
	}
}

// writeModelUsage writes model usage breakdown to the file.
func (cp *ConversationParser) writeModelUsage(file *os.File, modelUsage map[string]*ModelUsage) {
	fmt.Fprintf(file, "\n=====================================\n")
	fmt.Fprintf(file, "=== Model Usage Breakdown ===\n")
	fmt.Fprintf(file, "=====================================\n")

	for modelName, usage := range modelUsage {
		fmt.Fprintf(file, "\n%s:\n", modelName)
		fmt.Fprintf(file, "  Input Tokens: %d\n", usage.InputTokens)
		fmt.Fprintf(file, "  Output Tokens: %d\n", usage.OutputTokens)
		if usage.CacheReadInputTokens > 0 {
			fmt.Fprintf(file, "  Cache Read Tokens: %d\n", usage.CacheReadInputTokens)
		}
		if usage.CacheCreationInputTokens > 0 {
			fmt.Fprintf(file, "  Cache Creation Tokens: %d\n", usage.CacheCreationInputTokens)
		}
		fmt.Fprintf(file, "  Cost: $%.4f USD\n", usage.CostUSD)
	}
}

// writeStatistics writes summary statistics to the file.
func (cp *ConversationParser) writeStatistics(file *os.File, logs []FormattedLog) {
	fmt.Fprintf(file, "\n=====================================\n")
	fmt.Fprintf(file, "=== Statistics ===\n")
	fmt.Fprintf(file, "=====================================\n")

	// Count by message type
	counts := make(map[string]int)
	toolCounts := make(map[string]int)

	for _, log := range logs {
		counts[log.MessageType]++
		if log.MessageType == "tool_call" {
			toolCounts[log.ToolName]++
		}
	}

	fmt.Fprintf(file, "Message Types:\n")
	for typ, count := range counts {
		fmt.Fprintf(file, "  - %s: %d\n", typ, count)
	}

	if len(toolCounts) > 0 {
		fmt.Fprintf(file, "\nTool Usage:\n")
		for tool, count := range toolCounts {
			fmt.Fprintf(file, "  - %s: %d\n", tool, count)
		}
	}
}

// sanitizeFilename removes invalid characters from filename.
func sanitizeFilename(name string) string {
	// Replace invalid characters with underscore
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// ParseFromFile reads and parses conversation JSON from a file.
func (cp *ConversationParser) ParseFromFile(jsonFile string) (*ConversationData, error) {
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	return cp.Parse(string(data))
}

// SaveFormattedJSON saves the conversation as formatted JSON for easier reading.
func (cp *ConversationParser) SaveFormattedJSON(conversation *ConversationData, module, job string) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	sanitizedModule := sanitizeFilename(module)
	sanitizedJob := sanitizeFilename(job)
	jsonFilename := fmt.Sprintf("%s_%s_%s.json", sanitizedModule, sanitizedJob, timestamp)
	jsonPath := filepath.Join(cp.logDir, jsonFilename)

	// Ensure log directory exists
	if err := os.MkdirAll(cp.logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// Marshal with indentation
	data, err := json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write JSON file: %w", err)
	}

	return jsonPath, nil
}
