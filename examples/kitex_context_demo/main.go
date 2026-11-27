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
	btnStyle = style.S().
			Margin(0, 1).
			Padding(0, 1).
			Background(style.RGB(70, 70, 90)).
			Foreground(color.White)

	appContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Padding(1, 2).
				Background(style.RGB(15, 15, 20))

	appTitleStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.RGB(255, 200, 50)).
			Bold(true)

	buttonRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Margin(0, 0, 1, 0)

	providerBoxStyle = style.S().
				Margin(1, 0).
				Padding(1, 1).
				Border(style.SingleBorder())

	instructionsStyle = style.S().
				Margin(1, 0, 0, 0).
				Foreground(style.RGB(120, 120, 130))

	rootStyle = style.S().
			Width(style.Percent(100)).
			Height(style.Percent(100))
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
		bg = style.RGB(25, 25, 30)
		fg = style.RGB(240, 240, 245)
	case ThemeBlue:
		bg = style.RGB(15, 35, 75)
		fg = style.RGB(210, 230, 255)
	default: // ThemeLight
		bg = style.RGB(245, 245, 245)
		fg = style.RGB(30, 30, 35)
	}

	return style.S().
		Margin(1, 0).
		Padding(1, 2).
		Border(style.SingleBorder()).
		Background(bg).
		Foreground(fg)
}

// ConsumerComponent consumes ThemeContext and displays itself accordingly.
var ConsumerComponent = kitex.FC("ConsumerComponent", func(_ kitex.NoProps) kitex.Node {
	theme := kitex.UseContext(ThemeContext)
	themeStyle := getThemeStyle(theme)

	return kitex.Box.Style(
		themeStyle,
		kitex.Text(fmt.Sprintf("Consumer (Current Theme: %s)", theme)),
	)
})

// DefaultConsumer does not sit under any Provider and shows default value.
var DefaultConsumer = kitex.FC("DefaultConsumer", func(_ kitex.NoProps) kitex.Node {
	theme := kitex.UseContext(ThemeContext)
	themeStyle := getThemeStyle(theme)
	// Override margin/title styles
	themeStyle = themeStyle.Border(style.DoubleBorder())

	return kitex.Box.Style(
		themeStyle,
		kitex.Text(fmt.Sprintf("Default Consumer - Outside Provider (Theme: %s)", theme)),
	)
})

// App root component
var App = kitex.FC("App", func(_ kitex.NoProps) kitex.Node {
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

	return kitex.Box.Style(
		appContainerStyle,
		// Title
		kitex.Box.Style(appTitleStyle, kitex.Text("⚛️ Kitex Context System Demo")),

		// Top Actions
		kitex.Box.Style(
			buttonRowStyle,
			kitex.Button(kitex.ButtonProps{
				OnClick: func(e *event.MouseEvent) { toggleOuter() },
				Style:   btnStyle,
			}, kitex.Text("Toggle Outer Theme")),

			kitex.Button(kitex.ButtonProps{
				OnClick: func(e *event.MouseEvent) { toggleInner() },
				Style:   btnStyle,
			}, kitex.Text("Toggle Inner Theme")),
		),

		// Default consumer (outside any provider)
		DefaultConsumer(),

		// Theme Provider (Outer)
		ThemeContext.Provider(outerTheme(),
			kitex.Box.Style(
				providerBoxStyle,
				kitex.Text(fmt.Sprintf("Outer Provider (Providing: %s)", outerTheme())),

				// Outer Consumer
				ConsumerComponent(),

				// Nested Theme Provider (Inner)
				ThemeContext.Provider(innerTheme(),
					kitex.Box.Style(
						providerBoxStyle,
						kitex.Text(fmt.Sprintf("Nested Inner Provider (Providing: %s)", innerTheme())),

						// Inner Consumer
						ConsumerComponent(),
					),
				),
			),
		),

		// Help/Exit instructions
		kitex.Box.Style(instructionsStyle, kitex.Text("Press 'q' or 'ctrl+c' to quit.")),
	)
})

func main() {
	f, _ := os.Create("kitex_context_demo.log")
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
