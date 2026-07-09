# 005: 站点组装与 build 命令（internal/site ＋ internal/cli）

- 状态：已实现
- 关联：roadmap M2、ADR-001、ADR-005、specs 001–004、006

## 问题

把 source（内容集合）、config（规则求值）、render（渲染结果）、theme（模板）串成完整构建：产出镜像层 ＋ 浏览层 ＋ robots.txt，并接通 `repolens build`。这是 M2 的集成点与验收载体。

## 行为

1. **构建流程**：`source.Open` → `config.Load`（树根找 `.repolens.yml`）→ 遍历 `tree.Files`，对每个非 `Ignored` 文件：
   - 复制到镜像层 `dist/<path>`（逐字节，保留可执行位即可）；
   - `OptionsFor(path)` ＋ `Classify` 决定浏览页形态，渲染到 `dist/view/<path>`（文件页为 `view/<path>` 本身输出 HTML 内容——静态托管对无名扩展的处理不可控，因此**文件浏览页输出为 `view/<path>/index.html`，URL 以 `view/<path>/` 访问**；目录页为 `view/<dir>/index.html`）。
2. **浏览页形态**（Kind → 模板数据）：
   - Markdown：spec 003 渲染正文 ＋ TOC；
   - HTML：按 `html.view`——`embed`：iframe 指向镜像路径（相对），带"全屏打开"工具栏；`direct`：浏览页仅工具栏 ＋ 跳转链接；`source`：按 Code 展示源码。工具栏的"查看源码"入口 v1 缓行——它需要独立的源码子页面 URL（如 `view/<path>/source/`），URL 结构扩展留给后续与 spec 008 一并权衡（实现时确认，2026-07-04）；
   - Code：spec 004 高亮正文；Image：`<img>` 预览；Binary：下载页（名称 / 大小 / 镜像链接）。
   - `render:false` 的文件：只镜像，文件树中直接链到镜像路径。
3. **目录页**：每个目录（含根）生成 `view/<dir>/index.html`：存在 README（匹配顺序 `README.md`、`readme.md`，大小写完全匹配这两种）则渲染其正文，页尾附子项列表；否则仅列表（名称、Kind、大小、git 最后修改时间，目录在前按字典序）。
   - **`index.html` 冲突规则**（Issue #9，2026-07-05 确认）：目录含名为 `index.html` 的文件时，该文件**不生成独立浏览页**（其页面目录 `view/<dir>/index.html/` 与目录页文件同路径，文件系统上不可共存）；沿用 Web 惯例，其浏览形态按 `html.view` 模式**合并进目录页正文**（embed 工具栏＋iframe / direct 工具栏 / source 源码），页尾仍附子项列表；文件树与列表中该文件链接到 `view/<dir>/`。README 与 `index.html` 同在时 `index.html` 优先承载目录页正文，README 仍保留独立文件页。`render:false` 的 `index.html` 照常只镜像，目录页回退 README/列表。仓库**目录**本身名为 `index.html` 的同类冲突无法合并解决：构建报错并列出两个冲突路径（参照 spec 001 大小写冲突先例）。
4. **站点框架数据**（传给 theme 模板，契约见 spec 006）：完整文件树（当前路径的祖先展开、当前项高亮）、面包屑、页面标题、git 元信息、到镜像层原文件的相对链接、`rel=alternate` head 标签。
5. **相对链接不变量**：所有生成页面中的 href/src（含树、面包屑、CSS/JS 资源 `_assets/`）一律相对路径；**禁止**以 `/` 开头。深度由页面 URL 计算。构建后自检：扫描产物中 `href="/`、`src="/`，以及**会发起资源加载的外部引用**（`src="http(s)://`、CSS `url(http(s)://)`），命中即构建失败（零外部请求约束的机器化）。普通超链接 `<a href="http(s)://">` 允许——仓库文档本就含正常外部链接，导航跳转不违反零外部请求约束（实现时确认，2026-07-04）。
6. **根与杂项**：`dist/index.html` 为 meta-refresh ＋ 链接指向 `view/`（静态托管重定向不可移植）；例外：仓库根目录本身含 `index.html` 文件时**镜像优先**，不生成跳转页（镜像层逐字节不变量高于跳转便利；该镜像文件不纳入相对链接自检——它是用户内容非生成产物。Issue #9 连带确认，2026-07-05）；`access.noindex: true` 时输出 robots.txt（`Disallow: /`）并在每页注入 `<meta name="robots" content="noindex">`。
   - **`dist/404.html`**（2026-07-09 补充）：站点根恒生成自包含 not-found 页（无站内资源引用、无站内链接——它会被任意深度的未命中路径命中；恒带 noindex；文案随 `site.language` zh/en）。动机：Cloudflare Pages 等托管在缺少 404.html 时把未命中路径回退成根 index.html，与跳转页的相对 `view/` 叠加产生无限重定向（跨仓相对链接实测踩坑）。仓库根自带 `404.html` 时镜像优先、不生成（同 index.html 例外规则）。
7. **build 命令接线**：`repolens build [repo|path]`，flags 与现有脚手架一致（`--config` / `--ref` / `-o`）。输出目录不存在则创建；已存在时清空重建（仅当目录含上次构建的哨兵文件 `.repolens-build` 才允许清空，否则报错拒绝——防误删）。结束打印统计（文件数、页面数、耗时、Warning 列表）。

## 接口契约

```go
package site

type Builder struct{ /* 持有 render.Markdown、theme.Renderer */ }

type Stats struct {
    Files, Pages int
    Duration     time.Duration
    Warnings     []config.Warning
}

func NewBuilder(cfg *config.Config, theme *theme.Renderer) *Builder
func (b *Builder) Build(ctx context.Context, tree *source.Tree, outDir string) (Stats, error)
```

`internal/cli` 只做参数解析与进度输出，业务全部在 site。

## 边界与非目标

- llms.txt / index.json 属 spec 008，本 spec 只留生成钩子位置；
- 无增量构建、无并发管线优化（正确性优先，接口不排斥后续并行化）；
- 不处理 `output.base_path`（全相对链接下 v1 无需实现，字段保留）。

## 验收

- 端到端测试：对 testdata 固定小仓库（含 md/html/code/图片/二进制/子目录 README/`.repolens.yml`）构建，断言产物结构、镜像逐字节一致、目录页两种形态、iframe 三模式、`render:false` 行为；
- 相对链接自检测试：人为注入绝对链接的模板必须导致构建失败；
- **dogfood 验收（M2 门槛）**：`repolens build .` 对本仓库产出 dist/，`python3 -m http.server` 下人工可浏览：README 首页、docs 目录页、ADR 间跳转、Go 代码高亮；
- `gofmt` / `go vet` / `go test` 通过。
