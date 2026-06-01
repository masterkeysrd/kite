package event

import (
	"sync/atomic"

	"github.com/masterkeysrd/kite/event"
)

// registration holds a single listener registration on an EventTarget.
type registration struct {
	id        uint64
	fn        event.Listener
	capture   bool
	once      bool
	passive   bool
	cancelled atomic.Bool
}

func (r *registration) SetCapture(v bool) { r.capture = v }
func (r *registration) SetOnce(v bool)    { r.once = v }
func (r *registration) SetPassive(v bool) { r.passive = v }

// subscription wraps a *registration and the target it belongs to.
type subscription struct {
	reg    *registration
	target event.EventTarget
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

// Target manages event listeners for a single object. It should
// be embedded in (or stored alongside) objects that need to receive
// events.
//
// Target is not safe for concurrent use; it must be accessed from the
// single main-loop goroutine.
type Target struct {
	listeners map[event.EventType][]*registration
}

var _ event.EventTarget = (*Target)(nil)

// EventTarget implements event.EventTarget.
func (t *Target) EventTarget() event.EventTarget {
	return nil
}

// AddEventListener registers fn as a listener for event of type typ on this
// target. Options control the phase (capture vs bubble), auto-cancellation
// (once), and the passive hint. The returned Subscription can be used to
// remove the listener without pointer comparison.
func (t *Target) AddEventListener(typ event.EventType, fn event.Listener, opts ...event.Option) event.Subscription {
	reg := &registration{
		id: nextRegID(),
		fn: fn,
	}
	for _, o := range opts {
		o(reg)
	}
	if t.listeners == nil {
		t.listeners = make(map[event.EventType][]*registration)
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
func (t *Target) DispatchTo(e event.Event) {
	if isInteractionEvent(e.Type()) && isDisabled(e.CurrentTarget()) {
		return
	}
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
		if phase == event.PhaseCapture && !reg.capture {
			continue
		}
		if phase == event.PhaseBubble && reg.capture {
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
func (t *Target) DispatchToTarget(e event.Event) {
	if isInteractionEvent(e.Type()) && isDisabled(e.CurrentTarget()) {
		return
	}
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

type disableable interface {
	IsDisabled() bool
}

func isDisabled(et event.EventTarget) bool {
	if et == nil {
		return false
	}
	if userTarget := et.EventTarget(); userTarget != nil {
		if d, ok := userTarget.(disableable); ok && d.IsDisabled() {
			return true
		}
	}
	if d, ok := et.(disableable); ok && d.IsDisabled() {
		return true
	}
	return false
}

func isInteractionEvent(typ event.EventType) bool {
	switch typ {
	case event.EventClick,
		event.EventMouseDown,
		event.EventMouseUp,
		event.EventMouseMove,
		event.EventDrag,
		event.EventWheel,
		event.EventKeyDown,
		event.EventKeyUp,
		event.EventKeyPress:
		return true
	default:
		return false
	}
}
