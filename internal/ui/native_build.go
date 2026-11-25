package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/ConteMan/repolens/internal/source"
)

// NativeBuildResult identifies the artifact published by a completed native build.
type NativeBuildResult struct {
	Directory string
	Source    string
	Pages     int
	Warnings  []string
}

var ErrConfirmOutput = errors.New("confirm replacement of existing Repolens output")

// BuildNative shares the Web UI's output validation, staging and publication.
func BuildNative(ctx context.Context, root, external, destination string, confirm bool) (NativeBuildResult, error) {
	var result NativeBuildResult
	sourceSpec := source.Spec{Repo: root, Worktree: true}
	if external != "" {
		doc, err := config.LoadBuilderDocument(external)
		if err != nil {
			return result, err
		}
		cfg := doc.Settings
		if cfg.Source.Repo != nil && *cfg.Source.Repo != "" {
			sourceSpec.Repo = *cfg.Source.Repo
			if !filepath.IsAbs(sourceSpec.Repo) && !strings.Contains(sourceSpec.Repo, ":") {
				sourceSpec.Repo = filepath.Join(filepath.Dir(external), sourceSpec.Repo)
			}
		}
		if cfg.Source.Ref != nil {
			sourceSpec.Ref = *cfg.Source.Ref
		}
		if destination == "" {
			destination = NativeOutputPath(external, cfg.Output.Dir)
		}
		sourceSpec.Worktree = sourceSpec.Ref == "" && sourceSpec.Repo == root
	}
	service := newBuildService(ctx)
	target, err := service.buildTarget(root, destination, confirm)
	if err != nil {
		var output *buildOutputError
		if errors.As(err, &output) && output.code == "output_overwrite_confirmation_required" {
			return result, errors.Join(ErrConfirmOutput, err)
		}
		return result, err
	}
	if target.Mode == outputModeCustom && sourceSpec.Repo != root {
		if info, e := os.Stat(sourceSpec.Repo); e == nil && info.IsDir() {
			sourceRoot, e := resolveExistingPath(sourceSpec.Repo)
			if e != nil {
				return result, e
			}
			if e = validateOutputLocation(target.Path, sourceRoot); e != nil {
				return result, e
			}
		}
	}
	const id = "native"
	service.operations[id] = &buildOperation{}
	if err = service.buildSource(id, sourceSpec, external, target); err != nil {
		return result, err
	}
	op, _ := service.operation(id)
	result = NativeBuildResult{Directory: op.OutputPath, Source: sourceSpec.Repo, Pages: op.Stats.Pages, Warnings: op.Warnings}
	if sourceSpec.Ref != "" {
		result.Source += " @ " + sourceSpec.Ref
	}
	return result, nil
}

// NativeOutputPath resolves a builder's explicit destination relative to its configuration.
func NativeOutputPath(external string, dir *string) string {
	if dir == nil || *dir == "" {
		return ""
	}
	if filepath.IsAbs(*dir) {
		return filepath.Clean(*dir)
	}
	return filepath.Join(filepath.Dir(external), *dir)
}
