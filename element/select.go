package element

import (
	"fmt"
	"image/color"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/focus"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
)

// OptionElement represents a single option in a Select component.
type OptionElement struct {
	elementBase[OptionElement]
	value string
	text  string
}

var _ Element = (*OptionElement)(nil)

// NewOption creates a new OptionElement owned by doc.
func NewOption(doc dom.Document, text, value string) *OptionElement {
	o := &OptionElement{value: value, text: text}
	o.initBase(doc.CreateElement("option", o), o, style.Style{})
	return o
}

func (o *OptionElement) IntrinsicStyle() style.Style {
	return style.Style{Display: style.Some(style.DisplayNone)}
}

// Option creates a new OptionElement with the given text and value.
func Option(text, value string) *OptionElement {
	return NewOption(orphanDocument, text, value)
}

// SelectElement implements a dropdown selection component.
type SelectElement struct {
	elementBase[SelectElement]
	value    string
	options  []*OptionElement
	uaButton *ButtonElement
	overlay  *OverlayElement
	isOpen   bool
	onChange func(string)
	escSub   event.Subscription
	clickSub event.Subscription
}

var _ Element = (*SelectElement)(nil)

// NewSelect creates a new SelectElement owned by doc.
func NewSelect(doc dom.Document, children ...any) *SelectElement {
	s := &SelectElement{}
	el := doc.CreateElement("select", s)
	s.initBase(el, s, style.Style{}, style.Style{
		Display: style.Some(style.DisplayInlineBlock),
	})

	// UA Shadow Subtree: Trigger Button
	s.uaButton = NewButton(doc, "Select... ▼")
	uaRoot := doc.CreateElement("ua-select-root", nil)
	uaRoot.AppendChild(s.uaButton)
	el.AttachUARoot(uaRoot)

	s.processSelectChildren(children)

	s.wireEvents()
	s.syncValue()

	return s
}

func (s *SelectElement) processSelectChildren(children []any) {
	for _, child := range children {
		if child == nil {
			continue
		}
		switch v := child.(type) {
		case *OptionElement:
			s.options = append(s.options, v)
		case []*OptionElement:
			s.options = append(s.options, v...)
		case []any:
			s.processSelectChildren(v)
		case dom.Node:
			if opt, ok := v.EventTarget().(*OptionElement); ok {
				s.options = append(s.options, opt)
			} else {
				s.AppendChild(v)
			}
		default:
			// Fallback to standard child processing for strings, etc.
			processChildren(s, []any{child})
		}
	}
}

// Select creates a new SelectElement with the given children.
func Select(children ...any) *SelectElement {
	return NewSelect(orphanDocument, children...)
}

// Value returns the currently selected value.
func (s *SelectElement) Value() string {
	return s.value
}

// SetValue updates the selected value and updates the trigger button text.
func (s *SelectElement) SetValue(value string) *SelectElement {
	if s.value == value {
		return s
	}
	s.value = value
	s.syncValue()
	s.emitChange()
	return s
}

// OnChange sets a callback to be invoked when the selection changes.
func (s *SelectElement) OnChange(fn func(string)) *SelectElement {
	s.onChange = fn
	return s
}

func (s *SelectElement) wireEvents() {
	s.OnEvent(event.EventClick, func(e event.Event) {
		s.openDropdown()
	})
	s.OnEvent(event.EventMouseDown, func(e event.Event) {
		if me, ok := e.(*event.MouseEvent); ok && me.Button == event.ButtonLeft {
			s.uaButton.SetActive(true)
		}
	})
	s.OnEvent(event.EventMouseUp, func(e event.Event) {
		s.uaButton.SetActive(false)
	})
	s.OnEvent(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			if ke.MatchString(" ") || ke.MatchString("enter") {
				s.openDropdown()
				e.StopPropagation()
			}
		}
	})
}

func (s *SelectElement) syncValue() {
	text := "Select... ▼"
	if s.value != "" {
		for _, opt := range s.options {
			if opt.value == s.value {
				text = fmt.Sprintf("%s ▼", opt.text)
				break
			}
		}
	}
	s.uaButton.SetData(text)
}

func (s *SelectElement) openDropdown() {
	if s.isOpen {
		return
	}

	doc := s.OwnerDocument()
	if doc == nil {
		return
	}

	// Calculate width from Select element
	width := style.Cells(20)
	if ro := s.RenderObject(); ro != nil {
		width = style.Cells(ro.Fragment().Size.Width)
	}

	// Create overlay content
	content := Box().Style(style.Style{
		Width:      style.Some(width),
		MaxHeight:  style.Some(style.Cells(10)),
		Border:     style.SingleBorder().Some(),
		Background: style.Some[color.Color](color.RGBA{R: 30, G: 30, B: 30, A: 255}),
		OverflowY:  style.Some(style.OverflowAuto),
	})
	content.ScrollbarY(true)

	var firstBtn *ButtonElement
	for _, opt := range s.options {
		btn := Button(opt.text).Style(style.Style{
			Width:     style.Some(style.Percent(100)),
			TextAlign: style.Some(style.TextAlignLeft),
			Padding:   style.Some(style.Edges(0, 1)),
			Border:    style.EmptyBorder().Some(),
		})
		if firstBtn == nil {
			firstBtn = btn
		}
		btn.OnEvent(event.EventClick, func(e event.Event) {
			s.selectOption(opt)
		})
		content.AddChild(btn)
	}

	s.overlay = NewOverlay(doc, content, OverlayConfig{
		Anchor:    s,
		Placement: layout.PlacementBottom,
		Flip:      true,
		ZIndex:    1000,
	})

	s.isOpen = true
	doc.AppendChild(s.overlay)

	if fm, ok := doc.FocusManager().(*focus.Manager); ok {
		fm.PushScope(&focus.Scope{Root: s.overlay, Autofocus: firstBtn})
	}

	// Add global escape listener to close dropdown
	s.escSub = doc.AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke, ok := e.(*event.KeyEvent)
		if ok && ke.MatchString("esc") {
			s.closeDropdown()
			e.StopPropagation()
		}
	}, event.Capture())

	// Add global click listener to close dropdown
	s.clickSub = doc.AddEventListener(event.EventMouseDown, func(e event.Event) {
		me := e.(*event.MouseEvent)
		// If click is outside the overlay and not on the select itself, close.
		if !s.isDescendant(me.Target()) && !s.overlay.Contains(me.Target().(dom.Node)) {
			s.closeDropdown()
		}
	}, event.Capture())
}

func (s *SelectElement) isDescendant(target event.EventTarget) bool {
	n, ok := target.(dom.Node)
	if !ok {
		return false
	}
	return s.Contains(n)
}

func (s *SelectElement) selectOption(opt *OptionElement) {
	s.SetValue(opt.value)
	s.closeDropdown()
}

func (s *SelectElement) closeDropdown() {
	if !s.isOpen {
		return
	}
	s.isOpen = false
	if s.escSub != nil {
		s.escSub.Cancel()
		s.escSub = nil
	}
	if s.clickSub != nil {
		s.clickSub.Cancel()
		s.clickSub = nil
	}

	doc := s.OwnerDocument()
	if doc == nil {
		return
	}

	doc.RemoveChild(s.overlay)
	s.overlay = nil

	if fm, ok := doc.FocusManager().(*focus.Manager); ok {
		fm.PopScope()
	}
	s.uaButton.Focus()
}

func (s *SelectElement) emitChange() {
	if s.onChange != nil {
		s.onChange(s.value)
	}
	s.DispatchEvent(event.NewChange(s.value))
	if ro := s.RenderObject(); ro != nil {
		ro.MarkDirty(render.DirtyStyle)
	}
}

func (s *SelectElement) SetData(data string) {
	// Delegate to uaButton for simple text updates if used directly
	s.uaButton.SetData(data)
}
