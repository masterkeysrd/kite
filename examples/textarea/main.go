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
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

var (
	statusRowStyle = style.S().Margin(1, 0, 0, 0)
)

// ── palette ──────────────────────────────────────────────────────────────────

var (
	colBG     = style.RGB(18, 18, 23)    // app background
	colCard   = style.RGB(28, 30, 40)    // editor surface
	colBorder = style.RGB(60, 65, 90)    // editor border
	colTitle  = style.RGB(200, 210, 255) // title text
	colStatus = style.RGB(100, 200, 130) // status text
	colHint   = style.RGB(80, 85, 110)   // hint text
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
	eng := engine.New(b, engine.Options{})

	// ── textarea ─────────────────────────────────────────────────────────────
	txa := element.NewTextArea(eng.Document(), "").WithPlaceholder("Type your message here...")
	txa.Style(
		style.S().
			Width(style.Cells(50)).
			Height(style.Cells(10)).
			Padding(0, 1).
			Border(style.SingleBorder().Color(colBorder)).
			Background(colCard).
			Foreground(style.RGB(220, 225, 255)),
	)

	// ── status bar ───────────────────────────────────────────────────────────
	statusText := element.Text("Pos: (0, 0)")

	updateStatus := func() {
		state := txa.CursorState()
		statusText.SetData(fmt.Sprintf("Pos: (%d, %d)", state.X, state.Y))
		eng.RequestFrame()
	}

	txa.AddEventListener(event.EventKeyDown, func(e event.Event) {
		updateStatus()
	})

	// ── UI tree ──────────────────────────────────────────────────────────────
	root := element.Box(
		element.Box(
			// Title
			element.Box("  Kite Text Editor  ").Style(
				style.S().
					Margin(0, 0, 1, 0).
					Foreground(colTitle).
					Bold(true),
			),

			// Editor
			txa,

			// Status Bar
			element.Box(
				element.Span("Status: ").Style(style.S().Foreground(colHint)),
				element.Span(statusText).Style(
					style.S().
						Foreground(colStatus).
						Bold(true),
				),
			).Style(statusRowStyle),

			// Help Hints
			element.Box(
				element.Span(" Arrows").Style(
					style.S().
						Padding(0, 1).
						Background(colBorder).
						Foreground(colTitle),
				),
				element.Span(" navigate  ").Style(style.S().Foreground(colHint)),
				element.Span(" Enter").Style(
					style.S().
						Padding(0, 1).
						Background(colBorder).
						Foreground(colTitle),
				),
				element.Span(" newline  ").Style(style.S().Foreground(colHint)),
				element.Span(" Q").Style(
					style.S().
						Padding(0, 1).
						Background(colBorder).
						Foreground(colTitle),
				),
				element.Span(" quit").Style(style.S().Foreground(colHint)),
			).Style(statusRowStyle),
		).Style(
			style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				AlignItems(style.AlignStart).
				Padding(1, 2).
				Background(colBG),
		),
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
