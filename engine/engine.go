package engine

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/masterkeysrd/kite/backend"
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
//
// See ADR-0007 for the pipeline design.
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

	// Link the document root to the render view so element adoption walks can find the tree root.
	e.document.SetRenderObject(e.renderView)
	e.renderView.SetLogicalNode(e.document)

	e.renderView.SetViewportSize(b.Size())

	e.dispatcher = event.NewDispatcher()
	e.focusManager = focus.NewManager(e.renderView, e.dispatcher)
	e.synthesizer = event.NewSynthesizer(e, e, event.SynthesizerOptions{})

	// Link logical document to the root render view.
	e.document.SetRenderObject(e.renderView)

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
}

// FocusedTarget returns the currently focused event target, or nil.
// It satisfies the event.FocusReader interface.
func (e *Engine) FocusedTarget() event.EventTarget {
	if e.focusManager == nil {
		return nil
	}
	cur := e.focusManager.Current()
	if cur == nil {
		return nil
	}
	return cur.EventTarget()
}

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

// ancestorPath returns the ancestor chain from the tree root down to n
// (inclusive), ordered root → n. This is the path format required by
// event.Dispatcher.Dispatch.
func ancestorPath(n render.Object) []event.EventTarget {
	var chain []event.EventTarget
	for cur := n; cur != nil; cur = cur.Parent() {
		if et := cur.EventTarget(); et != nil {
			chain = append(chain, et)
		}
	}
	// Reverse to get root → n order.
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}
	return chain
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

// RequestFrame schedules a frame wake-up as soon as possible (after at least
// MinFrameInterval has elapsed since the last frame). Consecutive calls within
// MinFrameInterval coalesce.
//
// RequestFrame is safe to call from any goroutine.
func (e *Engine) RequestFrame() {
	e.frameRequested = true
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

	// 1 & 2. Drain macrotasks (budget-capped), draining microtasks between
	// each macrotask and at the end.
	e.drainMacroTasks()

	// 3. Drain microtasks until empty.
	e.drainMicroTasks()

	root := e.renderView

	// 4. Style phase — gated on DirtyStyle | ChildNeedsStyle.
	if root.Flags()&(render.DirtyStyle|render.ChildNeedsStyle) != 0 {
		style.ResolveTree(e.resolver, root)
	}

	// 5. Layout phase — gated on DirtyLayout | DirtyStructure | ChildNeedsLayout.
	if root.Flags()&(render.DirtyLayout|render.DirtyStructure|render.ChildNeedsLayout) != 0 {
		viewport := root.ViewportSize()
		e.logger.Info("engine: layout phase", "viewport", viewport)

		start := e.clock.Now()
		render.LayoutPhase(root, viewport)

		for _, overlay := range root.Overlays() {
			render.LayoutPhase(overlay, viewport)
		}
		duration := e.clock.Now().Sub(start)
		e.logger.Info("engine: layout complete", "duration_ms", duration.Milliseconds())

		// Reap detached render objects in the layout phase.
		e.reapDetached(root)
		for _, overlay := range root.Overlays() {
			e.reapDetached(overlay)
		}

		root.ClearDirtyRecursive(render.DirtyLayout | render.DirtyStructure | render.ChildNeedsLayout)
		root.MarkDirty(render.DirtyPaint)
	}

	e.logger.Info("engine: checking paint phase", "flags", root.Flags())

	// 6. Paint phase — gated on DirtyPaint | DirtyScroll | ChildNeedsPaint.
	if root.Flags()&(render.DirtyPaint|render.DirtyScroll|render.ChildNeedsPaint) != 0 {
		surface := e.backend.BeginFrame()
		e.logger.Info("engine: painting main content")
		e.paintEngine.Paint(root.Fragment(), surface)
		for _, overlay := range root.Overlays() {
			if overlay.Flags()&(render.DirtyPaint|render.DirtyScroll|render.ChildNeedsPaint) != 0 {
				e.logger.Info("engine: painting overlay")
				e.paintEngine.Paint(overlay.Fragment(), surface)
			}
		}
		// 7. Sync — hand the frame to the render goroutine.
		if err := e.backend.EndFrame(); err != nil {
			e.logger.Error("engine: EndFrame error", slog.Any("error", err))
		}
		root.ClearDirtyRecursive(render.DirtyPaint | render.DirtyScroll | render.ChildNeedsPaint)
		e.logger.Info("engine: frame committed", slog.Uint64("version", e.frameVersion))
		e.frameVersion++
	}

	e.nextFrameAt = time.Time{}
	e.frameRequested = false
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

// reapDetached walks the render tree rooted at root and removes any detached
// render objects from their parents. This is the Reap phase (ADR-0007).
func (e *Engine) reapDetached(root render.Object) {
	e.reapSubtree(root)
}

// reapSubtree recursively walks the subtree and unlinks detached children.
func (e *Engine) reapSubtree(obj render.Object) {
	child := obj.FirstChild()
	for child != nil {
		next := child.NextSibling()
		if child.IsDetached() {
			render.Unlink(child)
		} else {
			e.reapSubtree(child)
		}
		child = next
	}
}

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

// Stop shuts down the engine, waiting for workers to exit.
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

func (e *Engine) processRawEvent(raw event.RawEvent) {
	e.logger.Info("engine: received raw event", slog.Any("event", raw))
	evts := e.synthesizer.Process(raw)
	for _, ev := range evts {
		switch evt := ev.(type) {
		case *event.MouseEvent:
			e.dispatchMouseEvent(evt)
		case *event.KeyEvent:
			e.dispatchKeyEvent(evt)
		case *event.ResizeEvent:
			e.handleResize(evt)
		default:
			// For generic events (paste, etc), dispatch to focused element.
			if focused := e.focusManager.Current(); focused != nil {
				path := ancestorPath(focused)
				e.dispatcher.Dispatch(ev, path)
			}
		}
	}
}

func (e *Engine) dispatchMouseEvent(ev *event.MouseEvent) {
	target := ev.Target()
	if target == nil {
		return
	}
	if node, ok := target.(dom.Node); ok {
		path := nodeAncestorPath(node)
		e.dispatcher.Dispatch(ev, path)
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
		path = ancestorPath(focused)
	} else {
		// Fallback: Dispatch to document if nothing is focused
		path = []event.EventTarget{e.document}
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

func (e *Engine) shouldRunFrame() bool {
	if e.frameRequested {
		return true
	}
	if !e.nextFrameAt.IsZero() && e.clock.Now().After(e.nextFrameAt) {
		return true
	}
	// Check for dirty render tree.
	return e.renderView.Flags() != 0
}
