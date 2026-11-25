package desktop

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/ConteMan/repolens/internal/config"
)

// Session holds the rule draft and its configuration baseline for a worktree.
// Callers own edits to Rules and must synchronize them with session methods.
type Session struct {
	Root  string
	Rules []config.RepositoryRuleSettings

	mu        sync.Mutex
	baseline  *config.RepositoryDocument
	candidate *config.RepositoryDocument
}

// Open starts a desktop session at the root of a local Git worktree.
func Open(path string) (*Session, error) {
	if path == "" {
		return nil, errors.New("choose a Git worktree root directory")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dir, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fmt.Errorf("resolve selected directory: %w", err)
	}
	selected, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !selected.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", dir)
	}
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	cmd.Env = slices.DeleteFunc(cmd.Environ(), func(entry string) bool {
		key, _, _ := strings.Cut(entry, "=")
		return key == "GIT_DIR" || key == "GIT_WORK_TREE" || key == "GIT_COMMON_DIR"
	})
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%q must be the root of a local Git worktree: %w", dir, err)
	}
	gitRoot := strings.TrimSuffix(string(output), "\n")
	rootInfo, err := os.Stat(gitRoot)
	if err != nil {
		return nil, fmt.Errorf("inspect Git worktree root: %w", err)
	}
	if !os.SameFile(selected, rootInfo) {
		return nil, fmt.Errorf("select the Git worktree root %q", gitRoot)
	}
	session := &Session{Root: dir}
	if err := session.Reload(); err != nil {
		return nil, err
	}
	return session, nil
}

// Dirty reports whether the draft rules differ from the saved rules.
func (s *Session) Dirty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dirty()
}

func (s *Session) dirty() bool {
	return !slices.EqualFunc(s.Rules, documentRules(s.baseline), func(a, b config.RepositoryRuleSettings) bool {
		return reflect.DeepEqual(a, b)
	})
}

// Prepare validates the draft and captures a candidate tied to this revision.
func (s *Session) Prepare() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.candidate = nil

	before, err := rulesPreview(s.baseline)
	if err != nil {
		return "", err
	}
	next, err := s.baseline.Clone()
	if err != nil {
		return "", err
	}
	if s.dirty() {
		// A nil draft after deleting the last rule still means explicit [].
		rules := append([]config.RepositoryRuleSettings{}, s.Rules...)
		if err := next.ReplaceRules(&rules); err != nil {
			return "", err
		}
	} else if issues := config.ValidateRepositorySettings(next.Settings); len(issues) > 0 {
		return "", &config.RepositoryValidationError{Issues: issues}
	}
	after, err := rulesPreview(next)
	if err != nil {
		return "", err
	}
	s.candidate = next
	return config.RepositorySettingsDiff(before, after, next.Path), nil
}

// CancelSave drops the prepared candidate while retaining the draft.
func (s *Session) CancelSave() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.candidate = nil
}

// Save consumes one confirmation and commits its candidate if still current.
func (s *Session) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	candidate := s.candidate
	s.candidate = nil
	if candidate == nil || filepath.Dir(candidate.Path) != s.Root ||
		!slices.EqualFunc(s.Rules, documentRules(candidate), func(a, b config.RepositoryRuleSettings) bool {
			return reflect.DeepEqual(a, b)
		}) {
		return errors.New("prepare the current draft before confirming save")
	}
	if err := candidate.CheckRevision(); err != nil {
		return err
	}
	if !s.dirty() {
		return nil
	}
	if err := candidate.Write(); err != nil {
		return err
	}

	// ReplaceRules decoded its own snapshot, independent of the draft pointers.
	s.baseline = candidate
	return nil
}

// Reload replaces the baseline and draft only after the file loads successfully.
func (s *Session) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	baseline, err := config.LoadRepositoryDocument(s.Root)
	if err != nil {
		return err
	}
	draft, err := baseline.Clone()
	if err != nil {
		return err
	}
	s.baseline = baseline
	s.Rules = documentRules(draft)
	s.candidate = nil
	return nil
}

func documentRules(document *config.RepositoryDocument) []config.RepositoryRuleSettings {
	if document.Settings.Rules == nil {
		return nil
	}
	return *document.Settings.Rules
}

func rulesPreview(document *config.RepositoryDocument) (string, error) {
	data, err := document.YAML()
	if err != nil {
		return "", err
	}
	return config.RepositorySettingsPreview(data, "rules")
}
