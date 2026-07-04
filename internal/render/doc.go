// Package render implements the build-time rendering pipeline: goldmark for
// Markdown (GFM, TOC, anchors, front matter, mermaid) and chroma for syntax
// highlighting. Relative links pointing at in-repo files are rewritten to
// /view/ paths at the AST level via an ASTTransformer — never by text
// replacement on rendered output. User HTML files are never parsed or
// modified (ADR-001).
package render
