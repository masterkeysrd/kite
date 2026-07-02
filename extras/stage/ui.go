package stage

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
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

	toolbarStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			AlignItems(style.AlignCenter).
			Width(style.Percent(100)).
			Height(style.Cells(2)).
			BorderBottom(true, style.BorderSingle, color.RGBA{R: 55, G: 65, B: 81, A: 255}). // Gray 700
			Background(color.RGBA{R: 17, G: 24, B: 39, A: 255}).                             // Gray 900
			Padding(0, 1)

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
	Key          string
	Scene        *Scene
	Decorator    func(*Context, kitex.Node) kitex.Node
	Globals      []Control
	GlobalValues map[string]any
	SetGlobalVal func(string, any)
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

	// 3. Render the active scene inside a context configured with globals
	globalsMap := make(map[string]Control)
	for _, ctrl := range props.Globals {
		globalsMap[ctrl.Name] = ctrl
	}

	sceneContext := NewContext(knobValues(), setKnobValue, nil).
		WithGlobals(globalsMap, props.GlobalValues)

	sceneContext.onLogAdded = func() {
		setLogs(sceneContext.Logs())
	}
	renderedScene := props.Scene.Render(sceneContext)
	if renderedScene != nil && props.Decorator != nil {
		renderedScene = props.Decorator(sceneContext, renderedScene)
	}

	// 4. Render toolbar widgets (global controls)
	var toolbarWidgets []kitex.Node
	for _, ctrl := range props.Globals {
		var inputWidget kitex.Node
		switch ctrl.Type {
		case ControlTypeText:
			currentStr := ""
			if val, ok := props.GlobalValues[ctrl.Name]; ok {
				currentStr, _ = val.(string)
			} else if ctrl.Default != nil {
				currentStr, _ = ctrl.Default.(string)
			}
			capturedName := ctrl.Name
			inputWidget = kitex.Input(kitex.InputProps{
				Value: currentStr,
				Style: style.S().
					Width(style.Cells(15)).
					Background(color.RGBA{R: 31, G: 41, B: 55, A: 255}). // Gray 800
					Padding(0, 1),
				OnChange: func(e event.Event) {
					if ie, ok := e.(*event.InputEvent); ok {
						props.SetGlobalVal(capturedName, ie.Value)
					} else if ce, ok := e.(*event.ChangeEvent); ok {
						props.SetGlobalVal(capturedName, ce.Value)
					}
				},
			})
		case ControlTypeBool:
			currentBool := false
			if val, ok := props.GlobalValues[ctrl.Name]; ok {
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
						Padding(0, 1),
					OnClick: func(e event.Event) {
						props.SetGlobalVal(capturedName, !currentBool)
					},
				},
				kitex.Text(checkboxLabel),
			)
		case ControlTypeSelect:
			currentStr := ""
			if val, ok := props.GlobalValues[ctrl.Name]; ok {
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
							props.SetGlobalVal(capturedName, capturedOptions[nextIdx])
						}
					},
				},
				kitex.Text(cycleOptionsText),
			)
		case ControlTypeInt:
			currentInt := 0
			if val, ok := props.GlobalValues[ctrl.Name]; ok {
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
						Width(style.Cells(12)),
				},
				kitex.Button(
					kitex.ButtonProps{
						Style: stepperBtnStyle,
						OnClick: func(e event.Event) {
							props.SetGlobalVal(capturedName, currentInt-1)
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
							props.SetGlobalVal(capturedName, currentInt+1)
						},
					},
					kitex.Text("+"),
				),
			)
		}

		toolbarWidgets = append(toolbarWidgets, kitex.Box(
			kitex.BoxProps{
				Style: style.S().
					Display(style.DisplayFlex).
					FlexDirection(style.FlexRow).
					AlignItems(style.AlignCenter).
					MarginRight(2),
			},
			kitex.Box(
				kitex.BoxProps{
					Style: style.S().
						Foreground(color.RGBA{R: 156, G: 163, B: 175, A: 255}).
						MarginRight(1),
				},
				kitex.Text(ctrl.Name+":"),
			),
			inputWidget,
		))
	}

	// 5. Render controls panel rows
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

	// 6. Render action logs rows
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
		// Globals Toolbar (if any globals exist)
		func() kitex.Node {
			if len(props.Globals) > 0 {
				return kitex.Box(
					kitex.BoxProps{
						Style: toolbarStyle,
					},
					toolbarWidgets...,
				)
			}
			return nil
		}(),
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
	// 1. Trie structure to support nested sections
	type trieNode struct {
		Name     string
		FullPath string
		Children map[string]*trieNode
		Scenes   []Scene
		CompName string
	}

	normalizeComponentPath := func(compName string) []string {
		if compName == "" {
			return []string{"Default", "Default"}
		}
		parts := strings.Split(compName, "/")
		if len(parts) == 1 {
			return []string{"Default", parts[0]}
		} else if len(parts) == 2 {
			return []string{"Default", parts[0], parts[1]}
		}
		return parts
	}

	buildTrie := func(components map[string][]Scene) *trieNode {
		root := &trieNode{
			Children: make(map[string]*trieNode),
		}
		for compName, scenes := range components {
			parts := normalizeComponentPath(compName)
			curr := root
			var pathBuilder []string
			for _, part := range parts {
				pathBuilder = append(pathBuilder, part)
				fullPath := strings.Join(pathBuilder, "/")
				if curr.Children == nil {
					curr.Children = make(map[string]*trieNode)
				}
				child, ok := curr.Children[part]
				if !ok {
					child = &trieNode{
						Name:     part,
						FullPath: fullPath,
						Children: make(map[string]*trieNode),
					}
					curr.Children[part] = child
				}
				curr = child
			}
			curr.Scenes = scenes
			curr.CompName = compName
		}
		return root
	}

	rootTrieNode := buildTrie(props.Stage.components)

	type sceneRef struct {
		compName  string
		sceneName string
		index     int
	}

	var allScenes []sceneRef
	var traverseTree func(*trieNode)
	traverseTree = func(node *trieNode) {
		var childKeys []string
		for k := range node.Children {
			childKeys = append(childKeys, k)
		}
		sort.Strings(childKeys)
		for _, k := range childKeys {
			traverseTree(node.Children[k])
		}

		for i, sc := range node.Scenes {
			allScenes = append(allScenes, sceneRef{
				compName:  node.CompName,
				sceneName: sc.Name,
				index:     i,
			})
		}
	}
	traverseTree(rootTrieNode)

	// 2. State Hooks & Refs
	activeSceneIdx, setActiveSceneIdx := kitex.UseState(0)
	expandedComps, setExpandedComps := kitex.UseState(make(map[string]bool))
	globalValues, setGlobalValues := kitex.UseState(make(map[string]any))
	filterText, setFilterText := kitex.UseState("")
	rootRef := kitex.UseRef[dom.Node](nil)
	filterInputRef := kitex.UseRef[*element.InputElement](nil)

	// Trie filter matching helpers
	var nodeMatches func(*trieNode, string) bool
	nodeMatches = func(node *trieNode, query string) bool {
		if query == "" {
			return true
		}
		query = strings.ToLower(query)
		if strings.Contains(strings.ToLower(node.Name), query) {
			return true
		}
		for _, sc := range node.Scenes {
			if strings.Contains(strings.ToLower(sc.Name), query) {
				return true
			}
		}
		for _, child := range node.Children {
			if nodeMatches(child, query) {
				return true
			}
		}
		return false
	}

	type filteredSceneRef struct {
		sc    Scene
		index int
	}

	getFilteredScenes := func(node *trieNode, query string) []filteredSceneRef {
		var res []filteredSceneRef
		queryLower := strings.ToLower(query)
		compMatch := query == "" || strings.Contains(strings.ToLower(node.Name), queryLower)
		for i, sc := range node.Scenes {
			if compMatch || strings.Contains(strings.ToLower(sc.Name), queryLower) {
				res = append(res, filteredSceneRef{sc: sc, index: i})
			}
		}
		return res
	}

	isExpanded := func(path string, node *trieNode) bool {
		query := filterText()
		if query != "" {
			return nodeMatches(node, query)
		}

		m := expandedComps()
		if val, ok := m[path]; ok {
			return val
		}
		// Default expansion: expand components/folders along the path of the first scene
		if len(allScenes) > 0 {
			parts := normalizeComponentPath(allScenes[0].compName)
			firstCompNormalized := strings.Join(parts, "/")
			return path == firstCompNormalized || strings.HasPrefix(firstCompNormalized, path+"/")
		}
		return false
	}

	toggleComp := func(path string, node *trieNode) {
		m := expandedComps()
		next := make(map[string]bool)
		for k, v := range m {
			next[k] = v
		}
		next[path] = !isExpanded(path, node)
		setExpandedComps(next)
	}

	setGlobalVal := func(name string, val any) {
		prev := globalValues()
		next := make(map[string]any)
		for k, v := range prev {
			next[k] = v
		}
		next[name] = val
		setGlobalValues(next)
	}

	// Active scene reference
	var activeScene *Scene
	if len(allScenes) > 0 && activeSceneIdx() >= 0 && activeSceneIdx() < len(allScenes) {
		ref := allScenes[activeSceneIdx()]
		activeScene = &props.Stage.components[ref.compName][ref.index]
	}

	// Render sidebar list items as a nested collapsible tree
	var visibleIndices []int
	var sidebarItems []kitex.Node

	sceneKeyToIndex := make(map[string]int)
	for i, ref := range allScenes {
		key := fmt.Sprintf("%s:%d", ref.compName, ref.index)
		sceneKeyToIndex[key] = i
	}

	var renderNode func(*trieNode, int)
	renderNode = func(node *trieNode, depth int) {
		if node.FullPath == "" {
			var childKeys []string
			for k, child := range node.Children {
				if nodeMatches(child, filterText()) {
					childKeys = append(childKeys, k)
				}
			}
			sort.Strings(childKeys)
			for i, k := range childKeys {
				if i > 0 {
					// Add a visual separator or gap before this top-level category
					sidebarItems = append(sidebarItems, kitex.Box(kitex.BoxProps{
						Style: style.S().Height(style.Cells(1)),
					}))
				}
				renderNode(node.Children[k], 0)
			}
			return
		}

		expanded := isExpanded(node.FullPath, node)
		var icon string
		hasChildren := len(node.Children) > 0
		hasScenes := len(node.Scenes) > 0

		if depth == 0 {
			if hasChildren {
				if expanded {
					icon = "◇ " // Open Top-Level Folder
				} else {
					icon = "◆ " // Closed Top-Level Folder
				}
			} else {
				if expanded {
					icon = "✧ " // Open Top-Level Flat Component
				} else {
					icon = "✦ " // Closed Top-Level Flat Component
				}
			}
		} else {
			if hasChildren {
				if expanded {
					icon = "▼ " // Open Intermediate Folder
				} else {
					icon = "▶ " // Closed Intermediate Folder
				}
			} else if hasScenes {
				if expanded {
					icon = "⬡ " // Open Component Level
				} else {
					icon = "⬢ " // Closed Component Level
				}
			}
		}

		headerStyle := style.S().
			Padding(0, 1+depth*2).
			Width(style.Percent(100)).
			Bold(true)

		if hasChildren {
			headerStyle = headerStyle.Foreground(color.RGBA{R: 209, G: 213, B: 219, A: 255})
		} else {
			headerStyle = headerStyle.Foreground(color.RGBA{R: 229, G: 231, B: 235, A: 255})
		}

		capturedPath := node.FullPath
		capturedNode := node
		sidebarItems = append(sidebarItems, kitex.Box(kitex.BoxProps{
			Style: headerStyle,
			OnClick: func(e event.Event) {
				toggleComp(capturedPath, capturedNode)
			},
		}, kitex.Text(icon+node.Name)))

		if expanded {
			var childKeys []string
			for k, child := range node.Children {
				if nodeMatches(child, filterText()) {
					childKeys = append(childKeys, k)
				}
			}
			sort.Strings(childKeys)
			for _, k := range childKeys {
				renderNode(node.Children[k], depth+1)
			}

			for _, fsc := range getFilteredScenes(node, filterText()) {
				key := fmt.Sprintf("%s:%d", node.CompName, fsc.index)
				globalIdx, found := sceneKeyToIndex[key]
				if !found {
					continue
				}
				visibleIndices = append(visibleIndices, globalIdx)

				isSelected := globalIdx == activeSceneIdx()
				itemStyle := style.S().
					Padding(0, 1+(depth+1)*2+1).
					Width(style.Percent(100))
				if isSelected {
					itemStyle = itemStyle.
						Background(color.RGBA{R: 79, G: 70, B: 229, A: 255}).
						Foreground(color.RGBA{R: 255, G: 255, B: 255, A: 255})
				} else {
					itemStyle = itemStyle.
						Foreground(color.RGBA{R: 156, G: 163, B: 175, A: 255})
				}

				capturedIdx := globalIdx
				sidebarItems = append(sidebarItems, kitex.Box(kitex.BoxProps{
					Style: itemStyle,
					OnClick: func(e event.Event) {
						setActiveSceneIdx(capturedIdx)
					},
				}, kitex.Text("▪ "+fsc.sc.Name)))
			}
		}
	}
	renderNode(rootTrieNode, 0)

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

			// Focus filter input on "/"
			if ke.MatchString("/") {
				if filterInputRef.Current != nil {
					filterInputRef.Current.Focus()
				}
				e.PreventDefault()
				return
			}

			if ke.MatchString("up") {
				current := activeSceneIdx()
				pos := -1
				for i, v := range visibleIndices {
					if v == current {
						pos = i
						break
					}
				}
				if pos > 0 {
					setActiveSceneIdx(visibleIndices[pos-1])
				}
			} else if ke.MatchString("down") {
				current := activeSceneIdx()
				pos := -1
				for i, v := range visibleIndices {
					if v == current {
						pos = i
						break
					}
				}
				if pos >= 0 && pos < len(visibleIndices)-1 {
					setActiveSceneIdx(visibleIndices[pos+1])
				} else if pos == -1 && len(visibleIndices) > 0 {
					setActiveSceneIdx(visibleIndices[0])
				}
			}
		}

		sub := doc.AddEventListener(event.EventKeyDown, listener)
		return func() {
			sub.Cancel()
		}
	}, []any{len(allScenes), activeSceneIdx(), len(visibleIndices)})

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
			kitex.Input(kitex.InputProps{
				Ref:         filterInputRef,
				Value:       filterText(),
				Placeholder: "Filter (press /)...",
				Style: style.S().
					Width(style.Percent(100)).
					MarginBottom(1).
					Background(color.RGBA{R: 31, G: 41, B: 55, A: 255}).
					Foreground(color.RGBA{R: 243, G: 244, B: 246, A: 255}).
					Padding(0, 1),
				OnChange: func(e event.Event) {
					if ie, ok := e.(*event.InputEvent); ok {
						setFilterText(ie.Value)
					} else if ce, ok := e.(*event.ChangeEvent); ok {
						setFilterText(ce.Value)
					}
				},
				OnKeyDown: func(e event.Event) {
					ke := e.(*event.KeyEvent)
					if ke.MatchString("escape") || ke.MatchString("enter") {
						if filterInputRef.Current != nil {
							filterInputRef.Current.Blur()
						}
						e.PreventDefault()
					}
				},
			}),
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
					Key:          key,
					Scene:        activeScene,
					Decorator:    props.Stage.decorator,
					Globals:      props.Stage.globals,
					GlobalValues: globalValues(),
					SetGlobalVal: setGlobalVal,
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
