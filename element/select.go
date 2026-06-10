package element

import (
	"fmt"
	"image/color"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	internaldom "github.com/masterkeysrd/kite/internal/dom"
	"github.com/masterkeysrd/kite/internal/focus"
	"github.com/masterkeysrd/kite/style"
)

// OptionElement represents a single option in a Select component.
// It is a metadata-only element that does not participate in the main render tree.
type OptionElement struct {
	elementBase[OptionElement]
	value    string
	text     string
	disabled bool
}

var _ Element = (*OptionElement)(nil)

// NewOption creates a new OptionElement owned by doc.
func NewOption(doc dom.Document, text, value string) *OptionElement {
	o := &OptionElement{value: value, text: text}
	o.initBase(doc.CreateElement("option", o), o, style.S())
	return o
}

var intrinsicOptionStyle = style.S().
	Display(style.DisplayNone)

func (o *OptionElement) IntrinsicStyle() style.Style {
	// Options are metadata-only; they should never produce render objects.
	return intrinsicOptionStyle
}

// Option creates a new OptionElement with the given text and value.
func Option(text, value string) *OptionElement {
	return NewOption(orphanDocument, text, value)
}

// SetText sets the display text of the option.
func (o *OptionElement) SetText(text string) *OptionElement {
	o.text = text
	return o
}

// SetValue sets the value of the option.
func (o *OptionElement) SetValue(value string) *OptionElement {
	o.value = value
	return o
}

func (o *OptionElement) IsDisabled() bool   { return o.disabled }
func (o *OptionElement) SetDisabled(v bool) { o.disabled = v }
func (o *OptionElement) Disabled(v bool) *OptionElement {
	o.disabled = v
	return o
}

type uaSelectRoot struct {
	dom.Element
}

func (r *uaSelectRoot) Unwrap() dom.Node { return r.Element }

var defaultSelectStyle = style.S().
	Width(style.Cells(20))

var intrinsicSelectStyle = style.S().
	Display(style.DisplayInlineBlock).
	AlignSelf(style.AlignStart)

var selectButtonStyle = style.S().
	Display(style.DisplayBlock).
	Width(style.Percent(100)).
	Height(style.Auto)

var selectDropdownBaseStyle = style.S().
	Display(style.DisplayFlex).
	FlexDirection(style.FlexColumn).
	MaxHeight(style.Cells(10)).
	Border(style.SingleBorder(), color.RGBA{R: 100, G: 100, B: 150, A: 255}).
	Background(color.RGBA{R: 25, G: 25, B: 35, A: 255}).
	OverflowY(style.OverflowAuto)

var selectOptionButtonStyle = style.S().
	Display(style.DisplayBlock).
	Width(style.Percent(100)).
	TextAlign(style.TextAlignLeft).
	PaddingHorizontal(1).
	Border(style.EmptyBorder())

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

	name     string
	disabled bool
}

var (
	_ Element         = (*SelectElement)(nil)
	_ dom.FormControl = (*SelectElement)(nil)
)

// NewSelect creates a new SelectElement owned by doc.
func NewSelect(doc dom.Document, children ...any) *SelectElement {
	s := &SelectElement{}
	el := doc.CreateElement("select", s)
	s.initBase(el, s, defaultSelectStyle, intrinsicSelectStyle)

	// UA Shadow Subtree: Trigger Button. We force DisplayBlock here so that
	// the button correctly fills the host's available width even when the
	// host is an InlineBlock (as is the case for SelectElement).
	s.uaButton = NewButton(doc, "Select... ▼").Style(selectButtonStyle)
	uaRoot := &uaSelectRoot{}
	uaRootEl := doc.CreateElement("ua-select-root", uaRoot)
	uaRoot.Element = uaRootEl
	uaRoot.AppendChild(s.uaButton)
	el.AttachUARoot(uaRoot)

	s.processSelectChildren(children)

	s.wireEvents()
	s.syncValue()

	return s
}

// AppendChild overrides dom.Element.AppendChild to sync select options.
func (s *SelectElement) AppendChild(child dom.Node) dom.Node {
	res := s.Element.AppendChild(child)
	if opt, ok := child.EventTarget().(*OptionElement); ok {
		s.options = append(s.options, opt)
		s.syncValue()
	}
	return res
}

// RemoveChild overrides dom.Element.RemoveChild to sync select options.
func (s *SelectElement) RemoveChild(child dom.Node) dom.Node {
	res := s.Element.RemoveChild(child)
	if opt, ok := child.EventTarget().(*OptionElement); ok {
		for i, o := range s.options {
			if o == opt {
				s.options = append(s.options[:i], s.options[i+1:]...)
				break
			}
		}
		s.syncValue()
	}
	return res
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
			// If it's a node but not an OptionElement, add it to the group's metadata
			// if it wraps one, otherwise ignore or handle as public child.
			//
			// We do NOT add options to the public children list to avoid
			// rendering them in the main flow.
			if opt, ok := v.EventTarget().(*OptionElement); ok {
				s.options = append(s.options, opt)
			}
		default:
			// Strings and other types are ignored for Select unless they are options.
		}
	}
}

// Select creates a new SelectElement with the given children.
func Select(children ...any) *SelectElement {
	return NewSelect(orphanDocument, children...)
}

// Value returns the currently selected value.
func (s *SelectElement) Value() any {
	return s.value
}

// WithName sets the form control name and returns the SelectElement.
func (s *SelectElement) WithName(name string) *SelectElement {
	s.name = name
	return s
}

// Name returns the form control name.
func (s *SelectElement) Name() string {
	return s.name
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

// SetOptions updates the list of options available in the dropdown.
func (s *SelectElement) SetOptions(options []*OptionElement) *SelectElement {
	s.options = options
	s.syncValue()
	return s
}

// OnChange sets a callback to be invoked when the selection changes.
func (s *SelectElement) OnChange(fn func(string)) *SelectElement {
	s.onChange = fn
	return s
}

func (s *SelectElement) wireEvents() {
	s.OnEvent(event.EventClick, func(e event.Event) {
		if s.IsDisabled() {
			return
		}
		s.openDropdown(e)
	})
	s.OnEvent(event.EventMouseDown, func(e event.Event) {
		if s.IsDisabled() {
			return
		}
		if me, ok := e.(*event.MouseEvent); ok && me.Button == event.ButtonLeft {
			s.uaButton.SetActive(true)
		}
	})
	s.OnEvent(event.EventMouseUp, func(e event.Event) {
		if s.IsDisabled() {
			return
		}
		s.uaButton.SetActive(false)
	})
	s.OnEvent(event.EventKeyDown, func(e event.Event) {
		if s.IsDisabled() {
			return
		}
		if ke, ok := e.(*event.KeyEvent); ok {
			if ke.MatchString("up") || ke.MatchString("down") {
				s.openDropdown(e)
				e.PreventDefault()
				e.StopPropagation()
			} else if ke.MatchString(" ") || ke.MatchString("enter") {
				s.openDropdown(e)
				e.PreventDefault()
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

func (s *SelectElement) openDropdown(trigger event.Event) {
	if s.isOpen {
		return
	}

	doc := s.OwnerDocument()
	if doc == nil {
		return
	}

	// Calculate width from Select element
	width := style.Cells(25)
	if d := s.OwnerDocument(); d != nil {
		if v := d.DefaultView(); v != nil {
			if rect, ok := v.GetBoundingClientRect(s); ok {
				width = style.Cells(max(25, rect.Size.Width))
			}
		}
	}

	// Create overlay content
	content := Box().Style(selectDropdownBaseStyle.Width(width))
	content.ScrollbarY(true)

	for _, opt := range s.options {
		btn := Button(" " + opt.text).WithClass("select-option").Style(selectOptionButtonStyle)
		btn.OnEvent(event.EventClick, func(e event.Event) {
			s.selectOption(opt)
		})
		content.AddChild(btn)
	}

	s.overlay = NewOverlay(doc, content, OverlayConfig{
		Anchor:    s,
		Placement: geom.PlacementBottom,
		Flip:      true,
		ZIndex:    1000,
	})

	s.isOpen = true
	doc.AppendChild(s.overlay)

	// Determine initial focus
	var autofocus dom.Node
	if s.value != "" {
		// Find the button corresponding to the current value
		for i, opt := range s.options {
			if opt.value == s.value {
				// The content Box has buttons as children in the same order as s.options
				childIdx := 0
				for child := range content.ChildNodes() {
					if childIdx == i {
						autofocus = child
						break
					}
					childIdx++
				}
				break
			}
		}
	}
	if autofocus == nil {
		autofocus = content.FirstChild()
	}

	doc.PushScope(&focus.Scope{Root: s.overlay, Autofocus: autofocus.(dom.Element)})

	// Handle initial navigation if opened via arrow keys
	if trigger != nil {
		if ke, ok := trigger.(*event.KeyEvent); ok {
			if ke.MatchString("down") {
				if s.value != "" {
					doc.NextFocus()
				}
			} else if ke.MatchString("up") {
				if s.value == "" {
					// wrap to last
					doc.PreviousFocus()
				} else {
					doc.PreviousFocus()
				}
			}
		}
	}

	// Add global escape listener to close dropdown
	s.escSub = doc.AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke, ok := e.(*event.KeyEvent)
		if !ok {
			return
		}
		if ke.MatchString("esc") {
			s.closeDropdown()
			e.PreventDefault()
			e.StopPropagation()
		} else if ke.MatchString("up") {
			doc.PreviousFocus()
			e.PreventDefault()
			e.StopPropagation()
		} else if ke.MatchString("down") {
			doc.NextFocus()
			e.PreventDefault()
			e.StopPropagation()
		} else if ke.MatchString("enter") || ke.MatchString(" ") {
			if focused := doc.CurrentFocus(); focused != nil {
				if btn, ok := focused.(*ButtonElement); ok {
					btn.DispatchEvent(event.NewMouseEvent(event.EventClick, geom.Point{}, event.ButtonLeft, 0))
					e.PreventDefault()
					e.StopPropagation()
				}
			}
		}
	}, event.Capture())

	// Add global click listener to close dropdown
	s.clickSub = doc.AddEventListener(event.EventMouseDown, func(e event.Event) {
		me := e.(*event.MouseEvent)
		target, ok := me.Target().(dom.Node)
		if !ok {
			return
		}
		// If click is outside the overlay and not on the select itself, close.
		if !s.Contains(target) && !s.overlay.Contains(target) {
			s.closeDropdown()
		}
	}, event.Capture())
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

	doc.PopScope()
	s.uaButton.Focus()
}

func (s *SelectElement) emitChange() {
	if s.onChange != nil {
		s.onChange(s.value)
	}
	s.DispatchEvent(event.NewChange(s.value))
	if d := internaldom.AsDirty(s); d != nil {
		d.MarkNeedsSync()
	}
}

func (s *SelectElement) SetData(data string) {
	s.uaButton.SetData(data)
}

func (s *SelectElement) IsDisabled() bool   { return s.disabled }
func (s *SelectElement) SetDisabled(v bool) { s.disabled = v }
func (s *SelectElement) Disabled(v bool) *SelectElement {
	s.disabled = v
	return s
}
