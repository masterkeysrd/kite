package event_test

import (
	"testing"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
)

// BenchmarkDispatcher_3Phase_DepthN measures the cost of dispatching a
// bubbling event through a linear ancestor chain of depth N.
func BenchmarkDispatcher_3Phase_DepthN(b *testing.B) {
	const depth = 32
	nodes := make([]*stubObject, depth)
	for i := range depth {
		nodes[i] = newStub(geom.Rect{})
	}
	for i := 1; i < depth; i++ {
		nodes[i].parent = nodes[i-1]
	}

	d := event.NewDispatcher()
	// Register a capture and bubble listener at root.
	nodes[0].AddEventListener(event.EventClick, func(_ event.Event) {}, event.Capture())
	nodes[0].AddEventListener(event.EventClick, func(_ event.Event) {})
	// Register listeners at target.
	nodes[depth-1].AddEventListener(event.EventClick, func(_ event.Event) {})

	path := make([]event.EventTarget, depth)
	for i, n := range nodes {
		path[i] = n
	}

	b.ResetTimer()
	for range b.N {
		e := event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0)
		d.Dispatch(e, path)
	}
}

// BenchmarkHitTest_FullScreenTree measures hit-test performance over a
// representative flat tree (many siblings, no deep nesting).
func BenchmarkHitTest_FullScreenTree(b *testing.B) {
	const cols = 80
	const rows = 24
	root := newStub(geom.Rect{Size: geom.Size{Width: cols, Height: rows}})
	// Add one cell-sized child per cell.
	for y := range rows {
		for x := range cols {
			child := newStub(geom.Rect{
				Origin: geom.Point{X: x, Y: y},
				Size:   geom.Size{Width: 1, Height: 1},
			})
			addChild(root, child)
		}
	}

	view := &stubRenderView{stubObject: *root}
	view.stubObject.children = root.children
	ht := &testHitTester{view: view}

	b.ResetTimer()
	for i := range b.N {
		x := i % cols
		y := (i / cols) % rows
		_ = ht.HitTest(x, y)
	}
}

// BenchmarkSynthesizer_ClickStream_1k measures the cost of processing 1000
// mousedown+mouseup pairs through the synthesizer.
func BenchmarkSynthesizer_ClickStream_1k(b *testing.B) {
	target := newStub(geom.Rect{Size: geom.Size{Width: 80, Height: 24}})
	hit := &stubHitTester{result: target}
	s := event.NewSynthesizer(hit, nil, event.SynthesizerOptions{})

	b.ResetTimer()
	for range b.N {
		for range 1000 {
			s.Process(&event.RawMouseEvent{X: 5, Y: 5, Button: event.ButtonLeft})
			s.Process(&event.RawMouseEvent{X: 5, Y: 5, Button: event.ButtonLeft, Up: true})
		}
	}
}
