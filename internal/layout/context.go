package layout

import "github.com/masterkeysrd/kite/trace"

// Context carries state and utilities through the layout tree.
type Context struct {
	Tracer *trace.Tracer
}

// Begin is a helper to start a trace span.
func (c *Context) Begin(name string) func() {
	if c == nil {
		return trace.Noop()
	}
	return c.Tracer.Begin(name)
}
