package uv

import (
	"encoding/base64"
	"fmt"
	"image/color"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	kitelog "github.com/masterkeysrd/kite/log"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/terminal"
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
	buffer      *backend.Buffer
	cursor      cursorRecord
	insertAbove *backend.Buffer
	clear       bool
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

	// started indicates the backend has been started.
	started atomic.Bool

	// inline indicates inline mode is enabled.
	inline bool

	stopOnce     sync.Once
	stopMu       sync.RWMutex
	renderClosed bool

	// stopCh is closed to signal event loop termination.
	stopCh chan struct{}

	// onResize is called when the terminal is resized.
	onResize func(geom.Size)

	// cursorState is the buffered cursor state to be applied in the next frame.
	cursorState       cursorRecord
	lastPaintedCursor cursorRecord

	// used to stop the world so we can dump internal state for debugging without
	// concurrent modifications from the render loop.
	block sync.Mutex

	bufferPool sync.Pool

	// pendingRequests preserves output ordering before b.Start().
	pendingRequests []renderRequest

	// lastInlineBuf and lastInlineCursor cache the most recently rendered inline frame
	// so renderLoop can restore the active inline UI after an InsertAbove batch.
	lastInlineBuf    *backend.Buffer
	lastInlineCursor cursorRecord
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
		renderCh: make(chan renderRequest, 256),
		stopCh:   make(chan struct{}),
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

// SetInline configures the backend to run in inline mode.
func (b *Backend) SetInline(inline bool) {
	b.inline = inline
	if inline && b.screen != nil {
		b.screen.SetMouseMode(uv.MouseModeNone)
	}
}

// Start enters alt-screen, enables raw mode, and starts the render goroutine.
func (b *Backend) Start() error {
	if err := b.terminal.Start(); err != nil {
		return err
	}
	if !b.inline {
		b.screen.EnterAltScreen()
	}
	b.screen.HideCursor()
	// Enable focus events reporting.
	b.writeRaw("\x1b[?1004h")
	// Save window title on terminal stack.
	b.writeRaw("\x1b[22;0t")
	// Query Program Status Protocol support (OSC 7501).
	b.writeRaw("\x1b]7501;?\x1b\\")

	// Update dimensions from the now-started terminal/screen.
	if w := b.screen.Width(); w > 0 {
		b.width = w
	}
	if h := b.screen.Height(); h > 0 {
		b.height = h
	}

	// Initial flush to clear the screen (alt-screen only).
	if !b.inline {
		b.screen.Render()
		_ = b.screen.Flush()
	}

	b.started.Store(true)
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

	b.stopMu.RLock()
	defer b.stopMu.RUnlock()
	if !b.renderClosed {
		b.renderCh <- req
	}
	return nil
}

// Caps returns the terminal capabilities detected at startup.
func (b *Backend) Caps() backend.Caps { return b.caps }

func (b *Backend) stopRenderLoop() {
	b.stopOnce.Do(func() {
		if b.started.Load() {
			b.stopMu.Lock()
			b.renderClosed = true
			close(b.renderCh)
			b.stopMu.Unlock()

			b.renderWG.Wait()
		}
	})
}

func (b *Backend) Restore() {
	b.stopRenderLoop()

	if b.stopped.Swap(true) {
		return
	}
	close(b.stopCh)
	_ = b.terminal.Stop()

	// Restore window title from terminal stack.
	b.writeRaw("\x1b[23;0t")
	// Disable focus events reporting.
	b.writeRaw("\x1b[?1004l")
	if !b.inline {
		b.screen.ExitAltScreen()
	}
	b.screen.ShowCursor()
	_ = b.screen.Flush()
	if b.inline && b.screen.Height() > 0 {
		b.writeRaw("\r\n")
	}
}

// Stop closes the render goroutine gracefully and calls Restore.
func (b *Backend) Stop() {
	b.stopRenderLoop()
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
	if !b.inline {
		b.screen.Resize(size.Width, size.Height)
	}
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
	if b.terminal != nil {
		return b.terminal.Write(p)
	}
	if b.screen != nil {
		_, err := b.screen.WriteString(string(p))
		if err != nil {
			return 0, err
		}
		return len(p), b.screen.Flush()
	}
	return len(p), nil
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

func (b *Backend) SetTitle(s string) {
	b.screen.SetWindowTitle(s)
}

func (b *Backend) Title() string {
	return b.screen.WindowTitle()
}

func (b *Backend) Bell() {
	b.writeRaw("\x07")
}

func (b *Backend) SetProgressBar(state backend.ProgressBarState, percentage int) {
	var uvState uv.ProgressBarState
	switch state {
	case terminal.ProgressBarHide:
		uvState = uv.ProgressBarNone
	case terminal.ProgressBarNormal:
		uvState = uv.ProgressBarDefault
	case terminal.ProgressBarError:
		uvState = uv.ProgressBarError
	case terminal.ProgressBarIndeterminate:
		uvState = uv.ProgressBarIndeterminate
	case terminal.ProgressBarPaused:
		uvState = uv.ProgressBarWarning
	}
	b.screen.SetProgressBar(uv.NewProgressBar(uvState, percentage))
}

func (b *Backend) SetProgramStatus(status *terminal.ProgramStatus) {
	if status == nil {
		b.screen.SetProgramStatus(nil)
		return
	}
	uvStatus := &uv.ProgramStatus{
		State:       uv.ProgramState(status.State),
		ID:          status.ID,
		App:         status.App,
		Kind:        uv.ProgramStatusKind(status.Kind),
		Progress:    status.Progress,
		HasProgress: status.HasProgress,
		Title:       status.Title,
		Message:     status.Message,
	}
	b.screen.SetProgramStatus(uvStatus)
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

func cloneBuffer(buf *backend.Buffer) *backend.Buffer {
	if buf == nil || buf.Width < 0 || buf.Height < 0 {
		return nil
	}
	cp := backend.NewBuffer(buf.Width, buf.Height)
	copy(cp.Cells, buf.Cells)
	return cp
}

func (b *Backend) renderLoop() {
	defer b.renderWG.Done()

	b.block.Lock()
	pending := b.pendingRequests
	b.pendingRequests = nil
	b.block.Unlock()

	for _, req := range pending {
		if req.clear {
			b.processClear()
		} else {
			b.processInsertAbove(req.insertAbove)
		}
	}

	for req := range b.renderCh {
		if req.clear {
			b.processClear()
			continue
		}
		if req.insertAbove != nil {
			b.processInsertAbove(req.insertAbove)
			renderedFreshFrame := false

		drainLoop:
			for len(b.renderCh) > 0 {
				select {
				case next, ok := <-b.renderCh:
					if !ok {
						break drainLoop
					}
					if next.clear {
						b.processClear()
					} else if next.insertAbove != nil {
						b.processInsertAbove(next.insertAbove)
					} else {
						b.renderFrame(next)
						renderedFreshFrame = true
						break drainLoop
					}
				default:
					break drainLoop
				}
			}

			if !renderedFreshFrame && b.inline {
				b.block.Lock()
				if b.lastInlineBuf != nil {
					b.paintScreen(b.lastInlineBuf, b.lastInlineCursor)
				}
				b.block.Unlock()
			}
			continue
		}
		b.renderFrame(req)
	}
}

// Clear queues a terminal reset before subsequent inserts and frames.
func (b *Backend) Clear() {
	b.enqueueOutput(renderRequest{clear: true})
}

func (b *Backend) processClear() {
	b.block.Lock()
	defer b.block.Unlock()

	w := b.width
	if w <= 0 {
		w = b.screen.Width()
	}
	b.screen.HideCursor()
	b.screen.Resize(w, 0)
	b.screen.Render()
	if err := b.screen.Flush(); err != nil {
		kitelog.Error("uv: flush error before clear", "error", err)
	}
	if _, err := b.Write([]byte("\x1b[2J\x1b[3J\x1b[H")); err != nil {
		kitelog.Error("uv: clear error", "error", err)
	}
	b.lastInlineBuf = nil
	b.lastInlineCursor = cursorRecord{}
	b.lastPaintedCursor = cursorRecord{}
	if !b.inline {
		b.screen.Resize(w, b.height)
	}
}

// InsertAbove inserts a rendered cell buffer permanently into the terminal
// scrollback above the active inline frame.
func (b *Backend) InsertAbove(buf *backend.Buffer) {
	if buf == nil || buf.Width <= 0 || buf.Height <= 0 {
		return
	}
	cp := cloneBuffer(buf)
	b.enqueueOutput(renderRequest{insertAbove: cp})
}

func (b *Backend) enqueueOutput(req renderRequest) {
	if b.stopped.Load() {
		return
	}
	if !b.started.Load() {
		b.block.Lock()
		if !b.started.Load() {
			b.pendingRequests = append(b.pendingRequests, req)
			b.block.Unlock()
			return
		}
		b.block.Unlock()
	}
	b.stopMu.RLock()
	defer b.stopMu.RUnlock()
	if !b.renderClosed {
		b.renderCh <- req
	}
}

func (b *Backend) processInsertAbove(buf *backend.Buffer) {
	b.block.Lock()
	defer b.block.Unlock()

	if buf == nil || buf.Width <= 0 || buf.Height <= 0 {
		return
	}

	if b.screen != nil && b.screen.Height() > 0 {
		w := b.width
		if w <= 0 {
			w = b.screen.Width()
		}
		if w <= 0 {
			w = 80
		}
		b.screen.HideCursor()
		b.screen.Resize(w, 0)
		b.screen.Render()
		_ = b.screen.Flush()
		b.lastPaintedCursor = cursorRecord{}
	}

	uvBuf := uv.NewBuffer(buf.Width, buf.Height)
	var uvCell uv.Cell
	for y := 0; y < buf.Height; y++ {
		for x := 0; x < buf.Width; {
			c := buf.CellAt(x, y)
			populateUVCell(&uvCell, c)
			uvBuf.SetCell(x, y, &uvCell)
			x += max(1, uvCell.Width)
		}
	}

	for y := 0; y < len(uvBuf.Lines); y++ {
		line := uvBuf.Lines[y]
		for len(line) > 0 && line[len(line)-1].Equal(&uv.EmptyCell) {
			line = line[:len(line)-1]
		}
		uvBuf.Lines[y] = line
	}

	var sb strings.Builder
	sb.WriteByte('\r')
	for _, line := range uvBuf.Lines {
		sb.WriteString(line.Render())
		sb.WriteString("\x1b[K\r\n")
	}
	b.writeRaw(sb.String())
}

func (b *Backend) loopEvents() {
	for ev := range b.terminal.Events() {
		kitelog.Info("UV: Event from terminal", "event", fmt.Sprintf("%#v", ev))
		KiteEv := translateEvent(ev)
		if KiteEv != nil {
			select {
			case b.eventCh <- KiteEv:
			case <-b.stopCh:
				return
			}
		}
	}
}

func (b *Backend) renderFrame(req renderRequest) {
	b.block.Lock()
	defer b.block.Unlock()
	if req.buffer != nil {
		defer b.bufferPool.Put(req.buffer)
	}

	buf := req.buffer
	if b.inline && buf != nil {
		b.lastInlineBuf = cloneBuffer(buf)
		b.lastInlineCursor = req.cursor
	}

	b.paintScreen(buf, req.cursor)
}

func (b *Backend) paintScreen(buf *backend.Buffer, cursor cursorRecord) {
	if !b.inline && (buf == nil || buf.Width <= 0 || buf.Height <= 0) {
		if buf != nil {
			kitelog.Warn("uv: renderFrame called with non-positive dimensions, skipping", "width", buf.Width, "height", buf.Height)
		}
		return
	}
	if b.inline {
		if buf == nil || buf.Width <= 0 || buf.Height < 0 {
			if buf != nil {
				kitelog.Warn("uv: renderFrame called with invalid dimensions, skipping", "width", buf.Width, "height", buf.Height)
			}
			return
		}
		if b.screen.Width() != buf.Width || b.screen.Height() != buf.Height {
			b.screen.Resize(buf.Width, buf.Height)
		}
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
	cursorChanged := cursor != b.lastPaintedCursor
	if cursorChanged {
		b.lastPaintedCursor = cursor

		if cursor.Visible {
			uvShape, blink := translateCursorShape(cursor.Shape)
			b.screen.SetCursorStyle(uvShape, blink)
			if cursor.Color != nil {
				b.screen.SetCursorColor(cursor.Color)
			}
			b.screen.SetCursorPosition(cursor.X, cursor.Y)
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
	if cursor.Visible {
		b.screen.SetCursorPosition(cursor.X, cursor.Y)
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
