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

### Out（v2 及以后，实现前需先修订本文档）

- 全文搜索（预定 pagefind 后处理方案）
- 客户端加密（PageCrypt 思路，设计已在 ADR-005 预留）
- 多仓库聚合（v1 单仓，聚合交给部署层拼路径）
- 增量构建、数学公式渲染、Mermaid 之外的图表引擎
- 主题市场 / 插件系统、参与部署流程

## 里程碑

| 里程碑 | 内容 | 验收 |
|---|---|---|
| M1 骨架 | 仓库规范、设计文档、CLI 脚手架、CI 门禁 | `repolens version` 可用，CI 绿 |
| M2 核心管线 | source（git）→ config（双域级联）→ render（md/code）→ site（双层输出） | 对本仓 build 出可浏览的 dist/ |
| M3 浏览层 | 文件树、目录页、HTML iframe、主题、增强层 JS、serve | docu.md 级浏览体验 |
| M4 收尾 | Agent 视图、git 元数据、平台部署指南、dogfood CI、v1.0.0 发布 | 文档站上线，tag v1.0.0 |
