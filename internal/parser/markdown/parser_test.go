package markdown

import (
	"context"
	"strings"
	"testing"

	"github.com/morty/morty/internal/parser"
)

// TestNewParser tests the parser creation.
func TestNewParser(t *testing.T) {
	p := NewParser()
	if p == nil {
		t.Fatal("NewParser() returned nil")
	}
}

// TestParser_FileType tests the FileType method.
func TestParser_FileType(t *testing.T) {
	p := NewParser()
	if p.FileType() != parser.FileTypeMarkdown {
		t.Errorf("FileType() = %v, want %v", p.FileType(), parser.FileTypeMarkdown)
	}
}

// TestParser_Supports tests the Supports method.
func TestParser_Supports(t *testing.T) {
	p := NewParser()

	tests := []struct {
		fileType parser.FileType
		want     bool
	}{
		{parser.FileTypeMarkdown, true},
		{parser.FileTypeJSON, false},
		{parser.FileTypeYAML, false},
		{parser.FileTypeUnknown, false},
	}

	for _, tc := range tests {
		t.Run(string(tc.fileType), func(t *testing.T) {
			got := p.Supports(tc.fileType)
			if got != tc.want {
				t.Errorf("Supports(%v) = %v, want %v", tc.fileType, got, tc.want)
			}
		})
	}
}

// TestParser_ParseString_Headings tests heading parsing.
func TestParser_ParseString_Headings(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	content := `# Heading 1
## Heading 2
### Heading 3
#### Heading 4
##### Heading 5
###### Heading 6`

	result, err := p.ParseString(ctx, content)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	if result.Type != parser.FileTypeMarkdown {
		t.Errorf("result.Type = %v, want %v", result.Type, parser.FileTypeMarkdown)
	}

	doc, ok := result.Content.(*Document)
	if !ok {
		t.Fatalf("result.Content is not *Document")
	}

	if len(doc.Nodes) != 6 {
		t.Fatalf("expected 6 nodes, got %d", len(doc.Nodes))
	}

	// Verify each heading level
	for i, node := range doc.Nodes {
		expectedLevel := i + 1
		if node.Type != NodeTypeHeading {
			t.Errorf("node[%d].Type = %v, want %v", i, node.Type, NodeTypeHeading)
		}
		if node.Level != expectedLevel {
			t.Errorf("node[%d].Level = %d, want %d", i, node.Level, expectedLevel)
		}
		expectedContent := "Heading " + string(rune('1'+i))
		if node.Content != expectedContent {
			t.Errorf("node[%d].Content = %q, want %q", i, node.Content, expectedContent)
		}
	}
}

// TestParser_ParseString_Paragraphs tests paragraph parsing.
func TestParser_ParseString_Paragraphs(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	content := `This is the first paragraph.

This is the second paragraph with multiple words.

This is the third paragraph.`

	result, err := p.ParseString(ctx, content)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	doc := result.Content.(*Document)
	if len(doc.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(doc.Nodes))
	}

	for i, node := range doc.Nodes {
		if node.Type != NodeTypeParagraph {
			t.Errorf("node[%d].Type = %v, want %v", i, node.Type, NodeTypeParagraph)
		}
	}

	// Check content
	expectedContents := []string{
		"This is the first paragraph.",
		"This is the second paragraph with multiple words.",
		"This is the third paragraph.",
	}
	for i, expected := range expectedContents {
		if doc.Nodes[i].Content != expected {
			t.Errorf("node[%d].Content = %q, want %q", i, doc.Nodes[i].Content, expected)
		}
	}
}

// TestParser_ParseString_UnorderedLists tests unordered list parsing.
func TestParser_ParseString_UnorderedLists(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "dash items",
			content: `- First item
- Second item
- Third item`,
			expected: []string{"First item", "Second item", "Third item"},
		},
		{
			name: "asterisk items",
			content: `* First item
* Second item
* Third item`,
			expected: []string{"First item", "Second item", "Third item"},
		},
		{
			name: "plus items",
			content: `+ First item
+ Second item
+ Third item`,
			expected: []string{"First item", "Second item", "Third item"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := p.ParseString(ctx, tc.content)
			if err != nil {
				t.Fatalf("ParseString() error = %v", err)
			}

			doc := result.Content.(*Document)
			if len(doc.Nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(doc.Nodes))
			}

			node := doc.Nodes[0]
			if node.Type != NodeTypeList {
				t.Errorf("node.Type = %v, want %v", node.Type, NodeTypeList)
			}
			if node.ListType != ListTypeUnordered {
				t.Errorf("node.ListType = %v, want %v", node.ListType, ListTypeUnordered)
			}
			if len(node.Items) != len(tc.expected) {
				t.Errorf("len(node.Items) = %d, want %d", len(node.Items), len(tc.expected))
			}
			for i, item := range tc.expected {
				if node.Items[i] != item {
					t.Errorf("node.Items[%d] = %q, want %q", i, node.Items[i], item)
				}
			}
		})
	}
}

// TestParser_ParseString_OrderedLists tests ordered list parsing.
func TestParser_ParseString_OrderedLists(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	content := `1. First item
2. Second item
3. Third item
10. Tenth item`

	result, err := p.ParseString(ctx, content)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	doc := result.Content.(*Document)
	if len(doc.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(doc.Nodes))
	}

	node := doc.Nodes[0]
	if node.Type != NodeTypeList {
		t.Errorf("node.Type = %v, want %v", node.Type, NodeTypeList)
	}
	if node.ListType != ListTypeOrdered {
		t.Errorf("node.ListType = %v, want %v", node.ListType, ListTypeOrdered)
	}

	expectedItems := []string{"First item", "Second item", "Third item", "Tenth item"}
	if len(node.Items) != len(expectedItems) {
		t.Errorf("len(node.Items) = %d, want %d", len(node.Items), len(expectedItems))
	}
	for i, expected := range expectedItems {
		if node.Items[i] != expected {
			t.Errorf("node.Items[%d] = %q, want %q", i, node.Items[i], expected)
		}
	}
}

// TestParser_ParseString_CodeBlocks tests code block parsing.
func TestParser_ParseString_CodeBlocks(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	tests := []struct {
		name             string
		content          string
		expectedLanguage string
		expectedContent  string
	}{
		{
			name: "code block with language",
			content: "```go\n" +
				`package main

func main() {
    println("Hello")
}` +
				"\n```",
			expectedLanguage: "go",
			expectedContent: `package main

func main() {
    println("Hello")
}`,
		},
		{
			name: "code block without language",
			content: "```\n" +
				`some plain text
more text` +
				"\n```",
			expectedLanguage: "",
			expectedContent: `some plain text
more text`,
		},
		{
			name: "code block with different language",
			content: "```javascript\n" +
				`console.log("Hello");` +
				"\n```",
			expectedLanguage: "javascript",
			expectedContent:  `console.log("Hello");`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := p.ParseString(ctx, tc.content)
			if err != nil {
				t.Fatalf("ParseString() error = %v", err)
			}

			doc := result.Content.(*Document)
			if len(doc.Nodes) != 1 {
				t.Fatalf("expected 1 node, got %d", len(doc.Nodes))
			}

			node := doc.Nodes[0]
			if node.Type != NodeTypeCodeBlock {
				t.Errorf("node.Type = %v, want %v", node.Type, NodeTypeCodeBlock)
			}
			if node.Language != tc.expectedLanguage {
				t.Errorf("node.Language = %q, want %q", node.Language, tc.expectedLanguage)
			}
			if node.Content != tc.expectedContent {
				t.Errorf("node.Content = %q, want %q", node.Content, tc.expectedContent)
			}
		})
	}
}

// TestParser_ParseString_MixedContent tests parsing mixed markdown content.
func TestParser_ParseString_MixedContent(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	content := `# Document Title

This is an introduction paragraph.

## Section 1

- Item 1
- Item 2
- Item 3

## Section 2

1. First step
2. Second step

` + "```python\n" + `print("Hello World")
` + "```\n" +
		`This is the conclusion.`

	result, err := p.ParseString(ctx, content)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	doc := result.Content.(*Document)

	// Expected nodes:
	// 1. H1: Document Title
	// 2. Paragraph: This is an introduction paragraph.
	// 3. H2: Section 1
	// 4. Unordered list: [Item 1, Item 2, Item 3]
	// 5. H2: Section 2
	// 6. Ordered list: [First step, Second step]
	// 7. Code block (python)
	// 8. Paragraph: This is the conclusion.
	if len(doc.Nodes) != 8 {
		t.Errorf("expected 8 nodes, got %d", len(doc.Nodes))
		for i, node := range doc.Nodes {
			t.Logf("Node %d: %s", i, node.String())
		}
	}

	// Verify specific nodes
	if doc.Nodes[0].Type != NodeTypeHeading || doc.Nodes[0].Level != 1 {
		t.Errorf("node[0] expected H1, got %v level %d", doc.Nodes[0].Type, doc.Nodes[0].Level)
	}
	if doc.Nodes[1].Type != NodeTypeParagraph {
		t.Errorf("node[1] expected Paragraph, got %v", doc.Nodes[1].Type)
	}
	if doc.Nodes[3].Type != NodeTypeList || doc.Nodes[3].ListType != ListTypeUnordered {
		t.Errorf("node[3] expected Unordered List, got %v", doc.Nodes[3].Type)
	}
	if doc.Nodes[6].Type != NodeTypeCodeBlock {
		t.Errorf("node[6] expected CodeBlock, got %v", doc.Nodes[6].Type)
	}
	if doc.Nodes[7].Type != NodeTypeParagraph {
		t.Errorf("node[7] expected Paragraph, got %v", doc.Nodes[7].Type)
	}
}

// TestParser_Parse tests the Parse method with io.Reader.
func TestParser_Parse(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	content := `# Test Title

This is a test paragraph.`

	result, err := p.Parse(ctx, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.Type != parser.FileTypeMarkdown {
		t.Errorf("result.Type = %v, want %v", result.Type, parser.FileTypeMarkdown)
	}

	doc := result.Content.(*Document)
	if len(doc.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(doc.Nodes))
	}
}

// TestParser_ParseDocument tests the ParseDocument convenience method.
func TestParser_ParseDocument(t *testing.T) {
	p := NewParser()

	content := `## Heading 2

- List item`

	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	if len(doc.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(doc.Nodes))
	}

	if doc.Nodes[0].Type != NodeTypeHeading {
		t.Errorf("node[0].Type = %v, want %v", doc.Nodes[0].Type, NodeTypeHeading)
	}
	if doc.Nodes[1].Type != NodeTypeList {
		t.Errorf("node[1].Type = %v, want %v", doc.Nodes[1].Type, NodeTypeList)
	}
}

// TestParser_ParseString_EmptyContent tests parsing empty content.
func TestParser_ParseString_EmptyContent(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	result, err := p.ParseString(ctx, "")
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	doc := result.Content.(*Document)
	if len(doc.Nodes) != 0 {
		t.Errorf("expected 0 nodes for empty content, got %d", len(doc.Nodes))
	}
}

// TestParser_ParseString_WhitespaceOnly tests parsing whitespace-only content.
func TestParser_ParseString_WhitespaceOnly(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	result, err := p.ParseString(ctx, "   \n\n   \n")
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	doc := result.Content.(*Document)
	if len(doc.Nodes) != 0 {
		t.Errorf("expected 0 nodes for whitespace-only content, got %d", len(doc.Nodes))
	}
}

// TestNode_String tests the String method of Node.
func TestNode_String(t *testing.T) {
	tests := []struct {
		node Node
		want string
	}{
		{
			node: Node{Type: NodeTypeHeading, Level: 1, Content: "Title"},
			want: "Heading1: Title",
		},
		{
			node: Node{Type: NodeTypeParagraph, Content: "Some text"},
			want: "Paragraph: Some text",
		},
		{
			node: Node{Type: NodeTypeList, ListType: ListTypeUnordered, Items: []string{"a", "b"}},
			want: "List (unordered): [a b]",
		},
		{
			node: Node{Type: NodeTypeCodeBlock, Language: "go", Content: "code\nhere"},
			want: "CodeBlock (go): 2 lines",
		},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := tc.node.String()
			if got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestDocument_String tests the String method of Document.
func TestDocument_String(t *testing.T) {
	doc := &Document{
		Nodes: []Node{
			{Type: NodeTypeHeading, Level: 1, Content: "Title"},
			{Type: NodeTypeParagraph, Content: "Text"},
		},
	}

	got := doc.String()
	want := "Heading1: Title\nParagraph: Text"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// TestParser_CodeBlockUnclosed tests unclosed code blocks.
func TestParser_CodeBlockUnclosed(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	content := "```go\n" + `func main() {
    println("test")
}`

	result, err := p.ParseString(ctx, content)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	doc := result.Content.(*Document)
	if len(doc.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(doc.Nodes))
	}

	node := doc.Nodes[0]
	if node.Type != NodeTypeCodeBlock {
		t.Errorf("node.Type = %v, want %v", node.Type, NodeTypeCodeBlock)
	}
	if node.Language != "go" {
		t.Errorf("node.Language = %q, want %q", node.Language, "go")
	}
	if node.Content == "" {
		t.Error("node.Content should not be empty")
	}
}

// TestParser_SeparateLists tests that different list types don't merge.
func TestParser_SeparateLists(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	content := `- Unordered item 1
- Unordered item 2

1. Ordered item 1
2. Ordered item 2`

	result, err := p.ParseString(ctx, content)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	doc := result.Content.(*Document)
	if len(doc.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(doc.Nodes))
	}

	if doc.Nodes[0].ListType != ListTypeUnordered {
		t.Errorf("node[0].ListType = %v, want %v", doc.Nodes[0].ListType, ListTypeUnordered)
	}
	if doc.Nodes[1].ListType != ListTypeOrdered {
		t.Errorf("node[1].ListType = %v, want %v", doc.Nodes[1].ListType, ListTypeOrdered)
	}
}

// TestParser_HeadingWithSpaces tests headings with extra spaces.
func TestParser_HeadingWithSpaces(t *testing.T) {
	p := NewParser()
	ctx := context.Background()

	content := `###   Multiple spaces before text
#### Tab	before text`

	result, err := p.ParseString(ctx, content)
	if err != nil {
		t.Fatalf("ParseString() error = %v", err)
	}

	doc := result.Content.(*Document)
	if len(doc.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(doc.Nodes))
	}

	if doc.Nodes[0].Content != "Multiple spaces before text" {
		t.Errorf("node[0].Content = %q, want %q", doc.Nodes[0].Content, "Multiple spaces before text")
	}
	if doc.Nodes[1].Content != "Tab\tbefore text" {
		t.Errorf("node[1].Content = %q, want %q", doc.Nodes[1].Content, "Tab\tbefore text")
	}
}

// BenchmarkParser_ParseDocument benchmarks the parser.
func BenchmarkParser_ParseDocument(b *testing.B) {
	p := NewParser()
	content := `# Benchmark Test

This is a paragraph for benchmarking.

## Section

- Item 1
- Item 2
- Item 3

1. Ordered 1
2. Ordered 2

` + "```go\n" + `func main() {}
` + "```"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := p.ParseDocument(content)
		if err != nil {
			b.Fatal(err)
		}
	}
}
