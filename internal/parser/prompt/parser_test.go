// Package prompt provides tests for the prompt parser.
package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParsePrompt tests the ParsePrompt function with a real file.
func TestParsePrompt(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a test prompt file
	testContent := `---
name: test-prompt
description: A test prompt for unit testing
author: test-author
category: testing
---

# {{title}}

Hello {{name}}, welcome to {{place}}!

Your task is to:
{{task_description}}

Please complete by {{deadline}}.
`

	testFile := filepath.Join(tempDir, "test.prompt")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse the prompt file
	prompt, err := ParsePrompt(testFile)
	if err != nil {
		t.Fatalf("ParsePrompt() error = %v", err)
	}

	// Verify parsed data
	if prompt.Name != "test-prompt" {
		t.Errorf("Name = %q, want %q", prompt.Name, "test-prompt")
	}

	if prompt.Description != "A test prompt for unit testing" {
		t.Errorf("Description = %q, want %q", prompt.Description, "A test prompt for unit testing")
	}

	// Check metadata
	if prompt.Metadata["author"] != "test-author" {
		t.Errorf("Metadata[author] = %q, want %q", prompt.Metadata["author"], "test-author")
	}

	if prompt.Metadata["category"] != "testing" {
		t.Errorf("Metadata[category] = %q, want %q", prompt.Metadata["category"], "testing")
	}

	// Check template content
	if !strings.Contains(prompt.Template, "{{title}}") {
		t.Error("Template should contain {{title}}")
	}

	// Check extracted variables
	expectedVars := []string{"title", "name", "place", "task_description", "deadline"}
	if len(prompt.Variables) != len(expectedVars) {
		t.Errorf("Variables count = %d, want %d", len(prompt.Variables), len(expectedVars))
	}

	for _, expectedVar := range expectedVars {
		found := false
		for _, v := range prompt.Variables {
			if v == expectedVar {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Variable %q not found in parsed variables", expectedVar)
		}
	}
}

// TestParsePromptString tests parsing from a string.
func TestParsePromptString(t *testing.T) {
	tests := []struct {
		name            string
		content         string
		wantName        string
		wantDescription string
		wantVars        []string
		wantMetadata    map[string]string
	}{
		{
			name: "basic prompt with frontmatter",
			content: `---
name: my-prompt
description: My description
---

Hello {{name}}!`,
			wantName:        "my-prompt",
			wantDescription: "My description",
			wantVars:        []string{"name"},
			wantMetadata: map[string]string{
				"name":        "my-prompt",
				"description": "My description",
			},
		},
		{
			name: "prompt without frontmatter",
			content: `Hello {{name}},

Welcome to {{place}}!`,
			wantName:        "",
			wantDescription: "",
			wantVars:        []string{"name", "place"},
			wantMetadata:    map[string]string{},
		},
		{
			name: "prompt with multiple variables",
			content: `---
name: complex-prompt
---

{{greeting}} {{name}},

Your order {{order_id}} has been {{status}}.
Total: {{amount}}

{{closing}}`,
			wantName:        "complex-prompt",
			wantDescription: "",
			wantVars:        []string{"greeting", "name", "order_id", "status", "amount", "closing"},
			wantMetadata: map[string]string{
				"name": "complex-prompt",
			},
		},
		{
			name: "empty prompt",
			content: `---
name: empty
---`,
			wantName:        "empty",
			wantDescription: "",
			wantVars:        []string{},
			wantMetadata: map[string]string{
				"name": "empty",
			},
		},
		{
			name: "variables with spaces",
			content: `Hello {{ name }}, welcome to {{ place }}!`,
			wantName:        "",
			wantDescription: "",
			wantVars:        []string{"name", "place"},
			wantMetadata:    map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := ParsePromptString(tt.content)
			if err != nil {
				t.Fatalf("ParsePromptString() error = %v", err)
			}

			if prompt.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", prompt.Name, tt.wantName)
			}

			if prompt.Description != tt.wantDescription {
				t.Errorf("Description = %q, want %q", prompt.Description, tt.wantDescription)
			}

			// Check variables count
			if len(prompt.Variables) != len(tt.wantVars) {
				t.Errorf("Variables count = %d, want %d", len(prompt.Variables), len(tt.wantVars))
			}

			// Check each expected variable exists
			for _, wantVar := range tt.wantVars {
				found := false
				for _, v := range prompt.Variables {
					if v == wantVar {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Variable %q not found", wantVar)
				}
			}

			// Check metadata
			for key, wantValue := range tt.wantMetadata {
				if gotValue, ok := prompt.Metadata[key]; !ok || gotValue != wantValue {
					t.Errorf("Metadata[%q] = %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}

// TestReplaceVariables tests variable replacement.
func TestReplaceVariables(t *testing.T) {
	content := `---
name: test
---

Hello {{name}}, welcome to {{place}}!
Your task: {{task}}`

	prompt, err := ParsePromptString(content)
	if err != nil {
		t.Fatalf("ParsePromptString() error = %v", err)
	}

	tests := []struct {
		name      string
		variables map[string]string
		want      string
	}{
		{
			name: "replace all variables",
			variables: map[string]string{
				"name": "Alice",
				"place": "Wonderland",
				"task": "find the rabbit",
			},
			want: "Hello Alice, welcome to Wonderland!\nYour task: find the rabbit",
		},
		{
			name: "replace partial variables",
			variables: map[string]string{
				"name": "Bob",
				"place": "New York",
			},
			want: "Hello Bob, welcome to New York!\nYour task: {{task}}",
		},
		{
			name: "replace with empty string",
			variables: map[string]string{
				"name": "",
				"place": "Nowhere",
				"task": "",
			},
			want: "Hello , welcome to Nowhere!\nYour task: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prompt.ReplaceVariables(tt.variables)
			if got != tt.want {
				t.Errorf("ReplaceVariables() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestReplaceVariable tests single variable replacement.
func TestReplaceVariable(t *testing.T) {
	content := `Hello {{name}}, welcome to {{place}}!`
	prompt, _ := ParsePromptString(content)

	result := prompt.ReplaceVariable("name", "Alice")
	want := "Hello Alice, welcome to {{place}}!"
	if result != want {
		t.Errorf("ReplaceVariable() = %q, want %q", result, want)
	}
}

// TestHasVariable tests the HasVariable method.
func TestHasVariable(t *testing.T) {
	content := `{{first}} {{second}}`
	prompt, _ := ParsePromptString(content)

	tests := []struct {
		varName string
		want    bool
	}{
		{"first", true},
		{"second", true},
		{"third", false},
	}

	for _, tt := range tests {
		t.Run(tt.varName, func(t *testing.T) {
			got := prompt.HasVariable(tt.varName)
			if got != tt.want {
				t.Errorf("HasVariable(%q) = %v, want %v", tt.varName, got, tt.want)
			}
		})
	}
}

// TestValidate tests the Validate method.
func TestValidate(t *testing.T) {
	content := `---
name: test
---

Hello {{name}}, welcome to {{place}}!`
	prompt, _ := ParsePromptString(content)

	tests := []struct {
		name    string
		values  map[string]string
		wantErr bool
	}{
		{
			name: "all variables provided",
			values: map[string]string{
				"name":  "Alice",
				"place": "Wonderland",
			},
			wantErr: false,
		},
		{
			name: "missing one variable",
			values: map[string]string{
				"name": "Alice",
			},
			wantErr: true,
		},
		{
			name:    "no variables provided",
			values:  map[string]string{},
			wantErr: true,
		},
		{
			name: "extra variables allowed",
			values: map[string]string{
				"name":   "Alice",
				"place":  "Wonderland",
				"extra":  "ignored",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := prompt.Validate(tt.values)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRender tests the Render method.
func TestRender(t *testing.T) {
	content := `---
name: test
---

Hello {{name}}, welcome to {{place}}!`
	prompt, _ := ParsePromptString(content)

	tests := []struct {
		name    string
		values  map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "successful render",
			values: map[string]string{
				"name":  "Alice",
				"place": "Wonderland",
			},
			want:    "Hello Alice, welcome to Wonderland!",
			wantErr: false,
		},
		{
			name: "render with missing variable",
			values: map[string]string{
				"name": "Alice",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := prompt.Render(tt.values)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Render() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractFrontmatter tests frontmatter extraction.
func TestExtractFrontmatter(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantMetadata map[string]string
		wantContent  string
	}{
		{
			name: "basic frontmatter",
			content: `---
name: test
description: A test
---

Template content here`,
			wantMetadata: map[string]string{
				"name":        "test",
				"description": "A test",
			},
			wantContent: "Template content here",
		},
		{
			name:         "no frontmatter",
			content:      "Just plain content",
			wantMetadata: map[string]string{},
			wantContent:  "Just plain content",
		},
		{
			name: "empty frontmatter",
			content: `---
---

Content after`,
			wantMetadata: map[string]string{},
			wantContent:  "Content after",
		},
		{
			name: "frontmatter with various value types",
			content: `---
name: my-prompt
version: 1.0
count: 42
enabled: true
url: https://example.com/path
list: item1, item2, item3
---

Template`,
			wantMetadata: map[string]string{
				"name":    "my-prompt",
				"version": "1.0",
				"count":   "42",
				"enabled": "true",
				"url":     "https://example.com/path",
				"list":    "item1, item2, item3",
			},
			wantContent: "Template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMetadata, gotContent := extractFrontmatter(tt.content)

			// Check metadata
			for key, wantValue := range tt.wantMetadata {
				if gotValue, ok := gotMetadata[key]; !ok || gotValue != wantValue {
					t.Errorf("Metadata[%q] = %q, want %q", key, gotValue, wantValue)
				}
			}

			if gotContent != tt.wantContent {
				t.Errorf("Content = %q, want %q", gotContent, tt.wantContent)
			}
		})
	}
}

// TestExtractVariables tests variable extraction.
func TestExtractVariables(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "single variable",
			content: "Hello {{name}}!",
			want:    []string{"name"},
		},
		{
			name:    "multiple variables",
			content: "{{greeting}} {{name}}, welcome to {{place}}!",
			want:    []string{"greeting", "name", "place"},
		},
		{
			name:    "duplicate variables",
			content: "{{name}} and {{name}} again",
			want:    []string{"name"},
		},
		{
			name:    "variables with spaces",
			content: "{{ name }} {{  place  }}",
			want:    []string{"name", "place"},
		},
		{
			name:    "no variables",
			content: "Just plain text",
			want:    []string{},
		},
		{
			name:    "variables with underscores and hyphens",
			content: "{{first_name}} {{last-name}} {{user_id}}",
			want:    []string{"first_name", "last-name", "user_id"},
		},
		{
			name:    "variables with numbers",
			content: "{{var1}} {{var_2}} {{var3_test}}",
			want:    []string{"var1", "var_2", "var3_test"},
		},
		{
			name:    "empty braces not matched",
			content: "{{}} test",
			want:    []string{},
		},
		{
			name:    "invalid variable names not matched",
			content: "{{with space}} {{special!}}",
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractVariables(tt.content)

			if len(got) != len(tt.want) {
				t.Errorf("extractVariables() = %v, want %v", got, tt.want)
				return
			}

			for i, wantVar := range tt.want {
				if got[i] != wantVar {
					t.Errorf("extractVariables()[%d] = %q, want %q", i, got[i], wantVar)
				}
			}
		})
	}
}

// TestParsePromptFileNotFound tests error handling for non-existent files.
func TestParsePromptFileNotFound(t *testing.T) {
	_, err := ParsePrompt("/nonexistent/path/to/file.prompt")
	if err == nil {
		t.Error("ParsePrompt() expected error for non-existent file")
	}

	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("Error message should contain 'failed to read file', got: %v", err)
	}
}

// TestGetVariableValues tests the GetVariableValues method.
func TestGetVariableValues(t *testing.T) {
	content := `{{a}} {{b}} {{c}}`
	prompt, _ := ParsePromptString(content)

	tests := []struct {
		name       string
		values     map[string]string
		wantResult map[string]string
		wantErr    bool
	}{
		{
			name: "all values present",
			values: map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
			wantResult: map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
			wantErr: false,
		},
		{
			name: "missing value",
			values: map[string]string{
				"a": "1",
				"c": "3",
			},
			wantResult: map[string]string{
				"a": "1",
				"c": "3",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prompt.GetVariableValues(tt.values)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetVariableValues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for key, wantValue := range tt.wantResult {
				if gotValue, ok := result[key]; !ok || gotValue != wantValue {
					t.Errorf("GetVariableValues()[%q] = %q, want %q", key, gotValue, wantValue)
				}
			}
		})
	}
}

// BenchmarkParsePromptString benchmarks the ParsePromptString function.
func BenchmarkParsePromptString(b *testing.B) {
	content := `---
name: benchmark-prompt
description: A benchmark test prompt
author: tester
category: benchmark
version: 1.0.0
---

# {{title}}

Hello {{name}}, welcome to {{place}}!

Your task is to:
{{task_description}}

Please complete by {{deadline}}.

Best regards,
{{sender_name}}
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParsePromptString(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReplaceVariables benchmarks the ReplaceVariables method.
func BenchmarkReplaceVariables(b *testing.B) {
	content := `---
name: test
---

Hello {{name}}, welcome to {{place}}!
Your task: {{task_description}}
Deadline: {{deadline}}
Sender: {{sender_name}}`

	prompt, _ := ParsePromptString(content)
	values := map[string]string{
		"name":             "Alice",
		"place":            "Wonderland",
		"task_description": "find the rabbit",
		"deadline":         "tomorrow",
		"sender_name":      "The Queen",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = prompt.ReplaceVariables(values)
	}
}

// BenchmarkExtractVariables benchmarks the extractVariables function.
func BenchmarkExtractVariables(b *testing.B) {
	content := `{{var1}} {{var2}} {{var3}} {{var4}} {{var5}}
{{var6}} {{var7}} {{var8}} {{var9}} {{var10}}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractVariables(content)
	}
}
