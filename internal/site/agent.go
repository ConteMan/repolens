package site

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/ConteMan/repolens/internal/render"
	"github.com/bmatcuk/doublestar/v4"
)

const (
	llmsTxtPath      = "llms.txt"
	llmsFullPath     = "llms-full.txt"
	indexJSONPath    = "index.json"
	truncatedMarker  = "\n[truncated]\n"
	summaryMaxRunes  = 120
	agentOutputPerm  = 0o644
	frontMatterFence = "---"
)

// GeneratorVersion is the default version string used outside the CLI.
var GeneratorVersion = "dev"

type agentMarkdownEntry struct {
	file    fileEntry
	title   string
	summary string
}

func (b *Builder) writeAgentOutputs(outDir string, model siteModel, builtAt time.Time) error {
	if b.cfg.Agent.LLMSTxt {
		entries, err := b.markdownEntries(model)
		if err != nil {
			return err
		}
		if err := b.writeLLMSTxt(outDir, entries); err != nil {
			return err
		}
	}
	if b.cfg.Agent.LLMSFull.Enabled {
		if err := b.writeLLMSFull(outDir, model.files); err != nil {
			return err
		}
	}
	if b.cfg.Agent.IndexJSON {
		if err := b.writeIndexJSON(outDir, model, builtAt); err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) markdownEntries(model siteModel) ([]agentMarkdownEntry, error) {
	entries := make([]agentMarkdownEntry, 0)
	for _, file := range model.files {
		if !isMarkdownPath(file.Path) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(model.root, filepath.FromSlash(file.Path)))
		if err != nil {
			return nil, err
		}
		title, err := b.markdownTitle(data, file.Path)
		if err != nil {
			return nil, err
		}
		entries = append(entries, agentMarkdownEntry{
			file:    file,
			title:   title,
			summary: markdownSummary(data),
		})
	}
	return entries, nil
}

func (b *Builder) markdownTitle(data []byte, repoPath string) (string, error) {
	opts := b.cfg.OptionsFor(repoPath)
	result, err := b.markdown.Render(data, render.PageRef{Path: repoPath}, render.MarkdownOptions{
		TOC:              false,
		Anchors:          opts.Markdown.Anchors,
		Mermaid:          false,
		FrontmatterTitle: opts.Markdown.FrontmatterTitle,
	})
	if err != nil {
		return "", err
	}
	return result.Title, nil
}

func (b *Builder) writeLLMSTxt(outDir string, entries []agentMarkdownEntry) error {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s\n\n", b.siteTitle())
	fmt.Fprintf(&buf, "> Browseable static rendering of the %s repository for humans and AI agents.\n\n", b.siteTitle())
	buf.WriteString("Raw files are available at `/<repo-path>`.\n")
	buf.WriteString("Browser pages are available at `/view/<repo-path>/`.\n\n")

	byDir := make(map[string][]agentMarkdownEntry)
	var dirs []string
	for _, entry := range entries {
		dir := parentDir(entry.file.Path)
		if _, ok := byDir[dir]; !ok {
			dirs = append(dirs, dir)
		}
		byDir[dir] = append(byDir[dir], entry)
	}
	sort.Slice(dirs, func(i, j int) bool {
		if dirs[i] == "" {
			return true
		}
		if dirs[j] == "" {
			return false
		}
		return dirs[i] < dirs[j]
	})
	for _, dir := range dirs {
		if dir == "" {
			buf.WriteString("## /\n\n")
		} else {
			fmt.Fprintf(&buf, "## %s/\n\n", dir)
		}
		for _, entry := range byDir[dir] {
			fmt.Fprintf(&buf, "- [%s](%s)", escapeMarkdownLinkText(entry.title), entry.file.Path)
			if entry.summary != "" {
				fmt.Fprintf(&buf, ": %s", entry.summary)
			}
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	var refs []string
	if b.cfg.Agent.IndexJSON {
		refs = append(refs, "[index.json](index.json)")
	}
	if b.cfg.Agent.LLMSFull.Enabled {
		refs = append(refs, "[llms-full.txt](llms-full.txt)")
	}
	if len(refs) > 0 {
		buf.WriteString("## Agent Files\n\n")
		for _, ref := range refs {
			fmt.Fprintf(&buf, "- %s\n", ref)
		}
	}
	return os.WriteFile(filepath.Join(outDir, llmsTxtPath), buf.Bytes(), agentOutputPerm)
}

func (b *Builder) writeLLMSFull(outDir string, files []fileEntry) error {
	var buf bytes.Buffer
	limit := b.cfg.Agent.LLMSFull.MaxSize
	truncated := false
	for _, file := range files {
		if !includeInLLMSFull(file) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(outDir, filepath.FromSlash(file.Path)))
		if err != nil {
			return err
		}
		chunk := append([]byte(fmt.Sprintf("----- %s -----\n", file.Path)), data...)
		if len(data) == 0 || data[len(data)-1] != '\n' {
			chunk = append(chunk, '\n')
		}
		chunk = append(chunk, '\n')
		if limit >= 0 && int64(buf.Len()+len(chunk)) > limit {
			remaining := int(limit) - buf.Len()
			if remaining > 0 {
				trunc := chunk[:remaining]
				// 回退到合法 UTF-8 边界，避免截出半个多字节字符。
				for len(trunc) > 0 && !utf8.Valid(trunc) {
					trunc = trunc[:len(trunc)-1]
				}
				buf.Write(trunc)
			}
			truncated = true
			break
		}
		buf.Write(chunk)
	}
	if truncated {
		buf.WriteString(truncatedMarker)
	}
	return os.WriteFile(filepath.Join(outDir, llmsFullPath), buf.Bytes(), agentOutputPerm)
}

func includeInLLMSFull(file fileEntry) bool {
	if isMarkdownPath(file.Path) {
		return true
	}
	if file.Kind != render.KindCode {
		return false
	}
	return isPlainTextPath(file.Path)
}

func isPlainTextPath(repoPath string) bool {
	switch strings.ToLower(path.Ext(repoPath)) {
	case ".txt", ".text", ".log":
		return true
	default:
		return false
	}
}

type indexDocument struct {
	Generator string `json:"generator"`
	// Commit is null in worktree mode (no commit hash available).
	Commit  *string     `json:"commit"`
	BuiltAt string      `json:"built_at"`
	Site    indexSite   `json:"site"`
	Files   []indexFile `json:"files"`
}

type indexSite struct {
	Title string `json:"title"`
}

type indexFile struct {
	Path     string  `json:"path"`
	Kind     string  `json:"kind"`
	Size     int64   `json:"size"`
	Title    *string `json:"title"`
	Modified *string `json:"modified"`
	Raw      string  `json:"raw"`
	View     *string `json:"view"`
}

func (b *Builder) writeIndexJSON(outDir string, model siteModel, builtAt time.Time) error {
	files := make([]indexFile, 0, len(model.files))
	for _, file := range model.files {
		var title *string
		if isMarkdownPath(file.Path) {
			data, err := os.ReadFile(filepath.Join(model.root, filepath.FromSlash(file.Path)))
			if err != nil {
				return err
			}
			t, err := b.markdownTitle(data, file.Path)
			if err != nil {
				return err
			}
			title = &t
		}

		var modified *string
		if file.LastCommit != nil {
			value := file.LastCommit.Time.UTC().Format(time.RFC3339)
			modified = &value
		}

		var view *string
		if file.Render {
			value := model.browserURLForFile(file)
			view = &value
		}

		files = append(files, indexFile{
			Path:     file.Path,
			Kind:     file.KindName,
			Size:     file.Size,
			Title:    title,
			Modified: modified,
			Raw:      mirrorURL(file.Path),
			View:     view,
		})
	}

	var commit *string
	if model.commitHash != "" {
		commit = &model.commitHash
	}
	doc := indexDocument{
		Generator: "repolens " + b.version,
		Commit:    commit,
		BuiltAt:   builtAt.UTC().Format(time.RFC3339),
		Site:      indexSite{Title: b.siteTitle()},
		Files:     files,
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(outDir, indexJSONPath), data, agentOutputPerm)
}

func agentWarnings(cfg *config.Config) []config.Warning {
	if cfg == nil || !cfg.Access.Encrypt.Enabled {
		return nil
	}
	outputs := enabledAgentOutputPaths(cfg.Agent)
	if len(outputs) == 0 {
		return nil
	}
	if len(cfg.Access.Encrypt.Paths) == 0 {
		return []config.Warning{{Msg: "access.encrypt covers the whole site while agent outputs are enabled"}}
	}
	for _, pattern := range cfg.Access.Encrypt.Paths {
		for _, output := range outputs {
			matched, err := doublestar.Match(pattern, output)
			if err == nil && matched {
				return []config.Warning{{Msg: "access.encrypt paths overlap enabled agent outputs"}}
			}
		}
	}
	return nil
}

func enabledAgentOutputPaths(agent config.Agent) []string {
	var paths []string
	if agent.LLMSTxt {
		paths = append(paths, llmsTxtPath)
	}
	if agent.LLMSFull.Enabled {
		paths = append(paths, llmsFullPath)
	}
	if agent.IndexJSON {
		paths = append(paths, indexJSONPath)
	}
	return paths
}

func markdownSummary(src []byte) string {
	text := string(stripFrontMatter(bytes.TrimPrefix(src, []byte("\xef\xbb\xbf"))))
	paragraphs := splitMarkdownParagraphs(text)
	for _, paragraph := range paragraphs {
		if isHeadingParagraph(paragraph) {
			continue
		}
		summary := collapseWhitespace(paragraph)
		if summary != "" {
			return truncateRunes(summary, summaryMaxRunes)
		}
	}
	return ""
}

func stripFrontMatter(src []byte) []byte {
	if !bytes.HasPrefix(src, []byte(frontMatterFence+"\n")) && !bytes.HasPrefix(src, []byte(frontMatterFence+"\r\n")) {
		return src
	}
	lines := bytes.SplitAfter(src, []byte("\n"))
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(string(lines[i]))
		if line == frontMatterFence || line == "..." {
			return bytes.Join(lines[i+1:], nil)
		}
	}
	return src
}

func splitMarkdownParagraphs(text string) []string {
	lines := strings.Split(text, "\n")
	var paragraphs []string
	var current []string
	flush := func() {
		if len(current) == 0 {
			return
		}
		paragraphs = append(paragraphs, strings.Join(current, "\n"))
		current = nil
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		current = append(current, line)
	}
	flush()
	return paragraphs
}

func isHeadingParagraph(paragraph string) bool {
	lines := strings.Split(paragraph, "\n")
	if len(lines) == 0 {
		return false
	}
	first := strings.TrimSpace(lines[0])
	if strings.HasPrefix(first, "#") {
		rest := strings.TrimLeft(first, "#")
		return rest == "" || strings.HasPrefix(rest, " ")
	}
	if len(lines) >= 2 {
		second := strings.TrimSpace(lines[1])
		if second != "" && (allRunes(second, '=') || allRunes(second, '-')) {
			return true
		}
	}
	return false
}

func allRunes(s string, r rune) bool {
	if s == "" {
		return false
	}
	for _, got := range s {
		if got != r {
			return false
		}
	}
	return true
}

func collapseWhitespace(text string) string {
	return strings.TrimSpace(strings.Join(strings.FieldsFunc(text, unicode.IsSpace), " "))
}

func truncateRunes(text string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

func escapeMarkdownLinkText(text string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `[`, `\[`, `]`, `\]`)
	return replacer.Replace(text)
}

func isMarkdownPath(repoPath string) bool {
	switch strings.ToLower(path.Ext(repoPath)) {
	case ".md", ".markdown":
		return true
	default:
		return false
	}
}
