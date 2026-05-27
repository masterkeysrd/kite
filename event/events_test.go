package event_test

import (
	"iter"
	"testing"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// stubObject is a minimal render.Object used by tests. It supports Parent()
// to allow ancestor-chain construction.
type stubObject struct {
	event.Target
	parent   event.EventTarget
	bounds   geom.Rect
	children []event.EventTarget
}

func newStub(bounds geom.Rect) *stubObject { return &stubObject{bounds: bounds} }

func (s *stubObject) Parent() event.EventTarget { return s.parent }
func (s *stubObject) FirstChild() event.EventTarget {
	if len(s.children) > 0 {
		return s.children[0]
	}
	return nil
}
func (s *stubObject) LastChild() event.EventTarget {
	if n := len(s.children); n > 0 {
		return s.children[n-1]
	}
	return nil
}
func (s *stubObject) NextSibling() event.EventTarget     { return nil }
func (s *stubObject) PreviousSibling() event.EventTarget { return nil }
func (s *stubObject) EventTarget() event.EventTarget     { return s }
func (s *stubObject) Children() iter.Seq[event.EventTarget] {
	return func(yield func(event.EventTarget) bool) {
		for _, c := range s.children {
			if !yield(c) {
				return
			}
		}
	}
}
func (s *stubObject) Bounds() geom.Rect                { return s.bounds }
func (s *stubObject) SetBounds(r geom.Rect)            { s.bounds = r }
func (s *stubObject) LogicalNode() any                 { return nil }
func (s *stubObject) MarkDetached()                    {}
func (s *stubObject) IsDetached() bool                 { return false }
func (s *stubObject) MarkChildrenDirty()               {}
func (s *stubObject) RawStyle() style.Style            { return style.Style{} }
func (s *stubObject) DefaultStyle() style.Style        { return style.Style{} }
func (s *stubObject) ComputedStyle() *style.Computed   { return nil }
func (s *stubObject) SetComputedStyle(*style.Computed) {}
func (s *stubObject) Flags() render.DirtyFlag          { return 0 }
func (s *stubObject) MarkDirty(render.DirtyFlag)       {}
func (s *stubObject) ClearDirty(render.DirtyFlag)      {}
func (s *stubObject) IsDirtySet(render.DirtyFlag) bool { return false }
func (s *stubObject) IsDirtyStyle() bool               { return false }
func (s *stubObject) IsDirtyLayout() bool              { return false }
func (s *stubObject) IsDirtyPaint() bool               { return false }
func (s *stubObject) IsDirtyScroll() bool              { return false }
func (s *stubObject) Focusable() bool                  { return false }
func (s *stubObject) SetFocusable(bool)                {}
func (s *stubObject) Disabled() bool                   { return false }
func (s *stubObject) SetDisabled(bool)                 {}
func (s *stubObject) SelectedText() string             { return "" }

// addChild sets parent and appends child.
func addChild(parent, child *stubObject) {
	child.parent = parent
	parent.children = append(parent.children, child)
}

// buildPath builds a root→target path from ancestor chain by walking parents.
func buildPath(objs ...*stubObject) []event.EventTarget {
	path := make([]event.EventTarget, len(objs))
	for i, o := range objs {
		path[i] = o
	}
	return path
}

// newRegistry builds an event.Target map for registering targets.
func newRegistry() map[event.EventTarget]event.EventTarget {
	return make(map[event.EventTarget]event.EventTarget)
}

func ensureTarget(m map[event.EventTarget]event.EventTarget, obj event.EventTarget) event.EventTarget {
	return obj
}

// ---------------------------------------------------------------------------
// Dispatcher tests
// ---------------------------------------------------------------------------

func TestDispatcher_CaptureTargetBubble_Order(t *testing.T) {
	t.Parallel()

	root := newStub(geom.Rect{Size: geom.Size{Width: 80, Height: 24}})
	mid := newStub(geom.Rect{Size: geom.Size{Width: 40, Height: 12}})
	target := newStub(geom.Rect{Size: geom.Size{Width: 20, Height: 6}})
	addChild(root, mid)
	addChild(mid, target)

	_ = newRegistry()
	d := event.NewDispatcher()

	var order []string

	root.AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "root-capture")
	}, event.Capture())
	mid.AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "mid-capture")
	}, event.Capture())
	target.AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "target-capture")
	}, event.Capture())
	target.AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "target-bubble")
	})
	mid.AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "mid-bubble")
	})
	root.AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "root-bubble")
	})

	path := buildPath(root, mid, target)
	e := event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0)
	d.Dispatch(e, path)

	want := []string{
		"root-capture",
		"mid-capture",
		"target-capture",
		"target-bubble",
		"mid-bubble",
		"root-bubble",
	}
	if len(order) != len(want) {
		t.Fatalf("dispatch order length: got %v, want %v", order, want)
	}
	for i, v := range want {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestDispatcher_StopPropagation_HaltsBubble(t *testing.T) {
	t.Parallel()

	root := newStub(geom.Rect{})
	target := newStub(geom.Rect{})

	_ = newRegistry()
	d := event.NewDispatcher()

	reached := false
	target.AddEventListener(event.EventClick, func(e event.Event) {
		e.StopPropagation()
	})
	root.AddEventListener(event.EventClick, func(_ event.Event) {
		reached = true
	})

	path := buildPath(root, target)
	e := event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0)
	d.Dispatch(e, path)

	if reached {
		t.Error("root bubble listener should not have been reached after StopPropagation")
	}
}

func TestDispatcher_PreventDefault_FlagOnly(t *testing.T) {
	t.Parallel()

	root := newStub(geom.Rect{})
	target := newStub(geom.Rect{})

	_ = newRegistry()
	d := event.NewDispatcher()

	target.AddEventListener(event.EventClick, func(e event.Event) {
		e.PreventDefault()
	})

	rootReached := false
	root.AddEventListener(event.EventClick, func(_ event.Event) {
		rootReached = true
	})

	path := buildPath(root, target)
	e := event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0)
	d.Dispatch(e, path)

	if !e.DefaultPrevented() {
		t.Error("expected DefaultPrevented to be true")
	}
	if !rootReached {
		t.Error("PreventDefault should not stop propagation; root should still be reached")
	}
}

func TestSubscription_Cancel_RemovesListener(t *testing.T) {
	t.Parallel()

	obj := newStub(geom.Rect{})
	d := event.NewDispatcher()
	et := event.EventTarget(obj)

	calls := 0
	sub := et.AddEventListener(event.EventClick, func(_ event.Event) {
		calls++
	})
	sub.Cancel()

	path := buildPath(obj)
	e := event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0)
	d.Dispatch(e, path)

	if calls != 0 {
		t.Errorf("cancelled listener was called %d times, expected 0", calls)
	}
}

// ---------------------------------------------------------------------------
// Key matching tests
// ---------------------------------------------------------------------------

func TestKeyEvent_MatchString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pattern string
		code    rune
		mods    key.Mod
		want    bool
	}{
		{"ctrl+s", 's', key.ModCtrl, true},
		{"ctrl+s", 's', 0, false},
		{"ctrl+shift+p", 'p', key.ModCtrl | key.ModShift, true},
		{"ctrl+shift+p", 'p', key.ModCtrl, false},
		{"alt+enter", key.KeyEnter, key.ModAlt, true},
		{"alt+enter", key.KeyEnter, key.ModCtrl, false},
		{"escape", key.KeyEscape, 0, true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.pattern, func(t *testing.T) {
			t.Parallel()
			ke := event.NewKeyEvent(event.EventKeyDown, key.Key{Code: tc.code, Mod: tc.mods})
			if got := ke.MatchString(tc.pattern); got != tc.want {
				t.Errorf("KeyEvent(code=%q mods=%d).MatchString(%q) = %v, want %v",
					tc.code, tc.mods, tc.pattern, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Synthesizer tests
// ---------------------------------------------------------------------------

// stubHitTester returns a fixed event.EventTarget regardless of coordinates.
type stubHitTester struct {
	result event.EventTarget
}

func (h *stubHitTester) HitTest(_, _ int) event.EventTarget { return h.result }

// stubFocus always returns the same focused object.
type stubFocus struct {
	target event.EventTarget
}

func (f *stubFocus) FocusedTarget() event.EventTarget { return f.target }

func TestSynthesizer_ClickWithinTolerance(t *testing.T) {
	t.Parallel()

	target := newStub(geom.Rect{})
	hit := &stubHitTester{result: target}
	focus := &stubFocus{target: target}
	s := event.NewSynthesizer(hit, focus, event.SynthesizerOptions{ClickRadius: 3})

	// Mouse down at (5,5).
	evts := s.Process(&event.RawMouseEvent{X: 5, Y: 5, Button: event.ButtonLeft})
	hasType(t, evts, event.EventMouseDown)

	// Mouse up at (6,5) — within 3-cell tolerance → click synthesized.
	evts = s.Process(&event.RawMouseEvent{X: 6, Y: 5, Button: event.ButtonLeft, Up: true})
	hasType(t, evts, event.EventMouseUp)
	hasType(t, evts, event.EventClick)
}

func TestSynthesizer_DragBeyondTolerance(t *testing.T) {
	t.Parallel()

	target := newStub(geom.Rect{})
	hit := &stubHitTester{result: target}
	focus := &stubFocus{target: target}
	s := event.NewSynthesizer(hit, focus, event.SynthesizerOptions{ClickRadius: 3})

	// Mouse down.
	s.Process(&event.RawMouseEvent{X: 0, Y: 0, Button: event.ButtonLeft})

	// Move beyond tolerance (>3 cells).
	evts := s.Process(&event.RawMouseEvent{X: 10, Y: 0, Button: event.ButtonLeft, Move: true})
	hasType(t, evts, event.EventMouseMove)
	hasType(t, evts, event.EventDrag)

	// Mouse up — no click because drag cancelled it.
	evts = s.Process(&event.RawMouseEvent{X: 10, Y: 0, Button: event.ButtonLeft, Up: true})
	hasType(t, evts, event.EventMouseUp)
	if containsType(evts, event.EventClick) {
		t.Error("click should not be synthesized after drag beyond tolerance")
	}
}

func TestSynthesizer_ResolvesHitTarget(t *testing.T) {
	t.Parallel()

	target := newStub(geom.Rect{Origin: geom.Point{X: 5, Y: 5}, Size: geom.Size{Width: 10, Height: 5}})
	hit := &stubHitTester{result: target}
	focus := &stubFocus{}
	s := event.NewSynthesizer(hit, focus, event.SynthesizerOptions{})

	evts := s.Process(&event.RawMouseEvent{X: 7, Y: 6})
	if len(evts) == 0 {
		t.Fatal("expected event from synthesizer")
	}
	me, ok := evts[0].(*event.MouseEvent)
	if !ok {
		t.Fatal("expected *MouseEvent")
	}
	if me.Hit.Target != target {
		t.Errorf("expected hit target to be target, got %v", me.Hit.Target)
	}
}

// ---------------------------------------------------------------------------
// Hit test tests (via engine stub)
// ---------------------------------------------------------------------------

// stubRenderView is a minimal render tree for hit-test testing.
type stubRenderView struct {
	stubObject
	overlays []event.EventTarget
}

func (v *stubRenderView) Overlays() []event.EventTarget { return v.overlays }

// hitTestObject replicates the engine's logic for testing.
func hitTestObject(obj event.EventTarget, p geom.Point) event.EventTarget {
	sobj := obj.(*stubObject)
	if !sobj.Bounds().Contains(p) {
		return nil
	}
	// This is a simple stub, we don't have PreviousSibling in stubObject yet but we can walk children again or just assume it's linear.
	// For testing topmost, we should walk in reverse.
	children := make([]event.EventTarget, 0)
	for child := range sobj.Children() {
		children = append(children, child)
	}

	for i := len(children) - 1; i >= 0; i-- {
		if hit := hitTestObject(children[i], p); hit != nil {
			return hit
		}
	}
	return obj
}

type testHitTester struct {
	view *stubRenderView
}

func (h *testHitTester) HitTest(x, y int) event.EventTarget {
	p := geom.Point{X: x, Y: y}
	overlays := h.view.overlays
	for i := len(overlays) - 1; i >= 0; i-- {
		if hit := hitTestObject(overlays[i], p); hit != nil {
			return hit
		}
	}
	return hitTestObject(&h.view.stubObject, p)
}

func TestHitTest_OverlayBeforeDocument(t *testing.T) {
	t.Parallel()

	view := &stubRenderView{
		stubObject: stubObject{
			bounds: geom.Rect{Size: geom.Size{Width: 80, Height: 24}},
		},
	}
	// Document child at (0,0)-(20,10).
	docChild := newStub(geom.Rect{Size: geom.Size{Width: 20, Height: 10}})
	addChild(&view.stubObject, docChild)

	// Overlay at (5,5)-(15,15) covering the same point.
	overlay := newStub(geom.Rect{
		Origin: geom.Point{X: 5, Y: 5},
		Size:   geom.Size{Width: 10, Height: 10},
	})
	view.overlays = []event.EventTarget{overlay}

	ht := &testHitTester{view: view}

	// Point (7,7) is covered by both docChild and overlay.
	hit := ht.HitTest(7, 7)
	if hit != overlay {
		t.Errorf("expected overlay to win hit test, got %v", hit)
	}
}

func TestHitTest_TopmostObject(t *testing.T) {
	t.Parallel()

	parent := newStub(geom.Rect{Size: geom.Size{Width: 80, Height: 24}})
	child1 := newStub(geom.Rect{Size: geom.Size{Width: 40, Height: 10}})
	child2 := newStub(geom.Rect{
		Origin: geom.Point{X: 5, Y: 0},
		Size:   geom.Size{Width: 40, Height: 10},
	})
	addChild(parent, child1)
	addChild(parent, child2)

	view := &stubRenderView{stubObject: *parent}
	view.stubObject.children = parent.children
	ht := &testHitTester{view: view}

	// Point (10,5) is inside both child1 and child2 (they overlap).
	// child2 was added last, so it's "on top".
	hit := ht.HitTest(10, 5)
	if hit != child2 {
		t.Errorf("expected child2 (last child = topmost) to win, got %v", hit)
	}
}

// ---------------------------------------------------------------------------
// Wheel routing tests
// ---------------------------------------------------------------------------

type stubScrollable struct {
	calls []*event.WheelEvent
}

func (s *stubScrollable) OnWheel(e *event.WheelEvent) { s.calls = append(s.calls, e) }

func TestWheel_RoutesToFirstScrollableAncestor(t *testing.T) {
	t.Parallel()

	root := newStub(geom.Rect{})
	mid := newStub(geom.Rect{})
	target := newStub(geom.Rect{})
	addChild(root, mid)
	addChild(mid, target)

	d := event.NewDispatcher()

	sc := &stubScrollable{}
	scrollables := map[event.EventTarget]event.Scrollable{mid: sc}

	path := buildPath(root, mid, target)
	e := event.NewWheelEvent(geom.Point{}, 0, 3, 0)
	d.DispatchWheel(e, path, scrollables)

	if len(sc.calls) != 1 {
		t.Errorf("expected 1 wheel call on mid scrollable, got %d", len(sc.calls))
	}
}

func TestWheel_NoScrollable_NoOp(t *testing.T) {
	t.Parallel()

	root := newStub(geom.Rect{})
	target := newStub(geom.Rect{})

	d := event.NewDispatcher()

	rootCalled := false
	root.AddEventListener(event.EventWheel, func(_ event.Event) {
		rootCalled = true
	})

	path := buildPath(root, target)
	e := event.NewWheelEvent(geom.Point{}, 0, 1, 0)
	d.DispatchWheel(e, path, nil)

	// Without a Scrollable, the event still bubbles normally.
	if !rootCalled {
		t.Error("expected root listener to be called when no Scrollable in chain")
	}
}

// ---------------------------------------------------------------------------
// Paste / clipboard tests
// ---------------------------------------------------------------------------

// stubClipboard is a simple in-memory clipboard bridge.
type stubClipboard struct {
	data      string
	requested bool
}

func (c *stubClipboard) GetClipboard() string     { return c.data }
func (c *stubClipboard) SetClipboard(text string) { c.data = text }
func (c *stubClipboard) RequestClipboard()        { c.requested = true }

func TestPaste_BracketedSequence_BecomesSinglePasteEvent(t *testing.T) {
	t.Parallel()

	s := event.NewSynthesizer(nil, nil, event.SynthesizerOptions{})

	evts := s.Process(&event.RawBracketedPaste{Text: "hello"})

	var pasteEvt *event.PasteEvent
	for _, ev := range evts {
		if pe, ok := ev.(*event.PasteEvent); ok {
			pasteEvt = pe
			break
		}
	}
	if pasteEvt == nil {
		t.Fatal("expected a PasteEvent from bracketed paste sequence")
	}
	if pasteEvt.Text != "hello" {
		t.Errorf("PasteEvent.Text = %q, want %q", pasteEvt.Text, "hello")
	}
}

// stubSelectionProvider is an in-memory SelectionProvider.
type stubSelectionProvider struct {
	stubObject
	selected string
}

func (s *stubSelectionProvider) SelectedText() string { return s.selected }

func TestClipboard_CopyCut_FromSelectionProvider(t *testing.T) {
	t.Parallel()

	focusedObj := &stubSelectionProvider{
		stubObject: stubObject{
			bounds: geom.Rect{Size: geom.Size{Width: 80, Height: 24}},
		},
		selected: "selected text",
	}
	focus := &stubFocus{target: focusedObj}
	cb := &stubClipboard{}
	s := event.NewSynthesizer(nil, focus, event.SynthesizerOptions{})

	// Ctrl+C.
	evts := s.Process(&event.RawKeyEvent{Key: key.Key{Code: 'c', Mod: key.ModCtrl}})
	var copyEvt *event.ClipboardEvent
	for _, ev := range evts {
		if ce, ok := ev.(*event.ClipboardEvent); ok && ce.ClipType == event.ClipboardCopy {
			copyEvt = ce
			break
		}
	}
	if copyEvt == nil {
		t.Fatal("expected ClipboardEvent{Copy} on Ctrl+C")
	}
	if !copyEvt.Bubbles() {
		t.Error("ClipboardEvent should bubble")
	}
	if got := copyEvt.Text(); got != "selected text" {
		t.Errorf("copy text = %q, want %q", got, "selected text")
	}
	// Note: Synthesizer no longer calls SetClipboard; Document handler does.
	if cb.data != "" {
		t.Errorf("synthesizer should not write to clipboard directly; got %q", cb.data)
	}

	// Ctrl+X.
	evts = s.Process(&event.RawKeyEvent{Key: key.Key{Code: 'x', Mod: key.ModCtrl}})
	var cutEvt *event.ClipboardEvent
	for _, ev := range evts {
		if ce, ok := ev.(*event.ClipboardEvent); ok && ce.ClipType == event.ClipboardCut {
			cutEvt = ce
			break
		}
	}
	if cutEvt == nil {
		t.Fatal("expected ClipboardEvent{Cut} on Ctrl+X")
	}
}

func TestClipboard_Synthesizer_PlatformShortcuts(t *testing.T) {
	t.Parallel()

	focus := &stubFocus{}
	s := event.NewSynthesizer(nil, focus, event.SynthesizerOptions{})

	tests := []struct {
		name string
		key  key.Key
		want event.ClipboardType
	}{
		{"Cmd+C", key.Key{Code: 'c', Mod: key.ModSuper}, event.ClipboardCopy},
		{"Cmd+V", key.Key{Code: 'v', Mod: key.ModSuper}, event.ClipboardPaste},
		{"Alt+C", key.Key{Code: 'c', Mod: key.ModAlt}, event.ClipboardCopy},
		{"Alt+V", key.Key{Code: 'v', Mod: key.ModAlt}, event.ClipboardPaste},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evts := s.Process(&event.RawKeyEvent{Key: tt.key})
			var ce *event.ClipboardEvent
			for _, e := range evts {
				if c, ok := e.(*event.ClipboardEvent); ok && c.ClipType == tt.want {
					ce = c
					break
				}
			}
			if ce == nil {
				t.Fatalf("expected ClipboardEvent{%v} for %s", tt.want, tt.name)
			}
		})
	}
}

func TestClipboard_Paste_FromCtrlV_AndBracketedPaste(t *testing.T) {
	t.Parallel()

	s := event.NewSynthesizer(nil, nil, event.SynthesizerOptions{})

	// Ctrl+V → emits ClipboardEvent immediately.
	evts := s.Process(&event.RawKeyEvent{Key: key.Key{Code: 'v', Mod: key.ModCtrl}})
	var foundPaste bool
	for _, ev := range evts {
		if ce, ok := ev.(*event.ClipboardEvent); ok && ce.ClipType == event.ClipboardPaste {
			foundPaste = true
			break
		}
	}
	if !foundPaste {
		t.Fatal("expected ClipboardEvent{Paste} on Ctrl+V")
	}

	// Bracketed paste → ClipboardEvent{Paste}.
	evts = s.Process(&event.RawBracketedPaste{Text: "pasted"})
	var pasteEvt *event.ClipboardEvent
	for _, ev := range evts {
		if ce, ok := ev.(*event.ClipboardEvent); ok && ce.ClipType == event.ClipboardPaste {
			pasteEvt = ce
			break
		}
	}
	if pasteEvt == nil {
		t.Fatal("expected ClipboardEvent{Paste} from bracketed paste")
	}
	if got := pasteEvt.Text(); got != "pasted" {
		t.Errorf("paste text = %q, want %q", got, "pasted")
	}
}

// ---------------------------------------------------------------------------
// Engine dispatch resize test
// ---------------------------------------------------------------------------

func TestEngine_DispatchesResizeOnViewportChange(t *testing.T) {
	t.Parallel()

	s := event.NewSynthesizer(nil, nil, event.SynthesizerOptions{})
	evts := s.Process(&event.RawResizeEvent{Width: 120, Height: 40})
	if len(evts) != 1 {
		t.Fatalf("expected 1 event for resize, got %d", len(evts))
	}
	re, ok := evts[0].(*event.ResizeEvent)
	if !ok {
		t.Fatalf("expected *ResizeEvent, got %T", evts[0])
	}
	if re.Width != 120 || re.Height != 40 {
		t.Errorf("resize dimensions: got %dx%d, want 120x40", re.Width, re.Height)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func hasType(t *testing.T, evts []event.Event, typ event.EventType) {
	t.Helper()
	if !containsType(evts, typ) {
		t.Errorf("expected event type %q in %v", typ, typeNames(evts))
	}
}

func containsType(evts []event.Event, typ event.EventType) bool {
	for _, e := range evts {
		if e.Type() == typ {
			return true
		}
	}
	return false
}

func typeNames(evts []event.Event) []event.EventType {
	names := make([]event.EventType, len(evts))
	for i, e := range evts {
		names[i] = e.Type()
	}
	return names
}
