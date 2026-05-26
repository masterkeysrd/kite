package devtools

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/masterkeysrd/kite/devtools/inspector"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
)

const DefautlXRayHotkey = "ctrl+d"
const DefaultToolsHotkey = "f12"
const DefaultServerAddr = "localhost:8080"

// Options configures the developer tools.
type Options struct {
	// XRayHotkey is the key combination used to toggle X-Ray mode.
	// Defaults to "ctrl+d" if empty.
	XRayHotkey string

	// ServerAddr is the address to listen on for the devtools server (e.g. "127.0.0.1:8080").
	// If empty, the devtools server is not started.
	ServerAddr string

	// ToolsHotkey is the key combination used to toggle the tools dashboard.
	// Defaults to "f12" if empty.
	ToolsHotkey string
}

// Install registers standard developer tools on the given engine.
func Install(eng *engine.Engine, opts Options) (*inspector.Inspector, error) {
	ctx, cancel := context.WithCancel(context.Background())
	eng.OnStop(func() {
		slog.Info("devtools: engine stopping, cancelling context")
		cancel()
		time.Sleep(100 * time.Millisecond)
	})

	if opts.XRayHotkey == "" {
		opts.XRayHotkey = DefautlXRayHotkey
	}
	if opts.ServerAddr == "" {
		opts.ServerAddr = DefaultServerAddr
	}

	slog.Info("devtools: installing with options", "xrayHotkey", opts.XRayHotkey, "serverAddr", opts.ServerAddr)
	var insp *inspector.Inspector
	if opts.ServerAddr != "" {
		insp = inspector.New(eng)
		srv := NewDevToolsServer(eng, insp)
		mux := http.NewServeMux()
		srv.SetupRoutes(mux)

		slog.Info("devtools: starting inspector server", "requestedAddr", opts.ServerAddr)
		l, err := net.Listen("tcp", opts.ServerAddr)
		if err != nil {
			slog.Warn("devtools: requested address failed, trying random port", "addr", opts.ServerAddr, "err", err)
			l, err = net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return nil, fmt.Errorf("devtools: failed to find any available port: %w", err)
			}
		}

		go func() {
			slog.Info("devtools: starting HTTP server", "addr", l.Addr().String())
			if err := http.Serve(l, mux); err != nil {
				slog.Error("devtools: server error", "err", err)
			}
		}()
		slog.Info("devtools: inspector server started", "addr", l.Addr().String())
		eng.OnFrameRendered(srv.Broadcast)

		// Hook F12 to open inspector
		inspectorHotkey := "f12"
		slog.Info("devtools: setting up inspector hotkey", "hotkey", inspectorHotkey)
		eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
			ke, ok := e.(*event.KeyEvent)
			if !ok {
				return
			}

			if !ke.MatchString(inspectorHotkey) {
				return
			}
			slog.Info("devtools: inspector hotkey matched", "hotkey", inspectorHotkey)
			slog.Info("devtools: opening floating inspector window")
			go func() {
				if err := OpenFloatingInspector(ctx, "http://"+l.Addr().String()); err != nil {
					slog.Warn("devtools: failed to open floating inspector", "err", err)
				}
			}()
			e.StopPropagation()
		}, event.Capture())
	}

	return insp, nil
}
