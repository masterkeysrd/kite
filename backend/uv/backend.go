package uv

import (
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

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
type Backend struct {
	terminal *uv.Terminal
	screen   *uv.TerminalScreen
	caps     backend.Caps

	wg      sync.WaitGroup
	eventCh chan event.RawEvent

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
	onResize func(layout.Size)
}

// New creates a UV backend using the default controlling terminal.
func New() (*Backend, error) {
	opts := uv.DefaultOptions()
	// Match the working integration's event timeout.
	opts.EventTimeout = 8 * time.Millisecond

	t := uv.NewTerminal(nil, opts)
	screen := t.Screen()

	// Initial setup of the screen properties.
	screen.SetMouseEncoding(uv.MouseEncodingSGR)
	screen.SetMouseMode(uv.MouseModeClick)
	screen.SetSynchronizedUpdates(os.Getenv("TMUX") == "")

	w, h, err := t.GetSize()
	if err != nil {
		w, h = 80, 24
	}

	b := &Backend{
		terminal: t,
		screen:   screen,
		width:    w,
		height:   h,
		eventCh:  make(chan event.RawEvent),
		renderCh: make(chan renderRequest, 2),
	}
	b.caps = probeCapabilities(os.Environ())
	return b, nil
}

// SetResizeHandler registers fn as the callback invoked when the terminal is
// resized.
func (b *Backend) SetResizeHandler(fn func(layout.Size)) {
	b.onResize = fn
}

// Start enters alt-screen, enables raw mode, and starts the render goroutine.
func (b *Backend) Start() error {
	if err := b.terminal.Start(); err != nil {
		return err
	}
	b.screen.EnterAltScreen()
	b.screen.HideCursor()

	// Update dimensions from the now-started terminal/screen.
	b.width = b.screen.Width()
	b.height = b.screen.Height()

	// Initial flush to clear the screen.
	b.screen.Render()
	_ = b.screen.Flush()

	b.renderWG.Add(1)
	go b.renderLoop()

	b.wg.Go(func() {
		b.loopEvents()
	})

	return nil
}

// BeginFrame allocates a fresh FrameBuffer for the current terminal size.
func (b *Backend) BeginFrame() paint.Surface {
	b.current = paint.NewFrameBuffer(0, 0, b.width, b.height)
	b.current.BumpVersion()
	return b.current
}

// EndFrame hands the current FrameBuffer to the render goroutine.
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
// the cursor.
func (b *Backend) Restore() {
	if b.stopped.Swap(true) {
		return
	}
	b.screen.ExitAltScreen()
	b.screen.ShowCursor()
	_ = b.terminal.Stop()
}

// Stop closes the render goroutine gracefully and calls Restore.
func (b *Backend) Stop() {
	close(b.renderCh)
	close(b.eventCh)
	b.renderWG.Wait()
	b.Restore()
}

// Events returns the terminal event channel.
func (b *Backend) Events() <-chan event.RawEvent {
	return b.eventCh
}

// Resize updates the internal dimensions after a terminal resize event.
func (b *Backend) Resize(size layout.Size) {
	b.width = size.Width
	b.height = size.Height
	b.screen.Resize(size.Width, size.Height)
}

// Size returns the current terminal size.
func (b *Backend) Size() layout.Size { return layout.Size{Width: b.width, Height: b.height} }

func (b *Backend) renderLoop() {
	defer b.renderWG.Done()
	for req := range b.renderCh {
		b.renderFrame(req.fb)
	}
}

func (b *Backend) loopEvents() {
	for ev := range b.terminal.Events() {
		KiteEv := translateEvent(ev)
		b.eventCh <- KiteEv
	}
}

func (b *Backend) renderFrame(fb *paint.FrameBuffer) {
	slog.Info("uv: renderFrame started", "width", fb.Bounds().Size.Width, "height", fb.Bounds().Size.Height)
	bounds := fb.Bounds()
	if bounds.Size.Width <= 0 || bounds.Size.Height <= 0 {
		slog.Warn("uv: renderFrame called with non-positive dimensions, skipping", "width", bounds.Size.Width, "height", bounds.Size.Height)
		return
	}

	// For UV, we iterate the whole requested buffer and set cells.
	// UV's own diffing will handle efficient updates.
	for y := 0; y < bounds.Size.Height; y++ {
		for x := 0; x < bounds.Size.Width; {
			c := fb.CellAt(bounds.Origin.X+x, bounds.Origin.Y+y)
			uvCell := paintCellToUV(c)
			b.screen.SetCell(x, y, uvCell)
			x += max(1, uvCell.Width)
		}
	}
	b.screen.Render()
	slog.Info("uv: screen.Render called")
	if err := b.screen.Flush(); err != nil {
		slog.Error("uv: flush error", "error", err)
	} else {
		slog.Info("uv: screen.Flush called")
	}
}

func paintCellToUV(c paint.Cell) *uv.Cell {
	content := c.Content
	if content == "" || content == "\x00" {
		content = " "
	}

	cell := &uv.Cell{
		Content: content,
		Width:   max(1, c.Width),
	}

	cell.Style.Fg = c.FG
	if c.BG != nil {
		_, _, _, a := c.BG.RGBA()
		if a > 0 {
			cell.Style.Bg = c.BG
		}
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
