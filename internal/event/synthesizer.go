package event

import (
	"encoding/base64"
	"strings"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/dom"
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
	lastHovered        pub.EventTarget

	// pendingDown is set when a mousedown is received; cleared on mouseup.
	pendingDown *pub.MouseEvent
	// pendingDownPos is the screen position of the pending mousedown.
	pendingDownPos geom.Point

	// outBuf is a reused buffer for returned events.
	outBuf []pub.Event
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
	s.outBuf = s.outBuf[:0]
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
	case *backend.RawOscEvent:
		if e.Code == 52 {
			return s.processOsc52(e)
		}
	}
	return nil
}

func (s *Synthesizer) processOsc52(e *backend.RawOscEvent) []pub.Event {
	// Data format is "c;<base64>" or "p;<base64>" etc.
	parts := strings.SplitN(e.Data, ";", 2)
	if len(parts) == 2 {
		b64 := parts[1]
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err == nil {
			ce := pub.NewClipboardEvent(pub.EventPaste, pub.ClipboardPaste)
			ce.SetText(string(decoded))
			if s.focus != nil {
				if ie, ok := any(ce).(pub.InternalEvent); ok {
					ie.SetTarget(s.focus.FocusedTarget())
				}
			}
			s.outBuf = append(s.outBuf, ce)
			return s.outBuf
		}
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
		if ie, ok := any(ke).(pub.InternalEvent); ok {
			ie.SetTarget(s.focus.FocusedTarget())
		}
	}

	// Clipboard synthesis.
	s.outBuf = append(s.outBuf, ke)

	isPaste := !raw.Up && (raw.MatchString("ctrl+v") || raw.MatchString("cmd+v") || raw.MatchString("alt+v") || ke.Code == key.CtrlV)
	isCopy := !raw.Up && (raw.MatchString("ctrl+c") || raw.MatchString("cmd+c") || raw.MatchString("alt+c") || ke.Code == key.CtrlC)
	isCut := !raw.Up && (raw.MatchString("ctrl+x") || raw.MatchString("cmd+x") || raw.MatchString("alt+x") || ke.Code == key.CtrlX)

	switch {
	case isCopy:
		if ce := s.synthesizeCopy(ke); ce != nil {
			s.outBuf = append(s.outBuf, ce)
		}
	case isCut:
		if ce := s.synthesizeCut(ke); ce != nil {
			s.outBuf = append(s.outBuf, ce)
		}
	case isPaste:
		// Emit a paste event. The engine or document will handle fetching
		// data from providers if the Items map is empty.
		ce := pub.NewClipboardEvent(pub.EventPaste, pub.ClipboardPaste)
		if s.focus != nil {
			if ie, ok := any(ce).(pub.InternalEvent); ok {
				ie.SetTarget(s.focus.FocusedTarget())
			}
		}
		s.outBuf = append(s.outBuf, ce)
	}

	return s.outBuf
}

// synthesizeCopy creates a ClipboardEvent{Copy}.
func (s *Synthesizer) synthesizeCopy(_ *pub.KeyEvent) *pub.ClipboardEvent {
	ce := pub.NewClipboardEvent(pub.EventCopy, pub.ClipboardCopy)
	if s.focus != nil {
		focused := s.focus.FocusedTarget()
		if focused != nil {
			if ie, ok := any(ce).(pub.InternalEvent); ok {
				ie.SetTarget(focused)
			}
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
			if ie, ok := any(ce).(pub.InternalEvent); ok {
				ie.SetTarget(focused)
			}
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

	// Pre-calculate count of hover events to make a single slice allocation
	hoverCount := 0
	var oldNode, newNode dom.Node
	var commonAncestor dom.Node
	if hitTarget != s.lastHovered {
		if s.lastHovered != nil {
			oldNode, _ = s.lastHovered.(dom.Node)
		}
		if hitTarget != nil {
			newNode, _ = hitTarget.(dom.Node)
		}

		// Zero-allocation Lowest Common Ancestor (LCA)
		depthA := 0
		for cur := oldNode; cur != nil; cur = cur.Parent() {
			depthA++
		}
		depthB := 0
		for cur := newNode; cur != nil; cur = cur.Parent() {
			depthB++
		}

		currA := oldNode
		currB := newNode
		for depthA > depthB {
			currA = currA.Parent()
			depthA--
		}
		for depthB > depthA {
			currB = currB.Parent()
			depthB--
		}

		for currA != currB {
			currA = currA.Parent()
			currB = currB.Parent()
		}
		commonAncestor = currA

		if s.lastHovered != nil {
			hoverCount++ // mouseout
		}
		for n := oldNode; n != nil && n != commonAncestor; n = n.Parent() {
			hoverCount++
		}
		if hitTarget != nil {
			hoverCount++ // mouseover
		}
		for n := newNode; n != nil && n != commonAncestor; n = n.Parent() {
			hoverCount++
		}
	}

	// Ensure capacity in s.outBuf
	requiredCap := hoverCount + 2
	if cap(s.outBuf) < requiredCap {
		s.outBuf = make([]pub.Event, 0, requiredCap)
	} else {
		s.outBuf = s.outBuf[:0]
	}

	if hitTarget != s.lastHovered {
		s.outBuf = s.appendHoverTransition(s.outBuf, s.lastHovered, hitTarget, pos, raw.Mod, oldNode, newNode, commonAncestor)
		s.lastHovered = hitTarget
	}

	if raw.DeltaX != 0 || raw.DeltaY != 0 {
		// Wheel event.
		we := pub.NewWheelEvent(pos, raw.DeltaX, raw.DeltaY, raw.Mod)
		if ie, ok := any(we).(pub.InternalEvent); ok {
			ie.SetTarget(hitTarget)
		}
		s.outBuf = append(s.outBuf, we)
		return s.outBuf
	}

	if !raw.Up && !raw.Move {
		// Mouse down.
		me := pub.NewMouseEvent(pub.EventMouseDown, pos, raw.Button, raw.Mod)
		me.Hit = pub.HitResult{Target: hitTarget}
		if ie, ok := any(me).(pub.InternalEvent); ok {
			ie.SetTarget(hitTarget)
		}
		s.pendingDown = me
		s.pendingDownPos = pos
		s.outBuf = append(s.outBuf, me)
		return s.outBuf
	}

	if raw.Move {
		// Mouse move.
		me := pub.NewMouseEvent(pub.EventMouseMove, pos, raw.Button, raw.Mod)
		me.Hit = pub.HitResult{Target: hitTarget}
		if ie, ok := any(me).(pub.InternalEvent); ok {
			ie.SetTarget(hitTarget)
		}
		s.outBuf = append(s.outBuf, me)

		// Check for drag: down pending and movement beyond tolerance.
		if s.pendingDown != nil && s.beyondTolerance(s.pendingDownPos, pos) {
			drag := pub.NewMouseEvent(pub.EventDrag, pos, raw.Button, raw.Mod)
			drag.Hit = pub.HitResult{Target: hitTarget}
			if ie, ok := any(drag).(pub.InternalEvent); ok {
				ie.SetTarget(hitTarget)
			}
			s.outBuf = append(s.outBuf, drag)
			s.pendingDown = nil // drag cancels pending click
		}
		return s.outBuf
	}

	if raw.Up {
		// Mouse up.
		me := pub.NewMouseEvent(pub.EventMouseUp, pos, raw.Button, raw.Mod)
		me.Hit = pub.HitResult{Target: hitTarget}
		if ie, ok := any(me).(pub.InternalEvent); ok {
			ie.SetTarget(hitTarget)
		}
		s.outBuf = append(s.outBuf, me)

		// Synthesize click if we have a pending down on the same target and
		// within tolerance.
		if s.pendingDown != nil {
			if !s.beyondTolerance(s.pendingDownPos, pos) {
				click := pub.NewMouseEvent(pub.EventClick, pos, raw.Button, raw.Mod)
				click.Hit = pub.HitResult{Target: hitTarget}
				if ie, ok := any(click).(pub.InternalEvent); ok {
					ie.SetTarget(hitTarget)
				}
				s.outBuf = append(s.outBuf, click)
			}
			s.pendingDown = nil
		}
		return s.outBuf
	}

	return s.outBuf
}

func (s *Synthesizer) processResize(raw *backend.RawResizeEvent) []pub.Event {
	re := pub.NewResizeEvent(raw.Width, raw.Height)
	s.outBuf = append(s.outBuf, re)
	return s.outBuf
}

// processBracketedPaste converts a backend.RawBracketedPaste into a PasteEvent and
// a ClipboardEvent.
func (s *Synthesizer) processBracketedPaste(raw *backend.RawBracketedPaste) []pub.Event {
	pe := pub.NewPasteEvent(raw.Text)
	if s.focus != nil {
		if ie, ok := any(pe).(pub.InternalEvent); ok {
			ie.SetTarget(s.focus.FocusedTarget())
		}
	}
	ce := pub.NewClipboardEvent(pub.EventPaste, pub.ClipboardPaste)
	if s.focus != nil {
		if ie, ok := any(ce).(pub.InternalEvent); ok {
			ie.SetTarget(s.focus.FocusedTarget())
		}
	}
	ce.Items["text/plain"] = []byte(raw.Text)
	s.outBuf = append(s.outBuf, pe, ce)
	return s.outBuf
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
		if ie, ok := any(ce).(pub.InternalEvent); ok {
			ie.SetTarget(s.focus.FocusedTarget())
		}
	}
	// We use the raw content to preserve any newlines or other whitespace.
	ce.Items["text/plain"] = []byte(raw.Content)
	s.outBuf = append(s.outBuf, ce)
	return s.outBuf
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

func (s *Synthesizer) appendHoverTransition(
	events []pub.Event,
	oldTarget, newTarget pub.EventTarget,
	pos geom.Point,
	mods pub.Modifiers,
	oldNode, newNode, commonAncestor dom.Node,
) []pub.Event {
	// 1. Dispatch mouseout on oldTarget (bubbles)
	if oldTarget != nil {
		mo := pub.NewMouseEvent(pub.EventMouseOut, pos, pub.ButtonNone, mods)
		mo.Hit = pub.HitResult{Target: oldTarget}
		if ie, ok := any(mo).(pub.InternalEvent); ok {
			ie.SetTarget(oldTarget)
		}
		events = append(events, mo)
	}

	// 2. Dispatch mouseleave on oldNode and its ancestors up to commonAncestor (does not bubble)
	for n := oldNode; n != nil && n != commonAncestor; n = n.Parent() {
		ml := pub.NewMouseEvent(pub.EventMouseLeave, pos, pub.ButtonNone, mods)
		ml.Hit = pub.HitResult{Target: n}
		if ie, ok := any(ml).(pub.InternalEvent); ok {
			ie.SetTarget(n)
		}
		events = append(events, ml)
	}

	// 3. Dispatch mouseover on newTarget (bubbles)
	if newTarget != nil {
		mo := pub.NewMouseEvent(pub.EventMouseOver, pos, pub.ButtonNone, mods)
		mo.Hit = pub.HitResult{Target: newTarget}
		if ie, ok := any(mo).(pub.InternalEvent); ok {
			ie.SetTarget(newTarget)
		}
		events = append(events, mo)
	}

	// 4. Dispatch mouseenter on newNode and its ancestors up to commonAncestor (does not bubble)
	var enterBuf [64]dom.Node
	enterNodes := enterBuf[:0]
	for n := newNode; n != nil && n != commonAncestor; n = n.Parent() {
		if len(enterNodes) < len(enterBuf) {
			enterNodes = append(enterNodes, n)
		} else {
			// Fallback to heap allocation for extremely deep trees (should almost never happen)
			if len(enterNodes) == len(enterBuf) {
				temp := make([]dom.Node, len(enterNodes), len(enterNodes)+8)
				copy(temp, enterNodes)
				enterNodes = temp
			}
			enterNodes = append(enterNodes, n)
		}
	}
	// Reverse so we dispatch from highest ancestor down to target
	for i := len(enterNodes) - 1; i >= 0; i-- {
		n := enterNodes[i]
		me := pub.NewMouseEvent(pub.EventMouseEnter, pos, pub.ButtonNone, mods)
		me.Hit = pub.HitResult{Target: n}
		if ie, ok := any(me).(pub.InternalEvent); ok {
			ie.SetTarget(n)
		}
		events = append(events, me)
	}

	return events
}
