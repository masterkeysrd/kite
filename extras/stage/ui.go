package stage

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/style"
)

// UI styling definitions using standard color.RGBA and correct Border APIs
var (
	sidebarStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Width(style.Percent(25)).
			Height(style.Percent(100)).
			MinHeight(style.Cells(0)).
			BorderRight(true, style.BorderSingle, color.RGBA{R: 55, G: 65, B: 81, A: 255}). // Gray 700
			Background(color.RGBA{R: 17, G: 24, B: 39, A: 255}).                            // Gray 900
			Padding(1)

	playableAreaStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Width(style.Percent(75)).
				Height(style.Percent(100)).
				MinHeight(style.Cells(0))

	canvasStyle = style.S().
			Display(style.DisplayFlex).
			AlignItems(style.AlignCenter).
			JustifyContent(style.JustifyCenter).
			Width(style.Percent(100)).
			Height(style.Percent(70)).
			MinHeight(style.Cells(0)).
			Background(color.RGBA{R: 31, G: 41, B: 55, A: 255}) // Gray 800

	canvasInnerStyle = style.S().
				Display(style.DisplayFlex).
				AlignItems(style.AlignCenter).
				JustifyContent(style.JustifyCenter).
				Width(style.Percent(90)).
				Height(style.Percent(90)).
				MinHeight(style.Cells(0)).
				Border(true, style.BorderSingle, color.RGBA{R: 75, G: 85, B: 99, A: 255}) // Gray 600

	detailPanelStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				Width(style.Percent(100)).
				Height(style.Percent(30)).
				MinHeight(style.Cells(0)).
				BorderTop(true, style.BorderSingle, color.RGBA{R: 55, G: 65, B: 81, A: 255}). // Gray 700
				Background(color.RGBA{R: 17, G: 24, B: 39, A: 255})                           // Gray 900

	controlsPanelStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Width(style.Percent(50)).
				Height(style.Percent(100)).
				MinHeight(style.Cells(0)).
				BorderRight(true, style.BorderSingle, color.RGBA{R: 55, G: 65, B: 81, A: 255}). // Gray 700
				Padding(1)

	logsPanelStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Width(style.Percent(50)).
			Height(style.Percent(100)).
			MinHeight(style.Cells(0)).
			Padding(1)
)

func renderUI(stg *Stage) kitex.Node {
	return StageApp(StageAppProps{Stage: stg})
}

// StageAppProps is the properties for the master StageApp component.
type StageAppProps struct {
	Stage *Stage
}

// PlayableAreaProps is the properties for the PlayableArea component.
type PlayableAreaProps struct {
	Key       string
	Scene     *Scene
	Decorator func(kitex.Node) kitex.Node
}

// PlayableArea manages the active scene hooks, rendering, knobs context, and details panels.
// Because it is keyed by the active scene index, all hooks and knob values are cleanly destroyed
// and recreated when the active scene changes.
var PlayableArea = kitex.FC("PlayableArea", func(props PlayableAreaProps) kitex.Node {
	// 1. Scene state (resets automatically when scene key changes)
	knobValues, setKnobValues := kitex.UseState(make(map[string]any))
	logs, setLogs := kitex.UseState([]ActionLog{})

	// 2. Helper to update a knob value
	setKnobValue := func(name string, val any) {
		prev := knobValues()
		next := make(map[string]any)
		for k, v := range prev {
			next[k] = v
		}
		next[name] = val
		setKnobValues(next)
	}

	// 3. Render the active scene inside a context
	sceneContext := NewContext(knobValues(), setKnobValue, nil)
	sceneContext.onLogAdded = func() {
		setLogs(sceneContext.Logs())
	}
	renderedScene := props.Scene.Render(sceneContext)
	if renderedScene != nil && props.Decorator != nil {
		renderedScene = props.Decorator(renderedScene)
	}

	// 4. Render controls panel rows
	var controlRows []kitex.Node
	ctrls := sceneContext.Controls()
	sort.Slice(ctrls, func(i, j int) bool {
		return ctrls[i].Name < ctrls[j].Name
	})

	for _, ctrl := range ctrls {
		var inputWidget kitex.Node
		switch ctrl.Type {
		case ControlTypeText:
			currentStr := ""
			if val, ok := knobValues()[ctrl.Name]; ok {
				currentStr, _ = val.(string)
			} else if ctrl.Default != nil {
				currentStr, _ = ctrl.Default.(string)
			}
			capturedName := ctrl.Name
			inputWidget = kitex.Input(kitex.InputProps{
				Value: currentStr,
				Style: style.S().
					Width(style.Percent(100)).
					Background(color.RGBA{R: 31, G: 41, B: 55, A: 255}). // Gray 800
					Padding(0, 1),
				OnChange: func(e event.Event) {
					if ie, ok := e.(*event.InputEvent); ok {
						setKnobValue(capturedName, ie.Value)
					} else if ce, ok := e.(*event.ChangeEvent); ok {
						setKnobValue(capturedName, ce.Value)
					}
				},
			})
		case ControlTypeBool:
			currentBool := false
			if val, ok := knobValues()[ctrl.Name]; ok {
				currentBool, _ = val.(bool)
			} else if ctrl.Default != nil {
				currentBool, _ = ctrl.Default.(bool)
			}
			capturedName := ctrl.Name
			checkboxLabel := "[ ]"
			if currentBool {
				checkboxLabel = "[x]"
			}
			inputWidget = kitex.Button(
				kitex.ButtonProps{
					Style: style.S().
						Border(false).
						Background(color.RGBA{R: 55, G: 65, B: 81, A: 255}). // Gray 700
						Padding(0, 0),
					OnClick: func(e event.Event) {
						setKnobValue(capturedName, !currentBool)
					},
				},
				kitex.Text(checkboxLabel),
			)
		case ControlTypeSelect:
			currentStr := ""
			if val, ok := knobValues()[ctrl.Name]; ok {
				currentStr, _ = val.(string)
			} else if ctrl.Default != nil {
				currentStr, _ = ctrl.Default.(string)
			}
			capturedName := ctrl.Name
			capturedOptions := ctrl.Options
			cycleOptionsText := fmt.Sprintf("%s ▾", currentStr)
			if currentStr == "" {
				cycleOptionsText = "select..."
			}
			inputWidget = kitex.Button(
				kitex.ButtonProps{
					Style: style.S().
						Border(false).
						Background(color.RGBA{R: 55, G: 65, B: 81, A: 255}). // Gray 700
						Padding(0, 1),
					OnClick: func(e event.Event) {
						nextIdx := 0
						for optIdx, opt := range capturedOptions {
							if opt == currentStr {
								nextIdx = (optIdx + 1) % len(capturedOptions)
								break
							}
						}
						if len(capturedOptions) > 0 {
							setKnobValue(capturedName, capturedOptions[nextIdx])
						}
					},
				},
				kitex.Text(cycleOptionsText),
			)
		case ControlTypeInt:
			currentInt := 0
			if val, ok := knobValues()[ctrl.Name]; ok {
				switch v := val.(type) {
				case int:
					currentInt = v
				case float64:
					currentInt = int(v)
				}
			} else if ctrl.Default != nil {
				switch v := ctrl.Default.(type) {
				case int:
					currentInt = v
				case float64:
					currentInt = int(v)
				}
			}
			capturedName := ctrl.Name
			stepperBtnStyle := style.S().
				Border(false).
				Background(color.RGBA{R: 55, G: 65, B: 81, A: 255}). // Gray 700
				Foreground(color.RGBA{R: 209, G: 213, B: 219, A: 255}).
				Padding(0, 1)
			inputWidget = kitex.Box(
				kitex.BoxProps{
					Style: style.S().
						Display(style.DisplayFlex).
						FlexDirection(style.FlexRow).
						AlignItems(style.AlignCenter).
						Width(style.Percent(100)),
				},
				kitex.Button(
					kitex.ButtonProps{
						Style: stepperBtnStyle,
						OnClick: func(e event.Event) {
							setKnobValue(capturedName, currentInt-1)
						},
					},
					kitex.Text("−"),
				),
				kitex.Box(
					kitex.BoxProps{
						Style: style.S().
							Flex(1).
							Background(color.RGBA{R: 31, G: 41, B: 55, A: 255}). // Gray 800
							Foreground(color.RGBA{R: 229, G: 231, B: 235, A: 255}).
							Padding(0, 1),
					},
					kitex.Text(strconv.Itoa(currentInt)),
				),
				kitex.Button(
					kitex.ButtonProps{
						Style: stepperBtnStyle,
						OnClick: func(e event.Event) {
							setKnobValue(capturedName, currentInt+1)
						},
					},
					kitex.Text("+"),
				),
			)
		}

		controlRows = append(controlRows, kitex.Box(
			kitex.BoxProps{
				Style: style.S().
					Display(style.DisplayFlex).
					FlexDirection(style.FlexRow).
					Width(style.Percent(100)).
					AlignItems(style.AlignCenter).
					MarginBottom(1),
			},
			kitex.Box(
				kitex.BoxProps{
					Style: style.S().
						Width(style.Percent(40)).
						Foreground(color.RGBA{R: 156, G: 163, B: 175, A: 255}),
				},
				kitex.Text(ctrl.Name),
			),
			kitex.Box(
				kitex.BoxProps{
					Style: style.S().
						Display(style.DisplayFlex).
						Width(style.Percent(60)),
				},
				inputWidget,
			),
		))
	}

	// 5. Render action logs rows
	var logRows []kitex.Node
	for _, l := range logs() {
		logRows = append(logRows, kitex.Box(
			kitex.BoxProps{
				Style: style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).Width(style.Percent(100)),
			},
			kitex.Box(
				kitex.BoxProps{
					Style: style.S().
						Width(style.Cells(10)).
						Foreground(color.RGBA{R: 107, G: 114, B: 128, A: 255}),
				},
				kitex.Text(l.Timestamp),
			),
			kitex.Box(
				kitex.BoxProps{
					Style: style.S().Foreground(color.RGBA{R: 209, G: 213, B: 219, A: 255}),
				},
				kitex.Text(l.Message),
			),
		))
	}

	return kitex.Box(
		kitex.BoxProps{
			Style: playableAreaStyle,
		},
		// Component Canvas area
		kitex.Box(
			kitex.BoxProps{
				Style: canvasStyle,
			},
			kitex.Box(
				kitex.BoxProps{
					Style: canvasInnerStyle,
				},
				func() kitex.Node {
					if renderedScene != nil {
						return renderedScene
					}
					return kitex.Text("Select a scene to start")
				}(),
			),
		),
		// Bottom Details row (Height: 30%)
		kitex.Box(
			kitex.BoxProps{
				Style: detailPanelStyle,
			},
			// Controls Panel
			kitex.Box(
				kitex.BoxProps{
					Style: controlsPanelStyle,
				},
				kitex.Box(
					kitex.BoxProps{
						Style: style.S().
							MarginBottom(1).
							BorderBottom(true, style.BorderSingle, color.RGBA{R: 75, G: 85, B: 99, A: 255}),
					},
					kitex.Text("🛠️ CONTROLS"),
				),
				kitex.Box(
					kitex.BoxProps{
						Style: style.S().Flex(1).Overflow(style.OverflowAuto),
					},
					controlRows...,
				),
			),
			// Action logs view
			kitex.Box(
				kitex.BoxProps{
					Style: logsPanelStyle,
				},
				kitex.Box(
					kitex.BoxProps{
						Style: style.S().
							Display(style.DisplayFlex).
							FlexDirection(style.FlexRow).
							JustifyContent(style.JustifyBetween).
							MarginBottom(1).
							BorderBottom(true, style.BorderSingle, color.RGBA{R: 75, G: 85, B: 99, A: 255}),
					},
					kitex.Text("📋 ACTION LOG"),
					kitex.Button(
						kitex.ButtonProps{
							Style: style.S().
								Border(false).
								Background(color.RGBA{R: 79, G: 70, B: 229, A: 255}). // Indigo 600
								Padding(0, 1),
							OnClick: func(e event.Event) {
								setLogs([]ActionLog{})
								if sceneContext != nil {
									sceneContext.ClearLogs()
								}
							},
						},
						kitex.Text("Clear"),
					),
				),
				kitex.Box(
					kitex.BoxProps{
						Style: style.S().Flex(1).Overflow(style.OverflowAuto),
					},
					logRows...,
				),
			),
		),
	)
})

// StageApp renders the interactive playground user interface.
var StageApp = kitex.FC("StageApp", func(props StageAppProps) kitex.Node {
	// 1. Collect and sort scenes
	type sceneRef struct {
		compName  string
		sceneName string
		index     int
	}
	var compNames []string
	for k := range props.Stage.components {
		compNames = append(compNames, k)
	}
	sort.Strings(compNames)

	var allScenes []sceneRef
	for _, compName := range compNames {
		scenes := props.Stage.components[compName]
		for idx, sc := range scenes {
			allScenes = append(allScenes, sceneRef{
				compName:  compName,
				sceneName: sc.Name,
				index:     idx,
			})
		}
	}

	// 2. State Hooks
	activeSceneIdx, setActiveSceneIdx := kitex.UseState(0)
	rootRef := kitex.UseRef[dom.Node](nil)

	// Active scene reference
	var activeScene *Scene
	if len(allScenes) > 0 && activeSceneIdx() >= 0 && activeSceneIdx() < len(allScenes) {
		ref := allScenes[activeSceneIdx()]
		activeScene = &props.Stage.components[ref.compName][ref.index]
	}

	// Render sidebar list items
	var sidebarItems []kitex.Node
	for idx, ref := range allScenes {
		isSelected := idx == activeSceneIdx()
		itemStyle := style.S().
			Padding(0, 1).
			Width(style.Percent(100))
		if isSelected {
			itemStyle = itemStyle.
				Background(color.RGBA{R: 79, G: 70, B: 229, A: 255}). // Indigo 600
				Foreground(color.RGBA{R: 255, G: 255, B: 255, A: 255})
		} else {
			itemStyle = itemStyle.
				Foreground(color.RGBA{R: 156, G: 163, B: 175, A: 255}) // Gray 400
		}

		capturedIdx := idx
		sidebarItems = append(sidebarItems, kitex.Box(kitex.BoxProps{
			Style: itemStyle,
			OnClick: func(e event.Event) {
				setActiveSceneIdx(capturedIdx)
			},
		}, kitex.Text(fmt.Sprintf("%s / %s", ref.compName, ref.sceneName))))
	}

	// Global document-level navigation keyboard hook
	kitex.UseEffectCleanup(func() func() {
		if rootRef.Current == nil {
			return nil
		}
		doc := rootRef.Current.OwnerDocument()
		if doc == nil {
			return nil
		}

		listener := func(e event.Event) {
			ke := e.(*event.KeyEvent)
			if target := e.Target(); target != nil {
				if el, ok := target.(dom.Element); ok && el.TagName() == "input" {
					return
				}
			}

			if ke.MatchString("up") {
				current := activeSceneIdx()
				if current > 0 {
					setActiveSceneIdx(current - 1)
				}
			} else if ke.MatchString("down") {
				current := activeSceneIdx()
				if current < len(allScenes)-1 {
					setActiveSceneIdx(current + 1)
				}
			}
		}

		sub := doc.AddEventListener(event.EventKeyDown, listener)
		return func() {
			sub.Cancel()
		}
	}, []any{len(allScenes), activeSceneIdx()})

	// Master Screen Layout
	return kitex.Box(kitex.BoxProps{
		Ref: rootRef,
		Style: style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow). // Layout: Sidebar | PlayableArea
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Background(color.RGBA{R: 31, G: 41, B: 55, A: 255}), // Gray 800
	},
		// Sidebar component list (takes 25% width, full height)
		kitex.Box(
			kitex.BoxProps{
				Style: sidebarStyle,
			},
			kitex.Box(
				kitex.BoxProps{
					Style: style.S().
						MarginBottom(1).
						BorderBottom(true, style.BorderSingle, color.RGBA{R: 75, G: 85, B: 99, A: 255}),
				},
				kitex.Text("📖 SCENES"),
			),
			kitex.Box(
				kitex.BoxProps{
					Style: style.S().Flex(1).Overflow(style.OverflowAuto),
				},
				sidebarItems...,
			),
		),
		// Playable area containing Canvas, Controls, Logs (takes 75% width, full height)
		func() kitex.Node {
			if activeScene != nil {
				key := fmt.Sprintf("scene-%d", activeSceneIdx())
				return PlayableArea(PlayableAreaProps{
					Key:       key,
					Scene:     activeScene,
					Decorator: props.Stage.decorator,
				})
			}
			return kitex.Box(
				kitex.BoxProps{
					Style: playableAreaStyle.Background(color.RGBA{R: 31, G: 41, B: 55, A: 255}),
				},
				kitex.Text("Select a scene to start"),
			)
		}(),
	)
})
