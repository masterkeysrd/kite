package style

import (
	"encoding/json"
	"image/color"
	"testing"
)

// ---------------------------------------------------------------------------
// TestOptional_Merge
// ---------------------------------------------------------------------------

func TestOptional_Merge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     Optional[int]
		override Optional[int]
		wantVal  int
		wantSet  bool
	}{
		{
			name:    "both unset — result unset",
			wantSet: false,
		},
		{
			name:    "base set, override unset — base wins",
			base:    Some(42),
			wantVal: 42,
			wantSet: true,
		},
		{
			name:     "base unset, override set — override wins",
			override: Some(7),
			wantVal:  7,
			wantSet:  true,
		},
		{
			name:     "both set — override wins",
			base:     Some(1),
			override: Some(2),
			wantVal:  2,
			wantSet:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.base.Merge(tc.override)
			if got.IsSet() != tc.wantSet {
				t.Fatalf("IsSet()=%v want %v", got.IsSet(), tc.wantSet)
			}
			if tc.wantSet && got.Value() != tc.wantVal {
				t.Fatalf("Value()=%v want %v", got.Value(), tc.wantVal)
			}
		})
	}
}

func TestOptional_SetUnset(t *testing.T) {
	t.Parallel()

	var o Optional[string]
	if o.IsSet() {
		t.Fatal("new Optional should not be set")
	}

	o.Set("hello")
	if !o.IsSet() {
		t.Fatal("should be set after Set()")
	}
	if o.Value() != "hello" {
		t.Fatalf("Value()=%q want %q", o.Value(), "hello")
	}

	o.Unset()
	if o.IsSet() {
		t.Fatal("should not be set after Unset()")
	}
	if o.Value() != "" {
		t.Fatalf("Value() after Unset should be zero, got %q", o.Value())
	}
}

func TestOptional_JSON(t *testing.T) {
	t.Parallel()

	t.Run("marshal unset", func(t *testing.T) {
		var o Optional[int]
		data, err := json.Marshal(o)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "null" {
			t.Fatalf("got %q want null", string(data))
		}
	})

	t.Run("marshal set", func(t *testing.T) {
		o := Some(42)
		data, err := json.Marshal(o)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "42" {
			t.Fatalf("got %q want 42", string(data))
		}
	})

	t.Run("unmarshal null", func(t *testing.T) {
		var o Optional[int]
		o.Set(123)
		if err := json.Unmarshal([]byte("null"), &o); err != nil {
			t.Fatal(err)
		}
		if o.IsSet() {
			t.Fatal("should be unset after unmarshaling null")
		}
	})

	t.Run("unmarshal value", func(t *testing.T) {
		var o Optional[int]
		if err := json.Unmarshal([]byte("42"), &o); err != nil {
			t.Fatal(err)
		}
		if !o.IsSet() {
			t.Fatal("should be set after unmarshaling value")
		}
		if o.Value() != 42 {
			t.Fatalf("got %d want 42", o.Value())
		}
	})
}

// ---------------------------------------------------------------------------
// TestDimension_Kind
// ---------------------------------------------------------------------------

func TestDimension_Kind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dim  Dimension
		want DimensionKind
	}{
		{"Cells", Cells(10), KindCells},
		{"Percent", Percent(50), KindPercent},
		{"Fr", Fr(2.5), KindFr},
		{"Auto", Auto, KindAuto},
		{"Content", Content, KindContent},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.dim.Kind(); got != tc.want {
				t.Fatalf("Kind()=%v want %v", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestEdges_Variadic
// ---------------------------------------------------------------------------

func TestEdges_Variadic(t *testing.T) {
	t.Parallel()

	t.Run("1-arg all-equal", func(t *testing.T) {
		ev := Edges(5)
		if ev.Top != 5 || ev.Right != 5 || ev.Bottom != 5 || ev.Left != 5 {
			t.Fatalf("got %+v want all 5", ev)
		}
	})

	t.Run("2-arg vertical horizontal", func(t *testing.T) {
		ev := Edges(2, 4)
		if ev.Top != 2 || ev.Bottom != 2 {
			t.Fatalf("top/bottom should be 2, got top=%d bottom=%d", ev.Top, ev.Bottom)
		}
		if ev.Right != 4 || ev.Left != 4 {
			t.Fatalf("right/left should be 4, got right=%d left=%d", ev.Right, ev.Left)
		}
	})

	t.Run("4-arg TRBL", func(t *testing.T) {
		ev := Edges(1, 2, 3, 4)
		if ev.Top != 1 || ev.Right != 2 || ev.Bottom != 3 || ev.Left != 4 {
			t.Fatalf("got %+v want T=1 R=2 B=3 L=4", ev)
		}
	})

	t.Run("EdgeAll shorthand", func(t *testing.T) {
		ev := EdgeAll(7)
		if ev.Top != 7 || ev.Right != 7 || ev.Bottom != 7 || ev.Left != 7 {
			t.Fatalf("got %+v want all 7", ev)
		}
	})
}

// ---------------------------------------------------------------------------
// TestBorderGlyphs_CornerOverride
// ---------------------------------------------------------------------------

func TestBorderGlyphs_CornerOverride(t *testing.T) {
	t.Parallel()

	base := BorderGlyphs{
		H: "─", V: "│",
		TL: "╭", TR: "╮",
		BL: "╰", BR: "╯",
	}

	t.Run("no overrides — base glyphs returned", func(t *testing.T) {
		if got := base.EffectiveTL(); got != "╭" {
			t.Fatalf("EffectiveTL=%q want ╭", got)
		}
		if got := base.EffectiveBR(); got != "╯" {
			t.Fatalf("EffectiveBR=%q want ╯", got)
		}
	})

	t.Run("override BL and BR only", func(t *testing.T) {
		g := base
		g.OverrideBL.Set("├")
		g.OverrideBR.Set("┤")

		if got := g.EffectiveTL(); got != "╭" {
			t.Fatalf("EffectiveTL=%q want ╭ (unchanged)", got)
		}
		if got := g.EffectiveTR(); got != "╮" {
			t.Fatalf("EffectiveTR=%q want ╮ (unchanged)", got)
		}
		if got := g.EffectiveBL(); got != "├" {
			t.Fatalf("EffectiveBL=%q want ├", got)
		}
		if got := g.EffectiveBR(); got != "┤" {
			t.Fatalf("EffectiveBR=%q want ┤", got)
		}
	})

	t.Run("override TL and TR only", func(t *testing.T) {
		g := base
		g.OverrideTL.Set("├")
		g.OverrideTR.Set("┤")

		if got := g.EffectiveTL(); got != "├" {
			t.Fatalf("EffectiveTL=%q want ├", got)
		}
		if got := g.EffectiveTR(); got != "┤" {
			t.Fatalf("EffectiveTR=%q want ┤", got)
		}
		if got := g.EffectiveBL(); got != "╰" {
			t.Fatalf("EffectiveBL=%q want ╰ (unchanged)", got)
		}
		if got := g.EffectiveBR(); got != "╯" {
			t.Fatalf("EffectiveBR=%q want ╯ (unchanged)", got)
		}
	})

	t.Run("override cleared with Unset restores base", func(t *testing.T) {
		g := base
		g.OverrideBL.Set("X")
		g.OverrideBL.Unset()

		if got := g.EffectiveBL(); got != "╰" {
			t.Fatalf("EffectiveBL=%q want ╰ after Unset", got)
		}
	})
}

// ---------------------------------------------------------------------------
// TestStyle_Merge
// ---------------------------------------------------------------------------

func TestStyle_Merge(t *testing.T) {
	t.Parallel()

	base := Style{
		display:       Some(DisplayFlex),
		flexDirection: Some(FlexColumn),
		gap:           Some(Gap(2)),
		foreground:    Some(color.Color(color.RGBA{R: 200, G: 200, B: 200, A: 255})),
		bold:          Some(false),
	}

	t.Run("override replaces only set fields", func(t *testing.T) {
		override := Style{
			gap:  Some(Gap(4)),
			bold: Some(true),
		}
		merged := base.Merge(override)

		// Fields set in override must come from override.
		if v, _ := merged.gap.Value(), merged.gap.IsSet(); v != Gap(4) {
			t.Fatalf("Gap=%v want {4 4}", merged.gap.Value())
		}
		if v := merged.bold.Value(); !v {
			t.Fatal("Bold should be true after merge")
		}
		// Fields not set in override must come from base.
		if v := merged.display.Value(); v != DisplayFlex {
			t.Fatalf("Display=%v want DisplayFlex", v)
		}
		if !merged.flexDirection.IsSet() || merged.flexDirection.Value() != FlexColumn {
			t.Fatalf("FlexDirection=%v want FlexColumn", merged.flexDirection.Value())
		}
		if !merged.foreground.IsSet() {
			t.Fatal("Foreground should carry through from base")
		}
	})

	t.Run("unset override does not clear base", func(t *testing.T) {
		merged := base.Merge(Style{})
		if !merged.display.IsSet() || merged.display.Value() != DisplayFlex {
			t.Fatal("Display should be preserved when override has no Display")
		}
	})

	t.Run("merge is non-mutating", func(t *testing.T) {
		override := Style{gap: Some(Gap(99))}
		_ = base.Merge(override)
		if base.gap.Value() != Gap(2) {
			t.Fatal("base should not be mutated by Merge")
		}
	})

	t.Run("chained merge last-set wins", func(t *testing.T) {
		first := Style{gap: Some(Gap(10))}
		second := Style{gap: Some(Gap(20))}
		result := base.Merge(first).Merge(second)
		if result.gap.Value() != Gap(20) {
			t.Fatalf("Gap=%v want {20 20} (last set wins)", result.gap.Value())
		}
	})
}

// ---------------------------------------------------------------------------
// BenchmarkStyle_Merge
// ---------------------------------------------------------------------------

func BenchmarkStyle_Merge(b *testing.B) {
	base := Style{
		display:        Some(DisplayFlex),
		flexDirection:  Some(FlexColumn),
		justifyContent: Some(JustifyStart),
		alignItems:     Some(AlignStretch),
		gap:            Some(Gap(1)),
		width:          Some(Percent(100)),
		padding:        Some(EdgeAll(1)),
		foreground:     Some(color.Color(color.RGBA{R: 200, G: 200, B: 200, A: 255})),
		background:     Some(color.Color(color.RGBA{R: 30, G: 30, B: 30, A: 255})),
		bold:           Some(false),
	}
	override := Style{
		foreground: Some(color.Color(color.RGBA{R: 100, G: 200, B: 255, A: 255})),
		bold:       Some(true),
		gap:        Some(Gap(2)),
	}

	b.ReportAllocs()
	for b.Loop() {
		_ = base.Merge(override)
	}
}

// ---------------------------------------------------------------------------
// TestStyle_IndividualBordersMarginsPaddings
// ---------------------------------------------------------------------------

func TestStyle_IndividualBordersMarginsPaddings(t *testing.T) {
	t.Parallel()

	t.Run("borders", func(t *testing.T) {
		s := S().
			BorderTop(true).
			BorderRight(false).
			BorderBottom(true).
			BorderLeft(false)

		b := s.border.Value()
		if !b.Edges.Top || b.Edges.Right || !b.Edges.Bottom || b.Edges.Left {
			t.Errorf("got Edges %+v, want Top/Bottom set, Left/Right unset", b.Edges)
		}

		s2 := S().BorderHorizontal(true).BorderVertical(false)
		b2 := s2.border.Value()
		if b2.Edges.Top || !b2.Edges.Right || b2.Edges.Bottom || !b2.Edges.Left {
			t.Errorf("got Edges %+v, want Horizontal (Left/Right) set, Vertical (Top/Bottom) unset", b2.Edges)
		}
	})

	t.Run("margins", func(t *testing.T) {
		s := S().
			MarginTop(1).
			MarginRight(2).
			MarginBottom(3).
			MarginLeft(4)

		m := s.margin.Value()
		if m.Top != 1 || m.Right != 2 || m.Bottom != 3 || m.Left != 4 {
			t.Errorf("got margins %+v, want {1, 2, 3, 4}", m)
		}

		s2 := S().MarginHorizontal(5).MarginVertical(6)
		m2 := s2.margin.Value()
		if m2.Top != 6 || m2.Right != 5 || m2.Bottom != 6 || m2.Left != 5 {
			t.Errorf("got margins %+v, want horizontal 5, vertical 6", m2)
		}
	})

	t.Run("paddings", func(t *testing.T) {
		s := S().
			PaddingTop(1).
			PaddingRight(2).
			PaddingBottom(3).
			PaddingLeft(4)

		p := s.padding.Value()
		if p.Top != 1 || p.Right != 2 || p.Bottom != 3 || p.Left != 4 {
			t.Errorf("got paddings %+v, want {1, 2, 3, 4}", p)
		}

		s2 := S().PaddingHorizontal(5).PaddingVertical(6)
		p2 := s2.padding.Value()
		if p2.Top != 6 || p2.Right != 5 || p2.Bottom != 6 || p2.Left != 5 {
			t.Errorf("got paddings %+v, want horizontal 5, vertical 6", p2)
		}
	})

	t.Run("new border api", func(t *testing.T) {
		red := color.RGBA{255, 0, 0, 255}
		s := S().BorderTop(true, BorderDouble, red)
		b := s.BorderOpt().Value()

		if !b.Edges.Top {
			t.Error("Expected Top edge to be true")
		}
		if b.Styles.Top != BorderDouble {
			t.Errorf("Expected Top style to be BorderDouble, got %v", b.Styles.Top)
		}
		if b.Colors.Top != red {
			t.Errorf("Expected Top color to be red, got %v", b.Colors.Top)
		}

		s2 := S().Border(true, BorderThick)
		b2 := s2.BorderOpt().Value()
		if !b2.Edges.Top || !b2.Edges.Bottom || !b2.Edges.Left || !b2.Edges.Right {
			t.Error("Expected all edges to be true")
		}
		if b2.Styles.Top != BorderThick || b2.Styles.Bottom != BorderThick {
			t.Error("Expected BorderThick style on all edges")
		}

		s3 := S().Border(true)
		b3 := s3.BorderOpt().Value()
		if b3.Styles.Top != BorderSingle {
			t.Error("Expected default BorderSingle when calling Border(true)")
		}

		customGlyphs := BorderGlyphs{H: "#", V: "#", TL: "#", TR: "#", BL: "#", BR: "#"}
		s4 := S().BorderTop(true, customGlyphs)
		b4 := s4.BorderOpt().Value()
		if b4.Styles.Top != BorderCustom {
			t.Error("Expected BorderCustom when providing custom glyphs")
		}
		if b4.Glyphs.H != "#" {
			t.Error("Expected custom glyphs to be applied")
		}
	})
}
