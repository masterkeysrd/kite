package event

import (
	"sync/atomic"

	"github.com/masterkeysrd/kite/render"
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
	target *EventTarget
}

// Cancel removes the listener. Idempotent.
func (s *subscription) Cancel() {
	if s.reg.cancelled.CompareAndSwap(false, true) {
		s.target.removeRegistration(s.reg.id)
	}
}

// regIDGen is the global registration ID counter.
var regIDGen atomic.Uint64

func nextRegID() uint64 { return regIDGen.Add(1) }

// --- Dispatcher --------------------------------------------------------------

// HitTester resolves the render object at a screen-space point. This is
// typically implemented by the engine.
type HitTester interface {
	HitTest(x, y int) render.Object
}

// AncestorWalker returns the ancestor chain from target up to the root
// (inclusive), in child-to-root order.
type AncestorWalker func(target render.Object) []render.Object

// EventTargetResolver maps a render.Object to its EventTarget, or nil if
// the object has no target registered.
type EventTargetResolver func(render.Object) EventTarget

// Dispatcher performs 3-phase (capture → target → bubble) event dispatch.
// It does not own the event-target registry; callers supply a resolver.
//
// Dispatcher is not safe for concurrent use.
type Dispatcher struct {
	resolver EventTargetResolver
}

// NewDispatcher creates a Dispatcher that uses resolver to look up the
// EventTarget for each render object in the ancestor chain.
func NewDispatcher(resolver EventTargetResolver) *Dispatcher {
	return &Dispatcher{resolver: resolver}
}

// Dispatch routes e through the ancestor chain described by path.
// path must be ordered root → target (index 0 = root, last = target).
// Dispatch modifies e's internal phase/target/currentTarget fields in-place.
func (d *Dispatcher) Dispatch(e Event, path []render.Object) {
	if len(path) == 0 {
		return
	}
	target := path[len(path)-1]
	e.setTarget(target)

	// Phase 1: Capture — root → target's parent.
	e.setPhase(PhaseCapture)
	for _, obj := range path[:len(path)-1] {
		if e.PropagationStopped() {
			return
		}
		et := d.resolver(obj)
		if et == nil {
			continue
		}
		e.setCurrentTarget(obj)
		et.dispatchTo(e)
	}

	if e.PropagationStopped() {
		return
	}

	// Phase 2: Target — both capture and bubble listeners fire.
	e.setPhase(PhaseTarget)
	e.setCurrentTarget(target)
	et := d.resolver(target)
	if et != nil {
		// Invoke capture-registered listeners first, then bubble.
		et.dispatchToTarget(e)
	}

	if e.PropagationStopped() || !e.Bubbles() {
		return
	}

	// Phase 3: Bubble — target's parent → root.
	e.setPhase(PhaseBubble)
	for i := len(path) - 2; i >= 0; i-- {
		if e.PropagationStopped() {
			return
		}
		obj := path[i]
		et := d.resolver(obj)
		if et == nil {
			continue
		}
		e.setCurrentTarget(obj)
		et.dispatchTo(e)
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
	// Dispatch normally through capture/target; bubble stops at first Scrollable.
	target := path[len(path)-1]
	e.setTarget(target)

	// Capture.
	e.setPhase(PhaseCapture)
	for _, obj := range path[:len(path)-1] {
		if e.PropagationStopped() {
			return
		}
		et := d.resolver(obj)
		if et == nil {
			continue
		}
		e.setCurrentTarget(obj)
		et.dispatchTo(e)
		// et.dispatchTo(e)
	}

	if e.PropagationStopped() {
		return
	}

	// Target.
	e.setPhase(PhaseTarget)
	e.setCurrentTarget(target)
	if et := d.resolver(target); et != nil {
		et.dispatchToTarget(e)
	}
	// Check if target itself is Scrollable.
	if sc, ok := scrollables[target]; ok {
		sc.OnWheel(e)
		return
	}

	if e.PropagationStopped() {
		return
	}

	// Bubble — stop at first Scrollable ancestor.
	e.setPhase(PhaseBubble)
	for i := len(path) - 2; i >= 0; i-- {
		if e.PropagationStopped() {
			return
		}
		obj := path[i]
		if sc, ok := scrollables[obj]; ok {
			e.setCurrentTarget(obj)
			sc.OnWheel(e)
			return
		}
		et := d.resolver(obj)
		if et == nil {
			continue
		}
		e.setCurrentTarget(obj)
		et.dispatchTo(e)
	}
}

type EventTarget interface {
	AddEventListener(typ EventType, fn Listener, opts ...Option) Subscription
}

// eventTarget manages event listeners for a single render object. It should
// be embedded in (or stored alongside) render objects that need to receive
// event.
//
// EventTarget is not safe for concurrent use; it must be accessed from the
// single main-loop goroutine.
type eventTarget struct {
	listeners map[EventType][]*registration
}

// AddEventListener registers fn as a listener for event of type typ on this
// target. Options control the phase (capture vs bubble), auto-cancellation
// (once), and the passive hint. The returned Subscription can be used to
// remove the listener without pointer comparison.
func (t *eventTarget) AddEventListener(typ EventType, fn Listener, opts ...Option) Subscription {
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

// removeRegistration removes the registration with the given id. Called by
// subscription.Cancel.
func (t *eventTarget) removeRegistration(id uint64) {
	for typ, regs := range t.listeners {
		for i, r := range regs {
			if r.id == id {
				t.listeners[typ] = append(regs[:i], regs[i+1:]...)
				return
			}
		}
	}
}

// dispatchTo fires listeners on this target for the given event. It
// respects the phase and the once flag. Cancelled registrations are
// purged after each call.
func (t *eventTarget) dispatchTo(e Event) {
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

// dispatchToTarget invokes capture-registered listeners followed by
// bubble-registered listeners for the target phase. This mirrors the
// DOM specification where the target phase fires capture listeners then
// bubble listeners in registration order.
func (t *eventTarget) dispatchToTarget(e Event) {
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
