package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"strings"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

var (
	sourceBoxStyle       = style.S().Width(style.Percent(100)).Height(style.Cells(8)).Background(color.RGBA{R: 30, G: 30, B: 40, A: 255}).Border(style.SingleBorder().Color(color.RGBA{R: 100, G: 100, B: 255, A: 255})).Padding(style.Edges(1, 2)).Margin(style.Edges(0, 0, 1, 0))
	inputStyle           = style.S().Display(style.DisplayBlock).Width(style.Auto).Background(color.RGBA{R: 20, G: 20, B: 25, A: 255}).Border(style.SingleBorder().Color(color.RGBA{R: 100, G: 200, B: 100, A: 255})).Padding(style.Edges(0, 1))
	titleStyle           = style.S().Bold(true).TextAlign(style.TextAlignCenter).Margin(style.Edges(0, 0, 1, 0))
	labelStyle           = style.S().Bold(true)
	labelWithMarginStyle = style.S().Bold(true).Margin(style.Edges(1, 0, 0, 0))
	spacerStyle          = style.S().Flex(style.Flex(1))
	statusLabelStyle     = style.S().Foreground(color.RGBA{R: 150, G: 150, B: 150, A: 255})
	statusValueStyle     = style.S().Foreground(color.RGBA{R: 255, G: 200, B: 0, A: 255}).Bold(true)
	statusContainerStyle = style.S().Padding(style.Edges(1)).Background(color.RGBA{R: 25, G: 25, B: 30, A: 255}).Border(style.SingleBorder().Top(true))
	instructionsStyle    = style.S().Foreground(color.RGBA{R: 100, G: 100, B: 100, A: 255}).TextAlign(style.TextAlignCenter).Margin(style.Edges(1, 0, 0, 0))
	rootStyle            = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).AlignItems(style.AlignStretch).Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 15, G: 15, B: 20, A: 255}).Padding(style.Edges(1, 2))
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

	// 1. Initialize engine
	eng := engine.New(b, engine.Options{
		Logger:   logger,
		Profiler: true,
	})

	// 2. UI Components
	statusText := element.Text("Ready. Try selecting text and pressing Ctrl+C.")

	// A static box with selectable text
	sourceText := element.Box(
		"Kite System Clipboard Demo\n\n" +
			"This text is in a standard Box element. " +
			"You can click and drag to select parts of this text, " +
			"then press Ctrl+C to copy it to your system clipboard.",
	).Style(sourceBoxStyle)

	// An input field to paste into
	pasteInput := element.NewInput(eng.Document(), "")
	pasteInput.Style(inputStyle)

	// 3. Layout
	root := element.Box(
		element.Box("📋 Clipboard Integration Showcase").Style(titleStyle),

		element.Box("Source Text (Selectable):").Style(labelStyle),
		sourceText,

		element.Box("Paste Target (Input):").Style(labelWithMarginStyle),
		pasteInput,

		element.Box("").Style(spacerStyle), // Spacer

		element.Box(
			element.Span("Status: ").Style(statusLabelStyle),
			element.Span(statusText).Style(statusValueStyle),
		).Style(statusContainerStyle),

		element.Box("Instructions: Ctrl+C (Copy), Ctrl+V (Paste), Q (Quit)").Style(instructionsStyle),
	).Style(rootStyle)

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
