package ui

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ConteMan/repolens/internal/config"
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
		return BuildNative(context.Background(), repo, external, nil, confirm)
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

func TestNativeCacheOverridePreservesDeploymentFiles(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# Preview"), 0600); err != nil {
		t.Fatal(err)
	}
	deploy := t.TempDir()
	routes := filepath.Join(deploy, "_routes.json")
	if err := os.WriteFile(routes, []byte("deployment-owned"), 0600); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(external, []byte("output:\n  dir: "+deploy+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildNative(context.Background(), repo, external, nil, true); BuildOutputErrorCode(err) != "output_not_owned" {
		t.Fatalf("expected protected deployment directory, got %v", err)
	}
	cache := ""
	preview, err := BuildNative(context.Background(), repo, external, &cache, false)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Directory == deploy || preview.Pages == 0 {
		t.Fatalf("unexpected cache result: %#v", preview)
	}
	// This cache entry belongs only to this test's unique temporary repository.
	t.Cleanup(func() { _ = os.RemoveAll(preview.Directory) })
	data, err := os.ReadFile(routes)
	if err != nil || string(data) != "deployment-owned" {
		t.Fatalf("deployment contents changed: %q, %v", data, err)
	}
}

func TestNativeBuildLocations(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "repo")
	external := filepath.Join(dir, "conf", "builder.yml")
	repo, ref, output := "../other", "v1", "../public"
	cfg := config.BuilderSettings{
		Source: config.BuilderSourceSettings{Repo: &repo, Ref: &ref},
		Output: config.BuilderOutputSettings{Dir: &output},
	}
	locations, err := ResolveNativeBuildLocations(root, external, cfg, nil)
	if err != nil || locations.Source.Repo != filepath.Join(dir, "other") || locations.Source.Ref != ref || locations.Source.Worktree || locations.Output != filepath.Join(dir, "public") {
		t.Fatalf("configured locations = %+v, %v", locations, err)
	}
	remote := "git@example.org:owner/docs.git"
	cfg.Source.Repo = &remote
	cache := ""
	locations, err = ResolveNativeBuildLocations(root, external, cfg, &cache)
	if err != nil || locations.Source.Repo != remote || !filepath.IsAbs(locations.RemoteMirror) || !filepath.IsAbs(locations.Output) || locations.Output == filepath.Join(dir, "public") {
		t.Fatalf("remote and forced-cache locations = %+v, %v", locations, err)
	}
	wantCache := locations.Output
	locations, err = ResolveNativeBuildLocations(root, "", config.BuilderSettings{}, nil)
	if err != nil || !locations.Source.Worktree || locations.Source.Repo != root || locations.RemoteMirror != "" || locations.Output != wantCache {
		t.Fatalf("worktree and default cache = %+v, %v", locations, err)
	}
	if _, err := os.Stat(wantCache); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview created cache output %s: %v", wantCache, err)
	}
}
