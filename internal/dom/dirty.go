package dom

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/style"
)

// DirtyNode defines the internal interface for tracking structural and
// style-propagation dirty state on any logical node.
type DirtyNode interface {
	dom.Node

	MarkNeedsSync()
	NeedsSync() bool
	ChildNeedsSync() bool
	ClearSyncFlags()

	MarkNeedsScrollSync()
	NeedsScrollSync() bool

	HasDirtyStyleChild() bool
	MarkStyleChildDirty()
	ClearStyleFlags()
}

// DirtyElement defines the internal interface for tracking element-specific
// style properties and flags.
type DirtyElement interface {
	DirtyNode
	dom.Element

	MarkStyleDirty()
	IsDirtyStyle() bool
	ClearDirtyStyle()

	RawStyle() style.Style
	DefaultStyle() style.Style
	IntrinsicStyle() style.Style
}

// AsDirty returns the internal DirtyNode interface for the given node.
func AsDirty(n dom.Node) DirtyNode {
	if n == nil {
		return nil
	}
	if b, ok := n.(interface{ asBase() *BaseNode }); ok {
		return b.asBase()
	}
	// Fallback for wrappers.
	if unwrapped := n.Unwrap(); unwrapped != nil {
		return AsDirty(unwrapped)
	}
	return nil
}

// AsDirtyElement returns the internal DirtyElement interface for the given element.
func AsDirtyElement(n dom.Node) DirtyElement {
	if n == nil {
		return nil
	}
	// Check for Element specifically.
	if el, ok := n.(*Element); ok {
		return el
	}
	// Check for wrappers that wrap an *Element.
	if unwrapped := n.Unwrap(); unwrapped != nil {
		return AsDirtyElement(unwrapped)
	}
	return nil
}
