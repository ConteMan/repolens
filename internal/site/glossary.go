package site

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/ConteMan/repolens/internal/render"
	"github.com/ConteMan/repolens/internal/theme"
	"github.com/goccy/go-yaml"
)

const glossaryFieldLimit = 2000

type glossaryFile struct {
	Title   string          `yaml:"title"`
	Alias   string          `yaml:"alias"`
	Summary string          `yaml:"summary"`
	Page    string          `yaml:"page"`
	Warning string          `yaml:"warning"`
	Source  *glossarySource `yaml:"source"`
}

type glossarySource struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

// LoadGlossary reads the public glossary library from dir under root.
// A missing directory is an empty library. warnings are recoverable
// (invalid filenames, invalid URLs, truncated fields); err is for
// key conflicts and YAML parse failures.
func LoadGlossary(root, dir string) (render.Glossary, []string, error) {
	abs := filepath.Join(root, filepath.FromSlash(dir))
	entries, err := os.ReadDir(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return render.Glossary{}, nil, nil
		}
		return nil, nil, err
	}

	type namedFile struct {
		name string
		key  string
	}
	byKey := make(map[string][]namedFile)
	var warnings []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		base := strings.TrimSuffix(name, filepath.Ext(name))
		if !render.ValidGlossaryKey(base) {
			warnings = append(warnings, fmt.Sprintf("glossary file %q skipped: name is not a valid term key", path.Join(strings.TrimSuffix(dir, "/"), name)))
			continue
		}
		byKey[base] = append(byKey[base], namedFile{name: name, key: base})
	}

	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make(render.Glossary, len(byKey))
	repoDir := strings.TrimSuffix(strings.ReplaceAll(dir, "\\", "/"), "/")
	for _, key := range keys {
		files := byKey[key]
		if len(files) > 1 {
			names := make([]string, len(files))
			for i, f := range files {
				names[i] = f.name
			}
			sort.Strings(names)
			return nil, warnings, fmt.Errorf("glossary key %q has both %s", key, strings.Join(names, " and "))
		}
		file := files[0]
		rel := path.Join(repoDir, file.name)
		data, err := os.ReadFile(filepath.Join(abs, file.name))
		if err != nil {
			return nil, warnings, err
		}
		var raw glossaryFile
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, warnings, fmt.Errorf("parse glossary %s: %w", rel, err)
		}
		term, fieldWarnings := termFromFile(key, rel, raw)
		out[key] = term
		warnings = append(warnings, fieldWarnings...)
	}
	return out, warnings, nil
}

func termFromFile(key, rel string, raw glossaryFile) (render.GlossaryTerm, []string) {
	var warnings []string
	title, w := cutLoadedField(raw.Title, rel, key, "title")
	warnings = append(warnings, w...)
	alias, w := cutLoadedField(raw.Alias, rel, key, "alias")
	warnings = append(warnings, w...)
	summary, w := cutLoadedField(raw.Summary, rel, key, "summary")
	warnings = append(warnings, w...)
	page, w := cutLoadedField(raw.Page, rel, key, "page")
	warnings = append(warnings, w...)
	warning, w := cutLoadedField(raw.Warning, rel, key, "warning")
	warnings = append(warnings, w...)

	term := render.GlossaryTerm{
		Key:       key,
		Title:     render.ParseGlossaryText(title),
		Alias:     render.ParseGlossaryText(alias),
		Summary:   render.ParseGlossaryText(summary),
		Page:      render.ParseGlossaryText(page),
		Warning:   render.ParseGlossaryText(warning),
		DefinedIn: rel,
	}
	if raw.Source != nil && (strings.TrimSpace(raw.Source.Label) != "" || strings.TrimSpace(raw.Source.URL) != "") {
		src, srcWarnings := sourceFromFile(rel, key, *raw.Source)
		term.Source = src
		warnings = append(warnings, srcWarnings...)
	}
	return term, warnings
}

func sourceFromFile(rel, key string, raw glossarySource) (*render.GlossarySource, []string) {
	var warnings []string
	label, w := cutLoadedField(raw.Label, rel, key, "source.label")
	warnings = append(warnings, w...)
	rawURL, w := cutLoadedField(raw.URL, rel, key, "source.url")
	warnings = append(warnings, w...)
	rawURL = strings.TrimSpace(rawURL)
	if !httpOrHTTPS(rawURL) {
		warnings = append(warnings, fmt.Sprintf("glossary term %q at %s: invalid source URL %q (only http/https allowed); source ignored", key, rel, rawURL))
		return nil, warnings
	}
	return &render.GlossarySource{
		Label: render.ParseGlossaryText(label),
		URL:   rawURL,
	}, warnings
}

func cutLoadedField(s, rel, key, field string) (string, []string) {
	s, truncated := truncateGlossaryField(s)
	if !truncated {
		return s, nil
	}
	return s, []string{fmt.Sprintf("glossary term %q at %s: field %q truncated to %d characters", key, rel, field, glossaryFieldLimit)}
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

func glossaryStrictness(cfg *config.Config) render.GlossaryStrictness {
	if cfg == nil {
		return render.GlossaryStrictRefs
	}
	switch cfg.Glossary.Strict {
	case "off":
		return render.GlossaryStrictOff
	case "complete":
		return render.GlossaryStrictComplete
	default:
		return render.GlossaryStrictRefs
	}
}

func glossaryEnabledFor(cfg *config.Config, repoPath string) bool {
	return cfg.OptionsFor(repoPath).Markdown.Glossary
}

func anyMarkdownUsesGlossary(cfg *config.Config, files []fileEntry) bool {
	for _, file := range files {
		if file.Kind == render.KindMarkdown && glossaryEnabledFor(cfg, file.Path) {
			return true
		}
	}
	return false
}

func (b *Builder) prepareGlossary(root string, files []fileEntry, stats *Stats) error {
	b.glossary = nil
	b.termLabel = ""
	b.buildWarns = nil
	b.incomplete = nil
	b.seenOnPage = nil
	b.notedPages = nil
	if b.cfg == nil || !anyMarkdownUsesGlossary(b.cfg, files) {
		return nil
	}
	lang := ""
	if b.cfg != nil {
		lang = b.cfg.Site.Language
	}
	b.termLabel = theme.UIStrings(lang)["glossary_term_label"]
	glossary, warnings, err := LoadGlossary(root, b.cfg.Glossary.Dir)
	if err != nil {
		return err
	}
	b.glossary = glossary
	b.incomplete = make(map[string]render.GlossaryTerm)
	b.seenOnPage = make(map[string]string)
	b.notedPages = make(map[string]struct{})
	for _, w := range warnings {
		stats.Warnings = append(stats.Warnings, config.Warning{Msg: w})
	}
	return nil
}

func (b *Builder) markdownRenderOptions(repoPath string) render.MarkdownOptions {
	opts := b.cfg.OptionsFor(repoPath)
	md := render.MarkdownOptions{
		TOC:              opts.Markdown.TOC,
		TOCMinHeadings:   opts.Markdown.TOCMinHeadings,
		Anchors:          opts.Markdown.Anchors,
		Mermaid:          opts.Markdown.Mermaid,
		FrontmatterTitle: opts.Markdown.FrontmatterTitle,
		Glossary:         opts.Markdown.Glossary,
		GlossaryStrict:   glossaryStrictness(b.cfg),
	}
	if md.Glossary {
		md.Terms = b.glossary
		md.GlossaryTermLabel = b.termLabel
	}
	return md
}

func (b *Builder) noteMarkdownResult(pagePath string, result render.MarkdownResult) {
	if b.notedPages == nil {
		b.notedPages = make(map[string]struct{})
	}
	if _, seen := b.notedPages[pagePath]; seen {
		return
	}
	b.notedPages[pagePath] = struct{}{}
	for _, w := range result.Warnings {
		b.buildWarns = append(b.buildWarns, config.Warning{Msg: w})
	}
	if glossaryStrictness(b.cfg) != render.GlossaryStrictComplete {
		return
	}
	for _, term := range result.Terms {
		if !term.IsIncomplete() {
			continue
		}
		if _, seen := b.incomplete[term.Key]; seen {
			continue
		}
		b.incomplete[term.Key] = term
		b.seenOnPage[term.Key] = pagePath
	}
}

func (b *Builder) failIfIncomplete() error {
	if glossaryStrictness(b.cfg) != render.GlossaryStrictComplete || len(b.incomplete) == 0 {
		return nil
	}
	keys := make([]string, 0, len(b.incomplete))
	for key := range b.incomplete {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var bld strings.Builder
	for i, key := range keys {
		if i > 0 {
			bld.WriteByte('\n')
		}
		loc := b.incomplete[key].DefinedIn
		if loc == "" {
			loc = b.seenOnPage[key]
		}
		fmt.Fprintf(&bld, "incomplete glossary term %q at %s", key, loc)
	}
	return errors.New(bld.String())
}
