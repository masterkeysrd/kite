package kitex

import (
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/masterkeysrd/kite/dom"
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
			val, _ := activeNode.getHookState(idx)
			return val.(*hookState[T]).value
		}

		set := func(newVal T) {
			ref.mu.Lock()
			activeNode := ref.node
			ref.mu.Unlock()

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
	getState, setState := UseState[S](initial)
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
		return UseMemo[T](func() T { return callback }, []any{seq})
	}
	return UseMemo[T](func() T { return callback }, deps)
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
	postMacroFn          func(func())
)

// SetPostMacroFn sets the post-macro enqueuing function used by kitex.
func SetPostMacroFn(fn func(func())) {
	postMacroFn = fn
}

func scheduleEffectFlush() {
	if postMacroFn != nil {
		postMacroFn(flushPendingEffects)
	}
}

// UseEffect schedules a post-commit side effect that runs asynchronously after the frame is committed.
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
	changed := false
	if hs.deps == nil || deps == nil || !depsEqual(hs.deps, deps) {
		changed = true
	}
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
	changed := false
	if hs.deps == nil || deps == nil || !depsEqual(hs.deps, deps) {
		changed = true
	}
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
	changed := false
	if hs.deps == nil || deps == nil || !depsEqual(hs.deps, deps) {
		changed = true
	}
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
	changed := false
	if hs.deps == nil || deps == nil || !depsEqual(hs.deps, deps) {
		changed = true
	}
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
}
