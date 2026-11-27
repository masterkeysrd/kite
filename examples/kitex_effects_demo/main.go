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

var (
	timerBoxStyle = style.S().
			Margin(0, 0, 1, 0).
			Padding(1, 1).
			Border(style.SingleBorder()).
			Background(style.RGB(30, 30, 40))

	sectionHeaderStyle = style.S().
				Margin(0, 0, 1, 0).
				Bold(true)

	contentRowStyle = style.S().Margin(0, 0, 1, 0)

	subscriptionBoxStyle = style.S().
				Margin(0, 0, 1, 0).
				Padding(1, 1).
				Border(style.SingleBorder()).
				Background(style.RGB(30, 40, 30))

	appContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Width(style.Percent(100)).
				Height(style.Percent(100)).
				Padding(1, 2).
				Background(style.RGB(20, 20, 26))

	appTitleStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.RGB(90, 140, 255)).
			Bold(true).
			TextAlign(style.TextAlignCenter)

	appDescriptionStyle = style.S().
				Margin(0, 0, 1, 0).
				Foreground(style.RGB(160, 160, 180))

	actionsPanelStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				Margin(0, 0, 1, 0)

	broadcastBtnStyle = style.S().
				Margin(0, 1).
				Background(style.RGB(60, 120, 220)).
				Foreground(color.White)

	toggleBtnStyle = style.S().
			Margin(0, 1).
			Background(style.RGB(160, 80, 220)).
			Foreground(color.White)

	rootStyle = style.S().
			Width(style.Percent(100)).
			Height(style.Percent(100))
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
	btnBg := style.RGB(220, 80, 80)
	if !getIsActive() {
		statusText = "PAUSED"
		btnText = " Resume Timer "
		btnBg = style.RGB(80, 180, 100)
	}

	return kitex.Box(kitex.BoxProps{
		Style: timerBoxStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: sectionHeaderStyle,
		}, kitex.Text("⏳ 1. UseEffect Timer Demonstration")),

		kitex.Box(kitex.BoxProps{
			Style: contentRowStyle,
		}, kitex.Text(fmt.Sprintf("Elapsed Time: %d seconds (%s)", getSeconds(), statusText))),

		kitex.Button(kitex.ButtonProps{
			OnClick: func(e *event.MouseEvent) {
				setIsActive(!getIsActive())
			},
			Style: style.S().
				Background(btnBg).
				Foreground(color.White),
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
		Style: subscriptionBoxStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: sectionHeaderStyle,
		}, kitex.Text("📢 2. UseEffect Subscription Demonstration")),

		kitex.Box(kitex.BoxProps{
			Style: contentRowStyle,
		}, kitex.Text(fmt.Sprintf("Last Broadcast: %s", getMessage()))),

		kitex.Box(kitex.BoxProps{
			Style: contentRowStyle,
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

	handlePublish := func(e *event.MouseEvent) {
		phrase := publishPhrases[rand.Intn(len(publishPhrases))]
		globalBroker.Publish(phrase)
	}

	return kitex.Box(kitex.BoxProps{
		Style: appContainerStyle,
	},
		// Title
		kitex.Box(kitex.BoxProps{
			Style: appTitleStyle,
		}, kitex.Text("⚡ Kitex Effect Hooks & Lifecycle Demo ⚡")),

		// Subtitle / Info
		kitex.Box(kitex.BoxProps{
			Style: appDescriptionStyle,
		}, kitex.Text("Press 'q' to Quit. Pausing/Unmounting clean up background tickers and subscriptions.")),

		// Actions Panel
		kitex.Box(kitex.BoxProps{
			Style: actionsPanelStyle,
		},
			kitex.Button(kitex.ButtonProps{
				OnClick: handlePublish,
				Style:   broadcastBtnStyle,
			}, kitex.Text(" Broadcast Phrase ")),

			kitex.Button(kitex.ButtonProps{
				OnClick: func(e *event.MouseEvent) {
					setShowSub(!getShowSub())
				},
				Style: toggleBtnStyle,
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
		kitex.If(getShowSub(), SubscriptionComponent),
	)
})

func main() {
	f, _ := os.Create("kitex_effects_demo.log")
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
