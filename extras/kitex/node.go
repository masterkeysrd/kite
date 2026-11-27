package kitex

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

var EnableDevMode bool

// Node is the public interface representing a Virtual DOM node in the kitex framework.
type Node interface {
	// Instantiate constructs the corresponding real DOM node using the provided document.
	Instantiate(doc dom.Document) []dom.Node

	// Update applies properties from the current VDOM node onto the existing real DOM node,
	// using the old VDOM node as reference to determine which properties changed.
	Update(els []dom.Node, old Node)

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
	realNodes() []dom.Node
	setRefs([]dom.Node)
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

// ElementProps holds common fields and typed event listeners present on all DOM element nodes.
type ElementProps struct {
	Key             string
	ID              string
	Class           string
	Style           style.Style
	Hidden          bool
	Disabled        bool
	CursorNavigable bool
	Attributes      map[string]string
	OnKeyDown       func(*event.KeyEvent)
	OnKeyUp         func(*event.KeyEvent)
	OnKeyPress      func(*event.KeyEvent)
	OnMouseDown     func(*event.MouseEvent)
	OnMouseUp       func(*event.MouseEvent)
	OnMouseMove     func(*event.MouseEvent)
	OnClick         func(*event.MouseEvent)
	OnDrag          func(*event.MouseEvent)
	OnWheel         func(*event.WheelEvent)
	OnFocus         func(*event.FocusEvent)
	OnBlur          func(*event.FocusEvent)
	OnChange        func(*event.ChangeEvent)
	OnInput         func(*event.InputEvent)
	OnScroll        func(*event.ScrollEvent)
	OnMouseEnter    func(*event.MouseEvent)
	OnMouseLeave    func(*event.MouseEvent)
	OnMouseOver     func(*event.MouseEvent)
	OnMouseOut      func(*event.MouseEvent)
	Ref             refSetter
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
	refs        []dom.Node

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

func (n *elementNode[P]) TagName() string  { return n.tagName }
func (n *elementNode[P]) Props() any       { return n.props }
func (n *elementNode[P]) Children() []Node { return n.children }
func (n *elementNode[P]) Key() string      { return n.key }
func (n *elementNode[P]) realNodes() []dom.Node {
	return n.refs
}
func (n *elementNode[P]) setRefs(els []dom.Node) {
	n.refs = els
	if len(els) > 0 {
		n.ref = els[0]
	}
}
func (n *elementNode[P]) complexity() int         { return n.score }
func (n *elementNode[P]) containsProvider() bool  { return n.hasProvider }
func (n *elementNode[P]) isProvider() bool        { return false }
func (n *elementNode[P]) hasDirectProvider() bool { return n.hasDirectP }

func (n *elementNode[P]) Release() {}

func (n *elementNode[P]) getSource() (string, int, string, int) {
	return n.declFile, n.declLine, n.instFile, n.instLine
}

func (n *elementNode[P]) setSource(declFile string, declLine int, instFile string, instLine int) {
	n.declFile = declFile
	n.declLine = declLine
	n.instFile = instFile
	n.instLine = instLine
}

func (n *elementNode[P]) Instantiate(doc dom.Document) []dom.Node {
	n.ref = n.instantiate(doc)
	n.update(n.ref, nil, &n.props)
	parentEl := n.ref.(dom.Element)
	if n.hasDirectP {
		for _, child := range n.children {
			bindChild(child, parentEl, n)
		}
		flatChildren := flattenNodes(n.children, nil, nil)
		for _, childFlat := range flatChildren {
			childReals := instantiateFlat(parentEl, childFlat)
			for _, childReal := range childReals {
				if childReal != nil {
					n.ref.AppendChild(childReal)
				}
			}
		}
	} else {
		for _, child := range n.children {
			if child != nil {
				bindChild(child, parentEl, n)
				childReals := child.Instantiate(doc)
				for _, childReal := range childReals {
					if childReal != nil {
						n.ref.AppendChild(childReal)
					}
				}
			}
		}
	}
	if setter := getRefSetter(&n.props); setter != nil {
		setter.set(n.ref)
	}
	n.refs = []dom.Node{n.ref}
	return n.refs
}

func (n *elementNode[P]) Update(els []dom.Node, old Node) {
	n.refs = els
	if len(els) > 0 {
		n.ref = els[0]
	}
	if n.ref == nil {
		return
	}
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
	refs    []dom.Node

	declFile string
	declLine int
	instFile string
	instLine int
}

// complexity for a text leaf is always 1.
func (t *textNode) complexity() int { return 1 }

func (t *textNode) TagName() string  { return "#text" }
func (t *textNode) Props() any       { return t.content }
func (t *textNode) Children() []Node { return nil }
func (t *textNode) Key() string      { return "" }
func (t *textNode) realNodes() []dom.Node {
	return t.refs
}
func (t *textNode) setRefs(els []dom.Node) {
	t.refs = els
	if len(els) > 0 {
		t.ref = els[0]
	}
}
func (t *textNode) containsProvider() bool  { return false }
func (t *textNode) isProvider() bool        { return false }
func (t *textNode) hasDirectProvider() bool { return false }

func (t *textNode) Release() {}

func (t *textNode) getSource() (string, int, string, int) {
	return t.declFile, t.declLine, t.instFile, t.instLine
}

func (t *textNode) setSource(declFile string, declLine int, instFile string, instLine int) {
	t.declFile = declFile
	t.declLine = declLine
	t.instFile = instFile
	t.instLine = instLine
}

func (t *textNode) Instantiate(doc dom.Document) []dom.Node {
	t.ref = element.NewText(doc, t.content)
	t.refs = []dom.Node{t.ref}
	return t.refs
}

func (t *textNode) Update(els []dom.Node, old Node) {
	t.refs = els
	if len(els) > 0 {
		t.ref = els[0]
	}
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

var (
	textNodePool = sync.Pool{
		New: func() any { return &textNode{} },
	}

	elementNodePool = sync.Pool{
		New: func() any { return &elementNode[ElementProps]{} },
	}
)

// Text creates a VDOM representation of a text node.
func Text(data string) Node {
	t := textNodePool.Get().(*textNode)
	t.content = data
	return trackSource(t, 1)
}

// Textf creates a VDOM representation of a formatted text node.
func Textf(format string, args ...any) Node {
	t := textNodePool.Get().(*textNode)
	t.content = fmt.Sprintf(format, args...)
	return trackSource(t, 1)
}

type sourceTracker interface {
	getSource() (string, int, string, int)
}

type sourceTrackable interface {
	setSource(declFile string, declLine int, instFile string, instLine int)
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
