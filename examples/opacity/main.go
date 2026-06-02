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

func createHoverButton(eng *engine.Engine, label string, baseBg color.RGBA) *element.BoxElement {
	// The actual button inside, with a transparent background by default
	btn := element.Button(label).Style(style.Style{
		Background: style.Some[color.Color](color.Transparent),
		Foreground: style.Some[color.Color](color.White),
	})

	// When hovered, we set a semi-transparent white background on the button.
	// This will blend with the wrapper box's base background color in the framebuffer!
	btn.OnEvent(event.EventMouseEnter, func(e event.Event) {
		btn.Style(style.Style{
			Background: style.Some[color.Color](color.RGBA{R: 255, G: 255, B: 255, A: 60}), // 60/255 white overlay
			Foreground: style.Some[color.Color](color.White),
		})
		eng.RequestFrame()
	})

	btn.OnEvent(event.EventMouseLeave, func(e event.Event) {
		btn.Style(style.Style{
			Background: style.Some[color.Color](color.Transparent),
			Foreground: style.Some[color.Color](color.White),
		})
		eng.RequestFrame()
	})

	// A wrapper box that holds the solid base background color
	wrapper := element.Box(btn).Style(style.Style{
		Background: style.Some[color.Color](baseBg),
		Padding:    style.Some(style.Edges(0, 1)),
		Margin:     style.Some(style.Edges(0, 1)),
		Border:     style.SingleBorder().Color(color.RGBA{255, 255, 255, 80}).Some(),
	})

	return wrapper
}

func main() {
	f, _ := os.Create("kite.log")
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

	eng := engine.New(b, engine.Options{Logger: logger, Profiler: true})

	// Background grid with multiple colors to show off the transparency blending.
	root := element.Box(
		element.Box("Opacity & Color Blending Demonstration").Style(style.Style{
			TextAlign:  style.Some(style.TextAlignCenter),
			Background: style.Some[color.Color](color.RGBA{R: 50, G: 80, B: 200, A: 255}),
			Padding:    style.Some(style.Edges(1)),
			Bold:       style.Some(true),
		}),
		element.Box(
			element.Box("RED BAND (Solid Background)").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 200, G: 50, B: 50, A: 255}),
				Padding:    style.Some(style.Edges(1, 2)),
			}),
			element.Box("GREEN BAND (Solid Background)").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 50, G: 180, B: 50, A: 255}),
				Padding:    style.Some(style.Edges(1, 2)),
			}),
			element.Box("BLUE BAND (Solid Background)").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 50, G: 50, B: 200, A: 255}),
				Padding:    style.Some(style.Edges(1, 2)),
			}),
			element.Box("YELLOW BAND (Solid Background)").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 200, G: 180, B: 50, A: 255}),
				Padding:    style.Some(style.Edges(1, 2)),
			}),
		).Style(style.Style{
			Margin: style.Some(style.Edges(1, 0)),
		}),
		element.Box("Hover Buttons (Blends semi-transparent white hover state on base color)").Style(style.Style{
			Bold:   style.Some(true),
			Margin: style.Some(style.Edges(1, 0, 0, 0)),
		}),
		element.Box(
			createHoverButton(eng, "  Blue Button  ", color.RGBA{R: 30, G: 60, B: 180, A: 255}),
			createHoverButton(eng, "  Green Button  ", color.RGBA{R: 30, G: 150, B: 60, A: 255}),
			createHoverButton(eng, "  Purple Button  ", color.RGBA{R: 120, G: 30, B: 150, A: 255}),
		).Style(style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexRow),
			Margin:        style.Some(style.Edges(0, 0, 1, 0)),
		}),
		element.Box(
			"Press 'o' to toggle the semi-transparent overlay dialog.",
			"\nPress '+' or '-' to adjust overlay opacity when visible.",
			"\nPress 'q' or 'ctrl+c' to quit.",
		).Style(style.Style{
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
	devtools.Install(eng, devtools.Options{})

	// Interactive state
	alpha := uint8(128)
	overlayVisible := false

	// Create a dynamic function to build the overlay content at the current alpha.
	getOverlay := func() *element.BoxElement {
		return element.Box(
			element.Box("Semi-Transparent Dialog").Style(style.Style{
				TextAlign: style.Some(style.TextAlignCenter),
				Margin:    style.Some(style.Edges(0, 0, 1, 0)),
				Bold:      style.Some(true),
			}),
			fmt.Sprintf("Current Alpha Opacity: %d / 255", alpha),
			"\nNotice how the colored bands behind me\nblend through the dialog background!",
			element.Box("Press '+' to increase opacity, '-' to decrease.").Style(style.Style{
				Margin:     style.Some(style.Edges(1, 0, 0, 0)),
				TextAlign:  style.Some(style.TextAlignCenter),
				Foreground: style.Some[color.Color](color.RGBA{R: 100, G: 100, B: 100, A: 255}),
			}),
		).Style(style.Style{
			Width:  style.Some(style.Cells(45)),
			Height: style.Some(style.Cells(10)),
			// Using white base color with the dynamic alpha level for opacity blending
			Background: style.Some[color.Color](color.RGBA{R: 255, G: 255, B: 255, A: alpha}),
			// Dark text so it stands out against the light/transparent background
			Foreground: style.Some[color.Color](color.RGBA{R: 0, G: 0, B: 0, A: 255}),
			Border:     style.DoubleBorder().Color(color.RGBA{R: 255, G: 255, B: 255, A: 255}).Some(),
			Padding:    style.Some(style.Edges(1, 2)),
		})
	}

	// Full-screen overlay container
	var overlayDialog *element.BoxElement

	updateOverlay := func() {
		if overlayVisible && overlayDialog != nil {
			// If already showing, hide the old one first
			eng.Document().HideOverlay(overlayDialog)
		}
		overlayContent := getOverlay()
		overlayDialog = element.Box(overlayContent).Style(style.Style{
			Width:          style.Some(style.Percent(100)),
			Height:         style.Some(style.Percent(100)),
			Display:        style.Some(style.DisplayFlex),
			JustifyContent: style.Some(style.JustifyCenter),
			AlignItems:     style.Some(style.AlignCenter),
		})
		if overlayVisible {
			eng.Document().ShowOverlay(overlayDialog, 100)
		}
	}

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
				if overlayDialog != nil {
					eng.Document().HideOverlay(overlayDialog)
				}
				overlayVisible = false
			} else {
				overlayVisible = true
				updateOverlay()
			}
			return
		}

		if overlayVisible {
			if ke.MatchString("+") || ke.MatchString("=") {
				if alpha < 235 {
					alpha += 20
				} else {
					alpha = 255
				}
				updateOverlay()
			} else if ke.MatchString("-") {
				if alpha > 20 {
					alpha -= 20
				} else {
					alpha = 0
				}
				updateOverlay()
			}
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited with error: %v\n", err)
		os.Exit(1)
	}
}
