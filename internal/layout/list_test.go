package layout_test

import (
	"fmt"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	geometry "github.com/masterkeysrd/kite/geom"
	_ "github.com/masterkeysrd/kite/internal/dom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
)

func TestListLayout_MarkerSynthesis(t *testing.T) {
	tests := []struct {
		name       string
		styleType  style.ListStyleType
		expected   string
		expectFrag bool
	}{
		{"None", style.ListStyleNone, "", false},
		{"Disc", style.ListStyleDisc, "• ", true},
		{"Circle", style.ListStyleCircle, "○ ", true},
		{"Square", style.ListStyleSquare, "■ ", true},
		{"Decimal", style.ListStyleDecimal, "1. ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := render.NewBox(nil, nil)
			node.SetComputedStyle(&style.Computed{
				Display:       style.DisplayListItem,
				ListStyleType: tt.styleType,
			})

			space := layout.NewConstraintSpaceBuilder(geometry.Size{Width: 100, Height: 100}).ToConstraintSpace()
			algo := &layout.ListAlgorithm{}
			frag := algo.Layout(nil, node, space)

			if !tt.expectFrag {
				// Search for text fragment in children
				found := false
				for _, child := range frag.Children {
					if len(child.Fragment.Text) > 0 {
						found = true
						break
					}
				}
				if found {
					t.Error("expected no marker fragment, but found one")
				}
				return
			}

			// The marker should be the first child
			if len(frag.Children) == 0 {
				t.Fatal("expected at least one child (the marker)")
			}

			markerFrag := frag.Children[0].Fragment
			if len(markerFrag.Text) == 0 {
				t.Fatal("first child should be a text fragment (the marker)")
			}

			// Reconstruct string from shaped clusters
			got := ""
			for _, cluster := range markerFrag.Text {
				got += string(cluster.Bytes)
			}

			if got != tt.expected {
				t.Errorf("expected marker %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestListLayout_OrdinalCalculation(t *testing.T) {
	doc := dom.NewDocument()

	// Create a list of 5 items
	var nodes []layout.Node
	var domNodes []dom.Node

	for i := range 5 {
		el := doc.CreateElement("li", nil)
		ro := render.NewBox(el, el)
		ro.SetComputedStyle(&style.Computed{
			Display:       style.DisplayListItem,
			ListStyleType: style.ListStyleDecimal,
		})
		el.SetRenderObject(ro)

		if i > 0 {
			domNodes[i-1].Parent().AppendChild(el)
		} else {
			parent := doc.CreateElement("ul", nil)
			parent.AppendChild(el)
		}

		nodes = append(nodes, ro)
		domNodes = append(domNodes, el)
	}

	space := layout.NewConstraintSpaceBuilder(geometry.Size{Width: 100, Height: 100}).ToConstraintSpace()

	for i, node := range nodes {
		algo := &layout.ListAlgorithm{}
		frag := algo.Layout(nil, node, space)

		expectedMarker := fmt.Sprintf("%d. ", i+1)
		markerFrag := frag.Children[0].Fragment

		got := ""
		for _, cluster := range markerFrag.Text {
			got += string(cluster.Bytes)
		}

		if got != expectedMarker {
			t.Errorf("item %d: expected marker %q, got %q", i, expectedMarker, got)
		}
	}
}

func TestListLayout_InterruptedOrdinal(t *testing.T) {
	doc := dom.NewDocument()
	parent := doc.CreateElement("ul", nil)

	createItem := func(isListItem bool) (dom.Node, layout.Node) {
		el := doc.CreateElement("li", nil)
		ro := render.NewBox(el, el)
		display := style.DisplayBlock
		if isListItem {
			display = style.DisplayListItem
		}
		ro.SetComputedStyle(&style.Computed{
			Display:       display,
			ListStyleType: style.ListStyleDecimal,
		})
		el.SetRenderObject(ro)
		parent.AppendChild(el)
		return el, ro
	}

	_, i1 := createItem(true)
	_, i2 := createItem(true)
	createItem(false) // Interrupter
	_, i3 := createItem(true)

	space := layout.NewConstraintSpaceBuilder(geometry.Size{Width: 100, Height: 100}).ToConstraintSpace()

	check := func(node layout.Node, expected string) {
		algo := &layout.ListAlgorithm{}
		frag := algo.Layout(nil, node, space)
		markerFrag := frag.Children[0].Fragment
		got := ""
		for _, cluster := range markerFrag.Text {
			got += string(cluster.Bytes)
		}
		if got != expected {
			t.Errorf("expected marker %q, got %q", expected, got)
		}
	}

	check(i1, "1. ")
	check(i2, "2. ")
	check(i3, "1. ") // Reset because of interruption
}

func TestListLayout_MultiLineWrapping(t *testing.T) {
	// A list item with long text that should wrap
	node := render.NewBox(nil, nil)
	node.SetComputedStyle(&style.Computed{
		Display:       style.DisplayListItem,
		ListStyleType: style.ListStyleDisc, // "• " (width 2)
	})

	// Add a text child
	textNode := dom.NewDocument().CreateTextNode("this is a long text that should wrap", nil)
	node.InsertChild(render.NewText(textNode, nil), nil)

	// Available width 10. Marker is 2. Content width is 8.
	space := layout.NewConstraintSpaceBuilder(geometry.Size{Width: 10, Height: 100}).ToConstraintSpace()
	algo := &layout.ListAlgorithm{}
	frag := algo.Layout(nil, node, space)

	// Marker should be at (0,0)
	if frag.Children[0].Offset != (geometry.Point{X: 0, Y: 0}) {
		t.Errorf("expected marker at (0,0), got %v", frag.Children[0].Offset)
	}

	// Content lines should be at X=2
	for i := 1; i < len(frag.Children); i++ {
		if frag.Children[i].Offset.X != 2 {
			t.Errorf("line %d: expected X=2, got %v", i, frag.Children[i].Offset.X)
		}
		if frag.Children[i].Fragment.Size.Width > 8 {
			t.Errorf("line %d: expected width <= 8, got %d", i, frag.Children[i].Fragment.Size.Width)
		}
	}
}
