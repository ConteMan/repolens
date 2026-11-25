package desktop

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSiteSettingsDraftControls(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	target := filepath.Join(root, ".repolens.yml")
	const initial = "site:\n  title: Original\n  language: en\n"
	writeConfig(t, target, initial)
	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	d := &desktopWindow{session: session, english: true}
	view := ui.NewTester(func(c *ui.Context) { d.sitePage(c, tabBasics) }, 700, 400)
	for _, label := range []string{"English", "Simplified Chinese"} {
		if err := view.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if language := session.Settings.Site.Language; language == nil || *language != "zh-CN" {
		t.Fatalf("expected selection to update the draft, got %v", language)
	}
	changes, err := session.PrepareChanges()
	if err != nil || len(changes) != 1 || readConfig(t, target) != initial {
		t.Fatalf("expected a single prepared preview without disk changes: %v", err)
	}
	if err := session.Save(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(root)
	if err != nil || reopened.Settings.Site.Language == nil || *reopened.Settings.Site.Language != "zh-CN" {
		t.Fatalf("expected the saved site language to round-trip: %v", err)
	}
}
