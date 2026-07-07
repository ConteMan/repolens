package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunAlreadyLatest(t *testing.T) {
	server := newLocalHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/ConteMan/repolens/releases/latest" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"tag_name":"v1.2.3","assets":[]}`)
	}))

	var out bytes.Buffer
	err := Run(context.Background(), Options{
		CurrentVersion: "1.2.3",
		ExecutablePath: filepath.Join(t.TempDir(), "repolens"),
		APIBaseURL:     server.url,
		HTTPClient:     server.client,
		Out:            &out,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := out.String(); !strings.Contains(got, "already the latest") {
		t.Fatalf("output = %q, want already latest message", got)
	}
}

func TestRunCheckOnlyDoesNotDownload(t *testing.T) {
	server := newLocalHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/ConteMan/repolens/releases/latest":
			fmt.Fprintf(w, `{"tag_name":"v1.2.3","assets":[{"name":%q,"browser_download_url":"https://api.github.com/download/archive"}]}`,
				currentArchiveName("v1.2.3"))
		default:
			t.Fatalf("unexpected download during --check: %s", r.URL.Path)
		}
	}))

	var out bytes.Buffer
	err := Run(context.Background(), Options{
		CurrentVersion: "v1.0.0",
		ExecutablePath: filepath.Join(t.TempDir(), "repolens"),
		APIBaseURL:     server.url,
		HTTPClient:     server.client,
		CheckOnly:      true,
		Out:            &out,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := out.String(); !strings.Contains(got, "Latest repolens version is v1.2.3") {
		t.Fatalf("output = %q, want latest version message", got)
	}
}

func TestRunDownloadAndReplace(t *testing.T) {
	binary := []byte("new repolens binary")
	archiveName := currentArchiveName("v1.2.3")
	archive := makeArchive(t, archiveName, binary)
	checksums := checksumLine(archiveName, archive)

	var out bytes.Buffer
	exePath := filepath.Join(t.TempDir(), exeName())
	if err := os.WriteFile(exePath, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("WriteFile executable: %v", err)
	}
	server := releaseServer(t, archiveName, archive, checksums)

	err := Run(context.Background(), Options{
		CurrentVersion: "v1.0.0",
		ExecutablePath: exePath,
		APIBaseURL:     server.url,
		HTTPClient:     server.client,
		Out:            &out,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatalf("ReadFile executable: %v", err)
	}
	if string(got) != string(binary) {
		t.Fatalf("executable content = %q, want %q", string(got), string(binary))
	}
	if !strings.Contains(out.String(), "Updated repolens") {
		t.Fatalf("output = %q, want update message", out.String())
	}
}

func TestRunChecksumMismatch(t *testing.T) {
	binary := []byte("new repolens binary")
	archiveName := currentArchiveName("v1.2.3")
	archive := makeArchive(t, archiveName, binary)
	checksums := fmt.Sprintf("%064x  %s\n", 0, archiveName)

	exePath := filepath.Join(t.TempDir(), exeName())
	if err := os.WriteFile(exePath, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("WriteFile executable: %v", err)
	}
	server := releaseServer(t, archiveName, archive, checksums)

	err := Run(context.Background(), Options{
		CurrentVersion: "v1.0.0",
		ExecutablePath: exePath,
		APIBaseURL:     server.url,
		HTTPClient:     server.client,
		Out:            new(bytes.Buffer),
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("Run() error = %v, want checksum mismatch", err)
	}
	got, readErr := os.ReadFile(exePath)
	if readErr != nil {
		t.Fatalf("ReadFile executable: %v", readErr)
	}
	if string(got) != "old binary" {
		t.Fatalf("executable changed after checksum failure: %q", string(got))
	}
}

func TestRunNetworkFailureMessage(t *testing.T) {
	err := Run(context.Background(), Options{
		CurrentVersion: "v1.0.0",
		ExecutablePath: filepath.Join(t.TempDir(), "repolens"),
		APIBaseURL:     "https://api.github.com",
		HTTPClient: &http.Client{
			Transport: errorRoundTripper{},
		},
		Out: new(bytes.Buffer),
	})
	if err == nil {
		t.Fatal("Run() error = nil, want network error")
	}
	if got := err.Error(); !strings.Contains(got, "could not check latest release") || !strings.Contains(got, "network unavailable") {
		t.Fatalf("error = %q, want human-readable network failure", got)
	}
}

func TestDetectInstallSource(t *testing.T) {
	tests := []struct {
		path string
		want InstallSource
	}{
		{"/opt/homebrew/Cellar/repolens/1.2.3/bin/repolens", InstallSourceHomebrew},
		{"/opt/homebrew/bin/repolens", InstallSourceHomebrew},
		{`C:\Users\me\scoop\apps\repolens\current\repolens.exe`, InstallSourceScoop},
		{"/usr/local/bin/repolens", InstallSourceDirect},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := DetectInstallSource(tt.path); got != tt.want {
				t.Fatalf("DetectInstallSource(%q) = %s, want %s", tt.path, got, tt.want)
			}
		})
	}
}

func TestRunManagedInstallSources(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "homebrew", path: "/opt/homebrew/bin/repolens", want: "brew upgrade repolens"},
		{name: "scoop", path: `C:\Users\me\scoop\apps\repolens\current\repolens.exe`, want: "scoop update repolens"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := Run(context.Background(), Options{
				CurrentVersion: "v1.0.0",
				ExecutablePath: tt.path,
				APIBaseURL:     "http://127.0.0.1:1",
				Out:            &out,
			})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("output = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func releaseServer(t *testing.T, archiveName string, archive []byte, checksums string) localHTTPServer {
	t.Helper()
	const baseURL = "https://api.github.com"
	return newLocalHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/ConteMan/repolens/releases/latest":
			fmt.Fprintf(w, `{"tag_name":"v1.2.3","assets":[{"name":%q,"browser_download_url":%q},{"name":"checksums.txt","browser_download_url":%q}]}`,
				archiveName, baseURL+"/download/"+archiveName, baseURL+"/download/checksums.txt")
		case "/download/" + archiveName:
			_, _ = w.Write(archive)
		case "/download/checksums.txt":
			fmt.Fprint(w, checksums)
		default:
			http.NotFound(w, r)
		}
	}))
}

type localHTTPServer struct {
	url    string
	client *http.Client
}

func TestValidateDownloadURL(t *testing.T) {
	for _, tt := range []struct {
		url string
		ok  bool
	}{
		{url: "https://github.com/ConteMan/repolens/releases/download/v1/x.tar.gz", ok: true},
		{url: "https://objects.githubusercontent.com/abc", ok: true},
		{url: "https://api.github.com/repos/x", ok: true},
		{url: "https://evil.example.com/repolens.tar.gz", ok: false},
		{url: "http://github.com/insecure", ok: false},
		{url: "https://github.com.evil.com/x", ok: false},
		{url: "http://127.0.0.1:9999/test-fixture", ok: true},
	} {
		err := validateDownloadURL(tt.url)
		if tt.ok && err != nil {
			t.Errorf("validateDownloadURL(%q) = %v, want nil", tt.url, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("validateDownloadURL(%q) = nil, want error", tt.url)
		}
	}
}

func newLocalHTTPServer(handler http.Handler) localHTTPServer {
	return localHTTPServer{
		url: "https://api.github.com",
		client: &http.Client{
			Transport: localRoundTripper{handler: handler},
		},
	}
}

type localRoundTripper struct {
	handler http.Handler
}

func (rt localRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	rt.handler.ServeHTTP(recorder, req)
	return recorder.Result(), nil
}

type errorRoundTripper struct{}

func (errorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network unavailable")
}

func currentArchiveName(version string) string {
	return fmt.Sprintf("repolens_%s_%s_%s%s", strings.TrimPrefix(version, "v"), runtime.GOOS, runtime.GOARCH, archiveSuffix(runtime.GOOS))
}

func makeArchive(t *testing.T, archiveName string, binary []byte) []byte {
	t.Helper()
	if strings.HasSuffix(archiveName, ".zip") {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		w, err := zw.Create(exeName())
		if err != nil {
			t.Fatalf("zip Create: %v", err)
		}
		if _, err := w.Write(binary); err != nil {
			t.Fatalf("zip Write: %v", err)
		}
		if err := zw.Close(); err != nil {
			t.Fatalf("zip Close: %v", err)
		}
		return buf.Bytes()
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: exeName(), Mode: 0o755, Size: int64(len(binary))}); err != nil {
		t.Fatalf("tar WriteHeader: %v", err)
	}
	if _, err := tw.Write(binary); err != nil {
		t.Fatalf("tar Write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar Close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip Close: %v", err)
	}
	return buf.Bytes()
}

func checksumLine(filename string, data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x  %s\n", sum[:], filename)
}

func exeName() string {
	if runtime.GOOS == "windows" {
		return "repolens.exe"
	}
	return "repolens"
}
