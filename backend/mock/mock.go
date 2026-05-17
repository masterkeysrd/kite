package mock

import (
	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/paint"
)

// Compile-time assertion: Backend implements backend.Backend.
var _ backend.Backend = (*Backend)(nil)

// FrameRecord captures the state of one completed frame.
type FrameRecord struct {
	// Surface is the FrameBuffer that was returned by BeginFrame and
	// subsequently drawn into by the paint engine.
	Surface *paint.FrameBuffer
}

// Backend is a recording backend for tests. It implements backend.Backend and
// tracks every BeginFrame / EndFrame call so tests can assert on the sequence
// of frames and the cell content painted into each one.
//
// Backend is not safe for concurrent use.
type Backend struct {
	width, height int

	// caps holds the simulated terminal capabilities for this backend.
	// Tests may set fields on caps before passing the backend to an engine.
	caps backend.Caps

	// BeginFrameCalls counts how many times BeginFrame was called.
	BeginFrameCalls int
	// EndFrameCalls counts how many times EndFrame was called.
	EndFrameCalls int
	// RestoreCalls counts how many times Restore was called.
	RestoreCalls int
	// Frames records all completed frames (EndFrame called successfully).
	Frames []FrameRecord

	current *paint.FrameBuffer
}

// New creates a recording Backend for a terminal of the given dimensions.
func New(width, height int) *Backend {
	return &Backend{width: width, height: height}
}

// NewWithCaps creates a recording Backend with the given dimensions and
// terminal capabilities.
func NewWithCaps(width, height int, caps backend.Caps) *Backend {
	return &Backend{width: width, height: height, caps: caps}
}

// BeginFrame allocates a fresh FrameBuffer, records the call, and returns the
// buffer as a paint.Surface.
func (b *Backend) BeginFrame() paint.Surface {
	b.BeginFrameCalls++
	b.current = paint.NewFrameBuffer(0, 0, b.width, b.height)
	b.current.BumpVersion()
	return b.current
}

// EndFrame records the completed frame and returns nil. It panics if
// BeginFrame has not been called first.
func (b *Backend) EndFrame() error {
	if b.current == nil {
		panic("mock.Backend.EndFrame called without a preceding BeginFrame")
	}
	b.EndFrameCalls++
	b.Frames = append(b.Frames, FrameRecord{Surface: b.current})
	b.current = nil
	return nil
}

// Caps returns the simulated terminal capabilities.
func (b *Backend) Caps() backend.Caps { return b.caps }

// event returns a channel of input event. In the mock backend this returns
// nil by default; tests may set it if they need to simulate input.
func (b *Backend) Events() <-chan event.RawEvent { return nil }

// Restore records the call. In the mock backend this is a no-op.
func (b *Backend) Restore() { b.RestoreCalls++ }

// Size returns the simulated dimensions.
func (b *Backend) Size() layout.Size { return layout.Size{Width: b.width, Height: b.height} }

// LastFrame returns the most recently completed frame, or a zero FrameRecord
// if no frames have been completed yet.
func (b *Backend) LastFrame() FrameRecord {
	if len(b.Frames) == 0 {
		return FrameRecord{}
	}
	return b.Frames[len(b.Frames)-1]
}
