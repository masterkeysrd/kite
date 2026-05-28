// Package main demonstrates the TextAreaElement component introduced in TSK-025.
//
// It renders a multi-line text editor with a status bar. The demo shows:
//
//   - Multi-line editing with Enter, Backspace, and Delete
//   - 2D navigation using Arrow keys (Up/Down/Left/Right)
//   - Intrinsic soft-wrapping via white-space: pre-wrap
//   - Emergency breaking via overflow-wrap: break-word
//   - Programmatic scrolling to keep the cursor in view
//   - Real-time line and column count in the footer
//   - Q exits the application
package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/internal/term/osc52"
	"github.com/masterkeysrd/kite/style"
)

// ── palette ──────────────────────────────────────────────────────────────────

var (
	colBG     = color.RGBA{R: 18, G: 18, B: 23, A: 255}    // app background
	colCard   = color.RGBA{R: 28, G: 30, B: 40, A: 255}    // editor surface
	colBorder = color.RGBA{R: 60, G: 65, B: 90, A: 255}    // editor border
	colTitle  = color.RGBA{R: 200, G: 210, B: 255, A: 255} // title text
	colStatus = color.RGBA{R: 100, G: 200, B: 130, A: 255} // status text
	colHint   = color.RGBA{R: 80, G: 85, B: 110, A: 255}   // hint text
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
	var b *uv.Backend
	if os.Getenv("USE_MOCK_BACKEND") == "1" {
		// b = mock.New(80, 24)
	} else {
		b, err = uv.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize backend: %v\n", err)
			os.Exit(1)
		}
	}

	// ── engine ───────────────────────────────────────────────────────────────
	eng := engine.New(b, engine.Options{
		Logger:   logger,
		Profiler: true,
		Extensions: []backend.TerminalExtension{
			osc52.NewExtension(),
		},
	})

	// ── textarea ─────────────────────────────────────────────────────────────
	initialText := "Welcome to Kite TextArea!\n\n" +
		"This component supports:\n" +
		" • Multi-line editing\n" +
		" • 2D Arrow navigation\n" +
		" • Automatic soft-wrap\n" +
		" • LongUnbreakableRunsThatEmergencyWrap"

	txa := element.NewTextArea(eng.Document(), initialText)
	txa.Style(style.Style{
		Width:      style.Some(style.Cells(50)),
		Height:     style.Some(style.Cells(10)),
		Background: style.Some[color.Color](colCard),
		Foreground: style.Some[color.Color](color.RGBA{R: 220, G: 225, B: 255, A: 255}),
		Border:     style.SingleBorder().Color(colBorder).Some(),
		Padding:    style.Some(style.Edges(0, 1)),
	})

	// ── status bar ───────────────────────────────────────────────────────────
	statusText := element.Text("Pos: (0, 0)")

	updateStatus := func() {
		state := txa.CursorState()
		statusText.SetData(fmt.Sprintf("Pos: (%d, %d)", state.X, state.Y))
		eng.RequestFrame()
	}

	// ── UI tree ──────────────────────────────────────────────────────────────
	root := element.Box(
		element.Box(
			// Title
			element.Box("  Kite Text Editor  ").Style(style.Style{
				Foreground: style.Some[color.Color](colTitle),
				Bold:       style.Some(true),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
			}),

			// Editor
			txa,

			// Status Bar
			element.Box(
				element.Span("Status: ").Style(style.Style{Foreground: style.Some[color.Color](colHint)}),
				element.Span(statusText).Style(style.Style{Foreground: style.Some[color.Color](colStatus), Bold: style.Some(true)}),
			).Style(style.Style{
				Margin: style.Some(style.Edges(1, 0, 0, 0)),
			}),

			// Help Hints
			element.Box(
				element.Span(" Arrows").Style(style.Style{
					Background: style.Some[color.Color](colBorder),
					Foreground: style.Some[color.Color](colTitle),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" navigate  ").Style(style.Style{Foreground: style.Some[color.Color](colHint)}),
				element.Span(" Enter").Style(style.Style{
					Background: style.Some[color.Color](colBorder),
					Foreground: style.Some[color.Color](colTitle),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" newline  ").Style(style.Style{Foreground: style.Some[color.Color](colHint)}),
				element.Span(" Q").Style(style.Style{
					Background: style.Some[color.Color](colBorder),
					Foreground: style.Some[color.Color](colTitle),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" quit").Style(style.Style{Foreground: style.Some[color.Color](colHint)}),
			).Style(style.Style{
				Margin: style.Some(style.Edges(1, 0, 0, 0)),
			}),
		).Style(style.Style{
			Padding:       style.Some(style.Edges(1, 2)),
			Background:    style.Some[color.Color](colBG),
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexColumn),
			AlignItems:    style.Some(style.AlignStart),
		}),
	)

	// ── engine run ───────────────────────────────────────────────────────────
	eng.Mount(root)
	updateStatus() // Initial status

	// Install devtools (Inspector + X-Ray)
	devtools.Install(eng, devtools.Options{})

	// Exit handlers
	root.AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") {
			eng.Stop()
		}
		if ke.MatchString("ctrl+p") {
			_ = eng.Dump("kite-dump.json")
			b.DumpState()
		}
	})

	if err := eng.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited with error: %v\n", err)
		os.Exit(1)
	}
}
