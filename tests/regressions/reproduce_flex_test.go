package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestFragmentLayout_NegativeShrink(t *testing.T) {
	e := testenv.Default(160, 80)
	defer e.Close()

	var child4Ref kitex.Ref[dom.Element]
	var child5Ref kitex.Ref[dom.Element]

	App := kitex.FC("App", func(props struct{}) kitex.Node {
		child4Ref = kitex.UseRef[dom.Element](nil)
		child5Ref = kitex.UseRef[dom.Element](nil)

		return kitex.Box(kitex.BoxProps{
			Style: style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Overflow(style.OverflowAuto).
				Padding(1).
				Width(style.Cells(131)).
				Height(style.Cells(59)).
				Gap(1),
		},
			kitex.Box(kitex.BoxProps{Style: style.S().Height(style.Cells(5)).Width(style.Cells(100))}),
			kitex.Box(kitex.BoxProps{Style: style.S().Height(style.Cells(68)).Width(style.Cells(100))}),
			kitex.Box(kitex.BoxProps{Style: style.S().Height(style.Cells(20)).Width(style.Cells(100))}),
			kitex.Box(kitex.BoxProps{Style: style.S().Height(style.Cells(3)).Width(style.Cells(100))}),
			kitex.Box(kitex.BoxProps{
				Ref:   child4Ref,
				Style: style.S().Width(style.Cells(100)), // height is auto (0)
			}),
			kitex.Box(kitex.BoxProps{
				Ref:   child5Ref,
				Style: style.S().Height(style.Cells(4)).Width(style.Cells(100)),
			}),
		)
	})

	container := e.Document().CreateElement("div", nil)
	e.Document().AppendChild(container)

	kitex.Render(App(struct{}{}), container)
	e.RenderFrame()

	ro4 := e.RenderObject(child4Ref.Current)
	ro5 := e.RenderObject(child5Ref.Current)

	height4 := ro4.Fragment().Size.Height
	height5 := ro5.Fragment().Size.Height
	offset5 := ro5.Offset()

	t.Logf("Child 4 height: %d", height4)
	t.Logf("Child 5 height: %d", height5)
	t.Logf("Child 5 offset Y: %d", offset5.Y)

	if height4 < 0 {
		t.Errorf("Child 4 has negative height: %d", height4)
	}
}
