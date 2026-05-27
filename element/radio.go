package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/internal/render"
	"github.com/masterkeysrd/kite/style"
)

// RadioGroupElement manages a set of RadioElements and ensures only one is selected at a time.
type RadioGroupElement struct {
	elementBase[RadioGroupElement]
	value    string
	onChange func(string)
}

var _ Element = (*RadioGroupElement)(nil)

// NewRadioGroup creates a new RadioGroupElement owned by doc.
func NewRadioGroup(doc dom.Document, children ...any) *RadioGroupElement {
	rg := &RadioGroupElement{}
	el := doc.CreateElement("radiogroup", rg)
	rg.initBase(el, rg, style.Style{})
	processChildren(rg, children)
	rg.syncRadios()
	return rg
}

// RadioGroup creates a new RadioGroupElement with the given children.
func RadioGroup(children ...any) *RadioGroupElement {
	return NewRadioGroup(orphanDocument, children...)
}

// Value returns the currently selected value in the group.
func (rg *RadioGroupElement) Value() string {
	return rg.value
}

// SetValue updates the selected value and synchronizes all child radio buttons.
func (rg *RadioGroupElement) SetValue(value string) *RadioGroupElement {
	if rg.value == value {
		return rg
	}
	rg.value = value
	rg.syncRadios()
	rg.emitChange()
	return rg
}

// OnChange sets a callback to be invoked when the selected value changes.
func (rg *RadioGroupElement) OnChange(fn func(string)) *RadioGroupElement {
	rg.onChange = fn
	return rg
}

func (rg *RadioGroupElement) syncRadios() {
	rg.walkRadios(rg, func(r *RadioElement) {
		r.updateVisual()
	})
}

func (rg *RadioGroupElement) walkRadios(n dom.Node, fn func(*RadioElement)) {
	for child := range n.ChildNodes() {
		if r, ok := child.EventTarget().(*RadioElement); ok {
			fn(r)
		}
		rg.walkRadios(child, fn)
	}
}

func (rg *RadioGroupElement) notifySelected(value string) {
	rg.SetValue(value)
}

func (rg *RadioGroupElement) emitChange() {
	if rg.onChange != nil {
		rg.onChange(rg.value)
	}
	rg.DispatchEvent(event.NewChange(rg.value))
}

// AddChild overrides elementBase.AddChild to sync new radio children.
func (rg *RadioGroupElement) AddChild(child dom.Node) *RadioGroupElement {
	rg.elementBase.AddChild(child)
	rg.syncRadios()
	return rg
}

// AppendChild overrides dom.Element.AppendChild to sync new radio children.
func (rg *RadioGroupElement) AppendChild(child dom.Node) dom.Node {
	res := rg.Element.AppendChild(child)
	rg.syncRadios()
	return res
}

// RadioElement represents a single radio button within a RadioGroup.
type RadioElement struct {
	elementBase[RadioElement]
	value          string
	uncheckedGlyph string
	checkedGlyph   string
	uaText         dom.TextNode
	disabled       bool
	name           string
}

var (
	_ Element         = (*RadioElement)(nil)
	_ dom.FormControl = (*RadioElement)(nil)
)

// NewRadio creates a new RadioElement owned by doc with the given value.
func NewRadio(doc dom.Document, value string) *RadioElement {
	r := &RadioElement{
		value:          value,
		uncheckedGlyph: "( )",
		checkedGlyph:   "(•)",
	}

	el := doc.CreateElement("radio", r)
	r.initBase(el, r, style.Style{}, style.Style{
		Display: style.Some(style.DisplayInlineBlock),
	})

	// UA Shadow Subtree: a box containing the glyph text.
	r.uaText = doc.CreateTextNode("", nil)
	uaBox := doc.CreateElement("ua-radio-box", nil)
	uaBox.AppendChild(r.uaText)
	el.AttachUARoot(uaBox)

	r.wireEvents()
	r.updateVisual()

	return r
}

// Radio creates a new RadioElement with the given value.
func Radio(value string) *RadioElement {
	return NewRadio(orphanDocument, value)
}

// Checked reports whether the radio button is selected.
func (r *RadioElement) Checked() bool {
	if rg := r.findGroup(); rg != nil {
		return rg.Value() == r.value
	}
	return false
}

// Value returns the value associated with this radio button.
func (r *RadioElement) Value() any {
	return r.value
}

// WithName sets the form control name and returns the RadioElement.
func (r *RadioElement) WithName(name string) *RadioElement {
	r.name = name
	return r
}

// Name returns the form control name.
func (r *RadioElement) Name() string {
	return r.name
}

// SetValue updates the radio button's value.
func (r *RadioElement) SetValue(value string) *RadioElement {
	r.value = value
	r.updateVisual()
	return r
}

// SetGlyphs customizes the strings used for unchecked and checked states.
func (r *RadioElement) SetGlyphs(unchecked, checked string) *RadioElement {
	r.uncheckedGlyph = unchecked
	r.checkedGlyph = checked
	r.updateVisual()
	return r
}

func (r *RadioElement) updateVisual() {
	checked := false
	if rg := r.findGroup(); rg != nil {
		checked = rg.Value() == r.value
	}
	glyph := r.uncheckedGlyph
	if checked {
		glyph = r.checkedGlyph
	}
	r.uaText.SetData(glyph)
	if ro := r.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyStyle)
	}
}

func (r *RadioElement) findGroup() *RadioGroupElement {
	for p := r.Parent(); p != nil; p = p.Parent() {
		if rg, ok := p.EventTarget().(*RadioGroupElement); ok {
			return rg
		}
	}
	return nil
}

func (r *RadioElement) wireEvents() {
	r.OnEvent(event.EventClick, r.handleClick)
	r.OnEvent(event.EventKeyDown, r.handleKeyDown)
}

func (r *RadioElement) handleClick(e event.Event) {
	if r.IsDisabled() {
		return
	}
	r.selectSelf()
}

func (r *RadioElement) handleKeyDown(e event.Event) {
	if r.IsDisabled() {
		return
	}
	ke, ok := e.(*event.KeyEvent)
	if !ok {
		return
	}
	if ke.MatchString(" ") {
		r.selectSelf()
	}
}

func (r *RadioElement) selectSelf() {
	if rg := r.findGroup(); rg != nil {
		rg.notifySelected(r.value)
	}
}

func (r *RadioElement) IsDisabled() bool   { return r.disabled }
func (r *RadioElement) SetDisabled(v bool) { r.disabled = v }
func (r *RadioElement) Disabled(v bool) *RadioElement {
	r.disabled = v
	return r
}
