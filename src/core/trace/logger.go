package trace

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

// LogPkgKey is the key to log the package name under in the structured logger.
const LogPkgKey = "module"

// HLogger is the system's default logger using the log/slog package as the underlying logger.
//
// Message format:
//
//	<level> <module>: <message> | <args>
//
// Example:
//
//	logger.Info("sys.web", "template rendered", "template", "home.html")
//
//	INFO sys.web: template rendered | template=home.html
//
// TODO Add contextualized logger that can be instantiated With() some args upfront. This should be used in web.IO to always log important request information.
// TODO Further nesting of contextualized With() loggers could generally help to improve log traceability.
type HLogger struct {
	slog *slog.Logger
}

// TestLogger is a logger that writes to the test's log.
// TestLogger is intended to be used in tests only.
// Different log levels can be silenced by setting the environment variables
//   - TEST_LOG_SILENCE_DEBUG,
//   - TEST_LOG_SILENCE_INFO,
//   - TEST_LOG_SILENCE_WARN,
//   - TEST_LOG_SILENCE_ERROR.
type TestLogger struct {
	test *testing.T
}

// Logger is the interface for logging. A logger is expected to be safe for concurrent use.
// The underlying logging is not of concern to the caller. Most commonly a structured logger is used through the HLogger,
// a simple abstraction of the log/slog package implementing the trace.Logger interface.
// Each Logger should decide itself where logs are written to e.g. stdout, stderr, file, etc.
//
// For each log method a module name is expected. This is used to identify the module that logged the message.
// The message is a string that describes the log. Further args can be supplied to add more information to the log.
// It is up to the logger to decide how to handle the args. Preferably the args are logged in a structured way.
// Args should also usually be supplied in this order: key1, value1, key2, value2, ...
// As a convenience the Error method takes an errors as the third argument. The error is added to the args as "error".
type Logger interface {
	Debug(mod, msg string, args ...any)            // Debug is a message with no crucial information.
	Info(mod, msg string, args ...any)             // Info is a message with informative content.
	Warn(mod, msg string, args ...any)             // Warn is a message with information about a potential problem.
	Error(mod, msg string, err error, args ...any) // Error informs about a definite problem that occurred.
}

// NewLogger creates a new trace.HLogger using the log/slog package as the underlying logger.
// The logger writes to stdout.
func NewLogger() Logger {
	return &HLogger{
		slog: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
}

// Log logs a message with the given level, module and arguments.
func (l *HLogger) Log(level slog.Level, mod string, msg string, args ...any) {
	a := append([]any{slog.String(LogPkgKey, mod)}, args...)
	l.slog.Log(context.Background(), level, msg, a...)
}

// Debug is a message with no crucial information. It should be used to log information that is import for debugging.
func (l *HLogger) Debug(mod, msg string, args ...any) {
	l.Log(slog.LevelDebug, mod, msg, args...)
}

// Info is a message with informative content. E.g. a template was rendered.
func (l *HLogger) Info(mod, msg string, args ...any) {
	l.Log(slog.LevelInfo, mod, msg, args...)
}

// Warn is a message with information about a potential problem. E.g. a process is taking longer than expected.
func (l *HLogger) Warn(mod, msg string, args ...any) {
	l.Log(slog.LevelWarn, mod, msg, args...)
}

// Error informs about a definite problem that occurred. E.g. a database connection failed.
func (l *HLogger) Error(mod, msg string, err error, args ...any) {
	args = append(args, "error", err)
	l.Log(slog.LevelError, mod, msg, args...)
}

// NewTestLogger creates a new TestLogger that writes to the test's log.
// It is intended to be used in tests only. Different log levels can be silenced by setting the environment variables
func NewTestLogger(t *testing.T) Logger {
	return &TestLogger{
		test: t,
	}
}

// Debug is a message with no crucial information. Debug is silenced if the environment variable TEST_LOG_SILENCE_DEBUG is set.
func (l *TestLogger) Debug(mod, msg string, args ...any) {
	if os.Getenv("TEST_LOG_SILENCE_DEBUG") != "" {
		return
	}

	l.test.Logf("DEBUG %s: %s | %v", mod, msg, args)
}

// Info is a message with informative content. Info is silenced if the environment variable TEST_LOG_SILENCE_INFO is set.
func (l *TestLogger) Info(mod, msg string, args ...any) {
	if os.Getenv("TEST_LOG_SILENCE_INFO") != "" {
		return
	}

	l.test.Logf("INFO %s: %s | %v", mod, msg, args)
}

// Warn is a message with information about a potential problem. Warn is silenced if the environment variable TEST_LOG_SILENCE_WARN is set.
func (l *TestLogger) Warn(mod, msg string, args ...any) {
	if os.Getenv("TEST_LOG_SILENCE_WARN") != "" {
		return
	}

	l.test.Logf("WARN %s: %s | %v", mod, msg, args)
}

// Error informs about a definite problem that occurred. Error is silenced if the environment variable TEST_LOG_SILENCE_ERROR is set.
func (l *TestLogger) Error(mod, msg string, err error, args ...any) {
	if os.Getenv("TEST_LOG_SILENCE_ERROR") != "" {
		return
	}

	args = append(args, "error", err)
	l.test.Errorf("ERROR %s: %s | %v", mod, msg, args)
}
