package desktop

import (
	"fmt"
	"strings"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/egoist/mygo/ui"
)

type ignoreEditor struct {
	text                 string
	inherit, open, focus bool
	issue                *config.RepositoryValidationIssue
}

func (d *desktopWindow) openIgnoreEditor(issue *config.RepositoryValidationIssue) {
	e := &ignoreEditor{open: true, focus: true, inherit: d.sharedDraft().Ignore == nil, issue: issue}
	if !e.inherit {
		e.text = strings.Join(*d.sharedDraft().Ignore, "\n")
	}
	d.ignoreEditor = e
}

func (e *ignoreEditor) candidate() (*[]string, *config.RepositoryValidationIssue) {
	if e.inherit {
		return nil, nil
	}
	values := []string{}
	for _, line := range strings.Split(e.text, "\n") {
		if strings.TrimSpace(line) != "" {
			values = append(values, line)
		}
	}
	issues := config.ValidateRepositorySettings(config.RepositorySettings{Ignore: &values})
	if len(issues) > 0 {
		return nil, &issues[0]
	}
	return &values, nil
}

func (d *desktopWindow) ignoreView(c *ui.Context) {
	e := d.ignoreEditor
	t := d.text
	panel := ui.Modal(c.Key("ignore-editor"), &e.open, func() {
		ui.Column(c).Width(500).MaxWidthPercent(100).Gap(14).Children(func() {
			ui.Text(c, t("编辑忽略路径", "Edit Ignored Paths")).FontSize(16).Bold()
			if ui.Checkbox(c, &e.inherit, t("继承忽略路径", "Inherit ignored paths")).Changed() {
				e.issue = nil
			}
			if !e.inherit {
				ui.Text(c, t("每行一个 glob。留空可指定空列表。", "Enter one glob per line. Leave this blank for an empty list.")).TextColor(c.Theme().TextMuted)
				input := ui.TextArea(c, &e.text).Height(130).FillWidth().Label(t("忽略路径", "Ignored Paths"))
				if e.focus {
					input.Focus()
					e.focus = false
				}
				if e.issue != nil {
					input.BorderColor(c.Theme().Danger)
					var index int
					if n, _ := fmt.Sscanf(e.issue.Path, "ignore[%d]", &index); n == 1 {
						ui.Text(c, fmt.Sprintf(t("第 %d 个路径必须是有效的 glob。", "Pattern %d must be a valid glob."), index+1)).
							TextColor(c.Theme().Danger)
					}
				}
			}
			ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
				if ui.Button(c, t("取消", "Cancel")).Height(36).PaddingX(14).Clicked() {
					d.ignoreEditor = nil
				}
				if ui.PrimaryButton(c, t("应用列表", "Apply List")).Height(36).PaddingX(14).Clicked() {
					values, issue := e.candidate()
					if issue != nil {
						e.issue, e.focus = issue, true
						c.Invalidate()
						return
					}
					d.sharedDraft().Ignore = values
					d.ignoreEditor = nil
					d.edited()
				}
			})
		})
	})
	panel.Padding(24).Radius(12)
	if panel.Dismissed() {
		d.ignoreEditor = nil
	}
}
