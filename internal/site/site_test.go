package site

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
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
	if stats.Files != 12 || stats.Pages != 24 {
		t.Fatalf("stats = %#v, want 12 files and 24 pages", stats)
	}

	assertExists(t, outDir, ".repolens-build")
	assertExists(t, outDir, "index.html")
	assertExists(t, outDir, "robots.txt")
	assertExists(t, outDir, "_assets/site.css")
	assertExists(t, outDir, "_assets/site.js")
	assertExists(t, outDir, "llms.txt")
	assertExists(t, outDir, "llms-full.txt")
	assertExists(t, outDir, "index.json")
	assertExists(t, outDir, "search.json")
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
		"view/README.md/source/index.html",
		"view/page.html/index.html",
		"view/page.html/source/index.html",
		"view/direct.html/index.html",
		"view/direct.html/source/index.html",
		"view/source.html/index.html",
		"view/code/main.go/index.html",
		"view/assets/pixel.png/index.html",
		"view/blob.bin/index.html",
		"view/docs/index.html",
		"view/docs/guide.md/index.html",
		"view/docs/guide.md/source/index.html",
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

	readmePage := readOutput(t, outDir, "view/README.md/index.html")
	assertContains(t, readmePage, `<link rel="alternate" type="text/markdown" href="../../README.md">`)
	assertContains(t, readmePage, `id="btn-source" href="source/"`)
	assertContains(t, readmePage, `<span>大小</span><b>91 B</b>`)
	assertContains(t, readmePage, `<code data-page-path>README.md</code>`)
	assertContains(t, readmePage, `<small>initial</small>`)
	assertNotContains(t, readmePage, `footer class="meta"`)

	readmeSourcePage := readOutput(t, outDir, "view/README.md/source/index.html")
	assertContains(t, readmeSourcePage, `class="page page-markdown-source"`)
	assertContains(t, readmeSourcePage, `id="btn-source" href="../"`)
	assertContains(t, readmeSourcePage, `Guide`)
	assertContains(t, readmeSourcePage, `href="../../../README.md"`)

	docsPage := readOutput(t, outDir, "view/docs/index.html")
	assertContains(t, docsPage, "Docs")
	assertContains(t, docsPage, "guide.md")
	// 非祖先目录 code 深度 1 ≤ tree_expand_depth(默认 2)，树中应默认展开。
	assertContains(t, docsPage, `<details data-tree-path="code" open>`)

	// 小写 readme.md 回退。
	notesPage := readOutput(t, outDir, "view/notes/index.html")
	assertContains(t, notesPage, `<section class="readme">`)
	assertContains(t, notesPage, "Lowercase readme.")

	codeDirPage := readOutput(t, outDir, "view/code/index.html")
	assertContains(t, codeDirPage, `<section class="dir-list"`)
	if strings.Contains(codeDirPage, `<section class="readme">`) {
		t.Fatalf("code directory unexpectedly rendered a README section")
	}

	embedPage := readOutput(t, outDir, "view/page.html/index.html")
	assertContains(t, embedPage, `<iframe class="html-preview" src="../../page.html"`)

	directPage := readOutput(t, outDir, "view/direct.html/index.html")
	assertContains(t, directPage, "Open HTML file")
	assertContains(t, directPage, `id="btn-source" href="source/"`)
	assertNotContains(t, directPage, "<iframe")

	directSourcePage := readOutput(t, outDir, "view/direct.html/source/index.html")
	assertContains(t, directSourcePage, `class="page page-html-source"`)
	assertContains(t, directSourcePage, `id="btn-source" href="../"`)

	sourcePage := readOutput(t, outDir, "view/source.html/index.html")
	assertContains(t, sourcePage, "&lt;")
	assertContains(t, sourcePage, "strong")
	assertContains(t, sourcePage, "source")
	assertNotContains(t, sourcePage, "<iframe")
	assertNotContains(t, sourcePage, `id="btn-source"`)
	assertMissing(t, outDir, "view/source.html/source/index.html")

	imagePage := readOutput(t, outDir, "view/assets/pixel.png/index.html")
	assertContains(t, imagePage, `<img class="preview" src="../../../assets/pixel.png"`)
	assertNotContains(t, imagePage, `id="btn-source"`)

	binaryPage := readOutput(t, outDir, "view/blob.bin/index.html")
	assertContains(t, binaryPage, "Download")
	assertContains(t, binaryPage, "blob.bin")
	assertNotContains(t, binaryPage, `id="btn-source"`)

	skipDirPage := readOutput(t, outDir, "view/skip/index.html")
	assertContains(t, skipDirPage, `href="../../skip/hidden.md"`)

	robots := readOutput(t, outDir, "robots.txt")
	assertContains(t, robots, "Disallow: /")
}

func TestAgentOutputs(t *testing.T) {
	repo := newAgentTestRepo(t)
	outDir, _, err := buildSite(t, repo)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	llms := readOutput(t, outDir, "llms.txt")
	assertContains(t, llms, "# Agent Repo")
	assertContains(t, llms, "> Browseable static rendering")
	assertContains(t, llms, "Raw files are available at `/<repo-path>`.")
	assertContains(t, llms, "Browser pages are available at `/view/<repo-path>/`.")
	assertContains(t, llms, "## /")
	assertContains(t, llms, "## docs/")
	assertContains(t, llms, "- [Home Title](README.md): This summary comes after front matter and after the heading.")
	assertContains(t, llms, "- [Long](docs/long.md): 0123456789 0123456789")
	assertContains(t, llms, "[index.json](index.json)")
	assertContains(t, llms, "[llms-full.txt](llms-full.txt)")
	// front-matter 的键值不得泄漏进摘要（fixture 的 README 有 title: Home Title）。
	assertNotContains(t, llms, "title: Home Title")
	for _, line := range strings.Split(llms, "\n") {
		if !strings.Contains(line, "[Long]") {
			continue
		}
		summary := strings.TrimPrefix(line[strings.Index(line, "): ")+3:], "")
		if countRunes(summary) > 120 {
			t.Fatalf("summary length = %d, want <= 120: %q", countRunes(summary), summary)
		}
		if !strings.HasSuffix(summary, "...") {
			t.Fatalf("truncated summary = %q, want ellipsis", summary)
		}
	}

	full := readOutput(t, outDir, "llms-full.txt")
	assertContains(t, full, "----- README.md -----")
	assertContains(t, full, "----- docs/long.md -----")
	assertContains(t, full, "----- notes.txt -----")
	assertNotContains(t, full, "package main")

	rawIndex := []byte(readOutput(t, outDir, "index.json"))
	if !json.Valid(rawIndex) {
		t.Fatalf("index.json is not valid JSON:\n%s", rawIndex)
	}
	var index struct {
		Generator string `json:"generator"`
		Commit    string `json:"commit"`
		BuiltAt   string `json:"built_at"`
		Site      struct {
			Title string `json:"title"`
		} `json:"site"`
		Files []struct {
			Path     string  `json:"path"`
			Kind     string  `json:"kind"`
			Size     int64   `json:"size"`
			Title    *string `json:"title"`
			Modified *string `json:"modified"`
			Raw      string  `json:"raw"`
			View     *string `json:"view"`
		} `json:"files"`
	}
	if err := json.Unmarshal(rawIndex, &index); err != nil {
		t.Fatalf("Unmarshal(index.json): %v", err)
	}
	if index.Generator != "repolens dev" || index.Commit == "" || index.BuiltAt == "" || index.Site.Title != "Agent Repo" {
		t.Fatalf("index metadata = %#v", index)
	}
	gotPaths := make([]string, 0, len(index.Files))
	for _, file := range index.Files {
		gotPaths = append(gotPaths, file.Path)
		if strings.HasPrefix(file.Path, "/") || strings.HasPrefix(file.Raw, "/") {
			t.Fatalf("index paths must be relative: %#v", file)
		}
		assertExists(t, outDir, file.Raw)
		if file.View != nil {
			assertExists(t, outDir, *file.View+"index.html")
		}
		if file.Path == "README.md" {
			if file.Kind != "markdown" || file.Title == nil || *file.Title != "Home Title" || file.Modified == nil || file.Raw != "README.md" {
				t.Fatalf("README index entry = %#v", file)
			}
		}
		if file.Path == "image.bin" && (file.Kind != "binary" || file.Title != nil) {
			t.Fatalf("binary index entry = %#v", file)
		}
	}
	wantPaths := []string{"README.md", "code/main.go", "docs/long.md", "image.bin", "notes.txt"}
	if !slices.Equal(gotPaths, wantPaths) {
		t.Fatalf("index paths = %v, want %v", gotPaths, wantPaths)
	}
}

func TestLLMSFullTruncates(t *testing.T) {
	repo := newAgentTestRepo(t)
	outDir, _, err := buildSiteWithConfig(t, repo, func(cfg *config.Config) {
		cfg.Agent.LLMSFull.MaxSize = 80
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	full := readOutput(t, outDir, "llms-full.txt")
	assertContains(t, full, "----- README.md -----")
	assertContains(t, full, "[truncated]")
}

func TestAgentOutputSwitches(t *testing.T) {
	tests := []struct {
		name    string
		disable func(*config.Config)
		missing string
		present []string
	}{
		{
			name:    "llms txt",
			disable: func(cfg *config.Config) { cfg.Agent.LLMSTxt = false },
			missing: "llms.txt",
			present: []string{"llms-full.txt", "index.json"},
		},
		{
			name:    "llms full",
			disable: func(cfg *config.Config) { cfg.Agent.LLMSFull.Enabled = false },
			missing: "llms-full.txt",
			present: []string{"llms.txt", "index.json"},
		},
		{
			name:    "index json",
			disable: func(cfg *config.Config) { cfg.Agent.IndexJSON = false },
			missing: "index.json",
			present: []string{"llms.txt", "llms-full.txt"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newAgentTestRepo(t)
			outDir, _, err := buildSiteWithConfig(t, repo, tt.disable)
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			assertMissing(t, outDir, tt.missing)
			for _, rel := range tt.present {
				assertExists(t, outDir, rel)
			}
		})
	}
}

func TestSearchJSONIndexesBrowseableFilesAndAllMarkdownHeadings(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	writeFile(t, repo, "README.md", "# 首页\n\n## 第一节\n\n## 第二节\n")
	writeFile(t, repo, "docs/中文指南.md", "# 使用指南\n\n## 安装\n\n## 配置\n\n## 部署\n")
	writeFile(t, repo, "plain.txt", "plain text\n")
	writeFile(t, repo, ".repolens.yml", `
render:
  markdown:
    toc_min_headings: 99
`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")

	outDir, _, err := buildSite(t, repo)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	raw := []byte(readOutput(t, outDir, "search.json"))
	if !json.Valid(raw) {
		t.Fatalf("search.json is not valid JSON:\n%s", raw)
	}
	var index struct {
		Docs []struct {
			Path     string `json:"path"`
			Title    string `json:"title"`
			Kind     string `json:"kind"`
			View     string `json:"view"`
			Headings []struct {
				Text   string `json:"text"`
				Anchor string `json:"anchor"`
				Level  int    `json:"level"`
			} `json:"headings"`
		} `json:"docs"`
	}
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatalf("Unmarshal(search.json): %v", err)
	}
	if len(index.Docs) != 3 {
		t.Fatalf("search docs length = %d, want 3: %#v", len(index.Docs), index.Docs)
	}
	var guide *struct {
		Path     string `json:"path"`
		Title    string `json:"title"`
		Kind     string `json:"kind"`
		View     string `json:"view"`
		Headings []struct {
			Text   string `json:"text"`
			Anchor string `json:"anchor"`
			Level  int    `json:"level"`
		} `json:"headings"`
	}
	for i := range index.Docs {
		if index.Docs[i].Path == "docs/中文指南.md" {
			guide = &index.Docs[i]
		}
		if strings.HasPrefix(index.Docs[i].View, "/") {
			t.Fatalf("search view path must be relative: %#v", index.Docs[i])
		}
	}
	if guide == nil {
		t.Fatalf("Chinese path missing from search index: %#v", index.Docs)
	}
	if guide.Title != "使用指南" || guide.Kind != "markdown" || guide.View != "view/docs/中文指南.md/" {
		t.Fatalf("guide search entry = %#v", guide)
	}
	gotHeadings := make([]string, 0, len(guide.Headings))
	for _, heading := range guide.Headings {
		gotHeadings = append(gotHeadings, heading.Text)
		if heading.Anchor == "" || heading.Level <= 0 {
			t.Fatalf("heading missing anchor or level: %#v", heading)
		}
	}
	if !slices.Equal(gotHeadings, []string{"使用指南", "安装", "配置", "部署"}) {
		t.Fatalf("guide headings = %v", gotHeadings)
	}
	guidePage := readOutput(t, outDir, "view/docs/中文指南.md/index.html")
	assertNotContains(t, guidePage, `<aside class="toc-panel"`)
}

func TestSearchSwitchAndAgentIndexIndependence(t *testing.T) {
	repo := newAgentTestRepo(t)
	outDir, _, err := buildSiteWithConfig(t, repo, func(cfg *config.Config) {
		cfg.Agent.IndexJSON = false
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	assertMissing(t, outDir, "index.json")
	assertExists(t, outDir, "search.json")

	outDir, _, err = buildSiteWithConfig(t, repo, func(cfg *config.Config) {
		cfg.View.Search = false
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	assertMissing(t, outDir, "search.json")
	page := readOutput(t, outDir, "view/README.md/index.html")
	assertNotContains(t, page, `id="btn-search"`)
	assertNotContains(t, page, `data-tree-search-placeholder`)
	assertNotContains(t, page, `id="search-modal"`)
}

func TestAgentEncryptOverlapWarningInStats(t *testing.T) {
	repo := newAgentTestRepo(t)
	_, stats, err := buildSiteWithConfig(t, repo, func(cfg *config.Config) {
		cfg.Access.Encrypt.Enabled = true
		cfg.Access.Encrypt.Paths = []string{"llms.txt"}
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(stats.Warnings) != 1 || !strings.Contains(stats.Warnings[0].Msg, "overlap") {
		t.Fatalf("stats warnings = %#v, want overlap warning", stats.Warnings)
	}
}

func TestBuildSiteHomeOverridesRootPage(t *testing.T) {
	repo := newSiteTestRepo(t)
	outDir, _, err := buildSiteWithConfig(t, repo, func(cfg *config.Config) {
		cfg.Site.Home = "docs/guide.md"
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	rootPage := readOutput(t, outDir, "view/index.html")
	assertContains(t, rootPage, "Guide")
	// README 专属内容不应再出现在根页正文。
	assertNotContains(t, rootPage, "External")
	// site.language 注入 <html lang>（默认 zh-CN）。
	assertContains(t, rootPage, `<html class="no-js" lang="zh-CN">`)
}

func TestTOCPanelModeAffectsMarkdownPages(t *testing.T) {
	repo := newSiteTestRepo(t)
	outDir, _, err := buildSiteWithConfig(t, repo, func(cfg *config.Config) {
		cfg.Render.Markdown.TOCMinHeadings = 1
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	floating := readOutput(t, outDir, "view/README.md/index.html")
	assertContains(t, floating, `<aside class="toc-panel" id="toc-panel"`)
	assertContains(t, floating, `id="btn-toc"`)
	assertNotContains(t, floating, `<nav class="toc"`)

	outDir, _, err = buildSiteWithConfig(t, repo, func(cfg *config.Config) {
		cfg.Render.Markdown.TOCMinHeadings = 1
		cfg.View.TOCPanel = "inline"
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	inline := readOutput(t, outDir, "view/README.md/index.html")
	assertContains(t, inline, `<nav class="toc" aria-label="Table of contents">`)
	assertNotContains(t, inline, `<aside class="toc-panel"`)
	assertNotContains(t, inline, `id="btn-toc"`)
}

func TestUILanguageUsesEnglishForNonChinese(t *testing.T) {
	repo := newSiteTestRepo(t)
	outDir, _, err := buildSiteWithConfig(t, repo, func(cfg *config.Config) {
		cfg.Site.Language = "en"
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	page := readOutput(t, outDir, "view/README.md/index.html")
	for _, want := range []string{
		`lang="en"`,
		`title="Repository tree"`,
		`title="Page information"`,
		`<span>Last updated</span>`,
		`View raw file`,
		`Raw file README.md`,
		`Search (/)`,
	} {
		assertContains(t, page, want)
	}
}

func TestAgentEncryptWholeSiteWarningInStats(t *testing.T) {
	repo := newAgentTestRepo(t)
	_, stats, err := buildSiteWithConfig(t, repo, func(cfg *config.Config) {
		cfg.Access.Encrypt.Enabled = true
		// Paths 留空 = 全站加密，与 agent 产物必然冲突。
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(stats.Warnings) != 1 || !strings.Contains(stats.Warnings[0].Msg, "whole site") {
		t.Fatalf("stats warnings = %#v, want whole-site warning", stats.Warnings)
	}
}

func TestBuildMergesIndexHTMLIntoDirectoryPages(t *testing.T) {
	repo := newIndexHTMLConflictRepo(t)
	outDir, stats, err := buildSite(t, repo)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if stats.Files != 9 || stats.Pages != 14 {
		t.Fatalf("stats = %#v, want 9 files and 14 pages", stats)
	}

	for _, p := range []string{
		"wireframes/index.html",
		"direct/index.html",
		"source/index.html",
		"combo/index.html",
		"skip/index.html",
	} {
		assertMirrorEqual(t, repo, outDir, p)
	}

	wireframesPage := readOutput(t, outDir, "view/wireframes/index.html")
	// 合并页标题用目录名，而非文件名 "index.html"。
	assertContains(t, wireframesPage, "<h1>wireframes</h1>")
	assertContains(t, wireframesPage, `<iframe class="html-preview" src="../../wireframes/index.html"`)
	assertContains(t, wireframesPage, "child.html")
	assertContains(t, wireframesPage, `<a class="dir-entry-name" href=".">`)
	assertUnavailable(t, outDir, "view/wireframes/index.html/index.html")

	directPage := readOutput(t, outDir, "view/direct/index.html")
	assertContains(t, directPage, "Open HTML file")
	assertContains(t, directPage, `href="../../direct/index.html"`)
	assertNotContains(t, directPage, "<iframe")
	assertUnavailable(t, outDir, "view/direct/index.html/index.html")

	sourcePage := readOutput(t, outDir, "view/source/index.html")
	assertContains(t, sourcePage, "&lt;")
	assertContains(t, sourcePage, "strong")
	assertContains(t, sourcePage, "source index")
	assertNotContains(t, sourcePage, "<iframe")
	assertUnavailable(t, outDir, "view/source/index.html/index.html")

	comboPage := readOutput(t, outDir, "view/combo/index.html")
	assertContains(t, comboPage, `<iframe class="html-preview" src="../../combo/index.html"`)
	assertNotContains(t, comboPage, "Combo Readme")
	comboReadmePage := readOutput(t, outDir, "view/combo/README.md/index.html")
	assertContains(t, comboReadmePage, "Combo Readme")

	skipPage := readOutput(t, outDir, "view/skip/index.html")
	assertContains(t, skipPage, "Skip Readme")
	assertContains(t, skipPage, `href="../../skip/index.html"`)
	assertNotContains(t, skipPage, "<iframe")
	assertUnavailable(t, outDir, "view/skip/index.html/index.html")
}

func TestBuildFailsWhenRepositoryDirectoryIsNamedIndexHTML(t *testing.T) {
	repo := newIndexHTMLDirectoryRepo(t)
	_, _, err := buildSite(t, repo)
	if err == nil {
		t.Fatal("Build() error = nil, want index.html directory conflict")
	}
	msg := err.Error()
	for _, want := range []string{
		"wireframes/index.html",
		"view/wireframes/index.html",
		"view/wireframes/index.html/index.html",
		"collides",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Build() error = %q, want substring %q", msg, want)
		}
	}
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

func TestGeneratedLinkNeedlesOnlyScanCSSURLsInCSS(t *testing.T) {
	htmlNeedles := strings.Join(generatedLinkNeedles("view/code/main.go/index.html"), "\n")
	if strings.Contains(htmlNeedles, "url(http") {
		t.Fatalf("HTML needles = %q, should not scan literal code text as CSS url()", htmlNeedles)
	}
	cssNeedles := strings.Join(generatedLinkNeedles("_assets/site.css"), "\n")
	if !strings.Contains(cssNeedles, "url(http://") || !strings.Contains(cssNeedles, "@import") {
		t.Fatalf("CSS needles = %q, want CSS resource checks", cssNeedles)
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

func newIndexHTMLConflictRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")

	writeFile(t, repo, "README.md", "# Home\n")
	writeFile(t, repo, "wireframes/index.html", "<!doctype html><p>wireframes index</p>\n")
	writeFile(t, repo, "wireframes/child.html", "<!doctype html><p>child</p>\n")
	writeFile(t, repo, "direct/index.html", "<!doctype html><p>direct index</p>\n")
	writeFile(t, repo, "source/index.html", "<!doctype html><strong>source index</strong>\n")
	writeFile(t, repo, "combo/README.md", "# Combo Readme\n")
	writeFile(t, repo, "combo/index.html", "<!doctype html><p>combo index</p>\n")
	writeFile(t, repo, "skip/README.md", "# Skip Readme\n")
	writeFile(t, repo, "skip/index.html", "<!doctype html><p>skip index</p>\n")
	writeFile(t, repo, ".repolens.yml", `
rules:
  - match: "direct/index.html"
    html: { view: direct }
  - match: "source/index.html"
    html: { view: source }
  - match: "skip/index.html"
    render: false
`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

func newAgentTestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")

	writeFile(t, repo, "README.md", `---
title: Home Title
---

# Home Heading

This summary comes after front matter and after the heading.
`)
	writeFile(t, repo, "docs/long.md", "# Long\n\n"+strings.Repeat("0123456789 ", 20)+"\n")
	writeFile(t, repo, "notes.txt", "plain note\n")
	writeFile(t, repo, "code/main.go", "package main\n\nfunc main() {}\n")
	writeFileBytes(t, repo, "image.bin", []byte{0x00, 0x01, 0x02})
	writeFile(t, repo, ".repolens.yml", `
site:
  title: Agent Repo
`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

func TestBuildRootIndexHTMLKeepsMirror(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	writeFile(t, repo, "index.html", "<!doctype html><p>root index</p>\n")
	writeFile(t, repo, "docs/guide.md", "# Guide\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")

	outDir, _, err := buildSite(t, repo)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// 根 index.html 的镜像不被 meta-refresh 跳转页覆盖。
	assertMirrorEqual(t, repo, outDir, "index.html")
	rootIndex := readOutput(t, outDir, "index.html")
	assertNotContains(t, rootIndex, "meta http-equiv")

	// 根目录页合并其浏览形态（默认 embed → iframe），且无独立文件页。
	rootView := readOutput(t, outDir, "view/index.html")
	assertContains(t, rootView, `<iframe class="html-preview" src="../index.html"`)
	assertUnavailable(t, outDir, "view/index.html/index.html")
}

func TestBuildGenerates404Page(t *testing.T) {
	repo := newAgentTestRepo(t)

	// 默认（zh-CN）：站点根生成自包含 404 页，noindex、无站内资源引用。
	outDir, _, err := buildSite(t, repo)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	page := readOutput(t, outDir, "404.html")
	assertContains(t, page, "页面不存在")
	assertContains(t, page, `<meta name="robots" content="noindex">`)
	assertNotContains(t, page, `href=`)
	assertNotContains(t, page, `src=`)

	// site.language: en 时输出英文文案。
	outDir, _, err = buildSiteWithConfig(t, repo, func(cfg *config.Config) {
		cfg.Site.Language = "en"
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	page = readOutput(t, outDir, "404.html")
	assertContains(t, page, "Page not found")
	assertContains(t, page, ` lang="en"`)
}

func TestBuildRoot404HTMLKeepsMirror(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	writeFile(t, repo, "404.html", "<!doctype html><p>custom not found</p>\n")
	writeFile(t, repo, "README.md", "# Home\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")

	outDir, _, err := buildSite(t, repo)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	// 仓库自带根 404.html 时不覆盖镜像。
	assertMirrorEqual(t, repo, outDir, "404.html")
}

func newIndexHTMLDirectoryRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")

	writeFile(t, repo, "README.md", "# Home\n")
	writeFile(t, repo, "wireframes/index.html/page.html", "<!doctype html><p>bad dir</p>\n")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

func buildSite(t *testing.T, repo string) (string, Stats, error) {
	t.Helper()
	return buildSiteWithConfig(t, repo, nil)
}

func buildSiteWithConfig(t *testing.T, repo string, mutate func(*config.Config)) (string, Stats, error) {
	t.Helper()
	tree, err := source.Open(context.Background(), source.Spec{Repo: repo})
	if err != nil {
		t.Fatalf("source.Open() error = %v", err)
	}
	defer tree.Cleanup()

	cfg, _, err := config.Load(tree.Root, "", config.Flags{Repo: repo})
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if mutate != nil {
		mutate(cfg)
	}
	renderer, err := theme.New("", "", cfg.Theme.Vars)
	if err != nil {
		t.Fatalf("theme.New() error = %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "dist")
	stats, err := NewBuilder(cfg, renderer).Build(context.Background(), tree, outDir)
	return outDir, stats, err
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

func assertUnavailable(t *testing.T, root, rel string) {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	if err == nil {
		t.Fatalf("expected %s to be unavailable", rel)
	}
	// ENOENT 或路径前缀是普通文件（ENOTDIR）都算不可用。
	if !os.IsNotExist(err) && !errors.Is(err, syscall.ENOTDIR) {
		t.Fatalf("stat %s: unexpected error %v", rel, err)
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

func countRunes(s string) int {
	count := 0
	for range s {
		count++
	}
	return count
}
