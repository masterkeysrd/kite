package kitex

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/terminal"
)

type hookState[T any] struct {
	value T
	get   func() T
	set   func(T)
}

type hookValuer interface {
	getValue() any
}

func (h *hookState[T]) getValue() any {
	return h.value
}

// UseState initializes a state variable on first render, persists it across render cycles,
// and returns a getter and a setter. Setting the state flags the component dirty.
// If called outside of a component render cycle, it panics.
func UseState[T any](initial T) (func() T, func(T)) {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseState must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)
	idx := comp.incrementHookIndex()

	stateVal, exists := comp.getHookState(idx)
	if !exists {
		ref := comp.getRef()
		if ref == nil {
			panic("ComponentNode ref is not initialized. Ensure component is rendered via Instantiate/Update.")
		}

		hs := &hookState[T]{
			value: initial,
		}

		get := func() T {
			ref.mu.Lock()
			activeNode := ref.node
			ref.mu.Unlock()
			if activeNode == nil {
				return hs.value
			}
			val, _ := activeNode.getHookState(idx)
			return val.(*hookState[T]).value
		}

		set := func(newVal T) {
			ref.mu.Lock()
			activeNode := ref.node
			ref.mu.Unlock()

			if activeNode == nil {
				return
			}

			val, _ := activeNode.getHookState(idx)
			hsPtr := val.(*hookState[T])
			hsPtr.value = newVal
			activeNode.MarkDirty()
		}

		hs.get = get
		hs.set = set
		comp.setHookState(idx, hs)
		return get, set
	}

	hs := stateVal.(*hookState[T])
	return hs.get, hs.set
}

// refSetter is an unexported interface implemented by Ref[T] where T is a DOM Node.
type refSetter interface {
	set(dom.Node)
}

// RefObject is the container that holds the mutable Current value.
type RefObject[T any] struct {
	Current T
}

func (r *RefObject[T]) getValue() any {
	return r.Current
}

func (r *RefObject[T]) set(node dom.Node) {
	if val, ok := any(node).(T); ok {
		r.Current = val
	}
}

// Ref is a type alias for *RefObject[T].
type Ref[T any] = *RefObject[T]

// CreateRef creates a Ref outside of the render cycle.
func CreateRef[T any]() Ref[T] {
	return &RefObject[T]{}
}

// UseRef returns a persistent Ref using the component hook state mechanism.
// Modifying the Ref does not trigger component dirty/re-render.
func UseRef[T any](initial T) Ref[T] {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseRef must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)
	idx := comp.incrementHookIndex()

	stateVal, exists := comp.getHookState(idx)
	if !exists {
		r := &RefObject[T]{Current: initial}
		comp.setHookState(idx, r)
		return r
	}
	return stateVal.(*RefObject[T])
}

// memoHookState holds the cached result and dependency slice for UseMemo.
type memoHookState[T any] struct {
	value T
	deps  []any
}

func (m *memoHookState[T]) getValue() any { return m.value }

// UseMemo memoizes the result of factory and re-evaluates it only when the
// deps slice changes between renders. Dependency comparison uses
// reflect.DeepEqual on each element in the slice.
//
// Example:
//
//	expensive := kitex.UseMemo(func() []Row { return buildRows(data) }, []any{data})
func UseMemo[T any](factory func() T, deps []any) T {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseMemo must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)
	idx := comp.incrementHookIndex()

	stateVal, exists := comp.getHookState(idx)
	if !exists {
		// First render: evaluate and cache.
		val := factory()
		comp.setHookState(idx, &memoHookState[T]{value: val, deps: deps})
		return val
	}

	hs := stateVal.(*memoHookState[T])
	if !depsEqual(hs.deps, deps) {
		// Deps changed: re-evaluate and update the cached entry.
		hs.value = factory()
		hs.deps = deps
	}
	return hs.value
}

// depsEqual compares two dependency slices element by element using
// reflect.DeepEqual. Lengths must match; order matters.
func depsEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// UseReducer initializes a state variable with a reducer function, returning
// a getter function and a dispatch function. Setting the state via dispatch flags
// the component dirty and schedules a re-render.
func UseReducer[S, A any](reducer func(S, A) S, initial S) (func() S, func(A)) {
	getState, setState := UseState(initial)
	dispatch := func(action A) {
		setState(reducer(getState(), action))
	}
	return getState, dispatch
}

var useCallbackSeq uint64

// UseCallback memoizes a function callback reference across renders, re-evaluating
// it only when the dependency slice changes.
func UseCallback[T any](callback T, deps []any) T {
	if deps == nil {
		seq := atomic.AddUint64(&useCallbackSeq, 1)
		return UseMemo(func() T { return callback }, []any{seq})
	}
	return UseMemo(func() T { return callback }, deps)
}

type effectHookState struct {
	deps      []any
	cleanup   func()
	isLayout  bool
	pending   bool
	simpleFn  func()
	cleanupFn func() func()
}

var (
	pendingLayoutEffects []*effectHookState
	layoutEffectsBuffer  []*effectHookState
	pendingEffects       []*effectHookState
	effectsBuffer        []*effectHookState
	effectsMutex         sync.Mutex
	scheduler            terminal.Scheduler
)

func setInternalScheduler(s terminal.Scheduler) {
	scheduler = s
}

// PostMacro schedules a function to run as a macrotask on the main thread.
func PostMacro(fn func()) {
	if scheduler != nil {
		scheduler.QueueMacrotask(fn)
	} else {
		// Fallback or run directly if not registered (e.g. in tests)
		go fn()
	}
}

func scheduleEffectFlush() {
	if scheduler != nil {
		scheduler.QueueMacrotask(flushPendingEffects)
	}
}

// UseDocument returns a function that retrieves the logical document root associated with the current component.
// It retrieves the real DOM node and returns its OwnerDocument when called.
func UseDocument() func() dom.Document {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseDocument must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)

	ref := comp.getRef()

	return func() dom.Document {
		if ref == nil {
			return nil
		}

		ref.mu.Lock()
		activeNode := ref.node
		ref.mu.Unlock()

		if activeNode == nil {
			return nil
		}

		reals := activeNode.realNodes()
		if len(reals) > 0 && reals[0] != nil {
			return reals[0].OwnerDocument()
		}
		if parent := activeNode.getDOMParent(); parent != nil {
			return parent.OwnerDocument()
		}
		return nil
	}
}

// UseElement returns a function that retrieves the underlying DOM node associated with the current component.
func UseElement() func() dom.Node {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseElement must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)

	ref := comp.getRef()

	return func() dom.Node {
		if ref == nil {
			return nil
		}

		ref.mu.Lock()
		activeNode := ref.node
		ref.mu.Unlock()

		if activeNode == nil {
			return nil
		}

		reals := activeNode.realNodes()
		if len(reals) > 0 {
			return reals[0]
		}
		return nil
	}
}

func UseEffect(effect func(), deps []any) {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseEffect must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)
	idx := comp.incrementHookIndex()

	stateVal, exists := comp.getHookState(idx)
	if !exists {
		hs := &effectHookState{
			deps:     deps,
			pending:  true,
			isLayout: false,
			simpleFn: effect,
		}
		comp.setHookState(idx, hs)
		effectsMutex.Lock()
		pendingEffects = append(pendingEffects, hs)
		effectsMutex.Unlock()
		scheduleEffectFlush()
		return
	}

	hs := stateVal.(*effectHookState)
	changed := hs.deps == nil || deps == nil || !depsEqual(hs.deps, deps)

	hs.simpleFn = effect // Update closure anyway
	if changed {
		hs.deps = deps
		hs.pending = true
		effectsMutex.Lock()
		pendingEffects = append(pendingEffects, hs)
		effectsMutex.Unlock()
		scheduleEffectFlush()
	}
}

// UseEffectCleanup schedules a post-commit side effect with a cleanup function.
func UseEffectCleanup(effect func() func(), deps []any) {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseEffectCleanup must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)
	idx := comp.incrementHookIndex()

	stateVal, exists := comp.getHookState(idx)
	if !exists {
		hs := &effectHookState{
			deps:      deps,
			pending:   true,
			isLayout:  false,
			cleanupFn: effect,
		}
		comp.setHookState(idx, hs)
		effectsMutex.Lock()
		pendingEffects = append(pendingEffects, hs)
		effectsMutex.Unlock()
		scheduleEffectFlush()
		return
	}

	hs := stateVal.(*effectHookState)
	changed := hs.deps == nil || deps == nil || !depsEqual(hs.deps, deps)

	hs.cleanupFn = effect // Update closure anyway
	if changed {
		hs.deps = deps
		hs.pending = true
		effectsMutex.Lock()
		pendingEffects = append(pendingEffects, hs)
		effectsMutex.Unlock()
		scheduleEffectFlush()
	}
}

// UseLayoutEffect schedules a layout side effect that runs synchronously after reconciliation.
func UseLayoutEffect(effect func(), deps []any) {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseLayoutEffect must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)
	idx := comp.incrementHookIndex()

	stateVal, exists := comp.getHookState(idx)
	if !exists {
		hs := &effectHookState{
			deps:     deps,
			pending:  true,
			isLayout: true,
			simpleFn: effect,
		}
		comp.setHookState(idx, hs)
		effectsMutex.Lock()
		pendingLayoutEffects = append(pendingLayoutEffects, hs)
		effectsMutex.Unlock()
		return
	}

	hs := stateVal.(*effectHookState)
	changed := hs.deps == nil || deps == nil || !depsEqual(hs.deps, deps)

	hs.simpleFn = effect // Update closure anyway
	if changed {
		hs.deps = deps
		hs.pending = true
		effectsMutex.Lock()
		pendingLayoutEffects = append(pendingLayoutEffects, hs)
		effectsMutex.Unlock()
	}
}

// UseLayoutEffectCleanup schedules a layout side effect with a cleanup function.
func UseLayoutEffectCleanup(effect func() func(), deps []any) {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseLayoutEffectCleanup must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)
	idx := comp.incrementHookIndex()

	stateVal, exists := comp.getHookState(idx)
	if !exists {
		hs := &effectHookState{
			deps:      deps,
			pending:   true,
			isLayout:  true,
			cleanupFn: effect,
		}
		comp.setHookState(idx, hs)
		effectsMutex.Lock()
		pendingLayoutEffects = append(pendingLayoutEffects, hs)
		effectsMutex.Unlock()
		return
	}

	hs := stateVal.(*effectHookState)
	changed := hs.deps == nil || deps == nil || !depsEqual(hs.deps, deps)

	hs.cleanupFn = effect // Update closure anyway
	if changed {
		hs.deps = deps
		hs.pending = true
		effectsMutex.Lock()
		pendingLayoutEffects = append(pendingLayoutEffects, hs)
		effectsMutex.Unlock()
	}
}

// flushPendingEffects runs all pending post-commit effect functions and registers their cleanups.
func flushPendingEffects() {
	if len(pendingEffects) == 0 {
		return
	}
	effectsMutex.Lock()
	queue := pendingEffects
	if cap(effectsBuffer) < len(queue) {
		effectsBuffer = make([]*effectHookState, 0, len(queue))
	}
	pendingEffects = effectsBuffer
	effectsBuffer = queue[:0]
	effectsMutex.Unlock()

	for _, state := range queue {
		if state.pending {
			if state.cleanup != nil {
				state.cleanup()
			}
			if state.simpleFn != nil {
				state.simpleFn()
			} else if state.cleanupFn != nil {
				state.cleanup = state.cleanupFn()
			}
			state.pending = false
		}
	}
	for i := range queue {
		queue[i] = nil
	}
}

// drainLayoutEffects runs all pending layout effect functions synchronously and registers their cleanups.
func drainLayoutEffects() {
	if len(pendingLayoutEffects) == 0 {
		return
	}
	effectsMutex.Lock()
	queue := pendingLayoutEffects
	if cap(layoutEffectsBuffer) < len(queue) {
		layoutEffectsBuffer = make([]*effectHookState, 0, len(queue))
	}
	pendingLayoutEffects = layoutEffectsBuffer
	layoutEffectsBuffer = queue[:0]
	effectsMutex.Unlock()

	for _, state := range queue {
		if state.pending {
			if state.cleanup != nil {
				state.cleanup()
			}
			if state.simpleFn != nil {
				state.simpleFn()
			} else if state.cleanupFn != nil {
				state.cleanup = state.cleanupFn()
			}
			state.pending = false
		}
	}
	for i := range queue {
		queue[i] = nil
	}
}

// UseFocus returns whether the referenced DOM element currently has focus.
func UseFocus(ref Ref[dom.Element]) bool {
	focused, setFocused := UseState(false)

	UseEffectCleanup(func() func() {
		el := ref.Current
		if el == nil {
			return nil
		}

		onFocus := func(e event.Event) {
			setFocused(true)
		}
		onBlur := func(e event.Event) {
			setFocused(false)
		}

		s1 := el.AddEventListener(event.EventFocus, onFocus)
		s2 := el.AddEventListener(event.EventBlur, onBlur)

		return func() {
			s1.Cancel()
			s2.Cancel()
		}
	}, []any{ref.Current})

	return focused()
}

// UseKeyboard registers a scoped keyboard handler that listens for event.EventKeyPress
// on the document. The handler is invoked with the typed event.KeyEvent.
func UseKeyboard(handler func(event.KeyEvent), deps []any) {
	getDoc := UseDocument()

	UseEffectCleanup(func() func() {
		doc := getDoc()
		if doc == nil {
			return nil
		}

		onKeyDown := func(e event.Event) {
			if ke, ok := e.(*event.KeyEvent); ok {
				handler(*ke)
			}
		}

		s := doc.AddEventListener(event.EventKeyDown, onKeyDown)

		return func() {
			s.Cancel()
		}
	}, deps)
}

// UseTimeout schedules a callback to run after the specified delay.
// The timer is automatically cancelled if the component is unmounted
// or if the delay/dependencies change before the timeout elapses.
func UseTimeout(handler func(), delay time.Duration, deps []any) {
	getDoc := UseDocument()

	UseEffectCleanup(func() func() {
		doc := getDoc()
		if doc == nil {
			return nil
		}
		var sched terminal.Scheduler
		if doc.Terminal() != nil {
			sched = doc.Terminal().Scheduler()
		} else {
			sched = scheduler
		}
		if sched == nil {
			return nil
		}

		t := time.AfterFunc(delay, func() {
			sched.QueueMacrotask(handler)
		})

		return func() {
			t.Stop()
		}
	}, append([]any{delay}, deps...))
}

// UseInterval schedules a callback to run repeatedly at the specified interval.
// The interval timer is automatically cancelled when the component is unmounted
// or if the interval/dependencies change.
func UseInterval(handler func(), interval time.Duration, deps []any) {
	getDoc := UseDocument()

	UseEffectCleanup(func() func() {
		doc := getDoc()
		if doc == nil {
			return nil
		}
		var sched terminal.Scheduler
		if doc.Terminal() != nil {
			sched = doc.Terminal().Scheduler()
		} else {
			sched = scheduler
		}
		if sched == nil {
			return nil
		}

		ticker := time.NewTicker(interval)
		done := make(chan struct{})

		go func() {
			for {
				select {
				case <-ticker.C:
					sched.QueueMacrotask(handler)
				case <-done:
					return
				}
			}
		}()

		return func() {
			ticker.Stop()
			close(done)
		}
	}, append([]any{interval}, deps...))
}

// UseViewportSize returns the current viewport size (terminal dimensions)
// and registers a listener to trigger a re-render when the terminal is resized.
func UseViewportSize() geom.Size {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseViewportSize must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)

	getDoc := func() dom.Document {
		for _, node := range comp.realNodes() {
			if node != nil {
				if doc := node.OwnerDocument(); doc != nil {
					return doc
				}
			}
		}
		return nil
	}

	var initialSize geom.Size
	if doc := getDoc(); doc != nil {
		if view := doc.DefaultView(); view != nil {
			initialSize = view.ViewportSize()
		}
	}

	size, setSize := UseState(initialSize)

	UseEffectCleanup(func() func() {
		doc := getDoc()
		if doc == nil {
			return nil
		}

		// Initial sync
		if view := doc.DefaultView(); view != nil {
			currSize := view.ViewportSize()
			if currSize != size() {
				setSize(currSize)
			}
		}

		// Listen to resize events on the document
		sub := doc.AddEventListener(event.EventResize, func(ev event.Event) {
			if view := doc.DefaultView(); view != nil {
				setSize(view.ViewportSize())
			}
		})

		// Return the cleanup function
		return func() {
			sub.Cancel()
		}
	}, nil)

	return size()
}
