package engine

import (
	"fmt"
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/backend/mock"
	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	internaldom "github.com/masterkeysrd/kite/internal/dom"
	"github.com/masterkeysrd/kite/style"
)

func BenchmarkPipeline_Standard(b *testing.B) {
	mockBackend := mock.New(80, 24)
	e := New(mockBackend, Options{})
	for b.Loop() {
		e.Frame()
	}
}

func BenchmarkPipeline_Profiling(b *testing.B) {
	mockBackend := mock.New(80, 24)
	e := New(mockBackend, Options{Profiler: true})
	for b.Loop() {
		e.Frame()
	}
}

func BenchmarkPipeline_SyncAndLayout_100Nodes(b *testing.B) {
	mockBackend := mock.New(80, 24)
	e := New(mockBackend, Options{})

	// Add 100 child elements to the document
	doc := e.Document()
	for range 100 {
		child := doc.CreateElement("div", nil)
		doc.AppendChild(child)
	}

	b.ReportAllocs()
	for b.Loop() {
		// Mark the document sync-dirty to trigger diffChildren
		if dn := internaldom.AsDirty(doc); dn != nil {
			dn.MarkNeedsSync()
		}
		// Mark children style-dirty
		for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
			if de := internaldom.AsDirtyElement(child); de != nil {
				de.MarkStyleDirty()
			}
		}

		e.pipeline.Sync(e)
		e.pipeline.Style(e)
		e.pipeline.Layout(e)
	}
}

func BenchmarkEngine_RealisticApp_1000Nodes(b *testing.B) {
	mockBackend := mock.New(120, 40)
	e := New(mockBackend, Options{})

	doc := e.Document()

	// Header (1 box, 1 text)
	header := element.NewBox(doc)
	header.Style(style.S().
		Width(style.Percent(100)).
		Height(style.Cells(3)).
		Background(color.RGBA{R: 20, G: 20, B: 20, A: 255}).
		Border(style.SingleBorder()))
	headerText := doc.CreateTextNode("Dashboard Header", nil)
	header.AppendChild(headerText)

	// Main content container
	main := element.NewBox(doc)
	main.Style(style.S().
		Width(style.Percent(100)).
		Height(style.Percent(100)).
		Display(style.DisplayFlex).
		FlexDirection(style.FlexRow))

	// Sidebar (1 sidebar, 10 list boxes, 10 text nodes)
	sidebar := element.NewBox(doc)
	sidebar.Style(style.S().
		Width(style.Cells(25)).
		Height(style.Percent(100)).
		Background(color.RGBA{R: 30, G: 30, B: 30, A: 255}))
	for i := range 10 {
		item := element.NewBox(doc)
		item.Style(style.S().
			Padding(0, 1).
			Border(style.SingleBorder()))
		item.AppendChild(doc.CreateTextNode(fmt.Sprintf("Sidebar Link %d", i), nil))
		sidebar.AppendChild(item)
	}

	// Content Area
	content := element.NewBox(doc)
	content.Style(style.S().
		Flex(1, 1, style.Cells(0)).
		Height(style.Percent(100)).
		Display(style.DisplayFlex).
		FlexDirection(style.FlexColumn).
		Overflow(style.OverflowAuto))

	// Add 240 row items.
	// Each row has a container, titleBox (with dynamic text), badge (with text) = 5 nodes per row.
	// Total content nodes: 240 * 5 = 1200 nodes.
	var dynamicTextNodes []dom.TextNode
	for i := range 240 {
		row := element.NewBox(doc)
		row.Style(style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Padding(0, 1).
			Border(style.SingleBorder()))

		titleBox := element.NewBox(doc)
		titleBox.Style(style.S().Width(style.Percent(50)))
		txt := doc.CreateTextNode(fmt.Sprintf("Item Title #%d", i), nil)
		titleBox.AppendChild(txt)
		dynamicTextNodes = append(dynamicTextNodes, txt)

		badge := element.NewBox(doc)
		badge.Style(style.S().
			Background(color.RGBA{R: 100, G: 200, B: 100, A: 255}).
			Padding(0, 1))
		badge.AppendChild(doc.CreateTextNode("ACTIVE", nil))

		row.AppendChild(titleBox)
		row.AppendChild(badge)
		content.AppendChild(row)
	}

	main.AppendChild(sidebar)
	main.AppendChild(content)

	root := element.Box(header, main)
	root.Style(style.S().
		Width(style.Percent(100)).
		Height(style.Percent(100)).
		Display(style.DisplayFlex).
		FlexDirection(style.FlexColumn))

	e.Mount(root)
	e.Frame() // Initial sync, style, layout, paint

	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		// Mutate a text node to trigger dirty state and full re-render
		targetIdx := i % len(dynamicTextNodes)
		dynamicTextNodes[targetIdx].SetData(fmt.Sprintf("Updated Title #%d (run %d)", targetIdx, i))

		// Execute full pipeline frame
		e.Frame()
	}
}

func BenchmarkEngine_ScrollOnly_1000Nodes(b *testing.B) {
	mockBackend := mock.New(120, 40)
	e := New(mockBackend, Options{})

	doc := e.Document()

	// Content Area
	content := element.NewBox(doc)
	content.Style(style.S().
		Width(style.Percent(100)).
		Height(style.Percent(100)).
		Display(style.DisplayFlex).
		FlexDirection(style.FlexColumn).
		Overflow(style.OverflowAuto))

	// Add 200 row items to get around 1000 nodes total.
	for i := range 200 {
		row := element.NewBox(doc)
		row.Style(style.S().
			Display(style.DisplayFlex).
			FlexDirection(style.FlexRow).
			Padding(0, 1).
			Border(style.SingleBorder()))

		titleBox := element.NewBox(doc)
		titleBox.Style(style.S().Width(style.Percent(50)))
		txt := doc.CreateTextNode(fmt.Sprintf("Item Title #%d", i), nil)
		titleBox.AppendChild(txt)

		badge := element.NewBox(doc)
		badge.Style(style.S().
			Background(color.RGBA{R: 100, G: 200, B: 100, A: 255}).
			Padding(0, 1))
		badge.AppendChild(doc.CreateTextNode("ACTIVE", nil))

		row.AppendChild(titleBox)
		row.AppendChild(badge)
		content.AppendChild(row)
	}

	e.Mount(content)
	e.Frame() // Initial sync, style, layout, paint

	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		// Change scroll to trigger a scroll update
		content.ScrollTo(0, i%100)

		// Execute pipeline frame
		e.Frame()
	}
}
