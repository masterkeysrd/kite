package element

import (
	"image/color"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	internaldom "github.com/masterkeysrd/kite/internal/dom"
	"github.com/masterkeysrd/kite/style"
)

// ButtonElement implements a clickable button component.
type ButtonElement struct {
	elementBase[ButtonElement]
	active   bool
	btnType  string
	disabled bool
}

var _ Element = (*ButtonElement)(nil)

// NewButton creates a new ButtonElement owned by doc.
func NewButton(doc dom.Document, children ...any) *ButtonElement {
	b := &ButtonElement{
		btnType: "submit", // default button type
	}
	// Buttons use Flex by default to center their content.
	b.initBase(doc.CreateElement("button", b), b, defaultButtonStyle)
	processChildren(b, children)
	b.wireEvents()
	return b
}

// Button creates a new ButtonElement with the given children.
func Button(children ...any) *ButtonElement {
	return NewButton(orphanDocument, children...)
}

var defaultButtonStyle = style.Style{
	Display:        style.Some(style.DisplayInlineBlock),
	AlignItems:     style.Some(style.AlignCenter),
	JustifyContent: style.Some(style.JustifyCenter),
	WhiteSpace:     style.Some(style.WhiteSpacePre),
	TextAlign:      style.Some(style.TextAlignCenter),
	Padding:        style.Some(style.Edges(0, 1)),
	Border:         style.SingleBorder().Some(),
	Background:     style.Some(style.TerminalDefault),
	Foreground:     style.Some(style.TerminalDefault),
}

// DefaultStyle returns the element's default style, including dynamic state.
func (b *ButtonElement) DefaultStyle() style.Style {
	s := defaultButtonStyle
	if b.active {
		s.Reverse = style.Some(true)
	}
	if b.disabled {
		s.Foreground = style.Some[color.Color](color.RGBA{R: 100, G: 100, B: 100, A: 255})
	}
	return s
}

func (b *ButtonElement) wireEvents() {
	b.OnEvent(event.EventMouseDown, b.handleMouseDown)
	b.OnEvent(event.EventMouseUp, b.handleMouseUp)
	b.OnEvent(event.EventKeyDown, b.handleKeyDown)
}

func (b *ButtonElement) IntrinsicStyle() style.Style {
	return style.Style{}
}

// Type sets the button type (e.g. "button", "submit") and returns the ButtonElement.
func (b *ButtonElement) Type(btnType string) *ButtonElement {
	b.btnType = btnType
	return b
}

// ButtonType returns the type of the button.
func (b *ButtonElement) ButtonType() string {
	if b.btnType == "" {
		return "submit"
	}
	return b.btnType
}

func (b *ButtonElement) handleMouseDown(e event.Event) {
	if b.IsDisabled() {
		return
	}
	me, ok := e.(*event.MouseEvent)
	if !ok || me.Button != event.ButtonLeft {
		return
	}

	b.active = true
	if d := internaldom.AsDirty(b); d != nil {
		d.MarkNeedsSync()
	}
	b.Focus()
}

func (b *ButtonElement) handleMouseUp(e event.Event) {
	if b.IsDisabled() || !b.active {
		return
	}
	me, ok := e.(*event.MouseEvent)
	if !ok || me.Button != event.ButtonLeft {
		return
	}

	b.active = false
	if d := internaldom.AsDirty(b); d != nil {
		d.MarkNeedsSync()
	}
}

func (b *ButtonElement) handleKeyDown(e event.Event) {
	if b.IsDisabled() {
		return
	}
	ke, ok := e.(*event.KeyEvent)
	if !ok {
		return
	}

	// Space or Enter activates the button.
	if ke.MatchString(" ") || ke.MatchString("enter") {
		// Fire EventClick.
		click := event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonNone, ke.Mod)
		b.DispatchEvent(click)
		e.StopPropagation()
	}
}

// SetData replaces the button's children with a single text node containing data.
func (b *ButtonElement) SetData(data string) {
	for c := b.FirstChild(); c != nil; c = b.FirstChild() {
		b.RemoveChild(c)
	}
	b.AppendChild(NewText(b.OwnerDocument(), data))
}

// SetActive sets the button's active (pressed) state.
func (b *ButtonElement) SetActive(active bool) {
	if b.active == active {
		return
	}
	b.active = active
	if d := internaldom.AsDirty(b); d != nil {
		d.MarkNeedsSync()
	}
}

func (b *ButtonElement) IsDisabled() bool   { return b.disabled }
func (b *ButtonElement) SetDisabled(v bool) { b.disabled = v }
func (b *ButtonElement) Disabled(v bool) *ButtonElement {
	b.disabled = v
	return b
}
