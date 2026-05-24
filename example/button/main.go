package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"

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

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{Logger: logger, Profiler: true})

	count := 0
	counterText := element.Text("Clicks: 0")

	// Create a standard button
	btn1 := element.Button("  Click Me  ").Style(style.Style{
		Background: style.Some[color.Color](color.RGBA{R: 60, G: 60, B: 100, A: 255}),
		Foreground: style.Some[color.Color](color.White),
	})

	btn1.OnEvent(event.EventClick, func(e event.Event) {
		count++
		counterText.SetData(fmt.Sprintf("Clicks: %d", count))
		eng.RequestFrame()
	})

	// Create a disabled button
	btn2 := element.Button("Disabled").Disabled(true).Style(style.Style{
		Margin: style.Some(style.Edges(1, 0, 0, 0)),
	})

	// Create a styled "action" button
	btn3 := element.Button("  DANGER  ").Style(style.Style{
		Background: style.Some[color.Color](color.RGBA{R: 150, G: 40, B: 40, A: 255}),
		Foreground: style.Some[color.Color](color.White),
		Bold:       style.Some(true),
		Border:     style.DoubleBorder().Some(),
		Padding:    style.Some(style.Edges(0, 2)),
		Margin:     style.Some(style.Edges(1, 0, 0, 0)),
	})

	btn3.OnEvent(event.EventClick, func(e event.Event) {
		count = 0
		counterText.SetData("Clicks: 0 (Reset!)")
		eng.RequestFrame()
	})

	root := element.Box(
		element.Box(
			element.Box("Button Component Demonstration").Style(style.Style{
				Bold:      style.Some(true),
				TextAlign: style.Some(style.TextAlignCenter),
				Margin:    style.Some(style.Edges(0, 0, 1, 0)),
			}),

			element.Box(counterText).Style(style.Style{
				Foreground: style.Some[color.Color](color.RGBA{R: 100, G: 255, B: 100, A: 255}),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
			}),

			btn1,
			btn2,
			btn3,

			element.Box("\nInstructions:").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255})}),
			element.Box("- Click with Mouse").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255})}),
			element.Box("- Tab to focus, then press Space or Enter").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255})}),
			element.Box("- Press 'q' to quit").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255})}),
		).Style(style.Style{
			Display:        style.Some(style.DisplayFlex),
			FlexDirection:  style.Some(style.FlexColumn),
			AlignItems:     style.Some(style.AlignCenter),
			JustifyContent: style.Some(style.JustifyCenter),
			Width:          style.Some(style.Percent(100)),
			Height:         style.Some(style.Percent(100)),
			Background:     style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 20, A: 255}),
		}),
	)

	eng.Mount(root)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
		}
	})

	devtools.Install(eng, devtools.Options{})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
