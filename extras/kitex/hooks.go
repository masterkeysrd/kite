package kitex

import (
	"reflect"

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
