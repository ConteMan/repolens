package skill

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLookupNameAndAlias(t *testing.T) {
	byName, ok, err := Lookup("repolens-glossary")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Lookup(repolens-glossary) not found")
	}
	byAlias, ok, err := Lookup("glossary")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("Lookup(glossary) not found")
	}
	if byName.Name != "repolens-glossary" || byName.Alias != "glossary" {
		t.Fatalf("skill = %+v", byName)
	}
	if !bytes.Equal(byName.Content, byAlias.Content) {
		t.Fatal("name and alias returned different content")
	}
	if _, ok, err := Lookup("missing"); err != nil || ok {
		t.Fatalf("Lookup(missing) ok=%v err=%v", ok, err)
	}
}

func TestSplitAndInjectProvenance(t *testing.T) {
	s, ok, err := Lookup("repolens-glossary")
	if err != nil || !ok {
		t.Fatalf("lookup: ok=%v err=%v", ok, err)
	}
	_, origBody, _, err := splitFrontMatter(s.Content)
	if err != nil {
		t.Fatal(err)
	}
	got, err := materialize(s, "1.7.0")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseSkillFile(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Body, origBody) {
		t.Fatal("inject changed body bytes")
	}
	if !bytes.Contains(got, []byte("name: repolens-glossary\n")) {
		t.Fatal("original name line was rewritten")
	}
	if parsed.Metadata["repolens.skill"] != "repolens-glossary" {
		t.Fatalf("skill = %q", parsed.Metadata["repolens.skill"])
	}
	if parsed.Metadata["repolens.version"] != "1.7.0" {
		t.Fatalf("version = %q", parsed.Metadata["repolens.version"])
	}
	if parsed.Metadata["repolens.digest"] != shortDigest(origBody) {
		t.Fatalf("digest = %q, want %q", parsed.Metadata["repolens.digest"], shortDigest(origBody))
	}
}

func TestShortDigestEmpty(t *testing.T) {
	if got := shortDigest(nil); got != "e3b0c44298fc" {
		t.Fatalf("digest(nil) = %q", got)
	}
}

func TestValidateSkillName(t *testing.T) {
	valid := []string{"a", "repolens-glossary", "a1", "ab-c"}
	for _, name := range valid {
		if err := validateSkillName(name); err != nil {
			t.Errorf("validateSkillName(%q) = %v", name, err)
		}
	}
	invalid := []string{"", "-abc", "abc-", "a--b", "ABC", "has_underscore", strings.Repeat("a", 65)}
	for _, name := range invalid {
		if err := validateSkillName(name); err == nil {
			t.Errorf("validateSkillName(%q) = nil, want error", name)
		}
	}
}

func TestDetectSignals(t *testing.T) {
	root := t.TempDir()
	detected, err := Detect(root, ScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	if len(detected) != 0 {
		t.Fatalf("clean repo Detect = %+v", detected)
	}

	if err := os.Mkdir(filepath.Join(root, ".github"), 0o755); err != nil {
		t.Fatal(err)
	}
	detected, err = Detect(root, ScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	if len(detected) != 0 {
		t.Fatalf("bare .github Detect = %+v, want empty", detected)
	}

	if err := os.Mkdir(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, ".github/copilot-instructions.md", "# copilot\n")
	detected, err = Detect(root, ScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, target := range detected {
		got[target.Key] = true
		if !filepath.IsAbs(target.Dir) {
			t.Fatalf("Dir is not absolute: %s", target.Dir)
		}
	}
	for _, key := range []string{"claude", "codex", "copilot"} {
		if !got[key] {
			t.Errorf("missing detected key %s", key)
		}
	}
	if got["agents"] || got["cursor"] {
		t.Fatalf("unexpected keys: %+v", got)
	}
}

func TestResolveTargetsUnknown(t *testing.T) {
	_, err := ResolveTargets(t.TempDir(), ScopeProject, []string{"nope"})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, key := range KnownTargetKeys() {
		if !strings.Contains(msg, key) {
			t.Errorf("error %q missing %s", msg, key)
		}
	}
}

func TestTargetsGlobalHomeFailure(t *testing.T) {
	prev := userHomeDir
	userHomeDir = func() (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { userHomeDir = prev })
	_, err := Targets(t.TempDir(), ScopeGlobal)
	if err == nil {
		t.Fatal("expected home error")
	}
}

func TestInstallFallbackAndIdempotent(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	s := mustLookup(t, "glossary")
	target := mustTarget(t, root, ScopeProject, "agents")

	first, err := Install(s, target, "1.7.0", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Action != ActionCreated || first.State != StateAbsent {
		t.Fatalf("first = %+v", first)
	}
	wantPath := filepath.Join(root, ".agents", "skills", "repolens-glossary", "SKILL.md")
	if first.Path != wantPath {
		t.Fatalf("path = %s, want %s", first.Path, wantPath)
	}
	original, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}

	second, err := Install(s, target, "1.7.0", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Action != ActionSkipped || second.State != StateCurrent {
		t.Fatalf("second = %+v", second)
	}
	again, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, again) {
		t.Fatal("second install changed bytes")
	}
}

func TestInstallDryRunDoesNotWrite(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	s := mustLookup(t, "glossary")
	target := mustTarget(t, root, ScopeProject, "agents")
	res, err := Install(s, target, "1.7.0", Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != ActionCreated {
		t.Fatalf("action = %s", res.Action)
	}
	if exists(filepath.Join(root, ".agents")) {
		t.Fatal("dry-run created .agents")
	}
}

func TestInstallSkipsModifiedAndForeign(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	s := mustLookup(t, "glossary")
	target := mustTarget(t, root, ScopeProject, "agents")
	if _, err := Install(s, target, "1.7.0", Options{}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(target.Dir, s.Name, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte("\nlocal edit\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	skipped, err := Install(s, target, "1.7.0", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if skipped.Action != ActionSkipped || skipped.State != StateModified || skipped.Warning != WarnModified {
		t.Fatalf("modified = %+v", skipped)
	}

	forced, err := Install(s, target, "1.7.0", Options{Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if forced.Action != ActionUpdated {
		t.Fatalf("force = %+v", forced)
	}

	foreignRoot := t.TempDir()
	foreignTarget := mustTarget(t, foreignRoot, ScopeProject, "agents")
	writeTestFile(t, foreignRoot, ".agents/skills/repolens-glossary/SKILL.md", "---\nname: repolens-glossary\ndescription: mine\n---\n\n# custom\n")
	foreign, err := Install(s, foreignTarget, "1.7.0", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if foreign.Action != ActionSkipped || foreign.State != StateForeign || foreign.Warning != WarnForeign {
		t.Fatalf("foreign = %+v", foreign)
	}
	before, err := os.ReadFile(filepath.Join(foreignRoot, ".agents/skills/repolens-glossary/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(before, []byte("# custom")) {
		t.Fatal("foreign copy was rewritten without --force")
	}
	if _, err := Install(s, foreignTarget, "1.7.0", Options{Force: true}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(foreignRoot, ".agents/skills/repolens-glossary/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(after, []byte("# custom")) {
		t.Fatal("force did not overwrite foreign copy")
	}
}

func TestInstallPreservesSiblingFiles(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	s := mustLookup(t, "glossary")
	target := mustTarget(t, root, ScopeProject, "agents")
	note := "do not touch\n"
	writeTestFile(t, root, ".agents/skills/repolens-glossary/references/note.md", note)
	if _, err := Install(s, target, "1.6.3", Options{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, ".agents/skills/repolens-glossary/references/note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != note {
		t.Fatalf("note = %q", got)
	}
	if _, err := Update(root, "1.7.0", Options{}); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(filepath.Join(root, ".agents/skills/repolens-glossary/references/note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != note {
		t.Fatalf("note after update = %q", got)
	}
}

func TestUpdateRewritesOutdatedAndSkipsModified(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	s := mustLookup(t, "glossary")
	target := mustTarget(t, root, ScopeProject, "agents")
	if _, err := Install(s, target, "1.6.3", Options{}); err != nil {
		t.Fatal(err)
	}

	results, err := Update(root, "1.7.0", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Action != ActionUpdated || results[0].State != StateOutdated {
		t.Fatalf("update = %+v", results)
	}
	data, err := os.ReadFile(results[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseSkillFile(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Metadata["repolens.version"] != "1.7.0" {
		t.Fatalf("version = %q", parsed.Metadata["repolens.version"])
	}
	if parsed.Metadata["repolens.digest"] != shortDigest(parsed.Body) {
		t.Fatalf("digest mismatch: %q vs %q", parsed.Metadata["repolens.digest"], shortDigest(parsed.Body))
	}

	if err := os.WriteFile(results[0].Path, append(data, []byte("edited\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	skipped, err := Update(root, "1.8.0", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 1 || skipped[0].Action != ActionSkipped || skipped[0].State != StateModified {
		t.Fatalf("modified update = %+v", skipped)
	}
	forced, err := Update(root, "1.8.0", Options{Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(forced) != 1 || forced[0].Action != ActionUpdated {
		t.Fatalf("force update = %+v", forced)
	}
}

func TestUpdateDryRunAndDev(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	s := mustLookup(t, "glossary")
	target := mustTarget(t, root, ScopeProject, "agents")
	if _, err := Install(s, target, "dev", Options{}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(target.Dir, s.Name, "SKILL.md")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dry, err := Update(root, "dev", Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(dry) != 1 || dry[0].Action != ActionUpdated {
		t.Fatalf("dev dry-run = %+v", dry)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("dry-run rewrote file")
	}
}

func TestUpdateLeavesUnbundled(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	body := []byte("# leftover\n")
	content := "---\nname: repolens-old\ndescription: gone\nmetadata:\n  repolens.skill: repolens-old\n  repolens.version: \"1.0.0\"\n  repolens.digest: \"" + shortDigest(body) + "\"\n---\n" + string(body)
	writeTestFile(t, root, ".agents/skills/repolens-old/SKILL.md", content)
	results, err := Update(root, "1.7.0", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Action != ActionSkipped || results[0].Warning != WarnUnbundled {
		t.Fatalf("unbundled = %+v", results)
	}
	got, err := os.ReadFile(filepath.Join(root, ".agents/skills/repolens-old/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Fatal("unbundled copy was changed")
	}
}

func TestScanStates(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	s := mustLookup(t, "glossary")
	target := mustTarget(t, root, ScopeProject, "agents")
	if _, err := Install(s, target, "1.7.0", Options{}); err != nil {
		t.Fatal(err)
	}
	found, err := Scan(root, "1.7.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].State != StateCurrent || found[0].Version != "1.7.0" {
		t.Fatalf("current = %+v", found)
	}

	found, err = Scan(root, "1.8.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].State != StateOutdated {
		t.Fatalf("outdated = %+v", found)
	}

	path := found[0].Path
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, 'x'), 0o644); err != nil {
		t.Fatal(err)
	}
	found, err = Scan(root, "1.7.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].State != StateModified {
		t.Fatalf("modified = %+v", found)
	}
}

func TestInstallOutdatedOverwrites(t *testing.T) {
	isolateHome(t)
	root := t.TempDir()
	s := mustLookup(t, "glossary")
	target := mustTarget(t, root, ScopeProject, "agents")
	if _, err := Install(s, target, "1.6.3", Options{}); err != nil {
		t.Fatal(err)
	}
	res, err := Install(s, target, "1.7.0", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != ActionUpdated || res.State != StateOutdated {
		t.Fatalf("outdated install = %+v", res)
	}
}

func mustLookup(t *testing.T, name string) Skill {
	t.Helper()
	s, ok, err := Lookup(name)
	if err != nil || !ok {
		t.Fatalf("Lookup(%q): ok=%v err=%v", name, ok, err)
	}
	return s
}

func mustTarget(t *testing.T, root string, scope Scope, key string) Target {
	t.Helper()
	ts, err := ResolveTargets(root, scope, []string{key})
	if err != nil {
		t.Fatal(err)
	}
	return ts[0]
}

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectGlobalUsesHomeSignals(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	prev := userHomeDir
	userHomeDir = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHomeDir = prev })

	// The repository configures Cursor; the user profile has Claude Code and
	// Copilot. Global scope must follow the profile, not the repository.
	if err := os.Mkdir(filepath.Join(root, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "github-copilot"), 0o755); err != nil {
		t.Fatal(err)
	}

	detected, err := Detect(root, ScopeGlobal)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, target := range detected {
		got[target.Key] = target.Dir
	}
	if len(got) != 2 || got["claude"] == "" || got["copilot"] == "" {
		t.Fatalf("global Detect = %+v, want claude and copilot only", got)
	}
	if want := filepath.Join(home, ".claude", "skills"); got["claude"] != want {
		t.Errorf("claude dir = %s, want %s", got["claude"], want)
	}
	if want := filepath.Join(home, ".config", "github-copilot", "skills"); got["copilot"] != want {
		t.Errorf("copilot dir = %s, want %s", got["copilot"], want)
	}

	project, err := Detect(root, ScopeProject)
	if err != nil {
		t.Fatal(err)
	}
	if len(project) != 1 || project[0].Key != "cursor" {
		t.Fatalf("project Detect = %+v, want cursor only", project)
	}
}
