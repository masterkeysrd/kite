package uv

import (
	"os"
	"strings"

	"github.com/charmbracelet/colorprofile"

	"github.com/masterkeysrd/kite/backend"
)

// probeCapabilities detects terminal capabilities from environment variables.
// It is called once at backend creation time. The result is stored in
// Backend.caps and treated as immutable for the lifetime of the backend.
//
// environ is typically os.Environ().
func probeCapabilities(environ []string) backend.Caps {
	env := envMap(environ)

	return backend.Caps{
		TrueColor:      probeTrueColor(environ),
		OSC8Hyperlinks: probeOSC8(env),
		Mouse:          backend.MouseSupportClick, // default: click only
		BracketedPaste: true,                      // supported by most modern terminals
		Sixel:          probeSixel(env),
		KittyGraphics:  probeKittyGraphics(env),
		Title:          probeTitle(env),
		Bell:           true, // BEL is universally supported
		Clipboard:      []backend.ClipboardKind{backend.ClipboardOSC52},
	}
}

// probeTrueColor checks whether the terminal reports 24-bit color support.
func probeTrueColor(environ []string) bool {
	p := colorprofile.Detect(os.Stdout, environ)
	return p >= colorprofile.TrueColor
}

// probeOSC8 checks for OSC 8 hyperlink support via the TERM_PROGRAM or
// VTE_VERSION environment variables (best-effort heuristic).
func probeOSC8(env map[string]string) bool {
	// Known OSC 8 supporting terminals.
	prog := strings.ToLower(env["TERM_PROGRAM"])
	switch prog {
	case "iterm.app", "wezterm", "ghostty", "kitty", "vscode":
		return true
	}
	// VTE-based terminals (GNOME Terminal, Tilix, etc.) v0.50+.
	if _, ok := env["VTE_VERSION"]; ok {
		return true
	}
	return false
}

// probeSixel checks for Sixel graphics support.
func probeSixel(env map[string]string) bool {
	term := env["TERM"]
	return strings.Contains(term, "sixel") ||
		env["TERM_PROGRAM"] == "mlterm"
}

// probeKittyGraphics checks for the Kitty terminal graphics protocol.
func probeKittyGraphics(env map[string]string) bool {
	return env["TERM"] == "xterm-kitty" ||
		env["TERM_PROGRAM"] == "ghostty"
}

// probeTitle checks whether the terminal accepts window title sequences.
// Most terminal emulators support OSC 0/2; only pipe/non-TTY outputs do not.
func probeTitle(env map[string]string) bool {
	// If TERM is "dumb" or empty, we assume no title support.
	term := env["TERM"]
	return term != "" && term != "dumb"
}

// envMap converts an []string of "KEY=VALUE" pairs into a map for O(1) lookup.
func envMap(environ []string) map[string]string {
	m := make(map[string]string, len(environ))
	for _, e := range environ {
		k, v, _ := strings.Cut(e, "=")
		m[k] = v
	}
	return m
}
