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
		ui.Column(c).FillWidth().PaddingX(desktopInset(c)).Gap(20).Children(func() {
			locations, locationErr := build.ResolveNativeBuildLocations(d.session.Root, d.session.BuilderPath, d.session.Builder, b.output)
			pathRow := func(key, label, value, tip string, actions func()) {
				ui.Row(c.Key(key)).FillWidth().Height(44).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, label).Width(108).Shrink(0).Bold()
					ui.Text(c, value).Width(0).Grow(1).MinWidth(0).TextColor(c.Theme().TextMuted).SingleLine().TruncateMiddle().Tooltip(tip)
					if actions != nil {
						actions()
					}
				})
			}
			src := locations.Source
			tip := src.Repo
			if src.Worktree {
				tip += d.text("\n包含未提交的修改", "\nIncludes uncommitted changes")
			}
			pathRow("build-source", d.text("构建来源", "Build source"), src.Repo, tip, nil)
			if src.Ref != "" {
				pathRow("build-ref", d.text("Git 引用", "Git ref"), src.Ref, src.Ref, nil)
			}
			if locations.RemoteMirror != "" {
				pathRow("source-mirror", d.text("仓库缓存", "Repository cache"), locations.RemoteMirror, locations.RemoteMirror, nil)
			}
			if path := d.session.BuilderPath; path != "" {
				pathRow("build-config", d.text("构建者配置", "Builder config"), path, path, nil)
			}
			pathRow("build-output", d.text("输出位置", "Output location"), locations.Output, locations.Output, func() {
				if b.output != nil && ui.Button(c.Key("reset-output"), d.text("恢复默认", "Use default")).Disabled(d.busy).Clicked() {
					b.output = nil
					b.err = nil
					b.overwrite = false
				}
				if ui.Button(c.Key("output-directory"), d.text("更改目录", "Change folder")).Disabled(d.picking || d.busy).Clicked() {
					d.pickOutput()
				}
			})
			if locationErr != nil {
				d.buildIssue(c, locationErr)
			}
			name := d.text("构建站点", "Build site")
			if !b.finished.IsZero() {
				name = d.text("重新构建", "Rebuild")
			}
			if b.running {
				name = d.text("构建中…", "Building…")
			}
			ui.Row(c).Margin(20, 0, 0, 0).Gap(12).AlignItems(ui.Center).Children(func() {
				if ui.Button(c.Key("start-build"), name).Height(36).Disabled(d.busy || d.dirty() || locationErr != nil).Clicked() {
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
			date := b.finished.Format("2006-01-02 15:04")
			if b.finished.Format("2006-01-02") == time.Now().Format("2006-01-02") {
				date = d.text("今天 ", "Today ") + b.finished.Format("15:04")
			}
			ui.Text(c, fmt.Sprintf(d.text("%s  ·  %s  ·  %d 个页面", "%s  ·  %s  ·  %d pages"), date, source, b.result.Pages)).FillWidth().SingleLine().TruncateMiddle().TextColor(c.Theme().TextMuted).Tooltip(b.result.Source)
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
