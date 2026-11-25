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

var (
	siteIcon         = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16"><path d="M3 3h10v10H3zM3 6h10M7 6v7" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linejoin="round"/></svg>`))
	buildIcon        = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16"><path d="m6 4 6 4-6 4z" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linejoin="round"/></svg>`))
	preferencesIcon  = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16"><path d="m6.7 2-.4 1.5-1.5.9-1.5-.4-1.3 2.2 1.1 1.1v1.4L2 9.8 3.3 12l1.5-.4 1.5.9.4 1.5h2.6l.4-1.5 1.5-.9 1.5.4L14 9.8l-1.1-1.1V7.3L14 6.2 12.7 4l-1.5.4-1.5-.9L9.3 2z" fill="none" stroke="currentColor" stroke-linejoin="round"/><circle cx="8" cy="8" r="2" fill="none" stroke="currentColor"/></svg>`))
	externalLinkIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16"><path d="M6 10l6-6m-5 0h5v5" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
)

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
		ui.Column(c).FillWidth().Padding(0, 38, 16).Gap(16).Children(func() {
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
				if ui.Select(c.Key("interface-language"), &selected, labels).FillWidth().Height(36).Changed() {
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
				link := ui.ButtonBase(c.Key("project-link")).Label("github.com/ConteMan/repolens").Height(36).AlignItems(ui.Center).Gap(4)
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
			ui.Column(c).FillWidth().Gap(12).Children(func() {
				d.updateView(c)
			})
			if d.appError != nil {
				ui.Text(c, d.appError.Error()).TextColor(t.Danger).Selectable()
			}
		})
	})
}

func (d *desktopWindow) preferenceRow(c *ui.Context, label string, field func()) {
	width, _ := c.Size()
	ui.Row(c).FillWidth().Wrap().AlignItems(ui.Center).Gap(12).MinHeight(38).Children(func() {
		ui.Text(c, label).Width(min(248, width*0.24)).Bold()
		ui.Column(c).Grow(1).MinWidth(242).Children(field)
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
	switch state.stage {
	case updateChecking:
		ui.Button(c, d.text("正在检查…", "Checking…")).Height(36).Disabled(true)
	case updateAvailable:
		ui.Text(c, d.text("可下载版本 ", "Available version ")+state.release.Version)
		if ui.PrimaryButton(c.Key("download-update"), d.text("下载更新", "Download Update")).Height(36).Clicked() {
			d.downloadUpdate()
		}
	case updateDownloading:
		ui.Button(c, d.text("正在下载…", "Downloading…")).Height(36).Disabled(true)
		if state.total > 0 {
			ui.Textf(c, "%d%%", int(float64(state.received)/float64(state.total)*100))
		}
	case updateReady:
		ui.Text(c, d.text("更新已下载", "Update Downloaded")).Bold()
		ui.Text(c, fmt.Sprintf(d.text("%s 已就绪。重启后完成安装。", "%s is ready. Restart to finish installing."), state.release.Version)).TextColor(c.Theme().TextMuted)
		if ui.PrimaryButton(c.Key("restart-update"), d.text("重启并安装", "Restart and Install")).Height(36).Clicked() {
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
	default:
		if state.stage == updateCurrent {
			ui.Text(c, d.text("已是最新版本。", "You’re up to date."))
		}
		if state.stage == updateUnavailable {
			ui.Text(c, d.text("暂无适用于此 Mac 的桌面更新。", "No desktop update is available for this Mac yet.")).TextColor(c.Theme().TextMuted)
		}
		if ui.Button(c.Key("check-updates"), d.text("检查更新", "Check for Updates")).Height(36).Clicked() {
			d.checkUpdate(false)
		}
	}
	if state.err != nil {
		message := state.err.Error()
		if errors.Is(state.err, mygo.ErrUpdatesDisabled) {
			message = d.text("当前发行包未配置签名更新。", "Signed updates are unavailable for this build.")
		}
		ui.Text(c, message).TextColor(c.Theme().Danger).Selectable()
	}
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
	case err != nil:
		// MyGo 0.10 does not expose an HTTP status or a no-release error type.
		message := err.Error()
		if message == "no release is published" ||
			(strings.HasPrefix(message, "GET https://github.com/") && strings.HasSuffix(message, ": 404 Not Found")) {
			return desktopUpdate{stage: updateUnavailable}
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
				d.update.err = fmt.Errorf("%s: %w", d.text("下载更新失败", "Update download failed"), err)
			} else {
				d.update.stage = updateReady
			}
			d.window.Invalidate()
		})
	}()
}
