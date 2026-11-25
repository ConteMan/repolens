//go:build darwin || windows

package desktop

import (
	"strings"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/egoist/mygo/ui"
)

func (d *desktopWindow) sharedDraft() *config.RepositorySettings {
	if d.externalTarget {
		return &d.session.ExternalSettings
	}
	return &d.session.Settings
}

func (d *desktopWindow) rulesDraft() *[]config.RepositoryRuleSettings {
	if d.externalTarget {
		return &d.session.ExternalRules
	}
	return &d.session.Rules
}

func (d *desktopWindow) siteErrorKey(key string) string {
	if d.externalTarget && !d.externalInputKey(key) {
		return "external/" + key
	}
	return key
}

func (d *desktopWindow) externalInputKey(key string) bool {
	section, _, _ := strings.Cut(key, ".")
	return section == "source" || section == "output" || section == "access" || strings.HasPrefix(key, "external/")
}

func (d *desktopWindow) targetSelector(c *ui.Context) {
	repository := d.text("仓库配置", "Repository Configuration")
	external := d.text("外部配置", "External Configuration")
	ui.Row(c).FillWidth().PaddingX(40).Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Text(c, d.text("写入目标", "Write Target")).Bold()
		ui.RadioGroup(c.Key("configuration-target"), func() {
			ui.Row(c).Wrap().Gap(8).Children(func() {
				ui.Radio(c, &d.externalTarget, false, repository)
				ui.Radio(c, &d.externalTarget, true, external)
			})
		})
	})
}

func (d *desktopWindow) effectiveField(resolved, repository, external siteField) (string, string) {
	value := siteFieldValue(resolved.value)
	if value == nil {
		return "", ""
	}
	source := d.text("默认", "Default")
	if siteFieldValue(external.value) != nil {
		source = d.text("外部配置", "External Configuration")
	} else if siteFieldValue(repository.value) != nil {
		source = d.text("仓库配置", "Repository Configuration")
	}
	display := *value
	for _, option := range resolved.options {
		if option.value == display {
			display = d.text(option.zh, option.en)
			break
		}
	}
	if _, isBool := resolved.value.(**bool); isBool {
		if display == "true" {
			display = d.text("开启", "On")
		} else {
			display = d.text("关闭", "Off")
		}
	}
	return display, d.text("生效值", "Effective Value") + "  ·  " + display + "  ·  " + source
}
