package kitex

import "github.com/masterkeysrd/kite/dom"

type hookState[T any] struct {
	value T
	get   func() T
	set   func(T)
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
