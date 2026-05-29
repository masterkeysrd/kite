package kitex

import (
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

var EnableDevMode bool

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

	// Key returns the unique key of this node, used for list reconciliation.
	Key() string

	// Release returns the node (and its children) to the object pool.
	// This should only be called by the framework when a virtual tree is discarded.
	Release()
}

// nodeInternal is an unexported interface for internal framework access to the real DOM node.
type nodeInternal interface {
	Node
	realNode() dom.Node
	// complexity returns the approximate node-count of the subtree rooted at this
	// node (including itself). It is computed bottom-up at construction time in
	// O(N) and is used by ComponentNode to decide whether memoization is worth
	// the overhead of a reflection-based props comparison.
	complexity() int
	containsProvider() bool
	isProvider() bool
	hasDirectProvider() bool
}

// Ensure compile-time interface compliance.
var (
	_ Node         = (*textNode)(nil)
	_ nodeInternal = (*textNode)(nil)
	_ Node         = (*elementNode[struct{}])(nil)
	_ nodeInternal = (*elementNode[struct{}])(nil)
	_ nodeInternal = (*ComponentNode[struct{}])(nil)
)

// ElementProps holds common fields and event listeners present on all DOM element nodes.
type ElementProps struct {
	Key         string
	ID          string
	Class       string
	Style       style.Style
	Hidden      bool
	Disabled    bool
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
	Ref         refSetter
}

// elementNode is the base implementation of Node for element VDOM nodes.
type elementNode[P any] struct {
	tagName     string
	props       P
	children    []Node
	instantiate func(doc dom.Document) dom.Node
	update      func(el dom.Node, old, new *P)
	key         string
	ref         dom.Node

	// score is the pre-computed node count of the subtree rooted at this element
	// (1 for self + sum of children's complexity). Set once at construction time.
	score int

	// hasProvider is true if any node in the subtree rooted at this element
	// is a context provider.
	hasProvider bool

	// hasDirectP is true if any direct child is a context provider.
	hasDirectP bool

	declFile string
	declLine int
	instFile string
	instLine int
}

func (n *elementNode[P]) TagName() string         { return n.tagName }
func (n *elementNode[P]) Props() any              { return n.props }
func (n *elementNode[P]) Children() []Node        { return n.children }
func (n *elementNode[P]) Key() string             { return n.key }
func (n *elementNode[P]) realNode() dom.Node      { return n.ref }
func (n *elementNode[P]) complexity() int         { return n.score }
func (n *elementNode[P]) containsProvider() bool  { return n.hasProvider }
func (n *elementNode[P]) isProvider() bool        { return false }
func (n *elementNode[P]) hasDirectProvider() bool { return n.hasDirectP }

func (n *elementNode[P]) Release() {
	if n.tagName == "" {
		return
	}
	n.tagName = ""
	n.children = nil
	n.ref = nil
	n.instantiate = nil
	n.update = nil

	if p, ok := any(n).(*elementNode[ElementProps]); ok {
		elementNodePool.Put(p)
	}
}

func (n *elementNode[P]) Instantiate(doc dom.Document) dom.Node {
	n.ref = n.instantiate(doc)
	n.update(n.ref, nil, &n.props)
	if n.hasDirectP {
		flatChildren := flattenNodes(n.children, nil, nil)
		for _, childFlat := range flatChildren {
			childReal := instantiateFlat(n.ref.(dom.Element), childFlat)
			if childReal != nil {
				n.ref.AppendChild(childReal)
			}
		}
	} else {
		for _, child := range n.children {
			if child != nil {
				childReal := child.Instantiate(doc)
				if childReal != nil {
					n.ref.AppendChild(childReal)
				}
			}
		}
	}
	if setter := getRefSetter(&n.props); setter != nil {
		setter.set(n.ref)
	}
	return n.ref
}

func (n *elementNode[P]) Update(el dom.Node, old Node) {
	n.ref = el
	var oldProps *P
	if old != nil {
		if oldEl, ok := old.(*elementNode[P]); ok {
			oldProps = &oldEl.props
		}
	}
	n.update(n.ref, oldProps, &n.props)
	if setter := getRefSetter(&n.props); setter != nil {
		setter.set(n.ref)
	}
}

// textNode is the implementation of Node for VDOM text leaf nodes.
type textNode struct {
	content string
	ref     dom.Node

	declFile string
	declLine int
	instFile string
	instLine int
}

// complexity for a text leaf is always 1.
func (t *textNode) complexity() int { return 1 }

func (t *textNode) TagName() string         { return "#text" }
func (t *textNode) Props() any              { return t.content }
func (t *textNode) Children() []Node        { return nil }
func (t *textNode) Key() string             { return "" }
func (t *textNode) realNode() dom.Node      { return t.ref }
func (t *textNode) containsProvider() bool  { return false }
func (t *textNode) isProvider() bool        { return false }
func (t *textNode) hasDirectProvider() bool { return false }

func (t *textNode) Release() {
	if t.content == "" && t.ref == nil {
		return
	}
	t.content = ""
	t.ref = nil
	textNodePool.Put(t)
}
func (t *textNode) Instantiate(doc dom.Document) dom.Node {
	t.ref = element.NewText(doc, t.content)
	return t.ref
}

func (t *textNode) Update(el dom.Node, old Node) {
	t.ref = el
	txt, ok := t.ref.(*element.TextElement)
	if !ok {
		return
	}
	var oldContent string
	if old != nil {
		if oldTxt, ok := old.(*textNode); ok {
			oldContent = oldTxt.content
		}
	}
	if oldContent != t.content {
		txt.SetData(t.content)
	}
}

// Text creates a VDOM representation of a text node.
func Text(data string) Node {
	t := textNodePool.Get().(*textNode)
	t.content = data
	return trackSource(t, 1)
}

// --- Memoization Helpers ------------------------------------------------------

// memoComplexityThreshold is the minimum subtree node count required for
// ComponentNode to activate automatic memoization. Components whose rendered
// tree is cheap to re-render (score ≤ threshold) skip the reflection overhead.
const memoComplexityThreshold = 5

// computeComplexity returns the complexity of a Node, dispatching through the
// nodeInternal interface when available. Falls back to 1 for ComponentNodes
// (which wrap their rendered complexity in complexityScore).
func computeComplexity(n Node) int {
	if n == nil {
		return 0
	}
	if ni, ok := n.(nodeInternal); ok {
		return ni.complexity()
	}
	return 1
}

// buildElementInfo computes the score and provider-presence for an elementNode
// from its children slice and is called by every element factory.
func buildElementInfo(children []Node) (int, bool, bool) {
	s := 1
	hasP := false
	hasDirectP := false
	for _, c := range children {
		if c != nil {
			s += computeComplexity(c)
			if ni, ok := c.(nodeInternal); ok {
				isP := ni.isProvider()
				if isP {
					hasDirectP = true
					hasP = true
				} else if !hasP && ni.containsProvider() {
					hasP = true
				}
			}
		}
	}
	return s, hasP, hasDirectP
}

// deepEqualProps performs a depth-limited recursive equality check between two
// arbitrary prop values using reflection. It returns true only when the values
// are provably equal at every level up to maxDepth.
//
// Special cases:
//   - reflect.Func values are compared via the existing funcEquals helper so
//     that stable closures with the same pointer compare as equal.
//   - When depth == 0 the function conservatively returns false to avoid
//     unbounded recursion on large nested structures.
func deepEqualProps(oldProps, newProps any, maxDepth int) bool {
	if maxDepth <= 0 {
		return false
	}
	if oldProps == nil && newProps == nil {
		return true
	}
	if oldProps == nil || newProps == nil {
		return false
	}

	ov := reflect.ValueOf(oldProps)
	nv := reflect.ValueOf(newProps)

	// Dereference pointers once.
	if ov.Kind() == reflect.Pointer {
		if ov.IsNil() {
			return nv.Kind() == reflect.Pointer && nv.IsNil()
		}
		if nv.Kind() != reflect.Pointer || nv.IsNil() {
			return false
		}
		ov = ov.Elem()
		nv = nv.Elem()
	}

	if ov.Type() != nv.Type() {
		return false
	}

	return deepEqualValues(ov, nv, maxDepth)
}

// deepEqualValues is the reflection-recursive core of deepEqualProps.
// depth tracks the maximum number of nested *composite* levels (struct-in-struct,
// slice-of-structs, etc.) that are recursed into. Scalar leaf comparisons
// (int, string, bool, float, etc.) never consume depth and always succeed or
// fail immediately without further recursion.
func deepEqualValues(ov, nv reflect.Value, depth int) bool {
	switch ov.Kind() {
	case reflect.Func:
		// Use the existing funcEquals helper which compares function pointers.
		return funcEquals(ov.Interface(), nv.Interface())

	case reflect.Struct:
		// Entering a struct costs one depth level.
		if depth <= 0 {
			return false
		}
		for i := range ov.NumField() {
			of := ov.Field(i)
			nf := nv.Field(i)
			// Skip unexported fields — reflection cannot read them.
			if !of.CanInterface() {
				continue
			}
			if !deepEqualValues(of, nf, depth-1) {
				return false
			}
		}
		return true

	case reflect.Slice:
		if depth <= 0 {
			return false
		}
		if ov.IsNil() != nv.IsNil() {
			return false
		}
		if ov.Len() != nv.Len() {
			return false
		}
		for i := range ov.Len() {
			if !deepEqualValues(ov.Index(i), nv.Index(i), depth-1) {
				return false
			}
		}
		return true

	case reflect.Array:
		if depth <= 0 {
			return false
		}
		for i := range ov.Len() {
			if !deepEqualValues(ov.Index(i), nv.Index(i), depth-1) {
				return false
			}
		}
		return true

	case reflect.Map:
		if depth <= 0 {
			return false
		}
		if ov.IsNil() != nv.IsNil() {
			return false
		}
		if ov.Len() != nv.Len() {
			return false
		}
		for _, k := range ov.MapKeys() {
			nvVal := nv.MapIndex(k)
			if !nvVal.IsValid() {
				return false
			}
			if !deepEqualValues(ov.MapIndex(k), nvVal, depth-1) {
				return false
			}
		}
		return true

	case reflect.Pointer, reflect.Interface:
		if depth <= 0 {
			return false
		}
		if ov.IsNil() {
			return nv.IsNil()
		}
		if nv.IsNil() {
			return false
		}
		return deepEqualValues(ov.Elem(), nv.Elem(), depth-1)

	default:
		// Scalar leaf (bool, int*, uint*, float*, complex*, string, chan, etc.).
		// Scalars compare directly — they do NOT consume depth because they
		// cannot cause unbounded recursion.
		if ov.CanInterface() {
			return ov.Interface() == nv.Interface()
		}
		return false
	}
}

// --- Event Subscription Registry ----------------------------------------------

var (
	subMutex      sync.Mutex
	subscriptions = make(map[dom.Node]map[event.EventType]event.Subscription)

	textNodePool = sync.Pool{
		New: func() any { return &textNode{} },
	}

	elementNodePool = sync.Pool{
		New: func() any { return &elementNode[ElementProps]{} },
	}

	simpleComponentPool = sync.Pool{
		New: func() any { return &ComponentNode[struct{}]{} },
	}

	simpleFCCComponentPool = sync.Pool{
		New: func() any { return &ComponentNode[[]Node]{} },
	}

	componentRefPool = sync.Pool{
		New: func() any { return &componentRef{} },
	}
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

	// Try type assertion for common listener type to avoid reflection
	if fn1, ok1 := f1.(func(event.Event)); ok1 {
		if fn2, ok2 := f2.(func(event.Event)); ok2 {
			// Functions are only equal if they are both nil or the same pointer.
			// Comparison of non-nil functions is not allowed in Go directly,
			// but we can use reflection pointer comparison.
			return reflect.ValueOf(fn1).Pointer() == reflect.ValueOf(fn2).Pointer()
		}
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
	if fn, ok := f.(func(event.Event)); ok {
		return fn == nil
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

// setDisabled invokes the Disabled method of an element using a type switch.
func setDisabled(el element.Element, d bool) {
	switch x := el.(type) {
	case *element.ButtonElement:
		x.Disabled(d)
	case *element.CheckboxElement:
		x.Disabled(d)
	case *element.RadioElement:
		x.Disabled(d)
	case *element.SelectElement:
		x.Disabled(d)
	case *element.OptionElement:
		x.Disabled(d)
	case *element.InputElement:
		x.Disabled(d)
	case *element.TextAreaElement:
		x.Disabled(d)
	}
}

var emptyElementProps ElementProps

// updateElementBase syncs core style, identity and listeners on any element.
func updateElementBase(el element.Element, old, new *ElementProps) {
	if old == nil {
		old = &emptyElementProps
	}
	if old.ID != new.ID {
		el.SetID(new.ID)
	}
	if old.Class != new.Class {
		el.SetClass(new.Class)
	}
	if old.Hidden != new.Hidden {
		setHidden(el, new.Hidden)
	}
	if old.Disabled != new.Disabled {
		setDisabled(el, new.Disabled)
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

type BoxProps = ElementProps
type SpanProps = ElementProps
type BrProps = ElementProps
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
func Box(props BoxProps, children ...Node) Node {
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
func Div(props BoxProps, children ...Node) Node {
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
func Span(props SpanProps, children ...Node) Node {
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

// ButtonProps specifies attributes for Button elements.
type ButtonProps struct {
	Key         string
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
	Ref         refSetter
	Disabled    bool
	Active      bool
	Type        string
}

func (p ButtonProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		Disabled:  p.Disabled,
		OnKeyDown: p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnScroll: p.OnScroll,
		Ref: p.Ref,
	}
}

func buttonInstantiate(doc dom.Document) dom.Node {
	return element.NewButton(doc)
}

func buttonUpdate(el dom.Node, old, new *ButtonProps) {
	btn := el.(*element.ButtonElement)
	var oldEp ElementProps
	if old != nil {
		oldEp = old.elementProps()
	}
	newEp := new.elementProps()
	updateElementBase(btn, &oldEp, &newEp)
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
func Button(props ButtonProps, children ...Node) Node {
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

// CheckboxProps specifies attributes for Checkbox elements.
type CheckboxProps struct {
	Key            string
	ID             string
	Class          string
	Style          style.Style
	Hidden         bool
	Disabled       bool
	OnKeyDown      func(event.Event)
	OnKeyUp        func(event.Event)
	OnKeyPress     func(event.Event)
	OnMouseDown    func(event.Event)
	OnMouseUp      func(event.Event)
	OnMouseMove    func(event.Event)
	OnClick        func(event.Event)
	OnDrag         func(event.Event)
	OnWheel        func(event.Event)
	OnFocus        func(event.Event)
	OnBlur         func(event.Event)
	OnChange       func(event.Event)
	OnScroll       func(event.Event)
	Ref            refSetter
	Checked        bool
	UncheckedGlyph string
	CheckedGlyph   string
	Name           string
}

func (p CheckboxProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		Disabled:  p.Disabled,
		OnKeyDown: p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnScroll: p.OnScroll,
		Ref: p.Ref,
	}
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
func Checkbox(props CheckboxProps) Node {
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

// RadioGroupProps specifies attributes for RadioGroup elements.
type RadioGroupProps struct {
	Key       string
	ID        string
	Class     string
	Style     style.Style
	Hidden    bool
	Disabled  bool
	OnKeyDown func(event.Event)

	OnKeyUp       func(event.Event)
	OnKeyPress    func(event.Event)
	OnMouseDown   func(event.Event)
	OnMouseUp     func(event.Event)
	OnMouseMove   func(event.Event)
	OnClick       func(event.Event)
	OnDrag        func(event.Event)
	OnWheel       func(event.Event)
	OnFocus       func(event.Event)
	OnBlur        func(event.Event)
	OnChange      func(event.Event)
	OnScroll      func(event.Event)
	Ref           refSetter
	Value         string
	OnValueChange func(string)
}

func (p RadioGroupProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		Disabled:  p.Disabled,
		OnKeyDown: p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnScroll: p.OnScroll,
		Ref: p.Ref,
	}
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
func RadioGroup(props RadioGroupProps, children ...Node) Node {
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

// RadioProps specifies attributes for Radio elements.
type RadioProps struct {
	Key            string
	ID             string
	Class          string
	Style          style.Style
	Hidden         bool
	OnKeyDown      func(event.Event)
	OnKeyUp        func(event.Event)
	OnKeyPress     func(event.Event)
	OnMouseDown    func(event.Event)
	OnMouseUp      func(event.Event)
	OnMouseMove    func(event.Event)
	OnClick        func(event.Event)
	OnDrag         func(event.Event)
	OnWheel        func(event.Event)
	OnFocus        func(event.Event)
	OnBlur         func(event.Event)
	OnChange       func(event.Event)
	OnScroll       func(event.Event)
	Ref            refSetter
	Value          string
	UncheckedGlyph string
	CheckedGlyph   string
	Name           string
}

func (p RadioProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		OnKeyDown: p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnScroll: p.OnScroll,
		Ref: p.Ref,
	}
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
func Radio(props RadioProps) Node {
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

// SelectProps specifies attributes for Select elements.
type SelectProps struct {
	Key       string
	ID        string
	Class     string
	Style     style.Style
	Hidden    bool
	Disabled  bool
	OnKeyDown func(event.Event)

	OnKeyUp       func(event.Event)
	OnKeyPress    func(event.Event)
	OnMouseDown   func(event.Event)
	OnMouseUp     func(event.Event)
	OnMouseMove   func(event.Event)
	OnClick       func(event.Event)
	OnDrag        func(event.Event)
	OnWheel       func(event.Event)
	OnFocus       func(event.Event)
	OnBlur        func(event.Event)
	OnChange      func(event.Event)
	OnScroll      func(event.Event)
	Ref           refSetter
	Value         string
	OnValueChange func(string)
	Name          string
}

func (p SelectProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		Disabled:  p.Disabled,
		OnKeyDown: p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnScroll: p.OnScroll,
		Ref: p.Ref,
	}
}

// Select creates a VDOM representation of a SelectElement.
func Select(props SelectProps, children ...Node) Node {
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

// OptionProps specifies attributes for Option elements.
type OptionProps struct {
	Key         string
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
	Ref         refSetter
	Text        string
	Value       string
}

func (p OptionProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		OnKeyDown: p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnScroll: p.OnScroll,
		Ref: p.Ref,
	}
}

// Option creates a VDOM representation of an OptionElement metadata node.
func Option(props OptionProps) Node {
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

// InputProps specifies attributes for Input elements.
type InputProps struct {
	Key         string
	ID          string
	Class       string
	Style       style.Style
	Hidden      bool
	Disabled    bool
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
	Ref         refSetter
	Value       string
	Name        string
}

func (p InputProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		Disabled:  p.Disabled,
		OnKeyDown: p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnScroll: p.OnScroll,
		Ref: p.Ref,
	}
}

// Input creates a VDOM representation of an InputElement.
func Input(props InputProps) Node {
	return trackSource(&elementNode[InputProps]{
		tagName:  "input",
		props:    props,
		children: nil,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewInput(doc, props.Value)
		},
		update: func(el dom.Node, old, new *InputProps) {
			inp := el.(*element.InputElement)
			var oldEp ElementProps
			if old != nil {
				oldEp = old.elementProps()
			}
			newEp := new.elementProps()
			updateElementBase(inp, &oldEp, &newEp)
			if old != nil && old.Value != new.Value {
				inp.SetValue(new.Value)
			} else if old == nil {
				inp.SetValue(new.Value)
			}
			if old != nil && old.Name != new.Name {
				inp.WithName(new.Name)
			} else if old == nil {
				inp.WithName(new.Name)
			}
		},
		key:         props.Key,
		score:       1,
		hasProvider: false,
		hasDirectP:  false,
	}, 1)
}

// TextAreaProps specifies attributes for TextArea elements.
type TextAreaProps struct {
	Key         string
	ID          string
	Class       string
	Style       style.Style
	Hidden      bool
	Disabled    bool
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
	Ref         refSetter
	Value       string
	Name        string
}

func (p TextAreaProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		Disabled:  p.Disabled,
		OnKeyDown: p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnScroll: p.OnScroll,
		Ref: p.Ref,
	}
}

// TextArea creates a VDOM representation of a TextAreaElement.
func TextArea(props TextAreaProps) Node {
	return trackSource(&elementNode[TextAreaProps]{
		tagName:  "textarea",
		props:    props,
		children: nil,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewTextArea(doc, props.Value)
		},
		update: func(el dom.Node, old, new *TextAreaProps) {
			txa := el.(*element.TextAreaElement)
			var oldEp ElementProps
			if old != nil {
				oldEp = old.elementProps()
			}
			newEp := new.elementProps()
			updateElementBase(txa, &oldEp, &newEp)
			if old != nil && old.Value != new.Value {
				txa.SetValue(new.Value)
			} else if old == nil {
				txa.SetValue(new.Value)
			}
			if old != nil && old.Name != new.Name {
				txa.WithName(new.Name)
			} else if old == nil {
				txa.WithName(new.Name)
			}
		},
		key:         props.Key,
		score:       1,
		hasProvider: false,
		hasDirectP:  false,
	}, 1)
}

// Table creates a VDOM representation of a TableElement.
func Table(props TableProps, children ...Node) Node {
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

// THead creates a VDOM representation of a TableHeaderElement (thead).
func THead(props THeadProps, children ...Node) Node {
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
func TBody(props TBodyProps, children ...Node) Node {
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
func TFoot(props TFootProps, children ...Node) Node {
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
func TR(props TRProps, children ...Node) Node {
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

// TDProps specifies attributes for TD elements.
type TDProps struct {
	Key         string
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
	Ref         refSetter
	ColSpan     int
	RowSpan     int
}

func (p TDProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		OnKeyDown: p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnScroll: p.OnScroll,
		Ref: p.Ref,
	}
}

// TD creates a VDOM representation of a TableCellElement (td).
func TD(props TDProps, children ...Node) Node {
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
func Br(props BrProps) Node {
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

// OverlayProps specifies attributes for Overlay elements.
type OverlayProps struct {
	Key         string
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
	Ref         refSetter
	Anchor      dom.Element
	ZIndex      int
	Placement   geom.Placement
	Flip        bool
}

func (p OverlayProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		OnKeyDown: p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnScroll: p.OnScroll,
		Ref: p.Ref,
	}
}

// Overlay creates a VDOM representation of an OverlayElement.
func Overlay(props OverlayProps, content Node) Node {
	var children []Node
	if content != nil {
		children = []Node{content}
	}
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[OverlayProps]{
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

// DialogProps specifies attributes for Dialog elements.
type DialogProps struct {
	Key         string
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
	Ref         refSetter
	ZIndex      int
}

func (p DialogProps) elementProps() ElementProps {
	return ElementProps{
		Key: p.Key, ID: p.ID, Class: p.Class, Style: p.Style, Hidden: p.Hidden,
		OnKeyDown: p.OnKeyDown, OnKeyUp: p.OnKeyUp, OnKeyPress: p.OnKeyPress,
		OnMouseDown: p.OnMouseDown, OnMouseUp: p.OnMouseUp, OnMouseMove: p.OnMouseMove,
		OnClick: p.OnClick, OnDrag: p.OnDrag, OnWheel: p.OnWheel,
		OnFocus: p.OnFocus, OnBlur: p.OnBlur, OnChange: p.OnChange, OnScroll: p.OnScroll,
		Ref: p.Ref,
	}
}

// Dialog creates a VDOM representation of a DialogElement.
func Dialog(props DialogProps, content Node) Node {
	var children []Node
	if content != nil {
		children = []Node{content}
	}
	score, hasP, hasDirectP := buildElementInfo(children)
	return trackSource(&elementNode[DialogProps]{
		tagName:  "dialog",
		props:    props,
		children: children,
		instantiate: func(doc dom.Document) dom.Node {
			return element.NewDialog(doc, nil, props.ZIndex)
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

// --- Functional Components & Hooks --------------------------------------------

// componentRef holds a stable pointer to the active ComponentNode.
// This is used by hooks to always target the active component instance
// without needing to re-allocate getters/setters on every render cycle.
type componentRef struct {
	mu   sync.Mutex
	node componentInstance
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
	realNode() dom.Node
	setRef(dom.Node)
	Rendered() Node
	ReRender() Node
	Destroy()
}

// ComponentNode represents a declarative functional component in the VDOM tree.
// It implements the Node interface.
type ComponentNode[P any] struct {
	Name     string
	PropsVal P
	RenderFn func(P) Node

	// Internal state
	rendered     Node
	ref          dom.Node
	hooks        []any
	hookIndex    int
	isFirst      bool
	dirty        bool
	componentRef *componentRef
	key          string

	// Memoization: complexityScore holds the node count of the last rendered
	// subtree. shouldMemo is true when complexityScore > memoComplexityThreshold,
	// indicating that a deep props comparison is cheaper than a full re-render.
	complexityScore int
	shouldMemo      bool

	declFile string
	declLine int
	instFile string
	instLine int

	pool *sync.Pool
}

type componentNodeInspector interface {
	getHooks() []any
	getRendered() Node
	getDecl() (string, int)
	getInst() (string, int)
}

type sourceTracker interface {
	getSource() (string, int, string, int)
}

type sourceTrackable interface {
	setSource(declFile string, declLine int, instFile string, instLine int)
}

func (c *ComponentNode[P]) getHooks() []any        { return c.hooks }
func (c *ComponentNode[P]) getRendered() Node      { return c.rendered }
func (c *ComponentNode[P]) getDecl() (string, int) { return c.declFile, c.declLine }
func (c *ComponentNode[P]) getInst() (string, int) { return c.instFile, c.instLine }

func (c *ComponentNode[P]) getSource() (string, int, string, int) {
	return c.declFile, c.declLine, c.instFile, c.instLine
}

func (c *ComponentNode[P]) setSource(declFile string, declLine int, instFile string, instLine int) {
	c.declFile = declFile
	c.declLine = declLine
	c.instFile = instFile
	c.instLine = instLine
}

func (n *elementNode[P]) getSource() (string, int, string, int) {
	return n.declFile, n.declLine, n.instFile, n.instLine
}

func (n *elementNode[P]) setSource(declFile string, declLine int, instFile string, instLine int) {
	n.declFile = declFile
	n.declLine = declLine
	n.instFile = instFile
	n.instLine = instLine
}

func (t *textNode) getSource() (string, int, string, int) {
	return t.declFile, t.declLine, t.instFile, t.instLine
}

func (t *textNode) setSource(declFile string, declLine int, instFile string, instLine int) {
	t.declFile = declFile
	t.declLine = declLine
	t.instFile = instFile
	t.instLine = instLine
}

func trackSource(node Node, skip int) Node {
	if EnableDevMode {
		if st, ok := node.(sourceTrackable); ok {
			var declFile string
			var declLine int
			if _, file, line, ok := runtime.Caller(skip); ok {
				declFile = file
				declLine = line
			}
			var instFile string
			var instLine int
			for i := skip + 1; ; i++ {
				_, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				if !strings.Contains(file, "extras/kitex") {
					instFile = file
					instLine = line
					break
				}
			}
			st.setSource(declFile, declLine, instFile, instLine)
		}
	}
	return node
}

var _ Node = (*ComponentNode[struct{}])(nil)

func (c *ComponentNode[P]) TagName() string { return c.Name }
func (c *ComponentNode[P]) Props() any      { return c.PropsVal }
func (c *ComponentNode[P]) Key() string     { return c.key }
func (c *ComponentNode[P]) containsProvider() bool {
	if c.rendered != nil {
		if ni, ok := c.rendered.(nodeInternal); ok {
			return ni.containsProvider()
		}
	}
	return false
}
func (c *ComponentNode[P]) isProvider() bool        { return false }
func (c *ComponentNode[P]) hasDirectProvider() bool { return false }

func (c *ComponentNode[P]) Children() []Node {
	if c.rendered == nil {
		return nil
	}
	return []Node{c.rendered}
}

func (c *ComponentNode[P]) Instantiate(doc dom.Document) dom.Node {
	cr := componentRefPool.Get().(*componentRef)
	cr.node = c
	c.componentRef = cr
	pushCurrentComponent(c)
	c.isFirst = true
	c.hookIndex = 0
	c.rendered = c.RenderFn(c.PropsVal)
	c.isFirst = false
	popCurrentComponent()

	// Compute the complexity of the rendered subtree and decide whether future
	// updates should attempt memoization.
	c.complexityScore = computeComplexity(c.rendered)
	c.shouldMemo = c.complexityScore > memoComplexityThreshold

	if c.rendered != nil {
		c.ref = c.rendered.Instantiate(doc)
	}
	return c.ref
}

func (c *ComponentNode[P]) Update(el dom.Node, old Node) {
	oldComp, ok := old.(*ComponentNode[P])
	if !ok {
		return
	}
	// Transfer state
	c.hooks = oldComp.hooks
	c.isFirst = false
	c.ref = oldComp.ref
	c.complexityScore = oldComp.complexityScore
	c.shouldMemo = oldComp.shouldMemo
	c.componentRef = oldComp.componentRef
	if c.componentRef == nil {
		cr := componentRefPool.Get().(*componentRef)
		cr.node = c
		c.componentRef = cr
		oldComp.componentRef = c.componentRef
	} else {
		c.componentRef.mu.Lock()
		c.componentRef.node = c
		c.componentRef.mu.Unlock()
	}

	// Automatic memoization: when the rendered subtree is large enough and the
	// new props are deeply equal to the old props, skip the RenderFn entirely
	// and reuse the previously rendered subtree. The depth limit (3) caps the
	// worst-case reflection time so the frame budget is never blown by a single
	// pathological component.
	if c.shouldMemo && !oldComp.IsDirty() && deepEqualProps(oldComp.PropsVal, c.PropsVal, 3) {
		c.rendered = oldComp.rendered
		return
	}

	pushCurrentComponent(c)
	c.hookIndex = 0
	newRendered := c.RenderFn(c.PropsVal)
	popCurrentComponent()

	c.rendered = newRendered

	// Refresh the memoization score after re-rendering so it reflects the
	// current subtree shape.
	c.complexityScore = computeComplexity(c.rendered)
	c.shouldMemo = c.complexityScore > memoComplexityThreshold
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
	return c.componentRef
}

func (c *ComponentNode[P]) realNode() dom.Node {
	return c.ref
}

func (c *ComponentNode[P]) setRef(el dom.Node) {
	c.ref = el
}

func (c *ComponentNode[P]) Rendered() Node {
	return c.rendered
}

func (c *ComponentNode[P]) ReRender() Node {
	pushCurrentComponent(c)
	c.hookIndex = 0
	c.rendered = c.RenderFn(c.PropsVal)
	popCurrentComponent()
	c.complexityScore = computeComplexity(c.rendered)
	c.shouldMemo = c.complexityScore > memoComplexityThreshold
	return c.rendered
}

// complexity for a ComponentNode exposes the pre-computed score of its last
// rendered subtree so parent elements can accumulate it accurately.
func (c *ComponentNode[P]) complexity() int {
	return c.complexityScore
}

// IsDirty returns whether the component is dirty.
func (c *ComponentNode[P]) IsDirty() bool {
	return c.dirty
}

func (c *ComponentNode[P]) Release() {
	c.rendered = nil
	c.hooks = nil
	c.componentRef = nil

	if c.pool != nil {
		p := c.pool
		c.pool = nil
		p.Put(c)
	}
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

// Destroy cleans up component state, runs effect cleanups, and recursively destroys rendered subtree.
func (c *ComponentNode[P]) Destroy() {
	for _, h := range c.hooks {
		if eff, ok := h.(*effectHookState); ok {
			if eff.cleanup != nil {
				eff.cleanup()
				eff.cleanup = nil
			}
			eff.pending = false
		}
		if unsub, ok := h.(contextUnsubscriber); ok {
			unsub.unsubscribe(c)
		}
	}
	if c.componentRef != nil {
		c.componentRef.mu.Lock()
		c.componentRef.node = nil
		c.componentRef.mu.Unlock()
		componentRefPool.Put(c.componentRef)
		c.componentRef = nil
	}
	if c.rendered != nil {
		destroyNode(c.rendered)
	}
}

// destroyNode recursively walks the subtree and destroys any component instances.
func destroyNode(n Node) {
	if n == nil {
		return
	}
	if comp, ok := n.(componentInstance); ok {
		comp.Destroy()
	} else {
		for _, child := range n.Children() {
			destroyNode(child)
		}
	}
	n.Release()
}

// ReleaseTree recursively releases a virtual DOM tree back to its object pools.
func ReleaseTree(n Node) {
	if n == nil {
		return
	}
	for _, child := range n.Children() {
		ReleaseTree(child)
	}
	n.Release()
}

// OnComponentDirty is a package-level hook triggered when a component becomes dirty (e.g. state change).
var OnComponentDirty func(node Node)

// --- Execution Stack ----------------------------------------------------------

var (
	renderStackMutex sync.Mutex
	renderStack      = make([]any, 0, 32)
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

type structTypeInfo struct {
	keyFieldIdx          []int
	hasKey               bool
	elementPropsFieldIdx []int
	hasElementProps      bool
	elementPropsKeyIdx   []int
	hasElementPropsKey   bool
	childrenFieldIdx     []int
	hasChildren          bool
}

var typeInfoCache sync.Map

func getTypeInfo(t reflect.Type) *structTypeInfo {
	if val, ok := typeInfoCache.Load(t); ok {
		return val.(*structTypeInfo)
	}

	info := &structTypeInfo{}
	if t.Kind() == reflect.Struct {
		// Look for Key field
		if f, found := t.FieldByName("Key"); found {
			if f.Type.Kind() == reflect.String {
				info.keyFieldIdx = f.Index
				info.hasKey = true
			}
		}
		// Look for ElementProps field
		if ep, found := t.FieldByName("ElementProps"); found {
			if ep.Type.Kind() == reflect.Struct {
				info.elementPropsFieldIdx = ep.Index
				info.hasElementProps = true
				if k, foundKey := ep.Type.FieldByName("Key"); foundKey {
					if k.Type.Kind() == reflect.String {
						info.elementPropsKeyIdx = k.Index
						info.hasElementPropsKey = true
					}
				}
			}
		}
		// Look for Children field
		if ch, found := t.FieldByName("Children"); found {
			if ch.Type == reflect.TypeFor[[]Node]() {
				info.childrenFieldIdx = ch.Index
				info.hasChildren = true
			}
		}
	}

	typeInfoCache.Store(t, info)
	return info
}

func getKey(props any) string {
	if props == nil {
		return ""
	}
	t := reflect.TypeOf(props)
	isPtr := false
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		isPtr = true
	}
	if t.Kind() != reflect.Struct {
		return ""
	}

	info := getTypeInfo(t)
	v := reflect.ValueOf(props)
	if isPtr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	if info.hasKey {
		return v.FieldByIndex(info.keyFieldIdx).String()
	}
	if info.hasElementPropsKey {
		return v.FieldByIndex(info.elementPropsFieldIdx).FieldByIndex(info.elementPropsKeyIdx).String()
	}
	return ""
}

func getRefSetter(props any) refSetter {
	if props == nil {
		return nil
	}
	v := reflect.ValueOf(props)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	f := v.FieldByName("Ref")
	if f.IsValid() {
		switch f.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			if !f.IsNil() {
				if setter, ok := f.Interface().(refSetter); ok {
					return setter
				}
			}
		}
	}

	ep := v.FieldByName("ElementProps")
	if ep.IsValid() && ep.Kind() == reflect.Struct {
		f = ep.FieldByName("Ref")
		if f.IsValid() {
			switch f.Kind() {
			case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
				if !f.IsNil() {
					if setter, ok := f.Interface().(refSetter); ok {
						return setter
					}
				}
			}
		}
	}
	return nil
}

// FC creates a functional component wrapper that does not take children.
func FC[P any](name string, render func(P) Node) func(P) Node {
	var declFile string
	var declLine int
	if _, file, line, ok := runtime.Caller(1); ok {
		declFile = file
		declLine = line
	}

	pool := &sync.Pool{
		New: func() any { return &ComponentNode[P]{} },
	}

	return func(props P) Node {
		var instFile string
		var instLine int
		if EnableDevMode {
			if _, file, line, ok := runtime.Caller(1); ok {
				instFile = file
				instLine = line
			}
		}

		c := pool.Get().(*ComponentNode[P])
		*c = ComponentNode[P]{
			Name:     name,
			PropsVal: props,
			RenderFn: render,
			isFirst:  true,
			key:      getKey(&props),
			declFile: declFile,
			declLine: declLine,
			instFile: instFile,
			instLine: instLine,
			pool:     pool,
		}
		return c
	}
}

// FCC creates a functional component wrapper that can accept children.
// If the Props type has a Children field of type []Node, the variadic children
// will be injected using reflection.
func FCC[P any](name string, render func(P) Node) func(P, ...Node) Node {
	var declFile string
	var declLine int
	if _, file, line, ok := runtime.Caller(1); ok {
		declFile = file
		declLine = line
	}

	pool := &sync.Pool{
		New: func() any { return &ComponentNode[P]{} },
	}

	return func(props P, children ...Node) Node {
		var instFile string
		var instLine int
		if EnableDevMode {
			if _, file, line, ok := runtime.Caller(1); ok {
				instFile = file
				instLine = line
			}
		}

		propsWithChildren := injectChildren(props, children)
		c := pool.Get().(*ComponentNode[P])
		*c = ComponentNode[P]{
			Name:     name,
			PropsVal: propsWithChildren,
			RenderFn: render,
			isFirst:  true,
			key:      getKey(&propsWithChildren),
			declFile: declFile,
			declLine: declLine,
			instFile: instFile,
			instLine: instLine,
			pool:     pool,
		}
		return c
	}
}

func injectChildren[P any](props P, children []Node) P {
	var val any = props
	if val == nil {
		return props
	}
	t := reflect.TypeOf(val)
	isPtr := false
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		isPtr = true
	}
	if t.Kind() != reflect.Struct {
		return props
	}

	info := getTypeInfo(t)
	if !info.hasChildren {
		return props
	}

	v := reflect.ValueOf(props)
	if isPtr {
		if v.IsNil() {
			return props
		}
		f := v.Elem().FieldByIndex(info.childrenFieldIdx)
		if f.CanSet() {
			f.Set(reflect.ValueOf(children))
		}
		return props
	}

	ptr := reflect.New(t)
	ptr.Elem().Set(v)
	f := ptr.Elem().FieldByIndex(info.childrenFieldIdx)
	if f.CanSet() {
		f.Set(reflect.ValueOf(children))
	}
	return ptr.Elem().Interface().(P)
}

// SimpleFC creates a functional component wrapper for components with no props.
func SimpleFC(name string, render func() Node) func() Node {
	var declFile string
	var declLine int
	if _, file, line, ok := runtime.Caller(1); ok {
		declFile = file
		declLine = line
	}

	return func() Node {
		var instFile string
		var instLine int
		if EnableDevMode {
			if _, file, line, ok := runtime.Caller(1); ok {
				instFile = file
				instLine = line
			}
		}

		c := simpleComponentPool.Get().(*ComponentNode[struct{}])
		*c = ComponentNode[struct{}]{
			Name:     name,
			PropsVal: struct{}{},
			RenderFn: func(_ struct{}) Node { return render() },
			isFirst:  true,
			declFile: declFile,
			declLine: declLine,
			instFile: instFile,
			instLine: instLine,
			pool:     &simpleComponentPool,
		}
		return c
	}
}

// SimpleFCC creates a functional component wrapper for components that only take children.
func SimpleFCC(name string, render func([]Node) Node) func(...Node) Node {
	var declFile string
	var declLine int
	if _, file, line, ok := runtime.Caller(1); ok {
		declFile = file
		declLine = line
	}

	return func(children ...Node) Node {
		var instFile string
		var instLine int
		if EnableDevMode {
			if _, file, line, ok := runtime.Caller(1); ok {
				instFile = file
				instLine = line
			}
		}

		c := simpleFCCComponentPool.Get().(*ComponentNode[[]Node])
		*c = ComponentNode[[]Node]{
			Name:     name,
			PropsVal: children,
			RenderFn: func(p []Node) Node { return render(p) },
			isFirst:  true,
			declFile: declFile,
			declLine: declLine,
			instFile: instFile,
			instLine: instLine,
			pool:     &simpleFCCComponentPool,
		}
		return c
	}
}
