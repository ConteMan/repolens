package desktop

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const defaultSidebarWidth float32 = 236

var (
	repositoryChevron = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16"><path fill="none" stroke="currentColor" stroke-width="1.5" d="m4 6 4 4 4-4"/></svg>`))
	savedCheck        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16"><path fill="none" stroke="currentColor" stroke-width="1.6" d="m3 8 3.2 3.2L13 4.5"/></svg>`))
)

type desktopPage int

const (
	pageSite desktopPage = iota
	pageBuild
	pageApp
)

type desktopSiteTab = int

const (
	tabRules desktopSiteTab = iota
	tabBasics
	tabBrowse
	tabRender
	tabAgent
	tabBuilder
)

type desktopWindow struct {
	buildState      desktopBuildState
	page            desktopPage
	siteTab         desktopSiteTab
	preferences     desktopPreferences
	appError        error
	update          desktopUpdate
	window          *mygo.Window
	english         bool
	sidebar         float32
	session         *Session
	editor          *ruleEditor
	ignoreEditor    *ignoreEditor
	editIndex       int
	pending         *Session
	busy            bool
	discard         bool
	action          func()
	confirmed       bool
	picking         bool
	repositoryMenu  bool
	openPath        string
	openError       error
	openFocus       bool
	saveError       error
	preview         bool
	changes         []ConfigChange
	siteInputErrors map[string]string
	externalTarget  bool
	siteFocus       string
	saved           bool
}

func Run(path, language string) error {
	d := &desktopWindow{sidebar: defaultSidebarWidth, siteTab: tabBasics}
	mygo.App.WhenReady(func() {
		d.preferences, d.appError = loadDesktopPreferences()
		if d.preferences.Sidebar != 0 {
			d.sidebar = d.preferences.Sidebar
		}
		if language != "" {
			d.preferences.Language = language
		}
		d.applyLanguage()

		d.window = mygo.NewWindow(mygo.WindowOptions{
			Title: "Repolens", Width: 1024, Height: 632, MinWidth: 740, MinHeight: 450,
			TitleBarStyle:        mygo.TitleBarHiddenInset,
			TrafficLightPosition: &mygo.Point{X: 18, Y: 18},
			StateKey:             "workspace",
			Content:              ui.View(d.view),
		})
		d.watchWindowPlacement()
		d.window.OnClose(func(event *mygo.CloseEvent) {
			if d.guardClose(d.window.Close) {
				event.PreventDefault()
			}
		})
		mygo.App.OnBeforeQuit(func(event *mygo.QuitEvent) {
			if !event.SessionEnding && d.guardClose(mygo.App.Quit) {
				event.PreventDefault()
			}
		})
		if path != "" {
			d.open(path)
		}
		if d.preferences.AutoCheck && mygo.Updater.Enabled() {
			d.checkUpdate(true)
		}
	})
	return mygo.App.Run()
}

func (d *desktopWindow) text(chinese, english string) string {
	if d.english {
		return english
	}
	return chinese
}

func (d *desktopWindow) guardClose(close func()) bool {
	if d.confirmed {
		return false
	}
	if d.busy || d.picking || d.update.stage == updateDownloading {
		return true
	}
	if d.dirty() || d.editor != nil || d.ignoreEditor != nil {
		d.askDiscard(func() {
			d.confirmed = true
			close()
		})
		d.window.Invalidate()
		return true
	}
	return false
}

func (d *desktopWindow) dirty() bool {
	return d.session != nil && (d.session.Dirty() || len(d.siteInputErrors) != 0)
}

func (d *desktopWindow) askDiscard(action func()) {
	d.discard = true
	d.action = action
}

func (d *desktopWindow) open(path string) {
	d.openPath = path
	session, err := Open(path)
	d.openError = err
	d.openFocus = err != nil
	d.window.Invalidate()
	if err != nil {
		return
	}
	if d.session != nil && d.session.Root == session.Root {
		return
	}
	if d.dirty() || d.editor != nil || d.ignoreEditor != nil {
		d.pending = session
		d.askDiscard(func() { d.use(session) })
		return
	}
	d.use(session)
}

func (d *desktopWindow) use(session *Session) {
	first := d.session == nil
	d.buildState.reset()
	d.session = session
	d.externalTarget = false
	d.ignoreEditor = nil
	d.siteInputErrors = nil
	d.siteFocus = ""
	if first {
		d.page = pageSite
	}
	d.editor = nil
	d.saveError = nil
	d.saved = false
	d.window.SetDocumentEdited(false)
	d.window.Invalidate()
}

func (d *desktopWindow) choose() {
	if d.picking {
		return
	}
	d.picking = true
	path := ""
	if d.session != nil {
		path = d.session.Root
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: d.window, Directory: true, DefaultPath: path,
			Title:       d.text("选择本地仓库", "Choose a Local Repository"),
			ButtonLabel: d.text("打开", "Open"),
		})
		mygo.RunOnMain(func() {
			d.picking = false
			switch {
			case err != nil:
				d.openError = err
				d.openFocus = true
			case len(paths) > 0:
				d.open(paths[0])
			}
			d.window.Invalidate()
		})
	}()
}

func (d *desktopWindow) view(c *ui.Context) {
	theme := *c.Theme()
	if !theme.Dark {
		theme.Surface = ui.Hex("#f5f5f6")
		theme.SurfaceHover = ui.Hex("#e7e7e9")
		theme.Text = ui.Hex("#202124")
		theme.TextMuted = ui.Hex("#62656a")
		theme.Border = ui.Hex("#d8d9dc")
		theme.Accent = ui.Hex("#3958be")
		theme.InactiveAccent = theme.Accent
		theme.Focus = theme.Accent
		theme.Radius = 6
		c.SetTheme(&theme)
	}
	d.sidebar = min(max(d.sidebar, 192), 340)
	split := ui.Split(c.Key("workspace"), &d.sidebar, func() { d.navigation(c) }, func() { d.content(c) }).
		SplitReset(defaultSidebarWidth).Fill()
	if split.Changed() && d.window != nil {
		d.sidebar = min(max(d.sidebar, 192), 340)
		old := d.preferences
		d.preferences.Sidebar = d.sidebar
		d.savePreferences(old)
	}
	if d.ignoreEditor != nil && !d.discard {
		d.ignoreView(c)
	}
	if d.preview && !d.discard {
		d.confirmView(c)
	}
	if d.discard {
		d.discardView(c)
	}
}

func (d *desktopWindow) navigation(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Surface).Children(func() {
		ui.Row(c).FillWidth().Height(48).DragWindow()
		ui.Column(c).FillWidth().Gap(4).PaddingX(18).Children(func() {
			ui.Row(c).Height(36).FillWidth().AlignItems(ui.Center).DragWindow().Children(func() {
				ui.Text(c, "Repolens").FontSize(21).Bold()
			})
			d.repositorySelector(c)
		})
		if d.session != nil {
			ui.Column(c).FillWidth().Padding(12, 10, 0).Gap(4).Children(func() {
				ui.Row(c).Height(13).FillWidth().PaddingX(8).Children(func() {
					ui.Row(c).Height(1).FillWidth().Background(t.Border)
				})
				d.navigationButton(c.Key("site-settings"), pageSite, d.text("站点设置", "Site Settings"), siteIcon)
				d.navigationButton(c.Key("build-preview"), pageBuild, d.text("构建与预览", "Build and Preview"), buildIcon)
			})
		}
		ui.Row(c).Grow(1)
		ui.Column(c).FillWidth().Padding(0, 14, 28).Children(func() {
			d.navigationButton(c.Key("app-settings"), pageApp, d.text("应用设置", "App Settings"), preferencesIcon)
		})
	})
}

func (d *desktopWindow) navigationButton(c *ui.Context, page desktopPage, label string, icon *ui.SVG) {
	t := c.Theme()
	active := page == d.page && (d.session != nil || page == pageApp)
	button := ui.ButtonBase(c).Height(36).Radius(6).PaddingX(8).Gap(8).AlignItems(ui.Center).
		Label(label).AccessSelected(active)
	if page == pageApp {
		button.Width(36).Tooltip(label).Justify(ui.Center)
	} else {
		button.FillWidth().Justify(ui.Start)
	}
	if active || button.Hovered() || page == pageApp {
		button.Background(t.SurfaceHover)
	}
	button.Children(func() {
		ui.Icon(c, icon).Size(16, 16).TextColor(t.Text)
		if page != pageApp {
			caption := ui.Text(c, label).FontSize(14).SingleLine().TruncateMiddle()
			if active {
				caption.Bold()
			}
		}
	})
	if button.Clicked() {
		if active && page == pageApp && d.session == nil {
			d.page = pageSite
		} else {
			d.page = page
		}
	}
}

func (d *desktopWindow) repositorySelector(c *ui.Context) {
	t := c.Theme()
	name := d.text("选择仓库", "Choose Repository")
	description := name
	if d.session != nil {
		name = filepath.Base(d.session.Root)
		description = d.session.Root
	}
	button := ui.ButtonBase(c.Key("repository-picker")).FillWidth().Height(28).AlignItems(ui.Center).
		Gap(4).Label(name).Tooltip(description).Disabled(d.picking)
	if button.Hovered() {
		button.Background(t.Background).Radius(6)
	}
	button.Children(func() {
		ui.Text(c, name).FontSize(14).Grow(1).MinWidth(0).SingleLine().TruncateMiddle()
		ui.Icon(c, repositoryChevron).Size(14, 14).TextColor(t.TextMuted)
	})
	if button.Clicked() {
		if d.session == nil {
			d.choose()
		} else {
			d.repositoryMenu = !d.repositoryMenu
		}
	}
	if d.session == nil {
		return
	}
	if d.openError != nil && d.openFocus {
		d.repositoryMenu = true
	}
	popover := ui.Popover(c.Key("repository-menu"), button, &d.repositoryMenu, func() {
		ui.Column(c).Padding(14).Gap(12).Width(304).Children(func() {
			if d.openError == nil {
				ui.Text(c, d.session.Root).FontSize(13).TextColor(t.TextMuted).Selectable()
				if ui.Button(c.Key("switch-repository"), d.text("切换仓库…", "Switch Repository…")).Height(36).Clicked() {
					d.repositoryMenu = false
					d.choose()
				}
				return
			}
			d.openFailure(c)
			ui.Row(c).Gap(8).Wrap().Children(func() {
				retry := ui.PrimaryButton(c.Key("retry-picker"), d.text("重新选择文件夹", "Choose Another Folder")).Height(36).Disabled(d.picking)
				if d.openFocus {
					c.Announce(d.openError.Error())
					retry.Focus()
					d.openFocus = false
				}
				if retry.Clicked() {
					d.repositoryMenu = false
					d.choose()
				}
				if ui.Button(c.Key("dismiss-open-error"), d.text("取消", "Cancel")).Height(36).Clicked() {
					d.repositoryMenu = false
					d.openError = nil
					button.Focus()
				}
			})
		})
	})
	if popover.Dismissed() {
		d.openError = nil
		button.Focus()
	}
}

func (d *desktopWindow) content(c *ui.Context) {
	ui.Column(c).Fill().Background(c.Theme().Background).Children(func() {
		ui.Row(c).Height(48).FillWidth().DragWindow()
		switch d.page {
		case pageApp:
			d.appPage(c)
			return
		case pageBuild:
			d.buildPage(c)
			return
		}
		if d.session == nil {
			d.welcome(c)
			return
		}
		d.siteTabs(c)
		if d.session.BuilderPath != "" && d.siteTab != tabBuilder && d.editor == nil {
			d.targetSelector(c)
		}
		if d.siteTab != tabRules {
			d.sitePage(c, d.siteTab)
			d.footer(c)
			return
		}
		if d.editor != nil {
			ui.Column(c).Grow(1).MinHeight(0).FillWidth().Padding(28, 40).Children(func() {
				d.editor.view(c, d.english, d.applyRule, func() { d.editor = nil })
			})
		} else {
			d.rules(c)
		}
		d.footer(c)
	})
}

func (d *desktopWindow) siteTabs(c *ui.Context) {
	tabs := [...]desktopSiteTab{tabBasics, tabBrowse, tabRender, tabRules, tabAgent, tabBuilder}
	labels := [...]string{
		d.text("基础信息", "Basics"),
		d.text("浏览与主题", "Browse and Theme"),
		d.text("渲染与术语", "Rendering and Glossary"),
		d.text("路径与规则", "Paths and Rules"),
		d.text("Agent 输出", "Agent Output"),
		d.text("构建者配置", "Builder Settings"),
	}
	selected := 0
	for index, tab := range tabs {
		if tab == d.siteTab {
			selected = index
			break
		}
	}
	ui.ScrollHorizontal(c.Key("site-tabs")).Height(36).FillWidth().Margin(0, 38, 16, 38).Children(func() {
		ui.Tabs(c, &selected, labels[:]...)
	})
	d.siteTab = tabs[selected]
}

func (d *desktopWindow) welcome(c *ui.Context) {
	_, height := c.Size()
	top := min(max(height-480, 48), 152)
	ui.Scroll(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
		ui.Column(c).FillWidth().Padding(top, 38, 38, 81).Gap(18).Children(func() {
			ui.Text(c, d.text("打开 Git 仓库", "Open a Git Repository")).FontSize(30).Bold()
			label := d.text("选择文件夹…", "Choose Folder…")
			if d.openError != nil {
				d.openFailure(c)
				label = d.text("重新选择文件夹", "Choose Another Folder")
			} else {
				ui.Text(c, d.text("选择文档所在的 Git 工作树根目录。", "Choose the Git working tree that contains your documentation.")).
					FontSize(16).TextColor(c.Theme().TextMuted).Margin(0, 0, 18, 0)
			}
			button := ui.PrimaryButton(c.Key("welcome-picker"), label).Height(36).Disabled(d.picking)
			if d.openFocus {
				c.Announce(d.openError.Error())
				button.Focus()
				d.openFocus = false
			}
			if button.Clicked() {
				d.choose()
			}
		})
	})
}

func (d *desktopWindow) openFailure(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).FillWidth().MaxWidth(486).Gap(6).Padding(12).Radius(8).Border(1, t.Border).Background(t.Surface).Children(func() {
		ui.Text(c, d.text("无法打开仓库", "Could Not Open Repository")).Bold().TextColor(t.Danger)
		if d.openPath != "" {
			ui.Text(c, filepath.Base(d.openPath)).Bold().Tooltip(d.openPath)
		}
		ui.Text(c, d.openError.Error()).TextColor(t.TextMuted).Selectable()
	})
}

func (d *desktopWindow) rules(c *ui.Context) {
	ui.Column(c).Grow(1).MinHeight(0).FillWidth().Padding(6, 40, 0).Gap(12).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Margin(0, 0, 18).Children(func() {
			ui.Text(c, d.text("忽略路径", "Ignored Paths")).Bold().WidthPercent(24)
			summary := d.text("继承", "Inherit")
			if paths := d.sharedDraft().Ignore; paths != nil {
				summary = d.text("空列表", "Empty list")
				if len(*paths) > 0 {
					summary = strings.Join(*paths, "  ·  ")
				}
			}
			ui.Text(c, summary).Grow(1).MinWidth(0).SingleLine().TruncateMiddle().TextColor(c.Theme().TextMuted).Tooltip(summary)
			if ui.Button(c.Key("edit-ignore"), d.text("编辑列表", "Edit List")).Height(36).PaddingX(14).Clicked() {
				d.openIgnoreEditor(nil)
			}
		})
		if d.session.BuilderPath != "" {
			if d.session.ExternalSettings.Ignore != nil {
				d.effectiveIgnore(c, d.session.ExternalSettings.Ignore, d.text("外部配置", "External Configuration"))
			} else if d.session.Settings.Ignore != nil {
				d.effectiveIgnore(c, d.session.Settings.Ignore, d.text("仓库配置", "Repository Configuration"))
			}
		}
		ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
			ui.Text(c, d.text("路径规则", "Path Rules")).Bold()
			if ui.Button(c.Key("add-rule"), d.text("新增规则", "Add Rule")).Height(36).PaddingX(14).Clicked() {
				d.editIndex = -1
				d.editor = newRuleEditor(config.RepositoryRuleSettings{}, true, nil)
			}
		})
		ui.Text(c, d.text("按从上到下的顺序匹配，后面的规则可覆盖前面的设置。", "Rules apply in order. Later matching rules override earlier ones for the fields they set.")).
			TextColor(c.Theme().TextMuted)
		if d.session.BuilderPath != "" {
			fit := fmt.Sprintf(d.text("生效规则顺序含 %d 条仓库规则，再接 %d 条外部规则。", "Effective order contains %d repository rules followed by %d external rules."), len(d.session.Rules), len(d.session.ExternalRules))
			ui.Text(c, fit).TextColor(c.Theme().TextMuted)
		}
		ui.Scroll(c.Key("rule-list")).Grow(1).MinHeight(0).FillWidth().Children(func() {
			if len((*d.rulesDraft())) == 0 {
				ui.Text(c, d.text("暂无规则。", "No rules yet.")).TextColor(c.Theme().TextMuted).PaddingY(12)
			}
			for index, rule := range *d.rulesDraft() {
				if d.ruleRow(c.Key(index), index, rule) {
					break
				}
			}
		})
	})
}

func (d *desktopWindow) effectiveIgnore(c *ui.Context, paths *[]string, source string) {
	summary := d.text("空列表", "Empty list")
	if len(*paths) > 0 {
		summary = strings.Join(*paths, " · ")
	}
	label := d.text("生效的忽略路径", "Effective Ignored Paths") + " · " + summary + " · " + source
	ui.Text(c, label).SingleLine().TruncateMiddle().TextColor(c.Theme().TextMuted).Tooltip(label)
}

func (d *desktopWindow) ruleRow(c *ui.Context, index int, rule config.RepositoryRuleSettings) (changed bool) {
	t := c.Theme()
	pattern := d.text("缺少匹配路径", "Missing match pattern")
	if rule.Match != nil && *rule.Match != "" {
		pattern = *rule.Match
	}
	render := d.text("浏览页继承", "Browser Pages: Inherit")
	if rule.Render != nil {
		if *rule.Render {
			render = d.text("生成浏览页", "Generate Browser Pages")
		} else {
			render = d.text("跳过浏览页", "Skip Browser Pages")
		}
	}
	ui.Row(c).Wrap().FillWidth().Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).PaddingY(14).Children(func() {
		ui.Column(c).MinWidth(140).Grow(1).Gap(5).Children(func() {
			ui.Text(c, pattern).Bold().FillWidth().Selectable()
			ui.Text(c, render).TextColor(t.TextMuted)
		})
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if ui.Button(c.Key("edit"), d.text("编辑", "Edit")).Height(36).PaddingX(14).Clicked() {
				d.editIndex = index
				d.editor = newRuleEditor(rule, false, nil)
			}
			if ui.Button(c.Key("move-up"), d.text("上移", "Move Up")).Height(36).PaddingX(14).Disabled(index == 0).Clicked() {
				d.moveRule(index, index-1)
				changed = true
			}
			if ui.Button(c.Key("move-down"), d.text("下移", "Move Down")).Height(36).PaddingX(14).Disabled(index == len((*d.rulesDraft()))-1).Clicked() {
				d.moveRule(index, index+1)
				changed = true
			}
			if ui.Button(c.Key("remove"), d.text("删除", "Delete")).Height(36).PaddingX(14).Clicked() {
				(*d.rulesDraft()) = slices.Delete((*d.rulesDraft()), index, index+1)
				d.edited()
				changed = true
			}
		})
	})
	return changed
}

func (d *desktopWindow) applyRule(rule config.RepositoryRuleSettings) {
	if d.editIndex < 0 {
		(*d.rulesDraft()) = append((*d.rulesDraft()), rule)
	} else {
		(*d.rulesDraft())[d.editIndex] = rule
	}
	d.editor = nil
	d.edited()
}

func (d *desktopWindow) moveRule(from, to int) {
	(*d.rulesDraft())[from], (*d.rulesDraft())[to] = (*d.rulesDraft())[to], (*d.rulesDraft())[from]
	d.edited()
}

func (d *desktopWindow) edited() {
	d.saved = false
	if !errors.Is(d.saveError, config.ErrRevisionConflict) {
		d.saveError = nil
	}
	d.updateDocumentState()
}

func (d *desktopWindow) updateDocumentState() {
	if d.window != nil {
		d.window.SetDocumentEdited(d.dirty())
	}
}

func (d *desktopWindow) target() string {
	return filepath.Join(d.session.Root, ".repolens.yml")
}

func (d *desktopWindow) footer(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).FillWidth().Padding(12, 40, 30).Gap(8).Children(func() {
		if d.saveError != nil {
			ui.Text(c, d.saveError.Error()).TextColor(t.Danger).Selectable()
			if errors.Is(d.saveError, config.ErrRevisionConflict) {
				if ui.Button(c.Key("reload"), d.text("重新读取文件…", "Reload File…")).Clicked() {
					d.askDiscard(func() {
						if err := d.session.Reload(); err != nil {
							d.saveError = errors.Join(config.ErrRevisionConflict, err)
							return
						}
						d.saveError = nil
						d.siteInputErrors = nil
						d.siteFocus = ""
						d.saved = false
						d.updateDocumentState()
					})
				}
			}
		}
		ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
				if d.saved {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Icon(c, savedCheck).Size(16, 16).TextColor(t.Success)
						ui.Text(c, d.text("已保存", "Saved")).Bold()
					})
				}
				label := d.text("仓库 / .repolens.yml", "Repository / .repolens.yml")
				target := d.target()
				switch {
				case d.saved && len(d.changes) > 0:
					paths := make([]string, len(d.changes))
					for i, change := range d.changes {
						paths[i] = change.Path
					}
					target = strings.Join(paths, "\n")
					label = strings.Join(paths, "  ·  ")
				case d.session.BuilderDirty() && d.session.repositoryDirty():
					label = d.text("2 个配置文件", "2 configuration files")
					target += "\n" + d.session.BuilderPath
				case d.session.BuilderDirty() || ((d.siteTab == tabBuilder || d.externalTarget) && !d.dirty()):
					target = d.session.BuilderPath
					label = d.text("尚未选择外部文件", "No External File Selected")
					if target != "" {
						label = d.text("外部配置 / ", "External Configuration / ") + filepath.Base(target)
					}
				}
				if count := len(d.siteInputErrors); count > 0 {
					label = fmt.Sprintf(d.text("%d 项待修正", "%d fields to fix"), count) + "  ·  " + label
				} else if d.dirty() {
					label = d.text("有未保存修改", "Unsaved Changes") + "  ·  " + label
				}
				ui.Text(c, label).TextColor(t.TextMuted).SingleLine().TruncateMiddle().Tooltip(target)
			})
			label := d.text("保存设置", "Save Settings")
			disabled := (!d.dirty() && len(d.siteInputErrors) == 0) || d.busy
			makeButton := ui.PrimaryButton
			if disabled {
				makeButton = ui.Button
			}
			if makeButton(c.Key("prepare-save"), label).Height(36).PaddingX(14).Disabled(disabled).Clicked() {
				d.prepare()
			}
		})
	})
}

func (d *desktopWindow) prepare() {
	for _, tab := range []desktopSiteTab{tabBasics, tabBrowse, tabRender, tabAgent, tabBuilder} {
		fields := siteFields(&d.session.Settings, tab)
		if tab == tabBuilder {
			fields = builderFields(&d.session.Builder)
		}
		for _, field := range fields {
			_, invalidRepository := d.siteInputErrors[field.key]
			_, invalidExternal := d.siteInputErrors["external/"+field.key]
			if invalidRepository || invalidExternal {
				d.externalTarget = !invalidRepository && invalidExternal
				d.page = pageSite
				d.siteTab = tab
				d.siteFocus = field.key
				return
			}
		}
	}
	changes, err := d.session.PrepareChanges()
	d.saveError = err
	if err == nil {
		d.changes = changes
		d.preview = true
		return
	}
	var validation *config.RepositoryValidationError
	if errors.As(err, &validation) && len(validation.Issues) > 0 {
		var external *ExternalValidationError
		d.externalTarget = errors.As(err, &external)
		issue := validation.Issues[0]
		if strings.HasPrefix(issue.Path, "ignore[") {
			d.page = pageSite
			d.siteTab = tabRules
			d.openIgnoreEditor(&issue)
			return
		}
		var index int
		if matched, _ := fmt.Sscanf(issue.Path, "rules[%d].", &index); matched == 1 && index >= 0 && index < len((*d.rulesDraft())) {
			d.page = pageSite
			d.siteTab = tabRules
			d.editIndex = index
			d.editor = newRuleEditor((*d.rulesDraft())[index], false, &issue)
			return
		}
		for _, tab := range []desktopSiteTab{tabBasics, tabBrowse, tabRender, tabAgent, tabBuilder} {
			fields := siteFields(&d.session.Settings, tab)
			if tab == tabBuilder {
				fields = builderFields(&d.session.Builder)
			}
			for _, field := range fields {
				if field.key == issue.Path {
					d.page = pageSite
					d.siteTab = tab
					d.siteFocus = field.key
					return
				}
			}
		}
	}
}

func (d *desktopWindow) confirmView(c *ui.Context) {
	panel := ui.Modal(c.Key("save-confirmation"), &d.preview, func() {
		ui.Column(c).Width(586).MaxWidthPercent(100).MaxHeightPercent(90).Gap(20).Children(func() {
			ui.Text(c, d.text("保存站点设置？", "Save Site Settings?")).FontSize(16).Bold()
			ui.Scroll(c.Key("save-diff")).Height(194).Shrink(1).MinHeight(0).FillWidth().Children(func() {
				ui.Column(c).FillWidth().Gap(20).Children(func() {
					for _, change := range d.changes {
						d.configurationChangeView(c, change)
					}
				})
			})
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("cancel-save"), d.text("返回编辑", "Back to Editing")).Height(36).PaddingX(14).Clicked() {
					d.preview = false
					d.session.CancelSave()
				}
				if ui.PrimaryButton(c.Key("confirm-save"), d.text("确认保存", "Confirm Save")).Height(36).PaddingX(14).Disabled(d.busy).Clicked() {
					d.preview = false
					d.busy = true
					d.saveError = d.session.Save()
					d.saved = d.saveError == nil
					d.busy = false
					d.updateDocumentState()
				}
			})
		})
	})
	panel.Padding(24).Radius(12)
	if panel.Dismissed() {
		d.session.CancelSave()
	}
}

func (d *desktopWindow) discardView(c *ui.Context) {
	panel := ui.Modal(c.Key("discard-confirmation"), &d.discard, func() {
		ui.Column(c).Width(400).Gap(12).Children(func() {
			ui.Text(c, d.text("丢弃未保存的草稿？", "Discard Unsaved Draft?")).FontSize(19).Bold()
			if d.pending != nil {
				ui.Text(c, d.pending.Root).Selectable()
			}
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("keep-draft"), d.text("保留草稿", "Keep Draft")).Height(36).Clicked() {
					d.discard = false
					d.action = nil
					d.pending = nil
				}
				if ui.PrimaryButton(c.Key("discard-draft"), d.text("丢弃草稿", "Discard Draft")).Height(36).Clicked() && d.action != nil {
					action := d.action
					d.discard = false
					d.action = nil
					d.pending = nil
					action()
				}
			})
		})
	})
	if panel.Dismissed() {
		d.action = nil
		d.pending = nil
	}
}
