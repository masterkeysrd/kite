package render

import (
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

type Box struct {
	BaseRender
}

var _ Object = (*Box)(nil)

func NewBox(logicalNode dom.Node, target event.EventTarget) *Box {
	f := &Box{}
	f.Init(f, logicalNode, target)
	f.SetComputedStyle(&style.Computed{Display: style.DisplayBlock})
	return f
}
