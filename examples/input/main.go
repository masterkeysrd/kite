// Package main demonstrates the InputElement component introduced in TSK-024.
//
// It renders a simple login form with two single-line text fields (username
// and password placeholder) inside a flex column card. The demo shows:
//
//   - Tab / Shift-Tab navigation between inputs
//   - Live preview: the card footer echoes the current username value
//   - The terminal hardware cursor tracks the active input's caret position
//   - Intrinsic overflow-clip keeps text within the field boundary even as you
//     type past the visible width
//   - Ctrl+C or Q exits the application
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

var (
	spacerStyle    = style.S().Height(style.Cells(1))
	footerRowStyle = style.S().Margin(1, 0, 0, 0)

	shortcutBadgeStyle = style.S().
				Padding(0, 1).
				Background(style.RGB(60, 65, 90)).
				Foreground(style.RGB(200, 210, 255))
)

// ── palette ──────────────────────────────────────────────────────────────────

var (
	colBG        = style.RGB(18, 18, 23)    // app background
	colCard      = style.RGB(28, 30, 40)    // card surface
	colBorder    = style.RGB(60, 65, 90)    // card border
	colInputBG   = style.RGB(22, 24, 35)    // input background
	colLabel     = style.RGB(160, 165, 200) // label text
	colTitle     = style.RGB(200, 210, 255) // title text
	colHint      = style.RGB(80, 85, 110)   // hint / footer text
	colAccent    = style.RGB(100, 200, 130) // echo value color
	colSeparator = style.RGB(45, 48, 65)    // horizontal rule
)

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	// ── logging ──────────────────────────────────────────────────────────────
	f, err := os.Create("kite.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))
	_ = logger // prevent unused variable error
	slog.SetDefault(logger)

	// ── backend ──────────────────────────────────────────────────────────────
	var b backend.Backend
	if os.Getenv("USE_MOCK_BACKEND") == "1" {
		b = mock.New(80, 24)
	} else {
		b, err = uv.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize backend: %v\n", err)
			os.Exit(1)
		}
	}

	// ── engine ───────────────────────────────────────────────────────────────
	eng := engine.New(b, engine.Options{})

	// ── inputs ───────────────────────────────────────────────────────────────
	// Build inputs against the engine document so they are already in the
	// correct document and get adopted cleanly on Mount.
	const fieldWidth = 30

	usernameInp := element.NewInput(eng.Document(), "").WithPlaceholder("Type username...")
	usernameInp.Style(
		style.S().
			Width(style.Cells(fieldWidth)).
			Padding(0, 1).
			Border(style.SingleBorder().Color(colBorder)).
			Background(colInputBG).
			Foreground(style.RGB(220, 225, 255)),
	)

	passwordInp := element.NewInput(eng.Document(), "").
		WithPlaceholder("Type password...").
		WithPlaceholderStyle(
			style.S().
				Foreground(style.RGB(120, 100, 150)).
				Italic(true),
		)
	passwordInp.Style(
		style.S().
			Width(style.Cells(fieldWidth)).
			Padding(0, 1).
			Border(style.SingleBorder().Color(colBorder)).
			Background(colInputBG).
			Foreground(style.RGB(220, 225, 255)),
	)

	// ── live echo text node ──────────────────────────────────────────────────
	// A Text node that displays the current username value in real time.
	echoText := element.Text("(empty)")

	// Update the echo whenever a keydown is fired on the username input.
	usernameInp.AddEventListener(event.EventKeyDown, func(e event.Event) {
		v := usernameInp.Value().(string)
		if v == "" {
			echoText.SetData("(empty)")
		} else {
			echoText.SetData(v)
		}
		eng.RequestFrame()
	})

	// ── UI tree ──────────────────────────────────────────────────────────────
	root := element.Box(

		// ── card ─────────────────────────────────────────────────────────────
		element.Box(

			// Title
			element.Box("  Sign In  ").Style(
				style.S().
					Width(style.Percent(100)).
					Margin(0, 0, 1, 0).
					Foreground(colTitle).
					Bold(true).
					TextAlign(style.TextAlignCenter),
			),

			// Separator
			element.Box("").Style(
				style.S().
					Width(style.Percent(100)).
					Height(style.Cells(1)).
					Margin(0, 0, 1, 0).
					Background(colSeparator),
			),

			// Username label + field
			element.Box("Username").Style(
				style.S().
					Margin(0, 0, 0, 0).
					Foreground(colLabel),
			),
			usernameInp,

			// Spacer
			element.Box("").Style(spacerStyle),

			// Password label + field
			element.Box("Password").Style(
				style.S().
					Margin(0, 0, 0, 0).
					Foreground(colLabel),
			),
			passwordInp,

			// Separator
			element.Box("").Style(
				style.S().
					Width(style.Percent(100)).
					Height(style.Cells(1)).
					Margin(1, 0, 0, 0).
					Background(colSeparator),
			),

			// Live echo row
			element.Box(
				element.Span("Username value: ").Style(style.S().Foreground(colHint)),
				element.Span(echoText).Style(
					style.S().
						Foreground(colAccent).
						Bold(true),
				),
			).Style(footerRowStyle),

			// Hints footer
			element.Box(
				element.Span("Tab").Style(shortcutBadgeStyle),
				element.Span(" next field  ").Style(style.S().Foreground(colHint)),
				element.Span("Shift+Tab").Style(shortcutBadgeStyle),
				element.Span(" prev field  ").Style(style.S().Foreground(colHint)),
				element.Span("Q").Style(shortcutBadgeStyle),
				element.Span(" quit").Style(style.S().Foreground(colHint)),
			).Style(footerRowStyle),
		).Style(
			style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Width(style.Cells(fieldWidth+6)).
				Padding(1, 2).
				Border(style.SingleBorder().Color(colBorder)).
				Background(colCard),
		),
	).Style(
		style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			JustifyContent(style.JustifyCenter).
			AlignItems(style.AlignCenter).
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Background(colBG),
	)

	eng.Mount(root)

	// Install devtools (Inspector + X-Ray)
	devtools.Install(eng, devtools.Options{})

	// ── global key bindings ───────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke, ok := e.(*event.KeyEvent)
		if !ok {
			return
		}
		if ke.MatchString("ctrl+c") || ke.MatchString("q") {
			cancel()
		}

		if ke.MatchString("ctrl+p") {
			eng.Dump(fmt.Sprintf("dump-%d.txt", os.Getpid()))
			b.DumpState()
		}
	})

	// ── run ───────────────────────────────────────────────────────────────────
	if err := eng.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
		os.Exit(1)
	}
}
