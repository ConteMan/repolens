//go:build darwin || windows

package desktop

import (
	"fmt"
	"path/filepath"
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
			before, after := d.confirmationFields(change.before), d.confirmationFields(change.after)
			if change.before == nil {
				d.confirmationRow(c, title, func() {
					summary := valueText(after[0].value)
					if change.after.Render != nil {
						if *change.after.Render {
							summary += "  ·  " + t("生成浏览页", "Generate browser pages")
						} else {
							summary += "  ·  " + t("跳过浏览页", "Skip browser pages")
						}
					}
					ui.Text(c, summary).FillWidth().Selectable().TextColor(c.Theme().TextMuted)
				})
			} else {
				ui.Text(c, title).Bold()
			}
			if change.before != nil && change.after != nil {
				ui.Text(c, valueText(after[0].value)).FillWidth().Selectable()
			}
			valueBlock := func(label string, value *string) {
				ui.Text(c, label).TextColor(c.Theme().TextMuted)
				ui.Text(c, valueText(value)).FillWidth().Selectable()
			}
			for i := range before {
				if change.before == nil && i < 2 {
					continue
				}
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
		d.confirmationRow(c, label, func() {
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
}

func (d *desktopWindow) confirmationRow(c *ui.Context, label string, content func()) {
	ui.Row(c).FillWidth().Gap(12).Children(func() {
		ui.Text(c, label).Bold().WidthPercent(21)
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(content)
	})
}

type fieldDifference struct {
	label, before, after string
}

type confirmationGroup struct {
	change       ConfigChange
	fields       []fieldDifference
	rulesChanged bool
}

func (d *desktopWindow) confirmationGroups() []confirmationGroup {
	s := d.session
	s.mu.Lock()
	repository := s.baseline.Settings
	var external config.RepositorySettings
	if s.builderBaseline != nil {
		external = s.builderBaseline.Shared
	}
	s.mu.Unlock()
	before, _ := config.ResolveSharedSettings(repository, external)
	for _, change := range d.changes {
		if change.Path == s.BuilderPath {
			external = change.AfterSettings
		} else {
			repository = change.AfterSettings
		}
	}
	after, _ := config.ResolveSharedSettings(repository, external)
	var groups []confirmationGroup
	for _, change := range d.changes {
		group := confirmationGroup{
			change:       change,
			rulesChanged: !reflect.DeepEqual(change.BeforeSettings.Rules, change.AfterSettings.Rules),
		}
		if change.Path == s.BuilderPath {
			group.fields = d.fieldDifferences(builderFields(&change.BeforeBuilder), builderFields(&change.AfterBuilder), nil, nil)
		}
		for _, tab := range []desktopSiteTab{tabBasics, tabBrowse, tabRender, tabAgent} {
			group.fields = append(group.fields, d.fieldDifferences(
				siteFields(&change.BeforeSettings, tab), siteFields(&change.AfterSettings, tab),
				siteFields(&before, tab), siteFields(&after, tab),
			)...)
		}
		if old, next := change.BeforeSettings.Ignore, change.AfterSettings.Ignore; !reflect.DeepEqual(old, next) {
			group.fields = append(group.fields, fieldDifference{
				d.text("忽略路径", "Ignored Paths"), d.ignoreDifferenceValue(old), d.ignoreDifferenceValue(next),
			})
		}
		groups = append(groups, group)
	}
	return groups
}

func (d *desktopWindow) fieldDifferences(before, after, effectiveBefore, effectiveAfter []siteField) []fieldDifference {
	var differences []fieldDifference
	for i, field := range after {
		old, next := siteFieldValue(before[i].value), siteFieldValue(field.value)
		if reflect.DeepEqual(old, next) {
			continue
		}
		var oldEffective, nextEffective *string
		if effectiveBefore != nil {
			oldEffective = siteFieldValue(effectiveBefore[i].value)
			nextEffective = siteFieldValue(effectiveAfter[i].value)
		}
		differences = append(differences, fieldDifference{
			d.text(field.zh, field.en),
			d.differenceValue(field, old, oldEffective),
			d.differenceValue(field, next, nextEffective),
		})
	}
	return differences
}

func (d *desktopWindow) differenceValue(field siteField, explicit, effective *string) string {
	t := d.text
	value := explicit
	if value == nil {
		value = effective
	}
	if value == nil {
		return t("继承", "Inherit")
	}
	display := *value
	if display == "" {
		display = t("空字符串", "Empty string")
	}
	for _, option := range field.options {
		if option.value == display {
			display = t(option.zh, option.en)
			break
		}
	}
	if _, ok := field.value.(**bool); ok {
		if display == "true" {
			display = t("开启", "On")
		} else {
			display = t("关闭", "Off")
		}
	}
	if explicit == nil {
		display += t("（继承）", " (inherited)")
	}
	return display
}

func (d *desktopWindow) ignoreDifferenceValue(value *[]string) string {
	if value == nil {
		return d.text("继承", "Inherit")
	}
	if len(*value) == 0 {
		return d.text("空列表", "Empty list")
	}
	return strings.Join(*value, "\n")
}

func (g confirmationGroup) count() int {
	count := len(g.fields)
	if g.rulesChanged {
		count++
	}
	return count
}

func (d *desktopWindow) configurationChangeView(c *ui.Context, group confirmationGroup, expanded bool) {
	t := d.text
	change := group.change
	ui.Column(c).Key(change.Path).FillWidth().Gap(16).Children(func() {
		target := t("仓库 / .repolens.yml", "Repository / .repolens.yml")
		if change.Path == d.session.BuilderPath {
			target = t("外部配置 / ", "External configuration / ") + filepath.Base(change.Path)
		} else if !expanded && !group.rulesChanged {
			target = filepath.Base(d.session.Root) + " / .repolens.yml"
		}
		if expanded {
			count := fmt.Sprintf(t("%d 项更改", "%d changes"), group.count())
			if group.count() == 1 {
				count = t("1 项更改", "1 change")
			}
			ui.Text(c, target+"  ·  "+count).FillWidth().Bold().Tooltip(change.Path)
		} else {
			ui.Text(c, target).FillWidth().TextColor(c.Theme().TextMuted).Tooltip(change.Path)
		}
		for _, difference := range group.fields {
			if !expanded {
				d.confirmationRow(c, difference.label, func() {
					ui.Text(c, difference.before+"   →   "+difference.after).FillWidth().Selectable()
				})
				continue
			}
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				ui.Text(c, difference.label).Bold()
				d.confirmationValueRow(c, t("修改前", "Before"), difference.before, c.Theme().Surface)
				d.confirmationValueRow(c, t("修改后", "After"), difference.after, ui.RGBA(c.Theme().Accent.R, c.Theme().Accent.G, c.Theme().Accent.B, 0.08))
			})
		}
		if group.rulesChanged {
			data := ruleConfirmation{}
			if change.BeforeSettings.Rules != nil {
				data.before = *change.BeforeSettings.Rules
			}
			if change.AfterSettings.Rules != nil {
				data.after = *change.AfterSettings.Rules
			}
			d.confirmationDetails(c, data, ruleChanges(data.before, data.after))
		}

	})
}

func (d *desktopWindow) confirmationValueRow(c *ui.Context, label, value string, background ui.Color) {
	ui.Row(c).FillWidth().Gap(12).Padding(8, 10).Radius(4).Background(background).Children(func() {
		ui.Text(c, label).Width(58).TextColor(c.Theme().TextMuted)
		ui.Text(c, value).Grow(1).MinWidth(0).Selectable()
	})
}
