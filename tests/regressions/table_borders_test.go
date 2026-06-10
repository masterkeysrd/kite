package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestTableBorders(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	// Table with borders on every cell, no manual margins
	root := element.Box(
		element.Table(
			element.TR(
				element.TD("A").Style(style.S().Border(style.SingleBorder())),
				element.TD("B").Style(style.S().Border(style.SingleBorder())),
			),
			element.TR(
				element.TD("C").Style(style.S().Border(style.SingleBorder())),
				element.TD("D").Style(style.S().Border(style.SingleBorder())),
			),
		).Style(style.S().Width(style.Percent(100)).Border(style.SingleBorder())),
	).Style(style.S().Padding(1))

	env.Mount(root)
	env.RenderFrame()

	// Expect a border junction character somewhere in the rendered frame.
	testenv.Expect(t, root).ToHaveCellContentInFrame(env, "┼")
}
