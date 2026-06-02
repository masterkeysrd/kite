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
	itemRowStyle           = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).AlignItems(style.AlignCenter).Margin(style.Edges(0, 0, 1, 0)).Padding(style.Edges(0, 1)).Background(color.RGBA{R: 35, G: 35, B: 50, A: 255}).Border(style.SingleBorder())
	itemLabelStyle         = style.S().Foreground(color.RGBA{R: 220, G: 220, B: 230, A: 255}).Width(style.Percent(40))
	incrementBtnStyle      = style.S().Background(color.RGBA{R: 60, G: 120, B: 220, A: 255}).Foreground(color.White).Margin(style.Edges(0, 1))
	incrementBtnHoverStyle = style.S().Background(color.RGBA{R: 80, G: 140, B: 240, A: 255})
	deleteBtnStyle         = style.S().Background(color.RGBA{R: 200, G: 60, B: 60, A: 255}).Foreground(color.White).Margin(style.Edges(0, 1))
	deleteBtnHoverStyle    = style.S().Background(color.RGBA{R: 240, G: 80, B: 80, A: 255})
	appContainerStyle      = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).Width(style.Percent(100)).Height(style.Percent(100)).Background(color.RGBA{R: 20, G: 20, B: 30, A: 255}).Padding(style.Edges(1, 2))
	appTitleStyle          = style.S().Bold(true).Foreground(color.RGBA{R: 90, G: 140, B: 255, A: 255}).Margin(style.Edges(0, 0, 1, 0)).TextAlign(style.TextAlignCenter)
	instructionsStyle      = style.S().Foreground(color.RGBA{R: 150, G: 150, B: 170, A: 255}).Margin(style.Edges(0, 0, 1, 0))
	actionPanelStyle       = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexRow).Margin(style.Edges(0, 0, 1, 0))
	addBtnStyle            = style.S().Background(color.RGBA{R: 50, G: 180, B: 100, A: 255}).Foreground(color.White).Margin(style.Edges(0, 1))
	addBtnHoverStyle       = style.S().Background(color.RGBA{R: 70, G: 210, B: 120, A: 255})
	reverseBtnStyle        = style.S().Background(color.RGBA{R: 160, G: 80, B: 220, A: 255}).Foreground(color.White).Margin(style.Edges(0, 1))
	reverseBtnHoverStyle   = style.S().Background(color.RGBA{R: 190, G: 100, B: 250, A: 255})
	shuffleBtnStyle        = style.S().Background(color.RGBA{R: 220, G: 130, B: 40, A: 255}).Foreground(color.White).Margin(style.Edges(0, 1))
	shuffleBtnHoverStyle   = style.S().Background(color.RGBA{R: 250, G: 160, B: 60, A: 255})
	listContainerStyle     = style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).Border(style.DoubleBorder()).Padding(style.Edges(1, 1)).Background(color.RGBA{R: 25, G: 25, B: 38, A: 255})
	rootStyle              = style.S().Width(style.Percent(100)).Height(style.Percent(100))
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

type HoverButtonProps struct {
	OnClick    func(event.Event)
	Style      style.Style
	HoverStyle style.Style
	Text       string
}

// HoverButton is a functional component that wraps kitex.Button and adds hover style support.
var HoverButton = kitex.FC("HoverButton", func(props HoverButtonProps) kitex.Node {
	isHovered, setHovered := kitex.UseState(false)

	s := props.Style
	if isHovered() {
		s = s.Merge(props.HoverStyle)
	}

	return kitex.Button(kitex.ButtonProps{
		OnClick: props.OnClick,
		OnMouseEnter: func(e event.Event) {
			setHovered(true)
		},
		OnMouseLeave: func(e event.Event) {
			setHovered(false)
		},
		Style: s,
	}, kitex.Text(props.Text))
})

// ListItem is a functional component representing a single row in the list.
// It maintains its own local counter state (clicks) using UseState.
var ListItem = kitex.FC("ListItem", func(props ItemProps) kitex.Node {
	getClicks, setClicks := kitex.UseState(0)

	return kitex.Box(kitex.BoxProps{
		Style: itemRowStyle,
	},
		kitex.Span(kitex.SpanProps{
			Style: itemLabelStyle,
		}, kitex.Text(fmt.Sprintf("Item %s (Clicks: %d)", props.ID, getClicks()))),

		HoverButton(HoverButtonProps{
			OnClick: func(e event.Event) {
				setClicks(getClicks() + 1)
			},
			Style:      incrementBtnStyle,
			HoverStyle: incrementBtnHoverStyle,
			Text:       " +1 ",
		}),

		HoverButton(HoverButtonProps{
			OnClick: func(e event.Event) {
				props.OnDelete()
			},
			Style:      deleteBtnStyle,
			HoverStyle: deleteBtnHoverStyle,
			Text:       " Delete ",
		}),
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
		current := getItems()
		nextItems := make([]ItemData, 0, len(current))
		for _, item := range current {
			if item.Key != key {
				nextItems = append(nextItems, item)
			}
		}
		setItems(nextItems)
	}

	// renderItem is a named function variable passed directly to kitex.Map.
	// It captures handleDelete from the enclosing render scope — idiomatic Go
	// for parameterised render helpers that share component-level state.
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
		// Title bar
		kitex.Box(kitex.BoxProps{
			Style: appTitleStyle,
		}, kitex.Text("⚡ Kitex VDOM Reconciler Dashboard ⚡")),

		// Info / Instructions
		kitex.Box(kitex.BoxProps{
			Style: instructionsStyle,
		}, kitex.Text("Press 'q' to Quit. Click buttons below to Add, Reverse, Shuffle, or edit items.")),

		// Actions Panel
		kitex.Box(kitex.BoxProps{
			Style: actionPanelStyle,
		},
			HoverButton(HoverButtonProps{
				OnClick: func(e event.Event) {
					nid := getNextID()
					label := string(rune('A' + (nid-1)%26))
					if nid > 26 {
						label = fmt.Sprintf("%s%d", label, nid)
					}
					setItems(append(getItems(), ItemData{
						Key: fmt.Sprintf("%d", nid),
						ID:  label,
					}))
					setNextID(nid + 1)
				},
				Style:      addBtnStyle,
				HoverStyle: addBtnHoverStyle,
				Text:       " Add Item ",
			}),

			HoverButton(HoverButtonProps{
				OnClick: func(e event.Event) {
					items := getItems()
					n := len(items)
					reversed := make([]ItemData, n)
					for i, item := range items {
						reversed[n-1-i] = item
					}
					setItems(reversed)
				},
				Style:      reverseBtnStyle,
				HoverStyle: reverseBtnHoverStyle,
				Text:       " Reverse List ",
			}),

			HoverButton(HoverButtonProps{
				OnClick: func(e event.Event) {
					items := getItems()
					n := len(items)
					shuffled := make([]ItemData, n)
					copy(shuffled, items)
					rand.Shuffle(n, func(i, j int) {
						shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
					})
					setItems(shuffled)
				},
				Style:      shuffleBtnStyle,
				HoverStyle: shuffleBtnHoverStyle,
				Text:       " Shuffle List ",
			}),
		),

		// Keyed list — kitex.Map(getItems(), renderItem) passes the named
		// function directly; no intermediate slice variable needed.
		kitex.Box(kitex.BoxProps{
			Style: listContainerStyle,
		}, kitex.Map(getItems(), renderItem)),
	)
})

func main() {
	f, _ := os.Create("kitex_demo.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{Logger: logger})

	// Create VDOM rendering container element
	container := element.NewBox(eng.Document())
	container.Style(rootStyle)
	eng.Mount(container)

	kitex.EnableDevMode = true

	// Mount VDOM into host container
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
