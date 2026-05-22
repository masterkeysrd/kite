package engine_test

import (
	"testing"
)

func TestEngine_SyncOverlays(t *testing.T) {
	e, _ := newTestEngine(t)
	defer e.Stop()

	doc := e.Document()
	overlayEl := doc.CreateElement("dialog", nil)
	doc.ShowOverlay(overlayEl, 100)

	// Trigger sync phase.
	e.Frame()

	rv := e.RenderView()
	overlays := rv.Overlays()

	if len(overlays) != 1 {
		t.Fatalf("expected 1 overlay in RenderView, got %d", len(overlays))
	}

	if overlays[0].LogicalNode() != overlayEl {
		t.Errorf("overlay render object logical node mismatch")
	}

	// Verify it has a fragment (meaning layout ran).
	if overlays[0].Fragment() == nil {
		t.Error("overlay render object should have a fragment after layout")
	}

	// Hide overlay.
	doc.HideOverlay(overlayEl)
	e.Frame()

	overlays = rv.Overlays()
	if len(overlays) != 0 {
		t.Errorf("expected 0 overlays in RenderView after HideOverlay, got %d", len(overlays))
	}
}

func TestEngine_SyncOverlaySubtree(t *testing.T) {
	e, _ := newTestEngine(t)
	defer e.Stop()

	doc := e.Document()
	overlayEl := doc.CreateElement("dialog", nil)
	childEl := doc.CreateElement("span", nil)
	overlayEl.AppendChild(childEl)

	doc.ShowOverlay(overlayEl, 100)

	// Trigger sync phase.
	e.Frame()

	rv := e.RenderView()
	overlays := rv.Overlays()

	if len(overlays) != 1 {
		t.Fatalf("expected 1 overlay in RenderView, got %d", len(overlays))
	}

	overlayRO := overlays[0]

	// Verify child was synced.
	found := false
	for child := range overlayRO.Children() {
		if child.LogicalNode() == childEl {
			found = true
			break
		}
	}
	if !found {
		t.Error("overlay child was not synced to render tree")
	}
}
