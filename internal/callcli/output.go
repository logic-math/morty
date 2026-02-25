package callcli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
)

// OutputHandler manages command output based on the configured mode.
type OutputHandler struct {
	config     OutputConfig
	stdoutBuf  *bytes.Buffer
	stderrBuf  *bytes.Buffer
	stdoutFile *os.File
	stderrFile *os.File
	mu         sync.RWMutex
}

// NewOutputHandler creates a new output handler based on the configuration.
func NewOutputHandler(config OutputConfig) (*OutputHandler, error) {
	handler := &OutputHandler{
		config: config,
	}

	switch config.Mode {
	case OutputCapture:
		handler.stdoutBuf = &bytes.Buffer{}
		handler.stderrBuf = &bytes.Buffer{}

	case OutputStream:
		// No buffers needed, output goes directly to stdout/stderr

	case OutputCaptureAndStream:
		handler.stdoutBuf = &bytes.Buffer{}
		handler.stderrBuf = &bytes.Buffer{}

	case OutputSilent:
		// No buffers needed, output is discarded
	}

	// Handle output file if specified
	if config.OutputFile != "" {
		file, err := os.Create(config.OutputFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		handler.stdoutFile = file
		handler.stderrFile = file
	}

	return handler, nil
}

// StdoutWriter returns the appropriate writer for stdout.
func (h *OutputHandler) StdoutWriter() io.Writer {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Custom stdout takes precedence
	if h.config.CustomStdout != nil {
		writers := []io.Writer{h.config.CustomStdout}
		if h.stdoutFile != nil {
			writers = append(writers, h.stdoutFile)
		}
		if h.config.Mode == OutputCaptureAndStream || h.config.Mode == OutputCapture {
			writers = append(writers, h.getCaptureWriter(h.stdoutBuf))
		}
		if len(writers) == 1 {
			return writers[0]
		}
		return io.MultiWriter(writers...)
	}

	switch h.config.Mode {
	case OutputCapture:
		writers := []io.Writer{h.getCaptureWriter(h.stdoutBuf)}
		if h.stdoutFile != nil {
			writers = append(writers, h.stdoutFile)
		}
		if len(writers) == 1 {
			return writers[0]
		}
		return io.MultiWriter(writers...)

	case OutputStream:
		writers := []io.Writer{os.Stdout}
		if h.stdoutFile != nil {
			writers = append(writers, h.stdoutFile)
		}
		if len(writers) == 1 {
			return writers[0]
		}
		return io.MultiWriter(writers...)

	case OutputCaptureAndStream:
		writers := []io.Writer{os.Stdout}
		if h.stdoutFile != nil {
			writers = append(writers, h.stdoutFile)
		}
		writers = append(writers, h.getCaptureWriter(h.stdoutBuf))
		return io.MultiWriter(writers...)

	case OutputSilent:
		if h.stdoutFile != nil {
			return h.stdoutFile
		}
		return io.Discard

	default:
		writers := []io.Writer{h.getCaptureWriter(h.stdoutBuf)}
		if h.stdoutFile != nil {
			writers = append(writers, h.stdoutFile)
		}
		if len(writers) == 1 {
			return writers[0]
		}
		return io.MultiWriter(writers...)
	}
}

// StderrWriter returns the appropriate writer for stderr.
func (h *OutputHandler) StderrWriter() io.Writer {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Custom stderr takes precedence
	if h.config.CustomStderr != nil {
		writers := []io.Writer{h.config.CustomStderr}
		if h.stderrFile != nil {
			writers = append(writers, h.stderrFile)
		}
		if h.config.Mode == OutputCaptureAndStream || h.config.Mode == OutputCapture {
			writers = append(writers, h.getCaptureWriter(h.stderrBuf))
		}
		if len(writers) == 1 {
			return writers[0]
		}
		return io.MultiWriter(writers...)
	}

	switch h.config.Mode {
	case OutputCapture:
		writers := []io.Writer{h.getCaptureWriter(h.stderrBuf)}
		if h.stderrFile != nil {
			writers = append(writers, h.stderrFile)
		}
		if len(writers) == 1 {
			return writers[0]
		}
		return io.MultiWriter(writers...)

	case OutputStream:
		writers := []io.Writer{os.Stderr}
		if h.stderrFile != nil {
			writers = append(writers, h.stderrFile)
		}
		if len(writers) == 1 {
			return writers[0]
		}
		return io.MultiWriter(writers...)

	case OutputCaptureAndStream:
		writers := []io.Writer{os.Stderr}
		if h.stderrFile != nil {
			writers = append(writers, h.stderrFile)
		}
		writers = append(writers, h.getCaptureWriter(h.stderrBuf))
		return io.MultiWriter(writers...)

	case OutputSilent:
		if h.stderrFile != nil {
			return h.stderrFile
		}
		return io.Discard

	default:
		writers := []io.Writer{h.getCaptureWriter(h.stderrBuf)}
		if h.stderrFile != nil {
			writers = append(writers, h.stderrFile)
		}
		if len(writers) == 1 {
			return writers[0]
		}
		return io.MultiWriter(writers...)
	}
}

// getCaptureWriter returns a writer that respects MaxCaptureSize.
func (h *OutputHandler) getCaptureWriter(buf *bytes.Buffer) io.Writer {
	if h.config.MaxCaptureSize > 0 {
		return &limitedWriter{
			buf:     buf,
			maxSize: h.config.MaxCaptureSize,
		}
	}
	return buf
}

// GetStdout returns the captured stdout (if any).
func (h *OutputHandler) GetStdout() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.stdoutBuf != nil {
		return h.stdoutBuf.String()
	}
	return ""
}

// GetStderr returns the captured stderr (if any).
func (h *OutputHandler) GetStderr() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.stderrBuf != nil {
		return h.stderrBuf.String()
	}
	return ""
}

// Close closes any open files.
func (h *OutputHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var err error
	if h.stdoutFile != nil {
		// Sync before closing to ensure all data is written
		_ = h.stdoutFile.Sync()
		err = h.stdoutFile.Close()
		h.stdoutFile = nil
		h.stderrFile = nil
	}
	return err
}

// limitedWriter wraps a buffer with a size limit.
type limitedWriter struct {
	buf     *bytes.Buffer
	maxSize int64
	mu      sync.Mutex
}

// Write writes data to the buffer, respecting the size limit.
func (w *limitedWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	currentSize := int64(w.buf.Len())
	remaining := w.maxSize - currentSize

	if remaining <= 0 {
		// Already at limit, return all bytes written without error
		return len(p), nil
	}

	if int64(len(p)) <= remaining {
		return w.buf.Write(p)
	}

	// Truncate to fit within limit, but report all bytes as written
	// to avoid short write errors
	_, _ = w.buf.Write(p[:remaining])
	return len(p), nil
}

// OutputModeString returns a string representation of the output mode.
func OutputModeString(mode OutputMode) string {
	switch mode {
	case OutputCapture:
		return "capture"
	case OutputStream:
		return "stream"
	case OutputCaptureAndStream:
		return "capture_and_stream"
	case OutputSilent:
		return "silent"
	default:
		return "unknown"
	}
}
