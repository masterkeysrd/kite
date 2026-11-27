package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/masterkeysrd/kite/animation"
	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

var (
	stageStyle = style.S().
			Display(style.DisplayFlex).
			JustifyContent(style.JustifyCenter).
			AlignItems(style.AlignCenter).
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Border(style.SingleBorder().Color(style.RGB(60, 60, 70))).
			Background(style.RGB(20, 20, 25))

	titleStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.RGB(0, 255, 200)).
			Bold(true).
			TextAlign(style.TextAlignCenter)

	sectionLabelStyle = style.S().
				Margin(0, 0, 0, 1).
				Bold(true)

	buttonRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Margin(0, 0, 1, 0)

	spacerStyle = style.S().Flex(1)

	statusTextStyle = style.S().
			Foreground(style.RGB(255, 200, 0)).
			Bold(true)

	sidebarStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Width(style.Cells(45)).
			Height(style.Percent(100)).
			Padding(1, 2).
			Border(style.SingleBorder().Color(style.RGB(80, 80, 90))).
			Background(style.RGB(25, 25, 35))

	separatorStyle = style.S().Width(style.Cells(2))

	stageContainerStyle = style.S().
				Display(style.DisplayFlex).
				Flex(1).
				Height(style.Percent(100))

	mainLayoutStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			JustifyContent(style.JustifyBetween).
			Flex(1).
			Width(style.Percent(100))

	instructionsStyle = style.S().
				Margin(1, 0, 0, 0).
				Foreground(style.RGB(130, 130, 140)).
				TextAlign(style.TextAlignCenter)

	rootStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Padding(1, 2).
			Background(style.RGB(15, 15, 20))
)

// GroupAnimator runs multiple animations in parallel.
type GroupAnimator struct {
	Animators  []animation.Animator
	OnComplete func()
}

var _ animation.Animator = (*GroupAnimator)(nil)

func (g *GroupAnimator) Tick(dt time.Duration) bool {
	allFinished := true
	for i, v := range slices.Backward(g.Animators) {
		if v.Tick(dt) {
			g.Animators = append(g.Animators[:i], g.Animators[i+1:]...)
		} else {
			allFinished = false
		}
	}
	if allFinished {
		if g.OnComplete != nil {
			g.OnComplete()
		}
		return true
	}
	return false
}

// Interactive application state
var (
	eng *engine.Engine

	// Selected configuration
	selectedEasing       animation.EasingFunction = animation.EaseInOutCubic
	selectedEasingName                            = "EaseInOutCubic"
	selectedDuration     time.Duration            = 1 * time.Second
	selectedDurationName                          = "1s"
	selectedPropertyName                          = "All"

	// Animation target state variables
	targetWidth              = 20
	targetHeight             = 3
	targetColor  color.Color = style.RGB(138, 43, 226) // Purple

	// Run loop/stage state
	isAnimating = false
	forward     = true // true = animate from start to end, false = end to start

	// UI element pointers for dynamic styling and updates
	easingBtns   = make(map[string]*element.ButtonElement)
	durationBtns = make(map[string]*element.ButtonElement)
	propertyBtns = make(map[string]*element.ButtonElement)
	triggerBtn   *element.ButtonElement
	statusText   *element.TextElement
	targetBox    *element.BoxElement
)

func main() {
	f, _ := os.Create("kite-animation.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	_ = logger // prevent unused variable error
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := runWithBackend(ctx, b, logger, cancel); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}

func runWithBackend(ctx context.Context, b backend.Backend, logger *slog.Logger, cancel context.CancelFunc) error {
	eng = engine.New(b, engine.Options{})

	// 1. Create control buttons and register click/focus listeners
	easings := []struct {
		name string
		fn   animation.EasingFunction
	}{
		{"Linear", animation.Linear},
		{"EaseInQuad", animation.EaseInQuad},
		{"EaseOutQuad", animation.EaseOutQuad},
		{"EaseInOutCubic", animation.EaseInOutCubic},
	}
	for _, es := range easings {
		name := es.name
		fn := es.fn
		btn := element.Button(" " + name + " ")
		btn.OnEvent(event.EventClick, func(e event.Event) {
			if isAnimating {
				return
			}
			selectedEasing = fn
			selectedEasingName = name
			refreshControls()
		})
		btn.OnEvent(event.EventFocus, func(e event.Event) { refreshControls() })
		btn.OnEvent(event.EventBlur, func(e event.Event) { refreshControls() })
		easingBtns[name] = btn
	}

	durations := []struct {
		name string
		val  time.Duration
	}{
		{"500ms", 500 * time.Millisecond},
		{"1s", 1 * time.Second},
		{"2s", 2 * time.Second},
		{"3s", 3 * time.Second},
	}
	for _, d := range durations {
		name := d.name
		val := d.val
		btn := element.Button(" " + name + " ")
		btn.OnEvent(event.EventClick, func(e event.Event) {
			if isAnimating {
				return
			}
			selectedDuration = val
			selectedDurationName = name
			refreshControls()
		})
		btn.OnEvent(event.EventFocus, func(e event.Event) { refreshControls() })
		btn.OnEvent(event.EventBlur, func(e event.Event) { refreshControls() })
		durationBtns[name] = btn
	}

	properties := []string{"Width", "Height", "Color", "All"}
	for _, p := range properties {
		name := p
		btn := element.Button(" " + name + " ")
		btn.OnEvent(event.EventClick, func(e event.Event) {
			if isAnimating {
				return
			}
			selectedPropertyName = name
			refreshControls()
		})
		btn.OnEvent(event.EventFocus, func(e event.Event) { refreshControls() })
		btn.OnEvent(event.EventBlur, func(e event.Event) { refreshControls() })
		propertyBtns[name] = btn
	}

	// Trigger button
	triggerBtn = element.Button(getTriggerText())
	triggerBtn.OnEvent(event.EventClick, func(e event.Event) {
		if isAnimating {
			return
		}
		isAnimating = true
		triggerBtn.SetData(" Animating... ")
		refreshControls()

		var anims []animation.Animator

		// Create individual Property Tweens
		if selectedPropertyName == "Width" || selectedPropertyName == "All" {
			var start, end int
			if forward {
				start, end = 20, 60
			} else {
				start, end = 60, 20
			}
			wTween := animation.NewTween(start, end, selectedDuration, selectedEasing, animation.IntInterpolator, func(w int) {
				targetWidth = w
				updateTargetStyle()
			})
			anims = append(anims, wTween)
		}

		if selectedPropertyName == "Height" || selectedPropertyName == "All" {
			var start, end int
			if forward {
				start, end = 3, 10
			} else {
				start, end = 10, 3
			}
			hTween := animation.NewTween(start, end, selectedDuration, selectedEasing, animation.IntInterpolator, func(h int) {
				targetHeight = h
				updateTargetStyle()
			})
			anims = append(anims, hTween)
		}

		if selectedPropertyName == "Color" || selectedPropertyName == "All" {
			var start, end color.Color
			c1 := style.RGB(138, 43, 226) // Purple
			c2 := style.RGB(0, 206, 209)  // Cyan
			if forward {
				start, end = c1, c2
			} else {
				start, end = c2, c1
			}
			cTween := animation.NewTween(start, end, selectedDuration, selectedEasing, animation.ColorInterpolator, func(c color.Color) {
				targetColor = c
				updateTargetStyle()
			})
			anims = append(anims, cTween)
		}

		statusText.SetData(fmt.Sprintf("Status: Animating (%s, %s, %s)", selectedPropertyName, selectedEasingName, selectedDurationName))

		group := &GroupAnimator{
			Animators: anims,
			OnComplete: func() {
				isAnimating = false
				forward = !forward
				triggerBtn.SetData(getTriggerText())
				statusText.SetData(fmt.Sprintf("Status: Idle (At %s)", getTargetStateName()))
				refreshControls()
			},
		}

		eng.RegisterAnimation(group)
	})
	triggerBtn.OnEvent(event.EventFocus, func(e event.Event) { refreshControls() })
	triggerBtn.OnEvent(event.EventBlur, func(e event.Event) { refreshControls() })

	statusText = element.Text("Status: Idle (At Start)")

	// 2. Create target box
	targetBox = element.Box("  Kite Engine  ")
	updateTargetStyle()

	// 3. Create Stage container centering targetBox
	stage := element.Box(targetBox).Style(stageStyle)

	// 4. Assemble main layout
	root := element.Box(
		// Title
		element.Box(" ⚡ KITE INTERACTIVE ANIMATION SHOWCASE ⚡ ").Style(titleStyle),

		// Split workspace
		element.Box(
			// Left Panel: Controls
			element.Box(
				element.Box("Easing Function:").Style(sectionLabelStyle),
				element.Box(
					easingBtns["Linear"],
					easingBtns["EaseInQuad"],
					easingBtns["EaseOutQuad"],
					easingBtns["EaseInOutCubic"],
				).Style(buttonRowStyle),

				element.Box("Duration:").Style(sectionLabelStyle),
				element.Box(
					durationBtns["500ms"],
					durationBtns["1s"],
					durationBtns["2s"],
					durationBtns["3s"],
				).Style(buttonRowStyle),

				element.Box("Animate Property:").Style(sectionLabelStyle),
				element.Box(
					propertyBtns["Width"],
					propertyBtns["Height"],
					propertyBtns["Color"],
					propertyBtns["All"],
				).Style(buttonRowStyle),

				triggerBtn,

				element.Box("").Style(spacerStyle), // Push status text down

				element.Box(statusText).Style(statusTextStyle),
			).Style(sidebarStyle),

			// Spacer
			element.Box("").Style(separatorStyle),

			// Right Panel: Stage
			element.Box(
				stage,
			).Style(stageContainerStyle),
		).Style(mainLayoutStyle),

		// Footer instructions
		element.Box("Instructions: Tab / Shift+Tab to navigate controls. Space or Enter to select/run. Press 'q' to quit.").Style(instructionsStyle),
	).Style(rootStyle)

	eng.Mount(root)

	// Apply initial control selections styling
	refreshControls()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
		}
	})

	devtools.Install(eng, devtools.Options{})

	return eng.Run(ctx)
}

// updateTargetStyle modifies style properties of the animated box
func updateTargetStyle() {
	targetBox.Style(
		style.S().
			JustifyContent(style.JustifyCenter).
			AlignItems(style.AlignCenter).
			Width(style.Cells(targetWidth)).
			Height(style.Cells(targetHeight)).
			Border(style.DoubleBorder().Color(style.RGB(220, 220, 220))).
			Background(targetColor).
			Foreground(color.White).
			Bold(true),
	)
	if eng != nil {
		eng.RequestFrame()
	}
}

// refreshControls updates control button styles dynamically
func refreshControls() {
	focused := eng.FocusedTarget()

	// Easing buttons
	for name, btn := range easingBtns {
		selected := (name == selectedEasingName)
		isFocused := (focused == btn)
		applyButtonStyle(btn, selected, isFocused)
	}

	// Duration buttons
	for d, btn := range durationBtns {
		selected := (d == selectedDurationName)
		isFocused := (focused == btn)
		applyButtonStyle(btn, selected, isFocused)
	}

	// Property buttons
	for p, btn := range propertyBtns {
		selected := (p == selectedPropertyName)
		isFocused := (focused == btn)
		applyButtonStyle(btn, selected, isFocused)
	}

	// Trigger button
	isTriggerFocused := (focused == triggerBtn)
	applyTriggerButtonStyle(triggerBtn, isTriggerFocused)

	eng.RequestFrame()
}

func applyButtonStyle(btn *element.ButtonElement, selected bool, focused bool) {
	bg := style.RGB(35, 35, 45)
	fg := style.RGB(200, 200, 200)
	borderCol := style.RGB(70, 70, 80)

	if selected {
		bg = style.RGB(0, 200, 150) // Teal-Green for selected
		fg = color.Black
		borderCol = style.RGB(0, 255, 200)
	} else if focused {
		bg = style.RGB(50, 50, 65)
		fg = color.White
		borderCol = style.RGB(255, 165, 0) // Orange focus ring
	}

	btn.Style(
		style.S().
			Margin(0, 1, 0, 0).
			Padding(0, 1).
			Border(style.SingleBorder().Color(borderCol)).
			Background(bg).
			Foreground(fg),
	)
}

func applyTriggerButtonStyle(btn *element.ButtonElement, focused bool) {
	bg := style.RGB(255, 69, 0) // Red-Orange
	fg := color.Color(color.White)
	borderCol := style.RGB(255, 120, 0)

	if isAnimating {
		bg = style.RGB(60, 60, 70)
		fg = style.RGB(150, 150, 150)
		borderCol = style.RGB(90, 90, 100)
	} else if focused {
		bg = style.RGB(255, 99, 71)        // Tomato
		borderCol = style.RGB(255, 215, 0) // Gold focus ring
	}

	btn.Style(
		style.S().
			Margin(1, 0, 1, 0).
			Padding(0, 3).
			Border(style.DoubleBorder().Color(borderCol)).
			Background(bg).
			Foreground(fg).
			Bold(true).
			TextAlign(style.TextAlignCenter),
	)
}

func getTriggerText() string {
	if forward {
		return " ▶ Run Animation (Forward) "
	}
	return " ◀ Run Animation (Reverse) "
}

func getTargetStateName() string {
	if forward {
		return "Start"
	}
	return "End"
}
