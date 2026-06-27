package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/masterkeysrd/kite/animation"
	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/internal/collections"
	internaldom "github.com/masterkeysrd/kite/internal/dom"
	internalevent "github.com/masterkeysrd/kite/internal/event"
	"github.com/masterkeysrd/kite/internal/focus"
	"github.com/masterkeysrd/kite/promise"
	"github.com/masterkeysrd/kite/terminal"

	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/focus/spatial"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/paint"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/internal/styler"
	kitelog "github.com/masterkeysrd/kite/log"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/trace"
)

var eventsCounter uint64
var batchCounter uint64

// DefaultWorkers is the number of worker goroutines in the pool when not
// configured explicitly.
const DefaultWorkers = 4

// DefaultMacroTaskBudget is the maximum number of macrotasks drained per
// frame.
const DefaultMacroTaskBudget = 1024

// DefaultMacroTaskDuration is the wall-clock budget for the macrotask drain
// phase per frame.
const DefaultMacroTaskDuration = 4 * time.Millisecond

// MinFrameInterval is the minimum elapsed time between two consecutive frames.
// Frame requests within this window coalesce into a single frame.
const MinFrameInterval = 8 * time.Millisecond

// MouseMode configures what level of mouse event the backend enables.
type MouseMode uint8

const (
	// MouseModeOff disables all mouse event reporting.
	MouseModeOff MouseMode = iota
	// MouseModeClick enables button-press event only (default).
	MouseModeClick
	// MouseModeDrag enables drag event (motion while button held).
	MouseModeDrag
	// MouseModeTrack enables full motion tracking including hover event.
	MouseModeTrack
)

type syncTask struct {
	node dom.Node
	ro   render.Object
}

// Engine orchestrates the five-phase frame pipeline (Tasks → Style → Layout →
// Paint → Sync), the worker pool, and the macrotask / microtask queues.
//
// Engine must be used from a single main goroutine for all render-tree
// operations; the worker pool runs concurrent goroutines for user-submitted
// task only.
type Engine struct {
	document   dom.Document
	renderView *render.RenderView

	// resolver drives the Style phase.
	resolver *styler.Resolver

	// paintEngine drives the Paint pipeline.
	paintEngine *paint.PaintEngine

	// frameBuffer is the internal drawing surface used by the paintEngine.
	// Its contents are copied to the backend.Surface at the end of each frame.
	frameBuffer *paint.FrameBuffer

	// dispatcher performs 3-phase event dispatch.
	dispatcher event.Dispatcher

	// synthesizer converts raw backend input into structured events.
	synthesizer *internalevent.Synthesizer

	// focusManager owns focus state and scope stack.
	focusManager *focus.Manager

	// backend is the output target.
	backend backend.Backend

	// scheduler manages background tasks and task queues.
	scheduler *defaultScheduler

	// clock provides injectable time.
	clock Clock

	// cursor holds the engine-side cursor model.
	cursor cursorState

	// clipboard manages pending asynchronous clipboard operations.
	clipboard clipboardState

	// mouseMode is the active mouse-event level.
	mouseMode MouseMode

	// caps is the terminal capability snapshot.
	caps backend.Caps

	// frameVersion is a monotonically-increasing counter incremented each
	// time a frame is committed.
	frameVersion uint64

	// frameRequested is set when RequestFrame / RequestFrameAt has been
	// called since the last frame was committed.
	frameRequested bool

	// nextFrameAt is the scheduled time for the next frame (from
	// RequestFrameAt). Zero means "as soon as possible".
	nextFrameAt time.Time

	// macroTaskBudget caps macrotasks per frame (count).
	macroTaskBudget int

	// shutdownTimeout bounds how long Stop waits for pending jobs.
	shutdownTimeout time.Duration

	// lastCursorState tracks the hardware cursor state from the previous frame.
	lastCursorState cursorRecord

	// lastHardwareFocus tracks the focused element from the previous frame.
	lastHardwareFocus event.EventTarget

	// onFrameRenderedHooks are called after every frame is committed.
	onFrameRenderedHooks []func()

	// onStopHooks are called when the engine is stopping.
	onStopHooks []func()

	// eventBuffer holds raw events to be processed before the next frame.
	eventBuffer []backend.RawEvent

	// Reusable buffers for event coalescing to avoid per-frame allocations.
	rawCoalescedBuf []backend.RawEvent
	structuredBuf   []event.Event
	coalescedBuf    []event.Event
	wheelMap        map[event.EventTarget]*event.WheelEvent

	profilerMu sync.RWMutex

	pipeline Pipeline

	tracer *trace.Tracer

	activeAnimations []animation.Animator
	lastFrameTime    time.Time

	renderMap      map[dom.Node]render.Object
	isLayoutActive bool // Prevents infinite recursion during synchronous reflow

	syncStack []syncTask
	diffMap   map[render.Object]struct{}

	closeOnce sync.Once // protects Stop
	closeCh   chan struct{}
}

// Options configures an Engine.
type Options struct {
	// NumWorkers is the size of the worker pool (default DefaultWorkers).
	NumWorkers int
	// MacroTaskBudget caps macrotasks per frame by count (default DefaultMacroTaskBudget).
	MacroTaskBudget int
	// MacroTaskDuration caps macrotasks per frame by wall clock (default DefaultMacroTaskDuration).
	MacroTaskDuration time.Duration
	// Clock is the injectable time source. Defaults to RealClock().
	Clock Clock
	// ShutdownTimeout bounds how long Stop waits for pending jobs before
	// forcing exit. Default is 5 seconds.
	ShutdownTimeout time.Duration
	// Profiler enables the high-level phase profiling and deep-tree tracing.
	Profiler bool
}

// New creates an Engine configured with the given backend and options.
//
// Call Engine.Run to start the event loop (blocking), or call Engine.Frame
// directly for testing.
func New(b backend.Backend, opts Options) *Engine {
	numWorkers := opts.NumWorkers
	if numWorkers <= 0 {
		numWorkers = DefaultWorkers
	}
	macroTaskBudget := opts.MacroTaskBudget
	if macroTaskBudget <= 0 {
		macroTaskBudget = DefaultMacroTaskBudget
	}
	macroTaskDuration := opts.MacroTaskDuration
	if macroTaskDuration <= 0 {
		macroTaskDuration = DefaultMacroTaskDuration
	}
	clk := opts.Clock
	if clk == nil {
		clk = RealClock()
	}
	shutdownTimeout := opts.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 5 * time.Second
	}

	e := &Engine{
		renderView:      render.NewRenderView(),
		document:        dom.NewDocument(),
		resolver:        styler.NewResolver(),
		paintEngine:     paint.NewPaintEngine(),
		backend:         b,
		clock:           clk,
		macroTaskBudget: macroTaskBudget,
		diffMap:         make(map[render.Object]struct{}),
		shutdownTimeout: shutdownTimeout,
		mouseMode:       MouseModeClick,
		closeCh:         make(chan struct{}),
		wheelMap:        make(map[event.EventTarget]*event.WheelEvent),
		pipeline:        &StandardPipeline{},
		renderMap:       make(map[dom.Node]render.Object),
	}

	size := b.Size()
	e.frameBuffer = paint.NewFrameBuffer(0, 0, size.Width, size.Height)

	e.scheduler = newDefaultScheduler(numWorkers, clk, macroTaskDuration, nil)
	promise.SetScheduler(e.scheduler)

	if opts.Profiler {
		e.StartProfiling()
	}

	e.dispatcher = event.NewDispatcher()

	// Link document to render view
	e.setRenderObject(e.document, e.renderView)
	e.renderView.SetLogicalNode(e.document)
	e.renderView.SetViewportSize(b.Size())

	if d := internaldom.AsDirty(e.document); d != nil {
		d.MarkNeedsSync()
	}
	e.focusManager = focus.NewManager(e.document, e.dispatcher)

	e.document.SetFocusHandle(e.focusManager)
	e.document.SetTerminal(&TerminalProxy{e: e})
	if d, ok := e.document.(*internaldom.Document); ok {
		d.SetDefaultView(&domViewProxy{e: e})
	}
	e.synthesizer = internalevent.NewSynthesizer(e, e, internalevent.SynthesizerOptions{
		ScrollableResolver: e.resolveScrollable,
	})

	e.document.AddEventListener(event.EventSelectionChange, func(ev event.Event) {
		e.RequestFrame()
	})

	// Probe capabilities from the backend.
	e.caps = b.Caps()

	return e
}

// RenderView returns the root render object owned by this engine.
func (e *Engine) RenderView() *render.RenderView { return e.renderView }

// Document returns the logical document root.
func (e *Engine) Document() dom.Document { return e.document }

// PaintEngine returns the internal paint engine.
func (e *Engine) PaintEngine() *paint.PaintEngine { return e.paintEngine }

// AddEventListener registers fn as a listener for event of type typ on target.
// It returns a Subscription that can be used to remove the listener.
func (e *Engine) AddEventListener(target any, typ event.EventType, fn event.Listener, opts ...event.Option) event.Subscription {
	var et event.EventTarget
	switch t := target.(type) {
	case event.EventTarget:
		et = t
	case render.Object:
		et = t.EventTarget()
	}

	if et == nil {
		return nil
	}

	return et.AddEventListener(typ, fn, opts...)
}

// Mount appends root as the body of the document.
func (e *Engine) Mount(root dom.Element) {
	if currentBody := e.document.Body(); currentBody != nil {
		e.document.RemoveChild(currentBody)
	}
	if root != nil {
		e.document.AppendChild(root)
	}

	// Auto-focus the first focusable element in the new tree, if any, so the
	// user can start interacting with the UI right away. This mirrors browser behaviour: when a new page loads, the first tab stop is focused so the user can start typing or tabbing immediately without needing to click first. If no candidate exists, focus remains nil.
	e.focusManager.ResetScope()
}

// FocusedTarget returns the currently focused event target, or nil.
// It satisfies the event.FocusReader interface.
func (e *Engine) FocusedTarget() event.EventTarget {
	if e.focusManager == nil {
		return nil
	}
	return e.focusManager.Current()
}

// Tracer returns the engine's active tracer, or nil if profiling is disabled.
func (e *Engine) Tracer() *trace.Tracer {
	e.profilerMu.RLock()
	defer e.profilerMu.RUnlock()
	return e.tracer
}

// WithProfiler enables or disables the engine's built-in profiler.
func WithProfiler(enabled bool) func(*Options) {
	return func(o *Options) {
		o.Profiler = enabled
	}
}

// StartProfiling dynamically enables profiling by wrapping the engine's active
// pipeline with ProfilingPipeline (if not already wrapped) and starting a new tracer.
func (e *Engine) StartProfiling() {
	e.profilerMu.Lock()
	defer e.profilerMu.Unlock()

	e.tracer = trace.NewTracer()
	if _, ok := e.pipeline.(*ProfilingPipeline); !ok {
		e.pipeline = &ProfilingPipeline{wrapped: e.pipeline}
	}

	e.scheduler.onJobSubmit = func(name string) func() {
		e.profilerMu.RLock()
		tracer := e.tracer
		e.profilerMu.RUnlock()
		if tracer == nil {
			return noop
		}
		return tracer.BeginThread("JobSubmit:"+name, 1)
	}
	e.scheduler.onJobRun = func(name string, workerID int) func() {
		e.profilerMu.RLock()
		tracer := e.tracer
		e.profilerMu.RUnlock()
		if tracer == nil {
			return noop
		}
		return tracer.BeginThread("JobRun:"+name, workerID+1)
	}
}

// StopProfiling dynamically disables profiling by reverting the engine's active
// pipeline to the standard (unwrapped) pipeline and returning the accumulated tracer.
// If profiling was not active, it returns nil.
func (e *Engine) StopProfiling() *trace.Tracer {
	e.profilerMu.Lock()
	defer e.profilerMu.Unlock()

	t := e.tracer
	e.tracer = nil

	if pp, ok := e.pipeline.(*ProfilingPipeline); ok {
		e.pipeline = pp.wrapped
	}

	e.scheduler.onJobSubmit = nil
	e.scheduler.onJobRun = nil

	return t
}

// FocusManager returns the engine's focus.Manager so that tests and
// application code can drive focus programmatically (e.g. simulate a
// mousedown-to-focus or query the currently focused node).
func (e *Engine) FocusManager() *focus.Manager { return e.focusManager }

// HitTest walks the render tree at point (x, y) and returns the topmost
// event target at that position. It tests overlays (topmost-first) before
// falling through to the main tree. Returns nil when no target is hit.
func (e *Engine) HitTest(x, y int) event.EventTarget {
	e.EnsureFreshLayout()
	p := geom.Point{X: x, Y: y}

	// Walk overlays from the end (topmost) to start.
	overlays := e.renderView.Overlays()
	for i := len(overlays) - 1; i >= 0; i-- {
		ov := overlays[i]
		offset := ov.Offset()
		localP := geom.Point{
			X: p.X - offset.X,
			Y: p.Y - offset.Y,
		}
		if hit := hitTestFragment(ov.Fragment(), localP); hit != nil {
			return hit.EventTarget()
		}
	}
	// Fall through to the main tree.
	if hit := hitTestFragment(e.renderView.Fragment(), p); hit != nil {
		return hit.EventTarget()
	}
	return nil
}

// Caps returns the terminal capability snapshot probed at startup.
func (e *Engine) Caps() backend.Caps { return e.caps }

// Scheduler returns the engine's task scheduler.
func (e *Engine) Scheduler() terminal.Scheduler { return e.scheduler }

// Cursor returns the CursorController that widgets use to drive cursor state.
func (e *Engine) Cursor() *CursorController {
	return &CursorController{state: &e.cursor}
}

// SetMouseMode configures the mouse-event level. The backend toggles the
// relevant terminal protocol; the engine carries the policy.
//
// SetMouseMode must be called from the main goroutine.
func (e *Engine) SetMouseMode(mode MouseMode) {
	e.mouseMode = mode
}

// SetTitle sets the terminal window title. It is a no-op when Caps.Title is
// false.
//
// SetTitle must be called from the main goroutine.
func (e *Engine) SetTitle(s string) {
	if !e.caps.Title {
		return
	}
	kitelog.Info("engine: set title", slog.String("title", s))
}

// Bell emits a BEL character. It is a no-op when Caps.Bell is false.
//
// Bell must be called from the main goroutine.
func (e *Engine) Bell() {
	if !e.caps.Bell {
		return
	}
	kitelog.Info("engine: bell")
}

// SetDebugXRay toggles the visual layout debugging overlay.
func (e *Engine) SetDebugXRay(enabled bool) {
	e.paintEngine.DebugXRay = enabled
	// Force a repaint so the overlay appears even if the DOM hasn't changed.
	if e.renderView != nil {
		e.renderView.MarkDirty(render.DirtyPaint)
	}
	e.RequestFrame()
}

// RequestFrame schedules a frame wake-up as soon as possible (after at least
// MinFrameInterval has elapsed since the last frame). Consecutive calls within
// MinFrameInterval coalesce.
//
// RequestFrame is safe to call from any goroutine.
func (e *Engine) RequestFrame() {
	e.frameRequested = true
}

// RegisterAnimation registers an active animation to be ticked by the engine.
func (e *Engine) RegisterAnimation(anim animation.Animator) {
	e.activeAnimations = append(e.activeAnimations, anim)
	e.RequestFrame()
}

// OnFrameRendered registers a hook to be called after every frame is committed.
func (e *Engine) OnFrameRendered(fn func()) {
	e.onFrameRenderedHooks = append(e.onFrameRenderedHooks, fn)
}

// OnStop registers a hook to be called when the engine stops.
func (e *Engine) OnStop(fn func()) {
	e.onStopHooks = append(e.onStopHooks, fn)
}

// RequestFrameAt schedules a frame wake-up at time t. If t is before
// time.Now() the frame is scheduled immediately.
//
// RequestFrameAt must be called from the main goroutine.
func (e *Engine) RequestFrameAt(t time.Time) {
	e.frameRequested = true
	if e.nextFrameAt.IsZero() || t.Before(e.nextFrameAt) {
		e.nextFrameAt = t
	}
}

// Post schedules fn as a microtask. Microtasks run on the main thread and are
// drained (until the queue is empty) during each frame, between macrotask
// iterations and at the end of the task phase.
//
// Post is safe to call from any goroutine.
func (e *Engine) Post(fn func()) {
	e.scheduler.QueueMicrotask(fn)
}

// PostMacro schedules fn as a macrotask. Macrotasks are drained once per
// frame, subject to the configured count and duration budgets.
//
// PostMacro is safe to call from any goroutine.
func (e *Engine) PostMacro(fn func()) {
	e.scheduler.QueueMacrotask(fn)
}

// Timer represents a scheduled timeout or interval.
type Timer struct {
	stop func()
}

// Stop cancels the timer.
func (t *Timer) Stop() {
	if t.stop != nil {
		t.stop()
	}
}

// SetTimeout schedules fn to run on the main thread after the specified delay.
// It is safe to call from any goroutine.
func (e *Engine) SetTimeout(fn func(), delay time.Duration) *Timer {
	var stopOnce sync.Once
	t := time.AfterFunc(delay, func() {
		e.Post(fn)
	})
	return &Timer{
		stop: func() {
			stopOnce.Do(func() {
				t.Stop()
			})
		},
	}
}

// SetInterval schedules fn to run on the main thread repeatedly at the specified interval.
// It is safe to call from any goroutine.
func (e *Engine) SetInterval(fn func(), interval time.Duration) *Timer {
	var stopOnce sync.Once
	ticker := time.NewTicker(interval)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				e.Post(fn)
			case <-done:
				return
			}
		}
	}()

	return &Timer{
		stop: func() {
			stopOnce.Do(func() {
				ticker.Stop()
				close(done)
			})
		},
	}
}

// hitTestFragment walks the immutable layout Fragment tree and returns the deepest
// render.Object whose computed bounds contain p. p is in the local coordinate space
// of the given fragment.
func hitTestFragment(frag *layout.Fragment, p geom.Point) render.Object {
	if frag == nil {
		return nil
	}
	// Optimized point-in-rect check for (0,0,width,height)
	if p.X < 0 || p.Y < 0 || p.X >= frag.Size.Width || p.Y >= frag.Size.Height {
		return nil
	}

	// ── Account for Scroll Translation ───────────────────────────────────────
	scrollX, scrollY := 0, 0
	if frag.Node != nil {
		if el, ok := frag.Node.LogicalNode().(dom.Element); ok {
			s := frag.Node.Style()
			if isScrollable(s.OverflowX) || isScrollable(s.OverflowY) {
				rawX, rawY := el.Scroll()
				maxSX, maxSY := layout.MaxScroll(frag)
				scrollX = max(0, min(rawX, maxSX))
				scrollY = max(0, min(rawY, maxSY))
			}
		}
	}

	// Walk children in reverse paint order (last child is topmost).
	for i := len(frag.Children) - 1; i >= 0; i-- {
		link := frag.Children[i]
		// Translate point into child's coordinate space, accounting for scroll.
		childPoint := geom.Point{
			X: p.X - link.Offset.X + scrollX,
			Y: p.Y - link.Offset.Y + scrollY,
		}
		if hit := hitTestFragment(link.Fragment, childPoint); hit != nil {
			return hit
		}
	}
	if ro, ok := frag.Node.(render.Object); ok {
		return ro
	}
	return nil
}

// Frame executes one complete frame pipeline:
//  1. Sync Phase.
//  2. Task Phase (macrotasks + microtasks).
//  3. Style phase.
//  4. Layout phase.
//  5. Paint phase.
//  6. Commit.
//
// Frame must be called from the main goroutine.
func (e *Engine) Frame() {
	defer e.recoverFrame()

	e.profilerMu.RLock()
	pipe := e.pipeline
	tracer := e.tracer
	e.profilerMu.RUnlock()

	var endFrame = noop
	if tracer != nil {
		endFrame = tracer.BeginThread("Frame", 1)
	}
	defer endFrame()

	// 1. Drain tasks first so that any state updates they make are processed in the same frame.
	pipe.Tasks(e)

	// Tick active animations at the very top.
	if len(e.activeAnimations) > 0 {
		if tracer != nil {
			defer tracer.BeginWithArgs("Phase:Animations", map[string]any{
				"activeCount": len(e.activeAnimations),
			})()
		}
		now := e.clock.Now()
		var dt time.Duration
		if !e.lastFrameTime.IsZero() {
			dt = now.Sub(e.lastFrameTime)
		}
		e.lastFrameTime = now

		// Iterate backwards to allow safe removal of finished animations.
		for i := len(e.activeAnimations) - 1; i >= 0; i-- {
			anim := e.activeAnimations[i]
			finished := func() bool {
				if tracer != nil {
					defer tracer.Begin(fmt.Sprintf("Animation:%T", anim))()
				}
				return anim.Tick(dt)
			}()
			if finished {
				e.activeAnimations = collections.DeleteAt(e.activeAnimations, i)
			}
		}

		if len(e.activeAnimations) == 0 {
			e.lastFrameTime = time.Time{}
		}
	} else {
		e.lastFrameTime = time.Time{}
	}

	// 2. Check if a rendering frame is actually required.
	dirty := e.hasDirtyUIState() || e.frameRequested || len(e.activeAnimations) > 0
	if !dirty {
		return
	}

	e.isLayoutActive = true
	pipe.Sync(e)
	pipe.Style(e)
	layoutRan := pipe.Layout(e)
	e.isLayoutActive = false

	pipe.Paint(e, layoutRan)

	e.nextFrameAt = time.Time{}
	e.frameRequested = false

	// Self-scheduling: keep waking up as long as we have active animations.
	if len(e.activeAnimations) > 0 {
		e.RequestFrame()
	}

	for _, fn := range e.onFrameRenderedHooks {
		fn()
	}
}

// cursorRecord duplicates the backend-internal state for tracking changes in the engine.
type cursorRecord struct {
	Visible bool
	X, Y    int
	Shape   backend.CursorShape
	Color   color.Color
}

func (e *Engine) updateHardwareCursor(layoutRan bool) bool {
	e.EnsureFreshLayout()
	focused := e.focusManager.Current()

	// Short-circuit: if focus hasn't changed and the tree is clean, the cursor
	// physically cannot have moved.
	root := e.renderView
	treeDirty := root.Flags()&(render.DirtyPaint|render.DirtyScroll|render.ChildNeedsPaint) != 0
	if !layoutRan && !treeDirty && focused == e.lastHardwareFocus {
		return false
	}
	e.lastHardwareFocus = focused

	next := cursorRecord{}
	var ro render.Object
	if focused != nil {
		kitelog.Info("engine: determining cursor for focused target")
		ro = e.RenderObject(focused.(dom.Node))
		if ro != nil {
			provider, ok := ro.(cursor.Provider)
			if !ok {
				provider, ok = focused.(cursor.Provider)
			}
			if ok {
				state := provider.CursorState()
				if state.Visible {
					rootFrag := root.Fragment()
					if bounds, clip, found := layout.ScrolledAbsoluteBounds(rootFrag, ro); found {
						scrollX, scrollY := 0, 0
						rawX, rawY := focused.Scroll()
						maxSX, maxSY := layout.MaxScroll(ro.Fragment())
						scrollX = max(0, min(rawX, maxSX))
						scrollY = max(0, min(rawY, maxSY))
						cursorPos := geom.Point{
							X: bounds.Origin.X + state.X - scrollX,
							Y: bounds.Origin.Y + state.Y - scrollY,
						}

						// Hardware cursor is inside the content box.
						// If the element itself clips, we must intersect with its content box.
						cs := ro.ComputedStyle()
						if cs.OverflowX != style.OverflowVisible || cs.OverflowY != style.OverflowVisible {
							bw := cs.Border.Widths()
							contentBox := geom.Rect{
								Origin: geom.Point{
									X: bounds.Origin.X + bw.Left + cs.Padding.Left,
									Y: bounds.Origin.Y + bw.Top + cs.Padding.Top,
								},
								Size: geom.Size{
									Width:  max(0, bounds.Size.Width-bw.Left-bw.Right-cs.Padding.Left-cs.Padding.Right),
									Height: max(0, bounds.Size.Height-bw.Top-bw.Bottom-cs.Padding.Top-cs.Padding.Bottom),
								},
							}
							clip = clip.Intersect(contentBox)
						}

						// Hardware cursor should only be visible if it's within the clip region.
						// We use inclusive checks for the right/bottom edges because a
						// text cursor sits between characters and can logically sit
						// on the trailing boundary of its container.
						if cursorPos.X >= clip.Origin.X && cursorPos.X <= clip.Origin.X+clip.Size.Width &&
							cursorPos.Y >= clip.Origin.Y && cursorPos.Y <= clip.Origin.Y+clip.Size.Height {
							next.Visible = true
							next.X = cursorPos.X
							next.Y = cursorPos.Y

							// Resolve cursor shape from provider and style.
							cursorStyle := cs.Cursor
							if state.Style.Shape.IsSet() {
								cursorStyle.Shape = state.Style.Shape
							}
							if state.Style.Blink.IsSet() {
								cursorStyle.Blink = state.Style.Blink
							}
							if state.Style.Color.IsSet() {
								cursorStyle.Color = state.Style.Color
							}

							next.Shape = mapCursorShape(cursorStyle.Shape.UnwrapOr(style.CursorBlock))
							next.Color = cursorStyle.Color.UnwrapOr(style.TerminalDefault)
						}
					}
				}
			}
		}
	}

	if next != e.lastCursorState {
		e.lastCursorState = next
		if next.Visible {
			e.backend.SetCursorPos(next.X, next.Y)
			e.backend.SetCursorShape(next.Shape)
			if next.Color != nil && next.Color != style.TerminalDefault {
				e.backend.SetCursorColor(next.Color)
			}
			e.backend.ShowCursor(true)
		} else {
			e.backend.ShowCursor(false)
		}
		return true
	}
	return false
}

func mapCursorShape(s style.CursorShape) backend.CursorShape {
	switch s {
	case style.CursorBlock:
		return backend.CursorBlock
	case style.CursorBar:
		return backend.CursorBar
	case style.CursorUnderline:
		return backend.CursorUnderline
	default:
		return backend.CursorBlock
	}
}

// syncRenderTree walks the logical DOM and ensures the render tree matches
// its structure.
func (e *Engine) syncRenderTree(rootNode dom.Node, rootRO render.Object) {
	e.syncStack = append(e.syncStack[:0], syncTask{node: rootNode, ro: rootRO})

	for len(e.syncStack) > 0 {
		idx := len(e.syncStack) - 1
		task := e.syncStack[idx]
		e.syncStack = e.syncStack[:idx]

		dn := internaldom.AsDirty(task.node)
		if dn.NeedsSync() {
			if de := internaldom.AsDirtyElement(task.node); de != nil {
				de.MarkStyleDirty()
			}
			if _, ok := task.node.(dom.Document); ok {
				task.ro.MarkDirty(render.DirtyPaint)
			} else {
				task.ro.MarkDirty(render.DirtyLayout | render.DirtyPaint)
			}
			e.diffChildren(task.node, task.ro)
		} else if dn.ChildNeedsSync() {
			for child := task.node.FirstChild(); child != nil; child = child.NextSibling() {
				if childRO := e.RenderObject(child); childRO != nil {
					e.syncStack = append(e.syncStack, syncTask{node: child, ro: childRO})
				}
			}
			if el, ok := task.node.(dom.Element); ok {
				if uaRoot := internaldom.UARoot(el); uaRoot != nil {
					for child := uaRoot.FirstChild(); child != nil; child = child.NextSibling() {
						if childRO := e.RenderObject(child); childRO != nil {
							e.syncStack = append(e.syncStack, syncTask{node: child, ro: childRO})
						}
					}
				}
			}
		}
		dn.ClearSyncFlags()
	}
}

func (e *Engine) syncOverlays(d dom.Document) {
	var overlayROs []render.Object
	for overlayEl := range d.Overlays() {
		childRO := e.RenderObject(overlayEl)
		if childRO == nil {
			childRO = e.createRenderObject(overlayEl)
			e.setRenderObject(overlayEl, childRO)
		}

		// If the child was already there, it might still need internal sync.
		dn := internaldom.AsDirty(overlayEl)
		if dn.NeedsSync() || dn.ChildNeedsSync() {
			e.syncRenderTree(overlayEl, childRO)
		}

		overlayROs = append(overlayROs, childRO)
	}
	e.renderView.SetOverlays(overlayROs)
}

func (e *Engine) diffChild(child dom.Node, parentRO render.Object, lastRO render.Object) render.Object {
	if el, ok := child.(dom.Element); ok && internaldom.IsOverlay(el) {
		return lastRO
	}
	childRO := e.RenderObject(child)
	if childRO == nil {
		childRO = e.createRenderObject(child)
		e.setRenderObject(child, childRO)
	}

	delete(e.diffMap, childRO)

	// Ensure correct position in render tree.
	if childRO.Parent() != parentRO || childRO.PreviousSibling() != lastRO {
		render.Unlink(childRO)
		var before render.Object
		if lastRO == nil {
			before = parentRO.FirstChild()
		} else {
			before = lastRO.NextSibling()
		}
		parentRO.InsertChild(childRO, before)
	}

	// If the child was already there, it might still need internal sync.
	// If it was just created, createRenderObject already synced its children.
	dn := internaldom.AsDirty(child)
	if dn.NeedsSync() || dn.ChildNeedsSync() {
		e.syncStack = append(e.syncStack, syncTask{node: child, ro: childRO})
	}

	return childRO
}

// diffChildren synchronizes the children of n into the render object ro.
func (e *Engine) diffChildren(n dom.Node, parentRO render.Object) {
	// Map existing render children.
	clear(e.diffMap)
	for childRO := parentRO.FirstChild(); childRO != nil; childRO = childRO.NextSibling() {
		e.diffMap[childRO] = struct{}{}
	}

	var lastRO render.Object
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		lastRO = e.diffChild(child, parentRO, lastRO)
	}
	if el, ok := n.(dom.Element); ok {
		if uaRoot := internaldom.UARoot(el); uaRoot != nil {
			for child := uaRoot.FirstChild(); child != nil; child = child.NextSibling() {
				lastRO = e.diffChild(child, parentRO, lastRO)
			}
		}
	}

	// Remove orphaned render objects.
	for orphaned := range e.diffMap {
		e.clearRenderMapRecursive(orphaned)
		render.Unlink(orphaned)
	}
}

func (e *Engine) clearRenderMapRecursive(ro render.Object) {
	if ro == nil {
		return
	}
	for child := ro.FirstChild(); child != nil; child = child.NextSibling() {
		e.clearRenderMapRecursive(child)
	}
	if ln := ro.LogicalNode(); ln != nil {
		e.setRenderObject(ln, nil)
		e.resolver.Invalidate(ln)
	}
}

// createRenderObject creates a new render object for the given logical node.
// It also recursively creates render objects for any existing children.
func (e *Engine) createRenderObject(n dom.Node) render.Object {
	var ro render.Object

	if n.Kind() == dom.KindText {
		ro = render.NewText(n, n.EventTarget())
	} else if cp := unwrapProvider(n); cp != nil {
		ro = cp.CreateRenderObject()
	} else {
		// Fallback for nodes that don't implement CustomObjectProvider.
		ro = render.NewBox(n, n.EventTarget())
	}

	e.setRenderObject(n, ro)

	// Mark elements style-dirty so they get resolved in the next phase.
	if de := internaldom.AsDirtyElement(n); de != nil {
		de.MarkStyleDirty()
	}

	// Notify logical node of creation.
	if h, ok := n.(render.RenderObjectHook); ok {
		h.OnRenderObjectCreated(ro)
	}

	// Recursively build render subtree for existing DOM children
	// (using LayoutChildren so UA subtrees are included).
	e.diffChildren(n, ro)

	return ro
}

// recoverFrame is a deferred function that catches panics in the frame loop.
// It restores the terminal, logs the panic and stack trace, and re-panics so
// the process exits with a usable terminal.
func (e *Engine) recoverFrame() {
	if v := recover(); v != nil {
		kitelog.Error("engine: panic in frame loop",
			slog.Any("panic", v),
			slog.String("stack", string(debug.Stack())),
		)
		e.backend.Restore()
		panic(v)
	}
}

func noop() {}

func (e *Engine) Dump(path string) error {
	size := e.backend.Size()
	rawText := make([]string, size.Height)

	// Capture the current visible state by re-painting into a temporary buffer.
	// We cannot use e.backend.BeginFrame() because it returns an empty surface.
	fb := paint.NewFrameBuffer(0, 0, size.Width, size.Height)
	root := e.renderView
	e.paintEngine.PaintFragment(nil, root.Fragment(), geom.Point{}, fb)
	for _, overlay := range root.Overlays() {
		offset := overlay.Offset()
		e.paintEngine.PaintFragment(nil, overlay.Fragment(), offset, fb)
	}
	e.paintEngine.ResolveBorders(nil, fb)

	for y := 0; y < size.Height; y++ {
		var line strings.Builder
		for x := 0; x < size.Width; x++ {
			c := fb.CellAt(x, y)
			if c.Content == "" {
				line.WriteRune(' ')
			} else {
				line.WriteString(c.Content)
			}
		}
		rawText[y] = line.String()
	}

	type nodeDump struct {
		Name     string      `json:"name"`
		Kind     string      `json:"kind"`
		Data     string      `json:"data,omitempty"`
		Size     string      `json:"size,omitempty"`
		Children []*nodeDump `json:"children,omitempty"`
	}

	var dumpNode func(n dom.Node) *nodeDump
	dumpNode = func(n dom.Node) *nodeDump {
		d := &nodeDump{
			Name: n.NodeName(),
			Kind: n.Kind().String(),
		}
		if tn, ok := n.(dom.TextNode); ok {
			d.Data = tn.Data()
		}

		if ro := e.RenderObject(n); ro != nil {
			if frag := ro.Fragment(); frag != nil {
				d.Size = fmt.Sprintf("%dx%d", frag.Size.Width, frag.Size.Height)
			}
		}

		for child := range n.ChildNodes() {
			d.Children = append(d.Children, dumpNode(child))
		}
		return d
	}

	data := struct {
		ScreenSize struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"screen_size"`
		RawText        []string    `json:"raw_text"`
		DOMTree        *nodeDump   `json:"dom_tree"`
		Overlays       []*nodeDump `json:"overlays,omitempty"`
		Cursor         cursorRecord
		Fragments      *layout.Fragment
		SelectedText   string                `json:"selected_text,omitempty"`
		SelectionRects []paint.SelectionRect `json:"selection_rects,omitempty"`
	}{
		ScreenSize: struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}{Width: size.Width, Height: size.Height},
		RawText: rawText,
		Cursor:  e.lastCursorState,
		DOMTree: dumpNode(e.document),
	}

	for overlay := range e.document.Overlays() {
		data.Overlays = append(data.Overlays, dumpNode(overlay))
	}
	data.Fragments = root.Fragment()

	data.SelectedText = e.document.Selection().String()
	data.SelectionRects = e.resolveSelection()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// Done returns a channel that is closed when the engine stops.
func (e *Engine) Done() <-chan struct{} {
	return e.closeCh
}

func (e *Engine) Stop() {
	e.closeOnce.Do(func() {
		for _, fn := range e.onStopHooks {
			fn()
		}
		close(e.closeCh)
		e.scheduler.stop()
		e.backend.Restore()
	})
}

// Run starts the engine's main event loop. It blocks until Stop is called or
// the backend signals exit.
func (e *Engine) Run(ctx context.Context) error {
	if err := e.backend.Start(); err != nil {
		return err
	}

	// 1. Setup high-level services for the Document.
	e.document.SetClipboardProvider(e.document.Terminal().Clipboard())

	// Restore terminal state when leaving run loop
	defer e.backend.Restore()
	defer e.Stop()

	ticker := time.NewTicker(MinFrameInterval)
	defer ticker.Stop()

	input := e.backend.Events()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-e.closeCh:
			return nil
		case raw, ok := <-input:
			if !ok {
				return nil
			}
			e.eventBuffer = append(e.eventBuffer, raw)
			eventsCounter++
		case <-ticker.C:
			e.drainEvents()
			if e.shouldRunFrame() {
				e.Frame()
			}
		}
	}
}

// ProcessRawEvent converts a raw backend event into a sequence of DOM events
// and dispatches them through the tree. It is used primarily for testing
// and for custom event-loop implementations.
func (e *Engine) ProcessRawEvent(raw backend.RawEvent) {
	e.EnsureFreshLayout()
	e.processRawEvent(raw)
}

func (e *Engine) drainEvents() {
	if len(e.eventBuffer) == 0 {
		return
	}

	batchCounter++

	// 1. Coalesce raw events before synthesis to save allocations.
	rawCoalesced := e.coalesceRawEvents(e.eventBuffer)

	// 2. Synthesize coalesced raw events into structured events.
	for i := range e.structuredBuf {
		e.structuredBuf[i] = nil
	}
	e.structuredBuf = e.structuredBuf[:0]
	for _, raw := range rawCoalesced {
		e.structuredBuf = append(e.structuredBuf, e.synthesizer.Process(raw)...)
	}

	// 3. Coalesce structured events (handles wheel aggregation per target).
	coalesced := e.coalesceEvents(e.structuredBuf)

	// 4. Dispatch.
	for _, ev := range coalesced {
		e.dispatchEvent(ev)
	}

	// 5. Clean up all buffers to prevent leaks while idle.
	for i := range e.eventBuffer {
		e.eventBuffer[i] = nil
	}
	e.eventBuffer = e.eventBuffer[:0]

	for i := range e.structuredBuf {
		e.structuredBuf[i] = nil
	}
	e.structuredBuf = e.structuredBuf[:0]

	for i := range e.rawCoalescedBuf {
		e.rawCoalescedBuf[i] = nil
	}
	e.rawCoalescedBuf = e.rawCoalescedBuf[:0]

	for i := range e.coalescedBuf {
		e.coalescedBuf[i] = nil
	}
	e.coalescedBuf = e.coalescedBuf[:0]
}

func (e *Engine) coalesceRawEvents(events []backend.RawEvent) []backend.RawEvent {
	if len(events) <= 1 {
		return events
	}

	// Strategy:
	// - Keep the LAST MouseMove.
	// - Accumulate consecutive Wheel events IF they are at the same coordinate and have same modifiers.
	// - Keep all other events (clicks, keys) in order.

	for i := range e.rawCoalescedBuf {
		e.rawCoalescedBuf[i] = nil
	}
	e.rawCoalescedBuf = e.rawCoalescedBuf[:0]

	// Find index of the absolute last mouse move in the whole batch.
	lastMoveIdx := -1
	for i := len(events) - 1; i >= 0; i-- {
		if m, ok := events[i].(*backend.RawMouseEvent); ok && m.Move && m.DeltaX == 0 && m.DeltaY == 0 {
			lastMoveIdx = i
			break
		}
	}

	for i, ev := range events {
		m, isMouse := ev.(*backend.RawMouseEvent)
		if !isMouse {
			e.rawCoalescedBuf = append(e.rawCoalescedBuf, ev)
			continue
		}

		// It's a mouse event.
		isWheel := m.DeltaX != 0 || m.DeltaY != 0
		isMove := m.Move && !isWheel

		if isMove {
			if i == lastMoveIdx {
				e.rawCoalescedBuf = append(e.rawCoalescedBuf, ev)
			}
			continue
		}

		if isWheel {
			// Check if we can merge with the previous event in the result slice.
			if len(e.rawCoalescedBuf) > 0 {
				if prev, ok := e.rawCoalescedBuf[len(e.rawCoalescedBuf)-1].(*backend.RawMouseEvent); ok {
					if prev.X == m.X && prev.Y == m.Y && prev.Mod == m.Mod && (prev.DeltaX != 0 || prev.DeltaY != 0) {
						prev.DeltaX += m.DeltaX
						prev.DeltaY += m.DeltaY
						continue
					}
				}
			}
			// Cannot merge, add new.
			e.rawCoalescedBuf = append(e.rawCoalescedBuf, ev)
			continue
		}

		// Button press/release, keep it.
		e.rawCoalescedBuf = append(e.rawCoalescedBuf, ev)
	}

	return e.rawCoalescedBuf
}

func (e *Engine) coalesceEvents(events []event.Event) []event.Event {
	if len(events) == 0 {
		return nil
	}

	// Find the index of the last MouseMove event.
	lastMouseMoveIdx := -1
	for i, ev := range events {
		if me, ok := ev.(*event.MouseEvent); ok && me.Type() == event.EventMouseMove {
			lastMouseMoveIdx = i
		}
	}

	for i := range e.coalescedBuf {
		e.coalescedBuf[i] = nil
	}
	e.coalescedBuf = e.coalescedBuf[:0]
	clear(e.wheelMap)

	for i, ev := range events {
		switch evt := ev.(type) {
		case *event.MouseEvent:
			if evt.Type() == event.EventMouseMove {
				if i == lastMouseMoveIdx {
					e.coalescedBuf = append(e.coalescedBuf, evt)
				}
				// discard older moves
			} else {
				e.coalescedBuf = append(e.coalescedBuf, evt)
			}
		case *event.WheelEvent:
			target := evt.Target()
			if existing, ok := e.wheelMap[target]; ok {
				existing.DeltaX += evt.DeltaX
				existing.DeltaY += evt.DeltaY
			} else {
				e.wheelMap[target] = evt
				e.coalescedBuf = append(e.coalescedBuf, evt)
			}
		default:
			e.coalescedBuf = append(e.coalescedBuf, evt)
		}
	}

	return e.coalescedBuf
}

func (e *Engine) dispatchEvent(ev event.Event) {
	switch evt := ev.(type) {
	case *event.MouseEvent:
		e.dispatchMouseEvent(evt)
	case *event.WheelEvent:
		e.dispatchWheelEvent(evt)
	case *event.KeyEvent:
		e.dispatchKeyEvent(evt)
	case *event.ResizeEvent:
		e.handleResize(evt)
	case *event.ClipboardEvent:
		if evt.Type() == event.EventPaste {
			e.clipboard.resolvePending(evt.Items)
		}
		e.dispatchGenericEvent(evt)
	default:
		e.dispatchGenericEvent(ev)
	}
}

func (e *Engine) dispatchGenericEvent(ev event.Event) {
	// For generic events (paste, etc), dispatch to focused element.
	var target dom.Node = e.focusManager.Current()
	if target == nil {
		// Fallback to document for global events.
		target = e.document
	}

	if target != nil {
		path := nodeAncestorPath(target)
		e.dispatcher.Dispatch(ev, path)
	}
}

func (e *Engine) processRawEvent(raw backend.RawEvent) {
	evts := e.synthesizer.Process(raw)
	for _, ev := range evts {
		e.dispatchEvent(ev)
	}
}

func (e *Engine) dispatchWheelEvent(ev *event.WheelEvent) {
	target := ev.OriginalTarget()
	if target == nil {
		return
	}
	if node, ok := target.(dom.Node); ok {
		e.setLocalWheelCoords(ev, node)
		path := nodeAncestorPath(node)
		scrollables := e.synthesizer.ResolveScrollables(path)
		e.dispatcher.DispatchWheel(ev, path, scrollables)
	}
}

func (e *Engine) setLocalWheelCoords(ev *event.WheelEvent, target dom.Node) {
	ro := e.RenderObject(target)
	if ro == nil {
		return
	}
	root := e.renderView.Fragment()
	if bounds, _, found := layout.ScrolledAbsoluteBounds(root, ro); found {
		// ScrolledAbsoluteBounds returns the scrolled border-box.
		// Local coordinate in event should be relative to this scrolled box.
		ev.Local = geom.Point{
			X: ev.Screen.X - bounds.Origin.X,
			Y: ev.Screen.Y - bounds.Origin.Y,
		}
	}
}

func (e *Engine) resolveScrollable(target event.EventTarget) event.Scrollable {
	// 1. Author-registered or Widget-provided Scrollable.
	if sc, ok := target.(event.Scrollable); ok {
		return sc
	}

	// 2. Framework default if the target's element is a scroll container.
	var el dom.Element
	if n, ok := target.(dom.Element); ok {
		el = n
	} else if n, ok := target.(dom.Node); ok {
		el = n.ParentElement()
	}

	if el == nil {
		return nil
	}

	ro := e.RenderObject(el)
	if ro == nil {
		return nil
	}

	cs := ro.ComputedStyle()
	if cs == nil {
		return nil
	}

	if isScrollContainer(cs) {
		return dom.DefaultScroller(el)
	}

	return nil
}

func isScrollable(o style.Overflow) bool {
	return o == style.OverflowScroll || o == style.OverflowAuto || o == style.OverflowHidden || o == style.OverflowClip
}

func isScrollContainer(cs *style.Computed) bool {
	return isScrollable(cs.OverflowX) || isScrollable(cs.OverflowY)
}

func (e *Engine) dispatchMouseEvent(ev *event.MouseEvent) {
	target := ev.OriginalTarget()
	if target == nil {
		return
	}
	if node, ok := target.(dom.Node); ok {
		e.setLocalMouseCoords(ev, node)

		var path []event.EventTarget
		if ev.Type() == event.EventMouseEnter || ev.Type() == event.EventMouseLeave {
			path = []event.EventTarget{node}
		} else {
			path = nodeAncestorPath(node)
		}

		e.dispatcher.Dispatch(ev, path)

		// Move focus to the clicked node if it is focusable and the event
		// was a mousedown that was not cancelled by a listener. Focus on
		// mousedown (not click) matches browser behaviour: the element
		// becomes focused as soon as the button is pressed, so that key
		// events fired before the button is released already land on the
		// right target. Listeners may call ev.PreventDefault() on the
		// mousedown to opt out.
		if ev.Type() == event.EventMouseDown && !ev.DefaultPrevented() {
			focused := false
			if el, ok := node.(dom.Element); ok {
				if e.focusManager.SetFocus(el, focus.ReasonPointer) {
					focused = true
				}
			}
			if !focused {
				e.focusManager.Blur()
			}
		}
	}
}

func nodeAncestorPath(n dom.Node) []event.EventTarget {
	var chain []event.EventTarget
	for cur := n; cur != nil; {
		chain = append(chain, cur)
		parent := cur.Parent()
		if parent == nil {
			// 1. Cross UA shadow boundary: jump from UARoot to host element.
			// EventTarget() on a UAT node returns its host (ADR-0036).
			if et := cur.EventTarget(); et != nil {
				if host, ok := et.(dom.Node); ok && host != cur {
					cur = host
					continue
				}
			}

			// 2. Cross Overlay boundary: jump from Overlay to Anchor.
			if ov, ok := cur.(interface{ Anchor() dom.Element }); ok {
				if anchor := ov.Anchor(); anchor != nil {
					if aNode, ok := anchor.(dom.Node); ok {
						cur = aNode
						continue
					}
				}
			}

			// check if it is the document
			if doc := cur.OwnerDocument(); doc != nil && cur != doc {
				cur = doc
				continue
			}
			break
		}
		cur = parent
	}
	// Reverse to get root → n order.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
}

func (e *Engine) dispatchKeyEvent(ev *event.KeyEvent) {
	var path []event.EventTarget
	if focused := e.focusManager.Current(); focused != nil {
		path = nodeAncestorPath(focused)
	} else {
		// Nothing is focused yet. Auto-focus the first focusable element in
		// DOM tree order so the very first keystroke lands somewhere useful
		// (mirrors browser behaviour). If no candidate exists, fall back to
		// dispatching on the document.
		if e.focusManager.Next() {
			path = nodeAncestorPath(e.focusManager.Current())
		} else {
			path = []event.EventTarget{e.document}
		}
	}

	e.dispatcher.Dispatch(ev, path)
	if !ev.DefaultPrevented() {
		e.handleDefaultKeyAction(ev)
	}
}

func (e *Engine) handleResize(ev *event.ResizeEvent) {
	e.backend.Resize(geom.Size{Width: ev.Width, Height: ev.Height})
	e.frameBuffer = paint.NewFrameBuffer(0, 0, ev.Width, ev.Height)
	e.renderView.SetViewportSize(geom.Size{
		Width:  ev.Width,
		Height: ev.Height,
	})
	e.RequestFrame()
}

func (e *Engine) resolveSelection() []paint.SelectionRect {
	sel := e.document.Selection()
	if sel == nil || sel.RangeCount() == 0 {
		return nil
	}
	root := e.renderView.Fragment()
	if root == nil {
		return nil
	}

	doc, ok := e.document.(*internaldom.Document)
	if !ok {
		return nil
	}
	nodeOrder := doc.PreorderOrders()
	source := &selectionSourceAdapter{sel: sel, nodeOrder: nodeOrder}
	rs := render.ResolveSelection(root, source, nodeOrder)
	if len(rs) == 0 {
		return nil
	}
	ps := make([]paint.SelectionRect, len(rs))
	for i, r := range rs {
		ps[i] = paint.SelectionRect{
			Rect: r.Rect,
			Fg:   r.Fg,
			Bg:   r.Bg,
		}
	}
	return ps
}

type selectionSourceAdapter struct {
	sel       dom.Selection
	nodeOrder map[any]render.NodeOrder
}

func (a *selectionSourceAdapter) RangeCount() int {
	return a.sel.RangeCount()
}

func (a *selectionSourceAdapter) GetRangeAt(idx int) render.SelectionRange {
	r := a.sel.GetRangeAt(idx)
	if r == nil {
		return nil
	}
	return &selectionRangeAdapter{r: r, nodeOrder: a.nodeOrder}
}

type selectionRangeAdapter struct {
	r         dom.Range
	nodeOrder map[any]render.NodeOrder
}

func (a *selectionRangeAdapter) StartContainerAny() any {
	return a.r.StartContainer()
}
func (a *selectionRangeAdapter) EndContainerAny() any {
	return a.r.EndContainer()
}
func (a *selectionRangeAdapter) StartOffset() int  { return a.r.StartOffset() }
func (a *selectionRangeAdapter) EndOffset() int    { return a.r.EndOffset() }
func (a *selectionRangeAdapter) IsCollapsed() bool { return a.r.IsCollapsed() }

func getBaseNode(n dom.Node) any {
	if n == nil {
		return nil
	}
	curr := n
	for {
		if u := curr.Unwrap(); u != nil && u != curr {
			curr = u
		} else {
			break
		}
	}
	return curr
}

func (a *selectionRangeAdapter) StartIndex() int {
	startNode := a.r.StartContainer()
	if startNode == nil {
		return 0
	}
	baseStart := getBaseNode(startNode)
	ord, ok := a.nodeOrder[baseStart]
	if !ok {
		return 0
	}
	if _, ok := baseStart.(dom.TextNode); ok {
		return ord.First
	}
	// Element
	var children []dom.Node
	for child := range internaldom.LayoutChildren(startNode) {
		children = append(children, child)
	}
	offset := a.r.StartOffset()
	if offset < len(children) {
		childBase := getBaseNode(children[offset])
		if childOrd, ok := a.nodeOrder[childBase]; ok {
			return childOrd.First
		}
	}
	return ord.Last + 1
}

func (a *selectionRangeAdapter) EndIndex() int {
	endNode := a.r.EndContainer()
	if endNode == nil {
		return 0
	}
	baseEnd := getBaseNode(endNode)
	ord, ok := a.nodeOrder[baseEnd]
	if !ok {
		return 0
	}
	if _, ok := baseEnd.(dom.TextNode); ok {
		return ord.Last + 1
	}
	// Element
	var children []dom.Node
	for child := range internaldom.LayoutChildren(endNode) {
		children = append(children, child)
	}
	offset := a.r.EndOffset()
	if offset < len(children) {
		childBase := getBaseNode(children[offset])
		if childOrd, ok := a.nodeOrder[childBase]; ok {
			return childOrd.First
		}
	}
	return ord.Last + 1
}

func (e *Engine) handleDefaultKeyAction(ev *event.KeyEvent) {
	// Tab navigation, etc.
	switch {
	case ev.MatchString("tab"):
		e.focusManager.Next()
	case ev.MatchString("shift+tab"):
		e.focusManager.Previous()
	case ev.MatchString("up"), ev.MatchString("down"), ev.MatchString("left"), ev.MatchString("right"):
		dir := spatial.DirectionDown
		switch {
		case ev.MatchString("up"):
			dir = spatial.DirectionUp
		case ev.MatchString("left"):
			dir = spatial.DirectionLeft
		case ev.MatchString("right"):
			dir = spatial.DirectionRight
		}
		spatial.Navigate(e.focusManager, dir)
	}
}

func (e *Engine) setLocalMouseCoords(ev *event.MouseEvent, target dom.Node) {
	ro := e.RenderObject(target)
	if ro == nil {
		return
	}
	root := e.renderView.Fragment()
	if bounds, _, found := layout.ScrolledAbsoluteBounds(root, ro); found {
		// ScrolledAbsoluteBounds returns the scrolled border-box.
		// Local coordinate in event should be relative to this scrolled box.
		ev.Local = geom.Point{
			X: ev.Screen.X - bounds.Origin.X,
			Y: ev.Screen.Y - bounds.Origin.Y,
		}
	}
}

// hasDirtyUIState returns true if the DOM, overlays, or render tree are dirty
// and need to be synchronized, styled, laid out, or painted.
func (e *Engine) hasDirtyUIState() bool {
	dn := internaldom.AsDirty(e.document)
	if dn.NeedsSync() || dn.ChildNeedsSync() || dn.HasDirtyStyleChild() {
		return true
	}
	for overlay := range e.document.Overlays() {
		od := internaldom.AsDirty(overlay)
		if od.NeedsSync() || od.ChildNeedsSync() || od.HasDirtyStyleChild() {
			return true
		}
	}
	rootFlags := e.renderView.Flags()
	if rootFlags&(render.DirtyStyle|render.DirtyLayout|render.ChildNeedsLayout|render.DirtyPaint|render.DirtyScroll|render.ChildNeedsPaint) != 0 {
		return true
	}
	return false
}

func (e *Engine) shouldRunFrame() bool {
	if e.frameRequested {
		kitelog.Debug("shouldRunFrame returning true due to frameRequested")
		return true
	}
	now := e.clock.Now()
	if !e.nextFrameAt.IsZero() && !now.Before(e.nextFrameAt) {
		kitelog.Debug("shouldRunFrame returning true due to nextFrameAt", "nextFrameAt", e.nextFrameAt)
		return true
	}

	pending := e.scheduler.hasPendingTasks()
	dirty := e.hasDirtyUIState()

	if dirty || pending {
		kitelog.Debug("shouldRunFrame returning true due to dirty state",
			"dirty", dirty,
			"pending", pending,
		)
		return true
	}
	return false
}

func unwrapProvider(n dom.Node) render.CustomObjectProvider {
	if n == nil {
		return nil
	}
	if cp, ok := n.(render.CustomObjectProvider); ok {
		return cp
	}
	return unwrapProvider(n.Unwrap())
}

func (e *Engine) RenderObject(n dom.Node) render.Object {
	if n == nil {
		return nil
	}

	curr := n
	for {
		if u := curr.Unwrap(); u != nil && u != curr {
			curr = u
		} else {
			break
		}
	}
	return e.renderMap[curr]
}

func (e *Engine) setRenderObject(n dom.Node, ro render.Object) {
	if n == nil {
		return
	}

	curr := n
	for {
		if u := curr.Unwrap(); u != nil && u != curr {
			curr = u
		} else {
			break
		}
	}

	if ro == nil {
		delete(e.renderMap, curr)
	} else {
		e.renderMap[curr] = ro
	}
}

// EnsureFreshLayout forces an immediate Sync and Layout pass if the DOM or
// Render tree is dirty. This guarantees that subsequent reads of layout data
// (like element bounds or cursor positions) are accurate. This is analogous to
// forced synchronous layout in web browsers.
func (e *Engine) EnsureFreshLayout() {
	if e.isLayoutActive {
		return
	}

	dn := internaldom.AsDirty(e.document)
	needsSync := dn.NeedsSync() || dn.ChildNeedsSync()
	if !needsSync {
		for overlay := range e.document.Overlays() {
			od := internaldom.AsDirty(overlay)
			if od.NeedsSync() || od.ChildNeedsSync() {
				needsSync = true
				break
			}
		}
	}

	needsStyle := dn.HasDirtyStyleChild()
	if !needsStyle {
		for overlay := range e.document.Overlays() {
			od := internaldom.AsDirty(overlay)
			if od.HasDirtyStyleChild() {
				needsStyle = true
				break
			}
		}
	}

	needsLayout := e.renderView.Flags()&(render.DirtyStyle|render.DirtyLayout|render.ChildNeedsLayout) != 0
	if !needsLayout {
		for _, overlay := range e.renderView.Overlays() {
			if overlay.Flags()&(render.DirtyStyle|render.DirtyLayout|render.ChildNeedsLayout) != 0 {
				needsLayout = true
				break
			}
		}
	}

	if !needsSync && !needsStyle && !needsLayout {
		return
	}

	e.isLayoutActive = true
	defer func() { e.isLayoutActive = false }()

	if needsSync {
		e.syncRenderTree(e.document, e.renderView)
		e.syncOverlays(e.document)
	}

	// Re-evaluate needsStyle / needsLayout after synchronization
	if !needsStyle {
		needsStyle = dn.HasDirtyStyleChild()
		if !needsStyle {
			for overlay := range e.document.Overlays() {
				od := internaldom.AsDirty(overlay)
				if od.HasDirtyStyleChild() {
					needsStyle = true
					break
				}
			}
		}
	}

	if !needsLayout {
		needsLayout = e.renderView.Flags()&(render.DirtyStyle|render.DirtyLayout|render.ChildNeedsLayout) != 0
		if !needsLayout {
			for _, overlay := range e.renderView.Overlays() {
				if overlay.Flags()&(render.DirtyStyle|render.DirtyLayout|render.ChildNeedsLayout) != 0 {
					needsLayout = true
					break
				}
			}
		}
	}

	if needsLayout || needsStyle {
		e.pipeline.Style(e)
		e.pipeline.Layout(e)
	}
}
