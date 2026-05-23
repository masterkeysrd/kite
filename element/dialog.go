package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/style"
)

type DialogElement struct {
	elementBase[DialogElement]
	zIndex int
	scope  *focus.Scope
}

var _ Element = (*DialogElement)(nil)
var _ dom.Lifecycle = (*DialogElement)(nil)

func NewDialog(doc dom.Document, content dom.Node, zIndex int) *DialogElement {
	d := &DialogElement{zIndex: zIndex}

	// Requirement: Applies styles: Width: 100%, Height: 100%, Display: Flex,
	// JustifyContent: Center, AlignItems: Center.
	d.initBase(doc.CreateElement("dialog", d), d, style.Style{
		Width:          style.Some(style.Percent(100)),
		Height:         style.Some(style.Percent(100)),
		Display:        style.Some(style.DisplayFlex),
		JustifyContent: style.Some(style.JustifyCenter),
		AlignItems:     style.Some(style.AlignCenter),
	})

	if content != nil {
		d.AppendChild(content)
	}

	return d
}

func Dialog(content dom.Node, zIndex int) *DialogElement {
	return NewDialog(orphanDocument, content, zIndex)
}

func (d *DialogElement) OnConnected() {
	if doc := d.OwnerDocument(); doc != nil {
		doc.ShowOverlay(d, d.zIndex)

		if fm, ok := doc.FocusManager().(*focus.Manager); ok {
			d.scope = &focus.Scope{Root: d.self, Autofocus: d.self}
			fm.PushScope(d.scope)
		}
	}
}

func (d *DialogElement) OnDisconnected() {
	if doc := d.OwnerDocument(); doc != nil {
		doc.HideOverlay(d)
		if fm, ok := doc.FocusManager().(*focus.Manager); ok {
			fm.PopScope()
		}
	}
}

func (d *DialogElement) IsFocusable() bool {
	return true
}
