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
	fieldRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			AlignItems(style.AlignCenter).
			Width(style.Percent(100)).
			Margin(0, 0, 1, 0)

	rolesContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				Flex(1)

	buttonRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			JustifyContent(style.JustifyCenter).
			Width(style.Percent(100)).
			Margin(1, 0, 1, 0)

	instructionsStyle = style.S().
				Margin(1, 0, 0, 0).
				Foreground(style.RGB(150, 150, 150))

	rootStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Width(style.Percent(100)).
			Height(style.Percent(100))
)

// FormState holds the state of our form.
type FormState struct {
	Name      string
	Email     string
	Role      string
	Submitted bool
}

// Action represents the reducer action.
type Action struct {
	Type  string
	Value string
}

// formReducer manages transitions for FormState.
func formReducer(state FormState, action Action) FormState {
	switch action.Type {
	case "SET_NAME":
		state.Name = action.Value
		state.Submitted = false
	case "SET_EMAIL":
		state.Email = action.Value
		state.Submitted = false
	case "SET_ROLE":
		state.Role = action.Value
		state.Submitted = false
	case "SUBMIT":
		state.Submitted = true
	case "RESET":
		state.Name = ""
		state.Email = ""
		state.Role = "Developer"
		state.Submitted = false
	}
	return state
}

// Colors for premium styling
var (
	colBG       = style.RGB(20, 20, 30)
	colCard     = style.RGB(30, 30, 45)
	colInput    = style.RGB(40, 40, 60)
	colText     = style.RGB(220, 220, 240)
	colHeader   = style.RGB(120, 140, 255)
	colSuccess  = style.RGB(100, 220, 140)
	colButton   = style.RGB(70, 90, 200)
	colResetBtn = style.RGB(180, 60, 60)
	colBorder   = style.RGB(60, 60, 90)
)

var FormApp = kitex.FC("FormApp", func(_ kitex.NoProps) kitex.Node {
	getState, dispatch := kitex.UseReducer(formReducer, FormState{
		Name:      "",
		Email:     "",
		Role:      "Developer",
		Submitted: false,
	})

	state := getState()

	// Handle input changes
	onNameKeyDown := func(e *event.KeyEvent) {
		if inp, ok := e.Target().(*element.InputElement); ok {
			dispatch(Action{Type: "SET_NAME", Value: inp.Value().(string)})
		}
	}

	onEmailKeyDown := func(e *event.KeyEvent) {
		if inp, ok := e.Target().(*element.InputElement); ok {
			dispatch(Action{Type: "SET_EMAIL", Value: inp.Value().(string)})
		}
	}

	return kitex.Box.Style(
		style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			JustifyContent(style.JustifyCenter).
			AlignItems(style.AlignCenter).
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Padding(1, 2).
			Background(colBG),
		// Main Card Container
		kitex.Box.Style(
			style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Width(style.Percent(80)).
				Padding(1, 2).
				Border(style.DoubleBorder().Color(colBorder)).
				Background(colCard),
			// Header
			kitex.Box.Style(
				style.S().
					Margin(0, 0, 1, 0).
					Foreground(colHeader).
					Bold(true).
					TextAlign(style.TextAlignCenter),
				kitex.Text("⚡ Kitex VDOM UseReducer Form Demo ⚡"),
			),

			// Name Field
			kitex.Box.Style(
				fieldRowStyle,
				kitex.Span.Style(
					style.S().
						Width(style.Cells(10)).
						Foreground(colText),
					kitex.Text("Name: "),
				),
				kitex.Input(kitex.InputProps{
					Value:     state.Name,
					OnKeyDown: onNameKeyDown,
					Style: style.S().
						Flex(1).
						Padding(0, 1).
						Border(style.SingleBorder().Color(colBorder)).
						Background(colInput).
						Foreground(colText),
				}),
			),

			// Email Field
			kitex.Box.Style(
				fieldRowStyle,
				kitex.Span.Style(
					style.S().
						Width(style.Cells(10)).
						Foreground(colText),
					kitex.Text("Email: "),
				),
				kitex.Input(kitex.InputProps{
					Value:     state.Email,
					OnKeyDown: onEmailKeyDown,
					Style: style.S().
						Flex(1).
						Padding(0, 1).
						Border(style.SingleBorder().Color(colBorder)).
						Background(colInput).
						Foreground(colText),
				}),
			),

			// Role Selection (Buttons as Radio simulation)
			kitex.Box.Style(
				fieldRowStyle,
				kitex.Span.Style(
					style.S().
						Width(style.Cells(10)).
						Foreground(colText),
					kitex.Text("Role: "),
				),
				kitex.Box.Style(
					rolesContainerStyle,
					kitex.Button(kitex.ButtonProps{
						OnClick: func(e *event.MouseEvent) { dispatch(Action{Type: "SET_ROLE", Value: "Developer"}) },
						Style: style.S().
							Margin(0, 1, 0, 0).
							Background(func() color.Color {
								if state.Role == "Developer" {
									return colHeader
								}
								return colInput
							}()).
							Foreground(colText),
					}, kitex.Text(" Developer ")),
					kitex.Button(kitex.ButtonProps{
						OnClick: func(e *event.MouseEvent) { dispatch(Action{Type: "SET_ROLE", Value: "Designer"}) },
						Style: style.S().
							Margin(0, 1, 0, 0).
							Background(func() color.Color {
								if state.Role == "Designer" {
									return colHeader
								}
								return colInput
							}()).
							Foreground(colText),
					}, kitex.Text(" Designer ")),
					kitex.Button(kitex.ButtonProps{
						OnClick: func(e *event.MouseEvent) { dispatch(Action{Type: "SET_ROLE", Value: "Manager"}) },
						Style: style.S().
							Background(func() color.Color {
								if state.Role == "Manager" {
									return colHeader
								}
								return colInput
							}()).
							Foreground(colText),
					}, kitex.Text(" Manager ")),
				),
			),

			// Actions (Submit / Reset)
			kitex.Box.Style(
				buttonRowStyle,
				kitex.Button(kitex.ButtonProps{
					OnClick: func(e *event.MouseEvent) { dispatch(Action{Type: "SUBMIT"}) },
					Style: style.S().
						Margin(0, 2, 0, 0).
						Background(colButton).
						Foreground(colText),
				}, kitex.Text(" SUBMIT ")),
				kitex.Button(kitex.ButtonProps{
					OnClick: func(e *event.MouseEvent) { dispatch(Action{Type: "RESET"}) },
					Style: style.S().
						Background(colResetBtn).
						Foreground(colText),
				}, kitex.Text(" RESET ")),
			),

			kitex.Box.Style(
				style.S().
					Width(style.Percent(100)).
					Padding(1, 1).
					Border(style.SingleBorder().Color(colBorder)).
					Background(colInput),
				kitex.Box.Style(
					style.S().
						Margin(0, 0, 1, 0).
						Foreground(colHeader).
						Bold(true),
					kitex.Text("State Live Preview:"),
				),
				kitex.Box(kitex.BoxProps{}, kitex.Text(fmt.Sprintf("Name:  %s", state.Name))),
				kitex.Box(kitex.BoxProps{}, kitex.Text(fmt.Sprintf("Email: %s", state.Email))),
				kitex.Box(kitex.BoxProps{}, kitex.Text(fmt.Sprintf("Role:  %s", state.Role))),
				func() kitex.Node {
					if state.Submitted {
						return kitex.Box.Style(
							style.S().
								Margin(1, 0, 0, 0).
								Foreground(colSuccess).
								Bold(true),
							kitex.Text("✓ Form Submitted Successfully!"),
						)
					}
					return nil
				}(),
			),
		),
		kitex.Box.Style(instructionsStyle, kitex.Text("Press 'q' to Quit. Use Tab to navigate inputs.")),
	)
})

func main() {
	f, _ := os.Create("reducer_demo.log")
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
	kitex.Render(FormApp(), container)

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
