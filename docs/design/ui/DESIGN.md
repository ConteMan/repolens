# M6 图形化配置界面设计基线

> 状态：结构方向、CG-01～CG-02 合同与最终 Pencil 设计基线均已冻结（2026-07-22）。Issue #30 已按节点映射完成 React + TypeScript + Base UI 实现与自动化回归；配置语义以 `docs/design/config.md` 为准，跨层状态见 [contract-gaps.md](contract-gaps.md)。

当前交付由 GitHub Milestone [M6 UI design exploration](https://github.com/ConteMan/repolens/milestone/1) 跟踪：[#27](https://github.com/ConteMan/repolens/issues/27) 负责 Pencil 结构探索，[#28](https://github.com/ConteMan/repolens/issues/28) 负责合同收口，[#29](https://github.com/ConteMan/repolens/issues/29) 负责设计系统与完整页面，[#30](https://github.com/ConteMan/repolens/issues/30) 负责 React 实现与设计回归。

## 目标与信息架构

帮助不熟悉终端的维护者完成一条本地、可回退、可解释的路径：输入已存在 Git 工作树的绝对路径 → 编辑仓库可写配置 → 预览 diff → 写入 → 构建 → 查看静态产物结果。第一期不包含目录浏览器或 preview server。

当前实现已完成“打开 → 编辑 → 校验 → diff → 写入 → 构建”的单页闭环，但其布局和视觉不自动升级为最终设计事实。本轮已使用同一套真实 fixture 比较两种结构：

1. 方向 A：Repository / Configuration / Review / Build 四阶段步骤流；
2. 方向 B：分区导航 + 持续仓库上下文 + 独立 diff/build 状态。

结构评审结果为 A 86/100、B 89/100，选择方向 B 作为结构骨架。方向 B 只吸收 A 的单一主动作、Build 前 Review 和 revision 冲突恢复模式，不引入未经用户测试证明的双模式。完整证据见 [2026-07-17 结构 A/B 评审](reviews/2026-07-17-structure-ab.md)。

## 组件策略

采用“业务页面 → 项目自有 `components/ui` → Base UI primitives”的三层结构。Base UI 负责 ARIA、键盘和焦点等交互原语；项目组件负责视觉、Field 组合和错误语义；业务页面不应散落直接拼装 primitive。详细策略见 [component-strategy.md](component-strategy.md)。

优先复用：`ProjectPathField`、字段组与行内校验、`ConfigSourceBadge`、`RuleEditor`、`TrustDomainNotice`、Diff、`BuildOperationPanel`、Warning 列表与结果卡。每个任务状态只保留一个明确主动作。

## 非目标

本期不呈现远程仓库、加密、外部配置、目录浏览器和内嵌 serve；它们不能以“禁用但可点”的形式暗示可用。规则与主题是仓库域的真实 schema，提供结构化编辑；`source`、`output`、`access` 不是仓库域字段，绝不展示为可写控件。Spec 014 新增的输出目录是 Build 页的会话级执行参数，与仓库 `output` 配置明确分离，不进入配置表单或 YAML diff。

## 视觉与原型现状

桌面以 1440px 和 1024px 验收，窄屏以 390px 验收。窄屏需验证分区导航、规则独立编辑、diff 区域内横向滚动和 sticky action 不遮挡内容。路径、glob、diff 使用系统 mono；不得加载外部字体或 CDN。

低保真结构探索已完成 18 个顶层画板，覆盖 A/B 的 1440px 与 390px Project Open、Config Edit、Diff、Build 及结构评分卡。源文件为 `prototypes/repolens-ui-explorations.pen`，对应 PNG 位于 `reviews/2026-07-17-structure-ab-assets/`。该文件继续作为结构决策证据，不作为最终视觉事实源。

最终视觉事实源为 `prototypes/repolens-ui.pen`。文件包含与 [screen-inventory.md](screen-inventory.md) 一致的 11 个顶层画板、28 个 reusable component，以及 1440px、1024px 和 390px 的关键页面与状态。原 M6 基线已完成保存后关闭重开验证；Spec 014 新增画板均通过布局检查与 PNG 复核。节点、视口、状态和 PNG 映射见 [2026-07-22 最终设计基线评审](reviews/2026-07-22-final-baseline.md) 与 [UI 会话级输出设计基线评审](reviews/2026-07-22-session-output.md)。

Issue #29 已完成最终 `.pen`、Foundations、组件状态、选定方向页面与窄屏状态的设计交付。Issue #30 已直接按最终 `.pen` 的节点、变量和组件映射完成方向 B 工作区、桌面/窄屏布局、Diff/Write、Build 与恢复状态实现；回归证据见 [2026-07-22 实现回归](reviews/2026-07-22-implementation-regression.md)。Issue #46 的 Spec 014 会话级输出以新增的 `08`～`10` 画板为实现基线。真实用户任务验证仍是后续验收活动，不能由启发式设计评审或自动化浏览器回归替代。具体流程和退出门禁见 [exploration-brief.md](exploration-brief.md) 与 [screen-inventory.md](screen-inventory.md)。
