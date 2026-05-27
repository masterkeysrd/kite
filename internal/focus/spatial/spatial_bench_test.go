package spatial_test

import (
	"testing"

	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/focus"
	"github.com/masterkeysrd/kite/internal/focus/spatial"
)

// buildGridRightAnchor creates a root with n focusable children in a horizontal
// row. Current focus is at the far right so all others are candidates for
// DirectionLeft.
func buildGridRightAnchor(n int) (*spatialObj, *spatialObj, *focus.Manager) {
	root := newContainer()

	const w, h = 3, 2
	for i := range n - 1 {
		x := i * (w + 1)
		node := newFocusable(geom.Rect{
			Origin: geom.Point{X: x, Y: 0},
			Size:   geom.Size{Width: w, Height: h},
		})
		link(root, node)
	}

	curX := (n - 1) * (w + 1)
	cur := newFocusable(geom.Rect{
		Origin: geom.Point{X: curX, Y: 0},
		Size:   geom.Size{Width: w, Height: h},
	})
	link(root, cur)

	m := makeManager(root)
	m.SetFocus(cur, focus.ReasonProgrammatic)
	return root, cur, m
}

// buildGridLeftAnchor creates a root with n focusable children in a horizontal
// row. Current focus is at the far LEFT so DirectionLeft has NO candidates.
// Navigate returns false without calling m.Focus, isolating the pure traversal
// and scoring hot-path.
func buildGridLeftAnchor(n int) (*spatialObj, *spatialObj, *focus.Manager) {
	root := newContainer()

	const w, h = 3, 2
	cur := newFocusable(geom.Rect{
		Origin: geom.Point{X: 0, Y: 0},
		Size:   geom.Size{Width: w, Height: h},
	})
	link(root, cur)

	for i := range n - 1 {
		x := (i + 1) * (w + 1)
		node := newFocusable(geom.Rect{
			Origin: geom.Point{X: x, Y: 0},
			Size:   geom.Size{Width: w, Height: h},
		})
		link(root, node)
	}

	m := makeManager(root)
	m.SetFocus(cur, focus.ReasonProgrammatic)
	return root, cur, m
}

// BenchmarkNavigate_100Candidates measures Navigate with 100 candidates.
//
// The anchor is at the leftmost position so DirectionLeft finds no candidate
// and returns false without dispatching focus events. This isolates the
// traversal + scoring hot-path and must report 0 allocs/op.
func BenchmarkNavigate_100Candidates(b *testing.B) {
	_, _, m := buildGridLeftAnchor(100)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		spatial.Navigate(m, spatial.DirectionLeft)
	}
}

// BenchmarkNavigate_1000Candidates is a scaling sanity benchmark with 1000
// candidates. Same no-candidate setup as the 100-candidate benchmark.
func BenchmarkNavigate_1000Candidates(b *testing.B) {
	_, _, m := buildGridLeftAnchor(1000)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		spatial.Navigate(m, spatial.DirectionLeft)
	}
}

// BenchmarkNavigate_100Candidates_WithMove benchmarks the full Navigate path
// including m.Focus (focus mutation + event dispatch) when a candidate is
// found. Focus is reset to cur before each iteration so the scoring loop
// executes on every call.
func BenchmarkNavigate_100Candidates_WithMove(b *testing.B) {
	_, cur, m := buildGridRightAnchor(100)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		m.SetFocus(cur, focus.ReasonProgrammatic)
		spatial.Navigate(m, spatial.DirectionLeft)
	}
}
