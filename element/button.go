package element

import (
	"image/color"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// ButtonElement implements a clickable button component.
type ButtonElement struct {
	elementBase[ButtonElement]
	active bool
}

var _ Element = (*ButtonElement)(nil)

// NewButton creates a new ButtonElement owned by doc.
func NewButton(doc dom.Document, children ...any) *ButtonElement {
	b := &ButtonElement{}
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
	Padding:    style.Some(style.Edges(0, 1)),
	Border:     style.SingleBorder().Some(),
	Background: style.Some[color.Color](style.TerminalDefault),
	Foreground: style.Some[color.Color](style.TerminalDefault),
}

func (b *ButtonElement) wireEvents() {
	b.OnEvent(event.EventMouseDown, b.handleMouseDown)
	b.OnEvent(event.EventMouseUp, b.handleMouseUp)
	b.OnEvent(event.EventKeyDown, b.handleKeyDown)
}

func (b *ButtonElement) IntrinsicStyle() style.Style {
	s := style.Style{
		Display:        style.Some(style.DisplayInlineFlex),
		AlignItems:     style.Some(style.AlignCenter),
		JustifyContent: style.Some(style.JustifyCenter),
		WhiteSpace:     style.Some(style.WhiteSpacePre),
		TextAlign:      style.Some(style.TextAlignCenter),
	}
	if b.active {
		s.Reverse = style.Some(true)
	}
	return s
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
	if ro := b.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyStyle)
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
	if ro := b.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyStyle)
	}

	// Fire EventClick.
	click := event.NewMouseEvent(event.EventClick, me.Screen, event.ButtonLeft, me.Mods)
	b.DispatchEvent(click)
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
		click := event.NewMouseEvent(event.EventClick, layout.Point{}, event.ButtonNone, ke.Mod)
		b.dispatchEvent(click)
		e.StopPropagation()
	}
}

func (b *ButtonElement) dispatchEvent(e event.Event) {
	// Build the ancestor path for dispatch (root -> target).
	var path []event.EventTarget
	for p := dom.Node(b); p != nil; p = p.Parent() {
		path = append(path, p)
	}
	// Reverse the path.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	dispatcher := event.NewDispatcher()
	dispatcher.Dispatch(e, path)
}
