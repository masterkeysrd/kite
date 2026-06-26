package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

var (
	gridContainerStyle         = style.S().Width(style.Percent(100)).Height(style.Percent(100)).Padding(1, 2).Background(color.RGBA{15, 15, 15, 255})
	titleStyle                 = style.S().Foreground(color.RGBA{255, 255, 0, 255}).Margin(0, 0, 1, 0)
	basicGridStyle             = style.S().Display(style.DisplayGrid).GridTemplateColumns(style.Repeat(2, style.Fr(1))...).GridTemplateRows(style.Repeat(2, style.Fr(1))...).Height(style.Auto).Border(style.SingleBorder())
	footerHintStyle            = style.S().Foreground(color.RGBA{100, 100, 100, 255})
	spanningItemStyle          = style.S().Background(color.RGBA{150, 0, 150, 255}).Border(style.SingleBorder()).GridColumn(style.GridPlacement{Span: 2})
	spanningGridStyle          = style.S().Display(style.DisplayGrid).GridTemplateColumns(style.Repeat(2, style.Fr(1))...).GridTemplateRows(style.Repeat(2, style.Fr(1))...).Height(style.Auto).Border(style.SingleBorder()).Gap(1)
	autoPlacementGridStyle     = style.S().Display(style.DisplayGrid).GridTemplateColumns(style.Repeat(3, style.Fr(1))...).GridColumnGap(4).GridRowGap(1).Height(style.Auto).Border(style.SingleBorder()).Padding(1)
	mixedTracksGridStyle       = style.S().Display(style.DisplayGrid).GridTemplateColumns(style.Cells(10), style.Auto, style.Fr(1)).Height(style.Auto).Border(style.SingleBorder())
	explicitCell1Style         = style.S().Background(color.RGBA{100, 0, 0, 255}).Border(style.SingleBorder()).GridColumn(style.GridPlacement{Start: 1}).GridRow(style.GridPlacement{Start: 1})
	explicitCell2Style         = style.S().Background(color.RGBA{0, 100, 0, 255}).Border(style.SingleBorder()).GridColumn(style.GridPlacement{Start: 2}).GridRow(style.GridPlacement{Start: 2})
	explicitCell3Style         = style.S().Background(color.RGBA{0, 0, 100, 255}).Border(style.SingleBorder()).GridColumn(style.GridPlacement{Start: 3}).GridRow(style.GridPlacement{Start: 3})
	explicitGridStyle          = style.S().Display(style.DisplayGrid).GridTemplateColumns(style.Repeat(3, style.Fr(1))...).GridTemplateRows(style.Repeat(3, style.Cells(3))...).Height(style.Auto).Border(style.SingleBorder())
	holyGrailHeaderFooterStyle = style.S().Background(color.RGBA{150, 150, 150, 255}).Foreground(color.RGBA{0, 0, 0, 255}).Border(style.SingleBorder()).GridColumn(style.GridPlacement{Span: 3})
	holyGrailSidebarStyle      = style.S().Background(color.RGBA{100, 100, 100, 255}).Border(style.SingleBorder())
	holyGrailContentStyle      = style.S().Background(color.RGBA{50, 50, 50, 255}).Border(style.SingleBorder())
	holyGrailGridStyle         = style.S().Display(style.DisplayGrid).GridTemplateColumns(style.Cells(12), style.Fr(1), style.Cells(10)).GridTemplateRows(style.Cells(3), style.Fr(1), style.Cells(3)).Width(style.Percent(100)).Height(style.Cells(15)).Border(style.SingleBorder())
)

func main() {
	f, _ := os.Create("kite.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	_ = logger // prevent unused variable error
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{})

	var currentView int
	views := []func() element.Element{
		viewBasicGrid,
		viewSpanningGrid,
		viewAutoPlacement,
		viewMixedTracks,
		viewExplicitCoordinates,
		viewHolyGrail,
	}

	updateRoot := func() {
		eng.Mount(views[currentView]())
	}

	updateRoot()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("ctrl+c") || ke.MatchString("q") {
			cancel()
		} else if ke.MatchString("n") || ke.MatchString(" ") {
			currentView = (currentView + 1) % len(views)
			updateRoot()
		}
	})

	devtools.Install(eng, devtools.Options{})

	fmt.Println("Starting Grid Example...")
	fmt.Println("Press [SPACE] or [N] to switch views, [Q] to quit.")

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited with error: %v\n", err)
		os.Exit(1)
	}
}

func containerStyle() style.Style {
	return gridContainerStyle
}

func itemStyle(bg color.Color) style.Style {
	return style.S().Background(bg).Border(style.SingleBorder()).Width(style.Percent(100)).Height(style.Percent(100))
}

func viewBasicGrid() element.Element {
	return element.Box(
		element.Box("1. Basic 2x2 Grid (1fr tracks)").Style(titleStyle),
		element.Box(
			element.Box("1").Style(itemStyle(color.RGBA{150, 0, 0, 255})),
			element.Box("2").Style(itemStyle(color.RGBA{0, 150, 0, 255})),
			element.Box("3").Style(itemStyle(color.RGBA{0, 0, 150, 255})),
			element.Box("4").Style(itemStyle(color.RGBA{150, 150, 0, 255})),
		).Style(basicGridStyle),
		element.Box("\nPress [SPACE] to see Spanning Grid").Style(footerHintStyle),
	).Style(containerStyle())
}

func viewSpanningGrid() element.Element {
	return element.Box(
		element.Box("2. Spanning Grid").Style(titleStyle),
		element.Box(
			// Item 1 spans 2 columns
			element.Box("Spans 2 Cols").Style(spanningItemStyle),
			element.Box("3").Style(itemStyle(color.RGBA{0, 150, 150, 255})),
			element.Box("4").Style(itemStyle(color.RGBA{150, 150, 150, 255})),
		).Style(spanningGridStyle),
		element.Box("\nPress [SPACE] to see Auto Placement").Style(footerHintStyle),
	).Style(containerStyle())
}

func viewAutoPlacement() element.Element {
	return element.Box(
		element.Box("3. Auto Placement (with Gaps)").Style(titleStyle),
		element.Box(
			element.Box("A").Style(itemStyle(color.RGBA{100, 50, 0, 255})),
			element.Box("B").Style(itemStyle(color.RGBA{0, 100, 50, 255})),
			element.Box("C").Style(itemStyle(color.RGBA{50, 0, 100, 255})),
			element.Box("D").Style(itemStyle(color.RGBA{100, 100, 100, 255})),
			element.Box("E").Style(itemStyle(color.RGBA{50, 50, 50, 255})),
			element.Box("F").Style(itemStyle(color.RGBA{0, 0, 0, 255})),
		).Style(autoPlacementGridStyle),
		element.Box("\nPress [SPACE] to see Mixed Tracks").Style(footerHintStyle),
	).Style(containerStyle())
}

func viewMixedTracks() element.Element {
	return element.Box(
		element.Box("4. Mixed Tracks (Fixed, Auto, Fr)").Style(titleStyle),
		element.Box(
			element.Box("Fixed 10").Style(itemStyle(color.RGBA{150, 0, 0, 255})),
			element.Box("Auto (Shrink)").Style(itemStyle(color.RGBA{0, 150, 0, 255})),
			element.Box("1fr (Rest)").Style(itemStyle(color.RGBA{0, 0, 150, 255})),
		).Style(mixedTracksGridStyle),
		element.Box("\nPress [SPACE] to see Explicit Coordinates").Style(footerHintStyle),
	).Style(containerStyle())
}

func viewExplicitCoordinates() element.Element {
	return element.Box(
		element.Box("5. Explicit Coordinates").Style(titleStyle),
		element.Box(
			element.Box("1,1").Style(explicitCell1Style),
			element.Box("2,2").Style(explicitCell2Style),
			element.Box("3,3").Style(explicitCell3Style),
		).Style(explicitGridStyle),
		element.Box("\nPress [SPACE] to see Holy Grail Layout").Style(footerHintStyle),
	).Style(containerStyle())
}

func viewHolyGrail() element.Element {
	return element.Box(
		element.Box("6. Holy Grail Layout").Style(titleStyle),
		element.Box(
			element.Box("HEADER").Style(holyGrailHeaderFooterStyle),
			element.Box("SIDEBAR").Style(holyGrailSidebarStyle),
			element.Box("CONTENT").Style(holyGrailContentStyle),
			element.Box("RIGHT").Style(holyGrailSidebarStyle),
			element.Box("FOOTER").Style(holyGrailHeaderFooterStyle),
		).Style(holyGrailGridStyle),
		element.Box("\nPress [SPACE] to return to Basic Grid").Style(footerHintStyle),
	).Style(containerStyle())
}
