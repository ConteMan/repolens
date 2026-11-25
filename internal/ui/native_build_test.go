package ui

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeBuildHonorsExternalOutput(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	if err := exec.Command("git", "init", "-q", repo).Run(); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(dir, "builder.yml")
	for path, body := range map[string]string{filepath.Join(repo, "README.md"): "# Working tree\n", external: "site:\n  title: External title\noutput:\n  dir: published\n"} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	build := func(confirm bool) (NativeBuildResult, error) {
		return BuildNative(context.Background(), repo, external, "", confirm)
	}
	result, err := build(false)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(filepath.Join(dir, "published"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Directory != want || result.Source != repo || result.Pages == 0 {
		t.Fatalf("unexpected result %#v", result)
	}
	page, err := os.ReadFile(filepath.Join(result.Directory, "view", "index.html"))
	if err != nil || !strings.Contains(string(page), "External title") {
		t.Fatalf("missing rendered site title: %v", err)
	}
	if _, err = build(false); !errors.Is(err, ErrConfirmOutput) {
		t.Fatalf("expected overwrite confirmation, got %v", err)
	}
	if _, err = build(true); err != nil {
		t.Fatal(err)
	}
}
