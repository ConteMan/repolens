package theme

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
