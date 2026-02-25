// Package markdown provides Markdown metadata extraction functionality.
package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

// MetadataExtractor provides functionality to extract metadata from markdown documents
type MetadataExtractor struct {
	content string
}

// Regular expressions for parsing YAML frontmatter
var (
	// Frontmatter pattern: --- at start of file, followed by content, then ---
	// Matches the entire frontmatter block including delimiters
	frontmatterRegex = regexp.MustCompile(`(?s)^\s*---\s*\n(.*?)\n---\s*(?:\n|$)`)

	// Key-value pair pattern: key: value
	// Supports keys with letters, numbers, underscores, and hyphens
	kvRegex = regexp.MustCompile(`^[\s]*([a-zA-Z0-9_-]+)[\s]*:[\s]*(.*)$`)
)

// NewMetadataExtractor creates a new metadata extractor for the given content
func NewMetadataExtractor(content string) *MetadataExtractor {
	return &MetadataExtractor{content: content}
}

// ExtractMetadata extracts YAML frontmatter metadata from markdown content.
// Returns a map of key-value pairs found in the frontmatter.
// If no frontmatter is found, returns an empty map.
func ExtractMetadata(content string) (map[string]string, error) {
	extractor := NewMetadataExtractor(content)
	return extractor.extractMetadata()
}

// extractMetadata performs the actual metadata extraction
func (me *MetadataExtractor) extractMetadata() (map[string]string, error) {
	metadata := make(map[string]string)

	// Check if content has frontmatter
	matches := frontmatterRegex.FindStringSubmatch(me.content)
	if matches == nil {
		// No frontmatter found, return empty map
		return metadata, nil
	}

	// Extract the frontmatter content (without delimiters)
	frontmatterContent := matches[1]

	// Parse each line of the frontmatter
	lines := strings.Split(frontmatterContent, "\n")
	for _, line := range lines {
		// Skip empty lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Try to parse as key-value pair
		if kvMatches := kvRegex.FindStringSubmatch(line); kvMatches != nil {
			key := strings.TrimSpace(kvMatches[1])
			value := strings.TrimSpace(kvMatches[2])

			// Validate key is not empty
			if key == "" {
				continue
			}

			metadata[key] = value
		}
	}

	return metadata, nil
}

// ExtractMetadataFromDocument extracts metadata from a parsed Document.
// This is a convenience function that first reconstructs the document content.
func ExtractMetadataFromDocument(doc *Document) (map[string]string, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	// Reconstruct content from document nodes
	var content strings.Builder
	for i, node := range doc.Nodes {
		if i > 0 {
			content.WriteString("\n")
		}
		content.WriteString(nodeToRawContent(node))
	}

	return ExtractMetadata(content.String())
}

// nodeToRawContent converts a node back to raw markdown content
func nodeToRawContent(node Node) string {
	switch node.Type {
	case NodeTypeHeading:
		prefix := strings.Repeat("#", node.Level)
		return prefix + " " + node.Content
	case NodeTypeParagraph:
		return node.Content
	case NodeTypeList:
		var items []string
		for i, item := range node.Items {
			indent := ""
			if i < len(node.ItemIndents) {
				indent = strings.Repeat("  ", node.ItemIndents[i])
			}
			prefix := "- "
			if node.ListType == ListTypeOrdered {
				prefix = "1. "
			}
			items = append(items, indent+prefix+item)
		}
		return strings.Join(items, "\n")
	case NodeTypeCodeBlock:
		lang := node.Language
		return "```" + lang + "\n" + node.Content + "\n```"
	default:
		return ""
	}
}

// HasFrontmatter checks if the content has YAML frontmatter
func HasFrontmatter(content string) bool {
	return frontmatterRegex.MatchString(content)
}

// GetFrontmatterRaw returns the raw frontmatter content (without delimiters)
// Returns empty string if no frontmatter is found
func GetFrontmatterRaw(content string) string {
	matches := frontmatterRegex.FindStringSubmatch(content)
	if matches == nil {
		return ""
	}
	return matches[1]
}

// ValidateFrontmatter checks if the frontmatter format is valid
// Returns an error if the frontmatter is malformed
func ValidateFrontmatter(content string) error {
	// Check for unclosed frontmatter
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "---") {
		// Count frontmatter delimiters
		lines := strings.Split(trimmed, "\n")
		delimiterCount := 0
		for _, line := range lines {
			if strings.TrimSpace(line) == "---" {
				delimiterCount++
			}
		}

		if delimiterCount < 2 {
			return fmt.Errorf("unclosed frontmatter: missing closing '---'")
		}
	}

	return nil
}
