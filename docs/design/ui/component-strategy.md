# repolens UI 组件策略

> 状态：已冻结（2026-07-22）；Pencil 组件与页面基线见 [2026-07-22 最终设计基线评审](reviews/2026-07-22-final-baseline.md)
> 适用范围：`repolens ui` 本地管理界面
> 技术边界：React + TypeScript + Vite + Base UI；最终产物由 Go `embed` 进入单二进制；运行时零外部请求
## 1. 目标

本策略用于约束本地管理界面的组件分层、视觉语言和交互语义，重点解决三件事：

1. 让 Base UI 专注处理键盘、焦点、ARIA 和弹层行为；
2. 在项目内形成可修改、可复用、具有统一视觉语言的组件层；
3. 避免为了追求“组件齐全”而引入与本地配置工具无关的复杂度。

shadcn/ui 在本策略中是一种**开放源码组件层的设计方式和官方样例来源**，不是必须整体安装的运行时组件库。当前项目不把 Tailwind CSS、完整 shadcn CLI 初始化或完整 blocks 引入视为既定前提。

## 2. 推荐分层

```text
页面与业务流程
  └── features/*、App.tsx
      └── 项目组件层
          ├── components/ui/*       通用控件、状态和布局
          └── components/domain/*   配置项、规则编辑器、构建状态
              └── Base UI primitives
                  └── 原生 HTML / CSS / ARIA
```

各层职责如下：

- **页面与业务流程**：管理仓库打开、配置修改、校验、diff 确认、保存和构建状态，不直接处理弹层定位或键盘导航细节。
- **项目组件层**：提供稳定的项目内 API、语义样式和一致的状态表达。其源码归 repolens 所有，可以按需要修改。
- **Base UI primitives**：提供无样式的交互基础，包括焦点管理、键盘导航、ARIA、Portal 和弹层行为。
- **原生 HTML / CSS**：优先使用浏览器语义，样式通过项目内 CSS 变量和状态属性实现。

业务页面不应在多处重复拼装复杂的 Base UI primitive。成熟组合应收敛进 `components/ui`；带有 repolens 业务语义的组合则进入 `components/domain`。

## 3. 推荐目录结构

```text
internal/ui/frontend/src/
├── components/
│   ├── ui/
│   │   ├── button.tsx
│   │   ├── field.tsx
│   │   ├── input.tsx
│   │   ├── select.tsx
│   │   ├── switch.tsx
│   │   ├── dialog.tsx
│   │   └── status.tsx
│   └── domain/
│       ├── config-section.tsx
│       ├── rule-editor.tsx
│       ├── config-diff-dialog.tsx
│       └── build-status.tsx
├── features/
│   ├── repository/
│   ├── configuration/
│   └── build/
├── lib/
│   ├── api.ts
│   └── class-names.ts
├── styles/
│   ├── tokens.css
│   ├── base.css
│   └── components.css
└── App.tsx
```

目录按实际复杂度渐进拆分，不要求一次性重构到完整形态。只有出现第二个真实消费者时，才将页面内组合提升为通用组件。

## 4. 语义 token

颜色、圆角、间距和焦点样式必须通过语义变量表达。页面和业务组件不直接绑定具体色值。

建议最小 token 集：

```css
:root {
  --color-background: ...;
  --color-foreground: ...;
  --color-surface: ...;
  --color-surface-foreground: ...;
  --color-popover: ...;
  --color-popover-foreground: ...;
  --color-primary: ...;
  --color-primary-foreground: ...;
  --color-secondary: ...;
  --color-secondary-foreground: ...;
  --color-muted: ...;
  --color-muted-foreground: ...;
  --color-accent: ...;
  --color-accent-foreground: ...;
  --color-danger: ...;
  --color-danger-foreground: ...;
  --color-success: ...;
  --color-warning: ...;
  --color-border: ...;
  --color-input: ...;
  --color-focus-ring: ...;

  --radius-sm: ...;
  --radius-md: ...;
  --radius-lg: ...;

  --space-1: ...;
  --space-2: ...;
  --space-3: ...;
  --space-4: ...;
  --space-6: ...;
  --space-8: ...;
}
```

约束：

- 背景色与前景色成对设计，例如 `primary` / `primary-foreground`。
- 成功、警告和错误不能只靠颜色区分，必须同时提供文本或图标语义。
- Base UI 暴露的 `data-checked`、`data-unchecked`、`data-disabled`、`data-invalid`、`data-highlighted` 等状态属性是首选样式钩子。
- 弹层优先消费 Base UI 提供的尺寸与定位 CSS 变量，例如可用高度和锚点宽度。
- 必须保留清晰的 `:focus-visible` 样式，不允许全局移除 outline 后不提供替代方案。
- 字体使用系统字体栈或随二进制内嵌的本地资产，不使用外部字体服务。
- 暗色模式若后续引入，应覆盖同一组语义 token，不复制组件实现。

## 5. 组件分批

### 5.1 第一批：配置编辑主流程

第一批只覆盖当前核心流程。

| 项目组件 | 底层 primitive / HTML | 主要用途 |
| --- | --- | --- |
| `Button` | 原生 `button`；必要时 Base UI Button | 打开、校验、保存、构建、取消 |
| `Field` 家族 | Base UI Field / 原生 `label` | 标签、说明、控件、错误的统一组合 |
| `Input` | 原生 `input` | 仓库路径、站点标题、字符串配置 |
| `Textarea` | 原生 `textarea` | 真正需要多行输入的配置 |
| `NativeSelect` | 原生 `select` | 少量固定枚举，优先使用 |
| `Select` | Base UI Select | 需要统一弹层视觉或分组的固定枚举 |
| `Switch` | Base UI Switch | 明确的即时开/关设置 |
| `Checkbox` | Base UI Checkbox | 一组可独立选择的选项 |
| `Card` / `Section` | 原生 section / div | 配置域分组 |
| `Separator` | 原生语义或视觉分隔 | 划分相关配置组 |
| `Status` / `Alert` | `role=status` / `role=alert` | 加载、校验、保存、构建结果 |
| `Spinner` | CSS + 可访问文本 | 进行中的短时操作 |

### 5.2 第二批：确认和辅助操作

| 项目组件 | 底层 primitive | 主要用途 |
| --- | --- | --- |
| `ConfigDiffDialog` | Base UI Alert Dialog | 写入配置前展示 diff 并要求明确确认 |
| `Dialog` | Base UI Dialog | 普通的短编辑任务或仓库选择 |
| `UnsavedChangesDialog` | Dialog + 嵌套 Alert Dialog | 拦截关闭并确认丢弃未保存修改 |
| `Tooltip` | Base UI Tooltip | 解释无法仅凭文本理解的图标按钮 |
| `Popover` / `Menu` | Base UI Popover / Menu | 少量、锚定于触发器的辅助操作 |
| `Progress` | Base UI Progress 或原生 progress | 有确定进度值的构建过程 |

保存成功优先在页面状态区反馈。只有短暂、非关键且页面中已有持久结果时，才考虑增加 Sonner；不使用 shadcn/ui 已废弃的 Toast 组件。

### 5.3 第三批：信息架构增强

第三批需以真实页面增长为触发条件：

- `Tabs`：少量同级配置域，并且用户需要在域之间快速切换时使用；
- `Sidebar`：配置域稳定超过约五组、页面纵向扫描明显变差时使用；
- `Collapsible` / `Accordion`：高级配置需要默认收起时使用；
- `Item`：规则条目或构建产物需要统一的标题、描述和操作区时使用；
- `EmptyState`：未打开仓库、无规则、尚无构建结果时使用；
- `Skeleton`：只有加载时间可感知且页面结构稳定时使用。

同一层级不要同时使用 Sidebar 和 Tabs 表达相同导航关系。完整 dashboard、数据表、图表、用户菜单不是本地配置工具的默认组成部分。

## 6. Primitive 映射准则

| 交互意图 | 首选 | 不应替代为 |
| --- | --- | --- |
| 普通动作 | `button` / Button | 可点击 `div` |
| 页面跳转 | `a` | 伪装成链接的按钮 |
| 固定少量枚举 | 原生 Select | Combobox |
| 大量且需要过滤的选项 | Combobox | 不可过滤的 Select |
| 即时开/关 | Switch | 含糊的 Checkbox |
| 批量选择中的单项 | Checkbox | Switch |
| 普通短任务 | Dialog | Alert Dialog |
| 高风险或不可逆确认 | Alert Dialog | 普通 Popover |
| 少量锚定操作 | Menu / Popover | Dialog |
| 页面级成功或失败 | Status / Alert | 仅 Toast |

使用 Base UI `render` prop 组合项目按钮、链接或触发器时，自定义组件必须转发 `ref`，并将收到的属性传到实际 DOM 节点。一般不改变 primitive 默认元素；确需改变时，必须重新核对 HTML 语义和默认属性。

## 7. 表单与错误准则

### 7.1 字段结构

每个字段遵循统一结构：

```text
Field
├── FieldLabel
├── Input / Textarea / Select / Switch / Checkbox
├── FieldDescription（可选）
└── FieldError（出错时）
```

- 相关字段使用 `FieldSet` 和 `FieldLegend` 形成语义分组。
- 标签描述“这个值是什么”，帮助文本说明影响或格式，不重复标签。
- 错误文本说明如何修复，不只显示“无效”。
- 控件使用 `aria-invalid`，字段容器同时暴露 `data-invalid` 供样式使用。
- 服务端校验错误必须映射回具体字段；不能映射的错误放入表单级 Alert。
- 保存失败后保留用户输入，不清空表单。

### 7.2 状态矩阵

| 状态 | 视觉 | 可访问性 | 行为 |
| --- | --- | --- | --- |
| 默认 | 正常边框与文本 | 可访问名称存在 | 可编辑 |
| hover | 轻微表面变化 | 不作为唯一提示 | 不改变值 |
| focus-visible | 高对比焦点环 | 焦点顺序可预测 | 键盘可继续操作 |
| disabled | 降低强调度 | 原生 `disabled` 或等价语义 | 不可提交、不可聚焦时符合控件语义 |
| invalid | 错误边框、图标和文本 | `aria-invalid`，错误与字段关联 | 阻止提交或等待服务端修正 |
| loading | Spinner + 动作文本 | `aria-busy` 或状态区播报 | 防止重复提交 |
| success | 成功文本/图标 | `role=status`，非打断播报 | 保留可核验结果 |
| error | 错误文本/图标 | 关键错误用 `role=alert` | 提供重试或修复入口 |

当前表单可以继续使用 React 受控状态、原生约束校验和服务端校验。只有字段依赖、动态数组或客户端 schema 校验明显复杂后，才评估 React Hook Form、TanStack Form 等额外依赖。

## 8. 对话框准则

- `Dialog` 用于普通短任务；长表单应留在页面，不塞入弹窗。
- `AlertDialog` 只用于需要明确响应的重要确认，例如覆盖配置、丢弃修改、执行不可逆操作。
- diff 确认必须包含：变更摘要、可滚动的具体 diff、取消按钮、描述具体动作的确认按钮。
- Dialog 必须提供 Title、Description 和可见关闭入口，不能只依赖 Esc 或点击遮罩关闭。
- 打开时焦点进入对话框，关闭后回到合理触发点；使用 Base UI 默认焦点管理，除非有明确理由才覆盖。
- 表单关闭前若有未保存内容，应由受控 Dialog 在所有关闭路径上触发嵌套 Alert Dialog，包括 Esc、遮罩和关闭按钮。
- 长内容允许内部滚动，操作区保持可见；不依赖极高 `z-index` 解决层叠问题。
- 应用根容器创建独立 stacking context（`isolation: isolate`），保证 Portal 弹层稳定位于内容之上。

## 9. 选择器准则

- 少量固定选项优先使用 `NativeSelect`，获得原生行为、较低复杂度和更好的移动端选择体验。
- 需要分组、统一弹层样式或复杂展示时使用 Base UI Select。
- Select 只提供基础 typeahead，不适合大列表过滤；选项规模达到需要搜索时改用 Combobox。
- 每个 Select 都必须有可见标签，或在没有可见标签的特殊场景提供明确 `aria-label`。
- placeholder 不是标签；可清空选项需要作为明确 item 或外部“恢复默认”动作提供。
- 文件系统路径不伪装成 Select，应使用 Input 配合系统路径选择能力。

## 10. 可访问性验收清单

- 所有 Input、Checkbox、Switch、Select 均有可访问名称。
- 键盘可以完成核心流程：Tab、Shift+Tab、Enter、Space、Esc、方向键、Home、End。
- 所有图标按钮具有描述“动作 + 目标”的 `aria-label`。
- Dialog 和 Alert Dialog 具有明确 Title、Description 和焦点回归。
- focus-visible 在浅色、深色和错误状态下均有足够对比度。
- 成功、警告、错误和选中状态不只依靠颜色。
- 动态状态使用合适的 `role=status`、`role=alert` 或 `aria-live`，避免重复播报。
- 动画短促，并支持 `prefers-reduced-motion`。
- 200% 缩放和窄窗口下不丢失字段标签、错误文本或关键操作。
- Popup 不被容器裁剪，长 Select 和 Dialog 内容可滚动。
- 自动化测试之外，至少执行一次完整键盘走查。

## 11. 不建议做法

- 不在业务页面到处直接拼装 Base UI 复杂 primitive。
- 不把 shadcn/ui 当作不可修改的黑盒 npm 组件库。
- 不将 Tailwind CSS、完整 shadcn CLI 初始化或完整 block 作为采用此策略的前提。
- 不混用 Base UI 与 Radix primitive 构建同一套项目组件。
- 不一次性复制完整 dashboard、sidebar 集合或所有组件。
- 不为每个组件再包多层无业务价值的 wrapper。
- 不把所有枚举都实现为自定义 Select，也不用 Select 承担大列表过滤。
- 不把 Checkbox 与 Switch 当作纯视觉变体互换。
- 不用 Alert Dialog 承载普通编辑表单。
- 不用 Toast 作为字段错误、保存失败或构建结果的唯一反馈。
- 不用颜色作为唯一状态表达，不省略 focus-visible。
- 不在页面中散落具体颜色、阴影和圆角，绕过语义 token。
- 不加载外部 CDN、字体、图标或脚本；所有运行时资产必须进入构建产物并由 Go 嵌入。
- 不因官方提供暗色、动画或 dashboard 示例就在第一阶段增加非必要复杂度。

## 12. 官方参考

### Base UI

- [About Base UI](https://base-ui.com/react/overview/about)
- [Quick start](https://base-ui.com/react/overview/quick-start)
- [Accessibility](https://base-ui.com/react/overview/accessibility)
- [Styling](https://base-ui.com/react/handbook/styling)
- [Composition](https://base-ui.com/react/handbook/composition)
- [Forms](https://base-ui.com/react/handbook/forms)
- [Dialog](https://base-ui.com/react/components/dialog)
- [Select](https://base-ui.com/react/components/select)
- [Switch](https://base-ui.com/react/components/switch)

### shadcn/ui

- [Introduction](https://ui.shadcn.com/docs)
- [Base UI documentation announcement](https://ui.shadcn.com/docs/changelog/2026-01-base-ui)
- [Vite setup](https://ui.shadcn.com/docs/installation/vite)
- [Theming](https://ui.shadcn.com/docs/theming)
- [Field](https://ui.shadcn.com/docs/components/base/field)
- [Input](https://ui.shadcn.com/docs/components/base/input)
- [Native Select](https://ui.shadcn.com/docs/components/base/native-select)
- [Select](https://ui.shadcn.com/docs/components/base/select)
- [Dialog](https://ui.shadcn.com/docs/components/base/dialog)
- [Alert Dialog](https://ui.shadcn.com/docs/components/base/alert-dialog)
- [Tabs](https://ui.shadcn.com/docs/components/base/tabs)
- [Sidebar](https://ui.shadcn.com/docs/components/base/sidebar)
- [Blocks](https://ui.shadcn.com/blocks)
- [Sonner](https://ui.shadcn.com/docs/components/base/sonner)
