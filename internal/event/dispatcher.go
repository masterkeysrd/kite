package event

import (
	"github.com/masterkeysrd/kite/event"
)

// Dispatcher performs 3-phase (capture → target → bubble) event dispatch.
//
// Dispatcher is not safe for concurrent use.
type Dispatcher struct{}

var _ event.Dispatcher = (*Dispatcher)(nil)

// NewDispatcher creates a Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

// Dispatch routes e through the ancestor chain described by path.
// path must be ordered root → target (index 0 = root, last = target).
// Dispatch modifies e's internal phase/target/currentTarget fields in-place.
func (d *Dispatcher) Dispatch(e event.Event, path []event.EventTarget) {
	if len(path) == 0 {
		return
	}

	// Step i's target is the deepest ancestor of the real target that step i can see.
	targets := computeRetargeting(path)
	realTarget := path[len(path)-1]

	ie, ok := any(e).(event.InternalEvent)
	if !ok {
		// If the event doesn't implement InternalEvent, we can't dispatch it
		// properly because we can't set its phase/target.
		return
	}

	// Phase 1: Capture — root → target's parent.
	ie.SetPhase(event.PhaseCapture)
	for i, et := range path[:len(path)-1] {
		if e.PropagationStopped() {
			return
		}
		ie.SetTarget(targets[i])
		ie.SetCurrentTarget(et)
		et.DispatchTo(e)
	}

	if e.PropagationStopped() {
		return
	}

	// Phase 2: Target — both capture and bubble listeners fire.
	ie.SetPhase(event.PhaseTarget)
	ie.SetTarget(realTarget)
	ie.SetCurrentTarget(realTarget)
	realTarget.DispatchToTarget(e)

	if e.PropagationStopped() || !e.Bubbles() {
		return
	}

	// Phase 3: Bubble — target's parent → root.
	ie.SetPhase(event.PhaseBubble)
	for i := len(path) - 2; i >= 0; i-- {
		if e.PropagationStopped() {
			return
		}
		et := path[i]
		ie.SetTarget(targets[i])
		ie.SetCurrentTarget(et)
		et.DispatchTo(e)
	}
}

// DispatchWheel routes a WheelEvent through the ancestor chain, stopping at
// the first ancestor that implements Scrollable. Non-scrollable ancestors are
// skipped silently.
//
// path must be ordered root → target.
func (d *Dispatcher) DispatchWheel(e *event.WheelEvent, path []event.EventTarget, scrollables map[event.EventTarget]event.Scrollable) {
	if len(path) == 0 {
		return
	}

	targets := computeRetargeting(path)
	realTarget := path[len(path)-1]

	ie, ok := any(e).(event.InternalEvent)
	if !ok {
		return
	}

	// Capture.
	ie.SetPhase(event.PhaseCapture)
	for i, et := range path[:len(path)-1] {
		if e.PropagationStopped() {
			return
		}
		ie.SetTarget(targets[i])
		ie.SetCurrentTarget(et)
		et.DispatchTo(e)
	}

	if e.PropagationStopped() {
		return
	}

	// Target.
	ie.SetPhase(event.PhaseTarget)
	ie.SetTarget(realTarget)
	ie.SetCurrentTarget(realTarget)
	realTarget.DispatchToTarget(e)

	// Check if target itself is Scrollable.
	if sc, ok := scrollables[realTarget]; ok {
		sc.OnWheel(e)
		return
	}

	if e.PropagationStopped() {
		return
	}

	// Bubble — stop at first Scrollable ancestor.
	ie.SetPhase(event.PhaseBubble)
	for i := len(path) - 2; i >= 0; i-- {
		if e.PropagationStopped() {
			return
		}
		et := path[i]
		ie.SetTarget(targets[i])
		if sc, ok := scrollables[et]; ok {
			ie.SetCurrentTarget(et)
			sc.OnWheel(e)
			return
		}
		ie.SetCurrentTarget(et)
		et.DispatchTo(e)
	}
}

func computeRetargeting(path []event.EventTarget) []event.EventTarget {
	res := make([]event.EventTarget, len(path))
	if len(path) == 0 {
		return res
	}

	// We walk from target to root.
	// The current visible target starts as the deepest node.
	currentTarget := path[len(path)-1]

	for i := len(path) - 1; i >= 0; i-- {
		res[i] = currentTarget

		if i > 0 {
			parent := path[i-1]
			child := path[i]

			isBoundary := false

			// 1. UA Shadow Boundary: The child's visible identity is the parent.
			if et := child.EventTarget(); et != nil && et == parent {
				isBoundary = true
			}

			// 2. Overlay Boundary: The child is an overlay anchored to the parent.
			if ov, ok := child.(interface{ Anchor() any }); ok {
				if ov.Anchor() == parent {
					isBoundary = true
				}
			}

			if isBoundary {
				// Crossing a boundary outward. The new visible target for ancestors
				// is the boundary node itself.
				currentTarget = parent
				// If the boundary node itself has a different user-visible identity
				// (e.g., it's also a wrapper), we use that.
				if et := parent.EventTarget(); et != nil {
					currentTarget = et
				}
			}
		}
	}

	return res
}

// HitTester resolves the event target at a screen-space point. This is
// typically implemented by the engine.
type HitTester interface {
	HitTest(x, y int) event.EventTarget
}
