# 功能规格（Specs）

v1 拆分为 8 份可独立实现的 spec，每份含行为定义、**接口契约**与验收标准。实现 Agent 以单份 spec 为工作单元，无需本仓之外的上下文。

## 索引

| 编号 | 标题 | 包 | 里程碑 | 状态 |
|---|---|---|---|---|
| [001](001-git-source.md) | Git 内容源 | internal/source | M2 | 已实现 |
| [002](002-config-loading.md) | 配置加载与规则级联 | internal/config | M2 | 已实现 |
| [003](003-markdown-pipeline.md) | Markdown 渲染管线 | internal/render | M2 | 已确认 |
| [004](004-code-and-file-classification.md) | 文件分类与代码渲染 | internal/render | M2 | 已确认 |
| [005](005-site-assembly.md) | 站点组装与 build 命令 | internal/site, cli | M2 | 已确认 |
| [006](006-theme-and-templates.md) | 主题、模板与增强层 | internal/theme | M3 | 已确认 |
| [007](007-serve.md) | 本地预览 serve | internal/server | M3 | 已确认 |
| [008](008-agent-surface.md) | Agent 视图 | internal/site | M4 | 已确认 |

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

新增功能仍复制 [template.md](template.md)，编号递增。
