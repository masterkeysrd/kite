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
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

var (
	anchorStyle           = style.S().Background(color.RGBA{R: 0, G: 128, B: 0, A: 255}).Foreground(color.White).Padding(1, 2).Border(style.SingleBorder()).Width(style.Cells(12))
	boldLabelStyle        = style.S().Bold(true)
	controlsStyle         = style.S().Margin(1, 0)
	titleStyle            = style.S().TextAlign(style.TextAlignCenter).Background(color.RGBA{R: 50, G: 50, B: 50, A: 255}).Padding(1)
	infoTextStyle         = style.S().Foreground(color.RGBA{R: 255, G: 255, B: 0, A: 255})
	anchorWrapperStyle    = style.S().Margin(5, 20)
	contentContainerStyle = style.S().Padding(1, 2)
	rootStyle             = style.S().Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 20, G: 20, B: 20, A: 255})
	overlayStyle          = style.S().Background(color.RGBA{R: 128, G: 0, B: 0, A: 255}).Border(style.DoubleBorder()).Padding(0, 1)
)

func main() {
	f, _ := os.Create("overlay_tweaks.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	_ = logger // prevent unused variable error
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

	eng := engine.New(b, engine.Options{})

	// State for overlay configuration
	currentPlacement := geom.PlacementBottom
	flipEnabled := true

	// Anchor element
	anchor := element.Box("  ANCHOR  ").Style(anchorStyle)

	// Info display
	infoText := element.Text("")
	updateInfo := func() {
		placementStr := ""
		switch currentPlacement {
		case geom.PlacementTop:
			placementStr = "Top"
		case geom.PlacementBottom:
			placementStr = "Bottom"
		case geom.PlacementLeft:
			placementStr = "Left"
		case geom.PlacementRight:
			placementStr = "Right"
		}
		infoText.SetData(fmt.Sprintf("Placement: %s | Flip: %v", placementStr, flipEnabled))
	}
	updateInfo()

	// Control hints
	controls := element.Box(
		element.Box("Controls:").Style(boldLabelStyle),
		element.Box(" [1-4] Set Placement (Top, Bottom, Left, Right)"),
		element.Box(" [f]   Toggle Flip"),
		element.Box(" [q]   Quit"),
	).Style(controlsStyle)

	// Layout root
	root := element.Box(
		element.Box("Overlay Tweaks Example").Style(titleStyle),
		element.Box(
			element.Box(infoText).Style(infoTextStyle),
			controls,
			element.Box(anchor).Style(anchorWrapperStyle),
		).Style(contentContainerStyle),
	).Style(rootStyle)

	eng.Mount(root)
	devtools.Install(eng, devtools.Options{})

	// Overlay content
	ovlContent := element.Box(
		element.Box("I am an Overlay").Style(boldLabelStyle),
		element.Box("Try moving me!"),
	).Style(overlayStyle)

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
			currentPlacement = geom.PlacementTop
			changed = true
		} else if ke.MatchString("2") {
			currentPlacement = geom.PlacementBottom
			changed = true
		} else if ke.MatchString("3") {
			currentPlacement = geom.PlacementLeft
			changed = true
		} else if ke.MatchString("4") {
			currentPlacement = geom.PlacementRight
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
