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
	podCardStyle      = style.S().Padding(1, 2).Background(color.RGBA{R: 25, G: 35, B: 55, A: 255}).Border(style.SingleBorder()).Width(style.Percent(90)).Margin(1, 0)
	keyInfoStyle      = style.S().Margin(0, 0, 1, 0)
	statusRowStyle    = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).Margin(0, 0, 1, 0)
	bgFetchTextStyle  = style.S().Foreground(color.RGBA{R: 150, G: 150, B: 150, A: 255})
	buttonsRowStyle   = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).Gap(2)
	restartBtnStyle   = style.S().Background(color.RGBA{R: 200, G: 60, B: 60, A: 255}).Foreground(color.White)
	refetchBtnStyle   = style.S().Background(color.RGBA{R: 50, G: 120, B: 220, A: 255}).Foreground(color.White)
	appContainerStyle = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).AlignItems(style.AlignCenter).Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 18, G: 18, B: 24, A: 255}).Padding(1, 2)
	appHeaderStyle    = style.S().Bold(true).Foreground(color.RGBA{R: 240, G: 190, B: 90, A: 255}).Margin(0, 0, 1, 0).TextAlign(style.TextAlignCenter)
	instructionsStyle = style.S().Foreground(color.RGBA{R: 150, G: 150, B: 150, A: 255}).Margin(0, 0, 1, 0).TextAlign(style.TextAlignCenter)
	rootStyle         = style.S().Width(style.Percent(100)).Height(style.Percent(100))
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

	// Use wind query hook
	query := wind.Use(key, func(ctx context.Context, k PodKey) *promise.Promise[string] {
		return promise.New(func(ctx context.Context) (string, error) {
			return fetchPodStatus(ctx, k)
		})
	})

	// Use wind mutation hook
	mutation := wind.UseMutation(restartPod, wind.MutationOptions[PodKey, string]{
		OnSuccess: func(res string, vars PodKey, ctx wind.MutationContext) {
			// Invalidate the cache for this pod key to trigger background refetch
			ctx.Client.InvalidateQueries(vars)
		},
	})

	statusText := "Unknown"
	textColor := color.RGBA{R: 150, G: 150, B: 150, A: 255}

	if query.IsLoading {
		statusText = "Loading initial status..."
		textColor = color.RGBA{R: 240, G: 190, B: 90, A: 255}
	} else if query.IsError {
		statusText = fmt.Sprintf("Error: %v", query.Error)
		textColor = color.RGBA{R: 220, G: 60, B: 60, A: 255}
	} else if query.Data != "" {
		statusText = query.Data
		textColor = color.RGBA{R: 120, G: 220, B: 140, A: 255}
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
		// Key info
		kitex.Box(kitex.BoxProps{
			Style: keyInfoStyle,
		}, kitex.Text(fmt.Sprintf("🔑 Query Key: PodKey{Namespace: %q, ID: %q}", key.Namespace, key.ID))),

		// Status info
		kitex.Box(kitex.BoxProps{
			Style: statusRowStyle,
		},
			kitex.Text("Current Status: "),
			kitex.Box(kitex.BoxProps{
				Style: style.S().Foreground(textColor).Bold(true),
			}, kitex.Text(statusText)),
			kitex.Box(kitex.BoxProps{
				Style: bgFetchTextStyle,
			}, kitex.Text(bgStateText)),
		),

		// Action buttons
		kitex.Box(kitex.BoxProps{
			Style: buttonsRowStyle,
		},
			kitex.Button(kitex.ButtonProps{
				OnClick: func(e event.Event) {
					if !mutation.IsPending {
						mutation.Mutate(key)
					}
				},
				Style: restartBtnStyle,
			}, kitex.Text(fmt.Sprintf(" 🔄 %s ", mutationText))),

			kitex.Button(kitex.ButtonProps{
				OnClick: func(e event.Event) {
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
		// Header
		kitex.Box(kitex.BoxProps{
			Style: appHeaderStyle,
		}, kitex.Text("💨 Kite Async Data Fetching Demo (extras/wind)")),

		// Help
		kitex.Box(kitex.BoxProps{
			Style: instructionsStyle,
		}, kitex.Text("Use Tab to move focus. Press 'q' or 'ctrl+c' to quit.")),

		// Status View
		PodStatusView(),
	)
})

func main() {
	f, _ := os.Create("wind_demo.log")
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

	client := wind.NewClient()
	rootNode := wind.Provider(wind.ProviderProps{Client: client}, App())

	kitex.Render(rootNode, container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
