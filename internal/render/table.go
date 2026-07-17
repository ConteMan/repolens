package render

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extensionast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	goldhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

// markdownTableRenderer adds a keyboard-accessible overflow boundary around
// GFM tables. Header, row, and cell rendering remains owned by goldmark.
type markdownTableRenderer struct{}

func (markdownTableRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(extensionast.KindTable, renderMarkdownTable)
}

func renderMarkdownTable(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(`<div class="table-scroll" tabindex="0">`)
		_, _ = w.WriteString("\n<table")
		if node.Attributes() != nil {
			goldhtml.RenderAttributes(w, node, extension.TableAttributeFilter)
		}
		_, _ = w.WriteString(">\n")
	} else {
		_, _ = w.WriteString("</table>\n</div>\n")
	}
	return ast.WalkContinue, nil
}
