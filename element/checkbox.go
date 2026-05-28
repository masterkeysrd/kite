package element

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	internaldom "github.com/masterkeysrd/kite/internal/dom"
	"github.com/masterkeysrd/kite/style"
)

// CheckboxElement implements a toggleable checkbox component.
type CheckboxElement struct {
	elementBase[CheckboxElement]
	checked        bool
	uncheckedGlyph string
	checkedGlyph   string
	uaText         dom.TextNode

	name     string
	disabled bool
}

var (
	_ Element         = (*CheckboxElement)(nil)
	_ dom.FormControl = (*CheckboxElement)(nil)
)

// NewCheckbox creates a new CheckboxElement owned by doc.
func NewCheckbox(doc dom.Document, checked bool) *CheckboxElement {
	c := &CheckboxElement{
		checked:        checked,
		uncheckedGlyph: "[ ]",
		checkedGlyph:   "[X]",
	}

	el := doc.CreateElement("checkbox", c)
	c.initBase(el, c, style.Style{}, style.Style{
		Display: style.Some(style.DisplayInlineBlock),
	})

	// UA Shadow Subtree: a box containing the glyph text.
	c.uaText = doc.CreateTextNode(c.currentGlyph(), nil)
	uaBox := doc.CreateElement("ua-checkbox-box", nil)
	uaBox.AppendChild(c.uaText)
	el.AttachUARoot(uaBox)

	c.wireEvents()

	return c
}

// Checkbox creates a new CheckboxElement with the given checked state.
func Checkbox(checked bool) *CheckboxElement {
	return NewCheckbox(orphanDocument, checked)
}

// Checked reports whether the checkbox is checked.
func (c *CheckboxElement) Checked() bool {
	return c.checked
}

// Value returns the checked state of the checkbox.
func (c *CheckboxElement) Value() any {
	return c.checked
}

// WithName sets the form control name and returns the CheckboxElement.
func (c *CheckboxElement) WithName(name string) *CheckboxElement {
	c.name = name
	return c
}

// Name returns the form control name.
func (c *CheckboxElement) Name() string {
	return c.name
}

// SetChecked sets the checked state and updates the UI.
func (c *CheckboxElement) SetChecked(checked bool) *CheckboxElement {
	if c.checked == checked {
		return c
	}
	c.checked = checked
	c.uaText.SetData(c.currentGlyph())
	c.emitChange()
	return c
}

// SetGlyphs customizes the strings used for unchecked and checked states.
func (c *CheckboxElement) SetGlyphs(unchecked, checked string) *CheckboxElement {
	c.uncheckedGlyph = unchecked
	c.checkedGlyph = checked
	c.uaText.SetData(c.currentGlyph())
	return c
}

func (c *CheckboxElement) currentGlyph() string {
	if c.checked {
		return c.checkedGlyph
	}
	return c.uncheckedGlyph
}

func (c *CheckboxElement) wireEvents() {
	c.OnEvent(event.EventClick, c.handleClick)
	c.OnEvent(event.EventKeyDown, c.handleKeyDown)
}

func (c *CheckboxElement) handleClick(e event.Event) {
	if c.IsDisabled() {
		return
	}
	c.toggle()
}

func (c *CheckboxElement) handleKeyDown(e event.Event) {
	if c.IsDisabled() {
		return
	}
	ke, ok := e.(*event.KeyEvent)
	if !ok {
		return
	}
	if ke.MatchString(" ") {
		c.toggle()
	}
}

func (c *CheckboxElement) toggle() {
	c.checked = !c.checked
	c.uaText.SetData(c.currentGlyph())
	c.emitChange()
}

func (c *CheckboxElement) emitChange() {
	val := "false"
	if c.checked {
		val = "true"
	}
	c.DispatchEvent(event.NewChange(val))
	if d := internaldom.AsDirty(c); d != nil {
		d.MarkNeedsSync()
	}
}

func (c *CheckboxElement) IsDisabled() bool   { return c.disabled }
func (c *CheckboxElement) SetDisabled(v bool) { c.disabled = v }
func (c *CheckboxElement) Disabled(v bool) *CheckboxElement {
	c.disabled = v
	return c
}
