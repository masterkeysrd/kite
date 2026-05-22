package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
)

func TestTableBorders(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	// Table with borders on every cell, no manual margins
	root := element.Box(
		element.Table(
			element.TR(
				element.TD("A").Style(style.Style{Border: style.SingleBorder().Some()}),
				element.TD("B").Style(style.Style{Border: style.SingleBorder().Some()}),
			),
			element.TR(
				element.TD("C").Style(style.Style{Border: style.SingleBorder().Some()}),
				element.TD("D").Style(style.Style{Border: style.SingleBorder().Some()}),
			),
		).Style(style.Style{
			Width:  style.Some(style.Percent(100)),
			Border: style.SingleBorder().Some(),
		}),
	).Style(style.Style{
		Padding: style.Some(style.Edges(1)),
	})

	env.Mount(root)
	env.RenderFrame()

	// Expect a border junction character somewhere in the rendered frame.
	testenv.Expect(t, root).ToHaveCellContentInFrame(env, "┼")
}
