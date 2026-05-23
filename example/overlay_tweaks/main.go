package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"runtime"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	f, _ := os.Create("overlay_tweaks.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "unexpected error: %v\n", r)
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

	eng := engine.New(b, engine.Options{Logger: logger})

	// State for overlay configuration
	currentPlacement := layout.PlacementBottom
	flipEnabled := true

	// Anchor element
	anchor := element.Box("  ANCHOR  ").Style(style.Style{
		Background: style.Some[color.Color](color.RGBA{R: 0, G: 128, B: 0, A: 255}),
		Foreground: style.Some[color.Color](color.White),
		Padding:    style.Some(style.Edges(1, 2)),
		Border:     style.SingleBorder().Some(),
		Width:      style.Some(style.Cells(12)),
	})

	// Info display
	infoText := element.Text("")
	updateInfo := func() {
		placementStr := ""
		switch currentPlacement {
		case layout.PlacementTop:
			placementStr = "Top"
		case layout.PlacementBottom:
			placementStr = "Bottom"
		case layout.PlacementLeft:
			placementStr = "Left"
		case layout.PlacementRight:
			placementStr = "Right"
		}
		infoText.SetData(fmt.Sprintf("Placement: %s | Flip: %v", placementStr, flipEnabled))
	}
	updateInfo()

	// Control hints
	controls := element.Box(
		element.Box("Controls:").Style(style.Style{Bold: style.Some(true)}),
		element.Box(" [1-4] Set Placement (Top, Bottom, Left, Right)"),
		element.Box(" [f]   Toggle Flip"),
		element.Box(" [q]   Quit"),
	).Style(style.Style{
		Margin: style.Some(style.Edges(1, 0)),
	})

	// Layout root
	root := element.Box(
		element.Box("Overlay Tweaks Example").Style(style.Style{
			TextAlign:  style.Some(style.TextAlignCenter),
			Background: style.Some[color.Color](color.RGBA{R: 50, G: 50, B: 50, A: 255}),
			Padding:    style.Some(style.Edges(1)),
		}),
		element.Box(
			element.Box(infoText).Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 255, G: 255, B: 0, A: 255})}),
			controls,
			element.Box(anchor).Style(style.Style{
				Margin: style.Some(style.Edges(5, 20)),
			}),
		).Style(style.Style{
			Padding: style.Some(style.Edges(1, 2)),
		}),
	).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
		Background: style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 20, A: 255}),
	})

	eng.Mount(root)

	// Overlay content
	ovlContent := element.Box(
		element.Box("I am an Overlay").Style(style.Style{Bold: style.Some(true)}),
		element.Box("Try moving me!"),
	).Style(style.Style{
		Background: style.Some[color.Color](color.RGBA{R: 128, G: 0, B: 0, A: 255}),
		Border:     style.DoubleBorder().Some(),
		Padding:    style.Some(style.Edges(0, 1)),
	})

	var currentOverlay *element.OverlayElement

	updateOverlay := func() {
		if currentOverlay != nil {
			eng.Document().HideOverlay(currentOverlay)
		}
		currentOverlay = element.Overlay(ovlContent, element.OverlayConfig{
			Anchor:    anchor,
			Placement: currentPlacement,
			Flip:      flipEnabled,
		})
		eng.Document().ShowOverlay(currentOverlay, 100)
		updateInfo()
	}

	// Initial overlay
	updateOverlay()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		changed := false

		if ke.MatchString("1") {
			currentPlacement = layout.PlacementTop
			changed = true
		} else if ke.MatchString("2") {
			currentPlacement = layout.PlacementBottom
			changed = true
		} else if ke.MatchString("3") {
			currentPlacement = layout.PlacementLeft
			changed = true
		} else if ke.MatchString("4") {
			currentPlacement = layout.PlacementRight
			changed = true
		} else if ke.MatchString("f") {
			flipEnabled = !flipEnabled
			changed = true
		} else if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
			return
		}

		if changed {
			updateOverlay()
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited with error: %v\n", err)
		os.Exit(1)
	}
}
