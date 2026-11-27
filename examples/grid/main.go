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
	gridContainerStyle = style.S().
				Width(style.Percent(100)).
				Height(style.Percent(100)).
				Padding(1, 2).
				Background(style.RGB(15, 15, 15))

	titleStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.RGB(255, 255, 0))

	basicGridStyle = style.S().
			Display(style.DisplayGrid).
			GridTemplateColumns(style.Repeat(2, style.Fr(1))...).
			GridTemplateRows(style.Repeat(2, style.Fr(1))...).
			Height(style.Auto).
			Border(style.SingleBorder())

	footerHintStyle = style.S().Foreground(style.RGB(100, 100, 100))

	spanningItemStyle = style.S().
				GridColumn(style.Span(2)).
				Border(style.SingleBorder()).
				Background(style.RGB(150, 0, 150))

	spanningGridStyle = style.S().
				Display(style.DisplayGrid).
				Gap(1).
				GridTemplateColumns(style.Repeat(2, style.Fr(1))...).
				GridTemplateRows(style.Repeat(2, style.Fr(1))...).
				Height(style.Auto).
				Border(style.SingleBorder())

	autoPlacementGridStyle = style.S().
				Display(style.DisplayGrid).
				GridTemplateColumns(style.Repeat(3, style.Fr(1))...).
				GridColumnGap(4).
				GridRowGap(1).
				Height(style.Auto).
				Padding(1).
				Border(style.SingleBorder())

	mixedTracksGridStyle = style.S().
				Display(style.DisplayGrid).
				GridTemplateColumns(style.Cells(10), style.Auto, style.Fr(1)).
				Height(style.Auto).
				Border(style.SingleBorder())

	explicitCell1Style = style.S().
				GridColumn(style.GridLine(1)).
				GridRow(style.GridLine(1)).
				Border(style.SingleBorder()).
				Background(style.RGB(100, 0, 0))

	explicitCell2Style = style.S().
				GridColumn(style.GridLine(2)).
				GridRow(style.GridLine(2)).
				Border(style.SingleBorder()).
				Background(style.RGB(0, 100, 0))

	explicitCell3Style = style.S().
				GridColumn(style.GridLine(3)).
				GridRow(style.GridLine(3)).
				Border(style.SingleBorder()).
				Background(style.RGB(0, 0, 100))

	explicitGridStyle = style.S().
				Display(style.DisplayGrid).
				GridTemplateColumns(style.Repeat(3, style.Fr(1))...).
				GridTemplateRows(style.Repeat(3, style.Cells(3))...).
				Height(style.Auto).
				Border(style.SingleBorder())

	holyGrailHeaderFooterStyle = style.S().
					GridColumn(style.Span(3)).
					Border(style.SingleBorder()).
					Background(style.RGB(150, 150, 150)).
					Foreground(style.RGB(0, 0, 0))

	holyGrailSidebarStyle = style.S().
				Border(style.SingleBorder()).
				Background(style.RGB(100, 100, 100))

	holyGrailContentStyle = style.S().
				Border(style.SingleBorder()).
				Background(style.RGB(50, 50, 50))

	holyGrailGridStyle = style.S().
				Display(style.DisplayGrid).
				GridTemplateColumns(style.Cells(12), style.Fr(1), style.Cells(10)).
				GridTemplateRows(style.Cells(3), style.Fr(1), style.Cells(3)).
				Width(style.Percent(100)).
				Height(style.Cells(15)).
				Border(style.SingleBorder())
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
	return style.S().
		Width(style.Percent(100)).
		Height(style.Percent(100)).
		Border(style.SingleBorder()).
		Background(bg)
}

func viewBasicGrid() element.Element {
	return element.Box(
		element.Box("1. Basic 2x2 Grid (1fr tracks)").Style(titleStyle),
		element.Box(
			element.Box("1").Style(itemStyle(style.RGB(150, 0, 0))),
			element.Box("2").Style(itemStyle(style.RGB(0, 150, 0))),
			element.Box("3").Style(itemStyle(style.RGB(0, 0, 150))),
			element.Box("4").Style(itemStyle(style.RGB(150, 150, 0))),
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
			element.Box("3").Style(itemStyle(style.RGB(0, 150, 150))),
			element.Box("4").Style(itemStyle(style.RGB(150, 150, 150))),
		).Style(spanningGridStyle),
		element.Box("\nPress [SPACE] to see Auto Placement").Style(footerHintStyle),
	).Style(containerStyle())
}

func viewAutoPlacement() element.Element {
	return element.Box(
		element.Box("3. Auto Placement (with Gaps)").Style(titleStyle),
		element.Box(
			element.Box("A").Style(itemStyle(style.RGB(100, 50, 0))),
			element.Box("B").Style(itemStyle(style.RGB(0, 100, 50))),
			element.Box("C").Style(itemStyle(style.RGB(50, 0, 100))),
			element.Box("D").Style(itemStyle(style.RGB(100, 100, 100))),
			element.Box("E").Style(itemStyle(style.RGB(50, 50, 50))),
			element.Box("F").Style(itemStyle(style.RGB(0, 0, 0))),
		).Style(autoPlacementGridStyle),
		element.Box("\nPress [SPACE] to see Mixed Tracks").Style(footerHintStyle),
	).Style(containerStyle())
}

func viewMixedTracks() element.Element {
	return element.Box(
		element.Box("4. Mixed Tracks (Fixed, Auto, Fr)").Style(titleStyle),
		element.Box(
			element.Box("Fixed 10").Style(itemStyle(style.RGB(150, 0, 0))),
			element.Box("Auto (Shrink)").Style(itemStyle(style.RGB(0, 150, 0))),
			element.Box("1fr (Rest)").Style(itemStyle(style.RGB(0, 0, 150))),
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
