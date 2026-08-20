package render

import (
	"fmt"
	"html"
	"html/template"
	"net/url"
	"regexp"
	"strings"
)

// GlossaryText is a glossary field in two forms. Text is the source with
// inline-code markers stripped, used for aria-label, search, and llms.txt.
// HTML is the escaped rendering with paired backticks turned into <code>.
type GlossaryText struct {
	Text string
	HTML template.HTML
}

// ParseGlossaryText converts a glossary field into Text and HTML forms.
func ParseGlossaryText(raw string) GlossaryText {
	if raw == "" {
		return GlossaryText{}
	}

	var textBuf, htmlBuf strings.Builder
	i := 0
	for i < len(raw) {
		next := strings.IndexByte(raw[i:], '`')
		if next < 0 {
			chunk := raw[i:]
			textBuf.WriteString(chunk)
			htmlBuf.WriteString(html.EscapeString(chunk))
			break
		}
		next += i
		if next > i {
			chunk := raw[i:next]
			textBuf.WriteString(chunk)
			htmlBuf.WriteString(html.EscapeString(chunk))
		}
		closeAt := strings.IndexByte(raw[next+1:], '`')
		if closeAt < 0 {
			chunk := raw[next:]
			textBuf.WriteString(chunk)
			htmlBuf.WriteString(html.EscapeString(chunk))
			break
		}
		closeAt += next + 1
		if closeAt == next+1 {
			textBuf.WriteString("``")
			htmlBuf.WriteString("``")
			i = closeAt + 1
			continue
		}
		inner := raw[next+1 : closeAt]
		textBuf.WriteString(inner)
		htmlBuf.WriteString("<code>")
		htmlBuf.WriteString(html.EscapeString(inner))
		htmlBuf.WriteString("</code>")
		i = closeAt + 1
	}
	return GlossaryText{Text: textBuf.String(), HTML: template.HTML(htmlBuf.String())}
}

type GlossarySource struct {
	Label GlossaryText
	URL   string
}

type GlossaryTerm struct {
	Key     string
	Title   GlossaryText
	Alias   GlossaryText
	Summary GlossaryText
	Page    GlossaryText
	Warning GlossaryText
	Source  *GlossarySource
}

// IsIncomplete reports whether the term still lacks an explanation after
// front-matter overrides: both Summary and Page are empty. The result is
// meaningful for MarkdownResult.Terms, not for public-library entries.
func (t GlossaryTerm) IsIncomplete() bool {
	return strings.TrimSpace(t.Summary.Text) == "" && strings.TrimSpace(t.Page.Text) == ""
}

type GlossaryStrictness string

const (
	GlossaryStrictOff      GlossaryStrictness = "off"
	GlossaryStrictRefs     GlossaryStrictness = "refs"
	GlossaryStrictComplete GlossaryStrictness = "complete"
)

func (s GlossaryStrictness) effective() GlossaryStrictness {
	switch s {
	case GlossaryStrictOff, GlossaryStrictRefs, GlossaryStrictComplete:
		return s
	default:
		return GlossaryStrictRefs
	}
}

// Glossary is the build-time public term library, indexed by normalized key.
// It is read-only at render time and safe for concurrent use.
type Glossary map[string]GlossaryTerm

var glossaryKeyPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func validGlossaryKey(key string) bool {
	return glossaryKeyPattern.MatchString(key)
}

func cloneGlossaryTerm(t GlossaryTerm) GlossaryTerm {
	if t.Source != nil {
		src := *t.Source
		t.Source = &src
	}
	return t
}

func mergePageGlossary(public Glossary, meta map[string]any) Glossary {
	out := make(Glossary, len(public))
	for k, t := range public {
		key := strings.ToLower(k)
		cloned := cloneGlossaryTerm(t)
		if cloned.Key == "" {
			cloned.Key = key
		}
		out[key] = cloned
	}

	raw, ok := meta["glossary"]
	if !ok || raw == nil {
		return out
	}
	entries := asStringMap(raw)
	if entries == nil {
		return out
	}
	for rawKey, rawVal := range entries {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		if !validGlossaryKey(key) {
			continue
		}
		fields := asStringMap(rawVal)
		if fields == nil {
			continue
		}
		term, exists := out[key]
		if !exists {
			term = GlossaryTerm{Key: key}
		}
		out[key] = applyGlossaryOverrides(term, fields, key)
	}
	return out
}

func applyGlossaryOverrides(term GlossaryTerm, fields map[string]any, key string) GlossaryTerm {
	if s, ok := nonEmptyString(fields["title"]); ok {
		term.Title = ParseGlossaryText(s)
	}
	if s, ok := nonEmptyString(fields["alias"]); ok {
		term.Alias = ParseGlossaryText(s)
	}
	if s, ok := nonEmptyString(fields["summary"]); ok {
		term.Summary = ParseGlossaryText(s)
	}
	if s, ok := nonEmptyString(fields["page"]); ok {
		term.Page = ParseGlossaryText(s)
	}
	if s, ok := nonEmptyString(fields["warning"]); ok {
		term.Warning = ParseGlossaryText(s)
	}
	if raw, ok := fields["source"]; ok && isNonEmptySource(raw) {
		term.Source = parseGlossarySource(raw)
	}
	term.Key = key
	return term
}

func parseGlossarySource(v any) *GlossarySource {
	m := asStringMap(v)
	if m == nil {
		return nil
	}
	label, _ := asString(m["label"])
	rawURL, _ := asString(m["url"])
	rawURL = strings.TrimSpace(rawURL)
	if !httpOrHTTPS(rawURL) {
		return nil
	}
	return &GlossarySource{
		Label: ParseGlossaryText(label),
		URL:   rawURL,
	}
}

func isNonEmptySource(v any) bool {
	m := asStringMap(v)
	if m == nil {
		return false
	}
	label, _ := asString(m["label"])
	rawURL, _ := asString(m["url"])
	return strings.TrimSpace(label) != "" || strings.TrimSpace(rawURL) != ""
}

func httpOrHTTPS(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}

func asStringMap(v any) map[string]any {
	switch v := v.(type) {
	case map[string]any:
		return v
	case map[any]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			ks, ok := k.(string)
			if !ok {
				continue
			}
			out[ks] = val
		}
		return out
	default:
		return nil
	}
}

func asString(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func nonEmptyString(v any) (string, bool) {
	s, ok := asString(v)
	if !ok || strings.TrimSpace(s) == "" {
		return "", false
	}
	return s, true
}

type undefinedGlossaryRefError struct {
	Path string
	Line int
	Key  string
}

func (e undefinedGlossaryRefError) Error() string {
	return fmt.Sprintf("undefined glossary term %q at %s:%d", e.Key, e.Path, e.Line)
}
