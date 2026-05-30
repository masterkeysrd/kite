package mock

import (
	"bytes"
	"image/color"
	"io"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/geom"
)

// Compile-time assertion: Backend implements backend.Backend.
var _ backend.Backend = (*Backend)(nil)

// FrameRecord captures the state of one completed frame.
type FrameRecord struct {
	// Surface is the Buffer that was returned by BeginFrame and
	// subsequently drawn into by the paint engine.
	Surface *backend.Buffer
}

// Backend is a recording backend for tests. It implements backend.Backend and
// tracks every BeginFrame / EndFrame call so tests can assert on the sequence
// of frames and the cell content painted into each one.
//
// Backend is not safe for concurrent use.
type Backend struct {
	width, height int

	// events is the channel returned by Events().
	events chan backend.RawEvent

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

	// Output captures all raw sequences written to the backend via Writer().
	Output bytes.Buffer

	current *backend.Buffer
}

// CursorRecord captures one call to the cursor-management methods.
type CursorRecord struct {
	Visible bool
	X, Y    int
	Shape   backend.CursorShape
	Color   color.Color
}

// New creates a recording Backend for a terminal of the given dimensions.
func New(width, height int) *Backend {
	return &Backend{
		width:  width,
		height: height,
		events: make(chan backend.RawEvent),
	}
}

// NewWithCaps creates a recording Backend with the given dimensions and
// terminal capabilities.
func NewWithCaps(width, height int, caps backend.Caps) *Backend {
	return &Backend{
		width:  width,
		height: height,
		caps:   caps,
		events: make(chan backend.RawEvent),
	}
}

// Start records the startup call.
func (b *Backend) Start() error {
	return nil
}

// BeginFrame allocates a fresh Buffer, records the call, and returns the
// buffer as a backend.Surface.
func (b *Backend) BeginFrame() backend.Surface {
	b.BeginFrameCalls++
	b.current = backend.NewBuffer(b.width, b.height)
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
func (b *Backend) Events() <-chan backend.RawEvent { return b.events }

// Restore records the call. In the mock backend this is a no-op.
func (b *Backend) Restore() { b.RestoreCalls++ }

func (b *Backend) Resize(size geom.Size) {
	b.width = size.Width
	b.height = size.Height
}

// Size returns the simulated dimensions.
func (b *Backend) Size() geom.Size { return geom.Size{Width: b.width, Height: b.height} }

// Writer returns the mock output buffer.
func (b *Backend) Writer() io.Writer { return &b.Output }

type mockClipboard struct{}

func (m *mockClipboard) Set(mime string, data []byte) {}
func (m *mockClipboard) Request(mime string)          {}

func (b *Backend) Clipboard() backend.Clipboard {
	return &mockClipboard{}
}

func (b *Backend) ShowCursor(v bool)                    { b.Cursor.Visible = v }
func (b *Backend) SetCursorPos(x, y int)                { b.Cursor.X, b.Cursor.Y = x, y }
func (b *Backend) SetCursorColor(c color.Color)         { b.Cursor.Color = c }
func (b *Backend) SetCursorShape(s backend.CursorShape) { b.Cursor.Shape = s }

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
