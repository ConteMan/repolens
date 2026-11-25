//go:build darwin || windows

package desktop

import (
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// This exercises the same native view callbacks as the app without opening a window.
func TestDesktopRulesWorkflow(t *testing.T) {
	root := filepath.Join(t.TempDir(), "favus-documentation-portal")
	initGit(t, root)
	target := filepath.Join(root, ".repolens.yml")
	const initial = "ignore: ['node_modules/**']\nrules:\n  - match: 'docs/**'\n    render: true\n  - match: 'assets/**'\n    render: false\n"
	writeConfig(t, target, initial)
	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	d := &desktopWindow{session: session, english: true, sidebar: defaultSidebarWidth}
	view := ui.NewTester(d.view, 1024, 632)
	click := func(label string) {
		t.Helper()
		if err := view.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	capture := func(name string) {
		t.Helper()
		if dir := os.Getenv("REPOLENS_DESKTOP_EVIDENCE"); dir != "" {
			f, err := os.Create(filepath.Join(dir, name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, view.Image())
			if closeErr := f.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	view.Frame()
	click("Add Rule")
	view.Type("docs/private/**")
	click("Inherit")
	click("Off")
	capture("rule-editor")
	click("Add to draft")
	if len(session.Rules) != 3 || session.Rules[2].Render == nil || *session.Rules[2].Render {
		t.Fatal("new rule did not reach draft")
	}
	for _, node := range slices.Backward(view.AccessAll()) {
		if node.Name == "Move Up" {
			view.ClickAt(node.Bounds.X+node.Bounds.W/2, node.Bounds.Y+node.Bounds.H/2)
			break
		}
	}
	if *session.Rules[1].Match != "docs/private/**" {
		t.Fatal("new rule did not move up")
	}
	capture("rule-list")
	click("Save Settings")
	capture("confirmation")
	for _, line := range []string{"1    docs/**", "2    docs/private/**", "3    assets/**"} {
		node, visible := view.Access(line)
		if !visible || node.Bounds.H <= 0 || node.Bounds.Y+node.Bounds.H > 632 {
			t.Fatalf("final rule order is not visible: %q (%+v)", line, node.Bounds)
		}
	}
	click("Back to Editing")
	if readConfig(t, target) != initial {
		t.Fatal("preview changed the file")
	}
	click("Save Settings")
	click("Confirm Save")
	reopened, err := Open(root)
	if err != nil || len(reopened.Rules) != 3 || *reopened.Rules[1].Match != "docs/private/**" || session.Dirty() {
		t.Fatalf("saved rule did not round-trip: %v", err)
	}
}

func TestIgnoredPathsWorkflow(t *testing.T) {
	root := t.TempDir()
	initGit(t, root)
	target := filepath.Join(root, ".repolens.yml")
	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	d := &desktopWindow{session: session, english: true, sidebar: defaultSidebarWidth}
	view := ui.NewTester(d.view, 1024, 632)
	click := func(label string) {
		t.Helper()
		if err := view.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	click("Edit List")
	click("Inherit ignored paths")
	view.Type("[")
	click("Apply List")
	if d.ignoreEditor == nil || d.ignoreEditor.issue == nil || session.Settings.Ignore != nil {
		t.Fatal("invalid glob reached the draft or was not reported")
	}
	click("Cancel")
	click("Edit List")
	click("Inherit ignored paths")
	click("Apply List")
	if draft := session.Settings.Ignore; draft == nil || len(*draft) != 0 {
		t.Fatalf("expected explicit empty list, got %v", draft)
	}
	click("Save Settings")
	click("Confirm Save")
	if !strings.Contains(readConfig(t, target), "ignore: []") {
		t.Fatal("explicit empty list did not survive save")
	}
	click("Edit List")
	click("Inherit ignored paths")
	click("Apply List")
	click("Save Settings")
	click("Confirm Save")
	if strings.Contains(readConfig(t, target), "ignore:") {
		t.Fatal("inherited list still has an explicit YAML field")
	}
}

func TestDesktopSharedTargetSwitch(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	initGit(t, root)
	external := filepath.Join(t.TempDir(), "external.yml")
	writeConfig(t, external, "site:\n  title: External\n")
	session, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.LoadBuilder(external); err != nil {
		t.Fatal(err)
	}
	d := &desktopWindow{session: session, english: true, sidebar: defaultSidebarWidth, siteTab: tabBasics}
	view := ui.NewTester(d.view, 1024, 632)
	if err := view.Click("External Configuration"); err != nil {
		t.Fatal(err)
	}
	if !d.externalTarget || d.sharedDraft() != &session.ExternalSettings {
		t.Fatal("external target did not become the active editor")
	}
	if err := view.Click("Repository Configuration"); err != nil {
		t.Fatal(err)
	}
	if d.externalTarget || d.sharedDraft() != &session.Settings || session.Dirty() {
		t.Fatal("switching target changed configuration drafts")
	}
}
