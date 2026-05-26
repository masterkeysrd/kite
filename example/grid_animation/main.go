package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"time"

	"github.com/masterkeysrd/kite/animation"
	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

var eng *engine.Engine

func main() {
	f, _ := os.Create("grid_animation.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng = engine.New(b, engine.Options{Logger: logger, Profiler: true})

	// State
	isAnimating := false
	forward := true

	// Grid element
	gridContainer := element.Box(
		element.Box("Item 1").Style(itemStyle(color.RGBA{150, 50, 50, 255})),
		element.Box("Item 2").Style(itemStyle(color.RGBA{50, 150, 50, 255})),
		element.Box("Item 3").Style(itemStyle(color.RGBA{50, 50, 150, 255})),
		element.Box("Item 4").Style(itemStyle(color.RGBA{150, 150, 50, 255})),
		element.Box("Item 5").Style(itemStyle(color.RGBA{150, 50, 150, 255})),
		element.Box("Item 6").Style(itemStyle(color.RGBA{50, 150, 150, 255})),
	)

	startCols := []style.GridTrackSize{style.Fr(1), style.Fr(1), style.Fr(1)}
	endCols := []style.GridTrackSize{style.Fr(3), style.Fr(1), style.Cells(10)}

	updateGridStyle := func(cols []style.GridTrackSize) {
		gridContainer.Style(style.Style{
			Display:             style.Some(style.DisplayGrid),
			GridTemplateColumns: style.Some(cols),
			GridTemplateRows:    style.Some(style.Repeat(2, style.Fr(1))),
			Height:              style.Some(style.Cells(10)),
			Border:              style.SingleBorder().Some(),
			Gap:                 style.Some(style.Gap(1)),
			Padding:             style.Some(style.Edges(1)),
		})
		eng.RequestFrame()
	}

	updateGridStyle(startCols)

	var triggerBtn *element.ButtonElement
	triggerBtn = element.Button("  Animate Grid Tracks  ")

	defaultBtnStyle := style.Style{
		Background: style.Some[color.Color](color.RGBA{R: 255, G: 69, B: 0, A: 255}), // Red-Orange
		Foreground: style.Some[color.Color](color.White),
		Border:     style.SingleBorder().Some(),
		Bold:       style.Some(true),
		Padding:    style.Some(style.Edges(0, 2)),
	}
	focusBtnStyle := style.Style{
		Background: style.Some[color.Color](color.RGBA{R: 255, G: 99, B: 71, A: 255}), // Tomato
		Border:     style.SingleBorder().Color(color.RGBA{R: 255, G: 215, B: 0, A: 255}).Some(),
		Foreground: style.Some[color.Color](color.White),
		Bold:       style.Some(true),
		Padding:    style.Some(style.Edges(0, 2)),
	}

	triggerBtn.Style(defaultBtnStyle)
	triggerBtn.OnEvent(event.EventFocus, func(e event.Event) {
		triggerBtn.Style(focusBtnStyle)
		eng.RequestFrame()
	})
	triggerBtn.OnEvent(event.EventBlur, func(e event.Event) {
		triggerBtn.Style(defaultBtnStyle)
		eng.RequestFrame()
	})

	triggerBtn.OnEvent(event.EventClick, func(e event.Event) {
		if isAnimating {
			return
		}
		isAnimating = true
		triggerBtn.SetData(" Animating... ")
		eng.RequestFrame()

		start := startCols
		end := endCols
		if !forward {
			start = endCols
			end = startCols
		}

		tween := animation.NewTween(
			start, end,
			1*time.Second,
			animation.EaseInOutCubic,
			animation.InterpolateGridTracks,
			func(cols []style.GridTrackSize) {
				updateGridStyle(cols)
			},
		)

		anim := &CustomAnimator{
			Tween: tween,
			OnComplete: func() {
				isAnimating = false
				forward = !forward
				triggerBtn.SetData("  Animate Grid Tracks  ")
				eng.RequestFrame()
			},
		}

		eng.RegisterAnimation(anim)
	})

	root := element.Box(
		element.Box("⚡ Grid Animation Interpolator Showcase ⚡").Style(style.Style{
			Bold:       style.Some(true),
			Foreground: style.Some[color.Color](color.RGBA{R: 0, G: 255, B: 200, A: 255}),
			TextAlign:  style.Some(style.TextAlignCenter),
			Margin:     style.Some(style.Edges(0, 0, 2, 0)),
		}),
		gridContainer,
		element.Box(triggerBtn).Style(style.Style{
			Margin: style.Some(style.Edges(2, 0, 0, 0)),
		}),
		element.Box("Instructions: Press Tab to focus the button, then Space to click. 'q' to quit.").Style(style.Style{
			Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255}),
			Margin:     style.Some(style.Edges(2, 0, 0, 0)),
		}),
	).Style(style.Style{
		Display:       style.Some(style.DisplayFlex),
		FlexDirection: style.Some(style.FlexColumn),
		Width:         style.Some(style.Percent(100)),
		Height:        style.Some(style.Percent(100)),
		Background:    style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 25, A: 255}),
		Padding:       style.Some(style.Edges(2, 4)),
	})

	eng.Mount(root)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
		}
	})

	devtools.Install(eng, devtools.Options{})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}

func itemStyle(bg color.Color) style.Style {
	return style.Style{
		Background:     style.Some(bg),
		Border:         style.SingleBorder().Some(),
		Width:          style.Some(style.Percent(100)),
		Height:         style.Some(style.Percent(100)),
		AlignItems:     style.Some(style.AlignCenter),
		JustifyContent: style.Some(style.JustifyCenter),
		Display:        style.Some(style.DisplayFlex),
		Bold:           style.Some(true),
	}
}

type CustomAnimator struct {
	Tween      *animation.Tween[[]style.GridTrackSize]
	OnComplete func()
}

func (c *CustomAnimator) Tick(dt time.Duration) bool {
	finished := c.Tween.Tick(dt)
	if finished && c.OnComplete != nil {
		c.OnComplete()
	}
	return finished
}
