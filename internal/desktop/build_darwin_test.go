package desktop

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	build "github.com/ConteMan/repolens/internal/ui"
)

func TestNativePreviewServesLastBuild(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "view"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "view", "index.html"), []byte("preview fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	b := desktopBuildState{result: build.NativeBuildResult{Directory: dir}}
	if err := b.startPreview(); err != nil {
		t.Fatal(err)
	}
	defer b.reset()
	res, err := http.Get(b.previewURL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil || res.StatusCode != 200 || string(body) != "preview fixture" {
		t.Fatalf("preview status %d, body %q, read error %v", res.StatusCode, body, err)
	}
}
