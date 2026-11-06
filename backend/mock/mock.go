package mock

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"io"
	"sync"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/geom"
)

// Compile-time assertions.
var (
	_ backend.Backend       = (*Backend)(nil)
	_ backend.FrameInserter = (*Backend)(nil)
	_ io.Writer             = (*OutputBuffer)(nil)
)

// FrameRecord captures the state of one completed frame.
type FrameRecord struct {
	// Surface is the Buffer that was returned by BeginFrame and
	// subsequently drawn into by the paint engine.
	Surface *backend.Buffer
}

// OutputBuffer is a thread-safe bytes.Buffer wrapper used by Backend to record
// terminal output sequences.
type OutputBuffer struct {
	mu  sync.RWMutex
	buf bytes.Buffer
}

// Write appends p to the output buffer in a thread-safe manner.
func (o *OutputBuffer) Write(p []byte) (n int, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.buf.Write(p)
}

// String returns the contents of the buffer as a string.
func (o *OutputBuffer) String() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.buf.String()
}

// Bytes returns a copy of the buffer contents as a byte slice.
func (o *OutputBuffer) Bytes() []byte {
	o.mu.RLock()
	defer o.mu.RUnlock()
	cp := make([]byte, o.buf.Len())
	copy(cp, o.buf.Bytes())
	return cp
}

// Reset clears the buffer contents.
func (o *OutputBuffer) Reset() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.buf.Reset()
}

// Len returns the number of bytes in the buffer.
func (o *OutputBuffer) Len() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.buf.Len()
}

// CursorRecord captures one call to the cursor-management methods.
type CursorRecord struct {
	Visible bool
	X, Y    int
	Shape   backend.CursorShape
	Color   color.Color
}

// Backend is a recording backend for tests. It implements backend.Backend and
// tracks every BeginFrame / EndFrame call so tests can assert on the sequence
// of frames and the cell content painted into each one.
//
// Backend is safe for concurrent use.
type Backend struct {
	mu sync.RWMutex

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
	// ClearCalls counts how many times Clear was called.
	ClearCalls int
	// Frames records all completed frames (EndFrame called successfully).
	Frames []FrameRecord

	// Cursor is the most recent cursor state set via the backend methods.
	Cursor CursorRecord

	// Output captures all raw sequences written to the backend via Writer().
	Output OutputBuffer

	// title holds the simulated terminal window title.
	title string

	// ProgramStatus records the most recently set program status.
	ProgramStatus *backend.ProgramStatus

	// Inline reports whether inline mode is enabled.
	Inline bool
	// InsertedAbove stores buffers inserted above the inline frame.
	InsertedAbove []*backend.Buffer

	current *backend.Buffer

	frameWaiters map[chan FrameRecord]struct{}
}

// New creates a recording Backend for a terminal of the given dimensions.
func New(width, height int) *Backend {
	return &Backend{
		width:  width,
		height: height,
		events: make(chan backend.RawEvent, 256),
	}
}

// NewWithCaps creates a recording Backend with the given dimensions and
// terminal capabilities.
func NewWithCaps(width, height int, caps backend.Caps) *Backend {
	return &Backend{
		width:  width,
		height: height,
		caps:   caps,
		events: make(chan backend.RawEvent, 256),
	}
}

// Start records the startup call.
func (b *Backend) Start() error {
	return nil
}

// BeginFrame allocates a fresh Buffer, records the call, and returns the
// buffer as a backend.Surface.
func (b *Backend) BeginFrame() backend.Surface {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.BeginFrameCalls++
	b.current = backend.NewBuffer(b.width, b.height)
	return b.current
}

// EndFrame records the completed frame and returns nil. It panics if
// BeginFrame has not been called first.
func (b *Backend) EndFrame() error {
	b.mu.Lock()
	if b.current == nil {
		b.mu.Unlock()
		panic("mock.Backend.EndFrame called without a preceding BeginFrame")
	}
	b.EndFrameCalls++
	fr := FrameRecord{Surface: b.current}
	b.Frames = append(b.Frames, fr)
	b.current = nil

	for ch := range b.frameWaiters {
		select {
		case ch <- fr:
		default:
		}
	}
	b.mu.Unlock()
	return nil
}

// Caps returns the simulated terminal capabilities.
func (b *Backend) Caps() backend.Caps {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.caps
}

// SetCaps updates the simulated terminal capabilities.
func (b *Backend) SetCaps(caps backend.Caps) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.caps = caps
}

// Events returns a channel of input events.
func (b *Backend) Events() <-chan backend.RawEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.events == nil {
		b.events = make(chan backend.RawEvent, 256)
	}
	return b.events
}

// PostEvent sends a raw input event to the mock backend's event channel.
// It is safe for concurrent use.
func (b *Backend) PostEvent(ev backend.RawEvent) {
	b.mu.Lock()
	if b.events == nil {
		b.events = make(chan backend.RawEvent, 256)
	}
	ch := b.events
	b.mu.Unlock()

	ch <- ev
}

// PostEventContext sends a raw input event with context cancellation support.
func (b *Backend) PostEventContext(ctx context.Context, ev backend.RawEvent) error {
	b.mu.Lock()
	if b.events == nil {
		b.events = make(chan backend.RawEvent, 256)
	}
	ch := b.events
	b.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- ev:
		return nil
	}
}

// TryPostEvent attempts to send an event without blocking. Returns true if queued.
func (b *Backend) TryPostEvent(ev backend.RawEvent) bool {
	b.mu.Lock()
	if b.events == nil {
		b.events = make(chan backend.RawEvent, 256)
	}
	ch := b.events
	b.mu.Unlock()

	select {
	case ch <- ev:
		return true
	default:
		return false
	}
}

// Restore records the call. In the mock backend this is a no-op.
func (b *Backend) Restore() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.RestoreCalls++
}

// Resize updates the simulated terminal dimensions.
func (b *Backend) Resize(size geom.Size) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.width = size.Width
	b.height = size.Height
}

// Size returns the simulated dimensions.
func (b *Backend) Size() geom.Size {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return geom.Size{Width: b.width, Height: b.height}
}

// Writer returns the mock output buffer.
func (b *Backend) Writer() io.Writer {
	return &b.Output
}

type mockClipboard struct{}

func (m *mockClipboard) Set(mime string, data []byte) {}
func (m *mockClipboard) Request(mime string)          {}

// Clipboard returns the mock clipboard provider.
func (b *Backend) Clipboard() backend.Clipboard {
	return &mockClipboard{}
}

// ShowCursor sets the cursor visibility.
func (b *Backend) ShowCursor(v bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Cursor.Visible = v
}

// SetCursorPos sets the cursor position.
func (b *Backend) SetCursorPos(x, y int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Cursor.X, b.Cursor.Y = x, y
}

// SetCursorColor sets the terminal hardware cursor color.
func (b *Backend) SetCursorColor(c color.Color) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Cursor.Color = c
}

// SetCursorShape sets the cursor visual shape.
func (b *Backend) SetCursorShape(s backend.CursorShape) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Cursor.Shape = s
}

// CursorState returns a copy of the current cursor state.
func (b *Backend) CursorState() CursorRecord {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Cursor
}

// SetTitle sets the simulated window title and writes the OSC 2 sequence to Output.
func (b *Backend) SetTitle(s string) {
	b.mu.Lock()
	b.title = s
	b.mu.Unlock()
	fmt.Fprintf(&b.Output, "\x1b]2;%s\x07", s)
}

// Title returns the current simulated terminal window title.
func (b *Backend) Title() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.title
}

// Bell writes the BEL character to Output.
func (b *Backend) Bell() {
	fmt.Fprint(&b.Output, "\x07")
}

// SetProgressBar updates the terminal native progress bar escape sequence in Output.
func (b *Backend) SetProgressBar(state backend.ProgressBarState, percentage int) {
	fmt.Fprintf(&b.Output, "\x1b]9;4;%d;%d\x07", state, percentage)
}

// SetProgramStatus records and writes OSC 7501 Program Status to Output.
func (b *Backend) SetProgramStatus(status *backend.ProgramStatus) {
	b.mu.Lock()
	b.ProgramStatus = status
	b.mu.Unlock()

	if status == nil {
		fmt.Fprint(&b.Output, ansi.ClearProgramStatus)
		return
	}
	ps := ansi.ProgramStatus{
		State:       ansi.ProgramState(status.State),
		ID:          status.ID,
		App:         status.App,
		Kind:        ansi.ProgramStatusKind(status.Kind),
		Progress:    status.Progress,
		HasProgress: status.HasProgress,
		Title:       status.Title,
		Message:     status.Message,
	}
	fmt.Fprint(&b.Output, ansi.SetProgramStatus(ps))
}

// CurrentProgramStatus returns the most recently set program status.
func (b *Backend) CurrentProgramStatus() *backend.ProgramStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.ProgramStatus
}

// SetInline configures whether inline mode is enabled.
func (b *Backend) SetInline(inline bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Inline = inline
}

// IsInline reports whether inline mode is enabled.
func (b *Backend) IsInline() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Inline
}

// Clear clears the simulated screen and increments ClearCalls.
func (b *Backend) Clear() {
	b.mu.Lock()
	b.ClearCalls++
	b.mu.Unlock()
	fmt.Fprint(&b.Output, "\x1b[2J\x1b[3J\x1b[H")
}

// InsertAbove stores a copy of the given buffer in InsertedAbove.
func (b *Backend) InsertAbove(buf *backend.Buffer) {
	if buf == nil {
		return
	}
	cp := backend.NewBuffer(buf.Width, buf.Height)
	copy(cp.Cells, buf.Cells)

	b.mu.Lock()
	defer b.mu.Unlock()
	b.InsertedAbove = append(b.InsertedAbove, cp)
}

// InsertAboveForFrame implements backend.FrameInserter.
func (b *Backend) InsertAboveForFrame(buf *backend.Buffer) {
	b.InsertAbove(buf)
}

// InsertedAboveCount returns the number of buffers inserted above the inline frame.
func (b *Backend) InsertedAboveCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.InsertedAbove)
}

// AllInsertedAbove returns a copy of the slice of buffers inserted above.
func (b *Backend) AllInsertedAbove() []*backend.Buffer {
	b.mu.RLock()
	defer b.mu.RUnlock()
	res := make([]*backend.Buffer, len(b.InsertedAbove))
	copy(res, b.InsertedAbove)
	return res
}

// LastFrame returns the most recently completed frame, or a zero FrameRecord
// if no frames have been completed yet.
func (b *Backend) LastFrame() FrameRecord {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.Frames) == 0 {
		return FrameRecord{}
	}
	return b.Frames[len(b.Frames)-1]
}

// FrameCount returns the total number of completed frames recorded.
func (b *Backend) FrameCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.Frames)
}

// AllFrames returns a shallow copy of all recorded frames.
func (b *Backend) AllFrames() []FrameRecord {
	b.mu.RLock()
	defer b.mu.RUnlock()
	res := make([]FrameRecord, len(b.Frames))
	copy(res, b.Frames)
	return res
}

// Frame returns the recorded frame at index i, or false if out of range.
func (b *Backend) Frame(i int) (FrameRecord, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if i < 0 || i >= len(b.Frames) {
		return FrameRecord{}, false
	}
	return b.Frames[i], true
}

// WaitForNextFrame blocks until the next frame is committed or ctx is cancelled.
func (b *Backend) WaitForNextFrame(ctx context.Context) (FrameRecord, error) {
	ch := make(chan FrameRecord, 1)

	b.mu.Lock()
	if b.frameWaiters == nil {
		b.frameWaiters = make(map[chan FrameRecord]struct{})
	}
	b.frameWaiters[ch] = struct{}{}
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		delete(b.frameWaiters, ch)
		b.mu.Unlock()
	}()

	select {
	case <-ctx.Done():
		return FrameRecord{}, ctx.Err()
	case fr := <-ch:
		return fr, nil
	}
}

// WaitForFrame blocks until a frame is completed or the timeout expires.
func (b *Backend) WaitForFrame(timeout time.Duration) (FrameRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return b.WaitForNextFrame(ctx)
}

// Counts returns snapshot of call counters.
func (b *Backend) Counts() (begin, end, restore, clear int) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.BeginFrameCalls, b.EndFrameCalls, b.RestoreCalls, b.ClearCalls
}

// DumpState writes a debug dump of the backend state to a file.
func (b *Backend) DumpState() {
	panic("mock.Backend.DumpState not implemented")
}
