package site

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/ConteMan/repolens/internal/source"
	"github.com/ConteMan/repolens/internal/theme"
)

func TestBuildEndToEnd(t *testing.T) {
	repo := newSiteTestRepo(t)
	tree, err := source.Open(context.Background(), source.Spec{Repo: repo})
	if err != nil {
		t.Fatalf("source.Open() error = %v", err)
	}
	defer tree.Cleanup()

	cfg, warnings, err := config.Load(tree.Root, "", config.Flags{Repo: repo})
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	renderer, err := theme.New("", "", cfg.Theme.Vars)
	if err != nil {
		t.Fatalf("theme.New() error = %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "dist")
	stats, err := NewBuilder(cfg, renderer).Build(context.Background(), tree, outDir)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if stats.Files != 12 || stats.Pages != 18 {
		t.Fatalf("stats = %#v, want 12 files and 18 pages", stats)
	}

	assertExists(t, outDir, ".repolens-build")
	assertExists(t, outDir, "index.html")
	assertExists(t, outDir, "robots.txt")
	assertExists(t, outDir, "_assets/site.css")
	assertExists(t, outDir, "_assets/site.js")
	assertMissing(t, outDir, ".repolens.yml")

	for _, p := range []string{
		"README.md",
		"page.html",
		"direct.html",
		"source.html",
		"code/main.go",
		"assets/pixel.png",
		"blob.bin",
		"docs/README.md",
		"docs/guide.md",
		"plain/file.txt",
		"skip/hidden.md",
	} {
		assertMirrorEqual(t, repo, outDir, p)
	}

	for _, p := range []string{
		"view/index.html",
		"view/README.md/index.html",
		"view/page.html/index.html",
		"view/direct.html/index.html",
		"view/source.html/index.html",
		"view/code/main.go/index.html",
		"view/assets/pixel.png/index.html",
		"view/blob.bin/index.html",
		"view/docs/index.html",
		"view/docs/guide.md/index.html",
		"view/code/index.html",
		"view/plain/index.html",
		"view/skip/index.html",
	} {
		assertExists(t, outDir, p)
	}
	assertMissing(t, outDir, "view/skip/hidden.md/index.html")

	rootPage := readOutput(t, outDir, "view/index.html")
	assertContains(t, rootPage, `<meta name="robots" content="noindex">`)
	assertContains(t, rootPage, `<section class="readme">`)
	assertContains(t, rootPage, "Home")
	assertContains(t, rootPage, `href="docs/guide.md/"`)
	assertContains(t, rootPage, `src="../assets/pixel.png"`)
	assertContains(t, rootPage, "docs")

	// 普通外部超链接允许出现在产物中（自检只拦资源加载类外部引用）。
	assertContains(t, rootPage, `href="https://example.com/"`)

	docsPage := readOutput(t, outDir, "view/docs/index.html")
	assertContains(t, docsPage, "Docs")
	assertContains(t, docsPage, "guide.md")

	// 小写 readme.md 回退。
	notesPage := readOutput(t, outDir, "view/notes/index.html")
	assertContains(t, notesPage, `<section class="readme">`)
	assertContains(t, notesPage, "Lowercase readme.")

	codeDirPage := readOutput(t, outDir, "view/code/index.html")
	assertContains(t, codeDirPage, `<section class="dir-list">`)
	if strings.Contains(codeDirPage, `<section class="readme">`) {
		t.Fatalf("code directory unexpectedly rendered a README section")
	}

	embedPage := readOutput(t, outDir, "view/page.html/index.html")
	assertContains(t, embedPage, `<iframe class="html-preview" src="../../page.html"`)

	directPage := readOutput(t, outDir, "view/direct.html/index.html")
	assertContains(t, directPage, "Open HTML file")
	assertNotContains(t, directPage, "<iframe")

	sourcePage := readOutput(t, outDir, "view/source.html/index.html")
	assertContains(t, sourcePage, "&lt;")
	assertContains(t, sourcePage, "strong")
	assertContains(t, sourcePage, "source")
	assertNotContains(t, sourcePage, "<iframe")

	imagePage := readOutput(t, outDir, "view/assets/pixel.png/index.html")
	assertContains(t, imagePage, `<img class="preview" src="../../../assets/pixel.png"`)

	binaryPage := readOutput(t, outDir, "view/blob.bin/index.html")
	assertContains(t, binaryPage, "Download")
	assertContains(t, binaryPage, "blob.bin")

	skipDirPage := readOutput(t, outDir, "view/skip/index.html")
	assertContains(t, skipDirPage, `href="../../skip/hidden.md"`)

	robots := readOutput(t, outDir, "robots.txt")
	assertContains(t, robots, "Disallow: /")
}

func TestBuildFailsWhenGeneratedOutputHasAbsoluteLinks(t *testing.T) {
	repo := newSiteTestRepo(t)
	tree, err := source.Open(context.Background(), source.Spec{Repo: repo})
	if err != nil {
		t.Fatalf("source.Open() error = %v", err)
	}
	defer tree.Cleanup()

	cfg, _, err := config.Load(tree.Root, "", config.Flags{Repo: repo})
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	overrides := t.TempDir()
	writeFile(t, overrides, "layout.html", `<!doctype html><html><body><a href="/bad">bad</a><img src="https://example.com/bad.png" alt="">{{template "page" .}}</body></html>`)
	renderer, err := theme.New(overrides, "", cfg.Theme.Vars)
	if err != nil {
		t.Fatalf("theme.New() error = %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "dist")
	_, err = NewBuilder(cfg, renderer).Build(context.Background(), tree, outDir)
	if err == nil {
		t.Fatal("Build() error = nil, want relative link self-check failure")
	}
	if !strings.Contains(err.Error(), "non-relative links") {
		t.Fatalf("Build() error = %v, want non-relative link failure", err)
	}
}

func TestBuildRefusesExistingOutputWithoutSentinel(t *testing.T) {
	repo := newSiteTestRepo(t)
	tree, err := source.Open(context.Background(), source.Spec{Repo: repo})
	if err != nil {
		t.Fatalf("source.Open() error = %v", err)
	}
	defer tree.Cleanup()

	cfg, _, err := config.Load(tree.Root, "", config.Flags{Repo: repo})
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	renderer, err := theme.New("", "", cfg.Theme.Vars)
	if err != nil {
		t.Fatalf("theme.New() error = %v", err)
	}

	outDir := t.TempDir()
	writeFile(t, outDir, "keep.txt", "do not delete")
	_, err = NewBuilder(cfg, renderer).Build(context.Background(), tree, outDir)
	if err == nil {
		t.Fatal("Build() error = nil, want sentinel refusal")
	}
	if !strings.Contains(err.Error(), "without .repolens-build sentinel") {
		t.Fatalf("Build() error = %v, want sentinel refusal", err)
	}
}

func newSiteTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")

	writeFile(t, repo, "README.md", "# Home\n\n[Guide](docs/guide.md)\n\n[External](https://example.com/)\n\n![Dot](assets/pixel.png)\n")
	writeFile(t, repo, "page.html", "<!doctype html><p>embed</p>\n")
	writeFile(t, repo, "direct.html", "<!doctype html><p>direct</p>\n")
	writeFile(t, repo, "source.html", "<!doctype html><strong>source</strong>\n")
	writeFile(t, repo, "code/main.go", "package main\n\nfunc main() {}\n")
	writeFileBytes(t, repo, "assets/pixel.png", []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	})
	writeFileBytes(t, repo, "blob.bin", []byte{0x00, 0x01, 0x02, 0x03})
	writeFile(t, repo, "docs/README.md", "# Docs\n\nDocs readme.\n")
	writeFile(t, repo, "notes/readme.md", "# Notes\n\nLowercase readme.\n")
	writeFile(t, repo, "docs/guide.md", "# Guide\n")
	writeFile(t, repo, "plain/file.txt", "plain text\n")
	writeFile(t, repo, "skip/hidden.md", "# Hidden\n")
	writeFile(t, repo, ".repolens.yml", `
site:
  title: Test Repo
rules:
  - match: "direct.html"
    html: { view: direct }
  - match: "source.html"
    html: { view: source }
  - match: "skip/**"
    render: false
`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %s: %v", strings.Join(args, " "), stderr.String(), err)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	writeFileBytes(t, root, rel, []byte(content))
}

func writeFileBytes(t *testing.T, root, rel string, content []byte) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", p, err)
	}
}

func assertMirrorEqual(t *testing.T, repo, outDir, rel string) {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read source %s: %v", rel, err)
	}
	dst, err := os.ReadFile(filepath.Join(outDir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read mirror %s: %v", rel, err)
	}
	if !bytes.Equal(src, dst) {
		t.Fatalf("mirror %s does not match source", rel)
	}
}

func assertExists(t *testing.T, root, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		t.Fatalf("expected %s to exist: %v", rel, err)
	}
}

func assertMissing(t *testing.T, root, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be missing, stat err = %v", rel, err)
	}
}

func readOutput(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", rel, err)
	}
	return string(data)
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("output missing %q\n%s", want, got)
	}
}

func assertNotContains(t *testing.T, got, want string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Fatalf("output unexpectedly contains %q\n%s", want, got)
	}
}
