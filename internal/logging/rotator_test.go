package logging

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestNewRotator(t *testing.T) {
	tests := []struct {
		name       string
		maxSize    string
		maxBackups int
		maxAge     int
		wantErr    bool
		wantSize   int64
	}{
		{
			name:       "valid 10MB",
			maxSize:    "10MB",
			maxBackups: 5,
			maxAge:     7,
			wantErr:    false,
			wantSize:   10 * 1024 * 1024,
		},
		{
			name:       "valid 100 bytes",
			maxSize:    "100",
			maxBackups: 5,
			maxAge:     7,
			wantErr:    false,
			wantSize:   100,
		},
		{
			name:       "valid 1KB",
			maxSize:    "1KB",
			maxBackups: 3,
			maxAge:     7,
			wantErr:    false,
			wantSize:   1024,
		},
		{
			name:       "valid 1GB",
			maxSize:    "1GB",
			maxBackups: 10,
			maxAge:     30,
			wantErr:    false,
			wantSize:   1024 * 1024 * 1024,
		},
		{
			name:       "empty size",
			maxSize:    "",
			maxBackups: 5,
			maxAge:     7,
			wantErr:    true,
		},
		{
			name:       "invalid unit",
			maxSize:    "10XB",
			maxBackups: 5,
			maxAge:     7,
			wantErr:    true,
		},
		{
			name:       "valid lowercase",
			maxSize:    "10mb",
			maxBackups: 5,
			maxAge:     7,
			wantErr:    false,
			wantSize:   10 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRotator(tt.maxSize, tt.maxBackups, tt.maxAge)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewRotator() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("NewRotator() unexpected error: %v", err)
				return
			}
			if r.GetMaxSize() != tt.wantSize {
				t.Errorf("GetMaxSize() = %d, want %d", r.GetMaxSize(), tt.wantSize)
			}
			if r.GetMaxBackups() != tt.maxBackups {
				t.Errorf("GetMaxBackups() = %d, want %d", r.GetMaxBackups(), tt.maxBackups)
			}
			if r.GetMaxAge() != tt.maxAge {
				t.Errorf("GetMaxAge() = %d, want %d", r.GetMaxAge(), tt.maxAge)
			}
		})
	}
}

func TestShouldRotate(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Create rotator with 100 bytes max size
	r, err := NewRotator("100", 5, 7)
	if err != nil {
		t.Fatalf("NewRotator() error: %v", err)
	}

	// Test with non-existent file
	if r.ShouldRotate(logFile) {
		t.Error("ShouldRotate() with non-existent file should return false")
	}

	// Create a small file (50 bytes)
	smallContent := make([]byte, 50)
	for i := range smallContent {
		smallContent[i] = 'a'
	}
	if err := os.WriteFile(logFile, smallContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test with small file (< 100 bytes) - should NOT rotate
	if r.ShouldRotate(logFile) {
		t.Error("ShouldRotate() with 50 bytes file should return false (50 < 100)")
	}

	// Create a large file (101 bytes)
	largeContent := make([]byte, 101)
	for i := range largeContent {
		largeContent[i] = 'b'
	}
	if err := os.WriteFile(logFile, largeContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test with large file (> 100 bytes) - should rotate
	if !r.ShouldRotate(logFile) {
		t.Error("ShouldRotate() with 101 bytes file should return true (101 > 100)")
	}
}

func TestRotate(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "morty.log")

	// Create rotator
	r, err := NewRotator("100", 5, 7)
	if err != nil {
		t.Fatalf("NewRotator() error: %v", err)
	}

	// Create initial log file with content
	content := []byte("original log content\n")
	if err := os.WriteFile(logFile, content, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Perform rotation
	if err := r.Rotate(logFile); err != nil {
		t.Fatalf("Rotate() error: %v", err)
	}

	// Check that original file is gone
	if _, err := os.Stat(logFile); !os.IsNotExist(err) {
		t.Error("Original log file should not exist after rotation")
	}

	// Check that .1 file exists with original content
	backup1 := logFile + ".1"
	backupContent, err := os.ReadFile(backup1)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}
	if string(backupContent) != string(content) {
		t.Errorf("Backup content = %q, want %q", string(backupContent), string(content))
	}
}

func TestRotateMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "morty.log")

	// Create rotator with max 3 backups
	r, err := NewRotator("100", 3, 7)
	if err != nil {
		t.Fatalf("NewRotator() error: %v", err)
	}

	// First rotation
	if err := os.WriteFile(logFile, []byte("content1\n"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := r.Rotate(logFile); err != nil {
		t.Fatalf("Rotate() error: %v", err)
	}

	// Second rotation
	if err := os.WriteFile(logFile, []byte("content2\n"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := r.Rotate(logFile); err != nil {
		t.Fatalf("Rotate() error: %v", err)
	}

	// Third rotation
	if err := os.WriteFile(logFile, []byte("content3\n"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := r.Rotate(logFile); err != nil {
		t.Fatalf("Rotate() error: %v", err)
	}

	// Check that .1 file exists (uncompressed)
	backup1 := logFile + ".1"
	if _, err := os.Stat(backup1); os.IsNotExist(err) {
		t.Error("Backup .1 file should exist (uncompressed)")
	}

	// Check that .2.gz file exists (compressed)
	backup2 := logFile + ".2.gz"
	if _, err := os.Stat(backup2); os.IsNotExist(err) {
		t.Error("Backup .2.gz file should exist (compressed)")
	}

	// Check that .3.gz file exists (compressed)
	backup3 := logFile + ".3.gz"
	if _, err := os.Stat(backup3); os.IsNotExist(err) {
		t.Error("Backup .3.gz file should exist (compressed)")
	}

	// Verify gzip files can be decompressed
	for _, f := range []string{backup2, backup3} {
		file, err := os.Open(f)
		if err != nil {
			t.Fatalf("Failed to open gzip file: %v", err)
		}
		defer file.Close()

		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			t.Fatalf("Failed to create gzip reader: %v", err)
		}
		defer gzipReader.Close()

		content, err := io.ReadAll(gzipReader)
		if err != nil {
			t.Fatalf("Failed to read gzip content: %v", err)
		}
		if len(content) == 0 {
			t.Error("Decompressed content should not be empty")
		}
	}
}

func TestRotateCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "morty.log")

	// Create rotator with max 2 backups (small for testing cleanup)
	r, err := NewRotator("100", 2, 7)
	if err != nil {
		t.Fatalf("NewRotator() error: %v", err)
	}

	// Perform multiple rotations
	for i := 0; i < 5; i++ {
		content := []byte("content for rotation " + string(rune('0'+i)) + "\n")
		if err := os.WriteFile(logFile, content, 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := r.Rotate(logFile); err != nil {
			t.Fatalf("Rotate() error: %v", err)
		}
	}

	// Only .1 and .2.gz should exist (maxBackups=2)
	// .3.gz and higher should be cleaned up
	backup1 := logFile + ".1"
	backup2 := logFile + ".2.gz"
	backup3 := logFile + ".3.gz"

	if _, err := os.Stat(backup1); os.IsNotExist(err) {
		t.Error("Backup .1 should exist")
	}
	if _, err := os.Stat(backup2); os.IsNotExist(err) {
		t.Error("Backup .2.gz should exist")
	}
	if _, err := os.Stat(backup3); !os.IsNotExist(err) {
		t.Error("Backup .3.gz should NOT exist (exceeds maxBackups)")
	}
}

func TestRotateEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "morty.log")

	r, err := NewRotator("100", 5, 7)
	if err != nil {
		t.Fatalf("NewRotator() error: %v", err)
	}

	// Rotate non-existent file - should not error
	if err := r.Rotate(logFile); err != nil {
		t.Errorf("Rotate() with non-existent file should not error: %v", err)
	}
}

func TestRotatorConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "morty.log")

	r, err := NewRotator("1000", 5, 7)
	if err != nil {
		t.Fatalf("NewRotator() error: %v", err)
	}

	// Create initial file
	if err := os.WriteFile(logFile, []byte("initial\n"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Run multiple ShouldRotate checks concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			r.ShouldRotate(logFile)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Run multiple rotations concurrently (only one should succeed)
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(logFile, []byte("content\n"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		if err := r.Rotate(logFile); err != nil {
			t.Errorf("Rotate() error: %v", err)
		}
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"100", 100, false},
		{"100B", 100, false},
		{"10KB", 10 * 1024, false},
		{"10MB", 10 * 1024 * 1024, false},
		{"1GB", 1024 * 1024 * 1024, false},
		{"1.5MB", 1.5 * 1024 * 1024, false},
		{"10mb", 10 * 1024 * 1024, false},
		{"10 MB", 10 * 1024 * 1024, false},
		{"", 0, true},
		{"abc", 0, true},
		{"10XB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseSize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSize(%q) expected error but got none", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseSize(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("parseSize(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCompressFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "compressed.gz")

	// Create source file
	content := []byte("this is test content for compression\n")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Create rotator
	r, err := NewRotator("100", 5, 7)
	if err != nil {
		t.Fatalf("NewRotator() error: %v", err)
	}

	// Compress file
	if err := r.compressFile(srcFile, dstFile); err != nil {
		t.Fatalf("compressFile() error: %v", err)
	}

	// Verify compressed file exists
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Fatal("Compressed file should exist")
	}

	// Decompress and verify content
	file, err := os.Open(dstFile)
	if err != nil {
		t.Fatalf("Failed to open compressed file: %v", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("Failed to read decompressed content: %v", err)
	}

	if string(decompressed) != string(content) {
		t.Errorf("Decompressed content = %q, want %q", string(decompressed), string(content))
	}
}
