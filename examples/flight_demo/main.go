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
	homeCardStyle = style.S().
			Width(style.Percent(80)).
			Margin(1, 0).
			Padding(1, 2).
			Border(style.SingleBorder()).
			Background(style.RGB(25, 35, 55))

	homeTitleStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.RGB(90, 160, 255)).
			Bold(true)

	bodyTextStyle = style.S().Margin(0, 0, 1, 0)

	primaryBtnStyle = style.S().
			Background(style.RGB(50, 120, 220)).
			Foreground(color.White)

	detailsCardStyle = style.S().
				Width(style.Percent(80)).
				Margin(1, 0).
				Padding(1, 2).
				Border(style.SingleBorder()).
				Background(style.RGB(35, 55, 38))

	detailsTitleStyle = style.S().
				Margin(0, 0, 1, 0).
				Foreground(style.RGB(120, 220, 140)).
				Bold(true)

	backBtnStyle = style.S().
			Background(style.RGB(200, 60, 60)).
			Foreground(color.White)

	appContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				AlignItems(style.AlignCenter).
				Width(style.Percent(100)).
				Height(style.Percent(100)).
				Padding(1, 2).
				Background(style.RGB(18, 18, 24))

	appHeaderStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.RGB(240, 190, 90)).
			Bold(true).
			TextAlign(style.TextAlignCenter)

	instructionsStyle = style.S().
				Foreground(style.RGB(150, 150, 150)).
				TextAlign(style.TextAlignCenter)
)

type HomeRoute struct{}
type DetailsRoute struct {
	ID string
}

var HomeView = kitex.FC("HomeView", func(_ kitex.NoProps) kitex.Node {
	nav := flight.UseNavigation()

	kitex.UseKeyboard(func(e event.KeyEvent) {
		if e.MatchString("enter") {
			nav.Push(DetailsRoute{ID: "42"})
		}
	}, nav)

	return kitex.Box.Style(
		homeCardStyle,
		kitex.Box.Style(homeTitleStyle, kitex.Text("🏠 Home Screen")),

		kitex.Box.Style(bodyTextStyle, kitex.Text("Press 'Enter' or click the button below to view details for Item #42.")),

		kitex.Button(kitex.ButtonProps{
			OnPress: func() {
				nav.Push(DetailsRoute{ID: "42"})
			},
			Style: primaryBtnStyle,
		}, kitex.Text(" View Details (ID: 42) ")),
	)
})

var DetailsView = kitex.FC("DetailsView", func(props struct{ ID string }) kitex.Node {
	nav := flight.UseNavigation()

	kitex.UseKeyboard(func(e event.KeyEvent) {
		if (e.MatchString("escape") || e.MatchString("esc")) && nav.CanPop() {
			nav.Pop()
		}
	}, nav)

	return kitex.Box.Style(
		detailsCardStyle,
		kitex.Box.Style(detailsTitleStyle, kitex.Textf("ℹ️ Details Screen (Item ID: %s)", props.ID)),

		kitex.Box.Style(bodyTextStyle, kitex.Text("You are viewing details of item #42. Press 'Esc' or click below to go back.")),

		kitex.If(nav.CanPop(), func() kitex.Node {
			return kitex.Button(kitex.ButtonProps{
				OnPress: func() {
					nav.Pop()
				},
				Style: backBtnStyle,
			}, kitex.Text(" Go Back Home (Esc) "))
		}),
	)
})

var App = kitex.FC("App", func(_ kitex.NoProps) kitex.Node {
	return kitex.Box.Style(
		appContainerStyle,
		kitex.Box.Style(appHeaderStyle, kitex.Text("✈️ Kite Stack Navigation Demo (extras/flight)")),

		kitex.Box.Style(instructionsStyle, kitex.Text("Use Tab to move focus. Press 'q' or 'ctrl+c' to quit.")),

		flight.Stack(flight.StackProps{
			InitialRoute: HomeRoute{},
			RenderRoute: func(r flight.Route) kitex.Node {
				switch route := r.(type) {
				case HomeRoute:
					return HomeView()
				case DetailsRoute:
					return DetailsView.Props(struct{ ID string }{ID: route.ID})
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
	slog.SetDefault(slog.New(slog.NewTextHandler(f, nil)))

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{})
	container := element.NewBox(eng.Document()).Style(
		style.S().
			Width(style.Percent(100)).
			Height(style.Percent(100)),
	)
	eng.Mount(container)

	kitex.Render(App(), container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok && (ke.MatchString("q") || ke.MatchString("ctrl+c")) {
			cancel()
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
