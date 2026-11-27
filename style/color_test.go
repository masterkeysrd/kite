package style

import (
	"image/color"
	"testing"
)

func TestColorConstructors(t *testing.T) {
	t.Parallel()

	t.Run("RGB", func(t *testing.T) {
		got := RGB(12, 34, 56)
		want := color.RGBA{R: 12, G: 34, B: 56, A: 255}
		if got != want {
			t.Fatalf("RGB() = %#v, want %#v", got, want)
		}
	})

	t.Run("RGBA", func(t *testing.T) {
		got := RGBA(12, 34, 56, 128)
		want := color.RGBA{R: 12, G: 34, B: 56, A: 128}
		if got != want {
			t.Fatalf("RGBA() = %#v, want %#v", got, want)
		}
	})

	t.Run("HexInt", func(t *testing.T) {
		got := HexInt(0x1e293b)
		want := color.RGBA{R: 0x1e, G: 0x29, B: 0x3b, A: 255}
		if got != want {
			t.Fatalf("HexInt() = %#v, want %#v", got, want)
		}
	})
}

func TestParseHexAndHex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    color.Color
		wantErr bool
	}{
		{name: "3-digit with hash", input: "#f80", want: color.RGBA{R: 0xff, G: 0x88, B: 0x00, A: 255}},
		{name: "3-digit without hash", input: "abc", want: color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 255}},
		{name: "3-digit uppercase", input: "#F0A", want: color.RGBA{R: 0xff, G: 0x00, B: 0xaa, A: 255}},
		{name: "6-digit with hash", input: "#1e293b", want: color.RGBA{R: 0x1e, G: 0x29, B: 0x3b, A: 255}},
		{name: "6-digit without hash", input: "38BDF8", want: color.RGBA{R: 0x38, G: 0xbd, B: 0xf8, A: 255}},
		{name: "8-digit with hash", input: "#11223380", want: color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0x80}},
		{name: "8-digit without hash", input: "ff00ff00", want: color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0x00}},
		{name: "empty", input: "", wantErr: true},
		{name: "only hash", input: "#", wantErr: true},
		{name: "double hash", input: "##fff", wantErr: true},
		{name: "wrong length 4", input: "#1234", wantErr: true},
		{name: "wrong length 5", input: "#12345", wantErr: true},
		{name: "invalid hex char", input: "#12g456", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseHex(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseHex(%q) expected error, got %#v", tc.input, got)
				}
				if hexGot := Hex(tc.input); hexGot != (color.RGBA{A: 255}) {
					t.Fatalf("Hex(%q) on invalid input = %#v, want opaque black", tc.input, hexGot)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseHex(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("ParseHex(%q) = %#v, want %#v", tc.input, got, tc.want)
			}
			if hexGot := Hex(tc.input); hexGot != tc.want {
				t.Fatalf("Hex(%q) = %#v, want %#v", tc.input, hexGot, tc.want)
			}
		})
	}
}

func BenchmarkStyle_Hex(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Hex("#38bdf8")
	}
}
