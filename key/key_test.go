package key_test

import (
	"testing"

	"github.com/masterkeysrd/kite/key"
)

func TestKey_MatchString_NonPrintableShiftedVsPrintable(t *testing.T) {
	nonPrintableShiftPatterns := []string{
		"shift+tab",
		"shift+enter",
		"shift+up",
		"shift+down",
		"shift+left",
		"shift+right",
	}

	printableKeys := []struct {
		name string
		key  key.Key
	}{
		{"Space", key.Key{Code: ' ', Text: " "}},
		{"Digit", key.Key{Code: '1', Text: "1"}},
		{"Symbol", key.Key{Code: '/', Text: "/"}},
		{"Lowercase letter", key.Key{Code: 'a', Text: "a"}},
		{"Uppercase shifted letter", key.Key{Code: 'a', Text: "A", Mod: key.ModShift}},
		{"Uppercase letter without mod", key.Key{Code: 'A', Text: "A"}},
		{"Shifted digit", key.Key{Code: '1', Text: "!", Mod: key.ModShift}},
		{"Shifted symbol", key.Key{Code: '/', Text: "?", Mod: key.ModShift}},
	}

	for _, pattern := range nonPrintableShiftPatterns {
		for _, pk := range printableKeys {
			t.Run(pattern+"_vs_"+pk.name, func(t *testing.T) {
				if pk.key.MatchString(pattern) {
					t.Errorf("key %v (%s) unexpectedly matched pattern %q", pk.key, pk.name, pattern)
				}
			})
		}
	}
}

func TestKey_MatchString_ShiftNonPrintableKeys(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		key     key.Key
		want    bool
	}{
		{
			name:    "shift+tab matches Shift+KeyTab",
			pattern: "shift+tab",
			key:     key.Key{Code: key.KeyTab, Mod: key.ModShift},
			want:    true,
		},
		{
			name:    "shift+tab does not match unshifted KeyTab",
			pattern: "shift+tab",
			key:     key.Key{Code: key.KeyTab},
			want:    false,
		},
		{
			name:    "shift+tab does not match Ctrl+KeyTab",
			pattern: "shift+tab",
			key:     key.Key{Code: key.KeyTab, Mod: key.ModCtrl},
			want:    false,
		},
		{
			name:    "shift+enter matches Shift+KeyEnter",
			pattern: "shift+enter",
			key:     key.Key{Code: key.KeyEnter, Mod: key.ModShift},
			want:    true,
		},
		{
			name:    "shift+enter does not match unshifted KeyEnter",
			pattern: "shift+enter",
			key:     key.Key{Code: key.KeyEnter},
			want:    false,
		},
		{
			name:    "shift+up matches Shift+KeyUp",
			pattern: "shift+up",
			key:     key.Key{Code: key.KeyUp, Mod: key.ModShift},
			want:    true,
		},
		{
			name:    "shift+up does not match unshifted KeyUp",
			pattern: "shift+up",
			key:     key.Key{Code: key.KeyUp},
			want:    false,
		},
		{
			name:    "shift+down matches Shift+KeyDown",
			pattern: "shift+down",
			key:     key.Key{Code: key.KeyDown, Mod: key.ModShift},
			want:    true,
		},
		{
			name:    "shift+left matches Shift+KeyLeft",
			pattern: "shift+left",
			key:     key.Key{Code: key.KeyLeft, Mod: key.ModShift},
			want:    true,
		},
		{
			name:    "shift+right matches Shift+KeyRight",
			pattern: "shift+right",
			key:     key.Key{Code: key.KeyRight, Mod: key.ModShift},
			want:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.key.MatchString(tc.pattern)
			if got != tc.want {
				t.Errorf("Key(%+v).MatchString(%q) = %v; want %v", tc.key, tc.pattern, got, tc.want)
			}
		})
	}
}

func TestKey_MatchString_PrintableShiftedLetters(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		key     key.Key
		want    bool
	}{
		{
			name:    "shift+a matches letter 'a' with ModShift and Text 'A'",
			pattern: "shift+a",
			key:     key.Key{Code: 'a', Text: "A", Mod: key.ModShift},
			want:    true,
		},
		{
			name:    "shift+a matches letter 'A' with Text 'A'",
			pattern: "shift+a",
			key:     key.Key{Code: 'A', Text: "A"},
			want:    true,
		},
		{
			name:    "shift+a does not match unshifted letter 'a'",
			pattern: "shift+a",
			key:     key.Key{Code: 'a', Text: "a"},
			want:    false,
		},
		{
			name:    "shift+a does not match shifted letter 'b'",
			pattern: "shift+a",
			key:     key.Key{Code: 'b', Text: "B", Mod: key.ModShift},
			want:    false,
		},
		{
			name:    "shift+z matches shifted 'z'",
			pattern: "shift+z",
			key:     key.Key{Code: 'z', Text: "Z", Mod: key.ModShift},
			want:    true,
		},
		{
			name:    "shift+z matches 'Z'",
			pattern: "shift+z",
			key:     key.Key{Code: 'Z', Text: "Z"},
			want:    true,
		},
		{
			name:    "shift+z does not match unshifted 'z'",
			pattern: "shift+z",
			key:     key.Key{Code: 'z', Text: "z"},
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.key.MatchString(tc.pattern)
			if got != tc.want {
				t.Errorf("Key(%+v).MatchString(%q) = %v; want %v", tc.key, tc.pattern, got, tc.want)
			}
		})
	}
}

func TestKey_MatchString_UnshiftedPrintableAndModifiers(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		key     key.Key
		want    bool
	}{
		{
			name:    "pattern 'a' matches unshifted 'a'",
			pattern: "a",
			key:     key.Key{Code: 'a', Text: "a"},
			want:    true,
		},
		{
			name:    "pattern 'a' does not match 'b'",
			pattern: "a",
			key:     key.Key{Code: 'b', Text: "b"},
			want:    false,
		},
		{
			name:    "pattern 'space' matches KeySpace code",
			pattern: "space",
			key:     key.Key{Code: key.KeySpace},
			want:    true,
		},
		{
			name:    "pattern 'space' matches space rune with text",
			pattern: "space",
			key:     key.Key{Code: ' ', Text: " "},
			want:    true,
		},
		{
			name:    "pattern '1' matches digit '1'",
			pattern: "1",
			key:     key.Key{Code: '1', Text: "1"},
			want:    true,
		},
		{
			name:    "pattern '1' does not match digit '2'",
			pattern: "1",
			key:     key.Key{Code: '2', Text: "2"},
			want:    false,
		},
		{
			name:    "ctrl+c matches Code 'c' with ModCtrl",
			pattern: "ctrl+c",
			key:     key.Key{Code: 'c', Mod: key.ModCtrl},
			want:    true,
		},
		{
			name:    "ctrl+c matches ASCII 0x03 control code",
			pattern: "ctrl+c",
			key:     key.Key{Code: 0x03},
			want:    true,
		},
		{
			name:    "ctrl+c does not match unshifted 'c'",
			pattern: "ctrl+c",
			key:     key.Key{Code: 'c'},
			want:    false,
		},
		{
			name:    "ctrl+d matches Code 'd' with ModCtrl",
			pattern: "ctrl+d",
			key:     key.Key{Code: 'd', Mod: key.ModCtrl},
			want:    true,
		},
		{
			name:    "ctrl+d matches ASCII 0x04 control code",
			pattern: "ctrl+d",
			key:     key.Key{Code: 0x04},
			want:    true,
		},
		{
			name:    "ctrl+i matches Code 'i' with ModCtrl",
			pattern: "ctrl+i",
			key:     key.Key{Code: 'i', Mod: key.ModCtrl},
			want:    true,
		},
		{
			name:    "ctrl+i matches ASCII 0x09 (Tab) control code",
			pattern: "ctrl+i",
			key:     key.Key{Code: 0x09},
			want:    true,
		},
		{
			name:    "alt+enter matches KeyEnter with ModAlt",
			pattern: "alt+enter",
			key:     key.Key{Code: key.KeyEnter, Mod: key.ModAlt},
			want:    true,
		},
		{
			name:    "alt+enter does not match KeyEnter with ModCtrl",
			pattern: "alt+enter",
			key:     key.Key{Code: key.KeyEnter, Mod: key.ModCtrl},
			want:    false,
		},
		{
			name:    "super+k matches 'k' with ModSuper",
			pattern: "super+k",
			key:     key.Key{Code: 'k', Mod: key.ModSuper},
			want:    true,
		},
		{
			name:    "cmd+k matches 'k' with ModSuper",
			pattern: "cmd+k",
			key:     key.Key{Code: 'k', Mod: key.ModSuper},
			want:    true,
		},
		{
			name:    "super+k does not match unshifted 'k'",
			pattern: "super+k",
			key:     key.Key{Code: 'k'},
			want:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.key.MatchString(tc.pattern)
			if got != tc.want {
				t.Errorf("Key(%+v).MatchString(%q) = %v; want %v", tc.key, tc.pattern, got, tc.want)
			}
		})
	}
}

func TestKey_MatchString_Variadic(t *testing.T) {
	patterns := []string{"q", "ctrl+c", "esc"}

	k1 := key.Key{Code: 'q', Text: "q"}
	if !k1.MatchString(patterns...) {
		t.Errorf("expected 'q' to match %v", patterns)
	}

	k2 := key.Key{Code: 'c', Mod: key.ModCtrl}
	if !k2.MatchString(patterns...) {
		t.Errorf("expected 'ctrl+c' to match %v", patterns)
	}

	k3 := key.Key{Code: key.KeyEscape}
	if !k3.MatchString(patterns...) {
		t.Errorf("expected KeyEscape to match %v", patterns)
	}

	k4 := key.Key{Code: 'x', Text: "x"}
	if k4.MatchString(patterns...) {
		t.Errorf("expected 'x' not to match %v", patterns)
	}
}
