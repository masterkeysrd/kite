package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"time"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	var b backend.Backend
	if os.Getenv("USE_MOCK_BACKEND") == "1" {
		slog.Info("Using mock backend")
		b = mock.New(80, 24)
	} else {
		f, er := os.Create("kite.log")
		if er != nil {
			fmt.Fprintf(os.Stderr, "failed to create log file: %v\n", er)
			os.Exit(1)
		}
		defer f.Close()

		logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))
		slog.SetDefault(logger)

		slog.Info("Using UV backend")
		var err error
		b, err = uv.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
			os.Exit(1)
		}
	}

	if os.Getenv("AUTO_CLOSE") == "1" {
		go func() {
			slog.Info("Auto-close enabled, exiting in 5 seconds...")
			<-time.After(5 * time.Second)
			os.Exit(0)
		}()
	}

	// Initialize the rendering engine
	opts := engine.Options{
		Logger: slog.Default(),
	}
	eng := engine.New(b, opts)

	// Create the Document and root view container
	doc := eng.Document()

	// Create our app container
	root := element.NewBox(doc).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
		Padding:    style.Some(style.Edges(2, 4)),
		Background: style.Some[color.Color](color.RGBA{R: 0, G: 0, B: 255, A: 255}), // Blue background
	})

	// Create an inner box using LayoutNG border-box semantics
	inner := element.NewBox(doc).Style(style.Style{
		Width:      style.Some(style.Cells(40)),
		Height:     style.Some(style.Cells(10)),
		Margin:     style.Some(style.Edges(1, 2)),
		Background: style.Some[color.Color](color.RGBA{R: 255, G: 0, B: 0, A: 255}), // Red background
		Border: style.Some(style.Border{
			Width: style.Edges(1),
			Style: style.EdgeAll(style.BorderSingle),
			Color: style.EdgeAll[color.Color](color.RGBA{R: 0, G: 255, B: 0, A: 255}), // Green border
		}),
	})

	// Add inner to root
	root.AppendChild(inner)

	// Attach root logical element to the engine
	eng.Mount(root)

	// Add global quit listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doc.AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			if ke.MatchString("ctrl+c") || ke.MatchString("q") {
				cancel()
			}
		}
	})

	// Run the engine
	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited with error: %v\n", err)
		os.Exit(1)
	}
}
