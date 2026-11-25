package desktop

import (
	"errors"
	"os"
	"strings"

	"github.com/ConteMan/repolens/internal/config"
	build "github.com/ConteMan/repolens/internal/ui"
	"github.com/egoist/mygo/ui"
)

func (d *desktopWindow) desktopIssue(c *ui.Context, err error, zh, en string) {
	t := c.Theme()
	panel := ui.Column(c).Key(err.Error()).FillWidth().Gap(4)
	panel.Children(func() {
		ui.Text(c, d.text(zh, en)).FillWidth().TextColor(t.Danger)
		d.diagnostic(c, panel, err.Error())
	})
}

func (d *desktopWindow) diagnostic(c *ui.Context, panel ui.Element, detail string) {
	open := ui.Local(panel, "diagnostic", func() bool { return false })
	ui.Collapsible(c, d.text("技术详情", "Technical Details"), open, func() {
		ui.Text(c, detail).FillWidth().Selectable().TextColor(c.Theme().TextMuted)
	})
}

func (d *desktopWindow) openMessage(err error) string {
	switch {
	case errors.Is(err, errNotGitWorktree):
		return d.text("所选文件夹不是 Git 工作树。请选择 Git 仓库的根目录。", "This folder is not a Git working tree. Choose the root folder of a Git repository.")
	case errors.Is(err, os.ErrNotExist):
		return d.text("所选文件夹无法找到。请重新选择仓库。", "The selected folder could not be found. Choose the repository again.")
	case errors.Is(err, os.ErrPermission):
		return d.text("无法读取所选仓库。请检查文件夹权限。", "The selected repository could not be read. Check the folder permissions.")
	default:
		return d.text("无法读取所选仓库或其配置。请检查位置和 .repolens.yml。", "The selected repository or its configuration could not be read. Check the location and .repolens.yml.")
	}
}

func (d *desktopWindow) saveIssue(c *ui.Context) {
	var validation *config.RepositoryValidationError
	switch {
	case errors.Is(d.saveError, config.ErrRevisionConflict):
		d.desktopIssue(c, d.saveError, "配置文件在磁盘上已更改。重新读取后可继续编辑。", "The configuration changed on disk. Reload it to continue editing.")
	case errors.As(d.saveError, &validation):
		d.desktopIssue(c, d.saveError, "配置有待修正的字段。请检查标出的输入。", "Some configuration fields need attention. Check the highlighted input.")
	default:
		d.desktopIssue(c, d.saveError, "无法读取或保存配置。请检查文件路径和权限。", "The configuration could not be read or saved. Check the file path and permissions.")
	}
}

func (d *desktopWindow) buildIssue(c *ui.Context, err error) {
	switch build.BuildOutputErrorCode(err) {
	case "output_not_owned":
		d.desktopIssue(c, err, "非空输出目录缺少 .repolens-build。请选择空目录或使用应用缓存。", "The non-empty output directory has no .repolens-build marker. Choose an empty folder or use the app cache.")
		return
	case "unsafe_output_path":
		d.desktopIssue(c, err, "所选位置无法安全地用于构建输出。请选择另一个目录或使用应用缓存。", "The selected location cannot safely contain the build output. Choose another folder or use the app cache.")
		return
	case "invalid_output_path":
		d.desktopIssue(c, err, "无法访问输出目录。请检查路径和权限，或使用应用缓存。", "The output folder could not be accessed. Check its path and permissions, or use the app cache.")
		return
	}
	d.desktopIssue(c, err, "无法完成构建或打开预览。请查看技术详情。", "The build or preview could not be completed. See the technical details.")
}

func (d *desktopWindow) buildWarning(c *ui.Context, warning string) {
	label := d.text("构建警告", "Build Warning")
	switch {
	case strings.Contains(warning, "is ignored in repository config"):
		label = d.text("仓库配置内的构建者专用设置已忽略。", "Builder-only settings in the repository configuration were ignored.")
	case strings.Contains(warning, "unknown field or schema warning"):
		label = d.text("配置包含未知字段或格式警告。", "The configuration contains an unknown field or a schema warning.")
	case strings.HasPrefix(warning, "view."):
		label = d.text("浏览设置包含不支持的值。", "A browse setting contains an unsupported value.")
	case strings.HasPrefix(warning, "glossary "):
		label = d.text("术语库包含无法应用的内容。", "Some glossary content could not be applied.")
	case strings.HasPrefix(warning, "access.encrypt"):
		label = d.text("加密范围与已启用的 Agent 输出重叠。", "Encryption overlaps enabled Agent outputs.")
	}
	panel := ui.Column(c).Key(warning).FillWidth().Gap(4)
	panel.Children(func() {
		ui.Text(c, label).TextColor(c.Theme().TextMuted)
		d.diagnostic(c, panel, warning)
	})
}
