package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "repolens dev") {
		t.Errorf("unexpected version output: %q", got)
	}
}

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name             string
		injected         string
		buildInfoVersion string
		want             string
	}{
		{name: "ldflags wins", injected: "v1.2.3", buildInfoVersion: "v1.0.0", want: "v1.2.3"},
		{name: "go install fallback", injected: "dev", buildInfoVersion: "v1.2.3", want: "v1.2.3"},
		{name: "devel stays dev", injected: "dev", buildInfoVersion: "(devel)", want: "dev"},
		{name: "empty stays dev", injected: "", buildInfoVersion: "", want: "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.injected, tt.buildInfoVersion); got != tt.want {
				t.Fatalf("resolveVersion(%q, %q) = %q, want %q", tt.injected, tt.buildInfoVersion, got, tt.want)
			}
		})
	}
}

func TestBuildCommand(t *testing.T) {
	repo := newCLITestRepo(t)
	outDir := filepath.Join(t.TempDir(), "dist")
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"build", repo, "-o", outDir})

	if err := root.Execute(); err != nil {
		t.Fatalf("build command failed: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "Built ") {
		t.Fatalf("build output = %q, want stats", got)
	}
	if _, err := os.Stat(filepath.Join(outDir, "view", "index.html")); err != nil {
		t.Fatalf("view/index.html missing: %v", err)
	}
}

func newCLITestRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runCLIGit(t, repo, "init", "-b", "main")
	runCLIGit(t, repo, "config", "user.email", "test@example.com")
	runCLIGit(t, repo, "config", "user.name", "Test User")
	cliWriteFile(t, repo, "README.md", "# CLI\n")
	runCLIGit(t, repo, "add", ".")
	runCLIGit(t, repo, "commit", "-m", "initial")
	return repo
}

func runCLIGit(t *testing.T, dir string, args ...string) {
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

func cliWriteFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", p, err)
	}
}
