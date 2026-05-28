package dom

import (
	"iter"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/marker"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/terminal"
)

// Kind identifies the type of a Node. It is used by Node.Kind() to allow callers to determine
// the concrete type of a Node without relying on Go's type system or incurring the cost of
// a type assertion. The set of possible values is fixed by the current DOM implementation
// and may not be extended by user code.
type Kind int

const (
	KindDocument Kind = iota
	KindElement
	KindText
)

func (k Kind) String() string {
	switch k {
	case KindDocument:
		return "Document"
	case KindElement:
		return "Element"
	case KindText:
		return "Text"
	default:
		return "Unknown"
	}
}

// Node is the base interface for every node in the logical DOM tree.
// It carries parent/sibling links and owner-document reference.
// DOM nodes do not own dirty flags, layout state, or computed style — those
// live on render.Object.
type Node interface {
	event.EventTarget
	marker.Node

	// Kind returns the kind of this node (Document, Element, or Text).
	Kind() Kind

	// NodeName returns the name of this node. For Element nodes it is the tag
	// name; for Text nodes it is "#text"; for Document it is "#document".
	NodeName() string

	// Parent returns the parent Node, or nil if this node is the Document root.
	Parent() Node

	// ParentElement returns the parent Node if it is an Element, or nil if
	// this node has no parent or its parent is not an Element.
	ParentElement() Element

	// NextSibling returns the next sibling Node, or nil if there is none.
	NextSibling() Node

	// PreviousSibling returns the previous sibling Node, or nil if there is none.
	PreviousSibling() Node

	// OwnerDocument returns the Document that owns this node.
	OwnerDocument() Document

	// IsConnected reports whether this node is reachable from the Document
	// root. The document itself is always connected. The value is toggled by
	// the attach/detach walks run inside AppendChild, InsertBefore,
	// RemoveChild, and ReplaceChild.
	IsConnected() bool

	// AppendChild adds child as the last child of this node and returns child.
	AppendChild(child Node) Node

	// InsertBefore inserts newChild immediately before ref and returns newChild.
	// If ref is nil the call is equivalent to AppendChild.
	InsertBefore(newChild, ref Node) Node

	// RemoveChild removes child from this node and returns child.
	RemoveChild(child Node) Node

	// ReplaceChild inserts newChild in the position occupied by oldChild,
	// removes oldChild from the tree, and returns oldChild.
	ReplaceChild(newChild, oldChild Node) Node

	// FirstChild returns the first child Node, or nil if the node has no children.
	FirstChild() Node

	// LastChild returns the last child Node, or nil if the node has no children.
	LastChild() Node

	// HasChildNodes reports whether this node has any children.
	HasChildNodes() bool

	// Contains reports whether this node is an ancestor of descendant.
	Contains(descendant Node) bool

	// ChildNodes returns an iterator over direct children in document order.
	ChildNodes() iter.Seq[Node]

	// Unwrap returns the underlying Node being decorated by this wrapper, or nil
	// if this node is a base implementation.
	Unwrap() Node

	// TextContent returns the concatenation of all text content in this node's subtree. For Text
	// nodes it is the same as Data(). For Element and Document nodes it is the concatenation of
	// the TextContent of all descendant Text nodes in document order.
	TextContent() string

	// CloneNode returns a new Node that is a copy of this node. If deep is true, the clone also
	// includes clones of all descendant nodes; otherwise the clone has no children. The returned
	// node is detached (no parent).
	CloneNode(deep bool) Node

	// EventTarget returns the user-visible event target for this node. For
	// nodes in a UA shadow subtree, this returns the host element; otherwise
	// it returns the node itself (ADR-0036).
	EventTarget() event.EventTarget
}

// Element extends Node with identity (tag name, id).
type Element interface {
	Node

	// TagName returns the tag name used to create this element.
	TagName() string

	// ID returns the element's identifier attribute value.
	ID() string

	// SetID sets the element's identifier attribute value.
	SetID(id string)

	// Class returns the element's classification tag.
	Class() string

	// SetClass sets the element's classification tag.
	SetClass(class string)

	// QuerySelector returns the first element matching the selector in this element's subtree.
	// It pierces UA shadow subtrees.
	QuerySelector(selector string) Element

	// ReplaceWith replaces this element with the given nodes and returns this
	// element. The new nodes are appended in the order they are given. If this
	// element has no parent, the call is a no-op and this element is returned.
	ReplaceWith(nodes ...Node) Element

	// AttachUARoot attaches root as the host element's closed UA shadow subtree.
	// The UA subtree is invisible to all public traversal APIs (Children(),
	// FirstChild(), LastChild(), ChildNodes(), GetElementByID()) but is walked
	// by the engine during Sync, Style, Layout, and Paint phases via
	// dom.LayoutChildren().
	//
	// AttachUARoot recursively sets the outer back-pointer on every node in
	// root's subtree to this host element, so event.Target() and similar
	// identity queries collapse to the host (ADR-0036). It also marks the host
	// as NeedsSync so the engine picks up the new subtree on the next Sync pass.
	//
	// AttachUARoot must be called eagerly in the host's constructor, before the
	// host is connected to a document. The ua root must be created against the
	// same Document as the host. Calling AttachUARoot more than once panics.
	AttachUARoot(root Node)

	// Scroll returns the current raw scroll offset (x, y) in terminal cells.
	Scroll() (x, y int)

	// ScrollTo sets the raw scroll offset to (x, y) and marks the element
	// for a paint update (DirtyScroll). Dispatches event.EventScroll.
	ScrollTo(x, y int)

	// ScrollBy shifts the raw scroll offset by (dx, dy).
	ScrollBy(dx, dy int)

	// ScrollCursorIntoView scrolls the element so that the cursor (caret) is
	// visible within the content box. It is typically called by the engine
	// after layout if the element is focused.
	ScrollCursorIntoView()

	// ProvidesCursor reports whether this element provides a text cursor.
	// Used by the layout and paint engines to adjust scroll clamping.
	ProvidesCursor() bool

	// GetBoundingClientRect returns the physical terminal rectangle occupied
	// by this element. It returns (rect, true) if the element is connected
	// and has been laid out; otherwise it returns (Rect{}, false).
	GetBoundingClientRect() (geom.Rect, bool)

	// TabIndex returns the tab index of the element.
	TabIndex() int
	// SetTabIndex sets the tab index of the element.
	SetTabIndex(index int)
	// Focus attempts to move focus to this element.
	Focus()
	// Blur removes focus from this element.
	Blur()
	// IsFocusable reports whether this element is currently focusable.
	IsFocusable() bool
}

// TextNode is a leaf node that carries character data. It has no children.
type TextNode interface {
	Node

	// Data returns the current text content.
	Data() string

	// SetData replaces the text content and notifies the parent's render object.
	SetData(data string)
}

// Lifecycle is an optional interface that DOM nodes may implement to receive
// notifications when they enter or leave the live tree (i.e. when they become
// connected to or disconnected from the Document root).
//
// The attach walk fires OnConnected in pre-order (parent before children).
// The detach walk fires OnDisconnected in post-order (children before parent).
// Both callbacks run synchronously inside the mutation call that triggered the
// walk, before the mutation returns to the caller.
//
// Self- and descendant-mutations are permitted inside either callback.
// Ancestor-mutations are forbidden and panic in development builds.
type Lifecycle interface {
	// OnConnected is called when the node becomes reachable from the Document
	// root. The node's IsConnected predicate is already true when this fires.
	OnConnected()

	// OnDisconnected is called when the node is about to leave the live tree.
	// The node's IsConnected predicate is still true when this fires; it
	// becomes false after the callback returns.
	OnDisconnected()
}

// Disableable indicates that an element can be semantically disabled.
type Disableable interface {
	IsDisabled() bool
	SetDisabled(bool)
}

// FocusScope is a focus containment region. While a FocusScope is active, tab navigation
// and focusable-filter queries are restricted to the subtree rooted at Root.
//
// Lifecycle:
//   - PushScope captures the current focus into PreviousFocus.
//   - PopScope restores PreviousFocus with ReasonRestore, or blurs if the
//     previous node is no longer focusable.
type FocusScope struct {
	// Root is the logical node that acts as the boundary for tab navigation
	// and focus queries while this scope is active. Must not be nil.
	Root Node

	// Autofocus is the initial focus target when the scope is pushed.
	// If nil no autofocus is applied; focus stays on the previous element
	// until a navigation or programmatic Focus call.
	Autofocus Element

	// PreviousFocus is captured automatically on PushScope. It is restored
	// with ReasonRestore when PopScope is called. Do not set this field
	// manually; Manager writes it on PushScope.
	PreviousFocus Element
}

// Focusable indicates that an element can be focused.
type Focusable interface {
	IsFocusable() bool
	Focus()
	Blur()
}

// Document is the root of a DOM tree and the factory for all new nodes.
// Tree-global concerns such as focus management, and the task
// scheduler are out of scope here and will be added in later tasks.
type Document interface {
	Node

	// CreateElement returns a new Element with the given tag name, owned by
	// this document. If self is nil, the element's identity is itself;
	// if self is not nil, it is set as the element's identity (Task 02).
	// The returned node is detached (no parent).
	CreateElement(tag string, self Node) Element

	// CreateTextNode returns a new TextNode with the given data, owned by
	// this document. If self is nil, the node's identity is itself.
	// The returned node is detached (no parent).
	CreateTextNode(data string, self Node) TextNode

	// GetElementByID returns the Element whose ID equals id, or nil if no
	// such element exists in this document. The lookup is O(1); the registry
	// is maintained by SetID and RemoveChild.
	GetElementByID(id string) Element

	// FindAnchor returns the Element registered under name in the anchor
	// registry, or nil if no anchor with that name is known. The anchor
	// registry is separate from the ID registry so that anchor names and
	// element IDs do not shadow each other.
	FindAnchor(name string) Element

	// RegisterAnchor adds el to the anchor registry under name. Called by
	// Anchor elements (Task 18) when their Name property is set.
	RegisterAnchor(name string, el Element)

	// UnregisterAnchor removes the entry for name from the anchor registry.
	// Called by Anchor elements when they are destroyed or their name changes.
	UnregisterAnchor(name string)

	// Body returns the root element of the document (similar to <body>).
	// If no body is mounted, it returns nil.
	Body() Element

	// Focus acquisition and state.
	Focus(el Element)
	IsFocused(el Element) bool

	// Focus scoping.
	PushScope(scope *FocusScope)
	PopScope()
	ActiveScope() *FocusScope

	// Focus navigation.
	CurrentFocus() Element
	NextFocus() bool
	PreviousFocus() bool

	// QuerySelector returns the first element matching the selector in this document's subtree.
	// It pierces UA shadow subtrees.
	QuerySelector(selector string) Element

	// ShowOverlay adds el to the top layer at the specified z-index.
	// Elements in the top layer are rendered above the document body and
	// ignore the normal layout flow. If el is already an overlay, its
	// z-index is updated.
	ShowOverlay(el Element, zIndex int)

	// HideOverlay removes el from the top layer. If el is not an overlay,
	// the call is a no-op.
	HideOverlay(el Element)

	// Overlays returns an iterator over all elements currently in the top
	// layer, sorted by z-index (ascending) and then by insertion order.
	Overlays() iter.Seq[Element]

	// Selection returns the selection object for this document, which
	// represents the range of text selected by the user or current caret
	// position.
	Selection() Selection

	// CreateRange creates a new Range for this document.
	CreateRange() Range

	// SetFocusHandle injects the focus management implementation into the document.
	SetFocusHandle(handle FocusHandle)

	// Clipboard returns the high-level clipboard provider for this document.
	Clipboard() event.ClipboardProvider
	// SetClipboardProvider sets the high-level clipboard provider for this document.
	SetClipboardProvider(p event.ClipboardProvider)

	// Terminal returns the terminal object for this document.
	Terminal() terminal.Terminal
	// SetTerminal sets the terminal object for this document.
	SetTerminal(t terminal.Terminal)

	// View returns the layout/style view for this document.
	View() View
	// SetView sets the layout/style view for this document.
	SetView(v View)

	// FindNodeAtByteOffset performs a structural walk to find the text node
	// and rune offset corresponding to a flat byte offset in the document.
	FindNodeAtByteOffset(root Node, targetOffset int) (Node, int)
}

// View provides read-only access to computed style and layout information
// for a DOM tree without coupling the DOM to the render engine.
type View interface {
	// GetBoundingClientRect returns the physical terminal rectangle occupied
	// by the node.
	GetBoundingClientRect(n Node) (geom.Rect, bool)

	// GetComputedStyle returns the fully-resolved computed style for the node.
	GetComputedStyle(n Node) *style.Computed

	// GetSize returns the physical size of the node's layout fragment.
	GetSize(n Node) (geom.Size, bool)

	// GetMaxScroll returns the maximum horizontal and vertical scroll offsets
	// for the node.
	GetMaxScroll(n Node) (x, y int)

	// GetCaretPosition returns the content-relative coordinates (relative to
	// the content box origin) for the given rune offset in the node.
	GetCaretPosition(n Node, offset int) (geom.Point, bool)

	// MoveCursorVertically returns the new rune offset after moving up (delta < 0)
	// or down (delta > 0) from the current offset. x and y are the current
	// content-relative coordinates used for vertical tracking.
	MoveCursorVertically(n Node, offset int, delta int, x, y int) int

	// ByteOffsetAtPoint returns the byte offset within the node at the given
	// content-relative coordinates.
	ByteOffsetAtPoint(n Node, x, y int) int

	// NodeAtPoint returns the leaf node and its rune offset at the given
	// screen coordinates.
	NodeAtPoint(x, y int) (Node, int)
}

// FocusHandle is the interface for the focus management implementation injected
// by the engine. It decouples the logical DOM from the focus package.
type FocusHandle interface {
	Focus(el Element)
	IsFocused(el Element) bool
	PushScope(scope *FocusScope)
	PopScope()
	ActiveScope() *FocusScope
	Current() Element
	Next() bool
	Previous() bool
}

// Range represents a fragment of a document that can contain nodes and parts
// of text nodes.
type Range interface {
	// StartContainer returns the Node within which the Range starts.
	StartContainer() Node
	// StartOffset returns a number representing where in the StartContainer
	// the Range starts. For Text nodes, this is the number of runes from
	// the start of the data.
	StartOffset() int
	// EndContainer returns the Node within which the Range ends.
	EndContainer() Node
	// EndOffset returns a number representing where in the EndContainer the
	// Range ends.
	EndOffset() int

	// SetStart sets the start position of a Range.
	SetStart(node Node, offset int)
	// SetEnd sets the end position of a Range.
	SetEnd(node Node, offset int)
	// Collapse collapses the Range to one of its boundary points.
	Collapse(toStart bool)
	// IsCollapsed reports whether the Range's start and end points are the same.
	IsCollapsed() bool

	// String returns the text content of the Range.
	String() string
}

// Selection represents the range of text selected by the user or the current
// caret position.
type Selection interface {
	// RangeCount returns the number of ranges in the selection.
	RangeCount() int
	// GetRangeAt returns the range at the specified index.
	GetRangeAt(index int) Range
	// AddRange adds a Range to a Selection.
	AddRange(r Range)
	// RemoveAllRanges removes all ranges from the selection.
	RemoveAllRanges()
	// String returns a string currently being represented by the selection
	// (the combined text of all its ranges).
	String() string
}

// FormControl represents a DOM element that carries a name and a value
// for form submission.
type FormControl interface {
	Node
	Name() string
	Value() any
}
