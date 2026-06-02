package regressions

import (
	"fmt"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/extras/kitex"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

type ItemData struct {
	Key string
	ID  string
}

type ItemProps struct {
	Key string
	ID  string
}

var ListItem = kitex.FC("ListItem", func(props ItemProps) kitex.Node {
	return kitex.Box(kitex.BoxProps{
		ID:    "item-" + props.ID,
		Style: style.S().Height(style.Cells(1)),
	}, kitex.Text(fmt.Sprintf("Item %s", props.ID)))
})

func TestKitexReverseWithButtonClick(t *testing.T) {
	env := testenv.Default(80, 20)
	defer env.Close()

	App := kitex.FC("App", func(props struct{}) kitex.Node {
		items, set := kitex.UseState([]ItemData{
			{Key: "1", ID: "A"},
			{Key: "2", ID: "B"},
			{Key: "3", ID: "C"},
		})

		handleReverse := func(e event.Event) {
			curr := items()
			reversed := make([]ItemData, len(curr))
			for i, item := range curr {
				reversed[len(curr)-1-i] = item
			}
			set(reversed)
		}

		renderItem := func(item ItemData, _ int) kitex.Node {
			return ListItem(ItemProps{
				Key: item.Key,
				ID:  item.ID,
			})
		}

		return kitex.Box(kitex.BoxProps{
			ID: "root",
		},
			kitex.Button(kitex.ButtonProps{
				ID:      "reverse-btn",
				OnClick: handleReverse,
			}, kitex.Text("Reverse")),
			kitex.Box(kitex.BoxProps{
				ID: "list-container",
			}, kitex.Map(items(), renderItem)),
		)
	})

	container := element.NewBox(env.Document())
	env.Mount(container)
	kitex.Render(App(struct{}{}), container)
	env.Flush()

	// Initial check
	listContainer := env.QuerySelector("#list-container").(dom.Element)
	getIDs := func() []string {
		var ids []string
		for child := range listContainer.ChildNodes() {
			if el, ok := child.(dom.Element); ok {
				ids = append(ids, el.ID())
			}
		}
		return ids
	}

	initialIDs := getIDs()
	expectedInitial := []string{"item-A", "item-B", "item-C"}
	for i, id := range initialIDs {
		if id != expectedInitial[i] {
			t.Fatalf("initial order mismatch: got %v, want %v", initialIDs, expectedInitial)
		}
	}

	// Reverse via button click
	btn := env.QuerySelector("#reverse-btn").(dom.Element)
	rect, _ := btn.GetBoundingClientRect()
	env.Click(rect.Origin.X, rect.Origin.Y)
	env.Flush()

	afterReverseIDs := getIDs()
	expectedAfter := []string{"item-C", "item-B", "item-A"}
	if len(afterReverseIDs) != 3 {
		t.Fatalf("expected 3 items after reverse, got %d: %v", len(afterReverseIDs), afterReverseIDs)
	}
	for i, id := range afterReverseIDs {
		if id != expectedAfter[i] {
			t.Errorf("after reverse order mismatch: got %v, want %v", afterReverseIDs, expectedAfter)
			break
		}
	}
}

func TestKitexInsertAtStartWithReverse(t *testing.T) {
	env := testenv.Default(80, 20)
	defer env.Close()

	App := kitex.FC("App", func(props struct{}) kitex.Node {
		items, set := kitex.UseState([]ItemData{
			{Key: "1", ID: "A"},
			{Key: "2", ID: "B"},
		})

		handleAction := func(e event.Event) {
			set([]ItemData{
				{Key: "3", ID: "C"},
				{Key: "2", ID: "B"},
				{Key: "1", ID: "A"},
			})
		}

		renderItem := func(item ItemData, _ int) kitex.Node {
			return ListItem(ItemProps{
				Key: item.Key,
				ID:  item.ID,
			})
		}

		return kitex.Box(kitex.BoxProps{
			ID: "root",
		},
			kitex.Button(kitex.ButtonProps{
				ID:      "action-btn",
				OnClick: handleAction,
			}, kitex.Text("Action")),
			kitex.Box(kitex.BoxProps{
				ID: "list-container",
			}, kitex.Map(items(), renderItem)),
		)
	})

	container := element.NewBox(env.Document())
	env.Mount(container)
	kitex.Render(App(struct{}{}), container)
	env.Flush()

	// Action via button click
	btn := env.QuerySelector("#action-btn").(dom.Element)
	rect, _ := btn.GetBoundingClientRect()
	env.Click(rect.Origin.X, rect.Origin.Y)
	env.Flush()

	listContainer := env.QuerySelector("#list-container").(dom.Element)
	var ids []string
	for child := range listContainer.ChildNodes() {
		if el, ok := child.(dom.Element); ok {
			ids = append(ids, el.ID())
		}
	}

	expected := []string{"item-C", "item-B", "item-A"}
	if len(ids) != 3 {
		t.Fatalf("expected 3 items, got %d: %v", len(ids), ids)
	}
	for i, id := range ids {
		if id != expected[i] {
			t.Errorf("order mismatch: got %v, want %v", ids, expected)
			break
		}
	}
}
