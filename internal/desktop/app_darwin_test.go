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

func TestDesktopUpdateDesignStates(t *testing.T) {
	root := filepath.Join(t.TempDir(), "favus-documentation-portal")
	initGit(t, root)
	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	version := mygo.App.Version()
	mygo.App.SetVersion("1.0.0")
	t.Cleanup(func() { mygo.App.SetVersion(version) })
	for _, state := range []struct {
		name    string
		width   int
		english bool
		update  desktopUpdate
	}{
		{"APP-01-update-ready", 1024, false, desktopUpdate{stage: updateReady, release: &mygo.Update{Version: "1.0.1"}}},
		{"APP-02-up-to-date", 860, true, desktopUpdate{stage: updateCurrent}},
		{"APP-update-checking", 860, true, desktopUpdate{stage: updateChecking}},
		{"APP-update-downloading", 860, true, desktopUpdate{stage: updateDownloading, received: 20, total: 50}},
	} {
		t.Run(state.name, func(t *testing.T) {
			d := &desktopWindow{sidebar: defaultSidebarWidth, session: session, page: pageApp, english: state.english, update: state.update}
			if state.width == 860 {
				d.sidebar = 192
			}
			d.preferences.AutoCheck = true
			if state.english {
				d.preferences.Language = "en"
			}
			view := ui.NewTester(d.view, state.width, 632)
			view.Frame()
			if path := os.Getenv("REPOLENS_DESKTOP_EVIDENCE"); path != "" {
				f, err := os.Create(filepath.Join(path, state.name+".png"))
				if err != nil {
					t.Fatal(err)
				}
				encodeErr := png.Encode(f, view.Image())
				if err := errors.Join(encodeErr, f.Close()); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
