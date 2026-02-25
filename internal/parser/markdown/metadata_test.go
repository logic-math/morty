package markdown

import (
	"strings"
	"testing"
)

func TestExtractMetadata_EmptyContent(t *testing.T) {
	metadata, err := ExtractMetadata("")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(metadata) != 0 {
		t.Errorf("Expected empty map for empty content, got %v", metadata)
	}
}

func TestExtractMetadata_NoFrontmatter(t *testing.T) {
	content := "# Heading\n\nThis is a paragraph.\n\n- List item 1\n- List item 2"
	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(metadata) != 0 {
		t.Errorf("Expected empty map for content without frontmatter, got %v", metadata)
	}
}

func TestExtractMetadata_BasicFrontmatter(t *testing.T) {
	content := `---
title: My Document
author: John Doe
date: 2024-01-15
---

# Heading

Content here.`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := map[string]string{
		"title":  "My Document",
		"author": "John Doe",
		"date":   "2024-01-15",
	}

	if len(metadata) != len(expected) {
		t.Errorf("Expected %d metadata entries, got %d", len(expected), len(metadata))
	}

	for key, expectedValue := range expected {
		if value, ok := metadata[key]; !ok {
			t.Errorf("Missing key %q in metadata", key)
		} else if value != expectedValue {
			t.Errorf("Key %q: expected %q, got %q", key, expectedValue, value)
		}
	}
}

func TestExtractMetadata_WithWhitespace(t *testing.T) {
	content := `---
  title  :   My Document
  author : John Doe
---

Content here.`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if metadata["title"] != "My Document" {
		t.Errorf("Expected title 'My Document', got %q", metadata["title"])
	}
	if metadata["author"] != "John Doe" {
		t.Errorf("Expected author 'John Doe', got %q", metadata["author"])
	}
}

func TestExtractMetadata_EmptyFrontmatter(t *testing.T) {
	content := `---
---

# Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(metadata) != 0 {
		t.Errorf("Expected empty map for empty frontmatter, got %v", metadata)
	}
}

func TestExtractMetadata_OnlyFrontmatter(t *testing.T) {
	content := `---
title: Test
---`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(metadata) != 1 {
		t.Errorf("Expected 1 metadata entry, got %d", len(metadata))
	}
	if metadata["title"] != "Test" {
		t.Errorf("Expected title 'Test', got %q", metadata["title"])
	}
}

func TestExtractMetadata_MultilineValue(t *testing.T) {
	content := `---
title: My Title
description: This is a description
  that spans multiple lines
  in the frontmatter
tags: golang, markdown
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// The current implementation treats each line separately
	// Multi-line values are not supported in this simple implementation
	if metadata["title"] != "My Title" {
		t.Errorf("Expected title 'My Title', got %q", metadata["title"])
	}
	if metadata["tags"] != "golang, markdown" {
		t.Errorf("Expected tags 'golang, markdown', got %q", metadata["tags"])
	}
}

func TestExtractMetadata_SpecialCharactersInValue(t *testing.T) {
	content := `---
title: Hello: World
url: https://example.com
path: /usr/local/bin
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if metadata["url"] != "https://example.com" {
		t.Errorf("Expected url 'https://example.com', got %q", metadata["url"])
	}
	if metadata["path"] != "/usr/local/bin" {
		t.Errorf("Expected path '/usr/local/bin', got %q", metadata["path"])
	}
}

func TestExtractMetadata_KeysWithUnderscoresAndHyphens(t *testing.T) {
	content := `---
page_title: My Page
meta-description: A description
seo_keywords: keyword1, keyword2
camelCaseKey: value
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := map[string]string{
		"page_title":     "My Page",
		"meta-description": "A description",
		"seo_keywords":   "keyword1, keyword2",
		"camelCaseKey":   "value",
	}

	for key, expectedValue := range expected {
		if value, ok := metadata[key]; !ok {
			t.Errorf("Missing key %q in metadata", key)
		} else if value != expectedValue {
			t.Errorf("Key %q: expected %q, got %q", key, expectedValue, value)
		}
	}
}

func TestExtractMetadata_InvalidKeyFormat(t *testing.T) {
	content := `---
valid_key: valid value
key with spaces: invalid
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Only valid_key should be extracted
	if _, ok := metadata["valid_key"]; !ok {
		t.Errorf("Expected 'valid_key' to be present")
	}
	if _, ok := metadata["key with spaces"]; ok {
		t.Errorf("Expected 'key with spaces' to NOT be present (invalid key format)")
	}
}

func TestExtractMetadata_EmptyLinesInFrontmatter(t *testing.T) {
	content := `---
title: My Title

author: John Doe

date: 2024-01-01
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := map[string]string{
		"title":  "My Title",
		"author": "John Doe",
		"date":   "2024-01-01",
	}

	if len(metadata) != len(expected) {
		t.Errorf("Expected %d metadata entries, got %d", len(expected), len(metadata))
	}

	for key, expectedValue := range expected {
		if value, ok := metadata[key]; !ok {
			t.Errorf("Missing key %q in metadata", key)
		} else if value != expectedValue {
			t.Errorf("Key %q: expected %q, got %q", key, expectedValue, value)
		}
	}
}

func TestExtractMetadata_LeadingWhitespace(t *testing.T) {
	content := `

---
title: My Title
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if metadata["title"] != "My Title" {
		t.Errorf("Expected title 'My Title', got %q", metadata["title"])
	}
}

func TestExtractMetadata_ComplexDocument(t *testing.T) {
	content := `---
title: Project Documentation
author: Jane Smith
version: 1.2.3
category: Tutorial
tags: go, markdown, parser
---

# Project Documentation

This is the main content.

## Section 1

- Item 1
- Item 2

## Section 2

Some code:

` + "```go\nfmt.Println(\"Hello\")\n```" + `

End of document.`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := map[string]string{
		"title":    "Project Documentation",
		"author":   "Jane Smith",
		"version":  "1.2.3",
		"category": "Tutorial",
		"tags":     "go, markdown, parser",
	}

	if len(metadata) != len(expected) {
		t.Errorf("Expected %d metadata entries, got %d", len(expected), len(metadata))
	}

	for key, expectedValue := range expected {
		if value, ok := metadata[key]; !ok {
			t.Errorf("Missing key %q in metadata", key)
		} else if value != expectedValue {
			t.Errorf("Key %q: expected %q, got %q", key, expectedValue, value)
		}
	}
}

func TestExtractMetadataFromDocument_NilDocument(t *testing.T) {
	metadata, err := ExtractMetadataFromDocument(nil)
	if err == nil {
		t.Error("Expected error for nil document, got nil")
	}
	if metadata != nil {
		t.Errorf("Expected nil metadata for nil document, got %v", metadata)
	}
}

func TestExtractMetadataFromDocument_EmptyDocument(t *testing.T) {
	doc := &Document{Nodes: []Node{}}
	metadata, err := ExtractMetadataFromDocument(doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(metadata) != 0 {
		t.Errorf("Expected empty map for empty document, got %v", metadata)
	}
}

func TestExtractMetadataFromDocument_WithFrontmatterNode(t *testing.T) {
	// When document is parsed, frontmatter is not a node type
	// This tests the reconstruction approach
	parser := NewParser()
	doc, err := parser.ParseDocument("# Heading\n\nContent")
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	metadata, err := ExtractMetadataFromDocument(doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(metadata) != 0 {
		t.Errorf("Expected empty map for document without frontmatter, got %v", metadata)
	}
}

func TestHasFrontmatter(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{"", false},
		{"# Just a heading", false},
		{"---\n---", true},
		{"---\ntitle: Test\n---", true},
		{"  ---\n  title: Test\n  ---", true},
		{"text\n---\ntitle: Test\n---", false}, // Frontmatter must be at start
	}

	for _, test := range tests {
		result := HasFrontmatter(test.content)
		if result != test.expected {
			t.Errorf("HasFrontmatter(%q): expected %v, got %v", test.content, test.expected, result)
		}
	}
}

func TestGetFrontmatterRaw(t *testing.T) {
	content := `---
title: My Title
author: John Doe
---

Content here`

	raw := GetFrontmatterRaw(content)
	expected := "title: My Title\nauthor: John Doe"

	if raw != expected {
		t.Errorf("GetFrontmatterRaw: expected %q, got %q", expected, raw)
	}
}

func TestGetFrontmatterRaw_NoFrontmatter(t *testing.T) {
	content := "# Just content"
	raw := GetFrontmatterRaw(content)
	if raw != "" {
		t.Errorf("Expected empty string for content without frontmatter, got %q", raw)
	}
}

func TestValidateFrontmatter_Valid(t *testing.T) {
	tests := []string{
		"# No frontmatter",
		"---\n---",
		"---\ntitle: Test\n---",
		"---\ntitle: Test\nauthor: John\n---\n\nContent",
	}

	for _, content := range tests {
		err := ValidateFrontmatter(content)
		if err != nil {
			t.Errorf("ValidateFrontmatter(%q): unexpected error: %v", content, err)
		}
	}
}

func TestValidateFrontmatter_Unclosed(t *testing.T) {
	content := "---\ntitle: Test\n\nContent"
	err := ValidateFrontmatter(content)
	if err == nil {
		t.Error("Expected error for unclosed frontmatter, got nil")
	}
	if !strings.Contains(err.Error(), "unclosed frontmatter") {
		t.Errorf("Expected error message to contain 'unclosed frontmatter', got: %v", err)
	}
}

func TestValidateFrontmatter_UnclosedWithWhitespace(t *testing.T) {
	content := "  ---\n  title: Test\n\nContent"
	err := ValidateFrontmatter(content)
	if err == nil {
		t.Error("Expected error for unclosed frontmatter, got nil")
	}
}

func TestExtractMetadata_ValueWithColon(t *testing.T) {
	content := `---
title: Note: Important
time: 10:30:00
ratio: 16:9
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// The value should include everything after the first colon
	if !strings.HasPrefix(metadata["title"], "Note:") {
		t.Errorf("Expected title to start with 'Note:', got %q", metadata["title"])
	}
	if metadata["time"] != "10:30:00" {
		t.Errorf("Expected time '10:30:00', got %q", metadata["time"])
	}
	if metadata["ratio"] != "16:9" {
		t.Errorf("Expected ratio '16:9', got %q", metadata["ratio"])
	}
}

func TestExtractMetadata_NumericValues(t *testing.T) {
	content := `---
count: 42
version: 1.2.3
port: 8080
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if metadata["count"] != "42" {
		t.Errorf("Expected count '42', got %q", metadata["count"])
	}
	if metadata["version"] != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got %q", metadata["version"])
	}
	if metadata["port"] != "8080" {
		t.Errorf("Expected port '8080', got %q", metadata["port"])
	}
}

func TestExtractMetadata_BooleanLikeValues(t *testing.T) {
	content := `---
published: true
draft: false
enabled: yes
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if metadata["published"] != "true" {
		t.Errorf("Expected published 'true', got %q", metadata["published"])
	}
	if metadata["draft"] != "false" {
		t.Errorf("Expected draft 'false', got %q", metadata["draft"])
	}
	if metadata["enabled"] != "yes" {
		t.Errorf("Expected enabled 'yes', got %q", metadata["enabled"])
	}
}

func TestExtractMetadata_UnicodeContent(t *testing.T) {
	content := `---
title: 你好世界
author: 张三
emoji: 🎉
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if metadata["title"] != "你好世界" {
		t.Errorf("Expected title '你好世界', got %q", metadata["title"])
	}
	if metadata["author"] != "张三" {
		t.Errorf("Expected author '张三', got %q", metadata["author"])
	}
	if metadata["emoji"] != "🎉" {
		t.Errorf("Expected emoji '🎉', got %q", metadata["emoji"])
	}
}

func TestExtractMetadata_EmptyKey(t *testing.T) {
	content := `---
: empty key value
valid: value
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Empty key should be skipped
	if _, ok := metadata[""]; ok {
		t.Error("Expected empty key to be skipped")
	}
	if metadata["valid"] != "value" {
		t.Errorf("Expected valid key to have value 'value', got %q", metadata["valid"])
	}
}

func TestExtractMetadata_EmptyValue(t *testing.T) {
	content := `---
title:
author: John
blank:
---

Content`

	metadata, err := ExtractMetadata(content)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Empty values are allowed
	if _, ok := metadata["title"]; !ok {
		t.Error("Expected 'title' key to be present even with empty value")
	}
	if metadata["author"] != "John" {
		t.Errorf("Expected author 'John', got %q", metadata["author"])
	}
}

func BenchmarkExtractMetadata(b *testing.B) {
	content := `---
title: Benchmark Test
author: John Doe
date: 2024-01-15
category: Performance
tags: benchmark, test, markdown
version: 1.0.0
---

# Content

This is the main content of the document.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractMetadata(content)
	}
}

func BenchmarkExtractMetadata_NoFrontmatter(b *testing.B) {
	content := "# Heading\n\nThis is a paragraph without any frontmatter.\n\n- Item 1\n- Item 2"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractMetadata(content)
	}
}
