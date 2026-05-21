package event

import (
	"github.com/masterkeysrd/kite/layout"
)

// FocusReader returns the currently focused render object. The Synthesizer
// uses it to route key events and paste events.
type FocusReader interface {
	// FocusedObject returns the currently focused render object, or nil.
	FocusedTarget() EventTarget
}

// ClipboardBridge provides access to the system clipboard.
type ClipboardBridge interface {
	// GetClipboard returns the current clipboard text content.
	GetClipboard() string
	// SetClipboard stores text into the system clipboard.
	SetClipboard(text string)
}

// SynthesizerOptions configures a Synthesizer.
type SynthesizerOptions struct {
	// ClickRadius is the maximum cell distance that the mouse may move
	// between mousedown and mouseup and still be synthesized as a click.
	// Default is 3.
	ClickRadius int

	// Clipboard provides read/write access to the system clipboard.
	// If nil, clipboard events are synthesized without backend I/O.
	Clipboard ClipboardBridge

	// ScrollableResolver maps a target to its Scrollable, or nil.
	// Used by DispatchWheel.
	ScrollableResolver func(EventTarget) Scrollable
}

// Synthesizer converts raw backend input (RawEvent) into structured events
// ready for dispatch. Click and drag synthesis applies movement tolerance.
// Hit testing assigns the target render object.
//
// Synthesizer is not safe for concurrent use.
type Synthesizer struct {
	hit                HitTester
	focus              FocusReader
	clipboard          ClipboardBridge
	clickRadius        int
	scrollableResolver func(EventTarget) Scrollable

	// pendingDown is set when a mousedown is received; cleared on mouseup.
	pendingDown *MouseEvent
	// pendingDownPos is the screen position of the pending mousedown.
	pendingDownPos layout.Point
}

// NewSynthesizer creates a Synthesizer with the given HitTester, FocusReader,
// and options.
func NewSynthesizer(hit HitTester, focus FocusReader, opts SynthesizerOptions) *Synthesizer {
	radius := opts.ClickRadius
	if radius <= 0 {
		radius = 3
	}
	return &Synthesizer{
		hit:                hit,
		focus:              focus,
		clipboard:          opts.Clipboard,
		clickRadius:        radius,
		scrollableResolver: opts.ScrollableResolver,
	}
}

// ResolveScrollables walks the ancestor path of target and returns a map of
// targets to their resolved Scrollable implementations.
func (s *Synthesizer) ResolveScrollables(path []EventTarget) map[EventTarget]Scrollable {
	if s.scrollableResolver == nil {
		return nil
	}
	res := make(map[EventTarget]Scrollable)
	for _, et := range path {
		if sc := s.scrollableResolver(et); sc != nil {
			res[et] = sc
		}
	}
	return res
}

// Process converts a RawEvent into zero or more structured events. The
// returned slice may be empty (e.g. if no listeners are interested) or
// contain multiple events (e.g. a mouseup that produces both MouseUp and Click).
func (s *Synthesizer) Process(raw RawEvent) []Event {
	switch e := raw.(type) {
	case *RawKeyEvent:
		return s.processKey(e)
	case *RawMouseEvent:
		return s.processMouse(e)
	case *RawResizeEvent:
		return s.processResize(e)
	case *RawBracketedPaste:
		return s.processBracketedPaste(e)
	}
	return nil
}

// processKey converts a RawKeyEvent into a KeyEvent, and optionally a
// ClipboardEvent when Ctrl+C / Ctrl+X / Ctrl+V is pressed.
func (s *Synthesizer) processKey(raw *RawKeyEvent) []Event {
	typ := EventKeyDown
	if raw.Up {
		typ = EventKeyUp
	}
	ke := NewKeyEvent(typ, raw.Key)

	// Route to the focused element.
	if s.focus != nil {
		ke.setTarget(s.focus.FocusedTarget())
	}

	// Clipboard synthesis.
	var events []Event
	events = append(events, ke)

	switch {
	case raw.MatchString("ctrl+c"):
		if ce := s.synthesizeCopy(ke); ce != nil {
			events = append(events, ce)
		}
	case raw.MatchString("ctrl+x"):
		if ce := s.synthesizeCut(ke); ce != nil {
			events = append(events, ce)
		}
	case raw.MatchString("ctrl+v"):
		events = append(events, s.synthesizePasteFromClipboard())
	}

	return events
}

// synthesizeCopy creates a ClipboardEvent{Copy} if the focused element has a
// selection.
func (s *Synthesizer) synthesizeCopy(_ *KeyEvent) *ClipboardEvent {
	focused := s.focus.FocusedTarget()
	if focused == nil {
		return nil
	}
	sp, ok := focused.(SelectionProvider)
	if !ok {
		return nil
	}
	text := sp.SelectedText()
	if text == "" {
		return nil
	}
	if s.clipboard != nil {
		s.clipboard.SetClipboard(text)
	}
	ce := NewClipboardEvent(EventCopy, ClipboardCopy, text)
	ce.setTarget(focused)
	return ce
}

// synthesizeCut creates a ClipboardEvent{Cut} if the focused element has a
// selection.
func (s *Synthesizer) synthesizeCut(_ *KeyEvent) *ClipboardEvent {
	focused := s.focus.FocusedTarget()
	if focused == nil {
		return nil
	}
	sp, ok := focused.(SelectionProvider)
	if !ok {
		return nil
	}
	text := sp.SelectedText()
	if text == "" {
		return nil
	}
	if s.clipboard != nil {
		s.clipboard.SetClipboard(text)
	}
	ce := NewClipboardEvent(EventCut, ClipboardCut, text)
	ce.setTarget(focused)
	return ce
}

// synthesizePasteFromClipboard creates a ClipboardEvent{Paste} from the
// system clipboard.
func (s *Synthesizer) synthesizePasteFromClipboard() *ClipboardEvent {
	var data string
	if s.clipboard != nil {
		data = s.clipboard.GetClipboard()
	}
	ce := NewClipboardEvent(EventPaste, ClipboardPaste, data)
	if s.focus != nil {
		ce.setTarget(s.focus.FocusedTarget())
	}
	return ce
}

// processMouse converts a RawMouseEvent into mouse events, synthesizing click
// and drag as appropriate.
func (s *Synthesizer) processMouse(raw *RawMouseEvent) []Event {
	pos := layout.Point{X: raw.X, Y: raw.Y}
	hitTarget := s.hitTest(pos)

	var events []Event

	if raw.DeltaX != 0 || raw.DeltaY != 0 {
		// Wheel event.
		we := NewWheelEvent(pos, raw.DeltaX, raw.DeltaY, raw.Mod)
		we.setTarget(hitTarget)
		events = append(events, we)
		return events
	}

	if !raw.Up && !raw.Move {
		// Mouse down.
		me := NewMouseEvent(EventMouseDown, pos, raw.Button, raw.Mod)
		me.Hit = HitResult{Target: hitTarget}
		me.setTarget(hitTarget)
		s.pendingDown = me
		s.pendingDownPos = pos
		events = append(events, me)
		return events
	}

	if raw.Move {
		// Mouse move.
		me := NewMouseEvent(EventMouseMove, pos, raw.Button, raw.Mod)
		me.Hit = HitResult{Target: hitTarget}
		me.setTarget(hitTarget)
		events = append(events, me)

		// Check for drag: down pending and movement beyond tolerance.
		if s.pendingDown != nil && s.beyondTolerance(s.pendingDownPos, pos) {
			drag := NewMouseEvent(EventDrag, pos, raw.Button, raw.Mod)
			drag.Hit = HitResult{Target: hitTarget}
			drag.setTarget(hitTarget)
			events = append(events, drag)
			s.pendingDown = nil // drag cancels pending click
		}
		return events
	}

	if raw.Up {
		// Mouse up.
		me := NewMouseEvent(EventMouseUp, pos, raw.Button, raw.Mod)
		me.Hit = HitResult{Target: hitTarget}
		me.setTarget(hitTarget)
		events = append(events, me)

		// Synthesize click if we have a pending down on the same target and
		// within tolerance.
		if s.pendingDown != nil {
			if !s.beyondTolerance(s.pendingDownPos, pos) {
				click := NewMouseEvent(EventClick, pos, raw.Button, raw.Mod)
				click.Hit = HitResult{Target: hitTarget}
				click.setTarget(hitTarget)
				events = append(events, click)
			}
			s.pendingDown = nil
		}
		return events
	}

	return events
}

func (s *Synthesizer) processResize(raw *RawResizeEvent) []Event {
	re := NewResizeEvent(raw.Width, raw.Height)
	return []Event{re}
}

// processBracketedPaste converts a RawBracketedPaste into a PasteEvent and
// a ClipboardEvent.
func (s *Synthesizer) processBracketedPaste(raw *RawBracketedPaste) []Event {
	pe := NewPasteEvent(raw.Text)
	if s.focus != nil {
		pe.setTarget(s.focus.FocusedTarget())
	}
	ce := NewClipboardEvent(EventPaste, ClipboardPaste, raw.Text)
	if s.focus != nil {
		ce.setTarget(s.focus.FocusedTarget())
	}
	if s.clipboard != nil {
		s.clipboard.SetClipboard(raw.Text)
	}
	return []Event{pe, ce}
}

// hitTest resolves the target at p, or nil if the hit tester is unset.
func (s *Synthesizer) hitTest(p layout.Point) EventTarget {
	if s.hit == nil {
		return nil
	}
	return s.hit.HitTest(p.X, p.Y)
}

// beyondTolerance reports whether the distance between a and b exceeds the
// configured clickRadius in either axis (Chebyshev distance).
func (s *Synthesizer) beyondTolerance(a, b layout.Point) bool {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}
	return dx > s.clickRadius || dy > s.clickRadius
}
