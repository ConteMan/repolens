package site

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"

	"github.com/ConteMan/repolens/internal/render"
)

const searchJSONPath = "search.json"

type searchDocument struct {
	Docs []searchDoc `json:"docs"`
}

type searchDoc struct {
	Path     string          `json:"path"`
	Title    string          `json:"title"`
	Kind     string          `json:"kind"`
	View     string          `json:"view"`
	Headings []searchHeading `json:"headings"`
}

type searchHeading struct {
	Text   string `json:"text"`
	Anchor string `json:"anchor"`
	Level  int    `json:"level"`
}

func (b *Builder) writeSearchJSON(outDir string, model siteModel) error {
	docs := make([]searchDoc, 0, len(model.files))
	for _, file := range model.files {
		if !file.Render {
			continue
		}
		title := path.Base(file.Path)
		headings := make([]searchHeading, 0)
		if file.Kind == render.KindMarkdown {
			data, err := os.ReadFile(filepath.Join(model.root, filepath.FromSlash(file.Path)))
			if err != nil {
				return err
			}
			opts := b.cfg.OptionsFor(file.Path)
			result, err := b.markdown.Render(data, render.PageRef{Path: file.Path}, render.MarkdownOptions{
				TOC:              true,
				TOCMinHeadings:   1,
				Anchors:          opts.Markdown.Anchors,
				Mermaid:          false,
				FrontmatterTitle: opts.Markdown.FrontmatterTitle,
			})
			if err != nil {
				return err
			}
			title = result.Title
			headings = flattenSearchTOC(result.TOC)
		}
		docs = append(docs, searchDoc{
			Path:     file.Path,
			Title:    title,
			Kind:     file.KindName,
			View:     model.browserURLForFile(file),
			Headings: headings,
		})
	}

	data, err := json.MarshalIndent(searchDocument{Docs: docs}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(outDir, searchJSONPath), data, 0o644)
}

func flattenSearchTOC(items []render.TOCItem) []searchHeading {
	headings := make([]searchHeading, 0)
	var walk func([]render.TOCItem)
	walk = func(items []render.TOCItem) {
		for _, item := range items {
			headings = append(headings, searchHeading{
				Text:   item.Title,
				Anchor: item.Anchor,
				Level:  item.Level,
			})
			walk(item.Children)
		}
	}
	walk(items)
	return headings
}
