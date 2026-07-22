# 2026-07-22 最终 Pencil 设计基线评审

## 结论

通过，作为 Issue #30 的 React 实现与视觉回归基线。

唯一可编辑事实源为 `docs/design/ui/prototypes/repolens-ui.pen`。该文件已保存、关闭并重新打开，8 个顶层画板与 28 个 reusable component 均可再次定位；全量 `snapshot_layout(maxDepth: 12, problemsOnly: true)` 返回无布局问题。PNG 仅用于评审和 PR 差异浏览，不替代 `.pen`。

本次结论是合同、状态覆盖、可访问性意图和视觉一致性的启发式评审，不等同于真实用户任务测试。实现完成后仍需按 Issue #30 执行键盘、焦点、连续断点和 3～5 名目标用户任务验证。

## 冻结决策

- 采用方向 B：分区导航、持续仓库上下文与独立的 Diff / Build 状态。
- 只吸收方向 A 的单一主动作、Build 前 Review 和 revision 冲突后重新读取。
- 视觉 token 与现有 React 绿色身份对齐；使用系统字体，不依赖外部字体或 CDN。
- 配置编辑只呈现仓库信任域；`source`、`output`、`access` 只读提示，不进入可写 payload。
- 通用日志尾部、跨页面恢复最近成功结果、目录浏览器、远程仓库和 preview server 均不在本基线内。

## 画板与评审快照

| 画板 | 节点 ID | 覆盖视口与状态 | 快照 |
|---|---|---|---|
| `00 Foundations` | `RRkxu` | token、字体、间距、圆角、焦点、1440/1024/390 | [PNG](2026-07-22-final-baseline-assets/00-foundations-RRkxu.png) |
| `01 Core Components` | `sGnOn` | Button、Field、Badge、Alert、Dialog、Skeleton、桌面/移动壳层与状态矩阵 | [PNG](2026-07-22-final-baseline-assets/01-core-components-sGnOn.png) |
| `02 Project Open` | `mQGrN` | 1440 初始/空值/加载/相对路径错误/读取 warning，1024 已打开；附不可读目录、配置缺失、YAML 解析失败变体 | [PNG](2026-07-22-final-baseline-assets/02-project-open-mQGrN.png) |
| `03 Config Edit` | `oMxTz` | 1440 默认值与来源、dirty、warning、字段错误、有序规则，1024 过渡；附空配置与 saved 变体 | [PNG](2026-07-22-final-baseline-assets/03-config-edit-oMxTz.png) |
| `04 Diff and Write` | `bXQ2L` | 1440 diff dialog、无变更、冲突、权限失败、成功，1024 冲突；附取消与未知/非受控字段影响变体 | [PNG](2026-07-22-final-baseline-assets/04-diff-write-bXQ2L.png) |
| `05 Build` | `I9boV` | 1440 阶段/Stats/warning/路径/会话提示、运行/成功/失败/并发，1024 失败 | [PNG](2026-07-22-final-baseline-assets/05-build-I9boV.png) |
| `06 Narrow / Project and Config` | `mvZOJ` | 390 Project Open、Site、Rule 独立子页 | [PNG](2026-07-22-final-baseline-assets/06-narrow-project-config-mvZOJ.png) |
| `07 Narrow / Diff and Build` | `cgNoV` | 390 Diff、revision 冲突、Build 成功与失败 | [PNG](2026-07-22-final-baseline-assets/07-narrow-diff-build-cgNoV.png) |

## 可复用组件

源文件中冻结 28 个 reusable component，分为：4 个 Button、5 个 Field、3 个 Badge、2 个 Navigation Item、4 个 Alert、Section、Rule、Build、Diff Confirmation Dialog、Project Loading Skeleton，以及桌面和移动端 Header / Sidebar / Sticky Actions。Issue #30 应优先把这些节点映射到项目自有 `components/ui` 与 `components/domain`，再由页面组合；不在业务页面重复拼装 Base UI primitive。

## 关键同型状态映射

下列状态复用已冻结的 Field、Alert、确认弹层和持久状态区，不另造页面结构；节点和可执行文案已写入 `.pen`：

| 画板 | 变体容器节点 | 状态与恢复语义 |
|---|---|---|
| Project Open | `SxRcz` | 非目录或不可读：保留输入并返回路径字段；配置缺失：说明创建要求且保留仓库路径；YAML 解析失败：展示服务端错误并停留在 Project Open |
| Config Edit | `jYRZs` | 空配置：展示有效默认值与 `default` 来源且不静默写回；saved：清除 dirty、更新 revision、保留当前分区与最近成功结果 |
| Diff and Write | `ym2k2` | 取消：关闭确认弹层、保留 dirty 与当前分区、不发起 commit；未知或非受控字段：在确认区说明规范化和丢弃影响 |

## 验证记录

- 顶层节点数：8；名称与 `screen-inventory.md` 完全一致。
- reusable component 数：28；页面引用可从 `.pen` 检查。
- 源文件持久化：保存后关闭重开通过。
- 布局：全量检查无裁切、重叠或意外溢出。
- 视觉复核：8 张 PNG 均非空，长路径、长标题、三条规则、warning、失败值、Stats 和缓存路径可读。
- 范围核对：只呈现当前合同已经实现或冻结的状态；延期能力未被设计成可用入口。

## Issue #30 实现顺序

1. 从 `00 Foundations` 落地语义 token、系统字体、焦点和断点。
2. 从 `01 Core Components` 建立项目组件层与 Base UI 映射。
3. 依次实现 Project Open、Config Edit、Diff/Write、Build，保留现有 API 与状态合同。
4. 落地 390px 导航、Rule 子页、可滚动 diff 和 sticky action。
5. 以本文件的节点 ID 和 PNG 做视觉回归，再执行键盘、焦点、连续断点和真实用户任务测试。
