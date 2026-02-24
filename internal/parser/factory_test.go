package parser

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

// mockParser is a test implementation of the Parser interface
type mockParser struct {
	fileType FileType
	parseFn  func(ctx context.Context, r io.Reader) (*ParseResult, error)
}

func (m *mockParser) Parse(ctx context.Context, r io.Reader) (*ParseResult, error) {
	if m.parseFn != nil {
		return m.parseFn(ctx, r)
	}
	return &ParseResult{Type: m.fileType, Content: nil}, nil
}

func (m *mockParser) ParseString(ctx context.Context, content string) (*ParseResult, error) {
	return m.Parse(ctx, strings.NewReader(content))
}

func (m *mockParser) Supports(fileType FileType) bool {
	return m.fileType == fileType
}

func (m *mockParser) FileType() FileType {
	return m.fileType
}

// TestNewFactory tests factory creation
func TestNewFactory(t *testing.T) {
	f := NewFactory()
	if f == nil {
		t.Fatal("NewFactory() returned nil")
	}
	if f.parsers == nil {
		t.Error("parsers map not initialized")
	}
	if f.extensions == nil {
		t.Error("extensions map not initialized")
	}
}

func TestNewFactoryWithDefaults(t *testing.T) {
	f := NewFactoryWithDefaults()
	if f == nil {
		t.Fatal("NewFactoryWithDefaults() returned nil")
	}

	// Check that default extensions are registered
	if len(f.extensions) == 0 {
		t.Error("default extensions not registered")
	}

	// Verify specific extension mappings exist
	testCases := []struct {
		ext      string
		wantType FileType
	}{
		{".md", FileTypeMarkdown},
		{".markdown", FileTypeMarkdown},
		{".json", FileTypeJSON},
		{".yaml", FileTypeYAML},
		{".yml", FileTypeYAML},
	}

	for _, tc := range testCases {
		t.Run(tc.ext, func(t *testing.T) {
			got := f.extensions[tc.ext]
			if got != tc.wantType {
				t.Errorf("extension %s: got %v, want %v", tc.ext, got, tc.wantType)
			}
		})
	}
}

// TestFactory_Register tests parser registration
func TestFactory_Register(t *testing.T) {
	f := NewFactory()
	parser := &mockParser{fileType: FileTypeJSON}

	// Test successful registration
	t.Run("register new parser", func(t *testing.T) {
		err := f.Register(FileTypeJSON, parser)
		if err != nil {
			t.Errorf("Register() error = %v", err)
		}
		if !f.IsRegistered(FileTypeJSON) {
			t.Error("parser not registered")
		}
	})

	// Test duplicate registration
	t.Run("duplicate registration", func(t *testing.T) {
		err := f.Register(FileTypeJSON, parser)
		if !errors.Is(err, ErrParserAlreadyExists) {
			t.Errorf("expected ErrParserAlreadyExists, got %v", err)
		}
	})

	// Test nil parser registration
	t.Run("nil parser", func(t *testing.T) {
		err := f.Register(FileTypeYAML, nil)
		if !errors.Is(err, ErrNilParser) {
			t.Errorf("expected ErrNilParser, got %v", err)
		}
	})
}

// TestFactory_Get tests parser retrieval
func TestFactory_Get(t *testing.T) {
	f := NewFactory()
	parser := &mockParser{fileType: FileTypeJSON}
	f.Register(FileTypeJSON, parser)

	t.Run("get existing parser", func(t *testing.T) {
		got, err := f.Get(FileTypeJSON)
		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if got != parser {
			t.Error("Get() returned wrong parser")
		}
	})

	t.Run("get non-existent parser", func(t *testing.T) {
		_, err := f.Get(FileTypeYAML)
		if !errors.Is(err, ErrParserNotFound) {
			t.Errorf("expected ErrParserNotFound, got %v", err)
		}
	})
}

// TestFactory_DetectFileType tests file type detection
func TestFactory_DetectFileType(t *testing.T) {
	f := NewFactoryWithDefaults()

	testCases := []struct {
		filename string
		want     FileType
	}{
		{"test.md", FileTypeMarkdown},
		{"test.markdown", FileTypeMarkdown},
		{"test.MD", FileTypeMarkdown},
		{"test.json", FileTypeJSON},
		{"test.JSON", FileTypeJSON},
		{"test.yaml", FileTypeYAML},
		{"test.yml", FileTypeYAML},
		{"test.txt", FileTypeUnknown},
		{"test", FileTypeUnknown},
		{"", FileTypeUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			got := f.DetectFileType(tc.filename)
			if got != tc.want {
				t.Errorf("DetectFileType(%q) = %v, want %v", tc.filename, got, tc.want)
			}
		})
	}
}

// TestFactory_GetByExtension tests parser retrieval by extension
func TestFactory_GetByExtension(t *testing.T) {
	f := NewFactoryWithDefaults()
	parser := &mockParser{fileType: FileTypeJSON}
	f.Register(FileTypeJSON, parser)

	t.Run("get by known extension", func(t *testing.T) {
		got, err := f.GetByExtension("test.json")
		if err != nil {
			t.Errorf("GetByExtension() error = %v", err)
		}
		if got != parser {
			t.Error("GetByExtension() returned wrong parser")
		}
	})

	t.Run("get by unknown extension", func(t *testing.T) {
		_, err := f.GetByExtension("test.unknown")
		if !errors.Is(err, ErrUnknownFileType) {
			t.Errorf("expected ErrUnknownFileType, got %v", err)
		}
	})

	t.Run("get by unregistered type extension", func(t *testing.T) {
		// .md is known but no parser registered
		_, err := f.GetByExtension("test.md")
		if !errors.Is(err, ErrParserNotFound) {
			t.Errorf("expected ErrParserNotFound, got %v", err)
		}
	})
}

// TestFactory_RegisterExtension tests custom extension registration
func TestFactory_RegisterExtension(t *testing.T) {
	f := NewFactory()

	t.Run("register with dot", func(t *testing.T) {
		f.RegisterExtension(".custom", FileTypeJSON)
		if f.DetectFileType("test.custom") != FileTypeJSON {
			t.Error("extension with dot not registered correctly")
		}
	})

	t.Run("register without dot", func(t *testing.T) {
		f.RegisterExtension("nocustom", FileTypeYAML)
		if f.DetectFileType("test.nocustom") != FileTypeYAML {
			t.Error("extension without dot not registered correctly")
		}
	})

	t.Run("register uppercase", func(t *testing.T) {
		f.RegisterExtension(".UPPER", FileTypeMarkdown)
		if f.DetectFileType("test.UPPER") != FileTypeMarkdown {
			t.Error("uppercase extension not handled correctly")
		}
		if f.DetectFileType("test.upper") != FileTypeMarkdown {
			t.Error("lowercase lookup of uppercase registration failed")
		}
	})
}

// TestFactory_Unregister tests parser unregistration
func TestFactory_Unregister(t *testing.T) {
	f := NewFactory()
	parser := &mockParser{fileType: FileTypeJSON}
	f.Register(FileTypeJSON, parser)

	t.Run("unregister existing", func(t *testing.T) {
		f.Unregister(FileTypeJSON)
		if f.IsRegistered(FileTypeJSON) {
			t.Error("parser still registered after Unregister")
		}
	})

	t.Run("unregister non-existent", func(t *testing.T) {
		// Should not panic
		f.Unregister(FileTypeYAML)
	})
}

// TestFactory_IsRegistered tests registration check
func TestFactory_IsRegistered(t *testing.T) {
	f := NewFactory()

	if f.IsRegistered(FileTypeJSON) {
		t.Error("IsRegistered() returned true for unregistered type")
	}

	f.Register(FileTypeJSON, &mockParser{fileType: FileTypeJSON})

	if !f.IsRegistered(FileTypeJSON) {
		t.Error("IsRegistered() returned false for registered type")
	}
}

// TestFactory_ListRegistered tests listing registered parsers
func TestFactory_ListRegistered(t *testing.T) {
	f := NewFactory()

	// Empty list
	if len(f.ListRegistered()) != 0 {
		t.Error("ListRegistered() should return empty for new factory")
	}

	// Add parsers
	f.Register(FileTypeJSON, &mockParser{fileType: FileTypeJSON})
	f.Register(FileTypeYAML, &mockParser{fileType: FileTypeYAML})

	list := f.ListRegistered()
	if len(list) != 2 {
		t.Errorf("ListRegistered() returned %d items, want 2", len(list))
	}

	// Verify both types are in list
	hasJSON, hasYAML := false, false
	for _, ft := range list {
		if ft == FileTypeJSON {
			hasJSON = true
		}
		if ft == FileTypeYAML {
			hasYAML = true
		}
	}
	if !hasJSON {
		t.Error("ListRegistered() missing FileTypeJSON")
	}
	if !hasYAML {
		t.Error("ListRegistered() missing FileTypeYAML")
	}
}

// TestFactory_ParseFile tests the convenience parse method
func TestFactory_ParseFile(t *testing.T) {
	f := NewFactoryWithDefaults()
	parser := &mockParser{
		fileType: FileTypeJSON,
		parseFn: func(ctx context.Context, r io.Reader) (*ParseResult, error) {
			return &ParseResult{
				Type:    FileTypeJSON,
				Content: "parsed content",
			}, nil
		},
	}
	f.Register(FileTypeJSON, parser)

	t.Run("parse known file type", func(t *testing.T) {
		result, err := f.ParseFile(context.Background(), "test.json", `{"key": "value"}`)
		if err != nil {
			t.Errorf("ParseFile() error = %v", err)
		}
		if result.Type != FileTypeJSON {
			t.Errorf("ParseFile() result.Type = %v, want %v", result.Type, FileTypeJSON)
		}
	})

	t.Run("parse unknown file type", func(t *testing.T) {
		_, err := f.ParseFile(context.Background(), "test.unknown", "content")
		if !errors.Is(err, ErrUnknownFileType) {
			t.Errorf("expected ErrUnknownFileType, got %v", err)
		}
	})
}

// TestErrors tests error types
func TestErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrUnknownFileType", ErrUnknownFileType},
		{"ErrParserNotFound", ErrParserNotFound},
		{"ErrParserAlreadyExists", ErrParserAlreadyExists},
		{"ErrNilParser", ErrNilParser},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Error("error variable is nil")
			}
		})
	}
}

// TestExtensionMapping tests the global extension mapping
func TestExtensionMapping(t *testing.T) {
	expectedMappings := map[string]FileType{
		".md":       FileTypeMarkdown,
		".markdown": FileTypeMarkdown,
		".json":     FileTypeJSON,
		".yaml":     FileTypeYAML,
		".yml":      FileTypeYAML,
	}

	for ext, expectedType := range expectedMappings {
		if got := ExtensionMapping[ext]; got != expectedType {
			t.Errorf("ExtensionMapping[%q] = %v, want %v", ext, got, expectedType)
		}
	}
}

// TestFileTypeConstants tests file type constants
func TestFileTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		ft       FileType
		expected string
	}{
		{"FileTypeMarkdown", FileTypeMarkdown, "markdown"},
		{"FileTypeJSON", FileTypeJSON, "json"},
		{"FileTypeYAML", FileTypeYAML, "yaml"},
		{"FileTypeUnknown", FileTypeUnknown, "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.ft) != tc.expected {
				t.Errorf("%s = %q, want %q", tc.name, tc.ft, tc.expected)
			}
		})
	}
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	f := NewFactory()

	// Register initial parser
	f.Register(FileTypeJSON, &mockParser{fileType: FileTypeJSON})

	// Concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				f.Get(FileTypeJSON)
				f.IsRegistered(FileTypeJSON)
				f.DetectFileType("test.json")
				f.ListRegistered()
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				ft := FileType(string(FileTypeYAML) + string(rune(n)))
				f.Register(ft, &mockParser{fileType: ft})
				f.RegisterExtension(".ext"+string(rune(n)), ft)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we got here without panic or data race, thread safety is working
}
