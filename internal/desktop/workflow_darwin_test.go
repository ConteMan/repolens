package desktop

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// This exercises the same native view callbacks as the app without opening a window.
func TestDesktopRulesWorkflow(t *testing.T) {
	root := filepath.Join(t.TempDir(), "favus-documentation-portal")
	initGit(t, root)
	target := filepath.Join(root, ".repolens.yml")
	const initial = "rules:\n  - match: 'docs/**'\n    render: true\n"
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
	capture("rule-editor")
	click("Add to draft")
	if len(session.Rules) != 2 {
		t.Fatal("new rule did not reach draft")
	}
	capture("rule-list")
	click("Save Settings")
	capture("confirmation")
	click("Back to Editing")
	if readConfig(t, target) != initial {
		t.Fatal("preview changed the file")
	}
	click("Save Settings")
	click("Confirm Save")
	reopened, err := Open(root)
	if err != nil || len(reopened.Rules) != 2 || *reopened.Rules[1].Match != "docs/private/**" || session.Dirty() {
		t.Fatalf("saved rule did not round-trip: %v", err)
	}
}
