package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
)

func TestExampleApp_StartAndStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	b := mock.New(80, 24)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runWithBackend(ctx, b, logger, cancel)
	}()

	// Let it run for a moment, then dump the framebuffer
	time.Sleep(500 * time.Millisecond)

	if len(b.Frames) > 0 {
		frame := b.Frames[len(b.Frames)-1]
		surface := frame.Surface
		bounds := surface.Bounds()
		t.Logf("Frame buffer size: %dx%d", bounds.Size.Width, bounds.Size.Height)
		for y := 0; y < bounds.Size.Height; y++ {
			line := ""
			for x := 0; x < bounds.Size.Width; x++ {
				cell := surface.CellAt(x, y)
				if cell.Content == "" {
					line += " "
				} else {
					line += cell.Content
				}
			}
			t.Logf("%02d: %s", y, line)
		}
	} else {
		t.Log("No frames rendered yet!")
	}

	if eng != nil {
		t.Log("DOM and Layout Tree:")
		dumpDOMTree(t, eng.Document(), "")
	}

	logger.Info("TEST: calling cancel()")
	cancel()

	// Wait indefinitely for the error. If it hangs, go test will time out and dump stack trace.
	err := <-errCh
	if err != nil && err != context.Canceled {
		t.Errorf("Example app exited with error: %v", err)
	}
}

func dumpDOMTree(t *testing.T, n dom.Node, indent string) {
	if n == nil {
		return
	}
	tag := "text"
	if el, ok := n.(dom.Element); ok {
		tag = el.TagName()
	}
	fragSize := "nil"
	offset := "nil"
	if ro := n.RenderObject(); ro != nil {
		if f := ro.Fragment(); f != nil {
			fragSize = fmt.Sprintf("%dx%d", f.Size.Width, f.Size.Height)
		}
		p := ro.Offset()
		offset = fmt.Sprintf("%d,%d", p.X, p.Y)
	}
	t.Logf("%s<%s> size=%s offset=%s", indent, tag, fragSize, offset)
	for child := range n.ChildNodes() {
		dumpDOMTree(t, child, indent+"  ")
	}
}
