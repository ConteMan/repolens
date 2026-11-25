package desktop

import (
	"strconv"
	"strings"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/egoist/mygo/ui"
)

type ruleEditor struct {
	rule                           config.RepositoryRuleSettings
	adding                         bool
	markdownExplicit, codeExplicit bool
	match, tocMin, size, theme     string
	issue                          *config.RepositoryValidationIssue
	locate                         string
	selections                     map[string]*string
	selectionEnglish               bool
	dirtyInputs                    map[string]bool
}

func newRuleEditor(rule config.RepositoryRuleSettings, adding bool, issue *config.RepositoryValidationIssue) *ruleEditor {
	e := &ruleEditor{
		rule: rule, adding: adding, dirtyInputs: make(map[string]bool),
		markdownExplicit: rule.Markdown != (config.RepositoryMarkdownOptionsSettings{}),
		codeExplicit:     rule.Code != (config.RepositoryCodeOptionsSettings{}),
	}
	if rule.Match != nil {
		e.match = *rule.Match
	}
	if rule.Markdown.TOCMinHeadings != nil {
		e.tocMin = strconv.Itoa(*rule.Markdown.TOCMinHeadings)
	}
	if rule.MaxFileSize != nil {
		e.size = strconv.FormatInt(int64(*rule.MaxFileSize), 10)
	}
	if rule.Code.Theme != nil {
		e.theme = *rule.Code.Theme
	}
	e.showIssue(issue)
	if issue == nil {
		e.locate = "match"
	}
	return e
}

func editorFieldPath(path string) string {
	if _, field, ok := strings.Cut(path, "]."); ok {
		return field
	}
	return path
}

func (e *ruleEditor) showIssue(issue *config.RepositoryValidationIssue) {
	e.issue = issue
	if issue != nil {
		e.locate = editorFieldPath(issue.Path)
	}
}

func (e *ruleEditor) candidate() (config.RepositoryRuleSettings, *config.RepositoryValidationIssue) {
	rule := e.rule
	rule.Match = new(e.match)
	rule.Markdown.TOCMinHeadings = nil
	rule.MaxFileSize = nil

	var inputIssues []config.RepositoryValidationIssue
	if text := strings.TrimSpace(e.size); text != "" {
		size := new(config.RepositoryByteSize)
		if bytes, err := strconv.ParseInt(text, 10, 64); err == nil {
			*size = config.RepositoryByteSize(bytes)
		} else if size.UnmarshalYAML([]byte(strconv.Quote(text))) != nil {
			inputIssues = append(inputIssues, config.RepositoryValidationIssue{
				Path: "rules[0].max_file_size", Code: "invalid_size", Message: "Enter a size such as 0, 512 KB, or 5 MB.",
			})
		}
		rule.MaxFileSize = size
	}
	if text := strings.TrimSpace(e.tocMin); e.markdownExplicit && text != "" {
		count, err := strconv.Atoi(text)
		if err != nil {
			inputIssues = append(inputIssues, config.RepositoryValidationIssue{
				Path: "rules[0].markdown.toc_min_headings", Code: "invalid_count", Message: "Enter a non-negative integer.",
			})
		}
		rule.Markdown.TOCMinHeadings = new(count)
	}
	if !e.markdownExplicit {
		rule.Markdown = config.RepositoryMarkdownOptionsSettings{}
	}
	if !e.codeExplicit {
		rule.Code = config.RepositoryCodeOptionsSettings{}
	} else if rule.Code.Theme != nil {
		rule.Code.Theme = new(e.theme)
	}

	issues := config.ValidateRepositorySettings(config.RepositorySettings{Rules: &[]config.RepositoryRuleSettings{rule}})
	issues = append(issues, inputIssues...)
	for _, field := range []string{"match", "markdown.toc_min_headings", "html.view", "max_file_size"} {
		for i := range issues {
			if editorFieldPath(issues[i].Path) == field {
				return rule, &issues[i]
			}
		}
	}
	if len(issues) != 0 {
		return rule, &issues[0]
	}
	return rule, nil
}

func (e *ruleEditor) view(c *ui.Context, english bool, save func(config.RepositoryRuleSettings), cancel func()) ui.Element {
	t := func(zh, en string) string {
		if english {
			return en
		}
		return zh
	}
	if e.selections == nil || e.selectionEnglish != english {
		e.selections = make(map[string]*string)
		e.selectionEnglish = english
	}
	inherit := t("继承", "Inherit")
	boolOptions := []string{inherit, t("开启", "On"), t("关闭", "Off")}

	field := func(path, label string, control func() ui.Element) {
		ui.Row(c).Key(path).FillWidth().Gap(12).Children(func() {
			labelElement := ui.Text(c, label).Bold().WidthPercent(36).Margin(8, 0, 0)
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				input := control().LabelledBy(labelElement).FillWidth().Height(36).Background(c.Theme().Background)
				if e.locate == path {
					input.Focus().ScrollIntoView()
					e.locate = ""
				}
				if e.issue != nil && editorFieldPath(e.issue.Path) == path {
					input.BorderColor(c.Theme().Danger)
					ui.Text(c, editorIssueMessage(e.issue, english)).TextColor(c.Theme().Danger)
				}
			})
		})
	}
	selectValue := func(path, current string, options []string, update func(string)) ui.Element {
		selected := e.selections[path]
		if selected == nil {
			selected = new(current)
			e.selections[path] = selected
		}
		control := ui.Select(c, selected, options)
		if control.Changed() {
			update(*selected)
			if e.issue != nil && editorFieldPath(e.issue.Path) == path {
				_, e.issue = e.candidate()
			}
			c.Invalidate()
		}
		return control
	}
	boolean := func(path, label string, target **bool) {
		field(path, label, func() ui.Element {
			selected := boolOptions[0]
			if *target != nil {
				selected = boolOptions[2]
				if **target {
					selected = boolOptions[1]
				}
			}
			return selectValue(path, selected, boolOptions, func(choice string) {
				switch choice {
				case boolOptions[0]:
					*target = nil
				case boolOptions[1]:
					*target = new(true)
				case boolOptions[2]:
					*target = new(false)
				}
			})
		})
	}
	input := func(path, label, placeholder string, value *string) {
		field(path, label, func() ui.Element {
			control := ui.TextInput(c, value).Placeholder(placeholder)
			if control.Changed() {
				e.dirtyInputs[path] = true
			}
			if e.dirtyInputs[path] && !control.Focused() {
				delete(e.dirtyInputs, path)
				_, issue := e.candidate()
				if (issue != nil && editorFieldPath(issue.Path) == path) ||
					(e.issue != nil && editorFieldPath(e.issue.Path) == path) {
					e.issue = issue
					c.Invalidate()
				}
			}
			return control
		})
	}

	group := func(path, label string, explicit *bool, render func()) {
		field(path, label, func() ui.Element {
			configure := t("自定义", "Customize")
			selected := inherit
			if *explicit {
				selected = configure
			}
			return selectValue(path, selected, []string{inherit, configure}, func(choice string) {
				*explicit = choice == configure
				if !*explicit && e.issue != nil && strings.HasPrefix(editorFieldPath(e.issue.Path), path+".") {
					_, e.issue = e.candidate()
				}
			})
		})
		if *explicit {
			ui.Column(c).Key(path+".options").FillWidth().Gap(12).Padding(0, 0, 0, 16).Children(render)
		}
	}

	title := t("编辑规则", "Edit rule")
	action := t("更新草稿", "Update draft")
	if e.adding {
		title = t("新增规则", "Add rule")
		action = t("加入草稿", "Add to draft")
	}
	return ui.Column(c).Key(e).FillWidth().Grow(1).MinHeight(0).Gap(20).Children(func() {
		ui.Text(c, title).FontSize(16).Bold()
		ui.Scroll(c).FillWidth().Height(292).Shrink(1).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().Gap(12).Padding(0, 0, 8).Children(func() {
				input("match", t("匹配路径", "Match path"), "docs/private/**", &e.match)
				boolean("render", t("生成浏览页", "Generate browser pages"), &e.rule.Render)
				group("markdown", t("Markdown 配置", "Markdown"), &e.markdownExplicit, func() {
					boolean("markdown.toc", t("目录", "Table of contents"), &e.rule.Markdown.TOC)
					input("markdown.toc_min_headings", t("目录最少标题数", "Minimum headings for table of contents"), inherit, &e.tocMin)
					boolean("markdown.anchors", t("标题锚点", "Heading anchors"), &e.rule.Markdown.Anchors)
					boolean("markdown.mermaid", "Mermaid", &e.rule.Markdown.Mermaid)
					boolean("markdown.math", t("数学公式", "Math"), &e.rule.Markdown.Math)
					boolean("markdown.frontmatter_title", t("使用 Frontmatter 标题", "Use frontmatter title"), &e.rule.Markdown.FrontmatterTitle)
					boolean("markdown.glossary", t("术语", "Glossary"), &e.rule.Markdown.Glossary)
				})
				field("html.view", t("HTML 预览", "HTML preview"), func() ui.Element {
					selected := inherit
					if e.rule.HTML.View != nil {
						selected = *e.rule.HTML.View
					}
					return selectValue("html.view", selected, []string{inherit, "embed", "direct", "source"}, func(choice string) {
						e.rule.HTML.View = nil
						if choice != inherit {
							e.rule.HTML.View = new(choice)
						}
					})
				})
				group("code", t("代码配置", "Code"), &e.codeExplicit, func() {
					boolean("code.line_numbers", t("行号", "Line numbers"), &e.rule.Code.LineNumbers)
					field("code.theme", t("代码配色", "Code theme"), func() ui.Element {
						explicit := t("指定值", "Explicit value")
						selected := inherit
						if e.rule.Code.Theme != nil {
							selected = explicit
						}
						return selectValue("code.theme", selected, []string{inherit, explicit}, func(choice string) {
							e.rule.Code.Theme = nil
							if choice == explicit {
								e.rule.Code.Theme = new(e.theme)
							}
						})
					})
					if e.rule.Code.Theme != nil {
						input("code.theme.value", t("配色名称", "Theme name"), "github", &e.theme)
					}
				})
				input("max_file_size", t("最大文件大小", "Maximum file size"), inherit, &e.size)
			})
		})
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Box(c).WidthPercent(36).Margin(0, 4, 0, 0)
			if ui.PrimaryButton(c, action).Height(36).PaddingX(14).Clicked() {
				rule, issue := e.candidate()
				if issue == nil {
					save(rule)
				} else {
					e.showIssue(issue)
					c.Invalidate()
				}
			}
			if ui.Button(c, t("取消", "Cancel")).Height(36).PaddingX(14).Clicked() {
				cancel()
			}
		})
	})
}

func editorIssueMessage(issue *config.RepositoryValidationIssue, english bool) string {
	if english {
		return issue.Message
	}
	switch editorFieldPath(issue.Path) {
	case "match":
		if issue.Code == "required" {
			return "请输入匹配路径。"
		}
		return "请输入有效的 glob，例如 docs/**/*.md。"
	case "markdown.toc_min_headings":
		return "请输入非负整数，或留空以继承。"
	case "html.view":
		return "请选择 embed、direct、source 或继承。"
	case "max_file_size":
		return "请输入非负大小，例如 0、512 KB 或 5 MB，也可留空以继承。"
	default:
		return issue.Message
	}
}
