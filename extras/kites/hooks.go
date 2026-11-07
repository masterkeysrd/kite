package kites

import (
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/masterkeysrd/kite/extras/kitex"
)

var hookTickSeq atomic.Uint64

type storeHookInst[T any, U any] struct {
	mu          sync.Mutex
	selector    func(T) U
	isEqual     func(U, U) bool
	snapshot    U
	hasSnapshot bool
}

// Use integrates the external kites.Store with the kitex VDOM tree.
// It retrieves the current slice of the store using the provided selector,
// and subscribes to store changes using a thread-safe external-store bridge
// (modeled after React's useSyncExternalStore).
//
// Selector outputs may be comparable primitives, structs, slices, or maps.
// If the selected slice has not changed between updates, Use bails out without
// marking the component dirty or scheduling a VDOM re-render.
func Use[T any, U any](s *Store[T], selector func(T) U) U {
	return UseWith(s, selector, nil)
}

// UseStore subscribes the component to the entire state of s and returns the
// current state snapshot.
func UseStore[T any](s *Store[T]) T {
	return UseWith(s, identity[T], nil)
}

// UseWith integrates the external kites.Store with the kitex VDOM tree using a
// custom equality function isEqual to determine whether the selected slice changed.
func UseWith[T any, U any](s *Store[T], selector func(T) U, isEqual func(a, b U) bool) U {
	instRef := kitex.UseRef[*storeHookInst[T, U]](nil)
	if instRef.Current == nil {
		instRef.Current = &storeHookInst[T, U]{
			isEqual: makeDefaultEqual[U](),
		}
	}
	inst := instRef.Current

	_, setTick := kitex.UseState[uint64](0)

	inst.mu.Lock()
	currentSlice := selector(s.Get())
	inst.selector = selector
	if isEqual != nil {
		inst.isEqual = isEqual
	}
	inst.snapshot = currentSlice
	inst.hasSnapshot = true
	inst.mu.Unlock()

	kitex.UseLayoutEffectCleanup(func() func() {
		checkForUpdates := func() {
			inst.mu.Lock()
			if inst.selector == nil || inst.isEqual == nil {
				inst.mu.Unlock()
				return
			}
			next := inst.selector(s.Get())
			if inst.hasSnapshot && inst.isEqual(inst.snapshot, next) {
				inst.mu.Unlock()
				return
			}
			inst.snapshot = next
			inst.hasSnapshot = true
			inst.mu.Unlock()

			setTick(hookTickSeq.Add(1))
		}

		unsubscribe := s.Subscribe(func(_, _ T) {
			checkForUpdates()
		})

		// Re-check immediately after subscribing to close the tearing window
		// between s.Get() during Render and s.Subscribe() during layout effect.
		checkForUpdates()

		return unsubscribe
	}, []any{s})

	return currentSlice
}

func identity[T any](v T) T {
	return v
}

func makeDefaultEqual[U any]() func(a, b U) bool {
	if reflect.TypeFor[U]().Comparable() {
		return func(a, b U) bool {
			return any(a) == any(b)
		}
	}
	return func(a, b U) bool {
		return reflect.DeepEqual(a, b)
	}
}
