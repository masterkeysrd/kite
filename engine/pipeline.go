package engine

import (
	internaldom "github.com/masterkeysrd/kite/internal/dom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/paint"
	"github.com/masterkeysrd/kite/internal/render"
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
	dn := internaldom.AsDirty(e.document)
	if dn.NeedsSync() || dn.ChildNeedsSync() {
		e.syncRenderTree(e.document, e.renderView)
		e.syncOverlays(e.document)
	}

	// Clean up disconnected focus target.
	if focused := e.focusManager.Current(); focused != nil && !focused.IsConnected() {
		e.focusManager.Blur()
	}

	e.focusManager.SetInitialFocus()
}

func (p *StandardPipeline) Tasks(e *Engine) {
	e.scheduler.drainMacrotasks(e.macroTaskBudget)
	e.scheduler.drainMicrotasks()
}

func propagateStyleDirty(ro render.Object) {
	if ro == nil {
		return
	}
	n := ro.LogicalNode()
	if n != nil {
		if de := internaldom.AsDirtyElement(n); de != nil {
			if de.IsDirtyStyle() {
				ro.MarkDirty(render.DirtyStyle)
				de.ClearDirtyStyle()
			}
			de.ClearStyleFlags()
		} else if dn := internaldom.AsDirty(n); dn != nil {
			dn.ClearStyleFlags()
		}
	}
	for child := range ro.Children() {
		propagateStyleDirty(child)
	}
}

func (p *StandardPipeline) Style(e *Engine) {
	propagateStyleDirty(e.renderView)
	for _, overlay := range e.renderView.Overlays() {
		propagateStyleDirty(overlay)
	}

	e.resolver.ResolveTree(e.renderView, nil, false)

	for _, overlay := range e.renderView.Overlays() {
		e.resolver.ResolveTree(overlay, nil, false)
	}

	// Clean logical node style flags after resolution.
	// Since ResolveTree clears DirtyStyle on render objects, we still need to clear
	// the logical flags that triggered it.
	// TODO: Maybe ResolveTree should also clear logical flags?
	// For now, let's keep it consistent with how it was.
	dn := internaldom.AsDirty(e.document)
	dn.ClearStyleFlags()
	for overlayEl := range e.document.Overlays() {
		if de := internaldom.AsDirtyElement(overlayEl); de != nil {
			de.ClearDirtyStyle()
			de.ClearStyleFlags()
		}
	}
}

func (p *StandardPipeline) Layout(e *Engine) bool {
	root := e.renderView
	overlays := root.Overlays()
	rootFlags := root.Flags()
	layoutRan := false

	anyOverlayDirty := false
	for _, o := range overlays {
		if o.Flags()&(render.DirtyLayout|render.ChildNeedsLayout) != 0 {
			anyOverlayDirty = true
			break
		}
	}

	if anyOverlayDirty || rootFlags&(render.DirtyLayout|render.ChildNeedsLayout) != 0 {
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
		e.frameBuffer.Reset()
		e.frameBuffer.BumpVersion()

		ctx := &paint.Context{
			Tracer:    e.Tracer(),
			Selection: selection,
		}
		e.paintEngine.PaintFragment(ctx, root.Fragment(), root.Offset(), e.frameBuffer)
		for _, overlay := range overlays {
			e.paintEngine.PaintFragment(ctx, overlay.Fragment(), overlay.Offset(), e.frameBuffer)
		}
		e.paintEngine.ResolveBorders(ctx, e.frameBuffer)
		e.paintEngine.ApplySelection(e.frameBuffer, ctx.Selection)

		// Bridge to agnostic backend.
		surface := e.backend.BeginFrame()
		bounds := e.frameBuffer.Bounds()
		for y := 0; y < bounds.Size.Height; y++ {
			for x := 0; x < bounds.Size.Width; x++ {
				cell := e.frameBuffer.CellAt(bounds.Origin.X+x, bounds.Origin.Y+y)
				surface.Set(x, y, cell.Cell)
			}
		}

		if err := e.backend.EndFrame(); err != nil {
			e.logger.Warn("failed to end frame", "error", err)
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
