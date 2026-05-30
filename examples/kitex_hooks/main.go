package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/kitex/kitexdt"
	"github.com/masterkeysrd/kite/style"
)

func makeApp(cancel context.CancelFunc) kitex.Node {
	return kitex.SimpleFC("App", func() kitex.Node {
		buttonRef := kitex.UseRef[dom.Element](nil)
		isFocused := kitex.UseFocus(buttonRef)
		getLastKey, setLastKey := kitex.UseState("None")

		// UseKeyboard for global shortcuts and interactivity
		kitex.UseKeyboard(func(e event.KeyEvent) {
			if e.MatchString("q", "ctrl+c", "esc") {
				cancel()
				return
			}
			setLastKey(e.Text)
		}, []any{})

		borderColor := color.RGBA{G: 255, A: 255} // Green
		if isFocused {
			borderColor = color.RGBA{R: 255, A: 255} // Red
		}

		return kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Display:        style.Some(style.DisplayFlex),
				FlexDirection:  style.Some(style.FlexColumn),
				AlignItems:     style.Some(style.AlignCenter),
				JustifyContent: style.Some(style.JustifyCenter),
				Width:          style.Some(style.Percent(100)),
				Height:         style.Some(style.Percent(100)),
				Background:     style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 20, A: 255}),
			},
		},
			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Display:       style.Some(style.DisplayFlex),
					FlexDirection: style.Some(style.FlexColumn),
					Gap:           style.Some(style.Gap(1)),
					AlignItems:    style.Some(style.AlignCenter),
				},
			},
				kitex.Span(kitex.SpanProps{
					Style: style.Style{
						Bold:       style.Some(true),
						Foreground: style.Some[color.Color](color.RGBA{R: 90, G: 140, B: 255, A: 255}),
					},
				}, kitex.Text("⚡ Kitex Convenience Hooks Demo ⚡")),
				kitex.Text("Press Tab to focus the button."),
				kitex.Text("Press any key to see it update below."),
				kitex.Text("Press 'q', Esc, or Ctrl+C to exit."),

				kitex.Box(kitex.BoxProps{
					Style: style.Style{
						Margin:     style.Some(style.Edges(1, 0)),
						Padding:    style.Some(style.Edges(0, 1)),
						Background: style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 40, A: 255}),
						Border:     style.SingleBorder().Some(),
					},
				}, kitex.Text(fmt.Sprintf("Last key: %s", getLastKey()))),

				kitex.Button(kitex.ButtonProps{
					Ref: buttonRef,
					Style: style.Style{
						Border:     style.Some(style.SingleBorder().Color(borderColor)),
						Padding:    style.Some(style.Edges(0, 1)),
						Width:      style.Some(style.Cells(25)),
						Height:     style.Some(style.Cells(3)),
						Background: style.Some[color.Color](color.RGBA{R: 40, G: 40, B: 40, A: 255}),
					},
				}, kitex.Text(fmt.Sprintf("Focused: %v", isFocused))),
			),
		)
	})()
}

func main() {
	f, _ := os.Create("kitex_hooks.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{Logger: logger})

	// Create VDOM rendering container element
	container := element.NewBox(eng.Document())
	container.Style(style.Style{
		Width:  style.Some(style.Percent(100)),
		Height: style.Some(style.Percent(100)),
	})
	eng.Mount(container)

	kitex.EnableDevMode = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Mount VDOM into host container
	kitex.Render(makeApp(cancel), container)

	insp, _ := devtools.Install(eng, devtools.Options{})
	kitexdt.Register(insp)

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
