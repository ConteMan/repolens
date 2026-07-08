package theme

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ConteMan/repolens/internal/render"
	"github.com/ConteMan/repolens/internal/source"
)

//go:embed templates/*.tmpl assets/site.css assets/site.js assets/mermaid.min.js
var embedded embed.FS

// Renderer owns the parsed template set and static theme resources.
type Renderer struct {
	templates    *template.Template
	customCSS    string
	hasCustomCSS bool
	vars         map[string]string
}

// Crumb is one breadcrumb item. Href is already relative to the current page;
// Current marks the terminal crumb so templates can render it without a link.
type Crumb struct {
	Label   string
	Href    string
	Current bool
}

// TreeNode is the file-tree shape consumed by the "tree" template. Href is
// already relative to the current page. Directories use Expanded for the
// current-path ancestors and Current when the directory page itself is active.
type TreeNode struct {
	Name     string
	Path     string
	Href     string
	Kind     string
	IsDir    bool
	Current  bool
	Expanded bool
	Children []*TreeNode
}

// DirEntry is one child row rendered by the "dirlist" template. Href is
// already relative to the current directory page.
type DirEntry struct {
	Name       string
	Path       string
	Href       string
	Kind       string
	Size       int64
	LastCommit *source.Commit
	IsDir      bool
}

type PageData struct {
	Title, SiteTitle    string
	RelRoot             string
	Breadcrumbs         []Crumb
	Tree                *TreeNode
	Kind                string
	KindLabel           string
	Body                template.HTML
	TOC                 []render.TOCItem
	MirrorHref          string
	SourceHref          string
	FileSize            int64
	RepoPath            string
	TOCPanel            string
	UI                  map[string]string
	LastCommit          *source.Commit
	HasMermaid, NoIndex bool
	HeadExtra           template.HTML
	DirEntries          []DirEntry
	// Lang 为 <html lang> 属性值（site.language），空则省略该属性。
	Lang string
}

// New creates a Renderer from built-in templates plus optional user overrides.
func New(overrideDir, customCSS string, vars map[string]string) (*Renderer, error) {
	tpl, err := template.New("layout").Funcs(template.FuncMap{
		"shortHash":      shortHash,
		"shortHash7":     shortHash7,
		"formatDate":     formatDate,
		"formatInfoDate": formatInfoDate,
		"formatSize":     formatSize,
		"iconName":       iconName,
		"baseName":       pathBase,
	}).ParseFS(embedded, "templates/*.tmpl")
	if err != nil {
		return nil, err
	}
	if overrideDir != "" {
		if err := applyOverrides(tpl, overrideDir); err != nil {
			return nil, err
		}
	}

	copiedVars := make(map[string]string, len(vars))
	for k, v := range vars {
		copiedVars[k] = v
	}
	return &Renderer{
		templates:    tpl,
		customCSS:    customCSS,
		hasCustomCSS: customCSS != "",
		vars:         copiedVars,
	}, nil
}

func (r *Renderer) Page(w io.Writer, d PageData) error {
	view := struct {
		PageData
		VarsCSS      template.CSS
		HasCustomCSS bool
	}{
		PageData:     fillPageDefaults(d),
		VarsCSS:      template.CSS(r.varsCSS()),
		HasCustomCSS: r.hasCustomCSS,
	}
	return r.templates.ExecuteTemplate(w, "layout", view)
}

func fillPageDefaults(d PageData) PageData {
	if d.TOCPanel == "" {
		d.TOCPanel = "floating"
	}
	if d.UI == nil {
		d.UI = UIStrings(d.Lang)
	}
	if d.KindLabel == "" {
		d.KindLabel = fallbackKindLabel(d.Kind, d.UI)
	}
	return d
}

func (r *Renderer) WriteAssets(outDir string) error {
	assetsDir := filepath.Join(outDir, "_assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"site.css", "site.js", "mermaid.min.js"} {
		if err := writeEmbeddedAsset(assetsDir, name); err != nil {
			return err
		}
	}

	chromaCSS, err := chromaCSS()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "chroma.css"), []byte(chromaCSS), 0o644); err != nil {
		return err
	}
	if r.customCSS != "" {
		data, err := os.ReadFile(r.customCSS)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(assetsDir, "custom.css"), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeEmbeddedAsset(assetsDir, name string) error {
	data, err := fs.ReadFile(embedded, "assets/"+name)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(assetsDir, name), data, 0o644)
}

func chromaCSS() (string, error) {
	light, err := render.StylesCSS("github")
	if err != nil {
		return "", err
	}
	dark, err := render.StylesCSS("github-dark")
	if err != nil {
		return "", err
	}
	return light + "\n@media (prefers-color-scheme: dark) {\n" + dark + "\n}\n", nil
}

func applyOverrides(tpl *template.Template, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		switch name {
		case "layout", "page", "dirlist", "tree":
		default:
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		text := string(data)
		if !strings.Contains(text, "{{define") {
			text = `{{define "` + name + `"}}` + text + `{{end}}`
		}
		if _, err := tpl.Parse(text); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) varsCSS() string {
	if len(r.vars) == 0 {
		return ""
	}
	keys := make([]string, 0, len(r.vars))
	for key := range r.vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(":root{")
	for _, key := range keys {
		b.WriteString("--")
		b.WriteString(key)
		b.WriteByte(':')
		b.WriteString(r.vars[key])
		b.WriteByte(';')
	}
	b.WriteByte('}')
	return b.String()
}

func shortHash(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}

func shortHash7(hash string) string {
	if len(hash) <= 7 {
		return hash
	}
	return hash[:7]
}

func formatDate(commit *source.Commit) string {
	if commit == nil {
		return ""
	}
	return commit.Time.Format("2006-01-02")
}

func formatInfoDate(commit *source.Commit) string {
	if commit == nil {
		return ""
	}
	return commit.Time.Format("2006-01-02 15:04")
}

func formatSize(size int64, isDir bool) string {
	if isDir {
		return "-"
	}
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB"}
	value := float64(size)
	for _, unit := range units {
		value = value / 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%.1f PiB", value/1024)
}

func iconName(kind string, isDir bool) string {
	if isDir || kind == "dir" {
		return "folder"
	}
	switch kind {
	case "markdown", "markdown-source":
		return "markdown"
	case "html", "html-embed", "html-direct", "html-source":
		return "html"
	case "code":
		return "code"
	case "image":
		return "image"
	case "binary":
		return "binary"
	default:
		return "file"
	}
}

func pathBase(p string) string {
	if p == "" {
		return ""
	}
	return path.Base(p)
}

// UIStrings returns built-in theme UI strings for site.language. Chinese
// variants use zh; every other language falls back to English.
func UIStrings(lang string) map[string]string {
	source := uiEN
	if strings.HasPrefix(strings.ToLower(lang), "zh") {
		source = uiZH
	}
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func fallbackKindLabel(kind string, ui map[string]string) string {
	switch kind {
	case "markdown":
		return ui["kind_markdown"]
	case "markdown-source":
		return ui["kind_markdown_source"]
	case "html", "html-embed", "html-direct":
		return ui["kind_html"]
	case "html-source":
		return ui["kind_html_source"]
	case "code":
		return ui["kind_code"]
	case "image":
		return ui["kind_image"]
	case "binary":
		return ui["kind_binary"]
	case "dir":
		return ui["kind_dir"]
	default:
		return kind
	}
}

var uiZH = map[string]string{
	"tree":                 "文件树",
	"tree_title":           "文件树（收起/浮出）",
	"pin_tree":             "固定",
	"pin_tree_title":       "固定为侧栏",
	"back":                 "后退",
	"forward":              "前进",
	"toc":                  "本页目录",
	"toc_title":            "目录面板",
	"zoom_out":             "缩小",
	"zoom_in":              "放大",
	"width":                "布局宽度",
	"width_narrow":         "窄栏",
	"width_default":        "默认",
	"width_full":           "全宽",
	"info":                 "页面信息",
	"path":                 "路径",
	"type":                 "类型",
	"size":                 "大小",
	"last_updated":         "最后更新",
	"commit":               "Commit",
	"view_raw":             "查看原始文件",
	"copy_path":            "复制路径",
	"copied":               "已复制路径",
	"source":               "查看源码",
	"return_rendered":      "返回渲染视图",
	"download":             "下载",
	"download_raw":         "原始文件",
	"search":               "搜索（/）",
	"search_label":         "站内搜索",
	"search_placeholder":   "搜索文件与章节...",
	"kind_markdown":        "Markdown",
	"kind_markdown_source": "Markdown 源码",
	"kind_html":            "HTML",
	"kind_html_source":     "HTML 源码",
	"kind_code":            "代码",
	"kind_image":           "图片",
	"kind_binary":          "二进制",
	"kind_dir":             "目录",
}

var uiEN = map[string]string{
	"tree":                 "Repository tree",
	"tree_title":           "Repository tree",
	"pin_tree":             "Pin",
	"pin_tree_title":       "Pin as sidebar",
	"back":                 "Back",
	"forward":              "Forward",
	"toc":                  "On this page",
	"toc_title":            "Table of contents",
	"zoom_out":             "Zoom out",
	"zoom_in":              "Zoom in",
	"width":                "Layout width",
	"width_narrow":         "Narrow",
	"width_default":        "Default",
	"width_full":           "Full",
	"info":                 "Page information",
	"path":                 "Path",
	"type":                 "Type",
	"size":                 "Size",
	"last_updated":         "Last updated",
	"commit":               "Commit",
	"view_raw":             "View raw file",
	"copy_path":            "Copy path",
	"copied":               "Path copied",
	"source":               "View source",
	"return_rendered":      "Back to rendered view",
	"download":             "Download",
	"download_raw":         "Raw file",
	"search":               "Search (/)",
	"search_label":         "Site search",
	"search_placeholder":   "Search files and headings...",
	"kind_markdown":        "Markdown",
	"kind_markdown_source": "Markdown source",
	"kind_html":            "HTML",
	"kind_html_source":     "HTML source",
	"kind_code":            "Code",
	"kind_image":           "Image",
	"kind_binary":          "Binary",
	"kind_dir":             "Directory",
}
