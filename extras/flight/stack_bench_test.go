package flight

import (
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/internal/focus"
)

func BenchmarkStack_PushPop(b *testing.B) {
	doc := dom.NewDocument()
	fm := focus.NewManager(doc, event.NewDispatcher())
	doc.SetFocusHandle(fm)

	container := kitex.Div(kitex.BoxProps{}).Instantiate(doc).(dom.Element)
	doc.AppendChild(container)
	defer kitex.Render(nil, container)

	var nav Navigator

	HomeView := kitex.SimpleFC("HomeView", func() kitex.Node {
		nav = UseNavigation()
		return kitex.Box(kitex.BoxProps{ID: "home"},
			kitex.Button(kitex.ButtonProps{ID: "btn-home"}),
		)
	})

	DetailsView := kitex.FC("DetailsView", func(props struct{ ID string }) kitex.Node {
		nav = UseNavigation()
		return kitex.Box(kitex.BoxProps{ID: "details"},
			kitex.Button(kitex.ButtonProps{ID: "btn-details"}),
		)
	})

	stack := Stack(StackProps{
		InitialRoute: HomeRoute{},
		RenderRoute: func(r Route) kitex.Node {
			switch route := r.(type) {
			case HomeRoute:
				return HomeView()
			case DetailsRoute:
				return DetailsView(struct{ ID string }{ID: route.ID})
			default:
				return kitex.Box(kitex.BoxProps{})
			}
		},
	})

	// Initial render
	kitex.Render(stack, container)

	for b.Loop() {
		nav.Push(DetailsRoute{ID: "test"})
		nav.Pop()
	}
}

func BenchmarkStack_Init(b *testing.B) {
	for b.Loop() {
		b.StopTimer()
		doc := dom.NewDocument()
		fm := focus.NewManager(doc, event.NewDispatcher())
		doc.SetFocusHandle(fm)

		container := kitex.Div(kitex.BoxProps{}).Instantiate(doc).(dom.Element)
		doc.AppendChild(container)

		HomeView := kitex.SimpleFC("HomeView", func() kitex.Node {
			return kitex.Box(kitex.BoxProps{ID: "home"})
		})

		stack := Stack(StackProps{
			InitialRoute: HomeRoute{},
			RenderRoute: func(r Route) kitex.Node {
				return HomeView()
			},
		})

		b.StartTimer()
		kitex.Render(stack, container)
		b.StopTimer()

		kitex.Render(nil, container)
	}
}
