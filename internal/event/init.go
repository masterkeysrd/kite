package event

import (
	"github.com/masterkeysrd/kite/event"
)

func init() {
	event.RegisterImplementation(event.Implementation{
		NewDispatcher: func() event.Dispatcher {
			return NewDispatcher()
		},
		NewTarget: func() event.EventTarget {
			return &Target{}
		},
	})
}
