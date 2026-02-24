package markdown

import (
	"strings"
	"testing"
)

// TestExtractSections_Basic tests basic section extraction.
func TestExtractSections_Basic(t *testing.T) {
	content := `# Title

Introduction paragraph.

## Section 1

Content of section 1.

## Section 2

Content of section 2.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	sections, err := ExtractSections(doc)
	if err != nil {
		t.Fatalf("ExtractSections() error = %v", err)
	}

	if len(sections) != 1 {
		t.Fatalf("expected 1 top-level section, got %d", len(sections))
	}

	// Check top-level section
	if sections[0].Title != "Title" {
		t.Errorf("sections[0].Title = %q, want %q", sections[0].Title, "Title")
	}
	if sections[0].Level != 1 {
		t.Errorf("sections[0].Level = %d, want %d", sections[0].Level, 1)
	}

	// Check children
	if len(sections[0].Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(sections[0].Children))
	}

	if sections[0].Children[0].Title != "Section 1" {
		t.Errorf("children[0].Title = %q, want %q", sections[0].Children[0].Title, "Section 1")
	}
	if sections[0].Children[1].Title != "Section 2" {
		t.Errorf("children[1].Title = %q, want %q", sections[0].Children[1].Title, "Section 2")
	}
}

// TestExtractSections_MultipleTopLevel tests multiple top-level sections.
func TestExtractSections_MultipleTopLevel(t *testing.T) {
	content := `# First Title

Content 1.

# Second Title

Content 2.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	sections, err := ExtractSections(doc)
	if err != nil {
		t.Fatalf("ExtractSections() error = %v", err)
	}

	if len(sections) != 2 {
		t.Fatalf("expected 2 top-level sections, got %d", len(sections))
	}

	if sections[0].Title != "First Title" {
		t.Errorf("sections[0].Title = %q, want %q", sections[0].Title, "First Title")
	}
	if sections[1].Title != "Second Title" {
		t.Errorf("sections[1].Title = %q, want %q", sections[1].Title, "Second Title")
	}
}

// TestExtractSections_DeepNesting tests deeply nested sections.
func TestExtractSections_DeepNesting(t *testing.T) {
	content := `# Level 1

## Level 2

### Level 3

#### Level 4

##### Level 5

###### Level 6

Deep content.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	sections, err := ExtractSections(doc)
	if err != nil {
		t.Fatalf("ExtractSections() error = %v", err)
	}

	if len(sections) != 1 {
		t.Fatalf("expected 1 top-level section, got %d", len(sections))
	}

	// Navigate deep nesting
	sec := sections[0]
	if sec.Level != 1 {
		t.Errorf("top level = %d, want 1", sec.Level)
	}

	sec = sec.Children[0]
	if sec.Level != 2 {
		t.Errorf("level 2 = %d, want 2", sec.Level)
	}

	sec = sec.Children[0]
	if sec.Level != 3 {
		t.Errorf("level 3 = %d, want 3", sec.Level)
	}

	sec = sec.Children[0]
	if sec.Level != 4 {
		t.Errorf("level 4 = %d, want 4", sec.Level)
	}

	sec = sec.Children[0]
	if sec.Level != 5 {
		t.Errorf("level 5 = %d, want 5", sec.Level)
	}

	sec = sec.Children[0]
	if sec.Level != 6 {
		t.Errorf("level 6 = %d, want 6", sec.Level)
	}
}

// TestExtractSections_ContentRange tests section content range extraction.
func TestExtractSections_ContentRange(t *testing.T) {
	content := `# Main Title

Intro text.

## Section A

- Item 1
- Item 2

## Section B

Some code:

` + "```go\nfunc main() {}\n```\n" +
		`End of section.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	sections, err := ExtractSections(doc)
	if err != nil {
		t.Fatalf("ExtractSections() error = %v", err)
	}

	// Check main section content
	mainSection := sections[0]
	if !strings.Contains(mainSection.Content, "Intro text") {
		t.Errorf("main section content should contain 'Intro text'")
	}
	if !strings.Contains(mainSection.Content, "Main Title") {
		t.Errorf("main section content should contain 'Main Title'")
	}

	// Check Section A content
	if len(mainSection.Children) < 2 {
		t.Fatalf("expected at least 2 children, got %d", len(mainSection.Children))
	}

	sectionA := mainSection.Children[0]
	if sectionA.Title != "Section A" {
		t.Errorf("sectionA.Title = %q, want %q", sectionA.Title, "Section A")
	}
	if !strings.Contains(sectionA.Content, "Item 1") {
		t.Errorf("section A content should contain 'Item 1'")
	}

	// Check Section B content
	sectionB := mainSection.Children[1]
	if sectionB.Title != "Section B" {
		t.Errorf("sectionB.Title = %q, want %q", sectionB.Title, "Section B")
	}
	if !strings.Contains(sectionB.Content, "func main()") {
		t.Errorf("section B content should contain 'func main()'")
	}
}

// TestExtractSections_EmptyDocument tests extraction from empty document.
func TestExtractSections_EmptyDocument(t *testing.T) {
	p := NewParser()
	doc, err := p.ParseDocument("")
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	sections, err := ExtractSections(doc)
	if err != nil {
		t.Fatalf("ExtractSections() error = %v", err)
	}

	if len(sections) != 0 {
		t.Errorf("expected 0 sections for empty document, got %d", len(sections))
	}
}

// TestExtractSections_NoHeadings tests document without headings.
func TestExtractSections_NoHeadings(t *testing.T) {
	content := `This is just a paragraph.

- And a list item.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	sections, err := ExtractSections(doc)
	if err != nil {
		t.Fatalf("ExtractSections() error = %v", err)
	}

	if len(sections) != 0 {
		t.Errorf("expected 0 sections for document without headings, got %d", len(sections))
	}
}

// TestExtractSections_NilDocument tests extraction from nil document.
func TestExtractSections_NilDocument(t *testing.T) {
	sections, err := ExtractSections(nil)
	if err == nil {
		t.Error("ExtractSections(nil) should return an error")
	}
	if sections != nil {
		t.Error("ExtractSections(nil) should return nil sections")
	}
}

// TestFindSection_Basic tests basic section finding.
func TestFindSection_Basic(t *testing.T) {
	content := `# Introduction

Intro text.

## Installation

Install steps.

## Usage

Usage instructions.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	// Find top-level section
	section, err := FindSection(doc, "Introduction")
	if err != nil {
		t.Fatalf("FindSection('Introduction') error = %v", err)
	}
	if section.Title != "Introduction" {
		t.Errorf("section.Title = %q, want %q", section.Title, "Introduction")
	}
	if section.Level != 1 {
		t.Errorf("section.Level = %d, want %d", section.Level, 1)
	}

	// Find nested section
	section, err = FindSection(doc, "Installation")
	if err != nil {
		t.Fatalf("FindSection('Installation') error = %v", err)
	}
	if section.Title != "Installation" {
		t.Errorf("section.Title = %q, want %q", section.Title, "Installation")
	}
	if section.Level != 2 {
		t.Errorf("section.Level = %d, want %d", section.Level, 2)
	}
}

// TestFindSection_CaseInsensitive tests case-insensitive search.
func TestFindSection_CaseInsensitive(t *testing.T) {
	content := `# Getting Started

Some content.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	// Try different cases
	cases := []string{"getting started", "GETTING STARTED", "Getting Started", "gEtTiNg StArTeD"}
	for _, tc := range cases {
		section, err := FindSection(doc, tc)
		if err != nil {
			t.Errorf("FindSection(%q) error = %v", tc, err)
			continue
		}
		if section.Title != "Getting Started" {
			t.Errorf("FindSection(%q) returned %q, want %q", tc, section.Title, "Getting Started")
		}
	}
}

// TestFindSection_NotFound tests finding non-existent section.
func TestFindSection_NotFound(t *testing.T) {
	content := `# Title

Content.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	_, err = FindSection(doc, "NonExistent")
	if err == nil {
		t.Error("FindSection('NonExistent') should return an error")
	}
}

// TestFindSection_EmptyTitle tests finding with empty title.
func TestFindSection_EmptyTitle(t *testing.T) {
	content := `# Title

Content.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	_, err = FindSection(doc, "")
	if err == nil {
		t.Error("FindSection('') should return an error")
	}

	_, err = FindSection(doc, "   ")
	if err == nil {
		t.Error("FindSection('   ') should return an error")
	}
}

// TestFindSection_DeepNested tests finding deeply nested sections.
func TestFindSection_DeepNested(t *testing.T) {
	content := `# Level 1

## Level 2 A

### Level 3

Content here.

## Level 2 B

Other content.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	// Find deeply nested section
	section, err := FindSection(doc, "Level 3")
	if err != nil {
		t.Fatalf("FindSection('Level 3') error = %v", err)
	}
	if section.Title != "Level 3" {
		t.Errorf("section.Title = %q, want %q", section.Title, "Level 3")
	}
	if section.Level != 3 {
		t.Errorf("section.Level = %d, want %d", section.Level, 3)
	}
	if section.ParentTitle != "Level 2 A" {
		t.Errorf("section.ParentTitle = %q, want %q", section.ParentTitle, "Level 2 A")
	}
}

// TestFindSectionsByLevel tests finding sections by level.
func TestFindSectionsByLevel(t *testing.T) {
	content := `# H1 First

## H2 First

### H3 First

## H2 Second

# H1 Second

## H2 Third`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	// Find all H1 sections
	h1Sections, err := FindSectionsByLevel(doc, 1)
	if err != nil {
		t.Fatalf("FindSectionsByLevel(1) error = %v", err)
	}
	if len(h1Sections) != 2 {
		t.Errorf("expected 2 H1 sections, got %d", len(h1Sections))
	}

	// Find all H2 sections
	h2Sections, err := FindSectionsByLevel(doc, 2)
	if err != nil {
		t.Fatalf("FindSectionsByLevel(2) error = %v", err)
	}
	if len(h2Sections) != 3 {
		t.Errorf("expected 3 H2 sections, got %d", len(h2Sections))
	}

	// Find all H3 sections
	h3Sections, err := FindSectionsByLevel(doc, 3)
	if err != nil {
		t.Fatalf("FindSectionsByLevel(3) error = %v", err)
	}
	if len(h3Sections) != 1 {
		t.Errorf("expected 1 H3 section, got %d", len(h3Sections))
	}
}

// TestFindSectionsByLevel_InvalidLevel tests invalid level values.
func TestFindSectionsByLevel_InvalidLevel(t *testing.T) {
	content := `# Title`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	invalidLevels := []int{0, -1, 7, 10}
	for _, level := range invalidLevels {
		_, err := FindSectionsByLevel(doc, level)
		if err == nil {
			t.Errorf("FindSectionsByLevel(%d) should return an error", level)
		}
	}
}

// TestGetSectionContent tests extracting section content without heading.
func TestGetSectionContent(t *testing.T) {
	content := `# Section Title

Paragraph 1.

Paragraph 2.

- Item 1
- Item 2`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	sections, err := ExtractSections(doc)
	if err != nil {
		t.Fatalf("ExtractSections() error = %v", err)
	}

	sectionContent := GetSectionContent(sections[0])
	if strings.Contains(sectionContent, "# Section Title") {
		t.Error("GetSectionContent should not include the heading")
	}
	if !strings.Contains(sectionContent, "Paragraph 1") {
		t.Error("GetSectionContent should include 'Paragraph 1'")
	}
	if !strings.Contains(sectionContent, "Item 1") {
		t.Error("GetSectionContent should include 'Item 1'")
	}
}

// TestGetSectionHierarchy tests getting flat section hierarchy.
func TestGetSectionHierarchy(t *testing.T) {
	content := `# Level 1

## Level 2 A

### Level 3

## Level 2 B`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	hierarchy, err := GetSectionHierarchy(doc)
	if err != nil {
		t.Fatalf("GetSectionHierarchy() error = %v", err)
	}

	// Should have 4 sections total (L1, L2A, L3, L2B)
	if len(hierarchy) != 4 {
		t.Errorf("expected 4 sections in hierarchy, got %d", len(hierarchy))
	}

	// Check order
	expectedTitles := []string{"Level 1", "Level 2 A", "Level 3", "Level 2 B"}
	for i, expected := range expectedTitles {
		if hierarchy[i].Title != expected {
			t.Errorf("hierarchy[%d].Title = %q, want %q", i, hierarchy[i].Title, expected)
		}
	}
}

// TestCountSections tests counting total sections.
func TestCountSections(t *testing.T) {
	content := `# H1

## H2 A

### H3

## H2 B

# Another H1

## H2 C`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	count, err := CountSections(doc)
	if err != nil {
		t.Fatalf("CountSections() error = %v", err)
	}

	// 2 H1 + 3 H2 + 1 H3 = 6 sections
	if count != 6 {
		t.Errorf("CountSections() = %d, want %d", count, 6)
	}
}

// TestExtractSections_ComplexDocument tests extraction from a complex document.
func TestExtractSections_ComplexDocument(t *testing.T) {
	content := `# Project Documentation

Welcome to the project.

## Installation

### Requirements

- Go 1.21+
- Node.js 18+

### Steps

1. Clone repo
2. Run make

## API Reference

### Authentication

` + "```http\nPOST /api/auth\n```\n" +
		`### Endpoints

#### Users

User endpoints here.

#### Posts

Post endpoints here.

## Contributing

See CONTRIBUTING.md.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	sections, err := ExtractSections(doc)
	if err != nil {
		t.Fatalf("ExtractSections() error = %v", err)
	}

	// Verify top-level structure
	if len(sections) != 1 {
		t.Fatalf("expected 1 top-level section, got %d", len(sections))
	}

	root := sections[0]
	if root.Title != "Project Documentation" {
		t.Errorf("root.Title = %q, want %q", root.Title, "Project Documentation")
	}

	// Should have 3 H2 children: Installation, API Reference, Contributing
	if len(root.Children) != 3 {
		t.Fatalf("expected 3 H2 children, got %d", len(root.Children))
	}

	// Check Installation children (Requirements, Steps)
	installation := root.Children[0]
	if installation.Title != "Installation" {
		t.Errorf("children[0].Title = %q, want %q", installation.Title, "Installation")
	}
	if len(installation.Children) != 2 {
		t.Errorf("expected 2 children under Installation, got %d", len(installation.Children))
	}

	// Check API Reference children (Authentication, Endpoints)
	apiRef := root.Children[1]
	if apiRef.Title != "API Reference" {
		t.Errorf("children[1].Title = %q, want %q", apiRef.Title, "API Reference")
	}
	if len(apiRef.Children) != 2 {
		t.Errorf("expected 2 children under API Reference, got %d", len(apiRef.Children))
	}

	// Endpoints should have 2 H4 children (Users, Posts)
	endpoints := apiRef.Children[1]
	if endpoints.Title != "Endpoints" {
		t.Errorf("apiRef.Children[1].Title = %q, want %q", endpoints.Title, "Endpoints")
	}
	if len(endpoints.Children) != 2 {
		t.Errorf("expected 2 children under Endpoints, got %d", len(endpoints.Children))
	}
}

// TestExtractSections_IndexRange tests start and end index values.
func TestExtractSections_IndexRange(t *testing.T) {
	content := `# First

Content 1.

## Second

Content 2.

### Third

Content 3.`

	p := NewParser()
	doc, err := p.ParseDocument(content)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	sections, err := ExtractSections(doc)
	if err != nil {
		t.Fatalf("ExtractSections() error = %v", err)
	}

	first := sections[0]
	if first.StartIndex < 0 {
		t.Errorf("StartIndex should be >= 0, got %d", first.StartIndex)
	}
	if first.EndIndex <= first.StartIndex {
		t.Errorf("EndIndex (%d) should be > StartIndex (%d)", first.EndIndex, first.StartIndex)
	}
	if first.EndIndex > len(doc.Nodes) {
		t.Errorf("EndIndex (%d) should be <= len(Nodes) (%d)", first.EndIndex, len(doc.Nodes))
	}

	// Check nested section
	second := first.Children[0]
	if second.StartIndex <= first.StartIndex {
		t.Errorf("nested StartIndex (%d) should be > parent StartIndex (%d)", second.StartIndex, first.StartIndex)
	}
	if second.EndIndex > first.EndIndex {
		t.Errorf("nested EndIndex (%d) should be <= parent EndIndex (%d)", second.EndIndex, first.EndIndex)
	}
}

// BenchmarkExtractSections benchmarks section extraction.
func BenchmarkExtractSections(b *testing.B) {
	content := `# Title

Introduction.

## Section 1

Content 1.

### Subsection 1.1

Deep content.

## Section 2

Content 2.`

	p := NewParser()
	doc, _ := p.ParseDocument(content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ExtractSections(doc)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFindSection benchmarks section finding.
func BenchmarkFindSection(b *testing.B) {
	content := `# Title

## Section 1

### Subsection 1.1

## Section 2

## Section 3`

	p := NewParser()
	doc, _ := p.ParseDocument(content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FindSection(doc, "Subsection 1.1")
		if err != nil {
			b.Fatal(err)
		}
	}
}
