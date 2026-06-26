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
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

var (
	listHeaderStyle       = style.S().Margin(1, 0, 0, 0).Underline(true)
	titleStyle            = style.S().TextAlign(style.TextAlignCenter).Margin(0, 0, 1, 0).Bold(true)
	squareListStyle       = style.S().ListStyleType(style.ListStyleSquare)
	contentContainerStyle = style.S().Width(style.Percent(90)).Margin(1, 0).Padding(1, 2).Border(style.SingleBorder())
	rootStyle             = style.S().Width(style.Percent(100)).Height(style.Percent(100)).Padding(1, 2).Background(color.RGBA{R: 20, G: 20, B: 20, A: 255})
)

func main() {
	var b backend.Backend
	f, _ := os.Create("kite.log")
	defer f.Close()

	logger := slog.New(slog.NewTextHandler(f, nil))

	_ = logger // prevent unused variable error
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

	eng := engine.New(b, engine.Options{})

	headerStyle := listHeaderStyle

	root := element.Box(
		element.Box(
			element.Box("Kite List Components Demonstration").Style(titleStyle),

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
			).Style(squareListStyle),

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
		).Style(contentContainerStyle),
	).Style(rootStyle)

	eng.Mount(root)

	// Install devtools (Inspector + X-Ray)
	devtools.Install(eng, devtools.Options{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			if ke.MatchString("ctrl+c") || ke.MatchString("q") {
				cancel()
			}

			if ke.MatchString("f10") {
				eng.Dump("./example_dump.txt")
			}
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited with error: %v\n", err)
		os.Exit(1)
	}
}
