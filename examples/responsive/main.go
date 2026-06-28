package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"

	"time"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/terminal"
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

		// Use the new UseTitle and UseBell hooks
		_, setTitle := kitex.UseTitle("Kite Responsive Demo")
		ringBell := kitex.UseBell()

		// Use the new UseWindowFocus hook
		windowFocused := kitex.UseWindowFocus()

		// Use the new UseProgressBar hook
		updateProgress := kitex.UseProgressBar()
		progressVal, setProgressVal := kitex.UseState(0)

		// Increment progress bar automatically
		kitex.UseInterval(func() {
			nextVal := progressVal() + 2
			if nextVal > 100 {
				nextVal = 0
			}
			setProgressVal(nextVal)
			updateProgress(terminal.ProgressBarIndeterminate, nextVal)
		}, 100*time.Millisecond, []any{progressVal()})

		// Global keyboard listener to allow exiting the app
		kitex.UseKeyboard(func(e event.KeyEvent) {
			if e.MatchString("q", "ctrl+c", "esc") {
				// Clear the progress bar before exiting
				updateProgress(terminal.ProgressBarHide, 0)
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

		// Update title dynamically whenever the viewport size changes
		setTitle(fmt.Sprintf("Responsive: %d x %d", viewport.Width, viewport.Height))

		var focusText string
		if windowFocused {
			focusText = "🟢 Active (Focused)"
		} else {
			focusText = "🔴 Inactive (Blurred)"
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
				kitex.Text(fmt.Sprintf("Terminal window focus: %s", focusText)),
				kitex.Text(fmt.Sprintf("Dock/Tab progress bar: %d%%", progressVal())),
				kitex.Text(""),
				kitex.Button(kitex.ButtonProps{
					OnClick: func(ev event.Event) {
						ringBell()
					},
					Style: style.S().
						Border(style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 255, A: 255})).
						Padding(0, 1).
						Background(color.RGBA{R: 50, G: 50, B: 50, A: 255}),
				}, kitex.Text("🔔 Ring Terminal Bell")),
				kitex.Text(""),
				kitex.Text("Resize your terminal to see the background color, text, and window title change!"),
				kitex.Text("Unfocus the terminal window (switch window) to see the focus status change."),
				kitex.Text("Look at your terminal tab/dock header to see the native progress bar updating!"),
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
