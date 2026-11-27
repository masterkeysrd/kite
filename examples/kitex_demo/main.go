package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"math/rand"
	"os"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/extras/kitex/kitexdt"
	"github.com/masterkeysrd/kite/style"
)

var (
	itemRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			AlignItems(style.AlignCenter).
			Margin(0, 0, 1, 0).
			Padding(0, 1).
			Border(style.SingleBorder()).
			Background(style.RGB(35, 35, 50))

	itemLabelStyle = style.S().
			Width(style.Percent(40)).
			Foreground(style.RGB(220, 220, 230))

	incrementBtnStyle = style.S().
				Margin(0, 1).
				Background(style.RGB(60, 120, 220)).
				Foreground(color.White).
				Hover(style.S().Background(style.RGB(80, 140, 240)))

	deleteBtnStyle = style.S().
			Margin(0, 1).
			Background(style.RGB(200, 60, 60)).
			Foreground(color.White).
			Hover(style.S().Background(style.RGB(240, 80, 80)))

	appContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Width(style.Percent(100)).
				Height(style.Percent(100)).
				Padding(1, 2).
				Background(style.RGB(20, 20, 30))

	appTitleStyle = style.S().
			Margin(0, 0, 1, 0).
			Foreground(style.RGB(90, 140, 255)).
			Bold(true).
			TextAlign(style.TextAlignCenter)

	instructionsStyle = style.S().
				Margin(0, 0, 1, 0).
				Foreground(style.RGB(150, 150, 170))

	actionPanelStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				Margin(0, 0, 1, 0)

	addBtnStyle = style.S().
			Margin(0, 1).
			Background(style.RGB(50, 180, 100)).
			Foreground(color.White).
			Hover(style.S().Background(style.RGB(70, 210, 120)))

	reverseBtnStyle = style.S().
			Margin(0, 1).
			Background(style.RGB(160, 80, 220)).
			Foreground(color.White).
			Hover(style.S().Background(style.RGB(190, 100, 250)))

	shuffleBtnStyle = style.S().
			Margin(0, 1).
			Background(style.RGB(220, 130, 40)).
			Foreground(color.White).
			Hover(style.S().Background(style.RGB(250, 160, 60)))

	listContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Padding(1, 1).
				Border(style.DoubleBorder()).
				Background(style.RGB(25, 25, 38))
)

type ItemData struct {
	Key string
	ID  string
}

type ItemProps struct {
	Key      string
	ID       string
	OnDelete func()
}

// ListItem is a functional component representing a single row in the list.
// Interactive :hover states are handled natively by the style cascade without
// triggering VDOM re-renders.
var ListItem = kitex.FC("ListItem", func(props ItemProps) kitex.Node {
	getClicks, setClicks := kitex.UseState(0)

	return kitex.Box(kitex.BoxProps{
		Style: itemRowStyle,
	},
		kitex.Span(kitex.SpanProps{
			Style: itemLabelStyle,
		}, kitex.Textf("Item %s (Clicks: %d)", props.ID, getClicks())),

		kitex.Button(kitex.ButtonProps{
			OnPress: func() {
				setClicks.Update(func(n int) int { return n + 1 })
			},
			Style: incrementBtnStyle,
		}, kitex.Text(" +1 ")),

		kitex.Button(kitex.ButtonProps{
			OnPress: props.OnDelete,
			Style:   deleteBtnStyle,
		}, kitex.Text(" Delete ")),
	)
})

// App is the root functional component, maintaining list data and IDs.
var App = kitex.SimpleFC("App", func() kitex.Node {
	getItems, setItems := kitex.UseState([]ItemData{
		{Key: "1", ID: "A"},
		{Key: "2", ID: "B"},
		{Key: "3", ID: "C"},
	})
	getNextID, setNextID := kitex.UseState(4)

	handleDelete := func(key string) {
		setItems.Update(func(current []ItemData) []ItemData {
			nextItems := make([]ItemData, 0, len(current))
			for _, item := range current {
				if item.Key != key {
					nextItems = append(nextItems, item)
				}
			}
			return nextItems
		})
	}

	renderItem := func(item ItemData, _ int) kitex.Node {
		k := item.Key
		return ListItem(ItemProps{
			Key:      k,
			ID:       item.ID,
			OnDelete: func() { handleDelete(k) },
		})
	}

	return kitex.Box(kitex.BoxProps{
		Style: appContainerStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: appTitleStyle,
		}, kitex.Text("⚡ Kitex VDOM Reconciler Dashboard ⚡")),

		kitex.Box(kitex.BoxProps{
			Style: instructionsStyle,
		}, kitex.Text("Press 'q' to Quit. Click buttons below to Add, Reverse, Shuffle, or edit items.")),

		kitex.Box(kitex.BoxProps{
			Style: actionPanelStyle,
		},
			kitex.Button(kitex.ButtonProps{
				OnPress: func() {
					nid := getNextID()
					label := string(rune('A' + (nid-1)%26))
					if nid > 26 {
						label = fmt.Sprintf("%s%d", label, nid)
					}
					setItems.Update(func(items []ItemData) []ItemData {
						return append(items, ItemData{
							Key: fmt.Sprintf("%d", nid),
							ID:  label,
						})
					})
					setNextID.Update(func(n int) int { return n + 1 })
				},
				Style: addBtnStyle,
			}, kitex.Text(" Add Item ")),

			kitex.Button(kitex.ButtonProps{
				OnPress: func() {
					setItems.Update(func(items []ItemData) []ItemData {
						n := len(items)
						reversed := make([]ItemData, n)
						for i, item := range items {
							reversed[n-1-i] = item
						}
						return reversed
					})
				},
				Style: reverseBtnStyle,
			}, kitex.Text(" Reverse List ")),

			kitex.Button(kitex.ButtonProps{
				OnPress: func() {
					setItems.Update(func(items []ItemData) []ItemData {
						n := len(items)
						shuffled := make([]ItemData, n)
						copy(shuffled, items)
						rand.Shuffle(n, func(i, j int) {
							shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
						})
						return shuffled
					})
				},
				Style: shuffleBtnStyle,
			}, kitex.Text(" Shuffle List ")),
		),

		kitex.Box(kitex.BoxProps{
			Style: listContainerStyle,
		}, kitex.Map(getItems(), renderItem)),
	)
})

func main() {
	f, _ := os.Create("kitex_demo.log")
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
