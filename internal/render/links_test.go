package render

import (
	"fmt"
	"strings"
	"testing"
)

func TestLinkRewriteBranches(t *testing.T) {
	t.Parallel()

	ref := PageRef{
		Path: "docs/guide/current.md",
		Resolve: func(target string) string {
			switch target {
			case "docs/README.md", "docs/guide/other.md", "docs/a/d/e.md", "docs/guide/my file.md",
				"docs/guide/wireframes/index.html", "index.html":
				return "view"
			case "docs/guide/download.zip", "docs/guide/picture.png":
				return "mirror"
			default:
				return ""
			}
		},
	}

	tests := []struct {
		name  string
		image bool
		raw   string
		want  string
	}{
		{name: "absolute URL", raw: "https://example.com/a.md", want: "https://example.com/a.md"},
		{name: "scheme relative", raw: "//cdn.example.com/a.png", want: "//cdn.example.com/a.png"},
		{name: "page anchor", raw: "#intro", want: "#intro"},
		{name: "mailto", raw: "mailto:dev@example.com", want: "mailto:dev@example.com"},
		{name: "root absolute", raw: "/docs/a.md", want: "/docs/a.md"},
		{name: "view link with fragment", raw: "../README.md#top", want: "../../README.md/#top"},
		{name: "dot-slash prefix", raw: "./other.md", want: "../other.md/"},
		{name: "percent-encoded path", raw: "my%20file.md", want: "../my file.md/"},
		{name: "mirror link", raw: "download.zip", want: "../../../../docs/guide/download.zip"},
		{name: "image always mirror", image: true, raw: "picture.png", want: "../../../../docs/guide/picture.png"},
		{name: "image target with view still mirrors", image: true, raw: "other.md", want: "../../../../docs/guide/other.md"},
		{name: "unknown target", raw: "missing.md", want: "missing.md"},
		{name: "escape repo root", raw: "../../../secret.md", want: "../../../secret.md"},
		{name: "query and fragment", raw: "other.md?raw=1#sec", want: "../other.md/?raw=1#sec"},
		// 可渲染 index.html 并入目录页（Issue #9），链接落到目录 view URL。
		{name: "dir index.html merges to dir page", raw: "wireframes/index.html", want: "../wireframes/"},
		{name: "root index.html merges to view root", raw: "../../index.html", want: "../../../"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := rewriteLink(tt.raw, ref, tt.image)
			if got != tt.want {
				t.Fatalf("rewriteLink(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestLinkRewriteASTTransformer(t *testing.T) {
	t.Parallel()

	src := []byte("[doc](other.md#part)\n\n![pic](picture.png)\n")
	got, err := NewMarkdown().Render(src, PageRef{
		Path: "docs/guide/current.md",
		Resolve: func(target string) string {
			switch target {
			case "docs/guide/other.md":
				return "view"
			case "docs/guide/picture.png":
				return "mirror"
			default:
				return ""
			}
		},
	}, MarkdownOptions{})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	html := string(got.HTML)
	requireContains(t, html, `href="../other.md/#part"`)
	requireContains(t, html, `src="../../../../docs/guide/picture.png"`)
}

func TestRelTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from string
		to   string
		want string
	}{
		{from: "view/docs/a.md/", to: "view/docs/b.md/", want: "../b.md/"},
		{from: "view/docs/a/b.md/", to: "assets/logo.png", want: "../../../../assets/logo.png"},
		{from: "view/README.md/", to: "view/docs/intro.md/", want: "../docs/intro.md/"},
		{from: "view/docs/current.md", to: "view/docs/other.md/", want: "../other.md/"},
		{from: "view/docs/a/b/c.md/", to: "view/docs/a/d/e.md/", want: "../../d/e.md/"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%s_to_%s", tt.from, tt.to), func(t *testing.T) {
			t.Parallel()
			got := RelTo(tt.from, tt.to)
			if got != tt.want {
				t.Fatalf("RelTo(%q, %q) = %q, want %q", tt.from, tt.to, got, tt.want)
			}
			if strings.Contains(got, "//") {
				t.Fatalf("RelTo(%q, %q) contains duplicate slash: %q", tt.from, tt.to, got)
			}
		})
	}
}
