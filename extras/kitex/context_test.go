package kitex

import (
	"fmt"
	"testing"

	"github.com/masterkeysrd/kite/dom"
)

func TestCreateContext_DefaultValue(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	themeCtx := CreateContext("light")

	var themeVal string
	Consumer := FC("Consumer", func(props struct{}) Node {
		themeVal = UseContext(themeCtx)
		return Text(themeVal)
	})

	Render(Consumer(struct{}{}), container)
	if themeVal != "light" {
		t.Errorf("expected default value 'light', got %s", themeVal)
	}
}

func TestProvider_ProvidesValue(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	themeCtx := CreateContext("light")

	var themeVal string
	Consumer := FC("Consumer", func(props struct{}) Node {
		themeVal = UseContext(themeCtx)
		return Text(themeVal)
	})

	app := themeCtx.Provider("dark", Consumer(struct{}{}))
	Render(app, container)

	if themeVal != "dark" {
		t.Errorf("expected provided value 'dark', got %s", themeVal)
	}
}

func TestProvider_NestedProviders(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	themeCtx := CreateContext("light")

	var innerTheme, outerTheme string
	InnerConsumer := FC("InnerConsumer", func(props struct{}) Node {
		innerTheme = UseContext(themeCtx)
		return Text(innerTheme)
	})
	OuterConsumer := FC("OuterConsumer", func(props struct{}) Node {
		outerTheme = UseContext(themeCtx)
		return Box(BoxProps{},
			Text(outerTheme),
			themeCtx.Provider("blue", InnerConsumer(struct{}{})),
		)
	})

	app := themeCtx.Provider("dark", OuterConsumer(struct{}{}))
	Render(app, container)

	if outerTheme != "dark" {
		t.Errorf("expected outer theme 'dark', got %s", outerTheme)
	}
	if innerTheme != "blue" {
		t.Errorf("expected inner theme 'blue', got %s", innerTheme)
	}
}

func TestProvider_ValueChange_TriggersConsumerReRender(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	themeCtx := CreateContext("light")

	var themeVal string
	Consumer := FC("Consumer", func(props struct{}) Node {
		themeVal = UseContext(themeCtx)
		return Text(themeVal)
	})

	var setProviderValue func(string)
	ProviderWrapper := FC("ProviderWrapper", func(props struct{}) Node {
		val, setVal := UseState("dark")
		setProviderValue = setVal
		return themeCtx.Provider(val(), Consumer(struct{}{}))
	})

	Render(ProviderWrapper(struct{}{}), container)
	if themeVal != "dark" {
		t.Errorf("expected initial theme 'dark', got %s", themeVal)
	}

	setProviderValue("blue")
	if themeVal != "blue" {
		t.Errorf("expected updated theme 'blue', got %s", themeVal)
	}
}

func TestProvider_ValueChange_BypassesMemoization(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	themeCtx := CreateContext("light")

	var themeVal string
	Consumer := FC("Consumer", func(props struct{}) Node {
		themeVal = UseContext(themeCtx)
		return Text(themeVal)
	})

	Wrapper := FC("Wrapper", func(props struct{}) Node {
		// Render >= 5 nodes to trigger c.shouldMemo = true
		return Box(BoxProps{},
			Box(BoxProps{},
				Box(BoxProps{},
					Box(BoxProps{},
						Box(BoxProps{},
							Consumer(struct{}{}),
						),
					),
				),
			),
		)
	})

	var setProviderValue func(string)
	ProviderWrapper := FC("ProviderWrapper", func(props struct{}) Node {
		val, setVal := UseState("dark")
		setProviderValue = setVal
		return themeCtx.Provider(val(), Wrapper(struct{}{}))
	})

	Render(ProviderWrapper(struct{}{}), container)
	if themeVal != "dark" {
		t.Errorf("expected initial theme 'dark', got %s", themeVal)
	}

	setProviderValue("blue")
	if themeVal != "blue" {
		t.Errorf("expected updated theme 'blue', got %s", themeVal)
	}
}

func TestProvider_Deduplication(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	themeCtx := CreateContext("light")

	renderCount := 0
	Consumer := FC("Consumer", func(props struct{}) Node {
		renderCount++
		_ = UseContext(themeCtx)
		return Text("Consumer")
	})

	var setProviderValue func(string)
	ProviderWrapper := FC("ProviderWrapper", func(props struct{}) Node {
		val, setVal := UseState("dark")
		setProviderValue = setVal
		return themeCtx.Provider(val(), Consumer(struct{}{}))
	})

	Render(ProviderWrapper(struct{}{}), container)
	renderCountBefore := renderCount

	setProviderValue("blue")
	// The Consumer should render exactly once more (no double rendering from dirty state queue)
	if renderCount-renderCountBefore != 1 {
		t.Errorf("expected exactly 1 additional render, got %d", renderCount-renderCountBefore)
	}
}

func TestProvider_MultipleConsumers(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	themeCtx := CreateContext("light")

	var val1, val2 string
	Consumer1 := FC("Consumer1", func(props struct{}) Node {
		val1 = UseContext(themeCtx)
		return Text(val1)
	})
	Consumer2 := FC("Consumer2", func(props struct{}) Node {
		val2 = UseContext(themeCtx)
		return Text(val2)
	})

	var setProviderValue func(string)
	ProviderWrapper := FC("ProviderWrapper", func(props struct{}) Node {
		val, setVal := UseState("dark")
		setProviderValue = setVal
		return themeCtx.Provider(val(), Box(BoxProps{},
			Consumer1(struct{}{}),
			Consumer2(struct{}{}),
		))
	})

	Render(ProviderWrapper(struct{}{}), container)
	if val1 != "dark" || val2 != "dark" {
		t.Errorf("expected both to be 'dark', got %s and %s", val1, val2)
	}

	setProviderValue("blue")
	if val1 != "blue" || val2 != "blue" {
		t.Errorf("expected both to be updated to 'blue', got %s and %s", val1, val2)
	}
}

func TestProvider_Unmount_Unsubscribes(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	themeCtx := CreateContext("light")

	var themeVal string
	Consumer := FC("Consumer", func(props struct{}) Node {
		themeVal = UseContext(themeCtx)
		return Text(themeVal)
	})

	var setShowConsumer func(bool)
	var setProviderValue func(string)
	ProviderWrapper := FC("ProviderWrapper", func(props struct{}) Node {
		val, setVal := UseState("dark")
		setProviderValue = setVal
		show, setShow := UseState(true)
		setShowConsumer = setShow

		if show() {
			return themeCtx.Provider(val(), Consumer(struct{}{}))
		}
		return themeCtx.Provider(val(), Box(BoxProps{}, Text("Empty")))
	})

	Render(ProviderWrapper(struct{}{}), container)
	if themeVal != "dark" {
		t.Errorf("expected initial value 'dark', got %s", themeVal)
	}
	// Unmount consumer
	setShowConsumer(false)
	// Change value after unmount
	setProviderValue("blue")

	// The themeVal should NOT be updated to "blue" because Consumer was unmounted
	if themeVal != "dark" {
		t.Errorf("expected themeVal to remain 'dark', got %s", themeVal)
	}
}

func TestUseContext_StableAcrossReRenders(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	themeCtx := CreateContext("light")

	renderCount := 0
	Consumer := FC("Consumer", func(props struct{}) Node {
		renderCount++
		val := UseContext(themeCtx)
		return Text(val)
	})

	var triggerUpdate func()
	ProviderWrapper := FC("ProviderWrapper", func(props struct{}) Node {
		_, setDummy := UseState(0)
		triggerUpdate = func() { setDummy(1) }
		return themeCtx.Provider("dark", Consumer(struct{}{}))
	})

	Render(ProviderWrapper(struct{}{}), container)
	// Trigger re-render of Consumer directly via parent re-render (without provider value change).
	// On subsequent renders, UseContext should read from stored reference.
	triggerUpdate()

	if renderCount != 2 {
		t.Errorf("expected 2 renders, got %d", renderCount)
	}
}

func TestProvider_DifferentContextTypes(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc).(dom.Element)

	themeCtx := CreateContext("light")
	countCtx := CreateContext(100)

	var themeVal string
	var countVal int
	Consumer := FC("Consumer", func(props struct{}) Node {
		themeVal = UseContext(themeCtx)
		countVal = UseContext(countCtx)
		return Text(fmt.Sprintf("%s-%d", themeVal, countVal))
	})

	app := themeCtx.Provider("dark",
		countCtx.Provider(200,
			Consumer(struct{}{}),
		),
	)
	Render(app, container)

	if themeVal != "dark" {
		t.Errorf("expected theme 'dark', got %s", themeVal)
	}
	if countVal != 200 {
		t.Errorf("expected count 200, got %d", countVal)
	}
}
