# repolens UI 结构探索 Brief

> 状态：结构探索与最终设计基线已完成，待实现回归与真实用户验证（2026-07-22）
> 范围：`repolens ui` 本地管理界面的结构与交互，不改变配置 schema、CLI 行为或生成站点。
> 技术边界：React + TypeScript + Base UI；前端产物由 Go `embed` 进入单二进制，运行时不依赖 Node，不请求外部 CDN、字体或脚本。
> GitHub 跟踪：[#27 Pencil A/B 结构探索](https://github.com/ConteMan/repolens/issues/27)；后续设计基线由 [#29](https://github.com/ConteMan/repolens/issues/29) 跟踪。

## 1. 当前事实与证据等级

### 产品事实

- 用户输入本地 Git 工作树的绝对路径，读取并编辑仓库信任域中的 `site`、`ignore`、`render`、`rules`、`theme`、`view`、`agent`。
- `source`、`output`、`access` 不属于仓库可写域，不得成为可编辑控件或进入写回 payload。
- 写入前必须经过服务端校验、展示目标路径和 unified diff，并由用户明确确认；revision 冲突不得覆盖外部修改。
- 写入会规范化 YAML，不承诺保留注释、空行、原键顺序或未知字段，确认界面必须明确告知。
- 构建输出位于仓库外的本机缓存目录；界面展示阶段、统计、warning、错误和最近一次成功结果。本期不承担预览服务生命周期。
- 当前实现已经覆盖“打开仓库 → 编辑 → 校验 → diff → 原子写入 → 构建”的闭环，结构探索应重新组织体验，不得发明后端尚不存在的能力。

### 设计资产的使用规则

- `docs/design/ui/reviews/2026-07-11-assets/*.png` 只作为**历史线框和早期结构假设**，不得作为当前视觉、组件或页面完成度的事实源。
- `docs/design/ui/prototypes/repolens-ui.pen` 是 2026-07-22 冻结的唯一可编辑视觉事实源；8 个顶层画板、28 个 reusable component、状态覆盖与稳定节点映射见 [最终设计基线评审](reviews/2026-07-22-final-baseline.md)。
- `docs/design/ui/reviews/2026-07-11-m6-structure.md` 与 `prototypes/repolens-ui-explorations.pen` 只保留为历史结构证据，不得覆盖最终 `.pen` 的视觉、组件或状态合同。
- 最终源文件已通过保存后关闭重开、逐 frame 导出和全量布局检查；Issue #30 仍须以真实 API、浏览器连续断点、键盘/焦点和目标用户任务验证实现，不能把本次启发式评审当作真实用户证据。

## 2. 探索目标

为不熟悉 YAML 的维护者提供一条安全、可解释、可恢复的本地配置路径，同时不牺牲开发者处理复杂规则和长路径时的效率。

本轮需要回答：

1. 用户是否能立即判断“当前打开的是哪个仓库、修改是否已保存、下一步能做什么”？
2. 配置应采用任务步骤流，还是持续可见的工作台导航？
3. 大量配置项和有序规则如何分组，才能降低认知负担且不隐藏真实语义？
4. 校验、写入影响、revision 冲突和构建失败如何靠近问题出现的位置，并提供明确恢复动作？
5. 390px 窄屏到宽桌面下，核心任务是否保持可达，而不是仅缩放桌面稿？

非目标：远程仓库、最近项目、目录浏览器、原生桌面壳、外部配置、无损 YAML round-trip、内嵌 serve/watch、站点托管、仓库内容编辑。

## 3. 用户与核心任务

### 主要用户

- **首次维护者**：知道仓库路径，但不了解 `.repolens.yml` 的字段、信任域和规则覆盖语义。
- **熟练维护者**：频繁调整规则、主题和 Agent 输出，需要快速定位字段、比较变更并重复构建。

### 核心任务

1. 输入或粘贴绝对路径，确认仓库是否可读取。
2. 理解当前仓库、配置来源、可编辑范围和读取 warning。
3. 修改站点、渲染、规则、主题、浏览和 Agent 配置。
4. 在字段附近发现并修复校验错误，保留尚未提交的编辑上下文。
5. 检查写入目标、完整 diff 和规范化影响，明确确认或返回修改。
6. 处理 revision 冲突：重新读取，而不是覆盖磁盘上的外部修改。
7. 发起构建，查看阶段、统计、warning、错误、缓存路径和最近成功结果。
8. 从路径错误、配置错误、写入失败和构建失败中恢复。

核心流程不得出现死路：

`选择仓库 → 编辑配置 → 校验 → 审阅 diff → 确认写入 → 构建 → 查看结果/修复后重试`

## 4. 探索循环

每轮只解决一个明确问题，按以下顺序执行：

1. **提出假设**：写清目标用户、任务、风险和可观察的成功信号。
2. **选择 fixture**：使用第 7 节的真实规模内容，不使用短路径、单规则和全成功假数据掩盖问题。
3. **制作低保真流程**：先验证信息顺序、主动作和恢复路径，不先打磨阴影、插画或动效。
4. **补齐组件与状态**：使用 token 和 reusable component，覆盖正常、焦点、加载、错误、禁用和成功。
5. **串联可操作原型**：原型必须能完成一个端到端任务，而不是一组孤立截图。
6. **主控验收**：按本文件的阶段门禁、可访问性、响应式和 DoD 检查。
7. **记录结论**：保留选择、放弃、证据和未决项；通过后才进入下一轮。

## 5. 阶段输入、产出与退出门禁

| 阶段 | 输入 | 产出 | 退出门禁 |
|---|---|---|---|
| Discovery | Spec 013、当前 React 实现、真实错误、用户验收记录 | 用户类型、任务、痛点、约束、研究问题 | 每个核心判断能追溯到代码、测试或用户证据；需求和解决方案分离 |
| 任务与流程 | Discovery 结果 | 主路径、错误路径、恢复路径、状态转换 | 所有一期能力都归属于用户任务；每种错误都有用户可执行的下一步；无死路 |
| 信息架构 | 任务、配置 schema、信任域 | 页面/区域地图、导航、内容分组、稳定命名 | 不暴露 Go 包结构；同一概念只有一个名称；高频动作位置稳定；非目标没有伪入口 |
| 低保真 | IA、真实 fixture | A/B 两套关键页面线框及流程串联 | 不依赖颜色也能理解层级；主任务与恢复路径完整；真实长内容不破坏布局 |
| 设计系统 | 通过的结构、Base UI 能力 | 颜色/排版/间距/圆角/层级 token，基础组件及使用规则 | 页面无任意色值和间距；Pencil token 可映射到 CSS 变量；Base UI 只承担无样式行为基础 |
| 组件状态 | 流程、组件清单 | 状态矩阵、键盘行为、焦点去向、文案 | Button、Field、Select、TriState、Alert、Dialog、Rule、Diff、Build 均覆盖关键状态；异步结果不只靠短暂 toast |
| 可操作原型 | 页面、组件、fixture | 完整 `.pen`、可执行场景、逐 frame 导出 | happy path、校验失败、revision 冲突、构建失败可走通；保存后重新读取节点仍完整；导出无空白、纯黑、裁切和遮挡 |
| 可用性验证 | 原型、测试任务 | 观察记录、严重度、取舍和修订项 | 关键任务无重复出现的阻断问题；以能否完成任务判断，不以偏好投票代替 |
| 实现交付 | 通过的原型和规格 | 设计—组件—测试映射、实现验收清单 | 真实 API 和浏览器路径通过；视觉差异已修复或明确接受；`.pen`、截图、代码状态一致 |

## 6. A/B 结构方向

两套方向必须使用同一 fixture、相同状态和相同任务脚本测试，避免把视觉风格差异误判为结构优劣。

### A：引导式任务流

结构：`选择仓库 → 配置 → 审阅并写入 → 构建结果`。页面只突出当前步骤和一个主动作，配置内部按站点、渲染、规则、外观与 Agent 分组。

假设：首次维护者更容易理解安全边界、写入影响和下一步，误操作更少。

必须验证：

- 返回前一步是否保留编辑内容和错误上下文。
- 熟练用户是否因步骤切换而显著变慢。
- 配置较多时，步骤内分组是否仍然过长。
- 无变更、校验失败和 revision 冲突是否能回到正确步骤。

### B：持续上下文工作台

结构：顶部固定仓库与保存状态；左侧为 Overview、Site、Render、Rules、Theme、View、Agent；中央编辑当前分区；右侧或底部为校验、写入和构建状态。

假设：熟练维护者能快速跳转配置区，长表单更易定位，构建结果与编辑上下文可同时保留。

必须验证：

- 首次用户能否理解整体顺序，还是会跳过校验和 diff。
- 侧栏项目在 390px 如何进入抽屉且不丢失当前位置。
- 同时出现编辑、错误、构建信息时是否竞争注意力。
- 主动作是否在各分区保持稳定，而不是每屏出现多个同等级按钮。

### 选择门禁

记录以下同任务指标：任务完成与否、首次关键动作、回退次数、错误恢复成功率、需要解释的次数、完成时间和主观把握度。优先选择关键任务成功率和恢复能力更好的结构；时间与偏好只作辅助证据。若两者面向不同熟练度各有明显优势，可采用“首次引导 + 后续工作台”，但不得在没有证据时直接合并两套结构。

### 本轮结果

2026-07-17 的低保真结构评审完成了 A/B 各 8 个 1440px/390px 画板及评分卡。A 得分 86/100，B 得分 89/100，暂定选择 B 的持续上下文工作台作为后续骨架；仅吸收 A 的单一主动作、Build 前 Review 和 revision 冲突恢复模式。该结论尚未经过首次用户任务测试，不能外推为最终可用性结论。节点、状态、风险和下一证据门禁见 [结构 A/B 评审](reviews/2026-07-17-structure-ab.md)。

## 7. 真实规模 Fixture

原型和实现验收至少使用以下内容：

- 仓库路径：`/Users/mei/Workspaces/regulated-platform/architecture-and-config-migration-scenarios/repolens-documentation-monorepo`。
- 额外长路径：OneDrive 路径、带空格的 `/Volumes/Engineering Archive/...` 路径和中文本地化仓库路径。
- `site.title`：`Regulated Platform Architecture and Configuration Migration Reference`。
- `site.home`：`docs/architecture/2026-q3/long-running-migration-and-rollout-guide.md`。
- 至少三条保序规则：Markdown 数学公式、HTML direct、内部目录不渲染；包含长 glob。
- 主题变量、附加 CSS 路径和模板目录。
- 读取 warning：仓库内出现 `source`、`output`、`access` 与 `unknown_future_option`，它们不得进入写回。
- 字段错误：`rules[1].match: "["`、非数字 `render.max_file_size`、非法 `view.toc_panel: docked`。
- 文档错误：YAML 缩进错误；构建错误：主题 CSS 不存在。
- revision 冲突、无 diff、写入权限失败、同项目构建忙。
- 成功结果：`Files: 438`、`Pages: 721`、`Duration: 3.842s`，以及完整长缓存路径。
- 构建失败时仍展示最近一次成功路径。

所有页面至少覆盖：首次空状态、loading、成功、warning、字段错误、系统错误、dirty、已保存和不可执行状态。

## 8. 可访问性门禁

目标为 WCAG 2.2 AA；使用 Base UI 不等于应用已经无障碍。

- 仅键盘可以完成打开仓库、编辑、规则排序、校验、审阅 diff、确认写入和构建。
- DOM 阅读顺序、视觉顺序和 Tab 顺序一致；不使用正数 `tabindex` 修补布局。
- 所有输入有可见 label；hint、错误与控件建立程序化关联；表单错误既有摘要又靠近字段。
- `focus-visible` 清晰，Dialog 打开、关闭、错误修复后的焦点去向明确；焦点不被 sticky 区遮挡。
- 状态变化、构建进度、成功和失败通过合适的 live region/状态语义传达，不只改变颜色。
- 文本与控件对比度、200% 缩放、浏览器文本放大和系统高对比场景通过。
- 目标区域至少满足 WCAG 2.2 Target Size (Minimum) 的 24×24 CSS px 要求；规则排序同时提供按钮，不能只依赖拖拽。
- 自动检查无阻断问题，并完成人工键盘测试及至少一种屏幕阅读器冒烟。

## 9. 响应式门禁

repolens 是桌面优先的本地开发者工具，但不是固定 1440px 的静态画面。

- 必测宽度：390px、1024px、1440px；在临界宽度连续缩放检查，断点由内容失效位置决定。
- 390px 为单栏；导航进入可访问抽屉，规则可进入独立子页；主动作保持可达。
- 长路径、glob、diff 和日志在自身区域换行或滚动，不得造成整页横向滚动。
- sticky 操作区不得遮挡最后一个字段、错误或浏览器缩放后的内容。
- 宽屏控制内容行长和表单跨度，不能为填满空间把相关性低的字段强行并排。
- 中文、英文、长数字、空值和极端数量下无裁切、重叠、溢出或不可达控件。

## 10. Pencil 单编辑者任务模板

同一 `.pen` 在同一时刻只能有一个编辑者。实现 Agent 不并行改画布；主控负责拆单、验收和决定下一轮。

```text
目标：完成 [一个页面 / 一个组件状态组 / 一条错误恢复流程]。

事实输入：
- 用户任务：[任务]
- 产品契约：[Spec/API/字段与禁止项]
- Fixture：[本轮必须使用的真实内容]
- 参考：[已通过的 token、component origin、上一轮 frame]

允许修改：
- .pen 文件：[绝对路径]
- 节点/画板：[明确名称或 ID]

禁止修改：
- 其他画板和已验收 component origin
- 产品 schema、后端行为和非目标能力
- 未经主控批准的视觉 token

必须覆盖：
- 状态：[default/focus/loading/empty/error/success/dirty 等]
- 视口：[390/1024/1440 中本轮范围]
- 键盘与焦点行为：[明确]
- 极端内容：[长路径、长错误、大量规则等]

完成验证：
1. 保存 .pen；
2. 重新读取目标节点，确认内容持久存在；
3. 对目标 frame 执行布局检查；
4. 导出目标 frame 并逐张视觉检查；
5. 报告改动节点、使用的 component instance、未解决问题；
6. 按 brief 门禁逐项给出通过/不通过证据；
7. 不扩展任务范围，不以 PNG 代替 .pen 源文件。
```

Pencil 当前没有自动保存保障；每个稳定阶段都应保存并进入 Git。大范围修改前先提交或建立可恢复基线。组件必须先建立 origin，再在页面使用 instance，避免复制散件造成漂移。编辑者可以使用桌面/MCP，也可以使用官方 Pencil CLI 的 `--in` / `--out` headless 流程；无论入口如何，验收都以仓库中的输出 `.pen`、重新读取的节点和导出结果为准。

## 11. Definition of Done

- [ ] Discovery 结论、用户任务和产品约束均有仓内证据。
- [ ] A/B 两套结构使用同一 fixture 和任务脚本完成比较，并记录选择与放弃理由。
- [ ] 选定方向覆盖项目选择、配置编辑、写入确认、构建结果及关键恢复路径。
- [ ] `.pen` 包含 Foundations、组件及其状态、桌面页面和窄屏页面；不是空 frame 或截图容器。
- [ ] 页面使用 reusable component instance，视觉 token 可映射到 React/CSS 实现。
- [ ] happy path、路径失败、字段错误、无变更、revision 冲突、写入失败、构建成功含 warning、构建失败均可在原型中检查。
- [ ] 每个交互组件的 default、hover、focus-visible、active、disabled/loading、error、success 状态已定义；不适用状态有理由。
- [ ] WCAG 2.2 AA、键盘、焦点、状态播报、200% 缩放和最小目标尺寸门禁通过。
- [ ] 390px、1024px、1440px 以及临界宽度无整页横向滚动、裁切、遮挡或不可达操作。
- [ ] 每个交付 frame 均已保存、重新读取、布局检查、导出并逐张视觉验收；不存在空白或纯黑导出。
- [ ] `.pen`、导出图、组件/行为说明和实现验收矩阵一致，历史 PNG 未被误标为当前事实源。
- [ ] React 实现使用真实 API 完成端到端任务；视觉、交互、响应式和可访问性差异已修复或明确记录接受。
- [ ] 项目质量门禁和浏览器冒烟通过，且未引入远程请求或最终用户 Node 依赖。

## 12. 官方与成熟体系来源

- GOV.UK Service Manual：[User research for government services](https://www.gov.uk/service-manual/user-research/how-user-research-improves-service-design)、[User research in discovery](https://www.gov.uk/service-manual/user-research/user-research-in-discovery)、[Designing good government services](https://www.gov.uk/service-manual/design/introduction-designing-government-services)、[Making prototypes](https://www.gov.uk/service-manual/design/making-prototypes)。用于用户研究、端到端任务、持续迭代和原型验证原则。
- W3C WAI：[WCAG 2.2](https://www.w3.org/TR/WCAG22/)、[ARIA Authoring Practices Guide](https://www.w3.org/WAI/ARIA/apg/)、[Keyboard Interface](https://www.w3.org/WAI/ARIA/apg/practices/keyboard-interface/)。用于可访问性合规、组件语义、键盘和焦点门禁。
- Base UI：[About Base UI](https://base-ui.com/react/overview/about)、[Accessibility](https://base-ui.com/react/overview/accessibility)。用于明确 headless 组件能力，以及应用仍需负责焦点样式、对比度、label 和页面级测试。
- USWDS：[Design principles](https://designsystem.digital.gov/design-principles/)、[Accessibility](https://designsystem.digital.gov/documentation/accessibility/)、[Form](https://designsystem.digital.gov/components/form/)、[Layout grid](https://designsystem.digital.gov/utilities/layout-grid/)。用于真实用户需求、表单、状态和响应式检查。
- web.dev：[Learn Responsive Design](https://web.dev/learn/design/)、[Media queries](https://web.dev/learn/design/media-queries/)。用于内容驱动断点和不同输入方式的响应式设计。
- Pencil 官方文档：[AI Integration](https://docs.pencil.dev/getting-started/ai-integration)、[.pen Files](https://docs.pencil.dev/core-concepts/pen-files)、[Components](https://docs.pencil.dev/core-concepts/components)、[Design Libraries](https://docs.pencil.dev/core-concepts/design-libraries)、[Pencil CLI](https://docs.pencil.dev/for-developers/pencil-cli)、[Design ↔ Code](https://docs.pencil.dev/design-and-code/design-to-code)。用于 Agent 协作、显式输入输出、保存、版本控制、组件复用和设计—代码交付。
