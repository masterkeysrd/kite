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
	checkboxStyle        = style.S().Margin(0, 1, 0, 0)
	radioItemStyle       = style.S().Margin(0, 0, 0, 0)
	titleStyle           = style.S().Bold(true).TextAlign(style.TextAlignCenter).Margin(0, 0, 1, 0)
	checkboxLabelStyle   = style.S().Bold(true).Margin(0, 1, 0, 0)
	checkboxStatusStyle  = style.S().Foreground(color.RGBA{R: 100, G: 200, B: 255, A: 255})
	checkboxRowStyle     = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).AlignItems(style.AlignCenter).Margin(0, 0, 2, 0)
	radioGroupLabelStyle = style.S().Bold(true).Margin(0, 0, 1, 0)
	radioStatusStyle     = style.S().Foreground(color.RGBA{R: 100, G: 255, B: 100, A: 255}).Margin(1, 0, 0, 0)
	instructionsStyle    = style.S().Foreground(color.RGBA{R: 150, G: 150, B: 150, A: 255})
	contentWrapperStyle  = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).AlignItems(style.AlignStart).JustifyContent(style.JustifyCenter).Width(style.Percent(80)).Height(style.Percent(80)).Background(color.RGBA{R: 30, G: 30, B: 30, A: 255}).Padding(1, 2).Border(style.SingleBorder())
	rootStyle            = style.S().Display(style.DisplayFlex).JustifyContent(style.JustifyCenter).AlignItems(style.AlignCenter).Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 15, G: 15, B: 15, A: 255})
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

	// --- Checkbox Section ---
	cbStatus := element.Text("Unchecked")
	cb := element.Checkbox(false).Style(checkboxStyle)
	cb.OnEvent(event.EventChange, func(e event.Event) {
		if cb.Checked() {
			cbStatus.SetData("Checked")
		} else {
			cbStatus.SetData("Unchecked")
		}
		eng.RequestFrame()
	})

	// --- Radio Section ---
	radioStatus := element.Text("None selected")
	rg := element.RadioGroup(
		element.Box(
			element.Radio("option1"),
			element.Span(" Option 1"),
		).Style(radioItemStyle),
		element.Box(
			element.Radio("option2"),
			element.Span(" Option 2"),
		).Style(radioItemStyle),
		element.Box(
			element.Radio("option3"),
			element.Span(" Option 3"),
		).Style(radioItemStyle),
	).OnChange(func(val string) {
		radioStatus.SetData(fmt.Sprintf("Selected: %s", val))
		eng.RequestFrame()
	})

	root := element.Box(
		element.Box(
			element.Box("Checkbox and Radio Components").Style(titleStyle),

			// Checkbox Row
			element.Box(
				element.Box("Checkbox:").Style(checkboxLabelStyle),
				cb,
				element.Box(cbStatus).Style(checkboxStatusStyle),
			).Style(checkboxRowStyle),

			// Radio Group Section
			element.Box("Radio Group:").Style(radioGroupLabelStyle),
			rg,
			element.Box(radioStatus).Style(radioStatusStyle),

			element.Box("\nInstructions:").Style(instructionsStyle),
			element.Box("- Click with Mouse to toggle/select").Style(instructionsStyle),
			element.Box("- Tab to focus, then press Space to toggle/select").Style(instructionsStyle),
			element.Box("- Use Arrow keys to navigate Radio buttons").Style(instructionsStyle),
			element.Box("- Press 'q' to quit").Style(instructionsStyle),
		).Style(contentWrapperStyle),
	).Style(rootStyle)

	eng.Mount(root)

	// Add capturing event listeners for checkbox and radio buttons focus/blur styling
	eng.Document().AddEventListener(event.EventFocus, func(e event.Event) {
		if et := e.Target().EventTarget(); et != nil {
			switch el := et.(type) {
			case *element.CheckboxElement:
				s := el.RawStyle()
				s = s.Foreground(color.RGBA{R: 255, G: 215, B: 0, A: 255}) // Gold when focused
				el.Style(s)
			case *element.RadioElement:
				s := el.RawStyle()
				s = s.Foreground(color.RGBA{R: 255, G: 215, B: 0, A: 255}) // Gold when focused
				el.Style(s)
			}
		}
	}, event.Capture())

	eng.Document().AddEventListener(event.EventBlur, func(e event.Event) {
		if et := e.Target().EventTarget(); et != nil {
			switch el := et.(type) {
			case *element.CheckboxElement:
				s := el.RawStyle()
				s = s.Foreground(style.TerminalDefault) // Revert to default on blur
				el.Style(s)
			case *element.RadioElement:
				s := el.RawStyle()
				s = s.Foreground(style.TerminalDefault) // Revert to default on blur
				el.Style(s)
			}
		}
	}, event.Capture())

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
