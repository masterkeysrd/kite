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
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/kitex/kitexdt"
	"github.com/masterkeysrd/kite/style"
)

var (
	appContainerStyle   = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 20, G: 20, B: 26, A: 255}).Padding(1, 2)
	appTitleStyle       = style.S().Bold(true).Foreground(color.RGBA{R: 242, G: 194, B: 48, A: 255}).Margin(0, 0, 1, 0).TextAlign(style.TextAlignCenter)
	appDescriptionStyle = style.S().Foreground(color.RGBA{R: 160, G: 160, B: 180, A: 255}).Margin(0, 0, 1, 0)
	card1Style          = style.S().Border(style.SingleBorder()).Padding(1, 1).Margin(0, 0, 1, 0).Background(color.RGBA{R: 30, G: 30, B: 40, A: 255})
	sectionHeaderStyle  = style.S().Bold(true).Margin(0, 0, 1, 0)
	contentRowStyle     = style.S().Margin(0, 0, 1, 0)
	buttonRowStyle      = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow)
	incrementBtnStyle   = style.S().Background(color.RGBA{R: 70, G: 130, B: 180, A: 255}).Foreground(color.White).Margin(0, 1)
	forceRenderBtnStyle = style.S().Background(color.RGBA{R: 60, G: 179, B: 113, A: 255}).Foreground(color.White).Margin(0, 1)
	card2Style          = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).Border(style.SingleBorder()).Padding(1, 1).Background(color.RGBA{R: 30, G: 30, B: 40, A: 255})
	inputRowStyle       = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).AlignItems(style.AlignCenter).Margin(0, 0, 1, 0)
	inputFieldStyle     = style.S().Background(color.RGBA{R: 50, G: 50, B: 60, A: 255}).Foreground(color.White).Border(style.SingleBorder()).Padding(0, 1)
	focusBtnStyle       = style.S().Background(color.RGBA{R: 219, G: 112, B: 147, A: 255}).Foreground(color.White)
	rootStyle           = style.S().Width(style.Percent(100)).Height(style.Percent(100))
)

// App is the root functional component.
var App = kitex.SimpleFC("App", func() kitex.Node {
	// Dummy state to force a re-render
	getRenderCount, setRenderCount := kitex.UseState(1)

	// A Ref that persists a mutable integer state without triggering updates
	clicksRef := kitex.UseRef(0)

	// A Ref that binds to a DOM Element (the Input)
	inputRef := kitex.UseRef[element.Element](nil)

	// A button click handler to increment the non-rendering clicksRef counter
	handleRefClick := func(e event.Event) {
		clicksRef.Current++
	}

	// A button click handler to force a re-render
	handleForceRender := func(e event.Event) {
		setRenderCount(getRenderCount() + 1)
	}

	// A button click handler to programmatically focus the input element via its Ref
	handleFocusInput := func(e event.Event) {
		if inputRef.Current != nil {
			// Find document focus manager and focus the element
			doc := inputRef.Current.OwnerDocument()
			if doc != nil {
				doc.Focus(inputRef.Current)
			}
		}
	}

	return kitex.Box(kitex.BoxProps{
		Style: appContainerStyle,
	},
		// Title
		kitex.Box(kitex.BoxProps{
			Style: appTitleStyle,
		}, kitex.Text("✨ Kitex Ref & Hook Demonstration ✨")),

		// Instructions
		kitex.Box(kitex.BoxProps{
			Style: appDescriptionStyle,
		}, kitex.Text("Press 'q' to Quit. This demo shows how UseRef stores persistent mutable values without triggering re-renders, and how it binds to real DOM elements.")),

		// Section 1: Non-rendering Ref state
		kitex.Box(kitex.BoxProps{
			Style: card1Style,
		},
			kitex.Box(kitex.BoxProps{
				Style: sectionHeaderStyle,
			}, kitex.Text("1. Non-Rendering Ref State")),

			kitex.Box(kitex.BoxProps{
				Style: contentRowStyle,
			}, kitex.Text(fmt.Sprintf("Ref Clicks (won't update UI directly): %d", clicksRef.Current))),

			kitex.Box(kitex.BoxProps{
				Style: contentRowStyle,
			}, kitex.Text(fmt.Sprintf("Total Renders (force-render to update UI): %d", getRenderCount()))),

			kitex.Box(kitex.BoxProps{
				Style: buttonRowStyle,
			},
				kitex.Button(kitex.ButtonProps{
					OnClick: handleRefClick,
					Style:   incrementBtnStyle,
				}, kitex.Text(" Increment Ref Clicks ")),

				// Force render button
				kitex.Button(kitex.ButtonProps{
					OnClick: handleForceRender,
					Style:   forceRenderBtnStyle,
				}, kitex.Text(" Force Render ")),
			),
		),

		// Section 2: DOM Element Ref Wiring
		kitex.Box(kitex.BoxProps{
			Style: card2Style,
		},
			kitex.Box(kitex.BoxProps{
				Style: sectionHeaderStyle,
			}, kitex.Text("2. DOM Element Ref Wiring & Programmatic Focus")),

			// Ref wiring instructions
			kitex.Box(kitex.BoxProps{
				Style: contentRowStyle,
			}, kitex.Text("Use the button below to programmatically focus the input field using its Ref pointer:")),

			kitex.Box(kitex.BoxProps{
				Style: inputRowStyle,
			},
				kitex.Input(kitex.InputProps{
					Ref:   inputRef,
					Value: "Focus me programmatically!",
					Style: inputFieldStyle,
				}),
			),

			kitex.Box(kitex.BoxProps{
				Style: buttonRowStyle,
			},
				kitex.Button(kitex.ButtonProps{
					OnClick: handleFocusInput,
					Style:   focusBtnStyle,
				}, kitex.Text(" Focus Input Field via Ref ")),
			),
		),
	)
})

func main() {
	f, _ := os.Create("kitex_ref_demo.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{Logger: logger})

	container := element.NewBox(eng.Document())
	container.Style(rootStyle)
	eng.Mount(container)

	kitex.EnableDevMode = true

	kitex.Render(App(), container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
		}
	})

	insp, _ := devtools.Install(eng, devtools.Options{})
	kitexdt.Register(insp)

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
