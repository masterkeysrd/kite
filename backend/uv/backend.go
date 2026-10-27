package uv

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/color"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	kitelog "github.com/masterkeysrd/kite/log"

	"github.com/charmbracelet/colorprofile"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/terminal"
)

// Compile-time assertion: Backend implements backend.Backend.
var _ backend.Backend = (*Backend)(nil)
var _ backend.FrameInserter = (*Backend)(nil)

type cursorRecord struct {
	Visible bool
	X, Y    int
	Shape   backend.CursorShape
	Color   color.Color
}

// renderRequest orders terminal operations on the render goroutine.
type renderRequest struct {
	buffer       *backend.Buffer
	cursor       cursorRecord
	insertAbove  *backend.Buffer
	clear        bool
	resize       bool
	size         geom.Size
	deferRestore bool
}

// Backend is the ultraviolet terminal backend for kite/x. It owns a uv.Terminal
// for input and a uv.TerminalScreen for output. EndFrame copies the painted
// Buffer to the TerminalScreen and signals the render goroutine to diff
// and flush.
type Backend struct {
	terminal *uv.Terminal
	screen   *uv.TerminalScreen
	caps     backend.Caps
	profile  colorprofile.Profile

	wg      sync.WaitGroup
	eventCh chan backend.RawEvent

	// current is the Buffer prepared by BeginFrame and drawn into by the
	// paint engine. It is handed to the render goroutine in EndFrame.
	current *backend.Buffer

	// width and height of the terminal at last resize.
	width, height int

	// Physical terminal observations take precedence over queued engine resizes.
	terminalSize  atomic.Pointer[geom.Size]
	requestedSize atomic.Pointer[geom.Size]

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

	// These fields belong to the render loop, not the engine goroutine.
	renderSize     geom.Size
	inlineResized  bool
	inlineRenderer *uv.TerminalRenderer
	inlineFrame    *uv.RenderBuffer
	inlineOutput   bytes.Buffer

	// Mirror UV's negotiation without reading its mutable width state while rendering.
	graphemeWidth atomic.Bool
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

	console := uv.DefaultConsole()
	t := uv.NewTerminal(console, opts)
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
		profile:  colorprofile.Detect(console.Writer(), console.Environ()),
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

	size := b.Size()
	b.renderSize = size
	b.terminalSize.Store(&size)
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
		size:   b.Size(),
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
	if b.inline {
		if size.Width > 0 && size.Height > 0 {
			b.requestedSize.Store(&size)
		}
		b.enqueueOutput(renderRequest{resize: true, size: size})
	} else {
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
	if kitelog.Enabled(slog.LevelDebug) {
		s := strings.ReplaceAll(string(p), "\x1b", "\\x1b")
		kitelog.Debug("UV TRACING: ", "message", fmt.Sprintf("output: %q", s))
	}
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

	restoreInline := false
	for {
		var req renderRequest
		if len(pending) > 0 {
			req, pending = pending[0], pending[1:]
		} else {
			var ok bool
			req, ok = <-b.renderCh
			if !ok {
				return
			}
		}
		size := req.size
		if b.inline {
			if req.resize && req.size.Width > 0 && req.size.Height > 0 && req.size != b.renderSize {
				b.inlineResized = true
			}
			size = b.outputSize(size)
		}
		if size.Width > 0 && size.Height > 0 {
			if b.renderSize.Width > 0 && b.renderSize != size {
				b.inlineResized = true
			}
			b.renderSize = size
		}
		if b.inline && req.buffer != nil && req.size.Width > 0 && req.size.Height > 0 && req.size != size {
			b.bufferPool.Put(req.buffer)
			continue
		}
		switch {
		case req.clear:
			b.processClear()
			restoreInline = false
		case req.resize:
			// Wait for the engine's resized frame rather than painting the
			// cached frame at stale dimensions.
		case req.insertAbove != nil:
			if req.deferRestore {
				b.block.Lock()
				b.lastInlineBuf = nil
				b.lastInlineCursor = cursorRecord{}
				b.block.Unlock()
			}
			b.processInsertAbove(req.insertAbove)
			restoreInline = !req.deferRestore
		default:
			b.renderFrame(req)
			restoreInline = false
		}
		if restoreInline && len(pending) == 0 && len(b.renderCh) == 0 {
			b.block.Lock()
			if b.inline && b.lastInlineBuf != nil && !b.inlineResized {
				size := b.outputSize(b.renderSize)
				if size != b.renderSize {
					b.renderSize = size
					b.inlineResized = true
				} else {
					b.paintScreen(b.lastInlineBuf, b.lastInlineCursor)
				}
			}
			b.block.Unlock()
			restoreInline = false
		}
	}
}

// Clear queues a terminal reset before subsequent inserts and frames.
func (b *Backend) Clear() {
	b.enqueueOutput(renderRequest{clear: true, size: b.Size()})
}

func (b *Backend) processClear() {
	b.block.Lock()
	defer b.block.Unlock()

	size := b.renderDimensions()
	w := size.Width
	if w <= 0 {
		w = b.screen.Width()
	}
	b.screen.HideCursor()
	b.resizeScreen(w, 0)
	b.renderScreen()
	if err := b.flushScreen(); err != nil {
		kitelog.Error("uv: flush error before clear", "error", err)
	}
	if _, err := b.Write([]byte("\x1b[2J\x1b[3J\x1b[H")); err != nil {
		kitelog.Error("uv: clear error", "error", err)
	}
	if b.inlineRenderer != nil {
		b.inlineRenderer.SetPosition(0, 0)
	}
	b.lastInlineBuf = nil
	b.lastInlineCursor = cursorRecord{}
	b.lastPaintedCursor = cursorRecord{}
	b.inlineResized = false
	if !b.inline {
		b.screen.Resize(w, size.Height)
	}
}

// InsertAbove inserts a rendered cell buffer permanently into the terminal
// scrollback above the active inline frame.
func (b *Backend) InsertAbove(buf *backend.Buffer) {
	b.insertAbove(buf, false)
}

// InsertAboveForFrame queues static output without restoring a stale live frame.
// The engine commits the newly reconciled live frame after these inserts.
func (b *Backend) InsertAboveForFrame(buf *backend.Buffer) {
	b.insertAbove(buf, true)
}

func (b *Backend) insertAbove(buf *backend.Buffer, deferRestore bool) {
	if buf == nil || buf.Width <= 0 || buf.Height <= 0 {
		return
	}
	cp := cloneBuffer(buf)
	b.enqueueOutput(renderRequest{insertAbove: cp, size: b.Size(), deferRestore: deferRestore})
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
		w := b.renderDimensions().Width
		if w <= 0 {
			w = b.screen.Width()
		}
		if w <= 0 {
			w = 80
		}
		b.screen.HideCursor()
		b.resizeScreen(w, 0)
		b.renderScreen()
		if err := b.flushScreen(); err != nil {
			kitelog.Error("uv: flush error before inline insert", "error", err)
		}
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
		if size, ok := ev.(uv.WindowSizeEvent); ok && size.Width > 0 && size.Height > 0 {
			b.terminalSize.Store(&geom.Size{Width: size.Width, Height: size.Height})
		}
		if mr, ok := ev.(uv.ModeReportEvent); ok && mr.Mode == ansi.ModeUnicodeCore &&
			(mr.Value.IsSet() || mr.Value == ansi.ModeReset) {
			b.graphemeWidth.Store(true)
		}
		if kitelog.Enabled(slog.LevelDebug) {
			kitelog.Debug("UV: Event from terminal", "event", fmt.Sprintf("%#v", ev))
		}
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

func (b *Backend) renderDimensions() geom.Size {
	if b.renderSize.Width > 0 {
		return b.renderSize
	}
	return b.Size()
}

func (b *Backend) outputSize(fallback geom.Size) geom.Size {
	if size := b.terminalSize.Load(); size != nil {
		// The engine and input loop may still be processing older resize events.
		if b.terminal != nil {
			w, h, err := b.terminal.GetSize()
			if err != nil {
				kitelog.Warn("uv: terminal size query failed", "error", err)
			} else if w <= 0 || h <= 0 {
				kitelog.Warn("uv: terminal reported invalid dimensions", "width", w, "height", h)
			} else {
				current := &geom.Size{Width: w, Height: h}
				b.terminalSize.Store(current)
				return *current
			}
		}
		return *size
	}
	if size := b.requestedSize.Load(); size != nil {
		return *size
	}
	return fallback
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
		if b.inlineResized || b.screen.Width() != buf.Width || b.screen.Height() != buf.Height {
			// Keep the renderer's origin immediately after Static output.
			// Growth scrolls naturally; shrink erases below that same origin.
			b.resizeScreen(buf.Width, buf.Height)
		}
		b.inlineResized = false
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
			if !b.inline {
				b.screen.SetCursorPosition(cursor.X, cursor.Y)
			}
			b.screen.ShowCursor()
		} else {
			b.screen.HideCursor()
		}
	}

	b.renderScreen()
	if b.inline {
		if cursor.Visible {
			b.inlineRenderer.MoveTo(cursor.X, cursor.Y)
		} else {
			b.inlineRenderer.MoveTo(0, max(0, buf.Height-1))
		}
		if err := b.inlineRenderer.Flush(); err != nil {
			kitelog.Error("uv: inline cursor flush error", "error", err)
		}
	}
	if err := b.flushScreen(); err != nil {
		kitelog.Error("uv: flush error", "error", err)
	} else {
		kitelog.Debug("uv: screen.Flush called")
	}
	if b.inline {
		return
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
			kitelog.Debug("uv: screen.Flush called after cursor update")
		}
	}
}

func (b *Backend) resizeScreen(width, height int) {
	if b.inline && b.inlineRenderer != nil {
		size := b.renderDimensions()
		if b.inlineFrame.Width() > 0 && size.Width > 0 && size.Width != b.inlineFrame.Width() {
			x, y := b.inlineRenderer.Position()
			x, y = reflowCursor(b.inlineFrame, size.Width, x, y)
			b.inlineRenderer.SetPosition(x, y)
		}
		b.inlineRenderer.Resize(width, height)
		if b.inlineResized || b.inlineFrame.Width() != width {
			b.inlineRenderer.Erase()
		}
	}
	b.screen.Resize(width, height)
}

func (b *Backend) renderScreen() {
	if !b.inline {
		b.screen.Render()
		return
	}
	if b.inlineRenderer == nil {
		b.inlineRenderer = uv.NewTerminalRenderer(&b.inlineOutput, os.Environ())
		b.inlineRenderer.SetRelativeCursor(true)
		b.inlineRenderer.SetColorProfile(b.profile)
		b.inlineRenderer.Erase()
		b.inlineFrame = uv.NewRenderBuffer(0, 0)
	}
	width, height := b.screen.Width(), b.screen.Height()
	b.inlineFrame.Resize(width, height)
	b.inlineRenderer.SetGraphemeWidth(b.graphemeWidth.Load())
	for y := range height {
		for x := 0; x < width; {
			cell := b.screen.CellAt(x, y)
			b.inlineFrame.SetCell(x, y, cell)
			x += max(1, cell.Width)
		}
	}
	b.inlineRenderer.Render(b.inlineFrame)
	if err := b.inlineRenderer.Flush(); err != nil {
		kitelog.Error("uv: inline frame flush error", "error", err)
	}
}

func (b *Backend) flushScreen() error {
	if err := b.screen.Flush(); err != nil {
		return err
	}
	if !b.inline || b.inlineOutput.Len() == 0 {
		return nil
	}
	output := b.inlineOutput.Bytes()
	if bytes.Contains(output, []byte(ansi.EraseScreenBelow)) {
		// ED at the top-left can archive the live screen in tmux. Erase
		// lines without scrolling, restoring the cursor even when CUD clamps.
		erase := "\x1b7\x1b[K" +
			strings.Repeat("\x1b[B\r\x1b[K", max(0, b.renderDimensions().Height-1)) +
			"\x1b8"
		output = bytes.ReplaceAll(output, []byte(ansi.EraseScreenBelow), []byte(erase))
	}
	if b.terminal != nil {
		if b.screen.SynchronizedUpdates() {
			output = []byte(ansi.SetModeSynchronizedOutput + string(output) + ansi.ResetModeSynchronizedOutput)
		} else if b.screen.CursorVisible() {
			output = []byte(ansi.HideCursor + string(output) + ansi.ShowCursor)
		}
	}
	_, err := b.Write(output)
	b.inlineOutput.Reset()
	return err
}

// reflowCursor maps the renderer's cursor into the old frame after the terminal
// has rewrapped its hard lines. Cursor offsets must be recovered before erasing
// or the first wrapped rows become stranded above the live frame.
func reflowCursor(buf *uv.RenderBuffer, width, cursorX, cursorY int) (int, int) {
	if cursorX < 0 || cursorY < 0 || buf.Height() == 0 {
		return cursorX, cursorY
	}
	row := 0
	for y := 0; y <= min(cursorY, buf.Height()-1); y++ {
		end := buf.Width()
		for end > 0 {
			cell := buf.CellAt(end-1, y)
			if cell != nil && !cell.Equal(&uv.EmptyCell) {
				break
			}
			end--
		}
		if y == cursorY {
			if end <= width {
				return min(cursorX, width-1), row
			}
			end = min(cursorX, buf.Width())
		}
		column := 0
		for x := 0; x < end; {
			cell := buf.CellAt(x, y)
			span := 1
			if cell != nil {
				span = max(1, cell.Width)
			}
			if column+min(span, width) > width {
				column = 0
				row++
			}
			column += min(span, width)
			x += span
		}
		if y == cursorY {
			span := 1
			if cell := buf.CellAt(cursorX, y); cell != nil {
				span = max(1, cell.Width)
			}
			if column+min(span, width) > width {
				return 0, row + 1
			}
			return column, row
		}
		row++
	}
	return 0, row
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
	cell.Style.Underline = uv.UnderlineNone
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
	if c.Style&backend.CellUnderline != 0 {
		cell.Style.Underline = uv.UnderlineSingle
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
	if !kitelog.Enabled(slog.LevelDebug) {
		return
	}
	msg := fmt.Sprintf(format, args...)
	msg = strings.ReplaceAll(msg, "\x1b", "\\x1b")
	kitelog.Debug("UV TRACING: ", "message", msg)
}
