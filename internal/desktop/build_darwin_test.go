package desktop

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	build "github.com/ConteMan/repolens/internal/ui"
	"github.com/egoist/mygo/ui"
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

func TestBuildPathsShareValueColumn(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(filepath.Dir(session.Root), "public")
	locations, err := build.ResolveNativeBuildLocations(session.Root, "", session.Builder, &output)
	if err != nil {
		t.Fatal(err)
	}
	for _, width := range []int{740, 1024, 2280} {
		d := &desktopWindow{session: session, page: pageBuild, sidebar: 340, english: true, buildState: desktopBuildState{output: &output}}
		view := ui.NewTester(d.view, width, 720)
		source, sourceOK := view.Find(session.Root)
		target, targetOK := view.Find(locations.Output)
		button, buttonOK := view.Find("Change folder")
		if !sourceOK || !targetOK || !buttonOK || source.X != target.X || target.W <= 0 || button.X+button.W > float32(width) {
			t.Fatalf("width %d: source %+v, output %+v, folder button %+v", width, source, target, button)
		}
	}
}
