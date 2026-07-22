package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ConteMan/repolens/internal/config"
)

func TestLoopbackAddrRejectsNonLoopback(t *testing.T) {
	for _, addr := range []string{
		"0.0.0.0:8799",
		"192.168.1.10:8799",
		"localhost:8799",
		"[::]:8799",
	} {
		t.Run(addr, func(t *testing.T) {
			if _, err := loopbackAddr(addr); err == nil {
				t.Fatalf("loopbackAddr(%q) returned nil error", addr)
			}
		})
	}
}

func TestHealth(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)

	newHandler("test-token").ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := recorder.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("health body = %q", body)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("health Content-Type = %q", contentType)
	}
}

func TestRootServesEmbeddedShell(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	newHandler("test-token").ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("root status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("root Content-Type = %q", contentType)
	}
	if body := recorder.Body.String(); !strings.Contains(body, `content="test-token"`) {
		t.Fatalf("root body does not contain injected token: %q", body)
	}
	if csp := recorder.Header().Get("Content-Security-Policy"); strings.Contains(csp, "unsafe-inline") {
		t.Fatalf("root CSP permits inline assets: %q", csp)
	}
}

func TestEmbeddedFrontendAsset(t *testing.T) {
	assetPath := embeddedAssetPath(t, shellHTML, `src="`, `"`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, assetPath, nil)

	newHandler("test-token").ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("asset status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "javascript") {
		t.Fatalf("asset Content-Type = %q, want JavaScript", contentType)
	}
}

func embeddedAssetPath(t *testing.T, html, prefix, suffix string) string {
	t.Helper()
	start := strings.Index(html, prefix)
	if start < 0 {
		t.Fatalf("embedded HTML does not contain %q", prefix)
	}
	start += len(prefix)
	end := strings.Index(html[start:], suffix)
	if end < 0 {
		t.Fatalf("embedded HTML asset after %q does not contain %q", prefix, suffix)
	}
	return html[start : start+end]
}

func TestConfigAPIsRejectMissingOrWrongToken(t *testing.T) {
	for _, endpoint := range []string{
		"/api/build",
		"/api/config/validate",
		"/api/config/prepare-write",
		"/api/config/commit",
	} {
		for _, token := range []string{"", "wrong-token"} {
			t.Run(endpoint+"/"+token, func(t *testing.T) {
				recorder := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodPost, endpoint, nil)
				if token != "" {
					request.Header.Set(csrfTokenHeader, token)
				}

				newHandler("correct-token").ServeHTTP(recorder, request)

				if recorder.Code != http.StatusForbidden {
					t.Fatalf("%s status = %d, want %d", endpoint, recorder.Code, http.StatusForbidden)
				}
				var response struct {
					Code string `json:"code"`
				}
				if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if response.Code != "invalid_token" {
					t.Fatalf("error code = %q, want invalid_token", response.Code)
				}
			})
		}
	}
}

func TestConfigAPIsRejectInvalidPath(t *testing.T) {
	for _, endpoint := range []string{
		"/api/config/validate",
		"/api/config/prepare-write",
		"/api/config/commit",
	} {
		t.Run(endpoint, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := configRequest{
				Path:     "relative/repository",
				Revision: "revision",
				Confirm:  true,
			}

			newHandler("correct-token").ServeHTTP(recorder, newJSONRequest(t, http.MethodPost, endpoint, request, "correct-token"))

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("%s status = %d, want %d: %s", endpoint, recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}
			var response struct {
				Code string `json:"code"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if response.Code != "invalid_path" {
				t.Fatalf("error code = %q, want invalid_path", response.Code)
			}
		})
	}
}

func TestOpenProjectReturnsDocumentSettingsEffectiveValuesSourcesAndWarnings(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, ".repolens.yml"), []byte("source:\n  repo: ignored\nsite:\n  title: Project Lens\nview:\n  search: false\nrules:\n  - match: docs/**\n    markdown:\n      math: true\n"), 0o644); err != nil {
		t.Fatalf("write repository config: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := projectOpenRequest{Path: repoRoot}
	httpRequest := newJSONRequest(t, http.MethodPost, "/api/project/open", request, "correct-token")

	newHandler("correct-token").ServeHTTP(recorder, httpRequest)

	if recorder.Code != http.StatusOK {
		t.Fatalf("open status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("open Content-Type = %q", contentType)
	}
	var response projectOpenResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Revision == "" {
		t.Fatal("response revision is empty")
	}
	if response.Settings.Site.Title == nil || *response.Settings.Site.Title != "Project Lens" {
		t.Fatalf("response site settings = %#v", response.Settings.Site)
	}
	if response.Settings.View.Search == nil || *response.Settings.View.Search {
		t.Fatalf("response view settings = %#v", response.Settings.View)
	}
	if response.Effective.Site.Language == nil || *response.Effective.Site.Language != "zh-CN" {
		t.Fatalf("effective site settings = %#v", response.Effective.Site)
	}
	if response.Effective.View.Search == nil || *response.Effective.View.Search {
		t.Fatalf("effective view settings = %#v", response.Effective.View)
	}
	if response.Effective.Rules == nil || len(*response.Effective.Rules) != 1 {
		t.Fatalf("effective rules = %#v", response.Effective.Rules)
	}
	effectiveRule := (*response.Effective.Rules)[0]
	if effectiveRule.Markdown.Math == nil || !*effectiveRule.Markdown.Math || effectiveRule.Markdown.TOC != nil {
		t.Fatalf("effective rule lost repository presence: %#v", effectiveRule)
	}
	if response.Sources["site.title"] != "repository" || response.Sources["site.language"] != "default" || response.Sources["view.search"] != "repository" {
		t.Fatalf("response sources = %#v", response.Sources)
	}
	if response.Sources["rules[0].markdown.math"] != "repository" || response.Sources["rules[0].markdown.toc"] != "default" {
		t.Fatalf("rule sources = %#v", response.Sources)
	}
	if len(response.Warnings) == 0 || !strings.Contains(response.Warnings[0], "source") {
		t.Fatalf("response warnings = %#v", response.Warnings)
	}
}

func TestLoadProjectSnapshotRetriesChangesAndRejectsUnstableConfiguration(t *testing.T) {
	t.Run("retries once and returns one stable snapshot", func(t *testing.T) {
		revisions := []string{"before", "changed", "changed", "changed"}
		documentCalls := 0
		effectiveCalls := 0
		document, _, warnings, err := loadProjectSnapshotWith(
			"/repository",
			func(string) (*config.RepositoryDocument, error) {
				revision := revisions[documentCalls]
				documentCalls++
				return &config.RepositoryDocument{Revision: revision}, nil
			},
			func(string, string, config.Flags) (*config.Config, []config.Warning, error) {
				effectiveCalls++
				return &config.Config{}, []config.Warning{{Msg: "warning"}}, nil
			},
		)
		if err != nil {
			t.Fatalf("loadProjectSnapshotWith() error = %v", err)
		}
		if document.Revision != "changed" || documentCalls != 4 || effectiveCalls != 2 || len(warnings) != 1 {
			t.Fatalf("snapshot = revision %q, document calls %d, effective calls %d, warnings %#v", document.Revision, documentCalls, effectiveCalls, warnings)
		}
	})

	t.Run("rejects configuration that changes on every attempt", func(t *testing.T) {
		documentCalls := 0
		_, _, _, err := loadProjectSnapshotWith(
			"/repository",
			func(string) (*config.RepositoryDocument, error) {
				documentCalls++
				return &config.RepositoryDocument{Revision: fmt.Sprintf("revision-%d", documentCalls)}, nil
			},
			func(string, string, config.Flags) (*config.Config, []config.Warning, error) {
				return &config.Config{}, nil, nil
			},
		)
		if err == nil || !strings.Contains(err.Error(), "changed while loading") {
			t.Fatalf("loadProjectSnapshotWith() error = %v, want unstable configuration error", err)
		}
		if documentCalls != 6 {
			t.Fatalf("document calls = %d, want 6", documentCalls)
		}
	})
}

func TestOpenProjectRejectsInvalidPathAndToken(t *testing.T) {
	repoRoot := t.TempDir()
	notDirectory := filepath.Join(repoRoot, "not-a-directory")
	if err := os.WriteFile(notDirectory, []byte("file"), 0o644); err != nil {
		t.Fatalf("write file fixture: %v", err)
	}

	for _, test := range []struct {
		name       string
		path       string
		token      string
		wantStatus int
		wantCode   string
	}{
		{name: "missing token", path: repoRoot, wantStatus: http.StatusForbidden, wantCode: "invalid_token"},
		{name: "wrong token", path: repoRoot, token: "wrong-token", wantStatus: http.StatusForbidden, wantCode: "invalid_token"},
		{name: "relative path", path: "relative/repository", token: "correct-token", wantStatus: http.StatusBadRequest, wantCode: "invalid_path"},
		{name: "non-directory", path: notDirectory, token: "correct-token", wantStatus: http.StatusBadRequest, wantCode: "invalid_path"},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			httpRequest := newJSONRequest(t, http.MethodPost, "/api/project/open", projectOpenRequest{Path: test.path}, test.token)

			newHandler("correct-token").ServeHTTP(recorder, httpRequest)

			if recorder.Code != test.wantStatus {
				t.Fatalf("open status = %d, want %d: %s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			var response struct {
				Code string `json:"code"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if response.Code != test.wantCode {
				t.Fatalf("error code = %q, want %q", response.Code, test.wantCode)
			}
		})
	}
}

func TestBuildAPIBuildsWorktreeIntoCache(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# UI build\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	cacheRoot := filepath.Join(t.TempDir(), "builds")
	builds := newBuildService(context.Background())
	builds.cacheRoot = func() (string, error) { return cacheRoot, nil }
	handler := newHandlerWithBuildService("correct-token", builds)

	startRecorder := httptest.NewRecorder()
	handler.ServeHTTP(startRecorder, newJSONRequest(t, http.MethodPost, "/api/build", buildRequest{Path: repoRoot}, "correct-token"))
	if startRecorder.Code != http.StatusAccepted {
		t.Fatalf("start build status = %d, want %d: %s", startRecorder.Code, http.StatusAccepted, startRecorder.Body.String())
	}
	var started buildResponse
	if err := json.NewDecoder(startRecorder.Body).Decode(&started); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	if started.ID == "" || started.Stage == "" || started.OutputPath == "" {
		t.Fatalf("start response = %#v, want ID, stage, and output path", started)
	}

	completed := waitForBuild(t, handler, started.ID)
	if completed.Stage != "completed" {
		t.Fatalf("build stage = %q, want completed; error = %q", completed.Stage, completed.Error)
	}
	if completed.Stats == nil || completed.Stats.Files != 1 || completed.Stats.Pages == 0 {
		t.Fatalf("build stats = %#v", completed.Stats)
	}
	if completed.OutputPath != started.OutputPath {
		t.Fatalf("output path = %q, want %q", completed.OutputPath, started.OutputPath)
	}
	if _, err := os.Stat(filepath.Join(completed.OutputPath, "view", "index.html")); err != nil {
		t.Fatalf("generated site is missing: %v", err)
	}
	if strings.HasPrefix(completed.OutputPath, repoRoot+string(filepath.Separator)) {
		t.Fatalf("output path %q is inside repository %q", completed.OutputPath, repoRoot)
	}
}

func TestBuildAPIRejectsConcurrentBuildForRepository(t *testing.T) {
	repoRoot := t.TempDir()
	builds := newBuildService(context.Background())
	builds.cacheRoot = func() (string, error) { return t.TempDir(), nil }
	builds.mu.Lock()
	builds.repositories[repoRoot] = "existing-operation"
	builds.mu.Unlock()

	recorder := httptest.NewRecorder()
	newHandlerWithBuildService("correct-token", builds).ServeHTTP(recorder, newJSONRequest(t, http.MethodPost, "/api/build", buildRequest{Path: repoRoot}, "correct-token"))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("concurrent build status = %d, want %d: %s", recorder.Code, http.StatusConflict, recorder.Body.String())
	}
	var response struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Code != "build_in_progress" {
		t.Fatalf("error code = %q, want build_in_progress", response.Code)
	}
}

func TestBuildAPIReportsBuildFailure(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# UI build\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	builds := newBuildService(context.Background())
	cacheRoot := filepath.Join(t.TempDir(), "builds")
	builds.cacheRoot = func() (string, error) { return cacheRoot, nil }
	handler := newHandlerWithBuildService("correct-token", builds)

	firstRecorder := httptest.NewRecorder()
	handler.ServeHTTP(firstRecorder, newJSONRequest(t, http.MethodPost, "/api/build", buildRequest{Path: repoRoot}, "correct-token"))
	if firstRecorder.Code != http.StatusAccepted {
		t.Fatalf("start first build status = %d, want %d: %s", firstRecorder.Code, http.StatusAccepted, firstRecorder.Body.String())
	}
	var first buildResponse
	if err := json.NewDecoder(firstRecorder.Body).Decode(&first); err != nil {
		t.Fatalf("decode first build response: %v", err)
	}
	completed := waitForBuild(t, handler, first.ID)
	if completed.Stage != "completed" {
		t.Fatalf("first build response = %#v, want completed", completed)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".repolens.yml"), []byte("theme:\n  css: missing.css\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	failedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(failedRecorder, newJSONRequest(t, http.MethodPost, "/api/build", buildRequest{Path: repoRoot}, "correct-token"))
	if failedRecorder.Code != http.StatusAccepted {
		t.Fatalf("start failing build status = %d, want %d: %s", failedRecorder.Code, http.StatusAccepted, failedRecorder.Body.String())
	}
	var started buildResponse
	if err := json.NewDecoder(failedRecorder.Body).Decode(&started); err != nil {
		t.Fatalf("decode failing build response: %v", err)
	}
	failed := waitForBuild(t, handler, started.ID)
	if failed.Stage != "failed" || !strings.Contains(failed.Error, "build site") || !strings.Contains(failed.Error, "missing.css") {
		t.Fatalf("build response = %#v, want failed theme error", failed)
	}
	if failed.OutputPath != completed.OutputPath {
		t.Fatalf("failure output path = %q, want previous output path %q", failed.OutputPath, completed.OutputPath)
	}
	if _, err := os.Stat(filepath.Join(failed.OutputPath, "view", "index.html")); err != nil {
		t.Fatalf("successful output was not preserved: %v", err)
	}
}

func waitForBuild(t *testing.T, handler http.Handler, id string) buildResponse {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/build/"+id, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("get build status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		var response buildResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode build response: %v", err)
		}
		if response.Stage == "completed" || response.Stage == "failed" {
			return response
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("build operation %q did not finish", id)
	return buildResponse{}
}

func TestConfigValidateAndPrepareWriteDoNotWrite(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".repolens.yml")
	original := "site:\n  title: Before\nunknown: preserved\n"
	if err := os.WriteFile(configPath, []byte(original), 0o644); err != nil {
		t.Fatalf("write repository config: %v", err)
	}
	title := "After"
	request := configRequest{
		Path: repoRoot,
		Settings: repositorySettingsFromConfig(config.RepositorySettings{
			Site: config.RepositorySiteSettings{Title: &title},
		}),
	}

	validateRecorder := httptest.NewRecorder()
	validateRequest := newJSONRequest(t, http.MethodPost, "/api/config/validate", request, "correct-token")
	newHandler("correct-token").ServeHTTP(validateRecorder, validateRequest)
	if validateRecorder.Code != http.StatusOK {
		t.Fatalf("validate status = %d, want %d: %s", validateRecorder.Code, http.StatusOK, validateRecorder.Body.String())
	}
	var validateResponse configResponse
	if err := json.NewDecoder(validateRecorder.Body).Decode(&validateResponse); err != nil {
		t.Fatalf("decode validate response: %v", err)
	}
	if validateResponse.Revision == "" || validateResponse.Settings.Site.Title == nil || *validateResponse.Settings.Site.Title != title {
		t.Fatalf("validate response = %#v", validateResponse)
	}
	assertFileContent(t, configPath, original)

	prepareRecorder := httptest.NewRecorder()
	prepareRequest := newJSONRequest(t, http.MethodPost, "/api/config/prepare-write", request, "correct-token")
	newHandler("correct-token").ServeHTTP(prepareRecorder, prepareRequest)
	if prepareRecorder.Code != http.StatusOK {
		t.Fatalf("prepare status = %d, want %d: %s", prepareRecorder.Code, http.StatusOK, prepareRecorder.Body.String())
	}
	var prepareResponse prepareWriteResponse
	if err := json.NewDecoder(prepareRecorder.Body).Decode(&prepareResponse); err != nil {
		t.Fatalf("decode prepare response: %v", err)
	}
	if prepareResponse.Before != original || !strings.Contains(prepareResponse.After, "title: After") {
		t.Fatalf("prepare response before=%q after=%q", prepareResponse.Before, prepareResponse.After)
	}
	if !strings.Contains(prepareResponse.Diff, "-  title: Before") || !strings.Contains(prepareResponse.Diff, "+  title: After") {
		t.Fatalf("prepare diff = %q", prepareResponse.Diff)
	}
	assertFileContent(t, configPath, original)
}

func TestConfigAPIsMapRepositorySettingsAndPreserveTrustedSettings(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".repolens.yml")
	original := `source:
  repo: git@example.com:trusted/source.git
  ref: main
output:
  dir: trusted-dist
  base_path: /trusted/
access:
  noindex: true
  encrypt:
    enabled: true
    paths: [private/**]
site:
  title: Before
rules:
  - match: old/**
    render: false
`
	if err := os.WriteFile(configPath, []byte(original), 0o644); err != nil {
		t.Fatalf("write repository config: %v", err)
	}

	openRecorder := httptest.NewRecorder()
	newHandler("correct-token").ServeHTTP(openRecorder, newJSONRequest(t, http.MethodPost, "/api/project/open", projectOpenRequest{Path: repoRoot}, "correct-token"))
	if openRecorder.Code != http.StatusOK {
		t.Fatalf("open status = %d, want %d: %s", openRecorder.Code, http.StatusOK, openRecorder.Body.String())
	}
	openBody := openRecorder.Body.String()
	var openResponse projectOpenResponse
	if err := json.NewDecoder(strings.NewReader(openBody)).Decode(&openResponse); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if openResponse.Settings.Rules == nil || len(*openResponse.Settings.Rules) != 1 ||
		(*openResponse.Settings.Rules)[0].Match == nil || *(*openResponse.Settings.Rules)[0].Match != "old/**" {
		t.Fatalf("open rules = %#v", openResponse.Settings.Rules)
	}
	if openResponse.Effective.Rules == nil || len(*openResponse.Effective.Rules) != 1 ||
		(*openResponse.Effective.Rules)[0].Match == nil || *(*openResponse.Effective.Rules)[0].Match != "old/**" ||
		(*openResponse.Effective.Rules)[0].Render == nil || *(*openResponse.Effective.Rules)[0].Render {
		t.Fatalf("effective open rules = %#v", openResponse.Effective.Rules)
	}
	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal([]byte(openBody), &topLevel); err != nil {
		t.Fatalf("decode open response keys: %v", err)
	}
	for _, key := range []string{"source", "output", "access"} {
		if _, ok := topLevel[key]; ok {
			t.Fatalf("open response exposed trusted top-level setting %q: %s", key, openBody)
		}
	}
	for _, secret := range []string{"trusted/source.git", "trusted-dist", "private/**"} {
		if strings.Contains(openBody, secret) {
			t.Fatalf("open response exposed trusted setting value %q: %s", secret, openBody)
		}
	}

	trustedSettingsRecorder := httptest.NewRecorder()
	trustedSettingsRequest := map[string]any{
		"path": repoRoot,
		"settings": map[string]any{
			"source": map[string]any{"repo": "attacker-controlled"},
			"output": map[string]any{"dir": "attacker-controlled"},
			"access": map[string]any{"noindex": false},
		},
	}
	newHandler("correct-token").ServeHTTP(trustedSettingsRecorder, newJSONRequest(t, http.MethodPost, "/api/config/validate", trustedSettingsRequest, "correct-token"))
	if trustedSettingsRecorder.Code != http.StatusBadRequest {
		t.Fatalf("trusted settings status = %d, want %d: %s", trustedSettingsRecorder.Code, http.StatusBadRequest, trustedSettingsRecorder.Body.String())
	}
	assertFileContent(t, configPath, original)

	title := "After"
	match := "docs/**"
	math := true
	rules := []config.RepositoryRuleSettings{{
		Match:    &match,
		Markdown: config.RepositoryMarkdownOptionsSettings{Math: &math},
	}}
	request := configRequest{
		Path:     repoRoot,
		Revision: openResponse.Revision,
		Settings: repositorySettingsFromConfig(config.RepositorySettings{
			Site:  config.RepositorySiteSettings{Title: &title},
			Rules: &rules,
		}),
	}

	validateRecorder := httptest.NewRecorder()
	newHandler("correct-token").ServeHTTP(validateRecorder, newJSONRequest(t, http.MethodPost, "/api/config/validate", request, "correct-token"))
	if validateRecorder.Code != http.StatusOK {
		t.Fatalf("validate status = %d, want %d: %s", validateRecorder.Code, http.StatusOK, validateRecorder.Body.String())
	}
	var validateResponse configResponse
	if err := json.NewDecoder(validateRecorder.Body).Decode(&validateResponse); err != nil {
		t.Fatalf("decode validate response: %v", err)
	}
	if validateResponse.Settings.Site.Title == nil || *validateResponse.Settings.Site.Title != title ||
		validateResponse.Settings.Rules == nil || len(*validateResponse.Settings.Rules) != 1 ||
		(*validateResponse.Settings.Rules)[0].Match == nil || *(*validateResponse.Settings.Rules)[0].Match != match {
		t.Fatalf("validate response settings = %#v", validateResponse.Settings)
	}
	assertFileContent(t, configPath, original)

	prepareRecorder := httptest.NewRecorder()
	newHandler("correct-token").ServeHTTP(prepareRecorder, newJSONRequest(t, http.MethodPost, "/api/config/prepare-write", request, "correct-token"))
	if prepareRecorder.Code != http.StatusOK {
		t.Fatalf("prepare status = %d, want %d: %s", prepareRecorder.Code, http.StatusOK, prepareRecorder.Body.String())
	}
	var prepareResponse prepareWriteResponse
	if err := json.NewDecoder(prepareRecorder.Body).Decode(&prepareResponse); err != nil {
		t.Fatalf("decode prepare response: %v", err)
	}
	for _, want := range []string{"title: After", "match: docs/**", "math: true"} {
		if !strings.Contains(prepareResponse.After, want) {
			t.Errorf("prepare after does not contain %q:\n%s", want, prepareResponse.After)
		}
	}
	for _, forbidden := range []string{"source:", "output:", "access:", "trusted/source.git", "trusted-dist", "private/**"} {
		if strings.Contains(prepareResponse.Before, forbidden) || strings.Contains(prepareResponse.After, forbidden) || strings.Contains(prepareResponse.Diff, forbidden) {
			t.Errorf("prepare response exposed trusted setting %q", forbidden)
		}
	}
	assertFileContent(t, configPath, original)

	request.Confirm = true
	commitRecorder := httptest.NewRecorder()
	newHandler("correct-token").ServeHTTP(commitRecorder, newJSONRequest(t, http.MethodPost, "/api/config/commit", request, "correct-token"))
	if commitRecorder.Code != http.StatusOK {
		t.Fatalf("commit status = %d, want %d: %s", commitRecorder.Code, http.StatusOK, commitRecorder.Body.String())
	}
	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read committed config: %v", err)
	}
	for _, want := range []string{"repo: git@example.com:trusted/source.git", "dir: trusted-dist", "noindex: true", "paths: [private/**]", "title: After", "match: docs/**", "math: true"} {
		if !strings.Contains(string(written), want) {
			t.Errorf("committed config does not contain %q:\n%s", want, written)
		}
	}
}

func TestRepositorySettingsMapsEveryRepositoryDomain(t *testing.T) {
	title := "Project"
	ignore := []string{"generated/**"}
	render := false
	toc := true
	tocMinHeadings := 3
	anchors := true
	mermaid := true
	math := true
	frontmatterTitle := true
	htmlView := "embed"
	lineNumbers := true
	codeTheme := "github"
	maxFileSize := config.RepositoryByteSize(1024)
	match := "docs/**"
	vars := map[string]string{"accent": "red"}
	css := "theme.css"
	templates := "templates/"
	treePosition := "left"
	treeDepth := 2
	tocPanel := "floating"
	search := true
	llmsTxt := true
	llmsFull := true
	llmsFullMaxSize := config.RepositoryByteSize(2048)
	indexJSON := true
	settings := config.RepositorySettings{
		Ignore: &ignore,
		Render: config.RepositoryFileOptionsSettings{
			Render:      &render,
			Markdown:    config.RepositoryMarkdownOptionsSettings{TOC: &toc, TOCMinHeadings: &tocMinHeadings, Anchors: &anchors, Mermaid: &mermaid, Math: &math, FrontmatterTitle: &frontmatterTitle},
			HTML:        config.RepositoryHTMLOptionsSettings{View: &htmlView},
			Code:        config.RepositoryCodeOptionsSettings{LineNumbers: &lineNumbers, Theme: &codeTheme},
			MaxFileSize: &maxFileSize,
		},
		Rules: &[]config.RepositoryRuleSettings{{Match: &match, Render: &render, Markdown: config.RepositoryMarkdownOptionsSettings{Math: &math}, HTML: config.RepositoryHTMLOptionsSettings{View: &htmlView}, Code: config.RepositoryCodeOptionsSettings{Theme: &codeTheme}, MaxFileSize: &maxFileSize}},
		Theme: config.RepositoryThemeSettings{Vars: &vars, CSS: &css, Templates: &templates},
		Site:  config.RepositorySiteSettings{Title: &title},
		View:  config.RepositoryViewSettings{TreePosition: &treePosition, TreeExpandDepth: &treeDepth, TOCPanel: &tocPanel, Search: &search},
		Agent: config.RepositoryAgentSettings{LLMSTxt: &llmsTxt, LLMSFull: config.RepositoryAgentFullTextSettings{Enabled: &llmsFull, MaxSize: &llmsFullMaxSize}, IndexJSON: &indexJSON},
	}

	if got := repositorySettingsFromConfig(settings).config(); !reflect.DeepEqual(got, settings) {
		t.Fatalf("repository settings round trip = %#v, want %#v", got, settings)
	}
}

func TestConfigValidateRejectsSettingsIncompatibleWithDocument(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".repolens.yml")
	original := "site: null\n"
	if err := os.WriteFile(configPath, []byte(original), 0o644); err != nil {
		t.Fatalf("write repository config: %v", err)
	}
	title := "After"
	recorder := httptest.NewRecorder()
	request := configRequest{
		Path:     repoRoot,
		Settings: repositorySettingsFromConfig(config.RepositorySettings{Site: config.RepositorySiteSettings{Title: &title}}),
	}
	newHandler("correct-token").ServeHTTP(recorder, newJSONRequest(t, http.MethodPost, "/api/config/validate", request, "correct-token"))
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("validate status = %d, want %d: %s", recorder.Code, http.StatusUnprocessableEntity, recorder.Body.String())
	}
	var response struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Code != "validation_failed" {
		t.Fatalf("error code = %q, want validation_failed", response.Code)
	}
	assertFileContent(t, configPath, original)
}

func TestConfigEndpointsReturnStructuredValidationIssues(t *testing.T) {
	repoRoot := t.TempDir()
	invalidPattern := "["
	negativeSize := config.RepositoryByteSize(-1)
	invalidView := "unsupported"
	request := configRequest{
		Path: repoRoot,
		Settings: repositorySettingsFromConfig(config.RepositorySettings{
			Ignore: &[]string{invalidPattern},
			Render: config.RepositoryFileOptionsSettings{
				MaxFileSize: &negativeSize,
				HTML:        config.RepositoryHTMLOptionsSettings{View: &invalidView},
			},
		}),
	}
	document, err := config.LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("load document: %v", err)
	}
	request.Revision = document.Revision

	for _, endpoint := range []string{"/api/config/validate", "/api/config/prepare-write", "/api/config/commit"} {
		t.Run(endpoint, func(t *testing.T) {
			endpointRequest := request
			endpointRequest.Confirm = endpoint == "/api/config/commit"
			recorder := httptest.NewRecorder()
			newHandler("correct-token").ServeHTTP(recorder, newJSONRequest(t, http.MethodPost, endpoint, endpointRequest, "correct-token"))
			if recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusUnprocessableEntity, recorder.Body.String())
			}
			var response struct {
				Code   string            `json:"code"`
				Field  string            `json:"field"`
				Issues []validationIssue `json:"issues"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode validation response: %v", err)
			}
			if response.Code != "validation_failed" || response.Field != "ignore[0]" || len(response.Issues) != 3 {
				t.Fatalf("validation response = %#v", response)
			}
			for _, issue := range response.Issues {
				if issue.Path == "" || issue.Code == "" || issue.Message == "" || issue.Severity != "error" {
					t.Fatalf("validation issue = %#v", issue)
				}
			}
		})
	}
}

func TestConfigCommitClearsUnsetFieldsBackToDefaults(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".repolens.yml")
	if err := os.WriteFile(configPath, []byte("site:\n  title: Configured title\nrender:\n  markdown:\n    math: true\n"), 0o644); err != nil {
		t.Fatalf("write repository config: %v", err)
	}

	openRecorder := httptest.NewRecorder()
	newHandler("correct-token").ServeHTTP(openRecorder, newJSONRequest(t, http.MethodPost, "/api/project/open", projectOpenRequest{Path: repoRoot}, "correct-token"))
	if openRecorder.Code != http.StatusOK {
		t.Fatalf("open status = %d, want %d: %s", openRecorder.Code, http.StatusOK, openRecorder.Body.String())
	}
	var openResponse projectOpenResponse
	if err := json.NewDecoder(openRecorder.Body).Decode(&openResponse); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	settings := openResponse.Settings.config()
	settings.Site.Title = nil
	settings.Render.Markdown.Math = nil

	commitRecorder := httptest.NewRecorder()
	newHandler("correct-token").ServeHTTP(commitRecorder, newJSONRequest(t, http.MethodPost, "/api/config/commit", configRequest{
		Path: repoRoot, Revision: openResponse.Revision, Confirm: true, Settings: repositorySettingsFromConfig(settings),
	}, "correct-token"))
	if commitRecorder.Code != http.StatusOK {
		t.Fatalf("commit status = %d, want %d: %s", commitRecorder.Code, http.StatusOK, commitRecorder.Body.String())
	}
	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read committed config: %v", err)
	}
	if strings.Contains(string(written), "title:") || strings.Contains(string(written), "math:") {
		t.Fatalf("unset fields remained in YAML:\n%s", written)
	}
	effective, _, err := config.Load(repoRoot, "", config.Flags{})
	if err != nil {
		t.Fatalf("load effective config: %v", err)
	}
	if effective.Site.Title != "" || effective.Render.Markdown.Math {
		t.Fatalf("effective config did not return to defaults: %#v", effective)
	}
}

func TestConfigCommitClearsUnsetRuleFieldsAndPreservesUnknownNodes(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".repolens.yml")
	if err := os.WriteFile(configPath, []byte("rules:\n  - match: docs/**\n    markdown:\n      math: true\n    extension: retained\n"), 0o644); err != nil {
		t.Fatalf("write repository config: %v", err)
	}
	document, err := config.LoadRepositoryDocument(repoRoot)
	if err != nil {
		t.Fatalf("load document: %v", err)
	}
	match := "docs/**"
	settings := config.RepositorySettings{Rules: &[]config.RepositoryRuleSettings{{Match: &match}}}
	if err := document.Replace(settings); err != nil {
		t.Fatalf("Replace() error: %v", err)
	}
	if err := document.Write(); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read rewritten config: %v", err)
	}
	if strings.Contains(string(written), "math:") || !strings.Contains(string(written), "extension: retained") {
		t.Fatalf("rule replace did not clear controlled field and preserve unknown node:\n%s", written)
	}
}

func TestConfigCommitRequiresConfirmationAndRevision(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".repolens.yml")
	if err := os.WriteFile(configPath, []byte("site:\n  title: Before\n"), 0o644); err != nil {
		t.Fatalf("write repository config: %v", err)
	}

	openRecorder := httptest.NewRecorder()
	newHandler("correct-token").ServeHTTP(openRecorder, newJSONRequest(t, http.MethodPost, "/api/project/open", projectOpenRequest{Path: repoRoot}, "correct-token"))
	if openRecorder.Code != http.StatusOK {
		t.Fatalf("open status = %d, want %d: %s", openRecorder.Code, http.StatusOK, openRecorder.Body.String())
	}
	var openResponse projectOpenResponse
	if err := json.NewDecoder(openRecorder.Body).Decode(&openResponse); err != nil {
		t.Fatalf("decode open response: %v", err)
	}

	title := "After"
	request := configRequest{
		Path:     repoRoot,
		Revision: openResponse.Revision,
		Settings: repositorySettingsFromConfig(config.RepositorySettings{
			Site: config.RepositorySiteSettings{Title: &title},
		}),
	}
	confirmationRecorder := httptest.NewRecorder()
	newHandler("correct-token").ServeHTTP(confirmationRecorder, newJSONRequest(t, http.MethodPost, "/api/config/commit", request, "correct-token"))
	if confirmationRecorder.Code != http.StatusBadRequest {
		t.Fatalf("unconfirmed commit status = %d, want %d", confirmationRecorder.Code, http.StatusBadRequest)
	}
	assertFileContent(t, configPath, "site:\n  title: Before\n")

	request.Confirm = true
	commitRecorder := httptest.NewRecorder()
	newHandler("correct-token").ServeHTTP(commitRecorder, newJSONRequest(t, http.MethodPost, "/api/config/commit", request, "correct-token"))
	if commitRecorder.Code != http.StatusOK {
		t.Fatalf("commit status = %d, want %d: %s", commitRecorder.Code, http.StatusOK, commitRecorder.Body.String())
	}
	var commitResponse configResponse
	if err := json.NewDecoder(commitRecorder.Body).Decode(&commitResponse); err != nil {
		t.Fatalf("decode commit response: %v", err)
	}
	if commitResponse.Revision == openResponse.Revision {
		t.Fatal("commit response did not return a new revision")
	}
	assertFileContent(t, configPath, "site:\n  title: After\n")
}

func TestConfigCommitRejectsRevisionConflict(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".repolens.yml")
	if err := os.WriteFile(configPath, []byte("site:\n  title: Before\n"), 0o644); err != nil {
		t.Fatalf("write repository config: %v", err)
	}

	openRecorder := httptest.NewRecorder()
	newHandler("correct-token").ServeHTTP(openRecorder, newJSONRequest(t, http.MethodPost, "/api/project/open", projectOpenRequest{Path: repoRoot}, "correct-token"))
	var openResponse projectOpenResponse
	if err := json.NewDecoder(openRecorder.Body).Decode(&openResponse); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("site:\n  title: External\n"), 0o644); err != nil {
		t.Fatalf("write external change: %v", err)
	}
	title := "After"
	recorder := httptest.NewRecorder()
	request := configRequest{
		Path:     repoRoot,
		Revision: openResponse.Revision,
		Confirm:  true,
		Settings: repositorySettingsFromConfig(config.RepositorySettings{Site: config.RepositorySiteSettings{Title: &title}}),
	}
	newHandler("correct-token").ServeHTTP(recorder, newJSONRequest(t, http.MethodPost, "/api/config/commit", request, "correct-token"))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("commit status = %d, want %d: %s", recorder.Code, http.StatusConflict, recorder.Body.String())
	}
	var response struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Code != "revision_conflict" {
		t.Fatalf("error code = %q, want revision_conflict", response.Code)
	}
	assertFileContent(t, configPath, "site:\n  title: External\n")
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if got := string(data); got != want {
		t.Fatalf("file content = %q, want %q", got, want)
	}
}

func newJSONRequest(t *testing.T, method, target string, body any, token string) *http.Request {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set(csrfTokenHeader, token)
	}
	return request
}
