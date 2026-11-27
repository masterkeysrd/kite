package main

import (
	"context"
	"fmt"
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

var (
	appContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				JustifyContent(style.JustifyCenter).
				AlignItems(style.AlignCenter).
				Width(style.Percent(100)).
				Height(style.Percent(100)).
				Background(style.RGB(20, 20, 20))

	cardStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			AlignItems(style.AlignCenter).
			Gap(1)

	titleStyle = style.S().
			Foreground(style.RGB(90, 140, 255)).
			Bold(true)

	lastKeyBoxStyle = style.S().
			Margin(1, 0).
			Padding(0, 1).
			Border(style.SingleBorder()).
			Background(style.RGB(30, 30, 40))

	rootStyle = style.S().
			Width(style.Percent(100)).
			Height(style.Percent(100))
)

func makeApp(cancel context.CancelFunc) kitex.Node {
	return kitex.FC("App", func(_ kitex.NoProps) kitex.Node {
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
		})

		borderColor := style.RGB(0, 255, 0) // Green
		if isFocused {
			borderColor = style.RGB(255, 0, 0) // Red
		}

		return kitex.Box.Style(
			appContainerStyle,
			kitex.Box.Style(
				cardStyle,
				kitex.Span.Style(titleStyle, kitex.Text("⚡ Kitex Convenience Hooks Demo ⚡")),
				kitex.Text("Press Tab to focus the button."),
				kitex.Text("Press any key to see it update below."),
				kitex.Text("Press 'q', Esc, or Ctrl+C to exit."),

				kitex.Box.Style(lastKeyBoxStyle, kitex.Text(fmt.Sprintf("Last key: %s", getLastKey()))),

				kitex.Button(kitex.ButtonProps{
					Ref: buttonRef,
					Style: style.S().
						Width(style.Cells(25)).
						Height(style.Cells(3)).
						Padding(0, 1).
						Border(style.SingleBorder().Color(borderColor)).
						Background(style.RGB(40, 40, 40)),
				}, kitex.Text(fmt.Sprintf("Focused: %v", isFocused))),
			),
		)
	})()
}

func main() {
	f, _ := os.Create("kitex_hooks.log")
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

	// Create VDOM rendering container element
	container := element.NewBox(eng.Document())
	container.Style(rootStyle)
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
