package main

import (
	"context"
	"fmt"
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
			Margin(0, 0, 1, 0).
			Foreground(style.RGB(255, 255, 0))

	inlineFlexItemStyle = style.S().
				Padding(0, 1).
				Background(style.RGB(150, 0, 0))

	columnFlexItemStyle = style.S().
				Width(style.Auto).
				Padding(0, 2).
				Background(style.RGB(180, 80, 0))

	inlineFlexContainerStyle = style.S().
					Display(style.DisplayInlineFlex).
					Gap(0, 1).
					Padding(0, 1).
					Border(style.SingleBorder()).
					Background(style.RGB(0, 80, 150))

	inlineFlexWrapperStyle = style.S().Margin(0, 0, 2, 0)

	flexRowContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				JustifyContent(style.JustifyBetween).
				AlignItems(style.AlignCenter).
				Height(style.Cells(5)).
				Margin(0, 0, 2, 0).
				Padding(0, 2).
				Background(style.RGB(40, 40, 40))

	flexColContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				AlignItems(style.AlignEnd).
				Gap(1, 0).
				Width(style.Percent(50)).
				Margin(0, 0, 2, 0).
				Padding(1, 2).
				Background(style.RGB(30, 30, 60))

	reverseItemRedStyle = style.S().
				Padding(0, 1).
				Background(style.RGB(200, 0, 0))

	reverseItemGreenStyle = style.S().
				Padding(0, 1).
				Background(style.RGB(0, 200, 0))

	reverseItemBlueStyle = style.S().
				Padding(0, 1).
				Background(style.RGB(0, 0, 200))

	flexRowReverseContainerStyle = style.S().
					Display(style.DisplayFlex).
					FlexDirection(style.FlexRowReverse).
					Gap(0, 2).
					Margin(0, 0, 2, 0).
					Padding(0, 2).
					Background(style.RGB(60, 30, 30))

	orderItem1Style = style.S().
			Flex(1).
			Order(3).
			Padding(0, 1).
			Border(style.SingleBorder()).
			Background(style.RGB(200, 0, 0))

	orderItem2Style = style.S().
			Flex(1).
			Order(1).
			Padding(0, 1).
			Border(style.SingleBorder()).
			Background(style.RGB(0, 200, 0))

	orderItem3Style = style.S().
			Flex(1).
			Order(2).
			Padding(0, 1).
			Border(style.SingleBorder()).
			Background(style.RGB(0, 0, 200))

	flexOrderContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				Gap(2).
				Width(style.Percent(100)).
				Padding(0, 2).
				Border(style.SingleBorder()).
				Background(style.RGB(30, 60, 30))

	flexWrapContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				FlexWrap(style.FlexWrapOn).
				Gap(1, 1).
				Width(style.Percent(100)).
				Margin(0, 0, 2, 0).
				Padding(1, 2).
				Background(style.RGB(80, 30, 80))

	flexWrapItemStyle = style.S().
				Padding(0, 2).
				Border(style.SingleBorder()).
				Background(style.RGB(120, 60, 180))

	rootStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Padding(1, 2).
			Background(style.RGB(15, 15, 15))
)

func main() {
	f, er := os.Create("kite.log")
	if er != nil {
		fmt.Fprintf(os.Stderr, "failed to create log file: %v\n", er)
		os.Exit(1)
	}
	defer f.Close()

	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))

	_ = logger // prevent unused variable error
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

	opts := engine.Options{}
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
			element.Box(fmt.Sprintf("Row Item %d", i)).Style(
				style.S().
					Height(style.Cells(1+i%2)).
					Padding(0, 2).
					Background(style.RGB(0, 120, 0)),
			),
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
