# 006: 主题、模板与增强层（internal/theme）

- 状态：已确认
- 关联：roadmap M3、ADR-002、ADR-003

## 问题

浏览层的全部视觉与交互：内置模板集、CSS、文件树增强 JS、Mermaid 懒加载、三级定制机制。交互形态参照 docu.md（常驻文件树 ＋ 内容区 ＋ 顶栏）。

## 行为

1. **模板集**（html/template，go:embed，模板名即公开 API，宁缺毋滥）：
   - `layout` —— 页面骨架：顶栏（站点标题、到镜像原文件链接）、文件树侧栏、内容区、页脚（git 最后修改时间、commit 短 hash）；
   - `page` —— 文件页正文（Markdown / Code / HTML iframe / Image / Binary 五种形态的分支在此）；
   - `dirlist` —— 目录页（README 正文来自 `PageData.Body`，子项表格来自 `PageData.DirEntries`）；
   - `tree` —— 文件树侧栏（递归 partial：目录可折叠、类型图标、当前项高亮、祖先默认展开、其余按 `tree_expand_depth`）。
2. **CSS**：手写单文件 `_assets/site.css`，顶部集中定义 CSS 变量（颜色、字号、`--sidebar-width` 等）；`theme.vars` 在页面 `<head>` 内联 `:root { … }` 覆盖；支持 `prefers-color-scheme` 深色模式（同样走变量）。chroma 样式表由 spec 004 的 `StylesCSS` 生成为 `_assets/chroma.css`（亮/暗两份，media query 切换）。
3. **增强 JS**：手写单文件 `_assets/site.js`（无框架、无打包器，目标 ~200 行）：
   - 文件树折叠状态持久化（sessionStorage，key 为目录路径）；
   - 树滚动位置保持；
   - 可选 pjax：拦截站内浏览页链接，fetch 目标页并替换内容区 ＋ history.pushState，失败回退整页跳转。v1 允许先不做 pjax，树状态持久化必须有。
   - 无 JS 时一切可读可导航（`<details>` 或默认展开态兜底）。
4. **Mermaid**：vendor `mermaid.min.js`（UMD 构建，版本写入文件头注释与 CHANGELOG）到 `_assets/`；仅 `HasMermaid` 的页面注入 `<script defer>` 与初始化调用。
5. **三级定制**：
   - `theme.vars` → 内联 `:root` 覆盖（site 层传入）；
   - `theme.css` → 复制该文件到 `_assets/custom.css` 并在 layout 末尾引入；
   - `theme.templates` → 目录中与内置同名的模板覆盖内置（`template.ParseFS` 后再 Parse 覆盖）。
6. **资源输出**：`Renderer.WriteAssets(outDir)` 把全部静态资源写到 `dist/_assets/`；页面对资源的引用一律相对路径（由 site 层传入的深度前缀计算）。

## 接口契约

```go
package theme

type Renderer struct{ /* 解析后的模板集 ＋ 资源 */ }

// overrideDir / customCSS 可为空；vars 为 theme.vars
func New(overrideDir, customCSS string, vars map[string]string) (*Renderer, error)

type DirEntry struct {
    Name, Path, Href, Kind string // Href 为相对当前目录页的链接
    Size                   int64
    LastCommit             *source.Commit
    IsDir                  bool
}

type PageData struct {
    Title, SiteTitle string
    RelRoot          string   // 当前页到站点根的相对前缀，如 "../../"
    Breadcrumbs      []Crumb
    Tree             *TreeNode // 全树，含展开/高亮标记
    Kind             string    // markdown|html-embed|html-direct|code|image|binary|dir
    Body             template.HTML
    TOC              []render.TOCItem
    MirrorHref       string // 相对路径
    LastCommit       *source.Commit
    HasMermaid, NoIndex bool
    HeadExtra        template.HTML // rel=alternate 等，site 层拼装
    DirEntries       []DirEntry    // 仅目录页使用；由 dirlist 模板渲染子项表格
}

func (r *Renderer) Page(w io.Writer, d PageData) error
func (r *Renderer) WriteAssets(outDir string) error
```

## 边界与非目标

- 不做多主题内置（一套默认主题 ＋ 三级定制）；
- 不引入图标字体/外部字体（系统字体栈；文件类型图标用内联 SVG sprite）；
- 搜索 UI v1 不做（顶栏预留位置即可）。

## 验收

- 模板渲染单测：五种 Kind ＋ 目录页的 golden HTML；vars 覆盖生效；模板覆盖生效；
- 产物中无任何外部 origin 引用（配合 spec 005 自检）；
- 禁 JS 环境下手工验证树可导航、内容可读；
- 视觉验收：对本仓库构建，维护者认可默认主题观感（docu.md 级整洁度）；
- `gofmt` / `go vet` / `go test` 通过。
