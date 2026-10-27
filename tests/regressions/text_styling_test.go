package regressions

import (
	"testing"

	"github.com/masterkeysrd/kite/backend"
	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
	"github.com/masterkeysrd/kite/testenv"
)

func TestTextStyling_BoldAndAttributes(t *testing.T) {
	env := testenv.Default(40, 10)
	defer env.Close()

	root := element.Box(
		element.Box("Normal Text"),
		element.Box("Bold Text").Style(style.S().Bold(true)),
		element.Box("Italic Text").Style(style.S().Italic(true)),
		element.Box("Underline Text").Style(style.S().Underline(true)),
		element.Box("Strikethrough Text").Style(style.S().Strikethrough(true)),
	)

	env.Mount(root)
	env.Flush()

	frame := env.Backend.LastFrame()
	if frame.Surface == nil {
		t.Fatal("expected frame surface, got nil")
	}

	fb := frame.Surface

	// Row 0: Normal Text
	normalCell := fb.CellAt(0, 0)
	if normalCell.Content != "N" {
		t.Fatalf("expected 'N' at (0, 0), got %q", normalCell.Content)
	}
	if normalCell.Style&backend.CellBold != 0 {
		t.Errorf("expected normal text to not be bold, got style %v", normalCell.Style)
	}

	// Row 1: Bold Text
	boldCell := fb.CellAt(0, 1)
	if boldCell.Content != "B" {
		t.Fatalf("expected 'B' at (0, 1), got %q", boldCell.Content)
	}
	if boldCell.Style&backend.CellBold == 0 {
		t.Errorf("expected bold text to have CellBold, got style %v", boldCell.Style)
	}

	// Row 2: Italic Text
	italicCell := fb.CellAt(0, 2)
	if italicCell.Content != "I" {
		t.Fatalf("expected 'I' at (0, 2), got %q", italicCell.Content)
	}
	if italicCell.Style&backend.CellItalic == 0 {
		t.Errorf("expected italic text to have CellItalic, got style %v", italicCell.Style)
	}

	// Row 3: Underline Text
	underlineCell := fb.CellAt(0, 3)
	if underlineCell.Content != "U" {
		t.Fatalf("expected 'U' at (0, 3), got %q", underlineCell.Content)
	}
	if underlineCell.Style&backend.CellUnderline == 0 {
		t.Errorf("expected underline text to have CellUnderline, got style %v", underlineCell.Style)
	}

	// Row 4: Strikethrough Text
	strikeCell := fb.CellAt(0, 4)
	if strikeCell.Content != "S" {
		t.Fatalf("expected 'S' at (0, 4), got %q", strikeCell.Content)
	}
	if strikeCell.Style&backend.CellStrikethrough == 0 {
		t.Errorf("expected strikethrough text to have CellStrikethrough, got style %v", strikeCell.Style)
	}
}
