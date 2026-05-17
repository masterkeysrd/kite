package style

import "image/color"

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
