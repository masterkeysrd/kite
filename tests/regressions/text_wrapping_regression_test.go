package regressions

import (
	"fmt"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestTextWrappingInFlexScrollContainer(t *testing.T) {
	// A viewport of width 40, height 20.
	e := testenv.Default(40, 20)
	defer e.Close()

	var textRef kitex.Ref[dom.Element]

	App := kitex.FC("App", func(props struct{}) kitex.Node {
		textRef = kitex.UseRef[dom.Element](nil)

		// Nested layout:
		// Scroll Container (Flex column, overflow-y: auto, width: 40)
		//   -> Message Row (Flex row, width: 100%)
		//        -> Message Bubble (Flex column/block, flex-grow: 1, OverflowWrapBreakWord)
		//             -> Text Element (long string)
		return kitex.Box(kitex.BoxProps{
			Style: style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				OverflowY(style.OverflowAuto).
				Width(style.Cells(40)).
				Height(style.Cells(20)),
		},
			kitex.Box(kitex.BoxProps{
				Style: style.S().
					Display(style.DisplayFlex).
					FlexDirection(style.FlexRow).
					Width(style.Percent(100)),
			},
				kitex.Box(kitex.BoxProps{
					Style: style.S().
						Display(style.DisplayFlex).
						FlexDirection(style.FlexColumn).
						Flex(1).
						OverflowWrap(style.OverflowWrapAnywhere),
				},
					kitex.Box(kitex.BoxProps{
						Ref: textRef,
						Style: style.S().
							OverflowWrap(style.OverflowWrapAnywhere),
					},
						kitex.Text("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"),
					),
				),
			),
		)
	})

	container := e.Document().CreateElement("div", nil)
	e.Document().AppendChild(container)

	kitex.Render(App(struct{}{}), container)
	e.RenderFrame()

	ro := e.RenderObject(textRef.Current)
	if ro == nil {
		t.Fatal("text element has no render object")
	}

	// Find the root render object to print the whole tree.
	var rootObj render.Object = ro
	for rootObj.Parent() != nil {
		rootObj = rootObj.Parent()
	}

	var printTree func(obj render.Object, indent string)
	printTree = func(obj render.Object, indent string) {
		styleStr := ""
		if s := obj.Style(); s != nil {
			styleStr = fmt.Sprintf("Display=%v WidthKind=%v HeightKind=%v FlexDirection=%v Flex=%+v OverflowY=%v OverflowWrap=%v",
				s.Display, s.Width.Kind(), s.Height.Kind(), s.FlexDirection, s.Flex, s.OverflowY, s.OverflowWrap)
		}
		fragSize := "nil"
		if f := obj.Fragment(); f != nil {
			fragSize = fmt.Sprintf("Size=%dx%d Children=%d TextLen=%d", f.Size.Width, f.Size.Height, len(f.Children), len(f.Text))
		}
		t.Logf("%s%T: %s | %s", indent, obj, styleStr, fragSize)
		for child := range obj.Children() {
			printTree(child, indent+"  ")
		}
	}
	printTree(rootObj, "")

	height := ro.Fragment().Size.Height
	width := ro.Fragment().Size.Width
	t.Logf("Shaped text size: Width=%d, Height=%d", width, height)

	// Since width is 40 and text is 52 chars, with OverflowWrapBreakWord it should wrap
	// and have Height >= 2. If constraint propagation is broken, it will measure text
	// as infinite width, resulting in Width=52 (overflowing 40) and Height=1.
	if height < 2 {
		t.Errorf("Expected text to wrap and have height >= 2, got height %d, width %d", height, width)
	}
}
