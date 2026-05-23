package render

import (
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

// Block represents a standard block-level formatting object.
type Block struct {
	BaseRender
}

var _ Object = (*Block)(nil)

// NewBlock creates a new Block render object.
func NewBlock(logicalNode any, target event.EventTarget) *Block {
	b := &Block{}
	b.Init(b, logicalNode, target)
	// Default block style
	b.SetComputedStyle(&style.Computed{Display: style.DisplayBlock})
	return b
}
