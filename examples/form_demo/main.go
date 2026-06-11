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
	"github.com/masterkeysrd/kite/extras/form"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/kitex/kitexdt"
	"github.com/masterkeysrd/kite/style"
)

var (
	flexColStyle            = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn)
	fieldWrapperStyle       = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).Margin(0, 0, 1, 0)
	fullWidthStyle          = style.S().Width(style.Percent(100))
	flexRowStyle            = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow)
	radioOptionWrapperStyle = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).AlignItems(style.AlignCenter).Margin(0, 2, 0, 0)
	radioOptionAlignStyle   = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).AlignItems(style.AlignCenter)
	checkboxWrapperStyle    = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).AlignItems(style.AlignCenter).Margin(0, 0, 1, 0)
	hostContainerStyle      = style.S().Width(style.Percent(100)).Height(style.Percent(100))
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
	colError     = color.RGBA{R: 248, G: 113, B: 113, A: 255} // Red error
)

type ProfileData struct {
	Username    string `json:"username"`
	AccessClass string `json:"access_class"`
	Theme       string `json:"theme"`
	Newsletter  bool   `json:"newsletter"`
}

var App = kitex.SimpleFC("App", func() kitex.Node {
	notification, setNotification := kitex.UseState("")
	hasSubmitted, setHasSubmitted := kitex.UseState(false)

	f := form.Use(form.Options[ProfileData]{
		InitialValues: ProfileData{
			Username:    "neo",
			AccessClass: "operator",
			Theme:       "emerald",
			Newsletter:  true,
		},
		Validate: func(d ProfileData) map[string]string {
			errs := make(map[string]string)
			if len(d.Username) < 3 {
				errs["username"] = "Username must be at least 3 characters."
			}
			return errs
		},
		OnSubmit: func(d ProfileData) error {
			// Set success message and flag
			setNotification(fmt.Sprintf("SUCCESS: Profile for '%s' initialized.", d.Username))
			setHasSubmitted(true)
			return nil
		},
	})

	s := f.State()

	return kitex.Box(kitex.BoxProps{
		Style: flexColStyle.JustifyContent(style.JustifyCenter).AlignItems(style.AlignCenter).Width(style.Percent(100)).Height(style.Percent(100)).Background(colBG),
	},
		kitex.Box(kitex.BoxProps{
			Style: flexColStyle.Width(style.Cells(50)).Background(colCard).Border(style.SingleBorder().Color(colBorder)).Padding(1, 2),
		},
			// Success Notification
			kitex.If(notification() != "", func() kitex.Node {
				return kitex.Box(kitex.BoxProps{
					Style: style.S().Background(colAccent).Foreground(colBG).Padding(0, 1).Margin(0, 0, 1, 0).Bold(true).TextAlign(style.TextAlignCenter),
				}, kitex.Text(notification()))
			}),

			// Title
			kitex.Box(kitex.BoxProps{
				Style: style.S().Foreground(colText).Bold(true).TextAlign(style.TextAlignCenter).Margin(0, 0, 1, 0),
			}, kitex.Text("   Create Cyber Profile   ")),

			// Divider
			kitex.Box(kitex.BoxProps{
				Style: fullWidthStyle.Height(style.Cells(1)).Background(colSeparator).Margin(0, 0, 1, 0),
			}),

			// Form Wrapper
			kitex.Form(kitex.FormProps{
				OnSubmit: f.HandleSubmit,
				Style:    flexColStyle,
			},
				// Field: Username Box
				kitex.Box(kitex.BoxProps{
					Style: fieldWrapperStyle,
				},
					kitex.Span(kitex.SpanProps{
						Style: style.S().Foreground(colLabel),
					}, kitex.Text("Username")),
					kitex.Input(kitex.InputProps{
						Name:     "username",
						Value:    s.Values.Username,
						Disabled: s.IsSubmitting,
						Style:    fullWidthStyle.Background(colInputBG).Border(style.SingleBorder().Color(colBorder)).Padding(0, 1),
					}),
					kitex.If(s.Errors["username"] != "", func() kitex.Node {
						return kitex.Span(kitex.SpanProps{
							Style: style.S().Foreground(colError),
						}, kitex.Text(s.Errors["username"]))
					}),
				),

				// Field: Role (Select Dropdown) Box
				kitex.Box(kitex.BoxProps{
					Style: fieldWrapperStyle,
				},
					kitex.Span(kitex.SpanProps{
						Style: style.S().Foreground(colLabel),
					}, kitex.Text("Access Class")),
					kitex.Select(kitex.SelectProps{
						Name:     "access_class",
						Value:    s.Values.AccessClass,
						Disabled: s.IsSubmitting,
						Style:    fullWidthStyle,
					},
						kitex.Option(kitex.OptionProps{Text: "Administrator", Value: "admin"}),
						kitex.Option(kitex.OptionProps{Text: "Operator", Value: "operator"}),
						kitex.Option(kitex.OptionProps{Text: "Infiltrator", Value: "infiltrator"}),
					),
				),

				// Field: Theme (Radio Group) Box
				kitex.Box(kitex.BoxProps{
					Style: fieldWrapperStyle,
				},
					kitex.Span(kitex.SpanProps{
						Style: style.S().Foreground(colLabel),
					}, kitex.Text("Visual Spectrum")),
					kitex.RadioGroup(kitex.RadioGroupProps{
						Value:    s.Values.Theme,
						Disabled: s.IsSubmitting,
						Style:    flexRowStyle,
					},
						kitex.Box(kitex.BoxProps{
							Style: radioOptionWrapperStyle,
						},
							kitex.Radio(kitex.RadioProps{Name: "theme", Value: "emerald"}),
							kitex.Span(kitex.SpanProps{
								Style: style.S().Foreground(colText).Margin(0, 0, 0, 1),
							}, kitex.Text("Emerald")),
						),
						kitex.Box(kitex.BoxProps{
							Style: radioOptionAlignStyle,
						},
							kitex.Radio(kitex.RadioProps{Name: "theme", Value: "amber"}),
							kitex.Span(kitex.SpanProps{
								Style: style.S().Foreground(colText).Margin(0, 0, 0, 1),
							}, kitex.Text("Amber")),
						),
					),
				),

				// Field: Terms (Checkbox)
				kitex.Box(kitex.BoxProps{
					Style: checkboxWrapperStyle,
				},
					kitex.Checkbox(kitex.CheckboxProps{
						Name:     "newsletter",
						Checked:  s.Values.Newsletter,
						Disabled: s.IsSubmitting,
					}),
					kitex.Span(kitex.SpanProps{
						Style: style.S().Foreground(colText).Margin(0, 0, 0, 1),
					}, kitex.Text("Subscribe to intel feed")),
				),

				// Submit Button
				kitex.Button(kitex.ButtonProps{
					Type:     "submit",
					Disabled: s.IsSubmitting,
					Style:    style.S().Display(style.DisplayFlex).JustifyContent(style.JustifyCenter).AlignItems(style.AlignCenter).Background(colButtonBG).Foreground(colText).Border(style.SingleBorder().Color(colBorder)).Padding(0, 2).Height(style.Cells(3)).Margin(1, 0, 0, 0),
				}, kitex.Text(func() string {
					if s.IsSubmitting {
						return "Processing..."
					}
					return "Initialize Profile"
				}())),
			),

			// Divider
			kitex.Box(kitex.BoxProps{
				Style: fullWidthStyle.Height(style.Cells(1)).Background(colSeparator).Margin(1, 0, 1, 0),
			}),

			// Submitted Data Preview
			kitex.Box(kitex.BoxProps{
				Style: flexColStyle,
			},
				kitex.Span(kitex.SpanProps{
					Style: style.S().Foreground(colLabel).Bold(true),
				}, kitex.Text("Decrypted Transmission Payload:")),

				kitex.IfElse(hasSubmitted() || s.IsSubmitting,
					kitex.Span(kitex.SpanProps{
						Style: style.S().Foreground(colAccent).Bold(true).Margin(0, 0, 0, 0),
					}, kitex.Text(func() string {
						if s.IsSubmitting {
							return "Encrypting packet..."
						}
						return fmt.Sprintf("Username: %v | Access: %v | Spectrum: %v | Feed: %v",
							s.Values.Username, s.Values.AccessClass, s.Values.Theme, s.Values.Newsletter)
					}())),
					kitex.Span(kitex.SpanProps{
						Style: style.S().Foreground(colLabel).Italic(true),
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
	container.Style(hostContainerStyle)
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
