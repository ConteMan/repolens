package cli

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/ConteMan/repolens/internal/ui"
)

// The native UI normally builds the working tree, while build defaults to HEAD.
// Pin both entry points to the same commit, including its file history.
func TestCLIAndNativeBuildParity(t *testing.T) {
	repo := newCLITestRepo(t)
	cliWriteFile(t, repo, "章节/first page.md", "# 中文文档\n\n## Section\n")
	cliWriteFile(t, repo, "theme.css", "h2 { color: green; }\n")
	cliWriteFile(t, repo, ".repolens.yml", "site:\n  lang: en\ntheme:\n  css: theme.css\n")
	runCLIGit(t, repo, "add", ".")
	runCLIGit(t, repo, "commit", "-m", "add parity fixture")
	dir := t.TempDir()
	cliWriteFile(t, dir, "builder.yml", "site:\n  title: Parity fixture\nsource:\n  ref: HEAD\n")
	config := filepath.Join(dir, "builder.yml")
	cliOutput := filepath.Join(dir, "CLI 产物")
	appOutput := filepath.Join(dir, "桌面 产物")
	cmd := newRootCmd()
	cmd.SetArgs([]string{"build", repo, "--config", config, "--output", cliOutput})
	if err := cmd.ExecuteContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := ui.BuildNative(t.Context(), repo, config, &appOutput, false); err != nil {
		t.Fatal(err)
	}
	want, got := parityFiles(t, cliOutput), parityFiles(t, appOutput)
	for name, data := range want {
		if !bytes.Equal(got[name], data) || got[name] == nil {
			t.Errorf("desktop output differs at %q", name)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("unexpected desktop output %q", name)
		}
	}
}

func parityFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	dir := os.DirFS(root)
	b, err := fs.ReadFile(dir, "_assets/snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var meta struct{ Snapshot string }
	if err := json.Unmarshal(b, &meta); err != nil || !regexp.MustCompile("^[a-f0-9]{32}$").MatchString(meta.Snapshot) {
		t.Fatalf("invalid snapshot metadata %q", b)
	}
	files := make(map[string][]byte)
	err = fs.WalkDir(dir, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(dir, path)
		if err != nil {
			return err
		}
		// Snapshot identifiers and build timestamps are unique to each invocation.
		data = bytes.ReplaceAll(data, []byte(meta.Snapshot), []byte("<snapshot>"))
		if path == "index.json" {
			var index struct {
				BuiltAt string `json:"built_at"`
			}
			if err := json.Unmarshal(data, &index); err != nil {
				return err
			}
			if _, err := time.Parse(time.RFC3339, index.BuiltAt); err != nil {
				return err
			}
			data = bytes.Replace(data, []byte(`"built_at": "`+index.BuiltAt+`"`), []byte(`"built_at": "<timestamp>"`), 1)
		}
		files[path] = data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
