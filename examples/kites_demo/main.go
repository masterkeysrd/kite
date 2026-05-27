package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"time"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kites"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/style"
)

type AppState struct {
	CountA int
	CountB int
}

var (
	store        = kites.Create(AppState{CountA: 0, CountB: 0})
	renderCountA int
	renderCountB int
)

var CompA = kitex.SimpleFC("CompA", func() kitex.Node {
	renderCountA++

	countA := kites.Use(store, func(s AppState) int {
		return s.CountA
	})

	return kitex.Box(kitex.BoxProps{
		Style: style.Style{
			Padding:    style.Some(style.Edges(1, 2)),
			Background: style.Some[color.Color](color.RGBA{R: 35, G: 38, B: 55, A: 255}),
			Border:     style.SingleBorder().Some(),
			Width:      style.Some(style.Percent(45)),
			Margin:     style.Some(style.Edges(0, 1)),
		},
	},
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Bold:       style.Some(true),
				Foreground: style.Some[color.Color](color.RGBA{R: 90, G: 180, B: 255, A: 255}),
			},
		}, kitex.Text("Component A (Slices CountA)")),
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Margin: style.Some(style.Edges(1, 0)),
			},
		}, kitex.Text(fmt.Sprintf("Count A: %d", countA))),
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Foreground: style.Some[color.Color](color.RGBA{R: 160, G: 160, B: 180, A: 255}),
			},
		}, kitex.Text(fmt.Sprintf("Renders: %d", renderCountA))),
	)
})

var CompB = kitex.SimpleFC("CompB", func() kitex.Node {
	renderCountB++

	countB := kites.Use(store, func(s AppState) int {
		return s.CountB
	})

	return kitex.Box(kitex.BoxProps{
		Style: style.Style{
			Padding:    style.Some(style.Edges(1, 2)),
			Background: style.Some[color.Color](color.RGBA{R: 35, G: 55, B: 38, A: 255}),
			Border:     style.SingleBorder().Some(),
			Width:      style.Some(style.Percent(45)),
			Margin:     style.Some(style.Edges(0, 1)),
		},
	},
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Bold:       style.Some(true),
				Foreground: style.Some[color.Color](color.RGBA{R: 120, G: 220, B: 140, A: 255}),
			},
		}, kitex.Text("Component B (Slices CountB)")),
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Margin: style.Some(style.Edges(1, 0)),
			},
		}, kitex.Text(fmt.Sprintf("Count B: %d", countB))),
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Foreground: style.Some[color.Color](color.RGBA{R: 160, G: 160, B: 180, A: 255}),
			},
		}, kitex.Text(fmt.Sprintf("Renders: %d", renderCountB))),
	)
})

var App = kitex.SimpleFC("App", func() kitex.Node {
	return kitex.Box(kitex.BoxProps{
		Style: style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexColumn),
			Width:         style.Some(style.Percent(100)),
			Height:        style.Some(style.Percent(100)),
			Background:    style.Some[color.Color](color.RGBA{R: 18, G: 18, B: 24, A: 255}),
			Padding:       style.Some(style.Edges(1, 2)),
		},
	},
		// Title
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Bold:       style.Some(true),
				Foreground: style.Some[color.Color](color.RGBA{R: 240, G: 190, B: 90, A: 255}),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
				TextAlign:  style.Some(style.TextAlignCenter),
			},
		}, kitex.Text("🛰️ Kite Global State Manager Demo (extras/kites)")),

		// Description
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Foreground: style.Some[color.Color](color.RGBA{R: 180, G: 180, B: 200, A: 255}),
				Margin:     style.Some(style.Edges(0, 0, 2, 0)),
				TextAlign:  style.Some(style.TextAlignCenter),
			},
		}, kitex.Text("This app proves selector-based re-rendering. Incrementing A doesn't trigger B, and vice-versa.\nBackground routine will automatically increment CountA every 2s.")),

		// Components side-by-side
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Display:        style.Some(style.DisplayFlex),
				FlexDirection:  style.Some(style.FlexRow),
				JustifyContent: style.Some(style.JustifyAround),
				Margin:         style.Some(style.Edges(0, 0, 2, 0)),
			},
		}, CompA(), CompB()),

		// Buttons control panel
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Display:        style.Some(style.DisplayFlex),
				FlexDirection:  style.Some(style.FlexRow),
				JustifyContent: style.Some(style.JustifyCenter),
			},
		},
			kitex.Button(kitex.ButtonProps{
				OnClick: func(e event.Event) {
					store.Set(func(s AppState) AppState {
						s.CountA++
						return s
					})
				},
				Style: style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 50, G: 120, B: 220, A: 255}),
					Foreground: style.Some[color.Color](color.White),
					Margin:     style.Some(style.Edges(0, 1)),
				},
			}, kitex.Text(" Increment A ")),

			kitex.Button(kitex.ButtonProps{
				OnClick: func(e event.Event) {
					store.Set(func(s AppState) AppState {
						s.CountB++
						return s
					})
				},
				Style: style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 50, G: 180, B: 100, A: 255}),
					Foreground: style.Some[color.Color](color.White),
					Margin:     style.Some(style.Edges(0, 1)),
				},
			}, kitex.Text(" Increment B ")),

			kitex.Button(kitex.ButtonProps{
				OnClick: func(e event.Event) {
					go func() {
						time.Sleep(500 * time.Millisecond)
						store.Set(func(s AppState) AppState {
							s.CountB += 10
							return s
						})
					}()
				},
				Style: style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 180, G: 80, B: 220, A: 255}),
					Foreground: style.Some[color.Color](color.White),
					Margin:     style.Some(style.Edges(0, 1)),
				},
			}, kitex.Text(" Async +10 B (500ms delay) ")),
		),

		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Foreground: style.Some[color.Color](color.RGBA{R: 120, G: 120, B: 120, A: 255}),
				Margin:     style.Some(style.Edges(2, 0, 0, 0)),
				TextAlign:  style.Some(style.TextAlignCenter),
			},
		}, kitex.Text("Press 'q' or 'ctrl+c' to quit.")),
	)
})

func main() {
	f, _ := os.Create("kites_demo.log")
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
		Width:  style.Some(style.Percent(100)),
		Height: style.Some(style.Percent(100)),
	})
	eng.Mount(container)

	kitex.Render(App(), container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
		}
	})

	// Background ticker to automatically update CountA every 2s
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				store.Set(func(s AppState) AppState {
					s.CountA++
					return s
				})
			}
		}
	}()

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
