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
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/wind"
	"github.com/masterkeysrd/kite/promise"
	"github.com/masterkeysrd/kite/style"
)

var (
	podCardStyle = style.S().
			Width(style.Percent(90)).
			Margin(1, 0).
			Padding(1, 2).
			Border(style.SingleBorder()).
			Background(style.RGB(25, 35, 55))

	keyInfoStyle = style.S().Margin(0, 0, 1, 0)

	statusRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Margin(0, 0, 1, 0)

	bgFetchTextStyle = style.S().Foreground(style.RGB(150, 150, 150))

	buttonsRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Gap(2)

	restartBtnStyle = style.S().
			Background(style.RGB(200, 60, 60)).
			Foreground(color.White)

	refetchBtnStyle = style.S().
			Background(style.RGB(50, 120, 220)).
			Foreground(color.White)

	appContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				AlignItems(style.AlignCenter).
				Width(style.Percent(100)).
				Height(style.Percent(100)).
				Padding(1, 2).
				Background(style.RGB(18, 18, 24))

	appHeaderStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.RGB(240, 190, 90)).
			Bold(true).
			TextAlign(style.TextAlignCenter)

	instructionsStyle = style.S().
				Margin(0, 0, 1, 0).
				Foreground(style.RGB(150, 150, 150)).
				TextAlign(style.TextAlignCenter)
)

type PodKey struct {
	Namespace string
	ID        string
}

func fetchPodStatus(ctx context.Context, key PodKey) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(1 * time.Second):
		return fmt.Sprintf("Running (updated at %s)", time.Now().Format("15:04:05")), nil
	}
}

func restartPod(ctx context.Context, key PodKey) *promise.Promise[string] {
	return promise.New(func(ctx context.Context) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(800 * time.Millisecond):
			return "Restarted", nil
		}
	})
}

var PodStatusView = kitex.SimpleFC("PodStatusView", func() kitex.Node {
	key := PodKey{Namespace: "prod", ID: "nginx-web-app"}

	query := wind.Use(key, func(ctx context.Context, k PodKey) *promise.Promise[string] {
		return promise.New(func(ctx context.Context) (string, error) {
			return fetchPodStatus(ctx, k)
		})
	})

	mutation := wind.UseMutation(restartPod, wind.MutationOptions[PodKey, string]{
		OnSuccess: func(res string, vars PodKey, ctx wind.MutationContext) {
			ctx.Client.InvalidateQueries(vars)
		},
	})

	statusText := "Unknown"
	textColor := style.RGB(150, 150, 150)

	if query.IsLoading {
		statusText = "Loading initial status..."
		textColor = style.RGB(240, 190, 90)
	} else if query.IsError {
		statusText = fmt.Sprintf("Error: %v", query.Error)
		textColor = style.RGB(220, 60, 60)
	} else if query.Data != "" {
		statusText = query.Data
		textColor = style.RGB(120, 220, 140)
	}

	bgStateText := ""
	if query.IsFetching && !query.IsLoading {
		bgStateText = " (Refetching in background...)"
	}

	mutationText := "Restart Pod"
	if mutation.IsPending {
		mutationText = "Restarting Pod..."
	}

	return kitex.Box(kitex.BoxProps{
		Style: podCardStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: keyInfoStyle,
		}, kitex.Textf("🔑 Query Key: PodKey{Namespace: %q, ID: %q}", key.Namespace, key.ID)),

		kitex.Box(kitex.BoxProps{
			Style: statusRowStyle,
		},
			kitex.Text("Current Status: "),
			kitex.Box(kitex.BoxProps{
				Style: style.S().
					Foreground(textColor).
					Bold(true),
			}, kitex.Text(statusText)),
			kitex.Box(kitex.BoxProps{
				Style: bgFetchTextStyle,
			}, kitex.Text(bgStateText)),
		),

		kitex.Box(kitex.BoxProps{
			Style: buttonsRowStyle,
		},
			kitex.Button(kitex.ButtonProps{
				OnPress: func() {
					if !mutation.IsPending {
						mutation.Mutate(key)
					}
				},
				Style: restartBtnStyle,
			}, kitex.Textf(" 🔄 %s ", mutationText)),

			kitex.Button(kitex.ButtonProps{
				OnPress: func() {
					query.Refetch()
				},
				Style: refetchBtnStyle,
			}, kitex.Text(" 🔍 Manual Refetch ")),
		),
	)
})

var App = kitex.SimpleFC("App", func() kitex.Node {
	return kitex.Box(kitex.BoxProps{
		Style: appContainerStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: appHeaderStyle,
		}, kitex.Text("💨 Kite Async Data Fetching Demo (extras/wind)")),

		kitex.Box(kitex.BoxProps{
			Style: instructionsStyle,
		}, kitex.Text("Use Tab to move focus. Press 'q' or 'ctrl+c' to quit.")),

		PodStatusView(),
	)
})

func main() {
	f, _ := os.Create("wind_demo.log")
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

	client := wind.NewClient()
	rootNode := wind.Provider(wind.ProviderProps{Client: client}, App())
	kitex.Render(rootNode, container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok && (ke.MatchString("q") || ke.MatchString("ctrl+c")) {
			cancel()
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
