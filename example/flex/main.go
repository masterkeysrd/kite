package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/uv"
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
	b, err = uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	opts := engine.Options{
		Logger: slog.Default(),
	}
	eng := engine.New(b, opts)
	doc := eng.Document()

	root := element.NewBox(doc).Style(style.Style{
		Width:         style.Some(style.Percent(100)),
		Height:        style.Some(style.Percent(100)),
		Background:    style.Some[color.Color](color.RGBA{R: 15, G: 15, B: 15, A: 255}),
		Padding:       style.Some(style.Edges(1, 2)),
		FlexDirection: style.Some(style.FlexColumn),
		Display:       style.Some(style.DisplayFlex),
	})

	headerStyle := style.Style{
		Foreground: style.Some[color.Color](color.RGBA{R: 255, G: 255, B: 0, A: 255}),
		Margin:     style.Some(style.Edges(0, 0, 1, 0)),
	}

	// --- 1. Inline Flex Example ---
	root.AppendChild(element.NewBox(doc).Style(headerStyle).AppendChild(element.NewText(doc, "1. Inline Flex (Shrink-wrap content)")))

	inlineContainer := element.NewBox(doc).Style(style.Style{Margin: style.Some(style.Edges(0, 0, 2, 0))})
	inlineContainer.AppendChild(element.NewText(doc, "Text before -> "))

	inlineFlex := element.NewBox(doc).Style(style.Style{
		Display:    style.Some(style.DisplayInlineFlex),
		Background: style.Some[color.Color](color.RGBA{R: 0, G: 80, B: 150, A: 255}),
		Border:     style.Some(style.Border{Width: style.Edges(1), Style: style.EdgeAll(style.BorderSingle)}),
		Gap:        style.Some(style.Gap(0, 1)),
		Padding:    style.Some(style.Edges(0, 1)),
	})
	for i := 1; i <= 3; i++ {
		item := element.NewBox(doc).Style(style.Style{Background: style.Some[color.Color](color.RGBA{R: 150, G: 0, B: 0, A: 255}), Padding: style.Some(style.Edges(0, 1))})
		item.AppendChild(element.NewText(doc, fmt.Sprintf("Item %d", i)))
		inlineFlex.AppendChild(item)
	}
	inlineContainer.AppendChild(inlineFlex)
	inlineContainer.AppendChild(element.NewText(doc, " <- Text after"))
	root.AppendChild(inlineContainer)

	// --- 2. Flex Row Example ---
	root.AppendChild(element.NewBox(doc).Style(headerStyle).AppendChild(element.NewText(doc, "2. Flex Row (Justify: Space-Between, Align: Center)")))

	rowFlex := element.NewBox(doc).Style(style.Style{
		Display:        style.Some(style.DisplayFlex),
		FlexDirection:  style.Some(style.FlexRow),
		JustifyContent: style.Some(style.JustifyBetween),
		AlignItems:     style.Some(style.AlignCenter),
		Background:     style.Some[color.Color](color.RGBA{R: 40, G: 40, B: 40, A: 255}),
		Height:         style.Some(style.Cells(5)),
		Padding:        style.Some(style.Edges(0, 2)),
		Margin:         style.Some(style.Edges(0, 0, 2, 0)),
	})

	for i := 1; i <= 4; i++ {
		item := element.NewBox(doc).Style(style.Style{
			Background: style.Some[color.Color](color.RGBA{R: 0, G: 120, B: 0, A: 255}),
			Padding:    style.Some(style.Edges(0, 2)),
			Height:     style.Some(style.Cells(1 + i%2)), // Varying heights to show alignment
		})
		item.AppendChild(element.NewText(doc, fmt.Sprintf("Row Item %d", i)))
		rowFlex.AppendChild(item)
	}
	root.AppendChild(rowFlex)

	// --- 3. Flex Column Example ---
	root.AppendChild(element.NewBox(doc).Style(headerStyle).AppendChild(element.NewText(doc, "3. Flex Column (Align: End)")))

	colFlex := element.NewBox(doc).Style(style.Style{
		Display:       style.Some(style.DisplayFlex),
		FlexDirection: style.Some(style.FlexColumn),
		AlignItems:    style.Some(style.AlignEnd),
		Background:    style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 60, A: 255}),
		Width:         style.Some(style.Percent(50)),
		Padding:       style.Some(style.Edges(1, 2)),
		Gap:           style.Some(style.Gap(1, 0)),
	})

	for i := 1; i <= 3; i++ {
		item := element.NewBox(doc).Style(style.Style{
			Background: style.Some[color.Color](color.RGBA{R: 180, G: 80, B: 0, A: 255}),
			Padding:    style.Some(style.Edges(0, 2)),
			Width:      style.Some(style.Auto),
		})
		item.AppendChild(element.NewText(doc, fmt.Sprintf("Column Item %d (Stays Right)", i)))
		colFlex.AppendChild(item)
	}
	root.AppendChild(colFlex)

	eng.Mount(root)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doc.AddEventListener(event.EventKeyDown, func(e event.Event) {
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
