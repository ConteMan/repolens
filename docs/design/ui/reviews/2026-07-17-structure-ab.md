# repolens UI 结构 A/B 评审（2026-07-17）

> 结论：以方向 B“持续上下文工作台”为后续结构骨架；仅吸收方向 A 已有对照证据支持的单一主动作、Build 前 Review 门禁和 revision 冲突恢复模式。当前结论来自启发式结构评审，不代替用户测试。

## 交付事实

- 可编辑源文件：`docs/design/ui/prototypes/repolens-ui-explorations.pen`
- 评审导出：`docs/design/ui/reviews/2026-07-17-structure-ab-assets/`
- 顶层画板：18
- reusable components：3
- 视口：1440px、390px
- 布局检查：全文件 `snapshot_layout(maxDepth: 6, problemsOnly: true)` 无问题
- 持久化检查：Pencil CLI Headless 从磁盘独立重开后仍读取到 18 个顶层画板和 3 个 reusable components
- 视觉检查：18 张 PNG 均非空、非纯黑，无可见裁切、重叠或塌陷

本文件仍是低保真结构探索，不是完整设计系统或 React 实现验收基线。Foundations、完整组件状态和最终页面由 Issue #29 继续推进。

## 节点映射

| 画板 | 节点 ID |
|---|---|
| 00 Task and Fixture | `ImzxT` |
| A01 1440 Project Open | `QATTu` |
| A02 1440 Config Edit | `Z19TP` |
| A03 1440 Diff | `RQ7zL` |
| A04 1440 Build | `pHQAY` |
| A05 390 Project Open | `a8UWo` |
| A06 390 Config Edit | `PtSaP` |
| A07 390 Diff | `I5JCu` |
| A08 390 Build | `UaAPV` |
| B01 1440 Project Open | `AL9dP` |
| B02 1440 Config Edit | `S6PnR` |
| B03 1440 Diff | `BRT9u` |
| B04 1440 Build | `h4Djx` |
| B05 390 Project Open | `h7uK4X` |
| B06 390 Config Edit | `rqGNA` |
| B07 390 Diff | `YiNL7` |
| B08 390 Build | `UXIBF` |
| 99 Structure Scorecard | `tlpAw` |

Reusable components：

| 组件 | 节点 ID |
|---|---|
| Component Primary Button | `z49E6I` |
| Component Secondary Button | `EfKG5` |
| Component Text Field | `ktTOU` |

## 状态与范围验收

| 范围 | 1440px | 390px | 证据 |
|---|---|---|---|
| Project Open | 通过 | 通过 | 空、loading、路径错误、打开成功含 4 warnings；绝对路径和本地访问边界可见 |
| Config Edit | 通过 | 通过 | saved、dirty、三类字段错误、三条保序规则、上下移动和删除；`source`、`output`、`access` 仅为只读信任域说明 |
| Diff | 通过 | 通过 | `.repolens.yml`、规范化 YAML、unified diff、无变更、revision 冲突和重新读取 |
| Build | 通过 | 通过 | running、成功含 warning、438 Files、721 Pages、3.842s、完整缓存路径、主题 CSS 失败和最近成功 |

范围外的最近仓库、远程仓库、目录浏览器、preview、serve、部署入口均未保留。移动端方向 B 使用仓库状态条和明确的“分区”菜单触发器，不缩放桌面侧栏。

## 加权评分

评分规则：`1–5 分 ÷ 5 × 权重`。

| 维度 | 权重 | A 评分 | A 加权 | B 评分 | B 加权 |
|---|---:|---:|---:|---:|---:|
| 主任务与下一步清晰度 | 20 | 5 | 20 | 4 | 16 |
| 安全边界与写入影响理解 | 15 | 5 | 15 | 4 | 12 |
| 错误定位与恢复能力 | 20 | 4 | 16 | 5 | 20 |
| 配置查找与规则编辑效率 | 15 | 3 | 9 | 5 | 15 |
| 仓库、dirty、构建上下文连续性 | 10 | 3 | 6 | 5 | 10 |
| 390px 任务可达性 | 10 | 5 | 10 | 4 | 8 |
| 键盘、焦点与阅读顺序可实现性 | 5 | 5 | 5 | 4 | 4 |
| 与 React + Base UI/API 契约适配度 | 5 | 5 | 5 | 4 | 4 |
| **总计** | **100** |  | **86** |  | **89** |

### 方向 A

证据：

- 阶段、下一步和唯一主动作最明确；
- Review 是写入前独立门禁，安全含义直接；
- 390px 可以自然映射为线性阅读和焦点顺序。

风险：

- 熟练用户在编辑、diff、构建之间需要更多页面切换；
- 返回编辑时较难持续看见构建和 revision 上下文；
- 配置规模增长后，步骤内仍可能形成长页面。

### 方向 B

证据：

- 仓库、revision、dirty 和最近构建结果跨分区持续可见；
- 规则列表、选中规则和校验结果同处一个工作上下文；
- 构建失败时仍保留最近成功结果，恢复信息更完整。

风险：

- 首次用户面对的区域更多，需要验证是否理解正确顺序；
- 390px 必须依赖分区菜单和紧凑上下文条；
- 页面必须持续执行“一屏一个主动作”，否则容易退化为等权控制台。

## 选择与边界

后续使用方向 B 的工作台骨架，不直接拼接两套完整结构。只吸收以下 A 模式：

1. 每个分区只有一个视觉最强的主动作；
2. Build 之前必须经过独立 Review，不能从任意配置分区绕过 diff；
3. revision 冲突明确阻止写入，并以“重新读取”作为恢复动作。

暂不把“首次进入时显示完整步骤向导”纳入基线，因为当前没有用户行为证据证明需要双模式。

## 下一证据门禁

1. 用 3–5 次首次用户任务走查验证 1440px 与 390px；
2. 记录首次关键动作、回退次数、错误恢复、完成时间和需要解释次数；
3. 重点重评 B 的主任务清晰度、390px 任务可达性与键盘阅读顺序；
4. 在 Issue #28 收口合同缺口后，由 Issue #29 建立最终 Foundations、组件状态和完整页面；
5. 1024px 与连续断点验证留到最终页面阶段，不把本轮 1440/390 结果外推为已通过。

## Pencil CLI 验证结论

本轮使用 Pencil CLI `0.2.8` 验证了两种模式：

- App Mode：`pencil interactive -a desktop -i <file>` 连接当前桌面文档，执行 `save()` 后明确写入磁盘；
- Headless Mode：`pencil interactive -i <file> -o <temp>` 启动独立编辑器，使用 `get_editor_state`、`batch_get` 和 `snapshot_layout` 验证持久化内容，不保存临时输出即退出。

桌面 MCP 的 `export_nodes` 一次导出 18 个节点时报“wrong .pen file”，按 5/8/5 分组后全部成功。因此导出应采用有界分组，并以每组返回的绝对路径作为成功证据。

官方文档：<https://docs.pencil.dev/for-developers/pencil-cli>
