package mock_test

import (
	"testing"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
)

// TestBackend_Mock_RecordsBeginEndFrame verifies that the mock backend
// records BeginFrame / EndFrame call counts and stores completed frames.
func TestBackend_Mock_RecordsBeginEndFrame(t *testing.T) {
	t.Parallel()

	b := mock.New(80, 24)

	if b.BeginFrameCalls != 0 {
		t.Fatalf("initial BeginFrameCalls = %d, want 0", b.BeginFrameCalls)
	}
	if b.EndFrameCalls != 0 {
		t.Fatalf("initial EndFrameCalls = %d, want 0", b.EndFrameCalls)
	}

	// First frame.
	surface := b.BeginFrame()
	if surface == nil {
		t.Fatal("BeginFrame returned nil surface")
	}
	if b.BeginFrameCalls != 1 {
		t.Errorf("BeginFrameCalls = %d, want 1", b.BeginFrameCalls)
	}

	// Write a cell to confirm the surface is a usable Buffer.
	surface.Set(5, 3, backend.Cell{Content: "X", Width: 1})

	if err := b.EndFrame(); err != nil {
		t.Fatalf("EndFrame returned error: %v", err)
	}
	if b.EndFrameCalls != 1 {
		t.Errorf("EndFrameCalls = %d, want 1", b.EndFrameCalls)
	}
	if len(b.Frames) != 1 {
		t.Fatalf("Frames length = %d, want 1", len(b.Frames))
	}

	// The stored frame's surface must contain the cell we wrote.
	fr := b.LastFrame()
	got := fr.Surface.CellAt(5, 3)
	if got.Content != "X" {
		t.Errorf("stored frame cell at (5,3) = %q, want \"X\"", got.Content)
	}

	// Second frame.
	b.BeginFrame()
	if err := b.EndFrame(); err != nil {
		t.Fatalf("second EndFrame returned error: %v", err)
	}
	if b.BeginFrameCalls != 2 {
		t.Errorf("after 2 frames, BeginFrameCalls = %d, want 2", b.BeginFrameCalls)
	}
	if b.EndFrameCalls != 2 {
		t.Errorf("after 2 frames, EndFrameCalls = %d, want 2", b.EndFrameCalls)
	}
	if len(b.Frames) != 2 {
		t.Errorf("after 2 frames, len(Frames) = %d, want 2", len(b.Frames))
	}
}
