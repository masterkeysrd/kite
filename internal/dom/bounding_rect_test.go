package dom

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

type mockView struct {
	rects map[dom.Node]geom.Rect
}

func (m *mockView) GetBoundingClientRect(n dom.Node) (geom.Rect, bool) {
	r, ok := m.rects[n]
	return r, ok
}
func (m *mockView) GetComputedStyle(n dom.Node) *style.Computed { return nil }
func (m *mockView) NodeAtPoint(x, y int) (dom.Node, int)        { return nil, 0 }
func (m *mockView) ByteOffsetAtPoint(n dom.Node, x, y int) int  { return 0 }
func (m *mockView) GetCaretPosition(n dom.Node, offset int) (geom.Point, bool) {
	return geom.Point{}, false
}
func (m *mockView) MoveCursorVertically(n dom.Node, offset int, delta int, x, y int) int {
	return offset
}
func (m *mockView) GetSize(n dom.Node) (geom.Size, bool) { return geom.Size{}, false }
func (m *mockView) GetMaxScroll(n dom.Node) (x, y int)   { return 0, 0 }

func TestGetBoundingClientRect(t *testing.T) {
	doc := NewDocument()
	div := doc.CreateElement("div", nil)
	doc.AppendChild(div)

	expected := geom.Rect{
		Origin: geom.Point{X: 2, Y: 3},
		Size:   geom.Size{Width: 10, Height: 5},
	}

	doc.SetDefaultView(&mockView{
		rects: map[dom.Node]geom.Rect{
			div: expected,
		},
	})

	rect, ok := div.GetBoundingClientRect()
	if !ok {
		t.Fatal("GetBoundingClientRect returned !ok")
	}

	if rect != expected {
		t.Errorf("got %+v, want %+v", rect, expected)
	}
}
