package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	var b backend.Backend
	f, _ := os.Create("kite.log")
	defer f.Close()

	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	if os.Getenv("USE_MOCK_BACKEND") == "1" {
		b = mock.New(80, 24)
	} else {
		var err error
		b, err = uv.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
			os.Exit(1)
		}
	}

	eng := engine.New(b, engine.Options{Logger: logger})

	headerStyle := style.Style{Margin: style.Some(style.Edges(1, 0, 0, 0)), Underline: style.Some(true)}

	root := element.Box(
		element.Box(
			element.Box("Kite List Components Demonstration").Style(style.Style{
				TextAlign: style.Some(style.TextAlignCenter),
				Margin:    style.Some(style.Edges(0, 0, 1, 0)),
				Bold:      style.Some(true),
			}),

			// 1. Unordered List (Disc)
			element.Box("Unordered List (Default: Disc)").Style(headerStyle),
			element.UL(
				element.LI("First item"),
				element.LI("Second item with long text that should wrap around the marker correctly if the container is narrow enough."),
				element.LI("Third item"),
			),

			// 2. Ordered List (Decimal)
			element.Box("Ordered List (Default: Decimal)").Style(headerStyle),
			element.OL(
				element.LI("Initialize engine"),
				element.LI("Build DOM tree"),
				element.LI("Run frame loop"),
			),

			// 3. Custom Markers
			element.Box("Custom Markers (Square)").Style(headerStyle),
			element.UL(
				element.LI("Customized UL"),
				element.LI("Uses Square markers via inheritance"),
			).Style(style.Style{
				ListStyleType: style.Some(style.ListStyleSquare),
			}),

			// 4. Nested Lists
			element.Box("Nested Lists").Style(headerStyle),
			element.UL(
				element.LI(
					"Item with nested list:",
					element.OL(
						element.LI("Nested Step A"),
						element.LI("Nested Step B"),
					),
				),
				element.LI("Another parent item"),
			),
		).Style(style.Style{
			Width:   style.Some(style.Percent(90)),
			Margin:  style.Some(style.Edges(1, 0)),
			Padding: style.Some(style.Edges(1, 2)),
			Border: style.Some(style.Border{
				Width: style.Edges(1),
				Style: style.EdgeAll(style.BorderSingle),
			}),
		}),
	).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
		Padding:    style.Some(style.Edges(1, 2)),
		Background: style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 20, A: 255}),
	})

	eng.Mount(root)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			if ke.MatchString("ctrl+c") || ke.MatchString("q") {
				cancel()
			}
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited with error: %v\n", err)
		os.Exit(1)
	}
}
