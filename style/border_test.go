package style

import (
	"image/color"
	"testing"
)

func TestBorderFluentAPI(t *testing.T) {
	red := color.RGBA{255, 0, 0, 255}

	b := SingleBorder().Color(red).Top(false)

	if b.Edges.Top != false {
		t.Errorf("Expected Top edge to be false")
	}
	if b.Edges.Bottom != true {
		t.Errorf("Expected Bottom edge to be true")
	}
	if b.Styles.Bottom != BorderSingle {
		t.Errorf("Expected Bottom style to be BorderSingle")
	}
	if b.Colors.Bottom != red {
		t.Errorf("Expected Bottom color to be red")
	}

	w := b.Widths()
	if w.Top != 0 {
		t.Errorf("Expected Top width to be 0")
	}
	if w.Bottom != 1 {
		t.Errorf("Expected Bottom width to be 1")
	}
}

func TestRoundedBorder(t *testing.T) {
	b := RoundedBorder()
	if b.Styles.Top != BorderRounded {
		t.Errorf("Expected BorderRounded style")
	}
	if b.Edges.Top != true || b.Edges.Bottom != true || b.Edges.Left != true || b.Edges.Right != true {
		t.Errorf("Expected all edges to be true")
	}
}
