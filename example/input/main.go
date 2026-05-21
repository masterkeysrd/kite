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
	"image/color"
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

// ── palette ──────────────────────────────────────────────────────────────────

var (
	colBG        = color.RGBA{R: 18, G: 18, B: 23, A: 255}    // app background
	colCard      = color.RGBA{R: 28, G: 30, B: 40, A: 255}    // card surface
	colBorder    = color.RGBA{R: 60, G: 65, B: 90, A: 255}    // card border
	colInputBG   = color.RGBA{R: 22, G: 24, B: 35, A: 255}    // input background
	colLabel     = color.RGBA{R: 160, G: 165, B: 200, A: 255} // label text
	colTitle     = color.RGBA{R: 200, G: 210, B: 255, A: 255} // title text
	colHint      = color.RGBA{R: 80, G: 85, B: 110, A: 255}   // hint / footer text
	colAccent    = color.RGBA{R: 100, G: 200, B: 130, A: 255} // echo value color
	colSeparator = color.RGBA{R: 45, G: 48, B: 65, A: 255}    // horizontal rule
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
	eng := engine.New(b, engine.Options{Logger: logger})

	// ── inputs ───────────────────────────────────────────────────────────────
	// Build inputs against the engine document so they are already in the
	// correct document and get adopted cleanly on Mount.
	const fieldWidth = 30

	usernameInp := element.NewInput(eng.Document(), "")
	usernameInp.Style(style.Style{
		Width:      style.Some(style.Cells(fieldWidth)),
		Background: style.Some[color.Color](colInputBG),
		Foreground: style.Some[color.Color](color.RGBA{R: 220, G: 225, B: 255, A: 255}),
		Border:     style.SingleBorder().Color(colBorder).Some(),
		Padding:    style.Some(style.Edges(0, 1)),
	})

	passwordInp := element.NewInput(eng.Document(), "")
	passwordInp.Style(style.Style{
		Width:      style.Some(style.Cells(fieldWidth)),
		Background: style.Some[color.Color](colInputBG),
		Foreground: style.Some[color.Color](color.RGBA{R: 220, G: 225, B: 255, A: 255}),
		Border:     style.SingleBorder().Color(colBorder).Some(),
		Padding:    style.Some(style.Edges(0, 1)),
	})

	// ── live echo text node ──────────────────────────────────────────────────
	// A Text node that displays the current username value in real time.
	echoText := element.Text("(empty)")

	// Update the echo whenever a keydown is fired on the username input.
	usernameInp.AddEventListener(event.EventKeyDown, func(e event.Event) {
		v := usernameInp.Value()
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
			element.Box("  Sign In  ").Style(style.Style{
				Foreground: style.Some[color.Color](colTitle),
				Bold:       style.Some(true),
				TextAlign:  style.Some(style.TextAlignCenter),
				Width:      style.Some(style.Percent(100)),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
			}),

			// Separator
			element.Box("").Style(style.Style{
				Width:      style.Some(style.Percent(100)),
				Height:     style.Some(style.Cells(1)),
				Background: style.Some[color.Color](colSeparator),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
			}),

			// Username label + field
			element.Box("Username").Style(style.Style{
				Foreground: style.Some[color.Color](colLabel),
				Margin:     style.Some(style.Edges(0, 0, 0, 0)),
			}),
			usernameInp,

			// Spacer
			element.Box("").Style(style.Style{Height: style.Some(style.Cells(1))}),

			// Password label + field
			element.Box("Password").Style(style.Style{
				Foreground: style.Some[color.Color](colLabel),
				Margin:     style.Some(style.Edges(0, 0, 0, 0)),
			}),
			passwordInp,

			// Separator
			element.Box("").Style(style.Style{
				Width:      style.Some(style.Percent(100)),
				Height:     style.Some(style.Cells(1)),
				Background: style.Some[color.Color](colSeparator),
				Margin:     style.Some(style.Edges(1, 0, 0, 0)),
			}),

			// Live echo row
			element.Box(
				element.Span("Username value: ").Style(style.Style{
					Foreground: style.Some[color.Color](colHint),
				}),
				element.Span(echoText).Style(style.Style{
					Foreground: style.Some[color.Color](colAccent),
					Bold:       style.Some(true),
				}),
			).Style(style.Style{
				Margin: style.Some(style.Edges(1, 0, 0, 0)),
			}),

			// Hints footer
			element.Box(
				element.Span("Tab").Style(style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 60, G: 65, B: 90, A: 255}),
					Foreground: style.Some[color.Color](color.RGBA{R: 200, G: 210, B: 255, A: 255}),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" next field  ").Style(style.Style{
					Foreground: style.Some[color.Color](colHint),
				}),
				element.Span("Shift+Tab").Style(style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 60, G: 65, B: 90, A: 255}),
					Foreground: style.Some[color.Color](color.RGBA{R: 200, G: 210, B: 255, A: 255}),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" prev field  ").Style(style.Style{
					Foreground: style.Some[color.Color](colHint),
				}),
				element.Span("Q").Style(style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 60, G: 65, B: 90, A: 255}),
					Foreground: style.Some[color.Color](color.RGBA{R: 200, G: 210, B: 255, A: 255}),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" quit").Style(style.Style{
					Foreground: style.Some[color.Color](colHint),
				}),
			).Style(style.Style{
				Margin: style.Some(style.Edges(1, 0, 0, 0)),
			}),
		).Style(style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexColumn),
			Width:         style.Some(style.Cells(fieldWidth + 6)), // card width = field + padding
			Background:    style.Some[color.Color](colCard),
			Border:        style.SingleBorder().Color(colBorder).Some(),
			Padding:       style.Some(style.Edges(1, 2)),
		}),
	).Style(style.Style{
		Display:        style.Some(style.DisplayFlex),
		FlexDirection:  style.Some(style.FlexColumn),
		JustifyContent: style.Some(style.JustifyCenter),
		AlignItems:     style.Some(style.AlignCenter),
		Width:          style.Some(style.Percent(100)),
		Height:         style.Some(style.Percent(100)),
		Background:     style.Some[color.Color](colBG),
	})

	eng.Mount(root)

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
