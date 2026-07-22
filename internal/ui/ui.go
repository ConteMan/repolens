package ui

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/ConteMan/repolens/internal/site"
	"github.com/goccy/go-yaml"
)

const (
	defaultAddr     = "127.0.0.1:8799"
	csrfTokenHeader = "X-Repolens-CSRF-Token"
)

var errNonLoopbackAddress = errors.New("ui: address must use a loopback IP")

//go:embed dist
var frontendFS embed.FS

var (
	shellHTML      = mustReadFrontendFile("dist/index.html")
	frontendAssets = http.FileServer(http.FS(mustFrontendSub("dist/assets")))
)

// Options configures the local UI HTTP server.
type Options struct {
	Addr string
}

// Run starts the local UI server and blocks until ctx is canceled.
func Run(ctx context.Context, opts Options) error {
	addr, err := loopbackAddr(opts.Addr)
	if err != nil {
		return err
	}

	token, err := newToken()
	if err != nil {
		return fmt.Errorf("ui: create CSRF token: %w", err)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler: newHandlerWithBuildService(token, newBuildService(ctx)),
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
		}
		err := <-serveErr
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func loopbackAddr(addr string) (string, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = defaultAddr
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("ui: invalid address %q: %w", addr, err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", fmt.Errorf("%w: %q", errNonLoopbackAddress, addr)
	}
	return addr, nil
}

func newToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// newHandler constructs the HTTP surface.
func newHandler(token string) http.Handler {
	return newHandlerWithBuildService(token, newBuildService(context.Background()))
}

func newHandlerWithBuildService(token string, builds *buildService) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", frontendAssets))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		serveShell(w, token)
	})
	mux.HandleFunc("GET /api/health", health)
	mux.Handle("POST /api/project/open", requireToken(token, http.HandlerFunc(openProject)))
	mux.Handle("POST /api/config/validate", requireToken(token, http.HandlerFunc(validateConfig)))
	mux.Handle("POST /api/config/prepare-write", requireToken(token, http.HandlerFunc(prepareWrite)))
	mux.Handle("POST /api/config/commit", requireToken(token, http.HandlerFunc(commitConfig)))
	mux.Handle("POST /api/build", requireToken(token, http.HandlerFunc(builds.startBuild)))
	mux.HandleFunc("GET /api/build/{id}", builds.getBuild)

	return securityHeaders(mux)
}

type buildRequest struct {
	Path string `json:"path"`
}

type buildResponse struct {
	ID         string      `json:"id"`
	Stage      string      `json:"stage"`
	Stats      *site.Stats `json:"stats,omitempty"`
	Warnings   []string    `json:"warnings,omitempty"`
	Error      string      `json:"error,omitempty"`
	OutputPath string      `json:"output_path,omitempty"`
}

func (s *buildService) startBuild(w http.ResponseWriter, r *http.Request) {
	var request buildRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body must contain one object")
		return
	}
	document, err := loadDocument(request.Path)
	if err != nil {
		writeDocumentError(w, err)
		return
	}
	op, err := s.start(filepath.Dir(document.Path))
	if err != nil {
		if errors.Is(err, errBuildInProgress) {
			writeError(w, http.StatusConflict, "build_in_progress", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "build_start_failed", "unable to start build")
		return
	}
	writeJSON(w, http.StatusAccepted, buildResponseFromOperation(op))
}

func (s *buildService) getBuild(w http.ResponseWriter, r *http.Request) {
	op, ok := s.operation(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "build_not_found", "build operation was not found")
		return
	}
	writeJSON(w, http.StatusOK, buildResponseFromOperation(op))
}

func buildResponseFromOperation(op buildOperation) buildResponse {
	return buildResponse{
		ID:         op.ID,
		Stage:      op.Stage,
		Stats:      op.Stats,
		Warnings:   op.Warnings,
		Error:      op.Error,
		OutputPath: op.OutputPath,
	}
}

func serveShell(w http.ResponseWriter, token string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(strings.ReplaceAll(shellHTML, "__REPOLENS_CSRF_TOKEN__", token)))
}

func mustReadFrontendFile(name string) string {
	content, err := frontendFS.ReadFile(name)
	if err != nil {
		panic(fmt.Sprintf("ui: read embedded frontend file %q: %v", name, err))
	}
	return string(content)
}

func mustFrontendSub(directory string) fs.FS {
	root, err := fs.Sub(frontendFS, directory)
	if err != nil {
		panic(fmt.Sprintf("ui: open embedded frontend directory %q: %v", directory, err))
	}
	return root
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

type projectOpenRequest struct {
	Path string `json:"path"`
}

type projectOpenResponse struct {
	Settings  repositorySettings `json:"settings"`
	Effective repositorySettings `json:"effective"`
	Sources   map[string]string  `json:"sources"`
	Warnings  []string           `json:"warnings"`
	Revision  string             `json:"revision"`
}

func openProject(w http.ResponseWriter, r *http.Request) {
	var request projectOpenRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body must contain one object")
		return
	}

	document, effective, warnings, err := loadProjectSnapshot(request.Path)
	if err != nil {
		if errors.Is(err, errInvalidPath) {
			writeDocumentError(w, err)
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "project_open_failed", "unable to load effective repository configuration")
		return
	}

	writeJSON(w, http.StatusOK, projectOpenResponse{
		Settings:  repositorySettingsFromConfig(document.Settings),
		Effective: repositorySettingsFromEffectiveConfig(*effective, document.Settings.Rules),
		Sources:   repositorySettingSources(document.Settings),
		Warnings:  configWarningMessages(warnings),
		Revision:  document.Revision,
	})
}

func loadProjectSnapshot(repoPath string) (*config.RepositoryDocument, *config.Config, []config.Warning, error) {
	return loadProjectSnapshotWith(repoPath, loadDocument, config.Load)
}

func loadProjectSnapshotWith(
	repoPath string,
	documentLoader func(string) (*config.RepositoryDocument, error),
	effectiveLoader func(string, string, config.Flags) (*config.Config, []config.Warning, error),
) (*config.RepositoryDocument, *config.Config, []config.Warning, error) {
	for range 3 {
		before, err := documentLoader(repoPath)
		if err != nil {
			return nil, nil, nil, err
		}
		effective, warnings, err := effectiveLoader(repoPath, "", config.Flags{})
		if err != nil {
			return nil, nil, nil, err
		}
		after, err := documentLoader(repoPath)
		if err != nil {
			return nil, nil, nil, err
		}
		if before.Revision == after.Revision {
			return after, effective, warnings, nil
		}
	}
	return nil, nil, nil, errors.New("repository configuration changed while loading")
}

type configRequest struct {
	Path     string             `json:"path"`
	Settings repositorySettings `json:"settings"`
	Revision string             `json:"revision"`
	Confirm  bool               `json:"confirm"`
}

type configResponse struct {
	Settings repositorySettings `json:"settings"`
	Revision string             `json:"revision"`
}

// validationIssue is an API-facing field diagnostic. Severity allows callers
// to keep blocking validation failures separate from non-blocking warnings.
type validationIssue struct {
	Path     string `json:"path"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// repositorySettings defines the JSON representation accepted by the UI API.
// Its fields are intentionally identical to config.RepositorySettings, which
// contains only repository-trust-domain configuration.
type repositorySettings config.RepositorySettings

func repositorySettingsFromConfig(settings config.RepositorySettings) repositorySettings {
	return repositorySettings(settings)
}

func (settings repositorySettings) config() config.RepositorySettings {
	return config.RepositorySettings(settings)
}

func repositorySettingsFromEffectiveConfig(effective config.Config, repositoryRules *[]config.RepositoryRuleSettings) repositorySettings {
	copyStrings := append([]string(nil), effective.Ignore...)
	var rules *[]config.RepositoryRuleSettings
	if repositoryRules != nil {
		copyRules := append([]config.RepositoryRuleSettings(nil), (*repositoryRules)...)
		rules = &copyRules
	}
	return repositorySettings{
		Site: config.RepositorySiteSettings{
			Title: stringPointer(effective.Site.Title), Language: stringPointer(effective.Site.Language), Home: stringPointer(effective.Site.Home),
		},
		Ignore: &copyStrings,
		Render: config.RepositoryFileOptionsSettings{
			Render: boolPointer(effective.Render.Render),
			Markdown: config.RepositoryMarkdownOptionsSettings{
				TOC: boolPointer(effective.Render.Markdown.TOC), TOCMinHeadings: intPointer(effective.Render.Markdown.TOCMinHeadings),
				Anchors: boolPointer(effective.Render.Markdown.Anchors), Mermaid: boolPointer(effective.Render.Markdown.Mermaid),
				Math: boolPointer(effective.Render.Markdown.Math), FrontmatterTitle: boolPointer(effective.Render.Markdown.FrontmatterTitle),
			},
			HTML:        config.RepositoryHTMLOptionsSettings{View: stringPointer(effective.Render.HTML.View)},
			Code:        config.RepositoryCodeOptionsSettings{LineNumbers: boolPointer(effective.Render.Code.LineNumbers), Theme: stringPointer(effective.Render.Code.Theme)},
			MaxFileSize: byteSizePointer(effective.Render.MaxFileSize),
		},
		Rules: rules,
		Theme: config.RepositoryThemeSettings{Vars: mapPointer(effective.Theme.Vars), CSS: stringPointer(effective.Theme.CSS), Templates: stringPointer(effective.Theme.Templates)},
		View: config.RepositoryViewSettings{
			TreePosition: stringPointer(effective.View.TreePosition), TreeExpandDepth: intPointer(effective.View.TreeExpandDepth),
			TOCPanel: stringPointer(effective.View.TOCPanel), Search: boolPointer(effective.View.Search),
		},
		Agent: config.RepositoryAgentSettings{
			LLMSTxt:   boolPointer(effective.Agent.LLMSTxt),
			LLMSFull:  config.RepositoryAgentFullTextSettings{Enabled: boolPointer(effective.Agent.LLMSFull.Enabled), MaxSize: byteSizePointer(effective.Agent.LLMSFull.MaxSize)},
			IndexJSON: boolPointer(effective.Agent.IndexJSON),
		},
	}
}

func stringPointer(value string) *string { return &value }
func boolPointer(value bool) *bool       { return &value }
func intPointer(value int) *int          { return &value }

func byteSizePointer(value int64) *config.RepositoryByteSize {
	size := config.RepositoryByteSize(value)
	return &size
}

func mapPointer(value map[string]string) *map[string]string {
	if value == nil {
		return nil
	}
	copyValue := make(map[string]string, len(value))
	for key, item := range value {
		copyValue[key] = item
	}
	return &copyValue
}

func repositorySettingSources(settings config.RepositorySettings) map[string]string {
	sources := make(map[string]string)
	mark := func(path string, value any) {
		if !isNil(value) {
			sources[path] = "repository"
			return
		}
		sources[path] = "default"
	}
	mark("site.title", settings.Site.Title)
	mark("site.language", settings.Site.Language)
	mark("site.home", settings.Site.Home)
	mark("ignore", settings.Ignore)
	mark("render.render", settings.Render.Render)
	mark("render.markdown.toc", settings.Render.Markdown.TOC)
	mark("render.markdown.toc_min_headings", settings.Render.Markdown.TOCMinHeadings)
	mark("render.markdown.anchors", settings.Render.Markdown.Anchors)
	mark("render.markdown.mermaid", settings.Render.Markdown.Mermaid)
	mark("render.markdown.math", settings.Render.Markdown.Math)
	mark("render.markdown.frontmatter_title", settings.Render.Markdown.FrontmatterTitle)
	mark("render.html.view", settings.Render.HTML.View)
	mark("render.code.line_numbers", settings.Render.Code.LineNumbers)
	mark("render.code.theme", settings.Render.Code.Theme)
	mark("render.max_file_size", settings.Render.MaxFileSize)
	mark("theme.vars", settings.Theme.Vars)
	mark("theme.css", settings.Theme.CSS)
	mark("theme.templates", settings.Theme.Templates)
	mark("view.tree_position", settings.View.TreePosition)
	mark("view.tree_expand_depth", settings.View.TreeExpandDepth)
	mark("view.toc_panel", settings.View.TOCPanel)
	mark("view.search", settings.View.Search)
	mark("agent.llms_txt", settings.Agent.LLMSTxt)
	mark("agent.llms_full.enabled", settings.Agent.LLMSFull.Enabled)
	mark("agent.llms_full.max_size", settings.Agent.LLMSFull.MaxSize)
	mark("agent.index_json", settings.Agent.IndexJSON)
	if settings.Rules != nil {
		sources["rules"] = "repository"
		for index, rule := range *settings.Rules {
			prefix := fmt.Sprintf("rules[%d]", index)
			mark(prefix+".match", rule.Match)
			mark(prefix+".render", rule.Render)
			mark(prefix+".markdown.toc", rule.Markdown.TOC)
			mark(prefix+".markdown.toc_min_headings", rule.Markdown.TOCMinHeadings)
			mark(prefix+".markdown.anchors", rule.Markdown.Anchors)
			mark(prefix+".markdown.mermaid", rule.Markdown.Mermaid)
			mark(prefix+".markdown.math", rule.Markdown.Math)
			mark(prefix+".markdown.frontmatter_title", rule.Markdown.FrontmatterTitle)
			mark(prefix+".html.view", rule.HTML.View)
			mark(prefix+".code.line_numbers", rule.Code.LineNumbers)
			mark(prefix+".code.theme", rule.Code.Theme)
			mark(prefix+".max_file_size", rule.MaxFileSize)
		}
	} else {
		sources["rules"] = "default"
	}
	return sources
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	return reflected.Kind() == reflect.Ptr && reflected.IsNil()
}

func configWarningMessages(warnings []config.Warning) []string {
	messages := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		messages = append(messages, warning.Msg)
	}
	return messages
}

type prepareWriteResponse struct {
	configResponse
	Before string `json:"before"`
	After  string `json:"after"`
	Diff   string `json:"diff"`
}

func validateConfig(w http.ResponseWriter, r *http.Request) {
	document, request, ok := loadConfigRequest(w, r)
	if !ok {
		return
	}
	if !applyRequestedSettings(w, document, request.Settings.config()) {
		return
	}
	writeJSON(w, http.StatusOK, configResponse{Settings: repositorySettingsFromConfig(document.Settings), Revision: document.Revision})
}

func prepareWrite(w http.ResponseWriter, r *http.Request) {
	document, request, ok := loadConfigRequest(w, r)
	if !ok {
		return
	}
	before, err := os.ReadFile(document.Path)
	if err != nil && !os.IsNotExist(err) {
		writeError(w, http.StatusUnprocessableEntity, "prepare_write_failed", "unable to read repository configuration")
		return
	}
	if !applyRequestedSettings(w, document, request.Settings.config()) {
		return
	}
	after, err := document.YAML()
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "prepare_write_failed", "unable to serialize repository configuration")
		return
	}
	previewBefore, err := redactTrustedSettings(before)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "prepare_write_failed", "unable to prepare repository configuration preview")
		return
	}
	previewAfter, err := redactTrustedSettings(after)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "prepare_write_failed", "unable to prepare repository configuration preview")
		return
	}
	writeJSON(w, http.StatusOK, prepareWriteResponse{
		configResponse: configResponse{Settings: repositorySettingsFromConfig(document.Settings), Revision: document.Revision},
		Before:         previewBefore,
		After:          previewAfter,
		Diff:           unifiedDiff(previewBefore, previewAfter, document.Path),
	})
}

func redactTrustedSettings(data []byte) (string, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return "", nil
	}
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return "", err
	}
	delete(document, "source")
	delete(document, "output")
	delete(document, "access")
	preview, err := yaml.Marshal(document)
	if err != nil {
		return "", err
	}
	return string(preview), nil
}

func commitConfig(w http.ResponseWriter, r *http.Request) {
	document, request, ok := loadConfigRequest(w, r)
	if !ok {
		return
	}
	if !request.Confirm {
		writeError(w, http.StatusBadRequest, "confirmation_required", "commit requires confirm: true")
		return
	}
	if request.Revision == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "commit requires a revision")
		return
	}
	if request.Revision != document.Revision {
		writeError(w, http.StatusConflict, "revision_conflict", "repository configuration changed; reopen the project")
		return
	}
	if !applyRequestedSettings(w, document, request.Settings.config()) {
		return
	}
	if err := document.Write(); err != nil {
		if errors.Is(err, config.ErrRevisionConflict) {
			writeError(w, http.StatusConflict, "revision_conflict", "repository configuration changed; reopen the project")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "commit_failed", "unable to write repository configuration")
		return
	}
	writeJSON(w, http.StatusOK, configResponse{Settings: repositorySettingsFromConfig(document.Settings), Revision: document.Revision})
}

func applyRequestedSettings(w http.ResponseWriter, document *config.RepositoryDocument, settings config.RepositorySettings) bool {
	if issues := config.ValidateRepositorySettings(settings); len(issues) > 0 {
		writeValidationError(w, issues)
		return false
	}
	if err := document.Replace(settings); err != nil {
		writeValidationIssues(w, []validationIssue{{
			Code: "apply_failed", Message: "settings cannot be applied to the repository document", Severity: "error",
		}})
		return false
	}
	return true
}

func writeValidationError(w http.ResponseWriter, issues []config.RepositoryValidationIssue) {
	apiIssues := make([]validationIssue, 0, len(issues))
	for _, issue := range issues {
		apiIssues = append(apiIssues, validationIssue{Path: issue.Path, Code: issue.Code, Message: issue.Message, Severity: "error"})
	}
	writeValidationIssues(w, apiIssues)
}

func writeValidationIssues(w http.ResponseWriter, issues []validationIssue) {
	message := "settings contain validation errors"
	if len(issues) > 0 && issues[0].Message != "" {
		message = issues[0].Message
	}
	writeErrorDetails(w, http.StatusUnprocessableEntity, "validation_failed", message, issues, nil)
}

func loadConfigRequest(w http.ResponseWriter, r *http.Request) (*config.RepositoryDocument, configRequest, bool) {
	var request configRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body must contain one object")
		return nil, configRequest{}, false
	}
	document, err := loadDocument(request.Path)
	if err != nil {
		writeDocumentError(w, err)
		return nil, configRequest{}, false
	}
	return document, request, true
}

func loadDocument(repoPath string) (*config.RepositoryDocument, error) {
	repoPath = filepath.Clean(repoPath)
	if !filepath.IsAbs(repoPath) {
		return nil, errInvalidPath
	}
	info, err := os.Stat(repoPath)
	if err != nil || !info.IsDir() {
		return nil, errInvalidPath
	}
	document, err := config.LoadRepositoryDocument(repoPath)
	if err != nil {
		return nil, fmt.Errorf("load repository document: %w", err)
	}
	return document, nil
}

var errInvalidPath = errors.New("invalid repository path")

func writeDocumentError(w http.ResponseWriter, err error) {
	if errors.Is(err, errInvalidPath) {
		writeError(w, http.StatusBadRequest, "invalid_path", "path must name an existing absolute directory")
		return
	}
	writeError(w, http.StatusUnprocessableEntity, "project_open_failed", "unable to load repository document")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request contains more than one JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func unifiedDiff(before, after, filePath string) string {
	if before == after {
		return ""
	}
	beforeLines := strings.Split(strings.TrimSuffix(before, "\n"), "\n")
	afterLines := strings.Split(strings.TrimSuffix(after, "\n"), "\n")
	if before == "" {
		beforeLines = nil
	}
	if after == "" {
		afterLines = nil
	}
	var diff strings.Builder
	fmt.Fprintf(&diff, "--- %s\n+++ %s\n", filePath, filePath)
	fmt.Fprintf(&diff, "@@ -%d,%d +%d,%d @@\n", hunkStart(len(beforeLines)), len(beforeLines), hunkStart(len(afterLines)), len(afterLines))
	for _, line := range beforeLines {
		fmt.Fprintf(&diff, "-%s\n", line)
	}
	for _, line := range afterLines {
		fmt.Fprintf(&diff, "+%s\n", line)
	}
	return diff.String()
}

func hunkStart(lineCount int) int {
	if lineCount == 0 {
		return 0
	}
	return 1
}

func requireToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get(csrfTokenHeader)), []byte(token)) != 1 {
			writeError(w, http.StatusForbidden, "invalid_token", "a valid CSRF token is required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeErrorDetails(w, status, code, message, nil, nil)
}

func writeErrorDetails(w http.ResponseWriter, status int, code, message string, issues, warnings []validationIssue) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code     string            `json:"code"`
		Message  string            `json:"message"`
		Field    string            `json:"field,omitempty"`
		Issues   []validationIssue `json:"issues,omitempty"`
		Warnings []validationIssue `json:"warnings,omitempty"`
	}{Code: code, Message: message, Field: firstIssuePath(issues), Issues: issues, Warnings: warnings})
}

func firstIssuePath(issues []validationIssue) string {
	if len(issues) == 0 {
		return ""
	}
	return issues[0].Path
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; connect-src 'self'; script-src 'self'; style-src 'self'; base-uri 'none'; form-action 'self'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
