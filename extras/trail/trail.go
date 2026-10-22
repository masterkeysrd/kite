package trail

import (
	"bytes"
	"context"
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
)

type options struct {
	backend      backend.Backend
	engineOpts   engine.Options
	ctrlCExit    bool
	interceptLog bool
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

// WithInterceptLog redirects standard log output (log.Default()) above the inline frame during Run.
func WithInterceptLog(enabled bool) Option {
	return func(o *options) {
		o.interceptLog = enabled
	}
}

// Instance manages the lifecycle of an inline trail application.
type Instance struct {
	eng          *engine.Engine
	container    dom.Element
	doneCh       chan struct{}
	runOnce      sync.Once
	runErr       error
	running      atomic.Bool
	interceptLog bool

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
	if !inst.running.Load() {
		select {
		case <-inst.doneCh:
		default:
			close(inst.doneCh)
		}
	}
	inst.exitMu.Unlock()
	inst.eng.RequestFrame()
}

type logWriter struct {
	inst *Instance
	mu   sync.Mutex
	buf  []byte
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := string(bytes.TrimRight(w.buf[:idx], "\r"))
		w.buf = w.buf[idx+1:]
		if line != "" {
			w.inst.Println(line)
		}
	}
	return len(p), nil
}

// LogWriter returns an io.Writer adapter that splits written bytes into lines
// and writes them permanently into the terminal scrollback above the active frame.
func (inst *Instance) LogWriter() io.Writer {
	return &logWriter{inst: inst}
}

// Run executes the application event loop until context cancellation or Exit.
func (inst *Instance) Run(ctx context.Context) error {
	inst.runOnce.Do(func() {
		inst.running.Store(true)
		if inst.interceptLog {
			origWriter := log.Default().Writer()
			lw := inst.LogWriter()
			log.Default().SetOutput(lw)
			defer log.Default().SetOutput(origWriter)
		}
		defer func() {
			inst.running.Store(false)
			select {
			case <-inst.doneCh:
			default:
				close(inst.doneCh)
			}
		}()

		err := inst.eng.Run(ctx)
		inst.exitMu.Lock()
		if inst.exitErr != nil {
			inst.runErr = inst.exitErr
		} else {
			inst.runErr = err
		}
		inst.exitMu.Unlock()
	})

	<-inst.doneCh
	return inst.runErr
}

// WaitUntilExit blocks until the trail application completes and returns its exit error.
func (inst *Instance) WaitUntilExit() error {
	<-inst.doneCh
	inst.exitMu.Lock()
	defer inst.exitMu.Unlock()
	if inst.runErr != nil {
		return inst.runErr
	}
	return inst.exitErr
}

// Rerender updates the root component tree with a new kitex.Node.
func (inst *Instance) Rerender(app kitex.Node) {
	wrapped := appContext.Provider(inst, app)
	if inst.running.Load() {
		inst.eng.Post(func() {
			kitex.Render(wrapped, inst.container)
			inst.eng.RequestFrame()
		})
		inst.eng.RequestFrame()
	} else {
		kitex.Render(wrapped, inst.container)
		inst.eng.RequestFrame()
	}
}

// Clear clears the current dynamic inline frame output.
func (inst *Instance) Clear() {
	empty := Fragment()
	wrapped := appContext.Provider(inst, empty)
	if inst.running.Load() {
		inst.eng.Post(func() {
			kitex.Render(wrapped, inst.container)
			inst.eng.RequestFrame()
		})
		inst.eng.RequestFrame()
	} else {
		kitex.Render(wrapped, inst.container)
		inst.eng.RequestFrame()
	}
}

// Unmount unmounts the root VDOM tree (running effect cleanups) and exits the application.
func (inst *Instance) Unmount() {
	if inst.running.Load() {
		inst.eng.Post(func() {
			kitex.Render(nil, inst.container)
			inst.Exit(nil)
		})
		inst.eng.RequestFrame()
	} else {
		kitex.Render(nil, inst.container)
		inst.Exit(nil)
	}
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
	renderNode := func() {
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

	if inst.running.Load() {
		inst.eng.Post(func() {
			renderNode()
		})
		inst.eng.RequestFrame()
	} else {
		renderNode()
	}
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
	container := element.NewBox(eng.Document())
	eng.Mount(container)

	inst := &Instance{
		eng:          eng,
		container:    container,
		doneCh:       make(chan struct{}),
		interceptLog: cfg.interceptLog,
	}

	eng.OnStop(func() {
		kitex.Render(nil, inst.container)
	})

	eng.OnFrameRendered(func() {
		inst.exitMu.Lock()
		exiting := inst.exiting
		inst.exitMu.Unlock()
		if exiting {
			eng.Stop()
		}
	})

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

// Render initializes an inline trail application, starts its event loop in a background
// goroutine, and returns the running Instance.
func Render(ctx context.Context, app kitex.Node, opts ...Option) (*Instance, error) {
	inst, err := New(app, opts...)
	if err != nil {
		return nil, err
	}
	go func() {
		_ = inst.Run(ctx)
	}()
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
