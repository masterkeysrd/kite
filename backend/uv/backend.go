package uv

import (
	"image/color"
	"os"
	"sync"
	"sync/atomic"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/paint"
)

// Compile-time assertion: Backend implements backend.Backend.
var _ backend.Backend = (*Backend)(nil)

// renderRequest is sent from EndFrame to the render goroutine.
type renderRequest struct {
	fb      *paint.FrameBuffer
	version uint64
}

// Backend is the ultraviolet terminal backend for kite/x. It owns a uv.Terminal
// for input and a uv.TerminalScreen for output. EndFrame copies the painted
// FrameBuffer to the TerminalScreen and signals the render goroutine to diff
// and flush.
//
// The Backend must be created via New and used from a single main goroutine
// for frame operations; the render goroutine is internal and backend-owned.
type Backend struct {
	terminal *uv.Terminal
	screen   *uv.TerminalScreen
	caps     backend.Caps

	// current is the FrameBuffer prepared by BeginFrame and drawn into by the
	// paint engine. It is handed to the render goroutine in EndFrame.
	current *paint.FrameBuffer

	// width and height of the terminal at last resize.
	width, height int

	// renderCh carries frames from the main thread to the render goroutine.
	renderCh chan renderRequest

	// renderWG lets Stop wait for the render goroutine to finish.
	renderWG sync.WaitGroup

	// stopped indicates the backend has been stopped.
	stopped atomic.Bool

	// onResize is called when the terminal is resized.
	onResize func(width, height int)
}

// New creates a UV backend using the default controlling terminal.
//
// The returned Backend is ready for use but not yet started. Callers must
// call Start before invoking BeginFrame / EndFrame.
func New() (*Backend, error) {
	t := uv.DefaultTerminal()
	w, h, err := t.GetSize()
	if err != nil {
		return nil, err
	}

	b := &Backend{
		terminal: t,
		screen:   t.Screen(),
		width:    w,
		height:   h,
		renderCh: make(chan renderRequest, 2),
	}
	b.caps = probeCapabilities(os.Environ())
	return b, nil
}

// SetResizeHandler registers fn as the callback invoked when the terminal is
// resized. fn is called with the new dimensions. It is safe to call this
// before Start.
func (b *Backend) SetResizeHandler(fn func(width, height int)) {
	b.onResize = fn
}

// Start enters alt-screen, enables raw mode, and starts the render goroutine.
// It must be called once before BeginFrame / EndFrame.
func (b *Backend) Start() error {
	b.screen.EnterAltScreen()
	if err := b.terminal.Start(); err != nil {
		return err
	}
	b.renderWG.Add(1)
	go b.renderLoop()
	return nil
}

// BeginFrame allocates a fresh FrameBuffer for the current terminal size,
// bumps its version, and returns it as a paint.Surface.
func (b *Backend) BeginFrame() paint.Surface {
	b.current = paint.NewFrameBuffer(0, 0, b.width, b.height)
	b.current.BumpVersion()
	return b.current
}

// EndFrame hands the current FrameBuffer to the render goroutine. It panics
// if BeginFrame was not called first.
func (b *Backend) EndFrame() error {
	if b.current == nil {
		panic("uv.Backend.EndFrame called without a preceding BeginFrame")
	}
	req := renderRequest{fb: b.current, version: b.current.Version()}
	b.current = nil
	b.renderCh <- req
	return nil
}

// Caps returns the terminal capabilities detected at startup.
func (b *Backend) Caps() backend.Caps { return b.caps }

// Restore unconditionally exits alt-screen, restores terminal state, and shows
// the cursor. Safe to call from a signal handler or deferred panic recovery.
func (b *Backend) Restore() {
	if b.stopped.Swap(true) {
		return // already restored
	}
	b.screen.ExitAltScreen()
	b.screen.ShowCursor()
	_ = b.terminal.Stop()
}

// Stop closes the render goroutine gracefully and calls Restore.
func (b *Backend) Stop() {
	close(b.renderCh)
	b.renderWG.Wait()
	b.Restore()
}

// Events returns the terminal event channel. The engine's input goroutine
// ranges over this channel until the backend is stopped.
func (b *Backend) Events() <-chan event.RawEvent {
	ch := make(chan event.RawEvent)
	go func() {
		defer close(ch)
		for ev := range b.terminal.Events() {
			ch <- translateEvent(ev)
		}
	}()
	return ch
}

// Resize updates the internal dimensions after a terminal resize event.
func (b *Backend) Resize(width, height int) {
	b.width = width
	b.height = height
	b.screen.Resize(width, height)
}

// Width returns the current terminal width.
func (b *Backend) Width() int { return b.width }

// Height returns the current terminal height.
func (b *Backend) Height() int { return b.height }

// Size returns the current terminal size.
func (b *Backend) Size() layout.Size { return layout.Size{Width: b.width, Height: b.height} }

// renderLoop is the backend-owned goroutine that receives frames from EndFrame,
// converts them to uv.Cells, sets them on the TerminalScreen, and flushes.
//
// renderLoop exits when renderCh is closed (by Stop).
func (b *Backend) renderLoop() {
	defer b.renderWG.Done()
	for req := range b.renderCh {
		b.renderFrame(req.fb)
	}
}

// renderFrame copies cells from fb to the TerminalScreen and flushes.
func (b *Backend) renderFrame(fb *paint.FrameBuffer) {
	bounds := fb.Bounds()
	for y := 0; y < bounds.Size.Height; y++ {
		for x := 0; x < bounds.Size.Width; {
			c := fb.Get(bounds.Origin.X+x, bounds.Origin.Y+y)
			uvCell := paintCellToUV(c)
			b.screen.SetCell(x, y, uvCell)
			x += max(1, uvCell.Width)
		}
	}
	b.screen.Render()
	_ = b.screen.Flush()
}

// paintCellToUV converts a paint.Cell to a *uv.Cell for writing to the screen.
func paintCellToUV(c paint.Cell) *uv.Cell {
	content := string(c.Rune)
	if content == "" || content == "\x00" {
		content = " "
	}

	cell := &uv.Cell{
		Content: content,
		Width:   max(1, c.Width),
	}

	cell.Style.Fg = c.FG
	if c.BG != color.Transparent {
		cell.Style.Bg = c.BG
	}

	if c.Attrs&paint.AttrBold != 0 {
		cell.Style.Attrs |= uv.AttrBold
	}
	if c.Attrs&paint.AttrItalic != 0 {
		cell.Style.Attrs |= uv.AttrItalic
	}
	if c.Attrs&paint.AttrUnderline != 0 {
		cell.Style.Underline = uv.UnderlineSingle
	}
	if c.Attrs&paint.AttrInverse != 0 {
		cell.Style.Attrs |= uv.AttrReverse
	}
	return cell
}
