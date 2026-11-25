package desktop

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/goccy/go-yaml"
)

type siteField struct {
	key, zh, en string
	value       any
	example     string
	options     []siteOption
}

type siteOption struct{ value, zh, en string }

func siteFields(settings *config.RepositorySettings, tab desktopSiteTab) []siteField {
	s := settings
	switch tab {
	case tabBasics:
		return []siteField{
			{"site.title", "站点名称", "Site Title", &s.Site.Title, "", nil},
			{"site.language", "站点语言", "Site Language", &s.Site.Language, "", []siteOption{{"zh-CN", "简体中文", "Simplified Chinese"}, {"en", "English", "English"}}},
			{"site.home", "首页文件", "Home File", &s.Site.Home, "README.md", nil},
		}
	case tabBrowse:
		return []siteField{
			{"view.directory-files", "目录文件", "Directory Files", &s.View.DirectoryFiles, "", []siteOption{{"auto", "自动", "Auto"}, {"expanded", "展开", "Expanded"}, {"collapsed", "折叠", "Collapsed"}}},
			{"view.content-width", "正文宽度", "Content Width", &s.View.ContentWidth, "", []siteOption{{"standard", "标准", "Standard"}, {"narrow", "紧凑", "Narrow"}, {"full", "全宽", "Full"}}},
			{"view.content-align", "正文对齐", "Content Alignment", &s.View.ContentAlign, "", []siteOption{{"left", "靠左", "Left"}, {"center", "居中", "Center"}, {"right", "靠右", "Right"}}},
			{"view.tree_position", "文件树位置", "File Tree Position", &s.View.TreePosition, "", []siteOption{{"left", "左侧", "Left"}, {"right", "右侧", "Right"}}},
			{"view.tree_expand_depth", "默认展开深度", "Default Expansion Depth", &s.View.TreeExpandDepth, "2", nil},
			{"view.toc_panel", "页内目录", "On-page Table of Contents", &s.View.TOCPanel, "", []siteOption{{"floating", "浮动", "Floating"}, {"inline", "行内", "Inline"}}},
			{"view.search", "站内搜索", "Site Search", &s.View.Search, "", nil},
			{"theme.vars", "主题变量", "Theme Variables", &s.Theme.Vars, "", nil},
			{"theme.css", "自定义 CSS", "Custom CSS", &s.Theme.CSS, ".repolens/theme.css", nil},
			{"theme.templates", "自定义模板", "Custom Templates", &s.Theme.Templates, ".repolens/templates", nil},
		}
	case tabRender:
		return []siteField{
			{"render.render", "生成浏览页", "Generate Browser Pages", &s.Render.Render, "", nil},
			{"render.markdown.toc", "Markdown 目录", "Markdown Table of Contents", &s.Render.Markdown.TOC, "", nil},
			{"render.markdown.anchors", "标题锚点", "Heading Anchors", &s.Render.Markdown.Anchors, "", nil},
			{"render.markdown.mermaid", "Mermaid", "Mermaid", &s.Render.Markdown.Mermaid, "", nil},
			{"render.markdown.math", "数学公式", "Math", &s.Render.Markdown.Math, "", nil},
			{"render.markdown.toc_min_headings", "目录最少标题数", "Minimum Heading Count", &s.Render.Markdown.TOCMinHeadings, "3", nil},
			{"render.markdown.frontmatter_title", "使用 Frontmatter 标题", "Use Frontmatter Title", &s.Render.Markdown.FrontmatterTitle, "", nil},
			{"render.markdown.glossary", "术语标注", "Glossary Annotations", &s.Render.Markdown.Glossary, "", nil},
			{"glossary.dir", "术语库目录", "Glossary Directory", &s.Glossary.Dir, ".repolens/glossary", nil},
			{"glossary.strict", "术语校验", "Glossary Validation", &s.Glossary.Strict, "", []siteOption{{"off", "关闭", "Off"}, {"refs", "引用检查", "References"}, {"complete", "完整检查", "Complete"}}},
			{"render.html.view", "HTML 预览", "HTML Preview", &s.Render.HTML.View, "", []siteOption{{"embed", "内嵌", "Embedded"}, {"direct", "直接打开", "Direct"}, {"source", "源码", "Source"}}},
			{"render.code.line_numbers", "代码行号", "Code Line Numbers", &s.Render.Code.LineNumbers, "", nil},
			{"render.code.theme", "代码配色", "Code Theme", &s.Render.Code.Theme, "github", nil},
			{"render.max_file_size", "最大文件大小", "Maximum File Size", &s.Render.MaxFileSize, "5 MB", nil},
		}
	case tabAgent:
		return []siteField{
			{"agent.llms_txt", "llms.txt", "llms.txt", &s.Agent.LLMSTxt, "", nil},
			{"agent.llms_full.enabled", "llms-full.txt", "llms-full.txt", &s.Agent.LLMSFull.Enabled, "", nil},
			{"agent.llms_full.max_size", "全文大小上限", "Full-text Size Limit", &s.Agent.LLMSFull.MaxSize, "2 MB", nil},
			{"agent.index_json", "index.json", "index.json", &s.Agent.IndexJSON, "", nil},
		}
	}
	return nil
}

func builderFields(s *config.BuilderSettings) []siteField {
	return []siteField{
		{"source.repo", "Git 仓库", "Git Repository", &s.Source.Repo, "", nil},
		{"source.ref", "Git 引用", "Git Ref", &s.Source.Ref, "", nil},
		{"output.dir", "输出目录", "Output Directory", &s.Output.Dir, "dist", nil},
		{"output.base_path", "部署基础路径", "Deployment Base Path", &s.Output.BasePath, "/", nil},
		{"access.noindex", "禁止索引", "Disable Indexing", &s.Access.NoIndex, "", nil},
	}
}

func (d *desktopWindow) sitePage(c *ui.Context, tab desktopSiteTab) {
	ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
		ui.Column(c).Key(struct {
			session  *Session
			tab      desktopSiteTab
			path     string
			external bool
		}{d.session, tab, d.session.BuilderPath, d.externalTarget}).FillWidth().Padding(20, desktopInset(c), 24).Gap(12).Children(func() {
			fields := siteFields(d.sharedDraft(), tab)
			if tab == tabBuilder {
				s := d.session
				ui.Row(c).FillWidth().Gap(12).Children(func() {
					ui.Text(c, d.text("外部配置文件", "External Configuration")).Bold().WidthPercent(36).Margin(8, 0, 0)
					if s.BuilderPath != "" {
						ui.Text(c, s.BuilderPath).Grow(1).MinWidth(0).Selectable()
					}
					if ui.Button(c, d.text("选择文件", "Choose File")).Height(36).Disabled(d.window == nil || d.picking).Clicked() {
						d.chooseBuilderFile(c)
					}
				})
				if s.BuilderPath == "" {
					ui.Text(c, d.text("选择构建者配置文件后即可编辑。", "Choose a builder configuration file to edit it.")).TextColor(c.Theme().TextMuted)
					return
				}
				fields = builderFields(&s.Builder)
			}
			effective, effectiveErr := config.ResolveSharedSettings(d.session.Settings, d.session.ExternalSettings)
			resolved := siteFields(&effective, tab)
			repository := siteFields(&d.session.Settings, tab)
			external := siteFields(&d.session.ExternalSettings, tab)
			for index, field := range fields {
				display, source := "", ""
				if effectiveErr == nil && tab != tabBuilder {
					display, source = d.effectiveField(resolved[index], repository[index], external[index])
				}
				switch field.key {
				case "theme.vars":
					d.themeVariables(c, field, source)
				case "theme.css":
					d.themeAssets(c, field, fields[index+1])
				case "theme.templates":
				default:
					d.editSiteField(c, field, display, source)
				}
			}
		})
	})
}

func (d *desktopWindow) chooseBuilderFile(c *ui.Context) {
	if d.picking {
		return
	}
	s := d.session
	title := d.text("选择构建者配置文件", "Choose Builder Configuration")
	d.picking = true
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: d.window, Title: title})
		mygo.RunOnMain(func() {
			d.picking = false
			defer d.window.Invalidate()
			if d.session != s {
				return
			}
			if err != nil {
				d.saveError = err
				return
			}
			if len(paths) == 0 || paths[0] == s.BuilderPath {
				return
			}
			load := func() {
				d.saveError = s.LoadBuilder(paths[0])
				if d.saveError == nil {
					for key := range d.siteInputErrors {
						if d.externalInputKey(key) {
							delete(d.siteInputErrors, key)
						}
					}
					d.saved = false
					d.updateDocumentState()
				}
			}
			dirty := s.BuilderDirty()
			for key := range d.siteInputErrors {
				dirty = dirty || d.externalInputKey(key)
			}
			if dirty {
				d.askDiscard(load)
			} else {
				load()
			}
		})
	}()
	c.Invalidate()
}

func (d *desktopWindow) editSiteField(c *ui.Context, field siteField, display, source string) {
	ui.Row(c).Key(field.key).FillWidth().Gap(12).Children(func() {
		var label ui.Element
		ui.Row(c).WidthPercent(36).Gap(6).AlignItems(ui.Center).Margin(8, 0, 0).Children(func() {
			label = ui.Text(c, d.text(field.zh, field.en)).Bold()
			if source != "" {
				label.Tooltip(source)
			}
			hint := d.siteFieldHint(field.key)
			if hint != "" && source != "" {
				hint += "\n" + source
			}
			if hint != "" {
				ui.ButtonBase(c).Size(16, 16).Center().Border(1, c.Theme().TextMuted).Radius(8).
					Label(d.text("字段说明", "Field help")).Tooltip(hint).Children(func() {
					ui.Text(c, "i").FontSize(11).Bold().TextColor(c.Theme().TextMuted)
				})
			}
		})
		box := ui.Column(c).Grow(1).MinWidth(0).Gap(4)
		box.Children(func() {
			inherit := d.text("继承", "Inherit")
			inheritedLabel := inherit
			if display != "" {
				inheritedLabel = display + " · " + inherit
			}
			changed := func() {
				d.saved = false
				d.saveError = nil
				d.updateDocumentState()
				c.Invalidate()
			}
			selectValue := func(value string, options []string, update func(string)) {
				selected := ui.Local(box, "choice", func() string { return value })
				control := desktopSelect(c, selected, options).LabelledBy(label).FillWidth()
				if control.Changed() {
					update(*selected)
				} else if *selected != value {
					*selected = value
					c.Invalidate()
				}
				d.focusSiteField(control, field.key)
			}
			switch value := field.value.(type) {
			case **bool:
				if strings.HasPrefix(field.key, "render.markdown.") && field.key != "render.markdown.frontmatter_title" {
					state := ui.Local(box, "checkbox", func() ui.CheckState { return ui.CheckMixed })
					ui.Row(c).Gap(8).AlignItems(ui.Center).MinHeight(36).Children(func() {
						control := ui.CheckboxState(c, state, "").LabelledBy(label)
						if control.Changed() {
							*value = new(*state == ui.CheckOn)
							changed()
						} else {
							*state = ui.CheckMixed
							if *value != nil {
								*state = ui.CheckOff
								if **value {
									*state = ui.CheckOn
								}
							}
						}
						d.focusSiteField(control, field.key)
						if *value == nil {
							ui.Text(c, inheritedLabel).TextColor(c.Theme().TextMuted)
						} else if ui.Button(c, d.text("恢复继承", "Reset to Inherit")).Clicked() {
							*value = nil
							changed()
						}
					})
					return
				}
				on, off := d.text("开启", "On"), d.text("关闭", "Off")
				current := inheritedLabel
				if *value != nil {
					current = off
					if **value {
						current = on
					}
				}
				selectValue(current, []string{inheritedLabel, on, off}, func(choice string) {
					*value = nil
					if choice != inheritedLabel {
						*value = new(choice == on)
					}
					changed()
				})
				return
			case **string:
				if field.key == "render.code.theme" {
					current := inheritedLabel
					if *value != nil {
						current = **value
					}
					query := ui.Local(box, "theme", func() string { return current })
					chooser := ui.ComboboxBase(c, query)
					input := chooser.Input.LabelledBy(label).Height(36).FillWidth().PaddingX(12).
						Border(1, c.Theme().Border).Radius(6).AutoComplete(ui.AutoCompleteList).
						Placeholder(d.text("搜索代码配色", "Search code themes"))
					d.focusSiteField(input, field.key)
					chooser.Popup(func(panel ui.Element) {
						panel.Padding(4).Background(c.Theme().Background).Border(1, c.Theme().Border).Radius(8)
						ui.Scroll(c).FillWidth().MaxHeight(200).Children(func() {
							names := append([]string{inheritedLabel}, styles.Names()...)
							filter := strings.ToLower(*query)
							count := 0
							for _, option := range names {
								if chooser.Filtering() && !strings.Contains(strings.ToLower(option), filter) {
									continue
								}
								count++
								item := chooser.Item(option).Padding(6, 10).Radius(4)
								if item.Highlighted() {
									item.Background(c.Theme().Accent).TextColor(c.Theme().AccentText)
								}
								item.Children(func() { ui.Text(c, option) })
							}
							if count == 0 {
								ui.Text(c, d.text("没有匹配的配色", "No matching themes")).Padding(6, 10)
							}
						})
					})
					if choice, ok := chooser.Chosen(); ok {
						*value = nil
						if choice != inheritedLabel {
							*value = new(choice)
						}
						*query = choice
						changed()
					} else if !chooser.Open() && !input.Focused() && *query != current {
						*query = current
						c.Invalidate()
					}
					return
				}
				if field.key == "view.tree_position" {
					current := ""
					if *value != nil {
						current = **value
					}
					selected := ui.Local(box, "radio", func() string { return current })
					ui.RadioGroup(c, func() {
						ui.Row(c).AlignItems(ui.Center).Wrap().Gap(12).MinHeight(36).Children(func() {
							for _, option := range append([]siteOption{{"", "继承", "Inherit"}}, field.options...) {
								control := ui.Radio(c.Key(option.value), selected, option.value, d.text(option.zh, option.en))
								if control.Changed() {
									*value = nil
									if *selected != "" {
										*value = new(*selected)
									}
									changed()
								}
								if option.value == *selected {
									d.focusSiteField(control, field.key)
								}
							}
						})
					}).LabelledBy(label)
					return
				}
				if len(field.options) > 0 {
					current := inheritedLabel
					options := []string{inheritedLabel}
					byLabel := map[string]string{}
					for _, item := range field.options {
						text := d.text(item.zh, item.en)
						options = append(options, text)
						byLabel[text] = item.value
						if *value != nil && **value == item.value {
							current = text
						}
					}
					if *value != nil && current == inheritedLabel {
						current = **value
						options = append(options, current)
						byLabel[current] = current
					}
					selectValue(current, options, func(choice string) {
						*value = nil
						if choice != inheritedLabel {
							*value = new(byLabel[choice])
						}
						changed()
					})
					return
				}
			}
			initial := siteFieldValue(field.value)
			text := ui.Local(box, "text", func() string {
				if raw, invalid := d.siteInputErrors[d.siteErrorKey(field.key)]; invalid {
					return raw
				}
				if initial == nil {
					return ""
				}
				return *initial
			})
			placeholder := inheritedLabel
			if display != "" {
				placeholder = display
			} else if field.example != "" {
				placeholder += d.text("（例如 ", " (for example ") + field.example + d.text("）", ")")
			}
			var input ui.Element
			var inputChanged bool
			switch field.value.(type) {
			case **config.RepositoryByteSize:
				input, inputChanged = desktopNumeric(c, text, true, placeholder)
			case **int:
				input, inputChanged = desktopNumeric(c, text, false, placeholder)
				input.FillWidth()
			default:
				input = desktopInput(c, text).FillWidth().Placeholder(placeholder)
				inputChanged = input.Changed()
			}
			input.LabelledBy(label)
			d.focusSiteField(input, field.key)
			if inputChanged {
				if updateSiteField(field.value, *text) {
					delete(d.siteInputErrors, d.siteErrorKey(field.key))
				} else {
					if d.siteInputErrors == nil {
						d.siteInputErrors = make(map[string]string)
					}
					d.siteInputErrors[d.siteErrorKey(field.key)] = *text
				}
				changed()
			}
			if raw, invalid := d.siteInputErrors[d.siteErrorKey(field.key)]; invalid {
				message := d.siteFieldError(field, raw)
				input.Invalid(true).BorderColor(c.Theme().Danger)
				input.DescribedBy(ui.Text(c, message).TextColor(c.Theme().Danger))
			}
		})
	})
}

func (d *desktopWindow) siteFieldHint(key string) string {
	switch key {
	case "glossary.strict":
		return d.text("引用检查会验证术语引用。完整检查还会要求术语条目完整。", "References checks term references. Complete also requires complete glossary entries.")
	case "agent.llms_full.max_size":
		return d.text("控制 llms-full.txt 汇总正文的大小上限。", "Limits the total text collected in llms-full.txt.")
	}
	return ""
}

func (d *desktopWindow) siteFieldError(field siteField, raw string) string {
	switch field.value.(type) {
	case **config.RepositoryByteSize:
		if strings.HasPrefix(strings.TrimSpace(raw), "-") {
			return d.text("文件大小不能为负数。请输入 0 或更大的值。", "File size cannot be negative. Enter 0 or a larger value.")
		}
		return d.text("请输入有效的文件大小，例如 0、512 KB 或 5 MB。", "Enter a valid file size such as 0, 512 KB, or 5 MB.")
	case **int:
		return d.text("请输入非负整数。", "Enter a non-negative integer.")
	case **map[string]string:
		return d.text("每行请输入一个键和值，例如 accent: '#3752b5'。", "Enter a key and value on each line, for example accent: '#3752b5'.")
	}
	return d.text("请输入有效的配置值。", "Enter a valid configuration value.")
}

func (d *desktopWindow) focusSiteField(control ui.Element, key string) {
	if d.siteFocus == key {
		control.Focus().ScrollIntoView()
		d.siteFocus = ""
	}
}

func siteFieldValue(value any) *string {
	switch v := value.(type) {
	case **string:
		return *v
	case **int:
		return formatConfirmationValue(*v)
	case **config.RepositoryByteSize:
		if *v != nil {
			return new(displayByteSize(int64(**v)))
		}
	case **bool:
		return formatConfirmationValue(*v)
	case **map[string]string:
		if *v != nil {
			raw, err := yaml.Marshal(**v)
			if err == nil {
				return new(strings.TrimSuffix(string(raw), "\n"))
			}
		}
	}
	return nil
}

func displayByteSize(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unit := 0
	for bytes != 0 && bytes%1024 == 0 && unit < len(units)-1 {
		bytes /= 1024
		unit++
	}
	if unit == 0 {
		return strconv.FormatInt(bytes, 10)
	}
	return strconv.FormatInt(bytes, 10) + " " + units[unit]
}

func (d *desktopWindow) themeRow(c *ui.Context, label, action, hint string, actionState *bool) ui.Element {
	row := ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center)
	row.Children(func() {
		ui.Text(c, label).Bold().WidthPercent(36).Tooltip(hint)
		if ui.Button(c, action).Grow(1).Height(36).Clicked() {
			*actionState = true
		}
	})
	return row
}

func (d *desktopWindow) themeVariables(c *ui.Context, field siteField, hint string) {
	type draft struct {
		open, bad bool
		text      string
	}
	begin := false
	row := d.themeRow(c.Key(field.key), d.text(field.zh, field.en), d.text("编辑变量…", "Edit Variables…"), hint, &begin)
	edit := ui.Local(row, "draft", func() draft { return draft{} })
	if begin {
		edit.open, edit.bad, edit.text = true, false, ""
		if value := siteFieldValue(field.value); value != nil {
			edit.text = *value
		}
		c.Invalidate()
	}
	panel := ui.Modal(c.Key("theme-vars"), &edit.open, func() {
		ui.Column(c).Width(500).MaxWidthPercent(100).Gap(12).Children(func() {
			ui.Text(c, d.text("编辑主题变量", "Edit Theme Variables")).Bold().FontSize(16)
			ui.Text(c, d.text("每行输入一个键和值。清空内容可恢复继承。", "Enter one key and value per line. Clear the content to inherit.")).TextColor(c.Theme().TextMuted)
			input := desktopTextArea(c, &edit.text).Height(160).FillWidth().Placeholder("accent: '#3752b5'").
				Label(d.text("主题变量", "Theme Variables")).AutoFocus()
			if edit.bad {
				input.Invalid(true).BorderColor(c.Theme().Danger)
				input.DescribedBy(ui.Text(c, d.siteFieldError(field, edit.text)).TextColor(c.Theme().Danger))
			}
			d.themeEditorActions(c, &edit.open, func() bool {
				edit.bad = !updateSiteField(field.value, edit.text)
				return !edit.bad
			})
		})
	})
	panel.Padding(24).Radius(12)
}

func (d *desktopWindow) themeAssets(c *ui.Context, css, templates siteField) {
	type draft struct {
		open  bool
		paths [2]string
		err   error
	}
	fields := [2]siteField{css, templates}
	begin := false
	row := d.themeRow(c.Key("theme-assets"), d.text("自定义 CSS / 模板", "Custom CSS / Templates"),
		d.text("选择文件与目录…", "Choose File and Folder…"), "", &begin)
	edit := ui.Local(row, "draft", func() draft { return draft{} })
	if begin {
		edit.open, edit.err, edit.paths = true, nil, [2]string{}
		for i, field := range fields {
			if value := siteFieldValue(field.value); value != nil {
				edit.paths[i] = *value
			}
		}
		c.Invalidate()
	}
	panel := ui.Modal(c.Key("theme-assets-editor"), &edit.open, func() {
		ui.Column(c).Width(520).MaxWidthPercent(100).Gap(12).Children(func() {
			ui.Text(c, d.text("自定义 CSS / 模板", "Custom CSS / Templates")).Bold().FontSize(16)
			ui.Text(c, d.text("可手动输入路径。留空可恢复继承。", "You can enter paths manually. Leave a path blank to inherit.")).TextColor(c.Theme().TextMuted)
			for i, field := range fields {
				ui.Column(c).Key(field.key).FillWidth().Gap(4).Children(func() {
					label := ui.Text(c, d.text(field.zh, field.en)).Bold()
					ui.Row(c).FillWidth().Gap(8).Children(func() {
						desktopInput(c, &edit.paths[i]).Grow(1).MinWidth(0).LabelledBy(label).Placeholder(field.example)
						choose := d.text("选择文件", "Choose File")
						if i == 1 {
							choose = d.text("选择目录", "Choose Folder")
						}
						if ui.Button(c, choose).Height(36).Disabled(d.window == nil || d.picking).Clicked() {
							d.pickThemeAsset(i == 1, choose, func(path string, err error) {
								if edit.open {
									edit.err = err
									if path != "" {
										edit.paths[i] = path
									}
								}
							})
						}
					})
				})
			}
			if edit.err != nil {
				d.desktopIssue(c, edit.err, "无法选择主题文件。请检查路径和权限。", "The theme file could not be selected. Check its path and permissions.")
			}
			d.themeEditorActions(c, &edit.open, func() bool {
				for i, field := range fields {
					updateSiteField(field.value, edit.paths[i])
				}
				return true
			})
		})
	})
	panel.Padding(24).Radius(12)
}

func (d *desktopWindow) themeEditorActions(c *ui.Context, open *bool, apply func() bool) {
	ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
		if ui.Button(c, d.text("取消", "Cancel")).Height(36).Clicked() {
			*open = false
		}
		if ui.PrimaryButton(c, d.text("应用", "Apply")).Height(36).Disabled(d.picking).Clicked() {
			if apply() {
				*open = false
				d.edited()
			}
			c.Invalidate()
		}
	})
}

func (d *desktopWindow) pickThemeAsset(directory bool, title string, result func(string, error)) {
	session := d.session
	d.picking = true
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: d.window, Title: title, DefaultPath: session.Root, Directory: directory,
		})
		mygo.RunOnMain(func() {
			d.picking = false
			defer d.window.Invalidate()
			if d.session != session {
				return
			}
			if err != nil {
				result("", err)
			} else if len(paths) > 0 {
				path := paths[0]
				if relative, err := filepath.Rel(session.Root, path); err == nil && relative != ".." &&
					!strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
					path = filepath.ToSlash(relative)
				}
				result(path, nil)
			}
		})
	}()
}

func updateSiteField(value any, raw string) bool {
	text := strings.TrimSpace(raw)
	switch v := value.(type) {
	case **string:
		*v = nil
		if raw != "" {
			*v = new(raw)
		}
	case **int:
		if text == "" {
			*v = nil
		} else {
			n, err := strconv.Atoi(text)
			if err != nil || n < 0 {
				return false
			}
			*v = new(n)
		}
	case **config.RepositoryByteSize:
		if text == "" {
			*v = nil
		} else {
			size := new(config.RepositoryByteSize)
			if n, err := strconv.ParseInt(text, 10, 64); err == nil {
				*size = config.RepositoryByteSize(n)
			} else if size.UnmarshalYAML([]byte(strconv.Quote(text))) != nil {
				return false
			}
			if *size < 0 {
				return false
			}
			*v = size
		}
	case **map[string]string:
		if text == "" {
			*v = nil
		} else {
			var entries map[string]string
			if err := yaml.Unmarshal([]byte(raw), &entries); err != nil || entries == nil {
				return false
			}
			*v = &entries
		}
	default:
		panic(fmt.Sprintf("unsupported site field type %T", value))
	}
	return true
}
