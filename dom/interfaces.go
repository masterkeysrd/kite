package dom

import (
	"iter"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/render"
	"github.com/masterkeysrd/kite/style"
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
// It carries parent/sibling links, owner-document reference, and an optional
// back-reference to the render object created for this node by the engine.
// DOM nodes do not own dirty flags, layout state, or computed style — those
// live on render.Object.
type Node interface {
	event.EventTarget

	// Kind returns the kind of this node [Document], Element, or Text).
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

	// RenderObject returns the render.Object associated with this node, or nil
	// if the node does not (yet) participate in rendering.
	RenderObject() render.Object

	// SetRenderObject attaches or detaches the render object for this node.
	// Called by the engine during the attachment phase.
	SetRenderObject(render.Object)

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

	// NeedsSync reports whether this node's children need to be synchronized with the render tree.
	NeedsSync() bool

	// ChildNeedsSync reports whether one of this node's descendants needs synchronization.
	ChildNeedsSync() bool

	// MarkNeedsSync marks this node as needing synchronization and propagates the
	// ChildNeedsSync flag up to the document root.
	MarkNeedsSync()

	// ClearSyncFlags clears both NeedsSync and ChildNeedsSync flags on this node.
	ClearSyncFlags()

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

	// IntrinsicStyle returns the UA-mandated sparse style for this element.
	// Properties set here have the highest cascade precedence and cannot be
	// overridden by author styles (RawStyle). The default implementation on
	// the base element returns an empty style.Style{}, meaning no properties
	// are forced. Replaced and compound elements override this to enforce
	// UA-mandatory properties (e.g. display:inline-block, overflow:clip).
	//
	// See ADR-010 for the cascade-origin model.
	IntrinsicStyle() style.Style

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
	GetBoundingClientRect() (layout.Rect, bool)
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
}
