package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		repoYAML string
		extYAML  string
		flags    Flags
		assert   func(t *testing.T, cfg *Config, warnings []Warning)
	}{
		{
			name: "defaults",
			assert: func(t *testing.T, cfg *Config, warnings []Warning) {
				if len(warnings) != 0 {
					t.Fatalf("warnings = %v, want none", warnings)
				}
				if cfg.Output.Dir != "dist" || cfg.Output.BasePath != "/" {
					t.Fatalf("output defaults = %#v", cfg.Output)
				}
				if cfg.Access.NoIndex != true {
					t.Fatalf("access.noindex = false, want true")
				}
				if cfg.Site.Language != "zh-CN" || cfg.Site.Home != "README.md" {
					t.Fatalf("site defaults = %#v", cfg.Site)
				}
				if !slices.Equal(cfg.Ignore, []string{"node_modules/**", ".github/**"}) {
					t.Fatalf("ignore defaults = %v", cfg.Ignore)
				}
				if cfg.Render.MaxFileSize != 5*1024*1024 {
					t.Fatalf("max file size = %d", cfg.Render.MaxFileSize)
				}
				if !cfg.Render.Render || !cfg.Render.Markdown.TOC || !cfg.Render.Markdown.Anchors ||
					!cfg.Render.Markdown.Mermaid || !cfg.Render.Markdown.FrontmatterTitle ||
					cfg.Render.Markdown.Math || cfg.Render.Markdown.TOCMinHeadings != 3 {
					t.Fatalf("markdown defaults = %#v", cfg.Render.Markdown)
				}
				if cfg.Render.HTML.View != "embed" {
					t.Fatalf("html view = %q", cfg.Render.HTML.View)
				}
				if !cfg.Render.Code.LineNumbers || cfg.Render.Code.Theme != "github" {
					t.Fatalf("code defaults = %#v", cfg.Render.Code)
				}
				if cfg.View.TreePosition != "left" || cfg.View.TreeExpandDepth != 2 || cfg.View.TOCPanel != "floating" {
					t.Fatalf("view defaults = %#v", cfg.View)
				}
				if !cfg.Agent.LLMSTxt || !cfg.Agent.LLMSFull.Enabled ||
					cfg.Agent.LLMSFull.MaxSize != 2*1024*1024 || !cfg.Agent.IndexJSON {
					t.Fatalf("agent defaults = %#v", cfg.Agent)
				}
			},
		},
		{
			name: "repository forbidden sections are ignored with warnings",
			repoYAML: `
source:
  repo: evil
output:
  dir: /tmp/evil
access:
  noindex: false
site:
  title: Repo Title
`,
			assert: func(t *testing.T, cfg *Config, warnings []Warning) {
				if cfg.Source.Repo != "" {
					t.Fatalf("source.repo = %q, want ignored", cfg.Source.Repo)
				}
				if cfg.Output.Dir != "dist" {
					t.Fatalf("output.dir = %q, want default", cfg.Output.Dir)
				}
				if !cfg.Access.NoIndex {
					t.Fatalf("access.noindex = false, want default true")
				}
				if cfg.Site.Title != "Repo Title" {
					t.Fatalf("site.title = %q", cfg.Site.Title)
				}
				requireWarningContains(t, warnings, "source is ignored")
				requireWarningContains(t, warnings, "output is ignored")
				requireWarningContains(t, warnings, "access is ignored")
			},
		},
		{
			name: "external overrides repository",
			repoYAML: `
site:
  title: Repo
render:
  markdown:
    math: true
ignore: ["repo/**"]
`,
			extYAML: `
site:
  title: External
render:
  markdown:
    toc: false
ignore: ["external/**"]
`,
			assert: func(t *testing.T, cfg *Config, warnings []Warning) {
				if cfg.Site.Title != "External" {
					t.Fatalf("site.title = %q", cfg.Site.Title)
				}
				if !cfg.Render.Markdown.Math {
					t.Fatalf("repo markdown.math was not preserved")
				}
				if cfg.Render.Markdown.TOC {
					t.Fatalf("external markdown.toc did not override")
				}
				if !slices.Equal(cfg.Ignore, []string{"external/**"}) {
					t.Fatalf("ignore = %v", cfg.Ignore)
				}
			},
		},
		{
			name: "cli overrides external",
			extYAML: `
source:
  repo: from-config
  ref: main
output:
  dir: dist-from-config
  base_path: /docs/
`,
			flags: Flags{
				Repo:      "from-cli",
				Ref:       "release",
				OutputDir: "dist-from-cli",
				BasePath:  "/cli/",
			},
			assert: func(t *testing.T, cfg *Config, warnings []Warning) {
				if cfg.Source.Repo != "from-cli" || cfg.Source.Ref != "release" {
					t.Fatalf("source = %#v", cfg.Source)
				}
				if cfg.Output.Dir != "dist-from-cli" || cfg.Output.BasePath != "/cli/" {
					t.Fatalf("output = %#v", cfg.Output)
				}
			},
		},
		{
			name: "rules concatenate repository before external",
			repoYAML: `
rules:
  - match: "**/*.md"
    markdown: { math: true }
`,
			extYAML: `
rules:
  - match: "docs/**"
    html: { view: direct }
`,
			assert: func(t *testing.T, cfg *Config, warnings []Warning) {
				if len(cfg.Rules) != 2 {
					t.Fatalf("rules length = %d", len(cfg.Rules))
				}
				if cfg.Rules[0].Match != "**/*.md" || cfg.Rules[1].Match != "docs/**" {
					t.Fatalf("rules order = %#v", cfg.Rules)
				}
			},
		},
		{
			name: "size parsing and encryption agent warning",
			extYAML: `
render:
  max_file_size: 1.5MB
agent:
  llms_full:
    max_size: 768KB
access:
  encrypt:
    enabled: true
    paths: ["llms.txt"]
`,
			assert: func(t *testing.T, cfg *Config, warnings []Warning) {
				if cfg.Render.MaxFileSize != 1572864 {
					t.Fatalf("max_file_size = %d", cfg.Render.MaxFileSize)
				}
				if cfg.Agent.LLMSFull.MaxSize != 786432 {
					t.Fatalf("llms_full.max_size = %d", cfg.Agent.LLMSFull.MaxSize)
				}
				requireWarningContains(t, warnings, "overlap")
			},
		},
		{
			name: "toc panel override",
			extYAML: `
view:
  toc_panel: inline
`,
			assert: func(t *testing.T, cfg *Config, warnings []Warning) {
				if len(warnings) != 0 {
					t.Fatalf("warnings = %v, want none", warnings)
				}
				if cfg.View.TOCPanel != "inline" {
					t.Fatalf("toc_panel = %q, want inline", cfg.View.TOCPanel)
				}
			},
		},
		{
			name: "invalid toc panel warns",
			extYAML: `
view:
  toc_panel: drawer
`,
			assert: func(t *testing.T, cfg *Config, warnings []Warning) {
				if cfg.View.TOCPanel != "drawer" {
					t.Fatalf("toc_panel = %q, want raw invalid value preserved", cfg.View.TOCPanel)
				}
				requireWarningContains(t, warnings, "view.toc_panel")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repoRoot := t.TempDir()
			if tt.repoYAML != "" {
				writeFile(t, filepath.Join(repoRoot, repoConfigName), tt.repoYAML)
			}
			externalPath := ""
			if tt.extYAML != "" {
				externalPath = filepath.Join(t.TempDir(), "external.yml")
				writeFile(t, externalPath, tt.extYAML)
			}
			cfg, warnings, err := Load(repoRoot, externalPath, tt.flags)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			tt.assert(t, cfg, warnings)
		})
	}
}

func TestUnknownFieldWarning(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, repoConfigName), `
site:
  title: Known
unknown_top_level: true
`)
	cfg, warnings, err := Load(repoRoot, "", Flags{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Site.Title != "Known" {
		t.Fatalf("site.title = %q", cfg.Site.Title)
	}
	requireWarningContains(t, warnings, "unknown")
}

func TestOptionsForCascades(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeFile(t, filepath.Join(repoRoot, repoConfigName), `
render:
  markdown:
    math: false
  html:
    view: embed
rules:
  - match: "docs/**"
    markdown: { math: true }
    html: { view: direct }
  - match: "docs/private/**"
    markdown: { toc: false }
    html: { view: source }
    max_file_size: 10KB
  - match: "docs/private/archive/**"
    render: false
`)
	cfg, warnings, err := Load(repoRoot, "", Flags{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}

	opts := cfg.OptionsFor("docs/private/archive/a.md")
	if opts.Render {
		t.Fatalf("render = true, want false")
	}
	if !opts.Markdown.Math {
		t.Fatalf("markdown.math = false, want inherited true")
	}
	if opts.Markdown.TOC {
		t.Fatalf("markdown.toc = true, want later rule false")
	}
	if opts.HTML.View != "source" {
		t.Fatalf("html.view = %q, want source", opts.HTML.View)
	}
	if opts.MaxFileSize != 10*1024 {
		t.Fatalf("max_file_size = %d", opts.MaxFileSize)
	}

	before := cfg.Render
	first := cfg.OptionsFor("docs/private/archive/a.md")
	second := cfg.OptionsFor("docs/private/archive/a.md")
	if first != second {
		t.Fatalf("OptionsFor not idempotent: %#v != %#v", first, second)
	}
	if cfg.Render != before {
		t.Fatalf("OptionsFor mutated global render: before=%#v after=%#v", before, cfg.Render)
	}
}

func TestIgnored(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Ignore = []string{"vendor/**", "docs/**/*.tmp"}
	tests := []struct {
		path string
		want bool
	}{
		{path: ".git/config", want: true},
		{path: ".git", want: true},
		{path: ".repolens.yml", want: true},
		{path: "vendor/mod/file.go", want: true},
		{path: "docs/a/b/file.tmp", want: true},
		{path: "docs/a/b/file.md", want: false},
	}
	for _, tt := range tests {
		if got := cfg.Ignored(tt.path); got != tt.want {
			t.Fatalf("Ignored(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func writeFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func requireWarningContains(t *testing.T, warnings []Warning, text string) {
	t.Helper()
	for _, warning := range warnings {
		if strings.Contains(warning.Msg, text) {
			return
		}
	}
	t.Fatalf("warnings %v do not contain %q", warnings, text)
}
