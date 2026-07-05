package server

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/fsnotify/fsnotify"
)

const (
	defaultAddr     = "127.0.0.1:8788"
	rebuildDebounce = 300 * time.Millisecond
	gitPollInterval = 2 * time.Second
)

// Options configures the local preview server.
type Options struct {
	Addr     string
	Worktree bool
}

func init() {
	_ = mime.AddExtensionType(".md", "text/markdown; charset=utf-8")
	_ = mime.AddExtensionType(".markdown", "text/markdown; charset=utf-8")
}

// Run blocks until ctx is canceled. rebuild must build the site into a fresh
// directory and return that directory. Successful directories are removed on
// exit.
func Run(ctx context.Context, opts Options, rebuild func(ctx context.Context) (dir string, err error)) error {
	if rebuild == nil {
		return errors.New("server: rebuild function is required")
	}
	addr := opts.Addr
	if strings.TrimSpace(addr) == "" {
		addr = defaultAddr
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 先绑端口再做首次构建：端口被占用时立即报错，
	// 不让用户等一次完整构建才发现。
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	initialRoot, err := rebuild(runCtx)
	if err != nil {
		_ = listener.Close()
		return err
	}
	if initialRoot == "" {
		_ = listener.Close()
		return errors.New("server: rebuild returned an empty directory")
	}

	var rootsMu sync.Mutex
	roots := []string{initialRoot}
	defer func() {
		rootsMu.Lock()
		defer rootsMu.Unlock()
		for _, root := range roots {
			_ = os.RemoveAll(root)
		}
	}()
	rootsSnapshot := func() []string {
		rootsMu.Lock()
		defer rootsMu.Unlock()
		out := make([]string, len(roots))
		copy(out, roots)
		return out
	}

	var currentRoot atomic.Value
	currentRoot.Store(initialRoot)

	requestRebuild := make(chan struct{}, 1)
	var wg sync.WaitGroup
	if opts.Worktree {
		cwd, err := os.Getwd()
		if err != nil {
			_ = listener.Close()
			return err
		}
		watcher, err := newWorktreeWatcher(cwd, rootsSnapshot)
		if err != nil {
			_ = listener.Close()
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer watcher.Close()
			watcher.run(runCtx, requestRebuild)
		}()
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			_ = listener.Close()
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			pollGitHEAD(runCtx, cwd, requestRebuild)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		runRebuildLoop(runCtx, requestRebuild, rebuild, func(root string) {
			rootsMu.Lock()
			roots = append(roots, root)
			// 只保留当前与上一个站点根（在途请求可能仍在读上一个），
			// 更早的立即删除，避免长 session 下临时目录累积。
			for len(roots) > 2 {
				_ = os.RemoveAll(roots[0])
				roots = roots[1:]
			}
			rootsMu.Unlock()
			currentRoot.Store(root)
		})
	}()

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			root, _ := currentRoot.Load().(string)
			http.FileServer(http.Dir(root)).ServeHTTP(w, r)
		}),
	}

	serveErr := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stdout, "Serving at %s\n", displayURL(listener.Addr()))
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErr <- err
	}()

	select {
	case <-ctx.Done():
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
		}
		if err := <-serveErr; err != nil {
			wg.Wait()
			return err
		}
		wg.Wait()
		return nil
	case err := <-serveErr:
		cancel()
		wg.Wait()
		return err
	}
}

func runRebuildLoop(ctx context.Context, requests <-chan struct{}, rebuild func(context.Context) (string, error), switchRoot func(string)) {
	var timer *time.Timer
	var timerC <-chan time.Time
	stopTimer := func() {
		if timer == nil {
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer = nil
		timerC = nil
	}
	resetTimer := func() {
		if timer == nil {
			timer = time.NewTimer(rebuildDebounce)
			timerC = timer.C
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(rebuildDebounce)
	}

	defer stopTimer()
	for {
		select {
		case <-ctx.Done():
			return
		case <-requests:
			resetTimer()
		case <-timerC:
			timer = nil
			timerC = nil
			if err := ctx.Err(); err != nil {
				return
			}
			root, err := rebuild(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Rebuild failed: %v\n", err)
				continue
			}
			if root == "" {
				fmt.Fprintln(os.Stderr, "Rebuild failed: server: rebuild returned an empty directory")
				continue
			}
			switchRoot(root)
		}
	}
}

func signalRebuild(ch chan<- struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

type worktreeWatcher struct {
	root          string
	watcher       *fsnotify.Watcher
	outputRoots   func() []string
	watchedDirsMu sync.Mutex
	watchedDirs   map[string]bool
	// cfg 缓存仓库配置供 ignored() 使用，.repolens.yml 变化时重载，
	// 避免每个文件事件都做一次磁盘上的 config.Load。
	cfgMu sync.Mutex
	cfg   *config.Config
}

func newWorktreeWatcher(root string, outputRoots func() []string) (*worktreeWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &worktreeWatcher{
		root:        root,
		watcher:     watcher,
		outputRoots: outputRoots,
		watchedDirs: make(map[string]bool),
	}
	w.reloadConfig()
	if err := w.addRecursive(root); err != nil {
		_ = watcher.Close()
		return nil, err
	}
	return w, nil
}

func (w *worktreeWatcher) Close() error {
	return w.watcher.Close()
}

func (w *worktreeWatcher) run(ctx context.Context, rebuilds chan<- struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op == fsnotify.Chmod {
				continue
			}
			if filepath.Base(event.Name) == ".repolens.yml" {
				w.reloadConfig()
			}
			if w.ignored(event.Name) {
				continue
			}
			if event.Has(fsnotify.Create) {
				w.addIfDir(event.Name)
			}
			signalRebuild(rebuilds)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
		}
	}
}

func (w *worktreeWatcher) addIfDir(name string) {
	info, err := os.Stat(name)
	if err != nil || !info.IsDir() {
		return
	}
	if err := w.addRecursive(name); err != nil {
		fmt.Fprintf(os.Stderr, "Watcher error: %v\n", err)
	}
}

func (w *worktreeWatcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		if w.ignored(path) {
			return filepath.SkipDir
		}
		w.watchedDirsMu.Lock()
		already := w.watchedDirs[path]
		if !already {
			w.watchedDirs[path] = true
		}
		w.watchedDirsMu.Unlock()
		if already {
			return nil
		}
		return w.watcher.Add(path)
	})
}

func (w *worktreeWatcher) ignored(name string) bool {
	rel, err := filepath.Rel(w.root, name)
	if err == nil {
		rel = filepath.Clean(rel)
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) {
			return true
		}
		if rel != ".repolens.yml" {
			w.cfgMu.Lock()
			cfg := w.cfg
			w.cfgMu.Unlock()
			if cfg != nil && cfg.Ignored(filepath.ToSlash(rel)) {
				return true
			}
		}
	}
	for _, root := range w.outputRoots() {
		if root != "" && pathWithin(name, root) {
			return true
		}
	}
	return false
}

func (w *worktreeWatcher) reloadConfig() {
	cfg, _, err := config.Load(w.root, "", config.Flags{})
	if err != nil {
		return
	}
	w.cfgMu.Lock()
	w.cfg = cfg
	w.cfgMu.Unlock()
}

func pathWithin(name, root string) bool {
	nameAbs, err := filepath.Abs(name)
	if err != nil {
		return false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, nameAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func pollGitHEAD(ctx context.Context, dir string, rebuilds chan<- struct{}) {
	last, err := gitHEAD(ctx, dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Git poll failed: %v\n", err)
	}
	ticker := time.NewTicker(gitPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			head, err := gitHEAD(ctx, dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Git poll failed: %v\n", err)
				continue
			}
			if last != "" && head != last {
				signalRebuild(rebuilds)
			}
			last = head
		}
	}
}

func gitHEAD(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--verify", "HEAD^{commit}")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func displayURL(addr net.Addr) string {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "http://" + addr.String()
	}
	if host == "" || host == "::" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/"
}
