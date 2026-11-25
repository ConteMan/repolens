//go:build darwin || windows

package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const projectURL = "https://github.com/ConteMan/repolens"

type desktopPreferences struct {
	Language  string  `json:"language,omitempty"`
	AutoCheck bool    `json:"auto_check"`
	Sidebar   float32 `json:"sidebar_width,omitempty"`
}

func desktopPreferencePath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "Repolens", "preferences.json"), nil
}

func loadDesktopPreferences() (desktopPreferences, error) {
	preferences := desktopPreferences{AutoCheck: true}
	path, err := desktopPreferencePath()
	if err != nil {
		return preferences, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return preferences, nil
	}
	if err != nil {
		return preferences, err
	}
	err = json.Unmarshal(data, &preferences)
	return preferences, err
}

func (p desktopPreferences) save() error {
	path, err := desktopPreferencePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".preferences-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	writeErr := json.NewEncoder(f).Encode(p)
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func (d *desktopWindow) applyLanguage() {
	language := d.preferences.Language
	if language == "" {
		preferred := mygo.App.PreferredLanguages()
		if len(preferred) > 0 {
			language = preferred[0]
		}
	}
	d.english = !strings.HasPrefix(strings.ToLower(language), "zh")
}

type updateStage int

const (
	updateIdle updateStage = iota
	updateChecking
	updateCurrent
	updateUnavailable
	updateAvailable
	updateDownloading
	updateReady
)

type desktopUpdate struct {
	stage    updateStage
	release  *mygo.Update
	err      error
	received int64
	total    int64
}

func (d *desktopWindow) appPage(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c.Key("app-preferences")).FillWidth().Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).FillWidth().Padding(0, desktopInset(c), 16).Gap(0).Children(func() {
			d.preferenceRow(c, d.text("应用语言", "App Language"), func() {
				labels := []string{d.text("跟随系统", "System"), "简体中文", "English"}
				languages := []string{"", "zh", "en"}
				selected := labels[0]
				for index, language := range languages {
					if d.preferences.Language == language {
						selected = labels[index]
						break
					}
				}
				if desktopSelect(c.Key("interface-language"), &selected, labels).FillWidth().Changed() {
					old := d.preferences
					for index, label := range labels {
						if selected == label {
							d.preferences.Language = languages[index]
							break
						}
					}
					d.savePreferences(old)
					d.applyLanguage()
					c.Invalidate()
				}
			})
			d.preferenceRow(c, d.text("当前版本", "Version"), func() {
				version := mygo.App.Version()
				if version == "" {
					version = "dev"
				}
				ui.Text(c, version).Selectable()
			})
			d.preferenceRow(c, d.text("项目主页", "Source Code"), func() {
				link := ui.ButtonBase(c.Key("project-link")).Label("github.com/ConteMan/repolens").Height(36).AlignSelf(ui.Start).Justify(ui.Start).AlignItems(ui.Center).Gap(4)
				link.Children(func() {
					ui.Text(c, "github.com/ConteMan/repolens").TextColor(t.Accent)
					ui.Icon(c, externalLinkIcon).Size(14, 14).TextColor(t.Accent)
				})
				if link.Clicked() {
					d.appError = mygo.Shell.OpenExternal(projectURL)
				}
			})
			d.preferenceRow(c, d.text("自动更新", "Automatic Updates"), func() {
				old := d.preferences
				ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
					ui.Switch(c.Key("automatic-updates"), &d.preferences.AutoCheck).Label(d.text("启动时自动检查并下载", "Automatically check and download at launch")).Size(42, 24)
				})
				if old != d.preferences {
					d.savePreferences(old)
				}
			})
			ui.Column(c).FillWidth().Margin(32, 0, 0).Gap(16).Children(func() {
				d.updateView(c)
			})
			if d.appError != nil {
				d.desktopIssue(c, d.appError, "无法处理应用设置或打开链接。请查看错误详情。", "App settings or the link could not be handled. See the error details.")
			}
		})
	})
}

func (d *desktopWindow) preferenceRow(c *ui.Context, label string, field func()) {
	ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(12).Height(64).Children(func() {
		ui.Text(c, label).WidthPercent(35.5).Margin(10, 0, 0).Bold()
		ui.Column(c).Grow(1).MinWidth(0).Justify(ui.Center).MinHeight(36).Children(field)
	})
}

func (d *desktopWindow) savePreferences(previous desktopPreferences) {
	d.appError = d.preferences.save()
	if d.appError != nil {
		d.preferences = previous
	}
}

func (d *desktopWindow) updateView(c *ui.Context) {
	state := &d.update
	t := d.text
	check := t("检查更新", "Check for Updates")
	if state.err != nil {
		d.updateFailureView(c)
		return
	}
	var heading, detail string
	switch state.stage {
	case updateChecking:
		heading = t("正在检查更新…", "Checking for Updates…")
	case updateAvailable:
		heading = t("有可用更新", "Update Available")
		detail = fmt.Sprintf(t("%s 可下载。", "%s is available to download."), state.release.Version)
	case updateDownloading:
		heading = t("正在下载更新…", "Downloading Update…")
		if state.total > 0 {
			detail = fmt.Sprintf("%d%%", int(float64(state.received)/float64(state.total)*100))
		}
	case updateReady:
		heading = t("更新已下载", "Update Downloaded")
		detail = fmt.Sprintf(t("%s 已就绪。重启后完成安装。", "%s is ready. Restart to finish installing."), state.release.Version)
	case updateCurrent:
		heading = t("已是最新版本", "Up to date")
	case updateUnavailable:
		heading = t("暂无可用的桌面更新", "No desktop update available")
	}
	if heading != "" {
		ui.Text(c, heading).FontSize(18).Bold()
	}
	if detail != "" {
		ui.Text(c, detail).TextColor(c.Theme().TextMuted)
	}
	ui.Row(c).FillWidth().Wrap().Gap(8).Margin(11, 0, 0).Children(func() {
		switch state.stage {
		case updateReady:
			if ui.PrimaryButton(c.Key("restart-update"), t("重启并安装", "Restart and Install")).Height(36).PaddingX(14).Clicked() {
				restart := func() {
					if d.session == nil {
						mygo.App.Relaunch()
						return
					}
					mygo.App.Relaunch(d.session.Root)
				}
				if !d.guardClose(restart) {
					restart()
				}
			}
		case updateAvailable:
			if ui.PrimaryButton(c.Key("download-update"), t("下载更新", "Download Update")).Height(36).PaddingX(14).Clicked() {
				d.downloadUpdate()
			}
		}
		locked := state.stage == updateChecking || state.stage == updateDownloading || state.stage == updateReady
		if ui.Button(c.Key("check-updates"), check).Height(36).PaddingX(14).Disabled(locked).Clicked() {
			d.checkUpdate(false)
		}
	})
}

func (d *desktopWindow) updateFailureView(c *ui.Context) {
	detail := d.update.err.Error()
	heading := d.text("暂时无法检查更新", "Couldn’t check for updates")
	action := d.text("重新检查", "Retry Check")
	if d.update.release != nil {
		heading = d.text("更新下载失败", "Couldn’t download the update")
		action = d.text("重新下载", "Retry Download")
	}
	panel := ui.Column(c.Key("update-error")).FillWidth().Padding(16).Gap(8).
		Radius(8).Background(c.Theme().Surface)
	panel.Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Text(c, heading).Grow(1).MinWidth(0).Bold()
			if ui.Button(c.Key("retry-update"), action).Height(desktopControlHeight).PaddingX(14).Shrink(0).Clicked() {
				if d.update.release != nil {
					d.downloadUpdate()
				} else {
					d.checkUpdate(false)
				}
			}
		})
		d.diagnostic(c, panel, detail)
	})
}

func (d *desktopWindow) checkUpdate(automatic bool) {
	d.update = desktopUpdate{stage: updateChecking}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		release, err := mygo.Updater.Check(ctx)
		mygo.RunOnMain(func() {
			d.update = updateCheckResult(release, err)
			if d.update.stage == updateAvailable && automatic && d.preferences.AutoCheck {
				d.downloadUpdate()
			}
			d.window.Invalidate()
		})
	}()
}

func updateCheckResult(release *mygo.Update, err error) desktopUpdate {
	switch {
	case errors.Is(err, mygo.ErrUpdatesDisabled):
		return desktopUpdate{stage: updateUnavailable}
	case err != nil:
		// MyGo 0.10 does not expose an HTTP status or a no-release error type.
		for cause := err; cause != nil; cause = errors.Unwrap(cause) {
			message := cause.Error()
			if message == "no release is published" ||
				(strings.HasPrefix(message, "GET "+projectURL+"/releases/download/desktop-v") && strings.HasSuffix(message, ": 404 Not Found")) {
				return desktopUpdate{stage: updateUnavailable}
			}
		}
		return desktopUpdate{stage: updateIdle, err: err}
	case release == nil:
		return desktopUpdate{stage: updateCurrent}
	default:
		return desktopUpdate{stage: updateAvailable, release: release}
	}
}

func (d *desktopWindow) downloadUpdate() {
	release := d.update.release
	d.update = desktopUpdate{stage: updateDownloading, release: release}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		err := release.Install(ctx, func(received, total int64) {
			mygo.RunOnMain(func() {
				d.update.received = received
				d.update.total = total
				d.window.Invalidate()
			})
		})
		mygo.RunOnMain(func() {
			if err != nil {
				d.update.stage = updateAvailable
				d.update.err = err
			} else {
				d.update.stage = updateReady
			}
			d.window.Invalidate()
		})
	}()
}
