//go:build darwin || windows

package desktop

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ConteMan/repolens/internal/config"
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

func TestThemeSettingsOpenEditors(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	d := &desktopWindow{session: session, english: true}
	view := ui.NewTester(func(c *ui.Context) { d.sitePage(c, tabBrowse) }, 784, 632)
	click := func(name string) {
		t.Helper()
		if err := view.Click(name); err != nil {
			t.Fatal(err)
		}
	}
	click("Edit Variables…")
	view.Type("accent: red")
	click("Apply")
	if vars := session.Settings.Theme.Vars; vars == nil || (*vars)["accent"] != "red" {
		t.Fatalf("unexpected theme variables: %v", vars)
	}
	click("Choose File and Folder…")
	if _, ok := view.Access("Custom CSS"); !ok {
		t.Fatal("theme file editor did not present the CSS path")
	}
	click("Cancel")
}

func TestSiteByteSizeDisplayRoundTrips(t *testing.T) {
	for _, input := range []string{"2 MB", "1.5 MB", "0", "2049"} {
		var size *config.RepositoryByteSize
		if !updateSiteField(&size, input) || size == nil {
			t.Fatalf("could not read %q", input)
		}
		display := displayByteSize(int64(*size))
		if input == "2 MB" && display != input {
			t.Fatalf("2 MB displayed as %q", display)
		}
		var reopened *config.RepositoryByteSize
		if !updateSiteField(&reopened, display) || reopened == nil || *reopened != *size {
			t.Fatalf("display value %q does not round-trip", display)
		}
	}
}

func TestConfirmationShowsEffectiveInheritedSize(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	session.Settings.Agent.LLMSFull.MaxSize = new(config.RepositoryByteSize(3 * 1024 * 1024))
	changes, err := session.PrepareChanges()
	if err != nil {
		t.Fatal(err)
	}
	d := &desktopWindow{session: session, english: true, changes: changes, preview: true, sidebar: defaultSidebarWidth, siteTab: tabAgent}
	view := ui.NewTester(d.view, 1024, 632)
	if _, ok := view.Find("2 MB (inherited)   →   3 MB"); !ok {
		t.Fatal("confirmation did not show inherited default and the proposed value")
	}
	if path := os.Getenv("REPOLENS_DESKTOP_EVIDENCE"); path != "" {
		f, err := os.Create(filepath.Join(path, "confirmation-short.png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, view.Image()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConfirmationLongChangesKeepActionsVisible(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(t.TempDir(), "build.yml")
	writeConfig(t, external, "source:\n  ref: main\n")
	if err := session.LoadBuilder(external); err != nil {
		t.Fatal(err)
	}
	session.Settings.Site.Home = new("docs/operations/regional-workspaces/release-management/getting-started/overview-for-project-maintainers.md")
	session.Settings.Ignore = &[]string{"build/**", ".cache/**", "packages/documentation/generated/**"}
	session.Builder.Source.Ref = new("release/docs")
	changes, err := session.PrepareChanges()
	if err != nil {
		t.Fatal(err)
	}
	d := &desktopWindow{session: session, english: true, changes: changes, preview: true, sidebar: defaultSidebarWidth, siteTab: tabAgent}
	view := ui.NewTester(d.view, 1024, 632)
	for _, label := range []string{"3 changes  ·  2 configuration files", "Before", "After", "Confirm Save"} {
		bounds, ok := view.Find(label)
		if !ok || bounds.Y < 0 || bounds.Y+bounds.H > 632 {
			t.Fatalf("%q is outside the visible viewport: %+v", label, bounds)
		}
	}
	if err := view.Click("YAML Diff"); err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"Confirm Save", "Back to Editing"} {
		bounds, ok := view.Find(label)
		if !ok || bounds.Y < 0 || bounds.Y+bounds.H > 632 {
			t.Fatalf("%q is outside the YAML view: %+v", label, bounds)
		}
	}

	if path := os.Getenv("REPOLENS_DESKTOP_EVIDENCE"); path != "" {
		f, err := os.Create(filepath.Join(path, "confirmation-yaml-expanded.png"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, view.Image()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestConfirmationYAMLShowsOnlyChangedLines(t *testing.T) {
	before := "site:\n  title: Old\n  home: README.md\nview:\n  search: false\n"
	after := "site:\n  title: New\n  home: README.md\nview:\n  search: false\n"
	diff := confirmationYAMLDiff(before, after)
	if !strings.Contains(diff, "-  title: Old\n+  title: New") ||
		strings.Contains(diff, "-  home:") || strings.Contains(diff, "+  home:") {
		t.Fatalf("unexpected YAML changes:\n%s", diff)
	}
}
