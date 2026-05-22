package devtools

import (
	"github.com/masterkeysrd/kite/devtools/inspector"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
)

// Options configures the developer tools.
type Options struct {
	// XRayHotkey is the key combination used to toggle X-Ray mode.
	// Defaults to "ctrl+d" if empty.
	XRayHotkey string

	// InspectorAddr is the address to listen on for the web inspector (e.g. "127.0.0.1:8080").
	// If empty, the inspector is not started.
	InspectorAddr string

	// InspectorOptions allows configuring the underlying inspector.
	InspectorOptions inspector.Options
}

// Install registers standard developer tools on the given engine.
func Install(eng *engine.Engine, opts Options) error {
	// 1. Setup X-Ray Toggle
	hotkey := opts.XRayHotkey
	if hotkey == "" {
		hotkey = "ctrl+d"
	}

	var xrayEnabled bool
	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			if ke.MatchString(hotkey) {
				xrayEnabled = !xrayEnabled
				eng.SetDebugXRay(xrayEnabled)
				e.StopPropagation()
			}
		}
	})

	// 2. Setup Inspector if address is provided
	if opts.InspectorAddr != "" {
		if err := inspector.Attach(eng, opts.InspectorAddr, opts.InspectorOptions); err != nil {
			return err
		}
	}

	return nil
}
