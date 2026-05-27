package render

import (
	"testing"

	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/internal/layout"
	"github.com/masterkeysrd/kite/style"
)

type mockTarget struct{}

func (m *mockTarget) AddEventListener(typ event.EventType, fn event.Listener, opts ...event.Option) event.Subscription {
	return nil
}
func (m *mockTarget) DispatchTo(e event.Event)       {}
func (m *mockTarget) DispatchToTarget(e event.Event) {}
func (m *mockTarget) RemoveRegistration(id uint64)   {}
func (m *mockTarget) EventTarget() event.EventTarget { return m }

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
	for range text.LayoutChildren() {
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 layout children, got %d", count)
	}
}

func TestBaseRender_MarkDirty_Bubbles(t *testing.T) {
	parent := NewBlock(nil, nil)
	child := NewBlock(nil, nil)
	parent.InsertChild(child, nil)

	// Reset parent flags
	parent.ClearDirtyRecursive(DirtyLayout | DirtyStyle | DirtyPaint | ChildNeedsLayout | ChildNeedsStyle | ChildNeedsPaint)

	if parent.Flags() != Clean {
		t.Errorf("expected parent flags to be Clean, got %v", parent.Flags())
	}

	// 1. Style bubbling
	child.MarkDirty(DirtyStyle)
	if parent.Flags()&ChildNeedsStyle == 0 {
		t.Error("parent did not get ChildNeedsStyle after child DirtyStyle")
	}

	// 2. Layout bubbling
	child.MarkDirty(DirtyLayout)
	if parent.Flags()&ChildNeedsLayout == 0 {
		t.Error("parent did not get ChildNeedsLayout after child DirtyLayout")
	}

	// 3. Paint bubbling
	child.MarkDirty(DirtyPaint)
	if parent.Flags()&ChildNeedsPaint == 0 {
		t.Error("parent did not get ChildNeedsPaint after child DirtyPaint")
	}
}

func TestBaseRender_CachedLayout_Invalidation(t *testing.T) {
	parent := NewBlock(nil, nil)
	child := NewBlock(nil, nil)
	parent.InsertChild(child, nil)

	space := layout.ConstraintSpace{AvailableSize: geom.Size{Width: 100, Height: 100}}
	frag := &layout.Fragment{Size: geom.Size{Width: 100, Height: 100}}

	parent.SetCachedLayout(space, frag)
	parent.ClearDirtyRecursive(DirtyLayout | ChildNeedsLayout)

	if parent.CachedLayout(space) != frag {
		t.Fatal("expected cached fragment to be returned")
	}

	// Invalidation by self dirty
	parent.MarkDirty(DirtyLayout)
	if parent.CachedLayout(space) != nil {
		t.Error("expected cache to be invalidated by DirtyLayout")
	}

	// Restore and invalidate by child dirty
	parent.SetCachedLayout(space, frag)
	parent.ClearDirtyRecursive(DirtyLayout | ChildNeedsLayout)
	child.MarkDirty(DirtyLayout)

	if parent.CachedLayout(space) != nil {
		t.Error("expected cache to be invalidated by ChildNeedsLayout")
	}
}

func TestRenderView_SetViewportSize_DirtyFlags(t *testing.T) {
	view := NewRenderView()
	view.ClearDirtyRecursive(DirtyLayout | DirtyPaint)

	view.SetViewportSize(geom.Size{Width: 100, Height: 100})

	if view.Flags()&DirtyLayout == 0 {
		t.Error("expected DirtyLayout after SetViewportSize")
	}
	if view.Flags()&DirtyPaint == 0 {
		t.Error("expected DirtyPaint after SetViewportSize")
	}
}
