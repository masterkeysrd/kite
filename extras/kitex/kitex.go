package kitex

import (
	"reflect"
	"sync"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

// Node is the public interface representing a Virtual DOM node in the kitex framework.
type Node interface {
	// Instantiate constructs the corresponding real DOM node using the provided document.
	Instantiate(doc dom.Document) dom.Node

	// Update applies properties from the current VDOM node onto the existing real DOM node,
	// using the old VDOM node as reference to determine which properties changed.
	Update(el dom.Node, old Node)

	// Children returns the lightweight children of this VDOM node.
	Children() []Node

	// Props returns the raw properties struct or value of this VDOM node.
	Props() any

	// TagName returns the tag name or identifier of this VDOM node type (e.g. "button", "#text").
	TagName() string
}

// Ensure compile-time interface compliance.
var (
	_ Node = (*textNode)(nil)
	_ Node = (*elementNode[struct{}])(nil)
)

// ElementProps holds common fields and event listeners present on all DOM element nodes.
type ElementProps struct {
	ID          string
	Class       string
	Style       style.Style
	Hidden      bool
	OnKeyDown   func(event.Event)
	OnKeyUp     func(event.Event)
	OnKeyPress  func(event.Event)
	OnMouseDown func(event.Event)
	OnMouseUp   func(event.Event)
	OnMouseMove func(event.Event)
	OnClick     func(event.Event)
	OnDrag      func(event.Event)
	OnWheel     func(event.Event)
	OnFocus     func(event.Event)
	OnBlur      func(event.Event)
	OnChange    func(event.Event)
	OnScroll    func(event.Event)
}

// elementNode is the base implementation of Node for element VDOM nodes.
type elementNode[P any] struct {
	tagName     string
	props       P
	children    []Node
	instantiate func(doc dom.Document) dom.Node
	update      func(el dom.Node, old, new P)
}

func (n *elementNode[P]) TagName() string  { return n.tagName }
func (n *elementNode[P]) Props() any       { return n.props }
func (n *elementNode[P]) Children() []Node { return n.children }

func (n *elementNode[P]) Instantiate(doc dom.Document) dom.Node {
	el := n.instantiate(doc)
	n.update(el, *new(P), n.props)
	return el
}

func (n *elementNode[P]) Update(el dom.Node, old Node) {
	var oldProps P
	if old != nil {
		if oldEl, ok := old.(*elementNode[P]); ok {
			oldProps = oldEl.props
		}
	}
	n.update(el, oldProps, n.props)
}

// textNode is the implementation of Node for VDOM text leaf nodes.
type textNode struct {
	content string
}

func (t *textNode) TagName() string  { return "#text" }
func (t *textNode) Props() any       { return t.content }
func (t *textNode) Children() []Node { return nil }
func (t *textNode) Instantiate(doc dom.Document) dom.Node {
	return element.NewText(doc, t.content)
}

func (t *textNode) Update(el dom.Node, old Node) {
	txt, ok := el.(*element.TextElement)
	if !ok {
		return
	}
	var oldContent string
	if old != nil {
		if s, ok := old.Props().(string); ok {
			oldContent = s
		}
	}
	if oldContent != t.content {
		txt.SetData(t.content)
	}
}

// Text creates a VDOM representation of a text node.
func Text(data string) Node {
	return &textNode{content: data}
}

// --- Event Subscription Registry ----------------------------------------------

var (
	subMutex      sync.Mutex
	subscriptions = make(map[dom.Node]map[event.EventType]event.Subscription)
)

func setSubscription(node dom.Node, typ event.EventType, sub event.Subscription) {
	subMutex.Lock()
	defer subMutex.Unlock()
	m, ok := subscriptions[node]
	if !ok {
		m = make(map[event.EventType]event.Subscription)
		subscriptions[node] = m
	}
	if oldSub, ok := m[typ]; ok {
		oldSub.Cancel()
	}
	m[typ] = sub
}

func clearSubscription(node dom.Node, typ event.EventType) {
	subMutex.Lock()
	defer subMutex.Unlock()
	if m, ok := subscriptions[node]; ok {
		if oldSub, ok := m[typ]; ok {
			oldSub.Cancel()
			delete(m, typ)
		}
	}
}

// ClearAllSubscriptions cancels all event subscriptions associated with a DOM node
// and removes the node from the tracking map. Reconcilers should invoke this when deleting a node.
func ClearAllSubscriptions(node dom.Node) {
	subMutex.Lock()
	defer subMutex.Unlock()
	if m, ok := subscriptions[node]; ok {
		for _, sub := range m {
			sub.Cancel()
		}
		delete(subscriptions, node)
	}
}

func funcEquals(f1, f2 any) bool {
	n1 := isNilFunc(f1)
	n2 := isNilFunc(f2)
	if n1 || n2 {
		return n1 == n2
	}
	v1 := reflect.ValueOf(f1)
	v2 := reflect.ValueOf(f2)
	if v1.Kind() != reflect.Func || v2.Kind() != reflect.Func {
		return false
	}
	return v1.Pointer() == v2.Pointer()
}

func isNilFunc(f any) bool {
	if f == nil {
		return true
	}
	v := reflect.ValueOf(f)
	return v.Kind() == reflect.Func && v.IsNil()
}

func updateListener(el element.Element, typ event.EventType, oldFn, newFn any) {
	if funcEquals(oldFn, newFn) {
		return
	}
	clearSubscription(el, typ)
	if !isNilFunc(newFn) {
		if fn, ok := newFn.(func(event.Event)); ok {
			sub := el.AddEventListener(typ, fn)
			setSubscription(el, typ, sub)
		}
	}
}

// setStyle invokes the Style method of an element using a type switch to avoid reflection.
func setStyle(el element.Element, s style.Style) {
	switch x := el.(type) {
	case *element.BoxElement:
		x.Style(s)
	case *element.SpanElement:
		x.Style(s)
	case *element.ButtonElement:
		x.Style(s)
	case *element.CheckboxElement:
		x.Style(s)
	case *element.RadioGroupElement:
		x.Style(s)
	case *element.RadioElement:
		x.Style(s)
	case *element.SelectElement:
		x.Style(s)
	case *element.OptionElement:
		x.Style(s)
	case *element.InputElement:
		x.Style(s)
	case *element.TextAreaElement:
		x.Style(s)
	case *element.TableElement:
		x.Style(s)
	case *element.TableHeaderElement:
		x.Style(s)
	case *element.TableBodyElement:
		x.Style(s)
	case *element.TableFooterElement:
		x.Style(s)
	case *element.TableRowElement:
		x.Style(s)
	case *element.TableCellElement:
		x.Style(s)
	case *element.BrElement:
		x.Style(s)
	case *element.OverlayElement:
		x.Style(s)
	case *element.DialogElement:
		x.Style(s)
	}
}

// setHidden invokes the Hidden method of an element using a type switch to avoid reflection.
func setHidden(el element.Element, h bool) {
	switch x := el.(type) {
	case *element.BoxElement:
		x.Hidden(h)
	case *element.SpanElement:
		x.Hidden(h)
	case *element.ButtonElement:
		x.Hidden(h)
	case *element.CheckboxElement:
		x.Hidden(h)
	case *element.RadioGroupElement:
		x.Hidden(h)
	case *element.RadioElement:
		x.Hidden(h)
	case *element.SelectElement:
		x.Hidden(h)
	case *element.OptionElement:
		x.Hidden(h)
	case *element.InputElement:
		x.Hidden(h)
	case *element.TextAreaElement:
		x.Hidden(h)
	case *element.TableElement:
		x.Hidden(h)
	case *element.TableHeaderElement:
		x.Hidden(h)
	case *element.TableBodyElement:
		x.Hidden(h)
	case *element.TableFooterElement:
		x.Hidden(h)
	case *element.TableRowElement:
		x.Hidden(h)
	case *element.TableCellElement:
		x.Hidden(h)
	case *element.BrElement:
		x.Hidden(h)
	case *element.OverlayElement:
		x.Hidden(h)
	case *element.DialogElement:
		x.Hidden(h)
	}
}

// updateElementBase syncs core style, identity and listeners on any element.
func updateElementBase(el element.Element, old, new ElementProps) {
	if old.ID != new.ID {
		el.SetID(new.ID)
	}
	if old.Class != new.Class {
		el.SetClass(new.Class)
	}
	if old.Hidden != new.Hidden {
		setHidden(el, new.Hidden)
	}
	setStyle(el, new.Style)

	// Listeners
	updateListener(el, event.EventKeyDown, old.OnKeyDown, new.OnKeyDown)
	updateListener(el, event.EventKeyUp, old.OnKeyUp, new.OnKeyUp)
	updateListener(el, event.EventKeyPress, old.OnKeyPress, new.OnKeyPress)
	updateListener(el, event.EventMouseDown, old.OnMouseDown, new.OnMouseDown)
	updateListener(el, event.EventMouseUp, old.OnMouseUp, new.OnMouseUp)
	updateListener(el, event.EventMouseMove, old.OnMouseMove, new.OnMouseMove)
	updateListener(el, event.EventClick, old.OnClick, new.OnClick)
	updateListener(el, event.EventDrag, old.OnDrag, new.OnDrag)
	updateListener(el, event.EventWheel, old.OnWheel, new.OnWheel)
	updateListener(el, event.EventFocus, old.OnFocus, new.OnFocus)
	updateListener(el, event.EventBlur, old.OnBlur, new.OnBlur)
	updateListener(el, event.EventChange, old.OnChange, new.OnChange)
	updateListener(el, event.EventScroll, old.OnScroll, new.OnScroll)
}

// --- VDOM Factories -----------------------------------------------------------

// BoxProps specifies attributes for Box elements.
type BoxProps struct {
	ElementProps
}

func boxInstantiate(doc dom.Document) dom.Node {
	return element.NewBox(doc)
}

func boxUpdate(el dom.Node, old, new BoxProps) {
	updateElementBase(el.(element.Element), old.ElementProps, new.ElementProps)
}

// Box creates a VDOM representation of a BoxElement container.
func Box(props BoxProps, children ...Node) Node {
	return &elementNode[BoxProps]{
		tagName:     "box",
		props:       props,
		children:    children,
		instantiate: boxInstantiate,
		update:      boxUpdate,
	}
}

// Div creates a VDOM representation of a BoxElement container (web div alias).
func Div(props BoxProps, children ...Node) Node {
	node := Box(props, children...)
	node.(*elementNode[BoxProps]).tagName = "div"
	return node
}

// SpanProps specifies attributes for Span elements.
type SpanProps struct {
	ElementProps
}

func spanInstantiate(doc dom.Document) dom.Node {
	return element.NewSpan(doc)
}

func spanUpdate(el dom.Node, old, new SpanProps) {
	updateElementBase(el.(element.Element), old.ElementProps, new.ElementProps)
}

// Span creates a VDOM representation of a SpanElement container.
func Span(props SpanProps, children ...Node) Node {
	return &elementNode[SpanProps]{
		tagName:     "span",
		props:       props,
		children:    children,
		instantiate: spanInstantiate,
		update:      spanUpdate,
	}
}

// ButtonProps specifies attributes for Button elements.
type ButtonProps struct {
	ElementProps
	Disabled bool
	Active   bool
}

func buttonInstantiate(doc dom.Document) dom.Node {
	return element.NewButton(doc)
}

func buttonUpdate(el dom.Node, old, new ButtonProps) {
	btn := el.(*element.ButtonElement)
	updateElementBase(btn, old.ElementProps, new.ElementProps)
	if old.Disabled != new.Disabled {
		btn.Disabled(new.Disabled)
	}
	if old.Active != new.Active {
		btn.SetActive(new.Active)
	}
}

// Button creates a VDOM representation of a ButtonElement.
func Button(props ButtonProps, children ...Node) Node {
	return &elementNode[ButtonProps]{
		tagName:     "button",
		props:       props,
		children:    children,
		instantiate: buttonInstantiate,
		update:      buttonUpdate,
	}
}

// CheckboxProps specifies attributes for Checkbox elements.
type CheckboxProps struct {
	ElementProps
	Checked        bool
	UncheckedGlyph string
	CheckedGlyph   string
}

func checkboxInstantiate(doc dom.Document) dom.Node {
	return element.NewCheckbox(doc, false)
}

func checkboxUpdate(el dom.Node, old, new CheckboxProps) {
	cb := el.(*element.CheckboxElement)
	updateElementBase(cb, old.ElementProps, new.ElementProps)
	if old.Checked != new.Checked {
		cb.SetChecked(new.Checked)
	}
	if old.UncheckedGlyph != new.UncheckedGlyph || old.CheckedGlyph != new.CheckedGlyph {
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
}

// Checkbox creates a VDOM representation of a CheckboxElement.
func Checkbox(props CheckboxProps) Node {
	return &elementNode[CheckboxProps]{
		tagName:     "checkbox",
		props:       props,
		children:    nil,
		instantiate: checkboxInstantiate,
		update:      checkboxUpdate,
	}
}

// RadioGroupProps specifies attributes for RadioGroup elements.
type RadioGroupProps struct {
	ElementProps
	Value         string
	OnValueChange func(string)
}

func radioGroupInstantiate(doc dom.Document) dom.Node {
	return element.NewRadioGroup(doc)
}

func radioGroupUpdate(el dom.Node, old, new RadioGroupProps) {
	rg := el.(*element.RadioGroupElement)
	updateElementBase(rg, old.ElementProps, new.ElementProps)
	if old.Value != new.Value {
		rg.SetValue(new.Value)
	}
	if !funcEquals(old.OnValueChange, new.OnValueChange) {
		rg.OnChange(new.OnValueChange)
	}
}

// RadioGroup creates a VDOM representation of a RadioGroupElement.
func RadioGroup(props RadioGroupProps, children ...Node) Node {
	return &elementNode[RadioGroupProps]{
		tagName:     "radiogroup",
		props:       props,
		children:    children,
		instantiate: radioGroupInstantiate,
		update:      radioGroupUpdate,
	}
}

// RadioProps specifies attributes for Radio elements.
type RadioProps struct {
	ElementProps
	Value          string
	UncheckedGlyph string
	CheckedGlyph   string
}

func radioInstantiate(doc dom.Document) dom.Node {
	return element.NewRadio(doc, "")
}

func radioUpdate(el dom.Node, old, new RadioProps) {
	r := el.(*element.RadioElement)
	updateElementBase(r, old.ElementProps, new.ElementProps)
	if old.Value != new.Value && new.Value != "" {
		r.SetValue(new.Value)
	}
	if old.UncheckedGlyph != new.UncheckedGlyph || old.CheckedGlyph != new.CheckedGlyph {
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
}

// Radio creates a VDOM representation of a RadioElement.
func Radio(props RadioProps) Node {
	return &elementNode[RadioProps]{
		tagName:     "radio",
		props:       props,
		children:    nil,
		instantiate: radioInstantiate,
		update:      radioUpdate,
	}
}

// SelectProps specifies attributes for Select elements.
type SelectProps struct {
	ElementProps
	Value         string
	OnValueChange func(string)
}

// Select creates a VDOM representation of a SelectElement.
func Select(props SelectProps, children ...Node) Node {
	return &elementNode[SelectProps]{
		tagName:  "select",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewSelect(doc)
		},
		update: func(el dom.Node, old, new SelectProps) {
			s := el.(*element.SelectElement)
			updateElementBase(s, old.ElementProps, new.ElementProps)
			if old.Value != new.Value {
				s.SetValue(new.Value)
			}
			if !funcEquals(old.OnValueChange, new.OnValueChange) {
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
		},
	}
}

// OptionProps specifies attributes for Option elements.
type OptionProps struct {
	ElementProps
	Text  string
	Value string
}

// Option creates a VDOM representation of an OptionElement metadata node.
func Option(props OptionProps) Node {
	return &elementNode[OptionProps]{
		tagName:  "option",
		props:    props,
		children: nil,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewOption(doc, props.Text, props.Value)
		},
		update: func(el dom.Node, old, new OptionProps) {
			opt := el.(*element.OptionElement)
			updateElementBase(opt, old.ElementProps, new.ElementProps)
			if old.Text != new.Text {
				opt.SetText(new.Text)
			}
			if old.Value != new.Value {
				opt.SetValue(new.Value)
			}
		},
	}
}

// InputProps specifies attributes for Input elements.
type InputProps struct {
	ElementProps
	Value string
}

// Input creates a VDOM representation of an InputElement.
func Input(props InputProps) Node {
	return &elementNode[InputProps]{
		tagName:  "input",
		props:    props,
		children: nil,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewInput(doc, props.Value)
		},
		update: func(el dom.Node, old, new InputProps) {
			inp := el.(*element.InputElement)
			updateElementBase(inp, old.ElementProps, new.ElementProps)
			if old.Value != new.Value {
				inp.SetValue(new.Value)
			}
		},
	}
}

// TextAreaProps specifies attributes for TextArea elements.
type TextAreaProps struct {
	ElementProps
	Value string
}

// TextArea creates a VDOM representation of a TextAreaElement.
func TextArea(props TextAreaProps) Node {
	return &elementNode[TextAreaProps]{
		tagName:  "textarea",
		props:    props,
		children: nil,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTextArea(doc, props.Value)
		},
		update: func(el dom.Node, old, new TextAreaProps) {
			txa := el.(*element.TextAreaElement)
			updateElementBase(txa, old.ElementProps, new.ElementProps)
			if old.Value != new.Value {
				txa.SetValue(new.Value)
			}
		},
	}
}

// TableProps specifies attributes for Table elements.
type TableProps struct {
	ElementProps
}

// Table creates a VDOM representation of a TableElement.
func Table(props TableProps, children ...Node) Node {
	return &elementNode[TableProps]{
		tagName:  "table",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTable(doc)
		},
		update: func(el dom.Node, old, new TableProps) {
			updateElementBase(el.(element.Element), old.ElementProps, new.ElementProps)
		},
	}
}

// THeadProps specifies attributes for THead elements.
type THeadProps struct {
	ElementProps
}

// THead creates a VDOM representation of a TableHeaderElement (thead).
func THead(props THeadProps, children ...Node) Node {
	return &elementNode[THeadProps]{
		tagName:  "thead",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTableHeader(doc)
		},
		update: func(el dom.Node, old, new THeadProps) {
			updateElementBase(el.(element.Element), old.ElementProps, new.ElementProps)
		},
	}
}

// TBodyProps specifies attributes for TBody elements.
type TBodyProps struct {
	ElementProps
}

// TBody creates a VDOM representation of a TableBodyElement (tbody).
func TBody(props TBodyProps, children ...Node) Node {
	return &elementNode[TBodyProps]{
		tagName:  "tbody",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTableBody(doc)
		},
		update: func(el dom.Node, old, new TBodyProps) {
			updateElementBase(el.(element.Element), old.ElementProps, new.ElementProps)
		},
	}
}

// TFootProps specifies attributes for TFoot elements.
type TFootProps struct {
	ElementProps
}

// TFoot creates a VDOM representation of a TableFooterElement (tfoot).
func TFoot(props TFootProps, children ...Node) Node {
	return &elementNode[TFootProps]{
		tagName:  "tfoot",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTableFooter(doc)
		},
		update: func(el dom.Node, old, new TFootProps) {
			updateElementBase(el.(element.Element), old.ElementProps, new.ElementProps)
		},
	}
}

// TRProps specifies attributes for TR elements.
type TRProps struct {
	ElementProps
}

// TR creates a VDOM representation of a TableRowElement (tr).
func TR(props TRProps, children ...Node) Node {
	return &elementNode[TRProps]{
		tagName:  "tr",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTableRow(doc)
		},
		update: func(el dom.Node, old, new TRProps) {
			updateElementBase(el.(element.Element), old.ElementProps, new.ElementProps)
		},
	}
}

// TDProps specifies attributes for TD elements.
type TDProps struct {
	ElementProps
	ColSpan int
	RowSpan int
}

// TD creates a VDOM representation of a TableCellElement (td).
func TD(props TDProps, children ...Node) Node {
	return &elementNode[TDProps]{
		tagName:  "td",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTableCell(doc)
		},
		update: func(el dom.Node, old, new TDProps) {
			td := el.(*element.TableCellElement)
			updateElementBase(td, old.ElementProps, new.ElementProps)
			if old.ColSpan != new.ColSpan {
				td.SetColSpan(new.ColSpan)
			}
			if old.RowSpan != new.RowSpan {
				td.SetRowSpan(new.RowSpan)
			}
		},
	}
}

// BrProps specifies attributes for Br elements.
type BrProps struct {
	ElementProps
}

// Br creates a VDOM representation of a BrElement.
func Br(props BrProps) Node {
	return &elementNode[BrProps]{
		tagName:  "br",
		props:    props,
		children: nil,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewBr(doc)
		},
		update: func(el dom.Node, old, new BrProps) {
			updateElementBase(el.(element.Element), old.ElementProps, new.ElementProps)
		},
	}
}

// OverlayProps specifies attributes for Overlay elements.
type OverlayProps struct {
	ElementProps
	Anchor    dom.Element
	ZIndex    int
	Placement layout.OverlayPlacement
	Flip      bool
}

// Overlay creates a VDOM representation of an OverlayElement.
func Overlay(props OverlayProps, content Node) Node {
	var children []Node
	if content != nil {
		children = []Node{content}
	}
	return &elementNode[OverlayProps]{
		tagName:  "overlay",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			config := element.OverlayConfig{
				Anchor:    props.Anchor,
				ZIndex:    props.ZIndex,
				Placement: props.Placement,
				Flip:      props.Flip,
			}
			return element.NewOverlay(doc, nil, config)
		},
		update: func(el dom.Node, old, new OverlayProps) {
			o := el.(*element.OverlayElement)
			updateElementBase(o, old.ElementProps, new.ElementProps)
			if old.Anchor != new.Anchor || old.ZIndex != new.ZIndex || old.Placement != new.Placement || old.Flip != new.Flip {
				config := element.OverlayConfig{
					Anchor:    new.Anchor,
					ZIndex:    new.ZIndex,
					Placement: new.Placement,
					Flip:      new.Flip,
				}
				o.SetConfig(config)
			}
		},
	}
}

// DialogProps specifies attributes for Dialog elements.
type DialogProps struct {
	ElementProps
	ZIndex int
}

// Dialog creates a VDOM representation of a DialogElement.
func Dialog(props DialogProps, content Node) Node {
	var children []Node
	if content != nil {
		children = []Node{content}
	}
	return &elementNode[DialogProps]{
		tagName:  "dialog",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewDialog(doc, nil, props.ZIndex)
		},
		update: func(el dom.Node, old, new DialogProps) {
			d := el.(*element.DialogElement)
			updateElementBase(d, old.ElementProps, new.ElementProps)
			if old.ZIndex != new.ZIndex {
				d.SetZIndex(new.ZIndex)
			}
		},
	}
}

// --- Functional Components & Hooks --------------------------------------------

// componentRef holds a stable pointer to the active ComponentNode.
// This is used by hooks to always target the active component instance
// without needing to re-allocate getters/setters on every render cycle.
type componentRef struct {
	mu   sync.Mutex
	node componentInstance
}

type hookState[T any] struct {
	value T
	get   func() T
	set   func(T)
}

// componentInstance is the internal interface implemented by all ComponentNode instances
// to manage hooks state and updates.
type componentInstance interface {
	getHookState(index int) (any, bool)
	setHookState(index int, val any)
	incrementHookIndex() int
	MarkDirty()
	IsDirty() bool
	ClearDirty()
	getRef() *componentRef
}

// ComponentNode represents a declarative functional component in the VDOM tree.
// It implements the Node interface.
type ComponentNode[P any] struct {
	Name     string
	PropsVal P
	RenderFn func(P) Node

	// Internal state
	rendered  Node
	realNode  dom.Node
	hooks     []any
	hookIndex int
	isFirst   bool
	dirty     bool
	ref       *componentRef
}

var _ Node = (*ComponentNode[struct{}])(nil)

func (c *ComponentNode[P]) TagName() string { return c.Name }
func (c *ComponentNode[P]) Props() any      { return c.PropsVal }

func (c *ComponentNode[P]) Children() []Node {
	if c.rendered == nil {
		return nil
	}
	return []Node{c.rendered}
}

func (c *ComponentNode[P]) Instantiate(doc dom.Document) dom.Node {
	c.ref = &componentRef{node: c}
	pushCurrentComponent(c)
	c.isFirst = true
	c.hookIndex = 0
	c.rendered = c.RenderFn(c.PropsVal)
	c.isFirst = false
	popCurrentComponent()

	if c.rendered != nil {
		c.realNode = c.rendered.Instantiate(doc)
	}
	return c.realNode
}

func (c *ComponentNode[P]) Update(el dom.Node, old Node) {
	oldComp, ok := old.(*ComponentNode[P])
	if !ok {
		return
	}
	// Transfer state
	c.hooks = oldComp.hooks
	c.isFirst = false
	c.realNode = oldComp.realNode
	c.ref = oldComp.ref
	if c.ref == nil {
		c.ref = &componentRef{node: c}
		oldComp.ref = c.ref
	} else {
		c.ref.mu.Lock()
		c.ref.node = c
		c.ref.mu.Unlock()
	}

	pushCurrentComponent(c)
	c.hookIndex = 0
	c.rendered = c.RenderFn(c.PropsVal)
	popCurrentComponent()

	if c.rendered != nil && oldComp.rendered != nil {
		if c.rendered.TagName() == oldComp.rendered.TagName() {
			c.rendered.Update(c.realNode, oldComp.rendered)
		}
	}
}

func (c *ComponentNode[P]) getHookState(index int) (any, bool) {
	if index < len(c.hooks) {
		return c.hooks[index], true
	}
	return nil, false
}

func (c *ComponentNode[P]) setHookState(index int, val any) {
	if index >= len(c.hooks) {
		newHooks := make([]any, index+1)
		copy(newHooks, c.hooks)
		c.hooks = newHooks
	}
	c.hooks[index] = val
}

func (c *ComponentNode[P]) incrementHookIndex() int {
	idx := c.hookIndex
	c.hookIndex++
	return idx
}

func (c *ComponentNode[P]) getRef() *componentRef {
	return c.ref
}

// IsDirty returns whether the component is dirty.
func (c *ComponentNode[P]) IsDirty() bool {
	return c.dirty
}

// ClearDirty clears the dirty flag of the component.
func (c *ComponentNode[P]) ClearDirty() {
	c.dirty = false
}

// MarkDirty flags the component as dirty.
func (c *ComponentNode[P]) MarkDirty() {
	c.dirty = true
	if OnComponentDirty != nil {
		OnComponentDirty(c)
	}
}

// OnComponentDirty is a package-level hook triggered when a component becomes dirty (e.g. state change).
var OnComponentDirty func(node Node)

// --- Execution Stack ----------------------------------------------------------

var (
	renderStackMutex sync.Mutex
	renderStack      []any
)

func pushCurrentComponent(c any) {
	renderStackMutex.Lock()
	defer renderStackMutex.Unlock()
	renderStack = append(renderStack, c)
}

func popCurrentComponent() {
	renderStackMutex.Lock()
	defer renderStackMutex.Unlock()
	if len(renderStack) > 0 {
		renderStack = renderStack[:len(renderStack)-1]
	}
}

func getCurrentComponent() any {
	renderStackMutex.Lock()
	defer renderStackMutex.Unlock()
	if len(renderStack) == 0 {
		return nil
	}
	return renderStack[len(renderStack)-1]
}

// --- FC & FCC wrappers ---------------------------------------------------------

// FC creates a functional component wrapper that does not take children.
func FC[P any](name string, render func(P) Node) func(P) Node {
	return func(props P) Node {
		return &ComponentNode[P]{
			Name:     name,
			PropsVal: props,
			RenderFn: render,
			isFirst:  true,
		}
	}
}

// FCC creates a functional component wrapper that can accept children.
// If the Props type has a Children field of type []Node, the variadic children
// will be injected using reflection.
func FCC[P any](name string, render func(P) Node) func(P, ...Node) Node {
	return func(props P, children ...Node) Node {
		propsWithChildren := injectChildren(props, children)
		return &ComponentNode[P]{
			Name:     name,
			PropsVal: propsWithChildren,
			RenderFn: render,
			isFirst:  true,
		}
	}
}

func injectChildren[P any](props P, children []Node) P {
	v := reflect.ValueOf(props)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return props
		}
		elem := v.Elem()
		if elem.Kind() == reflect.Struct {
			f := elem.FieldByName("Children")
			if f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf([]Node{}) {
				f.Set(reflect.ValueOf(children))
			}
		}
		return props
	}

	if v.Kind() == reflect.Struct {
		ptr := reflect.New(v.Type())
		ptr.Elem().Set(v)
		f := ptr.Elem().FieldByName("Children")
		if f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf([]Node{}) {
			f.Set(reflect.ValueOf(children))
		}
		return ptr.Elem().Interface().(P)
	}

	return props
}

// --- Hook primitives ----------------------------------------------------------

// UseState initializes a state variable on first render, persists it across render cycles,
// and returns a getter and a setter. Setting the state flags the component dirty.
// If called outside of a component render cycle, it panics.
func UseState[T any](initial T) (func() T, func(T)) {
	compVal := getCurrentComponent()
	if compVal == nil {
		panic("UseState must be called inside a functional component render phase")
	}
	comp := compVal.(componentInstance)
	idx := comp.incrementHookIndex()

	stateVal, exists := comp.getHookState(idx)
	if !exists {
		ref := comp.getRef()
		if ref == nil {
			panic("ComponentNode ref is not initialized. Ensure component is rendered via Instantiate/Update.")
		}

		hs := &hookState[T]{
			value: initial,
		}

		get := func() T {
			ref.mu.Lock()
			activeNode := ref.node
			ref.mu.Unlock()
			val, _ := activeNode.getHookState(idx)
			return val.(*hookState[T]).value
		}

		set := func(newVal T) {
			ref.mu.Lock()
			activeNode := ref.node
			ref.mu.Unlock()

			val, _ := activeNode.getHookState(idx)
			hsPtr := val.(*hookState[T])
			hsPtr.value = newVal
			activeNode.MarkDirty()
		}

		hs.get = get
		hs.set = set
		comp.setHookState(idx, hs)
		return get, set
	}

	hs := stateVal.(*hookState[T])
	return hs.get, hs.set
}
