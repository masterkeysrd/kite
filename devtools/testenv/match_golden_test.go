package testenv

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/masterkeysrd/kite/element"
	"github.com/masterkeysrd/kite/style"
)

func TestMatchGolden(t *testing.T) {
	// Use a temporary testdata directory for this test
	tmpDir, err := os.MkdirTemp("", "kite-golden-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	os.Mkdir("testdata", 0755)

	env := Default(10, 5)
	defer env.Close()

	env.Mount(element.Box().Style(style.Style{
		Width:      style.Some(style.Cells(5)),
		Height:     style.Some(style.Cells(2)),
		Background: style.Some(color.Color(color.RGBA{R: 255, G: 0, B: 0, A: 255})),
	}))
	env.Flush()

	goldenName := "test-frame"
	goldenPath := filepath.Join("testdata", goldenName+".golden")

	// 1. Create golden file
	env.MatchGolden(t, goldenName)
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Errorf("expected golden file %s to be created", goldenPath)
	}

	// 2. Successful match
	env.MatchGolden(t, goldenName)

	// 3. Intentional failure
	env.Mount(element.Box().Style(style.Style{
		Width:      style.Some(style.Cells(6)), // Changed width
		Height:     style.Some(style.Cells(2)),
		Background: style.Some(color.Color(color.RGBA{R: 0, G: 255, B: 0, A: 255})), // Changed color
	}))
	env.Flush()

	actual, expected, _, actualPath, err := env.matchGolden(goldenName)
	if err != nil {
		t.Fatalf("matchGolden failed: %v", err)
	}
	if string(actual) == string(expected) {
		t.Errorf("expected mismatch, but they match")
	}

	if _, err := os.Stat(actualPath); os.IsNotExist(err) {
		t.Errorf("expected actual output %s to be created on failure", actualPath)
	}
}

func TestDumps(t *testing.T) {
	env := Default(10, 2)
	defer env.Close()

	env.Mount(element.Box().Style(style.Style{
		Width:      style.Some(style.Cells(10)),
		Height:     style.Some(style.Cells(2)),
		Background: style.Some(color.Color(color.RGBA{R: 0, G: 0, B: 255, A: 255})),
		Foreground: style.Some(color.Color(color.RGBA{R: 255, G: 255, B: 255, A: 255})),
	}).AddChild(element.Text("Hello")))
	env.Flush()

	t.Run("DumpText", func(t *testing.T) {
		got := env.DumpText()
		if !strings.Contains(got, "Hello") {
			t.Errorf("DumpText output does not contain 'Hello':\n%s", got)
		}
	})

	t.Run("DumpANSI", func(t *testing.T) {
		got := env.DumpANSI()
		// Check for some ANSI codes
		if !strings.Contains(got, "\x1b[38;2;255;255;255m") { // White FG
			t.Errorf("DumpANSI output does not contain expected FG color code:\n%q", got)
		}
		if !strings.Contains(got, "\x1b[48;2;0;0;255m") { // Blue BG
			t.Errorf("DumpANSI output does not contain expected BG color code:\n%q", got)
		}
		// Characters might be wrapped individually, so check for them
		for _, c := range "Hello" {
			if !strings.Contains(got, string(c)) {
				t.Errorf("DumpANSI output does not contain %q", c)
			}
		}
	})

	t.Run("DumpHTML", func(t *testing.T) {
		got := env.DumpHTML()
		if !strings.Contains(got, "<!DOCTYPE html>") {
			t.Errorf("DumpHTML output does not contain doctype")
		}
		if !strings.Contains(got, "color: #ffffff") {
			t.Errorf("DumpHTML output does not contain white color")
		}
		// Characters might be wrapped individually, so check for them
		for _, c := range "Hello" {
			if !strings.Contains(got, string(c)) {
				t.Errorf("DumpHTML output does not contain %q", c)
			}
		}
	})
}
