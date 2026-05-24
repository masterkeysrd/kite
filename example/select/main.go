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

	eng := engine.New(b, engine.Options{Logger: logger})

	statusText := element.Text("(none)")

	sel := element.Select(
		element.Option("Go", "go"),
		element.Option("Rust", "rust"),
		element.Option("C++", "cpp"),
		element.Option("Zig", "zig"),
		element.Option("Python", "python"),
		element.Option("TypeScript", "ts"),
	).OnChange(func(val string) {
		statusText.SetData(val)
		eng.RequestFrame()
	}).Style(style.Style{
		Width: style.Some(style.Cells(25)),
	})

	// Helper for creating instruction lines that are explicitly block-level.
	instr := func(t string) *element.BoxElement {
		return element.Box(t).Style(style.Style{
			Display:    style.Some(style.DisplayBlock),
			Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 180, A: 255}),
			Margin:     style.Some(style.Edges(0, 0, 0, 0)),
		})
	}

	root := element.Box(
		element.Box(
			element.Box("Select Component Demo").Style(style.Style{
				Bold:       style.Some(true),
				TextAlign:  style.Some(style.TextAlignCenter),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
				Background: style.Some[color.Color](color.RGBA{R: 60, G: 60, B: 120, A: 255}),
				Foreground: style.Some[color.Color](color.White),
			}),

			element.Box(
				element.Span("Pick a language: ").Style(style.Style{Bold: style.Some(true)}),
				sel,
			).Style(style.Style{
				Display:       style.Some(style.DisplayFlex),
				FlexDirection: style.Some(style.FlexRow),
				AlignItems:    style.Some(style.AlignCenter),
				Margin:        style.Some(style.Edges(1, 0, 2, 0)),
				Gap:           style.Some(style.Gap(0, 1)),
			}),

			element.Box(
				element.Span("Selected: "),
				element.Span(statusText).Style(style.Style{
					Foreground: style.Some[color.Color](color.RGBA{R: 100, G: 255, B: 100, A: 255}),
					Bold:       style.Some(true),
				}),
			).Style(style.Style{
				Margin: style.Some(style.Edges(0, 0, 2, 0)),
				Border: style.SingleBorder().Left(false).Right(false).Top(false).Color(color.RGBA{R: 50, G: 50, B: 50, A: 255}).Some(),
			}),

			element.Box("Instructions:").Style(style.Style{
				Bold:   style.Some(true),
				Margin: style.Some(style.Edges(0, 0, 1, 0)),
			}),
			instr("• Click or Space/Enter to toggle"),
			instr("• Arrow keys to navigate options"),
			instr("• Enter to confirm selection"),
			instr("• Esc or Click outside to close"),
			instr("• Press 'q' to quit"),
		).Style(style.Style{
			Display:        style.Some(style.DisplayFlex),
			FlexDirection:  style.Some(style.FlexColumn),
			AlignItems:     style.Some(style.AlignStart),
			JustifyContent: style.Some(style.JustifyStart),
			Width:          style.Some(style.Cells(50)),
			Height:         style.Some(style.Auto),
			Background:     style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 30, A: 255}),
			Padding:        style.Some(style.Edges(1, 3)),
			Border:         style.SingleBorder().Color(color.RGBA{R: 100, G: 100, B: 100, A: 255}).Some(),
		})).Style(style.Style{
		Display:        style.Some(style.DisplayFlex),
		JustifyContent: style.Some(style.JustifyCenter),
		AlignItems:     style.Some(style.AlignCenter),
		Width:          style.Some(style.Percent(100)),
		Height:         style.Some(style.Percent(100)),
		Background:     style.Some[color.Color](color.RGBA{R: 15, G: 15, B: 20, A: 255}),
	})

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
