# 003: Markdown 渲染管线（internal/render）

- 状态：已实现
- 关联：roadmap M2、ADR-002、ADR-003、[design/architecture.md](../design/architecture.md) 渲染管线一节

## 问题

把仓库内 Markdown 渲染为浏览页正文 HTML：GFM、TOC、标题锚点、front-matter、Mermaid，并在 AST 层把指向仓库内文件的相对链接改写为正确的站内路径。

## 行为

1. **goldmark 组装**：`yuin/goldmark` ＋ `extension.GFM` ＋ `go.abhg.dev/goldmark/anchor`（标题锚点）＋ `go.abhg.dev/goldmark/frontmatter` ＋ `go.abhg.dev/goldmark/toc`（提取 TOC 结构，不内联进正文，交模板渲染侧栏/页首）＋ `go.abhg.dev/goldmark/mermaid`（ClientRenderer 模式：输出 `<pre class="mermaid">`，脚本注入由 theme 层负责）。代码块经 chroma 高亮（与 spec 004 共用 highlighter）。GFM 表格由 NodeRenderer 在 `<table>` 外包裹可聚焦的 `.table-scroll` 区域，Header / Row / Cell 仍沿用 goldmark 默认渲染；不额外创建同名 `region` landmark。
2. **标题与 TOC**：`frontmatter_title: true` 时 front-matter 的 `title` 为页面标题，否则取首个 H1，再退化为文件名。TOC 仅在标题数 ≥ 阈值（默认 3）且 `toc: true` 时返回。
3. **链接改写**（ASTTransformer，作用于 Link 与 Image 节点）：
   - 跳过：含 scheme 的绝对 URL、`//` 开头、`#` 开头（页内锚点）、`mailto:` 等；
   - `/` 开头的绝对路径：保持不动（v1 不解释，文档声明）；
   - 相对路径：先与当前文件所在 repo 目录做 `path.Join` 归一化得到目标 repo 路径（越出仓库根的保持不动）；
   - 目标在内容集合中且**有浏览页**（由回调判定，见接口）：Link 改写为当前**浏览页**到目标**浏览页**的相对路径，fragment 保留；
   - 目标在内容集合中但无浏览页（二进制、超限、`render:false`）以及所有 Image：改写为当前浏览页到目标**镜像路径**的相对路径；
   - 目标不在内容集合中：保持不动。
   - 相对路径计算基准：当前页 URL 为 `view/<repo-path>`（文件页）；实现统一用一个 `relTo(fromURL, toURL string) string` 助手，配套充分测试。
4. **安全**：goldmark 默认不渲染原始 HTML；开启 `html.WithUnsafe()`——仓库内容本就是单租户可信内容（与镜像层直出 HTML 一致），文档中声明此立场。

## 接口契约

```go
package render

type MarkdownOptions struct {
    TOC bool; TOCMinHeadings int; Anchors bool
    Mermaid bool; FrontmatterTitle bool
}

type PageRef struct {
    Path string // 当前文件的 repo 相对路径
    // Resolve 判定目标 repo 路径的去向："view"（有浏览页）、
    // "mirror"（仅镜像）、""（不在内容集合，保持原样）
    Resolve func(target string) string
}

type TOCItem struct{ Title, Anchor string; Level int; Children []TOCItem }

type MarkdownResult struct {
    HTML       template.HTML
    Title      string
    TOC        []TOCItem
    HasMermaid bool
    Meta       map[string]any // front-matter 原文
}

func NewMarkdown() *Markdown // 组装一次，可并发复用
func (m *Markdown) Render(src []byte, ref PageRef, opts MarkdownOptions) (MarkdownResult, error)
```

`render` 不导入 `internal/config`，选项结构自持有；site 层负责 config → options 的映射。

## 边界与非目标

- 不解析、不修改用户的 `.html` 文件（ADR-001）；
- 数学公式（math）v1 Out，选项字段可预留但不实现；
- 不做渲染结果缓存（v1 全量构建）。

## 验收

- golden-file 测试：GFM 表格（含长 JSON、URL、token 与可聚焦滚动容器）/ 任务列表、TOC 提取、front-matter 标题、mermaid 代码块输出 `<pre class="mermaid">`、代码块高亮 class 存在；
- 链接改写表驱动测试覆盖行为 3 的每一分支（含 fragment 保留、`../` 越根、图片 vs 链接、深层目录相对路径正确性）；
- 并发 Render 无 data race（`go test -race`）；
- `gofmt` / `go vet` / `go test` 通过。
