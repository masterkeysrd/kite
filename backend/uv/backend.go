package uv

import (
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/paint"
)

// Compile-time assertion: Backend implements backend.Backend.
var _ backend.Backend = (*Backend)(nil)

// renderRequest is sent from EndFrame to the render goroutine.
type cursorRecord struct {
	Visible bool
	X, Y    int
	Shape   cursor.Shape
	Color   color.Color
}

type renderRequest struct {
	fb      *paint.FrameBuffer
	version uint64
	cursor  cursorRecord
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

	// syncCursorCh is needed to keep the cursor up-to-date in the render goroutine, since the
	// cursor state is buffered in the main thread and only sent to the render goroutine in
	// EndFrame. This channel allows cursor updates to be sent immediately, without waiting
	// for the next EndFrame.

	// renderWG lets Stop wait for the render goroutine to finish.
	renderWG sync.WaitGroup

	// stopped indicates the backend has been stopped.
	stopped atomic.Bool

	// onResize is called when the terminal is resized.
	onResize func(layout.Size)

	// cursorState is the buffered cursor state to be applied in the next frame.
	cursorState       cursorRecord
	lastPaintedCursor cursorRecord

	// used to stop the world so we can dump internal state for debugging without
	// concurrent modifications from the render loop.
	block sync.Mutex

	fbPool sync.Pool
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
	screen.EnableBracketedPaste()

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
	b.fbPool.New = func() any {
		return paint.NewFrameBuffer(0, 0, b.width, b.height)
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
	fb := b.fbPool.Get().(*paint.FrameBuffer)
	if fb.Bounds().Size.Width != b.width || fb.Bounds().Size.Height != b.height {
		fb = paint.NewFrameBuffer(0, 0, b.width, b.height)
	} else {
		fb.Reset()
	}
	b.current = fb
	b.current.BumpVersion()
	return b.current
}

// EndFrame hands the current FrameBuffer to the render goroutine.
func (b *Backend) EndFrame() error {
	if b.current == nil {
		panic("uv.Backend.EndFrame called without a preceding BeginFrame")
	}
	req := renderRequest{
		fb:      b.current,
		version: b.current.Version(),
		cursor:  b.cursorState,
	}
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

func (b *Backend) ShowCursor(v bool) {
	b.cursorState.Visible = v
}

func (b *Backend) SetCursorPos(x, y int) {
	b.cursorState.X = x
	b.cursorState.Y = y
}

func (b *Backend) SetCursorColor(c color.Color) {
	b.cursorState.Color = c
}

func (b *Backend) SetCursorShape(s cursor.Shape) {
	b.cursorState.Shape = s
}

func (b *Backend) DumpState() {
	b.block.Lock()
	defer b.block.Unlock()

	var sb strings.Builder
	sb.WriteString("=== Backend State Dump ===\n")
	fmt.Fprintf(&sb, "Terminal Size: %dx%d\n", b.width, b.height)

	sb.WriteString("Current FrameBuffer:\n")
	if b.current != nil {
		for y := 0; y < b.current.Bounds().Size.Height; y++ {
			for x := 0; x < b.current.Bounds().Size.Width; x++ {
				c := b.current.CellAt(b.current.Bounds().Origin.X+x, b.current.Bounds().Origin.Y+y)
				sb.WriteString(c.Content)
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("<nil>\n")
	}

	// Current screen state
	sb.WriteString("Current Screen State:\n")
	for y := 0; y < b.screen.Height(); y++ {
		for x := 0; x < b.screen.Width(); x++ {
			c := b.screen.CellAt(x, y)
			// sb.WriteString(c.Content)
			if c.Content == "" && c.Width == 0 {
				sb.WriteString(" ")
			} else {
				sb.WriteString(c.Content)
			}

		}
		sb.WriteString("\n")
	}

	sb.WriteString("Cursor State:\n")
	fmt.Fprintf(&sb, "  Visible: %v\n", b.cursorState.Visible)
	fmt.Fprintf(&sb, "  Position: (%d, %d)\n", b.cursorState.X, b.cursorState.Y)
	fmt.Fprintf(&sb, "  Shape: %v\n", b.cursorState.Shape)
	if b.cursorState.Color != nil {
		r, g, b, a := b.cursorState.Color.RGBA()
		fmt.Fprintf(&sb, "  Color: RGBA(%d, %d, %d, %d)\n", r, g, b, a)
	} else {
		sb.WriteString("  Color: <nil>\n")
	}

	f, err := os.Create(fmt.Sprintf("backend_dump_%d.txt", time.Now().Unix()))
	if err != nil {
		slog.Error("uv: failed to create dump file", "error", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(sb.String()); err != nil {
		slog.Error("uv: failed to write dump file", "error", err)
		return
	}

	slog.Info("uv: backend state dumped to file")
}

func (b *Backend) renderLoop() {
	defer b.renderWG.Done()
	for req := range b.renderCh {
		b.renderFrame(req)
	}
}

func (b *Backend) loopEvents() {
	for ev := range b.terminal.Events() {
		KiteEv := translateEvent(ev)
		b.eventCh <- KiteEv
	}
}

func (b *Backend) renderFrame(req renderRequest) {
	b.block.Lock()
	defer b.block.Unlock()
	defer b.fbPool.Put(req.fb)

	fb := req.fb
	slog.Info("uv: renderFrame started", "width", fb.Bounds().Size.Width, "height", fb.Bounds().Size.Height)
	bounds := fb.Bounds()
	if bounds.Size.Width <= 0 || bounds.Size.Height <= 0 {
		slog.Warn("uv: renderFrame called with non-positive dimensions, skipping", "width", bounds.Size.Width, "height", bounds.Size.Height)
		return
	}

	// Reuse a single uv.Cell object to avoid thousands of allocations per frame.
	var uvCell uv.Cell

	for y := 0; y < bounds.Size.Height; y++ {
		for x := 0; x < bounds.Size.Width; {
			c := fb.CellAt(bounds.Origin.X+x, bounds.Origin.Y+y)
			populateUVCell(&uvCell, c)
			b.screen.SetCell(x, y, &uvCell)
			x += max(1, uvCell.Width)
		}
	}

	// Determine if the cursor state has changed since the last painted frame.
	// If so, we may need to flush an additional time after updating the cursor
	// to ensure it appears correctly.
	cursorChanged := req.cursor != b.lastPaintedCursor
	if cursorChanged {
		b.lastPaintedCursor = req.cursor

		if req.cursor.Visible {
			uvShape, blink := translateCursorShape(req.cursor.Shape)
			b.screen.SetCursorStyle(uvShape, blink)
			if req.cursor.Color != nil {
				b.screen.SetCursorColor(req.cursor.Color)
			}
			b.screen.SetCursorPosition(req.cursor.X, req.cursor.Y)
			b.screen.ShowCursor()
		} else {
			b.screen.HideCursor()
		}
	}

	b.screen.Render()
	if err := b.screen.Flush(); err != nil {
		slog.Error("uv: flush error", "error", err)
	} else {
		slog.Info("uv: screen.Flush called")
	}

	// THE DOUBLE FLUSH WORKAROUND
	//
	// Why is this required?
	// When the first `screen.Flush()` is called above, it correctly looks at the
	// internal coordinates and generates the ANSI `MoveTo` command to place the cursor.
	// However, due to a bug in the `uv` package, it fails to flush that specific command
	// from the TerminalRenderer's internal memory into the main output buffer, leaving
	// the cursor trapped at the last painted cell.
	//
	// 1. We call an empty `Render()` to reach into the renderer and force it to
	//    dump its trapped `MoveTo` sequence into the main output buffer.
	// 2. We call a second `Flush()` to physically write that rescued buffer to the terminal.
	//
	// We strictly gate this behind `cursorChanged` so we only pay the performance
	// penalty of a double I/O write on the exact frames where the user moves the cursor.
	if req.cursor.Visible {
		b.screen.SetCursorPosition(req.cursor.X, req.cursor.Y)
		b.screen.Render()
		if err := b.screen.Flush(); err != nil {
			slog.Error("uv: flush error after cursor update", "error", err)
		} else {
			slog.Info("uv: screen.Flush called after cursor update")
		}
	}
}

func translateCursorShape(s cursor.Shape) (uv.CursorShape, bool) {
	switch s {
	case cursor.ShapeBlockBlink:
		return uv.CursorBlock, true
	case cursor.ShapeBlockSteady:
		return uv.CursorBlock, false
	case cursor.ShapeBarBlink:
		return uv.CursorBar, true
	case cursor.ShapeBarSteady:
		return uv.CursorBar, false
	case cursor.ShapeUnderlineBlink:
		return uv.CursorUnderline, true
	case cursor.ShapeUnderlineSteady:
		return uv.CursorUnderline, false
	default:
		return uv.CursorBlock, true
	}
}

func populateUVCell(cell *uv.Cell, c paint.Cell) {
	content := c.Content
	if content == "" || content == "\x00" {
		content = " "
	}

	cell.Content = content
	cell.Width = max(1, c.Width)

	cell.Style.Fg = c.FG
	if c.BG != nil {
		_, _, _, a := c.BG.RGBA()
		if a > 0 {
			cell.Style.Bg = c.BG
		} else {
			cell.Style.Bg = nil
		}
	} else {
		cell.Style.Bg = nil
	}

	cell.Style.Attrs = 0
	if c.Attrs&paint.AttrBold != 0 {
		cell.Style.Attrs |= uv.AttrBold
	}
	if c.Attrs&paint.AttrItalic != 0 {
		cell.Style.Attrs |= uv.AttrItalic
	}
	if c.Attrs&paint.AttrUnderline != 0 {
		cell.Style.Underline = uv.UnderlineSingle
	} else {
		cell.Style.Underline = uv.UnderlineNone
	}
	if c.Attrs&paint.AttrInverse != 0 {
		cell.Style.Attrs |= uv.AttrReverse
	}
}
