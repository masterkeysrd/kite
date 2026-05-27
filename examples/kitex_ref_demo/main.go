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
		Style: style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexColumn),
			Width:         style.Some(style.Percent(100)),
			Height:        style.Some(style.Percent(100)),
			Background:    style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 26, A: 255}),
			Padding:       style.Some(style.Edges(1, 2)),
		},
	},
		// Title
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Bold:       style.Some(true),
				Foreground: style.Some[color.Color](color.RGBA{R: 242, G: 194, B: 48, A: 255}),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
				TextAlign:  style.Some(style.TextAlignCenter),
			},
		}, kitex.Text("✨ Kitex Ref & Hook Demonstration ✨")),

		// Instructions
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Foreground: style.Some[color.Color](color.RGBA{R: 160, G: 160, B: 180, A: 255}),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
			},
		}, kitex.Text("Press 'q' to Quit. This demo shows how UseRef stores persistent mutable values without triggering re-renders, and how it binds to real DOM elements.")),

		// Section 1: Non-rendering Ref state
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Border:     style.SingleBorder().Some(),
				Padding:    style.Some(style.Edges(1, 1)),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
				Background: style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 40, A: 255}),
			},
		},
			kitex.Box(kitex.BoxProps{
				Style: style.Style{Bold: style.Some(true), Margin: style.Some(style.Edges(0, 0, 1, 0))},
			}, kitex.Text("1. Non-Rendering Ref State")),

			kitex.Box(kitex.BoxProps{
				Style: style.Style{Margin: style.Some(style.Edges(0, 0, 1, 0))},
			}, kitex.Text(fmt.Sprintf("Ref Clicks (won't update UI directly): %d", clicksRef.Current))),

			kitex.Box(kitex.BoxProps{
				Style: style.Style{Margin: style.Some(style.Edges(0, 0, 1, 0))},
			}, kitex.Text(fmt.Sprintf("Total Renders (force-render to update UI): %d", getRenderCount()))),

			kitex.Box(kitex.BoxProps{
				Style: style.Style{Display: style.Some(style.DisplayFlex), FlexDirection: style.Some(style.FlexRow)},
			},
				kitex.Button(kitex.ButtonProps{
					OnClick: handleRefClick,
					Style: style.Style{
						Background: style.Some[color.Color](color.RGBA{R: 70, G: 130, B: 180, A: 255}),
						Foreground: style.Some[color.Color](color.White),
						Margin:     style.Some(style.Edges(0, 1)),
					},
				}, kitex.Text(" Increment Ref Clicks ")),

				// Force render button
				kitex.Button(kitex.ButtonProps{
					OnClick: handleForceRender,
					Style: style.Style{
						Background: style.Some[color.Color](color.RGBA{R: 60, G: 179, B: 113, A: 255}),
						Foreground: style.Some[color.Color](color.White),
						Margin:     style.Some(style.Edges(0, 1)),
					},
				}, kitex.Text(" Force Render ")),
			),
		),

		// Section 2: DOM Element Ref Wiring
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Display:       style.Some(style.DisplayFlex),
				FlexDirection: style.Some(style.FlexColumn),
				Border:        style.SingleBorder().Some(),
				Padding:       style.Some(style.Edges(1, 1)),
				Background:    style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 40, A: 255}),
			},
		},
			kitex.Box(kitex.BoxProps{
				Style: style.Style{Bold: style.Some(true), Margin: style.Some(style.Edges(0, 0, 1, 0))},
			}, kitex.Text("2. DOM Element Ref Wiring & Programmatic Focus")),

			// Ref wiring instructions
			kitex.Box(kitex.BoxProps{
				Style: style.Style{Margin: style.Some(style.Edges(0, 0, 1, 0))},
			}, kitex.Text("Use the button below to programmatically focus the input field using its Ref pointer:")),

			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Display:       style.Some(style.DisplayFlex),
					FlexDirection: style.Some(style.FlexRow),
					AlignItems:    style.Some(style.AlignCenter),
					Margin:        style.Some(style.Edges(0, 0, 1, 0)),
				},
			},
				kitex.Input(kitex.InputProps{
					Ref:   inputRef,
					Value: "Focus me programmatically!",
					Style: style.Style{
						Background: style.Some[color.Color](color.RGBA{R: 50, G: 50, B: 60, A: 255}),
						Foreground: style.Some[color.Color](color.White),
						Border:     style.SingleBorder().Some(),
						Padding:    style.Some(style.Edges(0, 1)),
					},
				}),
			),

			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Display:       style.Some(style.DisplayFlex),
					FlexDirection: style.Some(style.FlexRow),
				},
			},
				kitex.Button(kitex.ButtonProps{
					OnClick: handleFocusInput,
					Style: style.Style{
						Background: style.Some[color.Color](color.RGBA{R: 219, G: 112, B: 147, A: 255}),
						Foreground: style.Some[color.Color](color.White),
					},
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
	container.Style(style.Style{
		Width:  style.Some(style.Percent(100)),
		Height: style.Some(style.Percent(100)),
	})
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
