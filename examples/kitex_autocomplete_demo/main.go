package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/kitex/kitexdt"
	"github.com/masterkeysrd/kite/geom"
	"github.com/masterkeysrd/kite/style"
)

var (
	rootStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			JustifyContent(style.JustifyCenter).
			AlignItems(style.AlignCenter).
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Background(style.Hex("#11141c"))

	cardStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Width(style.Cells(58)).
			Background(style.Hex("#1c202c")).
			Border(style.DoubleBorder().Color(style.Hex("#56658c"))).
			Padding(1, 2)

	titleStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.Hex("#c9d6ff")).
			Bold(true)

	bodyTextStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.Hex("#96a0be"))

	inputStyle = style.S().
			Width(style.Percent(100)).
			Padding(0, 1).
			Border(style.SingleBorder().Color(style.Hex("#586896"))).
			Background(style.Hex("#141822")).
			Foreground(style.Hex("#ebeeff"))

	hintRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			FlexWrap(style.FlexWrapOn).
			Margin(1, 0, 0, 0)

	hintBadgeStyle = style.S().
			Margin(0, 1, 0, 0).
			Padding(0, 1).
			Background(style.Hex("#3d4763")).
			Foreground(style.Hex("#e0e6ff"))

	hintTextStyle = style.S().
			Margin(0, 1, 0, 0).
			Foreground(style.Hex("#8992af"))

	statusStyle = style.S().
			Margin(1, 0, 0, 0).
			Foreground(style.Hex("#7bcda5"))

	overlayCardStyle = style.S().
				Width(style.Cells(42)).
				Padding(0, 1).
				Border(style.SingleBorder().Color(style.Hex("#6c7cab"))).
				Background(style.Hex("#181c26"))

	menuTitleStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.Hex("#b0bcdc")).
			Bold(true)

	menuListStyle = style.S().
			ListStyleType(style.ListStyleNone).
			Margin(0).
			Padding(0)

	menuRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Margin(0, 0, 1, 0).
			Padding(0, 1)

	menuRowActiveStyle = style.S().
				Background(style.Hex("#3f5491")).
				Foreground(style.Hex("#f5f8ff"))

	menuRowIdleStyle = style.S().
				Background(style.Hex("#212634")).
				Foreground(style.Hex("#e2e7f8"))

	menuDetailStyle = style.S().Foreground(style.Hex("#8e97b2"))

	noMatchStyle = style.S().
			Padding(0, 1).
			Foreground(style.Hex("#c49797"))
)

type suggestion struct {
	Label  string
	Detail string
}

var commandSuggestions = []suggestion{
	{Label: "git status", Detail: "Show the current worktree status."},
	{Label: "git switch -c feature/autocomplete", Detail: "Create and switch to a new branch."},
	{Label: "go test ./...", Detail: "Run the full Go test suite."},
	{Label: "go test ./extras/kitex", Detail: "Run the Kitex package tests."},
	{Label: "npm run dev", Detail: "Start the local development server."},
	{Label: "docker compose up", Detail: "Start the local service stack."},
	{Label: "kubectl get pods", Detail: "Inspect the current Kubernetes pods."},
	{Label: "gh pr create", Detail: "Open a pull request from the current branch."},
}

func clampIndex(idx, length int) int {
	if length <= 0 {
		return 0
	}
	if idx < 0 {
		return 0
	}
	if idx >= length {
		return length - 1
	}
	return idx
}

func autocompleteMenuNode(filtered []suggestion, selectedIndex int, applySuggestion func(suggestion)) kitex.Node {
	return kitex.Box(kitex.BoxProps{
		Style: overlayCardStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: menuTitleStyle,
		}, kitex.Text("Autocomplete")),
		kitex.IfElse(len(filtered) == 0,
			kitex.Box(kitex.BoxProps{
				Style: noMatchStyle,
			}, kitex.Text("No matches for the current query.")),
			kitex.UL(kitex.ULProps{
				Style: menuListStyle,
			},
				kitex.Map(filtered, func(item suggestion, idx int) kitex.Node {
					rowStyle := menuRowStyle.
						Merge(menuRowIdleStyle).
						When(idx == selectedIndex, menuRowActiveStyle)
					return kitex.LI(kitex.LIProps{
						Key:   fmt.Sprintf("%s-%d", item.Label, idx),
						Style: rowStyle,
						OnClick: func(*event.MouseEvent) {
							applySuggestion(item)
						},
					},
						kitex.Text(item.Label),
						kitex.Box(kitex.BoxProps{
							Style: menuDetailStyle,
						}, kitex.Text(item.Detail)),
					)
				}),
			),
		),
	)
}

var App = kitex.SimpleFC("AutocompleteApp", func() kitex.Node {
	inputRef := kitex.UseRef[dom.Element](nil)
	getQuery, setQuery := kitex.UseState("")
	getMenuOpen, setMenuOpen := kitex.UseState(false)
	getSelectedIndex, setSelectedIndex := kitex.UseState(0)

	filtered := kitex.UseMemo(func() []suggestion {
		query := strings.TrimSpace(strings.ToLower(getQuery()))
		matches := make([]suggestion, 0, len(commandSuggestions))
		for _, item := range commandSuggestions {
			label := strings.ToLower(item.Label)
			detail := strings.ToLower(item.Detail)
			if query == "" || strings.Contains(label, query) || strings.Contains(detail, query) {
				matches = append(matches, item)
			}
			if len(matches) == 5 {
				break
			}
		}
		return matches
	}, []any{getQuery()})

	selectedIndex := clampIndex(getSelectedIndex(), len(filtered))

	isInputFocused := func() bool {
		if inputRef.Current == nil {
			return false
		}
		doc := inputRef.Current.OwnerDocument()
		return doc != nil && doc.CurrentFocus() == inputRef.Current
	}

	applySuggestion := func(item suggestion) {
		setMenuOpen(false)
		setSelectedIndex(0)
		setQuery(item.Label)
		if inputRef.Current != nil {
			if doc := inputRef.Current.OwnerDocument(); doc != nil {
				doc.Focus(inputRef.Current)
			}
		}
	}

	kitex.UseLayoutEffect(func() {
		if inputRef.Current != nil {
			if doc := inputRef.Current.OwnerDocument(); doc != nil {
				doc.Focus(inputRef.Current)
			}
		}
	}, []any{})

	kitex.UseEffect(func() {
		if !getMenuOpen() {
			return
		}
		if len(filtered) == 0 {
			setSelectedIndex(0)
			return
		}
		if idx := getSelectedIndex(); idx != selectedIndex {
			setSelectedIndex(selectedIndex)
		}
	}, []any{getMenuOpen(), len(filtered), getSelectedIndex(), selectedIndex})

	statusText := "Ctrl+Space opens autocomplete for the focused input."
	if len(filtered) > 0 && getMenuOpen() {
		statusText = fmt.Sprintf("Preview: %s", filtered[selectedIndex].Label)
	}

	menu := kitex.Empty()
	if getMenuOpen() && inputRef.Current != nil {
		menu = kitex.Overlay(kitex.OverlayProps{
			Anchor:    inputRef.Current,
			Placement: geom.PlacementBottom,
			Flip:      true,
			ZIndex:    100,
		}, autocompleteMenuNode(filtered, selectedIndex, applySuggestion))
	}

	return kitex.Box(kitex.BoxProps{
		Style: rootStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: cardStyle,
		},
			kitex.Box(kitex.BoxProps{
				Style: titleStyle,
			}, kitex.Text("Kitex Autocomplete Demo")),
			kitex.Box(kitex.BoxProps{
				Style: bodyTextStyle,
			}, kitex.Text("Type to filter shell-style commands. Press Ctrl+Space while the input is focused to open the autocomplete menu.")),
			kitex.Input(kitex.InputProps{
				Ref:         inputRef,
				Value:       getQuery(),
				Placeholder: "Try: git, go, docker, kubectl...",
				Style:       inputStyle,
				OnValueChange: func(val string) {
					if getMenuOpen() {
						setSelectedIndex(0)
					}
					setQuery(val)
				},
				OnKeyDown: func(ke *event.KeyEvent) {
					if !isInputFocused() {
						return
					}

					switch {
					case ke.MatchString("ctrl+space"):
						setMenuOpen(!getMenuOpen())
						setSelectedIndex(0)
						ke.PreventDefault()
						ke.StopPropagation()
					case getMenuOpen() && ke.MatchString("escape"):
						setMenuOpen(false)
						ke.PreventDefault()
						ke.StopPropagation()
					case getMenuOpen() && len(filtered) > 0 && ke.MatchString("down"):
						setSelectedIndex((selectedIndex + 1) % len(filtered))
						ke.PreventDefault()
						ke.StopPropagation()
					case getMenuOpen() && len(filtered) > 0 && ke.MatchString("up"):
						next := selectedIndex - 1
						if next < 0 {
							next = len(filtered) - 1
						}
						setSelectedIndex(next)
						ke.PreventDefault()
						ke.StopPropagation()
					case getMenuOpen() && len(filtered) > 0 && ke.MatchString("enter"):
						applySuggestion(filtered[selectedIndex])
						ke.PreventDefault()
						ke.StopPropagation()
					}
				},
			}),
			menu,
			kitex.Box(kitex.BoxProps{
				Style: hintRowStyle,
			},
				kitex.Span(kitex.SpanProps{Style: hintBadgeStyle}, kitex.Text("Ctrl+Space")),
				kitex.Span(kitex.SpanProps{Style: hintTextStyle}, kitex.Text("open menu")),
				kitex.Span(kitex.SpanProps{Style: hintBadgeStyle}, kitex.Text("Up/Down")),
				kitex.Span(kitex.SpanProps{Style: hintTextStyle}, kitex.Text("move selection")),
				kitex.Span(kitex.SpanProps{Style: hintBadgeStyle}, kitex.Text("Enter")),
				kitex.Span(kitex.SpanProps{Style: hintTextStyle}, kitex.Text("apply suggestion")),
				kitex.Span(kitex.SpanProps{Style: hintBadgeStyle}, kitex.Text("Esc")),
				kitex.Span(kitex.SpanProps{Style: hintTextStyle}, kitex.Text("close menu")),
			),
			kitex.Box(kitex.BoxProps{
				Style: statusStyle,
			}, kitex.Text(statusText)),
		),
	)
})

func main() {
	f, _ := os.Create("kitex_autocomplete_demo.log")
	defer f.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(f, nil)))

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{})
	container := element.NewBox(eng.Document()).Style(
		style.S().
			Width(style.Percent(100)).
			Height(style.Percent(100)),
	)
	eng.Mount(container)

	kitex.EnableDevMode = true
	if insp, err := devtools.Install(eng, devtools.Options{}); err == nil {
		kitexdt.Register(insp)
	}

	kitex.Render(App(), container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok && (ke.MatchString("q") || ke.MatchString("ctrl+c")) {
			cancel()
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
