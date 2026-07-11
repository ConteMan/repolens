# M6 图形化配置界面设计基线

> 状态：设计中（2026-07-11）。本文件是 Spec 013 第一期的体验事实源；配置语义仍以 `docs/design/config.md` 为准。

## 目标与信息架构

帮助不熟悉终端的维护者完成一条本地、可回退、可解释的路径：输入已存在 Git 工作树的绝对路径 → 编辑仓库可写配置 → 预览 diff → 写入 → 构建 → 查看静态产物结果。第一期不包含目录浏览器或 preview server。

桌面采用两栏：左侧为项目路径与步骤，右侧为当前步骤的单一主任务。窄屏改为单栏步骤流，构建日志放到可展开区。主色沿用 repolens 的蓝色；危险语义只用于写入确认与不可恢复错误。

## 组件策略

复用语义组件：`ProjectPathField`、路径面包屑、步骤条、字段组、行内校验、`ConfigSourceBadge`、`RuleEditor`、`TrustDomainNotice`、Diff、`BuildOperationPanel`、Warning 列表与结果卡。每屏仅有一个主按钮：选择项目、继续到确认、确认写入或构建。

## 非目标

本期不呈现远程仓库、加密、外部配置、目录浏览器和内嵌 serve；它们不能以“禁用但可点”的形式暗示可用。规则与主题是仓库域的真实 schema，提供结构化编辑；`source`、`output`、`access` 不是仓库域字段，绝不展示为可写控件。

## 视觉基线与原型限制

桌面画板为 1440px，内容最大 1184px；窄屏为 390px，section 导航进入抽屉、规则编辑进入子页、YAML diff 保留代码区横向滚动。使用 `#f6f8fa` 背景、`#ffffff` 表面、`#0969da` 主操作、`#1a7f37` 成功、`#9a6700` 警告、`#cf222e` 阻断，4px 间距基线、36px 控件高、6px 圆角和系统 mono 的路径/glob/diff 文本。

本轮检测到 `/Applications/Pencil.app`，但 `orca computer capabilities --json` 返回 `runtime_unavailable`（Orca 为 `stale_bootstrap`），没有可调用的 Pencil 运行时；对目标文件的三次 Pencil `batch_design` 均返回“user cancelled MCP tool call”。依照工作流，没有创建或伪造 `prototypes/repolens-ui.pen`、PNG 或节点 ID；[评审记录](reviews/2026-07-11-m6-structure.md) 给出 Mermaid/ASCII 替代和恢复后的唯一 `.pen` 交付门槛。
