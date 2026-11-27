// Regression tests for Style PseudoStates — covers TSK-078.
package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestPseudoStates_HoverFocusActiveDisabled(t *testing.T) {
	env := testenv.New(t, 40, 12)
	defer env.Close()

	cBase := style.Hex("#111111")
	cHover := style.Hex("#222222")
	cFocus := style.Hex("#333333")
	cActive := style.Hex("#444444")
	cDisabled := style.Hex("#555555")

	btn1 := element.Button("Btn1").WithID("btn1").Style(
		style.S().
			Background(cBase).
			Hover(style.S().Background(cHover)).
			Focus(style.S().Background(cFocus)).
			Active(style.S().Background(cActive)),
	)
	btn2 := element.Button("Btn2").WithID("btn2").Style(
		style.S().
			Background(cBase).
			Focus(style.S().Background(cFocus)),
	)
	btnDisabled := element.Button("Dis").WithID("btnDis").Disabled(true).Style(
		style.S().
			Background(cBase).
			Disabled(style.S().Background(cDisabled)),
	)

	root := element.Box(btn1, btn2, btnDisabled).Style(
		style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).Gap(1),
	)
	env.Mount(root)
	env.Flush()

	// btn1 starts focused by initial auto-focus -> should show Focus color.
	ro1 := env.RenderObject(btn1)
	if got := ro1.ComputedStyle().Background; got != cFocus {
		t.Fatalf("initial btn1 Background = %v, want focus %v", got, cFocus)
	}

	// Disabled button starts with Disabled color.
	roDis := env.RenderObject(btnDisabled)
	if got := roDis.ComputedStyle().Background; got != cDisabled {
		t.Fatalf("btnDisabled Background = %v, want disabled %v", got, cDisabled)
	}

	// Tab moves focus to btn2; btn1 should revert to base.
	env.KeyPress("tab")
	env.Flush()
	if got := ro1.ComputedStyle().Background; got != cBase {
		t.Fatalf("after tab away btn1 Background = %v, want base %v", got, cBase)
	}
	ro2 := env.RenderObject(btn2)
	if got := ro2.ComputedStyle().Background; got != cFocus {
		t.Fatalf("after tab to btn2 Background = %v, want focus %v", got, cFocus)
	}

	// Hover over btn1 (inside its bounds).
	rect1, ok := btn1.GetBoundingClientRect()
	if !ok {
		t.Fatal("btn1 bounds not found")
	}
	env.MouseMove(rect1.Origin.X+1, rect1.Origin.Y+1)
	env.Flush()
	if got := ro1.ComputedStyle().Background; got != cHover {
		t.Fatalf("hover btn1 Background = %v, want hover %v", got, cHover)
	}

	// Leave btn1 to empty area far away -> reverts to base.
	env.MouseMove(38, 11)
	env.Flush()
	if got := ro1.ComputedStyle().Background; got != cBase {
		t.Fatalf("mouse leave btn1 Background = %v, want base %v", got, cBase)
	}

	// Primary mouse down on btn1 activates both :focus and :active (declared last -> cActive).
	env.MouseDown(rect1.Origin.X+1, rect1.Origin.Y+1, event.ButtonLeft)
	env.Flush()
	if got := ro1.ComputedStyle().Background; got != cActive {
		t.Fatalf("mousedown btn1 Background = %v, want active %v", got, cActive)
	}

	// Move pointer completely outside target (empty area) and release -> :active must clear!
	env.MouseMove(38, 11)
	env.MouseUp(38, 11, event.ButtonLeft)
	env.Flush()
	if got := ro1.ComputedStyle().Background; got != cFocus {
		t.Fatalf("off-target mouseup btn1 Background = %v, want focus %v (active cleared)", got, cFocus)
	}
}
