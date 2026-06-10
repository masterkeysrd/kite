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
	btnStyle          = style.S().Background(color.RGBA{R: 70, G: 70, B: 90, A: 255}).Foreground(color.White).Padding(0, 1).Margin(0, 1)
	appContainerStyle = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).Padding(1, 2).Background(color.RGBA{R: 15, G: 15, B: 20, A: 255})
	appTitleStyle     = style.S().Foreground(color.RGBA{R: 255, G: 200, B: 50, A: 255}).Bold(true).Margin(0, 0, 1, 0)
	buttonRowStyle    = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).Margin(0, 0, 1, 0)
	providerBoxStyle  = style.S().Border(style.SingleBorder()).Padding(1, 1).Margin(1, 0)
	instructionsStyle = style.S().Foreground(color.RGBA{R: 120, G: 120, B: 130, A: 255}).Margin(1, 0, 0, 0)
	rootStyle         = style.S().Width(style.Percent(100)).Height(style.Percent(100))
)

type Theme string

const (
	ThemeLight Theme = "light"
	ThemeDark  Theme = "dark"
	ThemeBlue  Theme = "blue"
)

var ThemeContext = kitex.CreateContext(ThemeLight)

// themeColors maps a theme to visual styles
func getThemeStyle(theme Theme) style.Style {
	var bg, fg color.Color
	switch theme {
	case ThemeDark:
		bg = color.RGBA{R: 25, G: 25, B: 30, A: 255}
		fg = color.RGBA{R: 240, G: 240, B: 245, A: 255}
	case ThemeBlue:
		bg = color.RGBA{R: 15, G: 35, B: 75, A: 255}
		fg = color.RGBA{R: 210, G: 230, B: 255, A: 255}
	default: // ThemeLight
		bg = color.RGBA{R: 245, G: 245, B: 245, A: 255}
		fg = color.RGBA{R: 30, G: 30, B: 35, A: 255}
	}

	return style.S().Background(bg).Foreground(fg).Border(style.SingleBorder()).Padding(1, 2).Margin(1, 0)
}

// ConsumerComponent consumes ThemeContext and displays itself accordingly.
var ConsumerComponent = kitex.SimpleFC("ConsumerComponent", func() kitex.Node {
	theme := kitex.UseContext(ThemeContext)
	themeStyle := getThemeStyle(theme)

	return kitex.Box(kitex.BoxProps{
		Style: themeStyle,
	},
		kitex.Text(fmt.Sprintf("Consumer (Current Theme: %s)", theme)),
	)
})

// DefaultConsumer does not sit under any Provider and shows default value.
var DefaultConsumer = kitex.SimpleFC("DefaultConsumer", func() kitex.Node {
	theme := kitex.UseContext(ThemeContext)
	themeStyle := getThemeStyle(theme)
	// Override margin/title styles
	themeStyle = themeStyle.Border(style.DoubleBorder())

	return kitex.Box(kitex.BoxProps{
		Style: themeStyle,
	},
		kitex.Text(fmt.Sprintf("Default Consumer - Outside Provider (Theme: %s)", theme)),
	)
})

// App root component
var App = kitex.SimpleFC("App", func() kitex.Node {
	outerTheme, setOuterTheme := kitex.UseState(ThemeDark)
	innerTheme, setInnerTheme := kitex.UseState(ThemeBlue)

	toggleOuter := func() {
		switch outerTheme() {
		case ThemeLight:
			setOuterTheme(ThemeDark)
		case ThemeDark:
			setOuterTheme(ThemeBlue)
		case ThemeBlue:
			setOuterTheme(ThemeLight)
		}
	}

	toggleInner := func() {
		switch innerTheme() {
		case ThemeLight:
			setInnerTheme(ThemeDark)
		case ThemeDark:
			setInnerTheme(ThemeBlue)
		case ThemeBlue:
			setInnerTheme(ThemeLight)
		}
	}

	btnStyle := btnStyle

	return kitex.Box(kitex.BoxProps{
		Style: appContainerStyle,
	},
		// Title
		kitex.Box(kitex.BoxProps{
			Style: appTitleStyle,
		}, kitex.Text("⚛️ Kitex Context System Demo")),

		// Top Actions
		kitex.Box(kitex.BoxProps{
			Style: buttonRowStyle,
		},
			kitex.Button(kitex.ButtonProps{
				OnClick: func(e event.Event) { toggleOuter() },
				Style:   btnStyle,
			}, kitex.Text("Toggle Outer Theme")),

			kitex.Button(kitex.ButtonProps{
				OnClick: func(e event.Event) { toggleInner() },
				Style:   btnStyle,
			}, kitex.Text("Toggle Inner Theme")),
		),

		// Default consumer (outside any provider)
		DefaultConsumer(),

		// Theme Provider (Outer)
		ThemeContext.Provider(outerTheme(),
			kitex.Box(kitex.BoxProps{
				Style: providerBoxStyle,
			},
				kitex.Text(fmt.Sprintf("Outer Provider (Providing: %s)", outerTheme())),

				// Outer Consumer
				ConsumerComponent(),

				// Nested Theme Provider (Inner)
				ThemeContext.Provider(innerTheme(),
					kitex.Box(kitex.BoxProps{
						Style: providerBoxStyle,
					},
						kitex.Text(fmt.Sprintf("Nested Inner Provider (Providing: %s)", innerTheme())),

						// Inner Consumer
						ConsumerComponent(),
					),
				),
			),
		),

		// Help/Exit instructions
		kitex.Box(kitex.BoxProps{
			Style: instructionsStyle,
		}, kitex.Text("Press 'q' or 'ctrl+c' to quit.")),
	)
})

func main() {
	f, _ := os.Create("kitex_context_demo.log")
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
