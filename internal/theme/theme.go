package theme

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ConteMan/repolens/internal/render"
	"github.com/ConteMan/repolens/internal/source"
)

const defaultTemplate = `
{{define "layout"}}<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
{{if .NoIndex}}<meta name="robots" content="noindex">{{end}}
<title>{{.Title}} - {{.SiteTitle}}</title>
<link rel="stylesheet" href="{{.RelRoot}}_assets/site.css">
<link rel="stylesheet" href="{{.RelRoot}}_assets/chroma.css">
{{if .HasCustomCSS}}<link rel="stylesheet" href="{{.RelRoot}}_assets/custom.css">{{end}}
{{if .VarsCSS}}<style>{{.VarsCSS}}</style>{{end}}
{{.HeadExtra}}
</head>
<body>
<header class="topbar">
  <a class="brand" href="{{.RelRoot}}view/">{{.SiteTitle}}</a>
  {{if .MirrorHref}}<a class="raw-link" href="{{.MirrorHref}}">Raw</a>{{end}}
</header>
<div class="shell">
  <aside class="sidebar">{{template "tree" .Tree}}</aside>
  <main class="content">
    <nav class="breadcrumbs" aria-label="Breadcrumbs">{{range .Breadcrumbs}}{{if .Current}}<span aria-current="page">{{.Label}}</span>{{else}}<a href="{{.Href}}">{{.Label}}</a>{{end}}{{end}}</nav>
    <h1>{{.Title}}</h1>
    {{template "page" .}}
    {{if .LastCommit}}<footer class="meta">Last updated {{.LastCommit.Time.Format "2006-01-02 15:04"}} · {{shortHash .LastCommit.Hash}}</footer>{{end}}
  </main>
</div>
<script defer src="{{.RelRoot}}_assets/site.js"></script>
</body>
</html>{{end}}

{{define "page"}}
{{if .TOC}}<nav class="toc" aria-label="Table of contents">{{range .TOC}}<a href="#{{.Anchor}}">{{.Title}}</a>{{end}}</nav>{{end}}
<article class="page page-{{.Kind}}">{{.Body}}</article>
{{end}}

{{define "tree"}}{{if .}}<ul class="tree">{{range .Children}}{{template "tree-node" .}}{{end}}</ul>{{end}}{{end}}

{{define "tree-node"}}<li class="{{if .Current}}current {{end}}{{if .IsDir}}dir{{else}}file{{end}}">
{{if .IsDir}}
<details {{if .Expanded}}open{{end}}>
  <summary><a href="{{.Href}}">{{.Name}}</a></summary>
  <ul>{{range .Children}}{{template "tree-node" .}}{{end}}</ul>
</details>
{{else}}
<a href="{{.Href}}">{{.Name}}</a>
{{end}}
</li>{{end}}
`

const siteCSS = `
:root {
  --bg: #ffffff;
  --fg: #24292f;
  --muted: #57606a;
  --border: #d0d7de;
  --accent: #0969da;
  --sidebar-width: 280px;
}
* { box-sizing: border-box; }
body { margin: 0; color: var(--fg); background: var(--bg); font: 16px/1.5 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
a { color: var(--accent); text-decoration: none; }
a:hover { text-decoration: underline; }
.topbar { height: 48px; border-bottom: 1px solid var(--border); display: flex; align-items: center; justify-content: space-between; padding: 0 16px; }
.brand { font-weight: 700; color: var(--fg); }
.shell { display: grid; grid-template-columns: minmax(180px, var(--sidebar-width)) minmax(0, 1fr); min-height: calc(100vh - 48px); }
.sidebar { border-right: 1px solid var(--border); padding: 12px; overflow: auto; }
.content { max-width: 980px; width: 100%; padding: 24px; }
.breadcrumbs { display: flex; flex-wrap: wrap; gap: 8px; color: var(--muted); font-size: 14px; }
.breadcrumbs a::after { content: "/"; margin-left: 8px; color: var(--muted); }
.tree, .tree ul { list-style: none; margin: 0; padding-left: 16px; }
.tree > li { padding-left: 0; }
.tree a { color: var(--fg); }
.tree .current > a, .tree .current > details > summary > a { color: var(--accent); font-weight: 700; }
details > summary { cursor: pointer; }
.toc { border: 1px solid var(--border); padding: 12px; margin: 16px 0; }
.toc a { display: block; }
.page img.preview { max-width: 100%; height: auto; border: 1px solid var(--border); }
.html-preview { width: 100%; min-height: 70vh; border: 1px solid var(--border); }
.toolbar { display: flex; gap: 12px; margin: 16px 0; }
.meta { margin-top: 32px; color: var(--muted); font-size: 14px; }
pre { overflow: auto; }
table { border-collapse: collapse; width: 100%; }
th, td { border: 1px solid var(--border); padding: 6px 8px; text-align: left; }
@media (max-width: 760px) {
  .shell { grid-template-columns: 1fr; }
  .sidebar { border-right: 0; border-bottom: 1px solid var(--border); max-height: 40vh; }
}
`

const siteJS = `
// Placeholder enhancement layer. Spec 006 will replace this with tree state
// persistence and optional partial navigation while preserving no-JS behavior.
`

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

type PageData struct {
	Title, SiteTitle    string
	RelRoot             string
	Breadcrumbs         []Crumb
	Tree                *TreeNode
	Kind                string
	Body                template.HTML
	TOC                 []render.TOCItem
	MirrorHref          string
	LastCommit          *source.Commit
	HasMermaid, NoIndex bool
	HeadExtra           template.HTML
}

// New creates a Renderer. This is intentionally a minimal skeleton for spec
// 005 integration; spec 006 will flesh out the built-in templates, CSS, and JS.
func New(overrideDir, customCSS string, vars map[string]string) (*Renderer, error) {
	tpl, err := template.New("layout").Funcs(template.FuncMap{
		"shortHash": shortHash,
	}).Parse(defaultTemplate)
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
		PageData:     d,
		VarsCSS:      template.CSS(r.varsCSS()),
		HasCustomCSS: r.hasCustomCSS,
	}
	return r.templates.ExecuteTemplate(w, "layout", view)
}

func (r *Renderer) WriteAssets(outDir string) error {
	assetsDir := filepath.Join(outDir, "_assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "site.css"), []byte(siteCSS), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "site.js"), []byte(siteJS), 0o644); err != nil {
		return err
	}
	chromaCSS, err := render.StylesCSS("github")
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

func applyOverrides(tpl *template.Template, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		// "dirlist" 尚未在默认模板中接线（目录列表暂由 site 层拼入 Body），
		// spec 006 落地时再加入白名单，避免用户覆盖文件静默无效。
		switch name {
		case "layout", "page", "tree":
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
