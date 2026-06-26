package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/backend/mock"
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
	rootStyle = style.S().
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Display(style.DisplayFlex).
			JustifyContent(style.JustifyCenter).
			AlignItems(style.AlignCenter).
			Background(color.RGBA{R: 20, G: 20, B: 20, A: 255}).
			Overflow(style.OverflowAuto)

	boxStyle = style.S().
			Width(style.Percent(80)).
			Height(style.Percent(80)).
			MinWidth(style.Cells(40)).
			MinHeight(style.Cells(20)).
			MaxWidth(style.Cells(100)).
			MaxHeight(style.Cells(40)).
			Background(color.RGBA{R: 50, G: 100, B: 200, A: 255}).
			Border(style.SingleBorder()).
			Padding(1, 2).
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Overflow(style.OverflowAuto)

	tabsContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexRow).
				Gap(0, 1).
				Margin(0, 0, 1, 0)

	tabBtnStyle = style.S().
			Padding(0, 1).
			Background(color.RGBA{R: 80, G: 80, B: 80, A: 255}).
			Foreground(color.RGBA{R: 220, G: 220, B: 220, A: 255}).
			Border(style.SingleBorder())

	activeTabBtnStyle = style.S().
				Padding(0, 1).
				Background(color.RGBA{R: 200, G: 200, B: 200, A: 255}).
				Foreground(color.RGBA{R: 0, G: 0, B: 0, A: 255}).
				Border(style.SingleBorder())

	contentStyle = style.S().
			Foreground(color.RGBA{R: 255, G: 255, B: 255, A: 255})

	contentContainerStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				Gap(1, 0)
)

var App = kitex.SimpleFC("App", func() kitex.Node {
	getActiveTab, setActiveTab := kitex.UseState(0)

	tabContents := []int{5, 15, 30} // Number of lines for each tab

	renderTabBtn := func(index int, label string) kitex.Node {
		isActive := getActiveTab() == index
		s := tabBtnStyle
		if isActive {
			s = activeTabBtnStyle
		}

		return kitex.Button(kitex.ButtonProps{
			Style: s,
			OnClick: func(e event.Event) {
				setActiveTab(index)
			},
		}, kitex.Text(label))
	}

	linesCount := tabContents[getActiveTab()]

	// Create content lines
	var contentNodes []kitex.Node
	for i := 1; i <= linesCount; i++ {
		contentNodes = append(contentNodes, kitex.Box(kitex.BoxProps{
			Style: contentStyle,
		}, kitex.Text(fmt.Sprintf("Tab %d - Line of content %d", getActiveTab()+1, i))))
	}

	return kitex.Box(kitex.BoxProps{
		Style: boxStyle,
	},
		kitex.Box(kitex.BoxProps{
			Style: tabsContainerStyle,
		},
			renderTabBtn(0, " Tab 1 (Small) "),
			renderTabBtn(1, " Tab 2 (Medium) "),
			renderTabBtn(2, " Tab 3 (Large) "),
		),
		kitex.Box(kitex.BoxProps{
			Style: contentContainerStyle,
		}, contentNodes...),
	)
})

func main() {
	f, er := os.Create("kite_repro.log")
	if er != nil {
		fmt.Fprintf(os.Stderr, "failed to create log file: %v\n", er)
		os.Exit(1)
	}
	defer f.Close()

	logger := slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))

	_ = logger // prevent unused variable error
	slog.SetDefault(logger)

	var b backend.Backend
	var err error
	if os.Getenv("USE_MOCK_BACKEND") == "1" {
		b = mock.New(80, 24)
	} else {
		b, err = uv.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
			os.Exit(1)
		}
	}

	opts := engine.Options{}
	eng := engine.New(b, opts)

	container := element.NewBox(eng.Document())
	container.Style(rootStyle)
	eng.Mount(container)

	kitex.EnableDevMode = true

	kitex.Render(App(), container)

	insp, _ := devtools.Install(eng, devtools.Options{})
	kitexdt.Register(insp)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		if ke, ok := e.(*event.KeyEvent); ok {
			if ke.MatchString("ctrl+c") || ke.MatchString("q") {
				cancel()
			}
		}
	})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited with error: %v\n", err)
		os.Exit(1)
	}
}
