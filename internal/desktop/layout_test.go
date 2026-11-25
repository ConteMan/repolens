//go:build darwin || windows

package desktop

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestDesktopLayoutFitsWindow(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []struct {
		name          string
		width, height int
	}{{"narrow", 740, 450}, {"wide", 1640, 860}} {
		t.Run(size.name, func(t *testing.T) {
			d := &desktopWindow{session: session, sidebar: 340, english: true, siteTab: tabBasics}
			view := ui.NewTester(d.view, size.width, size.height)
			for _, label := range []string{"Site Title", "Save Settings"} {
				bounds, ok := view.Find(label)
				if !ok || bounds.X < 0 || bounds.Y < 0 || bounds.X+bounds.W > float32(size.width) ||
					bounds.Y+bounds.H > float32(size.height) {
					t.Fatalf("%s overflows viewport: %+v", label, bounds)
				}
			}
			if path := os.Getenv("REPOLENS_DESKTOP_EVIDENCE"); path != "" {
				f, err := os.Create(filepath.Join(path, "layout-"+size.name+".png"))
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				if err := png.Encode(f, view.Image()); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
