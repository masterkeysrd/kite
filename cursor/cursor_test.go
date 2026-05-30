package cursor

import (
	"testing"

	"github.com/masterkeysrd/kite/style"
)

func TestState(t *testing.T) {
	s := State{
		Visible: true,
		X:       10,
		Y:       5,
		Style: style.Cursor{
			Shape: style.Some(style.CursorBar),
			Blink: style.Some(true),
		},
	}

	if !s.Visible {
		t.Errorf("expected Visible to be true")
	}
	if s.X != 10 || s.Y != 5 {
		t.Errorf("expected (10, 5), got (%d, %d)", s.X, s.Y)
	}
	if s.Style.Shape.UnwrapOr(style.CursorBlock) != style.CursorBar {
		t.Errorf("expected CursorBar, got %v", s.Style.Shape)
	}
}

type mockProvider struct {
	state State
}

func (m *mockProvider) CursorState() State {
	return m.state
}

func TestProvider(t *testing.T) {
	var _ Provider = (*mockProvider)(nil)
}
