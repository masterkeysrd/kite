package engine

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/focus/spatial"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/paint"
	"github.com/masterkeysrd/kite/internal/render"
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

// workerResult is the value carried from a worker goroutine back to the
// microtask queue after a job completes.
type workerResult struct {
	job Job
	err error
}

// Engine orchestrates the five-phase frame pipeline (Tasks → Style → Layout →
// Paint → Sync), the worker pool, and the macrotask / microtask queues.
//
// Engine must be used from a single main goroutine for all render-tree
// operations; the worker pool runs concurrent goroutines for user-submitted
// Job.Run methods only.
type Engine struct {
	document   dom.Document
	renderView *render.RenderView

	// resolver drives the Style phase.
	resolver *style.Resolver

	// layoutEngine was removed in favor of LayoutNG.
	// layoutEngine render.LayoutMeasurer

	// paintEngine drives the Paint pipeline.
	paintEngine *paint.PaintEngine

	// dispatcher performs 3-phase event dispatch.
	dispatcher *event.Dispatcher

	// synthesizer converts raw backend input into structured events.
	synthesizer *event.Synthesizer

	// focusManager owns focus state and scope stack.
	focusManager *focus.Manager

	// backend is the output target.
	backend backend.Backend

	// macroQueue holds pending macrotasks.
	macroQueue []func()
	// microQueue holds pending microtasks.
	microQueue []func()

	// workerResults receives completed job results from worker goroutines.
	workerResults chan workerResult

	// workerCtx / workerCancel control the lifetime of worker goroutines.
	workerCtx    context.Context
	workerCancel context.CancelFunc
	workerWG     sync.WaitGroup
	workerSem    chan struct{} // capacity = number of workers

	// jobQueue feeds work to available workers.
	jobQueue chan Job

	// numWorkers is the size of the worker pool.
	numWorkers int

	// clock provides injectable time.
	clock Clock

	// cursor holds the engine-side cursor model.
	cursor cursorState

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

	// logger is the slog logger for lifecycle event, panics, and profiler
	// spans. Defaults to io.Discard (silent) unless wired by the caller.
	logger *slog.Logger

	// macroTaskBudget caps macrotasks per frame (count).
	macroTaskBudget int
	// macroTaskDuration caps macrotasks per frame (wall clock).
	macroTaskDuration time.Duration

	// shutdownTimeout bounds how long Stop waits for pending jobs.
	shutdownTimeout time.Duration

	// lastCursorState tracks the hardware cursor state from the previous frame.
	lastCursorState cursorRecord

	// lastHardwareFocus tracks the focused element from the previous frame.
	lastHardwareFocus event.EventTarget

	// afterLayoutHooks are called once, after every layout+scroll phase, before
	// paint. Each call to OnAfterLayout appends one hook; all hooks fire in
	// registration order and are cleared after they fire (one-shot semantics).
	// Use this to read freshly-computed cursor positions from CursorState().
	afterLayoutHooks []func()

	// onFrameRenderedHooks are called after every frame is committed.
	onFrameRenderedHooks []func()

	// onStopHooks are called when the engine is stopping.
	onStopHooks []func()

	// extensions are terminal-specific protocol handlers.
	extensions []backend.TerminalExtension

	// eventBuffer holds raw events to be processed before the next frame.
	eventBuffer []event.RawEvent

	// Reusable buffers for event coalescing to avoid per-frame allocations.
	rawCoalescedBuf []event.RawEvent
	structuredBuf   []event.Event
	coalescedBuf    []event.Event
	wheelMap        map[event.EventTarget]*event.WheelEvent

	profilerMu sync.RWMutex
	jobIDs     sync.Map
	jobCounter uint64

	pipeline Pipeline

	tracer *trace.Tracer

	activeAnimations []animation.Animator
	lastFrameTime    time.Time

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
	// Logger receives structured log records from the engine. Defaults to a
	// no-op logger (io.Discard).
	Logger *slog.Logger
	// ShutdownTimeout bounds how long Stop waits for pending jobs before
	// forcing exit. Default is 5 seconds.
	ShutdownTimeout time.Duration
	// Profiler enables the high-level phase profiling and deep-tree tracing.
	Profiler bool
	// Extensions are terminal-specific protocol handlers.
	Extensions []backend.TerminalExtension
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
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	shutdownTimeout := opts.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 5 * time.Second
	}

	workerCtx, workerCancel := context.WithCancel(context.Background())

	e := &Engine{
		renderView:        render.NewRenderView(),
		document:          dom.NewDocument(),
		resolver:          style.NewResolver(),
		paintEngine:       paint.NewPaintEngine(),
		backend:           b,
		workerResults:     make(chan workerResult, numWorkers*2),
		workerCtx:         workerCtx,
		workerCancel:      workerCancel,
		workerSem:         make(chan struct{}, numWorkers),
		jobQueue:          make(chan Job, numWorkers*4),
		numWorkers:        numWorkers,
		clock:             clk,
		logger:            logger,
		macroTaskBudget:   macroTaskBudget,
		macroTaskDuration: macroTaskDuration,
		shutdownTimeout:   shutdownTimeout,
		mouseMode:         MouseModeClick,
		closeCh:           make(chan struct{}),
		wheelMap:          make(map[event.EventTarget]*event.WheelEvent),
		pipeline:          &StandardPipeline{},
		extensions:        opts.Extensions,
	}

	e.dispatcher = event.NewDispatcher()

	// Link document to render view
	e.document.SetRenderObject(e.renderView)
	e.renderView.SetLogicalNode(e.document)
	e.renderView.SetViewportSize(b.Size())

	// Mark document for initial sync
	e.document.MarkNeedsSync()
	e.focusManager = focus.NewManager(e.document, e.dispatcher)
	e.document.SetFocusManager(e.focusManager)
	e.synthesizer = event.NewSynthesizer(e, e, event.SynthesizerOptions{
		ScrollableResolver: e.resolveScrollable,
	})

	if opts.Profiler {
		e.tracer = trace.NewTracer()
		e.pipeline = &ProfilingPipeline{wrapped: e.pipeline}
	}

	// Probe capabilities from the backend.
	e.caps = b.Caps()

	// Start worker goroutines.
	for i := range numWorkers {
		e.workerWG.Add(1)
		go e.runWorker(workerCtx, i+1)
	}

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

// Cursor returns the CursorController that widgets use to drive cursor state.
// Changes take effect at the next Sync phase.
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
	e.logger.Info("engine: set title", slog.String("title", s))
}

// Bell emits a BEL character. It is a no-op when Caps.Bell is false.
//
// Bell must be called from the main goroutine.
func (e *Engine) Bell() {
	if !e.caps.Bell {
		return
	}
	e.logger.Info("engine: bell")
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

// OnAfterLayout registers a one-shot callback that fires once, after the next
// layout and scroll-into-view phase completes but before the paint phase.
// This is the correct place to read CursorState() when an accurate, freshly
// computed position is required (e.g. updating a status bar from a keydown
// listener). The hook is called exactly once and then discarded.
func (e *Engine) OnAfterLayout(fn func()) {
	e.afterLayoutHooks = append(e.afterLayoutHooks, fn)
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

// Submit enqueues job onto the worker pool. The job's Run method executes on
// a worker goroutine; its OnComplete method is dispatched onto the main thread
// via the microtask queue so it observes a consistent main-thread state.
//
// Submit must be called from the main goroutine.
func (e *Engine) Submit(j Job) {
	if tracer := e.Tracer(); tracer != nil {
		e.profilerMu.Lock()
		e.jobCounter++
		jobNum := e.jobCounter
		e.profilerMu.Unlock()

		jobID := fmt.Sprintf("job-%d", jobNum)
		e.jobIDs.Store(j, jobID)

		name := jobName(j)
		end := tracer.BeginThread("JobSubmit:"+name+":"+jobID, 1)
		end()
	}
	e.jobQueue <- j
}

// Post schedules fn as a microtask. Microtasks run on the main thread and are
// drained (until the queue is empty) during each frame, between macrotask
// iterations and at the end of the task phase.
//
// Post is safe to call from any goroutine.
func (e *Engine) Post(fn func()) {
	e.microQueue = append(e.microQueue, fn)
}

// PostMacro schedules fn as a macrotask. Macrotasks are drained once per
// frame, subject to the configured count and duration budgets.
//
// PostMacro is safe to call from any goroutine.
func (e *Engine) PostMacro(fn func()) {
	e.macroQueue = append(e.macroQueue, fn)
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
//  1. Drain worker results (onto microtask queue).
//  2. Sync Phase.
//  3. Task Phase (macrotasks + microtasks).
//  4. Style phase.
//  5. Layout phase.
//  6. Paint phase.
//  7. Commit.
//
// Frame must be called from the main goroutine.
func (e *Engine) Frame() {
	defer e.recoverFrame()

	e.profilerMu.RLock()
	pipe := e.pipeline
	tracer := e.tracer
	e.profilerMu.RUnlock()

	var endFrame func() = noop
	if tracer != nil {
		endFrame = tracer.BeginThread("Frame", 1)
	}
	defer endFrame()

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
				e.activeAnimations = append(e.activeAnimations[:i], e.activeAnimations[i+1:]...)
			}
		}

		if len(e.activeAnimations) == 0 {
			e.lastFrameTime = time.Time{}
		}
	} else {
		e.lastFrameTime = time.Time{}
	}

	e.drainWorkerResults()

	pipe.Sync(e)
	pipe.Tasks(e)
	pipe.Style(e)
	layoutRan := pipe.Layout(e)
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
	Shape   cursor.Shape
}

func (e *Engine) updateHardwareCursor(layoutRan bool) bool {
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
		e.logger.Info("engine: determining cursor for focused target")
		ro = focused.RenderObject()
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
						if el, ok := focused.(dom.Element); ok {
							rawX, rawY := el.Scroll()
							maxSX, maxSY := layout.MaxScroll(ro.Fragment())
							scrollX = max(0, min(rawX, maxSX))
							scrollY = max(0, min(rawY, maxSY))
						}
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
						if clip.Contains(cursorPos) {
							next.Visible = true
							next.X = cursorPos.X
							next.Y = cursorPos.Y
							next.Shape = state.Shape
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
			if ro != nil {
				comp := ro.Style()
				if comp != nil && comp.CursorColor != nil && comp.CursorColor != style.TerminalDefault {
					e.backend.SetCursorColor(comp.CursorColor)
				}
			}
			e.backend.ShowCursor(true)
		} else {
			e.backend.ShowCursor(false)
		}
		return true
	}
	return false
}

// drainWorkerResults drains all available completed job callbacks from the
// worker results channel and posts each as a microtask.
func (e *Engine) drainWorkerResults() {
	for {
		select {
		case r := <-e.workerResults:
			job := r.job
			err := r.err
			e.microQueue = append(e.microQueue, func() {
				job.OnComplete(nil, err)
			})
		default:
			return
		}
	}
}

// drainMacroTasks drains the macrotask queue subject to count and wall-clock
// budgets, flushing microtasks between each macrotask.
func (e *Engine) drainMacroTasks() {
	deadline := e.clock.Now().Add(e.macroTaskDuration)
	drained := 0
	for len(e.macroQueue) > 0 {
		if drained >= e.macroTaskBudget {
			break
		}
		if e.clock.Now().After(deadline) {
			break
		}
		task := e.macroQueue[0]
		e.macroQueue = e.macroQueue[1:]
		task()
		drained++
		e.drainMicroTasks()
	}
}

// drainMicroTasks empties the microtask queue. Any microtasks posted by a
// microtask callback are processed in the same drain call.
func (e *Engine) drainMicroTasks() {
	for len(e.microQueue) > 0 {
		task := e.microQueue[0]
		e.microQueue = e.microQueue[1:]
		task()
	}
}

// syncRenderTree walks the logical DOM and ensures the render tree matches
// its structure.
func (e *Engine) syncRenderTree(n dom.Node, ro render.Object) {
	if n.NeedsSync() {
		e.diffChildren(n, ro)
	} else if n.ChildNeedsSync() {
		for child := n.FirstLayoutChild(); child != nil; child = n.NextLayoutSibling(child) {
			if childRO := child.RenderObject(); childRO != nil {
				e.syncRenderTree(child, childRO)
			}
		}
	}
	n.ClearSyncFlags()
}

func (e *Engine) syncOverlays(d dom.Document) {
	var overlayROs []render.Object
	for overlayEl := range d.Overlays() {
		childRO := overlayEl.RenderObject()
		if childRO == nil {
			childRO = e.createRenderObject(overlayEl)
			overlayEl.SetRenderObject(childRO)
		}

		// If the child was already there, it might still need internal sync.
		if overlayEl.NeedsSync() || overlayEl.ChildNeedsSync() {
			e.syncRenderTree(overlayEl, childRO)
		}

		overlayROs = append(overlayROs, childRO)
	}
	e.renderView.SetOverlays(overlayROs)
}

// diffChildren synchronizes the children of n into the render object ro.
func (e *Engine) diffChildren(n dom.Node, parentRO render.Object) {
	// Map existing render children.
	existing := make(map[render.Object]struct{})
	for childRO := range parentRO.Children() {
		existing[childRO] = struct{}{}
	}

	var lastRO render.Object
	for child := n.FirstLayoutChild(); child != nil; child = n.NextLayoutSibling(child) {
		childRO := child.RenderObject()
		if childRO == nil {
			childRO = e.createRenderObject(child)
			child.SetRenderObject(childRO)
		}

		delete(existing, childRO)

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
		if child.NeedsSync() || child.ChildNeedsSync() {
			e.syncRenderTree(child, childRO)
		}

		lastRO = childRO
	}

	// Remove orphaned render objects.
	for orphaned := range existing {
		render.Unlink(orphaned)
	}
}

// createRenderObject creates a new render object for the given logical node.
// It also recursively creates render objects for any existing children.
func (e *Engine) createRenderObject(n dom.Node) render.Object {
	var ro render.Object
	target := n.EventTarget()

	if cp := unwrapProvider(n); cp != nil {
		ro = cp.CreateRenderObject()
	} else {
		// Fallback for nodes that don't implement CustomObjectProvider.
		ro = render.NewBox(n, target)
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

// reapDetached was removed in favor of the Sync phase.

// recoverFrame is a deferred function that catches panics in the frame loop.
// It restores the terminal, logs the panic and stack trace, and re-panics so
// the process exits with a usable terminal.
func (e *Engine) recoverFrame() {
	v := recover()
	if v == nil {
		return
	}
	e.backend.Restore()
	e.logger.Error("engine: panic in frame loop",
		slog.Any("panic", v),
		slog.String("stack", string(debug.Stack())),
	)
	panic(v)
}

// runWorker is the goroutine function for each worker. It reads jobs from
// jobQueue, executes them, and posts results back to the main thread via
// workerResults.
//
// runWorker exits when workerCtx is cancelled (i.e., when Stop() is called).
func (e *Engine) runWorker(ctx context.Context, workerID int) {
	defer e.workerWG.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-e.jobQueue:
			e.executeJob(ctx, j, workerID)
		}
	}
}

// executeJob runs j on the calling goroutine (a worker), catching panics and
// posting the result back to the microtask queue.
func (e *Engine) executeJob(ctx context.Context, j Job, workerID int) {
	var (
		result any
		jobErr error
	)
	var endRun func() = noop
	if v, ok := e.jobIDs.LoadAndDelete(j); ok {
		if tracer := e.Tracer(); tracer != nil {
			jobID := v.(string)
			endRun = tracer.BeginThread("JobRun:"+jobName(j)+":"+jobID, workerID+1)
		}
	}

	func() {
		defer func() {
			if v := recover(); v != nil {
				stack := string(debug.Stack())
				e.logger.Error("engine: panic in job",
					slog.Any("panic", v),
					slog.String("stack", stack),
				)
				jobErr = fmt.Errorf("job panic: %v", v)
			}
		}()
		jobErr = j.Run(ctx)
	}()
	_ = result

	endRun()

	e.workerResults <- workerResult{job: j, err: jobErr}
}

func noop() {}

func jobName(j Job) string {
	if j == nil {
		return "nil"
	}
	return fmt.Sprintf("%T", j)
}

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

		if ro := n.RenderObject(); ro != nil {
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
		RawText   []string    `json:"raw_text"`
		DOMTree   *nodeDump   `json:"dom_tree"`
		Overlays  []*nodeDump `json:"overlays,omitempty"`
		Cursor    cursorRecord
		Fragments *layout.Fragment
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
		e.workerCancel()

		done := make(chan struct{})
		go func() {
			e.workerWG.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(e.shutdownTimeout):
			e.logger.Warn("engine: stop timed out waiting for workers")
		}
		e.backend.Restore()
	})
}

type systemClipboard struct {
	backend backend.Backend
}

var _ event.ClipboardProvider = (*systemClipboard)(nil)

func (s *systemClipboard) Name() string { return "system" }

func (s *systemClipboard) SetClipboard(text string) {
	for _, ext := range s.backend.Extensions() {
		if cp, ok := ext.(event.ClipboardProvider); ok {
			cp.SetClipboard(text)
		}
	}
}

func (s *systemClipboard) RequestClipboard() {
	for _, ext := range s.backend.Extensions() {
		if cp, ok := ext.(event.ClipboardProvider); ok {
			cp.RequestClipboard()
		}
	}
}

// Run starts the engine's main event loop. It blocks until Stop is called or
// the backend signals exit.
func (e *Engine) Run(ctx context.Context) error {
	if err := e.backend.Start(); err != nil {
		return err
	}

	// 1. Initialize terminal extensions.
	writer := e.backend.Writer()
	// Sync engine extensions to backend if they were provided to Engine.Options
	if len(e.extensions) > 0 {
		if uvb, ok := e.backend.(interface {
			SetExtensions([]backend.TerminalExtension)
		}); ok {
			uvb.SetExtensions(e.extensions)
		}
	}

	slog.Info("engine: initializing terminal extensions", "count", len(e.extensions))
	for i, ext := range e.extensions {
		slog.Info("engine: initializing extension", "index", i, "type", fmt.Sprintf("%T", ext))
		ext.Init(writer)
	}

	// 2. Setup high-level services for the Document.
	e.document.SetClipboardProvider(&systemClipboard{backend: e.backend})

	// ... rest of Run ...

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
func (e *Engine) ProcessRawEvent(raw event.RawEvent) {
	e.processRawEvent(raw)
}

func (e *Engine) drainEvents() {
	if len(e.eventBuffer) == 0 {
		return
	}

	batchCounter++

	// 1. Coalesce raw events before synthesis to save allocations.
	rawCoalesced := e.coalesceRawEvents(e.eventBuffer)
	e.eventBuffer = e.eventBuffer[:0]

	// 2. Synthesize coalesced raw events into structured events.
	e.structuredBuf = e.structuredBuf[:0]
	for _, raw := range rawCoalesced {
		handledByExt := false
		for _, ext := range e.extensions {
			if handled, ev := ext.HandleEvent(raw); handled {
				slog.Debug("engine: event handled by extension", "event_type", fmt.Sprintf("%T", raw), "ext_type", fmt.Sprintf("%T", ext))
				if ev != nil {
					e.structuredBuf = append(e.structuredBuf, ev)
				}
				handledByExt = true
				break
			}
		}

		if !handledByExt {
			e.structuredBuf = append(e.structuredBuf, e.synthesizer.Process(raw)...)
		}
	}

	// 3. Coalesce structured events (handles wheel aggregation per target).
	coalesced := e.coalesceEvents(e.structuredBuf)

	// 4. Dispatch.
	for _, ev := range coalesced {
		e.dispatchEvent(ev)
	}
}

func (e *Engine) coalesceRawEvents(events []event.RawEvent) []event.RawEvent {
	if len(events) <= 1 {
		return events
	}

	// Strategy:
	// - Keep the LAST MouseMove.
	// - Accumulate consecutive Wheel events IF they are at the same coordinate and have same modifiers.
	// - Keep all other events (clicks, keys) in order.

	e.rawCoalescedBuf = e.rawCoalescedBuf[:0]

	// Find index of the absolute last mouse move in the whole batch.
	lastMoveIdx := -1
	for i := len(events) - 1; i >= 0; i-- {
		if m, ok := events[i].(*event.RawMouseEvent); ok && m.Move && m.DeltaX == 0 && m.DeltaY == 0 {
			lastMoveIdx = i
			break
		}
	}

	for i, ev := range events {
		m, isMouse := ev.(*event.RawMouseEvent)
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
				if prev, ok := e.rawCoalescedBuf[len(e.rawCoalescedBuf)-1].(*event.RawMouseEvent); ok {
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
	default:
		// For generic events (paste, etc), dispatch to focused element.
		target := e.focusManager.Current()
		if target == nil {
			// Fallback to document for global events.
			target = e.document
		}

		if target != nil {
			path := nodeAncestorPath(target)
			e.dispatcher.Dispatch(ev, path)
		}
	}
}

func (e *Engine) processRawEvent(raw event.RawEvent) {
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
	ro := target.RenderObject()
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

	ro := el.RenderObject()
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
		path := nodeAncestorPath(node)
		e.dispatcher.Dispatch(ev, path)

		// Move focus to the clicked node if it is focusable and the event
		// was a mousedown that was not cancelled by a listener. Focus on
		// mousedown (not click) matches browser behaviour: the element
		// becomes focused as soon as the button is pressed, so that key
		// events fired before the button is released already land on the
		// right target. Listeners may call ev.PreventDefault() on the
		// mousedown to opt out.
		if ev.Type() == event.EventMouseDown && !ev.DefaultPrevented() {
			e.focusManager.Focus(node, focus.ReasonPointer)
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

	nodeOrder := e.computeNodeOrder()
	source := &selectionSourceAdapter{sel: sel}
	rs := render.ResolveSelection(root, source, nodeOrder)
	if len(rs) == 0 {
		return nil
	}
	ps := make([]paint.SelectionRect, len(rs))
	for i, r := range rs {
		ps[i] = paint.SelectionRect{
			Rect: r.Rect,
			FG:   r.FG,
			BG:   r.BG,
		}
	}
	return ps
}

func (e *Engine) computeNodeOrder() map[any]render.NodeOrder {
	order := make(map[any]render.NodeOrder)
	count := 0
	var walk func(dom.Node) int
	walk = func(n dom.Node) int {
		if n == nil {
			return count
		}

		// Consistently unwrap to the canonical base node for identity.
		identity := any(n)
		curr := n
		for {
			if u := curr.Unwrap(); u != nil && u != curr {
				curr = u
				identity = u
				continue
			}
			break
		}

		first := count
		if _, ok := order[identity]; !ok {
			order[identity] = render.NodeOrder{First: count, Last: count}
			count++
		}

		for child := n.FirstLayoutChild(); child != nil; child = n.NextLayoutSibling(child) {
			walk(child)
		}

		last := count - 1
		o := order[identity]
		o.First = first
		o.Last = last
		order[identity] = o

		return count
	}
	walk(e.document)
	return order
}

type selectionSourceAdapter struct {
	sel dom.Selection
}

func (a *selectionSourceAdapter) RangeCount() int {
	return a.sel.RangeCount()
}

func (a *selectionSourceAdapter) GetRangeAt(idx int) render.SelectionRange {
	r := a.sel.GetRangeAt(idx)
	if r == nil {
		return nil
	}
	return &selectionRangeAdapter{r: r}
}

type selectionRangeAdapter struct {
	r dom.Range
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
	ro := target.RenderObject()
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

func (e *Engine) shouldRunFrame() bool {
	if e.frameRequested {
		return true
	}
	if !e.nextFrameAt.IsZero() && e.clock.Now().After(e.nextFrameAt) {
		return true
	}
	// Check for dirty DOM or render tree.
	return e.document.NeedsSync() || e.document.ChildNeedsSync() || e.renderView.Flags() != 0
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
