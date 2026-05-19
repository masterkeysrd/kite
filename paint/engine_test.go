package paint

import (
	"image/color"
	"testing"

	"github.com/masterkeysrd/kite/layout"
	"github.com/masterkeysrd/kite/style"
)

type mockNode struct {
	layout.Node
	s *style.Computed
}

func (m *mockNode) Style() *style.Computed { return m.s }

func TestPaint_InheritedStyle(t *testing.T) {
	pe := &PaintEngine{}

	red := color.RGBA{255, 0, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}

	tests := []struct {
		name        string
		nodeStyle   *style.Computed
		parentStyle *style.Computed
		wantFG      color.Color
		wantBG      color.Color
	}{
		{
			name: "Default style",
			nodeStyle: &style.Computed{
				Foreground: style.TerminalDefault,
				Background: color.Transparent,
			},
			wantFG: color.RGBA{255, 255, 255, 255}, // Default fallback
			wantBG: color.Transparent,
		},
		{
			name: "Explicit foreground on node",
			nodeStyle: &style.Computed{
				Foreground: red,
				Background: color.Transparent,
			},
			wantFG: red,
			wantBG: color.Transparent,
		},
		{
			name: "Inherit foreground from parent",
			nodeStyle: &style.Computed{
				Foreground: style.TerminalDefault,
				Background: color.Transparent,
			},
			parentStyle: &style.Computed{
				Foreground: blue,
			},
			wantFG: blue,
			wantBG: color.Transparent,
		},
		{
			name: "Explicit background on parent",
			nodeStyle: &style.Computed{
				Foreground: style.TerminalDefault,
				Background: color.Transparent,
			},
			parentStyle: &style.Computed{
				Background: blue,
			},
			wantFG: color.RGBA{255, 255, 255, 255},
			wantBG: blue,
		},
		{
			name: "Node background overrides parent",
			nodeStyle: &style.Computed{
				Background: red,
			},
			parentStyle: &style.Computed{
				Background: blue,
			},
			wantFG: color.RGBA{255, 255, 255, 255},
			wantBG: red,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frag := &layout.Fragment{
				Node: &mockNode{s: tt.nodeStyle},
			}
			if tt.parentStyle != nil {
				frag.ParentNode = &mockNode{s: tt.parentStyle}
			}

			gotFG, gotBG := pe.getInheritedStyle(frag)
			if gotFG != tt.wantFG {
				t.Errorf("gotFG = %v, want %v", gotFG, tt.wantFG)
			}
			if gotBG != tt.wantBG {
				t.Errorf("gotBG = %v, want %v", gotBG, tt.wantBG)
			}
		})
	}
}

func TestPaint_IsTransparent(t *testing.T) {
	tests := []struct {
		name string
		c    color.Color
		want bool
	}{
		{"Nil", nil, true},
		{"Transparent", color.Transparent, true},
		{"RGBA 0", color.RGBA{0, 0, 0, 0}, true},
		{"RGBA 1", color.RGBA{0, 0, 0, 1}, false},
		{"Opaque Red", color.RGBA{255, 0, 0, 255}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTransparent(tt.c); got != tt.want {
				t.Errorf("isTransparent() = %v, want %v", got, tt.want)
			}
		})
	}
}
