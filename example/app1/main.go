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
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	var b backend.Backend
	f, er := os.Create("kite.log")
	if er != nil {
		fmt.Fprintf(os.Stderr, "failed to create log file: %v\n", er)
		os.Exit(1)
	}
	defer f.Close()

	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if os.Getenv("USE_MOCK_BACKEND") == "1" {
		slog.Info("Using mock backend")
		b = mock.New(80, 24)
	} else {

		slog.Info("Using UV backend")
		var err error
		b, err = uv.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
			os.Exit(1)
		}
	}

	if os.Getenv("AUTO_CLOSE") == "1" {
		go func() {
			slog.Info("Auto-close enabled, exiting in 5 seconds...")
			<-time.After(5 * time.Second)
			os.Exit(0)
		}()
	}

	// Initialize the rendering engine
	opts := engine.Options{
		Logger: slog.Default(),
	}
	eng := engine.New(b, opts)

	// Create the Document and root view container
	doc := eng.Document()

	// Create our app container
	root := element.NewBox(doc).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
		Padding:    style.Some(style.Edges(2, 4)),
		Background: style.Some[color.Color](color.RGBA{R: 0, G: 0, B: 255, A: 255}), // Blue background
	})

	// Create an inner box using LayoutNG border-box semantics
	inner := element.NewBox(doc).Style(style.Style{
		Width:      style.Some(style.Percent(80)),
		Height:     style.Some(style.Auto),
		Margin:     style.Some(style.Edges(1, 2)),
		Background: style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 30, A: 255}), // Dark background
		Border: style.Some(style.Border{
			Width: style.Edges(1),
			Style: style.EdgeAll(style.BorderSingle),
			Color: style.EdgeAll[color.Color](color.RGBA{R: 200, G: 200, B: 200, A: 255}), // Gray border
		}),
		Padding: style.Some(style.Edges(1, 2)),
	})

	title := element.NewBox(doc).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Margin:     style.Some(style.Edges(0, 0, 1, 0)),
		TextAlign:  style.Some(style.TextAlignCenter),
		Background: style.Some[color.Color](color.RGBA{R: 100, G: 0, B: 200, A: 255}),
	})
	title.AppendChild(element.NewText(doc, "Kite Layout Engine Test"))
	inner.AppendChild(title)

	paragraph := element.NewBox(doc).Style(style.Style{
		AlignItems: style.Some(style.AlignCenter), // Center children (text/inline-blocks) vertically
	})
	paragraph.AppendChild(element.NewText(doc, "This is a demonstration of "))

	boldSpan := element.NewSpan(doc).Style(style.Style{
		Display:    style.Some(style.DisplayInline),
		Background: style.Some[color.Color](color.RGBA{R: 255, G: 255, B: 255, A: 255}), // White background for emphasis
		Foreground: style.Some[color.Color](color.Black),                                // Black text
	})
	boldSpan.AppendChild(element.NewText(doc, "inline elements"))
	paragraph.AppendChild(boldSpan)

	paragraph.AppendChild(element.NewText(doc, " and "))

	inlineBlock := element.NewBox(doc).Style(style.Style{
		Display:    style.Some(style.DisplayInlineBlock),
		Width:      style.Some(style.Cells(10)),
		Height:     style.Some(style.Cells(3)),
		Background: style.Some[color.Color](color.RGBA{R: 0, G: 200, B: 100, A: 255}),
		Margin:     style.Some(style.Edges(0, 1)),
		Border: style.Some(style.Border{
			Width: style.Edges(1),
			Style: style.EdgeAll(style.BorderSingle),
		}),
	})
	inlineBlock.AppendChild(element.NewText(doc, "Atomic!"))
	paragraph.AppendChild(inlineBlock)

	paragraph.AppendChild(element.NewText(doc, " working together in a single flow."))
	inner.AppendChild(paragraph)

	// List Test Section
	listSection := element.NewBox(doc).Style(style.Style{
		Margin:     style.Some(style.Edges(1, 0)),
		Background: style.Some[color.Color](color.RGBA{R: 40, G: 40, B: 60, A: 255}),
		Padding:    style.Some(style.Edges(1)),
	})
	listSection.AppendChild(element.NewText(doc, "Available Features:"))
	ul := element.NewUnorderedList(doc)
	ul.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Full LayoutNG engine")))
	ul.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Interactive DOM components")))
	ul.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Flexible styling system")))
	listSection.AppendChild(ul)
	inner.AppendChild(listSection)

	// Flexbox Test Section
	flexSection := element.NewBox(doc).Style(style.Style{
		Display:       style.Some(style.DisplayFlex),
		FlexDirection: style.Some(style.FlexRow),
		FlexWrap:      style.Some(style.FlexWrapOn),
		Width:         style.Some(style.Percent(100)),
		Margin:        style.Some(style.Edges(1, 0)),
		Padding:       style.Some(style.Edges(1)),
		Background:    style.Some[color.Color](color.RGBA{R: 50, G: 50, B: 50, A: 255}),
		Gap:           style.Some(style.Gap(1, 2)),
	})

	for i := 1; i <= 6; i++ {
		item := element.NewBox(doc).Style(style.Style{
			Width:      style.Some(style.Cells(12)),
			Height:     style.Some(style.Cells(3)),
			Background: style.Some[color.Color](color.RGBA{R: uint8(40 * i), G: 100, B: 150, A: 255}),
			Border: style.Some(style.Border{
				Width: style.Edges(1),
				Style: style.EdgeAll(style.BorderSingle),
			}),
			Flex: style.Some(style.Flex(1, 1, style.Cells(10))),
		})
		item.AppendChild(element.NewText(doc, fmt.Sprintf("Flex Item %d", i)))
		flexSection.AppendChild(item)
	}
	inner.AppendChild(flexSection)

	// Table Test Section
	tableSection := element.NewBox(doc).Style(style.Style{
		Margin:     style.Some(style.Edges(1, 0)),
		Padding:    style.Some(style.Edges(1)),
		Background: style.Some[color.Color](color.RGBA{R: 20, G: 60, B: 20, A: 255}),
	})
	tableSection.AppendChild(element.NewText(doc, "Grid Layout (Table):"))
	table := element.NewTable(doc).Style(style.Style{
		Width: style.Some(style.Percent(100)),
		Border: style.Some(style.Border{
			Width: style.Edges(1),
			Style: style.EdgeAll(style.BorderSingle),
		}),
	})

	row1 := element.NewTableRow(doc)
	row1.AddChild(element.NewTableCell(doc).AddChild(element.NewText(doc, "Header 1")).Style(style.Style{Width: style.Some(style.Percent(30))}))
	row1.AddChild(element.NewTableCell(doc).AddChild(element.NewText(doc, "Header 2")).Style(style.Style{Width: style.Some(style.Percent(70))}))

	row2 := element.NewTableRow(doc)
	row2.AddChild(element.NewTableCell(doc).AddChild(element.NewText(doc, "Row 1, Cell 1")))
	row2.AddChild(element.NewTableCell(doc).AddChild(element.NewText(doc, "Row 1, Cell 2")))

	table.AddChild(row1).AddChild(row2)
	tableSection.AppendChild(table)
	inner.AppendChild(tableSection)

	// Add inner to root
	root.AppendChild(inner)

	// Attach root logical element to the engine
	eng.Mount(root)

	// Add global quit listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doc.AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			if ke.MatchString("ctrl+c") || ke.MatchString("q") {
				cancel()
			}
		}
	})

	// Run the engine
	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited with error: %v\n", err)
		os.Exit(1)
	}
}
