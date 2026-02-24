// Package markdown provides Markdown section extraction functionality.
package markdown

import (
	"fmt"
	"strings"
)

// Section represents a markdown section (heading and its content).
type Section struct {
	Title       string    `json:"title"`       // Section title (heading content)
	Level       int       `json:"level"`       // Heading level (1-6)
	StartIndex  int       `json:"start_index"` // Start index in the document nodes
	EndIndex    int       `json:"end_index"`   // End index in the document nodes (exclusive)
	Content     string    `json:"content"`     // Raw content between this heading and the next same/higher level heading
	Children    []Section `json:"children"`    // Sub-sections (lower level headings)
	ParentTitle string    `json:"parent_title,omitempty"` // Parent section title (if any)
}

// SectionExtractor provides functionality to extract sections from markdown documents.
type SectionExtractor struct {
	doc *Document
}

// headingInfo holds information about a heading.
type headingInfo struct {
	index int
	level int
	title string
}

// NewSectionExtractor creates a new section extractor for the given document.
func NewSectionExtractor(doc *Document) *SectionExtractor {
	return &SectionExtractor{doc: doc}
}

// ExtractSections extracts all sections from the document.
// Returns a slice of top-level sections (H1), each with their nested children.
func ExtractSections(doc *Document) ([]Section, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	extractor := NewSectionExtractor(doc)
	return extractor.extractAllSections(), nil
}

// FindSection finds a section by its title (case-insensitive).
// Returns the first matching section or an error if not found.
func FindSection(doc *Document, title string) (Section, error) {
	if doc == nil {
		return Section{}, fmt.Errorf("document is nil")
	}

	if strings.TrimSpace(title) == "" {
		return Section{}, fmt.Errorf("title cannot be empty")
	}

	sections, err := ExtractSections(doc)
	if err != nil {
		return Section{}, err
	}

	return findSectionRecursive(sections, title)
}

// FindSectionsByLevel finds all sections at a specific heading level.
func FindSectionsByLevel(doc *Document, level int) ([]Section, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	if level < 1 || level > 6 {
		return nil, fmt.Errorf("invalid heading level: %d (must be 1-6)", level)
	}

	allSections, err := ExtractSections(doc)
	if err != nil {
		return nil, err
	}

	var result []Section
	collectSectionsByLevel(allSections, level, &result)
	return result, nil
}

// extractAllSections extracts all top-level sections and their children.
func (se *SectionExtractor) extractAllSections() []Section {
	if se.doc == nil || len(se.doc.Nodes) == 0 {
		return []Section{}
	}

	// First pass: identify all headings and their positions
	var headings []headingInfo

	for i, node := range se.doc.Nodes {
		if node.Type == NodeTypeHeading {
			headings = append(headings, headingInfo{
				index: i,
				level: node.Level,
				title: node.Content,
			})
		}
	}

	if len(headings) == 0 {
		return []Section{}
	}

	// Build section hierarchy and compute end indices
	return se.buildSections(headings, 0, len(headings), 0, len(se.doc.Nodes), nil)
}

// buildSections builds sections recursively.
// It creates sections from headings[start:end] at the given base level.
// parentStart and parentEnd define the range of the parent section in the document.
func (se *SectionExtractor) buildSections(headings []headingInfo, start, end, level, docEnd int, parentTitle *string) []Section {
	var sections []Section

	for i := start; i < end; {
		h := headings[i]

		// Find the range of this section (up to next heading at same or lower level)
		sectionEnd := docEnd
		for j := i + 1; j < end; j++ {
			if headings[j].level <= h.level {
				sectionEnd = headings[j].index
				break
			}
		}
		if sectionEnd == docEnd && end < len(headings) {
			sectionEnd = docEnd
		}

		// Create section
		newSection := Section{
			Title:      h.title,
			Level:      h.level,
			StartIndex: h.index,
			EndIndex:   sectionEnd,
			Content:    se.extractContent(h.index, sectionEnd),
		}
		if parentTitle != nil {
			newSection.ParentTitle = *parentTitle
		}

		// Find children (headings with higher level between this and next sibling)
		childStart := i + 1
		childEnd := i + 1
		for j := i + 1; j < end; j++ {
			if headings[j].level <= h.level {
				break
			}
			childEnd = j + 1
		}

		if childStart < childEnd {
			newSection.Children = se.buildSections(headings, childStart, childEnd, level+1, sectionEnd, &newSection.Title)
		}

		sections = append(sections, newSection)

		// Move to next sibling
		i = childEnd
	}

	return sections
}

// extractContent extracts the raw content between start and end indices.
// This includes the heading and all content until the next section.
func (se *SectionExtractor) extractContent(startIdx, endIdx int) string {
	if startIdx >= len(se.doc.Nodes) || startIdx >= endIdx {
		return ""
	}

	var contentParts []string
	for i := startIdx; i < endIdx && i < len(se.doc.Nodes); i++ {
		node := se.doc.Nodes[i]
		contentParts = append(contentParts, nodeToString(node))
	}

	return strings.Join(contentParts, "\n")
}

// nodeToString converts a node back to its string representation.
func nodeToString(node Node) string {
	switch node.Type {
	case NodeTypeHeading:
		return fmt.Sprintf("%s %s", strings.Repeat("#", node.Level), node.Content)
	case NodeTypeParagraph:
		return node.Content
	case NodeTypeList:
		var items []string
		for _, item := range node.Items {
			if node.ListType == ListTypeOrdered {
				items = append(items, fmt.Sprintf("%d. %s", len(items)+1, item))
			} else {
				items = append(items, fmt.Sprintf("- %s", item))
			}
		}
		return strings.Join(items, "\n")
	case NodeTypeCodeBlock:
		if node.Language != "" {
			return fmt.Sprintf("```%s\n%s\n```", node.Language, node.Content)
		}
		return fmt.Sprintf("```\n%s\n```", node.Content)
	default:
		return node.Content
	}
}

// findSectionRecursive recursively searches for a section by title.
func findSectionRecursive(sections []Section, title string) (Section, error) {
	searchTitle := strings.ToLower(strings.TrimSpace(title))

	for _, sec := range sections {
		if strings.ToLower(strings.TrimSpace(sec.Title)) == searchTitle {
			return sec, nil
		}

		// Search in children
		if len(sec.Children) > 0 {
			found, err := findSectionRecursive(sec.Children, title)
			if err == nil {
				return found, nil
			}
		}
	}

	return Section{}, fmt.Errorf("section with title %q not found", title)
}

// collectSectionsByLevel collects all sections at a specific level.
func collectSectionsByLevel(sections []Section, level int, result *[]Section) {
	for _, sec := range sections {
		if sec.Level == level {
			*result = append(*result, sec)
		}
		if len(sec.Children) > 0 {
			collectSectionsByLevel(sec.Children, level, result)
		}
	}
}

// GetSectionContent returns only the content of a section (excluding the heading).
func GetSectionContent(section Section) string {
	lines := strings.Split(section.Content, "\n")
	if len(lines) <= 1 {
		return ""
	}
	return strings.Join(lines[1:], "\n")
}

// GetSectionHierarchy returns a flat list of all sections with their full paths.
func GetSectionHierarchy(doc *Document) ([]Section, error) {
	sections, err := ExtractSections(doc)
	if err != nil {
		return nil, err
	}

	var result []Section
	flattenSections(sections, &result)
	return result, nil
}

// flattenSections flattens the section hierarchy.
func flattenSections(sections []Section, result *[]Section) {
	for _, sec := range sections {
		*result = append(*result, sec)
		if len(sec.Children) > 0 {
			flattenSections(sec.Children, result)
		}
	}
}

// CountSections returns the total number of sections at all levels.
func CountSections(doc *Document) (int, error) {
	sections, err := ExtractSections(doc)
	if err != nil {
		return 0, err
	}

	return countSectionsRecursive(sections), nil
}

// countSectionsRecursive counts sections recursively.
func countSectionsRecursive(sections []Section) int {
	count := len(sections)
	for _, sec := range sections {
		count += countSectionsRecursive(sec.Children)
	}
	return count
}
