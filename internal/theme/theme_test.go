package theme

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ConteMan/repolens/internal/render"
	"github.com/ConteMan/repolens/internal/source"
)

func TestPageKindsRender(t *testing.T) {
	t.Parallel()

	renderer := newRendererWithLayout(t, `{{template "page" .}}`)
	for _, tc := range []struct {
		name string
		kind string
		body string
		want string
	}{
		{name: "markdown", kind: "markdown", body: "<p>Markdown</p>", want: `class="page page-markdown"`},
		{name: "html embed", kind: "html-embed", body: `<iframe class="html-preview"></iframe>`, want: `class="page page-html-embed"`},
		{name: "html direct", kind: "html-direct", body: `<div class="toolbar"></div>`, want: `class="page page-html-direct"`},
		{name: "code", kind: "code", body: `<pre class="chroma"></pre>`, want: `class="page page-code"`},
		{name: "image", kind: "image", body: `<img class="preview" alt="">`, want: `class="page page-image"`},
		{name: "binary", kind: "binary", body: `<a download>Download</a>`, want: `class="page page-binary"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := renderer.Page(&out, PageData{
				Title:     tc.name,
				SiteTitle: "test",
				Kind:      tc.kind,
				Body:      html(tc.body),
			})
			if err != nil {
				t.Fatalf("Page() error = %v", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("output missing %q\n%s", tc.want, out.String())
			}
			assertGoldenCompact(t, "kind_"+tc.kind+".golden.html", out.String())
		})
	}
}

func TestDirlistGoldenHTML(t *testing.T) {
	t.Parallel()

	renderer := newRendererWithLayout(t, `{{template "page" .}}`)
	var out bytes.Buffer
	err := renderer.Page(&out, PageData{
		Title:     "docs",
		SiteTitle: "test",
		Kind:      "dir",
		Body:      html("<p>Intro</p>"),
		DirEntries: []DirEntry{
			{Name: "guides", Href: "guides/", Kind: "dir", IsDir: true},
			{
				Name:       "README.md",
				Href:       "README.md/",
				Kind:       "markdown",
				Size:       1536,
				LastCommit: &source.Commit{Hash: "0123456789abcdef", Time: time.Date(2026, 7, 4, 10, 0, 0, 0, time.UTC)},
			},
		},
	})
	if err != nil {
		t.Fatalf("Page() error = %v", err)
	}
	assertGoldenCompact(t, "dirlist.golden.html", out.String())
}

func TestVarsCustomCSSAndMermaidInjection(t *testing.T) {
	t.Parallel()

	customCSS := filepath.Join(t.TempDir(), "theme.css")
	if err := os.WriteFile(customCSS, []byte(".custom{color:red}\n"), 0o644); err != nil {
		t.Fatalf("write custom css: %v", err)
	}
	renderer, err := New("", customCSS, map[string]string{
		"sidebar-width": "340px",
		"accent":        "#c2410c",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var out bytes.Buffer
	if err := renderer.Page(&out, PageData{Title: "x", SiteTitle: "test", Kind: "markdown"}); err != nil {
		t.Fatalf("Page() error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, `<style>:root{--accent:#c2410c;--sidebar-width:340px;}</style>`) {
		t.Fatalf("vars CSS missing or unsorted\n%s", got)
	}
	if !strings.Contains(got, `_assets/custom.css`) {
		t.Fatalf("custom CSS link missing\n%s", got)
	}
	if strings.Contains(got, `mermaid.min.js`) {
		t.Fatalf("mermaid script injected without HasMermaid\n%s", got)
	}

	out.Reset()
	if err := renderer.Page(&out, PageData{Title: "x", SiteTitle: "test", Kind: "markdown", HasMermaid: true}); err != nil {
		t.Fatalf("Page() with mermaid error = %v", err)
	}
	if !strings.Contains(out.String(), `_assets/mermaid.min.js`) {
		t.Fatalf("mermaid script missing when HasMermaid=true\n%s", out.String())
	}
}

func TestHybridTreeLayoutRendering(t *testing.T) {
	t.Parallel()

	renderer, err := New("", "", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var out bytes.Buffer
	err = renderer.Page(&out, PageData{
		Title:     "guide.md",
		SiteTitle: "test",
		RelRoot:   "../../",
		Kind:      "markdown",
		Body:      html("<p>Guide</p>"),
		Tree: &TreeNode{IsDir: true, Children: []*TreeNode{
			{
				Name:     "docs",
				Path:     "docs",
				Href:     "../",
				Kind:     "dir",
				IsDir:    true,
				Expanded: true,
				Children: []*TreeNode{
					{Name: "guide.md", Path: "docs/guide.md", Href: ".", Kind: "markdown", Current: true},
				},
			},
		}},
	})
	if err != nil {
		t.Fatalf("Page() error = %v", err)
	}
	got := out.String()

	for _, want := range []string{
		`<div class="tb-group js-only">`,
		`<button class="tb-btn" id="btn-tree" type="button"`,
		`<button class="tb-btn" id="btn-back" type="button"`,
		`<button class="tb-btn" id="btn-fwd" type="button"`,
		`<nav class="tb-crumbs" aria-label="Breadcrumbs">`,
		`<button class="tb-btn js-only" id="btn-zoom-out" type="button"`,
		`<span class="zoom-readout js-only" id="zoom-readout">100%</span>`,
		`<button class="tb-btn js-only" id="btn-width" type="button"`,
		`<span class="info-wrap js-only" id="info-wrap">`,
		`<button class="tb-btn js-only" id="btn-search" type="button"`,
		`<use href="#icon-tree"></use>`,
		`<nav class="tree-nav" id="tree-src" aria-label="Repository tree">`,
		`<div class="tree-search" data-tree-search-placeholder hidden></div>`,
		`<span class="tree-chevron" aria-hidden="true"></span>`,
		`<div class="scrim" id="scrim" aria-hidden="true"></div>`,
		`<div class="overlay" id="tree-overlay" role="dialog" aria-label="Repository tree">`,
		`<nav class="overlay-tree" id="overlay-tree" data-tree-scroll aria-label="Repository tree"></nav>`,
		`<button class="tb-btn" id="btn-pin-tree" type="button"`,
		`<details data-tree-path="docs" open>`,
		`<symbol id="icon-markdown" viewBox="0 0 16 16"><path d="M3 1.5h6.5L13 5v9.5H3z"/><path d="M9.5 1.5V5H13"/><path d="M5.3 8.2h5.4M5.3 10.7h5.4"/></symbol>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q\n%s", want, got)
		}
	}
	if strings.Contains(got, `data-tree="collapsed"`) || strings.Contains(got, `data-tree-mode="floating"`) {
		t.Fatalf("no-JS fallback should render the fixed sidebar by default\n%s", got)
	}
}

func TestHybridTreeAssetsExposeContract(t *testing.T) {
	t.Parallel()

	renderer, err := New("", "", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	outDir := t.TempDir()
	if err := renderer.WriteAssets(outDir); err != nil {
		t.Fatalf("WriteAssets() error = %v", err)
	}

	css := readTestOutput(t, outDir, "_assets/site.css")
	for _, want := range []string{
		`.tree-chevron`,
		`border-right: 1.6px solid var(--muted);`,
		`body[data-tree="collapsed"] .sidebar`,
		`body[data-tree-mode="floating"] .sidebar`,
		`.scrim`,
		`.overlay`,
		`body[data-overlay="open"] .overlay`,
		`.tb-crumbs`,
		`.toc-panel`,
		`body[data-width="narrow"] .content`,
		`.info-panel`,
		`.dl-menu`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("site.css missing %q\n%s", want, css)
		}
	}

	js := readTestOutput(t, outDir, "_assets/site.js")
	for _, want := range []string{
		`var treePreferenceKey = "repolens:tree:preference";`,
		`storageGet(window.localStorage, key)`,
		`window.matchMedia("(max-width: 1023px)")`,
		`overlayTree.innerHTML = treeSource.innerHTML;`,
		`event.key === "Escape"`,
		`sessionSet(keyFor(detail), detail.open ? "open" : "closed");`,
		`sessionSet(scrollKey, String(container.scrollTop));`,
		`window.fetch(url.href, { credentials: "same-origin" })`,
		`window.history.pushState({ pjax: true }, "", url.href);`,
		`navigator.clipboard.writeText(text)`,
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("site.js missing %q\n%s", want, js)
		}
	}
}

func TestToolbarPageDataAndLanguage(t *testing.T) {
	t.Parallel()

	renderer, err := New("", "", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var out bytes.Buffer
	err = renderer.Page(&out, PageData{
		Title:      "guide.md",
		SiteTitle:  "test",
		Lang:       "en",
		RelRoot:    "../../",
		Kind:       "markdown",
		RepoPath:   "docs/guide.md",
		FileSize:   1536,
		MirrorHref: "../../docs/guide.md",
		SourceHref: "source/",
		TOCPanel:   "floating",
		TOC:        []render.TOCItem{{Level: 2, Anchor: "intro", Title: "Intro"}},
		LastCommit: &source.Commit{
			Hash:    "0123456789abcdef",
			Time:    time.Date(2026, 7, 8, 9, 30, 0, 0, time.UTC),
			Subject: "docs: update guide",
		},
		Breadcrumbs: []Crumb{
			{Label: "root", Href: "../../"},
			{Label: "docs", Href: "../"},
			{Label: "guide.md", Current: true},
		},
	})
	if err != nil {
		t.Fatalf("Page() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{
		`<html class="no-js" lang="en">`,
		`title="Page information"`,
		`<div class="info-row"><span>Path</span><code data-page-path>docs/guide.md</code></div>`,
		`<div class="info-row"><span>Size</span><b>1.5 KiB</b></div>`,
		`<code>0123456</code><small>docs: update guide</small>`,
		`href="../../docs/guide.md"`,
		`href="source/"`,
		`Raw file guide.md`,
		`Search (/)`,
		`<aside class="toc-panel" id="toc-panel" aria-label="Table of contents">`,
		`<a href="#intro" data-lv="2">Intro</a>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q\n%s", want, got)
		}
	}
	if strings.Contains(got, `footer class="meta"`) {
		t.Fatalf("footer meta should not render\n%s", got)
	}
}

func TestTemplateOverrideIncludesDirlist(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, dir, "layout.html", `{{template "page" .}}`)
	writeTestFile(t, dir, "dirlist.html", `override-dirlist {{len .DirEntries}}`)
	renderer, err := New(dir, "", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var out bytes.Buffer
	err = renderer.Page(&out, PageData{
		Title:      "dir",
		SiteTitle:  "test",
		Kind:       "dir",
		DirEntries: []DirEntry{{Name: "a", Href: "a", Kind: "code"}},
	})
	if err != nil {
		t.Fatalf("Page() error = %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "override-dirlist 1" {
		t.Fatalf("override output = %q", got)
	}
}

func TestWriteAssets(t *testing.T) {
	t.Parallel()

	customCSS := filepath.Join(t.TempDir(), "theme.css")
	if err := os.WriteFile(customCSS, []byte(".custom{color:red}\n"), 0o644); err != nil {
		t.Fatalf("write custom css: %v", err)
	}
	renderer, err := New("", customCSS, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	outDir := t.TempDir()
	if err := renderer.WriteAssets(outDir); err != nil {
		t.Fatalf("WriteAssets() error = %v", err)
	}

	for _, rel := range []string{
		"_assets/site.css",
		"_assets/site.js",
		"_assets/chroma.css",
		"_assets/custom.css",
		"_assets/mermaid.min.js",
	} {
		data, err := os.ReadFile(filepath.Join(outDir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if len(data) == 0 {
			t.Fatalf("%s is empty", rel)
		}
		if hasExternalResourceReference(string(data)) {
			t.Fatalf("%s contains an external resource reference", rel)
		}
	}
	chroma := readTestOutput(t, outDir, "_assets/chroma.css")
	if !strings.Contains(chroma, "@media (prefers-color-scheme: dark)") {
		t.Fatalf("chroma.css missing dark media query\n%s", chroma)
	}
}

func newRendererWithLayout(t *testing.T, layout string) *Renderer {
	t.Helper()
	dir := t.TempDir()
	writeTestFile(t, dir, "layout.html", layout)
	renderer, err := New(dir, "", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return renderer
}

func html(s string) template.HTML {
	return template.HTML(s)
}

func writeTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func readTestOutput(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func assertGoldenCompact(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("update golden %s: %v", name, err)
		}
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	if compactHTML(got) != compactHTML(string(data)) {
		t.Fatalf("golden %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, string(data))
	}
}

func compactHTML(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func hasExternalResourceReference(s string) bool {
	for _, needle := range []string{`href="/`, `src="/`, `src="http://`, `src="https://`, `url(http://`, `url(https://`} {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
