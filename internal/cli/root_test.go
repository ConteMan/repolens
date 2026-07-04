package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	root := newRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	if got := out.String(); !strings.Contains(got, "repolens dev") {
		t.Errorf("unexpected version output: %q", got)
	}
}

func TestBuildNotImplemented(t *testing.T) {
	root := newRootCmd()
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"build", "."})

	if err := root.Execute(); err == nil {
		t.Fatal("expected build to return not-implemented error")
	}
}
