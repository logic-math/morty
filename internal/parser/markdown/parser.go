// Package markdown provides a Markdown parser implementation.
package markdown

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/morty/morty/internal/parser"
)

// NodeType represents the type of a document node.
type NodeType string

const (
	// NodeTypeHeading represents a heading (H1-H6).
	NodeTypeHeading NodeType = "heading"
	// NodeTypeParagraph represents a paragraph.
	NodeTypeParagraph NodeType = "paragraph"
	// NodeTypeList represents a list (ordered or unordered).
	NodeTypeList NodeType = "list"
	// NodeTypeCodeBlock represents a code block.
	NodeTypeCodeBlock NodeType = "codeblock"
	// NodeTypeText represents plain text.
	NodeTypeText NodeType = "text"
)

// ListType represents the type of list.
type ListType string

const (
	// ListTypeOrdered represents an ordered list (1., 2., etc.).
	ListTypeOrdered ListType = "ordered"
	// ListTypeUnordered represents an unordered list (-, *, +).
	ListTypeUnordered ListType = "unordered"
)

// Node represents a single node in the parsed document.
type Node struct {
	Type     NodeType               `json:"type"`
	Level    int                    `json:"level,omitempty"`     // For headings (1-6)
	Content  string                 `json:"content,omitempty"`   // For text and paragraphs
	Language string                 `json:"language,omitempty"`  // For code blocks
	ListType ListType               `json:"listType,omitempty"`  // For lists
	Items    []string               `json:"items,omitempty"`     // For list items
	Children []Node                 `json:"children,omitempty"`  // For nested content
	Metadata map[string]interface{} `json:"metadata,omitempty"`  // Additional metadata
}

// Document represents a parsed Markdown document.
type Document struct {
	Nodes []Node `json:"nodes"`
}

// Parser implements the parser.Parser interface for Markdown files.
type Parser struct{}

// Ensure Parser implements the parser.Parser interface.
var _ parser.Parser = (*Parser)(nil)

// NewParser creates a new Markdown parser instance.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads from r and returns the parsed result.
func (p *Parser) Parse(ctx context.Context, r io.Reader) (*parser.ParseResult, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	doc, err := p.parseContent(string(content))
	if err != nil {
		return &parser.ParseResult{
			Type:    parser.FileTypeMarkdown,
			Content: nil,
			Errors:  []error{err},
		}, nil
	}

	return &parser.ParseResult{
		Type:    parser.FileTypeMarkdown,
		Content: doc,
		Errors:  nil,
	}, nil
}

// ParseString parses content from a string.
func (p *Parser) ParseString(ctx context.Context, content string) (*parser.ParseResult, error) {
	return p.Parse(ctx, strings.NewReader(content))
}

// Supports returns true if this parser can handle the given file type.
func (p *Parser) Supports(fileType parser.FileType) bool {
	return fileType == parser.FileTypeMarkdown
}

// FileType returns the file type this parser handles.
func (p *Parser) FileType() parser.FileType {
	return parser.FileTypeMarkdown
}

// ParseDocument parses markdown content and returns a Document.
func (p *Parser) ParseDocument(content string) (*Document, error) {
	return p.parseContent(content)
}

// Regular expressions for parsing.
var (
	// Heading pattern: ### Heading
	headingRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

	// Unordered list pattern: - item, * item, + item
	unorderedListRegex = regexp.MustCompile(`^[\s]*[-\*\+]\s+(.+)$`)

	// Ordered list pattern: 1. item
	orderedListRegex = regexp.MustCompile(`^[\s]*(\d+)\.\s+(.+)$`)

	// Code block start/end pattern: ```language or ```
	codeBlockRegex = regexp.MustCompile(`^\s*` + "`" + `{3}(\w*)\s*$`)
)

// parseContent parses the markdown content into a Document.
func (p *Parser) parseContent(content string) (*Document, error) {
	doc := &Document{
		Nodes: make([]Node, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	var currentCodeBlock *Node
	var codeBlockLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Handle code blocks
		if currentCodeBlock != nil {
			if codeBlockRegex.MatchString(line) {
				// End of code block
				currentCodeBlock.Content = strings.Join(codeBlockLines, "\n")
				doc.Nodes = append(doc.Nodes, *currentCodeBlock)
				currentCodeBlock = nil
				codeBlockLines = nil
			} else {
				codeBlockLines = append(codeBlockLines, line)
			}
			continue
		}

		// Check for code block start
		if matches := codeBlockRegex.FindStringSubmatch(line); matches != nil {
			currentCodeBlock = &Node{
				Type:     NodeTypeCodeBlock,
				Language: matches[1],
			}
			codeBlockLines = make([]string, 0)
			continue
		}

		// Parse other elements
		node := p.parseLine(line)
		if node != nil {
			doc.Nodes = append(doc.Nodes, *node)
		}
	}

	// Handle unclosed code block
	if currentCodeBlock != nil {
		currentCodeBlock.Content = strings.Join(codeBlockLines, "\n")
		doc.Nodes = append(doc.Nodes, *currentCodeBlock)
	}

	// Post-process to merge consecutive list items
	doc.Nodes = p.mergeListItems(doc.Nodes)

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning content: %w", err)
	}

	return doc, nil
}

// parseLine parses a single line and returns a Node.
func (p *Parser) parseLine(line string) *Node {
	trimmed := strings.TrimSpace(line)

	// Empty line - skip
	if trimmed == "" {
		return nil
	}

	// Heading
	if matches := headingRegex.FindStringSubmatch(trimmed); matches != nil {
		level := len(matches[1])
		return &Node{
			Type:    NodeTypeHeading,
			Level:   level,
			Content: strings.TrimSpace(matches[2]),
		}
	}

	// Unordered list item
	if matches := unorderedListRegex.FindStringSubmatch(trimmed); matches != nil {
		return &Node{
			Type:     NodeTypeList,
			ListType: ListTypeUnordered,
			Items:    []string{matches[1]},
		}
	}

	// Ordered list item
	if matches := orderedListRegex.FindStringSubmatch(trimmed); matches != nil {
		return &Node{
			Type:     NodeTypeList,
			ListType: ListTypeOrdered,
			Items:    []string{matches[2]},
			Metadata: map[string]interface{}{
				"start": matches[1],
			},
		}
	}

	// Paragraph (default)
	return &Node{
		Type:    NodeTypeParagraph,
		Content: trimmed,
	}
}

// mergeListItems merges consecutive list items of the same type.
func (p *Parser) mergeListItems(nodes []Node) []Node {
	if len(nodes) == 0 {
		return nodes
	}

	result := make([]Node, 0, len(nodes))
	var currentList *Node

	for i := range nodes {
		node := nodes[i]

		if node.Type == NodeTypeList {
			if currentList != nil && currentList.ListType == node.ListType {
				// Merge with current list
				currentList.Items = append(currentList.Items, node.Items...)
				// Merge metadata if ordered list
				if node.ListType == ListTypeOrdered && node.Metadata != nil {
					if currentList.Metadata == nil {
						currentList.Metadata = make(map[string]interface{})
					}
					currentList.Metadata["numbers"] = append(
						currentList.Metadata["numbers"].([]string),
						node.Metadata["start"].(string),
					)
				}
			} else {
				// Start a new list
				if currentList != nil {
					result = append(result, *currentList)
				}
				newList := node
				currentList = &newList
				if node.ListType == ListTypeOrdered && node.Metadata != nil {
					currentList.Metadata["numbers"] = []string{node.Metadata["start"].(string)}
				}
			}
		} else {
			// Not a list item, flush current list if any
			if currentList != nil {
				result = append(result, *currentList)
				currentList = nil
			}
			result = append(result, node)
		}
	}

	// Don't forget the last list
	if currentList != nil {
		result = append(result, *currentList)
	}

	return result
}

// String returns a string representation of the document (for debugging).
func (d *Document) String() string {
	var sb strings.Builder
	for i, node := range d.Nodes {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(node.String())
	}
	return sb.String()
}

// String returns a string representation of the node.
func (n *Node) String() string {
	switch n.Type {
	case NodeTypeHeading:
		return fmt.Sprintf("Heading%d: %s", n.Level, n.Content)
	case NodeTypeParagraph:
		return fmt.Sprintf("Paragraph: %s", n.Content)
	case NodeTypeList:
		return fmt.Sprintf("List (%s): %v", n.ListType, n.Items)
	case NodeTypeCodeBlock:
		return fmt.Sprintf("CodeBlock (%s): %d lines", n.Language, len(strings.Split(n.Content, "\n")))
	default:
		return fmt.Sprintf("Unknown: %s", n.Type)
	}
}
