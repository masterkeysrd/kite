package event_test

import (
	"iter"
	"testing"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// stubObject is a minimal render.Object used by tests. It supports Parent()
// to allow ancestor-chain construction.
type stubObject struct {
	parent   render.Object
	bounds   layout.Rect
	children []render.Object
}

func newStub(bounds layout.Rect) *stubObject { return &stubObject{bounds: bounds} }

func (s *stubObject) Parent() render.Object { return s.parent }
func (s *stubObject) FirstChild() render.Object {
	if len(s.children) > 0 {
		return s.children[0]
	}
	return nil
}
func (s *stubObject) LastChild() render.Object {
	if n := len(s.children); n > 0 {
		return s.children[n-1]
	}
	return nil
}
func (s *stubObject) NextSibling() render.Object     { return nil }
func (s *stubObject) PreviousSibling() render.Object { return nil }
func (s *stubObject) Children() iter.Seq[render.Object] {
	return func(yield func(render.Object) bool) {
		for _, c := range s.children {
			if !yield(c) {
				return
			}
		}
	}
}
func (s *stubObject) Bounds() layout.Rect               { return s.bounds }
func (s *stubObject) SetBounds(r layout.Rect)           { s.bounds = r }
func (s *stubObject) LogicalNode() any                  { return nil }
func (s *stubObject) MarkDetached()                     {}
func (s *stubObject) IsDetached() bool                  { return false }
func (s *stubObject) MarkChildrenDirty()                {}
func (s *stubObject) RawStyle() style.Style             { return style.Style{} }
func (s *stubObject) SetRawStyle(style.Style)           {}
func (s *stubObject) ComputedStyle() *style.Computed    { return nil }
func (s *stubObject) SetComputedStyle(*style.Computed)  {}
func (s *stubObject) Flags() render.DirtyFlag           { return 0 }
func (s *stubObject) MarkDirty(render.DirtyFlag)        {}
func (s *stubObject) ClearDirty(render.DirtyFlag)       {}
func (s *stubObject) IsDirtySet(render.DirtyFlag) bool  { return false }
func (s *stubObject) IsDirtyStyle() bool                { return false }
func (s *stubObject) IsDirtyLayout() bool               { return false }
func (s *stubObject) IsDirtyPaint() bool                { return false }
func (s *stubObject) IsDirtyScroll() bool               { return false }
func (s *stubObject) IsDirtyStructure() bool            { return false }
func (s *stubObject) LayoutFlags() render.LayoutFlag    { return 0 }
func (s *stubObject) SetLayoutFlag(render.LayoutFlag)   {}
func (s *stubObject) ClearLayoutFlag(render.LayoutFlag) {}
func (s *stubObject) Focusable() bool                   { return false }
func (s *stubObject) SetFocusable(bool)                 {}
func (s *stubObject) Disabled() bool                    { return false }
func (s *stubObject) SetDisabled(bool)                  {}

// addChild sets parent and appends child.
func addChild(parent, child *stubObject) {
	child.parent = parent
	parent.children = append(parent.children, child)
}

// buildPath builds a root→target path from ancestor chain by walking parents.
func buildPath(objs ...*stubObject) []render.Object {
	path := make([]render.Object, len(objs))
	for i, o := range objs {
		path[i] = o
	}
	return path
}

// newRegistry builds an EventTargetResolver and a map for registering targets.
func newRegistry() (event.EventTargetResolver, map[render.Object]*event.EventTarget) {
	m := make(map[render.Object]*event.EventTarget)
	resolver := func(obj render.Object) *event.EventTarget { return m[obj] }
	return resolver, m
}

func ensureTarget(m map[render.Object]*event.EventTarget, obj render.Object) *event.EventTarget {
	if _, ok := m[obj]; !ok {
		m[obj] = &event.EventTarget{}
	}
	return m[obj]
}

// ---------------------------------------------------------------------------
// Dispatcher tests
// ---------------------------------------------------------------------------

func TestDispatcher_CaptureTargetBubble_Order(t *testing.T) {
	t.Parallel()

	root := newStub(layout.Rect{Size: layout.Size{Width: 80, Height: 24}})
	mid := newStub(layout.Rect{Size: layout.Size{Width: 40, Height: 12}})
	target := newStub(layout.Rect{Size: layout.Size{Width: 20, Height: 6}})
	addChild(root, mid)
	addChild(mid, target)

	resolver, reg := newRegistry()
	d := event.NewDispatcher(resolver)

	var order []string

	ensureTarget(reg, root).AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "root-capture")
	}, event.Capture())
	ensureTarget(reg, mid).AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "mid-capture")
	}, event.Capture())
	ensureTarget(reg, target).AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "target-capture")
	}, event.Capture())
	ensureTarget(reg, target).AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "target-bubble")
	})
	ensureTarget(reg, mid).AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "mid-bubble")
	})
	ensureTarget(reg, root).AddEventListener(event.EventClick, func(e event.Event) {
		order = append(order, "root-bubble")
	})

	path := buildPath(root, mid, target)
	e := event.NewMouseEvent(event.EventClick, layout.Point{}, event.ButtonLeft, 0)
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

	root := newStub(layout.Rect{})
	target := newStub(layout.Rect{})

	resolver, reg := newRegistry()
	d := event.NewDispatcher(resolver)

	reached := false
	ensureTarget(reg, target).AddEventListener(event.EventClick, func(e event.Event) {
		e.StopPropagation()
	})
	ensureTarget(reg, root).AddEventListener(event.EventClick, func(_ event.Event) {
		reached = true
	})

	path := buildPath(root, target)
	e := event.NewMouseEvent(event.EventClick, layout.Point{}, event.ButtonLeft, 0)
	d.Dispatch(e, path)

	if reached {
		t.Error("root bubble listener should not have been reached after StopPropagation")
	}
}

func TestDispatcher_PreventDefault_FlagOnly(t *testing.T) {
	t.Parallel()

	root := newStub(layout.Rect{})
	target := newStub(layout.Rect{})

	resolver, reg := newRegistry()
	d := event.NewDispatcher(resolver)

	ensureTarget(reg, target).AddEventListener(event.EventClick, func(e event.Event) {
		e.PreventDefault()
	})

	rootReached := false
	ensureTarget(reg, root).AddEventListener(event.EventClick, func(_ event.Event) {
		rootReached = true
	})

	path := buildPath(root, target)
	e := event.NewMouseEvent(event.EventClick, layout.Point{}, event.ButtonLeft, 0)
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

	obj := newStub(layout.Rect{})
	resolver, reg := newRegistry()
	d := event.NewDispatcher(resolver)
	et := ensureTarget(reg, obj)

	calls := 0
	sub := et.AddEventListener(event.EventClick, func(_ event.Event) {
		calls++
	})
	sub.Cancel()

	path := buildPath(obj)
	e := event.NewMouseEvent(event.EventClick, layout.Point{}, event.ButtonLeft, 0)
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

// stubHitTester returns a fixed render.Object regardless of coordinates.
type stubHitTester struct {
	result render.Object
}

func (h *stubHitTester) HitTest(_, _ int) render.Object { return h.result }

// stubFocus always returns the same focused object.
type stubFocus struct {
	obj render.Object
}

func (f *stubFocus) FocusedObject() render.Object { return f.obj }

func TestSynthesizer_ClickWithinTolerance(t *testing.T) {
	t.Parallel()

	target := newStub(layout.Rect{})
	hit := &stubHitTester{result: target}
	focus := &stubFocus{obj: target}
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

	target := newStub(layout.Rect{})
	hit := &stubHitTester{result: target}
	focus := &stubFocus{obj: target}
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

	target := newStub(layout.Rect{Origin: layout.Point{X: 5, Y: 5}, Size: layout.Size{Width: 10, Height: 5}})
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
	if me.Hit.Object != target {
		t.Errorf("expected hit object to be target, got %v", me.Hit.Object)
	}
}

// ---------------------------------------------------------------------------
// Hit test tests (via engine stub)
// ---------------------------------------------------------------------------

// stubRenderView is a minimal render tree for hit-test testing.
type stubRenderView struct {
	stubObject
	overlays []render.Object
}

func (v *stubRenderView) Overlays() []render.Object { return v.overlays }

// hitTestObject replicates the engine's logic for testing.
func hitTestObject(obj render.Object, p layout.Point) render.Object {
	if !obj.Bounds().Contains(p) {
		return nil
	}
	var last render.Object
	for child := range obj.Children() {
		last = child
	}
	for child := last; child != nil; child = child.PreviousSibling() {
		if hit := hitTestObject(child, p); hit != nil {
			return hit
		}
	}
	return obj
}

type testHitTester struct {
	view *stubRenderView
}

func (h *testHitTester) HitTest(x, y int) render.Object {
	p := layout.Point{X: x, Y: y}
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
			bounds: layout.Rect{Size: layout.Size{Width: 80, Height: 24}},
		},
	}
	// Document child at (0,0)-(20,10).
	docChild := newStub(layout.Rect{Size: layout.Size{Width: 20, Height: 10}})
	addChild(&view.stubObject, docChild)

	// Overlay at (5,5)-(15,15) covering the same point.
	overlay := newStub(layout.Rect{
		Origin: layout.Point{X: 5, Y: 5},
		Size:   layout.Size{Width: 10, Height: 10},
	})
	view.overlays = []render.Object{overlay}

	ht := &testHitTester{view: view}

	// Point (7,7) is covered by both docChild and overlay.
	hit := ht.HitTest(7, 7)
	if hit != overlay {
		t.Errorf("expected overlay to win hit test, got %v", hit)
	}
}

func TestHitTest_TopmostObject(t *testing.T) {
	t.Parallel()

	parent := newStub(layout.Rect{Size: layout.Size{Width: 80, Height: 24}})
	child1 := newStub(layout.Rect{Size: layout.Size{Width: 40, Height: 10}})
	child2 := newStub(layout.Rect{
		Origin: layout.Point{X: 5, Y: 0},
		Size:   layout.Size{Width: 40, Height: 10},
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

	root := newStub(layout.Rect{})
	mid := newStub(layout.Rect{})
	target := newStub(layout.Rect{})
	addChild(root, mid)
	addChild(mid, target)

	resolver, _ := newRegistry()
	d := event.NewDispatcher(resolver)

	sc := &stubScrollable{}
	scrollables := map[render.Object]event.Scrollable{mid: sc}

	path := buildPath(root, mid, target)
	e := event.NewWheelEvent(layout.Point{}, 0, 3, 0)
	d.DispatchWheel(e, path, scrollables)

	if len(sc.calls) != 1 {
		t.Errorf("expected 1 wheel call on mid scrollable, got %d", len(sc.calls))
	}
}

func TestWheel_NoScrollable_NoOp(t *testing.T) {
	t.Parallel()

	root := newStub(layout.Rect{})
	target := newStub(layout.Rect{})

	resolver, reg := newRegistry()
	d := event.NewDispatcher(resolver)

	rootCalled := false
	ensureTarget(reg, root).AddEventListener(event.EventWheel, func(_ event.Event) {
		rootCalled = true
	})

	path := buildPath(root, target)
	e := event.NewWheelEvent(layout.Point{}, 0, 1, 0)
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
	data string
}

func (c *stubClipboard) GetClipboard() string     { return c.data }
func (c *stubClipboard) SetClipboard(text string) { c.data = text }

func TestPaste_BracketedSequence_BecomesSinglePasteEvent(t *testing.T) {
	t.Parallel()

	cb := &stubClipboard{}
	s := event.NewSynthesizer(nil, nil, event.SynthesizerOptions{Clipboard: cb})

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
type stubLogicalNode struct {
	selected string
}

func (s *stubLogicalNode) SelectedText() string { return s.selected }

// stubObjectWithLogical wraps stubObject to return a custom LogicalNode.
type stubObjectWithLogical struct {
	stubObject
	node any
}

func (s *stubObjectWithLogical) LogicalNode() any { return s.node }

func TestClipboard_CopyCut_FromSelectionProvider(t *testing.T) {
	t.Parallel()

	ln := &stubLogicalNode{selected: "selected text"}
	focusedObj := &stubObjectWithLogical{node: ln}
	focus := &stubFocus{obj: focusedObj}
	cb := &stubClipboard{}
	s := event.NewSynthesizer(nil, focus, event.SynthesizerOptions{Clipboard: cb})

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
		t.Fatal("expected ClipboardEvent{Copy} on Ctrl+C with selection")
	}
	if copyEvt.Data != "selected text" {
		t.Errorf("copy data = %q, want %q", copyEvt.Data, "selected text")
	}
	if cb.data != "selected text" {
		t.Errorf("clipboard not written; got %q", cb.data)
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
		t.Fatal("expected ClipboardEvent{Cut} on Ctrl+X with selection")
	}
}

func TestClipboard_Paste_FromCtrlV_AndBracketedPaste(t *testing.T) {
	t.Parallel()

	cb := &stubClipboard{data: "clipboard content"}
	s := event.NewSynthesizer(nil, nil, event.SynthesizerOptions{Clipboard: cb})

	// Ctrl+V → ClipboardEvent{Paste}.
	evts := s.Process(&event.RawKeyEvent{Key: key.Key{Code: 'v', Mod: key.ModCtrl}})
	var pasteEvt *event.ClipboardEvent
	for _, ev := range evts {
		if ce, ok := ev.(*event.ClipboardEvent); ok && ce.ClipType == event.ClipboardPaste {
			pasteEvt = ce
			break
		}
	}
	if pasteEvt == nil {
		t.Fatal("expected ClipboardEvent{Paste} on Ctrl+V")
	}
	if pasteEvt.Data != "clipboard content" {
		t.Errorf("paste data = %q, want %q", pasteEvt.Data, "clipboard content")
	}

	// Bracketed paste → ClipboardEvent{Paste}.
	evts = s.Process(&event.RawBracketedPaste{Text: "pasted"})
	pasteEvt = nil
	for _, ev := range evts {
		if ce, ok := ev.(*event.ClipboardEvent); ok && ce.ClipType == event.ClipboardPaste {
			pasteEvt = ce
			break
		}
	}
	if pasteEvt == nil {
		t.Fatal("expected ClipboardEvent{Paste} from bracketed paste")
	}
	if pasteEvt.Data != "pasted" {
		t.Errorf("paste data = %q, want %q", pasteEvt.Data, "pasted")
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
