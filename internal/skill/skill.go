// Package skill locates, installs, and updates the agent skills embedded in
// the repolens binary.
package skill

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/ConteMan/repolens/skills"
)

const namePrefix = "repolens-"

// Skill is a skill shipped inside the binary.
type Skill struct {
	Name        string // directory name, equal to the front matter name
	Alias       string // short name, Name without the "repolens-" prefix
	Description string
	Content     []byte // SKILL.md as embedded, without provenance
}

// Builtin returns every skill embedded in the binary, sorted by name.
func Builtin() ([]Skill, error) {
	matches, err := fs.Glob(skills.FS, "*/SKILL.md")
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	out := make([]Skill, 0, len(matches))
	for _, match := range matches {
		data, err := fs.ReadFile(skills.FS, match)
		if err != nil {
			return nil, err
		}
		parsed, err := parseSkillFile(data)
		if err != nil {
			return nil, fmt.Errorf("skill %s: %w", match, err)
		}
		dir := path.Dir(match)
		if err := validateEmbedded(match, dir, parsed); err != nil {
			return nil, err
		}
		out = append(out, Skill{
			Name:        parsed.Name,
			Alias:       strings.TrimPrefix(parsed.Name, namePrefix),
			Description: parsed.Description,
			Content:     data,
		})
	}
	return out, nil
}

func validateEmbedded(match, dir string, parsed parsedSkill) error {
	if parsed.hasMetadataField {
		return fmt.Errorf("skill %s: embedded SKILL.md must not contain metadata", match)
	}
	if parsed.Name != dir {
		return fmt.Errorf("skill %s: front matter name %q does not match directory %q", match, parsed.Name, dir)
	}
	if err := validateSkillName(parsed.Name); err != nil {
		return fmt.Errorf("skill %s: %w", match, err)
	}
	if strings.TrimSpace(parsed.Description) == "" {
		return fmt.Errorf("skill %s: description is empty", match)
	}
	if n := utf8.RuneCountInString(parsed.Description); n > 1024 {
		return fmt.Errorf("skill %s: description is %d characters, want ≤1024", match, n)
	}
	return nil
}

// Lookup returns the bundled skill matching name or its short alias.
func Lookup(name string) (Skill, bool, error) {
	all, err := Builtin()
	if err != nil {
		return Skill{}, false, err
	}
	for _, s := range all {
		if s.Name == name {
			return s, true, nil
		}
	}
	for _, s := range all {
		if s.Alias == name {
			return s, true, nil
		}
	}
	return Skill{}, false, nil
}

func builtinNames() (map[string]struct{}, error) {
	all, err := Builtin()
	if err != nil {
		return nil, err
	}
	names := make(map[string]struct{}, len(all))
	for _, s := range all {
		names[s.Name] = struct{}{}
	}
	return names, nil
}
