package trail

import (
	"context"
	"sync"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
)

type options struct {
	backend    backend.Backend
	engineOpts engine.Options
	ctrlCExit  bool
}

// Option configures a trail Instance.
type Option func(*options)

// WithBackend specifies a custom backend (e.g. mock.New for testing).
func WithBackend(b backend.Backend) Option {
	return func(o *options) {
		o.backend = b
	}
}

// WithEngineOptions supplies custom engine.Options.
func WithEngineOptions(opts engine.Options) Option {
	return func(o *options) {
		o.engineOpts = opts
	}
}

// WithCtrlCExit enables or disables automatic exit when Ctrl+C is pressed.
func WithCtrlCExit(enabled bool) Option {
	return func(o *options) {
		o.ctrlCExit = enabled
	}
}

// Instance manages the lifecycle of an inline trail application.
type Instance struct {
	eng     *engine.Engine
	exitErr error
	exiting bool
	exitMu  sync.Mutex
}

// Compile-time check that Instance implements AppHandle.
var _ AppHandle = (*Instance)(nil)

// Engine returns the underlying Kite Engine.
func (inst *Instance) Engine() *engine.Engine {
	return inst.eng
}

// Exit signals the application to exit with an optional error.
func (inst *Instance) Exit(err error) {
	inst.exitMu.Lock()
	if inst.exitErr == nil && err != nil {
		inst.exitErr = err
	}
	inst.exiting = true
	inst.exitMu.Unlock()
	inst.eng.RequestFrame()
}

// Run executes the application event loop until context cancellation or Exit.
func (inst *Instance) Run(ctx context.Context) error {
	err := inst.eng.Run(ctx)
	inst.exitMu.Lock()
	defer inst.exitMu.Unlock()
	if inst.exitErr != nil {
		return inst.exitErr
	}
	return err
}

// Frame runs a single render pipeline frame (useful in tests).
func (inst *Instance) Frame() {
	inst.eng.Frame()
}

// InsertAbove renders a kitex node into a temporary container and permanently
// flushes it to the terminal scrollback above the active inline frame.
func (inst *Instance) InsertAbove(node kitex.Node) {
	if node == nil {
		return
	}
	doc := dom.NewDocument()
	container := element.NewBox(doc)
	nodes := node.Instantiate(doc)
	for _, n := range nodes {
		if n != nil {
			container.AppendChild(n)
		}
	}
	inst.eng.InsertAbove(container)
}

// Println writes a text line above the inline frame.
func (inst *Instance) Println(text string) {
	inst.InsertAbove(kitex.Box(kitex.BoxProps{}, kitex.Text(text)))
}

// New initializes an inline trail Instance for the given root app component.
func New(app kitex.Node, opts ...Option) (*Instance, error) {
	cfg := options{
		ctrlCExit: true,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	b := cfg.backend
	if b == nil {
		var err error
		b, err = uv.New()
		if err != nil {
			return nil, err
		}
	}

	engOpts := cfg.engineOpts
	engOpts.Inline = true

	eng := engine.New(b, engOpts)
	inst := &Instance{
		eng: eng,
	}

	eng.OnFrameRendered(func() {
		inst.exitMu.Lock()
		exiting := inst.exiting
		inst.exitMu.Unlock()
		if exiting {
			eng.Stop()
		}
	})

	container := element.NewBox(eng.Document())
	eng.Mount(container)

	wrappedApp := appContext.Provider(inst, app)
	kitex.Render(wrappedApp, container)

	if cfg.ctrlCExit {
		eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
			if ke, ok := e.(*event.KeyEvent); ok {
				if ke.MatchString("ctrl+c") {
					inst.Exit(nil)
				}
			}
		})
	}

	return inst, nil
}

// Run creates a new Instance and executes its event loop.
func Run(ctx context.Context, app kitex.Node, opts ...Option) error {
	inst, err := New(app, opts...)
	if err != nil {
		return err
	}
	return inst.Run(ctx)
}
