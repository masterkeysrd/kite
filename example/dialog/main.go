package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"runtime"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

func main() {
	f, _ := os.Create("kite.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "unexpected error: %v\n", r)
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			fmt.Fprintf(os.Stderr, "stack trace:\n%s\n", string(buf[:n]))
			os.Exit(1)
		}
	}()

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{Logger: logger})

	root := element.Box(
		element.Box("Dialog Example").Style(style.Style{
			TextAlign: style.Some(style.TextAlignCenter),
			Padding:   style.Some(style.Edges(1)),
		}),
		"Press 'd' to open, 'Enter' or 'Esc' to close.",
	).Style(style.Style{
		Width:      style.Some(style.Percent(100)),
		Height:     style.Some(style.Percent(100)),
		Background: style.Some[color.Color](color.RGBA{R: 20, G: 20, B: 20, A: 255}),
	})

	eng.Mount(root)

	var activeDialog *element.DialogElement

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		slog.Info("Key pressed", "match_enter", ke.MatchString("enter"), "match_esc", ke.MatchString("escape"), "code", ke.Code)

		if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
			return
		}

		if activeDialog != nil {
			if ke.MatchString("enter") || ke.MatchString("escape") {
				slog.Info("Closing dialog")
				eng.Document().RemoveChild(activeDialog)
				activeDialog = nil
				e.StopPropagation()
				return
			}
		}

		if ke.MatchString("d") && activeDialog == nil {
			slog.Info("Opening dialog")
			content := element.Box("Hello! I am a Dialog.").Style(style.Style{
				Width:      style.Some(style.Cells(30)),
				Height:     style.Some(style.Cells(5)),
				Background: style.Some[color.Color](color.RGBA{R: 60, G: 60, B: 100, A: 255}),
				Border:     style.SingleBorder().Some(),
				Padding:    style.Some(style.Edges(1)),
			})
			activeDialog = element.Dialog(content, 100)
			eng.Document().AppendChild(activeDialog)
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
