package source

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const logMaxCount = 10000

var scpLikeURL = regexp.MustCompile(`^[^@]+@[^:]+:.+`)

// Spec identifies the repository content source to materialize.
type Spec struct {
	Repo     string // URL or local path.
	Ref      string // Empty means HEAD.
	Worktree bool
}

// Commit is the last Git commit known to have modified a file.
type Commit struct {
	Hash    string
	Time    time.Time
	Subject string
}

// File is a regular repository file in the materialized tree.
type File struct {
	Path       string // Repository-relative slash-separated path.
	Size       int64
	LastCommit *Commit
}

// Tree is the materialized repository content set.
type Tree struct {
	Root       string
	CommitHash string
	Files      []File

	cleanupRoot string
}

// Open materializes a deterministic repository content set. Callers must call
// Cleanup when the tree is no longer needed.
func Open(ctx context.Context, spec Spec) (*Tree, error) {
	if strings.TrimSpace(spec.Repo) == "" {
		return nil, sourceErr(errors.New("repo is required"))
	}
	if spec.Worktree {
		return openWorktree(ctx, spec.Repo)
	}

	repoDir, remote, err := prepareRepository(ctx, spec.Repo)
	if err != nil {
		return nil, sourceErr(err)
	}

	commitHash, err := resolveCommit(ctx, repoDir, spec.Repo, spec.Ref, remote)
	if err != nil {
		return nil, sourceErr(err)
	}

	root, err := os.MkdirTemp("", "repolens-source-*")
	if err != nil {
		return nil, sourceErr(err)
	}
	tree := &Tree{Root: root, CommitHash: commitHash, cleanupRoot: root}

	if err := materializeArchive(ctx, repoDir, commitHash, root); err != nil {
		_ = tree.Cleanup()
		return nil, sourceErr(err)
	}

	files, err := listFiles(root, false)
	if err != nil {
		_ = tree.Cleanup()
		return nil, sourceErr(err)
	}

	commits, err := lastCommitMap(ctx, repoDir, commitHash)
	if err != nil {
		_ = tree.Cleanup()
		return nil, sourceErr(err)
	}
	attachLastCommits(files, commits)
	tree.Files = files

	return tree, nil
}

// Cleanup removes temporary materialized content owned by this Tree.
func (t *Tree) Cleanup() error {
	if t == nil || t.cleanupRoot == "" {
		return nil
	}
	err := os.RemoveAll(t.cleanupRoot)
	t.cleanupRoot = ""
	return err
}

func openWorktree(ctx context.Context, repo string) (*Tree, error) {
	root, err := filepath.Abs(repo)
	if err != nil {
		return nil, sourceErr(err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, sourceErr(err)
	}
	if !info.IsDir() {
		return nil, sourceErr(fmt.Errorf("%s is not a directory", repo))
	}

	files, err := listFiles(root, true)
	if err != nil {
		return nil, sourceErr(err)
	}

	if commitHash, err := revParseCommit(ctx, root, "HEAD"); err == nil {
		if commits, err := lastCommitMap(ctx, root, commitHash); err == nil {
			attachLastCommits(files, commits)
		}
	}

	return &Tree{Root: root, Files: files}, nil
}

func prepareRepository(ctx context.Context, repo string) (string, bool, error) {
	if !isRemote(repo) {
		abs, err := filepath.Abs(repo)
		if err != nil {
			return "", false, err
		}
		return abs, false, nil
	}

	cacheDir, err := cachedRepoPath(repo)
	if err != nil {
		return "", true, err
	}
	if err := os.MkdirAll(filepath.Dir(cacheDir), 0o755); err != nil {
		return "", true, err
	}

	if _, err := os.Stat(cacheDir); err == nil {
		if err := runGit(ctx, cacheDir, "fetch", "--prune", "origin", "+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*"); err != nil {
			return "", true, err
		}
		return cacheDir, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", true, err
	}

	tmp, err := os.MkdirTemp(filepath.Dir(cacheDir), "clone-*")
	if err != nil {
		return "", true, err
	}
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.RemoveAll(tmp)
		}
	}()

	if err := runGit(ctx, "", "clone", "--bare", repo, tmp); err != nil {
		return "", true, err
	}
	if err := os.Rename(tmp, cacheDir); err != nil {
		return "", true, err
	}
	removeTmp = false

	return cacheDir, true, nil
}

func resolveCommit(ctx context.Context, repoDir, repo, ref string, remote bool) (string, error) {
	if ref != "" {
		return revParseCommit(ctx, repoDir, ref)
	}
	if remote {
		hash, err := lsRemoteHead(ctx, repo)
		if err != nil {
			return "", err
		}
		return revParseCommit(ctx, repoDir, hash)
	}
	return revParseCommit(ctx, repoDir, "HEAD")
}

func materializeArchive(ctx context.Context, repoDir, commitHash, root string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "archive", "--format=tar", commitHash)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return commandErr("git archive", stderr.String(), err)
	}

	extractErr := extractTar(stdout, root)
	if extractErr != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	waitErr := cmd.Wait()
	if waitErr != nil && extractErr == nil {
		return commandErr("git archive", stderr.String(), waitErr)
	}
	if extractErr != nil {
		return extractErr
	}
	return nil
}

func extractTar(r io.Reader, root string) error {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		name, err := safeArchivePath(header.Name)
		if err != nil {
			return err
		}
		if name == "." {
			continue
		}
		target := filepath.Join(root, name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := header.FileInfo().Mode().Perm()
			if mode == 0 {
				mode = 0o644
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(file, tr)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			continue
		}
	}
}

func safeArchivePath(name string) (string, error) {
	if name == "" {
		return ".", nil
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe archive path %q", name)
	}
	return clean, nil
}

func listFiles(root string, skipGitDir bool) ([]File, error) {
	var files []File
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if skipGitDir && entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, File{
			Path: filepath.ToSlash(rel),
			Size: info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func lastCommitMap(ctx context.Context, repoDir, commitHash string) (map[string]*Commit, error) {
	out, err := runGitOutput(ctx, repoDir,
		"log",
		fmt.Sprintf("--max-count=%d", logMaxCount),
		"--name-status",
		"--find-renames",
		"-z",
		"--format=%x1e%H%x00%cI%x00%s%x00",
		commitHash,
		"--",
	)
	if err != nil {
		return nil, err
	}
	return parseNameStatusLog(out)
}

func parseNameStatusLog(out []byte) (map[string]*Commit, error) {
	commits := make(map[string]*Commit)
	records := bytes.Split(out, []byte{0x1e})
	for _, record := range records {
		record = bytes.Trim(record, "\x00\n\r")
		if len(record) == 0 {
			continue
		}
		fields := bytes.Split(record, []byte{0})
		if len(fields) < 3 {
			continue
		}

		hash := string(bytes.Trim(fields[0], "\n\r"))
		commitTime, err := time.Parse(time.RFC3339, string(fields[1]))
		if err != nil {
			return nil, err
		}
		commit := Commit{
			Hash:    hash,
			Time:    commitTime,
			Subject: string(fields[2]),
		}

		for i := 3; i < len(fields); {
			status := strings.Trim(string(fields[i]), "\n\r")
			i++
			if status == "" {
				continue
			}

			var path string
			switch status[0] {
			case 'R':
				if i+1 >= len(fields) {
					continue
				}
				path = cleanLogPath(fields[i+1])
				i += 2
			case 'C':
				if i+1 >= len(fields) {
					continue
				}
				path = cleanLogPath(fields[i+1])
				i += 2
			default:
				if i >= len(fields) {
					continue
				}
				path = cleanLogPath(fields[i])
				i++
			}
			if path == "" {
				continue
			}
			if _, exists := commits[path]; !exists {
				entry := commit
				commits[path] = &entry
			}
		}
	}
	return commits, nil
}

func cleanLogPath(path []byte) string {
	return filepath.ToSlash(strings.Trim(string(path), "\n\r"))
}

func attachLastCommits(files []File, commits map[string]*Commit) {
	for i := range files {
		if commit, ok := commits[files[i].Path]; ok {
			files[i].LastCommit = commit
		}
	}
}

func cachedRepoPath(repo string) (string, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(repo))
	key := hex.EncodeToString(sum[:])[:16]
	return filepath.Join(cacheRoot, "repolens", "repos", key), nil
}

func isRemote(repo string) bool {
	if scpLikeURL.MatchString(repo) {
		return true
	}
	parsed, err := url.Parse(repo)
	if err != nil || parsed.Scheme == "" {
		return false
	}
	switch parsed.Scheme {
	case "file", "git", "http", "https", "ssh":
		return true
	default:
		return false
	}
}

func lsRemoteHead(ctx context.Context, repo string) (string, error) {
	out, err := runGitOutput(ctx, "", "ls-remote", "--symref", repo, "HEAD")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == "HEAD" && parts[0] != "ref:" {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("git ls-remote did not return HEAD for %s", repo)
}

func revParseCommit(ctx context.Context, repoDir, ref string) (string, error) {
	return runGitString(ctx, repoDir, "rev-parse", "--verify", ref+"^{commit}")
}

func runGitString(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := runGitOutput(ctx, dir, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func runGit(ctx context.Context, dir string, args ...string) error {
	_, err := runGitOutput(ctx, dir, args...)
	return err
}

func runGitOutput(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, commandErr("git "+strings.Join(args, " "), stderr.String(), err)
	}
	return out, nil
}

func commandErr(command, stderr string, err error) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return fmt.Errorf("%s: %w", command, err)
	}
	return fmt.Errorf("%s: %s: %w", command, stderr, err)
}

func sourceErr(err error) error {
	if err == nil {
		return nil
	}
	if strings.HasPrefix(err.Error(), "source: ") {
		return err
	}
	return fmt.Errorf("source: %w", err)
}
