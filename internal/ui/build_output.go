package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ConteMan/repolens/internal/site"
)

const (
	outputModeCache  = "cache"
	outputModeCustom = "custom"
)

type buildTarget struct {
	Path             string
	Mode             string
	ConfirmOverwrite bool
}

type buildOutputError struct {
	code    string
	message string
	status  int
	path    string
}

func (e *buildOutputError) Error() string { return e.message }

type outputState int

const (
	outputAbsent outputState = iota
	outputEmpty
	outputOwned
)

func (s *buildService) buildTarget(repository, requestedPath string, confirmOverwrite bool) (buildTarget, error) {
	if strings.TrimSpace(requestedPath) == "" {
		path, err := s.outputPath(repository)
		if err != nil {
			return buildTarget{}, err
		}
		return buildTarget{Path: path, Mode: outputModeCache}, nil
	}

	path, err := normalizeOutputPath(requestedPath)
	if err != nil {
		return buildTarget{}, err
	}
	repositoryPath, err := resolveExistingPath(repository)
	if err != nil {
		return buildTarget{}, newBuildOutputError("invalid_output_path", "unable to resolve the repository path", 400)
	}
	if err := validateOutputLocation(path, repositoryPath); err != nil {
		return buildTarget{}, err
	}
	state, err := inspectOutput(path)
	if err != nil {
		return buildTarget{}, err
	}
	if state == outputOwned && !confirmOverwrite {
		return buildTarget{}, &buildOutputError{
			code:    "output_overwrite_confirmation_required",
			message: fmt.Sprintf("output directory %q contains an existing repolens build", path),
			status:  409,
			path:    path,
		}
	}
	return buildTarget{Path: path, Mode: outputModeCustom, ConfirmOverwrite: state == outputOwned && confirmOverwrite}, nil
}

func normalizeOutputPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || !filepath.IsAbs(value) {
		return "", newBuildOutputError("invalid_output_path", "output path must be an absolute directory", 400)
	}
	path, err := resolveExistingPath(value)
	if err != nil {
		return "", newBuildOutputError("invalid_output_path", "output path cannot be resolved safely", 400)
	}
	return path, nil
}

func resolveExistingPath(value string) (string, error) {
	current := filepath.Clean(value)
	remaining := make([]string, 0)
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for index := len(remaining) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, remaining[index])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		remaining = append(remaining, filepath.Base(current))
		current = parent
	}
}

func validateOutputLocation(path, repository string) error {
	if filepath.Dir(path) == path {
		return newBuildOutputError("unsafe_output_path", "filesystem root cannot be used as build output", 422)
	}
	for _, root := range protectedOutputRoots() {
		if samePath(path, root) {
			return newBuildOutputError("unsafe_output_path", fmt.Sprintf("protected directory %q cannot be used as build output", root), 422)
		}
	}
	if pathContains(path, repository) || pathContains(repository, path) {
		return newBuildOutputError("unsafe_output_path", "output path must be outside and must not contain the repository", 422)
	}
	return nil
}

func protectedOutputRoots() []string {
	values := make([]string, 0, 3)
	for _, loader := range []func() (string, error){os.UserHomeDir, os.UserCacheDir} {
		if value, err := loader(); err == nil {
			if resolved, err := resolveExistingPath(value); err == nil {
				values = append(values, resolved)
			}
		}
	}
	if resolved, err := resolveExistingPath(os.TempDir()); err == nil {
		values = append(values, resolved)
	}
	return values
}

func samePath(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right)
}

func pathContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}

func inspectOutput(path string) (outputState, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return outputAbsent, nil
	}
	if err != nil {
		return outputAbsent, newBuildOutputError("invalid_output_path", "output path cannot be inspected safely", 400)
	}
	if !info.IsDir() {
		return outputAbsent, newBuildOutputError("invalid_output_path", "output path exists and is not a directory", 400)
	}
	owned, err := site.OwnsOutput(path)
	if err != nil {
		return outputAbsent, newBuildOutputError("invalid_output_path", "output ownership cannot be inspected safely", 400)
	}
	if owned {
		return outputOwned, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return outputAbsent, newBuildOutputError("invalid_output_path", "output directory cannot be inspected safely", 400)
	}
	if len(entries) == 0 {
		return outputEmpty, nil
	}
	return outputAbsent, newBuildOutputError("output_not_owned", "non-empty output directory is not owned by repolens", 422)
}

func recheckBuildTarget(target buildTarget) error {
	resolved, err := resolveExistingPath(target.Path)
	if err != nil || !samePath(resolved, target.Path) {
		return newBuildOutputError("unsafe_output_path", "output path changed while the build was running", 422)
	}
	state, err := inspectOutput(target.Path)
	if err != nil {
		return err
	}
	if state == outputOwned && !target.ConfirmOverwrite {
		return newBuildOutputError("output_not_owned", "output ownership changed while the build was running", 422)
	}
	return nil
}

func newBuildOutputError(code, message string, status int) error {
	return &buildOutputError{code: code, message: message, status: status}
}
