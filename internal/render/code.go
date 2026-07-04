package render

import (
	"bytes"
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

const (
	defaultChromaTheme = "github"
	lineAnchorPrefix   = "L"
	sniffLimit         = 512
)

// Kind is the browser-layer rendering category for a source file.
type Kind int

const (
	KindMarkdown Kind = iota
	KindHTML
	KindCode
	KindImage
	KindBinary
)

// Classify returns the rendering category for a repo-relative file path.
// maxFileSize <= 0 disables the size limit. A nil or empty sniff is treated
// as an empty (text) file.
func Classify(path string, size int64, sniff []byte, maxFileSize int64) Kind {
	if maxFileSize > 0 && size > maxFileSize {
		return KindBinary
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return KindMarkdown
	case ".html", ".htm":
		return KindHTML
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".avif", ".ico":
		return KindImage
	}

	if lexers.Match(path) != nil || isText(sniff) {
		return KindCode
	}
	return KindBinary
}

// CodeOptions controls code rendering without coupling render to internal/config.
type CodeOptions struct {
	LineNumbers bool
	Theme       string
}

// CodeResult is a rendered code block plus metadata for templates.
type CodeResult struct {
	HTML  template.HTML
	Lang  string
	Lines int
}

// Code renders a source file as class-based Chroma HTML.
func Code(src []byte, filename string, opts CodeOptions) (CodeResult, error) {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, string(src))
	if err != nil {
		return CodeResult{}, err
	}

	var buf bytes.Buffer
	if err := codeHTMLFormatter(opts.LineNumbers).Format(&buf, chromaStyle(opts.Theme), iterator); err != nil {
		return CodeResult{}, err
	}

	return CodeResult{
		HTML:  template.HTML(buf.String()),
		Lang:  lexerLang(lexer),
		Lines: countSourceLines(src),
	}, nil
}

// StylesCSS returns the CSS rules for Chroma class-based HTML.
func StylesCSS(theme string) (string, error) {
	style, ok := chromaStyleByName(theme)
	if !ok {
		return "", fmt.Errorf("unknown chroma theme %q", theme)
	}

	var buf bytes.Buffer
	if err := codeHTMLFormatter(true).WriteCSS(&buf, style); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// sharedHighlightTheme and sharedHighlightOptions are the single source of
// chroma configuration for both Markdown code blocks (spec 003) and
// standalone code files (spec 004).
func sharedHighlightTheme() string {
	return defaultChromaTheme
}

func sharedHighlightOptions() []chromahtml.Option {
	return []chromahtml.Option{chromahtml.WithClasses(true)}
}

func codeHTMLFormatter(lineNumbers bool) *chromahtml.Formatter {
	opts := append([]chromahtml.Option{}, sharedHighlightOptions()...)
	opts = append(opts,
		chromahtml.WithLineNumbers(lineNumbers),
		chromahtml.WithLinkableLineNumbers(lineNumbers, lineAnchorPrefix),
	)
	return chromahtml.New(opts...)
}

func chromaStyle(theme string) *chroma.Style {
	if style, ok := chromaStyleByName(theme); ok {
		return style
	}
	return styles.Get(defaultChromaTheme)
}

func chromaStyleByName(theme string) (*chroma.Style, bool) {
	name := strings.ToLower(chromaTheme(theme))
	style, ok := styles.Registry[name]
	return style, ok
}

func chromaTheme(theme string) string {
	theme = strings.TrimSpace(theme)
	if theme == "" {
		return defaultChromaTheme
	}
	return theme
}

func lexerLang(lexer chroma.Lexer) string {
	config := lexer.Config()
	if config.Name == lexers.Fallback.Config().Name {
		return "text"
	}
	if len(config.Aliases) > 0 {
		return strings.ToLower(config.Aliases[0])
	}
	return strings.ToLower(config.Name)
}

func countSourceLines(src []byte) int {
	if len(src) == 0 {
		return 0
	}
	lines := bytes.Count(src, []byte{'\n'})
	if src[len(src)-1] != '\n' {
		lines++
	}
	return lines
}

func isText(sniff []byte) bool {
	if len(sniff) > sniffLimit {
		sniff = sniff[:sniffLimit]
	}
	if bytes.IndexByte(sniff, 0) >= 0 {
		return false
	}
	if len(sniff) == 0 {
		return true
	}

	contentType := http.DetectContentType(sniff)
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil {
		if strings.HasPrefix(mediaType, "text/") {
			return true
		}
		switch mediaType {
		case "application/json", "application/xml", "application/x-ndjson":
			return true
		case "application/octet-stream":
			return false
		}
	}

	return utf8.Valid(sniff)
}
