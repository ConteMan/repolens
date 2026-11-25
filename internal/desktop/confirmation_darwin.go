package desktop

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/egoist/mygo/ui"
)

type ruleChange struct {
	before, after *config.RepositoryRuleSettings
}

type ruleConfirmation struct {
	target        string
	before, after []config.RepositoryRuleSettings
}

func (s *Session) confirmation() ruleConfirmation {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.candidate == nil {
		return ruleConfirmation{}
	}
	return ruleConfirmation{s.candidate.Path, documentRules(s.baseline), documentRules(s.candidate)}
}

func ruleChanges(before, after []config.RepositoryRuleSettings) []ruleChange {
	// Reserve exact matches before pairing edits to the same pattern. This
	// keeps repeated patterns and pure reordering from looking like edits.
	matches := make([]int, len(after))
	used := make([]bool, len(before))
	for i := range after {
		matches[i] = -1
		for j := range before {
			if !used[j] && reflect.DeepEqual(before[j], after[i]) {
				matches[i], used[j] = j, true
				break
			}
		}
	}
	var changes []ruleChange
	for i := range after {
		if matches[i] >= 0 {
			continue
		}
		change := ruleChange{after: &after[i]}
		for j := range before {
			if !used[j] && reflect.DeepEqual(before[j].Match, after[i].Match) {
				change.before, used[j] = &before[j], true
				break
			}
		}
		changes = append(changes, change)
	}
	for j := range before {
		if !used[j] {
			changes = append(changes, ruleChange{before: &before[j]})
		}
	}
	return changes
}

type confirmationField struct {
	label string
	value *string
}

func formatConfirmationValue[T any](value *T) *string {
	if value == nil {
		return nil
	}
	return new(fmt.Sprint(*value))
}

func (d *desktopWindow) confirmationFields(rule *config.RepositoryRuleSettings) []confirmationField {
	if rule == nil {
		rule = &config.RepositoryRuleSettings{}
	}
	t := d.text
	boolean := func(value *bool) *string {
		if value == nil {
			return nil
		}
		if *value {
			return new(t("开启", "On"))
		}
		return new(t("关闭", "Off"))
	}
	return []confirmationField{
		{t("匹配路径", "Match path"), rule.Match},
		{t("生成浏览页", "Generate browser pages"), boolean(rule.Render)},
		{t("Markdown 目录", "Markdown table of contents"), boolean(rule.Markdown.TOC)},
		{t("目录最少标题数", "Minimum headings for table of contents"), formatConfirmationValue(rule.Markdown.TOCMinHeadings)},
		{t("标题锚点", "Heading anchors"), boolean(rule.Markdown.Anchors)},
		{"Mermaid", boolean(rule.Markdown.Mermaid)},
		{t("数学公式", "Math"), boolean(rule.Markdown.Math)},
		{t("使用 Frontmatter 标题", "Use frontmatter title"), boolean(rule.Markdown.FrontmatterTitle)},
		{t("术语", "Glossary"), boolean(rule.Markdown.Glossary)},
		{t("HTML 预览", "HTML preview"), rule.HTML.View},
		{t("代码行号", "Code line numbers"), boolean(rule.Code.LineNumbers)},
		{t("代码配色", "Code theme"), rule.Code.Theme},
		{t("最大文件大小（字节）", "Maximum file size (bytes)"), formatConfirmationValue(rule.MaxFileSize)},
	}
}

func (d *desktopWindow) confirmationDetails(c *ui.Context, data ruleConfirmation, changes []ruleChange) {
	t := d.text
	valueText := func(value *string) string {
		if value == nil {
			return t("继承", "Inherit")
		}
		if *value == "" {
			return t("空字符串", "Empty string")
		}
		return *value
	}
	for _, change := range changes {
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			title := t("修改规则", "Edited rule")
			switch {
			case change.before == nil:
				title = t("新增规则", "Added rule")
			case change.after == nil:
				title = t("删除规则", "Removed rule")
			}
			ui.Text(c, title).Bold()
			before, after := d.confirmationFields(change.before), d.confirmationFields(change.after)
			if change.before != nil && change.after != nil {
				ui.Text(c, valueText(after[0].value)).FillWidth().Selectable()
			}
			valueBlock := func(label string, value *string) {
				ui.Text(c, label).TextColor(c.Theme().TextMuted)
				ui.Text(c, valueText(value)).FillWidth().Selectable()
			}
			for i := range before {
				if reflect.DeepEqual(before[i].value, after[i].value) {
					continue
				}
				ui.Column(c).FillWidth().Gap(4).Children(func() {
					ui.Text(c, after[i].label).Bold()
					switch {
					case change.before == nil:
						ui.Text(c, valueText(after[i].value)).FillWidth().Selectable()
					case change.after == nil:
						ui.Text(c, valueText(before[i].value)).FillWidth().Selectable()
					default:
						valueBlock(t("修改前", "Before"), before[i].value)
						valueBlock(t("修改后", "After"), after[i].value)
					}
				})
			}
		})
	}
	ordered := func(label string, rules []config.RepositoryRuleSettings) {
		ui.Column(c).FillWidth().Gap(6).Children(func() {
			ui.Text(c, label).Bold()
			if len(rules) == 0 {
				ui.Text(c, t("无规则", "No rules")).TextColor(c.Theme().TextMuted)
			}
			counts := make(map[string]int, len(rules))
			for _, rule := range rules {
				counts[valueText(rule.Match)]++
			}
			for index, rule := range rules {
				pattern := valueText(rule.Match)
				ui.Text(c, strconv.Itoa(index+1)+"    "+pattern).FillWidth().Selectable()
				if counts[pattern] < 2 {
					continue
				}
				for _, field := range d.confirmationFields(&rule)[1:] {
					if field.value != nil {
						ui.Row(c).FillWidth().Gap(6).PaddingX(20).Children(func() {
							ui.Text(c, field.label).TextColor(c.Theme().TextMuted)
							ui.Text(c, valueText(field.value)).Grow(1).MinWidth(0).Selectable()
						})
					}
				}
			}
		})
	}
	if len(changes) == 0 && !slices.EqualFunc(data.before, data.after, func(a, b config.RepositoryRuleSettings) bool {
		return reflect.DeepEqual(a, b)
	}) {
		ordered(t("原有顺序", "Previous order"), data.before)
	}
	ordered(t("最终顺序", "Final order"), data.after)
	raw := ui.Local(ui.Box(c).Size(0, 0), "raw", func() bool { return false })
	ui.Collapsible(c, t("查看 YAML 差异", "View YAML diff"), raw, func() {
		ui.Text(c, strings.TrimSuffix(d.diff, "\n")).Font("monospace").FillWidth().Selectable()
	})
}
