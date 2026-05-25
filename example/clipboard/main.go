package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"strings"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/internal/term/kitty"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	f, _ := os.Create("clipboard.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	// 1. Initialize engine with Kitty and OSC 52 Extensions
	eng := engine.New(b, engine.Options{
		Logger:   logger,
		Profiler: true,
		Extensions: []backend.TerminalExtension{
			kitty.NewExtension(),
			// osc52.NewExtension(),
		},
	})

	// 2. UI Components
	statusText := element.Text("Ready. Try selecting text and pressing Ctrl+C.")

	// A static box with selectable text
	sourceText := element.Box(
		"Kite System Clipboard Demo\n\n" +
			"This text is in a standard Box element. " +
			"You can click and drag to select parts of this text, " +
			"then press Ctrl+C to copy it to your system clipboard.",
	).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Cells(8)),
		Background: style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 40, A: 255}),
		Border:     style.SingleBorder().Color(color.RGBA{R: 100, G: 100, B: 255, A: 255}).Some(),
		Padding:    style.Some(style.Edges(1, 2)),
		Margin:     style.Some(style.Edges(0, 0, 1, 0)),
	})

	// An input field to paste into
	pasteInput := element.NewInput(eng.Document(), "")
	pasteInput.Style(style.Style{
		Display:    style.Some(style.DisplayBlock),
		Width:      style.Some(style.Auto),
		Background: style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 25, A: 255}),
		Border:     style.SingleBorder().Color(color.RGBA{R: 100, G: 200, B: 100, A: 255}).Some(),
		Padding:    style.Some(style.Edges(0, 1)),
	})

	// 3. Layout
	root := element.Box(
		element.Box("📋 Clipboard Integration Showcase").Style(style.Style{
			Bold:      style.Some(true),
			TextAlign: style.Some(style.TextAlignCenter),
			Margin:    style.Some(style.Edges(0, 0, 1, 0)),
		}),

		element.Box("Source Text (Selectable):").Style(style.Style{Bold: style.Some(true)}),
		sourceText,

		element.Box("Paste Target (Input):").Style(style.Style{Bold: style.Some(true), Margin: style.Some(style.Edges(1, 0, 0, 0))}),
		pasteInput,

		element.Box("").Style(style.Style{Flex: style.Some(style.Flex(1))}), // Spacer

		element.Box(
			element.Span("Status: ").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255})}),
			element.Span(statusText).Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 255, G: 200, B: 0, A: 255}), Bold: style.Some(true)}),
		).Style(style.Style{
			Padding:    style.Some(style.Edges(1)),
			Background: style.Some[color.Color](color.RGBA{R: 25, G: 25, B: 30, A: 255}),
			Border:     style.SingleBorder().Top(true).Some(),
		}),

		element.Box("Instructions: Ctrl+C (Copy), Ctrl+V (Paste), Q (Quit)").Style(style.Style{
			Foreground: style.Some[color.Color](color.RGBA{R: 100, G: 100, B: 100, A: 255}),
			TextAlign:  style.Some(style.TextAlignCenter),
			Margin:     style.Some(style.Edges(1, 0, 0, 0)),
		}),
	).Style(style.Style{
		Display:       style.Some(style.DisplayFlex),
		FlexDirection: style.Some(style.FlexColumn),
		AlignItems:    style.Some(style.AlignStretch),
		Width:         style.Some(style.Percent(100)),
		Height:        style.Some(style.Percent(100)),
		Background:    style.Some[color.Color](color.RGBA{R: 15, G: 15, B: 20, A: 255}),
		Padding:       style.Some(style.Edges(1, 2)),
	})

	// 4. Document-level Event Listeners for feedback
	eng.Document().AddEventListener(event.EventCopy, func(e event.Event) {
		ce := e.(*event.ClipboardEvent)
		text := ce.Text()
		if text != "" {
			statusText.SetData(fmt.Sprintf("Copied: %q", truncate(text, 30)))
		} else {
			statusText.SetData("Copy failed: nothing selected")
		}
		eng.RequestFrame()
	})

	eng.Document().AddEventListener(event.EventPaste, func(e event.Event) {
		ce := e.(*event.ClipboardEvent)
		text := ce.Text()

		hasImage := false
		imgType := ""
		for mime := range ce.Items {
			if strings.HasPrefix(mime, "image/") {
				hasImage = true
				imgType = mime
				break
			}
		}

		if hasImage {
			statusText.SetData(fmt.Sprintf("IMAGE DATA RECEIVED (%s): %d bytes", imgType, len(ce.Items[imgType])))
		} else {
			// Check if the text looks like an image path (from drag and drop)
			lower := strings.ToLower(text)
			isImagePath := strings.HasSuffix(lower, ".png") ||
				strings.HasSuffix(lower, ".jpg") ||
				strings.HasSuffix(lower, ".jpeg") ||
				strings.HasSuffix(lower, ".gif")

			if isImagePath {
				statusText.SetData(fmt.Sprintf("IMAGE PATH DETECTED: %q", truncate(text, 50)))
			} else {
				statusText.SetData(fmt.Sprintf("Pasted Text: %q", truncate(text, 50)))
			}
		}
		eng.RequestFrame()
	})

	// 5. Run App
	eng.Mount(root)
	devtools.Install(eng, devtools.Options{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") {
			cancel()
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
