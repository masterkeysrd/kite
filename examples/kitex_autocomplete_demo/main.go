package main

import (
	"context"
	"fmt"
	"image/color"
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
			Background(color.RGBA{R: 17, G: 20, B: 28, A: 255})

	cardStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Width(style.Cells(58)).
			Background(color.RGBA{R: 28, G: 32, B: 44, A: 255}).
			Border(style.DoubleBorder().Color(color.RGBA{R: 86, G: 101, B: 140, A: 255})).
			Padding(1, 2)

	titleStyle = style.S().
			Bold(true).
			Foreground(color.RGBA{R: 201, G: 214, B: 255, A: 255}).
			Margin(0, 0, 1, 0)

	bodyTextStyle = style.S().
			Foreground(color.RGBA{R: 150, G: 160, B: 190, A: 255}).
			Margin(0, 0, 1, 0)

	inputStyle = style.S().
			Width(style.Percent(100)).
			Background(color.RGBA{R: 20, G: 24, B: 34, A: 255}).
			Foreground(color.RGBA{R: 235, G: 238, B: 255, A: 255}).
			Border(style.SingleBorder().Color(color.RGBA{R: 88, G: 104, B: 150, A: 255})).
			Padding(0, 1)

	hintRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			FlexWrap(style.FlexWrapOn).
			Margin(1, 0, 0, 0)

	hintBadgeStyle = style.S().
			Background(color.RGBA{R: 61, G: 71, B: 99, A: 255}).
			Foreground(color.RGBA{R: 224, G: 230, B: 255, A: 255}).
			Padding(0, 1).
			Margin(0, 1, 0, 0)

	hintTextStyle = style.S().
			Foreground(color.RGBA{R: 137, G: 146, B: 175, A: 255}).
			Margin(0, 1, 0, 0)

	statusStyle = style.S().
			Foreground(color.RGBA{R: 123, G: 205, B: 165, A: 255}).
			Margin(1, 0, 0, 0)

	overlayCardStyle = style.S().
				Width(style.Cells(42)).
				Background(color.RGBA{R: 24, G: 28, B: 38, A: 255}).
				Border(style.SingleBorder().Color(color.RGBA{R: 108, G: 124, B: 171, A: 255})).
				Padding(0, 1)

	menuTitleStyle = style.S().
			Foreground(color.RGBA{R: 176, G: 188, B: 220, A: 255}).
			Bold(true).
			Margin(0, 0, 1, 0)

	menuListStyle = style.S().
			ListStyleType(style.ListStyleNone).
			Padding(0).
			Margin(0)

	menuRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	menuRowActiveStyle = style.S().
				Background(color.RGBA{R: 63, G: 84, B: 145, A: 255}).
				Foreground(color.RGBA{R: 245, G: 248, B: 255, A: 255})

	menuRowIdleStyle = style.S().
				Background(color.RGBA{R: 33, G: 38, B: 52, A: 255}).
				Foreground(color.RGBA{R: 226, G: 231, B: 248, A: 255})

	menuDetailStyle = style.S().
			Foreground(color.RGBA{R: 142, G: 151, B: 178, A: 255})

	noMatchStyle = style.S().
			Foreground(color.RGBA{R: 196, G: 151, B: 151, A: 255}).
			Padding(0, 1)

	hostStyle = style.S().Width(style.Percent(100)).Height(style.Percent(100))
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
					rowStyle := menuRowStyle.Merge(menuRowIdleStyle)
					if idx == selectedIndex {
						rowStyle = rowStyle.Merge(menuRowActiveStyle)
					}
					return kitex.LI(kitex.LIProps{
						Key:   fmt.Sprintf("%s-%d", item.Label, idx),
						Style: rowStyle,
						OnClick: func(event.Event) {
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
				OnChange: func(e event.Event) {
					if ie, ok := e.(*event.InputEvent); ok {
						if getMenuOpen() {
							setSelectedIndex(0)
						}
						setQuery(ie.Value)
					}
				},
				OnKeyDown: func(e event.Event) {
					ke, ok := e.(*event.KeyEvent)
					if !ok {
						return
					}

					if !isInputFocused() {
						return
					}

					switch {
					case ke.MatchString("ctrl+space"):
						setMenuOpen(!getMenuOpen())
						setSelectedIndex(0)
						e.PreventDefault()
						e.StopPropagation()
					case getMenuOpen() && ke.MatchString("escape"):
						setMenuOpen(false)
						e.PreventDefault()
						e.StopPropagation()
					case getMenuOpen() && len(filtered) > 0 && ke.MatchString("down"):
						setSelectedIndex((selectedIndex + 1) % len(filtered))
						e.PreventDefault()
						e.StopPropagation()
					case getMenuOpen() && len(filtered) > 0 && ke.MatchString("up"):
						next := selectedIndex - 1
						if next < 0 {
							next = len(filtered) - 1
						}
						setSelectedIndex(next)
						e.PreventDefault()
						e.StopPropagation()
					case getMenuOpen() && len(filtered) > 0 && ke.MatchString("enter"):
						applySuggestion(filtered[selectedIndex])
						e.PreventDefault()
						e.StopPropagation()
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
	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{Logger: logger})

	container := element.NewBox(eng.Document())
	container.Style(hostStyle)
	eng.Mount(container)

	kitex.EnableDevMode = true
	kitex.Render(App(), container)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
		}
	})

	insp, _ := devtools.Install(eng, devtools.Options{})
	kitexdt.Register(insp)

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}
