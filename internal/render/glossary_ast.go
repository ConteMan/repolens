package render

import (
	"bytes"
	"errors"
	"net/url"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var kindGlossaryLink = ast.NewNodeKind("GlossaryLink")

// glossaryLink is an inline term annotation rendered as a page-local anchor.
type glossaryLink struct {
	ast.BaseInline
	key       string
	ariaLabel string
}

func (n *glossaryLink) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{
		"Key":       n.key,
		"AriaLabel": n.ariaLabel,
	}, nil)
}

func (*glossaryLink) Kind() ast.NodeKind { return kindGlossaryLink }

type glossaryLinkRenderer struct{}

func (glossaryLinkRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindGlossaryLink, renderGlossaryLink)
}

func renderGlossaryLink(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*glossaryLink)
	if entering {
		_, _ = w.WriteString(`<a class="term" href="#glossary-`)
		_, _ = w.Write(util.EscapeHTML([]byte(n.key)))
		_, _ = w.WriteString(`" data-glossary="`)
		_, _ = w.Write(util.EscapeHTML([]byte(n.key)))
		_, _ = w.WriteString(`" aria-label="`)
		_, _ = w.Write(util.EscapeHTML([]byte(n.ariaLabel)))
		_, _ = w.WriteString(`">`)
		return ast.WalkContinue, nil
	}
	_, _ = w.WriteString(`</a>`)
	return ast.WalkContinue, nil
}

type termHit struct {
	link  *ast.Link
	dest  string
	key   string
	valid bool
	line  int
}

func (h termHit) reportKey() string {
	if h.key != "" {
		return h.key
	}
	return h.dest
}

func applyGlossary(doc *ast.Document, src []byte, ref PageRef, opts MarkdownOptions) ([]GlossaryTerm, []string, error) {
	return applyGlossaryWithWalk(doc, src, ref, opts, collectTermHits)
}

func applyGlossaryWithWalk(doc *ast.Document, src []byte, ref PageRef, opts MarkdownOptions, walk func(*ast.Document, []byte) []termHit) ([]GlossaryTerm, []string, error) {
	if !containsTermScheme(src) {
		return nil, nil, nil
	}

	hits := walk(doc, src)
	if len(hits) == 0 {
		return nil, nil, nil
	}

	if !opts.Glossary {
		for _, hit := range hits {
			unwrapInline(hit.link)
		}
		return nil, nil, nil
	}

	pageTerms, warnings := mergePageGlossary(opts.Terms, doc.Meta(), ref.Path)
	failUndefined := opts.GlossaryStrict.effective() != GlossaryStrictOff

	type resolved struct {
		hit  termHit
		term GlossaryTerm
		ok   bool
	}
	resolvedHits := make([]resolved, 0, len(hits))
	var undef []error
	for _, hit := range hits {
		term, ok := lookupPageTerm(pageTerms, hit)
		if !ok {
			if failUndefined {
				undef = append(undef, undefinedGlossaryRefError{
					Path: ref.Path,
					Line: hit.line,
					Key:  hit.reportKey(),
				})
			} else {
				warnings = append(warnings, undefinedGlossaryRefError{
					Path: ref.Path,
					Line: hit.line,
					Key:  hit.reportKey(),
				}.Error())
			}
			resolvedHits = append(resolvedHits, resolved{hit: hit})
			continue
		}
		resolvedHits = append(resolvedHits, resolved{hit: hit, term: term, ok: true})
	}
	if len(undef) > 0 {
		return nil, warnings, errors.Join(undef...)
	}

	seen := make(map[string]struct{}, len(resolvedHits))
	var terms []GlossaryTerm
	for _, item := range resolvedHits {
		if !item.ok {
			unwrapInline(item.hit.link)
			continue
		}
		replaceLinkWithTerm(item.hit.link, item.term.Key, item.term.Title.Text)
		if _, dup := seen[item.term.Key]; dup {
			continue
		}
		seen[item.term.Key] = struct{}{}
		terms = append(terms, item.term)
	}
	return terms, warnings, nil
}

// containsTermScheme is a cheap, case-insensitive substring check for the
// term: annotation scheme. A hit is not proof of a real *ast.Link; callers
// still walk the AST. A miss skips all glossary processing.
func containsTermScheme(src []byte) bool {
	const n = len("term:")
	if len(src) < n {
		return false
	}
	needle := []byte("term:")
	for i := 0; i <= len(src)-n; i++ {
		if bytes.EqualFold(src[i:i+n], needle) {
			return true
		}
	}
	return false
}

func lookupPageTerm(page Glossary, hit termHit) (GlossaryTerm, bool) {
	if !hit.valid {
		return GlossaryTerm{}, false
	}
	term, ok := page[hit.key]
	if !ok || strings.TrimSpace(term.Title.Text) == "" {
		return GlossaryTerm{}, false
	}
	term.Key = hit.key
	return term, true
}

func collectTermHits(doc *ast.Document, src []byte) []termHit {
	var hits []termHit
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Image:
			return ast.WalkSkipChildren, nil
		case *ast.Link:
			dest := string(n.Destination)
			if !isTermDestination(dest) {
				return ast.WalkSkipChildren, nil
			}
			key, valid := parseTermDestination(dest)
			hits = append(hits, termHit{
				link:  n,
				dest:  dest,
				key:   key,
				valid: valid,
				line:  nodeLine(src, n),
			})
			return ast.WalkSkipChildren, nil
		default:
			return ast.WalkContinue, nil
		}
	})
	return hits
}

func isTermDestination(dest string) bool {
	u, err := url.Parse(dest)
	return err == nil && strings.EqualFold(u.Scheme, "term")
}

func parseTermDestination(dest string) (string, bool) {
	u, err := url.Parse(dest)
	if err != nil || !strings.EqualFold(u.Scheme, "term") {
		return "", false
	}
	raw := u.Opaque
	if raw == "" {
		raw = strings.TrimPrefix(u.Path, "/")
	}
	key := strings.ToLower(raw)
	if u.RawQuery != "" || u.Fragment != "" {
		return key, false
	}
	if !ValidGlossaryKey(key) {
		return key, false
	}
	return key, true
}

func nodeLine(src []byte, n ast.Node) int {
	offset := -1
	_ = ast.Walk(n, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := child.(*ast.Text); ok {
			offset = t.Segment.Start
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	if offset < 0 {
		for p := n; p != nil; p = p.Parent() {
			if p.Type() == ast.TypeBlock && p.Lines() != nil && p.Lines().Len() > 0 {
				offset = p.Lines().At(0).Start
				break
			}
		}
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(src) {
		offset = len(src)
	}
	return bytes.Count(src[:offset], []byte("\n")) + 1
}

func unwrapInline(n ast.Node) {
	parent := n.Parent()
	if parent == nil {
		return
	}
	for c := n.FirstChild(); c != nil; {
		next := c.NextSibling()
		n.RemoveChild(n, c)
		parent.InsertBefore(parent, n, c)
		c = next
	}
	parent.RemoveChild(parent, n)
}

func replaceLinkWithTerm(link *ast.Link, key, ariaLabel string) {
	parent := link.Parent()
	if parent == nil {
		return
	}
	n := &glossaryLink{key: key, ariaLabel: ariaLabel}
	for c := link.FirstChild(); c != nil; {
		next := c.NextSibling()
		link.RemoveChild(link, c)
		n.AppendChild(n, c)
		c = next
	}
	parent.ReplaceChild(parent, link, n)
}
