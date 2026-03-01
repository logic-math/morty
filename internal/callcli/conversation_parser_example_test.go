package callcli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestParseRealClaudeCodeOutput tests parsing actual Claude Code output format.
func TestParseRealClaudeCodeOutput(t *testing.T) {
	// This is a simplified version of the actual Claude Code output format
	realOutput := `[
		{
			"type": "system",
			"subtype": "init",
			"session_id": "test-session-123",
			"model": "claude-opus-4-6[1m]",
			"tools": ["Read", "Write", "Bash"]
		},
		{
			"type": "assistant",
			"message": {
				"role": "assistant",
				"content": [
					{
						"type": "text",
						"text": "I'll help you with that task."
					}
				],
				"id": "msg_123",
				"model": "claude-opus-4-6"
			},
			"session_id": "test-session-123"
		},
		{
			"type": "assistant",
			"message": {
				"role": "assistant",
				"content": [
					{
						"type": "tool_use",
						"id": "toolu_456",
						"name": "Read",
						"input": {
							"file_path": "/path/to/file.txt"
						}
					}
				],
				"id": "msg_124",
				"model": "claude-opus-4-6"
			},
			"session_id": "test-session-123"
		},
		{
			"type": "user",
			"message": {
				"role": "user",
				"content": [
					{
						"type": "tool_result",
						"tool_use_id": "toolu_456",
						"content": "File content here..."
					}
				]
			},
			"session_id": "test-session-123"
		},
		{
			"type": "result",
			"result": "Task completed successfully",
			"duration_ms": 5000,
			"num_turns": 2,
			"total_cost_usd": 0.05,
			"usage": {
				"input_tokens": 1000,
				"output_tokens": 500,
				"cache_creation_input_tokens": 0,
				"cache_read_input_tokens": 0
			},
			"session_id": "test-session-123"
		}
	]`

	tmpDir := t.TempDir()
	parser := NewConversationParser(tmpDir)

	// Test parsing
	conversation, err := parser.Parse(realOutput)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify parsed data
	if len(conversation.Events) != 5 {
		t.Errorf("Expected 5 events, got %d", len(conversation.Events))
	}

	if conversation.SessionID != "test-session-123" {
		t.Errorf("Expected session_id 'test-session-123', got '%s'", conversation.SessionID)
	}

	if conversation.Model != "claude-opus-4-6[1m]" {
		t.Errorf("Expected model 'claude-opus-4-6[1m]', got '%s'", conversation.Model)
	}

	if conversation.TotalCostUSD != 0.05 {
		t.Errorf("Expected cost 0.05, got %.2f", conversation.TotalCostUSD)
	}

	// Test log extraction
	logs := parser.ExtractLogs(conversation)
	if len(logs) == 0 {
		t.Error("Expected at least one log entry")
	}

	// Verify log types
	hasSystemInit := false
	hasAssistantText := false
	hasToolCall := false
	hasToolResult := false
	hasSessionResult := false

	for _, log := range logs {
		switch log.MessageType {
		case "system_init":
			hasSystemInit = true
		case "assistant_text":
			hasAssistantText = true
		case "tool_call":
			hasToolCall = true
			if log.ToolName != "Read" {
				t.Errorf("Expected tool name 'Read', got '%s'", log.ToolName)
			}
		case "tool_result":
			hasToolResult = true
		case "session_result":
			hasSessionResult = true
		}
	}

	if !hasSystemInit {
		t.Error("Missing system_init log entry")
	}
	if !hasAssistantText {
		t.Error("Missing assistant_text log entry")
	}
	if !hasToolCall {
		t.Error("Missing tool_call log entry")
	}
	if !hasToolResult {
		t.Error("Missing tool_result log entry")
	}
	if !hasSessionResult {
		t.Error("Missing session_result log entry")
	}

	// Test saving logs
	logPath, err := parser.ParseAndSave(realOutput, "test_module", "test_job")
	if err != nil {
		t.Fatalf("Failed to save logs: %v", err)
	}

	// Verify log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created at %s", logPath)
	}

	// Read and verify log content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	requiredStrings := []string{
		"Claude Code Conversation Log",
		"Session ID: test-session-123",
		"Model: claude-opus-4-6",
		"SYSTEM INIT",
		"ASSISTANT:",
		"TOOL CALL: Read",
		"TOOL RESULT",
		"SESSION COMPLETED",
	}

	for _, required := range requiredStrings {
		if !stringContains(contentStr, required) {
			t.Errorf("Log file missing required string: %s", required)
		}
	}
}

// TestParseAndSaveRealFormat tests the full workflow with real format.
func TestParseAndSaveRealFormat(t *testing.T) {
	tmpDir := t.TempDir()
	parser := NewConversationParser(tmpDir)

	// Minimal valid event stream
	jsonData := `[
		{
			"type": "system",
			"subtype": "init",
			"session_id": "abc123",
			"model": "claude-opus-4-6"
		},
		{
			"type": "result",
			"result": "Done",
			"session_id": "abc123"
		}
	]`

	logPath, err := parser.ParseAndSave(jsonData, "module1", "job1")
	if err != nil {
		t.Fatalf("ParseAndSave failed: %v", err)
	}

	// Verify file exists and has correct name pattern
	if !stringContains(filepath.Base(logPath), "module1_job1") {
		t.Errorf("Log file name doesn't match pattern: %s", logPath)
	}

	// Verify content
	content, _ := os.ReadFile(logPath)
	if !stringContains(string(content), "Session ID: abc123") {
		t.Error("Log file missing session ID")
	}
}

// ExampleConversationParser demonstrates how to use the conversation parser.
func ExampleConversationParser() {
	// Create parser
	parser := NewConversationParser(".morty/logs")

	// Sample Claude Code output (event stream format)
	claudeOutput := `[
		{
			"type": "system",
			"subtype": "init",
			"session_id": "example-session",
			"model": "claude-opus-4-6"
		},
		{
			"type": "assistant",
			"message": {
				"role": "assistant",
				"content": [{"type": "text", "text": "Hello!"}]
			}
		}
	]`

	// Parse and save
	logPath, err := parser.ParseAndSave(claudeOutput, "my_module", "my_job")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Log saved to: %s\n", logPath)
	// Output will be like: Log saved to: .morty/logs/my_module_my_job_20260228_150405.log
}

// Helper function for string matching
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || stringContains(s[1:], substr))))
}
