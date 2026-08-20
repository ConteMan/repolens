package render

import (
	"fmt"
	"html"
	"html/template"
	"net/url"
	"reflect"
	"regexp"
	"sort"
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
	// DefinedIn is the repo-relative path of the public glossary file.
	// Private front-matter terms leave it empty.
	DefinedIn string
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

// ValidGlossaryKey reports whether key matches the glossary key pattern
// ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$.
func ValidGlossaryKey(key string) bool {
	return glossaryKeyPattern.MatchString(key)
}

func formatTermAriaLabel(format, display string) string {
	if strings.TrimSpace(format) == "" {
		return ""
	}
	return strings.Replace(format, "%s", display, 1)
}

func cloneGlossaryTerm(t GlossaryTerm) GlossaryTerm {
	if t.Source != nil {
		src := *t.Source
		t.Source = &src
	}
	return t
}

const glossaryFieldLimit = 2000

func mergePageGlossary(public Glossary, meta map[string]any, path string) (Glossary, []string) {
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
		return out, nil
	}
	entries := asStringMap(raw)
	if entries == nil {
		return out, nil
	}
	keys := make([]string, 0, len(entries))
	for rawKey := range entries {
		keys = append(keys, rawKey)
	}
	sort.Strings(keys)

	var warnings []string
	for _, rawKey := range keys {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		if !ValidGlossaryKey(key) {
			continue
		}
		fields := asStringMap(entries[rawKey])
		if fields == nil {
			continue
		}
		term, exists := out[key]
		if !exists {
			term = GlossaryTerm{Key: key}
		}
		var fieldWarnings []string
		term, fieldWarnings = applyGlossaryOverrides(term, fields, path, key)
		out[key] = term
		warnings = append(warnings, fieldWarnings...)
	}
	return out, warnings
}

// glossaryFrontMatter is the YAML keys applyGlossaryOverrides honors.
// The skill consistency test compares this set to SKILL.md.
type glossaryFrontMatter struct {
	Title   string `yaml:"title"`
	Alias   string `yaml:"alias"`
	Summary string `yaml:"summary"`
	Page    string `yaml:"page"`
	Warning string `yaml:"warning"`
	Source  any    `yaml:"source"`
}

// GlossaryFrontMatterFields returns the YAML keys honored by document
// front-matter glossary overlays.
func GlossaryFrontMatterFields() []string {
	return yamlFieldNames(glossaryFrontMatter{})
}

func yamlFieldNames(v any) []string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	names := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("yaml")
		name, _, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func applyGlossaryOverrides(term GlossaryTerm, fields map[string]any, path, key string) (GlossaryTerm, []string) {
	var warnings []string
	for _, name := range GlossaryFrontMatterFields() {
		switch name {
		case "source":
			raw, ok := fields[name]
			if !ok || !isNonEmptySource(raw) {
				continue
			}
			src, srcWarnings := parseGlossarySource(raw, path, key)
			term.Source = src
			warnings = append(warnings, srcWarnings...)
		default:
			s, ok := nonEmptyString(fields[name])
			if !ok {
				continue
			}
			s, w := cutGlossaryField(s, path, key, name)
			warnings = append(warnings, w...)
			text := ParseGlossaryText(s)
			switch name {
			case "title":
				term.Title = text
			case "alias":
				term.Alias = text
			case "summary":
				term.Summary = text
			case "page":
				term.Page = text
			case "warning":
				term.Warning = text
			}
		}
	}
	term.Key = key
	return term, warnings
}

func parseGlossarySource(v any, path, key string) (*GlossarySource, []string) {
	m := asStringMap(v)
	if m == nil {
		return nil, nil
	}
	label, _ := asString(m["label"])
	rawURL, _ := asString(m["url"])

	var warnings []string
	var w []string
	label, w = cutGlossaryField(label, path, key, "source.label")
	warnings = append(warnings, w...)
	rawURL, w = cutGlossaryField(rawURL, path, key, "source.url")
	warnings = append(warnings, w...)
	rawURL = strings.TrimSpace(rawURL)
	if !httpOrHTTPS(rawURL) {
		warnings = append(warnings, glossaryTermWarning(path, key, fmt.Sprintf("invalid source URL %q (only http/https allowed); source ignored", rawURL)))
		return nil, warnings
	}
	return &GlossarySource{
		Label: ParseGlossaryText(label),
		URL:   rawURL,
	}, warnings
}

func cutGlossaryField(s, path, key, field string) (string, []string) {
	s, truncated := truncateGlossaryField(s)
	if !truncated {
		return s, nil
	}
	return s, []string{glossaryTermWarning(path, key, fmt.Sprintf("field %q truncated to %d characters", field, glossaryFieldLimit))}
}

func truncateGlossaryField(s string) (string, bool) {
	n := 0
	for i := range s {
		if n == glossaryFieldLimit {
			return s[:i], true
		}
		n++
	}
	return s, false
}

func glossaryTermWarning(path, key, msg string) string {
	return fmt.Sprintf("glossary term %q at %s: %s", key, path, msg)
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
