package site

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/ConteMan/repolens/internal/source"
	"github.com/ConteMan/repolens/internal/theme"
)

func TestLoadGlossary(t *testing.T) {
	t.Parallel()

	t.Run("missing directory is empty", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		got, warnings, err := LoadGlossary(root, ".repolens/glossary")
		if err != nil {
			t.Fatalf("LoadGlossary() error = %v", err)
		}
		if len(got) != 0 || warnings != nil {
			t.Fatalf("got %#v warnings %v, want empty", got, warnings)
		}
	})

	t.Run("invalid filename skipped with warning", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeFile(t, root, ".repolens/glossary/Foo.yml", "title: Bad\n")
		writeFile(t, root, ".repolens/glossary/mediation.yml", "title: Mediation\nsummary: A layer.\n")
		got, warnings, err := LoadGlossary(root, ".repolens/glossary")
		if err != nil {
			t.Fatalf("LoadGlossary() error = %v", err)
		}
		if len(got) != 1 || got["mediation"].Title.Text != "Mediation" {
			t.Fatalf("glossary = %#v", got)
		}
		if got["mediation"].DefinedIn != ".repolens/glossary/mediation.yml" {
			t.Fatalf("DefinedIn = %q", got["mediation"].DefinedIn)
		}
		if len(warnings) != 1 || !strings.Contains(warnings[0], "Foo.yml") {
			t.Fatalf("warnings = %v", warnings)
		}
	})

	t.Run("yml and yaml conflict", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeFile(t, root, ".repolens/glossary/mediation.yml", "title: A\n")
		writeFile(t, root, ".repolens/glossary/mediation.yaml", "title: B\n")
		_, _, err := LoadGlossary(root, ".repolens/glossary")
		if err == nil {
			t.Fatal("LoadGlossary() error = nil, want conflict")
		}
		if !strings.Contains(err.Error(), "mediation") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("yaml parse error", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		writeFile(t, root, ".repolens/glossary/mediation.yml", "title: [\n")
		_, _, err := LoadGlossary(root, ".repolens/glossary")
		if err == nil {
			t.Fatal("LoadGlossary() error = nil, want parse error")
		}
		if !strings.Contains(err.Error(), "mediation.yml") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("invalid url and truncated field warn", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		long := strings.Repeat("x", 2001)
		writeFile(t, root, ".repolens/glossary/mediation.yml", "title: Mediation\nsummary: "+long+"\nsource:\n  label: x\n  url: javascript:alert(1)\n")
		got, warnings, err := LoadGlossary(root, ".repolens/glossary")
		if err != nil {
			t.Fatalf("LoadGlossary() error = %v", err)
		}
		if got["mediation"].Source != nil {
			t.Fatalf("source kept: %+v", got["mediation"].Source)
		}
		if got["mediation"].Summary.Text != strings.Repeat("x", 2000) {
			t.Fatalf("summary len = %d", len(got["mediation"].Summary.Text))
		}
		if len(warnings) != 2 {
			t.Fatalf("warnings = %v, want 2", warnings)
		}
		joined := strings.Join(warnings, "\n")
		if !strings.Contains(joined, "javascript:alert(1)") || !strings.Contains(joined, "summary") || !strings.Contains(joined, "mediation") {
			t.Fatalf("warnings = %v", warnings)
		}
	})
}

func TestGlossaryBuild(t *testing.T) {
	t.Parallel()

	t.Run("refs allows incomplete and fails undefined", func(t *testing.T) {
		t.Parallel()
		repo := newGlossaryGitRepo(t, map[string]string{
			".repolens.yml": `render:
  markdown:
    glossary: true
glossary:
  strict: refs
`,
			".repolens/glossary/mediation.yml": "title: Mediation\n",
			"README.md":                        "See [广告聚合](term:mediation).\n",
		})
		_, _, err := buildSite(t, repo)
		if err != nil {
			t.Fatalf("incomplete under refs should succeed: %v", err)
		}

		repo = newGlossaryGitRepo(t, map[string]string{
			".repolens.yml": `render:
  markdown:
    glossary: true
glossary:
  strict: refs
`,
			".repolens/glossary/mediation.yml": "title: Mediation\nsummary: A layer.\n",
			"README.md":                        "See [missing](term:unknown).\n",
		})
		_, _, err = buildSite(t, repo)
		if err == nil {
			t.Fatal("undefined under refs should fail")
		}
		if !strings.Contains(err.Error(), "unknown") || !strings.Contains(err.Error(), "README.md") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("complete fails incomplete with defined-in path", func(t *testing.T) {
		t.Parallel()
		repo := newGlossaryGitRepo(t, map[string]string{
			".repolens.yml": `render:
  markdown:
    glossary: true
glossary:
  strict: complete
`,
			".repolens/glossary/mediation.yml": "title: Mediation\n",
			"README.md":                        "See [广告聚合](term:mediation).\n",
		})
		_, _, err := buildSite(t, repo)
		if err == nil {
			t.Fatal("incomplete under complete should fail")
		}
		if !strings.Contains(err.Error(), "mediation") || !strings.Contains(err.Error(), ".repolens/glossary/mediation.yml") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("complete treats page-only as complete", func(t *testing.T) {
		t.Parallel()
		repo := newGlossaryGitRepo(t, map[string]string{
			".repolens.yml": `render:
  markdown:
    glossary: true
glossary:
  strict: complete
`,
			".repolens/glossary/mediation.yml": "title: Mediation\n",
			"README.md": `---
glossary:
  mediation:
    page: 本文特化解释。
---

See [广告聚合](term:mediation).
`,
		})
		_, _, err := buildSite(t, repo)
		if err != nil {
			t.Fatalf("page-only should be complete: %v", err)
		}
	})

	t.Run("external config overrides strict to complete", func(t *testing.T) {
		t.Parallel()
		repo := newGlossaryGitRepo(t, map[string]string{
			".repolens.yml": `render:
  markdown:
    glossary: true
glossary:
  strict: refs
`,
			".repolens/glossary/mediation.yml": "title: Mediation\n",
			"README.md":                        "See [广告聚合](term:mediation).\n",
		})
		_, _, err := buildSite(t, repo)
		if err != nil {
			t.Fatalf("refs should allow incomplete: %v", err)
		}
		ext := filepath.Join(t.TempDir(), "external.yml")
		if err := os.WriteFile(ext, []byte("glossary:\n  strict: complete\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, _, err = buildSiteWithExternal(t, repo, ext)
		if err == nil {
			t.Fatal("external complete should fail")
		}
		if !strings.Contains(err.Error(), "mediation") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("search terms and llms glossary", func(t *testing.T) {
		t.Parallel()
		repo := newGlossaryGitRepo(t, map[string]string{
			".repolens.yml": `render:
  markdown:
    glossary: true
site:
  title: Terms
`,
			".repolens/glossary/mediation.yml": "title: \"`af` layer\"\nalias: 聚合\nsummary: A mediation layer.\n",
			"README.md": `---
glossary:
  local-only:
    title: Local
    summary: Private term.
---

See [广告聚合](term:mediation) and [本地](term:local-only).
`,
		})
		outDir, _, err := buildSite(t, repo)
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}

		var index struct {
			Docs []struct {
				Path  string `json:"path"`
				Terms []struct {
					Key    string `json:"key"`
					Title  string `json:"title"`
					Alias  string `json:"alias"`
					Anchor string `json:"anchor"`
				} `json:"terms"`
			} `json:"docs"`
		}
		if err := json.Unmarshal([]byte(readOutput(t, outDir, "search.json")), &index); err != nil {
			t.Fatalf("search.json: %v", err)
		}
		var terms []struct {
			Key    string `json:"key"`
			Title  string `json:"title"`
			Alias  string `json:"alias"`
			Anchor string `json:"anchor"`
		}
		for _, doc := range index.Docs {
			if doc.Path == "README.md" {
				terms = doc.Terms
			}
		}
		if len(terms) != 2 {
			t.Fatalf("README terms = %#v", terms)
		}
		if terms[0].Key != "mediation" || terms[0].Title != "af layer" || terms[0].Alias != "聚合" || terms[0].Anchor != "glossary-mediation" {
			t.Fatalf("first term = %#v", terms[0])
		}
		if terms[1].Key != "local-only" || terms[1].Anchor != "glossary-local-only" {
			t.Fatalf("private term = %#v", terms[1])
		}

		llms := readOutput(t, outDir, "llms.txt")
		assertContains(t, llms, "## 术语表")
		assertContains(t, llms, "af layer")
		assertContains(t, llms, "聚合")
		assertContains(t, llms, "A mediation layer.")
		assertContains(t, llms, ".repolens/glossary/mediation.yml")
		section := llms[strings.Index(llms, "## 术语表"):]
		assertNotContains(t, section, "local-only")
		assertNotContains(t, section, "Private term")
		assertNotContains(t, section, "Local")

		full := readOutput(t, outDir, "llms-full.txt")
		assertContains(t, full, "----- README.md -----")
		assertContains(t, full, "(term:mediation)")
		assertNotContains(t, full, `class="term"`)

		assertExists(t, outDir, ".repolens/glossary/mediation.yml")
		page := readOutput(t, outDir, "view/README.md/index.html")
		assertContains(t, page, `data-glossary="mediation"`)
		assertNotContains(t, page, `href="term:`)
	})

	t.Run("warnings not double counted", func(t *testing.T) {
		t.Parallel()
		repo := newGlossaryGitRepo(t, map[string]string{
			".repolens.yml": `render:
  markdown:
    glossary: true
`,
			".repolens/glossary/mediation.yml": "title: Mediation\nsummary: Public.\nsource:\n  url: javascript:lib\n",
			"README.md": `---
glossary:
  mediation:
    page: override
    source:
      label: x
      url: javascript:page
---

See [广告聚合](term:mediation).
`,
		})
		_, stats, err := buildSite(t, repo)
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		var lib, page int
		for _, w := range stats.Warnings {
			if strings.Contains(w.Msg, "javascript:lib") {
				lib++
			}
			if strings.Contains(w.Msg, "javascript:page") {
				page++
			}
		}
		if lib != 1 || page != 1 {
			t.Fatalf("warnings = %#v, want one lib and one page", stats.Warnings)
		}
	})

	t.Run("zero impact when disabled", func(t *testing.T) {
		t.Parallel()
		repo := newSiteTestRepo(t)
		outDir, _, err := buildSite(t, repo)
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		search := readOutput(t, outDir, "search.json")
		if strings.Contains(search, `"terms"`) {
			t.Fatal("search.json grew a terms field while glossary is disabled")
		}
		llms := readOutput(t, outDir, "llms.txt")
		assertNotContains(t, llms, "## 术语表")
		assertNotContains(t, llms, "## Glossary")
		root := readOutput(t, outDir, "view/README.md/index.html")
		assertNotContains(t, root, `class="term"`)
		assertNotContains(t, root, `href="term:`)
	})
}

func newGlossaryGitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	for rel, content := range files {
		writeFile(t, repo, rel, content)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

func buildSiteWithExternal(t *testing.T, repo, externalPath string) (string, Stats, error) {
	t.Helper()
	return buildSiteWithConfigAndExternal(t, repo, externalPath, nil)
}

func buildSiteWithConfigAndExternal(t *testing.T, repo, externalPath string, mutate func(*config.Config)) (string, Stats, error) {
	t.Helper()
	tree, err := source.Open(context.Background(), source.Spec{Repo: repo})
	if err != nil {
		t.Fatalf("source.Open() error = %v", err)
	}
	defer tree.Cleanup()

	cfg, _, err := config.Load(tree.Root, externalPath, config.Flags{Repo: repo})
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
