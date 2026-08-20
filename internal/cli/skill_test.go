package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ConteMan/repolens/internal/skill"
)

func TestSkillListStates(t *testing.T) {
	isolateCLIHome(t)
	withVersion(t, "1.7.0")

	t.Run("none", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		isolateCLIHome(t)
		out, err := runSkill(t, "skill", "list")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "内置:") || !strings.Contains(out, "repolens-glossary (glossary)") {
			t.Fatalf("missing builtin section: %s", out)
		}
		if !strings.Contains(out, "已安装:") {
			t.Fatalf("missing installed section: %s", out)
		}
		if !strings.Contains(out, "repolens skill install") {
			t.Fatalf("missing install hint: %s", out)
		}
		if strings.Contains(out, "最新") || strings.Contains(out, "过期") || strings.Contains(out, "本地已修改") {
			t.Fatalf("unexpected status: %s", out)
		}
	})

	t.Run("current outdated modified", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		isolateCLIHome(t)
		s, ok, err := skill.Lookup("glossary")
		if err != nil || !ok {
			t.Fatalf("lookup: %v %v", ok, err)
		}

		currentRoot := filepath.Join(dir, "current")
		if err := os.Mkdir(currentRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		outdatedRoot := filepath.Join(dir, "outdated")
		if err := os.Mkdir(outdatedRoot, 0o755); err != nil {
			t.Fatal(err)
		}
		modifiedRoot := filepath.Join(dir, "modified")
		if err := os.Mkdir(modifiedRoot, 0o755); err != nil {
			t.Fatal(err)
		}

		currentTarget, err := skill.ResolveTargets(currentRoot, skill.ScopeProject, []string{"agents"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := skill.Install(s, currentTarget[0], "1.7.0", skill.Options{}); err != nil {
			t.Fatal(err)
		}
		t.Chdir(currentRoot)
		out, err := runSkill(t, "skill", "list")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "最新") {
			t.Fatalf("current list: %s", out)
		}

		outdatedTarget, err := skill.ResolveTargets(outdatedRoot, skill.ScopeProject, []string{"agents"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := skill.Install(s, outdatedTarget[0], "1.6.3", skill.Options{}); err != nil {
			t.Fatal(err)
		}
		t.Chdir(outdatedRoot)
		out, err = runSkill(t, "skill", "list")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "过期（1.6.3 → 1.7.0）") {
			t.Fatalf("outdated list: %s", out)
		}

		modifiedTarget, err := skill.ResolveTargets(modifiedRoot, skill.ScopeProject, []string{"agents"})
		if err != nil {
			t.Fatal(err)
		}
		res, err := skill.Install(s, modifiedTarget[0], "1.7.0", skill.Options{})
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(res.Path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(res.Path, append(data, []byte("\nedit\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Chdir(modifiedRoot)
		out, err = runSkill(t, "skill", "list")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, "本地已修改") {
			t.Fatalf("modified list: %s", out)
		}
	})
}

func TestSkillInstallFallback(t *testing.T) {
	isolateCLIHome(t)
	withVersion(t, "1.7.0")
	dir := t.TempDir()
	out, err := runSkill(t, "skill", "install", "glossary", "--dir", dir)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, ".agents", "skills", "repolens-glossary", "SKILL.md")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("missing install product: %v\n%s", err, out)
	}
	if !strings.Contains(out, "未探测到") || !strings.Contains(out, "--target") {
		t.Fatalf("missing fallback explanation: %s", out)
	}
	if !strings.Contains(out, want) || !strings.Contains(out, "新建") {
		t.Fatalf("missing created path: %s", out)
	}
	if !strings.Contains(out, "请重启 Agent 会话") {
		t.Fatalf("missing restart hint: %s", out)
	}
}

func TestSkillInstallDetectedTargets(t *testing.T) {
	isolateCLIHome(t)
	withVersion(t, "1.7.0")
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := runSkill(t, "skill", "install", "glossary", "--dir", dir)
	if err != nil {
		t.Fatal(err)
	}
	claude := filepath.Join(dir, ".claude", "skills", "repolens-glossary", "SKILL.md")
	codex := filepath.Join(dir, ".codex", "skills", "repolens-glossary", "SKILL.md")
	if _, err := os.Stat(claude); err != nil {
		t.Fatalf("claude copy: %v", err)
	}
	if _, err := os.Stat(codex); err != nil {
		t.Fatalf("codex copy: %v", err)
	}
	if !strings.Contains(out, claude) || !strings.Contains(out, codex) {
		t.Fatalf("output missing absolute paths: %s", out)
	}
}

func TestSkillInstallUnknownTarget(t *testing.T) {
	isolateCLIHome(t)
	dir := t.TempDir()
	_, err := runSkill(t, "skill", "install", "glossary", "--dir", dir, "--target", "nope")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, key := range skill.KnownTargetKeys() {
		if !strings.Contains(msg, key) {
			t.Errorf("error %q missing %s", msg, key)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".agents")); !os.IsNotExist(err) {
		t.Fatalf(".agents created on unknown target: %v", err)
	}
}

func TestSkillInstallDryRun(t *testing.T) {
	isolateCLIHome(t)
	withVersion(t, "1.7.0")
	dir := t.TempDir()
	out, err := runSkill(t, "skill", "install", "glossary", "--dir", dir, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if existsPath(filepath.Join(dir, ".agents")) {
		t.Fatal("dry-run created .agents")
	}
	if !strings.Contains(out, "新建") {
		t.Fatalf("dry-run output: %s", out)
	}

	if err := os.Mkdir(filepath.Join(dir, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = runSkill(t, "skill", "install", "glossary", "--dir", dir, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if existsPath(filepath.Join(dir, ".claude", "skills")) || existsPath(filepath.Join(dir, ".codex", "skills")) {
		t.Fatal("dry-run created detected target dirs")
	}
}

func TestSkillInstallIdempotentCLI(t *testing.T) {
	isolateCLIHome(t)
	withVersion(t, "1.7.0")
	dir := t.TempDir()
	if _, err := runSkill(t, "skill", "install", "glossary", "--dir", dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".agents", "skills", "repolens-glossary", "SKILL.md")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out, err := runSkill(t, "skill", "install", "glossary", "--dir", dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "已是最新") {
		t.Fatalf("second install: %s", out)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("second install changed bytes")
	}
}

func TestSkillInstallAndUpdateModifiedForeign(t *testing.T) {
	isolateCLIHome(t)
	withVersion(t, "1.7.0")
	dir := t.TempDir()
	if _, err := runSkill(t, "skill", "install", "glossary", "--dir", dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".agents", "skills", "repolens-glossary", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte("edit\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runSkill(t, "skill", "install", "glossary", "--dir", dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "正文已被本地修改") || !strings.Contains(out, "--force") {
		t.Fatalf("install modified: %s", out)
	}
	out, err = runSkill(t, "skill", "update", "--dir", dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "正文已被本地修改") {
		t.Fatalf("update modified: %s", out)
	}
	if _, err := runSkill(t, "skill", "install", "glossary", "--dir", dir, "--force"); err != nil {
		t.Fatal(err)
	}

	foreign := t.TempDir()
	writeCLIFile(t, foreign, ".agents/skills/repolens-glossary/SKILL.md", "---\nname: repolens-glossary\ndescription: mine\n---\n\n# x\n")
	out, err = runSkill(t, "skill", "install", "glossary", "--dir", foreign)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "非 repolens 安装") || !strings.Contains(out, "--force") {
		t.Fatalf("foreign: %s", out)
	}
	if _, err := runSkill(t, "skill", "install", "glossary", "--dir", foreign, "--force"); err != nil {
		t.Fatal(err)
	}
}

func TestSkillUpdateVersionBump(t *testing.T) {
	isolateCLIHome(t)
	withVersion(t, "1.7.0")
	dir := t.TempDir()
	s, ok, err := skill.Lookup("glossary")
	if err != nil || !ok {
		t.Fatal(err)
	}
	targets, err := skill.ResolveTargets(dir, skill.ScopeProject, []string{"agents"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := skill.Install(s, targets[0], "1.6.3", skill.Options{}); err != nil {
		t.Fatal(err)
	}
	out, err := runSkill(t, "skill", "update", "--dir", dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "更新") {
		t.Fatalf("update output: %s", out)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".agents/skills/repolens-glossary/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "repolens.version: \"1.7.0\"") {
		t.Fatalf("version not updated:\n%s", data)
	}
}

func TestSkillInstallGlobal(t *testing.T) {
	home := isolateCLIHome(t)
	withVersion(t, "1.7.0")
	dir := t.TempDir()
	out, err := runSkill(t, "skill", "install", "glossary", "--dir", dir, "--global")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".agents", "skills", "repolens-glossary", "SKILL.md")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("global install: %v\n%s", err, out)
	}
	if existsPath(filepath.Join(dir, ".agents")) {
		t.Fatal("global install wrote project dir")
	}
}

func runSkill(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func withVersion(t *testing.T, v string) {
	t.Helper()
	prev := Version
	Version = v
	t.Cleanup(func() { Version = prev })
}

func isolateCLIHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func writeCLIFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func existsPath(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
