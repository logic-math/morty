// Package markdown provides a Markdown parser implementation.
package markdown

import (
	"github.com/morty/morty/internal/parser"
)

// Register registers the Markdown parser with the given factory.
func Register(f *parser.Factory) error {
	return f.Register(parser.FileTypeMarkdown, NewParser())
}

// RegisterWithDefaults creates a factory with default extension mappings
// and registers the Markdown parser.
func RegisterWithDefaults() (*parser.Factory, error) {
	f := parser.NewFactoryWithDefaults()
	if err := Register(f); err != nil {
		return nil, err
	}
	return f, nil
}
