//go:build darwin || windows

package desktop

import (
	"testing"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/egoist/mygo/ui"
)

func TestRuleEditorControls(t *testing.T) {
	original := config.RepositoryRuleSettings{Match: new("docs/**"), Render: new(true)}
	editor := newRuleEditor(original, false, nil)
	var saved *config.RepositoryRuleSettings
	var canceled bool
	tester := ui.NewTester(func(c *ui.Context) {
		editor.view(c, true, func(rule config.RepositoryRuleSettings) { saved = &rule }, func() { canceled = true })
	}, 700, 680)
	click := func(name string) {
		t.Helper()
		if err := tester.Click(name); err != nil {
			t.Fatal(err)
		}
	}

	click("On")
	click("Off")
	click("Update draft")
	if saved == nil || saved.Render == nil || *saved.Render {
		t.Fatalf("expected an explicit false, got %#v", saved)
	}
	if !*original.Render {
		t.Fatal("editing changed the original rule")
	}
	click("Off")
	click("Inherit")
	click("Update draft")
	if saved == nil || saved.Render != nil {
		t.Fatalf("expected inherited render, got %#v", saved)
	}
	click("Cancel")
	if !canceled {
		t.Fatal("cancel did not close the form")
	}
}

func TestRuleEditorValidationFocus(t *testing.T) {
	editor := newRuleEditor(config.RepositoryRuleSettings{}, true, nil)
	var saved *config.RepositoryRuleSettings
	tester := ui.NewTester(func(c *ui.Context) {
		editor.view(c, true, func(rule config.RepositoryRuleSettings) { saved = &rule }, func() {})
	}, 700, 680)

	if err := tester.Click("Add to draft"); err != nil {
		t.Fatal(err)
	}
	if saved != nil || editor.issue == nil || editorFieldPath(editor.issue.Path) != "match" {
		t.Fatalf("missing match was not reported: %#v", editor.issue)
	}
	tester.Type("docs/private/**")
	if err := tester.Click("Add to draft"); err != nil {
		t.Fatal(err)
	}
	if saved == nil || saved.Match == nil || *saved.Match != "docs/private/**" {
		t.Fatalf("typing after validation did not update the focused match: %#v", saved)
	}
}

func TestRuleEditorRestoresExplicitValues(t *testing.T) {
	editor := newRuleEditor(config.RepositoryRuleSettings{
		Match: new("docs/**"), Markdown: config.RepositoryMarkdownOptionsSettings{TOCMinHeadings: new(0)},
		Code: config.RepositoryCodeOptionsSettings{Theme: new("")}, MaxFileSize: new(config.RepositoryByteSize(0)),
	}, false, nil)
	rule, issue := editor.candidate()
	if issue != nil || rule.Markdown.TOCMinHeadings == nil || *rule.Markdown.TOCMinHeadings != 0 ||
		rule.MaxFileSize == nil || *rule.MaxFileSize != 0 || rule.Code.Theme == nil || *rule.Code.Theme != "" {
		t.Fatalf("explicit values were not restored: %#v, %#v", rule, issue)
	}
	editor.tocMin = ""
	editor.size = "5 MB"
	rule, issue = editor.candidate()
	if issue != nil || rule.Markdown.TOCMinHeadings != nil || rule.MaxFileSize == nil || *rule.MaxFileSize != 5*1024*1024 {
		t.Fatalf("expected inherited heading count and parsed size: %#v, %#v", rule, issue)
	}
}
