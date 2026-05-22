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

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/focus/spatial"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/paint"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

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

	// afterLayoutHooks are called once, after every layout+scroll phase, before
	// paint. Each call to OnAfterLayout appends one hook; all hooks fire in
	// registration order and are cleared after they fire (one-shot semantics).
	// Use this to read freshly-computed cursor positions from CursorState().
	afterLayoutHooks []func()

	// onFrameRenderedHooks are called after every frame is committed.
	onFrameRenderedHooks []func()

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

	// Probe capabilities from the backend.
	e.caps = b.Caps()

	// Start worker goroutines.
	for range numWorkers {
		e.workerWG.Add(1)
		go e.runWorker(workerCtx)
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

// FocusManager returns the engine's focus.Manager so that tests and
// application code can drive focus programmatically (e.g. simulate a
// mousedown-to-focus or query the currently focused node).
func (e *Engine) FocusManager() *focus.Manager { return e.focusManager }

// HitTest walks the render tree at point (x, y) and returns the topmost
// event target at that position. It tests overlays (topmost-first) before
// falling through to the main tree. Returns nil when no target is hit.
func (e *Engine) HitTest(x, y int) event.EventTarget {
	p := layout.Point{X: x, Y: y}

	// Walk overlays from the end (topmost) to start.
	overlays := e.renderView.Overlays()
	for i := len(overlays) - 1; i >= 0; i-- {
		if hit := hitTestFragment(overlays[i].Fragment(), p); hit != nil {
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
func hitTestFragment(frag *layout.Fragment, p layout.Point) render.Object {
	if frag == nil {
		return nil
	}
	if !(layout.Rect{Size: frag.Size}).Contains(p) {
		return nil
	}
	// Walk children in reverse paint order (last child is topmost).
	for i := len(frag.Children) - 1; i >= 0; i-- {
		link := frag.Children[i]
		// Translate point into child's coordinate space
		childPoint := layout.Point{
			X: p.X - link.Offset.X,
			Y: p.Y - link.Offset.Y,
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
//  2. Drain macrotasks (budget-capped).
//  3. Drain microtasks.
//  4. Style phase (gated).
//  5. Layout phase (gated) + reap detached render objects.
//  6. Paint phase (gated).
//  7. Sync (EndFrame).
//
// Frame must be called from the main goroutine.
func (e *Engine) Frame() {
	defer e.recoverFrame()

	// 0. Collect completed job results into the microtask queue.
	e.drainWorkerResults()

	// 1. Sync Phase — walk logical DOM and project to render tree.
	if e.document.NeedsSync() || e.document.ChildNeedsSync() {
		e.syncRenderTree(e.document, e.renderView)
		e.syncOverlays(e.document)
	}

	// 1b. Autofocus Phase — if nothing is focused, try to find the first
	// focusable element. This ensures that the cursor and focus styles are
	// visible on the very first render without requiring a keystroke.
	if e.focusManager.Current() == nil {
		e.focusManager.Next()
	}

	// 2 & 3. Drain macrotasks (budget-capped), draining microtasks between
	// each macrotask and at the end.
	e.drainMacroTasks()

	// 3. Drain microtasks until empty.
	e.drainMicroTasks()

	root := e.renderView

	// 4. Style phase — gated on DirtyStyle | ChildNeedsStyle.
	if root.Flags()&(render.DirtyStyle|render.ChildNeedsStyle) != 0 {
		style.ResolveTree(e.resolver, root)
	}
	for _, overlay := range root.Overlays() {
		if overlay.Flags()&(render.DirtyStyle|render.ChildNeedsStyle) != 0 {
			style.ResolveTree(e.resolver, overlay)
		}
	}

	// 5. Layout phase — gated on DirtyLayout | ChildNeedsLayout.
	if root.Flags()&(render.DirtyLayout|render.ChildNeedsLayout) != 0 {
		viewport := root.ViewportSize()
		e.logger.Info("engine: layout phase", "viewport", viewport)

		start := e.clock.Now()
		render.LayoutPhase(root, viewport)

		duration := e.clock.Now().Sub(start)
		e.logger.Info("engine: layout complete", "duration_ms", duration.Milliseconds())

		root.ClearDirtyRecursive(render.DirtyLayout | render.ChildNeedsLayout)
		for _, overlay := range root.Overlays() {
			overlay.ClearDirtyRecursive(render.DirtyLayout | render.ChildNeedsLayout)
		}
	}

	// 5b. Auto-scroll phase — if an element is focused, ensure its cursor is visible.
	if focused := e.focusManager.Current(); focused != nil {
		if el, ok := focused.(dom.Element); ok {
			el.ScrollCursorIntoView()
		}
	}

	// 5c. After-layout hooks — fire once, then discard. These allow callers
	// (e.g. status-bar updates) to read freshly computed CursorState values.
	if len(e.afterLayoutHooks) > 0 {
		hooks := e.afterLayoutHooks
		e.afterLayoutHooks = nil
		for _, fn := range hooks {
			fn()
		}
	}

	// Always update the hardware cursor state after layout but before the
	// potential paint phase. This ensures that if a frame is produced, it
	// carries the most recent cursor position.
	cursorChanged := e.updateHardwareCursor()

	e.logger.Info("engine: checking paint phase", "flags", root.Flags())

	anyOverlayDirty := false
	for _, o := range root.Overlays() {
		if o.Flags()&(render.DirtyPaint|render.DirtyScroll|render.ChildNeedsPaint) != 0 {
			anyOverlayDirty = true
			break
		}
	}

	// 6. Paint phase — gated on DirtyPaint | DirtyScroll | ChildNeedsPaint
	// OR a cursor change (which requires a Flush even if no cells changed).
	if cursorChanged || anyOverlayDirty || root.Flags()&(render.DirtyPaint|render.DirtyScroll|render.ChildNeedsPaint) != 0 {
		surface := e.backend.BeginFrame()
		e.logger.Info("engine: painting main content")
		e.paintEngine.PaintFragment(root.Fragment(), layout.Point{}, surface)
		for _, overlay := range root.Overlays() {
			e.logger.Info("engine: painting overlay")
			offset := layout.Point{}
			if cs := overlay.ComputedStyle(); cs != nil {
				offset.X = cs.Margin.Left
				offset.Y = cs.Margin.Top
			}
			e.paintEngine.PaintFragment(overlay.Fragment(), offset, surface)
		}
		e.paintEngine.ResolveBorders(surface)

		// 7. Sync — hand the frame to the render goroutine.
		if err := e.backend.EndFrame(); err != nil {
			e.logger.Error("engine: EndFrame error", slog.Any("error", err))
		}
		root.ClearDirtyRecursive(render.DirtyPaint | render.DirtyScroll | render.ChildNeedsPaint)
		for _, overlay := range root.Overlays() {
			overlay.ClearDirtyRecursive(render.DirtyPaint | render.DirtyScroll | render.ChildNeedsPaint)
		}
		e.logger.Info("engine: frame committed", slog.Uint64("version", e.frameVersion))
		e.frameVersion++
	}

	e.nextFrameAt = time.Time{}
	e.frameRequested = false

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

func (e *Engine) updateHardwareCursor() bool {
	next := cursorRecord{}

	focused := e.focusManager.Current()
	var ro render.Object
	if focused != nil {
		e.logger.Info("engine: determining cursor for focused target", slog.Any("target", focused))
		ro = focused.RenderObject()
		if ro != nil {
			provider, ok := ro.(cursor.Provider)
			if !ok {
				provider, ok = focused.(cursor.Provider)
			}
			if ok {
				state := provider.CursorState()
				if state.Visible {
					root := e.renderView.Fragment()
					if bounds, clip, found := layout.ScrolledAbsoluteBounds(root, ro); found {
						scrollX, scrollY := 0, 0
						if el, ok := focused.(dom.Element); ok {
							rawX, rawY := el.Scroll()
							maxSX, maxSY := layout.MaxScroll(ro.Fragment())
							scrollX = max(0, min(rawX, maxSX))
							scrollY = max(0, min(rawY, maxSY))
						}
						cursorPos := layout.Point{
							X: bounds.Origin.X + state.X - scrollX,
							Y: bounds.Origin.Y + state.Y - scrollY,
						}

						// Hardware cursor is inside the content box.
						// If the element itself clips, we must intersect with its content box.
						cs := ro.ComputedStyle()
						if cs.OverflowX != style.OverflowVisible || cs.OverflowY != style.OverflowVisible {
							bw := cs.Border.Widths()
							contentBox := layout.Rect{
								Origin: layout.Point{
									X: bounds.Origin.X + bw.Left + cs.Padding.Left,
									Y: bounds.Origin.Y + bw.Top + cs.Padding.Top,
								},
								Size: layout.Size{
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
		for child := range dom.LayoutChildren(n) {
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
	for child := range dom.LayoutChildren(n) {
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
func (e *Engine) runWorker(ctx context.Context) {
	defer e.workerWG.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-e.jobQueue:
			e.executeJob(ctx, j)
		}
	}
}

// executeJob runs j on the calling goroutine (a worker), catching panics and
// posting the result back to the microtask queue.
func (e *Engine) executeJob(ctx context.Context, j Job) {
	var (
		result any
		jobErr error
	)
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

	e.workerResults <- workerResult{job: j, err: jobErr}
}

func (e *Engine) Dump(path string) error {
	size := e.backend.Size()
	rawText := make([]string, size.Height)

	// Capture the current visible state by re-painting into a temporary buffer.
	// We cannot use e.backend.BeginFrame() because it returns an empty surface.
	fb := paint.NewFrameBuffer(0, 0, size.Width, size.Height)
	root := e.renderView
	e.paintEngine.PaintFragment(root.Fragment(), layout.Point{}, fb)
	for _, overlay := range root.Overlays() {
		offset := overlay.Offset()
		e.paintEngine.PaintFragment(overlay.Fragment(), offset, fb)
	}
	e.paintEngine.ResolveBorders(fb)

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

func (e *Engine) Stop() {
	e.closeOnce.Do(func() {
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

// Run starts the engine's main event loop. It blocks until Stop is called or
// the backend signals exit.
func (e *Engine) Run(ctx context.Context) error {
	if err := e.backend.Start(); err != nil {
		return err
	}
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
			e.processRawEvent(raw)
		case <-ticker.C:
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

func (e *Engine) processRawEvent(raw event.RawEvent) {
	evts := e.synthesizer.Process(raw)
	for _, ev := range evts {
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
			if focused := e.focusManager.Current(); focused != nil {
				path := nodeAncestorPath(focused)
				e.dispatcher.Dispatch(ev, path)
			}
		}
	}
}

func (e *Engine) dispatchWheelEvent(ev *event.WheelEvent) {
	target := ev.Target()
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
		ev.Local = layout.Point{
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

func isScrollContainer(cs *style.Computed) bool {
	return cs.OverflowX == style.OverflowScroll || cs.OverflowX == style.OverflowAuto || cs.OverflowX == style.OverflowHidden ||
		cs.OverflowY == style.OverflowScroll || cs.OverflowY == style.OverflowAuto || cs.OverflowY == style.OverflowHidden
}

func (e *Engine) dispatchMouseEvent(ev *event.MouseEvent) {
	target := ev.Target()
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
	e.backend.Resize(layout.Size{Width: ev.Width, Height: ev.Height})
	e.renderView.SetViewportSize(layout.Size{
		Width:  ev.Width,
		Height: ev.Height,
	})
	e.RequestFrame()
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
	ro := target.RenderObject()
	if ro == nil {
		return
	}
	root := e.renderView.Fragment()
	if bounds, _, found := layout.ScrolledAbsoluteBounds(root, ro); found {
		// ScrolledAbsoluteBounds returns the scrolled border-box.
		// Local coordinate in event should be relative to this scrolled box.
		ev.Local = layout.Point{
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
