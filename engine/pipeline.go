package engine

import (
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/paint"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// Pipeline defines the core phases of the Kite rendering engine.
// Abstracting these into an interface allows for decorators like a Profiler
// to wrap the standard execution loop without polluting the core logic.
type Pipeline interface {
	Sync(e *Engine)
	Tasks(e *Engine)
	Style(e *Engine)
	Layout(e *Engine) bool // returns layoutRan
	Paint(e *Engine, layoutRan bool)
}

// StandardPipeline is the default implementation of the Kite rendering pipeline.
type StandardPipeline struct{}

func (p *StandardPipeline) Sync(e *Engine) {
	if e.document.NeedsSync() || e.document.ChildNeedsSync() {
		e.syncRenderTree(e.document, e.renderView)
		e.syncOverlays(e.document)
	}

	if e.focusManager.Current() == nil {
		e.focusManager.Next()
	}
}

func (p *StandardPipeline) Tasks(e *Engine) {
	e.drainMacroTasks()
	e.drainMicroTasks()
}

func (p *StandardPipeline) Style(e *Engine) {
	root := e.renderView
	overlays := root.Overlays()
	rootFlags := root.Flags()

	if rootFlags&(render.DirtyStyle|render.ChildNeedsStyle) != 0 {
		style.ResolveTree(e.resolver, root)
	}
	for _, overlay := range overlays {
		if overlay.Flags()&(render.DirtyStyle|render.ChildNeedsStyle) != 0 {
			style.ResolveTree(e.resolver, overlay)
		}
	}
}

func (p *StandardPipeline) Layout(e *Engine) bool {
	root := e.renderView
	overlays := root.Overlays()
	rootFlags := root.Flags()
	layoutRan := false

	if rootFlags&(render.DirtyLayout|render.ChildNeedsLayout) != 0 {
		layoutRan = true
		viewport := root.ViewportSize()
		ctx := &layout.Context{Tracer: e.Tracer()}
		render.LayoutPhase(ctx, root, viewport)

		root.ClearDirtyRecursive(render.DirtyLayout | render.ChildNeedsLayout)
		for _, overlay := range overlays {
			overlay.ClearDirtyRecursive(render.DirtyLayout | render.ChildNeedsLayout)
		}
	}

	focused := e.focusManager.Current()
	if focused != nil {
		if el, ok := focused.(interface{ ScrollCursorIntoView() }); ok {
			el.ScrollCursorIntoView()
		}
	}

	if len(e.afterLayoutHooks) > 0 {
		hooks := e.afterLayoutHooks
		e.afterLayoutHooks = nil
		for _, fn := range hooks {
			fn()
		}
	}

	return layoutRan
}

func (p *StandardPipeline) Paint(e *Engine, layoutRan bool) {
	root := e.renderView
	overlays := root.Overlays()
	cursorChanged := e.updateHardwareCursor(layoutRan)

	anyOverlayDirty := false
	for _, o := range overlays {
		if o.Flags()&(render.DirtyPaint|render.DirtyScroll|render.ChildNeedsPaint) != 0 {
			anyOverlayDirty = true
			break
		}
	}

	// Always update selection if it might have changed.
	// For now, we update it if any paint is needed.
	// Selection change itself doesn't (yet) mark paint dirty automatically.
	// We might need to add that to selectionImpl.changed().
	selection := e.resolveSelection()

	if cursorChanged || anyOverlayDirty || root.Flags()&(render.DirtyPaint|render.DirtyScroll|render.ChildNeedsPaint) != 0 || len(selection) > 0 {
		surface := e.backend.BeginFrame()
		ctx := &paint.Context{
			Tracer:    e.Tracer(),
			Selection: selection,
		}
		e.paintEngine.PaintFragment(ctx, root.Fragment(), root.Offset(), surface)
		for _, overlay := range overlays {
			e.paintEngine.PaintFragment(ctx, overlay.Fragment(), overlay.Offset(), surface)
		}
		e.paintEngine.ResolveBorders(ctx, surface)
		e.paintEngine.ApplySelection(surface, ctx.Selection)

		if err := e.backend.EndFrame(); err != nil {
			// error logging is handled in Engine.Frame if we wanted,
			// but for now we'll keep it here or pass it back.
		}
		root.ClearDirtyRecursive(render.DirtyPaint | render.DirtyScroll | render.ChildNeedsPaint)
		for _, overlay := range overlays {
			overlay.ClearDirtyRecursive(render.DirtyPaint | render.DirtyScroll | render.ChildNeedsPaint)
		}
		e.frameVersion++
	}
}

// ProfilingPipeline wraps a Pipeline and records phase durations using a trace.Tracer.
type ProfilingPipeline struct {
	wrapped Pipeline
}

func (p *ProfilingPipeline) Sync(e *Engine) {
	defer e.Tracer().Begin("Phase:Sync")()
	p.wrapped.Sync(e)
}

func (p *ProfilingPipeline) Tasks(e *Engine) {
	defer e.Tracer().Begin("Phase:Tasks")()
	p.wrapped.Tasks(e)
}

func (p *ProfilingPipeline) Style(e *Engine) {
	defer e.Tracer().Begin("Phase:Style")()
	p.wrapped.Style(e)
}

func (p *ProfilingPipeline) Layout(e *Engine) bool {
	defer e.Tracer().Begin("Phase:Layout")()
	return p.wrapped.Layout(e)
}

func (p *ProfilingPipeline) Paint(e *Engine, layoutRan bool) {
	defer e.Tracer().Begin("Phase:Paint")()
	p.wrapped.Paint(e, layoutRan)
}
