package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/kitex/kitexdt"
	"github.com/masterkeysrd/kite/style"
)

// PubSub is a simple pub/sub subscription broker for demonstrating UseEffectCleanup.
type PubSub struct {
	mu   sync.Mutex
	subs []func(string)
}

func (ps *PubSub) Subscribe(fn func(string)) func() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.subs = append(ps.subs, fn)
	idx := len(ps.subs) - 1
	return func() {
		ps.mu.Lock()
		defer ps.mu.Unlock()
		ps.subs[idx] = nil
	}
}

func (ps *PubSub) Publish(msg string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for _, sub := range ps.subs {
		if sub != nil {
			sub(msg)
		}
	}
}

var globalBroker = &PubSub{}

// TimerComponent demonstrates UseEffectCleanup managing a background time ticker.
var TimerComponent = kitex.SimpleFC("TimerComponent", func() kitex.Node {
	getSeconds, setSeconds := kitex.UseState(0)
	getIsActive, setIsActive := kitex.UseState(true)

	// Manage the ticker lifetime with UseEffectCleanup.
	// Only runs/stops when active state changes.
	kitex.UseEffectCleanup(func() func() {
		if !getIsActive() {
			return func() {}
		}
		ticker := time.NewTicker(time.Second)
		done := make(chan struct{})

		go func() {
			for {
				select {
				case <-ticker.C:
					setSeconds(getSeconds() + 1)
				case <-done:
					return
				}
			}
		}()

		return func() {
			ticker.Stop()
			close(done)
		}
	}, []any{getIsActive()})

	statusText := "RUNNING"
	btnText := " Pause Timer "
	btnBg := color.RGBA{R: 220, G: 80, B: 80, A: 255}
	if !getIsActive() {
		statusText = "PAUSED"
		btnText = " Resume Timer "
		btnBg = color.RGBA{R: 80, G: 180, B: 100, A: 255}
	}

	return kitex.Box(kitex.BoxProps{
		Style: style.Style{
			Border:     style.SingleBorder().Some(),
			Padding:    style.Some(style.Edges(1, 1)),
			Margin:     style.Some(style.Edges(0, 0, 1, 0)),
			Background: style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 40, A: 255}),
		},
	},
		kitex.Box(kitex.BoxProps{
			Style: style.Style{Bold: style.Some(true), Margin: style.Some(style.Edges(0, 0, 1, 0))},
		}, kitex.Text("⏳ 1. UseEffect Timer Demonstration")),

		kitex.Box(kitex.BoxProps{
			Style: style.Style{Margin: style.Some(style.Edges(0, 0, 1, 0))},
		}, kitex.Text(fmt.Sprintf("Elapsed Time: %d seconds (%s)", getSeconds(), statusText))),

		kitex.Button(kitex.ButtonProps{
			OnClick: func(e event.Event) {
				setIsActive(!getIsActive())
			},
			Style: style.Style{
				Background: style.Some[color.Color](btnBg),
				Foreground: style.Some[color.Color](color.White),
			},
		}, kitex.Text(btnText)),
	)
})

// SubscriptionComponent demonstrates UseEffectCleanup managing a subscription to an external event source.
var SubscriptionComponent = kitex.SimpleFC("SubscriptionComponent", func() kitex.Node {
	getMessage, setMessage := kitex.UseState("No broadcasts received yet.")
	getMsgCount, setMsgCount := kitex.UseState(0)

	// Set up subscription when component mounts, unsubscribe when it unmounts.
	kitex.UseEffectCleanup(func() func() {
		unsub := globalBroker.Subscribe(func(msg string) {
			setMessage(msg)
			setMsgCount(getMsgCount() + 1)
		})
		return unsub
	}, []any{})

	return kitex.Box(kitex.BoxProps{
		Style: style.Style{
			Border:     style.SingleBorder().Some(),
			Padding:    style.Some(style.Edges(1, 1)),
			Margin:     style.Some(style.Edges(0, 0, 1, 0)),
			Background: style.Some[color.Color](color.RGBA{R: 30, G: 40, B: 30, A: 255}),
		},
	},
		kitex.Box(kitex.BoxProps{
			Style: style.Style{Bold: style.Some(true), Margin: style.Some(style.Edges(0, 0, 1, 0))},
		}, kitex.Text("📢 2. UseEffect Subscription Demonstration")),

		kitex.Box(kitex.BoxProps{
			Style: style.Style{Margin: style.Some(style.Edges(0, 0, 1, 0))},
		}, kitex.Text(fmt.Sprintf("Last Broadcast: %s", getMessage()))),

		kitex.Box(kitex.BoxProps{
			Style: style.Style{Margin: style.Some(style.Edges(0, 0, 1, 0))},
		}, kitex.Text(fmt.Sprintf("Total Messages Received: %d", getMsgCount()))),
	)
})

// App is the root container and coordinates publishing.
var App = kitex.SimpleFC("App", func() kitex.Node {
	// Let's allow toggling the SubscriptionComponent visibility to demonstrate the cleanup/unsubscribe on unmount!
	getShowSub, setShowSub := kitex.UseState(true)

	publishPhrases := []string{
		"Hello from the outer bounds of Kitex!",
		"Effects are working flawlessly in the event loop.",
		"Timers and subscriptions are running asynchronously.",
		"Zero engine changes were made for this effect system!",
		"Go-based Virtual DOM is responsive and reactive.",
	}

	handlePublish := func(e event.Event) {
		phrase := publishPhrases[rand.Intn(len(publishPhrases))]
		globalBroker.Publish(phrase)
	}

	return kitex.Box(kitex.BoxProps{
		Style: style.Style{
			Display:       style.Some(style.DisplayFlex),
			FlexDirection: style.Some(style.FlexColumn),
			Width:         style.Some(style.Percent(100)),
			Height:        style.Some(style.Percent(100)),
			Background:    style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 26, A: 255}),
			Padding:       style.Some(style.Edges(1, 2)),
		},
	},
		// Title
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Bold:       style.Some(true),
				Foreground: style.Some[color.Color](color.RGBA{R: 90, G: 140, B: 255, A: 255}),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
				TextAlign:  style.Some(style.TextAlignCenter),
			},
		}, kitex.Text("⚡ Kitex Effect Hooks & Lifecycle Demo ⚡")),

		// Subtitle / Info
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Foreground: style.Some[color.Color](color.RGBA{R: 160, G: 160, B: 180, A: 255}),
				Margin:     style.Some(style.Edges(0, 0, 1, 0)),
			},
		}, kitex.Text("Press 'q' to Quit. Pausing/Unmounting clean up background tickers and subscriptions.")),

		// Actions Panel
		kitex.Box(kitex.BoxProps{
			Style: style.Style{
				Display:       style.Some(style.DisplayFlex),
				FlexDirection: style.Some(style.FlexRow),
				Margin:        style.Some(style.Edges(0, 0, 1, 0)),
			},
		},
			kitex.Button(kitex.ButtonProps{
				OnClick: handlePublish,
				Style: style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 60, G: 120, B: 220, A: 255}),
					Foreground: style.Some[color.Color](color.White),
					Margin:     style.Some(style.Edges(0, 1)),
				},
			}, kitex.Text(" Broadcast Phrase ")),

			kitex.Button(kitex.ButtonProps{
				OnClick: func(e event.Event) {
					setShowSub(!getShowSub())
				},
				Style: style.Style{
					Background: style.Some[color.Color](color.RGBA{R: 160, G: 80, B: 220, A: 255}),
					Foreground: style.Some[color.Color](color.White),
					Margin:     style.Some(style.Edges(0, 1)),
				},
			}, kitex.Text(func() string {
				if getShowSub() {
					return " Unmount Subscription Component "
				}
				return " Mount Subscription Component "
			}())),
		),

		// Timer component always visible
		TimerComponent(),

		// Subscription component is toggleable
		kitex.If(getShowSub(), SubscriptionComponent()),
	)
})

func main() {
	f, _ := os.Create("kitex_effects_demo.log")
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

	kitex.EnableDevMode = true

	// Mount VDOM into host container
	kitex.Render(App(), container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
		}
	})

	insp, _ := devtools.Install(eng, devtools.Options{})
	kitexdt.Register(insp)

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
