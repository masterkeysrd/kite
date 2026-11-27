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

var (
	btnStyle = style.S().
			Padding(0, 2).
			Border(style.SingleBorder()).
			Background(style.RGB(255, 69, 0)).
			Foreground(color.White).
			Bold(true)

	btnFocusStyle = style.S().
			Padding(0, 2).
			Border(style.SingleBorder().Color(style.RGB(255, 215, 0))).
			Background(style.RGB(255, 99, 71)).
			Foreground(color.White).
			Bold(true)

	titleStyle = style.S().
			Margin(0, 0, 2, 0).
			Foreground(style.RGB(0, 255, 200)).
			Bold(true).
			TextAlign(style.TextAlignCenter)

	btnWrapperStyle = style.S().Margin(2, 0, 0, 0)

	instructionsStyle = style.S().
				Margin(2, 0, 0, 0).
				Foreground(style.RGB(150, 150, 150))

	rootStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Padding(2, 4).
			Background(style.RGB(20, 20, 25))
)

var eng *engine.Engine

func main() {
	f, _ := os.Create("grid_animation.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	_ = logger // prevent unused variable error
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng = engine.New(b, engine.Options{})

	// State
	isAnimating := false
	forward := true

	// Grid element
	gridContainer := element.Box(
		element.Box("Item 1").Style(itemStyle(style.RGB(150, 50, 50))),
		element.Box("Item 2").Style(itemStyle(style.RGB(50, 150, 50))),
		element.Box("Item 3").Style(itemStyle(style.RGB(50, 50, 150))),
		element.Box("Item 4").Style(itemStyle(style.RGB(150, 150, 50))),
		element.Box("Item 5").Style(itemStyle(style.RGB(150, 50, 150))),
		element.Box("Item 6").Style(itemStyle(style.RGB(50, 150, 150))),
	)

	startCols := []style.GridTrackSize{style.Fr(1), style.Fr(1), style.Fr(1)}
	endCols := []style.GridTrackSize{style.Fr(3), style.Fr(1), style.Cells(10)}

	updateGridStyle := func(cols []style.GridTrackSize) {
		gridContainer.Style(
			style.S().
				Display(style.DisplayGrid).
				Gap(1).
				GridTemplateColumns(cols...).
				GridTemplateRows(style.Repeat(2, style.Fr(1))...).
				Height(style.Cells(10)).
				Padding(1).
				Border(style.SingleBorder()),
		)
		eng.RequestFrame()
	}

	updateGridStyle(startCols)

	var triggerBtn *element.ButtonElement
	triggerBtn = element.Button("  Animate Grid Tracks  ")

	defaultBtnStyle := btnStyle
	focusBtnStyle := btnFocusStyle

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
		element.Box("⚡ Grid Animation Interpolator Showcase ⚡").Style(titleStyle),
		gridContainer,
		element.Box(triggerBtn).Style(btnWrapperStyle),
		element.Box("Instructions: Press Tab to focus the button, then Space to click. 'q' to quit.").Style(instructionsStyle),
	).Style(rootStyle)

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
	return style.S().
		Display(style.DisplayFlex).
		JustifyContent(style.JustifyCenter).
		AlignItems(style.AlignCenter).
		Width(style.Percent(100)).
		Height(style.Percent(100)).
		Border(style.SingleBorder()).
		Background(bg).
		Bold(true)
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
