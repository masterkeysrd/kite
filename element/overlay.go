package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

type OverlayElement struct {
	elementBase[OverlayElement]
	config OverlayConfig
}

type OverlayConfig struct {
	Anchor    dom.Element
	ZIndex    int
	Placement layout.OverlayPlacement
	Flip      bool
}

var _ Element = (*OverlayElement)(nil)
var _ render.CustomObjectProvider = (*OverlayElement)(nil)
var _ layout.OverlayLever = (*OverlayElement)(nil)
var _ dom.Lifecycle = (*OverlayElement)(nil)

func NewOverlay(doc dom.Document, content dom.Node, config OverlayConfig) *OverlayElement {
	o := &OverlayElement{config: config}
	o.initBase(doc.CreateElement("overlay", o), o, style.Style{
		Display: style.Some(style.DisplayInlineBlock),
	})
	if content != nil {
		o.AppendChild(content)
	}
	return o
}

func Overlay(content dom.Node, config OverlayConfig) *OverlayElement {
	o := NewOverlay(orphanDocument, content, config)
	return o
}

// SetConfig updates the overlay element's configuration.
func (o *OverlayElement) SetConfig(config OverlayConfig) *OverlayElement {
	o.config = config
	if ro := o.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyLayout)
	}
	return o
}

// CreateRenderObject implements render.CustomObjectProvider.
func (o *OverlayElement) CreateRenderObject() render.Object {
	return render.NewOverlay(o, o.EventTarget())
}

// Anchor implements layout.OverlayLever.
func (o *OverlayElement) Anchor() any {
	return o.config.Anchor
}

// Placement implements layout.OverlayLever.
func (o *OverlayElement) Placement() layout.OverlayPlacement {
	return o.config.Placement
}

// Flip implements layout.OverlayLever.
func (o *OverlayElement) Flip() bool {
	return o.config.Flip
}

// OnConnected implements dom.Lifecycle.
func (o *OverlayElement) OnConnected() {
	if doc := o.OwnerDocument(); doc != nil {
		doc.ShowOverlay(o, o.config.ZIndex)
	}
}

// OnDisconnected implements dom.Lifecycle.
func (o *OverlayElement) OnDisconnected() {
	if doc := o.OwnerDocument(); doc != nil {
		doc.HideOverlay(o)
	}
}
