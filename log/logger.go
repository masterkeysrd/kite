package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
)

var globalLogger atomic.Pointer[slog.Logger]

func init() {
	// Start with a safe No-op logger to protect the TUI
	globalLogger.Store(NewNoopLogger())
}

// SetLogger allows the user to override Kite's internal logger.
// If nil is provided, it resets to a No-op logger.
func SetLogger(l *slog.Logger) {
	if l == nil {
		l = NewNoopLogger()
	}
	globalLogger.Store(l)
}

// Logger returns the current global Kite logger.
func Logger() *slog.Logger {
	return globalLogger.Load()
}

// Debug logs at LevelDebug using the global Kite logger.
func Debug(msg string, args ...any) {
	globalLogger.Load().Debug(msg, args...)
}

// Info logs at LevelInfo using the global Kite logger.
func Info(msg string, args ...any) {
	globalLogger.Load().Info(msg, args...)
}

// Warn logs at LevelWarn using the global Kite logger.
func Warn(msg string, args ...any) {
	globalLogger.Load().Warn(msg, args...)
}

// Error logs at LevelError using the global Kite logger.
func Error(msg string, args ...any) {
	globalLogger.Load().Error(msg, args...)
}

// LogAttrs logs using the global Kite logger with specific attributes.
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	globalLogger.Load().LogAttrs(ctx, level, msg, attrs...)
}

// With returns a new Logger that includes the given arguments.
func With(args ...any) *slog.Logger {
	return globalLogger.Load().With(args...)
}

// NewNoopLogger creates a logger that safely discards all output.
// It is ideal as the default logger in a TUI to prevent screen corruption.
func NewNoopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// NewFileLogger creates a JSON logger that writes to the specified file path.
func NewFileLogger(path string, level slog.Level) (*slog.Logger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}
	return slog.New(slog.NewJSONHandler(f, opts)), nil
}
