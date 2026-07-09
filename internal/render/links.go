package render

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type linkTransformer struct{}

func (linkTransformer) Transform(doc *ast.Document, _ text.Reader, pc parser.Context) {
	ref, ok := pc.Get(pageRefContextKey).(PageRef)
	if !ok || ref.Resolve == nil {
		return
	}

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := n.(type) {
		case *ast.Link:
			n.Destination = rewriteDestination(n.Destination, ref, false)
			return ast.WalkSkipChildren, nil
		case *ast.Image:
			n.Destination = rewriteDestination(n.Destination, ref, true)
			return ast.WalkSkipChildren, nil
		default:
			return ast.WalkContinue, nil
		}
	})
}

func rewriteDestination(dest []byte, ref PageRef, image bool) []byte {
	raw := string(dest)
	rewritten := rewriteLink(raw, ref, image)
	if rewritten == raw {
		return dest
	}
	return []byte(rewritten)
}

func rewriteLink(raw string, ref PageRef, image bool) string {
	if shouldSkipLink(raw) {
		return raw
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Path == "" {
		return raw
	}

	targetPath := parsed.Path
	if decoded, err := url.PathUnescape(targetPath); err == nil {
		targetPath = decoded
	}

	target := joinRepoPath(path.Dir(cleanRepoPath(ref.Path)), targetPath)
	if target == "" || escapesRepo(target) {
		return raw
	}

	mode := ref.Resolve(target)
	if mode == "" {
		return raw
	}

	var targetURL string
	if image || mode == "mirror" {
		targetURL = mirrorURL(target)
	} else if mode == "view" {
		// 可渲染的 index.html 没有独立浏览页——它按 Web 惯例并入目录页
		// （Issue #9），链接须落到父目录的 view URL。
		if path.Base(target) == "index.html" {
			targetURL = viewURL(path.Dir(target))
		} else {
			targetURL = viewURL(target)
		}
	} else {
		return raw
	}

	rewritten := RelTo(viewURL(ref.Path), targetURL)
	if parsed.RawQuery != "" {
		rewritten += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		rewritten += "#" + parsed.Fragment
	}
	return rewritten
}

func shouldSkipLink(raw string) bool {
	if raw == "" || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, "/") {
		return true
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.Scheme != "" {
		return true
	}
	return false
}

func joinRepoPath(baseDir, rel string) string {
	if baseDir == "." || baseDir == "/" {
		baseDir = ""
	}
	return cleanRepoPath(path.Join(baseDir, rel))
}

func escapesRepo(repoPath string) bool {
	return repoPath == ".." || strings.HasPrefix(repoPath, "../")
}

func cleanRepoPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return ""
	}
	cleaned := path.Clean(p)
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func viewURL(repoPath string) string {
	repoPath = cleanRepoPath(repoPath)
	if repoPath == "" {
		return "view/"
	}
	return "view/" + repoPath + "/"
}

func mirrorURL(repoPath string) string {
	return cleanRepoPath(repoPath)
}

// RelTo computes the relative URL from one site page URL to another. Both
// arguments are root-relative URLs like "view/docs/a.md/"; a trailing slash
// on toURL is preserved. Shared by the markdown link rewriter and the site
// assembly layer (spec 005).
func RelTo(fromURL, toURL string) string {
	from := cleanURLPath(fromURL)
	to, trailingSlash := cleanTargetURLPath(toURL)
	if from == "" {
		from = "."
	}
	if to == "" {
		to = "."
	}

	rel, err := filepath.Rel(from, to)
	if err != nil {
		return toURL
	}
	rel = filepath.ToSlash(rel)
	if trailingSlash && rel != "." && !strings.HasSuffix(rel, "/") {
		rel += "/"
	}
	return rel
}

func cleanURLPath(p string) string {
	p = strings.TrimPrefix(strings.ReplaceAll(p, "\\", "/"), "/")
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		return ""
	}
	cleaned := path.Clean(p)
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func cleanTargetURLPath(p string) (string, bool) {
	trailingSlash := strings.HasSuffix(p, "/")
	cleaned := cleanURLPath(p)
	return cleaned, trailingSlash
}
