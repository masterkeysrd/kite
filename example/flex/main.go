package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"time"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	f, er := os.Create("kite.log")
	if er != nil {
		fmt.Fprintf(os.Stderr, "failed to create log file: %v\n", er)
		os.Exit(1)
	}
	defer f.Close()

	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	var b backend.Backend
	var err error
	if os.Getenv("USE_MOCK_BACKEND") == "1" {
		b = mock.New(80, 24)
	} else {
		b, err = uv.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
			os.Exit(1)
		}
	}

	if os.Getenv("AUTO_CLOSE") == "1" {
		go func() {
			<-time.After(5 * time.Second)
			os.Exit(0)
		}()
	}

	opts := engine.Options{
		Logger: slog.Default(),
	}
	eng := engine.New(b, opts)

	headerStyle := style.Style{
		Foreground: style.Some[color.Color](color.RGBA{R: 255, G: 255, B: 0, A: 255}),
		Margin:     style.Some(style.Edges(0, 0, 1, 0)),
	}

	// Helper for creating inline flex items
	inlineFlexItems := make([]any, 0, 3)
	for i := 1; i <= 3; i++ {
		inlineFlexItems = append(inlineFlexItems,
			element.Box(fmt.Sprintf("Item %d", i)).Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 150, G: 0, B: 0, A: 255}),
				Padding:    style.Some(style.Edges(0, 1)),
			}),
		)
	}

	// Helper for creating row flex items
	rowFlexItems := make([]any, 0, 4)
	for i := 1; i <= 4; i++ {
		rowFlexItems = append(rowFlexItems,
			element.Box(fmt.Sprintf("Row Item %d", i)).Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 0, G: 120, B: 0, A: 255}),
				Padding:    style.Some(style.Edges(0, 2)),
				Height:     style.Some(style.Cells(1 + i%2)),
			}),
		)
	}

	// Helper for column items
	colFlexItems := make([]any, 0, 3)
	for i := 1; i <= 3; i++ {
		colFlexItems = append(colFlexItems,
			element.Box(fmt.Sprintf("Column Item %d (Stays Right)", i)).Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 180, G: 80, B: 0, A: 255}),
				Padding:    style.Some(style.Edges(0, 2)),
				Width:      style.Some(style.Auto),
			}),
		)
	}

	// Build UI declaratively
	root := element.Box(
		// 1. Inline Flex Example
		element.Box("1. Inline Flex (Shrink-wrap content)").Style(headerStyle),
		element.Box(
			"Text before -> ",
			element.Box(inlineFlexItems...).Style(style.Style{
				Display:    style.Some(style.DisplayInlineFlex),
				Background: style.Some[color.Color](color.RGBA{R: 0, G: 80, B: 150, A: 255}),
				Border:     style.SingleBorder().Some(),
				Gap:        style.Some(style.Gap(0, 1)),
				Padding:    style.Some(style.Edges(0, 1)),
			}),
			" <- Text after",
		).Style(style.Style{Margin: style.Some(style.Edges(0, 0, 2, 0))}),

		// 2. Flex Row Example
		element.Box("2. Flex Row (Justify: Space-Between, Align: Center)").Style(headerStyle),
		element.Box(rowFlexItems...).Style(style.Style{
			Display:        style.Some(style.DisplayFlex),
			FlexDirection:  style.Some(style.FlexRow),
			JustifyContent: style.Some(style.JustifyBetween),
			AlignItems:     style.Some(style.AlignCenter),
			Background:     style.Some[color.Color](color.RGBA{R: 40, G: 40, B: 40, A: 255}),
			Height:         style.Some(style.Cells(5)),
			Padding:        style.Some(style.Edges(0, 2)),
			Margin:         style.Some(style.Edges(0, 0, 2, 0)),
		}),

		// 3. Flex Column Example
		element.Box("3. Flex Column (Align: End)").Style(headerStyle),
		element.Box(colFlexItems...).Style(style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexColumn),
			AlignItems:    style.Some(style.AlignEnd),
			Background:    style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 60, A: 255}),
			Width:         style.Some(style.Percent(50)),
			Padding:       style.Some(style.Edges(1, 2)),
			Gap:           style.Some(style.Gap(1, 0)),
			Margin:        style.Some(style.Edges(0, 0, 2, 0)),
		}),

		// 4. Flex Row Reverse Example
		element.Box("4. Flex Row Reverse").Style(headerStyle),
		element.Box(
			element.Box("Reverse Item 1").Style(style.Style{Background: style.Some[color.Color](color.RGBA{200, 0, 0, 255}), Padding: style.Some(style.Edges(0, 1))}),
			element.Box("Reverse Item 2").Style(style.Style{Background: style.Some[color.Color](color.RGBA{0, 200, 0, 255}), Padding: style.Some(style.Edges(0, 1))}),
			element.Box("Reverse Item 3").Style(style.Style{Background: style.Some[color.Color](color.RGBA{0, 0, 200, 255}), Padding: style.Some(style.Edges(0, 1))}),
		).Style(style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexRowReverse),
			Background:    style.Some[color.Color](color.RGBA{R: 60, G: 30, B: 30, A: 255}),
			Padding:       style.Some(style.Edges(0, 2)),
			Margin:        style.Some(style.Edges(0, 0, 2, 0)),
			Gap:           style.Some(style.Gap(0, 2)),
		}),

		// 5. Flex Order Example
		element.Box("5. Flex Order Property").Style(headerStyle),
		element.Box(
			element.Box("First in DOM (Order 3)").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 200, G: 0, B: 0, A: 255}),
				Padding:    style.Some(style.Edges(0, 1)),
				Order:      style.Some(3),
				Flex:       style.Some(style.Flex(1)),
				Border:     style.SingleBorder().Some(),
			}),
			element.Box("Second in DOM (Order 1)").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 0, G: 200, B: 0, A: 255}),
				Padding:    style.Some(style.Edges(0, 1)),
				Order:      style.Some(1),
				Flex:       style.Some(style.Flex(1)),
				Border:     style.SingleBorder().Some(),
			}),
			element.Box("Third in DOM (Order 2)").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 0, G: 0, B: 200, A: 255}),
				Padding:    style.Some(style.Edges(0, 1)),
				Order:      style.Some(2),
				Flex:       style.Some(style.Flex(1)),
				Border:     style.SingleBorder().Some(),
			}),
		).Style(style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexRow),
			Background:    style.Some[color.Color](color.RGBA{R: 30, G: 60, B: 30, A: 255}),
			Width:         style.Some(style.Percent(100)),
			Padding:       style.Some(style.Edges(0, 2)),
			Gap:           style.Some(style.Gap(2)),
			Border:        style.SingleBorder().Some(),
		}),
	).Style(style.Style{
		Width:         style.Some(style.Percent(100)),
		Height:        style.Some(style.Percent(100)),
		Background:    style.Some[color.Color](color.RGBA{R: 15, G: 15, B: 15, A: 255}),
		Padding:       style.Some(style.Edges(1, 2)),
		FlexDirection: style.Some(style.FlexColumn),
		Display:       style.Some(style.DisplayFlex),
	})

	eng.Mount(root)

	// Install devtools (Inspector + X-Ray)
	devtools.Install(eng, devtools.Options{
	})

	// Global quit listener

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
