package dom

import (
	"github.com/masterkeysrd/kite/event"
)

// defaultScroller is the framework's internal WheelEvent handler for elements
// that indicate scroll containerness via their computed style.
type defaultScroller struct {
	host Element
}

// OnWheel implements event.Scrollable.
func (s *defaultScroller) OnWheel(e *event.WheelEvent) {
	s.host.ScrollBy(e.DeltaX, e.DeltaY)
	e.StopPropagation()
}

// DefaultScroller returns a new Scrollable implementation for the given host.
func DefaultScroller(host Element) event.Scrollable {
	return &defaultScroller{host: host}
}
