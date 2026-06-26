package stage

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/kitex/kitexdt"
	"github.com/masterkeysrd/kite/style"
)

// Scene represents a single rendering scenario of a component.
type Scene struct {
	Name   string
	Render func(c *Context) kitex.Node
}

// Stage manages components and their scenes for isolation testing.
type Stage struct {
	components map[string][]Scene
	decorator  func(*Context, kitex.Node) kitex.Node
	globals    []Control
}

// New creates a new Stage manager.
func New() *Stage {
	return &Stage{
		components: make(map[string][]Scene),
	}
}

// WithDecorator sets a wrapper function that is applied around every rendered
// scene node. Use it to inject global providers — theme contexts, store
// providers, or any kitex.Node wrapper — without modifying individual scenes.
//
// Example:
//
//	stg := stage.New()
//	stg.WithDecorator(func(n kitex.Node) kitex.Node {
//		return ThemeCtx.Provider(myTheme, n)
//	})
func (s *Stage) WithDecorator(fn func(kitex.Node) kitex.Node) *Stage {
	s.decorator = func(c *Context, n kitex.Node) kitex.Node {
		return fn(n)
	}
	return s
}

// WithContextDecorator sets a context-aware wrapper function that is applied
// around every rendered scene node, exposing global controls to the decorator.
func (s *Stage) WithContextDecorator(fn func(*Context, kitex.Node) kitex.Node) *Stage {
	s.decorator = fn
	return s
}

// GlobalText registers a global text control.
func (s *Stage) GlobalText(name string, defaultValue string) *Stage {
	s.globals = append(s.globals, Control{
		Name:    name,
		Type:    ControlTypeText,
		Default: defaultValue,
	})
	return s
}

// GlobalBool registers a global boolean control.
func (s *Stage) GlobalBool(name string, defaultValue bool) *Stage {
	s.globals = append(s.globals, Control{
		Name:    name,
		Type:    ControlTypeBool,
		Default: defaultValue,
	})
	return s
}

// GlobalSelect registers a global dropdown select control.
func (s *Stage) GlobalSelect(name string, options []string, defaultValue string) *Stage {
	s.globals = append(s.globals, Control{
		Name:    name,
		Type:    ControlTypeSelect,
		Default: defaultValue,
		Options: options,
	})
	return s
}

// GlobalInt registers a global integer control.
func (s *Stage) GlobalInt(name string, defaultValue int) *Stage {
	s.globals = append(s.globals, Control{
		Name:    name,
		Type:    ControlTypeInt,
		Default: defaultValue,
	})
	return s
}

// Register registers a list of scenes for a component.
func (s *Stage) Register(component string, scenes []Scene) {
	s.components[component] = append(s.components[component], scenes...)
}

// Run launches the Stage TUI explorer.
func (s *Stage) Run() {
	f, _ := os.Create("stage.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	_ = logger // prevent unused variable error
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{})

	// Create root container element
	root := element.NewBox(eng.Document())
	root.Style(style.S().Width(style.Percent(100)).Height(style.Percent(100)))
	eng.Mount(root)

	kitex.EnableDevMode = true

	// Mount VDOM into host container
	kitex.Render(renderUI(s), root)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Global key handler to quit Stage UI
	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("ctrl+c") {
			cancel()
		}
	})

	insp, _ := devtools.Install(eng, devtools.Options{})
	kitexdt.Register(insp)

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
