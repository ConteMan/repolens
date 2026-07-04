package source

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestOpenLocalPathMatchesGitTreeAndMetadata(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	tree, err := Open(ctx, Spec{Repo: repo})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer cleanupTree(t, tree)

	head := gitString(t, repo, "rev-parse", "HEAD")
	if tree.CommitHash != head {
		t.Fatalf("CommitHash = %q, want %q", tree.CommitHash, head)
	}

	gotPaths := filePaths(tree.Files)
	wantPaths := gitLines(t, repo, "ls-tree", "-r", "--name-only", "HEAD")
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("paths = %#v, want %#v", gotPaths, wantPaths)
	}

	readme := findFile(t, tree.Files, "README.md")
	wantReadmeCommit := gitString(t, repo, "log", "-1", "--format=%H", "HEAD", "--", "README.md")
	if readme.LastCommit == nil || readme.LastCommit.Hash != wantReadmeCommit {
		t.Fatalf("README.md LastCommit = %#v, want hash %s", readme.LastCommit, wantReadmeCommit)
	}

	app := findFile(t, tree.Files, "src/app.go")
	wantAppCommit := gitString(t, repo, "log", "-1", "--format=%H", "HEAD", "--", "src/app.go")
	if app.LastCommit == nil || app.LastCommit.Hash != wantAppCommit {
		t.Fatalf("src/app.go LastCommit = %#v, want hash %s", app.LastCommit, wantAppCommit)
	}
}

func TestOpenRemoteURLUsesCacheAndFetch(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))

	repo := newTestRepo(t)
	remote := filepath.Join(t.TempDir(), "remote.git")
	runGitTest(t, "", "clone", "--bare", repo, remote)
	remoteURL := "file://" + filepath.ToSlash(remote)

	first, err := Open(ctx, Spec{Repo: remoteURL})
	if err != nil {
		t.Fatalf("first Open() error = %v", err)
	}
	firstSummary := fileSummaries(first.Files)
	firstHash := first.CommitHash
	cleanupTree(t, first)

	cacheDir, err := cachedRepoPath(remoteURL)
	if err != nil {
		t.Fatalf("cachedRepoPath() error = %v", err)
	}
	marker := filepath.Join(cacheDir, "repolens-test-marker")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write cache marker: %v", err)
	}

	writeFile(t, repo, "CHANGELOG.md", "changes\n")
	runGitTest(t, repo, "add", "CHANGELOG.md")
	runGitTest(t, repo, "commit", "-m", "add changelog")
	runGitTest(t, repo, "push", remote, "HEAD:main")

	second, err := Open(ctx, Spec{Repo: remoteURL})
	if err != nil {
		t.Fatalf("second Open() error = %v", err)
	}
	defer cleanupTree(t, second)

	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("cache marker missing after second Open; cache may have been recloned: %v", err)
	}
	if second.CommitHash == firstHash {
		t.Fatalf("second CommitHash = first hash %s; fetch did not observe pushed commit", firstHash)
	}
	if _, ok := fileSummaries(second.Files)["CHANGELOG.md"]; !ok {
		t.Fatalf("second Open files missing fetched CHANGELOG.md: %#v", filePaths(second.Files))
	}

	local, err := Open(ctx, Spec{Repo: repo})
	if err != nil {
		t.Fatalf("local Open() error = %v", err)
	}
	defer cleanupTree(t, local)
	if !reflect.DeepEqual(fileSummaries(local.Files), fileSummaries(second.Files)) {
		t.Fatalf("remote files = %#v, want local files %#v", fileSummaries(second.Files), fileSummaries(local.Files))
	}
	if reflect.DeepEqual(firstSummary, fileSummaries(second.Files)) {
		t.Fatalf("second Open file summary did not change after fetch")
	}
}

func TestOpenSkipsSymlinks(t *testing.T) {
	ctx := context.Background()
	repo := newEmptyRepo(t)
	writeFile(t, repo, "target.txt", "target\n")
	if err := os.Symlink("target.txt", filepath.Join(repo, "link.txt")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	runGitTest(t, repo, "add", "target.txt", "link.txt")
	runGitTest(t, repo, "commit", "-m", "add symlink")

	tree, err := Open(ctx, Spec{Repo: repo})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer cleanupTree(t, tree)

	if paths := filePaths(tree.Files); !reflect.DeepEqual(paths, []string{"target.txt"}) {
		t.Fatalf("paths = %#v, want only target.txt", paths)
	}
	if _, err := os.Lstat(filepath.Join(tree.Root, "link.txt")); !os.IsNotExist(err) {
		t.Fatalf("materialized symlink Lstat error = %v, want not exist", err)
	}
}

func TestOpenWorktreeIncludesUncommittedFilesBestEffortMetadata(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)
	writeFile(t, repo, "draft.txt", "draft\n")

	tree, err := Open(ctx, Spec{Repo: repo, Worktree: true})
	if err != nil {
		t.Fatalf("Open(worktree) error = %v", err)
	}
	defer cleanupTree(t, tree)

	if tree.Root != repo {
		t.Fatalf("Root = %q, want %q", tree.Root, repo)
	}
	if tree.CommitHash != "" {
		t.Fatalf("CommitHash = %q, want empty in worktree mode", tree.CommitHash)
	}

	draft := findFile(t, tree.Files, "draft.txt")
	if draft.LastCommit != nil {
		t.Fatalf("draft LastCommit = %#v, want nil", draft.LastCommit)
	}
	readme := findFile(t, tree.Files, "README.md")
	if readme.LastCommit == nil {
		t.Fatalf("README.md LastCommit = nil, want best-effort metadata")
	}

	if err := tree.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if _, err := os.Stat(repo); err != nil {
		t.Fatalf("worktree root removed by Cleanup: %v", err)
	}
}

func TestOpenWorktreeWithoutGitReturnsNilMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	writeFile(t, root, "notes.txt", "notes\n")

	tree, err := Open(ctx, Spec{Repo: root, Worktree: true})
	if err != nil {
		t.Fatalf("Open(worktree non-git) error = %v", err)
	}
	defer cleanupTree(t, tree)

	if len(tree.Files) != 1 || tree.Files[0].Path != "notes.txt" {
		t.Fatalf("Files = %#v, want notes.txt", tree.Files)
	}
	if tree.Files[0].LastCommit != nil {
		t.Fatalf("LastCommit = %#v, want nil", tree.Files[0].LastCommit)
	}
}

func newTestRepo(t *testing.T) string {
	t.Helper()
	repo := newEmptyRepo(t)

	writeFile(t, repo, "README.md", "# repolens\n")
	writeFile(t, repo, "src/main.go", "package main\n")
	runGitTest(t, repo, "add", ".")
	runGitTest(t, repo, "commit", "-m", "initial commit")

	writeFile(t, repo, "README.md", "# repolens\n\nupdated\n")
	runGitTest(t, repo, "add", "README.md")
	runGitTest(t, repo, "commit", "-m", "update readme")

	runGitTest(t, repo, "mv", "src/main.go", "src/app.go")
	runGitTest(t, repo, "commit", "-m", "rename app")

	return repo
}

func newEmptyRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitTest(t, "", "init", "-q", "-b", "main", repo)
	runGitTest(t, repo, "config", "user.email", "test@example.com")
	runGitTest(t, repo, "config", "user.name", "Test User")
	return repo
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func runGitTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func gitString(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func gitLines(t *testing.T, dir string, args ...string) []string {
	t.Helper()
	out := gitString(t, dir, args...)
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	sort.Strings(lines)
	return lines
}

func filePaths(files []File) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

func fileSummaries(files []File) map[string]string {
	summaries := make(map[string]string, len(files))
	for _, file := range files {
		hash := ""
		if file.LastCommit != nil {
			hash = file.LastCommit.Hash
		}
		summaries[file.Path] = hash
	}
	return summaries
}

func findFile(t *testing.T, files []File, path string) File {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("file %q not found in %#v", path, filePaths(files))
	return File{}
}

func cleanupTree(t *testing.T, tree *Tree) {
	t.Helper()
	if err := tree.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
}
