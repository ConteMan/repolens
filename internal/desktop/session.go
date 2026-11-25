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

var errNotGitWorktree = errors.New("select a local Git worktree root")

// Session holds editable configuration and its saved baselines.
// Callers own draft edits and must synchronize them with session methods.
type Session struct {
	Root             string
	Rules            []config.RepositoryRuleSettings
	Settings         config.RepositorySettings
	BuilderPath      string
	Builder          config.BuilderSettings
	ExternalSettings config.RepositorySettings
	ExternalRules    []config.RepositoryRuleSettings

	mu               sync.Mutex
	baseline         *config.RepositoryDocument
	builderBaseline  *config.BuilderDocument
	builderPath      string
	candidate        *config.RepositoryDocument
	builderCandidate *config.BuilderDocument
	prepared         bool
}

// ConfigChange describes one file and the editable fields in its preview.
type ConfigChange struct {
	Path                          string
	Before, After, Diff           string
	BeforeSettings, AfterSettings config.RepositorySettings
	BeforeBuilder, AfterBuilder   config.BuilderSettings
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
		return nil, fmt.Errorf("%q: %w: %w", dir, errNotGitWorktree, err)
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

// Dirty reports whether either draft differs from its saved settings.
func (s *Session) Dirty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repositoryDirty() || s.builderDirty()
}

func (s *Session) repositoryDirty() bool {
	return !sameRules(s.Rules, documentRules(s.baseline)) ||
		!reflect.DeepEqual(siteSettings(s.Settings), siteSettings(s.baseline.Settings))
}

func (s *Session) builderDirty() bool {
	return s.builderBaseline != nil && (!reflect.DeepEqual(s.Builder, s.builderBaseline.Settings) || !reflect.DeepEqual(siteSettings(s.ExternalSettings), siteSettings(s.builderBaseline.Shared)) || !sameRules(s.ExternalRules, settingsRules(s.builderBaseline.Shared)))
}

// BuilderDirty reports whether the selected external file has draft edits.
func (s *Session) BuilderDirty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.builderDirty()
}

func siteSettings(settings config.RepositorySettings) config.RepositorySettings {
	settings.Rules = nil
	return settings
}

func sameRules(a, b []config.RepositoryRuleSettings) bool {
	return slices.EqualFunc(a, b, func(a, b config.RepositoryRuleSettings) bool {
		return reflect.DeepEqual(a, b)
	})
}

// Prepare validates both drafts and returns their textual differences.
func (s *Session) Prepare() (string, error) {
	changes, err := s.PrepareChanges()
	if err != nil {
		return "", err
	}
	var diff strings.Builder
	for _, change := range changes {
		diff.WriteString(change.Diff)
	}
	return diff.String(), nil
}

// PrepareChanges captures all changed files and binds them to a confirmation.
func (s *Session) PrepareChanges() ([]ConfigChange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearPrepared()

	repository, changes, err := s.prepareRepository()
	if err != nil {
		return nil, err
	}
	var builder *config.BuilderDocument
	if s.builderDirty() {
		builder, err = s.builderBaseline.Clone()
		if err != nil {
			return nil, err
		}
		if err := builder.Replace(s.Builder); err != nil {
			return nil, err
		}
		shared := siteSettings(s.ExternalSettings)
		shared.Rules = s.builderBaseline.Shared.Rules
		if !sameRules(s.ExternalRules, settingsRules(s.builderBaseline.Shared)) {
			rules := append([]config.RepositoryRuleSettings{}, s.ExternalRules...)
			shared.Rules = &rules
		}
		if !reflect.DeepEqual(shared, s.builderBaseline.Shared) {
			if err := builder.ReplaceShared(shared); err != nil {
				return nil, &ExternalValidationError{Err: err}
			}
		}
		before, err := s.builderBaseline.Preview()
		if err != nil {
			return nil, err
		}
		after, err := builder.Preview()
		if err != nil {
			return nil, err
		}
		changes = append(changes, ConfigChange{
			Path: s.builderPath, Before: before, After: after,
			Diff:          config.RepositorySettingsDiff(before, after, s.builderPath),
			BeforeBuilder: s.builderBaseline.Settings, AfterBuilder: builder.Settings,
			BeforeSettings: s.builderBaseline.Shared, AfterSettings: builder.Shared,
		})
	}
	s.candidate, s.builderCandidate, s.prepared = repository, builder, true
	return changes, nil
}

func (s *Session) prepareRepository() (*config.RepositoryDocument, []ConfigChange, error) {
	settings := siteSettings(s.Settings)
	settings.Rules = s.baseline.Settings.Rules
	rulesChanged := !sameRules(s.Rules, documentRules(s.baseline))
	if rulesChanged {
		// Deleting the last rule still means explicit [].
		rules := append([]config.RepositoryRuleSettings{}, s.Rules...)
		settings.Rules = &rules
	}
	if issues := config.ValidateRepositorySettings(settings); len(issues) > 0 {
		return nil, nil, &config.RepositoryValidationError{Issues: issues}
	}
	siteChanged := !reflect.DeepEqual(siteSettings(settings), siteSettings(s.baseline.Settings))
	if !rulesChanged && !siteChanged {
		return nil, nil, nil
	}
	next, err := s.baseline.Clone()
	if err != nil {
		return nil, nil, err
	}
	keys := []string{"rules"}
	if siteChanged {
		keys = append(keys, "site", "ignore", "render", "glossary", "theme", "view", "agent")
		if !rulesChanged {
			settings.Rules = nil
		}
		err = next.ReplaceSiteSettings(settings)
	} else {
		err = next.ReplaceRules(settings.Rules)
	}
	if err != nil {
		return nil, nil, err
	}
	before, err := repositoryPreview(s.baseline, keys)
	if err != nil {
		return nil, nil, err
	}
	after, err := repositoryPreview(next, keys)
	if err != nil {
		return nil, nil, err
	}
	return next, []ConfigChange{{
		Path: next.Path, Before: before, After: after,
		Diff:           config.RepositorySettingsDiff(before, after, next.Path),
		BeforeSettings: s.baseline.Settings, AfterSettings: next.Settings,
	}}, nil
}

// CancelSave drops the prepared candidates while retaining the drafts.
func (s *Session) CancelSave() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearPrepared()
}

func (s *Session) clearPrepared() {
	s.candidate, s.builderCandidate, s.prepared = nil, nil, false
}

// Save consumes one confirmation. Both revisions are checked before any write.
// If a later write fails, the successful file becomes the new baseline and the
// other draft remains available for a subsequent confirmation.
func (s *Session) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	repository, builder, prepared := s.candidate, s.builderCandidate, s.prepared
	s.clearPrepared()
	if !prepared || s.Root != filepath.Dir(s.baseline.Path) || s.BuilderPath != s.builderPath ||
		!s.matchesConfirmation(repository, builder) {
		return errors.New("prepare the current draft before confirming save")
	}
	if repository != nil {
		if err := repository.CheckRevision(); err != nil {
			return err
		}
	}
	if builder != nil {
		path, err := externalConfigPath(s.baseline.Path, s.builderPath)
		if err != nil {
			return err
		}
		if path != s.builderPath {
			return fmt.Errorf("external config %q: %w", s.builderPath, config.ErrRevisionConflict)
		}
		if err := builder.CheckRevision(); err != nil {
			return err
		}
	}
	if repository != nil {
		if err := repository.Write(); err != nil {
			return err
		}
		s.baseline = repository
	}
	if builder != nil {
		if err := builder.Write(); err != nil {
			if repository != nil {
				return fmt.Errorf("saved %q, but failed to save external configuration: %w", repository.Path, err)
			}
			return err
		}
		s.builderBaseline = builder
	}
	return nil
}

func (s *Session) matchesConfirmation(repository *config.RepositoryDocument, builder *config.BuilderDocument) bool {
	if repository == nil {
		if s.repositoryDirty() {
			return false
		}
	} else if !sameRules(s.Rules, documentRules(repository)) ||
		!reflect.DeepEqual(siteSettings(s.Settings), siteSettings(repository.Settings)) {
		return false
	}
	if builder == nil {
		return !s.builderDirty()
	}
	return reflect.DeepEqual(s.Builder, builder.Settings) && reflect.DeepEqual(siteSettings(s.ExternalSettings), siteSettings(builder.Shared)) && sameRules(s.ExternalRules, settingsRules(builder.Shared))
}

// LoadBuilder selects an existing trusted external configuration file.
func (s *Session) LoadBuilder(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resolved, err := externalConfigPath(s.baseline.Path, path)
	if err != nil {
		return err
	}
	baseline, draft, err := loadBuilderDraft(resolved)
	if err != nil {
		return err
	}
	s.builderBaseline = baseline
	s.Builder = draft.Settings
	s.ExternalSettings = siteSettings(draft.Shared)
	s.ExternalRules = settingsRules(draft.Shared)
	s.builderPath, s.BuilderPath = resolved, resolved
	s.clearPrepared()
	return nil
}

func externalConfigPath(repositoryPath, path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%q is not a regular configuration file", path)
	}
	repository, err := os.Stat(repositoryPath)
	if err == nil && os.SameFile(info, repository) {
		return "", errors.New("the repository configuration cannot be selected as trusted external configuration")
	}
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return resolved, nil
}

func loadBuilderDraft(path string) (*config.BuilderDocument, *config.BuilderDocument, error) {
	baseline, err := config.LoadBuilderDocument(path)
	if err != nil {
		return nil, nil, err
	}
	draft, err := baseline.Clone()
	if err != nil {
		return nil, nil, err
	}
	return baseline, draft, nil
}

// Reload replaces drafts and baselines only after all selected files load.
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
	if s.builderPath != "" {
		path, err := externalConfigPath(baseline.Path, s.builderPath)
		if err != nil {
			return err
		}
		if path != s.builderPath {
			return fmt.Errorf("external config %q: %w", s.builderPath, config.ErrRevisionConflict)
		}
		builder, builderDraft, err := loadBuilderDraft(path)
		if err != nil {
			return err
		}
		s.builderBaseline, s.Builder = builder, builderDraft.Settings
		s.ExternalSettings = siteSettings(builderDraft.Shared)
		s.ExternalRules = settingsRules(builderDraft.Shared)
	}
	s.baseline = baseline
	s.Settings = siteSettings(draft.Settings)
	s.Rules = documentRules(draft)
	s.clearPrepared()
	return nil
}

type ExternalValidationError struct{ Err error }

func (e *ExternalValidationError) Error() string { return e.Err.Error() }
func (e *ExternalValidationError) Unwrap() error { return e.Err }

func documentRules(document *config.RepositoryDocument) []config.RepositoryRuleSettings {
	return settingsRules(document.Settings)
}

func settingsRules(settings config.RepositorySettings) []config.RepositoryRuleSettings {
	if settings.Rules == nil {
		return nil
	}
	return *settings.Rules
}

func repositoryPreview(document *config.RepositoryDocument, keys []string) (string, error) {
	data, err := document.YAML()
	if err != nil {
		return "", err
	}
	return config.RepositorySettingsPreview(data, keys...)
}
