package style_test

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/style"
)

// ---------------------------------------------------------------------------
// TestResolver_ElementDefaults_AppliedBeforeInheritance
//
// The element-type default must sit below inheritance in the layering order:
//   root-baseline → element-default → parent-inheritance → own-style
//
// If a parent sets Foreground and the element has an element-default that also
// touches Foreground, the parent's inherited value must win.
// ---------------------------------------------------------------------------

func TestResolver_ElementDefaults_AppliedBeforeInheritance(t *testing.T) {
	t.Parallel()
	r := style.NewResolver()

	// Simulate an element type whose default is Display:Inline.
	// No parent, so we only verify the element default surfaces.
	node := &fakeNode{
		dirtyStyle: true,
		elementDefaultStyle: style.Style{
			Display: style.Some(style.DisplayInline),
		},
	}

	got := r.Resolve(node, nil)

	if got.Display != style.DisplayInline {
		t.Errorf("Display = %v, want DisplayInline (from element default)", got.Display)
	}
}

func TestResolver_ElementDefaults_InheritanceOverridesElementDefault(t *testing.T) {
	t.Parallel()
	// An element-default sets Foreground to red. The parent's Foreground is
	// blue. Because inheritance (layer 3) sits above element-defaults (layer 2),
	// the parent's value must win.
	r := style.NewResolver()

	parentFG := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	parentComputed := &style.Computed{
		Foreground: parentFG,
		Background: color.Transparent,
		TextWrap:   style.TextWrapWord,
	}

	defaultFG := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	node := &fakeNode{
		dirtyStyle: true,
		elementDefaultStyle: style.Style{
			Foreground: style.Some[color.Color](defaultFG),
		},
	}

	got := r.Resolve(node, parentComputed)

	if got.Foreground != parentFG {
		t.Errorf("Foreground = %v, want inherited parent value %v (not element default %v)",
			got.Foreground, parentFG, defaultFG)
	}
}

func TestResolver_ElementDefaults_OverriddenByExplicitStyle(t *testing.T) {
	t.Parallel()
	// The author's own style (layer 4) wins over element defaults (layer 2).
	r := style.NewResolver()

	node := &fakeNode{
		dirtyStyle: true,
		elementDefaultStyle: style.Style{
			Display: style.Some(style.DisplayInline),
		},
		rawStyle: style.Style{
			Display: style.Some(style.DisplayFlex),
		},
	}

	got := r.Resolve(node, nil)

	if got.Display != style.DisplayFlex {
		t.Errorf("Display = %v, want DisplayFlex (author style overrides element default)", got.Display)
	}
}

func TestResolver_ElementDefaults_ZeroStyleIsNoop(t *testing.T) {
	t.Parallel()
	// A zero ElementDefaultStyle must not change any property from the baseline.
	r := style.NewResolver()

	want := style.DefaultStyle()
	node := &fakeNode{
		dirtyStyle:          true,
		elementDefaultStyle: style.Style{}, // all unset
	}

	got := r.Resolve(node, nil)

	if got.Display != want.Display {
		t.Errorf("Display = %v, want %v (zero element default must not change baseline)",
			got.Display, want.Display)
	}
}

// ---------------------------------------------------------------------------
// TestStyleSheet_Create_ValidatesEntries
// ---------------------------------------------------------------------------

func TestStyleSheet_Create_ValidatesEntries(t *testing.T) {
	t.Parallel()

	t.Run("ValidSheet", func(t *testing.T) {
		t.Parallel()
		_, err := style.NewSheet(map[string]style.Style{
			"button": {Display: style.Some(style.DisplayFlex)},
			"label":  {Display: style.Some(style.DisplayInline)},
		})
		if err != nil {
			t.Errorf("NewSheet with valid styles returned error: %v", err)
		}
	})

	t.Run("EmptyKeyRejected", func(t *testing.T) {
		t.Parallel()
		_, err := style.NewSheet(map[string]style.Style{
			"": {Display: style.Some(style.DisplayBlock)},
		})
		if err == nil {
			t.Error("NewSheet with empty key must return an error")
		}
	})

	t.Run("NegativePaddingRejected", func(t *testing.T) {
		t.Parallel()
		_, err := style.NewSheet(map[string]style.Style{
			"bad": {
				Padding: style.Some(style.Edges(-1)),
			},
		})
		if err == nil {
			t.Error("NewSheet with negative padding must return an error")
		}
	})

	t.Run("NegativeMarginRejected", func(t *testing.T) {
		t.Parallel()
		_, err := style.NewSheet(map[string]style.Style{
			"bad": {
				Margin: style.Some(style.Edges(-1)),
			},
		})
		if err == nil {
			t.Error("NewSheet with negative margin must return an error")
		}
	})

	t.Run("EmptySheetIsValid", func(t *testing.T) {
		t.Parallel()
		_, err := style.NewSheet(map[string]style.Style{})
		if err != nil {
			t.Errorf("NewSheet with empty map must not error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TestStyleSheet_Get_ReturnsImmutable
// ---------------------------------------------------------------------------

func TestStyleSheet_Get_ReturnsImmutable(t *testing.T) {
	t.Parallel()
	sheet, _ := style.NewSheet(map[string]style.Style{
		"btn": {Display: style.Some(style.DisplayFlex)},
	})

	got1 := sheet.Get("btn")
	got2 := sheet.Get("btn")

	// Returned values should be identical copies.
	if got1.Display != got2.Display {
		t.Error("consecutive Get calls must return equal values")
	}

	// Mutating the returned value must not affect subsequent lookups.
	got1.Display = style.Some(style.DisplayBlock)
	got3 := sheet.Get("btn")
	if got3.Display.Value() != style.DisplayFlex {
		t.Error("mutating a returned Style must not affect the sheet's stored value")
	}
}

// ---------------------------------------------------------------------------
// TestStyleSheet_Get_UnknownKey_ReturnsZero
// ---------------------------------------------------------------------------

func TestStyleSheet_Get_UnknownKey_ReturnsZero(t *testing.T) {
	t.Parallel()
	sheet, _ := style.NewSheet(map[string]style.Style{
		"known": {Display: style.Some(style.DisplayBlock)},
	})

	got := sheet.Get("no-such-key")

	if got.Display.IsSet() {
		t.Error("Get for unknown key must return a zero Style (all fields unset)")
	}
}

// ---------------------------------------------------------------------------
// TestStyleSheet_Has
// ---------------------------------------------------------------------------

func TestStyleSheet_Has(t *testing.T) {
	t.Parallel()
	sheet, _ := style.NewSheet(map[string]style.Style{
		"present": {},
	})

	if !sheet.Has("present") {
		t.Error("Has must return true for a registered key")
	}
	if sheet.Has("absent") {
		t.Error("Has must return false for an unregistered key")
	}
}

// ---------------------------------------------------------------------------
// TestStyleSheet_Len
// ---------------------------------------------------------------------------

func TestStyleSheet_Len(t *testing.T) {
	t.Parallel()
	sheet, _ := style.NewSheet(map[string]style.Style{
		"a": {},
		"b": {},
		"c": {},
	})
	if sheet.Len() != 3 {
		t.Errorf("Len = %d, want 3", sheet.Len())
	}
}

// ---------------------------------------------------------------------------
// TestStyleSheet_InputMapMutation_DoesNotAffectSheet
// ---------------------------------------------------------------------------

func TestStyleSheet_InputMapMutation_DoesNotAffectSheet(t *testing.T) {
	t.Parallel()
	input := map[string]style.Style{
		"box": {Display: style.Some(style.DisplayFlex)},
	}
	sheet, _ := style.NewSheet(input)

	// Mutate the original map after creation.
	input["box"] = style.Style{Display: style.Some(style.DisplayBlock)}
	delete(input, "box")

	got := sheet.Get("box")
	if !got.Display.IsSet() || got.Display.Value() != style.DisplayFlex {
		t.Error("mutating the input map after NewSheet must not affect the sheet")
	}
}
