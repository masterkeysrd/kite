package kitex

//go:generate go run ./internal/genprops

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
)

// --- VDOM Factories -----------------------------------------------------------

type BoxProps = ElementProps
type SpanProps = ElementProps
type BrProps = ElementProps
type ULProps = ElementProps
type OLProps = ElementProps
type LIProps = ElementProps
type TableProps = ElementProps
type THeadProps = ElementProps
type TBodyProps = ElementProps
type TFootProps = ElementProps
type TRProps = ElementProps

func boxInstantiate(doc dom.Document) dom.Node {
	return element.NewBox(doc)
}

func boxUpdate(el dom.Node, old, new *BoxProps) {
	updateElementBase(el.(element.Element), old, new)
}

// Box creates a VDOM representation of a BoxElement container.
var Box ContainerTag[BoxProps] = func(props BoxProps, children ...Node) Node {
	n := elementNodePool.Get().(*elementNode[ElementProps])
	score, hasP, hasDirectP := buildElementInfo(children)
	*n = elementNode[ElementProps]{
		tagName:     "box",
		props:       props,
		children:    children,
		instantiate: boxInstantiate,
		update:      boxUpdate,
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}
	return trackSource(n, 1)
}

// Div creates a VDOM representation of a BoxElement container (web div alias).
var Div ContainerTag[BoxProps] = func(props BoxProps, children ...Node) Node {
	n := elementNodePool.Get().(*elementNode[ElementProps])
	score, hasP, hasDirectP := buildElementInfo(children)
	*n = elementNode[ElementProps]{
		tagName:     "div",
		props:       props,
		children:    children,
		instantiate: boxInstantiate,
		update:      boxUpdate,
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}
	return trackSource(n, 1)
}

func spanInstantiate(doc dom.Document) dom.Node {
	return element.NewSpan(doc)
}

func spanUpdate(el dom.Node, old, new *SpanProps) {
	updateElementBase(el.(element.Element), old, new)
}

// Span creates a VDOM representation of a SpanElement container.
var Span ContainerTag[SpanProps] = func(props SpanProps, children ...Node) Node {
	n := elementNodePool.Get().(*elementNode[ElementProps])
	score, hasP, hasDirectP := buildElementInfo(children)
	*n = elementNode[ElementProps]{
		tagName:     "span",
		props:       props,
		children:    children,
		instantiate: spanInstantiate,
		update:      spanUpdate,
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}
	return trackSource(n, 1)
}

func buttonInstantiate(doc dom.Document) dom.Node {
	return element.NewButton(doc)
}

func buttonUpdate(el dom.Node, old, new *ButtonProps) {
	btn := el.(*element.ButtonElement)
	var oldEp ElementProps
	var oldPress func()
	if old != nil {
		oldEp = old.elementProps()
		oldPress = old.OnPress
	}
	newEp := new.elementProps()
	updateElementBase(btn, &oldEp, &newEp)
	updatePressListener(btn, oldPress, new.OnPress)
	if old != nil && old.Disabled != new.Disabled {
		btn.Disabled(new.Disabled)
	} else if old == nil {
		btn.Disabled(new.Disabled)
	}
	if old != nil && old.Active != new.Active {
		btn.SetActive(new.Active)
	} else if old == nil {
		btn.SetActive(new.Active)
	}
	if old != nil && old.Type != new.Type {
		btn.Type(new.Type)
	} else if old == nil {
		btn.Type(new.Type)
	}
}

// Button creates a VDOM representation of a ButtonElement.
var Button ContainerTag[ButtonProps] = func(props ButtonProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[ButtonProps]{
		tagName:     "button",
		props:       props,
		children:    children,
		instantiate: buttonInstantiate,
		update:      buttonUpdate,
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

func checkboxInstantiate(doc dom.Document) dom.Node {
	return element.NewCheckbox(doc, false)
}

func checkboxUpdate(el dom.Node, old, new *CheckboxProps) {
	cb := el.(*element.CheckboxElement)
	var oldEp ElementProps
	if old != nil {
		oldEp = old.elementProps()
	}
	newEp := new.elementProps()
	updateElementBase(cb, &oldEp, &newEp)
	if old != nil && old.Checked != new.Checked {
		cb.SetChecked(new.Checked)
	} else if old == nil {
		cb.SetChecked(new.Checked)
	}
	if old == nil || !funcEquals(old.OnCheckedChange, new.OnCheckedChange) {
		cb.OnChange(new.OnCheckedChange)
	}
	if old == nil || (old.UncheckedGlyph != new.UncheckedGlyph || old.CheckedGlyph != new.CheckedGlyph) {
		un := "[ ]"
		ch := "[X]"
		if new.UncheckedGlyph != "" {
			un = new.UncheckedGlyph
		}
		if new.CheckedGlyph != "" {
			ch = new.CheckedGlyph
		}
		cb.SetGlyphs(un, ch)
	}
	if old != nil && old.Name != new.Name {
		cb.WithName(new.Name)
	} else if old == nil {
		cb.WithName(new.Name)
	}
}

// Checkbox creates a VDOM representation of a CheckboxElement.
var Checkbox LeafTag[CheckboxProps] = func(props CheckboxProps) Node {
	score, hasP, hasDirectP := buildElementInfo(nil)
	return trackSource(&elementNode[CheckboxProps]{
		tagName:     "checkbox",
		props:       props,
		children:    nil,
		instantiate: checkboxInstantiate,
		update:      checkboxUpdate,
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

func radioGroupInstantiate(doc dom.Document) dom.Node {
	return element.NewRadioGroup(doc)
}

func radioGroupUpdate(el dom.Node, old, new *RadioGroupProps) {
	rg := el.(*element.RadioGroupElement)
	var oldEp ElementProps
	if old != nil {
		oldEp = old.elementProps()
	}
	newEp := new.elementProps()
	updateElementBase(rg, &oldEp, &newEp)
	if old != nil && old.Value != new.Value {
		rg.SetValue(new.Value)
	} else if old == nil {
		rg.SetValue(new.Value)
	}
	if old == nil || !funcEquals(old.OnValueChange, new.OnValueChange) {
		rg.OnChange(new.OnValueChange)
	}
}

// RadioGroup creates a VDOM representation of a RadioGroupElement.
var RadioGroup ContainerTag[RadioGroupProps] = func(props RadioGroupProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[RadioGroupProps]{
		tagName:     "radiogroup",
		props:       props,
		children:    children,
		instantiate: radioGroupInstantiate,
		update:      radioGroupUpdate,
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

func radioInstantiate(doc dom.Document) dom.Node {
	return element.NewRadio(doc, "")
}

func radioUpdate(el dom.Node, old, new *RadioProps) {
	r := el.(*element.RadioElement)
	var oldEp ElementProps
	if old != nil {
		oldEp = old.elementProps()
	}
	newEp := new.elementProps()
	updateElementBase(r, &oldEp, &newEp)
	if (old == nil || old.Value != new.Value) && new.Value != "" {
		r.SetValue(new.Value)
	}
	if old == nil || (old.UncheckedGlyph != new.UncheckedGlyph || old.CheckedGlyph != new.CheckedGlyph) {
		un := "( )"
		ch := "(•)"
		if new.UncheckedGlyph != "" {
			un = new.UncheckedGlyph
		}
		if new.CheckedGlyph != "" {
			ch = new.CheckedGlyph
		}
		r.SetGlyphs(un, ch)
	}
	if old != nil && old.Name != new.Name {
		r.WithName(new.Name)
	} else if old == nil {
		r.WithName(new.Name)
	}
}

// Radio creates a VDOM representation of a RadioElement.
var Radio LeafTag[RadioProps] = func(props RadioProps) Node {
	score, hasP, hasDirectP := buildElementInfo(nil)
	return trackSource(&elementNode[RadioProps]{
		tagName:     "radio",
		props:       props,
		children:    nil,
		instantiate: radioInstantiate,
		update:      radioUpdate,
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// Select creates a VDOM representation of a SelectElement.
var Select ContainerTag[SelectProps] = func(props SelectProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[SelectProps]{
		tagName:  "select",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewSelect(doc)
		},
		update: func(el dom.Node, old, new *SelectProps) {
			s := el.(*element.SelectElement)
			var oldEp ElementProps
			if old != nil {
				oldEp = old.elementProps()
			}
			newEp := new.elementProps()
			updateElementBase(s, &oldEp, &newEp)
			if old != nil && old.Value != new.Value {
				s.SetValue(new.Value)
			} else if old == nil {
				s.SetValue(new.Value)
			}
			if old == nil || !funcEquals(old.OnValueChange, new.OnValueChange) {
				s.OnChange(new.OnValueChange)
			}
			// Synchronize options slice from DOM children list
			var opts []*element.OptionElement
			for child := range s.ChildNodes() {
				if opt, ok := child.EventTarget().(*element.OptionElement); ok {
					opts = append(opts, opt)
				}
			}
			s.SetOptions(opts)
			if old != nil && old.Name != new.Name {
				s.WithName(new.Name)
			} else if old == nil {
				s.WithName(new.Name)
			}
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// Option creates a VDOM representation of an OptionElement metadata node.
var Option LeafTag[OptionProps] = func(props OptionProps) Node {
	return trackSource(&elementNode[OptionProps]{
		tagName:  "option",
		props:    props,
		children: nil,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewOption(doc, props.Text, props.Value)
		},
		update: func(el dom.Node, old, new *OptionProps) {
			opt := el.(*element.OptionElement)
			var oldEp ElementProps
			if old != nil {
				oldEp = old.elementProps()
			}
			newEp := new.elementProps()
			updateElementBase(opt, &oldEp, &newEp)
			if old != nil && old.Text != new.Text {
				opt.SetText(new.Text)
			} else if old == nil {
				opt.SetText(new.Text)
			}
			if old != nil && old.Value != new.Value {
				opt.SetValue(new.Value)
			} else if old == nil {
				opt.SetValue(new.Value)
			}
		},
		key:         props.Key,
		score:       1,
		hasProvider: false,
		hasDirectP:  false,
	}, 1)
}

// Input creates a VDOM representation of an InputElement.
var Input LeafTag[InputProps] = func(props InputProps) Node {
	return trackSource(&elementNode[InputProps]{
		tagName:  "input",
		props:    props,
		children: nil,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewInput(doc, props.Value).
				WithPlaceholder(props.Placeholder).
				WithPlaceholderStyle(props.PlaceholderStyle)
		},
		update: func(el dom.Node, old, new *InputProps) {
			inp := el.(*element.InputElement)
			var oldEp ElementProps
			if old != nil {
				oldEp = old.elementProps()
			}
			newEp := new.elementProps()
			updateElementBase(inp, &oldEp, &newEp)

			if old == nil || !funcEquals(old.OnValueChange, new.OnValueChange) {
				inp.OnChange(new.OnValueChange)
			}

			if inp.Text() != new.Value {
				inp.SetValue(new.Value)
			}
			if old != nil && old.Name != new.Name {
				inp.WithName(new.Name)
			} else if old == nil {
				inp.WithName(new.Name)
			}
			if old != nil && old.Placeholder != new.Placeholder {
				inp.WithPlaceholder(new.Placeholder)
			} else if old == nil {
				inp.WithPlaceholder(new.Placeholder)
			}
			inp.WithPlaceholderStyle(new.PlaceholderStyle)
		},
		key:         props.Key,
		score:       1,
		hasProvider: false,
		hasDirectP:  false,
	}, 1)
}

// TextArea creates a VDOM representation of a TextAreaElement.
var TextArea LeafTag[TextAreaProps] = func(props TextAreaProps) Node {
	return trackSource(&elementNode[TextAreaProps]{
		tagName:  "textarea",
		props:    props,
		children: nil,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTextArea(doc, props.Value).
				WithPlaceholder(props.Placeholder).
				WithPlaceholderStyle(props.PlaceholderStyle)
		},
		update: func(el dom.Node, old, new *TextAreaProps) {
			txa := el.(*element.TextAreaElement)
			var oldEp ElementProps
			if old != nil {
				oldEp = old.elementProps()
			}
			newEp := new.elementProps()
			updateElementBase(txa, &oldEp, &newEp)

			if old == nil || !funcEquals(old.OnValueChange, new.OnValueChange) {
				txa.OnChange(new.OnValueChange)
			}

			if txa.Text() != new.Value {
				txa.SetValue(new.Value)
			}
			if old != nil && old.Name != new.Name {
				txa.WithName(new.Name)
			} else if old == nil {
				txa.WithName(new.Name)
			}
			if old != nil && old.Placeholder != new.Placeholder {
				txa.WithPlaceholder(new.Placeholder)
			} else if old == nil {
				txa.WithPlaceholder(new.Placeholder)
			}
			txa.WithPlaceholderStyle(new.PlaceholderStyle)
		},
		key:         props.Key,
		score:       1,
		hasProvider: false,
		hasDirectP:  false,
	}, 1)
}

// Table creates a VDOM representation of a TableElement.
var Table ContainerTag[TableProps] = func(props TableProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[TableProps]{
		tagName:  "table",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTable(doc)
		},
		update: func(el dom.Node, old, new *TableProps) {
			updateElementBase(el.(element.Element), old, new)
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// UL creates a VDOM representation of an UnorderedListElement (ul).
var UL ContainerTag[ULProps] = func(props ULProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[ULProps]{
		tagName:  "ul",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewUnorderedList(doc)
		},
		update: func(el dom.Node, old, new *ULProps) {
			updateElementBase(el.(element.Element), old, new)
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// OL creates a VDOM representation of an OrderedListElement (ol).
var OL ContainerTag[OLProps] = func(props OLProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[OLProps]{
		tagName:  "ol",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewOrderedList(doc)
		},
		update: func(el dom.Node, old, new *OLProps) {
			updateElementBase(el.(element.Element), old, new)
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// LI creates a VDOM representation of a ListItemElement (li).
var LI ContainerTag[LIProps] = func(props LIProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[LIProps]{
		tagName:  "li",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewListItem(doc)
		},
		update: func(el dom.Node, old, new *LIProps) {
			updateElementBase(el.(element.Element), old, new)
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// THead creates a VDOM representation of a TableHeaderElement (thead).
var THead ContainerTag[THeadProps] = func(props THeadProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[THeadProps]{
		tagName:  "thead",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTableHeader(doc)
		},
		update: func(el dom.Node, old, new *THeadProps) {
			updateElementBase(el.(element.Element), old, new)
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// TBody creates a VDOM representation of a TableBodyElement (tbody).
var TBody ContainerTag[TBodyProps] = func(props TBodyProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[TBodyProps]{
		tagName:  "tbody",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTableBody(doc)
		},
		update: func(el dom.Node, old, new *TBodyProps) {
			updateElementBase(el.(element.Element), old, new)
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// TFoot creates a VDOM representation of a TableFooterElement (tfoot).
var TFoot ContainerTag[TFootProps] = func(props TFootProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[TFootProps]{
		tagName:  "tfoot",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTableFooter(doc)
		},
		update: func(el dom.Node, old, new *TFootProps) {
			updateElementBase(el.(element.Element), old, new)
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// TR creates a VDOM representation of a TableRowElement (tr).
var TR ContainerTag[TRProps] = func(props TRProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[TRProps]{
		tagName:  "tr",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTableRow(doc)
		},
		update: func(el dom.Node, old, new *TRProps) {
			updateElementBase(el.(element.Element), old, new)
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// TD creates a VDOM representation of a TableCellElement (td).
var TD ContainerTag[TDProps] = func(props TDProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[TDProps]{
		tagName:  "td",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTableCell(doc)
		},
		update: func(el dom.Node, old, new *TDProps) {
			td := el.(*element.TableCellElement)
			var oldEp ElementProps
			if old != nil {
				oldEp = old.elementProps()
			}
			newEp := new.elementProps()
			updateElementBase(td, &oldEp, &newEp)
			if old == nil || old.ColSpan != new.ColSpan {
				td.SetColSpan(new.ColSpan)
			}
			if old == nil || old.RowSpan != new.RowSpan {
				td.SetRowSpan(new.RowSpan)
			}
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// Br creates a VDOM representation of a BrElement.
var Br LeafTag[BrProps] = func(props BrProps) Node {
	return trackSource(&elementNode[BrProps]{
		tagName:  "br",
		props:    props,
		children: nil,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewBr(doc)
		},
		update: func(el dom.Node, old, new *BrProps) {
			updateElementBase(el.(element.Element), old, new)
		},
		key:         props.Key,
		score:       1,
		hasProvider: false,
		hasDirectP:  false,
	}, 1)
}

// Overlay creates a VDOM representation of an OverlayElement.
var Overlay ContainerTag[OverlayProps] = func(props OverlayProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[OverlayProps]{
		tagName:  "overlay",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewOverlay(doc, props.Anchor).
				WithZIndex(props.ZIndex).
				WithPlacement(props.Placement).
				WithFlip(props.Flip)
		},
		update: func(el dom.Node, old, new *OverlayProps) {
			o := el.(*element.OverlayElement)
			var oldEp ElementProps
			if old != nil {
				oldEp = old.elementProps()
			}
			newEp := new.elementProps()
			updateElementBase(o, &oldEp, &newEp)
			if old == nil || (old.Anchor != new.Anchor || old.ZIndex != new.ZIndex || old.Placement != new.Placement || old.Flip != new.Flip) {
				config := element.OverlayConfig{
					Anchor:    new.Anchor,
					ZIndex:    new.ZIndex,
					Placement: new.Placement,
					Flip:      new.Flip,
				}
				o.SetConfig(config)
			}
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}

// Dialog creates a VDOM representation of a DialogElement.
var Dialog ContainerTag[DialogProps] = func(props DialogProps, children ...Node) Node {
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[DialogProps]{
		tagName:  "dialog",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewDialog(doc).ZIndex(props.ZIndex)
		},
		update: func(el dom.Node, old, new *DialogProps) {
			d := el.(*element.DialogElement)
			var oldEp ElementProps
			if old != nil {
				oldEp = old.elementProps()
			}
			newEp := new.elementProps()
			updateElementBase(d, &oldEp, &newEp)
			if old == nil || old.ZIndex != new.ZIndex {
				d.SetZIndex(new.ZIndex)
			}
		},
		key:         props.Key,
		score:       score,
		hasProvider: hasP,
		hasDirectP:  hasDirectP,
	}, 1)
}
