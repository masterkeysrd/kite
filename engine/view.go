package engine

import (
	"github.com/masterkeysrd/kite/cursor"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/style"
)

type domViewProxy struct {
	e *Engine
}

func (p *domViewProxy) GetBoundingClientRect(n dom.Node) (geom.Rect, bool) {
	p.e.EnsureFreshLayout()
	ro := p.e.RenderObject(n)
	if ro == nil {
		return geom.Rect{}, false
	}
	root := p.e.renderView.Fragment()
	if root == nil {
		return geom.Rect{}, false
	}
	rect, _, found := layout.ScrolledAbsoluteBounds(root, ro)
	return rect, found
}

func (p *domViewProxy) GetComputedStyle(n dom.Node) *style.Computed {
	p.e.EnsureFreshLayout()
	ro := p.e.RenderObject(n)
	if ro == nil {
		return nil
	}
	return ro.ComputedStyle()
}

func (p *domViewProxy) GetSize(n dom.Node) (geom.Size, bool) {
	p.e.EnsureFreshLayout()
	ro := p.e.RenderObject(n)
	if ro == nil {
		return geom.Size{}, false
	}
	frag := ro.Fragment()
	if frag == nil {
		return geom.Size{}, false
	}
	return frag.Size, true
}

func (p *domViewProxy) GetFragment(n dom.Node) *layout.Fragment {
	p.e.EnsureFreshLayout()
	ro := p.e.RenderObject(n)
	if ro == nil {
		return nil
	}
	return ro.Fragment()
}

func (p *domViewProxy) GetMaxScroll(n dom.Node) (x, y int) {
	p.e.EnsureFreshLayout()
	ro := p.e.RenderObject(n)
	if ro == nil {
		return 0, 0
	}
	return ro.MaxScroll()
}

func (p *domViewProxy) GetCaretPosition(n dom.Node, offset int) (geom.Point, bool) {
	p.e.EnsureFreshLayout()
	ro := p.e.RenderObject(n)
	if ro == nil {
		return geom.Point{}, false
	}
	frag := ro.Fragment()
	if frag == nil {
		return geom.Point{}, false
	}
	x, y, ok := cursor.FromTextFragment(frag, offset)
	return geom.Point{X: x, Y: y}, ok
}

func (p *domViewProxy) MoveCursorVertically(n dom.Node, offset int, delta int, x, y int) int {
	p.e.EnsureFreshLayout()
	ro := p.e.RenderObject(n)
	if ro == nil {
		return offset
	}
	frag := ro.Fragment()
	if frag == nil {
		return offset
	}

	targetY := y + delta
	if delta < 0 {
		if y <= 0 {
			return offset
		}
	} else {
		// Find max Y
		maxY := 0
		for _, c := range frag.Children {
			if c.Offset.Y > maxY {
				maxY = c.Offset.Y
			}
		}
		if y >= maxY {
			return offset
		}
	}

	return cursor.ByteOffsetAtPoint(frag, x, targetY)
}

func (p *domViewProxy) ByteOffsetAtPoint(n dom.Node, x, y int) int {
	p.e.EnsureFreshLayout()
	ro := p.e.RenderObject(n)
	if ro == nil {
		return 0
	}
	frag := ro.Fragment()
	if frag == nil {
		return 0
	}
	return cursor.ByteOffsetAtPoint(frag, x, y)
}

func (p *domViewProxy) NodeAtPoint(x, y int) (dom.Node, int) {
	p.e.EnsureFreshLayout()
	root := p.e.renderView.Fragment()
	if root == nil {
		return nil, 0
	}
	byteOffset := cursor.ByteOffsetAtPoint(root, x, y)
	return p.e.document.FindNodeAtByteOffset(p.e.document, byteOffset)
}

func (p *domViewProxy) GetScrolledAbsoluteBounds(n dom.Node) (geom.Rect, geom.Rect, bool) {
	p.e.EnsureFreshLayout()
	ro := p.e.RenderObject(n)
	if ro == nil {
		return geom.Rect{}, geom.Rect{}, false
	}
	root := p.e.renderView.Fragment()
	if root == nil {
		return geom.Rect{}, geom.Rect{}, false
	}
	return layout.ScrolledAbsoluteBounds(root, ro)
}

func (p *domViewProxy) ViewportSize() geom.Size {
	return p.e.renderView.ViewportSize()
}
