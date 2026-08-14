package render

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestMarkdownGolden(t *testing.T) {
	t.Parallel()

	md := NewMarkdown()
	src := readTestFile(t, "full.md")
	got, err := md.Render(src, PageRef{
		Path: "docs/full.md",
		Resolve: func(target string) string {
			switch target {
			case "docs/other.md":
				return "view"
			case "assets/logo.png":
				return "mirror"
			default:
				return ""
			}
		},
	}, MarkdownOptions{
		TOC:              true,
		Anchors:          true,
		Mermaid:          true,
		FrontmatterTitle: true,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	assertGolden(t, "full.html", string(got.HTML))

	if got.Title != "Front Matter Title" {
		t.Fatalf("Title = %q, want front matter title", got.Title)
	}
	if got.Meta["title"] != "Front Matter Title" {
		t.Fatalf("Meta[title] = %#v", got.Meta["title"])
	}
	if !got.HasMermaid {
		t.Fatalf("HasMermaid = false, want true")
	}
	wantTOC := []TOCItem{
		{
			Title:  "Document Heading",
			Anchor: "document-heading",
			Level:  1,
			Children: []TOCItem{
				{Title: "Table", Anchor: "table", Level: 2},
				{Title: "Tasks", Anchor: "tasks", Level: 2, Children: []TOCItem{
					{Title: "Mermaid", Anchor: "mermaid", Level: 3},
					{Title: "Code", Anchor: "code", Level: 3},
				}},
			},
		},
	}
	if !reflect.DeepEqual(got.TOC, wantTOC) {
		t.Fatalf("TOC = %#v, want %#v", got.TOC, wantTOC)
	}
}

func TestMarkdownTitleFallbackAndTOCThreshold(t *testing.T) {
	t.Parallel()

	got, err := NewMarkdown().Render([]byte("# Hello\n\n## One\n"), PageRef{Path: "docs/fallback.md"}, MarkdownOptions{
		TOC:              true,
		TOCMinHeadings:   3,
		FrontmatterTitle: true,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got.Title != "Hello" {
		t.Fatalf("Title = %q, want first H1", got.Title)
	}
	if got.TOC != nil {
		t.Fatalf("TOC = %#v, want nil below threshold", got.TOC)
	}
}

func TestMarkdownUnicodeHeadingIDs(t *testing.T) {
	t.Parallel()

	src := []byte("# Document Heading\n\n## 5.6 广告\n\n## 中文标题\n\n## API 接口\n\n## 重复 标题\n\n## 重复 标题\n\n## 😀\n")
	got, err := NewMarkdown().Render(src, PageRef{Path: "docs/headings.md"}, MarkdownOptions{
		TOC:            true,
		TOCMinHeadings: 1,
		Anchors:        true,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	for _, want := range []string{
		`<h1 id="document-heading">Document Heading <a class="anchor" href="#document-heading">`,
		`<h2 id="56-广告">5.6 广告 <a class="anchor" href="#56-广告">`,
		`<h2 id="中文标题">中文标题 <a class="anchor" href="#中文标题">`,
		`<h2 id="api-接口">API 接口 <a class="anchor" href="#api-接口">`,
		`<h2 id="重复-标题">重复 标题 <a class="anchor" href="#重复-标题">`,
		`<h2 id="重复-标题-1">重复 标题 <a class="anchor" href="#重复-标题-1">`,
		`<h2 id="heading">😀 <a class="anchor" href="#heading">`,
	} {
		requireContains(t, string(got.HTML), want)
	}

	wantTOC := []TOCItem{{
		Title:  "Document Heading",
		Anchor: "document-heading",
		Level:  1,
		Children: []TOCItem{
			{Title: "5.6 广告", Anchor: "56-广告", Level: 2},
			{Title: "中文标题", Anchor: "中文标题", Level: 2},
			{Title: "API 接口", Anchor: "api-接口", Level: 2},
			{Title: "重复 标题", Anchor: "重复-标题", Level: 2},
			{Title: "重复 标题", Anchor: "重复-标题-1", Level: 2},
			{Title: "😀", Anchor: "heading", Level: 2},
		},
	}}
	if !reflect.DeepEqual(got.TOC, wantTOC) {
		t.Fatalf("TOC = %#v, want %#v", got.TOC, wantTOC)
	}
}

func TestMarkdownTitleFilenameFallback(t *testing.T) {
	t.Parallel()

	got, err := NewMarkdown().Render([]byte("just a paragraph, no headings\n"), PageRef{Path: "docs/fallback.md"}, MarkdownOptions{
		FrontmatterTitle: true,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got.Title != "fallback.md" {
		t.Fatalf("Title = %q, want filename fallback", got.Title)
	}
}

func TestMarkdownRenderConcurrent(t *testing.T) {
	t.Parallel()

	md := NewMarkdown()
	src := readTestFile(t, "full.md")
	ref := PageRef{
		Path: "docs/full.md",
		Resolve: func(target string) string {
			if target == "docs/other.md" {
				return "view"
			}
			if target == "assets/logo.png" {
				return "mirror"
			}
			return ""
		},
	}
	opts := MarkdownOptions{TOC: true, Anchors: true, Mermaid: true, FrontmatterTitle: true}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := md.Render(src, ref, opts)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			if got.Title != "Front Matter Title" || !got.HasMermaid {
				t.Errorf("Render() = title %q, mermaid %v", got.Title, got.HasMermaid)
			}
		}()
	}
	wg.Wait()
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("update golden %s: %v", name, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	if got != string(want) {
		t.Fatalf("golden %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, string(want))
	}
}

func readTestFile(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}
	return data
}

func requireContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("output missing %q\n%s", substr, s)
	}
}
