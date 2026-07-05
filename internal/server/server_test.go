package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/ConteMan/repolens/internal/site"
	"github.com/ConteMan/repolens/internal/source"
	"github.com/ConteMan/repolens/internal/theme"
)

func TestRunServesAndRebuildsWorktree(t *testing.T) {
	repo := t.TempDir()
	writeServerTestFile(t, repo, "README.md", "# First\n\nInitial body.\n")
	addr := freeAddr(t)
	var builtDirs safeDirs

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := runServerInDir(t, ctx, repo, Options{Addr: addr, Worktree: true}, testRebuild(repo, &builtDirs, nil))

	baseURL := "http://" + addr
	waitForHTTP(t, baseURL+"/view/README.md/", func(resp *http.Response, body string) error {
		if resp.StatusCode != http.StatusOK {
			return errors.New(resp.Status)
		}
		if !strings.Contains(body, "First") {
			return errors.New("browser page does not contain initial content")
		}
		return nil
	})
	waitForHTTP(t, baseURL+"/README.md", func(resp *http.Response, body string) error {
		if resp.StatusCode != http.StatusOK {
			return errors.New(resp.Status)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/markdown") {
			return errors.New("mirror markdown Content-Type = " + ct)
		}
		if !strings.Contains(body, "Initial body") {
			return errors.New("mirror markdown body mismatch")
		}
		return nil
	})

	writeServerTestFile(t, repo, "README.md", "# Second\n\nUpdated body.\n")
	waitForHTTP(t, baseURL+"/view/README.md/", func(resp *http.Response, body string) error {
		if resp.StatusCode != http.StatusOK {
			return errors.New(resp.Status)
		}
		if !strings.Contains(body, "Second") || !strings.Contains(body, "Updated body") {
			return errors.New("browser page does not contain rebuilt content")
		}
		return nil
	})

	cancel()
	if err := waitServerDone(done); err != nil {
		t.Fatalf("server returned error: %v", err)
	}
	waitPortAvailable(t, addr)
	for _, dir := range builtDirs.snapshot() {
		if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("built directory %s still exists after shutdown: %v", dir, err)
		}
	}
}

func TestRunKeepsOldRootWhenRebuildFails(t *testing.T) {
	repo := t.TempDir()
	writeServerTestFile(t, repo, "README.md", "# Stable\n\nOld content.\n")
	addr := freeAddr(t)
	var builtDirs safeDirs
	var fail atomic.Bool
	var failedAttempts atomic.Int32

	rebuild := testRebuild(repo, &builtDirs, func() error {
		if fail.Load() {
			failedAttempts.Add(1)
			return errors.New("configured rebuild failure")
		}
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := runServerInDir(t, ctx, repo, Options{Addr: addr, Worktree: true}, rebuild)
	baseURL := "http://" + addr

	waitForHTTP(t, baseURL+"/view/README.md/", func(resp *http.Response, body string) error {
		if resp.StatusCode != http.StatusOK {
			return errors.New(resp.Status)
		}
		if !strings.Contains(body, "Stable") {
			return errors.New("browser page does not contain stable content")
		}
		return nil
	})

	fail.Store(true)
	writeServerTestFile(t, repo, "README.md", "# Broken\n\nNew content should not be served.\n")
	waitUntil(t, func() error {
		if failedAttempts.Load() == 0 {
			return errors.New("rebuild failure has not been observed")
		}
		return nil
	})
	waitForHTTP(t, baseURL+"/view/README.md/", func(resp *http.Response, body string) error {
		if resp.StatusCode != http.StatusOK {
			return errors.New(resp.Status)
		}
		if !strings.Contains(body, "Stable") {
			return errors.New("old content was not retained")
		}
		if strings.Contains(body, "Broken") {
			return errors.New("failed rebuild content leaked into served root")
		}
		return nil
	})

	cancel()
	if err := waitServerDone(done); err != nil {
		t.Fatalf("server returned error: %v", err)
	}
	for _, dir := range builtDirs.snapshot() {
		if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("built directory %s still exists after shutdown: %v", dir, err)
		}
	}
}

func TestRunServesGitModeOnHEADChange(t *testing.T) {
	repo := t.TempDir()
	runServerTestGit(t, repo, "init", "-b", "main")
	writeServerTestFile(t, repo, "README.md", "# First\n\nInitial body.\n")
	runServerTestGit(t, repo, "add", ".")
	runServerTestGit(t, repo, "commit", "-m", "initial")

	addr := freeAddr(t)
	var builtDirs safeDirs
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := runServerInDir(t, ctx, repo, Options{Addr: addr, Worktree: false}, testRebuildMode(repo, false, &builtDirs, nil))

	baseURL := "http://" + addr
	waitForHTTP(t, baseURL+"/view/README.md/", func(resp *http.Response, body string) error {
		if resp.StatusCode != http.StatusOK {
			return errors.New(resp.Status)
		}
		if !strings.Contains(body, "First") {
			return errors.New("browser page does not contain initial content")
		}
		return nil
	})

	// git 模式按 HEAD 轮询：提交后应触发重建（2s 轮询 + 300ms 防抖，
	// waitForHTTP 的 10s 轮询预算覆盖）。
	writeServerTestFile(t, repo, "README.md", "# Second\n\nUpdated body.\n")
	runServerTestGit(t, repo, "add", ".")
	runServerTestGit(t, repo, "commit", "-m", "update")
	waitForHTTP(t, baseURL+"/view/README.md/", func(resp *http.Response, body string) error {
		if resp.StatusCode != http.StatusOK {
			return errors.New(resp.Status)
		}
		if !strings.Contains(body, "Second") {
			return errors.New("browser page does not contain rebuilt content")
		}
		return nil
	})

	cancel()
	if err := waitServerDone(done); err != nil {
		t.Fatalf("server returned error: %v", err)
	}
	waitPortAvailable(t, addr)
	for _, dir := range builtDirs.snapshot() {
		if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("built directory %s still exists after shutdown: %v", dir, err)
		}
	}
}

func runServerTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

type safeDirs struct {
	mu   sync.Mutex
	dirs []string
}

func (d *safeDirs) append(dir string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dirs = append(d.dirs, dir)
}

func (d *safeDirs) snapshot() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]string, len(d.dirs))
	copy(out, d.dirs)
	return out
}

func testRebuild(repo string, builtDirs *safeDirs, beforeBuild func() error) func(context.Context) (string, error) {
	return testRebuildMode(repo, true, builtDirs, beforeBuild)
}

func testRebuildMode(repo string, worktree bool, builtDirs *safeDirs, beforeBuild func() error) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		if beforeBuild != nil {
			if err := beforeBuild(); err != nil {
				return "", err
			}
		}
		tree, err := source.Open(ctx, source.Spec{Repo: repo, Worktree: worktree})
		if err != nil {
			return "", err
		}
		defer tree.Cleanup()

		cfg, _, err := config.Load(tree.Root, "", config.Flags{Repo: repo})
		if err != nil {
			return "", err
		}
		renderer, err := theme.New("", "", cfg.Theme.Vars)
		if err != nil {
			return "", err
		}

		outDir, err := os.MkdirTemp("", "repolens-server-test-*")
		if err != nil {
			return "", err
		}
		// site.Build 拒绝清空无哨兵的已存在目录：删掉空壳让它自建。
		if err := os.Remove(outDir); err != nil {
			return "", err
		}
		keep := false
		defer func() {
			if !keep {
				_ = os.RemoveAll(outDir)
			}
		}()
		if _, err := site.NewBuilder(cfg, renderer).Build(ctx, tree, outDir); err != nil {
			return "", err
		}
		keep = true
		builtDirs.append(outDir)
		return outDir, nil
	}
}

// runServerInDir must not be used from parallel tests: it calls os.Chdir,
// which mutates process-global state.
func runServerInDir(t *testing.T, ctx context.Context, dir string, opts Options, rebuild func(context.Context) (string, error)) <-chan error {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%s): %v", dir, err)
	}

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, opts, rebuild)
	}()
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})
	return done
}

func waitForHTTP(t *testing.T, url string, check func(*http.Response, string) error) {
	t.Helper()
	client := &http.Client{Timeout: 500 * time.Millisecond}
	waitUntil(t, func() error {
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return check(resp, string(body))
	})
}

func waitUntil(t *testing.T, check func() error) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		if err := check(); err != nil {
			lastErr = err
			time.Sleep(50 * time.Millisecond)
			continue
		}
		return
	}
	t.Fatalf("condition not met before timeout: %v", lastErr)
}

func waitServerDone(done <-chan error) error {
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		return errors.New("server did not stop before timeout")
	}
}

func waitPortAvailable(t *testing.T, addr string) {
	t.Helper()
	waitUntil(t, func() error {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		return listener.Close()
	})
}

func freeAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skipf("local TCP listen is not permitted in this environment: %v", err)
		}
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func writeServerTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", p, err)
	}
}
