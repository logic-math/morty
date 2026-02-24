// Package parser provides a generic file parsing framework with support
// for multiple file formats including Markdown, JSON, and YAML.
package parser

import (
	"context"
	"io"
)

// FileType represents the type of file being parsed
type FileType string

// Supported file types
const (
	FileTypeMarkdown FileType = "markdown"
	FileTypeJSON     FileType = "json"
	FileTypeYAML     FileType = "yaml"
	FileTypeUnknown  FileType = "unknown"
)

// ParseResult represents the result of parsing a file
type ParseResult struct {
	Type    FileType
	Content interface{}
	Errors  []error
}

// Parser defines the interface for file parsers
type Parser interface {
	// Parse reads from r and returns the parsed result
	Parse(ctx context.Context, r io.Reader) (*ParseResult, error)

	// ParseString parses content from a string
	ParseString(ctx context.Context, content string) (*ParseResult, error)

	// Supports returns true if this parser can handle the given file type
	Supports(fileType FileType) bool

	// FileType returns the file type this parser handles
	FileType() FileType
}

// ExtensionMapping maps file extensions to file types
var ExtensionMapping = map[string]FileType{
	".md":        FileTypeMarkdown,
	".markdown":  FileTypeMarkdown,
	".json":      FileTypeJSON,
	".yaml":      FileTypeYAML,
	".yml":       FileTypeYAML,
}
