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

	// --- Checkbox Section ---
	cbStatus := element.Text("Unchecked")
	cb := element.Checkbox(false).Style(style.Style{
		Margin: style.Some(style.Edges(0, 1, 0, 0)),
	})
	cb.OnEvent(event.EventChange, func(e event.Event) {
		if cb.Checked() {
			cbStatus.SetData("Checked")
		} else {
			cbStatus.SetData("Unchecked")
		}
		eng.RequestFrame()
	})

	// --- Radio Section ---
	radioStatus := element.Text("None selected")
	rg := element.RadioGroup(
		element.Box(
			element.Radio("option1"),
			element.Span(" Option 1"),
		).Style(style.Style{Margin: style.Some(style.Edges(0, 0, 0, 0))}),
		element.Box(
			element.Radio("option2"),
			element.Span(" Option 2"),
		).Style(style.Style{Margin: style.Some(style.Edges(0, 0, 0, 0))}),
		element.Box(
			element.Radio("option3"),
			element.Span(" Option 3"),
		).Style(style.Style{Margin: style.Some(style.Edges(0, 0, 0, 0))}),
	).OnChange(func(val string) {
		radioStatus.SetData(fmt.Sprintf("Selected: %s", val))
		eng.RequestFrame()
	})

	root := element.Box(
		element.Box(
			element.Box("Checkbox and Radio Components").Style(style.Style{
				Bold:      style.Some(true),
				TextAlign: style.Some(style.TextAlignCenter),
				Margin:    style.Some(style.Edges(0, 0, 1, 0)),
			}),

			// Checkbox Row
			element.Box(
				element.Box("Checkbox:").Style(style.Style{Bold: style.Some(true), Margin: style.Some(style.Edges(0, 1, 0, 0))}),
				cb,
				element.Box(cbStatus).Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 100, G: 200, B: 255, A: 255})}),
			).Style(style.Style{
				Display:       style.Some(style.DisplayFlex),
				FlexDirection: style.Some(style.FlexRow),
				AlignItems:    style.Some(style.AlignCenter),
				Margin:        style.Some(style.Edges(0, 0, 2, 0)),
			}),

			// Radio Group Section
			element.Box("Radio Group:").Style(style.Style{Bold: style.Some(true), Margin: style.Some(style.Edges(0, 0, 1, 0))}),
			rg,
			element.Box(radioStatus).Style(style.Style{
				Foreground: style.Some[color.Color](color.RGBA{R: 100, G: 255, B: 100, A: 255}),
				Margin:     style.Some(style.Edges(1, 0, 0, 0)),
			}),

			element.Box("\nInstructions:").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255})}),
			element.Box("- Click with Mouse to toggle/select").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255})}),
			element.Box("- Tab to focus, then press Space to toggle/select").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255})}),
			element.Box("- Use Arrow keys to navigate Radio buttons").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255})}),
			element.Box("- Press 'q' to quit").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255})}),
		).Style(style.Style{
			Display:        style.Some(style.DisplayFlex),
			FlexDirection:  style.Some(style.FlexColumn),
			AlignItems:     style.Some(style.AlignStart),
			JustifyContent: style.Some(style.JustifyCenter),
			Width:          style.Some(style.Percent(80)),
			Height:         style.Some(style.Percent(80)),
			Background:     style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 30, A: 255}),
			Padding:        style.Some(style.Edges(1, 2)),
			Border:         style.SingleBorder().Some(),
		}),
	).Style(style.Style{
		Display:        style.Some(style.DisplayFlex),
		JustifyContent: style.Some(style.JustifyCenter),
		AlignItems:     style.Some(style.AlignCenter),
		Width:          style.Some(style.Percent(100)),
		Height:         style.Some(style.Percent(100)),
		Background:     style.Some[color.Color](color.RGBA{R: 15, G: 15, B: 15, A: 255}),
	})

	eng.Mount(root)

	// Add capturing event listeners for checkbox and radio buttons focus/blur styling
	eng.Document().AddEventListener(event.EventFocus, func(e event.Event) {
		if et := e.Target().EventTarget(); et != nil {
			switch el := et.(type) {
			case *element.CheckboxElement:
				s := el.RawStyle()
				s.Foreground = style.Some[color.Color](color.RGBA{R: 255, G: 215, B: 0, A: 255}) // Gold when focused
				el.Style(s)
			case *element.RadioElement:
				s := el.RawStyle()
				s.Foreground = style.Some[color.Color](color.RGBA{R: 255, G: 215, B: 0, A: 255}) // Gold when focused
				el.Style(s)
			}
		}
	}, event.Capture())

	eng.Document().AddEventListener(event.EventBlur, func(e event.Event) {
		if et := e.Target().EventTarget(); et != nil {
			switch el := et.(type) {
			case *element.CheckboxElement:
				s := el.RawStyle()
				s.Foreground = style.Some[color.Color](style.TerminalDefault) // Revert to default on blur
				el.Style(s)
			case *element.RadioElement:
				s := el.RawStyle()
				s.Foreground = style.Some[color.Color](style.TerminalDefault) // Revert to default on blur
				el.Style(s)
			}
		}
	}, event.Capture())

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
