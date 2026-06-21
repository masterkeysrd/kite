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

func TestEngine_OverlayNotRenderedInFlow(t *testing.T) {
	e, _ := newTestEngine(t)
	defer e.Stop()

	doc := e.Document()
	parentEl := doc.CreateElement("box", nil)
	overlayEl := doc.CreateElement("dialog", nil)
	parentEl.AppendChild(overlayEl)
	doc.AppendChild(parentEl)
	doc.ShowOverlay(overlayEl, 100)

	e.Frame()

	parentRO := e.RenderObject(parentEl)
	if parentRO == nil {
		t.Fatal("expected parent render object to exist")
	}
	for child := range parentRO.Children() {
		if child.LogicalNode() == overlayEl {
			t.Fatal("overlay should not remain in the normal render tree while shown in the top layer")
		}
	}

	overlays := e.RenderView().Overlays()
	if len(overlays) != 1 || overlays[0].LogicalNode() != overlayEl {
		t.Fatal("overlay should be present in the render view top layer")
	}

	doc.HideOverlay(overlayEl)
	e.Frame()

	found := false
	for child := range parentRO.Children() {
		if child.LogicalNode() == overlayEl {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("overlay should return to the normal render tree after HideOverlay")
	}
}
