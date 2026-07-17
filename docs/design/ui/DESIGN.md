# M6 图形化配置界面设计基线

> 状态：重新校准中（2026-07-17）。React + TypeScript + Base UI 已成为实现基线；体验决策仍在结构探索阶段。配置语义以 `docs/design/config.md` 为准，跨层缺口见 [contract-gaps.md](contract-gaps.md)。

当前交付由 GitHub Milestone [M6 UI design exploration](https://github.com/ConteMan/repolens/milestone/1) 跟踪：[#27](https://github.com/ConteMan/repolens/issues/27) 负责 Pencil 结构探索，[#28](https://github.com/ConteMan/repolens/issues/28) 负责合同收口，[#29](https://github.com/ConteMan/repolens/issues/29) 负责设计系统与完整页面，[#30](https://github.com/ConteMan/repolens/issues/30) 负责 React 实现与设计回归。

## 目标与信息架构

帮助不熟悉终端的维护者完成一条本地、可回退、可解释的路径：输入已存在 Git 工作树的绝对路径 → 编辑仓库可写配置 → 预览 diff → 写入 → 构建 → 查看静态产物结果。第一期不包含目录浏览器或 preview server。

当前实现已完成“打开 → 编辑 → 校验 → diff → 写入 → 构建”的单页闭环，但其布局和视觉不自动升级为最终设计事实。下一轮使用同一套真实 fixture 比较两种结构：

1. 分区侧栏 + 单页配置工作区 + 独立 diff/build 状态；
2. Repository / Configuration / Review / Build 四阶段步骤流。

结构评审以任务完成、错误恢复、真实数据密度和窄屏可达性收口，不以“更像现代 dashboard”作为理由。

## 组件策略

采用“业务页面 → 项目自有 `components/ui` → Base UI primitives”的三层结构。Base UI 负责 ARIA、键盘和焦点等交互原语；项目组件负责视觉、Field 组合和错误语义；业务页面不应散落直接拼装 primitive。详细策略见 [component-strategy.md](component-strategy.md)。

优先复用：`ProjectPathField`、字段组与行内校验、`ConfigSourceBadge`、`RuleEditor`、`TrustDomainNotice`、Diff、`BuildOperationPanel`、Warning 列表与结果卡。每个任务状态只保留一个明确主动作。

## 非目标

本期不呈现远程仓库、加密、外部配置、目录浏览器和内嵌 serve；它们不能以“禁用但可点”的形式暗示可用。规则与主题是仓库域的真实 schema，提供结构化编辑；`source`、`output`、`access` 不是仓库域字段，绝不展示为可写控件。

## 视觉与原型现状

桌面以 1440px 和 1280px 验收，窄屏以 390px 验收。窄屏需验证分区导航、规则独立编辑、diff 区域内横向滚动和 sticky action 不遮挡内容。路径、glob、diff 使用系统 mono；不得加载外部字体或 CDN。

蓝色旧基线与当前 React 实现的绿色方向尚未完成同任务对比，因此颜色、圆角和最大内容宽度暂不锁定。语义 token、组件状态和实现映射应先于页面视觉润色。

当前磁盘上的 `prototypes/repolens-ui.pen` 只包含一个 800×600 空白 frame，不能作为可编辑视觉事实源，也不能证明评审 PNG 来自已持久化源文件。`Hday5`、`NuC1D`、`L2W5X` 和 `bi8Au` 仅作为历史线框证据：它们缺少完整控件状态，且部分流程与当前实现不一致。

下一轮必须先建立独立结构探索，再创建干净的最终 `.pen`。交付成立需同时满足：源文件含非空顶层画板、reusable component/ref 可检查、布局检查通过、PNG 与节点映射一致。具体流程和退出门禁见 [exploration-brief.md](exploration-brief.md) 与 [screen-inventory.md](screen-inventory.md)。
