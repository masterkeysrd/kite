package main

import (
	"image/color"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/stage"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	stg := stage.New()

	// 1. Button component scenes
	stg.Register("Button", []stage.Scene{
		{
			Name: "Interactive Button",
			Render: func(c *stage.Context) kitex.Node {
				label := c.Text("Label Text", "Submit Form")
				disabled := c.Bool("Disabled", false)

				btnStyle := style.S().
					Foreground(color.RGBA{R: 255, G: 255, B: 255, A: 255}).
					Padding(0, 2)
				if disabled {
					btnStyle = btnStyle.Background(color.RGBA{R: 107, G: 114, B: 128, A: 255}) // Gray 500
				} else {
					btnStyle = btnStyle.Background(color.RGBA{R: 79, G: 70, B: 229, A: 255}) // Indigo 600
				}

				return kitex.Button(
					kitex.ButtonProps{
						Disabled: disabled,
						Style:    btnStyle,
						OnClick: func(e event.Event) {
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
							Background(color.RGBA{R: 220, G: 38, B: 38, A: 255}). // Red 600
							Foreground(color.RGBA{R: 255, G: 255, B: 255, A: 255}).
							Padding(0, 2),
						OnClick: func(e event.Event) {
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
							Border(true, style.BorderSingle, color.RGBA{R: 156, G: 163, B: 175, A: 255}),
						OnChange: func(e event.Event) {
							if ie, ok := e.(*event.InputEvent); ok {
								c.Log("Input value changed to: " + ie.Value)
							} else if ce, ok := e.(*event.ChangeEvent); ok {
								c.Log("Input value changed to: " + ce.Value)
							}
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

				badgeStyle := style.S().Padding(0, 2).Foreground(color.RGBA{R: 255, G: 255, B: 255, A: 255})
				switch selectedStatus {
				case "Success":
					badgeStyle = badgeStyle.Background(color.RGBA{R: 22, G: 163, B: 74, A: 255}) // Green 600
				case "Warning":
					badgeStyle = badgeStyle.Background(color.RGBA{R: 202, G: 138, B: 4, A: 255}) // Yellow 600
				case "Error":
					badgeStyle = badgeStyle.Background(color.RGBA{R: 220, G: 38, B: 38, A: 255}) // Red 600
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

	stg.Run()
}
