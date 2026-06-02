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

var (
	headerStyle          = style.S().Width(style.Percent(100)).Margin(style.Edges(0, 0, 1, 0)).TextAlign(style.TextAlignCenter).Background(color.RGBA{R: 100, G: 0, B: 200, A: 255})
	highlightSpanStyle   = style.S().Background(color.RGBA{R: 255, G: 255, B: 255, A: 255}).Foreground(color.Black)
	atomicBoxStyle       = style.S().Display(style.DisplayInlineBlock).Width(style.Cells(10)).Height(style.Cells(3)).Background(color.RGBA{R: 0, G: 200, B: 100, A: 255}).Margin(style.Edges(0, 1)).Border(style.SingleBorder())
	paragraphStyle       = style.S().AlignItems(style.AlignCenter)
	featuresSectionStyle = style.S().Margin(style.Edges(1, 0)).Background(color.RGBA{R: 40, G: 40, B: 60, A: 255}).Padding(style.Edges(1))
	flexContainerStyle   = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).FlexWrap(style.FlexWrapOn).Width(style.Percent(100)).Margin(style.Edges(1, 0)).Padding(style.Edges(1)).Background(color.RGBA{R: 50, G: 50, B: 50, A: 255}).Gap(style.Gap(1, 2))
	col1Style            = style.S().Width(style.Percent(30))
	col2Style            = style.S().Width(style.Percent(70))
	tableStyle           = style.S().Width(style.Percent(100)).Border(style.SingleBorder())
	tableSectionStyle    = style.S().Margin(style.Edges(1, 0)).Padding(style.Edges(1)).Background(color.RGBA{R: 20, G: 60, B: 20, A: 255})
	contentWrapperStyle  = style.S().Width(style.Percent(80)).Height(style.Auto).Margin(style.Edges(1, 2)).Background(color.RGBA{R: 30, G: 30, B: 30, A: 255}).Border(style.SingleBorder().Color(color.RGBA{R: 200, G: 200, B: 200, A: 255})).Padding(style.Edges(1, 2))
	rootStyle            = style.S().Width(style.Percent(100)).Height(style.Percent(100)).Padding(style.Edges(2, 4)).Background(color.RGBA{R: 0, G: 0, B: 255, A: 255})
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
		Logger:   slog.Default(),
		Profiler: true,
	}
	eng := engine.New(b, opts)

	// Create items for flex section
	flexItems := make([]any, 0, 6)
	for i := 1; i <= 6; i++ {
		item := element.Box(fmt.Sprintf("Flex Item %d", i)).Style(style.S().Width(style.Cells(12)).Height(style.Cells(3)).Background(color.RGBA{R: uint8(40 * i), G: 100, B: 150, A: 255}).Border(style.SingleBorder()).Flex(style.Flex(1, 1, style.Cells(10))))
		flexItems = append(flexItems, item)
	}

	// Build UI declaratively
	root := element.Box(
		element.Box(
			// Title
			element.Box("Kite Layout Engine Test").Style(headerStyle),

			// Paragraph
			element.Box(
				"This is a demonstration of ",
				element.Span("inline elements").Style(highlightSpanStyle),
				" and ",
				element.Box("Atomic!").Style(atomicBoxStyle),
				" working together in a single flow.",
			).Style(paragraphStyle),

			// List Section
			element.Box(
				"Available Features:",
				element.UL(
					element.LI("Full LayoutNG engine"),
					element.LI("Interactive DOM components"),
					element.LI("Flexible styling system"),
				),
			).Style(featuresSectionStyle),

			// Flex Section
			element.Box(
				flexItems...,
			).Style(flexContainerStyle),

			// Table Section
			element.Box(
				"Grid Layout (Table):",
				element.Table(
					element.TR(
						element.TD("Header 1").Style(col1Style),
						element.TD("Header 2").Style(col2Style),
					),
					element.TR(
						element.TD("Row 1, Cell 1"),
						element.TD("Row 1, Cell 2"),
					),
				).Style(tableStyle),
			).Style(tableSectionStyle),
		).Style(contentWrapperStyle),
	).Style(rootStyle)

	// Attach root logical element to the engine
	eng.Mount(root)

	// Install devtools (Inspector + X-Ray)
	devtools.Install(eng, devtools.Options{})

	// Run the engine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Global quit listener
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
