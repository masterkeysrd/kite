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
	"github.com/masterkeysrd/kite/extras/flight"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/style"
)

var (
	homeCardStyle     = style.S().Padding(1, 2).Background(color.RGBA{R: 25, G: 35, B: 55, A: 255}).Border(style.SingleBorder()).Width(style.Percent(80)).Margin(1, 0)
	homeTitleStyle    = style.S().Bold(true).Foreground(color.RGBA{R: 90, G: 160, B: 255, A: 255}).Margin(0, 0, 1, 0)
	bodyTextStyle     = style.S().Margin(0, 0, 1, 0)
	primaryBtnStyle   = style.S().Background(color.RGBA{R: 50, G: 120, B: 220, A: 255}).Foreground(color.White)
	detailsCardStyle  = style.S().Padding(1, 2).Background(color.RGBA{R: 35, G: 55, B: 38, A: 255}).Border(style.SingleBorder()).Width(style.Percent(80)).Margin(1, 0)
	detailsTitleStyle = style.S().Bold(true).Foreground(color.RGBA{R: 120, G: 220, B: 140, A: 255}).Margin(0, 0, 1, 0)
	backBtnStyle      = style.S().Background(color.RGBA{R: 200, G: 60, B: 60, A: 255}).Foreground(color.White)
	appContainerStyle = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).AlignItems(style.AlignCenter).Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 18, G: 18, B: 24, A: 255}).Padding(1, 2)
	appHeaderStyle    = style.S().Bold(true).Foreground(color.RGBA{R: 240, G: 190, B: 90, A: 255}).Margin(0, 0, 1, 0).TextAlign(style.TextAlignCenter)
	instructionsStyle = style.S().Foreground(color.RGBA{R: 150, G: 150, B: 150, A: 255}).TextAlign(style.TextAlignCenter)
	rootStyle         = style.S().Width(style.Percent(100)).Height(style.Percent(100))
)

type HomeRoute struct{}
type DetailsRoute struct {
	ID string
}

var HomeView = kitex.SimpleFC("HomeView", func() kitex.Node {
	nav := flight.UseNavigation()

	// Register keyboard hook: 'enter' goes to DetailsRoute with ID "42"
	kitex.UseKeyboard(func(e event.KeyEvent) {
		if e.MatchString("enter") {
			nav.Push(DetailsRoute{ID: "42"})
		}
	}, []any{nav})

	return kitex.Box(kitex.BoxProps{
		Style: homeCardStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: homeTitleStyle,
		}, kitex.Text("🏠 Home Screen")),

		kitex.Box(kitex.BoxProps{
			Style: bodyTextStyle,
		}, kitex.Text("Press 'Enter' or click the button below to view details for Item #42.")),

		kitex.Button(kitex.ButtonProps{
			OnClick: func(e event.Event) {
				nav.Push(DetailsRoute{ID: "42"})
			},
			Style: primaryBtnStyle,
		}, kitex.Text(" View Details (ID: 42) ")),
	)
})

var DetailsView = kitex.FC("DetailsView", func(props struct{ ID string }) kitex.Node {
	nav := flight.UseNavigation()

	// Register keyboard hook: 'esc' pops back to HomeView
	kitex.UseKeyboard(func(e event.KeyEvent) {
		if e.MatchString("escape") || e.MatchString("esc") {
			nav.Pop()
		}
	}, []any{nav})

	return kitex.Box(kitex.BoxProps{
		Style: detailsCardStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: detailsTitleStyle,
		}, kitex.Text(fmt.Sprintf("ℹ️ Details Screen (Item ID: %s)", props.ID))),

		kitex.Box(kitex.BoxProps{
			Style: bodyTextStyle,
		}, kitex.Text("You are viewing details of item #42. Press 'Esc' or click below to go back.")),

		kitex.Button(kitex.ButtonProps{
			OnClick: func(e event.Event) {
				nav.Pop()
			},
			Style: backBtnStyle,
		}, kitex.Text(" Go Back Home (Esc) ")),
	)
})

var App = kitex.SimpleFC("App", func() kitex.Node {
	return kitex.Box(kitex.BoxProps{
		Style: appContainerStyle,
	},
		// Header
		kitex.Box(kitex.BoxProps{
			Style: appHeaderStyle,
		}, kitex.Text("✈️ Kite Stack Navigation Demo (extras/flight)")),

		// Instruction Info
		kitex.Box(kitex.BoxProps{
			Style: instructionsStyle,
		}, kitex.Text("Use Tab to move focus. Press 'q' or 'ctrl+c' to quit.")),

		// Flight Stack Router
		flight.Stack(flight.StackProps{
			InitialRoute: HomeRoute{},
			RenderRoute: func(r flight.Route) kitex.Node {
				switch route := r.(type) {
				case HomeRoute:
					return HomeView()
				case DetailsRoute:
					return DetailsView(struct{ ID string }{ID: route.ID})
				default:
					return kitex.Box(kitex.BoxProps{})
				}
			},
		}),
	)
})

func main() {
	f, _ := os.Create("flight_demo.log")
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

	container := element.NewBox(eng.Document())
	container.Style(rootStyle)
	eng.Mount(container)

	kitex.Render(App(), container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
