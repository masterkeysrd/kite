package dom

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
)

// defaultScroller is the framework's internal WheelEvent handler for elements
// that indicate scroll containerness via their computed style.
type defaultScroller struct {
	host dom.Element
}

// OnWheel implements event.Scrollable.
func (s *defaultScroller) OnWheel(e *event.WheelEvent) {
	var maxSX, maxSY int
	hasView := false
	if doc := s.host.OwnerDocument(); doc != nil {
		if view := doc.DefaultView(); view != nil {
			maxSX, maxSY = view.GetMaxScroll(s.host)
			hasView = true
		}
	}

	if !hasView {
		s.host.ScrollBy(e.DeltaX, e.DeltaY)
		e.StopPropagation()
		return
	}

	oldX, oldY := s.host.Scroll()
	oldX = max(0, min(oldX, maxSX))
	oldY = max(0, min(oldY, maxSY))

	targetX := oldX + e.DeltaX
	targetY := oldY + e.DeltaY

	newX := max(0, min(targetX, maxSX))
	newY := max(0, min(targetY, maxSY))

	if newX != oldX || newY != oldY {
		s.host.ScrollTo(newX, newY)
		e.StopPropagation()
	}
}

// DefaultScroller returns a new Scrollable implementation for the given host.
func DefaultScroller(host dom.Element) event.Scrollable {
	return &defaultScroller{host: host}
}
