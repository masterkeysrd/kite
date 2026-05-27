package kitex

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
)

func TestUseFocus(t *testing.T) {
	doc := dom.NewDocument()

	type State struct {
		isFocused bool
	}

	state := &State{}

	FocusApp := SimpleFC("FocusApp", func() Node {
		ref := UseRef[dom.Element](nil)

		isFocused := UseFocus(ref)
		state.isFocused = isFocused

		return Box(BoxProps{
			Ref: ref,
		}, Text("Focus me!"))
	})

	container := doc.CreateElement("container", nil)
	Render(FocusApp(), container)

	// Flush effects
	var pendingFn func()
	SetPostMacroFn(func(fn func()) {
		pendingFn = fn
	})

	Render(FocusApp(), container)

	if pendingFn != nil {
		pendingFn()
	}

	// TestUseFocus_InitiallyFalse
	if state.isFocused != false {
		t.Errorf("Expected isFocused to be false initially")
	}

	boxEl := container.FirstChild().(dom.Element)

	// TestUseFocus_TrueOnFocus
	focusEv := event.NewFocusEvent(event.EventFocus, nil)
	boxEl.DispatchToTarget(focusEv)

	// Since we are mocking render loop, the state update schedules a re-render.
	// We need to re-render to see the new state.
	Render(FocusApp(), container)

	if state.isFocused != true {
		t.Errorf("Expected isFocused to be true after focus event")
	}

	// TestUseFocus_FalseOnBlur
	blurEv := event.NewFocusEvent(event.EventBlur, nil)
	boxEl.DispatchToTarget(blurEv)

	Render(FocusApp(), container)

	if state.isFocused != false {
		t.Errorf("Expected isFocused to be false after blur event")
	}

	// TestUseFocus_NilRef
	// Let's create another component with a nil ref
	NilRefApp := SimpleFC("NilRefApp", func() Node {
		ref := CreateRef[dom.Element]() // Empty ref
		UseFocus(ref)                   // Should not panic
		return Box(BoxProps{}, Text("Nil ref"))
	})

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("UseFocus panicked on nil ref: %v", r)
		}
	}()
	Render(NilRefApp(), container)
}
