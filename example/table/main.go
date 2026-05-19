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

	// Build UI declaratively
	ui := element.Box(
		// Table 1: Well-formed
		element.Box("Well-formed Table").Style(style.Style{Margin: style.Some(style.Edges(1, 0))}),
		element.Table(
			element.TR(
				element.TD("Name").Style(style.Style{Width: style.Some(style.Cells(15))}),
				element.TD("Role").Style(style.Style{Width: style.Some(style.Cells(20))}),
			),
			element.TR(
				element.TD("Alice"),
				element.TD("Developer"),
			),
			element.TR(
				element.TD("Total Users: 1 (Spanning)").SetColSpan(2),
			),
		).Style(style.Style{
			Width:  style.Some(style.Percent(100)),
			Border: style.SingleBorder().Color(color.RGBA{R: 100, G: 100, B: 255, A: 255}).Some(),
		}),

		// Table 2: Malformed Table
		element.Box("Malformed Table (Cells without Rows)").Style(style.Style{Margin: style.Some(style.Edges(1, 0))}),
		element.Table(
			// Directly add cells to table
			element.TD("Direct Cell 1").Style(style.Style{Width: style.Some(style.Cells(15))}),
			element.TD("Direct Cell 2").Style(style.Style{Width: style.Some(style.Cells(20))}),
		).Style(style.Style{
			Width:  style.Some(style.Percent(100)),
			Border: style.SingleBorder().Color(color.RGBA{R: 255, G: 100, B: 100, A: 255}).Some(),
		}),

		// Table 3: Grouped Table (thead, tbody, tfoot)
		element.Box("Grouped Table (thead, tbody, tfoot)").Style(style.Style{Margin: style.Some(style.Edges(1, 0))}),
		element.Table(
			element.THead(
				element.TR(
					element.TD("Header Col 1").Style(style.Style{Width: style.Some(style.Cells(15))}),
					element.TD("Header Col 2").Style(style.Style{Width: style.Some(style.Cells(20))}),
				),
			).Style(style.Style{
				Border: style.SingleBorder().Top(false).Right(false).Left(false).Some(),
			}),
			element.TBody(
				element.TR(
					element.TD("Body Row 1, C1"),
					element.TD("Body Row 1, C2"),
				),
				element.TR(
					element.TD("Body Row 2, C1"),
					element.TD("Body Row 2, C2"),
				),
			),
			element.TFoot(
				element.TR(
					element.TD("Footer 1"),
					element.TD("Footer 2"),
				),
			).Style(style.Style{
				Border: style.SingleBorder().Bottom(false).Right(false).Left(false).Some(),
			}),
		).Style(style.Style{
			Width:  style.Some(style.Percent(100)),
			Border: style.SingleBorder().Color(color.RGBA{R: 100, G: 255, B: 100, A: 255}).Some(),
		}),
	).Style(style.Style{
		Display:       style.Some(style.DisplayFlex),
		FlexDirection: style.Some(style.FlexColumn),
		Width:         style.Some(style.Percent(100)),
		Height:        style.Some(style.Percent(100)),
		Background:    style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 30, A: 255}),
		Padding:       style.Some(style.Edges(2, 4)),
		Gap:           style.Some(style.Gap(2, 0)),
	})

	eng.Mount(ui)

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
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
