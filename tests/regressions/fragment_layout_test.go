package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestFragmentLayout_OverflowAuto(t *testing.T) {
	e := testenv.Default(80, 20)
	defer e.Close()

	var setItems func([]string)
	var staticRef kitex.Ref[dom.Element]

	App := kitex.FC("App", func(props struct{}) kitex.Node {
		items, setItemsFn := kitex.UseState([]string{})
		setItems = setItemsFn
		staticRef = kitex.UseRef[dom.Element](nil)

		var children []kitex.Node
		for range items() {
			children = append(children, kitex.Box(kitex.BoxProps{
				Style: style.S().Height(style.Cells(2)).Width(style.Cells(10)),
			}))
		}

		return kitex.Box(kitex.BoxProps{
			Style: style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Overflow(style.OverflowAuto).
				Width(style.Cells(40)).
				Height(style.Cells(10)),
		},
			kitex.Fragment(children...),
			kitex.Box(kitex.BoxProps{
				Ref:   staticRef,
				Style: style.S().Height(style.Cells(1)).Width(style.Cells(10)),
			}),
		)
	})

	container := e.Document().CreateElement("div", nil)
	e.Document().AppendChild(container)

	kitex.Render(App(struct{}{}), container)
	e.RenderFrame()

	if staticRef == nil || staticRef.Current == nil {
		t.Fatal("staticRef is nil after initial render")
	}

	ro1 := e.RenderObject(staticRef.Current)
	if ro1 == nil {
		t.Fatal("staticRef has no render object")
	}
	offset1 := ro1.Offset()
	if offset1.Y != 0 {
		t.Errorf("expected initial Y offset to be 0, got %d", offset1.Y)
	}

	// Add 6 items of height 2 each (total 12 cells, which > 10 cells viewport)
	setItems([]string{"item1", "item2", "item3", "item4", "item5", "item6"})
	e.RenderFrame()

	ro2 := e.RenderObject(staticRef.Current)
	offset2 := ro2.Offset()
	// Total height of prepended dynamic items is 12 cells.
	// So the static sibling must be offset to Y = 12.
	if offset2.Y != 12 {
		t.Errorf("expected Y offset after update to be 12, got %d (sibling did not move down correctly!)", offset2.Y)
	}
}
