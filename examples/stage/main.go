package main

import (
	"fmt"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/stage"
	"github.com/masterkeysrd/kite/style"
)

type CardProps struct {
	Title       string `stage:"label:Card Title;default:Welcome to Kite"`
	Subtitle    string `stage:"label:Card Subtitle;default:A fast TUI library"`
	ShowAction  bool   `stage:"label:Show Action Button;default:true"`
	BorderWidth int    `stage:"label:Border Width;min:1;max:4;default:1"`
	Theme       string `stage:"select:Ocean,Forest,Sunset;default:Ocean"`
}

func main() {
	stg := stage.New()

	// Register global controls (toolbar items)
	stg.GlobalSelect("Theme", []string{"Light", "Dark"}, "Dark")
	stg.GlobalBool("Show Border", true)

	// Context-aware decorator that wraps all rendered scene nodes
	stg.WithContextDecorator(func(c *stage.Context, node kitex.Node) kitex.Node {
		theme := c.GlobalString("Theme", "Dark")
		showBorder := c.GlobalBool("Show Border", true)

		// Stylize the decorator based on current global options
		bgColor := style.RGB(17, 24, 39)    // Dark gray 900
		fgColor := style.RGB(243, 244, 246) // Light gray 100
		if theme == "Light" {
			bgColor = style.RGB(243, 244, 246) // Light gray 100
			fgColor = style.RGB(17, 24, 39)    // Dark gray 900
		}

		wrapperStyle := style.S().
			Padding(2).
			Background(bgColor).
			Foreground(fgColor)

		if showBorder {
			wrapperStyle = wrapperStyle.
				Border(true, style.BorderDouble, style.RGB(79, 70, 229)) // Indigo 600
		}

		return kitex.Box(
			kitex.BoxProps{
				Style: wrapperStyle,
			},
			node,
		)
	})

	// 1. Button component scenes
	stg.Register("DesignSystem/Buttons/Button", []stage.Scene{
		{
			Name: "Interactive Button",
			Render: func(c *stage.Context) kitex.Node {
				label := c.Text("Label Text", "Submit Form")
				disabled := c.Bool("Disabled", false)

				btnStyle := style.S().
					Padding(0, 2).
					Foreground(style.RGB(255, 255, 255))
				if disabled {
					btnStyle = btnStyle.Background(style.RGB(107, 114, 128)) // Gray 500
				} else {
					btnStyle = btnStyle.Background(style.RGB(79, 70, 229)) // Indigo 600
				}

				return kitex.Button(
					kitex.ButtonProps{
						Disabled: disabled,
						Style:    btnStyle,
						OnClick: func(e *event.MouseEvent) {
							c.Log("Primary Button clicked!")
						},
					},
					kitex.Text(label),
				)
			},
		},
		{
			Name: "Danger Variation",
			Render: func(c *stage.Context) kitex.Node {
				label := c.Text("Danger Label", "Delete Account")

				return kitex.Button(
					kitex.ButtonProps{
						Style: style.S().
							Padding(0, 2).
							Background(style.RGB(220, 38, 38)). // Red 600
							Foreground(style.RGB(255, 255, 255)),
						OnClick: func(e *event.MouseEvent) {
							c.Log("Danger Button clicked!")
						},
					},
					kitex.Text(label),
				)
			},
		},
	})

	// 2. Input component scenes
	stg.Register("Text Input", []stage.Scene{
		{
			Name: "Basic Input",
			Render: func(c *stage.Context) kitex.Node {
				placeholder := c.Text("Placeholder Text", "Enter username...")
				val := c.Text("Value String", "")

				return kitex.Box(
					kitex.BoxProps{
						Style: style.S().
							Display(style.DisplayFlex).
							FlexDirection(style.FlexColumn).
							Width(style.Cells(40)),
					},
					kitex.Input(kitex.InputProps{
						Value:       val,
						Placeholder: placeholder,
						Style: style.S().
							Width(style.Percent(100)).
							Padding(0, 1).
							Border(true, style.BorderSingle, style.RGB(156, 163, 175)),
						OnValueChange: func(val string) {
							c.Log("Input value changed to: " + val)
						},
					}),
				)
			},
		},
	})

	// 3. Dropdown-style Select component scenes
	stg.Register("Status Badges", []stage.Scene{
		{
			Name: "Themed Badges",
			Render: func(c *stage.Context) kitex.Node {
				statusOptions := []string{"Success", "Warning", "Error"}
				selectedStatus := c.Select("Status Selection", statusOptions, "Success")

				badgeStyle := style.S().
					Padding(0, 2).
					Foreground(style.RGB(255, 255, 255))
				switch selectedStatus {
				case "Success":
					badgeStyle = badgeStyle.Background(style.RGB(22, 163, 74)) // Green 600
				case "Warning":
					badgeStyle = badgeStyle.Background(style.RGB(202, 138, 4)) // Yellow 600
				case "Error":
					badgeStyle = badgeStyle.Background(style.RGB(220, 38, 38)) // Red 600
				}

				return kitex.Box(
					kitex.BoxProps{
						Style: badgeStyle,
					},
					kitex.Text(selectedStatus),
				)
			},
		},
	})

	// 4. Counter component — showcases the Int control
	stg.Register("Counter", []stage.Scene{
		{
			Name: "Basic Counter",
			Render: func(c *stage.Context) kitex.Node {
				count := c.Int("Count", 0)

				countStyle := style.S().
					Padding(0, 3).
					Background(style.RGB(79, 70, 229)). // Indigo 600
					Foreground(style.RGB(255, 255, 255))

				return kitex.Box(
					kitex.BoxProps{
						Style: style.S().
							Display(style.DisplayFlex).
							FlexDirection(style.FlexColumn).
							AlignItems(style.AlignCenter),
					},
					kitex.Box(
						kitex.BoxProps{
							Style: style.S().
								MarginBottom(1).
								Foreground(style.RGB(156, 163, 175)),
						},
						kitex.Text("Current Count"),
					),
					kitex.Box(
						kitex.BoxProps{Style: countStyle},
						kitex.Text(fmt.Sprintf("%d", count)),
					),
					kitex.Box(
						kitex.BoxProps{
							Style: style.S().MarginTop(1),
						},
						kitex.Button(
							kitex.ButtonProps{
								Style: style.S().
									Padding(0, 2).
									Border(false).
									Background(style.RGB(16, 185, 129)). // Emerald 500
									Foreground(style.RGB(255, 255, 255)),
								OnClick: func(e *event.MouseEvent) {
									c.Log(fmt.Sprintf("Count is %d", count))
								},
							},
							kitex.Text("Log Value"),
						),
					),
				)
			},
		},
		{
			Name: "Progress Bar",
			Render: func(c *stage.Context) kitex.Node {
				progress := c.Int("Progress (0-100)", 42)
				progress = max(0, min(100, progress))

				barColor := style.RGB(22, 163, 74) // Green 600
				switch {
				case progress < 34:
					barColor = style.RGB(220, 38, 38) // Red 600
				case progress < 67:
					barColor = style.RGB(202, 138, 4) // Yellow 600
				}

				return kitex.Box(
					kitex.BoxProps{
						Style: style.S().
							Display(style.DisplayFlex).
							FlexDirection(style.FlexColumn).
							Width(style.Cells(40)),
					},
					kitex.Box(
						kitex.BoxProps{
							Style: style.S().
								Display(style.DisplayFlex).
								FlexDirection(style.FlexRow).
								JustifyContent(style.JustifyBetween).
								MarginBottom(1),
						},
						kitex.Text("Progress"),
						kitex.Text(fmt.Sprintf("%d%%", progress)),
					),
					// Track background
					kitex.Box(
						kitex.BoxProps{
							Style: style.S().
								Width(style.Percent(100)).
								Background(style.RGB(55, 65, 81)), // Gray 700
						},
						// Filled portion
						kitex.Box(
							kitex.BoxProps{
								Style: style.S().
									Width(style.Percent(float32(progress))).
									Background(barColor),
							},
							kitex.Text(" "),
						),
					),
				)
			},
		},
	})

	// Create a sub-registry (simulating an external package registry)
	cardRegistry := stage.NewRegistry()

	// Register our Card component using the new generic reflection-based Register function!
	stage.Register(cardRegistry, stage.ComponentConfig[CardProps]{
		Name: "Card",
		DefaultProps: CardProps{
			Title:      "Kite TUI",
			Subtitle:   "Sleek and terminal-native",
			ShowAction: true,
			Theme:      "Ocean",
		},
		Render: func(c *stage.Context, props CardProps) kitex.Node {
			themeColor := style.RGBA(0, 0, 0, 0)
			switch props.Theme {
			case "Ocean":
				themeColor = style.RGB(59, 130, 246) // Blue 500
			case "Sunset":
				themeColor = style.RGB(239, 68, 68) // Red 500
			case "Forest":
				themeColor = style.RGB(16, 185, 129) // Green 500
			}

			cardStyle := style.S().
				Padding(1).
				Border(true, style.BorderSingle, themeColor)

			var children []kitex.Node
			children = append(children, kitex.Box(
				kitex.BoxProps{
					Style: style.S().
						Foreground(themeColor).
						Bold(true),
				},
				kitex.Text(props.Title),
			))
			children = append(children, kitex.Text(props.Subtitle))

			if props.ShowAction {
				children = append(children, kitex.Box(
					kitex.BoxProps{Style: style.S().Height(style.Cells(1))}, // Spacing
				))
				children = append(children, kitex.Button(
					kitex.ButtonProps{
						Style: style.S().
							Padding(0, 1).
							Background(themeColor).
							Foreground(style.RGB(255, 255, 255)),
						OnClick: func(e *event.MouseEvent) {
							c.Log("Card action clicked!")
						},
					},
					kitex.Text("Action"),
				))
			}

			return kitex.Box(kitex.BoxProps{Style: cardStyle}, children...)
		},
		Scenes: []stage.SceneConfig[CardProps]{
			{
				Name: "Compact Variation",
				Props: CardProps{
					Title:      "Compact Card",
					Subtitle:   "Minimal variation",
					ShowAction: false,
					Theme:      "Forest",
				},
			},
		},
	})

	// Merge the sub-registry under a clean namespace
	stg.Merge("DesignSystem/Layouts", cardRegistry)

	stg.Run()
}
