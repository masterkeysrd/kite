package backend

import (
	"image/color"
	"io"

	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/paint"
)

// Backend is the interface that decouples the paint engine from the terminal
// output target. Implementations vend a Surface for the PaintEngine to draw
// onto and commit the result to the screen.
//
// A Backend must be used from a single goroutine except where individual
// method docs note otherwise. The typical frame loop is:
//
//	surface := backend.BeginFrame()
//	engine.Paint(root, surface)
//	if err := backend.EndFrame(); err != nil { … }
type Backend interface {
	// Start initializes the terminal (e.g., enters alt-screen, enables raw mode).
	// It must be called by the engine before any other methods.
	Start() error

	// BeginFrame prepares a new frame and returns the Surface the paint
	// engine should draw into. BeginFrame must be called before EndFrame.
	BeginFrame() paint.Surface

	// EndFrame commits the current frame to the output target. It returns
	// an error if the commit fails (e.g., a write error to the terminal).
	EndFrame() error

	// Caps returns the terminal capabilities detected at startup. The
	// returned value is immutable after the backend is started.
	Caps() Caps

	// Events returns a read-only channel of input events from the terminal. The
	// channel is closed when the backend stops.
	Events() <-chan event.RawEvent

	// Restore unconditionally restores the terminal to its state before the
	// backend was started (exit alt-screen, disable raw mode, show cursor).
	// Restore is safe to call from a signal handler or a deferred panic
	// recovery; it must not block.
	Restore()

	Resize(layout.Size)

	// Size returns the current dimensions of the backend's output area.
	Size() layout.Size

	// Writer returns the terminal's output writer. Used by terminal extensions
	// to send initialization or protocol-specific sequences.
	Writer() io.Writer

	// ShowCursor sets the cursor visibility.
	ShowCursor(bool)

	// SetCursorPos sets the cursor position.
	SetCursorPos(x, y int)

	// SetCursorShape sets the cursor visual shape.
	SetCursorShape(cursor.Shape)

	// SetCursorColor sets the terminal hardware cursor color.
	SetCursorColor(color.Color)

	// GetClipboard returns the current clipboard text content.
	GetClipboard() string
	// SetClipboard stores text into the system clipboard.
	SetClipboard(text string)
	// RequestClipboard asks the terminal to send the current system clipboard
	// content. The response is delivered asynchronously as an event.
	RequestClipboard()

	DumpState()
}

// TerminalExtension is implemented by terminal-specific protocol handlers
// (e.g. Kitty advanced paste, graphics protocols).
type TerminalExtension interface {
	// Init is called once at backend startup. It should write any necessary
	// initialization sequences (e.g. enabling DEC modes) to out.
	Init(out io.Writer)

	// HandleEvent intercepts a raw backend event. If the extension recognizes
	// the event, it returns handled=true and an optional structured event
	// to be dispatched.
	HandleEvent(raw event.RawEvent) (handled bool, ev event.Event)
}

// MouseSupport describes the level of mouse event support available from the
// terminal.
type MouseSupport uint8

const (
	// MouseSupportNone means the terminal does not support mouse events.
	MouseSupportNone MouseSupport = iota

	// MouseSupportClick means the terminal supports button-press events only.
	MouseSupportClick

	// MouseSupportDrag means the terminal supports drag events (motion while
	// button held) in addition to click events.
	MouseSupportDrag

	// MouseSupportTrack means the terminal supports full motion tracking
	// (mouse-move events even without a button pressed).
	MouseSupportTrack
)

// Caps is a snapshot of the terminal capabilities detected at startup by the
// backend. Consumers (paint engine, widget implementations) should treat Caps
// as read-only after the engine starts.
type Caps struct {
	// TrueColor reports whether the terminal supports 24-bit RGB color. When
	// false, the paint engine down-samples to the highest supported palette.
	TrueColor bool

	// OSC8Hyperlinks reports whether the terminal understands OSC 8 hyperlink
	// sequences. Anchor widgets suppress link rendering when this is false.
	OSC8Hyperlinks bool

	// Mouse is the highest mouse-tracking mode available. The engine never
	// enables a mode higher than MouseMode config allows; Caps.Mouse reflects
	// what the terminal can do, not what is currently enabled.
	Mouse MouseSupport

	// BracketedPaste reports whether the terminal supports bracketed-paste
	// mode (surrounding pasted text with escape markers).
	BracketedPaste bool

	// Sixel reports whether the terminal can render Sixel pixel graphics.
	Sixel bool

	// KittyGraphics reports whether the terminal supports the Kitty terminal
	// graphics protocol.
	KittyGraphics bool

	// Title reports whether the terminal accepts window-title escape sequences
	// (OSC 0 / OSC 2). engine.SetTitle is a no-op when this is false.
	Title bool

	// Bell reports whether the terminal should respond to the BEL character.
	// engine.Bell is a no-op when this is false.
	Bell bool
}
