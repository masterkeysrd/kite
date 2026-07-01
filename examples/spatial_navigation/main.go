package main

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"os"

	"github.com/masterkeysrd/kite/backend/uv"
	"github.com/masterkeysrd/kite/devtools"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/key"
	"github.com/masterkeysrd/kite/style"
)

var (
	// Base styles for elements
	baseStyle = style.S().
			Border(style.SingleBorder().Color(color.RGBA{70, 70, 80, 255})).
			Background(color.RGBA{30, 30, 35, 255}).
			Foreground(color.RGBA{200, 200, 200, 255}).
			Padding(0, 1).
			Height(style.Cells(3)).
			WhiteSpace(style.WhiteSpacePre).
			JustifyContent(style.JustifyCenter).
			AlignItems(style.AlignCenter)

	focusedStyle = style.S().
			Border(style.DoubleBorder().Color(color.RGBA{0, 220, 255, 255})).
			Background(color.RGBA{0, 50, 100, 255}).
			Foreground(color.White).
			Bold(true).
			Padding(0, 1).
			Height(style.Cells(3)).
			WhiteSpace(style.WhiteSpacePre).
			JustifyContent(style.JustifyCenter).
			AlignItems(style.AlignCenter)

	hoverStyle = style.S().
			Border(style.SingleBorder().Color(color.RGBA{0, 220, 255, 255})).
			Background(color.RGBA{40, 40, 50, 255}).
			Foreground(color.White).
			Padding(0, 1).
			Height(style.Cells(3)).
			WhiteSpace(style.WhiteSpacePre).
			JustifyContent(style.JustifyCenter).
			AlignItems(style.AlignCenter)

	// Container & Layout Styles
	rootStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Width(style.Percent(100)).
			Height(style.Percent(100)).
			Background(color.RGBA{18, 18, 20, 255}).
			Padding(1, 2)

	sidebarStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Width(style.Cells(22)).
			Border(style.SingleBorder().Color(color.RGBA{60, 60, 70, 255})).
			Padding(1, 1).
			Gap(1, 0).
			Margin(0, 2, 0, 0)

	contentStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexColumn).
			Flex(1).
			Gap(1, 0)

	headerStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Gap(0, 2).
			Margin(0, 0, 1, 0)

	gridRowStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Gap(0, 4).
			Margin(0, 0, 1, 0)

	logPanelStyle = style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Height(style.Cells(6)).
			Border(style.SingleBorder().Color(color.RGBA{60, 60, 70, 255})).
			Padding(1, 2).
			Margin(1, 0, 0, 0)

	virtualArrowPadStyle = style.S().
				Display(style.DisplayFlex).
				FlexDirection(style.FlexColumn).
				AlignItems(style.AlignCenter).
				JustifyContent(style.JustifyCenter).
				Width(style.Cells(14)).
				Border(style.SingleBorder().Color(color.RGBA{45, 45, 50, 255})).
				Margin(0, 0, 0, 4)

	arrowNormal = style.S().Foreground(color.RGBA{90, 90, 100, 255})
	arrowActive = style.S().Foreground(color.RGBA{0, 220, 255, 255}).Bold(true)
)

func setupButton(btn *element.ButtonElement, requestFrame func()) *element.ButtonElement {
	btn.Style(baseStyle)
	var isFocused bool

	btn.OnEvent(event.EventFocus, func(e event.Event) {
		isFocused = true
		btn.Style(focusedStyle)
		requestFrame()
	})

	btn.OnEvent(event.EventBlur, func(e event.Event) {
		isFocused = false
		btn.Style(baseStyle)
		requestFrame()
	})

	btn.OnEvent(event.EventMouseEnter, func(e event.Event) {
		if !isFocused {
			btn.Style(hoverStyle)
			requestFrame()
		}
	})

	btn.OnEvent(event.EventMouseLeave, func(e event.Event) {
		if !isFocused {
			btn.Style(baseStyle)
			requestFrame()
		}
	})

	return btn
}

func main() {
	f, _ := os.Create("kite.log")
	defer f.Close()
	logger := slog.New(slog.NewTextHandler(f, nil))
	slog.SetDefault(logger)

	b, err := uv.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize UV backend: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(b, engine.Options{})
	requestFrame := func() {
		eng.RequestFrame()
	}

	// 1. Sidebar Buttons
	sidebarTitle := element.Box("  SIDEBAR MENU").Style(style.S().Bold(true).Margin(0, 0, 1, 0).Foreground(color.RGBA{130, 130, 140, 255}))
	btnHome := setupButton(element.Button("  Home  ").WithID("menu-home"), requestFrame)
	btnAnalytics := setupButton(element.Button("  Analytics  ").WithID("menu-analytics"), requestFrame)
	btnProfile := setupButton(element.Button("  Profile  ").WithID("menu-profile"), requestFrame)
	btnSettings := setupButton(element.Button("  Settings  ").WithID("menu-settings"), requestFrame)

	sidebar := element.Box(
		sidebarTitle,
		btnHome,
		btnAnalytics,
		btnProfile,
		btnSettings,
	).Style(sidebarStyle)

	// 2. Header Tabs
	tab1 := setupButton(element.Button("  Tab One  ").WithID("tab-1").CursorNavigable(true), requestFrame)
	tab2 := setupButton(element.Button("  Tab Two  ").WithID("tab-2").CursorNavigable(true), requestFrame)
	tab3 := setupButton(element.Button("  Tab Three  ").WithID("tab-3").CursorNavigable(true), requestFrame)

	header := element.Box(
		tab1,
		tab2,
		tab3,
	).Style(headerStyle)

	// 3. Grid of Cards
	card1 := setupButton(element.Button("   Card A   ").WithID("card-a").CursorNavigable(true), requestFrame)
	card2 := setupButton(element.Button("   Card B   ").WithID("card-b").CursorNavigable(true), requestFrame)
	card3 := setupButton(element.Button("   Card C   ").WithID("card-c").CursorNavigable(true), requestFrame)
	card4 := setupButton(element.Button("   Card D   ").WithID("card-d").CursorNavigable(true), requestFrame)

	row1 := element.Box(card1, card2).Style(gridRowStyle)
	row2 := element.Box(card3, card4).Style(gridRowStyle)

	// 4. Log Panel & Virtual Arrow Pad
	focusText := element.Text("Focused: None")
	keyText := element.Text("Last Key: None")

	logBox := element.Box(
		element.Box("LOG CONSOLE").Style(style.S().Bold(true).Foreground(color.RGBA{130, 130, 140, 255}).Margin(0, 0, 1, 0)),
		element.Box(focusText),
		element.Box(keyText),
	).Style(style.S().Display(style.DisplayFlex).FlexDirection(style.FlexColumn).Flex(1))

	arrowUp := element.Span("▲").Style(arrowNormal)
	arrowLeft := element.Span("◄ ").Style(arrowNormal)
	arrowDown := element.Span("▼").Style(arrowNormal)
	arrowRight := element.Span(" ►").Style(arrowNormal)

	arrowPad := element.Box(
		element.Box(arrowUp),
		element.Box(arrowLeft, arrowDown, arrowRight),
	).Style(virtualArrowPadStyle)

	logPanel := element.Box(
		logBox,
		arrowPad,
	).Style(logPanelStyle)

	// 5. Main Layout Assembly
	contentArea := element.Box(
		header,
		element.Box("Action Cards Grid (Move cursor with Left/Right arrows):").Style(style.S().Bold(true).Foreground(color.RGBA{160, 160, 170, 255}).Margin(0, 0, 1, 0)),
		row1,
		row2,
		logPanel,
	).Style(contentStyle)

	root := element.Box(
		sidebar,
		contentArea,
	).Style(rootStyle)

	eng.Mount(root)

	// Focus/Blur listeners at document level to show active focus details in the console
	eng.Document().AddEventListener(event.EventFocus, func(e event.Event) {
		if target, ok := e.Target().EventTarget().(dom.Element); ok {
			var caretStr string
			if sel := eng.Document().Selection(); sel.RangeCount() > 0 {
				r := sel.GetRangeAt(0)
				caretStr = fmt.Sprintf(" [Caret: %d]", r.EndOffset())
			}
			focusText.SetData(fmt.Sprintf("Focused: %s#%s%s", target.TagName(), target.ID(), caretStr))
			requestFrame()
		}
	}, event.Capture())

	// Keyboard listeners to show visual key pressed in the Arrow Pad
	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		keyText.SetData(fmt.Sprintf("Last Key: %s", keyToString(ke.Key)))

		switch {
		case ke.MatchString("up"):
			arrowUp.Style(arrowActive)
		case ke.MatchString("down"):
			arrowDown.Style(arrowActive)
		case ke.MatchString("left"):
			arrowLeft.Style(arrowActive)
		case ke.MatchString("right"):
			arrowRight.Style(arrowActive)
		}

		if target, ok := eng.FocusManager().Current().(dom.Element); ok {
			var caretStr string
			if sel := eng.Document().Selection(); sel.RangeCount() > 0 {
				r := sel.GetRangeAt(0)
				caretStr = fmt.Sprintf(" [Caret: %d]", r.EndOffset())
			}
			focusText.SetData(fmt.Sprintf("Focused: %s#%s%s", target.TagName(), target.ID(), caretStr))
		}
		requestFrame()
	})

	eng.Document().AddEventListener(event.EventKeyUp, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		switch {
		case ke.MatchString("up"):
			arrowUp.Style(arrowNormal)
		case ke.MatchString("down"):
			arrowDown.Style(arrowNormal)
		case ke.MatchString("left"):
			arrowLeft.Style(arrowNormal)
		case ke.MatchString("right"):
			arrowRight.Style(arrowNormal)
		}
		requestFrame()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eng.Document().AddEventListener(event.EventKeyDown, func(e event.Event) {
		ke := e.(*event.KeyEvent)
		if ke.MatchString("q") || ke.MatchString("ctrl+c") {
			cancel()
		}
	})

	devtools.Install(eng, devtools.Options{})

	if err := eng.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "engine exited: %v\n", err)
	}
}

func keyToString(k key.Key) string {
	if k.Text != "" {
		return k.Text
	}
	switch k.Code {
	case key.KeyUp:
		return "up"
	case key.KeyDown:
		return "down"
	case key.KeyLeft:
		return "left"
	case key.KeyRight:
		return "right"
	case key.KeyTab:
		return "tab"
	case key.KeyEscape:
		return "escape"
	case key.KeyEnter:
		return "enter"
	case key.KeySpace:
		return "space"
	case key.KeyBackspace:
		return "backspace"
	default:
		return fmt.Sprintf("Code(%d)", k.Code)
	}
}
