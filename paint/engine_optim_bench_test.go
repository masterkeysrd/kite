package paint

import (
	"testing"

	"image/color"

	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/text"
)

func BenchmarkPaint_All(b *testing.B) {
	// A more complex tree to exercise all optimizations
	// 1000 nodes total, some offscreen, some with borders, some nested.
	const totalNodes = 1000
	const viewportWidth = 100
	const viewportHeight = 50

	root := &layout.Fragment{
		Size: layout.Size{Width: viewportWidth, Height: 2000}, // Very tall
		Node: &mockNode{s: &style.Computed{}},
	}

	for i := 0; i < totalNodes; i++ {
		s := &style.Computed{}
		if i%5 == 0 {
			s.Border = style.SingleBorder()
		}
		if i%10 == 0 {
			s.Background = color.RGBA{100, 100, 100, 255}
		}

		child := &layout.Fragment{
			Size: layout.Size{Width: 20, Height: 1},
			Node: &mockNode{s: s},
			Text: []text.Cluster{{Bytes: []byte("hello world"), CellWidth: 11}},
		}

		root.Children = append(root.Children, layout.FragmentLink{
			Offset:   layout.Point{X: 0, Y: i * 2},
			Fragment: child,
		})
	}

	pe := &PaintEngine{}
	fb := NewFrameBuffer(0, 0, viewportWidth, viewportHeight)

	b.Run("FullPaint", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			fb.BumpVersion()
			pe.Paint(root, fb)
		}
	})
}

func BenchmarkResolveBorders(b *testing.B) {
	const w, h = 200, 60
	fb := NewFrameBuffer(0, 0, w, h)
	pe := NewPaintEngine()

	// Fill with some borders using the same mechanism as the engine
	pe.rootSurface = fb
	pe.clipStack = append(pe.clipStack, fb.Bounds())
	for y := 0; y < h; y += 2 {
		for x := 0; x < w; x += 2 {
			pe.setCell(x, y, Cell{BorderStyle: BorderSingle, Content: "│"})
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		pe.resolveBorders(fb)
	}
}
