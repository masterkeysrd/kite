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
	doc := eng.Document()

	root := element.NewBox(doc).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
		Padding:    style.Some(style.Edges(1, 2)),
		Background: style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 20, A: 255}),
	})

	container := element.NewBox(doc).Style(style.Style{
		Width:   style.Some(style.Percent(90)),
		Margin:  style.Some(style.Edges(1, 0)),
		Padding: style.Some(style.Edges(1, 2)),
		Border: style.Some(style.Border{
			Width: style.Edges(1),
			Style: style.EdgeAll(style.BorderSingle),
		}),
	})

	title := element.NewBox(doc).Style(style.Style{
		TextAlign: style.Some(style.TextAlignCenter),
		Margin:    style.Some(style.Edges(0, 0, 1, 0)),
		Bold:      style.Some(true),
	})
	title.AddChild(element.NewText(doc, "Kite List Components Demonstration"))
	container.AddChild(title)

	// 1. Unordered List (Disc)
	ulHeader := element.NewBox(doc).Style(style.Style{Margin: style.Some(style.Edges(1, 0, 0, 0)), Underline: style.Some(true)})
	ulHeader.AddChild(element.NewText(doc, "Unordered List (Default: Disc)"))
	container.AddChild(ulHeader)

	ul := element.NewUnorderedList(doc)
	ul.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "First item")))
	ul.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Second item with long text that should wrap around the marker correctly if the container is narrow enough.")))
	ul.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Third item")))
	container.AddChild(ul)

	// 2. Ordered List (Decimal)
	olHeader := element.NewBox(doc).Style(style.Style{Margin: style.Some(style.Edges(1, 0, 0, 0)), Underline: style.Some(true)})
	olHeader.AddChild(element.NewText(doc, "Ordered List (Default: Decimal)"))
	container.AddChild(olHeader)

	ol := element.NewOrderedList(doc)
	ol.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Initialize engine")))
	ol.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Build DOM tree")))
	ol.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Run frame loop")))
	container.AddChild(ol)

	// 3. Custom Markers
	customHeader := element.NewBox(doc).Style(style.Style{Margin: style.Some(style.Edges(1, 0, 0, 0)), Underline: style.Some(true)})
	customHeader.AddChild(element.NewText(doc, "Custom Markers (Square)"))
	container.AddChild(customHeader)

	customList := element.NewUnorderedList(doc).Style(style.Style{
		ListStyleType: style.Some(style.ListStyleSquare),
	})
	customList.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Customized UL")))
	customList.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Uses Square markers via inheritance")))
	container.AddChild(customList)

	// 4. Nested Lists
	nestedHeader := element.NewBox(doc).Style(style.Style{Margin: style.Some(style.Edges(1, 0, 0, 0)), Underline: style.Some(true)})
	nestedHeader.AddChild(element.NewText(doc, "Nested Lists"))
	container.AddChild(nestedHeader)

	parentUl := element.NewUnorderedList(doc)
	liWithNested := element.NewListItem(doc)
	liWithNested.AddChild(element.NewText(doc, "Item with nested list:"))

	childOl := element.NewOrderedList(doc)
	childOl.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Nested Step A")))
	childOl.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Nested Step B")))

	liWithNested.AddChild(childOl)
	parentUl.AddChild(liWithNested)
	parentUl.AddChild(element.NewListItem(doc).AddChild(element.NewText(doc, "Another parent item")))

	container.AddChild(parentUl)

	root.AddChild(container)
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
