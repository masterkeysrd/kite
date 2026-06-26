package uv

import (
	"encoding/base64"
	"fmt"
	"image/color"
	"io"
	kitelog "github.com/masterkeysrd/kite/log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/geom"
)

// Compile-time assertion: Backend implements backend.Backend.
var _ backend.Backend = (*Backend)(nil)

// renderRequest is sent from EndFrame to the render goroutine.
type cursorRecord struct {
	Visible bool
	X, Y    int
	Shape   backend.CursorShape
	Color   color.Color
}

type renderRequest struct {
	buffer *backend.Buffer
	cursor cursorRecord
}

// Backend is the ultraviolet terminal backend for kite/x. It owns a uv.Terminal
// for input and a uv.TerminalScreen for output. EndFrame copies the painted
// Buffer to the TerminalScreen and signals the render goroutine to diff
// and flush.
type Backend struct {
	terminal *uv.Terminal
	screen   *uv.TerminalScreen
	caps     backend.Caps

	wg      sync.WaitGroup
	eventCh chan backend.RawEvent

	// current is the Buffer prepared by BeginFrame and drawn into by the
	// paint engine. It is handed to the render goroutine in EndFrame.
	current *backend.Buffer

	// width and height of the terminal at last resize.
	width, height int

	// renderCh carries frames from the main thread to the render goroutine.
	renderCh chan renderRequest

	// renderWG lets Stop wait for the render goroutine to finish.
	renderWG sync.WaitGroup

	// stopped indicates the backend has been stopped.
	stopped atomic.Bool

	// onResize is called when the terminal is resized.
	onResize func(geom.Size)

	// cursorState is the buffered cursor state to be applied in the next frame.
	cursorState       cursorRecord
	lastPaintedCursor cursorRecord

	// used to stop the world so we can dump internal state for debugging without
	// concurrent modifications from the render loop.
	block sync.Mutex

	bufferPool sync.Pool
}

const (
	// Terminal escape headers.
	escOSC = "\x1b]"
	escST  = "\x1b\\"

	// Internal event prefixes.
	prefixCSI = "CSI:"
	prefixDCS = "DCS:"
	prefixAPC = "APC:"
	prefixUNK = "UNK:"
	prefixBGC = "BGC:"
)

// New creates a UV backend using the default controlling terminal.
func New() (*Backend, error) {
	opts := uv.DefaultOptions()
	opts.Logger = logWriter{}
	// Match the working integration's event timeout.
	opts.EventTimeout = 8 * time.Millisecond

	t := uv.NewTerminal(nil, opts)
	screen := t.Screen()

	// Initial setup of the screen properties.
	screen.SetMouseEncoding(uv.MouseEncodingSGR)
	screen.SetMouseMode(uv.MouseModeMotion)
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
		eventCh:  make(chan backend.RawEvent),
		renderCh: make(chan renderRequest, 2),
	}
	b.bufferPool.New = func() any {
		return backend.NewBuffer(b.width, b.height)
	}
	b.caps = probeCapabilities(os.Environ())
	return b, nil
}

// SetResizeHandler registers fn as the callback invoked when the terminal is
// resized.
func (b *Backend) SetResizeHandler(fn func(geom.Size)) {
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

// BeginFrame allocates a fresh Buffer for the current terminal size.
func (b *Backend) BeginFrame() backend.Surface {
	buf := b.bufferPool.Get().(*backend.Buffer)
	if buf.Width != b.width || buf.Height != b.height {
		buf = backend.NewBuffer(b.width, b.height)
	} else {
		buf.Reset()
	}
	b.current = buf
	return b.current
}

// EndFrame hands the current Buffer to the render goroutine.
func (b *Backend) EndFrame() error {
	if b.current == nil {
		panic("uv.Backend.EndFrame called without a preceding BeginFrame")
	}
	req := renderRequest{
		buffer: b.current,
		cursor: b.cursorState,
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
func (b *Backend) Events() <-chan backend.RawEvent {
	return b.eventCh
}

// Resize updates the internal dimensions after a terminal resize event.
func (b *Backend) Resize(size geom.Size) {
	b.width = size.Width
	b.height = size.Height
	b.screen.Resize(size.Width, size.Height)
}

// Size returns the current terminal size.
func (b *Backend) Size() geom.Size { return geom.Size{Width: b.width, Height: b.height} }

// Writer returns the terminal output writer (the backend itself).
func (b *Backend) Writer() io.Writer { return b }

type osc52Clipboard struct {
	b *Backend
}

func (c *osc52Clipboard) Set(mime string, data []byte) {
	if mime != "text/plain" {
		return
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	c.b.writeRaw(fmt.Sprintf("\x1b]52;c;%s\x1b\\", b64))
}

func (c *osc52Clipboard) Request(mime string) {
	if mime != "text/plain" {
		return
	}
	c.b.writeRaw("\x1b]52;c;?\x1b\\")
}

func (b *Backend) Clipboard() backend.Clipboard {
	return &osc52Clipboard{b: b}
}

func (b *Backend) writeRaw(s string) {
	b.Write([]byte(s))
}

func (b *Backend) Write(p []byte) (n int, err error) {
	s := strings.ReplaceAll(string(p), "\x1b", "\\x1b")
	kitelog.Debug("UV TRACING: ", "message", fmt.Sprintf("output: %q", s))
	return b.terminal.Write(p)
}

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

func (b *Backend) SetCursorShape(s backend.CursorShape) {
	b.cursorState.Shape = s
}

func (b *Backend) DumpState() {
	b.block.Lock()
	defer b.block.Unlock()

	var sb strings.Builder
	sb.WriteString("=== Backend State Dump ===\n")
	fmt.Fprintf(&sb, "Terminal Size: %dx%d\n", b.width, b.height)

	sb.WriteString("Current Buffer:\n")
	if b.current != nil {
		for y := 0; y < b.current.Height; y++ {
			for x := 0; x < b.current.Width; x++ {
				c := b.current.CellAt(x, y)
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
		kitelog.Error("uv: failed to create dump file", "error", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(sb.String()); err != nil {
		kitelog.Error("uv: failed to write dump file", "error", err)
		return
	}

	kitelog.Info("uv: backend state dumped to file")
}

func (b *Backend) renderLoop() {
	defer b.renderWG.Done()
	for req := range b.renderCh {
		b.renderFrame(req)
	}
}

func (b *Backend) loopEvents() {
	for ev := range b.terminal.Events() {
		kitelog.Info("UV: Event from terminal", "event", fmt.Sprintf("%#v", ev))
		KiteEv := translateEvent(ev)
		if KiteEv != nil {
			b.eventCh <- KiteEv
		}
	}
}

func (b *Backend) renderFrame(req renderRequest) {
	b.block.Lock()
	defer b.block.Unlock()
	defer b.bufferPool.Put(req.buffer)

	buf := req.buffer
	if buf.Width <= 0 || buf.Height <= 0 {
		kitelog.Warn("uv: renderFrame called with non-positive dimensions, skipping", "width", buf.Width, "height", buf.Height)
		return
	}

	// Reuse a single uv.Cell object to avoid thousands of allocations per frame.
	var uvCell uv.Cell
	for y := 0; y < buf.Height; y++ {
		for x := 0; x < buf.Width; {
			c := buf.CellAt(x, y)
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
		kitelog.Error("uv: flush error", "error", err)
	} else {
		kitelog.Info("uv: screen.Flush called")
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
			kitelog.Error("uv: flush error after cursor update", "error", err)
		} else {
			kitelog.Info("uv: screen.Flush called after cursor update")
		}
	}
}

func translateCursorShape(s backend.CursorShape) (uv.CursorShape, bool) {
	switch s {
	case backend.CursorBlock:
		return uv.CursorBlock, true
	case backend.CursorUnderline:
		return uv.CursorUnderline, true
	case backend.CursorBar:
		return uv.CursorBar, true
	default:
		return uv.CursorBlock, true
	}
}

func populateUVCell(cell *uv.Cell, c backend.Cell) {
	content := c.Content
	if content == "" || content == "\x00" {
		content = " "
	}

	cell.Content = content
	cell.Width = max(1, c.Width)

	cell.Style.Attrs = 0
	cell.Style.Fg = c.Fg
	if c.Bg != nil {
		_, _, _, a := c.Bg.RGBA()
		if a > 0 {
			cell.Style.Bg = c.Bg
		} else {
			cell.Style.Bg = nil
		}
	} else {
		cell.Style.Bg = nil
	}

	if c.Style&backend.CellBold != 0 {
		cell.Style.Attrs |= uv.AttrBold
	}
	if c.Style&backend.CellFaint != 0 {
		cell.Style.Attrs |= uv.AttrFaint
	}
	if c.Style&backend.CellItalic != 0 {
		cell.Style.Attrs |= uv.AttrItalic
	}
	if c.Style&backend.CellBlink != 0 {
		cell.Style.Attrs |= uv.AttrBlink
	}
	if c.Style&backend.CellReverse != 0 {
		cell.Style.Attrs |= uv.AttrReverse
	}
	if c.Style&backend.CellConceal != 0 {
		cell.Style.Attrs |= uv.AttrConceal
	}
	if c.Style&backend.CellStrikethrough != 0 {
		cell.Style.Attrs |= uv.AttrStrikethrough
	}
}

type logWriter struct{}

func (w logWriter) Printf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	msg = strings.ReplaceAll(msg, "\x1b", "\\x1b")
	kitelog.Debug("UV TRACING: ", "message", msg)
}
