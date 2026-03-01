package callcli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConversationParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		wantErr bool
	}{
		{
			name: "valid conversation with messages",
			jsonStr: `{
				"messages": [
					{
						"role": "user",
						"content": "Hello, can you help me?"
					},
					{
						"role": "assistant",
						"content": "Of course! I'd be happy to help."
					}
				],
				"model": "claude-opus-4-6"
			}`,
			wantErr: false,
		},
		{
			name: "conversation with tool calls",
			jsonStr: `{
				"messages": [
					{
						"role": "user",
						"content": "Read the file test.txt"
					},
					{
						"role": "assistant",
						"content": [
							{"type": "text", "text": "I'll read that file for you."}
						],
						"tool_use": {
							"type": "tool_use",
							"id": "toolu_123",
							"name": "Read",
							"input": {"file_path": "test.txt"}
						}
					}
				]
			}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			jsonStr: `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewConversationParser(t.TempDir())
			conversation, err := parser.Parse(tt.jsonStr)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if conversation == nil {
				t.Error("expected conversation but got nil")
			}
		})
	}
}

func TestConversationParser_ExtractLogs(t *testing.T) {
	parser := NewConversationParser(t.TempDir())

	conversation := &ConversationData{
		Messages: []ConversationMessage{
			{
				Role:      "user",
				Content:   "Test user message",
				Timestamp: time.Now(),
			},
			{
				Role:      "assistant",
				Content:   "Test assistant response",
				Timestamp: time.Now(),
			},
		},
	}

	logs := parser.ExtractLogs(conversation)

	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}

	// Check first log (user message)
	if logs[0].MessageType != "user_message" {
		t.Errorf("expected message type 'user_message', got '%s'", logs[0].MessageType)
	}
	if logs[0].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", logs[0].Role)
	}
	if logs[0].Content != "Test user message" {
		t.Errorf("expected content 'Test user message', got '%s'", logs[0].Content)
	}

	// Check second log (assistant message)
	if logs[1].MessageType != "assistant_text" {
		t.Errorf("expected message type 'assistant_text', got '%s'", logs[1].MessageType)
	}
	if logs[1].Role != "assistant" {
		t.Errorf("expected role 'assistant', got '%s'", logs[1].Role)
	}
}

func TestConversationParser_ExtractContentString(t *testing.T) {
	parser := NewConversationParser(t.TempDir())

	tests := []struct {
		name     string
		content  interface{}
		expected string
	}{
		{
			name:     "string content",
			content:  "Simple string",
			expected: "Simple string",
		},
		{
			name: "array of text blocks",
			content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "First part",
				},
				map[string]interface{}{
					"type": "text",
					"text": "Second part",
				},
			},
			expected: "First part\nSecond part",
		},
		{
			name: "array with content field",
			content: []interface{}{
				map[string]interface{}{
					"type":    "text",
					"content": "Text content",
				},
			},
			expected: "Text content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractContentString(tt.content)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestConversationParser_ParseAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	parser := NewConversationParser(tmpDir)

	conversationJSON := `{
		"messages": [
			{
				"role": "user",
				"content": "Hello"
			},
			{
				"role": "assistant",
				"content": "Hi there!"
			}
		],
		"model": "claude-opus-4-6"
	}`

	logPath, err := parser.ParseAndSave(conversationJSON, "test_module", "test_job")
	if err != nil {
		t.Fatalf("ParseAndSave failed: %v", err)
	}

	// Check that log file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("log file was not created at %s", logPath)
	}

	// Read and verify log content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !contains(contentStr, "USER:") {
		t.Error("log file should contain 'USER:' marker")
	}
	if !contains(contentStr, "ASSISTANT:") {
		t.Error("log file should contain 'ASSISTANT:' marker")
	}
	if !contains(contentStr, "Hello") {
		t.Error("log file should contain user message 'Hello'")
	}
	if !contains(contentStr, "Hi there!") {
		t.Error("log file should contain assistant message 'Hi there!'")
	}
}

func TestConversationParser_SaveFormattedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	parser := NewConversationParser(tmpDir)

	conversation := &ConversationData{
		Messages: []ConversationMessage{
			{
				Role:    "user",
				Content: "Test message",
			},
		},
		Model: "claude-opus-4-6",
	}

	jsonPath, err := parser.SaveFormattedJSON(conversation, "test_module", "test_job")
	if err != nil {
		t.Fatalf("SaveFormattedJSON failed: %v", err)
	}

	// Check that JSON file was created
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Errorf("JSON file was not created at %s", jsonPath)
	}

	// Read and verify JSON content
	content, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	// Parse to verify it's valid JSON
	var parsed ConversationData
	if err := json.Unmarshal(content, &parsed); err != nil {
		t.Errorf("saved JSON is not valid: %v", err)
	}

	if parsed.Model != "claude-opus-4-6" {
		t.Errorf("expected model 'claude-opus-4-6', got '%s'", parsed.Model)
	}
}

func TestConversationParser_ParseFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	parser := NewConversationParser(tmpDir)

	// Create a test JSON file
	testJSON := `{
		"messages": [
			{
				"role": "user",
				"content": "Test from file"
			}
		]
	}`

	jsonFile := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(jsonFile, []byte(testJSON), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Parse from file
	conversation, err := parser.ParseFromFile(jsonFile)
	if err != nil {
		t.Fatalf("ParseFromFile failed: %v", err)
	}

	if len(conversation.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(conversation.Messages))
	}

	if conversation.Messages[0].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", conversation.Messages[0].Role)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename",
			input:    "test_file",
			expected: "test_file",
		},
		{
			name:     "filename with spaces",
			input:    "test file name",
			expected: "test_file_name",
		},
		{
			name:     "filename with invalid chars",
			input:    "test/file:name*",
			expected: "test_file_name_",
		},
		{
			name:     "chinese characters",
			input:    "测试模块",
			expected: "测试模块",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestConversationParser_ExtractToolCalls(t *testing.T) {
	parser := NewConversationParser(t.TempDir())

	conversation := &ConversationData{
		Messages: []ConversationMessage{
			{
				Role: "assistant",
				ToolUse: &ToolUseBlock{
					Type: "tool_use",
					ID:   "toolu_123",
					Name: "Read",
					Input: map[string]interface{}{
						"file_path": "test.txt",
					},
				},
			},
		},
	}

	logs := parser.ExtractLogs(conversation)

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].MessageType != "tool_call" {
		t.Errorf("expected message type 'tool_call', got '%s'", logs[0].MessageType)
	}

	if logs[0].ToolName != "Read" {
		t.Errorf("expected tool name 'Read', got '%s'", logs[0].ToolName)
	}

	if logs[0].ToolParams["file_path"] != "test.txt" {
		t.Errorf("expected file_path 'test.txt', got '%v'", logs[0].ToolParams["file_path"])
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
