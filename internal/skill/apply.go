package skill

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// State is the classification of a skill copy on disk.
type State int

const (
	StateAbsent State = iota
	StateCurrent
	StateOutdated
	StateModified
	StateForeign
)

func (s State) String() string {
	switch s {
	case StateAbsent:
		return "absent"
	case StateCurrent:
		return "current"
	case StateOutdated:
		return "outdated"
	case StateModified:
		return "modified"
	case StateForeign:
		return "foreign"
	default:
		return fmt.Sprintf("State(%d)", int(s))
	}
}

// Installed is one copy of a skill found on disk.
type Installed struct {
	Path    string
	Target  Target
	Name    string
	Version string
	State   State
}

// Options controls write behaviour for Install and Update.
type Options struct {
	Force  bool
	DryRun bool
}

// Action is what Install or Update did (or would do, for DryRun).
type Action int

const (
	ActionCreated Action = iota
	ActionUpdated
	ActionSkipped
)

func (a Action) String() string {
	switch a {
	case ActionCreated:
		return "created"
	case ActionUpdated:
		return "updated"
	case ActionSkipped:
		return "skipped"
	default:
		return fmt.Sprintf("Action(%d)", int(a))
	}
}

// Result is one target's outcome.
type Result struct {
	Path    string
	Action  Action
	State   State // state found before writing
	Warning string
}

const (
	// WarnModified is returned when a provenance copy has a locally edited body.
	WarnModified = "locally modified; use --force to overwrite"
	// WarnForeign is returned when a same-name file has no repolens provenance.
	WarnForeign = "existing skill has no repolens provenance; use --force to overwrite"
	// WarnUnbundled is returned when a provenance copy names a skill no longer shipped.
	WarnUnbundled = "skill is no longer bundled; left in place"
)

// Scan finds installed copies under all known project and personal targets.
// version is the current binary version, used to classify current vs outdated.
func Scan(root, version string) ([]Installed, error) {
	names, err := builtinNames()
	if err != nil {
		return nil, err
	}
	project, err := Targets(root, ScopeProject)
	if err != nil {
		return nil, err
	}
	global, err := Targets(root, ScopeGlobal)
	if err != nil {
		return nil, err
	}
	var out []Installed
	for _, t := range append(append([]Target{}, project...), global...) {
		found, err := scanTarget(t, version, names)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func scanTarget(t Target, version string, names map[string]struct{}) ([]Installed, error) {
	entries, err := os.ReadDir(t.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Installed
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(t.Dir, e.Name(), "SKILL.md")
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		state, name, ver, err := inspectPath(p, version)
		if err != nil {
			return nil, err
		}
		if name == "" {
			name = e.Name()
		}
		_, named := names[name]
		_, dirNamed := names[e.Name()]
		if state == StateForeign && !named && !dirNamed {
			continue
		}
		out = append(out, Installed{
			Path:    p,
			Target:  t,
			Name:    name,
			Version: ver,
			State:   state,
		})
	}
	return out, nil
}

func inspectPath(path, currentVersion string) (state State, name, version string, err error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return StateAbsent, "", "", nil
	}
	if err != nil {
		return 0, "", "", err
	}
	parsed, err := parseSkillFile(data)
	if err != nil {
		return StateForeign, filepath.Base(filepath.Dir(path)), "", nil
	}
	dirName := filepath.Base(filepath.Dir(path))
	skillName, recVer, recDigest, ok := parsed.provenance()
	if !ok {
		name = parsed.Name
		if name == "" {
			name = dirName
		}
		return StateForeign, name, "", nil
	}
	if shortDigest(parsed.Body) != recDigest {
		return StateModified, skillName, recVer, nil
	}
	if recVer == currentVersion {
		return StateCurrent, skillName, recVer, nil
	}
	return StateOutdated, skillName, recVer, nil
}

// Install writes s into t, injecting provenance for version.
func Install(s Skill, t Target, version string, opts Options) (Result, error) {
	dest := filepath.Join(t.Dir, s.Name, "SKILL.md")
	state, _, _, err := inspectPath(dest, version)
	if err != nil {
		return Result{}, err
	}
	switch state {
	case StateAbsent:
		return writeSkill(s, dest, version, opts, state, ActionCreated)
	case StateCurrent:
		return Result{Path: dest, Action: ActionSkipped, State: state}, nil
	case StateOutdated:
		return writeSkill(s, dest, version, opts, state, ActionUpdated)
	case StateModified, StateForeign:
		if opts.Force {
			return writeSkill(s, dest, version, opts, state, ActionUpdated)
		}
		warn := WarnModified
		if state == StateForeign {
			warn = WarnForeign
		}
		return Result{Path: dest, Action: ActionSkipped, State: state, Warning: warn}, nil
	default:
		return Result{Path: dest, Action: ActionSkipped, State: state}, nil
	}
}

// Update rewrites provenance copies under root that are outdated relative to
// version. Copies whose body was edited locally are skipped unless Force is set.
func Update(root, version string, opts Options) ([]Result, error) {
	installed, err := Scan(root, version)
	if err != nil {
		return nil, err
	}
	var out []Result
	for _, item := range installed {
		if item.State == StateForeign {
			continue
		}
		s, ok, err := Lookup(item.Name)
		if err != nil {
			return nil, err
		}
		if !ok {
			out = append(out, Result{
				Path:    item.Path,
				Action:  ActionSkipped,
				State:   item.State,
				Warning: WarnUnbundled,
			})
			continue
		}
		res, err := applyUpdate(s, item, version, opts)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func applyUpdate(s Skill, item Installed, version string, opts Options) (Result, error) {
	switch item.State {
	case StateCurrent:
		if version == "dev" || item.Version == "dev" {
			return writeSkill(s, item.Path, version, opts, item.State, ActionUpdated)
		}
		return Result{Path: item.Path, Action: ActionSkipped, State: item.State}, nil
	case StateOutdated:
		return writeSkill(s, item.Path, version, opts, item.State, ActionUpdated)
	case StateModified:
		if opts.Force {
			return writeSkill(s, item.Path, version, opts, item.State, ActionUpdated)
		}
		return Result{Path: item.Path, Action: ActionSkipped, State: item.State, Warning: WarnModified}, nil
	default:
		return Result{Path: item.Path, Action: ActionSkipped, State: item.State}, nil
	}
}

func writeSkill(s Skill, dest, version string, opts Options, before State, action Action) (Result, error) {
	data, err := materialize(s, version)
	if err != nil {
		return Result{}, err
	}
	res := Result{Path: dest, Action: action, State: before}
	if opts.DryRun {
		return res, nil
	}
	if err := writeFileAtomic(dest, data); err != nil {
		return Result{}, err
	}
	return res, nil
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".SKILL.md.*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(0o644); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		if err2 := os.Remove(path); err2 == nil {
			err = os.Rename(tmp, path)
		}
		if err != nil {
			return err
		}
	}
	removeTmp = false
	return nil
}
