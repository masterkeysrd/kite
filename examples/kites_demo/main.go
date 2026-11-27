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
	cardAStyle = style.S().
			Width(style.Percent(45)).
			Margin(0, 1).
			Padding(1, 2).
			Border(style.SingleBorder()).
			Background(style.RGB(35, 38, 55))

	cardATitleStyle = style.S().
			Foreground(style.RGB(90, 180, 255)).
			Bold(true)

	cardBodyStyle        = style.S().Margin(1, 0)
	cardRendersTextStyle = style.S().Foreground(style.RGB(160, 160, 180))

	cardBStyle = style.S().
			Width(style.Percent(45)).
			Margin(0, 1).
			Padding(1, 2).
			Border(style.SingleBorder()).
			Background(style.RGB(35, 55, 38))

	cardBTitleStyle = style.S().
			Foreground(style.RGB(120, 220, 140)).
			Bold(true)

	appContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Width(style.Percent(100)).
				Height(style.Percent(100)).
				Padding(1, 2).
				Background(style.RGB(18, 18, 24))

	appHeaderStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.RGB(240, 190, 90)).
			Bold(true).
			TextAlign(style.TextAlignCenter)

	appDescriptionStyle = style.S().
				Margin(0, 0, 2, 0).
				Foreground(style.RGB(180, 180, 200)).
				TextAlign(style.TextAlignCenter)

	cardsRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			JustifyContent(style.JustifyAround).
			Margin(0, 0, 2, 0)

	buttonsRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			JustifyContent(style.JustifyCenter)

	btnAStyle = style.S().
			Margin(0, 1).
			Background(style.RGB(50, 120, 220)).
			Foreground(color.White)

	btnBStyle = style.S().
			Margin(0, 1).
			Background(style.RGB(50, 180, 100)).
			Foreground(color.White)

	btnAsyncStyle = style.S().
			Margin(0, 1).
			Background(style.RGB(180, 80, 220)).
			Foreground(color.White)

	footerStyle = style.S().
			Margin(2, 0, 0, 0).
			Foreground(style.RGB(120, 120, 120)).
			TextAlign(style.TextAlignCenter)

	rootStyle = style.S().
			Width(style.Percent(100)).
			Height(style.Percent(100))
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

var CompA = kitex.FC("CompA", func(_ kitex.NoProps) kitex.Node {
	renderCountA++

	countA := kites.Use(store, func(s AppState) int {
		return s.CountA
	})

	return kitex.Box.Style(
		cardAStyle,
		kitex.Box.Style(cardATitleStyle, kitex.Text("Component A (Slices CountA)")),
		kitex.Box.Style(cardBodyStyle, kitex.Text(fmt.Sprintf("Count A: %d", countA))),
		kitex.Box.Style(cardRendersTextStyle, kitex.Text(fmt.Sprintf("Renders: %d", renderCountA))),
	)
})

var CompB = kitex.FC("CompB", func(_ kitex.NoProps) kitex.Node {
	renderCountB++

	countB := kites.Use(store, func(s AppState) int {
		return s.CountB
	})

	return kitex.Box.Style(
		cardBStyle,
		kitex.Box.Style(cardBTitleStyle, kitex.Text("Component B (Slices CountB)")),
		kitex.Box.Style(cardBodyStyle, kitex.Text(fmt.Sprintf("Count B: %d", countB))),
		kitex.Box.Style(cardRendersTextStyle, kitex.Text(fmt.Sprintf("Renders: %d", renderCountB))),
	)
})

var App = kitex.FC("App", func(_ kitex.NoProps) kitex.Node {
	return kitex.Box.Style(
		appContainerStyle,
		// Title
		kitex.Box.Style(appHeaderStyle, kitex.Text("🛰️ Kite Global State Manager Demo (extras/kites)")),

		// Description
		kitex.Box.Style(appDescriptionStyle, kitex.Text("This app proves selector-based re-rendering. Incrementing A doesn't trigger B, and vice-versa.\nBackground routine will automatically increment CountA every 2s.")),

		// Components side-by-side
		kitex.Box.Style(cardsRowStyle, CompA(), CompB()),

		// Buttons control panel
		kitex.Box.Style(
			buttonsRowStyle,
			kitex.Button(kitex.ButtonProps{
				OnClick: func(e *event.MouseEvent) {
					store.Set(func(s AppState) AppState {
						s.CountA++
						return s
					})
				},
				Style: btnAStyle,
			}, kitex.Text(" Increment A ")),

			kitex.Button(kitex.ButtonProps{
				OnClick: func(e *event.MouseEvent) {
					store.Set(func(s AppState) AppState {
						s.CountB++
						return s
					})
				},
				Style: btnBStyle,
			}, kitex.Text(" Increment B ")),

			kitex.Button(kitex.ButtonProps{
				OnClick: func(e *event.MouseEvent) {
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

		kitex.Box.Style(footerStyle, kitex.Text("Press 'q' or 'ctrl+c' to quit.")),
	)
})

func main() {
	f, _ := os.Create("kites_demo.log")
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
