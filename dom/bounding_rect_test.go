package dom_test

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/style"
)

type mockRO struct {
	fakeRO
	fragment *layout.Fragment
	logical  any
	style    *style.Computed
}

func (m *mockRO) Fragment() *layout.Fragment { return m.fragment }
func (m *mockRO) LogicalNode() any           { return m.logical }
func (m *mockRO) Style() *style.Computed     { return m.style }

func TestGetBoundingClientRect(t *testing.T) {
	doc := dom.NewDocument()
	div := doc.CreateElement("div", nil)
	doc.AppendChild(div)

	// Build Fragment tree
	// Target fragment (for div)
	childFrag := &layout.Fragment{
		Size: geom.Size{Width: 10, Height: 5},
	}

	// Root fragment (for Document)
	rootFrag := &layout.Fragment{
		Children: []layout.FragmentLink{
			{
				Offset:   geom.Point{X: 2, Y: 3},
				Fragment: childFrag,
			},
		},
		Size: geom.Size{Width: 80, Height: 24},
	}

	// We need to set RenderObjects
	docRO := &mockRO{
		fragment: rootFrag,
		style:    &style.Computed{},
	}
	divRO := &mockRO{
		fragment: childFrag,
		style:    &style.Computed{},
	}

	doc.SetRenderObject(docRO)
	div.SetRenderObject(divRO)

	// IMPORTANT: layout.AbsoluteBounds (and ScrolledAbsoluteBounds) compares Fragment.Node with the target Node.
	childFrag.Node = divRO
	rootFrag.Node = docRO

	rect, ok := div.GetBoundingClientRect()
	if !ok {
		t.Fatal("GetBoundingClientRect returned !ok")
	}

	expected := geom.Rect{
		Origin: geom.Point{X: 2, Y: 3},
		Size:   geom.Size{Width: 10, Height: 5},
	}
	if rect != expected {
		t.Errorf("got %+v, want %+v", rect, expected)
	}
}

func TestGetBoundingClientRect_Scrolled(t *testing.T) {
	doc := dom.NewDocument()
	container := doc.CreateElement("div", nil)
	doc.AppendChild(container)

	div := doc.CreateElement("div", nil)
	container.AppendChild(div)

	// Build Fragment tree
	// Target fragment (for div)
	childFrag := &layout.Fragment{
		Size: geom.Size{Width: 10, Height: 5},
	}

	// Container fragment
	containerFrag := &layout.Fragment{
		Children: []layout.FragmentLink{
			{
				Offset:   geom.Point{X: 5, Y: 5},
				Fragment: childFrag,
			},
		},
		Size: geom.Size{Width: 5, Height: 5},
	}

	// Root fragment
	rootFrag := &layout.Fragment{
		Children: []layout.FragmentLink{
			{
				Offset:   geom.Point{X: 0, Y: 0},
				Fragment: containerFrag,
			},
		},
		Size: geom.Size{Width: 80, Height: 24},
	}

	// Mock RenderObjects
	docRO := &mockRO{
		fragment: rootFrag,
		style:    &style.Computed{},
	}
	containerRO := &mockRO{
		fragment: containerFrag,
		logical:  container,
		style: &style.Computed{
			OverflowX: style.OverflowScroll,
			OverflowY: style.OverflowScroll,
		},
	}
	divRO := &mockRO{
		fragment: childFrag,
		style:    &style.Computed{},
	}

	doc.SetRenderObject(docRO)
	container.SetRenderObject(containerRO)
	div.SetRenderObject(divRO)

	rootFrag.Node = docRO
	containerFrag.Node = containerRO
	childFrag.Node = divRO

	// Initial check (no scroll)
	rect, ok := div.GetBoundingClientRect()
	if !ok {
		t.Fatal("GetBoundingClientRect returned !ok")
	}
	expected := geom.Rect{
		Origin: geom.Point{X: 5, Y: 5},
		Size:   geom.Size{Width: 10, Height: 5},
	}
	if rect != expected {
		t.Errorf("Initial: got %+v, want %+v", rect, expected)
	}

	// Apply scroll to container
	container.ScrollTo(2, 3)

	// Check again
	rect, ok = div.GetBoundingClientRect()
	if !ok {
		t.Fatal("GetBoundingClientRect returned !ok after scroll")
	}
	expectedScrolled := geom.Rect{
		Origin: geom.Point{X: 3, Y: 2}, // (5-2, 5-3)
		Size:   geom.Size{Width: 10, Height: 5},
	}
	if rect != expectedScrolled {
		t.Errorf("Scrolled: got %+v, want %+v", rect, expectedScrolled)
	}
}

func TestGetBoundingClientRect_Disconnected(t *testing.T) {
	doc := dom.NewDocument()
	div := doc.CreateElement("div", nil)
	// NOT appended to doc

	_, ok := div.GetBoundingClientRect()
	if ok {
		t.Error("GetBoundingClientRect should return !ok for disconnected element")
	}
}

func TestGetBoundingClientRect_NoFragment(t *testing.T) {
	doc := dom.NewDocument()
	div := doc.CreateElement("div", nil)
	doc.AppendChild(div)

	// doc has RO but no fragment
	doc.SetRenderObject(&mockRO{fragment: nil})
	div.SetRenderObject(&mockRO{fragment: nil})

	_, ok := div.GetBoundingClientRect()
	if ok {
		t.Error("GetBoundingClientRect should return !ok when no fragment is available")
	}
}
