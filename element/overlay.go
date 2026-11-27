package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/geom"
	internaldom "github.com/masterkeysrd/kite/internal/dom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
)

type OverlayElement struct {
	elementBase[OverlayElement]
	config OverlayConfig
}

type OverlayConfig struct {
	Anchor    dom.Element
	ZIndex    int
	Placement geom.Placement
	Flip      bool
}

var _ Element = (*OverlayElement)(nil)
var _ render.CustomObjectProvider = (*OverlayElement)(nil)
var _ layout.OverlayLever = (*OverlayElement)(nil)
var _ dom.Lifecycle = (*OverlayElement)(nil)

func NewOverlay(doc dom.Document, anchor dom.Element, children ...any) *OverlayElement {
	o := &OverlayElement{config: OverlayConfig{Anchor: anchor}}
	o.initBase(doc.CreateElement("overlay", o), o, style.S().Display(style.DisplayInlineBlock))
	processChildren(o, children)
	return o
}

func Overlay(anchor dom.Element, children ...any) *OverlayElement {
	return NewOverlay(orphanDocument, anchor, children...)
}

func (o *OverlayElement) syncConfig() {
	if doc := o.OwnerDocument(); doc != nil && o.IsConnected() {
		doc.ShowOverlay(o, o.config.ZIndex)
	}
	if d := internaldom.AsDirty(o); d != nil {
		d.MarkNeedsSync()
	}
}

// SetConfig updates the overlay element's configuration.
func (o *OverlayElement) SetConfig(config OverlayConfig) *OverlayElement {
	o.config = config
	o.syncConfig()
	return o
}

// WithAnchor sets the anchor element for positioning the overlay.
func (o *OverlayElement) WithAnchor(a dom.Element) *OverlayElement {
	o.config.Anchor = a
	o.syncConfig()
	return o
}

// WithPlacement sets the preferred placement relative to the anchor.
func (o *OverlayElement) WithPlacement(p geom.Placement) *OverlayElement {
	o.config.Placement = p
	o.syncConfig()
	return o
}

// WithFlip configures whether the overlay flips when overflowing the viewport.
func (o *OverlayElement) WithFlip(flip bool) *OverlayElement {
	o.config.Flip = flip
	o.syncConfig()
	return o
}

// WithZIndex sets the overlay's top-layer z-index.
func (o *OverlayElement) WithZIndex(zIndex int) *OverlayElement {
	o.config.ZIndex = zIndex
	o.syncConfig()
	return o
}

// CreateRenderObject implements render.CustomObjectProvider.
func (o *OverlayElement) CreateRenderObject() render.Object {
	return render.NewOverlay(o, o.EventTarget())
}

// Anchor implements layout.OverlayLever.
func (o *OverlayElement) Anchor() dom.Node {
	return o.config.Anchor
}

// Placement implements layout.OverlayLever.
func (o *OverlayElement) Placement() geom.Placement {
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
