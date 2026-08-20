package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Scope is whether a target lives in the project or the user profile.
type Scope int

const (
	ScopeProject Scope = iota
	ScopeGlobal
)

func (s Scope) String() string {
	switch s {
	case ScopeProject:
		return "project"
	case ScopeGlobal:
		return "global"
	default:
		return fmt.Sprintf("Scope(%d)", int(s))
	}
}

// Target is a directory an agent product scans for skills.
type Target struct {
	Key      string
	Dir      string // absolute
	Scope    Scope
	Detected bool
}

type targetSpec struct {
	key           string
	projectRel    []string
	globalRel     []string
	detectProject func(root string) bool
	detectGlobal  func(home string) bool
}

var targetSpecs = []targetSpec{
	{
		key:        "claude",
		projectRel: []string{".claude", "skills"},
		globalRel:  []string{".claude", "skills"},
		detectProject: func(root string) bool {
			return existsDir(filepath.Join(root, ".claude"))
		},
		detectGlobal: func(home string) bool {
			return existsDir(filepath.Join(home, ".claude"))
		},
	},
	{
		key:        "codex",
		projectRel: []string{".codex", "skills"},
		globalRel:  []string{".codex", "skills"},
		detectProject: func(root string) bool {
			return existsDir(filepath.Join(root, ".codex"))
		},
		detectGlobal: func(home string) bool {
			return existsDir(filepath.Join(home, ".codex"))
		},
	},
	{
		key:        "cursor",
		projectRel: []string{".cursor", "skills"},
		globalRel:  []string{".cursor", "skills"},
		detectProject: func(root string) bool {
			return existsDir(filepath.Join(root, ".cursor"))
		},
		detectGlobal: func(home string) bool {
			return existsDir(filepath.Join(home, ".cursor"))
		},
	},
	{
		key:        "copilot",
		projectRel: []string{".github", "skills"},
		globalRel:  []string{".config", "github-copilot", "skills"},
		detectProject: func(root string) bool {
			return exists(filepath.Join(root, ".github", "skills")) ||
				exists(filepath.Join(root, ".github", "copilot-instructions.md"))
		},
		detectGlobal: func(home string) bool {
			return existsDir(filepath.Join(home, ".config", "github-copilot"))
		},
	},
	{
		key:        "agents",
		projectRel: []string{".agents", "skills"},
		globalRel:  []string{".agents", "skills"},
		detectProject: func(root string) bool {
			return existsDir(filepath.Join(root, ".agents"))
		},
		detectGlobal: func(home string) bool {
			return existsDir(filepath.Join(home, ".agents"))
		},
	},
}

var userHomeDir = os.UserHomeDir

// KnownTargetKeys returns the install target keys in table order.
func KnownTargetKeys() []string {
	keys := make([]string, len(targetSpecs))
	for i, spec := range targetSpecs {
		keys[i] = spec.key
	}
	return keys
}

// Targets returns every known install target for scope. Project-scope
// detection signals are evaluated against root; global-scope directories and
// their detection signals are both resolved from the user home directory, so
// --global picks the agent products the user actually installed rather than
// whatever the current repository happens to configure.
func Targets(root string, scope Scope) ([]Target, error) {
	rootAbs, err := absRoot(root)
	if err != nil {
		return nil, err
	}
	var home string
	if scope == ScopeGlobal {
		home, err = userHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home directory: %w", err)
		}
		if home == "" {
			return nil, fmt.Errorf("resolve home directory: empty")
		}
		home, err = filepath.Abs(home)
		if err != nil {
			return nil, err
		}
	}
	out := make([]Target, 0, len(targetSpecs))
	for _, spec := range targetSpecs {
		t := Target{Key: spec.key, Scope: scope}
		switch scope {
		case ScopeGlobal:
			t.Dir = filepath.Join(append([]string{home}, spec.globalRel...)...)
			t.Detected = spec.detectGlobal(home)
		default:
			t.Dir = filepath.Join(append([]string{rootAbs}, spec.projectRel...)...)
			t.Detected = spec.detectProject(rootAbs)
		}
		out = append(out, t)
	}
	return out, nil
}

// Detect returns the targets in scope whose detection signal is present.
func Detect(root string, scope Scope) ([]Target, error) {
	all, err := Targets(root, scope)
	if err != nil {
		return nil, err
	}
	out := make([]Target, 0, len(all))
	for _, t := range all {
		if t.Detected {
			out = append(out, t)
		}
	}
	return out, nil
}

// ResolveTargets maps keys to targets in the given scope. Unknown keys return
// an error that lists the legal values.
func ResolveTargets(root string, scope Scope, keys []string) ([]Target, error) {
	all, err := Targets(root, scope)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]Target, len(all))
	for _, t := range all {
		byKey[t.Key] = t
	}
	out := make([]Target, 0, len(keys))
	for _, raw := range keys {
		k := strings.TrimSpace(raw)
		if k == "" {
			continue
		}
		t, ok := byKey[k]
		if !ok {
			return nil, fmt.Errorf("unknown target %q (want one of: %s)", k, strings.Join(KnownTargetKeys(), ", "))
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("unknown target %q (want one of: %s)", strings.Join(keys, ","), strings.Join(KnownTargetKeys(), ", "))
	}
	return out, nil
}

func absRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	return filepath.Abs(root)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func existsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
