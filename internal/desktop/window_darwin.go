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

type desktopWindow struct {
	window         *mygo.Window
	english        bool
	sidebar        float32
	session        *Session
	editor         *ruleEditor
	editIndex      int
	pending        *Session
	busy           bool
	discard        bool
	action         func()
	confirmed      bool
	picking        bool
	repositoryMenu bool
	openPath       string
	openError      error
	openFocus      bool
	saveError      error
	preview        bool
	diff           string
	saved          bool
}

func Run(path, language string) error {
	d := &desktopWindow{sidebar: defaultSidebarWidth}
	mygo.App.WhenReady(func() {
		if language == "" {
			preferred := mygo.App.PreferredLanguages()
			if len(preferred) > 0 {
				language = preferred[0]
			}
		}
		d.english = !strings.HasPrefix(strings.ToLower(language), "zh")
		d.window = mygo.NewWindow(mygo.WindowOptions{
			Title: "Repolens", Width: 1060, Height: 720, MinWidth: 740, MinHeight: 450,
			TitleBarStyle: mygo.TitleBarHiddenInset,
			Content:       ui.View(d.view),
		})
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
	if d.busy || d.picking {
		return true
	}
	if d.dirty() || d.editor != nil {
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
	return d.session != nil && d.session.Dirty()
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
	if d.dirty() || d.editor != nil {
		d.pending = session
		d.askDiscard(func() { d.use(session) })
		return
	}
	d.use(session)
}

func (d *desktopWindow) use(session *Session) {
	d.session = session
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
	d.sidebar = min(max(d.sidebar, 192), 340)
	ui.Split(c.Key("workspace"), &d.sidebar, func() { d.navigation(c) }, func() { d.content(c) }).
		SplitReset(defaultSidebarWidth).Fill()
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
		ui.Column(c).FillWidth().PaddingX(18).Gap(10).Children(func() {
			ui.Text(c, "Repolens").FontSize(18).Bold()
			if d.session == nil {
				return
			}
			name := filepath.Base(d.session.Root)
			button := ui.ButtonBase(c.Key("repository-picker")).FillWidth().Height(32).AlignItems(ui.Center).
				Label(name).Tooltip(d.session.Root).Disabled(d.picking)
			button.Children(func() {
				ui.Text(c, name).Grow(1).MinWidth(0).SingleLine().TruncateMiddle()
				ui.Icon(c, repositoryChevron).Size(16, 16).TextColor(t.TextMuted)
			})
			if button.Clicked() {
				d.repositoryMenu = !d.repositoryMenu
			}
			ui.Popover(c.Key("repository-menu"), button, &d.repositoryMenu, func() {
				ui.Column(c).Padding(8).Gap(8).Width(272).Children(func() {
					ui.Text(c, d.session.Root).Selectable()
					if ui.Button(c.Key("switch-repository"), d.text("切换仓库…", "Switch Repository…")).Clicked() {
						d.repositoryMenu = false
						d.choose()
					}
				})
			})
			if d.openError != nil {
				d.openFailure(c)
				retry := ui.Button(c.Key("retry-picker"), d.text("重新选择…", "Choose Again…")).Disabled(d.picking)
				if d.openFocus {
					c.Announce(d.openError.Error())
					retry.Focus()
					d.openFocus = false
				}
				if retry.Clicked() {
					d.choose()
				}
			}
			ui.Row(c).Height(1).FillWidth().Background(t.Border)
			ui.Text(c, d.text("站点设置", "Site Settings")).Bold().Padding(6, 8).Radius(6).Background(t.Background).FillWidth()
		})
	})
}

func (d *desktopWindow) content(c *ui.Context) {
	ui.Column(c).Fill().Background(c.Theme().Background).Children(func() {
		ui.Row(c).Height(48).FillWidth().DragWindow()
		if d.session == nil {
			d.welcome(c)
			return
		}
		if d.editor != nil {
			ui.Column(c).Grow(1).MinHeight(0).FillWidth().Padding(20, 40).Children(func() {
				d.editor.view(c, d.english, d.applyRule, func() { d.editor = nil })
			})
		} else {
			d.rules(c)
		}
		d.footer(c)
	})
}

func (d *desktopWindow) welcome(c *ui.Context) {
	ui.Column(c).Fill().Center().Padding(40).Gap(12).Children(func() {
		ui.Text(c, d.text("打开本地仓库", "Open a Local Repository")).FontSize(26).Bold()
		if d.openError != nil {
			d.openFailure(c)
		} else {
			ui.Text(c, d.text("选择 Git 工作树的根目录以编辑站点规则。", "Choose the root of a Git working tree to edit its site rules.")).
				TextColor(c.Theme().TextMuted)
		}
		label := d.text("选择文件夹…", "Choose Folder…")
		if d.openError != nil {
			label = d.text("重新选择文件夹…", "Choose Another Folder…")
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
}

func (d *desktopWindow) openFailure(c *ui.Context) {
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Text(c, d.text("无法打开仓库", "Could Not Open Repository")).Bold().TextColor(c.Theme().Danger)
		if d.openPath != "" {
			ui.Text(c, filepath.Base(d.openPath)).Bold().Tooltip(d.openPath)
		}
		ui.Text(c, d.openError.Error()).TextColor(c.Theme().TextMuted)
	})
}

func (d *desktopWindow) rules(c *ui.Context) {
	ui.Column(c).Grow(1).MinHeight(0).FillWidth().PaddingX(40).Gap(14).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
			ui.Text(c, d.text("路径与规则", "Paths and Rules")).FontSize(20).Bold()
			if ui.Button(c.Key("add-rule"), d.text("新增规则", "Add Rule")).Height(36).Clicked() {
				d.editIndex = -1
				d.editor = newRuleEditor(config.RepositoryRuleSettings{}, true, nil)
			}
		})
		ui.Text(c, d.text("按列表顺序应用规则。后匹配规则的显式值覆盖先匹配规则。", "Rules apply in order. Later matching rules override earlier ones for the fields they set.")).
			TextColor(c.Theme().TextMuted)
		ui.Scroll(c.Key("rule-list")).Grow(1).MinHeight(0).FillWidth().Children(func() {
			if len(d.session.Rules) == 0 {
				ui.Text(c, d.text("暂无规则。", "No rules yet.")).TextColor(c.Theme().TextMuted).PaddingY(12)
			}
			for index, rule := range d.session.Rules {
				if d.ruleRow(c.Key(index), index, rule) {
					break
				}
			}
		})
	})
}

func (d *desktopWindow) ruleRow(c *ui.Context, index int, rule config.RepositoryRuleSettings) (changed bool) {
	t := c.Theme()
	pattern := d.text("缺少匹配路径", "Missing match pattern")
	if rule.Match != nil && *rule.Match != "" {
		pattern = *rule.Match
	}
	render := d.text("浏览页：继承", "Browse Pages: Inherit")
	if rule.Render != nil {
		if *rule.Render {
			render = d.text("浏览页：生成", "Browse Pages: Generate")
		} else {
			render = d.text("浏览页：跳过", "Browse Pages: Skip")
		}
	}
	ui.Row(c).Wrap().FillWidth().Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).PaddingY(12).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		ui.Column(c).MinWidth(140).Grow(1).Gap(5).Children(func() {
			ui.Text(c, pattern).Font("monospace").Bold().Selectable()
			ui.Text(c, render).TextColor(t.TextMuted)
		})
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if ui.Button(c.Key("edit"), d.text("编辑", "Edit")).Clicked() {
				d.editIndex = index
				d.editor = newRuleEditor(rule, false, nil)
			}
			if ui.Button(c.Key("move-up"), d.text("上移", "Move Up")).Disabled(index == 0).Clicked() {
				d.moveRule(index, index-1)
				changed = true
			}
			if ui.Button(c.Key("move-down"), d.text("下移", "Move Down")).Disabled(index == len(d.session.Rules)-1).Clicked() {
				d.moveRule(index, index+1)
				changed = true
			}
			if ui.Button(c.Key("remove"), d.text("删除", "Delete")).Clicked() {
				d.session.Rules = slices.Delete(d.session.Rules, index, index+1)
				d.edited()
				changed = true
			}
		})
	})
	return changed
}

func (d *desktopWindow) applyRule(rule config.RepositoryRuleSettings) {
	if d.editIndex < 0 {
		d.session.Rules = append(d.session.Rules, rule)
	} else {
		d.session.Rules[d.editIndex] = rule
	}
	d.editor = nil
	d.edited()
}

func (d *desktopWindow) moveRule(from, to int) {
	d.session.Rules[from], d.session.Rules[to] = d.session.Rules[to], d.session.Rules[from]
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
	ui.Column(c).FillWidth().Padding(14, 40).Gap(8).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
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
				ui.Text(c, d.target()).TextColor(t.TextMuted).SingleLine().TruncateMiddle().Tooltip(d.target())
			})
			if ui.PrimaryButton(c.Key("prepare-save"), d.text("保存设置", "Save Settings")).
				Height(36).Disabled(!d.dirty() || d.busy).Clicked() {
				d.prepare()
			}
		})
	})
}

func (d *desktopWindow) prepare() {
	diff, err := d.session.Prepare()
	d.saveError = err
	if err == nil {
		d.diff = diff
		d.preview = true
		return
	}
	var validation *config.RepositoryValidationError
	if errors.As(err, &validation) && len(validation.Issues) > 0 {
		issue := validation.Issues[0]
		var index int
		if matched, _ := fmt.Sscanf(issue.Path, "rules[%d].", &index); matched == 1 && index >= 0 && index < len(d.session.Rules) {
			d.editIndex = index
			d.editor = newRuleEditor(d.session.Rules[index], false, &issue)
		}
	}
}

func (d *desktopWindow) confirmView(c *ui.Context) {
	panel := ui.Modal(c.Key("save-confirmation"), &d.preview, func() {
		data := d.session.confirmation()
		changes := ruleChanges(data.before, data.after)
		ui.Column(c).Width(660).MaxWidthPercent(100).MaxHeightPercent(90).Gap(12).Children(func() {
			ui.Text(c, d.text("保存站点设置？", "Save Site Settings?")).FontSize(19).Bold()
			switch len(changes) {
			case 0:
				ui.Text(c, d.text("规则顺序变更", "Rule order changed")).TextColor(c.Theme().TextMuted)
			case 1:
				ui.Text(c, d.text("1 项规则更改", "1 rule change")).TextColor(c.Theme().TextMuted)
			default:
				ui.Text(c, fmt.Sprintf(d.text("%d 项规则更改", "%d rule changes"), len(changes))).TextColor(c.Theme().TextMuted)
			}
			ui.Text(c, d.text("仓库 / .repolens.yml", "Repository / .repolens.yml")).Bold()
			ui.Scroll(c.Key("save-diff")).Height(280).Shrink(1).MinHeight(0).FillWidth().Background(c.Theme().Surface).Padding(12).Children(func() {
				ui.Column(c).FillWidth().Gap(16).Children(func() {
					ui.Text(c, data.target).FillWidth().Font("monospace").TextColor(c.Theme().TextMuted).Selectable()
					d.confirmationDetails(c, data, changes)
				})
			})
			ui.Row(c).FillWidth().Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("cancel-save"), d.text("返回编辑", "Back to Editing")).Height(36).Clicked() {
					d.preview = false
					d.session.CancelSave()
				}
				if ui.PrimaryButton(c.Key("confirm-save"), d.text("确认保存", "Confirm Save")).Height(36).Disabled(d.busy).Clicked() {
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
