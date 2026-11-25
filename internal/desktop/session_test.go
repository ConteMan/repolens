package desktop

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ConteMan/repolens/internal/config"
)

func TestOpenWorktreeRoot(t *testing.T) {
	parent := t.TempDir()
	worktree := filepath.Join(parent, "work tree")
	bare := filepath.Join(parent, "bare")
	initGit(t, worktree)
	initGit(t, bare, "--bare")
	subdir := filepath.Join(worktree, "child")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(parent, "alias")
	if err := os.Symlink(worktree, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"", parent, bare, subdir} {
		if _, err := Open(path); err == nil {
			t.Errorf("Open(%q) succeeded", path)
		}
	}
	session, err := Open(alias)
	if err != nil || len(session.Rules) != 0 || session.Dirty() {
		t.Fatalf("Open(alias) = %#v, %v", session, err)
	}
	writeConfig(t, filepath.Join(worktree, ".repolens.yml"), "rules: invalid\n")
	if _, err := Open(worktree); err == nil {
		t.Fatal("opened an invalid repository config")
	}
}

func TestSaveRulesPreservesRepositorySettings(t *testing.T) {
	worktree := t.TempDir()
	initGit(t, worktree)
	path := filepath.Join(worktree, ".repolens.yml")
	const original = `source:
  repo: private-source
output:
  dir: private-output
access:
  encrypt:
    password_env: PRIVATE_PASSWORD
site:
  title: Keep this title
glossary:
  dir: glossary
rules:
  - match: "**/*.md"
    render: false
    markdown:
      toc: false
      vendor_option: keep
  - match: "docs/**"
    render: true
`
	writeConfig(t, path, original)
	session, err := Open(worktree)
	if err != nil {
		t.Fatal(err)
	}
	// Direct pointer edits must not mutate the saved baseline.
	*session.Rules[0].Markdown.TOC = true
	if !session.Dirty() {
		t.Fatal("editing an existing pointed-to value did not mark the draft dirty")
	}
	session.Rules[0].Markdown.TOC = nil
	slices.Reverse(session.Rules)
	session.Rules = append(session.Rules, config.RepositoryRuleSettings{Match: new("notes/**")})
	diff, err := session.Prepare()
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"private-source", "private-output", "PRIVATE_PASSWORD", "Keep this title"} {
		if strings.Contains(diff, value) {
			t.Errorf("preview exposed %q", value)
		}
	}
	if !strings.Contains(diff, "notes/**") || readConfig(t, path) != original {
		t.Fatalf("prepare changed the file or missed the draft: %s", diff)
	}
	session.CancelSave()
	if err := session.Save(); err == nil || !session.Dirty() || readConfig(t, path) != original {
		t.Fatalf("cancel did not preserve the draft and file: %v", err)
	}
	if _, err := session.Prepare(); err != nil {
		t.Fatal(err)
	}
	if err := session.Save(); err != nil {
		t.Fatal(err)
	}
	if session.Dirty() || session.Save() == nil {
		t.Fatal("save did not update the baseline or consume the confirmation")
	}
	for _, value := range []string{"private-source", "private-output", "PRIVATE_PASSWORD", "Keep this title", "vendor_option: keep"} {
		if !strings.Contains(readConfig(t, path), value) {
			t.Errorf("write dropped %q", value)
		}
	}
	reopened, err := Open(worktree)
	if err != nil {
		t.Fatal(err)
	}
	if rule := reopened.Rules[1]; *rule.Match != "**/*.md" || rule.Render == nil || *rule.Render || rule.Markdown.TOC != nil {
		t.Fatalf("reopened rule did not retain order, explicit false and inheritance: %+v", rule)
	}
	cfg, _, err := config.Load(worktree, "", config.Flags{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OptionsFor("docs/readme.md").Render || !cfg.OptionsFor("docs/readme.go").Render {
		t.Fatal("saved order did not control the effective rendering rule")
	}

	session.Rules = nil
	if _, err := session.Prepare(); err != nil {
		t.Fatal(err)
	}
	if err := session.Save(); err != nil {
		t.Fatal(err)
	}
	document, err := config.LoadRepositoryDocument(worktree)
	if err != nil {
		t.Fatal(err)
	}
	if document.Settings.Rules == nil || len(*document.Settings.Rules) != 0 {
		t.Fatal("deleting the final rule did not persist an explicit empty list")
	}
}

func TestSaveRulesRejectsStaleConfirmationAndRevision(t *testing.T) {
	worktree := t.TempDir()
	initGit(t, worktree)
	path := filepath.Join(worktree, ".repolens.yml")
	session, err := Open(worktree)
	if err != nil {
		t.Fatal(err)
	}
	session.Rules = []config.RepositoryRuleSettings{{Match: new("[")}}
	var validation *config.RepositoryValidationError
	if _, err := session.Prepare(); !errors.As(err, &validation) || validation.Issues[0].Path != "rules[0].match" {
		t.Fatalf("Prepare() validation error = %v", err)
	}
	session.Rules[0].Match = new("docs/**")
	if _, err := session.Prepare(); err != nil {
		t.Fatal(err)
	}
	*session.Rules[0].Match = "notes/**"
	if session.Save() == nil {
		t.Fatal("saved a draft that changed after prepare")
	}
	if _, err := session.Prepare(); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := session.Save(); err == nil || !session.Dirty() {
		t.Fatalf("failed write did not retain the draft: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Prepare(); err != nil {
		t.Fatal(err)
	}
	const external = "site:\n  title: External edit\n"
	writeConfig(t, path, external)
	if err := session.Save(); !errors.Is(err, config.ErrRevisionConflict) {
		t.Fatalf("Save() error = %v, want revision conflict", err)
	}
	if !session.Dirty() || readConfig(t, path) != external {
		t.Fatal("conflict discarded draft or changed the file")
	}
	draft := session.Rules
	if err := session.Reload(); err != nil {
		t.Fatal(err)
	}
	session.Rules = draft
	if _, err := session.Prepare(); err != nil {
		t.Fatal(err)
	}
	if err := session.Save(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readConfig(t, path), external) {
		t.Fatal("save after reload overwrote external edit")
	}
}

func initGit(t *testing.T, path string, opts ...string) {
	t.Helper()
	args := append([]string{"init", "-q"}, opts...)
	if output, err := exec.Command("git", append(args, path)...).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
}

func writeConfig(t *testing.T, path, config string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readConfig(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSiteAndBuilderChangesShareConfirmation(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	local := filepath.Join(root, ".repolens.yml")
	external := filepath.Join(t.TempDir(), "builder.yml")
	const localSource = "source:\n  repo: untrusted-source\n"
	const externalSource = "access:\n  encrypt:\n    password_env: PRIVATE_PASSWORD\n"
	writeConfig(t, local, localSource)
	writeConfig(t, external, externalSource)

	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if session.LoadBuilder(local) == nil {
		t.Fatal("selected repository configuration as trusted")
	}
	if err := session.LoadBuilder(external); err != nil {
		t.Fatal(err)
	}
	session.Settings.Site.Title = new("New title")
	session.Settings.Glossary.Strict = new("complete")
	session.Settings.Ignore = new([]string{"drafts/**"})
	session.Builder.Access.NoIndex = new(false)
	session.Builder.Output.Dir = new("build")
	changes, err := session.PrepareChanges()
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 || changes[0].Path != session.baseline.Path ||
		changes[1].Path != session.BuilderPath ||
		!strings.Contains(changes[0].Diff, "New title") ||
		!strings.Contains(changes[1].Diff, "noindex: false") ||
		strings.Contains(changes[0].Diff, "untrusted-source") ||
		strings.Contains(changes[1].Diff, "PRIVATE_PASSWORD") {
		t.Fatalf("unexpected changes: %+v", changes)
	}
	writeConfig(t, external, externalSource+"vendor: separate edit\n")
	if err := session.Save(); !errors.Is(err, config.ErrRevisionConflict) {
		t.Fatalf("Save() error = %v, want external revision conflict", err)
	}
	if readConfig(t, local) != localSource || !session.Dirty() || !session.BuilderDirty() {
		t.Fatal("existing file or drafts changed after conflict")
	}
	writeConfig(t, external, externalSource)
	if _, err := session.PrepareChanges(); err != nil {
		t.Fatal(err)
	}
	if err := session.Save(); err != nil {
		t.Fatal(err)
	}
	if session.Dirty() || !strings.Contains(readConfig(t, local), localSource) ||
		!strings.Contains(readConfig(t, external), externalSource) {
		t.Fatal("save dropped configuration or left unapplied drafts")
	}
	loaded, _, err := config.Load(root, external, config.Flags{})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Site.Title != "New title" || loaded.Glossary.Strict != "complete" ||
		!loaded.Ignored("drafts/example.md") || loaded.Access.NoIndex || loaded.Output.Dir != "build" ||
		loaded.Source.Repo != "" {
		t.Fatalf("saved values did not reach both configuration domains: %+v", loaded)
	}
}

func TestSharedExternalDraftAppliesBuildPrecedence(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	initGit(t, root)
	repository := filepath.Join(root, ".repolens.yml")
	external := filepath.Join(t.TempDir(), "external.yml")
	const repoYAML = "site:\n  title: Repository\nrender:\n  markdown:\n    toc: true\n"
	const externalYAML = "site:\n  title: External\naccess:\n  noindex: false\ncustom: keep\n"
	writeConfig(t, repository, repoYAML)
	writeConfig(t, external, externalYAML)
	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.LoadBuilder(external); err != nil {
		t.Fatal(err)
	}

	session.Settings.Site.Title = new("Repository Draft")
	session.ExternalSettings.Site.Title = new("External Draft")
	session.ExternalSettings.Render.Markdown.TOC = new(false)
	session.ExternalRules = append(session.ExternalRules, config.RepositoryRuleSettings{Match: new("docs/**")})

	resolved, err := config.ResolveSharedSettings(session.Settings, session.ExternalSettings)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Site.Title == nil || *resolved.Site.Title != "External Draft" ||
		resolved.Render.Markdown.TOC == nil || *resolved.Render.Markdown.TOC {
		t.Fatal("shared draft precedence is incorrect")
	}

	changes, err := session.PrepareChanges()
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 || !strings.Contains(changes[0].Diff, "Repository Draft") ||
		!strings.Contains(changes[1].Diff, "External Draft") || !strings.Contains(changes[1].Diff, "docs/**") {
		t.Fatalf("unexpected shared differences %#v", changes)
	}
	if readConfig(t, external) != externalYAML || readConfig(t, repository) != repoYAML {
		t.Fatal("preparing changed a file")
	}
	if err := session.Save(); err != nil {
		t.Fatal(err)
	}
	effective, _, err := config.Load(root, external, config.Flags{})
	if err != nil {
		t.Fatal(err)
	}
	if effective.Site.Title != "External Draft" || effective.Render.Markdown.TOC ||
		effective.Access.NoIndex || len(effective.Rules) != 1 || !strings.Contains(readConfig(t, external), "custom: keep") ||
		session.Dirty() {
		t.Fatalf("persisted configuration does not match the prepared draft: %#v", effective)
	}
}
