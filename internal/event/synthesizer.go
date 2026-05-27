package event

import (
	"strings"

	"github.com/masterkeysrd/kite/backend"
	pub "github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/key"
)

// FocusReader returns the currently focused render object. The Synthesizer
// uses it to route key events and paste events.
type FocusReader interface {
	// FocusedObject returns the currently focused render object, or nil.
	FocusedTarget() pub.EventTarget
}

// SynthesizerOptions configures a Synthesizer.
type SynthesizerOptions struct {
	// ClickRadius is the maximum cell distance that the mouse may move
	// between mousedown and mouseup and still be synthesized as a click.
	// Default is 3.
	ClickRadius int

	// ScrollableResolver maps a target to its Scrollable, or nil.
	// Used by DispatchWheel.
	ScrollableResolver func(pub.EventTarget) pub.Scrollable
}

// Synthesizer converts raw backend input (backend.RawEvent) into structured events
// ready for dispatch. Click and drag synthesis applies movement tolerance.
// Hit testing assigns the target render object.
//
// Synthesizer is not safe for concurrent use.
type Synthesizer struct {
	hit                pub.HitTester
	focus              FocusReader
	clickRadius        int
	scrollableResolver func(pub.EventTarget) pub.Scrollable

	// pendingDown is set when a mousedown is received; cleared on mouseup.
	pendingDown *pub.MouseEvent
	// pendingDownPos is the screen position of the pending mousedown.
	pendingDownPos geom.Point
}

// NewSynthesizer creates a Synthesizer with the given HitTester, FocusReader,
// and options.
func NewSynthesizer(hit pub.HitTester, focus FocusReader, opts SynthesizerOptions) *Synthesizer {
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
func (s *Synthesizer) ResolveScrollables(path []pub.EventTarget) map[pub.EventTarget]pub.Scrollable {
	if s.scrollableResolver == nil {
		return nil
	}
	res := make(map[pub.EventTarget]pub.Scrollable)
	for _, t := range path {
		if s := s.scrollableResolver(t); s != nil {
			res[t] = s
		}
	}
	return res
}

// Process converts a raw backend event into zero or more high-level events.
func (s *Synthesizer) Process(raw backend.RawEvent) []pub.Event {
	switch e := raw.(type) {
	case *backend.RawMouseEvent:
		return s.processMouse(e)
	case *backend.RawKeyEvent:
		return s.processKey(e)
	case *backend.RawResizeEvent:
		return s.processResize(e)
	case *backend.RawBracketedPaste:
		return s.processBracketedPaste(e)
	case *backend.RawClipboardEvent:
		return s.processClipboard(e)
	}
	return nil
}

// processKey converts a backend.RawKeyEvent into a KeyEvent, and optionally a
// ClipboardEvent when Ctrl+C / Ctrl+X / Ctrl+V is pressed.
func (s *Synthesizer) processKey(raw *backend.RawKeyEvent) []pub.Event {
	typ := pub.EventKeyDown
	if raw.Up {
		typ = pub.EventKeyUp
	}
	ke := pub.NewKeyEvent(typ, raw.Key)

	// Route to the focused element.
	if s.focus != nil {
		ke.SetTarget(s.focus.FocusedTarget())
	}

	// Clipboard synthesis.
	var events []pub.Event
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
		ce := pub.NewClipboardEvent(pub.EventPaste, pub.ClipboardPaste)
		if s.focus != nil {
			ce.SetTarget(s.focus.FocusedTarget())
		}
		events = append(events, ce)
	}

	return events
}

// synthesizeCopy creates a ClipboardEvent{Copy}.
func (s *Synthesizer) synthesizeCopy(_ *pub.KeyEvent) *pub.ClipboardEvent {
	ce := pub.NewClipboardEvent(pub.EventCopy, pub.ClipboardCopy)
	if s.focus != nil {
		focused := s.focus.FocusedTarget()
		if focused != nil {
			ce.SetTarget(focused)
			if sp, ok := focused.(pub.SelectionProvider); ok {
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
func (s *Synthesizer) synthesizeCut(_ *pub.KeyEvent) *pub.ClipboardEvent {
	ce := pub.NewClipboardEvent(pub.EventCut, pub.ClipboardCut)
	if s.focus != nil {
		focused := s.focus.FocusedTarget()
		if focused != nil {
			ce.SetTarget(focused)
			if sp, ok := focused.(pub.SelectionProvider); ok {
				text := sp.SelectedText()
				if text != "" {
					ce.Items["text/plain"] = []byte(text)
				}
			}
		}
	}
	return ce
}

// processMouse converts a backend.RawMouseEvent into mouse events, synthesizing click
// and drag as appropriate.
func (s *Synthesizer) processMouse(raw *backend.RawMouseEvent) []pub.Event {
	pos := geom.Point{X: raw.X, Y: raw.Y}
	hitTarget := s.hitTest(pos)

	var events []pub.Event

	if raw.DeltaX != 0 || raw.DeltaY != 0 {
		// Wheel event.
		we := pub.NewWheelEvent(pos, raw.DeltaX, raw.DeltaY, raw.Mod)
		we.SetTarget(hitTarget)
		events = append(events, we)
		return events
	}

	if !raw.Up && !raw.Move {
		// Mouse down.
		me := pub.NewMouseEvent(pub.EventMouseDown, pos, raw.Button, raw.Mod)
		me.Hit = pub.HitResult{Target: hitTarget}
		me.SetTarget(hitTarget)
		s.pendingDown = me
		s.pendingDownPos = pos
		events = append(events, me)
		return events
	}

	if raw.Move {
		// Mouse move.
		me := pub.NewMouseEvent(pub.EventMouseMove, pos, raw.Button, raw.Mod)
		me.Hit = pub.HitResult{Target: hitTarget}
		me.SetTarget(hitTarget)
		events = append(events, me)

		// Check for drag: down pending and movement beyond tolerance.
		if s.pendingDown != nil && s.beyondTolerance(s.pendingDownPos, pos) {
			drag := pub.NewMouseEvent(pub.EventDrag, pos, raw.Button, raw.Mod)
			drag.Hit = pub.HitResult{Target: hitTarget}
			drag.SetTarget(hitTarget)
			events = append(events, drag)
			s.pendingDown = nil // drag cancels pending click
		}
		return events
	}

	if raw.Up {
		// Mouse up.
		me := pub.NewMouseEvent(pub.EventMouseUp, pos, raw.Button, raw.Mod)
		me.Hit = pub.HitResult{Target: hitTarget}
		me.SetTarget(hitTarget)
		events = append(events, me)

		// Synthesize click if we have a pending down on the same target and
		// within tolerance.
		if s.pendingDown != nil {
			if !s.beyondTolerance(s.pendingDownPos, pos) {
				click := pub.NewMouseEvent(pub.EventClick, pos, raw.Button, raw.Mod)
				click.Hit = pub.HitResult{Target: hitTarget}
				click.SetTarget(hitTarget)
				events = append(events, click)
			}
			s.pendingDown = nil
		}
		return events
	}

	return events
}

func (s *Synthesizer) processResize(raw *backend.RawResizeEvent) []pub.Event {
	re := pub.NewResizeEvent(raw.Width, raw.Height)
	return []pub.Event{re}
}

// processBracketedPaste converts a backend.RawBracketedPaste into a PasteEvent and
// a ClipboardEvent.
func (s *Synthesizer) processBracketedPaste(raw *backend.RawBracketedPaste) []pub.Event {
	pe := pub.NewPasteEvent(raw.Text)
	if s.focus != nil {
		pe.SetTarget(s.focus.FocusedTarget())
	}
	ce := pub.NewClipboardEvent(pub.EventPaste, pub.ClipboardPaste)
	if s.focus != nil {
		ce.SetTarget(s.focus.FocusedTarget())
	}
	ce.Items["text/plain"] = []byte(raw.Text)
	return []pub.Event{pe, ce}
}

// processClipboard converts a backend.RawClipboardEvent into a ClipboardEvent.
//
// This happend when the terminal sends us clipboard content directly in response of an OSC 52 request.
func (s *Synthesizer) processClipboard(raw *backend.RawClipboardEvent) []pub.Event {
	content := strings.TrimSpace(raw.Content)
	if len(content) == 0 {
		// OSC 52 response with empty content is sent when the clipboard is empty or
		// unsupported. We ignore these.
		return nil
	}

	if raw.Selection != pub.SystemClipboard && raw.Selection != pub.UnknownClipboard {
		// We don't synthesize primary selection events, as they are not user-initiated.
		// Primary is just mouse-driven and doesn't have a standard paste
		//shortcut, so we can safely ignore it.
		return nil
	}

	ce := pub.NewClipboardEvent(pub.EventPaste, pub.ClipboardPaste)
	if s.focus != nil {
		ce.SetTarget(s.focus.FocusedTarget())
	}
	// We use the raw content to preserve any newlines or other whitespace.
	ce.Items["text/plain"] = []byte(raw.Content)
	return []pub.Event{ce}
}

// hitTest resolves the target at p, or nil if the hit tester is unset.
func (s *Synthesizer) hitTest(p geom.Point) pub.EventTarget {
	if s.hit == nil {
		return nil
	}
	return s.hit.HitTest(p.X, p.Y)
}

// beyondTolerance reports whether b is outside the click radius relative to a.
func (s *Synthesizer) beyondTolerance(a, b geom.Point) bool {
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
