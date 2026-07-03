package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

type DialogElement struct {
	elementBase[DialogElement]
	zIndex int
	scope  *dom.FocusScope
}

var _ Element = (*DialogElement)(nil)
var _ dom.Lifecycle = (*DialogElement)(nil)
var _ event.Scrollable = (*DialogElement)(nil)

var defaultDialogStyle = style.S().
	Width(style.Percent(100)).
	Height(style.Percent(100)).
	Display(style.DisplayFlex).
	JustifyContent(style.JustifyCenter).
	AlignItems(style.AlignCenter).
	AlignContent(style.AlignCenter)

func NewDialog(doc dom.Document, content dom.Node, zIndex int) *DialogElement {
	d := &DialogElement{zIndex: zIndex}

	// Requirement: Applies styles: Width: 100%, Height: 100%, Display: Flex,
	// JustifyContent: Center, AlignItems: Center.
	d.initBase(doc.CreateElement("dialog", d), d, defaultDialogStyle)

	if content != nil {
		d.AppendChild(content)
	}

	return d
}

func Dialog(content dom.Node, zIndex int) *DialogElement {
	return NewDialog(orphanDocument, content, zIndex)
}

// SetZIndex updates the dialog's overlay z-index.
func (d *DialogElement) SetZIndex(zIndex int) *DialogElement {
	d.zIndex = zIndex
	if doc := d.OwnerDocument(); doc != nil && d.IsConnected() {
		doc.ShowOverlay(d, zIndex)
	}
	return d
}

func (d *DialogElement) OnConnected() {
	if doc := d.OwnerDocument(); doc != nil {
		doc.ShowOverlay(d, d.zIndex)
		doc.PushScope(&dom.FocusScope{Root: d.self, Autofocus: d.self})
	}
}

func (d *DialogElement) OnDisconnected() {
	if doc := d.OwnerDocument(); doc != nil {
		doc.HideOverlay(d)
		doc.PopScope()
	}
}

func (d *DialogElement) IsFocusable() bool {
	return true
}

// OnWheel implements event.Scrollable to trap scroll/wheel events inside the dialog.
func (d *DialogElement) OnWheel(e *event.WheelEvent) {
	e.StopPropagation()
}
