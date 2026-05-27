package kitex

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
)

func TestUseKeyboard(t *testing.T) {
	doc := dom.NewDocument()

	type State struct {
		lastPressed string
	}

	state := &State{}

	KeyboardApp := SimpleFC("KeyboardApp", func() Node {
		UseKeyboard(func(e event.KeyEvent) {
			state.lastPressed = e.Text
		}, nil)

		return Box(BoxProps{}, Text("Keyboard hook app"))
	})

	// TestUseKeyboard_HandlerCalled
	// We need to simulate the macro/micro loop to flush the UseEffect
	var pendingFn func()
	SetPostMacroFn(func(fn func()) {
		pendingFn = fn
	})

	container := doc.CreateElement("container", nil)
	Render(KeyboardApp(), container)

	if pendingFn != nil {
		pendingFn() // Flush effect to register listener
	}

	// Dispatch key press to document
	keyEv := event.NewKeyEvent(event.EventKeyDown, key.Key{Text: "A", Code: 'A'})
	doc.DispatchToTarget(keyEv)

	if state.lastPressed != "A" {
		t.Errorf("Expected lastPressed to be 'A', got %q", state.lastPressed)
	}

	// TestUseKeyboard_Cleanup
	// Unmount the component to trigger cleanup
	Render(nil, container)

	if pendingFn != nil {
		pendingFn() // Flush cleanup effect
	}

	// Dispatch another key press
	keyEv2 := event.NewKeyEvent(event.EventKeyDown, key.Key{Text: "B", Code: 'B'})
	doc.DispatchToTarget(keyEv2)

	// Since the component is unmounted and listener removed, state should not update
	if state.lastPressed == "B" {
		t.Errorf("Expected handler to be removed on unmount, but it was still called")
	}
}
