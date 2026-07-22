# M6 画板与状态清单

本清单以当前 `internal/ui/frontend` 与 `internal/ui` 的真实行为为起点。Pencil 画板用于收口尚未确定的体验与状态，不以评审文字或单张 PNG 代替可编辑 `.pen` 中的节点、组件和状态。

## 优先级清单

| 优先级 | 画板 | 用户任务 | 关键状态 | 合同依赖 |
|---|---|---|---|---|
| P0 | `00 Foundations` | 理解界面的语义层级和反馈 | default、info、success、warning、blocking、focus、disabled | 无；需先统一本文件与 `internal/ui/frontend/src/styles.css` 的颜色、圆角和内容宽度 |
| P0 | `01 Core Components` | 使用一致、可键盘操作的基础控件 | Button / Input / Select / TriState / Textarea / Badge / Alert / Dialog / Skeleton 的 normal、hover、focus、disabled、error | Base UI 组件边界见 `docs/decisions/ADR-006-react-base-ui-admin.md`；现有共享组件见 `internal/ui/frontend/src/components.tsx` |
| P0 | `02 Project Open` | 输入本地 Git 工作树绝对路径并读取配置 | 初始、读取中、空路径、相对路径、非目录或不可读、配置不存在、YAML 解析失败 | CG-01、CG-02、CG-07；`POST /api/project/open` 返回同一快照的 `settings`、`effective`、`sources`、`warnings` 与 `revision` |
| P0 | `03 Config Edit` | 编辑仓库域字段及保序规则 | 默认、空配置、dirty、saved、仓库 warning、字段错误、规则新增/删除/排序 | CG-01、CG-02；字段范围以 `config.RepositorySettings` 和 `docs/design/config.md` 为准 |
| P0 | `04 Diff and Write` | 理解写入影响并明确确认 | 正常 diff、无变更、取消、revision 冲突、写入失败、写入成功、未知或非受控字段影响 | CG-03；当前接口为 `prepare-write` 与 `commit`，冲突返回 `revision_conflict` |
| P0 | `05 Build` | 查看构建进度与可用结果 | opening、loading_config、loading_theme、building、成功含 warning、失败、并发构建、失败但已有成功产物 | CG-04、CG-05；通用日志尾部与跨页面恢复最近成功结果已明确延期，不得在本期画板暗示可用 |
| P1 | `06 Narrow / Project and Config` | 在 390px 视口完成项目打开与配置编辑 | 与桌面相同的加载/错误/dirty 状态；分区导航；规则独立子页 | 不能只依赖 CSS 单列堆叠；需先确定导航与子页状态模型 |
| P1 | `07 Narrow / Diff and Build` | 在 390px 视口确认 diff 并查看构建结果 | 横向可滚动 diff、冲突恢复、纵向 Stats、warning/错误、sticky action 不遮挡内容 | CG-03、CG-04、CG-05 |

## 真实规模样本

所有业务画板使用 `docs/design/ui/fixtures.md` 中的长路径、长标题、三条规则、warning、失败值、Stats 和缓存路径。不得用只有短路径、单条规则或空白控件的示意内容代替真实规模验收。

## Desktop 验收

- 视口宽度 1440px；长仓库路径、长标题、三条规则、主题变量和 warning 均不裁切主操作。
- Project Open 的路径错误与 Config Edit 的字段错误应靠近对应控件，并保留页面级摘要；CG-02 已提供结构化问题、字段映射和焦点恢复，画板应直接消费该合同。
- 每个页面只有一个明确的主任务；校验、预览写入、确认写入和开始构建的可用条件与 `internal/ui/frontend/src/App.tsx` 的 dirty/busy 状态一致。
- diff 显示目标 `.repolens.yml`、规范化影响和完整文本；无变更不出现确认写入动作；revision 冲突提供明确的重新读取路径。
- Build 逐一覆盖 `internal/ui/build.go` 已存在的阶段、Stats、Warnings、错误和输出路径；通用日志尾部已明确延期，不得在本期画板中出现。
- 所有 normal、focus、disabled、error 状态可从 `.pen` 中直接定位；业务组件使用 reusable component/ref，不以导出 PNG 反推组件结构。

## Narrow 验收

- 视口宽度 390px；路径、glob、缓存路径和按钮文案不造成页面横向溢出。
- 配置分区通过可发现的窄屏导航访问；规则编辑进入独立子页，保留新增、删除、上移、下移与错误状态。
- diff 代码区保持原始换行并允许自身横向滚动，不把整个页面撑宽。
- 构建 Stats 改为纵向列表，warning、错误和最近成功结果保持清晰层级。
- sticky action 不遮挡最后一个字段、dialog 操作或构建结果；键盘焦点进入和退出 dialog 后位置正确。

## 原型交付门槛

唯一源文件为 `docs/design/ui/prototypes/repolens-ui.pen`。交付前逐项核对：

1. 顶层画板名称与本清单一致，并包含非空业务节点；
2. Foundations、Core Components、Desktop 和 Narrow 均可从源文件独立定位；
3. reusable components/ref 与具体页面引用关系可检查；
4. 对具体页面运行布局检查，确认无裁切和意外溢出；
5. PNG 仅作为评审快照，命名为 `<画板 ID>-<Pencil 节点 ID>.png`，并在评审记录中回填视口与状态。
