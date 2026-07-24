# Roadmap

> 里程碑叙事在此维护；具体任务与进度以 GitHub Issues / Milestones 为准。修改 v1 In / Out 边界需维护者确认。

## v1 范围

### In

- `repolens build <repo|path>`：git 源（clone / fetch / ls-tree）与本地路径源
- 双层输出：镜像层 ＋ `/view/` 浏览层，前缀式 URL，全相对链接
- Markdown 渲染：GFM、TOC、标题锚点、front-matter、Mermaid（客户端懒加载）、AST 层链接改写
- 代码文件浏览页：chroma 高亮 ＋ 行号
- HTML 文件：镜像原样 ＋ iframe 浏览页（embed / direct / source 三模式）
- 文件树侧栏（图标、当前高亮、状态保持）、目录页（README 优先，listing 兜底）
- git 元数据：每页最后修改时间（一次性 log 建映射）
- 配置：双信任域、级联规则、主题三级定制（CSS 变量 / 追加 CSS / 模板覆盖）
- Agent 视图：llms.txt、llms-full.txt、index.json、`rel=alternate`
- 访问控制：noindex 默认开、零外部请求约束、部署平台认证指南文档
- `repolens serve`：本地预览 ＋ fsnotify 重建（`--worktree` 预览未提交内容）
- CI 质量门禁 ＋ dogfood（用 repolens 构建本仓 docs/ 并部署）

## v1.x 范围（2026-07-06 维护者确认方向）

> 状态：已按 M7 → M5 → M6 → M8 顺序完成（2026-07-22）。后续范围通过 GitHub Issues / Milestones 重新立项。

### In

- 多平台分发与升级：GoReleaser 流水线、Homebrew tap、Windows 单 exe ＋ Scoop、`repolens upgrade`（[009](specs/009-release-distribution.md)）
- 文件树混合形态：固定侧栏可收起 ＋ 浮动覆盖层，窄屏自动浮动（[010](specs/010-hybrid-tree.md)）
- 顶部工具栏：树/TOC 开关、前进后退、缩放、布局宽度、源码视图（`view/<path>/source/`）、下载、搜索入口；pjax 导航（[011](specs/011-toolbar-and-pjax.md)）
- 站内搜索：文件名 ＋ 标题级构建期索引，客户端检索，零外部请求（[012](specs/012-site-search.md)）
- 图形化管理界面：`repolens ui` 本地 Web GUI——选目录、可视化配置写回 YAML、构建与预览（[013](specs/013-config-ui.md)）
- UI 会话级构建输出：默认缓存与自定义绝对目录二选一，保持工作树语义与输出所有权检查，不写入仓库配置（[014](specs/014-ui-session-output.md)）

### Out（v2 及以后，实现前需先修订本文档）

- 全文搜索（v1.x 仅文件名＋标题级；全文仍预定 pagefind 后处理方案）
- 格式导出（PDF / docx / epub）
- 原生桌面壳（Wails/Tauri——Web UI 验证后再议）
- 客户端加密（PageCrypt 思路，设计已在 ADR-005 预留）
- 多仓库聚合（v1 单仓，聚合交给部署层拼路径）
- 增量构建、数学公式渲染、Mermaid 之外的图表引擎
- 主题市场 / 插件系统、参与部署流程

## 里程碑

| 里程碑 | 内容 | Specs | 验收 | 状态 |
|---|---|---|---|---|
| M1 骨架 | 仓库规范、设计文档、CLI 脚手架、CI 门禁 | — | `repolens version` 可用，CI 绿 | 已完成 |
| M2 核心管线 | source（git）→ config（双域级联）→ render（md/code）→ site（双层输出） | [001–005](specs/README.md) | 对本仓 build 出可浏览的 dist/ | 已完成 |
| M3 浏览层 | 文件树、目录页、HTML iframe、主题、增强层 JS、serve | [006–007](specs/README.md) | docu.md 级浏览体验 | 已完成 |
| M4 收尾 | Agent 视图、git 元数据、平台部署指南、dogfood CI、v1.0.0 发布 | [008](specs/README.md) ＋ 收尾项 | 文档站上线，tag v1.0.0 | 已完成（2026-07-05） |
| M7 分发 | GoReleaser、brew tap、Windows 单 exe/Scoop、upgrade、version 修复 | [009](specs/README.md) | 三平台免 Go 环境安装实测通过 | 已完成 |
| M5 浏览体验 | 混合树、顶部工具栏、pjax、源码视图、站内搜索 | [010–012](specs/README.md) | 维护者对照 docu.md 交互验收 | 已完成 |
| M6 图形界面 | `repolens ui`：项目选择、可视化配置、构建预览闭环 | [013](specs/README.md) | 全新用户零终端完成建站路径 | 已完成 |
| M8 UI 输出 | `repolens ui` 会话级缓存/自定义输出目录、安全覆盖与原子发布 | [014](specs/README.md) | 工作树构建到用户指定本地目录且不破坏未知内容 | 已完成（2026-07-22） |
