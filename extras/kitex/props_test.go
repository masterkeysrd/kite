package kitex

import (
	"reflect"
	"testing"

	"github.com/masterkeysrd/kite/dom"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/event"
	"github.com/masterkeysrd/kite/style"
)

func TestGeneratedProps_ForwardAllElementProps(t *testing.T) {
	ref := CreateRef[dom.Node]()
	expected := ElementProps{
		Key:             "key-1",
		ID:              "id-1",
		Class:           "cls-1",
		Style:           style.S().Bold(true),
		Hidden:          true,
		Disabled:        true,
		CursorNavigable: true,
		Attributes:      map[string]string{"data-test": "ok"},
		OnKeyDown:       func(*event.KeyEvent) {},
		OnKeyUp:         func(*event.KeyEvent) {},
		OnKeyPress:      func(*event.KeyEvent) {},
		OnMouseDown:     func(*event.MouseEvent) {},
		OnMouseUp:       func(*event.MouseEvent) {},
		OnMouseMove:     func(*event.MouseEvent) {},
		OnClick:         func(*event.MouseEvent) {},
		OnDrag:          func(*event.MouseEvent) {},
		OnWheel:         func(*event.WheelEvent) {},
		OnFocus:         func(*event.FocusEvent) {},
		OnBlur:          func(*event.FocusEvent) {},
		OnFocusIn:       func(*event.FocusEvent) {},
		OnFocusOut:      func(*event.FocusEvent) {},
		OnChange:        func(*event.ChangeEvent) {},
		OnInput:         func(*event.InputEvent) {},
		OnScroll:        func(*event.ScrollEvent) {},
		OnPaste:         func(*event.ClipboardEvent) {},
		OnCopy:          func(*event.ClipboardEvent) {},
		OnCut:           func(*event.ClipboardEvent) {},
		OnMouseEnter:    func(*event.MouseEvent) {},
		OnMouseLeave:    func(*event.MouseEvent) {},
		OnMouseOver:     func(*event.MouseEvent) {},
		OnMouseOut:      func(*event.MouseEvent) {},
		Ref:             ref,
	}

	type converter interface {
		elementProps() ElementProps
	}

	propTypes := []reflect.Type{
		reflect.TypeFor[ButtonProps](),
		reflect.TypeFor[CheckboxProps](),
		reflect.TypeFor[RadioGroupProps](),
		reflect.TypeFor[RadioProps](),
		reflect.TypeFor[SelectProps](),
		reflect.TypeFor[OptionProps](),
		reflect.TypeFor[InputProps](),
		reflect.TypeFor[TextAreaProps](),
		reflect.TypeFor[TDProps](),
		reflect.TypeFor[OverlayProps](),
		reflect.TypeFor[DialogProps](),
		reflect.TypeFor[FormProps](),
	}

	srcVal := reflect.ValueOf(expected)
	srcType := srcVal.Type()

	for _, pt := range propTypes {
		t.Run(pt.Name(), func(t *testing.T) {
			dst := reflect.New(pt).Elem()
			for i := range srcType.NumField() {
				sf := srcType.Field(i)
				df := dst.FieldByName(sf.Name)
				if !df.IsValid() {
					t.Fatalf("%s missing ElementProps field %q", pt.Name(), sf.Name)
				}
				if df.Type() != sf.Type {
					t.Fatalf("%s.%s has type %v, want %v", pt.Name(), sf.Name, df.Type(), sf.Type)
				}
				df.Set(srcVal.Field(i))
			}

			got := dst.Interface().(converter).elementProps()
			gotVal := reflect.ValueOf(got)
			for i := range srcType.NumField() {
				sf := srcType.Field(i)
				gf := gotVal.Field(i)
				ef := srcVal.Field(i)

				switch sf.Type.Kind() {
				case reflect.Func:
					if gf.IsNil() {
						t.Errorf("%s.elementProps().%s is nil", pt.Name(), sf.Name)
					} else if gf.Pointer() != ef.Pointer() {
						t.Errorf("%s.elementProps().%s pointer mismatch", pt.Name(), sf.Name)
					}
				case reflect.Interface:
					if gf.IsNil() || gf.Interface() != ef.Interface() {
						t.Errorf("%s.elementProps().%s interface mismatch", pt.Name(), sf.Name)
					}
				default:
					if !reflect.DeepEqual(gf.Interface(), ef.Interface()) {
						t.Errorf("%s.elementProps().%s = %v, want %v", pt.Name(), sf.Name, gf.Interface(), ef.Interface())
					}
				}
			}
		})
	}
}

func TestElementProps_NewEventListeners(t *testing.T) {
	doc := dom.NewDocument()
	container := Div(BoxProps{}).Instantiate(doc)[0].(dom.Element)

	var (
		pastedText string
		copiedText string
		cutText    string
		focusIn    int
		focusOut   int
	)

	Render(Box(BoxProps{
		OnPaste:    func(e *event.ClipboardEvent) { pastedText = e.Text() },
		OnCopy:     func(e *event.ClipboardEvent) { copiedText = e.Text() },
		OnCut:      func(e *event.ClipboardEvent) { cutText = e.Text() },
		OnFocusIn:  func(e *event.FocusEvent) { focusIn++ },
		OnFocusOut: func(e *event.FocusEvent) { focusOut++ },
	}), container)

	box := container.FirstChild().(element.Element)

	pasteEv := event.NewClipboardEvent(event.EventPaste, event.ClipboardPaste)
	pasteEv.SetText("pasted")
	box.DispatchToTarget(pasteEv)

	copyEv := event.NewClipboardEvent(event.EventCopy, event.ClipboardCopy)
	copyEv.SetText("copied")
	box.DispatchToTarget(copyEv)

	cutEv := event.NewClipboardEvent(event.EventCut, event.ClipboardCut)
	cutEv.SetText("cut")
	box.DispatchToTarget(cutEv)

	box.DispatchToTarget(event.NewFocusEvent(event.EventFocusIn, nil))
	box.DispatchToTarget(event.NewFocusEvent(event.EventFocusOut, nil))

	if pastedText != "pasted" || copiedText != "copied" || cutText != "cut" {
		t.Errorf("clipboard events mismatch: paste=%q copy=%q cut=%q", pastedText, copiedText, cutText)
	}
	if focusIn != 1 || focusOut != 1 {
		t.Errorf("focusin/focusout counts mismatch: in=%d out=%d", focusIn, focusOut)
	}

	// Clearing listeners unsubscribes cleanly.
	Render(Box(BoxProps{}), container)
	box.DispatchToTarget(event.NewFocusEvent(event.EventFocusIn, nil))
	if focusIn != 1 {
		t.Errorf("expected OnFocusIn to be unsubscribed after clearing, got %d", focusIn)
	}
}

func TestTagStyle(t *testing.T) {
	doc := dom.NewDocument()
	s := style.S().Padding(1, 2).Bold(true)

	t.Run("ContainerTag.Style", func(t *testing.T) {
		cases := []struct {
			name    string
			node    Node
			tagName string
			hasKids bool
		}{
			{"Box.Style", Box.Style(s, Text("hello")), "box", true},
			{"Span.Style", Span.Style(s, Text("world")), "span", true},
			{"Button.Style", Button.Style(s, Text("ok")), "button", true},
			{"TD.Style", TD.Style(s, Text("cell")), "td", true},
			{"Form.Style", Form.Style(s, Text("field")), "form", true},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.node.TagName() != tc.tagName {
					t.Fatalf("TagName() = %q, want %q", tc.node.TagName(), tc.tagName)
				}
				if tc.hasKids && len(tc.node.Children()) != 1 {
					t.Fatalf("len(Children()) = %d, want 1", len(tc.node.Children()))
				}
				reals := tc.node.Instantiate(doc)
				if len(reals) != 1 {
					t.Fatalf("len(Instantiate) = %d, want 1", len(reals))
				}
				el, ok := reals[0].(element.Element)
				if !ok {
					t.Fatalf("expected element.Element, got %T", reals[0])
				}
				if !el.RawStyle().Equal(s) {
					t.Errorf("RawStyle() = %+v, want %+v", el.RawStyle(), s)
				}
			})
		}
	})

	t.Run("LeafTag.Style", func(t *testing.T) {
		cases := []struct {
			name    string
			node    Node
			tagName string
		}{
			{"Input.Style", Input.Style(s), "input"},
			{"Br.Style", Br.Style(s), "br"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.node.TagName() != tc.tagName {
					t.Fatalf("TagName() = %q, want %q", tc.node.TagName(), tc.tagName)
				}
				if len(tc.node.Children()) != 0 {
					t.Fatalf("len(Children()) = %d, want 0", len(tc.node.Children()))
				}
				reals := tc.node.Instantiate(doc)
				if len(reals) != 1 {
					t.Fatalf("len(Instantiate) = %d, want 1", len(reals))
				}
				el, ok := reals[0].(element.Element)
				if !ok {
					t.Fatalf("expected element.Element, got %T", reals[0])
				}
				if !el.RawStyle().Equal(s) {
					t.Errorf("RawStyle() = %+v, want %+v", el.RawStyle(), s)
				}
			})
		}
	})

	t.Run("Nil tag guards", func(t *testing.T) {
		var nilContainer ContainerTag[BoxProps]
		if n := nilContainer.Style(s, Text("x")); n == nil || n.TagName() != "#empty" {
			t.Fatalf("nil ContainerTag.Style() = %v, want Empty()", n)
		}

		var nilLeaf LeafTag[InputProps]
		if n := nilLeaf.Style(s); n == nil || n.TagName() != "#empty" {
			t.Fatalf("nil LeafTag.Style() = %v, want Empty()", n)
		}
	})
}
