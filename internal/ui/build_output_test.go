package ui

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ConteMan/repolens/internal/site"
)

func TestBuildTargetValidatesCustomOutput(t *testing.T) {
	root := t.TempDir()
	repository := filepath.Join(root, "repository")
	if err := os.Mkdir(repository, 0o755); err != nil {
		t.Fatalf("create repository: %v", err)
	}
	service := newBuildService(context.Background())

	t.Run("accepts missing and empty directories", func(t *testing.T) {
		missing := filepath.Join(root, "missing-output")
		resolvedMissing, err := resolveExistingPath(missing)
		if err != nil {
			t.Fatalf("resolve missing output: %v", err)
		}
		target, err := service.buildTarget(repository, missing, false)
		if err != nil || target.Path != resolvedMissing || target.Mode != outputModeCustom || target.ConfirmOverwrite {
			t.Fatalf("missing target = %#v, %v", target, err)
		}
		empty := filepath.Join(root, "empty-output")
		if err := os.Mkdir(empty, 0o755); err != nil {
			t.Fatalf("create empty output: %v", err)
		}
		if _, err := service.buildTarget(repository, empty, false); err != nil {
			t.Fatalf("empty target error = %v", err)
		}
	})

	t.Run("resolves existing symlink components", func(t *testing.T) {
		realParent := filepath.Join(root, "real-parent")
		if err := os.Mkdir(realParent, 0o755); err != nil {
			t.Fatalf("create real parent: %v", err)
		}
		linkParent := filepath.Join(root, "linked-parent")
		if err := os.Symlink(realParent, linkParent); err != nil {
			t.Fatalf("create parent symlink: %v", err)
		}
		target, err := service.buildTarget(repository, filepath.Join(linkParent, "site"), false)
		if err != nil {
			t.Fatalf("symlink target error = %v", err)
		}
		resolvedRealParent, err := resolveExistingPath(filepath.Join(realParent, "site"))
		if err != nil {
			t.Fatalf("resolve real parent: %v", err)
		}
		if target.Path != resolvedRealParent {
			t.Fatalf("target path = %q", target.Path)
		}
	})

	t.Run("requires confirmation for owned output", func(t *testing.T) {
		owned := filepath.Join(root, "owned-output")
		if err := os.Mkdir(owned, 0o755); err != nil {
			t.Fatalf("create owned output: %v", err)
		}
		if err := os.WriteFile(filepath.Join(owned, ".repolens-build"), []byte("repolens build output\n"), 0o644); err != nil {
			t.Fatalf("write sentinel: %v", err)
		}
		assertBuildOutputError(t, func() error {
			_, err := service.buildTarget(repository, owned, false)
			return err
		}(), "output_overwrite_confirmation_required")
		target, err := service.buildTarget(repository, owned, true)
		if err != nil || !target.ConfirmOverwrite {
			t.Fatalf("confirmed target = %#v, %v", target, err)
		}
	})

	t.Run("rejects files and unowned directories", func(t *testing.T) {
		file := filepath.Join(root, "output-file")
		if err := os.WriteFile(file, []byte("file"), 0o644); err != nil {
			t.Fatalf("write output file: %v", err)
		}
		assertBuildOutputError(t, func() error {
			_, err := service.buildTarget(repository, file, false)
			return err
		}(), "invalid_output_path")

		unowned := filepath.Join(root, "unowned-output")
		if err := os.Mkdir(unowned, 0o755); err != nil {
			t.Fatalf("create unowned output: %v", err)
		}
		if err := os.WriteFile(filepath.Join(unowned, "keep.txt"), []byte("keep"), 0o644); err != nil {
			t.Fatalf("write unowned content: %v", err)
		}
		assertBuildOutputError(t, func() error {
			_, err := service.buildTarget(repository, unowned, true)
			return err
		}(), "output_not_owned")
	})

	for _, test := range []struct {
		name string
		path string
		code string
	}{
		{name: "relative", path: "dist", code: "invalid_output_path"},
		{name: "filesystem root", path: string(filepath.Separator), code: "unsafe_output_path"},
		{name: "repository", path: repository, code: "unsafe_output_path"},
		{name: "inside repository", path: filepath.Join(repository, "dist"), code: "unsafe_output_path"},
		{name: "repository ancestor", path: root, code: "unsafe_output_path"},
	} {
		t.Run("rejects "+test.name, func(t *testing.T) {
			assertBuildOutputError(t, func() error {
				_, err := service.buildTarget(repository, test.path, true)
				return err
			}(), test.code)
		})
	}
}

func TestRecheckBuildTargetRejectsOwnershipChanges(t *testing.T) {
	output := filepath.Join(t.TempDir(), "output")
	resolvedOutput, err := resolveExistingPath(output)
	if err != nil {
		t.Fatalf("resolve output: %v", err)
	}
	target := buildTarget{Path: resolvedOutput, Mode: outputModeCustom}
	if err := os.Mkdir(output, 0o755); err != nil {
		t.Fatalf("create output: %v", err)
	}
	if err := os.WriteFile(filepath.Join(output, ".repolens-build"), []byte("repolens build output\n"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	assertBuildOutputError(t, recheckBuildTarget(target), "output_not_owned")

	confirmed := buildTarget{Path: resolvedOutput, Mode: outputModeCustom, ConfirmOverwrite: true}
	if err := os.Remove(filepath.Join(output, ".repolens-build")); err != nil {
		t.Fatalf("remove sentinel: %v", err)
	}
	if err := os.WriteFile(filepath.Join(output, "unknown.txt"), []byte("unknown"), 0o644); err != nil {
		t.Fatalf("write unknown content: %v", err)
	}
	assertBuildOutputError(t, recheckBuildTarget(confirmed), "output_not_owned")
}

func TestReplaceBuildOutputDoesNotDeleteFixedPreviousDirectory(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "site")
	temporary := filepath.Join(root, "temporary")
	previous := output + ".previous"
	for _, directory := range []string{output, temporary, previous} {
		if err := os.Mkdir(directory, 0o755); err != nil {
			t.Fatalf("create %s: %v", directory, err)
		}
	}
	if err := os.WriteFile(filepath.Join(temporary, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatalf("write temporary output: %v", err)
	}
	if err := os.WriteFile(filepath.Join(previous, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("write fixed previous content: %v", err)
	}
	if err := replaceBuildOutput(temporary, output); err != nil {
		t.Fatalf("replaceBuildOutput() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, "new.txt")); err != nil {
		t.Fatalf("new output is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(previous, "keep.txt")); err != nil {
		t.Fatalf("fixed previous directory was modified: %v", err)
	}
}

func TestBuildAPICustomOutputAndErrors(t *testing.T) {
	repository := t.TempDir()
	if err := os.WriteFile(filepath.Join(repository, "README.md"), []byte("# Worktree content\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	service := newBuildService(context.Background())
	handler := newHandlerWithBuildService("correct-token", service)
	output := filepath.Join(t.TempDir(), "published-site")
	resolvedOutput, err := resolveExistingPath(output)
	if err != nil {
		t.Fatalf("resolve custom output: %v", err)
	}

	start := httptest.NewRecorder()
	handler.ServeHTTP(start, newJSONRequest(t, http.MethodPost, "/api/build", buildRequest{Path: repository, OutputPath: output}, "correct-token"))
	if start.Code != http.StatusAccepted {
		t.Fatalf("custom build status = %d: %s", start.Code, start.Body.String())
	}
	var started buildResponse
	if err := json.NewDecoder(start.Body).Decode(&started); err != nil {
		t.Fatalf("decode custom build response: %v", err)
	}
	if started.OutputMode != outputModeCustom || started.OutputPath != resolvedOutput {
		t.Fatalf("custom build response = %#v", started)
	}
	completed := waitForBuild(t, handler, started.ID)
	if completed.Stage != "completed" || completed.OutputMode != outputModeCustom {
		t.Fatalf("custom build result = %#v", completed)
	}
	if owned, err := site.OwnsOutput(output); err != nil || !owned {
		t.Fatalf("published output ownership = %v, %v", owned, err)
	}

	for _, test := range []struct {
		name   string
		path   string
		status int
		code   string
	}{
		{name: "relative", path: "dist", status: http.StatusBadRequest, code: "invalid_output_path"},
		{name: "inside repository", path: filepath.Join(repository, "dist"), status: http.StatusUnprocessableEntity, code: "unsafe_output_path"},
		{name: "owned confirmation", path: output, status: http.StatusConflict, code: "output_overwrite_confirmation_required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, newJSONRequest(t, http.MethodPost, "/api/build", buildRequest{Path: repository, OutputPath: test.path}, "correct-token"))
			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, test.status, recorder.Body.String())
			}
			var response struct {
				Code  string `json:"code"`
				Field string `json:"field"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if response.Code != test.code || response.Field != "output_path" {
				t.Fatalf("error response = %#v", response)
			}
		})
	}
}

func assertBuildOutputError(t *testing.T, err error, code string) {
	t.Helper()
	var outputErr *buildOutputError
	if !errors.As(err, &outputErr) || outputErr.code != code {
		t.Fatalf("error = %v, want build output code %q", err, code)
	}
}
