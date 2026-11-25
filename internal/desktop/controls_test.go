//go:build darwin || windows

package desktop

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestDesktopNumericRejectsInvalidInput(t *testing.T) {
	value := ""
	view := ui.NewTester(func(c *ui.Context) {
		field, _ := desktopNumeric(c, &value, true, "2 MB")
		field.Label("Size").AutoFocus()
	}, 450, 200)
	view.Type("1.5")
	if value != "1.5 MB" {
		t.Fatalf("expected decimal number with independent unit, got %q", value)
	}
	view.Type("MB")
	if value != "1.5 MB" {
		t.Fatalf("letters changed the number: %q", value)
	}
	view.Key(ui.Super, ui.KeyA)
	view.Key(0, ui.KeyBackspace)
	if value != "" {
		t.Fatalf("clearing the number should inherit, got %q", value)
	}
}

func TestDesktopNumericPreservesExactBytes(t *testing.T) {
	value := "9007199254740993"
	view := ui.NewTester(func(c *ui.Context) {
		field, _ := desktopNumeric(c, &value, true, "2 MB")
		field.Label("Size").AutoFocus()
	}, 450, 200)
	view.Key(ui.Super, ui.KeyA)
	view.Type("9007199254740997")
	if value != "9007199254740997" {
		t.Fatalf("expected exact byte count, got %q", value)
	}
}
