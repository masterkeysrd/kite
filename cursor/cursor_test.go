package cursor

import "testing"

func TestState(t *testing.T) {
	s := State{
		Visible: true,
		X:       10,
		Y:       5,
		Shape:   ShapeBarBlink,
	}

	if !s.Visible {
		t.Errorf("expected Visible to be true")
	}
	if s.X != 10 || s.Y != 5 {
		t.Errorf("expected (10, 5), got (%d, %d)", s.X, s.Y)
	}
	if s.Shape != ShapeBarBlink {
		t.Errorf("expected ShapeBarBlink, got %v", s.Shape)
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
