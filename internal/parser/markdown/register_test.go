package markdown

import (
	"testing"

	"github.com/morty/morty/internal/parser"
)

// TestRegister tests the Register function.
func TestRegister(t *testing.T) {
	f := parser.NewFactory()

	// Register the markdown parser
	err := Register(f)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Verify it's registered
	if !f.IsRegistered(parser.FileTypeMarkdown) {
		t.Error("Markdown parser not registered")
	}

	// Try to get the parser
	p, err := f.Get(parser.FileTypeMarkdown)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Verify it's a markdown parser
	mdParser, ok := p.(*Parser)
	if !ok {
		t.Error("Registered parser is not *markdown.Parser")
	}

	// Verify it supports markdown
	if !mdParser.Supports(parser.FileTypeMarkdown) {
		t.Error("Parser does not support markdown")
	}
}

// TestRegister_Duplicate tests registering twice returns error.
func TestRegister_Duplicate(t *testing.T) {
	f := parser.NewFactory()

	// First registration should succeed
	err := Register(f)
	if err != nil {
		t.Fatalf("First Register() error = %v", err)
	}

	// Second registration should fail
	err = Register(f)
	if err == nil {
		t.Error("Second Register() should return error")
	}
}

// TestRegisterWithDefaults tests the RegisterWithDefaults function.
func TestRegisterWithDefaults(t *testing.T) {
	f, err := RegisterWithDefaults()
	if err != nil {
		t.Fatalf("RegisterWithDefaults() error = %v", err)
	}

	if f == nil {
		t.Fatal("RegisterWithDefaults() returned nil factory")
	}

	// Verify markdown parser is registered
	if !f.IsRegistered(parser.FileTypeMarkdown) {
		t.Error("Markdown parser not registered")
	}

	// Verify extension mappings are set
	tests := []struct {
		filename string
		wantType parser.FileType
	}{
		{"test.md", parser.FileTypeMarkdown},
		{"test.markdown", parser.FileTypeMarkdown},
		{"test.json", parser.FileTypeJSON},
		{"test.yaml", parser.FileTypeYAML},
		{"test.yml", parser.FileTypeYAML},
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			got := f.DetectFileType(tc.filename)
			if got != tc.wantType {
				t.Errorf("DetectFileType(%q) = %v, want %v", tc.filename, got, tc.wantType)
			}
		})
	}
}

// TestFactory_GetByExtension tests retrieving markdown parser by extension.
func TestFactory_GetByExtension_Markdown(t *testing.T) {
	f, err := RegisterWithDefaults()
	if err != nil {
		t.Fatalf("RegisterWithDefaults() error = %v", err)
	}

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "markdown .md",
			filename: "test.md",
			wantErr:  false,
		},
		{
			name:     "markdown .markdown",
			filename: "test.markdown",
			wantErr:  false,
		},
		{
			name:     "unknown extension",
			filename: "test.unknown",
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := f.GetByExtension(tc.filename)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("GetByExtension() error = %v", err)
				return
			}
			if p == nil {
				t.Error("GetByExtension() returned nil parser")
			}
			if !p.Supports(parser.FileTypeMarkdown) {
				t.Error("parser does not support markdown")
			}
		})
	}
}

// TestFactory_ParseFile_Markdown tests parsing markdown files through the factory.
func TestFactory_ParseFile_Markdown(t *testing.T) {
	f, err := RegisterWithDefaults()
	if err != nil {
		t.Fatalf("RegisterWithDefaults() error = %v", err)
	}

	content := `# Test Document

This is a test.

- Item 1
- Item 2

` + "```go\n" + `fmt.Println("Hello")
` + "```"

	result, err := f.ParseFile(nil, "test.md", content)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	if result.Type != parser.FileTypeMarkdown {
		t.Errorf("result.Type = %v, want %v", result.Type, parser.FileTypeMarkdown)
	}

	doc, ok := result.Content.(*Document)
	if !ok {
		t.Fatal("result.Content is not *Document")
	}

	// Should have: H1, Paragraph, List, CodeBlock
	if len(doc.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(doc.Nodes))
		for i, node := range doc.Nodes {
			t.Logf("Node %d: %s", i, node.String())
		}
	}
}

// TestFactory_ListRegistered_WithMarkdown tests listing registered parsers.
func TestFactory_ListRegistered_WithMarkdown(t *testing.T) {
	f, err := RegisterWithDefaults()
	if err != nil {
		t.Fatalf("RegisterWithDefaults() error = %v", err)
	}

	registered := f.ListRegistered()
	if len(registered) != 1 {
		t.Errorf("expected 1 registered parser, got %d", len(registered))
	}

	if len(registered) > 0 && registered[0] != parser.FileTypeMarkdown {
		t.Errorf("expected FileTypeMarkdown, got %v", registered[0])
	}
}
