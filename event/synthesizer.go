package event

import (
	"strings"

	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/layout"
)

// FocusReader returns the currently focused render object. The Synthesizer
// uses it to route key events and paste events.
type FocusReader interface {
	// FocusedObject returns the currently focused render object, or nil.
	FocusedTarget() EventTarget
}

// SynthesizerOptions configures a Synthesizer.
type SynthesizerOptions struct {
	// ClickRadius is the maximum cell distance that the mouse may move
	// between mousedown and mouseup and still be synthesized as a click.
	// Default is 3.
	ClickRadius int

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
		clickRadius:        radius,
		scrollableResolver: opts.ScrollableResolver,
	}
}

// ResolveScrollables takes a path of event targets and returns a map of those
// that implement Scrollable.
func (s *Synthesizer) ResolveScrollables(path []EventTarget) map[EventTarget]Scrollable {
	if s.scrollableResolver == nil {
		return nil
	}
	res := make(map[EventTarget]Scrollable)
	for _, t := range path {
		if s := s.scrollableResolver(t); s != nil {
			res[t] = s
		}
	}
	return res
}

// Process converts a raw backend event into zero or more high-level events.
func (s *Synthesizer) Process(raw RawEvent) []Event {
	switch e := raw.(type) {
	case *RawMouseEvent:
		return s.processMouse(e)
	case *RawKeyEvent:
		return s.processKey(e)
	case *RawResizeEvent:
		return s.processResize(e)
	case *RawBracketedPaste:
		return s.processBracketedPaste(e)
	case *RawClipboardEvent:
		return s.processClipboard(e)
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

	isPaste := !raw.Up && (raw.MatchString("ctrl+v") || raw.MatchString("cmd+v") || raw.MatchString("alt+v") || ke.Code == key.CtrlV)
	isCopy := !raw.Up && (raw.MatchString("ctrl+c") || raw.MatchString("cmd+c") || raw.MatchString("alt+c") || ke.Code == key.CtrlC)
	isCut := !raw.Up && (raw.MatchString("ctrl+x") || raw.MatchString("cmd+x") || raw.MatchString("alt+x") || ke.Code == key.CtrlX)

	switch {
	case isCopy:
		if ce := s.synthesizeCopy(ke); ce != nil {
			events = append(events, ce)
		}
	case isCut:
		if ce := s.synthesizeCut(ke); ce != nil {
			events = append(events, ce)
		}
	case isPaste:
		// Emit a paste event. The engine or document will handle fetching
		// data from providers if the Items map is empty.
		ce := NewClipboardEvent(EventPaste, ClipboardPaste)
		if s.focus != nil {
			ce.setTarget(s.focus.FocusedTarget())
		}
		events = append(events, ce)
	}

	return events
}

// synthesizeCopy creates a ClipboardEvent{Copy}.
func (s *Synthesizer) synthesizeCopy(_ *KeyEvent) *ClipboardEvent {
	ce := NewClipboardEvent(EventCopy, ClipboardCopy)
	if s.focus != nil {
		focused := s.focus.FocusedTarget()
		if focused != nil {
			ce.setTarget(focused)
			if sp, ok := focused.(SelectionProvider); ok {
				text := sp.SelectedText()
				if text != "" {
					ce.Items["text/plain"] = []byte(text)
				}
			}
		}
	}
	return ce
}

// synthesizeCut creates a ClipboardEvent{Cut}.
func (s *Synthesizer) synthesizeCut(_ *KeyEvent) *ClipboardEvent {
	ce := NewClipboardEvent(EventCut, ClipboardCut)
	if s.focus != nil {
		focused := s.focus.FocusedTarget()
		if focused != nil {
			ce.setTarget(focused)
			if sp, ok := focused.(SelectionProvider); ok {
				text := sp.SelectedText()
				if text != "" {
					ce.Items["text/plain"] = []byte(text)
				}
			}
		}
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
	ce := NewClipboardEvent(EventPaste, ClipboardPaste)
	if s.focus != nil {
		ce.setTarget(s.focus.FocusedTarget())
	}
	ce.Items["text/plain"] = []byte(raw.Text)
	return []Event{pe, ce}
}

// processClipboard converts a RawClipboardEvent into a ClipboardEvent.
//
// This happend when the terminal sends us clipboard content directly in response of an OSC 52 request.
func (s *Synthesizer) processClipboard(raw *RawClipboardEvent) []Event {
	content := strings.TrimSpace(raw.Content)
	if len(content) == 0 {
		// OSC 52 response with empty content is sent when the clipboard is empty or
		// unsupported. We ignore these.
		return nil
	}

	if raw.Selection != SystemClipboard && raw.Selection != UnknownClipboard {
		// We don't synthesize primary selection events, as they are not user-initiated.
		// Primary is just mouse-driven and doesn't have a standard paste
		//shortcut, so we can safely ignore it.
		return nil
	}

	ce := NewClipboardEvent(EventPaste, ClipboardPaste)
	if s.focus != nil {
		ce.setTarget(s.focus.FocusedTarget())
	}
	// We use the raw content to preserve any newlines or other whitespace.
	ce.Items["text/plain"] = []byte(raw.Content)
	return []Event{ce}
}

// hitTest resolves the target at p, or nil if the hit tester is unset.
func (s *Synthesizer) hitTest(p layout.Point) EventTarget {
	if s.hit == nil {
		return nil
	}
	return s.hit.HitTest(p.X, p.Y)
}

// beyondTolerance reports whether b is outside the click radius relative to a.
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
