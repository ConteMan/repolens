package skill

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/ConteMan/repolens/skills"
)

func TestBehavior8Consistency(t *testing.T) {
	t.Run("name matches directory and naming rules", func(t *testing.T) {
		all, err := Builtin()
		if err != nil {
			t.Fatal(err)
		}
		if len(all) == 0 {
			t.Fatal("no bundled skills")
		}
		for _, s := range all {
			if err := validateSkillName(s.Name); err != nil {
				t.Errorf("%s: %v", s.Name, err)
			}
			parsed, err := parseSkillFile(s.Content)
			if err != nil {
				t.Errorf("%s: %v", s.Name, err)
				continue
			}
			if parsed.Name != s.Name {
				t.Errorf("%s: front matter name %q", s.Name, parsed.Name)
			}
		}
	})

	t.Run("embedded front matter has no metadata", func(t *testing.T) {
		all, err := Builtin()
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range all {
			parsed, err := parseSkillFile(s.Content)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.hasMetadataField {
				t.Errorf("%s: embedded SKILL.md contains metadata", s.Name)
			}
		}
	})

	t.Run("embed set matches skills directory", func(t *testing.T) {
		disk := diskSkillNames(t)
		all, err := Builtin()
		if err != nil {
			t.Fatal(err)
		}
		embedded := map[string]bool{}
		for _, s := range all {
			embedded[s.Name] = true
		}
		for name := range disk {
			if !embedded[name] {
				t.Errorf("skills/%s/SKILL.md exists on disk but is not embedded", name)
			}
		}
		for name := range embedded {
			if !disk[name] {
				t.Errorf("embedded skill %s has no skills/%s/SKILL.md on disk", name, name)
			}
		}

		diskFile := filepath.Join(repoSkillsDir(t), "repolens-glossary", "SKILL.md")
		want, err := os.ReadFile(diskFile)
		if err != nil {
			t.Fatal(err)
		}
		s := mustLookup(t, "repolens-glossary")
		if !bytes.Equal(s.Content, want) {
			t.Fatal("embedded SKILL.md differs from skills/repolens-glossary/SKILL.md")
		}
	})

	t.Run("example is not embedded", func(t *testing.T) {
		err := fs.WalkDir(skills.FS, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if p != "." && (path.Base(p) == "example" || strings.Contains(p, "/example/")) {
				t.Errorf("embedded unexpected path %s", p)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("installed product round-trips provenance", func(t *testing.T) {
		isolateHome(t)
		root := t.TempDir()
		s := mustLookup(t, "glossary")
		target := mustTarget(t, root, ScopeProject, "agents")
		res, err := Install(s, target, "1.7.0", Options{})
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(res.Path)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parseSkillFile(data)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateSkillName(parsed.Name); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(parsed.Description) == "" {
			t.Fatal("description is empty")
		}
		if n := utf8.RuneCountInString(parsed.Description); n > 1024 {
			t.Fatalf("description length %d", n)
		}
		if !parsed.hasMetadataField {
			t.Fatal("installed file missing metadata")
		}
		for k, v := range parsed.Metadata {
			if strings.TrimSpace(v) == "" {
				t.Errorf("metadata %q is empty", k)
			}
		}
		if parsed.Metadata["repolens.skill"] != s.Name {
			t.Fatalf("repolens.skill = %q", parsed.Metadata["repolens.skill"])
		}
		if parsed.Metadata["repolens.version"] != "1.7.0" {
			t.Fatalf("repolens.version = %q", parsed.Metadata["repolens.version"])
		}
		if parsed.Metadata["repolens.digest"] != shortDigest(parsed.Body) {
			t.Fatalf("repolens.digest = %q, body digest %q", parsed.Metadata["repolens.digest"], shortDigest(parsed.Body))
		}
	})
}

func diskSkillNames(t *testing.T) map[string]bool {
	t.Helper()
	dir := repoSkillsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "SKILL.md")); err != nil {
			continue
		}
		out[e.Name()] = true
	}
	return out
}

func repoSkillsDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "skills")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("skills dir: %v", err)
	}
	return dir
}
