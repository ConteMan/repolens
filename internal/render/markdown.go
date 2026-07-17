package render

import (
	"bytes"
	"html/template"
	"path"
	"strings"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	goldhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/anchor"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/mermaid"
	"go.abhg.dev/goldmark/toc"
)

const defaultTOCMinHeadings = 3

var pageRefContextKey = parser.NewContextKey()

// MarkdownOptions controls Markdown rendering without coupling render to
// internal/config.
type MarkdownOptions struct {
	TOC              bool
	TOCMinHeadings   int
	Anchors          bool
	Mermaid          bool
	FrontmatterTitle bool
}

// PageRef identifies the current source file and resolves in-repository links.
type PageRef struct {
	Path string

	// Resolve returns "view" for files with a browsing page, "mirror" for
	// files that should link to the raw mirror layer, or "" for unknown paths.
	Resolve func(target string) string
}

// TOCItem is the template-friendly table of contents structure returned by
// Render.
type TOCItem struct {
	Title    string
	Anchor   string
	Level    int
	Children []TOCItem
}

// MarkdownResult is the rendered Markdown body plus metadata collected while
// parsing.
type MarkdownResult struct {
	HTML       template.HTML
	Title      string
	TOC        []TOCItem
	HasMermaid bool
	Meta       map[string]any
}

// Markdown renders Markdown documents. It is safe to reuse concurrently.
type Markdown struct {
	variants [2][2]goldmark.Markdown // indexed [anchors][mermaid], 0=off 1=on
}

// NewMarkdown assembles the reusable Markdown rendering pipelines.
func NewMarkdown() *Markdown {
	m := &Markdown{}
	for anchors := range 2 {
		for mermaids := range 2 {
			m.variants[anchors][mermaids] = newGoldmark(anchors == 1, mermaids == 1)
		}
	}
	return m
}

// Render converts a Markdown document into HTML and extracts title, TOC, and
// front matter metadata.
func (m *Markdown) Render(src []byte, ref PageRef, opts MarkdownOptions) (MarkdownResult, error) {
	md := m.variant(opts)
	ctx := parser.NewContext()
	ctx.Set(pageRefContextKey, ref)

	docNode := md.Parser().Parse(text.NewReader(src), parser.WithContext(ctx))
	doc := docNode.(*ast.Document)

	meta := copyMeta(doc.OwnerDocument().Meta())
	title := pageTitle(src, doc, ref.Path, meta, opts.FrontmatterTitle)
	hasMermaid := opts.Mermaid && containsMermaid(doc)

	var resultTOC []TOCItem
	if opts.TOC {
		minHeadings := opts.TOCMinHeadings
		if minHeadings <= 0 {
			minHeadings = defaultTOCMinHeadings
		}
		if countHeadings(doc) >= minHeadings {
			tree, err := toc.Inspect(doc, src)
			if err != nil {
				return MarkdownResult{}, err
			}
			resultTOC = convertTOC(tree.Items, 1)
		}
	}

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, src, doc); err != nil {
		return MarkdownResult{}, err
	}

	return MarkdownResult{
		HTML:       template.HTML(buf.String()),
		Title:      title,
		TOC:        resultTOC,
		HasMermaid: hasMermaid,
		Meta:       meta,
	}, nil
}

func (m *Markdown) variant(opts MarkdownOptions) goldmark.Markdown {
	anchorIndex := 0
	if opts.Anchors {
		anchorIndex = 1
	}
	mermaidIndex := 0
	if opts.Mermaid {
		mermaidIndex = 1
	}
	return m.variants[anchorIndex][mermaidIndex]
}

func newGoldmark(withAnchors, withMermaid bool) goldmark.Markdown {
	extensions := []goldmark.Extender{
		extension.GFM,
		&frontmatter.Extender{Mode: frontmatter.SetMetadata},
		highlighting.NewHighlighting(
			highlighting.WithStyle(sharedHighlightTheme()),
			highlighting.WithFormatOptions(sharedHighlightOptions()...),
		),
	}
	if withAnchors {
		extensions = append(extensions, &anchor.Extender{})
	}
	if withMermaid {
		extensions = append(extensions, &mermaid.Extender{
			RenderMode: mermaid.RenderModeClient,
			NoScript:   true,
		})
	}

	return goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(util.Prioritized(linkTransformer{}, 500)),
		),
		goldmark.WithRendererOptions(
			goldhtml.WithUnsafe(),
			renderer.WithNodeRenderers(
				// GFM's table renderer uses priority 500. A lower priority is
				// registered later and only overrides KindTable.
				util.Prioritized(markdownTableRenderer{}, 400),
			),
		),
	)
}

func copyMeta(meta map[string]any) map[string]any {
	if len(meta) == 0 {
		return nil
	}
	out := make(map[string]any, len(meta))
	for k, v := range meta {
		out[k] = v
	}
	return out
}

func pageTitle(src []byte, doc *ast.Document, repoPath string, meta map[string]any, frontmatterTitle bool) string {
	if frontmatterTitle {
		if title, ok := metaString(meta["title"]); ok {
			return title
		}
	}
	if title := firstH1(src, doc); title != "" {
		return title
	}
	if base := path.Base(cleanRepoPath(repoPath)); base != "." && base != "/" {
		return base
	}
	return "Untitled"
}

func metaString(v any) (string, bool) {
	switch v := v.(type) {
	case string:
		v = strings.TrimSpace(v)
		return v, v != ""
	default:
		return "", false
	}
}

func firstH1(src []byte, doc *ast.Document) string {
	var title string
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || title != "" {
			return ast.WalkContinue, nil
		}
		heading, ok := n.(*ast.Heading)
		if !ok || heading.Level != 1 {
			return ast.WalkContinue, nil
		}
		title = nodeText(src, heading)
		return ast.WalkStop, nil
	})
	return title
}

func nodeText(src []byte, n ast.Node) string {
	var buf bytes.Buffer
	_ = ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() == anchor.Kind {
			return ast.WalkSkipChildren, nil
		}
		switch n := n.(type) {
		case *ast.Text:
			buf.Write(n.Segment.Value(src))
			if n.SoftLineBreak() || n.HardLineBreak() {
				buf.WriteByte(' ')
			}
		case *ast.String:
			buf.Write(n.Value)
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(buf.String())
}

func countHeadings(doc *ast.Document) int {
	var count int
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if _, ok := n.(*ast.Heading); ok {
				count++
				return ast.WalkSkipChildren, nil
			}
		}
		return ast.WalkContinue, nil
	})
	return count
}

func containsMermaid(doc *ast.Document) bool {
	var found bool
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Kind() == mermaid.Kind {
			found = true
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return found
}

func convertTOC(items toc.Items, level int) []TOCItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]TOCItem, 0, len(items))
	for _, item := range items {
		children := convertTOC(item.Items, level+1)
		if len(item.Title) == 0 {
			out = append(out, children...)
			continue
		}
		out = append(out, TOCItem{
			Title:    string(item.Title),
			Anchor:   string(item.ID),
			Level:    level,
			Children: children,
		})
	}
	return out
}
