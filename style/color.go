package style

import (
	"fmt"
	"image/color"
	"strings"
)

// TerminalDefault is the sentinel color meaning "use the terminal's default
// foreground or background." Backends recognise this value and emit the
// terminal's default-color escape sequence instead of an explicit RGB color.
//
// Callers should compare against this value using == (it is a package-level
// variable, not a type), or via a type assertion to the unexported
// terminalDefaultColor type.
var TerminalDefault color.Color = terminalDefaultColor{}

// terminalDefaultColor implements color.Color as a sentinel.
type terminalDefaultColor struct{}

// RGBA implements color.Color. The backend treats this value specially and
// never uses the returned components directly.
func (terminalDefaultColor) RGBA() (r, g, b, a uint32) { return 0, 0, 0, 0 }

// IsTerminalDefault reports whether c is the [TerminalDefault] sentinel.
func IsTerminalDefault(c color.Color) bool {
	if c == nil {
		return false
	}
	_, ok := c.(terminalDefaultColor)
	return ok
}

// RGB returns an opaque color with the given 8-bit red, green, and blue channels.
func RGB(r, g, b uint8) color.Color {
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// RGBA returns a color with the given 8-bit red, green, blue, and alpha channels.
func RGBA(r, g, b, a uint8) color.Color {
	return color.RGBA{R: r, G: g, B: b, A: a}
}

// ParseHex parses a CSS-style hex color string in #RGB, RGB, #RRGGBB, RRGGBB,
// #RRGGBBAA, or RRGGBBAA format (case-insensitive).
func ParseHex(hex string) (color.Color, error) {
	s := strings.TrimPrefix(hex, "#")
	switch len(s) {
	case 3:
		r, ok1 := parseHexNibble(s[0])
		g, ok2 := parseHexNibble(s[1])
		b, ok3 := parseHexNibble(s[2])
		if !ok1 || !ok2 || !ok3 {
			return nil, fmt.Errorf("style.ParseHex: invalid hex color %q", hex)
		}
		return color.RGBA{R: r * 17, G: g * 17, B: b * 17, A: 255}, nil
	case 6:
		r, ok1 := parseHexByte(s[0], s[1])
		g, ok2 := parseHexByte(s[2], s[3])
		b, ok3 := parseHexByte(s[4], s[5])
		if !ok1 || !ok2 || !ok3 {
			return nil, fmt.Errorf("style.ParseHex: invalid hex color %q", hex)
		}
		return color.RGBA{R: r, G: g, B: b, A: 255}, nil
	case 8:
		r, ok1 := parseHexByte(s[0], s[1])
		g, ok2 := parseHexByte(s[2], s[3])
		b, ok3 := parseHexByte(s[4], s[5])
		a, ok4 := parseHexByte(s[6], s[7])
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return nil, fmt.Errorf("style.ParseHex: invalid hex color %q", hex)
		}
		return color.RGBA{R: r, G: g, B: b, A: a}, nil
	default:
		return nil, fmt.Errorf("style.ParseHex: invalid hex color %q", hex)
	}
}

// Hex parses a CSS-style hex color string via [ParseHex]. On invalid input, it
// returns opaque black (color.RGBA{A: 255}) without panicking.
func Hex(hex string) color.Color {
	c, err := ParseHex(hex)
	if err != nil {
		return color.RGBA{A: 255}
	}
	return c
}

// HexInt converts a 24-bit RGB integer (0xRRGGBB) into an opaque color.Color.
func HexInt(v uint32) color.Color {
	return color.RGBA{
		R: uint8((v >> 16) & 0xff),
		G: uint8((v >> 8) & 0xff),
		B: uint8(v & 0xff),
		A: 255,
	}
}

func parseHexNibble(b byte) (uint8, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	default:
		return 0, false
	}
}

func parseHexByte(hi, lo byte) (uint8, bool) {
	h, ok1 := parseHexNibble(hi)
	l, ok2 := parseHexNibble(lo)
	if !ok1 || !ok2 {
		return 0, false
	}
	return (h << 4) | l, true
}
