package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/style"
)

var (
	// Base container style.
	containerStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			AlignItems(style.AlignCenter).
			JustifyContent(style.JustifyCenter).
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Background(color.RGBA{R: 20, G: 20, B: 255, A: 255}). // Blue background on small screen
		// Conditional media query style using the new static media rules system:
		Media(style.Query().MinWidth(80), style.S().
			Background(color.RGBA{R: 20, G: 255, B: 20, A: 255})) // Green background on wide screen

	cardStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Padding(1).
			Border(style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 255, A: 255})).
			Background(color.RGBA{R: 30, G: 30, B: 30, A: 255})

	titleStyle = style.S().
			Bold(true).
			Foreground(color.RGBA{R: 255, G: 255, B: 255, A: 255})

	rootStyle = style.S().
			Width(style.Percent(100)).
			Height(style.Percent(100))
)

func makeApp(cancel context.CancelFunc) kitex.Node {
	return kitex.SimpleFC("App", func() kitex.Node {
		// Use the new UseViewportSize reactive hook
		viewport := kitex.UseViewportSize()

		// Global keyboard listener to allow exiting the app
		kitex.UseKeyboard(func(e event.KeyEvent) {
			if e.MatchString("q", "ctrl+c", "esc") {
				cancel()
			}
		}, []any{})

		// Decide what layout to show based on the viewport width.
		var infoText string
		if viewport.Width >= 80 {
			infoText = fmt.Sprintf("🖥️ WIDE SCREEN DETECTED (%d cells wide) - Background is GREEN", viewport.Width)
		} else {
			infoText = fmt.Sprintf("📱 SMALL SCREEN DETECTED (%d cells wide) - Background is BLUE", viewport.Width)
		}

		return kitex.Box(kitex.BoxProps{
			Style: containerStyle,
		},
			kitex.Box(kitex.BoxProps{
				Style: cardStyle,
			},
				kitex.Span(kitex.SpanProps{
					Style: titleStyle,
				}, kitex.Text("⚡ Responsive Styling & Hook Demo ⚡")),
				kitex.Text(""),
				kitex.Text(infoText),
				kitex.Text(fmt.Sprintf("Current viewport dimensions: %d x %d", viewport.Width, viewport.Height)),
				// Empty text cells to act as gap spacing
				kitex.Text(""),
				kitex.Text("Resize your terminal to see the background color and text change!"),
				kitex.Text("Press 'q', Esc, or Ctrl+C to exit."),
			),
		)
	})()
}

func main() {
	f, _ := os.Create("responsive_demo.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{})

	container := element.NewBox(eng.Document())
	container.Style(rootStyle)
	eng.Mount(container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Mount VDOM App into host container
	kitex.Render(makeApp(cancel), container)

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
