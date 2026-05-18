package render

import (
	"testing"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

type mockTarget struct {
	event.Target
}

func TestInlineRenderObject(t *testing.T) {
	target := &mockTarget{}
	inline := NewInline("logical", target)

	if inline.ComputedStyle().Display != style.DisplayInline {
		t.Errorf("expected DisplayInline, got %v", inline.ComputedStyle().Display)
	}

	if inline.LogicalNode() != "logical" {
		t.Errorf("expected logical node 'logical', got %v", inline.LogicalNode())
	}
}

func TestTextRenderObject(t *testing.T) {
	target := &mockTarget{}
	text := NewText("logical", target)

	if text.ComputedStyle().Display != style.DisplayInline {
		t.Errorf("expected DisplayInline, got %v", text.ComputedStyle().Display)
	}

	if text.LayoutChildren() == nil {
		t.Error("expected LayoutChildren iterator to be non-nil")
	}

	// Verify it has no children
	count := 0
	for _ = range text.LayoutChildren() {
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 layout children, got %d", count)
	}
}
