package doing

import (
	"context"

	"github.com/morty/morty/internal/logging"
)

// mockLogger implements logging.Logger interface for testing
type mockLogger struct {
	logs []struct {
		level   string
		message string
	}
}

func (m *mockLogger) Debug(msg string, attrs ...logging.Attr) {}
func (m *mockLogger) Info(msg string, attrs ...logging.Attr) {
	m.logs = append(m.logs, struct {
		level   string
		message string
	}{"INFO", msg})
}
func (m *mockLogger) Warn(msg string, attrs ...logging.Attr) {
	m.logs = append(m.logs, struct {
		level   string
		message string
	}{"WARN", msg})
}
func (m *mockLogger) Error(msg string, attrs ...logging.Attr) {
	m.logs = append(m.logs, struct {
		level   string
		message string
	}{"ERROR", msg})
}
func (m *mockLogger) Fatal(msg string, attrs ...logging.Attr) {}
func (m *mockLogger) Success(msg string, attrs ...logging.Attr) {}
func (m *mockLogger) Loop(msg string, attrs ...logging.Attr) {}
func (m *mockLogger) WithContext(ctx context.Context) logging.Logger { return m }
func (m *mockLogger) WithJob(module, job string) logging.Logger { return m }
func (m *mockLogger) WithAttrs(attrs ...logging.Attr) logging.Logger { return m }
func (m *mockLogger) SetLevel(level logging.Level) {}
func (m *mockLogger) GetLevel() logging.Level { return logging.InfoLevel }
func (m *mockLogger) IsEnabled(level logging.Level) bool { return true }
