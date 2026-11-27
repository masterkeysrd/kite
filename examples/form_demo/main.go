package main

import (
	"context"
	"fmt"
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
	flexColStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn)

	fieldWrapperStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Margin(0, 0, 1, 0)

	fullWidthStyle = style.S().Width(style.Percent(100))

	flexRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow)

	radioOptionWrapperStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				AlignItems(style.AlignCenter).
				Margin(0, 2, 0, 0)

	radioOptionAlignStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				AlignItems(style.AlignCenter)

	checkboxWrapperStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				AlignItems(style.AlignCenter).
				Margin(0, 0, 1, 0)
)

// Sleek dark palette
var (
	colBG        = style.Hex("#0f1016") // Deep space black
	colCard      = style.Hex("#181a25") // Matte obsidian
	colBorder    = style.Hex("#353b53") // Indigo dusk
	colInputBG   = style.Hex("#12131b") // Soft void
	colLabel     = style.Hex("#939ec3") // Blue slate
	colText      = style.Hex("#dce1f5") // Arctic white
	colAccent    = style.Hex("#4ade80") // Emerald neon
	colButtonBG  = style.Hex("#4f46e5") // Cyber indigo
	colSeparator = style.Hex("#262a3b") // Dark iron
	colError     = style.Hex("#f87171") // Red error
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
	})

	onSubmit := func(d ProfileData) error {
		setNotification(fmt.Sprintf("SUCCESS: Profile for '%s' initialized.", d.Username))
		setHasSubmitted(true)
		return nil
	}

	s := f.State()

	return kitex.Box(kitex.BoxProps{
		Style: flexColStyle.
			JustifyContent(style.JustifyCenter).
			AlignItems(style.AlignCenter).
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Background(colBG),
	},
		kitex.Box(kitex.BoxProps{
			Style: flexColStyle.
				Width(style.Cells(50)).
				Padding(1, 2).
				Border(style.SingleBorder().Color(colBorder)).
				Background(colCard),
		},
			kitex.If(notification() != "", func() kitex.Node {
				return kitex.Box(kitex.BoxProps{
					Style: style.S().
						Margin(0, 0, 1, 0).
						Padding(0, 1).
						Background(colAccent).
						Foreground(colBG).
						Bold(true).
						TextAlign(style.TextAlignCenter),
				}, kitex.Text(notification()))
			}),

			kitex.Box(kitex.BoxProps{
				Style: style.S().
					Margin(0, 0, 1, 0).
					Foreground(colText).
					Bold(true).
					TextAlign(style.TextAlignCenter),
			}, kitex.Text("   Create Cyber Profile   ")),

			kitex.Box(kitex.BoxProps{
				Style: fullWidthStyle.
					Height(style.Cells(1)).
					Margin(0, 0, 1, 0).
					Background(colSeparator),
			}),

			kitex.Form(kitex.FormProps{
				OnSubmit: f.HandleSubmit(onSubmit),
				Style:    flexColStyle,
			},
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
						Style: fullWidthStyle.
							Padding(0, 1).
							Border(style.SingleBorder().Color(colBorder)).
							Background(colInputBG),
					}),
					kitex.If(s.Errors["username"] != "", func() kitex.Node {
						return kitex.Span(kitex.SpanProps{
							Style: style.S().Foreground(colError),
						}, kitex.Text(s.Errors["username"]))
					}),
				),

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
								Style: style.S().
									Margin(0, 0, 0, 1).
									Foreground(colText),
							}, kitex.Text("Emerald")),
						),
						kitex.Box(kitex.BoxProps{
							Style: radioOptionAlignStyle,
						},
							kitex.Radio(kitex.RadioProps{Name: "theme", Value: "amber"}),
							kitex.Span(kitex.SpanProps{
								Style: style.S().
									Margin(0, 0, 0, 1).
									Foreground(colText),
							}, kitex.Text("Amber")),
						),
					),
				),

				kitex.Box(kitex.BoxProps{
					Style: checkboxWrapperStyle,
				},
					kitex.Checkbox(kitex.CheckboxProps{
						Name:     "newsletter",
						Checked:  s.Values.Newsletter,
						Disabled: s.IsSubmitting,
					}),
					kitex.Span(kitex.SpanProps{
						Style: style.S().
							Margin(0, 0, 0, 1).
							Foreground(colText),
					}, kitex.Text("Subscribe to intel feed")),
				),

				kitex.Button(kitex.ButtonProps{
					Type:     "submit",
					Disabled: s.IsSubmitting,
					Style: style.S().
						Display(style.DisplayFlex).
						JustifyContent(style.JustifyCenter).
						AlignItems(style.AlignCenter).
						Height(style.Cells(3)).
						Margin(1, 0, 0, 0).
						Padding(0, 2).
						Border(style.SingleBorder().Color(colBorder)).
						Background(colButtonBG).
						Foreground(colText),
				}, kitex.Text(func() string {
					if s.IsSubmitting {
						return "Processing..."
					}
					return "Initialize Profile"
				}())),
			),

			kitex.Box(kitex.BoxProps{
				Style: fullWidthStyle.
					Height(style.Cells(1)).
					Margin(1, 0, 1, 0).
					Background(colSeparator),
			}),

			kitex.Box(kitex.BoxProps{
				Style: flexColStyle,
			},
				kitex.Span(kitex.SpanProps{
					Style: style.S().
						Foreground(colLabel).
						Bold(true),
				}, kitex.Text("Decrypted Transmission Payload:")),

				kitex.IfElse(hasSubmitted() || s.IsSubmitting,
					kitex.Span(kitex.SpanProps{
						Style: style.S().
							Foreground(colAccent).
							Bold(true),
					}, kitex.Text(func() string {
						if s.IsSubmitting {
							return "Encrypting packet..."
						}
						return fmt.Sprintf("Username: %v | Access: %v | Spectrum: %v | Feed: %v",
							s.Values.Username, s.Values.AccessClass, s.Values.Theme, s.Values.Newsletter)
					}())),
					kitex.Span(kitex.SpanProps{
						Style: style.S().
							Foreground(colLabel).
							Italic(true),
					}, kitex.Text("Awaiting submission...")),
				),
			),
		),
	)
})

func main() {
	f, _ := os.Create("form_demo.log")
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

	kitex.EnableDevMode = true
	if insp, err := devtools.Install(eng, devtools.Options{}); err == nil {
		kitexdt.Register(insp)
	}

	kitex.Render(App(), container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok && (ke.MatchString("ctrl+c") || ke.MatchString("q")) {
			cancel()
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
