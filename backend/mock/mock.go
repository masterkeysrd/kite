package mock

import (
	"image/color"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/cursor"
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

	// events is the channel returned by Events().
	events chan event.RawEvent

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

	// Cursor is the most recent cursor state set via the backend methods.
	Cursor CursorRecord

	current *paint.FrameBuffer
}

// CursorRecord captures one call to the cursor-management methods.
type CursorRecord struct {
	Visible bool
	X, Y    int
	Shape   cursor.Shape
	Color   color.Color
}

// New creates a recording Backend for a terminal of the given dimensions.
func New(width, height int) *Backend {
	return &Backend{
		width:  width,
		height: height,
		events: make(chan event.RawEvent),
	}
}

// NewWithCaps creates a recording Backend with the given dimensions and
// terminal capabilities.
func NewWithCaps(width, height int, caps backend.Caps) *Backend {
	return &Backend{
		width:  width,
		height: height,
		caps:   caps,
		events: make(chan event.RawEvent),
	}
}

// Start records the startup call.
func (b *Backend) Start() error {
	return nil
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

// Events returns a channel of input events.
func (b *Backend) Events() <-chan event.RawEvent { return b.events }

// Restore records the call. In the mock backend this is a no-op.
func (b *Backend) Restore() { b.RestoreCalls++ }

func (b *Backend) Resize(size layout.Size) {
	b.width = size.Width
	b.height = size.Height
}

// Size returns the simulated dimensions.
func (b *Backend) Size() layout.Size { return layout.Size{Width: b.width, Height: b.height} }

func (b *Backend) ShowCursor(v bool)             { b.Cursor.Visible = v }
func (b *Backend) SetCursorPos(x, y int)         { b.Cursor.X, b.Cursor.Y = x, y }
func (b *Backend) SetCursorColor(c color.Color)  { b.Cursor.Color = c }
func (b *Backend) SetCursorShape(s cursor.Shape) { b.Cursor.Shape = s }

// LastFrame returns the most recently completed frame, or a zero FrameRecord
// if no frames have been completed yet.
func (b *Backend) LastFrame() FrameRecord {
	if len(b.Frames) == 0 {
		return FrameRecord{}
	}
	return b.Frames[len(b.Frames)-1]
}

func (b *Backend) DumpState() {
	panic("mock.Backend.DumpState not implemented")
}
