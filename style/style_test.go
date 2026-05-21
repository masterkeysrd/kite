package style_test

import (
	"encoding/json"
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// TestOptional_Merge
// ---------------------------------------------------------------------------

func TestOptional_Merge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     style.Optional[int]
		override style.Optional[int]
		wantVal  int
		wantSet  bool
	}{
		{
			name:    "both unset — result unset",
			wantSet: false,
		},
		{
			name:    "base set, override unset — base wins",
			base:    style.Some(42),
			wantVal: 42,
			wantSet: true,
		},
		{
			name:     "base unset, override set — override wins",
			override: style.Some(7),
			wantVal:  7,
			wantSet:  true,
		},
		{
			name:     "both set — override wins",
			base:     style.Some(1),
			override: style.Some(2),
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

	var o style.Optional[string]
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
		var o style.Optional[int]
		data, err := json.Marshal(o)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "null" {
			t.Fatalf("got %q want null", string(data))
		}
	})

	t.Run("marshal set", func(t *testing.T) {
		o := style.Some(42)
		data, err := json.Marshal(o)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "42" {
			t.Fatalf("got %q want 42", string(data))
		}
	})

	t.Run("unmarshal null", func(t *testing.T) {
		var o style.Optional[int]
		o.Set(123)
		if err := json.Unmarshal([]byte("null"), &o); err != nil {
			t.Fatal(err)
		}
		if o.IsSet() {
			t.Fatal("should be unset after unmarshaling null")
		}
	})

	t.Run("unmarshal value", func(t *testing.T) {
		var o style.Optional[int]
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
		dim  style.Dimension
		want style.DimensionKind
	}{
		{"Cells", style.Cells(10), style.KindCells},
		{"Percent", style.Percent(50), style.KindPercent},
		{"Fr", style.Fr(2), style.KindFr},
		{"Auto", style.Auto, style.KindAuto},
		{"Content", style.Content, style.KindContent},
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
		ev := style.Edges(5)
		if ev.Top != 5 || ev.Right != 5 || ev.Bottom != 5 || ev.Left != 5 {
			t.Fatalf("got %+v want all 5", ev)
		}
	})

	t.Run("2-arg vertical horizontal", func(t *testing.T) {
		ev := style.Edges(2, 4)
		if ev.Top != 2 || ev.Bottom != 2 {
			t.Fatalf("top/bottom should be 2, got top=%d bottom=%d", ev.Top, ev.Bottom)
		}
		if ev.Right != 4 || ev.Left != 4 {
			t.Fatalf("right/left should be 4, got right=%d left=%d", ev.Right, ev.Left)
		}
	})

	t.Run("4-arg TRBL", func(t *testing.T) {
		ev := style.Edges(1, 2, 3, 4)
		if ev.Top != 1 || ev.Right != 2 || ev.Bottom != 3 || ev.Left != 4 {
			t.Fatalf("got %+v want T=1 R=2 B=3 L=4", ev)
		}
	})

	t.Run("EdgeAll shorthand", func(t *testing.T) {
		ev := style.EdgeAll(7)
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

	base := style.BorderGlyphs{
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

	base := style.Style{
		Display:       style.Some(style.DisplayFlex),
		FlexDirection: style.Some(style.FlexColumn),
		Gap:           style.Some(style.Gap(2)),
		Foreground:    style.Some(color.Color(color.RGBA{R: 200, G: 200, B: 200, A: 255})),
		Bold:          style.Some(false),
	}

	t.Run("override replaces only set fields", func(t *testing.T) {
		override := style.Style{
			Gap:  style.Some(style.Gap(4)),
			Bold: style.Some(true),
		}
		merged := base.Merge(override)

		// Fields set in override must come from override.
		if v, _ := merged.Gap.Value(), merged.Gap.IsSet(); v != style.Gap(4) {
			t.Fatalf("Gap=%v want {4 4}", merged.Gap.Value())
		}
		if v := merged.Bold.Value(); !v {
			t.Fatal("Bold should be true after merge")
		}
		// Fields not set in override must come from base.
		if v := merged.Display.Value(); v != style.DisplayFlex {
			t.Fatalf("Display=%v want DisplayFlex", v)
		}
		if !merged.FlexDirection.IsSet() || merged.FlexDirection.Value() != style.FlexColumn {
			t.Fatalf("FlexDirection=%v want FlexColumn", merged.FlexDirection.Value())
		}
		if !merged.Foreground.IsSet() {
			t.Fatal("Foreground should carry through from base")
		}
	})

	t.Run("unset override does not clear base", func(t *testing.T) {
		merged := base.Merge(style.Style{})
		if !merged.Display.IsSet() || merged.Display.Value() != style.DisplayFlex {
			t.Fatal("Display should be preserved when override has no Display")
		}
	})

	t.Run("merge is non-mutating", func(t *testing.T) {
		override := style.Style{Gap: style.Some(style.Gap(99))}
		_ = base.Merge(override)
		if base.Gap.Value() != style.Gap(2) {
			t.Fatal("base should not be mutated by Merge")
		}
	})

	t.Run("chained merge last-set wins", func(t *testing.T) {
		first := style.Style{Gap: style.Some(style.Gap(10))}
		second := style.Style{Gap: style.Some(style.Gap(20))}
		result := base.Merge(first).Merge(second)
		if result.Gap.Value() != style.Gap(20) {
			t.Fatalf("Gap=%v want {20 20} (last set wins)", result.Gap.Value())
		}
	})
}

// ---------------------------------------------------------------------------
// BenchmarkStyle_Merge
// ---------------------------------------------------------------------------

func BenchmarkStyle_Merge(b *testing.B) {
	base := style.Style{
		Display:        style.Some(style.DisplayFlex),
		FlexDirection:  style.Some(style.FlexColumn),
		JustifyContent: style.Some(style.JustifyStart),
		AlignItems:     style.Some(style.AlignStretch),
		Gap:            style.Some(style.Gap(1)),
		Width:          style.Some(style.Percent(100)),
		Padding:        style.Some(style.EdgeAll(1)),
		Foreground:     style.Some(color.Color(color.RGBA{R: 200, G: 200, B: 200, A: 255})),
		Background:     style.Some(color.Color(color.RGBA{R: 30, G: 30, B: 30, A: 255})),
		Bold:           style.Some(false),
	}
	override := style.Style{
		Foreground: style.Some(color.Color(color.RGBA{R: 100, G: 200, B: 255, A: 255})),
		Bold:       style.Some(true),
		Gap:        style.Some(style.Gap(2)),
	}

	b.ReportAllocs()
	for b.Loop() {
		_ = base.Merge(override)
	}
}
