package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"runtime"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	f, _ := os.Create("kite.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "unexpected error: %v\n", r)

			// Print the stack trace for debugging
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			fmt.Fprintf(os.Stderr, "stack trace:\n%s\n", string(buf[:n]))
			os.Exit(1)
		}
	}()

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{Logger: logger, Profiler: true})

	// Create a main background element
	root := element.Box(
		element.Box("Overlay API Example").Style(style.Style{
			TextAlign:  style.Some(style.TextAlignCenter),
			Background: style.Some[color.Color](color.RGBA{R: 50, G: 50, B: 80, A: 255}),
			Padding:    style.Some(style.Edges(1)),
		}),
		element.Box(
			"Press 'o' to toggle the Overlay.",
			"\nPress 'q' or 'ctrl+c' to quit.",
		).Style(style.Style{
			Margin:  style.Some(style.Edges(2, 0)),
			Padding: style.Some(style.Edges(1, 2)),
			Border:  style.SingleBorder().Some(),
		}),
	).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
		Background: style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 30, A: 255}),
		Padding:    style.Some(style.Edges(2)),
	})

	eng.Mount(root)

	// Install devtools (Inspector + X-Ray)
	devtools.Install(eng, devtools.Options{})

	// Create the overlay content
	overlayContent := element.Box(
		element.Box("I am an Overlay!").Style(style.Style{
			TextAlign: style.Some(style.TextAlignCenter),
			Margin:    style.Some(style.Edges(0, 0, 1, 0)),
			Bold:      style.Some(true),
		}),
		"I am rendered in the Top Layer,\nabove the normal document flow.",
		element.Box("Press 'o' to close me.").Style(style.Style{
			Margin:     style.Some(style.Edges(1, 0, 0, 0)),
			TextAlign:  style.Some(style.TextAlignCenter),
			Foreground: style.Some[color.Color](color.RGBA{R: 200, G: 200, B: 200, A: 255}),
		}),
	).Style(style.Style{
		Width:      style.Some(style.Cells(40)),
		Height:     style.Some(style.Cells(10)),
		Background: style.Some[color.Color](color.RGBA{R: 80, G: 40, B: 40, A: 255}),
		Border:     style.DoubleBorder().Color(color.RGBA{R: 255, G: 100, B: 100, A: 255}).Some(),
		Padding:    style.Some(style.Edges(1, 2)),
	})

	// Create a full-screen container to center the overlay content.
	// This is the robust way to center overlays regardless of terminal size.
	overlayDialog := element.Box(overlayContent).Style(style.Style{
		Width:          style.Some(style.Percent(100)),
		Height:         style.Some(style.Percent(100)),
		Display:        style.Some(style.DisplayFlex),
		JustifyContent: style.Some(style.JustifyCenter),
		AlignItems:     style.Some(style.AlignCenter),
	})

	overlayVisible := false

	// Global key listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("ctrl+c") || ke.MatchString("q") {
			cancel()
			return
		}

		if ke.MatchString("o") {
			if overlayVisible {
				eng.Document().HideOverlay(overlayDialog)
				overlayVisible = false
			} else {
				eng.Document().ShowOverlay(overlayDialog, 100)
				overlayVisible = true
			}
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited with error: %v\n", err)
		os.Exit(1)
	}
}
