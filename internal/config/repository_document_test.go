package config

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestRepositoryDocumentReadApplyWritePreservesUncontrolledNodes(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, repoConfigName)
	writeFile(t, path, `
# This comment is outside the UI contract.
source:
  repo: git@example.com:trusted/source.git
  ref: main
output:
  dir: trusted-dist
  base_path: /trusted/
ignore: [old/**]
site:
  title: Before
  language: en
  home: docs/start.md
  future_site: preserved
view:
  tree_position: right
  tree_expand_depth: 1
  toc_panel: inline
  search: true
  future_view:
    mode: compact
access:
  noindex: true
  encrypt:
    enabled: true
    paths: [private/**]
render:
  markdown:
    math: true
agent:
  llms_txt: true
  llms_full:
    enabled: true
    max_size: 1MB
  index_json: true
  future_agent: preserved
rules:
  - match: docs/**
    markdown: { math: true }
    future_rule: preserved
theme:
  vars:
    accent: red
  css: old.css
  templates: old-templates/
  future_theme:
    palette: custom
unknown_top_level:
  nested: [one, { two: 2 }]
`)

	doc, err := LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("LoadRepositoryDocument() error = %v", err)
	}
	if doc.Path != path || doc.Revision == "" {
		t.Fatalf("document identity = %#v", doc)
	}
	if got := *doc.Settings.Site.Title; got != "Before" {
		t.Fatalf("site.title = %q, want Before", got)
	}
	if got := *doc.Settings.Ignore; !slices.Equal(got, []string{"old/**"}) {
		t.Fatalf("ignore = %v, want [old/**]", got)
	}
	if got := *doc.Settings.Theme.Vars; !maps.Equal(got, map[string]string{"accent": "red"}) {
		t.Fatalf("theme.vars = %v, want accent red", got)
	}
	if got := *doc.Settings.Theme.CSS; got != "old.css" {
		t.Fatalf("theme.css = %q, want old.css", got)
	}
	if got := *doc.Settings.Theme.Templates; got != "old-templates/" {
		t.Fatalf("theme.templates = %q, want old-templates/", got)
	}
	if got := *doc.Settings.View.TreeExpandDepth; got != 1 {
		t.Fatalf("view.tree_expand_depth = %d, want 1", got)
	}
	if got := *doc.Settings.Agent.LLMSFull.Enabled; !got {
		t.Fatal("agent.llms_full.enabled = false, want true")
	}

	title := "After"
	settings := RepositorySettings{Site: RepositorySiteSettings{Title: &title}}
	if err := doc.Apply(settings); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if doc.Settings.Render.Markdown.Math == nil || !*doc.Settings.Render.Markdown.Math {
		t.Fatalf("render settings after Apply = %#v, want markdown.math true", doc.Settings.Render)
	}
	if doc.Settings.Rules == nil || len(*doc.Settings.Rules) != 1 ||
		(*doc.Settings.Rules)[0].Match == nil || *(*doc.Settings.Rules)[0].Match != "docs/**" {
		t.Fatalf("rules settings after Apply = %#v, want original rule", doc.Settings.Rules)
	}
	if err := doc.Write(); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"future_site: preserved",
		"repo: git@example.com:trusted/source.git",
		"dir: trusted-dist",
		"base_path: /trusted/",
		"noindex: true",
		"paths: [private/**]",
		"math: true",
		"tree_position: right",
		"toc_panel: inline",
		"mode: compact",
		"enabled: true\n    paths:",
		"max_size: 1MB",
		"future_agent: preserved",
		"future_rule: preserved",
		"future_theme:",
		"unknown_top_level:",
		"nested: [one, {two: 2}]",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("written YAML does not preserve %q:\n%s", want, text)
		}
	}
	var got RepositorySettings
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal written settings: %v", err)
	}
	if *got.Site.Title != title || *got.Site.Language != "en" || *got.Site.Home != "docs/start.md" {
		t.Fatalf("site settings = %#v", got.Site)
	}
	if !slices.Equal(*got.Ignore, []string{"old/**"}) {
		t.Fatalf("ignore = %v, want [old/**]", *got.Ignore)
	}
	if !maps.Equal(*got.Theme.Vars, map[string]string{"accent": "red"}) || *got.Theme.CSS != "old.css" || *got.Theme.Templates != "old-templates/" {
		t.Fatalf("theme settings = %#v", got.Theme)
	}
	if got.Render.Markdown.Math == nil || !*got.Render.Markdown.Math {
		t.Fatalf("render settings = %#v, want original markdown.math true", got.Render)
	}
	if got.Rules == nil || len(*got.Rules) != 1 || (*got.Rules)[0].Match == nil ||
		*(*got.Rules)[0].Match != "docs/**" || (*got.Rules)[0].Markdown.Math == nil ||
		!*(*got.Rules)[0].Markdown.Math {
		t.Fatalf("rules settings = %#v, want original docs rule", got.Rules)
	}
	if *got.View.TreePosition != "right" || *got.View.TreeExpandDepth != 1 ||
		*got.View.TOCPanel != "inline" || !*got.View.Search {
		t.Fatalf("view settings = %#v", got.View)
	}
	if !*got.Agent.LLMSTxt ||
		!*got.Agent.LLMSFull.Enabled || *got.Agent.LLMSFull.MaxSize != 1024*1024 || !*got.Agent.IndexJSON {
		t.Fatalf("agent settings = %#v", got.Agent)
	}

	loaded, warnings, err := Load(repoRoot, "", Flags{})
	if err != nil {
		t.Fatalf("Load() after Write() error = %v", err)
	}
	_ = warnings // unknown preserved fields are expected to warn during strict load.
	if loaded.Site.Title != title || loaded.Site.Language != "en" || loaded.Site.Home != "docs/start.md" {
		t.Fatalf("loaded site settings = %#v", loaded.Site)
	}
	if !slices.Equal(loaded.Ignore, []string{"old/**"}) {
		t.Fatalf("loaded ignore = %v, want [old/**]", loaded.Ignore)
	}
	if !maps.Equal(loaded.Theme.Vars, map[string]string{"accent": "red"}) || loaded.Theme.CSS != "old.css" || loaded.Theme.Templates != "old-templates/" {
		t.Fatalf("loaded theme = %#v", loaded.Theme)
	}
	if loaded.View.TreePosition != "right" || loaded.View.TreeExpandDepth != 1 || loaded.View.TOCPanel != "inline" || !loaded.View.Search {
		t.Fatalf("loaded view settings = %#v", loaded.View)
	}
	if !loaded.Agent.LLMSTxt || !loaded.Agent.LLMSFull.Enabled || loaded.Agent.LLMSFull.MaxSize != 1024*1024 || !loaded.Agent.IndexJSON {
		t.Fatalf("loaded agent settings = %#v", loaded.Agent)
	}
}

func TestRepositoryDocumentReplaceClearsControlledFieldsAndPreservesTrustedSections(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, repoConfigName)
	writeFile(t, path, `
source:
  repo: git@example.com:trusted/source.git
output:
  dir: trusted-dist
access:
  noindex: false
site:
  title: Configured title
render:
  markdown:
    math: true
rules:
  - match: docs/**
    markdown:
      math: true
    future_rule: retained
`)

	doc, err := LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("LoadRepositoryDocument() error = %v", err)
	}
	match := "docs/**"
	settings := RepositorySettings{Rules: &[]RepositoryRuleSettings{{Match: &match}}}
	if err := doc.Replace(settings); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	if err := doc.Write(); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read replaced config: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"repo: git@example.com:trusted/source.git",
		"dir: trusted-dist",
		"noindex: false",
		"match: docs/**",
		"future_rule: retained",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("replaced config missing %q:\n%s", want, text)
		}
	}
	for _, removed := range []string{"title:", "math:"} {
		if strings.Contains(text, removed) {
			t.Errorf("replaced config retained controlled field %q:\n%s", removed, text)
		}
	}
	if doc.Settings.Site.Title != nil || (*doc.Settings.Rules)[0].Markdown.Math != nil {
		t.Fatalf("Replace() settings retained cleared values: %#v", doc.Settings)
	}
}

func TestRepositoryDocumentApplyUpdatesOnlyProvidedRenderFieldsAndRules(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, repoConfigName)
	writeFile(t, path, `render:
  render: true
  markdown:
    toc: true
    math: true
rules:
  - match: docs/**
    render: true
    markdown:
      toc: true
      math: true
    future_rule: preserved
`)

	doc, err := LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("LoadRepositoryDocument() error = %v", err)
	}
	falseValue := false
	match := "docs/**"
	rules := []RepositoryRuleSettings{{
		Match:    &match,
		Markdown: RepositoryMarkdownOptionsSettings{Math: &falseValue},
	}}
	if err := doc.Apply(RepositorySettings{
		Render: RepositoryFileOptionsSettings{
			Markdown: RepositoryMarkdownOptionsSettings{Math: &falseValue},
		},
		Rules: &rules,
	}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	data, err := doc.YAML()
	if err != nil {
		t.Fatalf("YAML() error = %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"render: true",
		"toc: true",
		"math: false",
		"future_rule: preserved",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("patched YAML does not preserve or update %q:\n%s", want, text)
		}
	}
}

func TestRepositoryDocumentMoveRulePreservesOrderAndUnknownRuleNodes(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, repoConfigName)
	writeFile(t, path, `rules:
  - match: first/**
    unknown: first
  - match: second/**
    unknown: second
  - match: third/**
    unknown: third
`)

	doc, err := LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("LoadRepositoryDocument() error = %v", err)
	}
	if err := doc.MoveRule(2, 0); err != nil {
		t.Fatalf("MoveRule() error = %v", err)
	}
	data, err := doc.YAML()
	if err != nil {
		t.Fatalf("YAML() error = %v", err)
	}
	text := string(data)
	third := strings.Index(text, "- match: third/**\n    unknown: third")
	first := strings.Index(text, "- match: first/**\n    unknown: first")
	second := strings.Index(text, "- match: second/**\n    unknown: second")
	if third == -1 || first == -1 || second == -1 || !(third < first && first < second) {
		t.Fatalf("moved rules and their unknown fields are not ordered correctly:\n%s", text)
	}
}

func TestRepositoryDocumentApplyUpdatesRenderAndRules(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, repoConfigName)
	writeFile(t, path, `source:
  repo: git@example.com:trusted/source.git
output:
  dir: trusted-dist
access:
  encrypt:
    enabled: true
    paths: [private/**]
render:
  render: true
  max_file_size: 1MB
  markdown:
    toc: true
    toc_min_headings: 4
    anchors: true
    mermaid: true
    math: true
    frontmatter_title: true
    future_markdown: preserved
  html:
    view: embed
  code:
    line_numbers: true
    theme: github
rules:
  - match: docs/**
    markdown:
      math: true
      future_markdown: preserved
    future_rule: preserved
unknown_top_level:
  future: preserved
`)

	doc, err := LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("LoadRepositoryDocument() error = %v", err)
	}

	falseValue := false
	two := 2
	direct := "direct"
	monokai := "monokai"
	maxSize := RepositoryByteSize(2 * 1024 * 1024)
	assets := "assets/**"
	docs := "docs/**"
	trueValue := true
	source := "source"
	dracula := "dracula"
	ruleMaxSize := RepositoryByteSize(4096)
	settings := RepositorySettings{
		Render: RepositoryFileOptionsSettings{
			Render: &falseValue,
			Markdown: RepositoryMarkdownOptionsSettings{
				TOC:              &falseValue,
				TOCMinHeadings:   &two,
				Anchors:          &falseValue,
				Mermaid:          &falseValue,
				Math:             &falseValue,
				FrontmatterTitle: &falseValue,
			},
			HTML:        RepositoryHTMLOptionsSettings{View: &direct},
			Code:        RepositoryCodeOptionsSettings{LineNumbers: &falseValue, Theme: &monokai},
			MaxFileSize: &maxSize,
		},
		Rules: &[]RepositoryRuleSettings{
			{
				Match:       &assets,
				Render:      &trueValue,
				Markdown:    RepositoryMarkdownOptionsSettings{Math: &trueValue},
				HTML:        RepositoryHTMLOptionsSettings{View: &source},
				Code:        RepositoryCodeOptionsSettings{LineNumbers: &falseValue, Theme: &dracula},
				MaxFileSize: &ruleMaxSize,
			},
			{
				Match:    &docs,
				Markdown: RepositoryMarkdownOptionsSettings{Math: &falseValue},
			},
		},
	}
	if err := doc.Apply(settings); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if err := doc.Write(); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"repo: git@example.com:trusted/source.git",
		"dir: trusted-dist",
		"enabled: true\n    paths: [private/**]",
		"unknown_top_level:",
		"future: preserved",
		"future_markdown: preserved",
		"future_rule: preserved",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("written YAML does not preserve %q:\n%s", want, text)
		}
	}
	assetsIndex := strings.Index(text, "match: assets/**")
	docsIndex := strings.Index(text, "match: docs/**")
	if assetsIndex == -1 || docsIndex == -1 || assetsIndex >= docsIndex {
		t.Fatalf("rules are not written in settings order:\n%s", text)
	}
	if strings.Contains(text, "match: assets/**\n    future_rule: preserved") ||
		!strings.Contains(text, "match: docs/**\n    markdown:\n      math: false\n      future_markdown: preserved\n    future_rule: preserved") {
		t.Fatalf("unknown rule nodes were not retained with their matching rule:\n%s", text)
	}

	var got RepositorySettings
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal written settings: %v", err)
	}
	if got.Render.Render == nil || *got.Render.Render ||
		got.Render.Markdown.TOC == nil || *got.Render.Markdown.TOC ||
		got.Render.Markdown.TOCMinHeadings == nil || *got.Render.Markdown.TOCMinHeadings != two ||
		got.Render.Markdown.Anchors == nil || *got.Render.Markdown.Anchors ||
		got.Render.Markdown.Mermaid == nil || *got.Render.Markdown.Mermaid ||
		got.Render.Markdown.Math == nil || *got.Render.Markdown.Math ||
		got.Render.Markdown.FrontmatterTitle == nil || *got.Render.Markdown.FrontmatterTitle ||
		got.Render.HTML.View == nil || *got.Render.HTML.View != direct ||
		got.Render.Code.LineNumbers == nil || *got.Render.Code.LineNumbers ||
		got.Render.Code.Theme == nil || *got.Render.Code.Theme != monokai ||
		got.Render.MaxFileSize == nil || *got.Render.MaxFileSize != maxSize {
		t.Fatalf("render settings = %#v", got.Render)
	}
	if got.Rules == nil || len(*got.Rules) != 2 ||
		(*got.Rules)[0].Match == nil || *(*got.Rules)[0].Match != assets ||
		(*got.Rules)[0].Render == nil || !*(*got.Rules)[0].Render ||
		(*got.Rules)[0].Markdown.Math == nil || !*(*got.Rules)[0].Markdown.Math ||
		(*got.Rules)[0].HTML.View == nil || *(*got.Rules)[0].HTML.View != source ||
		(*got.Rules)[0].Code.LineNumbers == nil || *(*got.Rules)[0].Code.LineNumbers ||
		(*got.Rules)[0].Code.Theme == nil || *(*got.Rules)[0].Code.Theme != dracula ||
		(*got.Rules)[0].MaxFileSize == nil || *(*got.Rules)[0].MaxFileSize != ruleMaxSize ||
		(*got.Rules)[1].Match == nil || *(*got.Rules)[1].Match != docs ||
		(*got.Rules)[1].Markdown.Math == nil || *(*got.Rules)[1].Markdown.Math {
		t.Fatalf("rules settings = %#v", got.Rules)
	}
}

func TestRepositoryDocumentApplyUpdatesRenderAndRulesInOrder(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, repoConfigName)
	writeFile(t, path, `source: { repo: trusted/repo }
output: { dir: trusted-dist }
access: { noindex: true }
unknown_top_level: preserved
render:
  markdown: { math: true }
rules:
  - match: docs/**
    markdown: { math: true }
    unknown_rule: docs
  - match: legacy/**
    render: false
    unknown_rule: legacy
`)

	doc, err := LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("LoadRepositoryDocument() error = %v", err)
	}
	render := false
	toc := false
	tocMinHeadings := 2
	anchors := false
	mermaid := false
	math := false
	frontmatterTitle := false
	htmlView := "direct"
	lineNumbers := false
	codeTheme := "monokai"
	maxFileSize := RepositoryByteSize(2 * 1024 * 1024)
	legacyMatch := "legacy/**"
	docsMatch := "docs/**"
	legacyRender := true
	docsMath := false
	docsTheme := "dracula"
	rules := []RepositoryRuleSettings{
		{
			Match:  &legacyMatch,
			Render: &legacyRender,
		},
		{
			Match:    &docsMatch,
			Markdown: RepositoryMarkdownOptionsSettings{Math: &docsMath},
			Code:     RepositoryCodeOptionsSettings{Theme: &docsTheme},
		},
	}
	if err := doc.Apply(RepositorySettings{
		Render: RepositoryFileOptionsSettings{
			Render: &render,
			Markdown: RepositoryMarkdownOptionsSettings{
				TOC:              &toc,
				TOCMinHeadings:   &tocMinHeadings,
				Anchors:          &anchors,
				Mermaid:          &mermaid,
				Math:             &math,
				FrontmatterTitle: &frontmatterTitle,
			},
			HTML:        RepositoryHTMLOptionsSettings{View: &htmlView},
			Code:        RepositoryCodeOptionsSettings{LineNumbers: &lineNumbers, Theme: &codeTheme},
			MaxFileSize: &maxFileSize,
		},
		Rules: &rules,
	}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	data, err := doc.YAML()
	if err != nil {
		t.Fatalf("YAML() error = %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"source: {repo: trusted/repo}",
		"output: {dir: trusted-dist}",
		"access: {noindex: true}",
		"unknown_top_level: preserved",
		"max_file_size: 2097152",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("written YAML does not preserve %q:\n%s", want, text)
		}
	}
	legacyIndex := strings.Index(text, "match: legacy/**")
	docsIndex := strings.Index(text, "match: docs/**")
	if legacyIndex == -1 || docsIndex == -1 || legacyIndex >= docsIndex {
		t.Fatalf("rules are not written in settings order:\n%s", text)
	}
	if strings.Index(text, "unknown_rule: legacy") < legacyIndex ||
		strings.Index(text, "unknown_rule: docs") < docsIndex {
		t.Fatalf("unknown rule fields did not stay with their rules:\n%s", text)
	}

	var got RepositorySettings
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal written settings: %v", err)
	}
	if got.Render.Render == nil || *got.Render.Render != render ||
		got.Render.Markdown.TOC == nil || *got.Render.Markdown.TOC != toc ||
		got.Render.Markdown.TOCMinHeadings == nil || *got.Render.Markdown.TOCMinHeadings != tocMinHeadings ||
		got.Render.Markdown.Anchors == nil || *got.Render.Markdown.Anchors != anchors ||
		got.Render.Markdown.Mermaid == nil || *got.Render.Markdown.Mermaid != mermaid ||
		got.Render.Markdown.Math == nil || *got.Render.Markdown.Math != math ||
		got.Render.Markdown.FrontmatterTitle == nil || *got.Render.Markdown.FrontmatterTitle != frontmatterTitle ||
		got.Render.HTML.View == nil || *got.Render.HTML.View != htmlView ||
		got.Render.Code.LineNumbers == nil || *got.Render.Code.LineNumbers != lineNumbers ||
		got.Render.Code.Theme == nil || *got.Render.Code.Theme != codeTheme ||
		got.Render.MaxFileSize == nil || *got.Render.MaxFileSize != maxFileSize {
		t.Fatalf("render settings = %#v", got.Render)
	}
	if got.Rules == nil || len(*got.Rules) != 2 || (*got.Rules)[0].Match == nil ||
		*(*got.Rules)[0].Match != legacyMatch || (*got.Rules)[1].Match == nil ||
		*(*got.Rules)[1].Match != docsMatch || (*got.Rules)[0].Render == nil ||
		*(*got.Rules)[0].Render != legacyRender || (*got.Rules)[1].Markdown.Math == nil ||
		*(*got.Rules)[1].Markdown.Math != docsMath || (*got.Rules)[1].Code.Theme == nil ||
		*(*got.Rules)[1].Code.Theme != docsTheme {
		t.Fatalf("rules settings = %#v", got.Rules)
	}
}

func TestRepositoryDocumentAddAndRemoveRule(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, repoConfigName), `rules:
  - match: first/**
    unknown: preserved
`)
	doc, err := LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("LoadRepositoryDocument() error = %v", err)
	}
	second := "second/**"
	if err := doc.AddRule(RepositoryRuleSettings{Match: &second}); err != nil {
		t.Fatalf("AddRule() error = %v", err)
	}
	if doc.Settings.Rules == nil || len(*doc.Settings.Rules) != 2 || *(*doc.Settings.Rules)[1].Match != second {
		t.Fatalf("rules after AddRule() = %#v", doc.Settings.Rules)
	}
	if err := doc.RemoveRule(0); err != nil {
		t.Fatalf("RemoveRule() error = %v", err)
	}
	if doc.Settings.Rules == nil || len(*doc.Settings.Rules) != 1 || *(*doc.Settings.Rules)[0].Match != second {
		t.Fatalf("rules after RemoveRule() = %#v", doc.Settings.Rules)
	}
	before, err := doc.YAML()
	if err != nil {
		t.Fatalf("YAML() error = %v", err)
	}
	if err := doc.RemoveRule(1); err == nil {
		t.Fatal("RemoveRule() error = nil, want out-of-range error")
	}
	after, err := doc.YAML()
	if err != nil {
		t.Fatalf("YAML() after out-of-range removal error = %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("out-of-range removal mutated YAML:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestRepositorySettingsValidation(t *testing.T) {
	t.Parallel()

	invalidPattern := "["
	invalidView := "popup"
	invalidPosition := "top"
	invalidTOCPanel := "drawer"
	negativeSize := RepositoryByteSize(-1)
	negativeDepth := -1
	settings := RepositorySettings{
		Ignore: &[]string{invalidPattern},
		Render: RepositoryFileOptionsSettings{
			HTML:        RepositoryHTMLOptionsSettings{View: &invalidView},
			MaxFileSize: &negativeSize,
		},
		Rules: &[]RepositoryRuleSettings{{}},
		View: RepositoryViewSettings{
			TreePosition:    &invalidPosition,
			TreeExpandDepth: &negativeDepth,
			TOCPanel:        &invalidTOCPanel,
		},
		Agent: RepositoryAgentSettings{
			LLMSFull: RepositoryAgentFullTextSettings{MaxSize: &negativeSize},
		},
	}
	issues := ValidateRepositorySettings(settings)
	for _, path := range []string{
		"ignore[0]",
		"render.html.view",
		"render.max_file_size",
		"rules[0].match",
		"view.tree_position",
		"view.tree_expand_depth",
		"view.toc_panel",
		"agent.llms_full.max_size",
	} {
		if !hasRepositoryValidationIssue(issues, path) {
			t.Errorf("issues = %#v, missing %q", issues, path)
		}
	}

	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, repoConfigName)
	writeFile(t, path, "site:\n  title: Before")
	doc, err := LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("LoadRepositoryDocument() error = %v", err)
	}
	if err := doc.Apply(settings); err == nil {
		t.Fatal("Apply() error = nil, want RepositoryValidationError")
	} else {
		var validationErr *RepositoryValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("Apply() error = %T, want RepositoryValidationError", err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after failed Apply(): %v", err)
	}
	if got := string(data); got != "site:\n  title: Before\n" {
		t.Fatalf("failed Apply() mutated config on disk: %q", got)
	}
}

func hasRepositoryValidationIssue(issues []RepositoryValidationIssue, path string) bool {
	for _, issue := range issues {
		if issue.Path == path {
			return true
		}
	}
	return false
}

func TestRepositoryDocumentEmptyFileCanBeUpdatedAndWritten(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, repoConfigName)
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("create empty config: %v", err)
	}
	doc, err := LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("LoadRepositoryDocument() error = %v", err)
	}
	if doc.Revision != revisionFor(nil) {
		t.Fatalf("empty file revision = %q, want empty-content revision", doc.Revision)
	}
	title := "Repository"
	if err := doc.Apply(RepositorySettings{Site: RepositorySiteSettings{Title: &title}}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if err := doc.Write(); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written config: %v", err)
	}
	var settings RepositorySettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		t.Fatalf("unmarshal written settings: %v", err)
	}
	if settings.Site.Title == nil || *settings.Site.Title != title {
		t.Fatalf("empty document site.title = %#v, want %q; YAML = %q", settings.Site.Title, title, data)
	}
	if doc.Revision != revisionFor(data) {
		t.Fatalf("written revision = %q, want written-content revision", doc.Revision)
	}
}

func TestRepositoryDocumentWriteRejectsRevisionConflict(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, repoConfigName)
	writeFile(t, path, "site:\n  title: Before")
	doc, err := LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("LoadRepositoryDocument() error = %v", err)
	}
	title := "After"
	if err := doc.Apply(RepositorySettings{Site: RepositorySiteSettings{Title: &title}}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	writeFile(t, path, "site:\n  title: External")

	err = doc.Write()
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("Write() error = %v, want ErrRevisionConflict", err)
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read config after conflict: %v", readErr)
	}
	if got := string(data); got != "site:\n  title: External\n" {
		t.Fatalf("config after conflict = %q", got)
	}
}
