package styler_test

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/internal/styler"
	"github.com/masterkeysrd/kite/style"
)

func TestResolver_ElementDefaults_AppliedBeforeInheritance(t *testing.T) {
	r := styler.NewResolver()

	node := &fakeNode{
		kind: dom.KindElement,
		elementDefaultStyle: style.Style{
			Display: style.Some(style.DisplayInline),
		},
	}
	ro := render.NewBox(node, nil)
	ro.MarkDirty(render.DirtyStyle)

	got := r.Resolve(ro, nil)

	if got.Display != style.DisplayInline {
		t.Errorf("Display = %v, want DisplayInline (from element default)", got.Display)
	}
}

func TestResolver_ElementDefaults_InheritanceOverridesElementDefault(t *testing.T) {
	r := styler.NewResolver()

	parentFG := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	parentComputed := &style.Computed{
		Foreground: parentFG,
	}

	defaultFG := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	node := &fakeNode{
		kind: dom.KindElement,
		elementDefaultStyle: style.Style{
			Foreground: style.Some[color.Color](defaultFG),
		},
	}
	ro := render.NewBox(node, nil)
	ro.MarkDirty(render.DirtyStyle)

	got := r.Resolve(ro, parentComputed)

	if got.Foreground != parentFG {
		t.Errorf("Foreground = %v, want inherited parent value %v (not element default %v)",
			got.Foreground, parentFG, defaultFG)
	}
}

func TestResolver_ElementDefaults_OverriddenByExplicitStyle(t *testing.T) {
	r := styler.NewResolver()

	node := &fakeNode{
		kind: dom.KindElement,
		elementDefaultStyle: style.Style{
			Display: style.Some(style.DisplayInline),
		},
		rawStyle: style.Style{
			Display: style.Some(style.DisplayFlex),
		},
	}
	ro := render.NewBox(node, nil)
	ro.MarkDirty(render.DirtyStyle)

	got := r.Resolve(ro, nil)

	if got.Display != style.DisplayFlex {
		t.Errorf("Display = %v, want DisplayFlex (author style overrides element default)", got.Display)
	}
}

func TestResolver_ElementDefaults_ZeroStyleIsNoop(t *testing.T) {
	r := styler.NewResolver()

	want := style.DefaultStyle()
	node := &fakeNode{
		kind:                dom.KindElement,
		elementDefaultStyle: style.Style{}, // all unset
	}
	ro := render.NewBox(node, nil)
	ro.MarkDirty(render.DirtyStyle)

	got := r.Resolve(ro, nil)

	if got.Display != want.Display {
		t.Errorf("Display = %v, want %v (zero element default must not change baseline)",
			got.Display, want.Display)
	}
}

func TestStyleSheet_Create_ValidatesEntries(t *testing.T) {
	t.Run("ValidSheet", func(t *testing.T) {
		_, err := style.NewSheet(map[string]style.Style{
			"button": {Display: style.Some(style.DisplayFlex)},
			"label":  {Display: style.Some(style.DisplayInline)},
		})
		if err != nil {
			t.Errorf("NewSheet with valid styles returned error: %v", err)
		}
	})

	t.Run("EmptyKeyRejected", func(t *testing.T) {
		_, err := style.NewSheet(map[string]style.Style{
			"": {Display: style.Some(style.DisplayBlock)},
		})
		if err == nil {
			t.Error("NewSheet with empty key must return an error")
		}
	})

	t.Run("NegativePaddingRejected", func(t *testing.T) {
		_, err := style.NewSheet(map[string]style.Style{
			"bad": {
				Padding: style.Some(style.Edges(-1)),
			},
		})
		if err == nil {
			t.Error("NewSheet with negative padding must return an error")
		}
	})
}
