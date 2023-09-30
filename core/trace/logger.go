package trace

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

const LogPkgKey = "module"

type Logger interface {
	Debug(mod, msg string, args ...any)
	Info(mod, msg string, args ...any)
	Warn(mod, msg string, args ...any)
	Error(mod, msg string, err error, args ...any)
}

// StdLogger is the system's default logger.
type StdLogger struct {
	// slog is the underlying structured logger used by the StdLogger.
	slog *slog.Logger
}

// NewLogger creates a new standard logger that writes to stdout.
func NewLogger() Logger {
	return &StdLogger{
		slog: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

func (l *StdLogger) Log(level slog.Level, mod string, msg string, args ...any) {
	a := append([]any{slog.String(LogPkgKey, mod)}, args...)
	l.slog.Log(context.Background(), level, msg, a...)
}

func (l *StdLogger) Debug(mod, msg string, args ...any) {
	l.Log(slog.LevelDebug, mod, msg, args...)
}

func (l *StdLogger) Info(mod, msg string, args ...any) {
	l.Log(slog.LevelInfo, mod, msg, args...)
}

func (l *StdLogger) Warn(mod, msg string, args ...any) {
	l.Log(slog.LevelWarn, mod, msg, args...)
}

func (l *StdLogger) Error(mod, msg string, err error, args ...any) {
	args = append(args, "error", err)
	l.Log(slog.LevelError, mod, msg, args...)
}

// TestLogger is a logger that writes to the test's log.
type TestLogger struct {
	test *testing.T
}

func NewTestLogger(t *testing.T) Logger {
	return &TestLogger{
		test: t,
	}
}

func (l *TestLogger) Debug(mod, msg string, args ...any) {
	l.test.Logf("DEBUG %s: %s | %v", mod, msg, args)
}

func (l *TestLogger) Info(mod, msg string, args ...any) {
	l.test.Logf("INFO %s: %s | %v", mod, msg, args)
}

func (l *TestLogger) Warn(mod, msg string, args ...any) {
	l.test.Logf("WARN %s: %s | %v", mod, msg, args)
}

func (l *TestLogger) Error(mod, msg string, err error, args ...any) {
	args = append(args, "error", err)
	l.test.Errorf("ERROR %s: %s | %v", mod, msg, args)
}
