//go:build darwin || windows

package desktop

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

type yamlEdit struct {
	kind byte
	line string
}

func yamlPreviewLines(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(value, "\n"), "\n")
}

func yamlEdits(before, after []string) []yamlEdit {
	var edits []yamlEdit
	for len(before) > 0 && len(after) > 0 && before[0] == after[0] {
		edits = append(edits, yamlEdit{' ', before[0]})
		before, after = before[1:], after[1:]
	}
	commonEnd := 0
	for commonEnd < len(before) && commonEnd < len(after) && before[len(before)-1-commonEnd] == after[len(after)-1-commonEnd] {
		commonEnd++
	}
	tail := before[len(before)-commonEnd:]
	before, after = before[:len(before)-commonEnd], after[:len(after)-commonEnd]

	// Cap the alignment table for unusually large replacements.
	if len(before) > 0 && len(after) > 0 && len(before) <= 1_000_000/len(after) {
		stride := len(after) + 1
		lengths := make([]int, (len(before)+1)*stride)
		for i := len(before) - 1; i >= 0; i-- {
			for j := len(after) - 1; j >= 0; j-- {
				at := i*stride + j
				if before[i] == after[j] {
					lengths[at] = 1 + lengths[at+stride+1]
				} else {
					lengths[at] = max(lengths[at+stride], lengths[at+1])
				}
			}
		}
		i, j := 0, 0
		for i < len(before) && j < len(after) {
			switch {
			case before[i] == after[j]:
				edits = append(edits, yamlEdit{' ', before[i]})
				i++
				j++
			case lengths[(i+1)*stride+j] >= lengths[i*stride+j+1]:
				edits = append(edits, yamlEdit{'-', before[i]})
				i++
			default:
				edits = append(edits, yamlEdit{'+', after[j]})
				j++
			}
		}
		before, after = before[i:], after[j:]
	}
	for _, line := range before {
		edits = append(edits, yamlEdit{'-', line})
	}
	for _, line := range after {
		edits = append(edits, yamlEdit{'+', line})
	}
	for _, line := range tail {
		edits = append(edits, yamlEdit{' ', line})
	}
	return edits
}

func confirmationYAMLDiff(before, after string) string {
	edits := yamlEdits(yamlPreviewLines(before), yamlPreviewLines(after))
	const context = 3
	var diff strings.Builder
	oldLine, newLine := 1, 1
	for cursor := 0; cursor < len(edits); {
		first := cursor
		for first < len(edits) && edits[first].kind == ' ' {
			first++
		}
		if first == len(edits) {
			break
		}
		start, end := max(cursor, first-context), min(len(edits), first+context+1)
		last := first
		for i := first + 1; i < len(edits) && i <= last+2*context+1; i++ {
			if edits[i].kind != ' ' {
				last = i
				end = min(len(edits), i+context+1)
			}
		}
		oldLine += start - cursor
		newLine += start - cursor
		oldCount, newCount := 0, 0
		for _, edit := range edits[start:end] {
			if edit.kind != '+' {
				oldCount++
			}
			if edit.kind != '-' {
				newCount++
			}
		}
		fmt.Fprintf(&diff, "@@ -%s +%s @@\n", yamlDiffRange(oldLine, oldCount), yamlDiffRange(newLine, newCount))
		for _, edit := range edits[start:end] {
			fmt.Fprintf(&diff, "%c%s\n", edit.kind, edit.line)
		}
		oldLine += oldCount
		newLine += newCount
		cursor = end
	}
	return strings.TrimSuffix(diff.String(), "\n")
}

func yamlDiffRange(start, count int) string {
	if count == 0 {
		start--
	}
	return fmt.Sprintf("%d,%d", start, count)
}

func (d *desktopWindow) yamlConfirmationGroup(c *ui.Context, change ConfigChange) {
	theme := c.Theme()
	ui.Column(c).Key(change.Path).FillWidth().Gap(8).Children(func() {
		ui.Text(c, change.Path).FillWidth().Bold().Selectable()
		diff := confirmationYAMLDiff(change.Before, change.After)
		if diff == "" {
			ui.Text(c, d.text("无 YAML 文本差异。", "No YAML text changes.")).TextColor(theme.TextMuted)
			return
		}
		ui.Column(c).FillWidth().MinWidth(0).Padding(12).Radius(6).Background(theme.Surface).Children(func() {
			ui.Text(c, diff).Font("monospace").FillWidth().MinWidth(0).Selectable()
		})
	})
}
