package parser

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// Common errors
var (
	ErrUnknownFileType   = fmt.Errorf("unknown file type")
	ErrParserNotFound    = fmt.Errorf("parser not found")
	ErrParserAlreadyExists = fmt.Errorf("parser already registered")
	ErrNilParser         = fmt.Errorf("cannot register nil parser")
)

// Factory manages parser registration and retrieval
type Factory struct {
	parsers    map[FileType]Parser
	extensions map[string]FileType
	mu         sync.RWMutex
}

// NewFactory creates a new parser factory with default mappings
func NewFactory() *Factory {
	return &Factory{
		parsers:    make(map[FileType]Parser),
		extensions: make(map[string]FileType),
	}
}

// NewFactoryWithDefaults creates a factory with built-in parsers registered
func NewFactoryWithDefaults() *Factory {
	f := NewFactory()

	// Register default extension mappings
	for ext, fileType := range ExtensionMapping {
		f.extensions[strings.ToLower(ext)] = fileType
	}

	return f
}

// Register registers a parser for a specific file type
func (f *Factory) Register(fileType FileType, parser Parser) error {
	if parser == nil {
		return ErrNilParser
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.parsers[fileType]; exists {
		return fmt.Errorf("%w: parser for type '%s' already exists", ErrParserAlreadyExists, fileType)
	}

	f.parsers[fileType] = parser
	return nil
}

// Get retrieves a parser for a specific file type
func (f *Factory) Get(fileType FileType) (Parser, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	parser, exists := f.parsers[fileType]
	if !exists {
		return nil, fmt.Errorf("%w: no parser registered for type '%s'", ErrParserNotFound, fileType)
	}

	return parser, nil
}

// GetByExtension retrieves a parser based on file extension
func (f *Factory) GetByExtension(filename string) (Parser, error) {
	fileType := f.DetectFileType(filename)
	if fileType == FileTypeUnknown {
		return nil, fmt.Errorf("%w: cannot detect type for file '%s'", ErrUnknownFileType, filename)
	}

	return f.Get(fileType)
}

// DetectFileType detects the file type based on extension
func (f *Factory) DetectFileType(filename string) FileType {
	ext := strings.ToLower(filepath.Ext(filename))

	f.mu.RLock()
	defer f.mu.RUnlock()

	// First check custom registered extensions
	if fileType, exists := f.extensions[ext]; exists {
		return fileType
	}

	// Fall back to global extension mapping
	if fileType, exists := ExtensionMapping[ext]; exists {
		return fileType
	}

	return FileTypeUnknown
}

// RegisterExtension registers a custom extension mapping
func (f *Factory) RegisterExtension(extension string, fileType FileType) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ext := strings.ToLower(extension)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	f.extensions[ext] = fileType
}

// Unregister removes a parser for a specific file type
func (f *Factory) Unregister(fileType FileType) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.parsers, fileType)
}

// IsRegistered checks if a parser is registered for the given file type
func (f *Factory) IsRegistered(fileType FileType) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, exists := f.parsers[fileType]
	return exists
}

// ListRegistered returns a list of registered file types
func (f *Factory) ListRegistered() []FileType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]FileType, 0, len(f.parsers))
	for t := range f.parsers {
		types = append(types, t)
	}
	return types
}

// ParseFile is a convenience method to parse a file using the appropriate parser
func (f *Factory) ParseFile(ctx context.Context, filename string, content string) (*ParseResult, error) {
	parser, err := f.GetByExtension(filename)
	if err != nil {
		return nil, err
	}

	return parser.ParseString(ctx, content)
}
