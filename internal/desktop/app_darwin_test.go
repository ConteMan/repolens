package desktop

import (
	"errors"
	"github.com/egoist/mygo"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestDesktopAppSettingsWithoutRepository(t *testing.T) {
	window := &desktopWindow{english: true, sidebar: defaultSidebarWidth}
	view := ui.NewTester(window.view, 860, 632)
	view.Frame()
	if err := view.Click("App Settings"); err != nil {
		t.Fatal(err)
	}
	if window.page != pageApp {
		t.Fatal("App Settings did not open")
	}
	if err := view.Click("App Settings"); err != nil {
		t.Fatal(err)
	}
	if window.page != pageSite {
		t.Fatal("App Settings did not return to repository selection")
	}
}

func TestDesktopNoUpdateChannelStatus(t *testing.T) {
	d := &desktopWindow{sidebar: defaultSidebarWidth, page: pageApp, update: updateCheckResult(nil, mygo.ErrUpdatesDisabled)}
	view := ui.NewTester(d.view, 1024, 632)
	view.Frame()
	if d.update.err != nil || d.update.stage != updateUnavailable {
		t.Fatal("an unavailable channel must not show a failure")
	}
	if path := os.Getenv("REPOLENS_DESKTOP_EVIDENCE"); path != "" {
		f, err := os.Create(filepath.Join(path, "app-no-update-channel.png"))
		if err != nil {
			t.Fatal(err)
		}
		encodeErr := png.Encode(f, view.Image())
		if err := errors.Join(encodeErr, f.Close()); err != nil {
			t.Fatal(err)
		}
	}
}
