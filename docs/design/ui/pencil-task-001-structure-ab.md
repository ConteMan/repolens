# Pencil Task 001：repolens UI 结构 A/B 探索

> 类型：低保真结构探索
> 执行方式：单一 Pencil 编辑者串行完成
> 上游 brief：`docs/design/ui/exploration-brief.md`

## 1. 目标

使用完全相同的用户任务、fixture、状态和视口，对比两种 `repolens ui` 结构：

- **方向 A：引导式任务流**——`Project Open → Config Edit → Diff → Build`，每一步只突出当前主任务。
- **方向 B：持续上下文工作台**——仓库与保存状态持续可见，配置分区可快速跳转，校验、写入和构建状态保留在工作上下文中。

本任务只回答以下结构问题：

1. 首次维护者能否理解当前所处阶段、下一步和安全边界？
2. 熟练维护者能否快速定位配置、检查修改并重复构建？
3. 路径错误、字段错误、revision 冲突和构建失败是否有明确恢复动作？
4. 同一信息架构在 1440px 和 390px 下是否保持任务可达？

本任务不是视觉定稿，不先建设完整设计系统，不评价品牌风格、动效或装饰精度。

## 2. 输入文件

编辑者开始前必须读取：

1. `docs/design/ui/exploration-brief.md`——本任务的阶段门禁、A/B 假设和验收原则。
2. `docs/specs/013-config-ui.md`——产品范围、接口契约、非目标与安全边界。
3. `docs/design/ui/flows/m6-config-ui.md`——主路径和状态矩阵。
4. `docs/design/ui/fixtures.md`——真实规模内容。
5. `internal/ui/frontend/src/App.tsx`——当前已实现能力，不代表目标结构。
6. `internal/ui/frontend/src/types.ts`——仓库可写配置字段。

历史资产只可用于了解旧假设：

- `docs/design/ui/reviews/2026-07-11-assets/*.png` 是历史线框，不是当前视觉事实源。
- `docs/design/ui/prototypes/repolens-ui.pen` 当前只有空白 frame，不是有效原型事实源。

## 3. 输出文件与编辑边界

唯一交付文件：

`docs/design/ui/prototypes/repolens-ui-explorations.pen`

要求：

- 必须新建该文件。
- **不得覆盖、移动、重命名或修改** `docs/design/ui/prototypes/repolens-ui.pen`。
- 开始 MCP 编辑前，必须先在 Pencil 中创建并打开目标文件，确认 `get_editor_state` 返回的活动路径就是目标路径。向不存在的路径传 `batch_design.filePath` 不等于创建或切换文档，可能仍写入当前活动文件。
- 不得修改代码、spec、brief、fixture、评审文档、历史 PNG 或其他仓库文件。
- 同一时刻只能有一个编辑者写入目标 `.pen`；其他 Agent 只能只读审查。
- 每完成一个稳定批次立即保存；任务结束前关闭并重新打开文件，确认节点实际持久化。

## 4. 统一 Fixture

A、B 两个方向必须原样使用同一组内容，不能用更短文案或更简单状态偏袒某一方向。

### 仓库与配置

- 仓库路径：`/Users/mei/Workspaces/regulated-platform/architecture-and-config-migration-scenarios/repolens-documentation-monorepo`
- 长路径补充：`/Volumes/Engineering Archive/Legacy Migration/2022-2026/transaction-processing-platform-and-disaster-recovery-handbook`
- `site.title`：`Regulated Platform Architecture and Configuration Migration Reference`
- `site.home`：`docs/architecture/2026-q3/long-running-migration-and-rollout-guide.md`
- 规则 1：`docs/architecture-and-governance/**/*.md`，`markdown.math: true`
- 规则 2：`docs/partner-integrations/legacy-embedded-reports/**/*.html`，`html.view: direct`
- 规则 3：`internal/customer-data-export/**`，`render: false`
- `theme.vars.accent: "#0969da"`
- `theme.css: docs/assets/regulated-platform-overrides.css`

### Warning、错误和结果

- 读取 warning：仓库内出现 `source`、`output`、`access`、`unknown_future_option`；它们不得成为可编辑控件或进入写回。
- 字段错误：`rules[1].match: "["`、`render.max_file_size: "six MB"`、`view.toc_panel: docked`。
- revision 冲突：确认 diff 前磁盘文件已被外部修改，必须重新读取，不得覆盖。
- 成功结果：`Files: 438`、`Pages: 721`、`Duration: 3.842s`。
- 缓存路径：`/Users/mei/Library/Caches/repolens/ui/builds/9d6bc2d6e52d0e1e421b4d0a4cf7af832f268bc1e8b9f8a4b4a129b8cda3d5f1`。
- 构建失败：主题 CSS 不存在；最近一次成功结果仍可见。

## 5. 允许画板

目标文件只能包含以下顶层画板。命名必须保留数字前缀，便于按顺序审查。

### 说明与覆盖矩阵

- `00 Task and Fixture`：一句话任务、统一 fixture 摘要、状态覆盖矩阵、A/B 图例。不得扩展为完整 Foundations 或视觉规范板。

### 方向 A：引导式任务流

- `A01 1440 Project Open`
- `A02 1440 Config Edit`
- `A03 1440 Diff`
- `A04 1440 Build`
- `A05 390 Project Open`
- `A06 390 Config Edit`
- `A07 390 Diff`
- `A08 390 Build`

### 方向 B：持续上下文工作台

- `B01 1440 Project Open`
- `B02 1440 Config Edit`
- `B03 1440 Diff`
- `B04 1440 Build`
- `B05 390 Project Open`
- `B06 390 Config Edit`
- `B07 390 Diff`
- `B08 390 Build`

### 结构评分

- `99 Structure Scorecard`：使用第 10 节评分表，记录 A/B 得分、证据、风险和建议选择。不得只写总分。

不得增加 Dashboard、最近项目、远程仓库、预览站点、登录、目录浏览器或营销页面。

## 6. 每组画板必须覆盖的状态

每个方向的 1440 和 390 画板都必须表达以下状态。可以在主页面旁放置同页面的局部状态变体，不要求为每种状态新增顶层画板。

### Project Open

- 首次空状态：绝对路径输入、用途说明、唯一主动作。
- loading：正在读取仓库，防止重复提交。
- 路径失败：相对路径、非 Git/不可读目录中的至少一种，错误靠近路径字段且可直接修复。
- 打开成功：当前仓库路径、配置来源、warning 数量和下一步清晰。

### Config Edit

- 已保存与 dirty 状态均可识别。
- 展示 `site`、`render`、`rules`、`theme`、`view`、`agent` 的组织方式；必须使用真实长值和三条保序规则。
- `source`、`output`、`access` 只作为不可写信任域说明，不得伪装成禁用字段。
- 三类字段错误能定位到对应分区/字段，并有错误摘要。
- 规则可新增、删除、上移、下移；不得只提供拖拽。
- 校验、审阅 diff 的主次关系明确。

### Diff

- 显示目标 `<repo>/.repolens.yml`、规范化 YAML 影响和完整 unified diff 区域。
- 返回编辑与确认原子写入的含义清楚。
- 无变更状态不显示可写入主动作。
- revision 冲突说明不会覆盖外部修改，并提供“重新读取”恢复动作。

### Build

- 进行中：阶段和活动状态可识别。
- 成功含 warning：展示 Files、Pages、Duration、warning 和完整缓存路径。
- 失败：展示失败原因、重试/返回配置动作，并保留最近一次成功缓存路径。
- 不出现 preview、serve、打开部署站点等尚不存在的能力。

## 7. 1440px 结构要求

- 画板宽 1440px；内容区域应有稳定最大宽度或工作台边界，不能无限拉长表单。
- 当前仓库、保存状态和主动作在关键页面可快速发现。
- 相关字段成组；不为填满横向空间而把无关字段并排。
- 长路径、glob、diff 和错误文案必须使用完整 fixture 检查。
- 每屏只能有一个视觉上最强的主动作；破坏性或不可逆含义不能只靠颜色表达。

## 8. 390px 结构要求

- 画板宽 390px，按单栏任务流设计，不得简单缩放桌面画板。
- 方向 B 的分区导航进入明确的菜单/抽屉触发器；关闭后仍能识别当前位置。
- 规则编辑可以进入独立子层或折叠块，但顺序、错误和返回路径不能丢失。
- diff 在自身代码区横向滚动；整页不得横向滚动。
- 长路径允许合理换行、截断加完整访问方式或区域内滚动，不能遮挡主动作。
- 底部操作区不得遮挡最后一项内容、错误提示或聚焦控件。

## 9. 禁止事项

- 不建设完整颜色、排版、图标、阴影、圆角、动效或品牌系统。
- 不制作高保真视觉稿，不追求像素级装饰；使用灰阶、一个主强调色和必要的成功/警告/错误语义即可。
- 不从 Shadcn、Base UI、Design Goodies 或其他样例整页照搬视觉；它们只能帮助理解模式。
- 不新增产品功能、字段、API、路由或非目标入口。
- 不把后端配置对象原样平铺成一张超长表单而不做任务分组。
- 不用纯色、图标或 toast 作为唯一状态表达。
- 不用截图、图片或扁平化组合冒充可编辑页面结构。
- 不复制大量散件模拟组件；允许建立本任务需要的最小 reusable primitives，但不扩展成完整组件库。
- 不并行编辑同一 `.pen`，不依赖未保存的 Pencil 状态，不以导出 PNG 代替源文件。

## 10. 结构选择评分表

在 `99 Structure Scorecard` 中复制此表。每项按 1–5 分评分，并写出对应画板、状态或观察证据；加权分为 `评分 ÷ 5 × 权重`。

| 维度 | 权重 | 方向 A（1–5） | A 证据/风险 | 方向 B（1–5） | B 证据/风险 |
|---|---:|---:|---|---:|---|
| 主任务与下一步清晰度 | 20 |  |  |  |  |
| 安全边界与写入影响理解 | 15 |  |  |  |  |
| 错误定位与恢复能力 | 20 |  |  |  |  |
| 配置查找与规则编辑效率 | 15 |  |  |  |  |
| 仓库、dirty、构建上下文连续性 | 10 |  |  |  |  |
| 390px 任务可达性 | 10 |  |  |  |  |
| 键盘、焦点与阅读顺序可实现性 | 5 |  |  |  |  |
| 与现有 React + Base UI/API 契约适配度 | 5 |  |  |  |  |
| **总计** | **100** |  |  |  |  |

选择门禁：

- “主任务与下一步清晰度”和“错误定位与恢复能力”均不得低于 4 分。
- 任一方向若违反产品契约、安全边界、键盘可达或 390px 核心任务可达，直接淘汰，不以总分补偿。
- 总分只辅助决策；最终建议必须写清选择、放弃理由和仍需用户测试的问题。
- 不得在没有对照证据时直接混合 A/B。若建议混合，必须指出保留哪个方向的骨架、只吸收另一个方向的哪些局部模式。

## 11. 逐项验收

### 文件与画板

- [ ] 新建并只修改 `docs/design/ui/prototypes/repolens-ui-explorations.pen`。
- [ ] 未修改或覆盖现有 `repolens-ui.pen`。
- [ ] 顶层画板名称、数量和顺序符合第 5 节，没有范围外页面。
- [ ] 文件保存后关闭并重新打开，所有节点仍存在。

### 对照有效性

- [ ] A/B 使用完全相同的 fixture、页面任务和状态。
- [ ] A/B 同时完成 1440px 与 390px，不用一套方向的桌面稿对比另一套的窄屏稿。
- [ ] 差异集中于信息架构、导航、上下文和动作组织，不用视觉精致度制造偏差。

### 产品状态

- [ ] Project Open 的空、loading、失败、成功均可检查。
- [ ] Config Edit 的 saved、dirty、warning、三类字段错误和规则排序均可检查。
- [ ] Diff 的正常确认、无变更、revision 冲突均可检查。
- [ ] Build 的进行中、成功含 warning、失败且保留最近成功结果均可检查。
- [ ] `source`、`output`、`access` 未成为可写字段。
- [ ] 未出现 preview/serve、远程仓库、目录浏览器或其他非目标能力。

### 结构与内容

- [ ] 主动作、返回/取消和错误恢复动作语义明确。
- [ ] 仓库路径、保存状态和当前阶段在需要时可发现。
- [ ] 真实长路径、长标题、glob、diff、缓存路径和错误均已放入画板。
- [ ] 390px 没有整页横向滚动假设；diff 仅在自身区域滚动。
- [ ] 页面结构具备合理 DOM/键盘顺序的实现路径，焦点不会被 sticky 区遮挡。

### Pencil 质量

- [ ] 页面由可编辑节点构成，不是截图容器。
- [ ] 重复控件使用最小 reusable primitives/instance，且未提前扩展完整设计系统。
- [ ] 每个顶层 frame 均执行布局检查，不存在裁切、遮挡、意外重叠或零尺寸内容。
- [ ] 每个顶层 frame 均导出并逐张视觉检查，不存在空白、纯黑或缺失内容。
- [ ] `99 Structure Scorecard` 已填写分数、证据、风险和建议，不只给出总分。

## 12. 完成报告格式

编辑者完成后只报告事实：

```text
输出文件：docs/design/ui/prototypes/repolens-ui-explorations.pen
修改范围：仅该 .pen 文件
画板：列出顶层画板名称与节点 ID
状态覆盖：Project Open / Config Edit / Diff / Build 逐项通过或不通过
视口：1440 / 390 逐项通过或不通过
布局检查：逐画板结果
视觉检查：逐画板结果
A 得分：__/100
B 得分：__/100
建议：A / B / 有证据的局部混合
未决问题：列出；没有则写“无”
```
