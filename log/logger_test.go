package log

import (
	"bytes"
	"log/slog"
	"testing"
)

func TestNewNoopLogger(t *testing.T) {
	l := NewNoopLogger()
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
	// Handler should discard
	if l.Handler() != slog.DiscardHandler {
		t.Errorf("expected slog.DiscardHandler, got %v", l.Handler())
	}
}

func TestEnabled(t *testing.T) {
	var buf bytes.Buffer
	orig := Logger()
	defer SetLogger(orig)

	// Set logger with Info level
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	SetLogger(logger)

	if Enabled(slog.LevelDebug) {
		t.Error("expected LevelDebug to be disabled")
	}
	if !Enabled(slog.LevelInfo) {
		t.Error("expected LevelInfo to be enabled")
	}
	if !Enabled(slog.LevelWarn) {
		t.Error("expected LevelWarn to be enabled")
	}

	// Reset to noop logger
	SetLogger(nil)
	if Enabled(slog.LevelInfo) {
		t.Error("expected noop logger to report Enabled as false")
	}
}
