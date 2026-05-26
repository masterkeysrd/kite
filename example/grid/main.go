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

func main() {
	f, _ := os.Create("kite.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{Logger: logger})

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
	return style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
		Padding:    style.Some(style.Edges(1, 2)),
		Background: style.Some[color.Color](color.RGBA{15, 15, 15, 255}),
	}
}

func itemStyle(bg color.Color) style.Style {
	return style.Style{
		Background: style.Some(bg),
		Border:     style.SingleBorder().Some(),
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
	}
}

func viewBasicGrid() element.Element {
	return element.Box(
		element.Box("1. Basic 2x2 Grid (1fr tracks)").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{255, 255, 0, 255}), Margin: style.Some(style.Edges(0, 0, 1, 0))}),
		element.Box(
			element.Box("1").Style(itemStyle(color.RGBA{150, 0, 0, 255})),
			element.Box("2").Style(itemStyle(color.RGBA{0, 150, 0, 255})),
			element.Box("3").Style(itemStyle(color.RGBA{0, 0, 150, 255})),
			element.Box("4").Style(itemStyle(color.RGBA{150, 150, 0, 255})),
		).Style(style.Style{
			Display:             style.Some(style.DisplayGrid),
			GridTemplateColumns: style.Some(style.Repeat(2, style.Fr(1))),
			GridTemplateRows:    style.Some(style.Repeat(2, style.Fr(1))),
			Height:              style.Some(style.Auto),
			Border:              style.SingleBorder().Some(),
		}),
		element.Box("\nPress [SPACE] to see Spanning Grid").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{100, 100, 100, 255})}),
	).Style(containerStyle())
}

func viewSpanningGrid() element.Element {
	return element.Box(
		element.Box("2. Spanning Grid").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{255, 255, 0, 255}), Margin: style.Some(style.Edges(0, 0, 1, 0))}),
		element.Box(
			// Item 1 spans 2 columns
			element.Box("Spans 2 Cols").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{150, 0, 150, 255}),
				Border:     style.SingleBorder().Some(),
				GridColumn: style.Some(style.GridPlacement{Span: 2}),
			}),
			element.Box("3").Style(itemStyle(color.RGBA{0, 150, 150, 255})),
			element.Box("4").Style(itemStyle(color.RGBA{150, 150, 150, 255})),
		).Style(style.Style{
			Display:             style.Some(style.DisplayGrid),
			GridTemplateColumns: style.Some(style.Repeat(2, style.Fr(1))),
			GridTemplateRows:    style.Some(style.Repeat(2, style.Fr(1))),
			Height:              style.Some(style.Auto),
			Border:              style.SingleBorder().Some(),
			Gap:                 style.Some(style.Gap(1)),
		}),
		element.Box("\nPress [SPACE] to see Auto Placement").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{100, 100, 100, 255})}),
	).Style(containerStyle())
}

func viewAutoPlacement() element.Element {
	return element.Box(
		element.Box("3. Auto Placement (with Gaps)").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{255, 255, 0, 255}), Margin: style.Some(style.Edges(0, 0, 1, 0))}),
		element.Box(
			element.Box("A").Style(itemStyle(color.RGBA{100, 50, 0, 255})),
			element.Box("B").Style(itemStyle(color.RGBA{0, 100, 50, 255})),
			element.Box("C").Style(itemStyle(color.RGBA{50, 0, 100, 255})),
			element.Box("D").Style(itemStyle(color.RGBA{100, 100, 100, 255})),
			element.Box("E").Style(itemStyle(color.RGBA{50, 50, 50, 255})),
			element.Box("F").Style(itemStyle(color.RGBA{0, 0, 0, 255})),
		).Style(style.Style{
			Display:             style.Some(style.DisplayGrid),
			GridTemplateColumns: style.Some(style.Repeat(3, style.Fr(1))),
			GridColumnGap:       style.Some(4),
			GridRowGap:          style.Some(1),
			Height:              style.Some(style.Auto),
			Border:              style.SingleBorder().Some(),
			Padding:             style.Some(style.Edges(1)),
		}),
		element.Box("\nPress [SPACE] to see Mixed Tracks").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{100, 100, 100, 255})}),
	).Style(containerStyle())
}

func viewMixedTracks() element.Element {
	return element.Box(
		element.Box("4. Mixed Tracks (Fixed, Auto, Fr)").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{255, 255, 0, 255}), Margin: style.Some(style.Edges(0, 0, 1, 0))}),
		element.Box(
			element.Box("Fixed 10").Style(itemStyle(color.RGBA{150, 0, 0, 255})),
			element.Box("Auto (Shrink)").Style(itemStyle(color.RGBA{0, 150, 0, 255})),
			element.Box("1fr (Rest)").Style(itemStyle(color.RGBA{0, 0, 150, 255})),
		).Style(style.Style{
			Display:             style.Some(style.DisplayGrid),
			GridTemplateColumns: style.Some([]style.GridTrackSize{style.Cells(10), style.Auto, style.Fr(1)}),
			Height:              style.Some(style.Auto),
			Border:              style.SingleBorder().Some(),
		}),
		element.Box("\nPress [SPACE] to see Explicit Coordinates").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{100, 100, 100, 255})}),
	).Style(containerStyle())
}

func viewExplicitCoordinates() element.Element {
	return element.Box(
		element.Box("5. Explicit Coordinates").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{255, 255, 0, 255}), Margin: style.Some(style.Edges(0, 0, 1, 0))}),
		element.Box(
			element.Box("1,1").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{100, 0, 0, 255}),
				Border:     style.SingleBorder().Some(),
				GridColumn: style.Some(style.GridPlacement{Start: 1}),
				GridRow:    style.Some(style.GridPlacement{Start: 1}),
			}),
			element.Box("2,2").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{0, 100, 0, 255}),
				Border:     style.SingleBorder().Some(),
				GridColumn: style.Some(style.GridPlacement{Start: 2}),
				GridRow:    style.Some(style.GridPlacement{Start: 2}),
			}),
			element.Box("3,3").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{0, 0, 100, 255}),
				Border:     style.SingleBorder().Some(),
				GridColumn: style.Some(style.GridPlacement{Start: 3}),
				GridRow:    style.Some(style.GridPlacement{Start: 3}),
			}),
		).Style(style.Style{
			Display:             style.Some(style.DisplayGrid),
			GridTemplateColumns: style.Some(style.Repeat(3, style.Fr(1))),
			GridTemplateRows:    style.Some(style.Repeat(3, style.Cells(3))),
			Height:              style.Some(style.Auto),
			Border:              style.SingleBorder().Some(),
		}),
		element.Box("\nPress [SPACE] to see Holy Grail Layout").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{100, 100, 100, 255})}),
	).Style(containerStyle())
}

func viewHolyGrail() element.Element {
	return element.Box(
		element.Box("6. Holy Grail Layout").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{255, 255, 0, 255}), Margin: style.Some(style.Edges(0, 0, 1, 0))}),
		element.Box(
			element.Box("HEADER").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{150, 150, 150, 255}),
				Foreground: style.Some[color.Color](color.RGBA{0, 0, 0, 255}),
				Border:     style.SingleBorder().Some(),
				GridColumn: style.Some(style.GridPlacement{Span: 3}),
			}),
			element.Box("SIDEBAR").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{100, 100, 100, 255}),
				Border:     style.SingleBorder().Some(),
			}),
			element.Box("CONTENT").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{50, 50, 50, 255}),
				Border:     style.SingleBorder().Some(),
			}),
			element.Box("RIGHT").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{100, 100, 100, 255}),
				Border:     style.SingleBorder().Some(),
			}),
			element.Box("FOOTER").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{150, 150, 150, 255}),
				Foreground: style.Some[color.Color](color.RGBA{0, 0, 0, 255}),
				Border:     style.SingleBorder().Some(),
				GridColumn: style.Some(style.GridPlacement{Span: 3}),
			}),
		).Style(style.Style{
			Display:             style.Some(style.DisplayGrid),
			GridTemplateColumns: style.Some([]style.GridTrackSize{style.Cells(12), style.Fr(1), style.Cells(10)}),
			GridTemplateRows:    style.Some([]style.GridTrackSize{style.Cells(3), style.Fr(1), style.Cells(3)}),
			Width:               style.Some(style.Percent(100)),
			Height:              style.Some(style.Cells(15)),
			Border:              style.SingleBorder().Some(),
		}),
		element.Box("\nPress [SPACE] to return to Basic Grid").Style(style.Style{Foreground: style.Some[color.Color](color.RGBA{100, 100, 100, 255})}),
	).Style(containerStyle())
}
