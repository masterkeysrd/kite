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

// Sleek dark palette for a premium feel
var (
	colBG        = color.RGBA{R: 15, G: 16, B: 22, A: 255}    // Deep space black
	colCard      = color.RGBA{R: 24, G: 26, B: 37, A: 255}    // Matte obsidian
	colBorder    = color.RGBA{R: 53, G: 59, B: 83, A: 255}    // Indigo dusk
	colInputBG   = color.RGBA{R: 18, G: 19, B: 27, A: 255}    // Soft void
	colLabel     = color.RGBA{R: 147, G: 158, B: 195, A: 255} // Blue slate
	colText      = color.RGBA{R: 220, G: 225, B: 245, A: 255} // Arctic white
	colAccent    = color.RGBA{R: 74, G: 222, B: 128, A: 255}  // Emerald neon
	colButtonBG  = color.RGBA{R: 79, G: 70, B: 229, A: 255}   // Cyber indigo
	colSeparator = color.RGBA{R: 38, G: 42, B: 59, A: 255}    // Dark iron
)

var App = kitex.SimpleFC("App", func() kitex.Node {
	// Local state to store submitted form data
	submittedData, setSubmittedData := kitex.UseState[map[string]any](nil)

	handleSubmit := func(data map[string]any) {
		setSubmittedData(data)
	}

	return kitex.Box(kitex.BoxProps{
		Style: style.Style{
			Display:        style.Some(style.DisplayFlex),
			FlexDirection:  style.Some(style.FlexColumn),
			JustifyContent: style.Some(style.JustifyCenter),
			AlignItems:     style.Some(style.AlignCenter),
			Width:          style.Some(style.Percent(100)),
			Height:         style.Some(style.Percent(100)),
			Background:     style.Some[color.Color](colBG),
		},
	},
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Display:       style.Some(style.DisplayFlex),
				FlexDirection: style.Some(style.FlexColumn),
				Width:         style.Some(style.Cells(50)),
				Background:    style.Some[color.Color](colCard),
				Border:        style.SingleBorder().Color(colBorder).Some(),
				Padding:       style.Some(style.Edges(1, 2)),
			},
		},
			// Title
			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Foreground: style.Some[color.Color](colText),
					Bold:       style.Some(true),
					TextAlign:  style.Some(style.TextAlignCenter),
					Margin:     style.Some(style.Edges(0, 0, 1, 0)),
				},
			}, kitex.Text("   Create Cyber Profile   ")),

			// Divider
			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Width:      style.Some(style.Percent(100)),
					Height:     style.Some(style.Cells(1)),
					Background: style.Some[color.Color](colSeparator),
					Margin:     style.Some(style.Edges(0, 0, 1, 0)),
				},
			}),

			// Form Wrapper
			kitex.Form(kitex.FormProps{
				OnSubmit: handleSubmit,
				Style: style.Style{
					Display:       style.Some(style.DisplayFlex),
					FlexDirection: style.Some(style.FlexColumn),
				},
			},
				// Field: Username Box
				kitex.Box(kitex.BoxProps{
					Style: style.Style{
						Display:       style.Some(style.DisplayFlex),
						FlexDirection: style.Some(style.FlexColumn),
						Margin:        style.Some(style.Edges(0, 0, 1, 0)),
					},
				},
					kitex.Span(kitex.SpanProps{
						Style: style.Style{Foreground: style.Some[color.Color](colLabel)},
					}, kitex.Text("Username")),
					kitex.Input(kitex.InputProps{
						Name:  "username",
						Value: "neo",
						Style: style.Style{
							Width:      style.Some(style.Percent(100)),
							Background: style.Some[color.Color](colInputBG),
							Border:     style.SingleBorder().Color(colBorder).Some(),
							Padding:    style.Some(style.Edges(0, 1)),
						},
					}),
				),

				// Field: Role (Select Dropdown) Box
				kitex.Box(kitex.BoxProps{
					Style: style.Style{
						Display:       style.Some(style.DisplayFlex),
						FlexDirection: style.Some(style.FlexColumn),
						Margin:        style.Some(style.Edges(0, 0, 1, 0)),
					},
				},
					kitex.Span(kitex.SpanProps{
						Style: style.Style{Foreground: style.Some[color.Color](colLabel)},
					}, kitex.Text("Access Class")),
					kitex.Select(kitex.SelectProps{
						Name:  "access_class",
						Value: "operator",
						Style: style.Style{
							Width: style.Some(style.Percent(100)),
						},
					},
						kitex.Option(kitex.OptionProps{Text: "Administrator", Value: "admin"}),
						kitex.Option(kitex.OptionProps{Text: "Operator", Value: "operator"}),
						kitex.Option(kitex.OptionProps{Text: "Infiltrator", Value: "infiltrator"}),
					),
				),

				// Field: Theme (Radio Group) Box
				kitex.Box(kitex.BoxProps{
					Style: style.Style{
						Display:       style.Some(style.DisplayFlex),
						FlexDirection: style.Some(style.FlexColumn),
						Margin:        style.Some(style.Edges(0, 0, 1, 0)),
					},
				},
					kitex.Span(kitex.SpanProps{
						Style: style.Style{Foreground: style.Some[color.Color](colLabel)},
					}, kitex.Text("Visual Spectrum")),
					kitex.RadioGroup(kitex.RadioGroupProps{
						Value: "emerald",
						Style: style.Style{
							Display:       style.Some(style.DisplayFlex),
							FlexDirection: style.Some(style.FlexRow),
						},
					},
						kitex.Box(kitex.BoxProps{
							Style: style.Style{
								Display:       style.Some(style.DisplayFlex),
								FlexDirection: style.Some(style.FlexRow),
								AlignItems:    style.Some(style.AlignCenter),
								Margin:        style.Some(style.Edges(0, 2, 0, 0)),
							},
						},
							kitex.Radio(kitex.RadioProps{Name: "theme", Value: "emerald"}),
							kitex.Span(kitex.SpanProps{
								Style: style.Style{Foreground: style.Some[color.Color](colText), Margin: style.Some(style.Edges(0, 0, 0, 1))},
							}, kitex.Text("Emerald")),
						),
						kitex.Box(kitex.BoxProps{
							Style: style.Style{
								Display:       style.Some(style.DisplayFlex),
								FlexDirection: style.Some(style.FlexRow),
								AlignItems:    style.Some(style.AlignCenter),
							},
						},
							kitex.Radio(kitex.RadioProps{Name: "theme", Value: "amber"}),
							kitex.Span(kitex.SpanProps{
								Style: style.Style{Foreground: style.Some[color.Color](colText), Margin: style.Some(style.Edges(0, 0, 0, 1))},
							}, kitex.Text("Amber")),
						),
					),
				),

				// Field: Terms (Checkbox)
				kitex.Box(kitex.BoxProps{
					Style: style.Style{
						Display:       style.Some(style.DisplayFlex),
						FlexDirection: style.Some(style.FlexRow),
						AlignItems:    style.Some(style.AlignCenter),
						Margin:        style.Some(style.Edges(0, 0, 1, 0)),
					},
				},
					kitex.Checkbox(kitex.CheckboxProps{
						Name:    "newsletter",
						Checked: true,
					}),
					kitex.Span(kitex.SpanProps{
						Style: style.Style{Foreground: style.Some[color.Color](colText), Margin: style.Some(style.Edges(0, 0, 0, 1))},
					}, kitex.Text("Subscribe to intel feed")),
				),

				// Submit Button
				kitex.Button(kitex.ButtonProps{
					Type: "submit",
					Style: style.Style{
						Display:        style.Some(style.DisplayFlex),
						JustifyContent: style.Some(style.JustifyCenter),
						AlignItems:     style.Some(style.AlignCenter),
						Background:     style.Some[color.Color](colButtonBG),
						Foreground:     style.Some[color.Color](colText),
						Border:         style.SingleBorder().Color(colBorder).Some(),
						Padding:        style.Some(style.Edges(0, 2)),
						Height:         style.Some(style.Cells(3)),
						Margin:         style.Some(style.Edges(1, 0, 0, 0)),
					},
				}, kitex.Text("Initialize Profile")),
			),

			// Divider
			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Width:      style.Some(style.Percent(100)),
					Height:     style.Some(style.Cells(1)),
					Background: style.Some[color.Color](colSeparator),
					Margin:     style.Some(style.Edges(1, 0, 1, 0)),
				},
			}),

			// Submitted Data Preview
			kitex.Box(kitex.BoxProps{
				Style: style.Style{
					Display:       style.Some(style.DisplayFlex),
					FlexDirection: style.Some(style.FlexColumn),
				},
			},
				kitex.Span(kitex.SpanProps{
					Style: style.Style{Foreground: style.Some[color.Color](colLabel), Bold: style.Some(true)},
				}, kitex.Text("Decrypted Transmission Payload:")),

				kitex.IfElse(submittedData() != nil,
					kitex.Span(kitex.SpanProps{
						Style: style.Style{Foreground: style.Some[color.Color](colAccent), Bold: style.Some(true), Margin: style.Some(style.Edges(0, 0, 0, 0))},
					}, kitex.Text(func() string {
						data := submittedData()
						if data == nil {
							return ""
						}
						return fmt.Sprintf("Username: %v | Access: %v | Spectrum: %v | Feed: %v",
							data["username"], data["access_class"], data["theme"], data["newsletter"])
					}())),
					kitex.Span(kitex.SpanProps{
						Style: style.Style{Foreground: style.Some[color.Color](colLabel), Italic: style.Some(true)},
					}, kitex.Text("Awaiting submission...")),
				),
			),
		),
	)
})

func main() {
	f, err := os.Create("form_demo.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Create backend first
	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create backend: %v\n", err)
		os.Exit(1)
	}

	// Create and start engine
	eng := engine.New(b, engine.Options{Logger: logger})

	// Create VDOM rendering container element
	container := element.NewBox(eng.Document())
	container.Style(style.Style{
		Width:  style.Some(style.Percent(100)),
		Height: style.Some(style.Percent(100)),
	})
	eng.Mount(container)

	kitex.EnableDevMode = true

	// Mount VDOM into host container
	kitex.Render(App(), container)

	// Devtools
	insp, _ := devtools.Install(eng, devtools.Options{})
	kitexdt.Register(insp)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Quit on Ctrl+C or Q
	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke, ok := e.(*event.KeyEvent)
		if !ok {
			return
		}
		if ke.MatchString("ctrl+c") || ke.MatchString("q") {
			cancel()
		}
	})

	if err := eng.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
		os.Exit(1)
	}
}
