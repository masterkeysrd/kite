package event

import (
	"sync/atomic"
)

// Subscription is a cancellable event listener registration. Call Cancel to
// remove the listener from its target. Subscription values are safe to cancel
// from any goroutine; the actual removal is deferred to the next listener
// invocation to avoid map mutation under iteration.
type Subscription interface {
	// Cancel removes this listener registration. It is idempotent: calling
	// Cancel more than once is safe and has no additional effect.
	Cancel()
}

// Listener is a function that handles an Event.
type Listener func(Event)

// Option configures how a listener is registered.
type Option func(*registration)

// Capture returns an Option that registers the listener for the capture phase
// (default is bubble phase).
func Capture() Option {
	return func(r *registration) { r.capture = true }
}

// Once returns an Option that causes the listener to auto-cancel after its
// first invocation.
func Once() Option {
	return func(r *registration) { r.once = true }
}

// Passive returns an Option that hints the engine that the handler will never
// call PreventDefault (used for performance; not enforced).
func Passive() Option {
	return func(r *registration) { r.passive = true }
}

// registration holds a single listener registration on an EventTarget.
type registration struct {
	id        uint64
	fn        Listener
	capture   bool
	once      bool
	passive   bool
	cancelled atomic.Bool
}

// subscription wraps a *registration and the target it belongs to.
type subscription struct {
	reg    *registration
	target EventTarget
}

// Cancel removes the listener. Idempotent.
func (s *subscription) Cancel() {
	if s.reg.cancelled.CompareAndSwap(false, true) {
		s.target.RemoveRegistration(s.reg.id)
	}
}

// regIDGen is the global registration ID counter.
var regIDGen atomic.Uint64

func nextRegID() uint64 { return regIDGen.Add(1) }

// --- Dispatcher --------------------------------------------------------------

// Dispatcher performs 3-phase (capture → target → bubble) event dispatch.
//
// Dispatcher is not safe for concurrent use.
type Dispatcher struct{}

// NewDispatcher creates a Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

// Dispatch routes e through the ancestor chain described by path.
// path must be ordered root → target (index 0 = root, last = target).
// Dispatch modifies e's internal phase/target/currentTarget fields in-place.
func (d *Dispatcher) Dispatch(e Event, path []EventTarget) {
	if len(path) == 0 {
		return
	}

	// Step i's target is the deepest ancestor of the real target that step i can see.
	targets := computeRetargeting(path)
	realTarget := path[len(path)-1]

	// Phase 1: Capture — root → target's parent.
	e.SetPhase(PhaseCapture)
	for i, et := range path[:len(path)-1] {
		if e.PropagationStopped() {
			return
		}
		e.SetTarget(targets[i])
		e.SetCurrentTarget(et)
		et.DispatchTo(e)
	}

	if e.PropagationStopped() {
		return
	}

	// Phase 2: Target — both capture and bubble listeners fire.
	e.SetPhase(PhaseTarget)
	e.SetTarget(realTarget)
	e.SetCurrentTarget(realTarget)
	realTarget.DispatchToTarget(e)

	if e.PropagationStopped() || !e.Bubbles() {
		return
	}

	// Phase 3: Bubble — target's parent → root.
	e.SetPhase(PhaseBubble)
	for i := len(path) - 2; i >= 0; i-- {
		if e.PropagationStopped() {
			return
		}
		et := path[i]
		e.SetTarget(targets[i])
		e.SetCurrentTarget(et)
		et.DispatchTo(e)
	}
}

// DispatchWheel routes a WheelEvent through the ancestor chain, stopping at
// the first ancestor that implements Scrollable. Non-scrollable ancestors are
// skipped silently.
//
// path must be ordered root → target.
func (d *Dispatcher) DispatchWheel(e *WheelEvent, path []EventTarget, scrollables map[EventTarget]Scrollable) {
	if len(path) == 0 {
		return
	}

	targets := computeRetargeting(path)
	realTarget := path[len(path)-1]

	// Capture.
	e.SetPhase(PhaseCapture)
	for i, et := range path[:len(path)-1] {
		if e.PropagationStopped() {
			return
		}
		e.SetTarget(targets[i])
		e.SetCurrentTarget(et)
		et.DispatchTo(e)
	}

	if e.PropagationStopped() {
		return
	}

	// Target.
	e.SetPhase(PhaseTarget)
	e.SetTarget(realTarget)
	e.SetCurrentTarget(realTarget)
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
	e.SetPhase(PhaseBubble)
	for i := len(path) - 2; i >= 0; i-- {
		if e.PropagationStopped() {
			return
		}
		et := path[i]
		e.SetTarget(targets[i])
		if sc, ok := scrollables[et]; ok {
			e.SetCurrentTarget(et)
			sc.OnWheel(e)
			return
		}
		e.SetCurrentTarget(et)
		et.DispatchTo(e)
	}
}

func computeRetargeting(path []EventTarget) []EventTarget {
	res := make([]EventTarget, len(path))
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
	HitTest(x, y int) EventTarget
}

type EventTarget interface {
	AddEventListener(typ EventType, fn Listener, opts ...Option) Subscription
	DispatchTo(e Event)
	DispatchToTarget(e Event)
	RemoveRegistration(id uint64)
	// EventTarget returns the user-visible event target for this object.
	// For logical nodes in a UA shadow subtree, this returns the host element;
	// otherwise it returns the object itself.
	EventTarget() EventTarget
}

// Target manages event listeners for a single object. It should
// be embedded in (or stored alongside) objects that need to receive
// events.
//
// Target is not safe for concurrent use; it must be accessed from the
// single main-loop goroutine.
type Target struct {
	listeners map[EventType][]*registration
}

// EventTarget implements event.EventTarget.
func (t *Target) EventTarget() EventTarget {
	return nil
}

// AddEventListener registers fn as a listener for event of type typ on this
// target. Options control the phase (capture vs bubble), auto-cancellation
// (once), and the passive hint. The returned Subscription can be used to
// remove the listener without pointer comparison.
func (t *Target) AddEventListener(typ EventType, fn Listener, opts ...Option) Subscription {
	reg := &registration{
		id: nextRegID(),
		fn: fn,
	}
	for _, o := range opts {
		o(reg)
	}
	if t.listeners == nil {
		t.listeners = make(map[EventType][]*registration)
	}
	t.listeners[typ] = append(t.listeners[typ], reg)
	return &subscription{reg: reg, target: t}
}

// RemoveRegistration removes the registration with the given id.
func (t *Target) RemoveRegistration(id uint64) {
	for typ, regs := range t.listeners {
		for i, r := range regs {
			if r.id == id {
				t.listeners[typ] = append(regs[:i], regs[i+1:]...)
				return
			}
		}
	}
}

// DispatchTo fires listeners on this target for the given event. It
// respects the phase and the once flag. Cancelled registrations are
// purged after each call.
func (t *Target) DispatchTo(e Event) {
	typ := e.Type()
	regs := t.listeners[typ]
	if len(regs) == 0 {
		return
	}

	// Snapshot to avoid aliasing during mutation.
	snap := make([]*registration, len(regs))
	copy(snap, regs)

	phase := e.Phase()
	for _, reg := range snap {
		if reg.cancelled.Load() {
			continue
		}
		// Only fire on the correct phase.
		if phase == PhaseCapture && !reg.capture {
			continue
		}
		if phase == PhaseBubble && reg.capture {
			continue
		}
		reg.fn(e)
		if reg.once {
			reg.cancelled.Store(true)
		}
		if e.PropagationStopped() {
			break
		}
	}

	// Purge cancelled registrations.
	surviving := t.listeners[typ][:0]
	for _, reg := range t.listeners[typ] {
		if !reg.cancelled.Load() {
			surviving = append(surviving, reg)
		}
	}
	t.listeners[typ] = surviving
}

// DispatchToTarget invokes capture-registered listeners followed by
// bubble-registered listeners for the target phase. This mirrors the
// DOM specification where the target phase fires capture listeners then
// bubble listeners in registration order.
func (t *Target) DispatchToTarget(e Event) {
	typ := e.Type()
	regs := t.listeners[typ]
	if len(regs) == 0 {
		return
	}

	// Work on a copy so we don't alias the backing array.
	snap := make([]*registration, len(regs))
	copy(snap, regs)

	// Capture-phase registrations first.
	for _, reg := range snap {
		if !reg.capture || reg.cancelled.Load() {
			continue
		}
		reg.fn(e)
		if reg.once {
			reg.cancelled.Store(true)
		}
		if e.PropagationStopped() {
			break
		}
	}

	if !e.PropagationStopped() {
		// Bubble-phase registrations.
		for _, reg := range snap {
			if reg.capture || reg.cancelled.Load() {
				continue
			}
			reg.fn(e)
			if reg.once {
				reg.cancelled.Store(true)
			}
			if e.PropagationStopped() {
				break
			}
		}
	}

	// Purge cancelled registrations.
	surviving := t.listeners[typ][:0]
	for _, reg := range t.listeners[typ] {
		if !reg.cancelled.Load() {
			surviving = append(surviving, reg)
		}
	}
	t.listeners[typ] = surviving
}
