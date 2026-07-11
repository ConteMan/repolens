package ui

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/ConteMan/repolens/internal/site"
	"github.com/ConteMan/repolens/internal/source"
	"github.com/ConteMan/repolens/internal/theme"
)

var errBuildInProgress = errors.New("a build is already running for this repository")

type buildService struct {
	ctx          context.Context
	cacheRoot    func() (string, error)
	mu           sync.RWMutex
	operations   map[string]*buildOperation
	repositories map[string]string
}

type buildOperation struct {
	ID         string
	Repository string
	Stage      string
	Stats      *site.Stats
	Warnings   []string
	Error      string
	OutputPath string
}

func newBuildService(ctx context.Context) *buildService {
	return &buildService{
		ctx:          ctx,
		cacheRoot:    defaultBuildCacheRoot,
		operations:   make(map[string]*buildOperation),
		repositories: make(map[string]string),
	}
}

func defaultBuildCacheRoot() (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find user cache directory: %w", err)
	}
	return filepath.Join(root, "repolens", "ui", "builds"), nil
}

func (s *buildService) start(repository string) (buildOperation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.repositories[repository]; ok {
		return buildOperation{}, errBuildInProgress
	}
	outputPath, err := s.outputPath(repository)
	if err != nil {
		return buildOperation{}, err
	}
	id, err := newBuildID()
	if err != nil {
		return buildOperation{}, err
	}
	op := &buildOperation{ID: id, Repository: repository, Stage: "opening", OutputPath: outputPath}
	s.operations[id] = op
	s.repositories[repository] = id
	go s.run(id, repository)
	return cloneBuildOperation(op), nil
}

func (s *buildService) operation(id string) (buildOperation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	op, ok := s.operations[id]
	if !ok {
		return buildOperation{}, false
	}
	return cloneBuildOperation(op), true
}

func (s *buildService) run(id, repository string) {
	defer s.release(repository)
	if err := s.build(id, repository); err != nil {
		s.fail(id, err)
	}
}

func (s *buildService) build(id, repository string) error {
	s.setStage(id, "opening")
	tree, err := source.Open(s.ctx, source.Spec{Repo: repository, Worktree: true})
	if err != nil {
		return fmt.Errorf("open working tree: %w", err)
	}
	defer tree.Cleanup()

	s.setStage(id, "loading_config")
	cfg, warnings, err := config.Load(tree.Root, "", config.Flags{})
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	s.setWarnings(id, warnings)

	s.setStage(id, "loading_theme")
	renderer, err := theme.New(
		resolveBuildSourcePath(tree.Root, cfg.Theme.Templates),
		resolveBuildSourcePath(tree.Root, cfg.Theme.CSS),
		cfg.Theme.Vars,
	)
	if err != nil {
		return fmt.Errorf("load theme: %w", err)
	}

	outputPath, err := s.outputPath(repository)
	if err != nil {
		return err
	}
	temporaryPath, err := os.MkdirTemp(filepath.Dir(outputPath), "."+filepath.Base(outputPath)+"-")
	if err != nil {
		return fmt.Errorf("create build output directory: %w", err)
	}
	if err := os.Remove(temporaryPath); err != nil {
		return fmt.Errorf("prepare build output directory: %w", err)
	}
	defer os.RemoveAll(temporaryPath)

	s.setStage(id, "building")
	stats, err := site.NewBuilder(cfg, renderer).Build(s.ctx, tree, temporaryPath)
	if err != nil {
		return fmt.Errorf("build site: %w", err)
	}
	warnings = append(warnings, stats.Warnings...)
	if err := replaceBuildOutput(temporaryPath, outputPath); err != nil {
		return fmt.Errorf("publish build output: %w", err)
	}
	s.complete(id, outputPath, stats, warnings)
	return nil
}

func resolveBuildSourcePath(root, value string) string {
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(root, filepath.FromSlash(value))
}

func (s *buildService) outputPath(repository string) (string, error) {
	root, err := s.cacheRoot()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create build cache directory: %w", err)
	}
	sum := sha256.Sum256([]byte(repository))
	return filepath.Join(root, hex.EncodeToString(sum[:])), nil
}

func replaceBuildOutput(temporaryPath, outputPath string) error {
	backupPath := outputPath + ".previous"
	if err := os.RemoveAll(backupPath); err != nil {
		return err
	}
	hadPrevious := false
	if _, err := os.Stat(outputPath); err == nil {
		if err := os.Rename(outputPath, backupPath); err != nil {
			return err
		}
		hadPrevious = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(temporaryPath, outputPath); err != nil {
		if hadPrevious {
			_ = os.Rename(backupPath, outputPath)
		}
		return err
	}
	return os.RemoveAll(backupPath)
}

func (s *buildService) setStage(id, stage string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if op, ok := s.operations[id]; ok {
		op.Stage = stage
	}
}

func (s *buildService) setWarnings(id string, warnings []config.Warning) {
	values := warningMessages(warnings)
	s.mu.Lock()
	defer s.mu.Unlock()
	if op, ok := s.operations[id]; ok {
		op.Warnings = values
	}
}

func (s *buildService) complete(id, outputPath string, stats site.Stats, warnings []config.Warning) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if op, ok := s.operations[id]; ok {
		copy := stats
		copy.Warnings = nil
		op.Stage = "completed"
		op.Stats = &copy
		op.Warnings = warningMessages(warnings)
		op.OutputPath = outputPath
	}
}

func (s *buildService) fail(id string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if op, ok := s.operations[id]; ok {
		op.Stage = "failed"
		op.Error = err.Error()
	}
}

func (s *buildService) release(repository string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.repositories, repository)
}

func warningMessages(warnings []config.Warning) []string {
	if len(warnings) == 0 {
		return nil
	}
	values := make([]string, len(warnings))
	for index, warning := range warnings {
		values[index] = warning.Msg
	}
	return values
}

func cloneBuildOperation(op *buildOperation) buildOperation {
	copy := *op
	if op.Stats != nil {
		stats := *op.Stats
		copy.Stats = &stats
	}
	copy.Warnings = append([]string(nil), op.Warnings...)
	return copy
}

func newBuildID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("create build operation ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
