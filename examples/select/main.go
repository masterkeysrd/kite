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
	selectDropdownStyle = style.S().Width(style.Cells(25))

	instructionLineStyle = style.S().
				Display(style.DisplayBlock).
				Margin(0, 0, 0, 0).
				Foreground(style.RGB(150, 150, 180))

	titleStyle = style.S().
			Margin(0, 0, 1, 0).
			Background(style.RGB(60, 60, 120)).
			Foreground(color.White).
			Bold(true).
			TextAlign(style.TextAlignCenter)

	pickLabelStyle = style.S().Bold(true)

	pickerRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			AlignItems(style.AlignCenter).
			Gap(0, 1).
			Margin(1, 0, 2, 0)

	selectedStatusStyle = style.S().
				Foreground(style.RGB(100, 255, 100)).
				Bold(true)

	statusRowStyle = style.S().
			Margin(0, 0, 2, 0).
			Border(style.SingleBorder().Left(false).Right(false).Top(false).Color(style.RGB(50, 50, 50)))

	instructionsHeaderStyle = style.S().
				Margin(0, 0, 1, 0).
				Bold(true)

	cardContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				JustifyContent(style.JustifyStart).
				AlignItems(style.AlignStart).
				Width(style.Cells(50)).
				Height(style.Auto).
				Padding(1, 3).
				Border(style.SingleBorder().Color(style.RGB(100, 100, 100))).
				Background(style.RGB(30, 30, 30))

	rootStyle = style.S().
			Display(style.DisplayFlex).
			JustifyContent(style.JustifyCenter).
			AlignItems(style.AlignCenter).
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Background(style.RGB(15, 15, 20))
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
			s = s.Foreground(style.RGB(255, 215, 0)) // Gold/Yellow text when focused
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
