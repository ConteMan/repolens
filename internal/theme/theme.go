package theme

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
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
	Body                template.HTML
	TOC                 []render.TOCItem
	MirrorHref          string
	LastCommit          *source.Commit
	HasMermaid, NoIndex bool
	HeadExtra           template.HTML
	DirEntries          []DirEntry
}

// New creates a Renderer from built-in templates plus optional user overrides.
func New(overrideDir, customCSS string, vars map[string]string) (*Renderer, error) {
	tpl, err := template.New("layout").Funcs(template.FuncMap{
		"shortHash":  shortHash,
		"formatDate": formatDate,
		"formatSize": formatSize,
		"iconName":   iconName,
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

func formatDate(commit *source.Commit) string {
	if commit == nil {
		return ""
	}
	return commit.Time.Format("2006-01-02")
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
	case "markdown":
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
