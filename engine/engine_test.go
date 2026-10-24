package engine_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/terminal"
	"image/color"
)

// ---------------------------------------------------------------------------
// Fake helpers
// ---------------------------------------------------------------------------

// fakeClock is an injectable Clock for deterministic time tests.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{now: t} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	go func() {
		c.mu.Lock()
		target := c.now.Add(d)
		c.mu.Unlock()
		for {
			time.Sleep(time.Millisecond)
			c.mu.Lock()
			n := c.now
			c.mu.Unlock()
			if !n.Before(target) {
				ch <- n
				return
			}
		}
	}()
	return ch
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

// newTestEngine creates an Engine with a mock backend and fake layout engine
// for unit tests. Width and height default to 80x24.
func newTestEngine(t *testing.T, opts ...engine.Options) (*engine.Engine, *mock.Backend) {
	t.Helper()
	b := mock.New(80, 24)
	opt := engine.Options{}
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Clock == nil {
		opt.Clock = engine.RealClock()
	}
	e := engine.New(b, opt)
	return e, b
}

// newTestEngineWithCaps creates an Engine with specific backend capabilities.
func newTestEngineWithCaps(t *testing.T, caps backend.Caps) (*engine.Engine, *mock.Backend) {
	t.Helper()
	b := mock.NewWithCaps(80, 24, caps)
	e := engine.New(b, engine.Options{Clock: engine.RealClock()})
	return e, b
}

// ---------------------------------------------------------------------------
// Phase-gate tests
// ---------------------------------------------------------------------------

// TestEngine_Frame_NoOp_WhenClean verifies that a frame with no dirty bits
// does not trigger BeginFrame / EndFrame on the backend.
func TestEngine_Frame_NoOp_WhenClean(t *testing.T) {
	t.Parallel()

	e, b := newTestEngine(t)
	defer e.Stop()

	// Clear initial dirty bits from engine initialization (SetViewportSize).
	e.Frame()
	b.BeginFrameCalls = 0
	b.EndFrameCalls = 0

	e.Frame()

	if b.BeginFrameCalls != 0 {
		t.Errorf("BeginFrameCalls = %d, want 0 (no dirty bits)", b.BeginFrameCalls)
	}
	if b.EndFrameCalls != 0 {
		t.Errorf("EndFrameCalls = %d, want 0 (no dirty bits)", b.EndFrameCalls)
	}
}

// TestEngine_Frame_RunsAllPhases_WhenAllDirty verifies that a frame with all
// dirty bits set triggers all five phases.
func TestEngine_Frame_RunsAllPhases_WhenAllDirty(t *testing.T) {
	t.Parallel()

	e, b := newTestEngine(t)
	defer e.Stop()

	// Clear initial dirty flags.
	e.Frame()
	b.BeginFrameCalls = 0
	b.EndFrameCalls = 0

	root := e.RenderView()
	root.MarkDirty(render.DirtyStyle | render.DirtyLayout | render.DirtyPaint)

	e.Frame()

	if b.BeginFrameCalls != 1 {
		t.Errorf("BeginFrameCalls = %d, want 1", b.BeginFrameCalls)
	}
	if b.EndFrameCalls != 1 {
		t.Errorf("EndFrameCalls = %d, want 1", b.EndFrameCalls)
	}
	if root.Flags()&render.DirtyLayout != 0 {
		t.Error("layout engine did not clear DirtyLayout")
	}
}

// TestEngine_PhaseGate_StyleOnly verifies that only the style phase runs when
// only DirtyStyle is set.
func TestEngine_PhaseGate_StyleOnly(t *testing.T) {
	t.Parallel()

	e, b := newTestEngine(t)
	defer e.Stop()

	// Clear initial dirty flags (from SetViewportSize in newTestEngine).
	e.Frame()
	b.BeginFrameCalls = 0
	b.EndFrameCalls = 0

	root := e.RenderView()
	root.MarkDirty(render.DirtyStyle)

	e.Frame()

	// Style phase ran but no paint (no paint dirty).
	if b.BeginFrameCalls != 0 {
		t.Errorf("BeginFrameCalls = %d, want 0 (no paint dirty)", b.BeginFrameCalls)
	}
	// Layout Engine checks removed since LayoutNG runs implicitly through root node
}

// TestEngine_PhaseGate_LayoutOnly verifies that only the layout phase runs
// when only DirtyLayout is set.
func TestEngine_PhaseGate_LayoutOnly(t *testing.T) {
	t.Parallel()

	e, b := newTestEngine(t)
	defer e.Stop()

	// Clear initial dirty flags.
	e.Frame()
	b.BeginFrameCalls = 0
	b.EndFrameCalls = 0

	root := e.RenderView()
	root.MarkDirty(render.DirtyLayout)

	e.Frame()

	// Layout ran and triggered paint.
	if b.BeginFrameCalls != 1 {
		t.Errorf("BeginFrameCalls = %d, want 1 (layout triggers paint)", b.BeginFrameCalls)
	}
	if root.Flags()&render.DirtyLayout != 0 {
		t.Error("layout engine did not clear DirtyLayout")
	}
}

// TestEngine_PhaseGate_PaintOnly verifies that only the paint phase runs when
// only DirtyPaint is set.
func TestEngine_PhaseGate_PaintOnly(t *testing.T) {
	t.Parallel()

	e, b := newTestEngine(t)
	defer e.Stop()

	// Clear initial dirty flags.
	e.Frame()
	b.BeginFrameCalls = 0
	b.EndFrameCalls = 0

	root := e.RenderView()
	root.MarkDirty(render.DirtyPaint)

	e.Frame()

	// Paint phase ran.
	if b.BeginFrameCalls != 1 {
		t.Errorf("BeginFrameCalls = %d, want 1 (paint dirty)", b.BeginFrameCalls)
	}
}

// ---------------------------------------------------------------------------
// Frame loop / lifecycle tests
// ---------------------------------------------------------------------------

// TestEngine_Run_BlocksUntilStop verifies that Run blocks until Stop is called.
func TestEngine_Run_BlocksUntilStop(t *testing.T) {
	t.Parallel()

	e, _ := newTestEngine(t)

	done := make(chan struct{})
	go func() {
		err := e.Run(context.Background())
		if err != nil {
			t.Errorf("Run returned error: %v", err)
		}
		close(done)
	}()

	// Give it a moment to start.
	time.Sleep(20 * time.Millisecond)
	e.Stop()

	select {
	case <-done:
		// OK.
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not exit after Stop")
	}
}

// ---------------------------------------------------------------------------
// Macro / Microtask tests
// ---------------------------------------------------------------------------

// TestEngine_Post_RunsInMicrotaskPhase verifies that microtasks run during
// the frame.
func TestEngine_Post_RunsInMicrotaskPhase(t *testing.T) {
	t.Parallel()

	e, _ := newTestEngine(t)
	defer e.Stop()

	var run bool
	e.Post(func() { run = true })

	e.Frame()

	if !run {
		t.Error("microtask was not executed")
	}
}

// TestEngine_PostMacro_RespectsBudget verifies that macrotasks are capped
// by the count budget.
func TestEngine_PostMacro_RespectsBudget(t *testing.T) {
	t.Parallel()

	e, _ := newTestEngine(t, engine.Options{
		MacroTaskBudget: 2,
	})
	defer e.Stop()

	var count atomic.Int32
	for range 5 {
		e.PostMacro(func() { count.Add(1) })
	}

	e.Frame()

	if count.Load() != 2 {
		t.Errorf("executed %d macrotasks, want 2 (budget-capped)", count.Load())
	}

	e.Frame()

	if count.Load() != 4 {
		t.Errorf("after 2nd frame, executed %d total macrotasks, want 4", count.Load())
	}
}

// ---------------------------------------------------------------------------
// Worker pool tests
// ---------------------------------------------------------------------------

func TestEngine_Scheduler_RunBackground(t *testing.T) {
	t.Parallel()

	e, _ := newTestEngine(t)
	defer e.Stop()

	var runs atomic.Int32
	var completes atomic.Int32

	e.Scheduler().RunBackground(func(ctx context.Context) {
		runs.Add(1)
		e.Scheduler().QueueMicrotask(func() {
			completes.Add(1)
		})
	})

	// Wait for job to run and result to be posted.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if runs.Load() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if runs.Load() == 0 {
		t.Fatal("task did not run on worker")
	}

	// Result is posted as a microtask; must run a frame to execute completion.
	e.Frame()

	if completes.Load() == 0 {
		t.Error("completion microtask was not called")
	}
}

// ---------------------------------------------------------------------------
// Resize and Re-layout tests
// ---------------------------------------------------------------------------

func TestEngine_Resize(t *testing.T) {
	b := mock.New(80, 24)
	e := engine.New(b, engine.Options{})
	defer e.Stop()

	// 1. Initial Frame
	e.Frame()
	if e.RenderView().ViewportSize().Width != 80 {
		t.Errorf("initial viewport width = %d, want 80", e.RenderView().ViewportSize().Width)
	}
	if b.BeginFrameCalls != 1 {
		t.Errorf("initial BeginFrameCalls = %d, want 1", b.BeginFrameCalls)
	}

	// 2. Simulate Resize
	e.RenderView().SetViewportSize(geom.Size{Width: 120, Height: 40})

	// Check if dirty bits are set
	if e.RenderView().Flags()&render.DirtyLayout == 0 {
		t.Error("DirtyLayout not set after SetViewportSize")
	}
	if e.RenderView().Flags()&render.DirtyPaint == 0 {
		t.Error("DirtyPaint not set after SetViewportSize")
	}

	b.BeginFrameCalls = 0
	e.Frame()

	if e.RenderView().ViewportSize().Width != 120 {
		t.Errorf("after resize viewport width = %d, want 120", e.RenderView().ViewportSize().Width)
	}
	if b.BeginFrameCalls != 1 {
		t.Errorf("after resize BeginFrameCalls = %d, want 1", b.BeginFrameCalls)
	}
}

func TestEngine_ChildDirtyLayout_TriggersParentRelayout(t *testing.T) {
	b := mock.New(80, 24)
	e := engine.New(b, engine.Options{})
	defer e.Stop()

	doc := e.Document()
	root := e.RenderView()

	// Create a child
	child := element.NewBox(doc)
	doc.AppendChild(child)

	e.Frame() // Initial layout
	b.BeginFrameCalls = 0
	b.EndFrameCalls = 0

	// Get child's render object
	childRO := e.RenderObject(child)
	if childRO == nil {
		t.Fatal("child render object not found")
	}

	// Mark child dirty
	childRO.MarkDirty(render.DirtyLayout | render.DirtyPaint)

	// Parent (root) should now have ChildNeedsLayout
	if root.Flags()&render.ChildNeedsLayout == 0 {
		t.Error("root does not have ChildNeedsLayout after child MarkDirty")
	}

	e.Frame()

	if b.BeginFrameCalls != 1 {
		t.Errorf("BeginFrameCalls = %d, want 1 (child was dirty)", b.BeginFrameCalls)
	}
}

// ---------------------------------------------------------------------------
// Custom Render Object tests
// ---------------------------------------------------------------------------

type customNode struct {
	dom.Element
	created bool
}

func (c *customNode) CreateRenderObject() render.Object {
	c.created = true
	return render.NewBox(c, c)
}

func (c *customNode) Unwrap() dom.Node {
	return c.Element
}

func TestEngine_CustomRenderObject(t *testing.T) {
	e, _ := newTestEngine(t)
	defer e.Stop()

	doc := e.Document()
	cn := &customNode{}
	cn.Element = dom.NewElement(doc, "custom", cn)
	doc.AppendChild(cn)

	e.Frame()

	if !cn.created {
		t.Error("CustomObjectProvider.CreateRenderObject was not called")
	}

	if e.RenderObject(cn) == nil {
		t.Error("custom node has no render object")
	}
}

type wrappedNode struct {
	dom.Element
	inner dom.Node
}

func (w *wrappedNode) Unwrap() dom.Node {
	return w.inner
}

func TestEngine_CustomRenderObject_Wrapped(t *testing.T) {
	e, _ := newTestEngine(t)
	defer e.Stop()

	doc := e.Document()
	cn := &customNode{}
	cn.Element = dom.NewElement(doc, "inner", cn)

	wn := &wrappedNode{inner: cn}
	wn.Element = dom.NewElement(doc, "wrapper", wn)

	doc.AppendChild(wn)

	e.Frame()

	if !cn.created {
		t.Error("CustomObjectProvider.CreateRenderObject was not called on inner node")
	}

	if e.RenderObject(wn) == nil {
		t.Error("wrapped node has no render object")
	}
}

func TestEngine_SetTimeout(t *testing.T) {
	e, _ := newTestEngine(t)
	defer e.Stop()

	var called atomic.Bool
	timer := e.SetTimeout(func() {
		called.Store(true)
	}, 10*time.Millisecond)
	defer timer.Stop()

	// Initially not called
	e.Frame()
	if called.Load() {
		t.Error("SetTimeout called prematurely")
	}

	// Wait for the timeout to elapse
	time.Sleep(30 * time.Millisecond)

	// Since callback is queued as a microtask, we need to run a frame to execute it
	e.Frame()
	if !called.Load() {
		t.Error("SetTimeout callback was not executed after delay")
	}
}

func TestEngine_SetTimeout_Cancel(t *testing.T) {
	e, _ := newTestEngine(t)
	defer e.Stop()

	var called atomic.Bool
	timer := e.SetTimeout(func() {
		called.Store(true)
	}, 20*time.Millisecond)

	// Stop it immediately
	timer.Stop()

	// Wait for delay to pass
	time.Sleep(40 * time.Millisecond)

	e.Frame()
	if called.Load() {
		t.Error("SetTimeout callback was executed despite being stopped")
	}
}

func TestEngine_SetInterval(t *testing.T) {
	e, _ := newTestEngine(t)
	defer e.Stop()

	var count atomic.Int32
	timer := e.SetInterval(func() {
		count.Add(1)
	}, 10*time.Millisecond)

	// Wait for a few intervals
	time.Sleep(35 * time.Millisecond)

	e.Frame()
	c := count.Load()
	if c < 2 {
		t.Errorf("expected at least 2 ticks, got %d", c)
	}

	// Stop the interval
	timer.Stop()
	time.Sleep(20 * time.Millisecond)

	e.Frame()
	c2 := count.Load()
	if c2 != c {
		t.Errorf("count changed after Stop: was %d, now %d", c, c2)
	}
}

func TestEngine_DynamicStyleChange(t *testing.T) {
	e, b := newTestEngine(t)
	defer e.Stop()

	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	box := element.NewBox(e.Document())
	box.Style(style.S().Foreground(red))

	e.Document().AppendChild(box)

	e.Frame() // Initial frame should paint

	// Verify initial computed style
	ro := e.RenderObject(box)
	if ro == nil {
		t.Fatalf("expected render object for box")
	}
	if ro.ComputedStyle().Foreground != red {
		t.Fatalf("expected foreground to be red, got %v", ro.ComputedStyle().Foreground)
	}

	b.BeginFrameCalls = 0
	b.EndFrameCalls = 0

	// Now update the style to blue
	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	box.Style(style.S().Foreground(blue))

	e.Frame() // Second frame

	// Verify updated computed style
	if ro.ComputedStyle().Foreground != blue {
		t.Errorf("expected foreground to be updated to blue, got %v", ro.ComputedStyle().Foreground)
	}

	// Verify that it repainted!
	if b.BeginFrameCalls != 1 {
		t.Errorf("expected BeginFrameCalls = 1 (repaint on style change), got %d", b.BeginFrameCalls)
	}
}

func TestTerminalProxy_SetTitleAndBell(t *testing.T) {
	e, b := newTestEngineWithCaps(t, backend.Caps{Title: true, Bell: true, ProgressBar: true})
	defer e.Stop()

	term := e.Document().Terminal()
	if term == nil {
		t.Fatalf("expected terminal object on document")
	}

	// 1. Test SetTitle
	term.SetTitle("Hello World")
	if !strings.Contains(b.Output.String(), "\x1b]2;Hello World\x07") {
		t.Errorf("expected SetTitle sequence in output, got %q", b.Output.String())
	}

	// Reset output buffer
	b.Output.Reset()

	// 2. Test Bell
	term.Bell()
	if !strings.Contains(b.Output.String(), "\x07") {
		t.Errorf("expected Bell sequence in output, got %q", b.Output.String())
	}

	// Reset output buffer
	b.Output.Reset()

	// 3. Test SetProgressBar
	term.SetProgressBar(terminal.ProgressBarNormal, 75)
	if !strings.Contains(b.Output.String(), "\x1b]9;4;1;75\x07") {
		t.Errorf("expected SetProgressBar sequence in output, got %q", b.Output.String())
	}

	// Reset output buffer
	b.Output.Reset()

	// 4. Test SetProgramStatus
	e.SetProgramStatus(&terminal.ProgramStatus{
		State: terminal.ProgramStateWorking,
	})
	// Initially Caps.ProgramStatus is false in this test
	if b.ProgramStatus != nil {
		t.Errorf("expected SetProgramStatus to be no-op when caps disabled, got %v", b.ProgramStatus)
	}
}

func TestEngine_ProgramStatusSupportEvent(t *testing.T) {
	e, b := newTestEngine(t)
	defer e.Stop()

	if e.Caps().ProgramStatus {
		t.Errorf("expected initial Caps.ProgramStatus to be false")
	}

	received := false
	e.Document().AddEventListener(event.EventProgramStatusSupport, func(ev event.Event) {
		if ev.Type() == event.EventProgramStatusSupport {
			received = true
		}
	})

	e.ProcessRawEvent(&backend.RawProgramStatusSupportEvent{})

	if !received {
		t.Errorf("expected EventProgramStatusSupport to be dispatched to document")
	}

	if !e.Caps().ProgramStatus {
		t.Errorf("expected Caps.ProgramStatus to be true after support event")
	}

	// Now SetProgramStatus should work through TerminalProxy
	term := e.Document().Terminal()
	term.SetProgramStatus(&terminal.ProgramStatus{
		State: terminal.ProgramStateWorking,
		Title: "Building...",
	})

	if b.ProgramStatus == nil || b.ProgramStatus.State != terminal.ProgramStateWorking {
		t.Errorf("expected ProgramStatus to be set on backend, got %v", b.ProgramStatus)
	}

	if !strings.Contains(b.Output.String(), "\x1b]7501;working\x1b\\") {
		t.Errorf("expected OSC 7501 sequence in output, got %q", b.Output.String())
	}

	// Clear program status
	term.SetProgramStatus(nil)
	if b.ProgramStatus != nil {
		t.Errorf("expected ProgramStatus to be nil after clear, got %v", b.ProgramStatus)
	}
}

func TestEngine_InlineMode(t *testing.T) {
	b := mock.New(80, 24)
	e := engine.New(b, engine.Options{Inline: true})
	defer e.Stop()

	if !e.IsInline() {
		t.Fatalf("expected e.IsInline() to be true")
	}
	if !b.Inline {
		t.Fatalf("expected b.Inline to be true")
	}

	box1 := element.NewBox(e.Document())
	box1.Style(style.S().Height(style.Cells(1)))
	box2 := element.NewBox(e.Document())
	box2.Style(style.S().Height(style.Cells(1)))
	container := element.NewBox(e.Document())
	container.AppendChild(box1)
	container.AppendChild(box2)
	e.Mount(container)

	e.Frame()

	frame1 := b.LastFrame().Surface
	if frame1 == nil {
		t.Fatalf("expected a rendered frame")
	}
	if frame1.Height != 2 {
		t.Errorf("expected 2-row frame, got height %d", frame1.Height)
	}
	if frame1.Width != 80 {
		t.Errorf("expected width 80, got width %d", frame1.Width)
	}

	// Dynamically grow to 4 rows
	box3 := element.NewBox(e.Document())
	box3.Style(style.S().Height(style.Cells(1)))
	box4 := element.NewBox(e.Document())
	box4.Style(style.S().Height(style.Cells(1)))
	container.AppendChild(box3)
	container.AppendChild(box4)

	e.Frame()

	frame2 := b.LastFrame().Surface
	if frame2.Height != 4 {
		t.Errorf("expected 4-row frame after growing, got height %d", frame2.Height)
	}

	// Dynamically shrink back to 2 rows
	container.RemoveChild(box3)
	container.RemoveChild(box4)

	e.Frame()

	frame3 := b.LastFrame().Surface
	if frame3.Height != 2 {
		t.Errorf("expected 2-row frame after shrinking, got height %d", frame3.Height)
	}

	// Shrink to 0 rows
	container.RemoveChild(box1)
	container.RemoveChild(box2)

	e.Frame()

	frame4 := b.LastFrame().Surface
	if frame4.Height != 0 {
		t.Errorf("expected 0-row frame after removing all, got height %d", frame4.Height)
	}
}

func TestEngine_Clear_RepaintsMountedUI(t *testing.T) {
	for _, inline := range []bool{false, true} {
		t.Run(fmt.Sprintf("inline=%t", inline), func(t *testing.T) {
			e, b := newTestEngine(t, engine.Options{Inline: inline})
			defer e.Stop()

			doc := e.Document()
			box := element.NewBox(doc)
			box.Style(style.S().Height(style.Cells(2)))
			box.AppendChild(doc.CreateTextNode("Composer", nil))
			e.Mount(box)
			e.Frame()
			initial := b.LastFrame().Surface
			e.Frame()
			if b.EndFrameCalls != 1 {
				t.Fatal("expected clean UI not to repaint")
			}

			e.Clear()

			if b.ClearCalls != 1 {
				t.Fatalf("expected one backend clear, got %d", b.ClearCalls)
			}
			if e.RenderView().Flags()&render.DirtyPaint == 0 {
				t.Error("expected Clear to mark render view paint-dirty")
			}
			e.Frame()
			if b.EndFrameCalls != 2 {
				t.Fatalf("expected Clear to force repaint, got %d frames", b.EndFrameCalls)
			}
			frame := b.LastFrame().Surface
			if frame.Height != initial.Height || frame.CellAt(0, 0).Content != "C" {
				t.Errorf("expected mounted UI repainted at row 0, got %#v", frame)
			}
		})
	}
}

func TestEngine_InsertAbove(t *testing.T) {
	b := mock.New(80, 24)
	e := engine.New(b, engine.Options{Inline: true})
	defer e.Stop()

	doc := dom.NewDocument()
	el := element.NewBox(doc)
	el.Style(style.S().Width(style.Cells(10)).Height(style.Cells(3)).Border(style.SingleBorder()))
	txt := doc.CreateTextNode("Hi", nil)
	el.AppendChild(txt)

	e.InsertAbove(el)

	if len(b.InsertedAbove) != 1 {
		t.Fatalf("expected 1 buffer inserted above, got %d", len(b.InsertedAbove))
	}

	buf := b.InsertedAbove[0]
	if buf.Height != 3 {
		t.Errorf("expected inserted buffer height 3, got %d", buf.Height)
	}
	if buf.Width != 80 {
		t.Errorf("expected inserted buffer width 80, got %d", buf.Width)
	}

	// Check top-left corner has border
	c00 := buf.CellAt(0, 0)
	if c00.Content != "┌" {
		t.Errorf("expected top-left border ┌, got %q", c00.Content)
	}

	// Check text "H" at (1, 1) and "i" at (2, 1)
	c11 := buf.CellAt(1, 1)
	if c11.Content != "H" {
		t.Errorf("expected 'H' at (1,1), got %q", c11.Content)
	}
	c21 := buf.CellAt(2, 1)
	if c21.Content != "i" {
		t.Errorf("expected 'i' at (2,1), got %q", c21.Content)
	}
}

func TestEngine_InsertAbove_RepaintsInlineActiveFrameEvenWhenDOMUnchanged(t *testing.T) {
	e, b := newTestEngine(t, engine.Options{Inline: true})
	defer e.Stop()

	doc := e.Document()
	box := element.NewBox(doc)
	box.Style(style.S().Width(style.Cells(30)).Height(style.Cells(2)))
	txt := doc.CreateTextNode("Active inline UI", nil)
	box.AppendChild(txt)
	e.Mount(box)

	// First frame mounts and paints the active inline UI.
	e.Frame()

	if b.BeginFrameCalls != 1 || b.EndFrameCalls != 1 {
		t.Fatalf("expected 1 initial frame, got Begin=%d End=%d", b.BeginFrameCalls, b.EndFrameCalls)
	}
	lastFrame := b.LastFrame().Surface
	if lastFrame == nil || lastFrame.Height != 2 {
		t.Fatalf("expected 2-row initial frame, got %#v", lastFrame)
	}
	if lastFrame.CellAt(0, 0).Content != "A" {
		t.Errorf("expected 'A' at (0, 0), got %q", lastFrame.CellAt(0, 0).Content)
	}

	// Reset frame counters
	b.BeginFrameCalls = 0
	b.EndFrameCalls = 0

	// A clean frame without any DOM or dirty changes should NOT repaint.
	e.Frame()
	if b.BeginFrameCalls != 0 || b.EndFrameCalls != 0 {
		t.Fatalf("expected 0 frames when clean, got Begin=%d End=%d", b.BeginFrameCalls, b.EndFrameCalls)
	}

	// Insert static content above using a detached DOM node.
	staticDoc := dom.NewDocument()
	staticEl := element.NewBox(staticDoc)
	staticEl.Style(style.S().Width(style.Cells(20)).Height(style.Cells(1)))
	staticEl.AppendChild(staticDoc.CreateTextNode("Static item", nil))

	e.InsertAbove(staticEl)

	if len(b.InsertedAbove) != 1 {
		t.Fatalf("expected 1 buffer in InsertedAbove, got %d", len(b.InsertedAbove))
	}
	if b.InsertedAbove[0].CellAt(0, 0).Content != "S" {
		t.Errorf("expected 'S' in static buffer, got %q", b.InsertedAbove[0].CellAt(0, 0).Content)
	}

	// Calling InsertAbove should have marked renderView paint-dirty.
	if e.RenderView().Flags()&render.DirtyPaint == 0 {
		t.Errorf("expected RenderView to have DirtyPaint flag set after InsertAbove")
	}

	// Even though the active DOM tree was NOT modified, the next Frame() must repaint
	// the active inline content.
	e.Frame()

	if b.BeginFrameCalls != 1 || b.EndFrameCalls != 1 {
		t.Fatalf("expected active frame to be repainted after InsertAbove, got Begin=%d End=%d", b.BeginFrameCalls, b.EndFrameCalls)
	}
	repaintedFrame := b.LastFrame().Surface
	if repaintedFrame == nil || repaintedFrame.Height != 2 {
		t.Fatalf("expected repainted frame height 2, got %#v", repaintedFrame)
	}
	if repaintedFrame.CellAt(0, 0).Content != "A" {
		t.Errorf("expected 'A' at (0, 0) in repainted frame, got %q", repaintedFrame.CellAt(0, 0).Content)
	}
}

func TestEngine_InlineMode_OverflowClampsAndBottomAnchors(t *testing.T) {
	b := mock.New(80, 20)
	e := engine.New(b, engine.Options{Inline: true})
	defer e.Stop()

	doc := e.Document()
	container := element.NewBox(doc)

	// 50 lines of streaming text
	for i := range 50 {
		line := element.NewBox(doc)
		line.Style(style.S().Height(style.Cells(1)))
		line.AppendChild(doc.CreateTextNode(fmt.Sprintf("streaming line %02d", i), nil))
		container.AppendChild(line)
	}

	// Bottom input composer (height 1)
	input := element.NewInput(doc, "composer text")
	input.Style(style.S().Height(style.Cells(1)).Width(style.Cells(30)))
	container.AppendChild(input)

	// Bottom status bar (height 1)
	statusBar := element.NewBox(doc)
	statusBar.Style(style.S().Height(style.Cells(1)))
	statusBar.AppendChild(doc.CreateTextNode("Status: Running", nil))
	container.AppendChild(statusBar)

	e.Mount(container)
	input.Focus()

	e.Frame()

	// 1. The painted frame buffer height is clamped to viewport.Height (20)
	lastFrame := b.LastFrame().Surface
	if lastFrame == nil {
		t.Fatalf("expected a rendered frame")
	}
	if lastFrame.Height != 20 {
		t.Fatalf("expected painted frame buffer height clamped to 20, got %d", lastFrame.Height)
	}

	// 2. The frame shows the bottom 20 rows (lines 32..49 + input composer + status bar)
	getLine := func(buf *backend.Buffer, y int) string {
		var sb strings.Builder
		for x := 0; x < buf.Width; x++ {
			sb.WriteString(buf.CellAt(x, y).Content)
		}
		return strings.TrimRight(sb.String(), " ")
	}

	// Row 0 of the clamped viewport should show line 32 (tail of streaming text, NOT line 00)
	if !strings.HasPrefix(getLine(lastFrame, 0), "streaming line 32") {
		t.Errorf("expected row 0 to show 'streaming line 32', got %q", getLine(lastFrame, 0))
	}
	// Row 18 shows the input composer
	if !strings.Contains(getLine(lastFrame, 18), "composer text") {
		t.Errorf("expected row 18 to contain 'composer text', got %q", getLine(lastFrame, 18))
	}
	// Row 19 shows the status bar
	if !strings.Contains(getLine(lastFrame, 19), "Status: Running") {
		t.Errorf("expected row 19 to contain 'Status: Running', got %q", getLine(lastFrame, 19))
	}

	// 3. Hardware cursor for the focused input:
	// It should be visible and positioned at row 18 within the 20-row viewport.
	if !b.Cursor.Visible {
		t.Errorf("expected hardware cursor to be visible")
	}
	if b.Cursor.Y != 18 {
		t.Errorf("expected hardware cursor Y = 18, got %d", b.Cursor.Y)
	}
}
