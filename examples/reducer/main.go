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
	colBG       = color.RGBA{R: 20, G: 20, B: 30, A: 255}
	colCard     = color.RGBA{R: 30, G: 30, B: 45, A: 255}
	colInput    = color.RGBA{R: 40, G: 40, B: 60, A: 255}
	colText     = color.RGBA{R: 220, G: 220, B: 240, A: 255}
	colHeader   = color.RGBA{R: 120, G: 140, B: 255, A: 255}
	colSuccess  = color.RGBA{R: 100, G: 220, B: 140, A: 255}
	colButton   = color.RGBA{R: 70, G: 90, B: 200, A: 255}
	colResetBtn = color.RGBA{R: 180, G: 60, B: 60, A: 255}
	colBorder   = color.RGBA{R: 60, G: 60, B: 90, A: 255}
)

var FormApp = kitex.SimpleFC("FormApp", func() kitex.Node {
	getState, dispatch := kitex.UseReducer(formReducer, FormState{
		Name:      "",
		Email:     "",
		Role:      "Developer",
		Submitted: false,
	})

	state := getState()

	// Handle input changes
	onNameKeyDown := func(e event.Event) {
		if inp, ok := e.Target().(*element.InputElement); ok {
			dispatch(Action{Type: "SET_NAME", Value: inp.Value().(string)})
		}
	}

	onEmailKeyDown := func(e event.Event) {
		if inp, ok := e.Target().(*element.InputElement); ok {
			dispatch(Action{Type: "SET_EMAIL", Value: inp.Value().(string)})
		}
	}

	return kitex.Box(kitex.BoxProps{
		Style: style.Style{
			Display:        style.Some(style.DisplayFlex),
			FlexDirection:  style.Some(style.FlexColumn),
			AlignItems:     style.Some(style.AlignCenter),
			JustifyContent: style.Some(style.JustifyCenter),
			Width:          style.Some(style.Percent(100)),
			Height:         style.Some(style.Percent(100)),
			Background:     style.Some[color.Color](colBG),
			Padding:        style.Some(style.Edges(1, 2)),
		},
	},
		// Main Card Container
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Display:       style.Some(style.DisplayFlex),
				FlexDirection: style.Some(style.FlexColumn),
				Width:         style.Some(style.Percent(80)),
				Background:    style.Some[color.Color](colCard),
				Border:        style.DoubleBorder().Color(colBorder).Some(),
				Padding:       style.Some(style.Edges(1, 2)),
			},
		},
			// Header
			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Foreground: style.Some[color.Color](colHeader),
					Bold:       style.Some(true),
					Margin:     style.Some(style.Edges(0, 0, 1, 0)),
					TextAlign:  style.Some(style.TextAlignCenter),
				},
			}, kitex.Text("⚡ Kitex VDOM UseReducer Form Demo ⚡")),

			// Name Field
			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Display:       style.Some(style.DisplayFlex),
					FlexDirection: style.Some(style.FlexRow),
					AlignItems:    style.Some(style.AlignCenter),
					Width:         style.Some(style.Percent(100)),
					Margin:        style.Some(style.Edges(0, 0, 1, 0)),
				},
			},
				kitex.Span(kitex.SpanProps{
					Style: style.Style{Foreground: style.Some[color.Color](colText), Width: style.Some(style.Cells(10))},
				}, kitex.Text("Name: ")),
				kitex.Input(kitex.InputProps{
					Value:     state.Name,
					OnKeyDown: onNameKeyDown,
					Style: style.Style{
						Flex:       style.Some(style.Flex(1)),
						Background: style.Some[color.Color](colInput),
						Foreground: style.Some[color.Color](colText),
						Border:     style.SingleBorder().Color(colBorder).Some(),
						Padding:    style.Some(style.Edges(0, 1)),
					},
				}),
			),

			// Email Field
			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Display:       style.Some(style.DisplayFlex),
					FlexDirection: style.Some(style.FlexRow),
					AlignItems:    style.Some(style.AlignCenter),
					Width:         style.Some(style.Percent(100)),
					Margin:        style.Some(style.Edges(0, 0, 1, 0)),
				},
			},
				kitex.Span(kitex.SpanProps{
					Style: style.Style{Foreground: style.Some[color.Color](colText), Width: style.Some(style.Cells(10))},
				}, kitex.Text("Email: ")),
				kitex.Input(kitex.InputProps{
					Value:     state.Email,
					OnKeyDown: onEmailKeyDown,
					Style: style.Style{
						Flex:       style.Some(style.Flex(1)),
						Background: style.Some[color.Color](colInput),
						Foreground: style.Some[color.Color](colText),
						Border:     style.SingleBorder().Color(colBorder).Some(),
						Padding:    style.Some(style.Edges(0, 1)),
					},
				}),
			),

			// Role Selection (Buttons as Radio simulation)
			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Display:       style.Some(style.DisplayFlex),
					FlexDirection: style.Some(style.FlexRow),
					AlignItems:    style.Some(style.AlignCenter),
					Width:         style.Some(style.Percent(100)),
					Margin:        style.Some(style.Edges(0, 0, 1, 0)),
				},
			},
				kitex.Span(kitex.SpanProps{
					Style: style.Style{Foreground: style.Some[color.Color](colText), Width: style.Some(style.Cells(10))},
				}, kitex.Text("Role: ")),
				kitex.Box(kitex.BoxProps{
					Style: style.Style{
						Display:       style.Some(style.DisplayFlex),
						FlexDirection: style.Some(style.FlexRow),
						Flex:          style.Some(style.Flex(1)),
					},
				},
					kitex.Button(kitex.ButtonProps{
						OnClick: func(e event.Event) { dispatch(Action{Type: "SET_ROLE", Value: "Developer"}) },
						Style: style.Style{
							Background: style.Some[color.Color](func() color.Color {
								if state.Role == "Developer" {
									return colHeader
								}
								return colInput
							}()),
							Foreground: style.Some[color.Color](colText),
							Margin:     style.Some(style.Edges(0, 1, 0, 0)),
						},
					}, kitex.Text(" Developer ")),
					kitex.Button(kitex.ButtonProps{
						OnClick: func(e event.Event) { dispatch(Action{Type: "SET_ROLE", Value: "Designer"}) },
						Style: style.Style{
							Background: style.Some[color.Color](func() color.Color {
								if state.Role == "Designer" {
									return colHeader
								}
								return colInput
							}()),
							Foreground: style.Some[color.Color](colText),
							Margin:     style.Some(style.Edges(0, 1, 0, 0)),
						},
					}, kitex.Text(" Designer ")),
					kitex.Button(kitex.ButtonProps{
						OnClick: func(e event.Event) { dispatch(Action{Type: "SET_ROLE", Value: "Manager"}) },
						Style: style.Style{
							Background: style.Some[color.Color](func() color.Color {
								if state.Role == "Manager" {
									return colHeader
								}
								return colInput
							}()),
							Foreground: style.Some[color.Color](colText),
						},
					}, kitex.Text(" Manager ")),
				),
			),

			// Actions (Submit / Reset)
			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Display:        style.Some(style.DisplayFlex),
					FlexDirection:  style.Some(style.FlexRow),
					JustifyContent: style.Some(style.JustifyCenter),
					Width:          style.Some(style.Percent(100)),
					Margin:         style.Some(style.Edges(1, 0, 1, 0)),
				},
			},
				kitex.Button(kitex.ButtonProps{
					OnClick: func(e event.Event) { dispatch(Action{Type: "SUBMIT"}) },
					Style: style.Style{
						Background: style.Some[color.Color](colButton),
						Foreground: style.Some[color.Color](colText),
						Margin:     style.Some(style.Edges(0, 2, 0, 0)),
					},
				}, kitex.Text(" SUBMIT ")),
				kitex.Button(kitex.ButtonProps{
					OnClick: func(e event.Event) { dispatch(Action{Type: "RESET"}) },
					Style: style.Style{
						Background: style.Some[color.Color](colResetBtn),
						Foreground: style.Some[color.Color](colText),
					},
				}, kitex.Text(" RESET ")),
			),

			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Background: style.Some[color.Color](colInput),
					Padding:    style.Some(style.Edges(1, 1)),
					Border:     style.SingleBorder().Color(colBorder).Some(),
					Width:      style.Some(style.Percent(100)),
				},
			},
				kitex.Box(kitex.BoxProps{
					Style: style.Style{Foreground: style.Some[color.Color](colHeader), Bold: style.Some(true), Margin: style.Some(style.Edges(0, 0, 1, 0))},
				}, kitex.Text("State Live Preview:")),
				kitex.Box(kitex.BoxProps{}, kitex.Text(fmt.Sprintf("Name:  %s", state.Name))),
				kitex.Box(kitex.BoxProps{}, kitex.Text(fmt.Sprintf("Email: %s", state.Email))),
				kitex.Box(kitex.BoxProps{}, kitex.Text(fmt.Sprintf("Role:  %s", state.Role))),
				func() kitex.Node {
					if state.Submitted {
						return kitex.Box(kitex.BoxProps{
							Style: style.Style{
								Foreground: style.Some[color.Color](colSuccess),
								Bold:       style.Some(true),
								Margin:     style.Some(style.Edges(1, 0, 0, 0)),
							},
						}, kitex.Text("✓ Form Submitted Successfully!"))
					}
					return nil
				}(),
			),
		),
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Foreground: style.Some[color.Color](color.RGBA{R: 150, G: 150, B: 150, A: 255}),
				Margin:     style.Some(style.Edges(1, 0, 0, 0)),
			},
		}, kitex.Text("Press 'q' to Quit. Use Tab to navigate inputs.")),
	)
})

func main() {
	f, _ := os.Create("reducer_demo.log")
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
		Display:       style.Some(style.DisplayFlex),
		FlexDirection: style.Some(style.FlexColumn),
		Width:         style.Some(style.Percent(100)),
		Height:        style.Some(style.Percent(100)),
	})
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
