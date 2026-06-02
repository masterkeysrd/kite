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
	selectDropdownStyle     = style.S().Width(style.Cells(25))
	instructionLineStyle    = style.S().Display(style.DisplayBlock).Foreground(color.RGBA{R: 150, G: 150, B: 180, A: 255}).Margin(style.Edges(0, 0, 0, 0))
	titleStyle              = style.S().Bold(true).TextAlign(style.TextAlignCenter).Margin(style.Edges(0, 0, 1, 0)).Background(color.RGBA{R: 60, G: 60, B: 120, A: 255}).Foreground(color.White)
	pickLabelStyle          = style.S().Bold(true)
	pickerRowStyle          = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).AlignItems(style.AlignCenter).Margin(style.Edges(1, 0, 2, 0)).Gap(style.Gap(0, 1))
	selectedStatusStyle     = style.S().Foreground(color.RGBA{R: 100, G: 255, B: 100, A: 255}).Bold(true)
	statusRowStyle          = style.S().Margin(style.Edges(0, 0, 2, 0)).Border(style.SingleBorder().Left(false).Right(false).Top(false).Color(color.RGBA{R: 50, G: 50, B: 50, A: 255}))
	instructionsHeaderStyle = style.S().Bold(true).Margin(style.Edges(0, 0, 1, 0))
	cardContainerStyle      = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).AlignItems(style.AlignStart).JustifyContent(style.JustifyStart).Width(style.Cells(50)).Height(style.Auto).Background(color.RGBA{R: 30, G: 30, B: 30, A: 255}).Padding(style.Edges(1, 3)).Border(style.SingleBorder().Color(color.RGBA{R: 100, G: 100, B: 100, A: 255}))
	rootStyle               = style.S().Display(style.DisplayFlex).JustifyContent(style.JustifyCenter).AlignItems(style.AlignCenter).Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 15, G: 15, B: 20, A: 255})
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

	statusText := element.Text("(none)")

	sel := element.Select(
		element.Option("Go", "go"),
		element.Option("Rust", "rust"),
		element.Option("C++", "cpp"),
		element.Option("Zig", "zig"),
		element.Option("Python", "python"),
		element.Option("TypeScript", "ts"),
	).OnChange(func(val string) {
		statusText.SetData(val)
		eng.RequestFrame()
	}).Style(selectDropdownStyle)

	// Show how the user can customize the text color of the focused item in the options
	// by listening to Focus/Blur events globally on the Document.
	eng.Document().AddEventListener(event.EventFocus, func(e event.Event) {
		if btn, ok := e.Target().(*element.ButtonElement); ok && btn.Class() == "select-option" {
			s := btn.RawStyle()
			s = s.Foreground(color.RGBA{R: 255, G: 215, B: 0, A: 255}) // Gold/Yellow text when focused
			btn.Style(s)
		}
	}, event.Capture())

	eng.Document().AddEventListener(event.EventBlur, func(e event.Event) {
		if btn, ok := e.Target().(*element.ButtonElement); ok && btn.Class() == "select-option" {
			s := btn.RawStyle()
			s = s.Foreground(style.TerminalDefault) // Revert to default text color on blur
			btn.Style(s)
		}
	}, event.Capture())

	// Helper for creating instruction lines that are explicitly block-level.
	instr := func(t string) *element.BoxElement {
		return element.Box(t).Style(instructionLineStyle)
	}

	root := element.Box(
		element.Box(
			element.Box("Select Component Demo").Style(titleStyle),

			element.Box(
				element.Span("Pick a language: ").Style(pickLabelStyle),
				sel,
			).Style(pickerRowStyle),

			element.Box(
				element.Span("Selected: "),
				element.Span(statusText).Style(selectedStatusStyle),
			).Style(statusRowStyle),

			element.Box("Instructions:").Style(instructionsHeaderStyle),
			instr("• Click or Space/Enter to toggle"),
			instr("• Arrow keys to navigate options"),
			instr("• Enter to confirm selection"),
			instr("• Esc or Click outside to close"),
			instr("• Press 'q' to quit"),
		).Style(cardContainerStyle)).Style(rootStyle)

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
