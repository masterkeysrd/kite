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
	headerStyle                  = style.S().Foreground(color.RGBA{R: 255, G: 255, B: 0, A: 255}).Margin(style.Edges(0, 0, 1, 0))
	inlineFlexItemStyle          = style.S().Background(color.RGBA{R: 150, G: 0, B: 0, A: 255}).Padding(style.Edges(0, 1))
	columnFlexItemStyle          = style.S().Background(color.RGBA{R: 180, G: 80, B: 0, A: 255}).Padding(style.Edges(0, 2)).Width(style.Auto)
	inlineFlexContainerStyle     = style.S().Display(style.DisplayInlineFlex).Background(color.RGBA{R: 0, G: 80, B: 150, A: 255}).Border(style.SingleBorder()).Gap(style.Gap(0, 1)).Padding(style.Edges(0, 1))
	inlineFlexWrapperStyle       = style.S().Margin(style.Edges(0, 0, 2, 0))
	flexRowContainerStyle        = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).JustifyContent(style.JustifyBetween).AlignItems(style.AlignCenter).Background(color.RGBA{R: 40, G: 40, B: 40, A: 255}).Height(style.Cells(5)).Padding(style.Edges(0, 2)).Margin(style.Edges(0, 0, 2, 0))
	flexColContainerStyle        = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).AlignItems(style.AlignEnd).Background(color.RGBA{R: 30, G: 30, B: 60, A: 255}).Width(style.Percent(50)).Padding(style.Edges(1, 2)).Gap(style.Gap(1, 0)).Margin(style.Edges(0, 0, 2, 0))
	reverseItemRedStyle          = style.S().Background(color.RGBA{200, 0, 0, 255}).Padding(style.Edges(0, 1))
	reverseItemGreenStyle        = style.S().Background(color.RGBA{0, 200, 0, 255}).Padding(style.Edges(0, 1))
	reverseItemBlueStyle         = style.S().Background(color.RGBA{0, 0, 200, 255}).Padding(style.Edges(0, 1))
	flexRowReverseContainerStyle = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRowReverse).Background(color.RGBA{R: 60, G: 30, B: 30, A: 255}).Padding(style.Edges(0, 2)).Margin(style.Edges(0, 0, 2, 0)).Gap(style.Gap(0, 2))
	orderItem1Style              = style.S().Background(color.RGBA{R: 200, G: 0, B: 0, A: 255}).Padding(style.Edges(0, 1)).Order(3).Flex(style.Flex(1)).Border(style.SingleBorder())
	orderItem2Style              = style.S().Background(color.RGBA{R: 0, G: 200, B: 0, A: 255}).Padding(style.Edges(0, 1)).Order(1).Flex(style.Flex(1)).Border(style.SingleBorder())
	orderItem3Style              = style.S().Background(color.RGBA{R: 0, G: 0, B: 200, A: 255}).Padding(style.Edges(0, 1)).Order(2).Flex(style.Flex(1)).Border(style.SingleBorder())
	flexOrderContainerStyle      = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).Background(color.RGBA{R: 30, G: 60, B: 30, A: 255}).Width(style.Percent(100)).Padding(style.Edges(0, 2)).Gap(style.Gap(2)).Border(style.SingleBorder())
	flexWrapContainerStyle         = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).FlexWrap(style.FlexWrapOn).Background(color.RGBA{R: 80, G: 30, B: 80, A: 255}).Padding(style.Edges(1, 2)).Gap(style.Gap(1, 1)).Margin(style.Edges(0, 0, 2, 0)).Width(style.Percent(100))
	flexWrapItemStyle              = style.S().Background(color.RGBA{R: 120, G: 60, B: 180, A: 255}).Padding(style.Edges(0, 2)).Border(style.SingleBorder())
	rootStyle                    = style.S().Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 15, G: 15, B: 15, A: 255}).Padding(style.Edges(1, 2)).FlexDirection(style.FlexColumn).Display(style.DisplayFlex)
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
		Logger:   slog.Default(),
		Profiler: true,
	}
	eng := engine.New(b, opts)

	headerStyle := headerStyle

	// Helper for creating inline flex items
	inlineFlexItems := make([]any, 0, 3)
	for i := 1; i <= 3; i++ {
		inlineFlexItems = append(inlineFlexItems,
			element.Box(fmt.Sprintf("Item %d", i)).Style(inlineFlexItemStyle),
		)
	}

	// Helper for creating row flex items
	rowFlexItems := make([]any, 0, 4)
	for i := 1; i <= 4; i++ {
		rowFlexItems = append(rowFlexItems,
			element.Box(fmt.Sprintf("Row Item %d", i)).Style(style.S().Background(color.RGBA{R: 0, G: 120, B: 0, A: 255}).Padding(style.Edges(0, 2)).Height(style.Cells(1+i%2))),
		)
	}

	// Helper for column items
	colFlexItems := make([]any, 0, 3)
	for i := 1; i <= 3; i++ {
		colFlexItems = append(colFlexItems,
			element.Box(fmt.Sprintf("Column Item %d (Stays Right)", i)).Style(columnFlexItemStyle),
		)
	}

	// Build UI declaratively
	root := element.Box(
		// 1. Inline Flex Example
		element.Box("1. Inline Flex (Shrink-wrap content)").Style(headerStyle),
		element.Box(
			"Text before -> ",
			element.Box(inlineFlexItems...).Style(inlineFlexContainerStyle),
			" <- Text after",
		).Style(inlineFlexWrapperStyle),

		// 2. Flex Row Example
		element.Box("2. Flex Row (Justify: Space-Between, Align: Center)").Style(headerStyle),
		element.Box(rowFlexItems...).Style(flexRowContainerStyle),

		// 3. Flex Column Example
		element.Box("3. Flex Column (Align: End)").Style(headerStyle),
		element.Box(colFlexItems...).Style(flexColContainerStyle),

		// 4. Flex Row Reverse Example
		element.Box("4. Flex Row Reverse").Style(headerStyle),
		element.Box(
			element.Box("Reverse Item 1").Style(reverseItemRedStyle),
			element.Box("Reverse Item 2").Style(reverseItemGreenStyle),
			element.Box("Reverse Item 3").Style(reverseItemBlueStyle),
		).Style(flexRowReverseContainerStyle),

		// 5. Flex Order Example
		element.Box("5. Flex Order Property").Style(headerStyle),
		element.Box(
			element.Box("First in DOM (Order 3)").Style(orderItem1Style),
			element.Box("Second in DOM (Order 1)").Style(orderItem2Style),
			element.Box("Third in DOM (Order 2)").Style(orderItem3Style),
		).Style(flexOrderContainerStyle),

		// 6. Flex Wrap Example
		element.Box("6. Flex Wrap (Wrap items onto multiple lines)").Style(headerStyle),
		element.Box(
			element.Box("Long Item 1").Style(flexWrapItemStyle),
			element.Box("Long Item 2").Style(flexWrapItemStyle),
			element.Box("Long Item 3").Style(flexWrapItemStyle),
			element.Box("Long Item 4").Style(flexWrapItemStyle),
			element.Box("Long Item 5").Style(flexWrapItemStyle),
			element.Box("Long Item 6").Style(flexWrapItemStyle),
		).Style(flexWrapContainerStyle),
	).Style(rootStyle)

	eng.Mount(root)

	// Install devtools (Inspector + X-Ray)
	devtools.Install(eng, devtools.Options{})

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
