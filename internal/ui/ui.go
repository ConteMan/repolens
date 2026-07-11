package ui

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
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

//go:embed assets/index.html
var shellHTML string

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
	_, _ = w.Write([]byte(strings.ReplaceAll(shellHTML, "{{CSRF_TOKEN}}", token)))
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

type projectOpenRequest struct {
	Path string `json:"path"`
}

type projectOpenResponse struct {
	Settings repositorySettings `json:"settings"`
	Revision string             `json:"revision"`
}

func openProject(w http.ResponseWriter, r *http.Request) {
	var request projectOpenRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request body must contain one object")
		return
	}

	document, err := loadDocument(request.Path)
	if err != nil {
		writeDocumentError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(projectOpenResponse{
		Settings: repositorySettingsFromConfig(document.Settings),
		Revision: document.Revision,
	})
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
	settings := request.Settings.config()
	if err := document.Apply(settings); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_failed", "settings cannot be applied to the repository document")
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
	settings := request.Settings.config()
	if err := document.Apply(settings); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_failed", "settings cannot be applied to the repository document")
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
	settings := request.Settings.config()
	if err := document.Apply(settings); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_failed", "settings cannot be applied to the repository document")
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Code: code, Message: message})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; connect-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'self'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
