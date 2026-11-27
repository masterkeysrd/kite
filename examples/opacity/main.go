package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"runtime"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

var (
	hoverBtnBaseStyle = style.S().
				Background(color.Transparent).
				Foreground(color.White)

	hoverBtnHoverStyle = style.S().
				Background(style.RGBA(255, 255, 255, 60)).
				Foreground(color.White)

	titleStyle = style.S().
			Padding(1).
			Background(style.RGB(50, 80, 200)).
			Bold(true).
			TextAlign(style.TextAlignCenter)

	redBandStyle = style.S().
			Padding(1, 2).
			Background(style.RGB(200, 50, 50))

	greenBandStyle = style.S().
			Padding(1, 2).
			Background(style.RGB(50, 180, 50))

	blueBandStyle = style.S().
			Padding(1, 2).
			Background(style.RGB(50, 50, 200))

	yellowBandStyle = style.S().
			Padding(1, 2).
			Background(style.RGB(200, 180, 50))

	bandsContainerStyle = style.S().Margin(1, 0)

	btnSectionHeaderStyle = style.S().
				Margin(1, 0, 0, 0).
				Bold(true)

	btnRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Margin(0, 0, 1, 0)

	instructionsStyle = style.S().
				Padding(1, 2).
				Border(style.SingleBorder())

	mainContainerStyle = style.S().
				Width(style.Percent(100)).
				Height(style.Percent(100)).
				Padding(2).
				Background(style.RGB(20, 20, 30))

	overlayTitleStyle = style.S().
				Margin(0, 0, 1, 0).
				Bold(true).
				TextAlign(style.TextAlignCenter)

	overlayFooterStyle = style.S().
				Margin(1, 0, 0, 0).
				Foreground(style.RGB(100, 100, 100)).
				TextAlign(style.TextAlignCenter)

	overlayWrapperStyle = style.S().
				Display(style.DisplayFlex).
				JustifyContent(style.JustifyCenter).
				AlignItems(style.AlignCenter).
				Width(style.Percent(100)).
				Height(style.Percent(100))
)

func createHoverButton(eng *engine.Engine, label string, baseBg color.RGBA) *element.BoxElement {
	// The actual button inside, with a transparent background by default
	btn := element.Button(label).Style(hoverBtnBaseStyle)

	// When hovered, we set a semi-transparent white background on the button.
	// This will blend with the wrapper box's base background color in the framebuffer!
	btn.OnEvent(event.EventMouseEnter, func(e event.Event) {
		btn.Style(hoverBtnHoverStyle)
		eng.RequestFrame()
	})

	btn.OnEvent(event.EventMouseLeave, func(e event.Event) {
		btn.Style(hoverBtnBaseStyle)
		eng.RequestFrame()
	})

	// A wrapper box that holds the solid base background color
	wrapper := element.Box(btn).Style(
		style.S().
			Margin(0, 1).
			Padding(0, 1).
			Border(style.SingleBorder().Color(style.RGBA(255, 255, 255, 80))).
			Background(baseBg),
	)

	return wrapper
}

func main() {
	f, _ := os.Create("kite.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	_ = logger // prevent unused variable error
	slog.SetDefault(logger)

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "unexpected error: %v\n", r)
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			fmt.Fprintf(os.Stderr, "stack trace:\n%s\n", string(buf[:n]))
			os.Exit(1)
		}
	}()

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{})

	// Background grid with multiple colors to show off the transparency blending.
	root := element.Box(
		element.Box("Opacity & Color Blending Demonstration").Style(titleStyle),
		element.Box(
			element.Box("RED BAND (Solid Background)").Style(redBandStyle),
			element.Box("GREEN BAND (Solid Background)").Style(greenBandStyle),
			element.Box("BLUE BAND (Solid Background)").Style(blueBandStyle),
			element.Box("YELLOW BAND (Solid Background)").Style(yellowBandStyle),
		).Style(bandsContainerStyle),
		element.Box("Hover Buttons (Blends semi-transparent white hover state on base color)").Style(btnSectionHeaderStyle),
		element.Box(
			createHoverButton(eng, "  Blue Button  ", color.RGBA{R: 30, G: 60, B: 180, A: 255}),
			createHoverButton(eng, "  Green Button  ", color.RGBA{R: 30, G: 150, B: 60, A: 255}),
			createHoverButton(eng, "  Purple Button  ", color.RGBA{R: 120, G: 30, B: 150, A: 255}),
		).Style(btnRowStyle),
		element.Box(
			"Press 'o' to toggle the semi-transparent overlay dialog.",
			"\nPress '+' or '-' to adjust overlay opacity when visible.",
			"\nPress 'q' or 'ctrl+c' to quit.",
		).Style(instructionsStyle),
	).Style(mainContainerStyle)

	eng.Mount(root)
	devtools.Install(eng, devtools.Options{})

	// Interactive state
	alpha := uint8(128)
	overlayVisible := false

	// Create a dynamic function to build the overlay content at the current alpha.
	getOverlay := func() *element.BoxElement {
		return element.Box(
			element.Box("Semi-Transparent Dialog").Style(overlayTitleStyle),
			fmt.Sprintf("Current Alpha Opacity: %d / 255", alpha),
			"\nNotice how the colored bands behind me\nblend through the dialog background!",
			element.Box("Press '+' to increase opacity, '-' to decrease.").Style(overlayFooterStyle),
		).Style(
			style.S().
				Width(style.Cells(45)).
				Height(style.Cells(10)).
				Padding(1, 2).
				Border(style.DoubleBorder().Color(style.RGB(255, 255, 255))).
				Background(style.RGBA(255, 255, 255, alpha)).
				Foreground(style.RGB(0, 0, 0)),
		)
	}

	// Full-screen overlay container
	var overlayDialog *element.BoxElement

	updateOverlay := func() {
		if overlayVisible && overlayDialog != nil {
			// If already showing, hide the old one first
			eng.Document().HideOverlay(overlayDialog)
		}
		overlayContent := getOverlay()
		overlayDialog = element.Box(overlayContent).Style(overlayWrapperStyle)
		if overlayVisible {
			eng.Document().ShowOverlay(overlayDialog, 100)
		}
	}

	// Global key listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("ctrl+c") || ke.MatchString("q") {
			cancel()
			return
		}

		if ke.MatchString("o") {
			if overlayVisible {
				if overlayDialog != nil {
					eng.Document().HideOverlay(overlayDialog)
				}
				overlayVisible = false
			} else {
				overlayVisible = true
				updateOverlay()
			}
			return
		}

		if overlayVisible {
			if ke.MatchString("+") || ke.MatchString("=") {
				if alpha < 235 {
					alpha += 20
				} else {
					alpha = 255
				}
				updateOverlay()
			} else if ke.MatchString("-") {
				if alpha > 20 {
					alpha -= 20
				} else {
					alpha = 0
				}
				updateOverlay()
			}
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited with error: %v\n", err)
		os.Exit(1)
	}
}
