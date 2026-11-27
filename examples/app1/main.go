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
	headerStyle = style.S().
			Width(style.Percent(100)).
			Margin(0, 0, 1, 0).
			Background(style.RGB(100, 0, 200)).
			TextAlign(style.TextAlignCenter)

	highlightSpanStyle = style.S().
				Background(style.Hex("#ffffff")).
				Foreground(color.Black)

	atomicBoxStyle = style.S().
			Display(style.DisplayInlineBlock).
			Width(style.Cells(10)).
			Height(style.Cells(3)).
			Margin(0, 1).
			Border(style.SingleBorder()).
			Background(style.RGB(0, 200, 100))

	paragraphStyle = style.S().AlignItems(style.AlignCenter)

	featuresSectionStyle = style.S().
				Margin(1, 0).
				Padding(1).
				Background(style.RGB(40, 40, 60))

	flexContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				FlexWrap(style.FlexWrapOn).
				Gap(1, 2).
				Width(style.Percent(100)).
				Margin(1, 0).
				Padding(1).
				Background(style.RGB(50, 50, 50))

	col1Style = style.S().Width(style.Percent(30))
	col2Style = style.S().Width(style.Percent(70))

	tableStyle = style.S().
			Width(style.Percent(100)).
			Border(style.SingleBorder())

	tableSectionStyle = style.S().
				Margin(1, 0).
				Padding(1).
				Background(style.RGB(20, 60, 20))

	contentWrapperStyle = style.S().
				Width(style.Percent(80)).
				Height(style.Auto).
				Margin(1, 2).
				Padding(1, 2).
				Border(style.SingleBorder().Color(style.RGB(200, 200, 200))).
				Background(style.RGB(30, 30, 30))

	rootStyle = style.S().
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Padding(2, 4).
			Background(style.RGB(0, 0, 255))
)

func main() {
	var b backend.Backend
	f, err := os.Create("kite.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

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

	eng := engine.New(b, engine.Options{})

	// Create items for flex section
	flexItems := make([]any, 0, 6)
	for i := 1; i <= 6; i++ {
		item := element.Box(fmt.Sprintf("Flex Item %d", i)).Style(
			style.S().
				Flex(1, 1, style.Cells(10)).
				Width(style.Cells(12)).
				Height(style.Cells(3)).
				Border(style.SingleBorder()).
				Background(style.RGB(uint8(40*i), 100, 150)),
		)
		flexItems = append(flexItems, item)
	}

	// Build UI declaratively
	root := element.Box(
		element.Box(
			element.Box("Kite Layout Engine Test").Style(headerStyle),

			element.Box(
				"This is a demonstration of ",
				element.Span("inline elements").Style(highlightSpanStyle),
				" and ",
				element.Box("Atomic!").Style(atomicBoxStyle),
				" working together in a single flow.",
			).Style(paragraphStyle),

			element.Box(
				"Available Features:",
				element.UL(
					element.LI("Full LayoutNG engine"),
					element.LI("Interactive DOM components"),
					element.LI("Flexible styling system"),
				),
			).Style(featuresSectionStyle),

			element.Box(
				flexItems...,
			).Style(flexContainerStyle),

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

	eng.Mount(root)
	devtools.Install(eng, devtools.Options{})

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
