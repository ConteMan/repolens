# 功能规格（Specs）

v1 拆分为 8 份可独立实现的 spec（001–008，已全部实现）；v1.x 新增 009–015（M5–M9，见 roadmap），每份含行为定义、**接口契约**与验收标准。实现 Agent 以单份 spec 为工作单元，无需本仓之外的上下文。

## 索引

| 编号 | 标题 | 包 | 里程碑 | 状态 |
|---|---|---|---|---|
| [001](001-git-source.md) | Git 内容源 | internal/source | M2 | 已实现 |
| [002](002-config-loading.md) | 配置加载与规则级联 | internal/config | M2 | 已实现 |
| [003](003-markdown-pipeline.md) | Markdown 渲染管线 | internal/render | M2 | 已实现 |
| [004](004-code-and-file-classification.md) | 文件分类与代码渲染 | internal/render | M2 | 已实现 |
| [005](005-site-assembly.md) | 站点组装与 build 命令 | internal/site, cli | M2 | 已实现 |
| [006](006-theme-and-templates.md) | 主题、模板与增强层 | internal/theme | M3 | 已实现 |
| [007](007-serve.md) | 本地预览 serve | internal/server | M3 | 已实现 |
| [008](008-agent-surface.md) | Agent 视图 | internal/site | M4 | 已实现 |
| [009](009-release-distribution.md) | 发布流水线与分发 | .github, cli | M7 | 已实现 |
| [010](010-hybrid-tree.md) | 文件树混合形态 | internal/theme | M5 | 已实现 |
| [011](011-toolbar-and-pjax.md) | 顶部工具栏与 pjax | internal/theme, site | M5 | 已实现 |
| [012](012-site-search.md) | 站内搜索 | internal/site, theme | M5 | 已实现 |
| [013](013-config-ui.md) | 图形化管理界面 | internal/ui | M6 | 已实现 |
| [014](014-ui-session-output.md) | UI 会话级构建输出目录 | internal/ui | M8 | 已实现 |
| [015](015-resizable-fixed-tree.md) | 桌面固定文件树侧栏调宽 | internal/theme | M9 | 已确认 |

## 依赖与实现顺序

```
001 source ─┐
002 config ─┼─→ 005 site ←─ 006 theme
003 markdown┤       ↓
004 code   ─┘   007 serve, 008 agent
```

- **001–004 相互独立，可并行**（003/004 同包不同文件，注意协调）；
- 005 是集成点，依赖 001–004 与 006 的接口（006 可先交付最小模板集供 005 集成，视觉打磨后置）；
- 007、008 依赖 005 完成。

## 实现 Agent 工作规则

1. 开工前通读：本 spec ＋ [AGENTS.md](../../AGENTS.md) ＋ spec 中"关联"所列文档；
2. **接口契约是包间合同**：函数签名与类型允许实现期微调，但任何调整必须回写进 spec（同一 PR），并检查依赖方 spec 是否受影响；
3. spec 与 design/ADR 冲突、或发现 spec 遗漏关键行为时：**停下来在 PR / Issue 中报告**，不自行拍板；
4. 完成定义 = spec 全部验收项通过 ＋ 质量门禁绿 ＋ spec 状态改为"已实现"（同一 PR）；
5. 一份 spec 一个 PR，commit 遵循 Conventional Commits。
6. UI 探索、设计基线和实现分阶段验收；探索稿不直接产生 `design-gap`。
7. UI 假设必须关联已实现的接口合同，或明确引用 `docs/design/ui/contract-gaps.md` 中的缺口。
8. Pencil 设计验收必须包含：可关闭重开的 `.pen`、节点 ID、固定 fixture、桌面/窄屏证据、布局检查和逐张导出检查。
9. 只有评审冻结的设计基线才能约束实现；实现 PR 必须引用对应 `.pen` 节点与状态。

新增功能仍复制 [template.md](template.md)，编号递增。
