package desktop

import (
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
