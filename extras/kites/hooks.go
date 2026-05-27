package kites

import (
	"github.com/masterkeysrd/kite/extras/kitex"
)

// Use integrates the external kites.Store with the kitex VDOM tree.
// It retrieves the current slice of the store using the provided selector,
// registers a state variable, and subscribes to the store changes.
// The subscription is established on mount and cleaned up on unmount using UseLayoutEffectCleanup.
// If the selected state has not changed (determined via comparison of selector(new) != selector(old)),
// it bails out and avoids triggering a VDOM re-render.
func Use[T any, U comparable](s *Store[T], selector func(T) U) U {
	// Retrieve initial slice value
	initialSlice := selector(s.Get())

	// Register local state in kitex
	getState, setState := kitex.UseState(initialSlice)

	// Subscribe to store changes. The effect runs on mount and cleans up on unmount.
	kitex.UseLayoutEffectCleanup(func() func() {
		unsubscribe := s.Subscribe(func(newVal T, oldVal T) {
			newSlice := selector(newVal)
			oldSlice := selector(oldVal)
			if newSlice != oldSlice {
				setState(newSlice)
			}
		})
		return unsubscribe
	}, []any{s})

	return getState()
}
