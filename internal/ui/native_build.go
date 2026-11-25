package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"

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

// BuildOutputErrorCode returns the stable code associated with an output validation failure.
func BuildOutputErrorCode(err error) string {
	var e *buildOutputError
	if errors.As(err, &e) {
		return e.code
	}
	return ""
}

// BuildNative shares the Web UI's output validation, staging and publication.
// A nil override follows the external configuration. An explicit empty one uses the application cache.
func BuildNative(ctx context.Context, root, external string, override *string, confirm bool) (NativeBuildResult, error) {
	var result NativeBuildResult
	var cfg config.BuilderSettings
	if external != "" {
		doc, err := config.LoadBuilderDocument(external)
		if err != nil {
			return result, err
		}
		cfg = doc.Settings
	}
	sourceSpec, destination := nativeBuildInputs(root, external, cfg, override)
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

// NativeBuildLocations describes the paths used for a desktop build before it runs.
type NativeBuildLocations struct {
	Source       source.Spec
	RemoteMirror string
	Output       string
}

// ResolveNativeBuildLocations resolves the configured paths without creating any directories.
func ResolveNativeBuildLocations(root, external string, cfg config.BuilderSettings, override *string) (NativeBuildLocations, error) {
	input, destination := nativeBuildInputs(root, external, cfg, override)
	locations := NativeBuildLocations{Source: input}
	local, remote, err := source.RepositoryDirectory(input.Repo)
	if err != nil {
		return locations, err
	}
	if remote {
		locations.RemoteMirror = local
	}
	if destination == "" {
		locations.Output, err = cachedBuildOutput(root, defaultBuildCacheRoot)
	} else {
		locations.Output, err = normalizeOutputPath(destination)
	}
	return locations, err
}

func nativeBuildInputs(root, external string, cfg config.BuilderSettings, override *string) (source.Spec, string) {
	input := source.Spec{Repo: root}
	if repo := cfg.Source.Repo; repo != nil && *repo != "" {
		input.Repo = *repo
		if !source.IsRemote(*repo) && !filepath.IsAbs(*repo) {
			input.Repo = filepath.Join(filepath.Dir(external), *repo)
		}
	}
	if ref := cfg.Source.Ref; ref != nil {
		input.Ref = *ref
	}
	input.Worktree = input.Ref == "" && input.Repo == root
	var destination string
	if dir := cfg.Output.Dir; dir != nil && *dir != "" {
		destination = *dir
		if !filepath.IsAbs(destination) {
			destination = filepath.Join(filepath.Dir(external), destination)
		}
	}
	if override != nil {
		destination = *override
	}
	return input, destination
}
