package trace

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

const LOG_MOD_KEY = "module"

type Logger interface {
	Debug(mod, msg string, args ...any)
	Info(mod, msg string, args ...any)
	Warn(mod, msg string, args ...any)
	Error(mod, msg string, args ...any)
}

type StdLogger struct {
	slog *slog.Logger
}

func NewStdLogger() Logger {
	return &StdLogger{
		slog: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

func (l *StdLogger) Log(level slog.Level, mod string, msg string, args ...any) {
	a := append([]any{slog.String(LOG_MOD_KEY, mod)}, args...)
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

func (l *StdLogger) Error(mod, msg string, args ...any) {
	l.Log(slog.LevelError, mod, msg, args...)
}

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

func (l *TestLogger) Error(mod, msg string, args ...any) {
	l.test.Errorf("ERROR %s: %s | %v", mod, msg, args)
}
