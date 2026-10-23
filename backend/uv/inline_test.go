package uv

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

// inlineTerminal models the cursor, screen, and scrollback affected by inline
// output. Checking the resulting rows catches errors that sequence matching
// alone misses, such as repainting a composer over a freshly inserted log.
type inlineTerminal struct {
	parser  *ansi.Parser
	width   int
	height  int
	x, y    int
	rows    [][]rune
	wrapped []bool
	history []string
	output  strings.Builder
	visible bool
	wrap    bool
	onWrite func([]byte)
	// tmux can archive the visible screen when ED starts at the top-left.
	archiveErase bool
	savedX       int
	savedY       int
}

func newInlineTerminal(t *testing.T, width, height int) *inlineTerminal {
	v := &inlineTerminal{width: width, height: height, wrap: true}
	v.rows = make([][]rune, height)
	v.wrapped = make([]bool, height)
	for y := range v.rows {
		v.rows[y] = make([]rune, width)
	}
	v.parser = ansi.NewParser()
	v.parser.SetHandler(ansi.Handler{
		Print: func(r rune) {
			if v.x >= v.width {
				if v.wrap {
					v.x = 0
					v.linefeed()
					v.wrapped[v.y] = true
				} else {
					v.x = v.width - 1
				}
			}
			v.rows[v.y][v.x] = r
			v.x++
		},
		Execute: func(b byte) {
			switch b {
			case '\r':
				v.x = 0
			case '\n':
				v.linefeed()
			case '\b':
				v.x = max(0, v.x-1)
			case '\t':
				v.x = min(v.width-1, (v.x/8+1)*8)
			default:
				t.Errorf("unsupported terminal control: %q", b)
			}
		},
		HandleCsi: func(cmd ansi.Cmd, params ansi.Params) {
			param := func(i, def int) int {
				n, _, _ := params.Param(i, def)
				return n
			}
			n := max(1, param(0, 1))
			switch cmd.Final() {
			case 'A':
				v.y = max(0, v.y-n)
			case 'B':
				v.y = min(v.height-1, v.y+n)
			case 'C':
				v.x = min(v.width-1, v.x+n)
			case 'D':
				v.x = max(0, v.x-n)
			case 'G':
				v.x = min(v.width-1, n-1)
			case 'H', 'f':
				v.y = min(v.height-1, n-1)
				v.x = min(v.width-1, max(1, param(1, 1))-1)
			case 'J':
				switch param(0, 0) {
				case 0:
					if v.archiveErase && v.x == 0 && v.y == 0 {
						last := v.height - 1
						for last >= 0 && v.line(last) == "" {
							last--
						}
						for y := 0; y <= last; y++ {
							v.history = append(v.history, v.line(y))
						}
					}
					clear(v.rows[v.y][min(v.x, v.width):])
					for y := v.y + 1; y < v.height; y++ {
						clear(v.rows[y])
					}
				case 2:
					for _, row := range v.rows {
						clear(row)
					}
				case 3:
					v.history = nil
				default:
					t.Errorf("unsupported erase display: %v", params)
				}
			case 'K':
				clear(v.rows[v.y][min(v.x, v.width):])
			case 'X':
				clear(v.rows[v.y][min(v.x, v.width):min(v.width, v.x+n)])
			case 'h', 'l':
				switch param(0, 0) {
				case 7:
					v.wrap = cmd.Final() == 'h'
				case 25:
					v.visible = cmd.Final() == 'h'
				case 9, 1000, 1002, 1003:
					// Mouse reporting does not affect terminal output.
				default:
					t.Errorf("unexpected terminal mode: %v", params)
				}
			case 'm', 'q':
				// Styling does not change cursor position or screen content.
			default:
				t.Errorf("unsupported CSI: %q %v", cmd.Final(), params)
			}
		},
		HandleEsc: func(cmd ansi.Cmd) {
			switch cmd.Final() {
			case '7':
				v.savedX, v.savedY = v.x, v.y
			case '8':
				v.x, v.y = v.savedX, v.savedY
			case 'M':
				if v.y > 0 {
					v.y--
				} else {
					copy(v.rows[1:], v.rows[:v.height-1])
					copy(v.wrapped[1:], v.wrapped[:v.height-1])
					v.rows[0] = make([]rune, v.width)
					v.wrapped[0] = false
				}
			default:
				t.Errorf("unsupported ESC: %q", cmd.Final())
			}
		},
	})
	return v
}

func (v *inlineTerminal) Write(p []byte) (int, error) {
	v.output.Write(p)
	v.parser.Parse(p)
	if v.onWrite != nil {
		v.onWrite(p)
	}
	return len(p), nil
}

func (v *inlineTerminal) linefeed() {
	if v.y < v.height-1 {
		v.y++
		v.wrapped[v.y] = false
		return
	}
	v.history = append(v.history, v.line(0))
	copy(v.rows, v.rows[1:])
	copy(v.wrapped, v.wrapped[1:])
	v.rows[v.height-1] = make([]rune, v.width)
	v.wrapped[v.height-1] = false
}

func (v *inlineTerminal) line(y int) string {
	return strings.TrimRight(strings.ReplaceAll(string(v.rows[y]), "\x00", " "), " ")
}

func (v *inlineTerminal) resize(width, height int) {
	// Keep the cursor visible when shrinking; a grow adds empty rows below.
	for v.y >= height {
		v.history = append(v.history, v.line(0))
		v.rows = v.rows[1:]
		v.wrapped = v.wrapped[1:]
		v.y--
	}
	rows := make([][]rune, height)
	wrapped := make([]bool, height)
	for y := range rows {
		rows[y] = make([]rune, width)
		if y < len(v.rows) {
			copy(rows[y], v.rows[y])
			wrapped[y] = v.wrapped[y]
		}
	}
	v.width, v.height, v.rows = width, height, rows
	v.wrapped = wrapped
	v.x = min(v.x, width-1)
}

// resizeReflow models terminals that rewrap primary-screen text, including the
// movement of an explicitly positioned cursor and joining soft-wrapped rows.
func (v *inlineTerminal) resizeReflow(width, height int) {
	var rows [][]rune
	var wrapped []bool
	cursorX, cursorY := 0, 0
	for y := 0; y < len(v.rows); {
		var line []rune
		cursorOffset := -1
		for {
			if y == v.y {
				cursorOffset = len(line) + min(v.x, v.width-1)
			}
			continues := y+1 < len(v.rows) && v.wrapped[y+1]
			n := len([]rune(v.line(y)))
			if continues {
				n = v.width
			}
			if y == v.y {
				n = max(n, min(v.x, v.width-1)+1)
			}
			line = append(line, v.rows[y][:n]...)
			y++
			if !continues {
				break
			}
		}
		if cursorOffset >= 0 {
			cursorX = cursorOffset % width
			cursorY = len(rows) + cursorOffset/width
		}
		for start := 0; start < max(1, len(line)); start += width {
			row := make([]rune, width)
			if start < len(line) {
				copy(row, line[start:min(start+width, len(line))])
			}
			rows = append(rows, row)
			wrapped = append(wrapped, start > 0)
		}
	}
	v.rows, v.wrapped = rows, wrapped
	v.width, v.height, v.x, v.y = width, height, cursorX, cursorY
	v.resize(width, height)
}

func inlineBuffer(width int, lines ...string) *backend.Buffer {
	buf := backend.NewBuffer(width, len(lines))
	for y, line := range lines {
		for x, ch := range line {
			buf.Set(x, y, backend.Cell{Content: string(ch), Width: 1})
		}
	}
	return buf
}

type inlineConsole struct {
	uv.Console
	out io.Writer
}

func (c inlineConsole) Writer() io.Writer           { return c.out }
func (c inlineConsole) Environ() []string           { return nil }
func (c inlineConsole) Write(p []byte) (int, error) { return c.out.Write(p) }
func (c inlineConsole) GetSize() (int, int, error) {
	if terminal, ok := c.out.(*inlineTerminal); ok {
		return terminal.width, terminal.height, nil
	}
	return 0, 0, fmt.Errorf("test console does not expose terminal dimensions")
}

func inlineBackend(v *inlineTerminal) *Backend {
	terminal := uv.NewTerminal(inlineConsole{out: v}, nil)
	return &Backend{
		terminal: terminal,
		screen:   terminal.Screen(),
		renderCh: make(chan renderRequest, 256),
		width:    v.width,
		height:   v.height,
		inline:   true,
	}
}

// Drain synchronously so each assertion observes the completed terminal output,
// with no sleeps or concurrent reads of the terminal model.
func drainInline(b *Backend) {
	b.started.Store(true)
	close(b.renderCh)
	b.renderWG.Add(1)
	b.renderLoop()
	b.renderCh = make(chan renderRequest, 256)
}

func queueInlineFrame(b *Backend, lines ...string) {
	b.current = inlineBuffer(b.width, lines...)
	b.cursorState = cursorRecord{Visible: true, X: 3, Y: max(0, len(lines)-2), Shape: backend.CursorBar}
	if err := b.EndFrame(); err != nil {
		panic(err)
	}
}

func assertInlineBottom(t *testing.T, v *inlineTerminal) {
	t.Helper()
	if v.line(v.height-2) != "Composer" || v.line(v.height-1) != "Status" {
		t.Fatalf("expected bottom composer/status, got %q / %q; output=%q",
			v.line(v.height-2), v.line(v.height-1), v.output.String())
	}
	if !v.visible || v.x != 3 || v.y != v.height-2 {
		t.Errorf("expected visible composer cursor (3,%d), got (%d,%d), visible=%t",
			v.height-2, v.x, v.y, v.visible)
	}
	var visible strings.Builder
	for y := range v.height {
		visible.WriteString(v.line(y))
		visible.WriteByte('\n')
	}
	if strings.Count(visible.String(), "Composer") != 1 || strings.Count(visible.String(), "Status") != 1 {
		t.Errorf("stale active rows left on screen: %q", visible.String())
	}
}

func assertInlineRows(t *testing.T, v *inlineTerminal, want []string) {
	t.Helper()
	all := append([]string{}, v.history...)
	for y := range v.height {
		all = append(all, v.line(y))
	}
	for len(all) > len(want) && all[len(all)-1] == "" {
		all = all[:len(all)-1]
	}
	if strings.Join(all, "\n") != strings.Join(want, "\n") {
		t.Fatalf("Static/live boundary drifted or live rows leaked:\nwant %q\ngot  %q", want, all)
	}
}

func TestInlineTail_ResizePreservesStaticAndCursor(t *testing.T) {
	v := newInlineTerminal(t, 20, 6)
	b := inlineBackend(v)
	logs := make([]string, 8)
	for i := range logs {
		logs[i] = fmt.Sprintf("log%02d", i)
		b.InsertAbove(inlineBuffer(20, logs[i]))
	}
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	assertInlineRows(t, v, append(append([]string{}, logs...), "Composer", "Status"))
	if !v.visible || v.x != 3 || v.y != len(logs)-len(v.history) {
		t.Fatalf("resize misplaced composer cursor: (%d,%d), visible=%t", v.x, v.y, v.visible)
	}

	for _, size := range []geom.Size{
		{Width: 20, Height: 10},
		{Width: 16, Height: 5},
		{Width: 24, Height: 12},
	} {
		v.resize(size.Width, size.Height)
		b.Resize(size)
		queueInlineFrame(b, "Composer", "Status")
		drainInline(b)
		assertInlineRows(t, v, append(append([]string{}, logs...), "Composer", "Status"))
		if !v.visible || v.x != 3 || v.y != len(logs)-len(v.history) {
			t.Fatalf("resize misplaced composer cursor: (%d,%d), visible=%t", v.x, v.y, v.visible)
		}
	}
	// New static output must scroll the active frame rather than be overwritten.
	b.InsertAbove(inlineBuffer(b.width, "last log"))
	drainInline(b)
	assertInlineRows(t, v, append(append([]string{}, logs...), "last log", "Composer", "Status"))
	// A terminal may clamp the hardware cursor before the frame's last row
	// fits. Repainting must not overwrite the newest static line.
	v.resize(v.width, 4)
	b.Resize(geom.Size{Width: v.width, Height: v.height})
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	assertInlineBottom(t, v)
	all := append([]string{}, v.history...)
	for y := range v.height {
		all = append(all, v.line(y))
	}
	text := strings.Join(all, "\n")
	for i := range 8 {
		if n := strings.Count(text, fmt.Sprintf("log%02d", i)); n != 1 {
			t.Errorf("static log %d appears %d times in %q", i, n, text)
		}
	}
	if strings.Count(text, "last log") != 1 {
		t.Errorf("new static log was lost or duplicated: %q", text)
	}
}

func TestInlineTail_WidthReflowRecoversFrameOrigin(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		for _, tail := range []bool{false, true} {
			t.Run(fmt.Sprintf("hidden=%t/tail=%t", hidden, tail), func(t *testing.T) {
				v := newInlineTerminal(t, 20, 8)
				b := inlineBackend(v)
				if tail {
					b.InsertAbove(inlineBuffer(20, "log0", "log1", "log2", "log3", "log4", "log5", "log6", "log7"))
				} else {
					b.InsertAbove(inlineBuffer(20, "Static header"))
				}
				paint := func(lines ...string) {
					b.current = inlineBuffer(b.width, lines...)
					b.cursorState = cursorRecord{Visible: !hidden, X: 3, Y: max(0, len(lines)-2), Shape: backend.CursorBar}
					if err := b.EndFrame(); err != nil {
						t.Fatal(err)
					}
					drainInline(b)
				}
				paint("ABCDEFGHIJKLMNOPQR", "Composer", "Status")
				paint("abcdefghijklmnopqr", "Composer", "Status")
				v.resizeReflow(10, 8)
				b.Resize(geom.Size{Width: 10, Height: 8})
				paint("abcdefghij", "klmnopqr", "Composer", "Status")

				top := 2 // The static header itself reflows onto two rows.
				if tail {
					top = v.height - 4
				}
				for i, want := range []string{"abcdefghij", "klmnopqr", "Composer", "Status"} {
					if got := v.line(top + i); got != want {
						t.Errorf("row %d: want %q, got %q; output=%q", top+i, want, got, v.output.String())
					}
				}
				all := append([]string{}, v.history...)
				for y := range v.height {
					all = append(all, v.line(y))
				}
				text := strings.Join(all, "\n")
				for _, live := range []string{"abcdefghij", "klmnopqr", "Composer", "Status"} {
					if n := strings.Count(text, live); n != 1 {
						t.Errorf("live row %q appears %d times: %q", live, n, text)
					}
				}
				if tail {
					for i := range 8 {
						if n := strings.Count(text, fmt.Sprintf("log%d", i)); n != 1 {
							t.Errorf("static log %d appears %d times: %q", i, n, text)
						}
					}
				} else if v.line(0) != "Static hea" || v.line(1) != "der" {
					t.Errorf("reflow lost static header: %q / %q", v.line(0), v.line(1))
				}
				if v.visible == hidden || (!hidden && (v.x != 3 || v.y != top+2)) {
					t.Errorf("incorrect cursor: (%d,%d), visible=%t", v.x, v.y, v.visible)
				}
			})
		}
	}
}

func TestInlineTail_WidthReflowBeforeStaticOrEmptyFrame(t *testing.T) {
	for _, insert := range []bool{false, true} {
		t.Run(fmt.Sprintf("insert=%t", insert), func(t *testing.T) {
			v := newInlineTerminal(t, 20, 8)
			b := inlineBackend(v)
			b.InsertAbove(inlineBuffer(20, "Header"))
			queueInlineFrame(b, "abcdefghijklmnopqr", "Composer", "Status")
			drainInline(b)
			v.resizeReflow(10, 8)
			b.Resize(geom.Size{Width: 10, Height: 8})
			if insert {
				b.InsertAboveForFrame(inlineBuffer(10, "New static"))
				queueInlineFrame(b, "Composer", "Status")
			} else {
				queueInlineFrame(b)
			}
			drainInline(b)
			if v.line(0) != "Header" {
				t.Errorf("lost previous header: %q", v.line(0))
			}
			want := []string{"", "", "", "", "", "", ""}
			if insert {
				want = []string{"New static", "Composer", "Status", "", "", "", ""}
			}
			for y, text := range want {
				if got := v.line(y + 1); got != text {
					t.Errorf("row %d: want %q, got %q", y+1, text, got)
				}
			}
			if len(v.history) != 0 {
				t.Errorf("resize leaked live output into scrollback: %q", v.history)
			}
		})
	}
}

func TestInlineTail_WidthReflowCursorInsideWrappedLine(t *testing.T) {
	v := newInlineTerminal(t, 20, 8)
	b := inlineBackend(v)
	b.InsertAbove(inlineBuffer(20, "Header"))
	b.current = inlineBuffer(20, "abcdefghijklmnopqr", "Status")
	b.cursorState = cursorRecord{Visible: true, X: 14, Y: 0}
	if err := b.EndFrame(); err != nil {
		t.Fatal(err)
	}
	drainInline(b)
	v.resizeReflow(10, 8)
	b.Resize(geom.Size{Width: 10, Height: 8})
	b.current = inlineBuffer(10, "abcdefghij", "klmnopqr", "Status")
	b.cursorState.X, b.cursorState.Y = 4, 1
	if err := b.EndFrame(); err != nil {
		t.Fatal(err)
	}
	drainInline(b)
	for y, want := range []string{"Header", "abcdefghij", "klmnopqr", "Status", ""} {
		if got := v.line(y); got != want {
			t.Errorf("row %d: want %q, got %q", y, want, got)
		}
	}
	if v.x != 4 || v.y != 2 || !v.visible {
		t.Errorf("incorrect cursor: (%d,%d), visible=%t", v.x, v.y, v.visible)
	}
}

func TestInlineTail_WidthReflowRoundTrip(t *testing.T) {
	v := newInlineTerminal(t, 20, 10)
	b := inlineBackend(v)
	b.InsertAbove(inlineBuffer(20, "Static header"))
	queueInlineFrame(b, "abcdefghijklmnopqr", "Composer", "Status")
	drainInline(b)
	for range 3 {
		v.resizeReflow(10, 10)
		b.Resize(geom.Size{Width: 10, Height: 10})
		queueInlineFrame(b, "abcdefghij", "klmnopqr", "Composer", "Status")
		drainInline(b)
		v.resizeReflow(20, 10)
		b.Resize(geom.Size{Width: 20, Height: 10})
		queueInlineFrame(b, "abcdefghijklmnopqr", "Composer", "Status")
		drainInline(b)
		for y, want := range []string{"Static header", "abcdefghijklmnopqr", "Composer", "Status", "", ""} {
			if got := v.line(y); got != want {
				t.Errorf("row %d: want %q, got %q", y, want, got)
			}
		}
		if v.x != 3 || v.y != 2 || len(v.history) != 0 {
			t.Errorf("resize changed cursor or scrollback: cursor=(%d,%d), history=%q", v.x, v.y, v.history)
		}
	}
}

func TestInlineTail_WidthReflowCoalescedResizes(t *testing.T) {
	v := newInlineTerminal(t, 20, 10)
	b := inlineBackend(v)
	b.InsertAbove(inlineBuffer(20, "Static header"))
	queueInlineFrame(b, "abcdefghijklmnopqr", "Composer", "Status")
	drainInline(b)
	for _, width := range []int{10, 20} {
		v.resizeReflow(width, 10)
		b.Resize(geom.Size{Width: width, Height: 10})
	}
	queueInlineFrame(b, "abcdefghijklmnopqr", "Composer", "Status")
	drainInline(b)
	for y, want := range []string{"Static header", "abcdefghijklmnopqr", "Composer", "Status", "", ""} {
		if got := v.line(y); got != want {
			t.Errorf("row %d: want %q, got %q", y, want, got)
		}
	}
	if v.x != 3 || v.y != 2 || len(v.history) != 0 {
		t.Errorf("resize changed cursor or scrollback: cursor=(%d,%d), history=%q", v.x, v.y, v.history)
	}
}

func TestInlineTail_ShrinkDiscardsQueuedOldSizeFrames(t *testing.T) {
	for _, size := range []geom.Size{
		{Width: 10, Height: 8},
		{Width: 20, Height: 6},
		{Width: 10, Height: 6},
	} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			v := newInlineTerminal(t, 20, 8)
			b := inlineBackend(v)
			logs := make([]string, 20)
			for i := range logs {
				logs[i] = fmt.Sprintf("log%02d", i)
			}
			b.InsertAbove(inlineBuffer(20, logs...))
			queueInlineFrame(b, "Initial", "Composer", "Status")
			drainInline(b)
			queueInlineFrame(b, "obsolete abcdefghijkl", "obsolete abcdefghijkl", "obsolete abcdefghijkl", "obsolete abcdefghijkl", "Composer", "Status")

			v.resizeReflow(size.Width, size.Height)
			b.Resize(size)
			queueInlineFrame(b, "Fresh", "Composer", "Status")
			v.output.Reset()
			drainInline(b)
			if strings.Contains(v.output.String(), "obsolete") {
				t.Errorf("painted an obsolete-size frame after shrinking: %q", v.output.String())
			}
			all := append([]string{}, v.history...)
			for y := range v.height {
				all = append(all, v.line(y))
			}
			want := strings.Join(append(logs, "Fresh", "Composer", "Status"), "\n")
			if got := strings.Join(all, "\n"); got != want {
				t.Errorf("shrink added gaps, leaked live rows, or lost static output:\nwant: %q\ngot:  %q", want, got)
			}
		})
	}
}

func TestInlineTail_PhysicalResizePrecedesEngineResize(t *testing.T) {
	v := newInlineTerminal(t, 20, 8)
	b := inlineBackend(v)
	queueInlineFrame(b, "Initial", "Composer", "Status")
	drainInline(b)
	b.terminalSize.Store(&geom.Size{Width: 20, Height: 8})
	v.resizeReflow(10, 6)
	// Both the cached observation and the engine request can lag the actual PTY.
	b.Resize(geom.Size{Width: 20, Height: 8})
	queueInlineFrame(b, "obsolete", "Composer", "Status")
	v.output.Reset()
	drainInline(b)
	if v.output.Len() != 0 {
		t.Fatalf("painted before the engine caught up with the physical resize: %q", v.output.String())
	}
	if b.renderSize != (geom.Size{Width: 10, Height: 6}) {
		t.Fatalf("render size followed the stale engine request: %v", b.renderSize)
	}
	b.Resize(geom.Size{Width: 10, Height: 6})
	queueInlineFrame(b, "Fresh", "Composer", "Status")
	drainInline(b)
	if v.line(0) != "Fresh" || v.line(1) != "Composer" || v.line(2) != "Status" {
		t.Errorf("fresh frame did not recover the original live region")
	}
}

func TestInlineTail_CoalescedClippingStillRepaints(t *testing.T) {
	v := newInlineTerminal(t, 20, 8)
	b := inlineBackend(v)
	queueInlineFrame(b, "abcdefghijklmnopqr", "Composer", "Status")
	drainInline(b)
	for _, width := range []int{10, 20} {
		v.resize(width, 8)
		b.Resize(geom.Size{Width: width, Height: 8})
	}
	queueInlineFrame(b, "abcdefghijklmnopqr", "Composer", "Status")
	drainInline(b)
	if v.line(0) != "abcdefghijklmnopqr" {
		t.Errorf("coalesced resizes failed to repaint clipped content: %q", v.line(0))
	}
}

func TestInlineTail_ShrinkPreservesQueuedStaticAndClear(t *testing.T) {
	for _, clear := range []bool{false, true} {
		t.Run(fmt.Sprintf("clear=%t", clear), func(t *testing.T) {
			v := newInlineTerminal(t, 20, 8)
			b := inlineBackend(v)
			b.InsertAbove(inlineBuffer(20, "Header"))
			queueInlineFrame(b, "Initial", "Composer", "Status")
			drainInline(b)
			queueInlineFrame(b, "obsolete", "Composer", "Status")
			if clear {
				b.Clear()
			}
			b.InsertAboveForFrame(inlineBuffer(20, "static after shrink"))
			v.resizeReflow(10, 6)
			b.Resize(geom.Size{Width: 10, Height: 6})
			queueInlineFrame(b, "Fresh", "Composer", "Status")
			v.output.Reset()
			drainInline(b)
			if strings.Contains(v.output.String(), "obsolete") {
				t.Errorf("painted stale live rows around static output: %q", v.output.String())
			}
			all := append([]string{}, v.history...)
			for y := range v.height {
				all = append(all, v.line(y))
			}
			want := []string{"static aft", "er shrink", "Fresh", "Composer", "Status"}
			if !clear {
				want = append([]string{"Header"}, want...)
			} else {
				want = append(want, "")
			}
			if strings.Join(all, "\n") != strings.Join(want, "\n") {
				t.Errorf("lost or reordered queued output: want %q, got %q", want, all)
			}
		})
	}
}

func TestInlineTail_ResizeDuringInsertDoesNotRestoreStaleFrame(t *testing.T) {
	v := newInlineTerminal(t, 20, 8)
	b := inlineBackend(v)
	b.terminalSize.Store(&geom.Size{Width: 20, Height: 8})
	b.InsertAbove(inlineBuffer(20, "one", "two", "three", "four", "five", "six", "seven", "eight"))
	queueInlineFrame(b, "cached live", "Composer", "Status")
	drainInline(b)
	offset := -1
	v.onWrite = func(p []byte) {
		if strings.Contains(string(p), "new static") {
			v.resizeReflow(10, 6)
			b.terminalSize.Store(&geom.Size{Width: 10, Height: 6})
			offset = v.output.Len()
		}
	}
	b.InsertAbove(inlineBuffer(20, "new static"))
	drainInline(b)
	v.onWrite = nil
	if offset < 0 {
		t.Fatal("resize was not triggered during the static write")
	}
	if output := v.output.String()[offset:]; strings.Contains(output, "cached") ||
		strings.Contains(output, "Composer") || strings.Contains(output, "Status") {
		t.Errorf("restored an old-size frame after the static write resized the terminal: %q", output)
	}
	b.Resize(geom.Size{Width: 10, Height: 6})
	queueInlineFrame(b, "Fresh", "Composer", "Status")
	drainInline(b)
	if v.line(3) != "Fresh" || v.line(4) != "Composer" || v.line(5) != "Status" {
		t.Errorf("fresh frame did not recover the live region after insertion")
	}
}

func TestInlineTail_FrameAndCursorUseSynchronizedOutput(t *testing.T) {
	var out bytes.Buffer
	terminal := uv.NewTerminal(inlineConsole{out: &out}, nil)
	b := &Backend{
		terminal: terminal,
		screen:   terminal.Screen(),
		renderCh: make(chan renderRequest, 256),
		width:    20,
		height:   8,
		inline:   true,
	}
	b.screen.SetSynchronizedUpdates(true)
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	output := out.String()
	start, end := "\x1b[?2026h", "\x1b[?2026l"
	var frame string
	for _, write := range strings.Split(output, end) {
		if strings.Contains(write, "Composer") {
			frame = write
		}
	}
	if !strings.HasPrefix(frame, start) || !strings.Contains(frame, "Status") ||
		!strings.HasSuffix(output, end) {
		t.Fatalf("frame and cursor should share one synchronized write: %q", output)
	}
	v := newInlineTerminal(t, 20, 8)
	v.parser.Parse([]byte(strings.TrimPrefix(frame, start)))
	if v.x != 3 || v.y != 0 {
		t.Errorf("cursor was not included in synchronized frame: (%d,%d)", v.x, v.y)
	}
}

func TestReflowCursor(t *testing.T) {
	for _, tc := range []struct {
		name         string
		width, x, y  int
		wantX, wantY int
	}{
		{name: "short rows", width: 10, x: 3, y: 0, wantX: 3, wantY: 0},
		{name: "wrapped row above", width: 6, x: 3, y: 2, wantX: 3, wantY: 3},
		{name: "wide cells", width: 5, x: 0, y: 1, wantX: 0, wantY: 3},
		{name: "cursor in wrapped row", width: 5, x: 8, y: 0, wantX: 0, wantY: 2},
		{name: "blank cursor padding", width: 10, x: 14, y: 1, wantX: 9, wantY: 1},
		{name: "unknown cursor", width: 10, x: -1, y: -1, wantX: -1, wantY: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf := uv.NewRenderBuffer(20, 3)
			for x := 0; x < 10; x += 2 {
				buf.SetCell(x, 0, &uv.Cell{Content: "\u754c", Width: 2})
			}
			buf.SetCell(0, 1, &uv.Cell{Content: "X", Width: 1})
			gotX, gotY := reflowCursor(buf, tc.width, tc.x, tc.y)
			if gotX != tc.wantX || gotY != tc.wantY {
				t.Errorf("want (%d,%d), got (%d,%d)", tc.wantX, tc.wantY, gotX, gotY)
			}
		})
	}
}

func TestInlineTail_NaturalUntilFilledAndClearResets(t *testing.T) {
	v := newInlineTerminal(t, 20, 6)
	b := inlineBackend(v)
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	if v.line(0) != "Composer" || v.line(1) != "Status" {
		t.Fatal("short initial UI should keep natural inline positioning")
	}
	v.resize(20, 10)
	b.Resize(geom.Size{Width: 20, Height: 10})
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	if v.line(0) != "Composer" || v.line(1) != "Status" {
		t.Fatal("resizing short output should not force it to the bottom")
	}
	b.InsertAbove(inlineBuffer(20, "one", "two", "three", "four", "five", "six", "seven", "eight"))
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	assertInlineBottom(t, v)

	b.Clear()
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	if v.line(0) != "Composer" || v.line(1) != "Status" || len(v.history) != 0 {
		t.Fatalf("clear should reset output and origin: history=%v", v.history)
	}
	if v.x != 3 || v.y != 0 {
		t.Errorf("cursor should be on row 0 after clear, got (%d,%d)", v.x, v.y)
	}
}

func TestInlineTail_OverflowThenShrinkLiveFrame(t *testing.T) {
	v := newInlineTerminal(t, 20, 6)
	b := inlineBackend(v)
	v.Write([]byte("shell\r\n"))
	queueInlineFrame(b, "live1", "live2", "live3", "live4", "Composer", "Status")
	drainInline(b)
	assertInlineBottom(t, v)
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	assertInlineRows(t, v, []string{"shell", "Composer", "Status"})
	if v.x != 3 || v.y != 0 {
		t.Errorf("shrink should preserve the live origin, got cursor (%d,%d)", v.x, v.y)
	}
	if len(v.history) == 0 || v.history[0] != "shell" {
		t.Errorf("first overflowing frame should scroll shell history, got %v", v.history)
	}
}
func TestInlineTail_LiveHeightChangesPreserveHistory(t *testing.T) {
	v := newInlineTerminal(t, 20, 6)
	b := inlineBackend(v)
	for i := range 8 {
		b.InsertAbove(inlineBuffer(20, fmt.Sprintf("log%02d", i)))
	}
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	queueInlineFrame(b, "extra1", "extra2", "Composer", "Status")
	drainInline(b)
	assertInlineBottom(t, v)
	queueInlineFrame(b)
	drainInline(b)
	b.InsertAbove(inlineBuffer(20, "after empty"))
	queueInlineFrame(b)
	drainInline(b)
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	assertInlineRows(t, v, []string{"log00", "log01", "log02", "log03", "log04", "log05", "log06", "log07", "after empty", "Composer", "Status"})
	all := append([]string{}, v.history...)
	for y := range v.height {
		all = append(all, v.line(y))
	}
	text := strings.Join(all, "\n")
	for i := range 8 {
		if strings.Count(text, fmt.Sprintf("log%02d", i)) != 1 {
			t.Errorf("lost or duplicated log %d after live frame height changes: %q", i, text)
		}
	}
	if strings.Count(text, "after empty") != 1 {
		t.Errorf("lost static output above zero-height frame: %q", text)
	}
}

func TestInlineRegion_RepeatedGrowthAndShrinkRemainAdjacent(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		t.Run(fmt.Sprintf("hidden=%t", hidden), func(t *testing.T) {
			v := newInlineTerminal(t, 40, 30)
			v.archiveErase = true
			b := inlineBackend(v)
			logs := make([]string, 40)
			for i := range logs {
				logs[i] = fmt.Sprintf("log%02d", i)
			}
			b.InsertAbove(inlineBuffer(40, logs...))
			paint := func(height int) {
				lines := make([]string, height)
				for i := range lines {
					lines[i] = fmt.Sprintf("live%02d", i)
				}
				b.current = inlineBuffer(b.width, lines...)
				b.cursorState = cursorRecord{Visible: !hidden, X: 3, Y: max(0, height-2)}
				if err := b.EndFrame(); err != nil {
					t.Fatal(err)
				}
				drainInline(b)
				want := append(append([]string{}, logs...), lines...)
				assertInlineRows(t, v, want)
				if !hidden && (v.x != 3 || v.y != len(logs)-len(v.history)+max(0, height-2)) {
					t.Fatalf("height %d: cursor moved away from live content: (%d,%d)", height, v.x, v.y)
				}
			}
			paint(10)
			for range 5 {
				paint(20)
				paint(10)
				paint(10)
			}
			for _, height := range []int{0, 1, 30, 10, 0, 20, 10} {
				paint(height)
			}
			for _, size := range []geom.Size{{Width: 20, Height: 40}, {Width: 40, Height: 25}} {
				v.resizeReflow(size.Width, size.Height)
				b.Resize(size)
				paint(10)
				paint(20)
				paint(10)
			}
			for _, deferred := range []bool{false, true} {
				paint(v.height)
				logs = append(logs, "new static")
				if deferred {
					b.InsertAboveForFrame(inlineBuffer(b.width, "new static"))
				} else {
					b.InsertAbove(inlineBuffer(b.width, "new static"))
				}
				drainInline(b)
				paint(10)
				paint(20)
				paint(10)
			}
		})
	}
}

func TestInlineRegion_SingleRowTerminal(t *testing.T) {
	v := newInlineTerminal(t, 20, 1)
	v.archiveErase = true
	b := inlineBackend(v)
	b.InsertAbove(inlineBuffer(20, "Static"))
	for _, lines := range [][]string{{"Live"}, {}, {"Updated"}, {}, {"Live"}} {
		queueInlineFrame(b, lines...)
		drainInline(b)
		assertInlineRows(t, v, append([]string{"Static"}, lines...))
	}
	b.InsertAbove(inlineBuffer(20, "New static"))
	drainInline(b)
	assertInlineRows(t, v, []string{"Static", "New static", "Live"})
}

func TestInlineRegion_PreservesPrecedingShellOutput(t *testing.T) {
	v := newInlineTerminal(t, 20, 6)
	b := inlineBackend(v)
	v.Write([]byte("shell1\r\nshell2\r\nshell3\r\n"))
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	b.InsertAbove(inlineBuffer(20, "log"))
	drainInline(b)
	v.resize(20, 10)
	b.Resize(geom.Size{Width: 20, Height: 10})
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	assertInlineRows(t, v, []string{"shell1", "shell2", "shell3", "log", "Composer", "Status"})
}

func TestInlineTail_ConcurrentResizeRequests(t *testing.T) {
	var out bytes.Buffer
	b := &Backend{
		screen:     uv.NewTerminalScreen(&out, nil),
		renderCh:   make(chan renderRequest, 256),
		width:      20,
		height:     6,
		inline:     true,
		renderSize: geom.Size{Width: 20, Height: 6},
	}
	b.started.Store(true)
	b.renderWG.Add(1)
	go b.renderLoop()
	for i := range 30 {
		b.Resize(geom.Size{Width: 20 + i%3, Height: 6 + i%4})
		if i%7 == 0 {
			b.Clear()
		}
		b.InsertAbove(inlineBuffer(b.width, fmt.Sprintf("log%02d", i)))
		queueInlineFrame(b, "Composer", "Status")
	}
	b.stopRenderLoop()
	if !strings.Contains(out.String(), "Composer") {
		t.Error("expected active frame after concurrent rendering and resize requests")
	}
}

func TestInlineTail_HiddenCursorAndSteadyRepaint(t *testing.T) {
	v := newInlineTerminal(t, 20, 6)
	b := inlineBackend(v)
	paint := func(text string) {
		b.current = inlineBuffer(b.width, text, "Status")
		if err := b.EndFrame(); err != nil {
			t.Fatal(err)
		}
		drainInline(b)
	}
	b.InsertAbove(inlineBuffer(20, "one", "two", "three", "four", "five", "six"))
	paint("Composer")
	v.resize(20, 10)
	b.Resize(geom.Size{Width: 20, Height: 10})
	paint("Composer")
	if v.visible || v.line(4) != "Composer" || v.line(5) != "Status" {
		t.Fatal("resize should preserve the live origin without showing a hidden cursor")
	}
	v.output.Reset()
	paint("Updated")
	if v.line(4) != "Updated" || v.line(5) != "Status" {
		t.Fatal("steady repaint changed the frame origin")
	}
	if strings.Contains(v.output.String(), "\x1b[J") || strings.Contains(v.output.String(), "\x1b[9;1H") {
		t.Errorf("steady frame unnecessarily cleared or repositioned output: %q", v.output.String())
	}
}

func TestInlineTail_StreamingBecomesStatic(t *testing.T) {
	v := newInlineTerminal(t, 20, 6)
	b := inlineBackend(v)
	queueInlineFrame(b, "stream0", "stream1", "stream2", "stream3", "Composer", "Status")
	drainInline(b)
	v.output.Reset()
	// The render loop may catch up between individual Static items while the
	// engine is still reconciling the live tree.
	for i := range 4 {
		b.InsertAboveForFrame(inlineBuffer(20, fmt.Sprintf("static%d", i)))
		drainInline(b)
	}
	if output := v.output.String(); strings.Contains(output, "stream") ||
		strings.Contains(output, "Composer") || strings.Contains(output, "Status") {
		t.Fatalf("backend repainted stale live content between static items: %q", output)
	}
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	assertInlineBottom(t, v)
	for y := range 4 {
		if v.line(y) != fmt.Sprintf("static%d", y) {
			t.Errorf("expected static item on visible row %d, got %q; history=%v", y, v.line(y), v.history)
		}
	}
	for _, line := range v.history {
		if strings.Contains(line, "stream") || strings.Contains(line, "Composer") || strings.Contains(line, "Status") {
			t.Errorf("live content leaked into permanent history: %q", line)
		}
	}
}

func TestInlineTail_FullWidthLiveRowsNeverBecomeStatic(t *testing.T) {
	v := newInlineTerminal(t, 20, 6)
	b := inlineBackend(v)
	full := strings.Repeat("X", 20)
	queueInlineFrame(b, full, full, full, full, "Composer", "Status")
	drainInline(b)
	for range 3 {
		b.InsertAboveForFrame(inlineBuffer(20, "static"))
		drainInline(b)
	}
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	for _, line := range v.history {
		if strings.Contains(line, "X") || strings.Contains(line, "Composer") || strings.Contains(line, "Status") {
			t.Errorf("live content leaked into permanent history: %q; output=%q", line, v.output.String())
		}
	}
}

type eagerInlineBackend struct{ *Backend }

func (b *eagerInlineBackend) InsertAbove(buf *backend.Buffer) {
	b.Backend.InsertAbove(buf)
	drainInline(b.Backend)
}

func (b *eagerInlineBackend) InsertAboveForFrame(buf *backend.Buffer) {
	b.Backend.InsertAboveForFrame(buf)
	drainInline(b.Backend)
}

// This test backend has no real terminal to restore.
func (b *eagerInlineBackend) Restore() {}

func TestInlineTail_EngineCommitsStaticWithFreshFrame(t *testing.T) {
	v := newInlineTerminal(t, 20, 6)
	b := &eagerInlineBackend{inlineBackend(v)}
	b.bufferPool.New = func() any { return backend.NewBuffer(20, 6) }
	e := engine.New(b, engine.Options{Inline: true})
	defer e.Stop()
	doc := e.Document()
	root := element.NewBox(doc)
	var streaming []*element.BoxElement
	for i := range 4 {
		row := element.NewBox(doc)
		row.Style(style.S().Height(style.Cells(1)))
		row.AppendChild(doc.CreateTextNode(fmt.Sprintf("stream%d", i), nil))
		root.AppendChild(row)
		streaming = append(streaming, row)
	}
	for _, text := range []string{"Composer", "Status"} {
		row := element.NewBox(doc)
		row.Style(style.S().Height(style.Cells(1)))
		row.AppendChild(doc.CreateTextNode(text, nil))
		root.AppendChild(row)
	}
	e.Mount(root)
	e.Frame()
	drainInline(b.Backend)
	e.Post(func() {
		for _, row := range streaming {
			root.RemoveChild(row)
		}
		for i := range 4 {
			row := element.NewBox(doc)
			row.Style(style.S().Height(style.Cells(1)))
			row.AppendChild(doc.CreateTextNode(fmt.Sprintf("static%d", i), nil))
			e.InsertAbove(row)
		}
	})
	e.Frame()
	drainInline(b.Backend)
	for y := range 4 {
		if v.line(y) != fmt.Sprintf("static%d", y) {
			t.Errorf("expected visible static item on row %d, got %q", y, v.line(y))
		}
	}
	if v.line(4) != "Composer" || v.line(5) != "Status" {
		t.Errorf("expected fresh two-row active frame, got %q / %q", v.line(4), v.line(5))
	}
	for _, line := range v.history {
		if strings.Contains(line, "stream") || strings.Contains(line, "Composer") || strings.Contains(line, "Status") {
			t.Errorf("stale live content was committed to history: %q", line)
		}
	}
}

func TestInlineRegion_EngineMenuGrowthAndShrink(t *testing.T) {
	v := newInlineTerminal(t, 40, 30)
	b := &eagerInlineBackend{inlineBackend(v)}
	b.bufferPool.New = func() any { return backend.NewBuffer(40, 30) }
	e := engine.New(b, engine.Options{Inline: true})
	defer e.Stop()
	doc := e.Document()
	root := element.NewBox(doc)
	rows := make([]*element.BoxElement, 20)
	lines := make([]string, len(rows))
	for i := range rows {
		lines[i] = fmt.Sprintf("live%02d", i)
		rows[i] = element.NewBox(doc)
		rows[i].Style(style.S().Height(style.Cells(1)))
		rows[i].AppendChild(doc.CreateTextNode(lines[i], nil))
		if i < 10 {
			root.AppendChild(rows[i])
		}
	}
	e.Mount(root)
	logs := make([]string, 40)
	e.Post(func() {
		for i := range logs {
			logs[i] = fmt.Sprintf("log%02d", i)
			row := element.NewBox(doc)
			row.AppendChild(doc.CreateTextNode(logs[i], nil))
			e.InsertAbove(row)
		}
	})
	e.Frame()
	drainInline(b.Backend)
	assertInlineRows(t, v, append(append([]string{}, logs...), lines[:10]...))
	for range 3 {
		e.Post(func() {
			for _, row := range rows[10:] {
				root.AppendChild(row)
			}
		})
		e.Frame()
		drainInline(b.Backend)
		assertInlineRows(t, v, append(append([]string{}, logs...), lines...))
		e.Post(func() {
			for _, row := range rows[10:] {
				root.RemoveChild(row)
			}
		})
		e.Frame()
		drainInline(b.Backend)
		assertInlineRows(t, v, append(append([]string{}, logs...), lines[:10]...))
	}
}

func TestInlineTail_DeferredInsertsRespectClear(t *testing.T) {
	v := newInlineTerminal(t, 20, 6)
	b := inlineBackend(v)
	queueInlineFrame(b, "live0", "live1", "live2", "live3", "Composer", "Status")
	drainInline(b)
	b.InsertAboveForFrame(inlineBuffer(20, "discarded"))
	b.Clear()
	b.InsertAboveForFrame(inlineBuffer(20, "Header"))
	queueInlineFrame(b, "Composer", "Status")
	drainInline(b)
	if len(v.history) != 0 || v.line(0) != "Header" || v.line(1) != "Composer" || v.line(2) != "Status" {
		t.Fatalf("expected fresh static header and live frame after clear; history=%v rows=%q/%q/%q",
			v.history, v.line(0), v.line(1), v.line(2))
	}
}
