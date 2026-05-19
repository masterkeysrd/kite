package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	f, err := os.Create("table_test.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	be, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(be, engine.Options{
		Logger: logger,
	})

	doc := eng.Document()

	// Create a wrapper for body styles since body is a dom.Element
	container := element.NewBox(doc).Style(style.Style{
		Display:       style.Some(style.DisplayFlex),
		FlexDirection: style.Some(style.FlexColumn),
		Width:         style.Some(style.Percent(100)),
		Height:        style.Some(style.Percent(100)),
		Background:    style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 30, A: 255}),
		Padding:       style.Some(style.Edges(2, 4)),
		Gap:           style.Some(style.Gap(2, 0)),
	})

	// Table 1: Well-formed
	title1 := element.NewBox(doc).Style(style.Style{Margin: style.Some(style.Edges(1, 0))})
	title1.AppendChild(element.NewText(doc, "Well-formed Table"))
	container.AppendChild(title1)

	table1 := element.NewTable(doc).Style(style.Style{
		Width: style.Some(style.Percent(100)),
		Border: style.Some(style.Border{
			Width: style.Edges(1),
			Style: style.EdgeAll(style.BorderSingle),
			Color: style.EdgeAll[color.Color](color.RGBA{R: 100, G: 100, B: 255, A: 255}),
		}),
	})

	row1 := element.NewTableRow(doc)
	td11 := element.NewTableCell(doc).AddChild(element.NewText(doc, "Name")).Style(style.Style{Width: style.Some(style.Cells(15))})
	td12 := element.NewTableCell(doc).AddChild(element.NewText(doc, "Role")).Style(style.Style{Width: style.Some(style.Cells(20))})
	row1.AddChild(td11).AddChild(td12)

	row2 := element.NewTableRow(doc)
	td21 := element.NewTableCell(doc).AddChild(element.NewText(doc, "Alice"))
	td22 := element.NewTableCell(doc).AddChild(element.NewText(doc, "Developer"))
	row2.AddChild(td21).AddChild(td22)

	// ColSpan example
	row3 := element.NewTableRow(doc)
	td31 := element.NewTableCell(doc).AddChild(element.NewText(doc, "Total Users: 1 (Spanning)")).SetColSpan(2)
	row3.AddChild(td31)

	table1.AddChild(row1).AddChild(row2).AddChild(row3)
	container.AppendChild(table1)

	// Table 2: Malformed Table
	title2 := element.NewBox(doc).Style(style.Style{Margin: style.Some(style.Edges(1, 0))})
	title2.AppendChild(element.NewText(doc, "Malformed Table (Cells without Rows)"))
	container.AppendChild(title2)

	table2 := element.NewTable(doc).Style(style.Style{
		Width: style.Some(style.Percent(100)),
		Border: style.Some(style.Border{
			Width: style.Edges(1),
			Style: style.EdgeAll(style.BorderSingle),
			Color: style.EdgeAll[color.Color](color.RGBA{R: 255, G: 100, B: 100, A: 255}),
		}),
	})

	// Directly add cells to table
	table2.AddChild(element.NewTableCell(doc).AddChild(element.NewText(doc, "Direct Cell 1")).Style(style.Style{Width: style.Some(style.Cells(15))}))
	table2.AddChild(element.NewTableCell(doc).AddChild(element.NewText(doc, "Direct Cell 2")).Style(style.Style{Width: style.Some(style.Cells(20))}))

	container.AppendChild(table2)
	eng.Mount(container)

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doc.AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			if ke.MatchString("ctrl+c") || ke.MatchString("q") {
				cancel()
			}
		}
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	fmt.Println("Starting engine...")
	time.Sleep(1 * time.Second) // allow terminal to catch up
	eng.Run(ctx)
}
