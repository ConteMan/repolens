package render

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/yuin/goldmark/ast"
)

func sampleGlossary() Glossary {
	return Glossary{
		"mediation": {
			Key:     "mediation",
			Title:   ParseGlossaryText("Mediation"),
			Alias:   ParseGlossaryText("聚合"),
			Summary: ParseGlossaryText("A mediation layer."),
			Source: &GlossarySource{
				Label: ParseGlossaryText("Old Docs"),
				URL:   "https://old.example/docs",
			},
		},
		"organic": {
			Key:     "organic",
			Title:   ParseGlossaryText("`organic` traffic"),
			Summary: ParseGlossaryText("Non-paid users."),
		},
	}
}

func TestParseGlossaryText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		wantText string
		wantHTML string
	}{
		{name: "empty", raw: "", wantText: "", wantHTML: ""},
		{name: "plain text", raw: "plain explanation", wantText: "plain explanation", wantHTML: "plain explanation"},
		{name: "paired backticks", raw: "use `af_ad_revenue` event", wantText: "use af_ad_revenue event", wantHTML: "use <code>af_ad_revenue</code> event"},
		{name: "multiple pairs", raw: "`a` and `b`", wantText: "a and b", wantHTML: "<code>a</code> and <code>b</code>"},
		{name: "unpaired trailing", raw: "use `af_ad_revenue", wantText: "use `af_ad_revenue", wantHTML: "use `af_ad_revenue"},
		{name: "unpaired only", raw: "`", wantText: "`", wantHTML: "`"},
		{name: "empty pair", raw: "empty `` pair", wantText: "empty `` pair", wantHTML: "empty `` pair"},
		{name: "only empty pair", raw: "``", wantText: "``", wantHTML: "``"},
		{name: "escape is literal", raw: "\\`code\\`", wantText: "\\code\\", wantHTML: "\\<code>code\\</code>"},
		{name: "html outside code", raw: "<b>`x`</b> and **md**", wantText: "<b>x</b> and **md**", wantHTML: "&lt;b&gt;<code>x</code>&lt;/b&gt; and **md**"},
		{name: "html inside code", raw: "`<script>alert(1)</script>`", wantText: "<script>alert(1)</script>", wantHTML: "<code>&lt;script&gt;alert(1)&lt;/script&gt;</code>"},
		{name: "unicode around code", raw: "术语 `af_ad_revenue` 事件", wantText: "术语 af_ad_revenue 事件", wantHTML: "术语 <code>af_ad_revenue</code> 事件"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ParseGlossaryText(tt.raw)
			if got.Text != tt.wantText {
				t.Fatalf("Text = %q, want %q", got.Text, tt.wantText)
			}
			if string(got.HTML) != tt.wantHTML {
				t.Fatalf("HTML = %q, want %q", got.HTML, tt.wantHTML)
			}
		})
	}
}

func TestGlossaryDisabledDoesNotChangeTermFreeOutput(t *testing.T) {
	t.Parallel()

	md := NewMarkdown()
	src := readTestFile(t, "full.md")
	ref := PageRef{
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
	}
	base := MarkdownOptions{TOC: true, Anchors: true, Mermaid: true, FrontmatterTitle: true}
	off, err := md.Render(src, ref, base)
	if err != nil {
		t.Fatalf("disabled Render() error = %v", err)
	}
	on, err := md.Render(src, ref, MarkdownOptions{
		TOC: true, Anchors: true, Mermaid: true, FrontmatterTitle: true,
		Glossary: true, Terms: sampleGlossary(),
	})
	if err != nil {
		t.Fatalf("enabled Render() error = %v", err)
	}
	if off.HTML != on.HTML {
		t.Fatalf("Glossary true changed term-free output\n--- off ---\n%s\n--- on ---\n%s", off.HTML, on.HTML)
	}
	if on.Terms != nil {
		t.Fatalf("Terms = %#v, want nil when nothing is referenced", on.Terms)
	}
	assertGolden(t, "full.html", string(off.HTML))
}

func TestGlossarySyntax(t *testing.T) {
	t.Parallel()

	lib := sampleGlossary()
	const path = "docs/guide.md"

	tests := []struct {
		name        string
		src         string
		opts        MarkdownOptions
		contains    []string
		notContains []string
		wantKeys    []string
		wantErr     bool
		errKey      string
		errLine     int
		ariaLabel   string
		imageKept   bool
	}{
		{
			name: "valid key",
			src:  "See [广告聚合](term:mediation).\n",
			opts: MarkdownOptions{Glossary: true, Terms: lib, GlossaryTermLabel: "%s（术语，查看解释）"},
			contains: []string{
				`<a class="term" href="#glossary-mediation" data-glossary="mediation" aria-label="广告聚合（术语，查看解释）">广告聚合</a>`,
			},
			wantKeys: []string{"mediation"},
		},
		{
			name: "case normalized",
			src:  "See [广告聚合](term:Mediation).\n",
			opts: MarkdownOptions{Glossary: true, Terms: lib},
			contains: []string{
				`href="#glossary-mediation"`,
				`data-glossary="mediation"`,
			},
			wantKeys: []string{"mediation"},
		},
		{
			name:    "illegal key underscore",
			src:     "See [x](term:foo_bar).\n",
			opts:    MarkdownOptions{Glossary: true, Terms: lib, GlossaryStrict: GlossaryStrictRefs},
			wantErr: true,
			errKey:  "foo_bar",
			errLine: 1,
		},
		{
			name:    "illegal key leading hyphen",
			src:     "See [x](term:-abc).\n",
			opts:    MarkdownOptions{Glossary: true, Terms: lib, GlossaryStrict: GlossaryStrictRefs},
			wantErr: true,
			errKey:  "-abc",
			errLine: 1,
		},
		{
			name:    "query treated as undefined",
			src:     "See [x](term:mediation?x=1).\n",
			opts:    MarkdownOptions{Glossary: true, Terms: lib, GlossaryStrict: GlossaryStrictRefs},
			wantErr: true,
			errKey:  "mediation",
			errLine: 1,
		},
		{
			name:    "fragment treated as undefined",
			src:     "See [x](term:mediation#z).\n",
			opts:    MarkdownOptions{Glossary: true, Terms: lib, GlossaryStrict: GlossaryStrictRefs},
			wantErr: true,
			errKey:  "mediation",
			errLine: 1,
		},
		{
			name: "image term destination kept",
			src:  "![chart](term:mediation)\n",
			opts: MarkdownOptions{Glossary: true, Terms: lib},
			contains: []string{
				`src="term:mediation"`,
			},
			notContains: []string{`class="term"`},
			imageKept:   true,
		},
		{
			name: "undefined off degrades to text",
			src:  "See [missing](term:unknown).\n",
			opts: MarkdownOptions{Glossary: true, Terms: lib, GlossaryStrict: GlossaryStrictOff},
			contains: []string{
				`<p>See missing.</p>`,
			},
			notContains: []string{`<a `, `class="term"`},
		},
		{
			name:    "undefined refs errors",
			src:     "See [missing](term:unknown).\n",
			opts:    MarkdownOptions{Glossary: true, Terms: lib, GlossaryStrict: GlossaryStrictRefs},
			wantErr: true,
			errKey:  "unknown",
			errLine: 1,
		},
		{
			name:    "undefined complete errors",
			src:     "See [missing](term:unknown).\n",
			opts:    MarkdownOptions{Glossary: true, Terms: lib, GlossaryStrict: GlossaryStrictComplete},
			wantErr: true,
			errKey:  "unknown",
			errLine: 1,
		},
		{
			name:    "empty strict equals refs",
			src:     "See [missing](term:unknown).\n",
			opts:    MarkdownOptions{Glossary: true, Terms: lib},
			wantErr: true,
			errKey:  "unknown",
			errLine: 1,
		},
		{
			name: "disabled strips to text",
			src:  "See [广告聚合](term:mediation).\n",
			opts: MarkdownOptions{Glossary: false, Terms: lib, GlossaryStrict: GlossaryStrictRefs},
			contains: []string{
				`<p>See 广告聚合.</p>`,
			},
			notContains: []string{`<a `, `class="term"`},
		},
		{
			name: "disabled illegal key still strips",
			src:  "See [x](term:foo_bar).\n",
			opts: MarkdownOptions{},
			contains: []string{
				`<p>See x.</p>`,
			},
			notContains: []string{`<a `, `class="term"`},
		},
		{
			name: "reference style term link",
			src:  "[广告聚合][m]\n\n[m]: term:mediation\n",
			opts: MarkdownOptions{Glossary: true, Terms: lib},
			contains: []string{
				`data-glossary="mediation"`,
			},
			wantKeys: []string{"mediation"},
		},
		{
			name: "aria-label uses display text plus format",
			src:  "See [自然量](term:organic).\n",
			opts: MarkdownOptions{Glossary: true, Terms: lib, GlossaryTermLabel: "%s (term, view definition)"},
			contains: []string{
				`aria-label="自然量 (term, view definition)"`,
				`data-glossary="organic"`,
			},
			wantKeys: []string{"organic"},
		},
		{
			name: "empty label format omits aria-label",
			src:  "See [广告聚合](term:mediation).\n",
			opts: MarkdownOptions{Glossary: true, Terms: lib},
			contains: []string{
				`<a class="term" href="#glossary-mediation" data-glossary="mediation">广告聚合</a>`,
			},
			notContains: []string{`aria-label=`},
			wantKeys:    []string{"mediation"},
		},
		{
			name: "duplicate refs keep first-seen order",
			src:  "[b](term:organic) then [a](term:mediation) then [b](term:organic).\n",
			opts: MarkdownOptions{Glossary: true, Terms: lib},
			contains: []string{
				`data-glossary="organic"`,
				`data-glossary="mediation"`,
			},
			wantKeys: []string{"organic", "mediation"},
		},
		{
			name: "emphasis inside display text",
			src:  "[**bold**](term:mediation)\n",
			opts: MarkdownOptions{Glossary: true, Terms: lib, GlossaryTermLabel: "%s (term, view definition)"},
			contains: []string{
				`<a class="term" href="#glossary-mediation" data-glossary="mediation" aria-label="bold (term, view definition)"><strong>bold</strong></a>`,
			},
			wantKeys: []string{"mediation"},
		},
		{
			name: "disabled unwrap keeps emphasis",
			src:  "[**bold**](term:mediation)\n",
			opts: MarkdownOptions{},
			contains: []string{
				`<p><strong>bold</strong></p>`,
			},
			notContains: []string{`<a `, `class="term"`},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewMarkdown().Render([]byte(tt.src), PageRef{Path: path}, tt.opts)
			html := string(got.HTML)
			if !tt.imageKept {
				requireNoTermHref(t, html)
			}
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Render() error = nil, want error for key %q\n%s", tt.errKey, html)
				}
				msg := err.Error()
				if !strings.Contains(msg, tt.errKey) {
					t.Fatalf("error %q missing key %q", msg, tt.errKey)
				}
				if !strings.Contains(msg, path) {
					t.Fatalf("error %q missing path %q", msg, path)
				}
				if tt.errLine > 0 && !strings.Contains(msg, fmt.Sprintf(":%d", tt.errLine)) {
					t.Fatalf("error %q missing line %d", msg, tt.errLine)
				}
				return
			}
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			for _, want := range tt.contains {
				requireContains(t, html, want)
			}
			for _, not := range tt.notContains {
				requireNotContains(t, html, not)
			}
			gotKeys := termKeys(got.Terms)
			if strings.Join(gotKeys, ",") != strings.Join(tt.wantKeys, ",") {
				t.Fatalf("Terms keys = %v, want %v", gotKeys, tt.wantKeys)
			}
			if tt.ariaLabel != "" {
				if len(got.Terms) == 0 || got.Terms[0].Title.Text != tt.ariaLabel {
					t.Fatalf("Title.Text = %q, want aria-label form %q", firstTermText(got.Terms), tt.ariaLabel)
				}
			}
		})
	}
}

func TestGlossaryUndefinedErrorLineNumber(t *testing.T) {
	t.Parallel()

	src := []byte(`---
title: Attribution
---

Intro.

See [missing](term:unknown).
`)
	_, err := NewMarkdown().Render(src, PageRef{Path: "docs/a.md"}, MarkdownOptions{
		Glossary: true,
		Terms:    sampleGlossary(),
	})
	if err == nil {
		t.Fatal("Render() error = nil, want undefined term")
	}
	msg := err.Error()
	for _, want := range []string{`unknown`, `docs/a.md`, `:7`} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestGlossaryMerge(t *testing.T) {
	t.Parallel()

	lib := sampleGlossary()

	t.Run("public library only", func(t *testing.T) {
		t.Parallel()
		got, err := NewMarkdown().Render([]byte("See [广告聚合](term:mediation).\n"), PageRef{Path: "docs/a.md"}, MarkdownOptions{
			Glossary: true,
			Terms:    lib,
		})
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if len(got.Terms) != 1 {
			t.Fatalf("Terms len = %d, want 1", len(got.Terms))
		}
		term := got.Terms[0]
		if term.Title.Text != "Mediation" || term.Summary.Text != "A mediation layer." {
			t.Fatalf("term = %+v", term)
		}
		if term.Page.Text != "" {
			t.Fatalf("Page = %q, want empty", term.Page.Text)
		}
		if term.Source == nil || term.Source.URL != "https://old.example/docs" {
			t.Fatalf("Source = %+v", term.Source)
		}
	})

	t.Run("field-level override", func(t *testing.T) {
		t.Parallel()
		lib := sampleGlossary()
		med := lib["mediation"]
		med.DefinedIn = ".repolens/glossary/mediation.yml"
		lib["mediation"] = med
		src := []byte(`---
glossary:
  mediation:
    page: 主聚合平台决定展示哪家广告源的广告。
    warning: 不要与渠道混淆。
---

See [广告聚合](term:mediation).
`)
		got, err := NewMarkdown().Render(src, PageRef{Path: "docs/a.md"}, MarkdownOptions{
			Glossary: true,
			Terms:    lib,
		})
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if len(got.Terms) != 1 {
			t.Fatalf("Terms len = %d", len(got.Terms))
		}
		term := got.Terms[0]
		if term.Title.Text != "Mediation" {
			t.Fatalf("Title overridden unexpectedly: %q", term.Title.Text)
		}
		if term.Summary.Text != "A mediation layer." {
			t.Fatalf("Summary overridden unexpectedly: %q", term.Summary.Text)
		}
		if term.Page.Text != "主聚合平台决定展示哪家广告源的广告。" {
			t.Fatalf("Page = %q", term.Page.Text)
		}
		if term.Warning.Text != "不要与渠道混淆。" {
			t.Fatalf("Warning = %q", term.Warning.Text)
		}
		if lib["mediation"].Page.Text != "" {
			t.Fatal("public glossary Page was mutated")
		}
		if got.Terms[0].DefinedIn != ".repolens/glossary/mediation.yml" {
			t.Fatalf("DefinedIn = %q, want public library path", got.Terms[0].DefinedIn)
		}
	})

	t.Run("source replaced as a whole", func(t *testing.T) {
		t.Parallel()
		src := []byte(`---
glossary:
  mediation:
    source:
      label: New Docs
      url: https://new.example/docs
---

See [广告聚合](term:mediation).
`)
		got, err := NewMarkdown().Render(src, PageRef{Path: "docs/a.md"}, MarkdownOptions{
			Glossary: true,
			Terms:    lib,
		})
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		term := got.Terms[0]
		if term.Source == nil {
			t.Fatal("Source = nil")
		}
		if term.Source.Label.Text != "New Docs" || term.Source.URL != "https://new.example/docs" {
			t.Fatalf("Source = %+v", term.Source)
		}
		if lib["mediation"].Source == nil || lib["mediation"].Source.URL != "https://old.example/docs" {
			t.Fatalf("public source mutated: %+v", lib["mediation"].Source)
		}
		if lib["mediation"].Source.Label.Text != "Old Docs" {
			t.Fatal("public source label mutated")
		}
	})

	t.Run("private term", func(t *testing.T) {
		t.Parallel()
		src := []byte(`---
glossary:
  local-only:
    title: Local Term
    summary: Visible only on this page.
---

See [本地](term:local-only).
`)
		got, err := NewMarkdown().Render(src, PageRef{Path: "docs/a.md"}, MarkdownOptions{
			Glossary: true,
			Terms:    lib,
		})
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if _, ok := lib["local-only"]; ok {
			t.Fatal("private term leaked into public glossary")
		}
		if len(got.Terms) != 1 || got.Terms[0].Key != "local-only" {
			t.Fatalf("Terms = %+v", got.Terms)
		}
		if got.Terms[0].Title.Text != "Local Term" {
			t.Fatalf("Title = %q", got.Terms[0].Title.Text)
		}
		requireContains(t, string(got.HTML), `data-glossary="local-only"`)
	})

	t.Run("empty title after merge is undefined", func(t *testing.T) {
		t.Parallel()
		src := []byte(`---
glossary:
  mystery:
    summary: has summary but no title
---

See [x](term:mystery).
`)
		_, err := NewMarkdown().Render(src, PageRef{Path: "docs/a.md"}, MarkdownOptions{
			Glossary:       true,
			GlossaryStrict: GlossaryStrictRefs,
			Terms:          lib,
		})
		if err == nil {
			t.Fatal("Render() error = nil, want undefined for empty title")
		}
		if !strings.Contains(err.Error(), "mystery") || !strings.Contains(err.Error(), "docs/a.md") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("empty string override does not wipe public field", func(t *testing.T) {
		t.Parallel()
		src := []byte(`---
glossary:
  mediation:
    summary: ""
    page: 本文覆盖。
---

See [广告聚合](term:mediation).
`)
		got, err := NewMarkdown().Render(src, PageRef{Path: "docs/a.md"}, MarkdownOptions{
			Glossary: true,
			Terms:    lib,
		})
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if got.Terms[0].Summary.Text != "A mediation layer." {
			t.Fatalf("empty summary wiped public field: %q", got.Terms[0].Summary.Text)
		}
		if got.Terms[0].Page.Text != "本文覆盖。" {
			t.Fatalf("Page = %q", got.Terms[0].Page.Text)
		}
	})
}

func TestGlossaryIsIncomplete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		term GlossaryTerm
		want bool
	}{
		{name: "both empty", want: true},
		{name: "whitespace only", term: GlossaryTerm{Summary: ParseGlossaryText("  "), Page: ParseGlossaryText("\n")}, want: true},
		{name: "summary only", term: GlossaryTerm{Summary: ParseGlossaryText("explained")}, want: false},
		{name: "page only", term: GlossaryTerm{Page: ParseGlossaryText("this doc")}, want: false},
		{name: "both set", term: GlossaryTerm{Summary: ParseGlossaryText("a"), Page: ParseGlossaryText("b")}, want: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.term.IsIncomplete(); got != tt.want {
				t.Fatalf("IsIncomplete() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("page only after merge is complete", func(t *testing.T) {
		t.Parallel()
		src := []byte(`---
glossary:
  mediation:
    page: 本文特化解释。
---

See [广告聚合](term:mediation).
`)
		lib := Glossary{
			"mediation": {Key: "mediation", Title: ParseGlossaryText("Mediation")},
		}
		got, err := NewMarkdown().Render(src, PageRef{Path: "docs/a.md"}, MarkdownOptions{
			Glossary:       true,
			GlossaryStrict: GlossaryStrictComplete,
			Terms:          lib,
		})
		if err != nil {
			t.Fatalf("Render() error = %v", err)
		}
		if got.Terms[0].IsIncomplete() {
			t.Fatal("page-only term reported incomplete")
		}
	})
}

func TestGlossaryFieldEscaping(t *testing.T) {
	t.Parallel()

	src := []byte(`---
glossary:
  xss:
    title: "<img src=x onerror=alert(1)>"
    summary: "use ` + "`" + `<script>` + "`" + ` and **bold**"
    page: "[link](https://example.com)"
---

See [bad](term:xss).
`)
	got, err := NewMarkdown().Render(src, PageRef{Path: "docs/a.md"}, MarkdownOptions{Glossary: true})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	term := got.Terms[0]
	if term.Title.Text != "<img src=x onerror=alert(1)>" {
		t.Fatalf("Title.Text = %q", term.Title.Text)
	}
	if string(term.Title.HTML) != "&lt;img src=x onerror=alert(1)&gt;" {
		t.Fatalf("Title.HTML = %q", term.Title.HTML)
	}
	requireContains(t, string(got.HTML), `data-glossary="xss"`)
	requireNotContains(t, string(got.HTML), `aria-label=`)
	if !strings.Contains(string(term.Summary.HTML), "<code>&lt;script&gt;</code>") {
		t.Fatalf("Summary.HTML = %q", term.Summary.HTML)
	}
	if strings.Contains(string(term.Summary.HTML), "<script>") {
		t.Fatalf("Summary.HTML leaked raw script: %q", term.Summary.HTML)
	}
	if term.Page.Text != "[link](https://example.com)" {
		t.Fatalf("Page.Text = %q", term.Page.Text)
	}
	if strings.Contains(string(term.Page.HTML), "<a ") {
		t.Fatalf("Page.HTML parsed markdown: %q", term.Page.HTML)
	}
	requireNoTermHref(t, string(got.HTML))
}

func TestContainsTermScheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want bool
	}{
		{name: "empty", src: "", want: false},
		{name: "plain markdown", src: "# Hello\n\nNo annotations here.\n", want: false},
		{name: "inline annotation", src: "See [x](term:mediation).\n", want: true},
		{name: "uppercase scheme", src: "See [x](TERM:mediation).\n", want: true},
		{name: "reference definition", src: "[x][m]\n\n[m]: term:mediation\n", want: true},
		{name: "angle destination", src: "[x](<term:foo>)\n", want: true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := containsTermScheme([]byte(tt.src)); got != tt.want {
				t.Fatalf("containsTermScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlossaryPrecheckSkipsWalk(t *testing.T) {
	t.Parallel()

	walked := false
	walk := func(*ast.Document, []byte) []termHit {
		walked = true
		return nil
	}

	if _, _, err := applyGlossaryWithWalk(nil, []byte("# Hello\n\nplain text\n"), PageRef{Path: "docs/a.md"}, MarkdownOptions{Glossary: true, Terms: sampleGlossary()}, walk); err != nil {
		t.Fatalf("applyGlossaryWithWalk() error = %v", err)
	}
	if walked {
		t.Fatal("glossary AST walk ran for source without term: annotation")
	}

	if _, _, err := applyGlossaryWithWalk(nil, []byte("See [x](term:mediation).\n"), PageRef{Path: "docs/a.md"}, MarkdownOptions{Glossary: true}, walk); err != nil {
		t.Fatalf("applyGlossaryWithWalk() error = %v", err)
	}
	if !walked {
		t.Fatal("glossary AST walk skipped after precheck hit")
	}
}

func TestGlossaryInvalidSourceURLDropped(t *testing.T) {
	t.Parallel()

	src := []byte(`---
glossary:
  mediation:
    source:
      label: Exploit
      url: javascript:alert(1)
---

See [广告聚合](term:mediation).
`)
	got, err := NewMarkdown().Render(src, PageRef{Path: "docs/a.md"}, MarkdownOptions{
		Glossary: true,
		Terms:    sampleGlossary(),
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got.Terms[0].Source != nil {
		t.Fatalf("javascript: source kept: %+v", got.Terms[0].Source)
	}
	if len(got.Warnings) != 1 {
		t.Fatalf("Warnings = %#v, want 1", got.Warnings)
	}
	for _, want := range []string{"mediation", "docs/a.md", "javascript:alert(1)"} {
		if !strings.Contains(got.Warnings[0], want) {
			t.Fatalf("warning %q missing %q", got.Warnings[0], want)
		}
	}
}

func TestGlossaryFrontMatterFieldTruncated(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("x", glossaryFieldLimit+1)
	src := fmt.Sprintf(`---
glossary:
  mediation:
    summary: %q
---

See [广告聚合](term:mediation).
`, long)
	got, err := NewMarkdown().Render([]byte(src), PageRef{Path: "docs/a.md"}, MarkdownOptions{
		Glossary: true,
		Terms:    sampleGlossary(),
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got.Terms[0].Summary.Text != strings.Repeat("x", glossaryFieldLimit) {
		t.Fatalf("Summary.Text len = %d, want %d", len(got.Terms[0].Summary.Text), glossaryFieldLimit)
	}
	if len(got.Warnings) != 1 {
		t.Fatalf("Warnings = %#v, want 1", got.Warnings)
	}
	for _, want := range []string{"mediation", "docs/a.md", "summary"} {
		if !strings.Contains(got.Warnings[0], want) {
			t.Fatalf("warning %q missing %q", got.Warnings[0], want)
		}
	}
}

func TestGlossaryPublicLibraryIssuesDoNotWarn(t *testing.T) {
	t.Parallel()

	lib := sampleGlossary()
	term := lib["mediation"]
	term.Summary = ParseGlossaryText(strings.Repeat("p", glossaryFieldLimit+50))
	term.Source = &GlossarySource{
		Label: ParseGlossaryText("Exploit"),
		URL:   "javascript:alert(1)",
	}
	lib["mediation"] = term

	got, err := NewMarkdown().Render([]byte("See [广告聚合](term:mediation).\n"), PageRef{Path: "docs/a.md"}, MarkdownOptions{
		Glossary: true,
		Terms:    lib,
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got.Warnings != nil {
		t.Fatalf("Warnings = %#v, want nil for public-library issues", got.Warnings)
	}
	if got.Terms[0].Summary.Text != strings.Repeat("p", glossaryFieldLimit+50) {
		t.Fatal("public summary was truncated in Render")
	}
	if got.Terms[0].Source == nil || got.Terms[0].Source.URL != "javascript:alert(1)" {
		t.Fatalf("public source rewritten: %+v", got.Terms[0].Source)
	}
}

func TestGlossaryExactFieldLimitHasNoWarning(t *testing.T) {
	t.Parallel()

	exact := strings.Repeat("x", glossaryFieldLimit)
	src := fmt.Sprintf(`---
glossary:
  mediation:
    summary: %q
---

See [广告聚合](term:mediation).
`, exact)
	got, err := NewMarkdown().Render([]byte(src), PageRef{Path: "docs/a.md"}, MarkdownOptions{
		Glossary: true,
		Terms:    sampleGlossary(),
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got.Terms[0].Summary.Text != exact {
		t.Fatalf("Summary.Text = %q", got.Terms[0].Summary.Text)
	}
	if got.Warnings != nil {
		t.Fatalf("Warnings = %#v, want nil at exact limit", got.Warnings)
	}
}

func TestGlossaryOffUndefinedWarns(t *testing.T) {
	t.Parallel()

	got, err := NewMarkdown().Render([]byte("See [missing](term:unknown).\n"), PageRef{Path: "docs/a.md"}, MarkdownOptions{
		Glossary:       true,
		GlossaryStrict: GlossaryStrictOff,
		Terms:          sampleGlossary(),
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if len(got.Warnings) != 1 {
		t.Fatalf("Warnings = %#v, want 1", got.Warnings)
	}
	for _, want := range []string{"unknown", "docs/a.md", ":1"} {
		if !strings.Contains(got.Warnings[0], want) {
			t.Fatalf("warning %q missing %q", got.Warnings[0], want)
		}
	}
}

func TestGlossaryCoexistsWithLinkRewrite(t *testing.T) {
	t.Parallel()

	src := []byte("See [广告聚合](term:mediation) and [doc](other.md#part).\n")
	got, err := NewMarkdown().Render(src, PageRef{
		Path: "docs/guide/current.md",
		Resolve: func(target string) string {
			if target == "docs/guide/other.md" {
				return "view"
			}
			return ""
		},
	}, MarkdownOptions{Glossary: true, Terms: sampleGlossary()})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	html := string(got.HTML)
	requireContains(t, html, `data-glossary="mediation"`)
	requireContains(t, html, `href="../other.md/#part"`)
	requireNoTermHref(t, html)
}

func TestGlossaryRenderConcurrent(t *testing.T) {
	t.Parallel()

	md := NewMarkdown()
	lib := sampleGlossary()
	src := []byte(`---
glossary:
  mediation:
    page: override
---

See [广告聚合](term:mediation) and [doc](other.md).
`)
	ref := PageRef{
		Path: "docs/guide/current.md",
		Resolve: func(target string) string {
			if target == "docs/guide/other.md" {
				return "view"
			}
			return ""
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			opts := MarkdownOptions{
				Glossary:       i%2 == 0,
				GlossaryStrict: GlossaryStrictOff,
				Terms:          lib,
				Anchors:        i%3 == 0,
				Mermaid:        i%5 == 0,
			}
			got, err := md.Render(src, ref, opts)
			if err != nil {
				t.Errorf("Render() error = %v", err)
				return
			}
			html := string(got.HTML)
			if strings.Contains(html, `href="term:`) || strings.Contains(html, `href='term:`) {
				t.Errorf("output contains term href:\n%s", html)
			}
			if opts.Glossary {
				if len(got.Terms) != 1 || got.Terms[0].Page.Text != "override" {
					t.Errorf("Terms = %+v", got.Terms)
				}
			} else if got.Terms != nil {
				t.Errorf("disabled Terms = %+v, want nil", got.Terms)
			}
		}(i)
	}
	wg.Wait()

	if lib["mediation"].Page.Text != "" {
		t.Fatal("public glossary mutated under concurrent Render")
	}
	if lib["mediation"].Source == nil || lib["mediation"].Source.URL != "https://old.example/docs" {
		t.Fatalf("public source mutated: %+v", lib["mediation"].Source)
	}
}

func termKeys(terms []GlossaryTerm) []string {
	if len(terms) == 0 {
		return nil
	}
	out := make([]string, len(terms))
	for i, term := range terms {
		out[i] = term.Key
	}
	return out
}

func firstTermText(terms []GlossaryTerm) string {
	if len(terms) == 0 {
		return ""
	}
	return terms[0].Title.Text
}

func requireNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Fatalf("output unexpectedly contains %q\n%s", substr, s)
	}
}

func requireNoTermHref(t *testing.T, s string) {
	t.Helper()
	requireNotContains(t, s, `href="term:`)
	requireNotContains(t, s, `href='term:`)
}
