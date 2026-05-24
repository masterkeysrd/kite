package regressions

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/paint"
	"github.com/masterkeysrd/kite/style"
)

func TestTextSelectionPaint(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	// 1. Build a document with some text.
	doc := env.Document()
	div := element.NewBox(doc)
	div.Style(style.Style{
		Background: style.Some[color.Color](color.RGBA{R: 0, G: 0, B: 0, A: 255}),
		Foreground: style.Some[color.Color](color.RGBA{R: 255, G: 255, B: 255, A: 255}),
	})

	t1 := element.Text("Hello ")
	t2 := element.Text("Selection")
	t3 := element.Text(" World")

	div.AppendChild(t1)
	div.AppendChild(t2)
	div.AppendChild(t3)
	doc.AppendChild(div)

	// 2. Set selection on the middle text node.
	sel := doc.Selection()
	rng := doc.CreateRange()
	rng.SetStart(t2, 0)
	rng.SetEnd(t2, 9)
	sel.AddRange(rng)

	env.Flush()

	// 3. Verify that the "Selection" text has the inversion attribute.
	for x := 6; x < 15; x++ {
		testenv.ExpectScreen(t, env).CellAt(x, 0).ToHaveAttribute(paint.AttrInverse)
	}

	// "Hello " should NOT have inversion.
	testenv.ExpectScreen(t, env).CellAt(0, 0).ToHaveContent("H")
}

func TestTextSelectionCustomColors(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	selFG := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	selBG := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	doc := env.Document()
	div := element.NewBox(doc)
	div.Style(style.Style{
		SelectionForeground: style.Some[color.Color](selFG),
		SelectionBackground: style.Some[color.Color](selBG),
	})

	t1 := element.Text("Selected")
	div.AppendChild(t1)
	doc.AppendChild(div)

	sel := doc.Selection()
	rng := doc.CreateRange()
	rng.SetStart(t1, 0)
	rng.SetEnd(t1, 8)
	sel.AddRange(rng)

	env.Flush()

	// Verify custom colors
	for x := 0; x < 8; x++ {
		cell := env.Backend.LastFrame().Surface.CellAt(x, 0)
		if cell.FG != selFG {
			t.Errorf("at x=%d, expected FG %v, got %v", x, selFG, cell.FG)
		}
		if cell.BG != selBG {
			t.Errorf("at x=%d, expected BG %v, got %v", x, selBG, cell.BG)
		}
	}
}
