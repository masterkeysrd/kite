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

var (
	cardAStyle           = style.S().Padding(1, 2).Background(color.RGBA{R: 35, G: 38, B: 55, A: 255}).Border(style.SingleBorder()).Width(style.Percent(45)).Margin(0, 1)
	cardATitleStyle      = style.S().Bold(true).Foreground(color.RGBA{R: 90, G: 180, B: 255, A: 255})
	cardBodyStyle        = style.S().Margin(1, 0)
	cardRendersTextStyle = style.S().Foreground(color.RGBA{R: 160, G: 160, B: 180, A: 255})
	cardBStyle           = style.S().Padding(1, 2).Background(color.RGBA{R: 35, G: 55, B: 38, A: 255}).Border(style.SingleBorder()).Width(style.Percent(45)).Margin(0, 1)
	cardBTitleStyle      = style.S().Bold(true).Foreground(color.RGBA{R: 120, G: 220, B: 140, A: 255})
	appContainerStyle    = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 18, G: 18, B: 24, A: 255}).Padding(1, 2)
	appHeaderStyle       = style.S().Bold(true).Foreground(color.RGBA{R: 240, G: 190, B: 90, A: 255}).Margin(0, 0, 1, 0).TextAlign(style.TextAlignCenter)
	appDescriptionStyle  = style.S().Foreground(color.RGBA{R: 180, G: 180, B: 200, A: 255}).Margin(0, 0, 2, 0).TextAlign(style.TextAlignCenter)
	cardsRowStyle        = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).JustifyContent(style.JustifyAround).Margin(0, 0, 2, 0)
	buttonsRowStyle      = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).JustifyContent(style.JustifyCenter)
	btnAStyle            = style.S().Background(color.RGBA{R: 50, G: 120, B: 220, A: 255}).Foreground(color.White).Margin(0, 1)
	btnBStyle            = style.S().Background(color.RGBA{R: 50, G: 180, B: 100, A: 255}).Foreground(color.White).Margin(0, 1)
	btnAsyncStyle        = style.S().Background(color.RGBA{R: 180, G: 80, B: 220, A: 255}).Foreground(color.White).Margin(0, 1)
	footerStyle          = style.S().Foreground(color.RGBA{R: 120, G: 120, B: 120, A: 255}).Margin(2, 0, 0, 0).TextAlign(style.TextAlignCenter)
	rootStyle            = style.S().Width(style.Percent(100)).Height(style.Percent(100))
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
		Style: cardAStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: cardATitleStyle,
		}, kitex.Text("Component A (Slices CountA)")),
		kitex.Box(kitex.BoxProps{
			Style: cardBodyStyle,
		}, kitex.Text(fmt.Sprintf("Count A: %d", countA))),
		kitex.Box(kitex.BoxProps{
			Style: cardRendersTextStyle,
		}, kitex.Text(fmt.Sprintf("Renders: %d", renderCountA))),
	)
})

var CompB = kitex.SimpleFC("CompB", func() kitex.Node {
	renderCountB++

	countB := kites.Use(store, func(s AppState) int {
		return s.CountB
	})

	return kitex.Box(kitex.BoxProps{
		Style: cardBStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: cardBTitleStyle,
		}, kitex.Text("Component B (Slices CountB)")),
		kitex.Box(kitex.BoxProps{
			Style: cardBodyStyle,
		}, kitex.Text(fmt.Sprintf("Count B: %d", countB))),
		kitex.Box(kitex.BoxProps{
			Style: cardRendersTextStyle,
		}, kitex.Text(fmt.Sprintf("Renders: %d", renderCountB))),
	)
})

var App = kitex.SimpleFC("App", func() kitex.Node {
	return kitex.Box(kitex.BoxProps{
		Style: appContainerStyle,
	},
		// Title
		kitex.Box(kitex.BoxProps{
			Style: appHeaderStyle,
		}, kitex.Text("🛰️ Kite Global State Manager Demo (extras/kites)")),

		// Description
		kitex.Box(kitex.BoxProps{
			Style: appDescriptionStyle,
		}, kitex.Text("This app proves selector-based re-rendering. Incrementing A doesn't trigger B, and vice-versa.\nBackground routine will automatically increment CountA every 2s.")),

		// Components side-by-side
		kitex.Box(kitex.BoxProps{
			Style: cardsRowStyle,
		}, CompA(), CompB()),

		// Buttons control panel
		kitex.Box(kitex.BoxProps{
			Style: buttonsRowStyle,
		},
			kitex.Button(kitex.ButtonProps{
				OnClick: func(e event.Event) {
					store.Set(func(s AppState) AppState {
						s.CountA++
						return s
					})
				},
				Style: btnAStyle,
			}, kitex.Text(" Increment A ")),

			kitex.Button(kitex.ButtonProps{
				OnClick: func(e event.Event) {
					store.Set(func(s AppState) AppState {
						s.CountB++
						return s
					})
				},
				Style: btnBStyle,
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
				Style: btnAsyncStyle,
			}, kitex.Text(" Async +10 B (500ms delay) ")),
		),

		kitex.Box(kitex.BoxProps{
			Style: footerStyle,
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
	container.Style(rootStyle)
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
