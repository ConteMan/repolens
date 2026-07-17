# M6 结构评审（2026-07-11）

选择：单项目、路径输入、仓库域字段编辑、写入前 Diff、构建结果页。构建输出进入仓库外缓存根，不承担预览服务生命周期。

放弃：服务端目录浏览器、远程仓库、外部可信配置、内嵌 serve 和新标签预览。规则与主题仍属于仓库可写 schema，因此采用结构化编辑，不应被错误地延期；`access.noindex` 必须延期，因为现有 `config.Load` 已证明仓库内 `access` 会被忽略。

Pencil 阻塞复盘（2026-07-11）：旧环境曾把问题归因于 Orca runtime；后续复查发现 Pencil 停留在 `New File` 模板选择器时没有活动文档，MCP 会提示需要先打开文件。这个发现解释了工具调用失败，但**不等于源文件已成功持久化**。

恢复步骤：先在 Pencil 中打开任意已有 `.pen`，使 MCP 获得活动文档；随后可在 `batch_design` 显式传入目标路径创建并保存新文件，不需要 Orca。Pencil 的 Save As 对话框若未正确切换目录，可能先保存到 Documents；将该 Pencil 生成的文件移动至目标目录后，MCP 可在目标路径继续读写。

恢复尝试产生了若干 PNG，但当前磁盘上的 `docs/design/ui/prototypes/repolens-ui.pen` 实际只包含一个 800×600 空白 frame `bi8Au`，没有评审文字曾列出的 Components、Desktop 或 Narrow 节点。因此无法从源文件核实 `Hday5`、`NuC1D`、`L2W5X` 的画板映射或布局检查结果；该 `.pen` 不是有效事实源，本轮不提交它。

评审资产排除：`kJ6aB.png`、`XKVIY.png`、`JD91b.png` 与 `zbGBK.png` 经逐张检查为空白或纯黑，不代表有效画板，均已删除。`bi8Au.png`、`Hday5.png`、`NuC1D.png` 与 `L2W5X.png` 可作为历史线框证据，但不能用于实施验收：`Hday5` 缺少真实控件和值，`NuC1D` 把 diff 与构建合并成一步，`L2W5X` 的菜单和规则子页尚无实现对应。

下一步不继续润色这些 PNG。先按 `exploration-brief.md` 使用同一 fixture 完成结构 A/B，再依 `screen-inventory.md` 建立 Foundations、Core Components、Project Open 三个 P0 画板。字段错误、open warnings/effective defaults/source、日志尾部和最近成功恢复范围先按 `contract-gaps.md` 收口，原型不得提前发明合同。

审计证据：React + TypeScript + Base UI 迁移已由 PR #26 合并；当前实现与设计差距以 `internal/ui/frontend`、`internal/ui`、Playwright 测试及 `contract-gaps.md` 为准。本评审只校准设计证据，不修改产品代码。
