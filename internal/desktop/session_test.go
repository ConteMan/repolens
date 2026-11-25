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
