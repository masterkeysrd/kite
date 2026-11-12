// Regression tests for Kitex Reconciler — covers GitHub issue #16 (moved VDOM nodes changing root element type must not resurrect detached DOM nodes).
package regressions

import (
	"slices"
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestKitexMovedSegmentRootTypeChangeDoesNotResurrectDOM(t *testing.T) {
	env := testenv.New(t, 80, 5)
	defer env.Close()

	type segProps struct {
		Label   string
		WithSep bool
	}

	ModeSegment := kitex.FC("ModeSegment", func(p segProps) kitex.Node {
		if p.WithSep {
			return kitex.Box(kitex.BoxProps{ID: "mode-box"},
				kitex.Span(kitex.SpanProps{}, kitex.Text(p.Label)),
				kitex.Span(kitex.SpanProps{}, kitex.Text(" · ")),
			)
		}
		return kitex.Span(kitex.SpanProps{ID: "mode-span"}, kitex.Text(p.Label))
	})

	UsageSegment := kitex.FC("UsageSegment", func(p segProps) kitex.Node {
		if p.WithSep {
			return kitex.Box(kitex.BoxProps{ID: "usage-box"},
				kitex.Span(kitex.SpanProps{}, kitex.Text(p.Label)),
				kitex.Span(kitex.SpanProps{}, kitex.Text(" · ")),
			)
		}
		return kitex.Span(kitex.SpanProps{ID: "usage-span"}, kitex.Text(p.Label))
	})

	HintSegment := kitex.SimpleFC("HintSegment", func() kitex.Node {
		return kitex.Span(kitex.SpanProps{ID: "hint"}, kitex.Text("hint "))
	})

	Spacer := kitex.SimpleFC("Spacer", func() kitex.Node {
		return kitex.Box(kitex.BoxProps{
			ID:    "spacer",
			Style: style.S().Flex(1),
		})
	})

	ModelSegment := kitex.SimpleFC("ModelSegment", func() kitex.Node {
		return kitex.Span(kitex.SpanProps{ID: "model"}, kitex.Text(" | model"))
	})

	var setShowHint func(bool)
	StatusBar := kitex.SimpleFC("StatusBar", func() kitex.Node {
		showHint, setter := kitex.UseState(true)
		setShowHint = setter

		var children []kitex.Node
		if showHint() {
			children = append(children,
				HintSegment(),
				Spacer(),
				ModeSegment(segProps{Label: "auto", WithSep: true}),
				UsageSegment(segProps{Label: "usage", WithSep: true}),
				ModelSegment(),
			)
		} else {
			children = append(children,
				ModeSegment(segProps{Label: "auto", WithSep: false}),
				Spacer(),
				UsageSegment(segProps{Label: "usage", WithSep: false}),
				ModelSegment(),
			)
		}

		return kitex.Box(kitex.BoxProps{
			ID:    "status-row",
			Style: style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).Width(style.Cells(80)).Height(style.Cells(1)),
		}, children...)
	})

	container := element.Box()
	env.Mount(container)
	kitex.Render(StatusBar(), container)
	env.Flush()

	// Hide HintSegment -> shifts keyless siblings and flips ModeSegment & UsageSegment from Box to Span.
	setShowHint(false)
	env.Flush()

	statusRow := env.QuerySelector("#status-row")
	var gotTagsAndIDs []string
	for child := range statusRow.ChildNodes() {
		if el, ok := child.(dom.Element); ok {
			gotTagsAndIDs = append(gotTagsAndIDs, el.TagName()+"#"+el.ID())
		}
	}
	wantTagsAndIDs := []string{"span#mode-span", "box#spacer", "span#usage-span", "span#model"}
	if !slices.Equal(gotTagsAndIDs, wantTagsAndIDs) {
		t.Fatalf("unexpected DOM children after hiding hint: got %v, want %v", gotTagsAndIDs, wantTagsAndIDs)
	}

	line := env.DumpText()
	if strings.Count(line, "auto") != 1 {
		t.Fatalf("expected 'auto' to appear once on screen, got %d in %q", strings.Count(line, "auto"), line)
	}
	if strings.Count(line, "usage") != 1 {
		t.Fatalf("expected 'usage' to appear once on screen, got %d in %q", strings.Count(line, "usage"), line)
	}
	if strings.Contains(line, " · ") {
		t.Fatalf("expected old separator ' · ' from replaced Box to be gone, got %q", line)
	}
}
