package desktop

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

const desktopContentInset = 40
const desktopControlHeight = 36

func desktopInset(c *ui.Context) float32 {
	width, _ := c.Size()
	return min(desktopContentInset, max(16, (width-640)/4))
}

func desktopInput(c *ui.Context, value *string) ui.Element {
	return ui.TextInput(c, value).Height(desktopControlHeight).PaddingX(12).Background(c.Theme().Background)
}

func desktopTextArea(c *ui.Context, value *string) ui.Element {
	return ui.TextArea(c, value).PaddingX(12).Background(c.Theme().Background)
}

func desktopSelect(c *ui.Context, value *string, choices []string) ui.Element {
	return ui.Select(c, value, choices).Height(desktopControlHeight).PaddingX(12).Background(c.Theme().Background)
}

func desktopNumeric(c *ui.Context, value *string, bytes bool, placeholder string) (ui.Element, bool) {
	if !bytes {
		field := desktopInput(c, value).TextAlign(ui.End).Keyboard(ui.KeyboardNumber).Placeholder(placeholder)
		return field, acceptNumeric(c, field, value, false)
	}
	row := ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8)
	type parts struct {
		number, unit string
	}
	draft := ui.Local(row, "byte-parts", func() parts {
		number, unit, _ := strings.Cut(strings.TrimSpace(*value), " ")
		if number == "" {
			_, unit, _ = strings.Cut(placeholder, " ")
		}
		if unit == "" {
			unit = "B"
		}
		return parts{number, unit}
	})
	var field ui.Element
	var changed bool
	row.Children(func() {
		hint, _, _ := strings.Cut(placeholder, " ")
		field = desktopInput(c, &draft.number).TextAlign(ui.End).Keyboard(ui.KeyboardDecimal).
			Grow(1).MinWidth(0).Placeholder(hint)
		numberChanged := acceptNumeric(c, field, &draft.number, true)
		unit := desktopSelect(c, &draft.unit, []string{"B", "KB", "MB", "GB", "TB"}).
			Width(90).MinWidth(0).Label("B / KB / MB / GB / TB")
		changed = numberChanged || unit.Changed()
	})
	if changed {
		*value = draft.number
		if *value != "" && draft.unit != "B" {
			*value += " " + draft.unit
		}
	}
	return field, changed
}

func acceptNumeric(c *ui.Context, field ui.Element, value *string, decimal bool) bool {
	previous := ui.Local(field, "accepted-number", func() string { return *value })
	if !field.Changed() {
		*previous = *value
		return false
	}
	if !validNumericText(*value, decimal) {
		*value = *previous
		c.Invalidate()
		return false
	}
	*previous = *value
	return true
}

func validNumericText(value string, decimal bool) bool {
	for _, char := range value {
		switch {
		case char >= '0' && char <= '9':
		case char == '.' && decimal:
			decimal = false
		default:
			return false
		}
	}
	return true
}
