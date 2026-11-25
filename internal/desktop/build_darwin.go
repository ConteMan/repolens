package desktop

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"time"

	build "github.com/ConteMan/repolens/internal/ui"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type desktopBuildState struct {
	output             *string
	result             build.NativeBuildResult
	finished           time.Time
	running, overwrite bool
	err                error
	server             *http.Server
	previewURL         string
}

func (b *desktopBuildState) reset() {
	if b.server != nil {
		_ = b.server.Close()
	}
	*b = desktopBuildState{}
}

func (d *desktopWindow) runBuild(confirm bool) {
	if d.busy || d.session == nil {
		return
	}
	root, external, output := d.session.Root, d.session.BuilderPath, d.buildState.output
	session := d.session
	d.busy = true
	d.buildState.running = true
	d.buildState.overwrite = false
	d.buildState.err = nil
	go func() {
		result, err := build.BuildNative(context.Background(), root, external, output, confirm)
		mygo.RunOnMain(func() {
			d.busy = false
			defer d.window.Invalidate()
			if d.session != session {
				return
			}
			b := &d.buildState
			b.running = false
			b.overwrite = errors.Is(err, build.ErrConfirmOutput)
			if !b.overwrite {
				b.err = err
			}
			if err == nil {
				if b.server != nil {
					_ = b.server.Close()
					b.server = nil
				}
				b.result = result
				b.finished = time.Now()
			}
		})
	}()
}

func (b *desktopBuildState) startPreview() error {
	if b.server == nil {
		ln, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			return err
		}
		b.server = &http.Server{Handler: http.FileServer(http.Dir(b.result.Directory)), ReadHeaderTimeout: 5 * time.Second}
		b.previewURL = "http://" + ln.Addr().String() + "/view/"
		go b.server.Serve(ln)
	}
	return nil
}

func (d *desktopWindow) pickOutput() {
	d.picking = true
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: d.window, Directory: true, Title: d.text("选择输出目录", "Choose Output Directory")})
		mygo.RunOnMain(func() {
			d.picking = false
			d.buildState.err = err
			if len(paths) > 0 {
				d.buildState.output = &paths[0]
				d.buildState.overwrite = false
			}
			d.window.Invalidate()
		})
	}()
}

func (d *desktopWindow) buildPage(c *ui.Context) {
	b := &d.buildState
	ui.Scroll(c.Key("build-panel")).Fill().Children(func() {
		ui.Column(c).FillWidth().PaddingX(38).Gap(20).Children(func() {
			ui.Row(c).FillWidth().Height(44).AlignItems(ui.Center).Children(func() {
				ui.Text(c, d.text("构建来源", "Build source")).WidthPercent(37).Bold()
				label := d.text("当前工作目录（含未提交修改）", "Working tree, including local changes")
				if src := d.session.Builder.Source.Repo; src != nil && *src != "" {
					label = *src
				}
				if ref := d.session.Builder.Source.Ref; ref != nil && *ref != "" {
					label += " @ " + *ref
				}
				ui.Text(c, label).TextColor(c.Theme().TextMuted).Grow(1).TruncateMiddle().Tooltip(label)
			})
			ui.Row(c).FillWidth().Height(44).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, d.text("输出位置", "Output location")).WidthPercent(37).Bold()
				label := build.NativeOutputPath(d.session.BuilderPath, d.session.Builder.Output.Dir)
				if b.output != nil {
					label = *b.output
				}
				if label == "" {
					label = d.text("应用缓存", "Application cache")
				}
				ui.Text(c, label).TextColor(c.Theme().TextMuted).Grow(1).MinWidth(0).TruncateMiddle().Tooltip(label)
				if b.output != nil && ui.Button(c.Key("reset-output"), d.text("恢复默认", "Use default")).Disabled(d.busy).Clicked() {
					b.output = nil
					b.err = nil
					b.overwrite = false
				}
				if ui.Button(c.Key("output-directory"), d.text("更改目录", "Change folder")).Disabled(d.picking || d.busy).Clicked() {
					d.pickOutput()
				}
			})
			name := d.text("构建站点", "Build site")
			if !b.finished.IsZero() {
				name = d.text("重新构建", "Rebuild")
			}
			if b.running {
				name = d.text("构建中…", "Building…")
			}
			ui.Row(c).Margin(20, 0, 0, 0).Gap(12).AlignItems(ui.Center).Children(func() {
				if ui.Button(c.Key("start-build"), name).Height(36).Disabled(d.busy || d.dirty()).Clicked() {
					d.runBuild(false)
				}
				if d.dirty() {
					ui.Text(c, d.text("请先保存站点设置。", "Save your settings first.")).TextColor(c.Theme().TextMuted)
				}
			})
			if b.overwrite {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, d.text("替换此目录中的已有站点？", "Replace the existing site in this folder?"))
					if ui.Button(c, d.text("取消", "Cancel")).Clicked() {
						b.overwrite = false
					}
					if ui.PrimaryButton(c, d.text("替换并构建", "Replace and build")).Clicked() {
						b.overwrite = false
						d.runBuild(true)
					}
				})
			}
			if b.err != nil {
				d.buildIssue(c, b.err)
			}
			if build.BuildOutputErrorCode(b.err) == "output_not_owned" {
				if ui.Button(c.Key("build-in-cache"), d.text("在应用缓存中构建", "Build in application cache")).Disabled(d.busy || d.dirty()).Clicked() {
					cache := ""
					b.output = &cache
					d.runBuild(false)
				}
			}
			if b.finished.IsZero() {
				return
			}
			ui.Row(c).Margin(22, 0, 10, 0).Height(1).FillWidth().Background(c.Theme().Border)
			ui.Text(c, d.text("构建成功", "Build succeeded")).FontSize(24).Bold()
			source := b.result.Source
			if source == d.session.Root {
				source = d.text("本地工作目录", "Local working tree")
			}
			date := b.finished.Format("2006-01-02 15:04")
			if b.finished.Format("2006-01-02") == time.Now().Format("2006-01-02") {
				date = d.text("今天 ", "Today ") + b.finished.Format("15:04")
			}
			ui.Text(c, fmt.Sprintf(d.text("%s  ·  %s  ·  %d 个页面", "%s  ·  %s  ·  %d pages"), date, source, b.result.Pages)).TextColor(c.Theme().TextMuted).Tooltip(b.result.Source)
			ui.Row(c).Margin(20, 0, 0, 0).Gap(8).Children(func() {
				if ui.PrimaryButton(c.Key("preview-build"), d.text("打开站点预览", "Open site preview")).Height(36).PaddingX(14).Clicked() {
					b.err = b.startPreview()
					if b.err == nil {
						b.err = mygo.Shell.OpenExternal(b.previewURL)
					}
				}
				if ui.Button(c.Key("reveal-build"), d.text("在文件夹中查看", "Show in Finder")).Height(36).PaddingX(14).Clicked() {
					mygo.Shell.ShowItemInFolder(b.result.Directory)
				}
			})
			for _, warning := range b.result.Warnings {
				d.buildWarning(c, warning)
			}
		})
	})
}
