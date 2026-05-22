package devtools

import (
	"context"
	"log/slog"
	"time"

	"github.com/masterkeysrd/kite/devtools/inspector"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
)

// Options configures the developer tools.
type Options struct {
	// XRayHotkey is the key combination used to toggle X-Ray mode.
	// Defaults to "ctrl+d" if empty.
	XRayHotkey string

	// InspectorHotkey is the key combination used to open the web inspector.
	// Defaults to "ctrl+i" if empty.
	InspectorHotkey string

	// InspectorAddr is the address to listen on for the web inspector (e.g. "127.0.0.1:8080").
	// If empty, the inspector is not started.
	InspectorAddr string

	// InspectorOptions allows configuring the underlying inspector.
	InspectorOptions inspector.Options
}

// Install registers standard developer tools on the given engine.
func Install(eng *engine.Engine, opts Options) error {
	// Derive a context that is cancelled when the engine stops so that any
	// launched browser processes are terminated.
	ctx, cancel := context.WithCancel(context.Background())
	eng.OnStop(func() {
		slog.Info("devtools: engine stopping, cancelling context")
		cancel()
		// Give the OS a moment to deliver signals to the browser process group.
		time.Sleep(100 * time.Millisecond)
	})
	return InstallWithContext(ctx, eng, opts)
}

// InstallWithContext installs devtools and, if requested, opens a floating
// inspector window that is bound to the provided context. When `ctx` is
// cancelled the inspector process started in app-mode will be terminated.
func InstallWithContext(ctx context.Context, eng *engine.Engine, opts Options) error {
	// 1. Setup X-Ray Toggle
	hotkey := opts.XRayHotkey
	if hotkey == "" {
		hotkey = "ctrl+d"
	}

	var xrayEnabled bool
	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			if ke.MatchString(hotkey) {
				slog.Info("devtools: xray hotkey matched", "hotkey", hotkey)
				xrayEnabled = !xrayEnabled
				eng.SetDebugXRay(xrayEnabled)
				e.StopPropagation()
			}
		}
	}, event.Capture())

	// 2. Setup Inspector if address is provided
	if opts.InspectorAddr != "" {
		inspectorHotkey := opts.InspectorHotkey
		if inspectorHotkey == "" {
			inspectorHotkey = "ctrl+i"
		}

		var actualAddr string
		eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
			ke, ok := e.(*event.KeyEvent)
			if !ok {
				return
			}
			slog.Info("devtools: key event received", "key", ke.Key, "target_hotkey", inspectorHotkey)
			if !ke.MatchString(inspectorHotkey) {
				return
			}
			slog.Info("devtools: inspector hotkey matched", "hotkey", inspectorHotkey)

			if actualAddr == "" {
				slog.Info("devtools: starting inspector server")
				addr, err := inspector.Attach(eng, opts.InspectorAddr, opts.InspectorOptions)
				if err != nil {
					slog.Error("devtools: failed to attach inspector", "err", err)
					return
				}
				actualAddr = addr
				slog.Info("devtools: inspector server started", "addr", actualAddr)
			}

			slog.Info("devtools: opening floating inspector window")
			go func() {
				if err := OpenFloatingInspector(ctx, "http://"+actualAddr); err != nil {
					slog.Warn("devtools: failed to open floating inspector", "err", err)
				}
			}()
			e.StopPropagation()
		}, event.Capture())
	}

	return nil
}
