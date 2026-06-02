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
	primaryBtnStyle      = style.S().Background(color.RGBA{R: 60, G: 60, B: 100, A: 255}).Foreground(color.White)
	primaryBtnHoverStyle = style.S().Background(color.RGBA{R: 80, G: 80, B: 140, A: 255}).Foreground(color.White)
	defaultBtnStyle      = style.S().Margin(style.Edges(1, 0, 0, 0))
	defaultBtnHoverStyle = style.S().Background(color.RGBA{R: 70, G: 70, B: 70, A: 255}).Foreground(color.White).Margin(style.Edges(1, 0, 0, 0))
	dangerBtnStyle       = style.S().Background(color.RGBA{R: 150, G: 40, B: 40, A: 255}).Foreground(color.White).Bold(true).Border(style.DoubleBorder()).Padding(style.Edges(0, 2)).Margin(style.Edges(1, 0, 0, 0))
	dangerBtnHoverStyle  = style.S().Background(color.RGBA{R: 200, G: 50, B: 50, A: 255}).Foreground(color.White).Bold(true).Border(style.DoubleBorder()).Padding(style.Edges(0, 2)).Margin(style.Edges(1, 0, 0, 0))
	titleStyle           = style.S().Bold(true).TextAlign(style.TextAlignCenter).Margin(style.Edges(0, 0, 1, 0))
	counterStyle         = style.S().Foreground(color.RGBA{R: 100, G: 255, B: 100, A: 255}).Margin(style.Edges(0, 0, 1, 0))
	instructionsStyle    = style.S().Foreground(color.RGBA{R: 150, G: 150, B: 150, A: 255})
	containerStyle       = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).AlignItems(style.AlignCenter).JustifyContent(style.JustifyCenter).Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 20, G: 20, B: 20, A: 255})
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

	eng := engine.New(b, engine.Options{Logger: logger, Profiler: true})

	count := 0
	counterText := element.Text("Clicks: 0")

	// Create a standard button
	btn1 := element.Button("  Click Me  ").Style(primaryBtnStyle)

	btn1.OnEvent(event.EventClick, func(e event.Event) {
		count++
		counterText.SetData(fmt.Sprintf("Clicks: %d", count))
		eng.RequestFrame()
	})

	// Add hover effects to btn1
	btn1.OnEvent(event.EventMouseEnter, func(e event.Event) {
		btn1.Style(primaryBtnHoverStyle)
		eng.RequestFrame()
	})
	btn1.OnEvent(event.EventMouseLeave, func(e event.Event) {
		btn1.Style(primaryBtnStyle)
		eng.RequestFrame()
	})

	// Create a default button showcasing hover styling (no initial background/foreground set)
	btnDefault := element.Button("  Hover Me  ").Style(defaultBtnStyle)

	btnDefault.OnEvent(event.EventMouseEnter, func(e event.Event) {
		btnDefault.Style(defaultBtnHoverStyle)
		eng.RequestFrame()
	})
	btnDefault.OnEvent(event.EventMouseLeave, func(e event.Event) {
		btnDefault.Style(defaultBtnStyle)
		eng.RequestFrame()
	})

	// Create a disabled button
	btn2 := element.Button("Disabled").Disabled(true).Style(defaultBtnStyle)

	// Create a styled "action" button
	btn3 := element.Button("  DANGER  ").Style(dangerBtnStyle)

	btn3.OnEvent(event.EventClick, func(e event.Event) {
		count = 0
		counterText.SetData("Clicks: 0 (Reset!)")
		eng.RequestFrame()
	})

	// Add hover effects to btn3
	btn3.OnEvent(event.EventMouseEnter, func(e event.Event) {
		btn3.Style(dangerBtnHoverStyle)
		eng.RequestFrame()
	})
	btn3.OnEvent(event.EventMouseLeave, func(e event.Event) {
		btn3.Style(dangerBtnStyle)
		eng.RequestFrame()
	})

	root := element.Box(
		element.Box(
			element.Box("Button Component Demonstration").Style(titleStyle),

			element.Box(counterText).Style(counterStyle),

			btn1,
			btnDefault,
			btn2,
			btn3,

			element.Box("\nInstructions:").Style(instructionsStyle),
			element.Box("- Click with Mouse").Style(instructionsStyle),
			element.Box("- Tab to focus, then press Space or Enter").Style(instructionsStyle),
			element.Box("- Press 'q' to quit").Style(instructionsStyle),
		).Style(containerStyle),
	)

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
