package site

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ConteMan/repolens/internal/config"
	"github.com/ConteMan/repolens/internal/render"
	"github.com/ConteMan/repolens/internal/source"
	"github.com/ConteMan/repolens/internal/theme"
)

const sentinelName = ".repolens-build"

// OwnsOutput reports whether dir contains the regular sentinel file written by
// repolens. A symlink or directory with the sentinel name does not establish
// output ownership.
func OwnsOutput(dir string) (bool, error) {
	info, err := os.Lstat(filepath.Join(dir, sentinelName))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode().IsRegular(), nil
}

type Builder struct {
	cfg      *config.Config
	theme    *theme.Renderer
	markdown *render.Markdown
	version  string
}

type Stats struct {
	// Pages counts generated browser-layer HTML pages. A renderable index.html
	// merged into its directory page is counted only as that directory page.
	Files, Pages int
	Duration     time.Duration
	// Warnings collects build-time warnings produced by site assembly.
	// Config cascade warnings are reported by the cli layer before these.
	Warnings []config.Warning
}

func NewBuilder(cfg *config.Config, renderer *theme.Renderer) *Builder {
	return &Builder{
		cfg:      cfg,
		theme:    renderer,
		markdown: render.NewMarkdown(),
		version:  GeneratorVersion,
	}
}

func (b *Builder) SetGeneratorVersion(version string) {
	if strings.TrimSpace(version) != "" {
		b.version = version
	}
}

func (b *Builder) Build(ctx context.Context, tree *source.Tree, outDir string) (Stats, error) {
	start := time.Now()
	stats := Stats{}
	if b == nil || b.cfg == nil || b.theme == nil {
		return stats, errors.New("site: builder requires config and theme")
	}
	if tree == nil {
		return stats, errors.New("site: source tree is required")
	}
	if outDir == "" {
		return stats, errors.New("site: output directory is required")
	}

	files, err := b.collectFiles(tree)
	if err != nil {
		return stats, err
	}
	dirs := collectDirs(files)
	if err := validateIndexHTMLDirs(dirs); err != nil {
		return stats, err
	}
	model := newSiteModel(tree.Root, tree.CommitHash, files, dirs)

	if err := prepareOutput(outDir); err != nil {
		return stats, err
	}
	if err := b.theme.WriteAssets(outDir); err != nil {
		return stats, fmt.Errorf("write theme assets: %w", err)
	}

	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		if err := copyMirror(tree.Root, outDir, file); err != nil {
			return stats, err
		}
		stats.Files++
		if !file.Render {
			continue
		}
		if model.mergesIntoDirPage(file) {
			continue
		}
		if err := b.writeFilePage(outDir, model, file); err != nil {
			return stats, err
		}
		stats.Pages++
		if supportsSourcePage(file, b.cfg.OptionsFor(file.Path)) {
			if err := b.writeSourcePage(outDir, model, file); err != nil {
				return stats, err
			}
			stats.Pages++
		}
	}

	for _, dir := range dirs {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		if err := b.writeDirPage(outDir, model, dir); err != nil {
			return stats, err
		}
		stats.Pages++
	}

	_, hasRootIndex := model.fileByPath["index.html"]
	generatedRootIndex := !hasRootIndex
	if generatedRootIndex {
		if err := b.writeRootIndex(outDir); err != nil {
			return stats, err
		}
	}
	_, hasRoot404 := model.fileByPath["404.html"]
	generated404 := !hasRoot404
	if generated404 {
		if err := b.write404Page(outDir); err != nil {
			return stats, err
		}
	}
	if b.cfg.Access.NoIndex {
		if err := os.WriteFile(filepath.Join(outDir, "robots.txt"), []byte("User-agent: *\nDisallow: /\n"), 0o644); err != nil {
			return stats, err
		}
	}
	if b.cfg.View.Search {
		if err := b.writeSearchJSON(outDir, model); err != nil {
			return stats, err
		}
	}
	if err := b.writeAgentOutputs(outDir, model, time.Now()); err != nil {
		return stats, err
	}
	stats.Warnings = append(stats.Warnings, agentWarnings(b.cfg)...)
	if err := checkRelativeLinks(outDir, generatedRootIndex, generated404); err != nil {
		return stats, err
	}
	stats.Duration = time.Since(start)
	return stats, nil
}

type fileEntry struct {
	source.File
	Kind     render.Kind
	KindName string
	Render   bool
}

func (b *Builder) collectFiles(tree *source.Tree) ([]fileEntry, error) {
	files := make([]fileEntry, 0, len(tree.Files))
	for _, file := range tree.Files {
		if b.cfg.Ignored(file.Path) {
			continue
		}
		opts := b.cfg.OptionsFor(file.Path)
		sniff, err := readSniff(filepath.Join(tree.Root, filepath.FromSlash(file.Path)), file.Size, opts.MaxFileSize)
		if err != nil {
			return nil, err
		}
		kind := render.Classify(file.Path, file.Size, sniff, opts.MaxFileSize)
		files = append(files, fileEntry{
			File:     file,
			Kind:     kind,
			KindName: kindName(kind),
			Render:   opts.Render,
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func readSniff(filePath string, size, maxFileSize int64) ([]byte, error) {
	if maxFileSize > 0 && size > maxFileSize {
		return nil, nil
	}
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return buf[:n], nil
}

func collectDirs(files []fileEntry) []string {
	set := map[string]bool{"": true}
	for _, file := range files {
		dir := path.Dir(file.Path)
		if dir == "." {
			dir = ""
		}
		for {
			set[dir] = true
			if dir == "" {
				break
			}
			dir = path.Dir(dir)
			if dir == "." {
				dir = ""
			}
		}
	}
	dirs := make([]string, 0, len(set))
	for dir := range set {
		dirs = append(dirs, dir)
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
	return dirs
}

func validateIndexHTMLDirs(dirs []string) error {
	for _, dir := range dirs {
		if !isIndexHTMLDir(dir) {
			continue
		}
		parentPage := outputHTMLRel(viewDirURL(parentDir(dir)))
		dirPage := outputHTMLRel(viewDirURL(dir))
		return fmt.Errorf("site: repository directory %q collides with generated browser paths %q and %q; rename the directory before building", dir, parentPage, dirPage)
	}
	return nil
}

func prepareOutput(outDir string) error {
	info, err := os.Stat(outDir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("site: output path %q exists and is not a directory", outDir)
		}
		owned, err := OwnsOutput(outDir)
		if err != nil {
			return err
		}
		if !owned {
			return fmt.Errorf("site: refusing to clear %q without %s sentinel", outDir, sentinelName)
		}
		if err := clearDir(outDir); err != nil {
			return err
		}
	} else if os.IsNotExist(err) {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return err
		}
	} else {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, sentinelName), []byte("repolens build output\n"), 0o644)
}

func clearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyMirror(root, outDir string, file fileEntry) error {
	srcPath := filepath.Join(root, filepath.FromSlash(file.Path))
	dstPath := filepath.Join(outDir, filepath.FromSlash(file.Path))
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

type siteModel struct {
	root             string
	commitHash       string
	files            []fileEntry
	fileByPath       map[string]fileEntry
	dirs             map[string]bool
	children         map[string][]listItem
	mergedIndexByDir map[string]fileEntry
}

type listItem struct {
	Name       string
	Path       string
	Kind       string
	Size       int64
	LastCommit *source.Commit
	IsDir      bool
}

func newSiteModel(root, commitHash string, files []fileEntry, dirs []string) siteModel {
	m := siteModel{
		root:             root,
		commitHash:       commitHash,
		files:            files,
		fileByPath:       make(map[string]fileEntry, len(files)),
		dirs:             make(map[string]bool, len(dirs)),
		children:         make(map[string][]listItem),
		mergedIndexByDir: make(map[string]fileEntry),
	}
	for _, dir := range dirs {
		m.dirs[dir] = true
	}
	for _, file := range files {
		m.fileByPath[file.Path] = file
		if file.Render && isIndexHTMLFile(file.Path) {
			m.mergedIndexByDir[parentDir(file.Path)] = file
		}
	}
	seenDirs := make(map[string]map[string]bool)
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		parent := parentDir(dir)
		if seenDirs[parent] == nil {
			seenDirs[parent] = make(map[string]bool)
		}
		if !seenDirs[parent][dir] {
			seenDirs[parent][dir] = true
			m.children[parent] = append(m.children[parent], listItem{
				Name:  path.Base(dir),
				Path:  dir,
				Kind:  "dir",
				IsDir: true,
			})
		}
	}
	for _, file := range files {
		dir := parentDir(file.Path)
		m.children[dir] = append(m.children[dir], listItem{
			Name:       path.Base(file.Path),
			Path:       file.Path,
			Kind:       file.KindName,
			Size:       file.Size,
			LastCommit: file.LastCommit,
		})
	}
	for dir := range m.children {
		sort.Slice(m.children[dir], func(i, j int) bool {
			a, b := m.children[dir][i], m.children[dir][j]
			if a.IsDir != b.IsDir {
				return a.IsDir
			}
			return a.Name < b.Name
		})
	}
	return m
}

func (m siteModel) mergesIntoDirPage(file fileEntry) bool {
	index, ok := m.mergedIndexByDir[parentDir(file.Path)]
	return ok && index.Path == file.Path
}

func (m siteModel) browserURLForFile(file fileEntry) string {
	if m.mergesIntoDirPage(file) {
		return viewDirURL(parentDir(file.Path))
	}
	return viewFileURL(file.Path)
}

func (b *Builder) writeFilePage(outDir string, model siteModel, file fileEntry) error {
	currentURL := viewFileURL(file.Path)
	body, toc, title, hasMermaid, err := b.fileBody(model, currentURL, file)
	if err != nil {
		return err
	}
	if title == "" {
		title = path.Base(file.Path)
	}
	data := theme.PageData{
		Title:       title,
		Breadcrumbs: breadcrumbs(currentURL, file.Path, false),
		Tree:        buildTree(model, currentURL, file.Path, b.cfg.View.TreeExpandDepth),
		Kind:        pageKind(file, b.cfg.OptionsFor(file.Path)),
		Body:        body,
		TOC:         toc,
		MirrorHref:  render.RelTo(currentURL, mirrorURL(file.Path)),
		SourceHref:  sourceHref(currentURL, file, b.cfg.OptionsFor(file.Path)),
		FileSize:    file.Size,
		RepoPath:    file.Path,
		LastCommit:  file.LastCommit,
		HasMermaid:  hasMermaid,
		HeadExtra:   alternateHead(currentURL, file.Path),
	}
	b.fillCommonPageData(currentURL, &data)
	return b.writePage(outDir, currentURL, data)
}

func (b *Builder) writeSourcePage(outDir string, model siteModel, file fileEntry) error {
	currentURL := viewSourceURL(file.Path)
	opts := b.cfg.OptionsFor(file.Path)
	data, err := os.ReadFile(filepath.Join(model.root, filepath.FromSlash(file.Path)))
	if err != nil {
		return err
	}
	code, err := render.Code(data, file.Path, render.CodeOptions{
		LineNumbers: opts.Code.LineNumbers,
		Theme:       opts.Code.Theme,
	})
	if err != nil {
		return err
	}
	kind := "markdown-source"
	if file.Kind == render.KindHTML {
		kind = "html-source"
	}
	page := theme.PageData{
		Title:       path.Base(file.Path),
		Breadcrumbs: breadcrumbs(currentURL, file.Path, false),
		Tree:        buildTree(model, currentURL, file.Path, b.cfg.View.TreeExpandDepth),
		Kind:        kind,
		Body:        code.HTML,
		MirrorHref:  render.RelTo(currentURL, mirrorURL(file.Path)),
		SourceHref:  render.RelTo(currentURL, model.browserURLForFile(file)),
		FileSize:    file.Size,
		RepoPath:    file.Path,
		LastCommit:  file.LastCommit,
		HeadExtra:   alternateHead(currentURL, file.Path),
	}
	b.fillCommonPageData(currentURL, &page)
	return b.writePage(outDir, currentURL, page)
}

func (b *Builder) fileBody(model siteModel, currentURL string, file fileEntry) (template.HTML, []render.TOCItem, string, bool, error) {
	opts := b.cfg.OptionsFor(file.Path)
	switch file.Kind {
	case render.KindMarkdown:
		data, err := os.ReadFile(filepath.Join(model.root, filepath.FromSlash(file.Path)))
		if err != nil {
			return "", nil, "", false, err
		}
		result, err := b.markdown.Render(data, render.PageRef{
			Path: file.Path,
			Resolve: func(target string) string {
				entry, ok := model.fileByPath[cleanRepoPath(target)]
				if !ok {
					return ""
				}
				if entry.Render {
					return "view"
				}
				return "mirror"
			},
		}, render.MarkdownOptions{
			TOC:              opts.Markdown.TOC,
			TOCMinHeadings:   opts.Markdown.TOCMinHeadings,
			Anchors:          opts.Markdown.Anchors,
			Mermaid:          opts.Markdown.Mermaid,
			FrontmatterTitle: opts.Markdown.FrontmatterTitle,
		})
		if err != nil {
			return "", nil, "", false, err
		}
		return result.HTML, result.TOC, result.Title, result.HasMermaid, nil
	case render.KindHTML:
		switch opts.HTML.View {
		case "direct":
			return htmlDirectBody(currentURL, file.Path), nil, path.Base(file.Path), false, nil
		case "source":
			data, err := os.ReadFile(filepath.Join(model.root, filepath.FromSlash(file.Path)))
			if err != nil {
				return "", nil, "", false, err
			}
			code, err := render.Code(data, file.Path, render.CodeOptions{
				LineNumbers: opts.Code.LineNumbers,
				Theme:       opts.Code.Theme,
			})
			if err != nil {
				return "", nil, "", false, err
			}
			return code.HTML, nil, path.Base(file.Path), false, nil
		default:
			return htmlEmbedBody(currentURL, file.Path), nil, path.Base(file.Path), false, nil
		}
	case render.KindCode:
		data, err := os.ReadFile(filepath.Join(model.root, filepath.FromSlash(file.Path)))
		if err != nil {
			return "", nil, "", false, err
		}
		code, err := render.Code(data, file.Path, render.CodeOptions{
			LineNumbers: opts.Code.LineNumbers,
			Theme:       opts.Code.Theme,
		})
		if err != nil {
			return "", nil, "", false, err
		}
		return code.HTML, nil, path.Base(file.Path), false, nil
	case render.KindImage:
		return imageBody(currentURL, file.Path), nil, path.Base(file.Path), false, nil
	default:
		return binaryBody(currentURL, file), nil, path.Base(file.Path), false, nil
	}
}

func (b *Builder) writeDirPage(outDir string, model siteModel, dir string) error {
	currentURL := viewDirURL(dir)
	body, toc, title, hasMermaid, lastCommit, err := b.dirBody(model, currentURL, dir)
	if err != nil {
		return err
	}
	if title == "" {
		title = dirTitle(dir)
	}
	data := theme.PageData{
		Title:       title,
		Breadcrumbs: breadcrumbs(currentURL, dir, true),
		Tree:        buildTree(model, currentURL, dir, b.cfg.View.TreeExpandDepth),
		Kind:        "dir",
		Body:        body,
		TOC:         toc,
		LastCommit:  lastCommit,
		HasMermaid:  hasMermaid,
		RepoPath:    dir,
		DirEntries:  dirEntries(model, currentURL, dir),
	}
	b.fillCommonPageData(currentURL, &data)
	return b.writePage(outDir, currentURL, data)
}

func (b *Builder) fillCommonPageData(currentURL string, data *theme.PageData) {
	data.SiteTitle = b.siteTitle()
	data.RelRoot = relRoot(currentURL)
	data.NoIndex = b.cfg.Access.NoIndex
	data.SearchEnabled = b.cfg.View.Search
	data.Lang = b.cfg.Site.Language
	data.TOCPanel = tocPanelMode(b.cfg.View.TOCPanel)
	data.UI = theme.UIStrings(b.cfg.Site.Language)
	if data.Tree != nil {
		data.Tree.SearchEnabled = b.cfg.View.Search
	}
}

func (b *Builder) dirBody(model siteModel, currentURL, dir string) (template.HTML, []render.TOCItem, string, bool, *source.Commit, error) {
	var toc []render.TOCItem
	var title string
	var hasMermaid bool
	var lastCommit *source.Commit
	var body template.HTML

	if index, ok := model.mergedIndexByDir[dir]; ok {
		body, toc, _, hasMermaid, err := b.fileBody(model, currentURL, index)
		if err != nil {
			return "", nil, "", false, nil, err
		}
		// 合并页标题留空走 dirTitle 回退：fileBody 对 HTML 恒返回
		// 文件名 "index.html"，作为目录页标题无意义。
		return body, toc, "", hasMermaid, index.LastCommit, nil
	}

	if readme, ok := b.dirDocFile(model, dir); ok {
		data, err := os.ReadFile(filepath.Join(model.root, filepath.FromSlash(readme.Path)))
		if err != nil {
			return "", nil, "", false, nil, err
		}
		opts := b.cfg.OptionsFor(readme.Path)
		result, err := b.markdown.Render(data, render.PageRef{
			Path: readme.Path,
			Resolve: func(target string) string {
				entry, ok := model.fileByPath[cleanRepoPath(target)]
				if !ok {
					return ""
				}
				if entry.Render {
					return "view"
				}
				return "mirror"
			},
		}, render.MarkdownOptions{
			TOC:              opts.Markdown.TOC,
			TOCMinHeadings:   opts.Markdown.TOCMinHeadings,
			Anchors:          opts.Markdown.Anchors,
			Mermaid:          opts.Markdown.Mermaid,
			FrontmatterTitle: opts.Markdown.FrontmatterTitle,
		})
		if err != nil {
			return "", nil, "", false, nil, err
		}
		html := rebaseRenderedLinks(string(result.HTML), viewFileURL(readme.Path), currentURL)
		body = template.HTML(html)
		toc = result.TOC
		title = result.Title
		hasMermaid = result.HasMermaid
		lastCommit = readme.LastCommit
	}
	return body, toc, title, hasMermaid, lastCommit, nil
}

func (b *Builder) writePage(outDir, pageURL string, data theme.PageData) error {
	target := outputHTMLPath(outDir, pageURL)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := b.theme.Page(&buf, data); err != nil {
		return err
	}
	return os.WriteFile(target, buf.Bytes(), 0o644)
}

func (b *Builder) writeRootIndex(outDir string) error {
	noindex := ""
	if b.cfg.Access.NoIndex {
		noindex = `<meta name="robots" content="noindex">` + "\n"
	}
	html := "<!doctype html>\n<html><head><meta charset=\"utf-8\">\n" +
		noindex +
		"<meta http-equiv=\"refresh\" content=\"0; url=view/\">\n" +
		"<title>" + template.HTMLEscapeString(b.siteTitle()) + "</title></head>\n" +
		"<body><a href=\"view/\">Open site</a></body></html>\n"
	return os.WriteFile(filepath.Join(outDir, "index.html"), []byte(html), 0o644)
}

// write404Page emits a standalone not-found page at the site root. Static
// hosts serve it for unmatched paths；没有它时 Cloudflare Pages 会把任意未命中
// 路径回退成根 index.html，与根跳转页的相对 view/ 链接叠加成无限重定向。
// 页面会被任意深度的路径命中，因此不引用任何站点资源、不放站内链接。
// 仓库自带根 404.html 时让位（镜像层已占该输出路径）。
func (b *Builder) write404Page(outDir string) error {
	lang := strings.TrimSpace(b.cfg.Site.Language)
	heading, hint := "Page not found", "This path has no content on this site. The link may point outside the repository."
	if strings.HasPrefix(strings.ToLower(lang), "zh") {
		heading, hint = "页面不存在", "你访问的路径在本站没有对应内容，可能来自指向仓库之外的链接。"
	}
	langAttr := ""
	if lang != "" {
		langAttr = ` lang="` + template.HTMLEscapeString(lang) + `"`
	}
	html := "<!doctype html>\n<html" + langAttr + "><head><meta charset=\"utf-8\">\n" +
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n" +
		"<meta name=\"robots\" content=\"noindex\">\n" +
		"<title>404 - " + template.HTMLEscapeString(b.siteTitle()) + "</title>\n" +
		"<style>body{margin:0;display:grid;min-height:100vh;place-items:center;font:15px/1.6 ui-sans-serif,system-ui,sans-serif;color:#24292f;background:#fff}main{padding:2rem;text-align:center}h1{margin:0 0 .4rem;font-size:2.6rem}p{margin:.2rem 0;color:#6e7781}@media(prefers-color-scheme:dark){body{color:#e6edf3;background:#0d1117}p{color:#8b949e}}</style></head>\n" +
		"<body><main><h1>404</h1><p><b>" + template.HTMLEscapeString(heading) + "</b></p><p>" +
		template.HTMLEscapeString(hint) + "</p><p>" + template.HTMLEscapeString(b.siteTitle()) + "</p></main></body></html>\n"
	return os.WriteFile(filepath.Join(outDir, "404.html"), []byte(html), 0o644)
}

func (b *Builder) siteTitle() string {
	if strings.TrimSpace(b.cfg.Site.Title) != "" {
		return b.cfg.Site.Title
	}
	if strings.TrimSpace(b.cfg.Source.Repo) != "" {
		base := path.Base(filepath.ToSlash(strings.TrimRight(b.cfg.Source.Repo, `/\`)))
		if base != "." && base != "/" && base != "" {
			return strings.TrimSuffix(base, ".git")
		}
	}
	return "repolens"
}

// dirDocFile 选目录页正文来源：根目录优先 site.home（缺失或不可渲染
// 时回退 README），其余目录按 README.md、readme.md 顺序。
func (b *Builder) dirDocFile(model siteModel, dir string) (fileEntry, bool) {
	if dir == "" && b.cfg != nil {
		if home := cleanRepoPath(b.cfg.Site.Home); home != "" {
			if file, ok := model.fileByPath[home]; ok && file.Render && file.Kind == render.KindMarkdown {
				return file, true
			}
		}
	}
	if readme, ok := findReadme(model, dir); ok && readme.Render && readme.Kind == render.KindMarkdown {
		return readme, true
	}
	return fileEntry{}, false
}

func findReadme(model siteModel, dir string) (fileEntry, bool) {
	for _, name := range []string{"README.md", "readme.md"} {
		p := name
		if dir != "" {
			p = dir + "/" + name
		}
		if file, ok := model.fileByPath[p]; ok {
			return file, true
		}
	}
	return fileEntry{}, false
}

func dirEntries(model siteModel, currentURL, dir string) []theme.DirEntry {
	items := model.children[dir]
	entries := make([]theme.DirEntry, 0, len(items))
	for _, item := range items {
		href := ""
		if item.IsDir {
			href = render.RelTo(currentURL, viewDirURL(item.Path))
		} else if file, ok := model.fileByPath[item.Path]; ok && file.Render {
			href = render.RelTo(currentURL, model.browserURLForFile(file))
		} else {
			href = render.RelTo(currentURL, mirrorURL(item.Path))
		}
		entries = append(entries, theme.DirEntry{
			Name:       item.Name,
			Path:       item.Path,
			Href:       href,
			Kind:       item.Kind,
			Size:       item.Size,
			LastCommit: item.LastCommit,
			IsDir:      item.IsDir,
		})
	}
	return entries
}

func htmlEmbedBody(currentURL, repoPath string) template.HTML {
	mirror := template.HTMLEscapeString(render.RelTo(currentURL, mirrorURL(repoPath)))
	return template.HTML(`<div class="toolbar"><a href="` + mirror + `">Open full screen</a></div><iframe class="html-preview" src="` + mirror + `"></iframe>`)
}

func htmlDirectBody(currentURL, repoPath string) template.HTML {
	mirror := template.HTMLEscapeString(render.RelTo(currentURL, mirrorURL(repoPath)))
	return template.HTML(`<div class="toolbar"><a href="` + mirror + `">Open HTML file</a></div>`)
}

func imageBody(currentURL, repoPath string) template.HTML {
	mirror := template.HTMLEscapeString(render.RelTo(currentURL, mirrorURL(repoPath)))
	return template.HTML(`<p><a href="` + mirror + `">Open image</a></p><img class="preview" src="` + mirror + `" alt="">`)
}

func binaryBody(currentURL string, file fileEntry) template.HTML {
	mirror := template.HTMLEscapeString(render.RelTo(currentURL, mirrorURL(file.Path)))
	name := template.HTMLEscapeString(path.Base(file.Path))
	return template.HTML(fmt.Sprintf(`<p>%s · %d bytes</p><p><a href="%s" download>Download</a></p>`, name, file.Size, mirror))
}

func alternateHead(currentURL, repoPath string) template.HTML {
	href := template.HTMLEscapeString(render.RelTo(currentURL, mirrorURL(repoPath)))
	typ := mime.TypeByExtension(path.Ext(repoPath))
	if isMarkdownPath(repoPath) {
		typ = "text/markdown"
	}
	if typ == "" {
		typ = "application/octet-stream"
	}
	return template.HTML(`<link rel="alternate" type="` + template.HTMLEscapeString(typ) + `" href="` + href + `">`)
}

// renderedAttrPattern only matches double-quoted attributes: goldmark always
// emits double quotes. Raw HTML embedded in Markdown with single-quoted
// href/src is passed through un-rebased (known limitation).
var renderedAttrPattern = regexp.MustCompile(`\b(href|src)="([^"]*)"`)

func rebaseRenderedLinks(html, fromURL, toURL string) string {
	return renderedAttrPattern.ReplaceAllStringFunc(html, func(match string) string {
		parts := renderedAttrPattern.FindStringSubmatch(match)
		if len(parts) != 3 || skipRebase(parts[2]) {
			return match
		}
		target, suffix := splitURLSuffix(parts[2])
		resolved := resolveRelativeURL(fromURL, target)
		if resolved == "" {
			return match
		}
		return parts[1] + `="` + template.HTMLEscapeString(render.RelTo(toURL, resolved)+suffix) + `"`
	})
}

func skipRebase(raw string) bool {
	if raw == "" || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return true
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme != ""
}

func splitURLSuffix(raw string) (string, string) {
	index := len(raw)
	if i := strings.IndexByte(raw, '?'); i >= 0 && i < index {
		index = i
	}
	if i := strings.IndexByte(raw, '#'); i >= 0 && i < index {
		index = i
	}
	return raw[:index], raw[index:]
}

func resolveRelativeURL(fromURL, raw string) string {
	base := cleanURLPath(fromURL)
	if base == "" {
		base = "."
	}
	resolved := path.Clean(path.Join(base, raw))
	if resolved == "." {
		return ""
	}
	if strings.HasSuffix(raw, "/") && !strings.HasSuffix(resolved, "/") {
		resolved += "/"
	}
	return resolved
}

// buildTree assembles the sidebar tree. Ancestors of currentPath are always
// expanded; other directories expand while their depth <= expandDepth
// (config view.tree_expand_depth).
func buildTree(model siteModel, currentURL, currentPath string, expandDepth int) *theme.TreeNode {
	root := &theme.TreeNode{Name: "root", IsDir: true, Expanded: true, Href: render.RelTo(currentURL, viewDirURL(""))}
	for _, file := range model.files {
		parts := strings.Split(file.Path, "/")
		node := root
		dirPath := ""
		for i, part := range parts {
			isLast := i == len(parts)-1
			if !isLast {
				if dirPath == "" {
					dirPath = part
				} else {
					dirPath += "/" + part
				}
				child := findChild(node, part, true)
				if child == nil {
					child = &theme.TreeNode{Name: part, Path: dirPath, IsDir: true, Kind: "dir"}
					node.Children = append(node.Children, child)
				}
				child.Href = render.RelTo(currentURL, viewDirURL(dirPath))
				child.Expanded = isAncestorOrSelf(dirPath, currentPath) || i+1 <= expandDepth
				child.Current = currentPath == dirPath
				node = child
				continue
			}
			href := render.RelTo(currentURL, mirrorURL(file.Path))
			if file.Render {
				href = render.RelTo(currentURL, model.browserURLForFile(file))
			}
			node.Children = append(node.Children, &theme.TreeNode{
				Name:    part,
				Path:    file.Path,
				Href:    href,
				Kind:    file.KindName,
				Current: currentPath == file.Path,
			})
		}
	}
	sortTree(root)
	return root
}

func findChild(node *theme.TreeNode, name string, isDir bool) *theme.TreeNode {
	for _, child := range node.Children {
		if child.Name == name && child.IsDir == isDir {
			return child
		}
	}
	return nil
}

func sortTree(node *theme.TreeNode) {
	sort.Slice(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		return a.Name < b.Name
	})
	for _, child := range node.Children {
		sortTree(child)
	}
}

func breadcrumbs(currentURL, repoPath string, dir bool) []theme.Crumb {
	var crumbs []theme.Crumb
	crumbs = append(crumbs, theme.Crumb{Label: "root", Href: render.RelTo(currentURL, viewDirURL(""))})
	if repoPath == "" {
		crumbs[0].Current = true
		return crumbs
	}
	parts := strings.Split(repoPath, "/")
	for i, part := range parts {
		p := strings.Join(parts[:i+1], "/")
		last := i == len(parts)-1
		target := viewDirURL(p)
		if last && !dir {
			target = viewFileURL(p)
		}
		crumbs = append(crumbs, theme.Crumb{
			Label:   part,
			Href:    render.RelTo(currentURL, target),
			Current: last,
		})
	}
	return crumbs
}

func checkRelativeLinks(outDir string, generatedRootIndex, generated404 bool) error {
	var hits []string
	err := filepath.WalkDir(outDir, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(outDir, filePath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !isGeneratedText(rel, generatedRootIndex, generated404) {
			return nil
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		text := string(data)
		for _, needle := range generatedLinkNeedles(rel) {
			if strings.Contains(text, needle) {
				hits = append(hits, rel+": "+needle)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(hits) > 0 {
		sort.Strings(hits)
		return fmt.Errorf("site: generated output contains non-relative links: %s", strings.Join(hits, "; "))
	}
	return nil
}

func generatedLinkNeedles(rel string) []string {
	needles := []string{`href="/`, `src="/`, `src="http://`, `src="https://`}
	if strings.HasSuffix(rel, ".css") {
		needles = append(needles, `url(http://`, `url(https://`, `@import`)
	}
	return needles
}

func isGeneratedText(rel string, generatedRootIndex, generated404 bool) bool {
	return generatedRootIndex && rel == "index.html" ||
		generated404 && rel == "404.html" ||
		strings.HasPrefix(rel, "view/") && strings.HasSuffix(rel, ".html") ||
		strings.HasPrefix(rel, "_assets/") && (strings.HasSuffix(rel, ".css") || strings.HasSuffix(rel, ".js"))
}

func outputHTMLPath(outDir, pageURL string) string {
	rel := outputHTMLRel(pageURL)
	if rel == "index.html" {
		return filepath.Join(outDir, "index.html")
	}
	return filepath.Join(outDir, filepath.FromSlash(rel))
}

func outputHTMLRel(pageURL string) string {
	clean := strings.TrimSuffix(cleanURLPath(pageURL), "/")
	if clean == "" {
		return "index.html"
	}
	return clean + "/index.html"
}

func viewFileURL(repoPath string) string {
	return "view/" + cleanRepoPath(repoPath) + "/"
}

func viewSourceURL(repoPath string) string {
	return viewFileURL(repoPath) + "source/"
}

func viewDirURL(dir string) string {
	dir = cleanRepoPath(dir)
	if dir == "" {
		return "view/"
	}
	return "view/" + dir + "/"
}

func mirrorURL(repoPath string) string {
	return cleanRepoPath(repoPath)
}

func relRoot(currentURL string) string {
	current := cleanURLPath(currentURL)
	if current == "" {
		return ""
	}
	depth := len(strings.Split(current, "/"))
	if depth == 0 {
		return ""
	}
	return strings.Repeat("../", depth)
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

func parentDir(repoPath string) string {
	dir := path.Dir(repoPath)
	if dir == "." {
		return ""
	}
	return dir
}

func isIndexHTMLFile(repoPath string) bool {
	return path.Base(repoPath) == "index.html"
}

func isIndexHTMLDir(dir string) bool {
	return dir != "" && path.Base(dir) == "index.html"
}

func isAncestorOrSelf(dir, current string) bool {
	return dir == current || strings.HasPrefix(current, dir+"/")
}

func kindName(kind render.Kind) string {
	switch kind {
	case render.KindMarkdown:
		return "markdown"
	case render.KindHTML:
		return "html"
	case render.KindCode:
		return "code"
	case render.KindImage:
		return "image"
	default:
		return "binary"
	}
}

func supportsSourcePage(file fileEntry, opts config.FileOptions) bool {
	switch file.Kind {
	case render.KindMarkdown:
		return true
	case render.KindHTML:
		view := strings.TrimSpace(opts.HTML.View)
		return view == "" || view == "embed" || view == "direct"
	default:
		return false
	}
}

func sourceHref(currentURL string, file fileEntry, opts config.FileOptions) string {
	if !supportsSourcePage(file, opts) {
		return ""
	}
	return render.RelTo(currentURL, viewSourceURL(file.Path))
}

func tocPanelMode(mode string) string {
	if mode == "inline" {
		return "inline"
	}
	return "floating"
}

func pageKind(file fileEntry, opts config.FileOptions) string {
	if file.Kind == render.KindHTML {
		view := strings.TrimSpace(opts.HTML.View)
		if view == "" {
			view = "embed"
		}
		return "html-" + view
	}
	return file.KindName
}

func dirTitle(dir string) string {
	if dir == "" {
		return "Repository"
	}
	return path.Base(dir)
}
