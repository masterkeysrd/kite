package regressions

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/devtools/testenv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
)

func TestExampleApp1Golden(t *testing.T) {
	env := testenv.Default(80, 50)
	defer env.Close()

	// Replicate root from examples/app1/main.go
	flexItems := make([]any, 0, 6)
	for i := 1; i <= 6; i++ {
		item := element.Box(fmt.Sprintf("Flex Item %d", i)).Style(style.Style{
			Width:      style.Some(style.Cells(12)),
			Height:     style.Some(style.Cells(3)),
			Background: style.Some[color.Color](color.RGBA{R: uint8(40 * i), G: 100, B: 150, A: 255}),
			Border:     style.SingleBorder().Some(),
			Flex:       style.Some(style.Flex(1, 1, style.Cells(10))),
		})
		flexItems = append(flexItems, item)
	}

	root := element.Box(
		element.Box(
			element.Box("Kite Layout Engine Test").Style(style.Style{
				Width:      style.Some(style.Percent(100)),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
				TextAlign:  style.Some(style.TextAlignCenter),
				Background: style.Some[color.Color](color.RGBA{R: 100, G: 0, B: 200, A: 255}),
			}),

			element.Box(
				"This is a demonstration of ",
				element.Span("inline elements").Style(style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 255, G: 255, B: 255, A: 255}),
					Foreground: style.Some[color.Color](color.Black),
				}),
				" and ",
				element.Box("Atomic!").Style(style.Style{
					Display:    style.Some(style.DisplayInlineBlock),
					Width:      style.Some(style.Cells(10)),
					Height:     style.Some(style.Cells(3)),
					Background: style.Some[color.Color](color.RGBA{R: 0, G: 200, B: 100, A: 255}),
					Margin:     style.Some(style.Edges(0, 1)),
					Border:     style.SingleBorder().Some(),
				}),
				" working together in a single flow.",
			).Style(style.Style{
				AlignItems: style.Some(style.AlignCenter),
			}),

			element.Box(
				"Available Features:",
				element.UL(
					element.LI("Full LayoutNG engine"),
					element.LI("Interactive DOM components"),
					element.LI("Flexible styling system"),
				),
			).Style(style.Style{
				Margin:     style.Some(style.Edges(1, 0)),
				Background: style.Some[color.Color](color.RGBA{R: 40, G: 40, B: 60, A: 255}),
				Padding:    style.Some(style.Edges(1)),
			}),

			element.Box(
				flexItems...,
			).Style(style.Style{
				Display:       style.Some(style.DisplayFlex),
				FlexDirection: style.Some(style.FlexRow),
				FlexWrap:      style.Some(style.FlexWrapOn),
				Width:         style.Some(style.Percent(100)),
				Margin:        style.Some(style.Edges(1, 0)),
				Padding:       style.Some(style.Edges(1)),
				Background:    style.Some[color.Color](color.RGBA{R: 50, G: 50, B: 50, A: 255}),
				Gap:           style.Some(style.Gap(1, 2)),
			}),

			element.Box(
				"Grid Layout (Table):",
				element.Table(
					element.TR(
						element.TD("Header 1").Style(style.Style{Width: style.Some(style.Percent(30))}),
						element.TD("Header 2").Style(style.Style{Width: style.Some(style.Percent(70))}),
					),
					element.TR(
						element.TD("Row 1, Cell 1"),
						element.TD("Row 1, Cell 2"),
					),
				).Style(style.Style{
					Width:  style.Some(style.Percent(100)),
					Border: style.SingleBorder().Some(),
				}),
			).Style(style.Style{
				Margin:     style.Some(style.Edges(1, 0)),
				Padding:    style.Some(style.Edges(1)),
				Background: style.Some[color.Color](color.RGBA{R: 20, G: 60, B: 20, A: 255}),
			}),
		).Style(style.Style{
			Width:      style.Some(style.Percent(80)),
			Height:     style.Some(style.Auto),
			Margin:     style.Some(style.Edges(1, 2)),
			Background: style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 30, A: 255}),
			Border:     style.SingleBorder().Color(color.RGBA{R: 200, G: 200, B: 200, A: 255}).Some(),
			Padding:    style.Some(style.Edges(1, 2)),
		}),
	).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
		Padding:    style.Some(style.Edges(2, 4)),
		Background: style.Some[color.Color](color.RGBA{R: 0, G: 0, B: 255, A: 255}),
	})

	env.Mount(root)
	env.Flush()

	env.MatchGolden(t, "example_app1")
}

func TestExampleFlexGolden(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	headerStyle := style.Style{
		Foreground: style.Some[color.Color](color.RGBA{R: 255, G: 255, B: 0, A: 255}),
		Margin:     style.Some(style.Edges(0, 0, 1, 0)),
	}

	// Helper for creating inline flex items
	inlineFlexItems := make([]any, 0, 3)
	for i := 1; i <= 3; i++ {
		inlineFlexItems = append(inlineFlexItems,
			element.Box(fmt.Sprintf("Item %d", i)).Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 150, G: 0, B: 0, A: 255}),
				Padding:    style.Some(style.Edges(0, 1)),
			}),
		)
	}

	// Helper for creating row flex items
	rowFlexItems := make([]any, 0, 4)
	for i := 1; i <= 4; i++ {
		rowFlexItems = append(rowFlexItems,
			element.Box(fmt.Sprintf("Row Item %d", i)).Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 0, G: 120, B: 0, A: 255}),
				Padding:    style.Some(style.Edges(0, 2)),
				Height:     style.Some(style.Cells(1 + i%2)),
			}),
		)
	}

	// Helper for column items
	colFlexItems := make([]any, 0, 3)
	for i := 1; i <= 3; i++ {
		colFlexItems = append(colFlexItems,
			element.Box(fmt.Sprintf("Column Item %d (Stays Right)", i)).Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 180, G: 80, B: 0, A: 255}),
				Padding:    style.Some(style.Edges(0, 2)),
				Width:      style.Some(style.Auto),
			}),
		)
	}

	// Build UI declaratively
	root := element.Box(
		// 1. Inline Flex Example
		element.Box("1. Inline Flex (Shrink-wrap content)").Style(headerStyle),
		element.Box(
			"Text before -> ",
			element.Box(inlineFlexItems...).Style(style.Style{
				Display:    style.Some(style.DisplayInlineFlex),
				Background: style.Some[color.Color](color.RGBA{R: 0, G: 80, B: 150, A: 255}),
				Border:     style.SingleBorder().Some(),
				Gap:        style.Some(style.Gap(0, 1)),
				Padding:    style.Some(style.Edges(0, 1)),
			}),
			" <- Text after",
		).Style(style.Style{Margin: style.Some(style.Edges(0, 0, 2, 0))}),

		// 2. Flex Row Example
		element.Box("2. Flex Row (Justify: Space-Between, Align: Center)").Style(headerStyle),
		element.Box(rowFlexItems...).Style(style.Style{
			Display:        style.Some(style.DisplayFlex),
			FlexDirection:  style.Some(style.FlexRow),
			JustifyContent: style.Some(style.JustifyBetween),
			AlignItems:     style.Some(style.AlignCenter),
			Background:     style.Some[color.Color](color.RGBA{R: 40, G: 40, B: 40, A: 255}),
			Height:         style.Some(style.Cells(5)),
			Padding:        style.Some(style.Edges(0, 2)),
			Margin:         style.Some(style.Edges(0, 0, 2, 0)),
		}),

		// 3. Flex Column Example
		element.Box("3. Flex Column (Align: End)").Style(headerStyle),
		element.Box(colFlexItems...).Style(style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexColumn),
			AlignItems:    style.Some(style.AlignEnd),
			Background:    style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 60, A: 255}),
			Width:         style.Some(style.Percent(50)),
			Padding:       style.Some(style.Edges(1, 2)),
			Gap:           style.Some(style.Gap(1, 0)),
			Margin:        style.Some(style.Edges(0, 0, 2, 0)),
		}),

		// 4. Flex Row Reverse Example
		element.Box("4. Flex Row Reverse").Style(headerStyle),
		element.Box(
			element.Box("Reverse Item 1").Style(style.Style{Background: style.Some[color.Color](color.RGBA{R: 200, G: 0, B: 0, A: 255}), Padding: style.Some(style.Edges(0, 1))}),
			element.Box("Reverse Item 2").Style(style.Style{Background: style.Some[color.Color](color.RGBA{R: 0, G: 200, B: 0, A: 255}), Padding: style.Some(style.Edges(0, 1))}),
			element.Box("Reverse Item 3").Style(style.Style{Background: style.Some[color.Color](color.RGBA{R: 0, G: 0, B: 200, A: 255}), Padding: style.Some(style.Edges(0, 1))}),
		).Style(style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexRowReverse),
			Background:    style.Some[color.Color](color.RGBA{R: 60, G: 30, B: 30, A: 255}),
			Padding:       style.Some(style.Edges(0, 2)),
			Margin:        style.Some(style.Edges(0, 0, 2, 0)),
			Gap:           style.Some(style.Gap(0, 2)),
		}),

		// 5. Flex Order Example
		element.Box("5. Flex Order Property").Style(headerStyle),
		element.Box(
			element.Box("First in DOM (Order 3)").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 200, G: 0, B: 0, A: 255}),
				Padding:    style.Some(style.Edges(0, 1)),
				Order:      style.Some(3),
				Flex:       style.Some(style.Flex(1)),
				Border:     style.SingleBorder().Some(),
			}),
			element.Box("Second in DOM (Order 1)").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 0, G: 200, B: 0, A: 255}),
				Padding:    style.Some(style.Edges(0, 1)),
				Order:      style.Some(1),
				Flex:       style.Some(style.Flex(1)),
				Border:     style.SingleBorder().Some(),
			}),
			element.Box("Third in DOM (Order 2)").Style(style.Style{
				Background: style.Some[color.Color](color.RGBA{R: 0, G: 0, B: 200, A: 255}),
				Padding:    style.Some(style.Edges(0, 1)),
				Order:      style.Some(2),
				Flex:       style.Some(style.Flex(1)),
				Border:     style.SingleBorder().Some(),
			}),
		).Style(style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexRow),
			Background:    style.Some[color.Color](color.RGBA{R: 30, G: 60, B: 30, A: 255}),
			Width:         style.Some(style.Percent(100)),
			Padding:       style.Some(style.Edges(0, 2)),
			Gap:           style.Some(style.Gap(2)),
			Border:        style.SingleBorder().Some(),
		}),
	).Style(style.Style{
		Width:         style.Some(style.Percent(100)),
		Height:        style.Some(style.Percent(100)),
		Background:    style.Some[color.Color](color.RGBA{R: 15, G: 15, B: 15, A: 255}),
		Padding:       style.Some(style.Edges(1, 2)),
		FlexDirection: style.Some(style.FlexColumn),
		Display:       style.Some(style.DisplayFlex),
	})

	env.Mount(root)
	env.Flush()

	env.MatchGolden(t, "example_flex")
}

func TestExampleInputGolden(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	colCard := color.RGBA{R: 28, G: 30, B: 40, A: 255}      // card surface
	colBorder := color.RGBA{R: 60, G: 65, B: 90, A: 255}    // card border
	colInputBG := color.RGBA{R: 22, G: 24, B: 35, A: 255}   // input background
	colLabel := color.RGBA{R: 160, G: 165, B: 200, A: 255}  // label text
	colTitle := color.RGBA{R: 200, G: 210, B: 255, A: 255}  // title text
	colHint := color.RGBA{R: 80, G: 85, B: 110, A: 255}     // hint / footer text
	colAccent := color.RGBA{R: 100, G: 200, B: 130, A: 255} // echo value color
	colSeparator := color.RGBA{R: 45, G: 48, B: 65, A: 255} // horizontal rule

	const fieldWidth = 30

	echoText := element.Text("(empty)")

	usernameInp := element.NewInput(env.Engine.Document(), "")
	usernameInp.Style(style.Style{
		Width:      style.Some(style.Cells(fieldWidth)),
		Background: style.Some[color.Color](colInputBG),
		Foreground: style.Some[color.Color](color.RGBA{R: 220, G: 225, B: 255, A: 255}),
		Border:     style.SingleBorder().Color(colBorder).Some(),
		Padding:    style.Some(style.Edges(0, 1)),
	})

	passwordInp := element.NewInput(env.Engine.Document(), "")
	passwordInp.Style(style.Style{
		Width:      style.Some(style.Cells(fieldWidth)),
		Background: style.Some[color.Color](colInputBG),
		Foreground: style.Some[color.Color](color.RGBA{R: 220, G: 225, B: 255, A: 255}),
		Border:     style.SingleBorder().Color(colBorder).Some(),
		Padding:    style.Some(style.Edges(0, 1)),
	})

	usernameInp.AddEventListener(event.EventKeyDown, func(e event.Event) {
		v := usernameInp.Value()
		if v == "" {
			echoText.SetData("(empty)")
		} else {
			echoText.SetData(v)
		}
		env.Engine.RequestFrame()
	})

	root := element.Box(
		element.Box(
			element.Box("  Sign In  ").Style(style.Style{
				Foreground: style.Some[color.Color](colTitle),
				Bold:       style.Some(true),
				TextAlign:  style.Some(style.TextAlignCenter),
				Width:      style.Some(style.Percent(100)),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
			}),

			element.Box("").Style(style.Style{
				Width:      style.Some(style.Percent(100)),
				Height:     style.Some(style.Cells(1)),
				Background: style.Some[color.Color](colSeparator),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
			}),

			element.Box("Username").Style(style.Style{
				Foreground: style.Some[color.Color](colLabel),
				Margin:     style.Some(style.Edges(0, 0, 0, 0)),
			}),
			usernameInp,

			element.Box("").Style(style.Style{Height: style.Some(style.Cells(1))}),

			element.Box("Password").Style(style.Style{
				Foreground: style.Some[color.Color](colLabel),
				Margin:     style.Some(style.Edges(0, 0, 0, 0)),
			}),
			passwordInp,

			element.Box("").Style(style.Style{
				Width:      style.Some(style.Percent(100)),
				Height:     style.Some(style.Cells(1)),
				Background: style.Some[color.Color](colSeparator),
				Margin:     style.Some(style.Edges(1, 0, 0, 0)),
			}),

			element.Box(
				element.Span("Username value: ").Style(style.Style{
					Foreground: style.Some[color.Color](colHint),
				}),
				element.Span(echoText).Style(style.Style{
					Foreground: style.Some[color.Color](colAccent),
					Bold:       style.Some(true),
				}),
			).Style(style.Style{
				Margin: style.Some(style.Edges(1, 0, 0, 0)),
			}),

			element.Box(
				element.Span("Tab").Style(style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 60, G: 65, B: 90, A: 255}),
					Foreground: style.Some[color.Color](color.RGBA{R: 200, G: 210, B: 255, A: 255}),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" next field  ").Style(style.Style{
					Foreground: style.Some[color.Color](colHint),
				}),
				element.Span("Shift+Tab").Style(style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 60, G: 65, B: 90, A: 255}),
					Foreground: style.Some[color.Color](color.RGBA{R: 200, G: 210, B: 255, A: 255}),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" prev field  ").Style(style.Style{
					Foreground: style.Some[color.Color](colHint),
				}),
				element.Span("Q").Style(style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 60, G: 65, B: 90, A: 255}),
					Foreground: style.Some[color.Color](color.RGBA{R: 200, G: 210, B: 255, A: 255}),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" quit").Style(style.Style{
					Foreground: style.Some[color.Color](colHint),
				}),
			).Style(style.Style{
				Margin: style.Some(style.Edges(1, 0, 0, 0)),
			}),
		).Style(style.Style{
			Width:      style.Some(style.Cells(fieldWidth + 8)),
			Height:     style.Some(style.Auto),
			Background: style.Some[color.Color](colCard),
			Border:     style.SingleBorder().Color(colBorder).Some(),
			Padding:    style.Some(style.Edges(1, 2)),
			Margin:     style.Some(style.Edges(2, 2)),
		}),
	).Style(style.Style{
		Width:          style.Some(style.Percent(100)),
		Height:         style.Some(style.Percent(100)),
		Background:     style.Some[color.Color](color.RGBA{R: 18, G: 18, B: 23, A: 255}),
		Display:        style.Some(style.DisplayFlex),
		AlignItems:     style.Some(style.AlignCenter),
		JustifyContent: style.Some(style.JustifyCenter),
	})

	env.Mount(root)
	env.RenderFrame()
	env.Flush()

	env.MatchGolden(t, "example_input_default")

	// 1. Focus username and type
	env.Engine.FocusManager().Focus(usernameInp, focus.ReasonKeyboard)
	env.Type("KiteUser")
	env.Flush()

	env.MatchGolden(t, "example_input_typed")

	// 2. Clear username
	for range 8 {
		env.SendKey(key.Key{Code: key.KeyBackspace})
	}
	env.Type("Admin")
	env.Flush()

	env.MatchGolden(t, "example_input_cleared_retyped")
}

func TestExampleListGolden(t *testing.T) {
	env := testenv.Default(80, 40)
	defer env.Close()

	headerStyle := style.Style{Margin: style.Some(style.Edges(1, 0, 0, 0)), Underline: style.Some(true)}

	root := element.Box(
		element.Box(
			element.Box("Kite List Components Demonstration").Style(style.Style{
				TextAlign: style.Some(style.TextAlignCenter),
				Margin:    style.Some(style.Edges(0, 0, 1, 0)),
				Bold:      style.Some(true),
			}),

			// 1. Unordered List (Disc)
			element.Box("Unordered List (Default: Disc)").Style(headerStyle),
			element.UL(
				element.LI("First item"),
				element.LI("Second item with long text that should wrap around the marker correctly if the container is narrow enough."),
				element.LI("Third item"),
			),

			// 2. Ordered List (Decimal)
			element.Box("Ordered List (Default: Decimal)").Style(headerStyle),
			element.OL(
				element.LI("Initialize engine"),
				element.LI("Build DOM tree"),
				element.LI("Run frame loop"),
			),

			// 3. Custom Markers
			element.Box("Custom Markers (Square)").Style(headerStyle),
			element.UL(
				element.LI("Customized UL"),
				element.LI("Uses Square markers via inheritance"),
			).Style(style.Style{
				ListStyleType: style.Some(style.ListStyleSquare),
			}),

			// 4. Nested Lists
			element.Box("Nested Lists").Style(headerStyle),
			element.UL(
				element.LI(
					"Item with nested list:",
					element.OL(
						element.LI("Nested Step A"),
						element.LI("Nested Step B"),
					),
				),
				element.LI("Another parent item"),
			),
		).Style(style.Style{
			Width:   style.Some(style.Percent(90)),
			Margin:  style.Some(style.Edges(1, 0)),
			Padding: style.Some(style.Edges(1, 2)),
			Border:  style.SingleBorder().Some(),
		}),
	).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
		Padding:    style.Some(style.Edges(1, 2)),
		Background: style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 20, A: 255}),
	})

	env.Mount(root)
	env.Flush()

	env.MatchGolden(t, "example_list")
}

func TestExampleOverlayGolden(t *testing.T) {
	env := testenv.Default(80, 24)
	defer env.Close()

	// Create a main background element
	root := element.Box(
		element.Box("Overlay API Example").Style(style.Style{
			TextAlign:  style.Some(style.TextAlignCenter),
			Background: style.Some[color.Color](color.RGBA{R: 50, G: 50, B: 80, A: 255}),
			Padding:    style.Some(style.Edges(1)),
		}),
		element.Box(
			"Press 'o' to toggle the Overlay.",
			"\nPress 'q' or 'ctrl+c' to quit.",
		).Style(style.Style{
			Margin:  style.Some(style.Edges(2, 0)),
			Padding: style.Some(style.Edges(1, 2)),
			Border:  style.SingleBorder().Some(),
		}),
	).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
		Background: style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 30, A: 255}),
		Padding:    style.Some(style.Edges(2)),
	})

	env.Mount(root)
	env.Flush()

	env.MatchGolden(t, "example_overlay_closed")

	// Create the overlay element (a centered dialog)
	overlayDialog := element.Box(
		element.Box("I am an Overlay!").Style(style.Style{
			TextAlign: style.Some(style.TextAlignCenter),
			Margin:    style.Some(style.Edges(0, 0, 1, 0)),
			Bold:      style.Some(true),
		}),
		"I am rendered in the Top Layer,\nabove the normal document flow.",
		element.Box("Press 'o' to close me.").Style(style.Style{
			Margin:     style.Some(style.Edges(1, 0, 0, 0)),
			TextAlign:  style.Some(style.TextAlignCenter),
			Foreground: style.Some[color.Color](color.RGBA{R: 200, G: 200, B: 200, A: 255}),
		}),
	).Style(style.Style{
		Width:      style.Some(style.Cells(40)),
		Height:     style.Some(style.Cells(10)),
		Background: style.Some[color.Color](color.RGBA{R: 80, G: 40, B: 40, A: 255}),
		Border:     style.DoubleBorder().Color(color.RGBA{R: 255, G: 100, B: 100, A: 255}).Some(),
		Padding:    style.Some(style.Edges(1, 2)),
		// Centering the overlay
		Margin: style.Some(style.Edges(6, 20)), // Simple centering for 80x24
	})

	env.Engine.Document().ShowOverlay(overlayDialog, 100)
	env.Flush()

	env.MatchGolden(t, "example_overlay_opened")
}

func TestExampleTableGolden(t *testing.T) {
	env := testenv.Default(80, 50)
	defer env.Close()

	styles := map[string]style.Style{
		"title": {
			Width:  style.Some(style.Percent(100)),
			Border: style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 100, A: 255}).Some(),
		},
		"table": {
			Width:  style.Some(style.Percent(100)),
			Border: style.SingleBorder().Color(color.RGBA{R: 100, G: 255, B: 100, A: 255}).Some(),
		},
		"th": {
			Width:  style.Some(style.Percent(50)),
			Border: style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 100, A: 255}).Some(),
		},
		"tr": {
			Width:  style.Some(style.Percent(50)),
			Border: style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 100, A: 255}).Some(),
		},
		"name_cell": {
			Width:  style.Some(style.Cells(15)),
			Border: style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 100, A: 255}).Some(),
		},
		"role_cell": {
			Width:  style.Some(style.Cells(20)),
			Border: style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 100, A: 255}).Some(),
		},
		"cell": {
			Width:  style.Some(style.Percent(100)),
			Border: style.SingleBorder().Color(color.RGBA{R: 255, G: 255, B: 100, A: 255}).Some(),
		},
	}

	// Build UI declaratively
	ui := element.Box(
		// Table 1: Well-formed
		element.Box("Well-formed Table").Style(styles["title"]),
		element.Table(
			element.TR(
				element.TD("Name").Style(styles["name_cell"]),
				element.TD("Role").Style(styles["role_cell"]),
			).Style(styles["tr"]),
			element.TR(
				element.TD("Alice").Style(styles["cell"]),
				element.TD("Developer").Style(styles["cell"]),
			).Style(styles["tr"]),
			element.TR(
				element.TD("Total Users: 1 (Spanning)").
					Style(styles["cell"]).
					SetColSpan(2),
			).Style(styles["tr"]),
		).Style(style.Style{
			Width:  style.Some(style.Percent(100)),
			Border: style.SingleBorder().Color(color.RGBA{R: 100, G: 100, B: 255, A: 255}).Some(),
		}),

		// Table 2: Malformed Table
		element.Box("Malformed Table (Cells without Rows)").Style(style.Style{Margin: style.Some(style.Edges(1, 0))}),
		element.Table(
			// Directly add cells to table
			element.TD("Direct Cell 1").Style(style.Style{Width: style.Some(style.Cells(15))}),
			element.TD("Direct Cell 2").Style(style.Style{Width: style.Some(style.Cells(20))}),
		).Style(style.Style{
			Width:  style.Some(style.Percent(100)),
			Border: style.SingleBorder().Color(color.RGBA{R: 255, G: 100, B: 100, A: 255}).Some(),
		}),

		// Table 3: Grouped Table (thead, tbody, tfoot)
		element.Box("Grouped Table (thead, tbody, tfoot)").Style(style.Style{Margin: style.Some(style.Edges(1, 0))}),
		element.Table(
			element.THead(
				element.TR(
					element.TD("Header Col 1").Style(style.Style{Width: style.Some(style.Cells(15))}),
					element.TD("Header Col 2").Style(style.Style{Width: style.Some(style.Cells(20))}),
				),
			).Style(style.Style{
				Border: style.SingleBorder().Top(false).Right(false).Left(false).Some(),
			}),
			element.TBody(
				element.TR(
					element.TD("Body Row 1, C1"),
					element.TD("Body Row 1, C2"),
				),
				element.TR(
					element.TD("Body Row 2, C1"),
					element.TD("Body Row 2, C2"),
				),
			),
			element.TFoot(
				element.TR(
					element.TD("Footer 1"),
					element.TD("Footer 2"),
				),
			).Style(style.Style{
				Border: style.SingleBorder().Bottom(false).Right(false).Left(false).Some(),
			}),
		).Style(style.Style{
			Width:  style.Some(style.Percent(100)),
			Border: style.SingleBorder().Color(color.RGBA{R: 100, G: 255, B: 100, A: 255}).Some(),
		}),
	).Style(style.Style{
		Display:       style.Some(style.DisplayFlex),
		FlexDirection: style.Some(style.FlexColumn),
		Width:         style.Some(style.Percent(100)),
		Height:        style.Some(style.Percent(100)),
		Background:    style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 30, A: 255}),
		Padding:       style.Some(style.Edges(2, 4)),
		Gap:           style.Some(style.Gap(2, 0)),
	})

	env.Mount(ui)
	env.Flush()

	env.MatchGolden(t, "example_table")

	// Constrained height test to check table scrolling behavior
	env.Resize(80, 20)
	env.Flush()

	env.MatchGolden(t, "example_table_constrained_height")
}

func TestExampleTextAreaGolden(t *testing.T) {
	env := testenv.Default(80, 30)
	defer env.Close()

	colBG := color.RGBA{R: 18, G: 18, B: 23, A: 255}        // app background
	colCard := color.RGBA{R: 28, G: 30, B: 40, A: 255}      // editor surface
	colBorder := color.RGBA{R: 60, G: 65, B: 90, A: 255}    // editor border
	colTitle := color.RGBA{R: 200, G: 210, B: 255, A: 255}  // title text
	colStatus := color.RGBA{R: 100, G: 200, B: 130, A: 255} // status text
	colHint := color.RGBA{R: 80, G: 85, B: 110, A: 255}     // hint text

	initialText := "Welcome to Kite TextArea!\n\n" +
		"This component supports:\n" +
		" • Multi-line editing\n" +
		" • 2D Arrow navigation\n" +
		" • Automatic soft-wrap\n" +
		" • LongUnbreakableRunsThatEmergencyWrap"

	txa := element.NewTextArea(env.Engine.Document(), initialText)
	txa.Style(style.Style{
		Width:      style.Some(style.Cells(50)),
		Height:     style.Some(style.Cells(10)),
		Background: style.Some[color.Color](colCard),
		Foreground: style.Some[color.Color](color.RGBA{R: 220, G: 225, B: 255, A: 255}),
		Border:     style.SingleBorder().Color(colBorder).Some(),
		Padding:    style.Some(style.Edges(0, 1)),
	})

	statusText := element.Text("Pos: (0, 0)")
	updateStatus := func() {
		state := txa.CursorState()
		statusText.SetData(fmt.Sprintf("Pos: (%d, %d)", state.X, state.Y))
		env.Engine.RequestFrame()
	}

	txa.AddEventListener(event.EventKeyDown, func(e event.Event) {
		env.Engine.OnAfterLayout(updateStatus)
	})

	root := element.Box(
		element.Box(
			// Title
			element.Box("  Kite Text Editor  ").Style(style.Style{
				Foreground: style.Some[color.Color](colTitle),
				Bold:       style.Some(true),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
			}),

			// Editor
			txa,

			// Status Bar
			element.Box(
				element.Span("Status: ").Style(style.Style{Foreground: style.Some[color.Color](colHint)}),
				element.Span(statusText).Style(style.Style{Foreground: style.Some[color.Color](colStatus), Bold: style.Some(true)}),
			).Style(style.Style{
				Margin: style.Some(style.Edges(1, 0, 0, 0)),
			}),

			// Help Hints
			element.Box(
				element.Span(" Arrows").Style(style.Style{
					Background: style.Some[color.Color](colBorder),
					Foreground: style.Some[color.Color](colTitle),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" navigate  ").Style(style.Style{Foreground: style.Some[color.Color](colHint)}),
				element.Span(" Enter").Style(style.Style{
					Background: style.Some[color.Color](colBorder),
					Foreground: style.Some[color.Color](colTitle),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" newline  ").Style(style.Style{Foreground: style.Some[color.Color](colHint)}),
				element.Span(" Q").Style(style.Style{
					Background: style.Some[color.Color](colBorder),
					Foreground: style.Some[color.Color](colTitle),
					Padding:    style.Some(style.Edges(0, 1)),
				}),
				element.Span(" quit").Style(style.Style{Foreground: style.Some[color.Color](colHint)}),
			).Style(style.Style{
				Margin: style.Some(style.Edges(1, 0, 0, 0)),
			}),
		).Style(style.Style{
			Padding:       style.Some(style.Edges(1, 2)),
			Background:    style.Some[color.Color](colBG),
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexColumn),
			AlignItems:    style.Some(style.AlignStart),
		}),
	)

	env.Mount(root)
	updateStatus()
	env.Flush()

	env.MatchGolden(t, "example_textarea_default")

	// 1. Type at the end
	env.Engine.FocusManager().Focus(txa, focus.ReasonKeyboard)
	env.Type("\nAdding some new content here.")
	env.Flush()
	env.MatchGolden(t, "example_textarea_typed")

	// 2. Delete some content (backspace)
	for range 5 {
		env.SendKey(key.Key{Code: key.KeyBackspace})
	}
	env.Flush()
	env.MatchGolden(t, "example_textarea_deleted")

	// 3. Scroll content
	env.ScrollTo(txa, 0, 5)
	env.Flush()
	env.MatchGolden(t, "example_textarea_scrolled")
}
