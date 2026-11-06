package mock_test

import (
	"context"
	"fmt"
	"image/color"
	"sync"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/key"
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

// TestBackend_Mock_ThreadSafety verifies that concurrent access to Backend
// does not trigger race conditions or panic.
func TestBackend_Mock_ThreadSafety(t *testing.T) {
	t.Parallel()

	b := mock.New(80, 24)
	const workers = 8
	const iterations = 100

	var wg sync.WaitGroup

	// Goroutine 1: Frame loop (renderer)
	wg.Go(func() {
		for i := range iterations {
			s := b.BeginFrame()
			s.Set(i%80, 0, backend.Cell{Content: "A", Width: 1})
			_ = b.EndFrame()
		}
	})

	// Goroutine 2: Event poster
	wg.Go(func() {
		for i := range iterations {
			b.PostEvent(&backend.RawKeyEvent{
				Key: key.Key{Code: rune('a' + (i % 26))},
			})
		}
	})

	// Goroutine 3: Event consumer (simulates engine select)
	wg.Go(func() {
		events := b.Events()
		count := 0
		for count < iterations {
			select {
			case <-events:
				count++
			case <-time.After(2 * time.Second):
				return
			}
		}
	})

	// Goroutine 4: Frame reader / assertion checker
	wg.Go(func() {
		for i := range iterations {
			_ = b.LastFrame()
			_ = b.FrameCount()
			_ = b.AllFrames()
			_, _ = b.Frame(i)
		}
	})

	// Goroutine 5: Cursor updater and reader
	wg.Go(func() {
		for i := range iterations {
			b.ShowCursor(i%2 == 0)
			b.SetCursorPos(i%80, (i*2)%24)
			b.SetCursorShape(backend.CursorBar)
			b.SetCursorColor(color.RGBA{R: 255, A: 255})
			_ = b.CursorState()
		}
	})

	// Goroutine 6: Output / terminal writes & reads
	wg.Go(func() {
		for i := range iterations {
			b.SetTitle(fmt.Sprintf("Title %d", i))
			_ = b.Title()
			b.Bell()
			b.Clear()
			_ = b.Output.String()
			_ = b.Output.Bytes()
		}
	})

	// Goroutine 7: InsertAbove calls & readers
	wg.Go(func() {
		for range iterations {
			buf := backend.NewBuffer(20, 1)
			buf.Set(0, 0, backend.Cell{Content: "S"})
			b.InsertAbove(buf)
			_ = b.InsertedAboveCount()
			_ = b.AllInsertedAbove()
		}
	})

	// Goroutine 8: Resizes and status updates
	wg.Go(func() {
		for i := range iterations {
			b.Resize(geom.Size{Width: 80 + (i % 20), Height: 24 + (i % 10)})
			_ = b.Size()
			b.SetProgramStatus(&backend.ProgramStatus{State: backend.ProgramStateWorking})
			_ = b.CurrentProgramStatus()
			b.SetInline(i%2 == 0)
			_ = b.IsInline()
		}
	})

	wg.Wait()

	if b.FrameCount() != iterations {
		t.Errorf("FrameCount = %d, want %d", b.FrameCount(), iterations)
	}
}

// TestBackend_Mock_WaitForFrame verifies notification on frame completion.
func TestBackend_Mock_WaitForFrame(t *testing.T) {
	t.Parallel()

	b := mock.New(80, 24)

	// Waiting when a frame arrives
	done := make(chan struct{})
	go func() {
		fr, err := b.WaitForFrame(1 * time.Second)
		if err != nil {
			t.Errorf("WaitForFrame returned error: %v", err)
		}
		if fr.Surface == nil {
			t.Errorf("WaitForFrame returned nil surface")
		}
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	b.BeginFrame()
	_ = b.EndFrame()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for frame notification")
	}

	// Timeout test
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := b.WaitForNextFrame(ctx)
	if err == nil {
		t.Error("expected error on timeout, got nil")
	}
}

// TestBackend_Mock_OutputBuffer verifies the thread-safe OutputBuffer methods.
func TestBackend_Mock_OutputBuffer(t *testing.T) {
	t.Parallel()

	var buf mock.OutputBuffer

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, _ = fmt.Fprintf(&buf, "msg%d\n", id)
			_ = buf.String()
			_ = buf.Bytes()
			_ = buf.Len()
		}(i)
	}
	wg.Wait()

	if buf.Len() == 0 {
		t.Fatal("expected non-empty buffer")
	}

	buf.Reset()
	if buf.Len() != 0 {
		t.Fatalf("expected 0 length after reset, got %d", buf.Len())
	}
}

// TestBackend_Mock_TryPostEvent verifies non-blocking event posting.
func TestBackend_Mock_TryPostEvent(t *testing.T) {
	t.Parallel()

	b := mock.New(80, 24)

	ev := &backend.RawKeyEvent{Key: key.Key{Code: 'x'}}
	if !b.TryPostEvent(ev) {
		t.Fatal("expected TryPostEvent to succeed on buffered channel")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := b.PostEventContext(ctx, ev); err != nil {
		t.Fatalf("PostEventContext failed: %v", err)
	}
}
