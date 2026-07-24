# 015: 桌面固定文件树侧栏调宽

- 状态：已实现（2026-07-24）
- 关联：roadmap M9、spec 006 / 010 / 011、ADR-002、Issue #38、Issue #39、Issue #58

## 问题

桌面固定文件树只有站点作者通过 `theme.vars.sidebar-width` 设置的构建时宽度。访问者面对深层目录或长路径时无法临时扩大侧栏，文件树较小时也无法回收阅读空间。

Spec 010 明确把拖拽调宽排除在混合文件树实现之外；本 Spec 独立定义访问者调宽的交互、约束、持久化、重置与无障碍合同，不改变构建时配置的信任域。

## 行为

### 生效范围与尺寸

1. 调宽只在宽屏（CSS 视口 `>= 1024px`）、固定侧栏展开时可用。固定树收起、浮动树打开或窄屏时，分隔线隐藏且不进入 Tab 顺序。
2. 有效宽度限制为 `220px` 到 `min(520px, 45vw)`，像素上限向下取整。站点作者默认值和访问者偏好都在使用时钳制到当前视口范围；视口缩窄只改变本次有效值，不覆盖已保存偏好，视口恢复后可恢复原值。
3. 浮动树继续使用独立的 `min(320px, 88vw)`，不读取、不修改桌面固定侧栏宽度。本 Spec 不增加浮动树宽度配置。
4. 固定树收起时保留偏好；重新固定后恢复当前视口允许的有效宽度。PJAX 换页不得重置宽度或重复绑定事件。
5. `view.tree_position` 为 `left` 或 `right` 时，分隔线都位于侧栏靠内容的一侧。Issue #58 必须先恢复该既有配置合同，再进入本 Spec 实现。

### CSS 变量与重置

1. `theme.vars.sidebar-width` 继续定义站点作者默认值，对应 `--sidebar-width`。
2. JavaScript 把访问者有效值写入根元素的 `--sidebar-width-user`；固定侧栏布局使用 `var(--sidebar-width-user, var(--sidebar-width))`，因此访问者值优先、站点作者值兜底。
3. 指针拖动期间宽度实时生效，但只在 `pointerup` 时持久化最终整数 CSS 像素值。`pointercancel` 或拖动期间按 `Escape` 恢复本次拖动前的值且不写存储。
4. 键盘每次操作立即应用并持久化。双击分隔线或聚焦时按 `Enter`，清除访问者值和对应存储键，立即恢复经过当前视口钳制的站点作者默认值。
5. 重置只删除访问者偏好，不修改 `.repolens.yml`、外部配置、生成产物或站点作者 CSS。

### 指针、触摸与键盘

1. 分隔线使用 Pointer Events，只响应主指针并在拖动期间调用 pointer capture；鼠标、触控笔与触摸共享同一套逻辑。拖动命中区设置 `touch-action: none`，不得触发页面横向滚动或文本选择。
2. 分隔线是可聚焦的 `role="separator"`，`aria-orientation="vertical"`，并实时维护 `aria-valuemin="220"`、`aria-valuemax`、`aria-valuenow` 与本地化 `aria-label`。
3. `ArrowLeft` / `ArrowRight` 把分隔线沿屏幕对应方向移动 `8px`；按住 `Shift` 时步长为 `32px`。由于侧栏可能位于右侧，方向键表达分隔线的视觉移动方向，而不是固定的数值增减方向。
4. `Home` 设置最小宽度，`End` 设置当前视口最大宽度，`Enter` 恢复站点作者默认值。所有键盘操作都阻止默认页面滚动并受同一尺寸边界约束。
5. 调宽开始前分隔线保持焦点；完成、取消、双击重置、固定树收起再展开后，不把焦点强制移动到无关控件。若当前模式切换使分隔线隐藏，焦点回到顶栏文件树开关。

### 持久化与 Issue #39 边界

1. localStorage 键为 `repolens:sidebar-width:v1:<base-path>`，值为十进制整数 CSS 像素。localStorage 已按 origin 隔离；`<base-path>` 从当前站点 `_assets/site.js` 的绝对 URL pathname 推导并规范为以 `/` 开始、以 `/` 结束，使同一 origin 下不同部署子路径互不污染。
2. 读取失败、存储不可用、值不是十进制整数时静默回退站点作者默认值；不得影响导航和正文阅读。读取到超出当前视口范围的有效整数时只钳制本次有效值。
3. 本 Spec 不创建统一阅读偏好对象或偏好面板。拖动中的中间值只更新页面；`pointerup` 和每次键盘操作产生的提交值才是 Issue #39 后续消费的最终值。
4. Issue #39 若引入统一、有版本的站点偏好对象，负责从本键读取一次、迁移已提交值并定义后续删除策略；不得重新定义调宽交互、边界或重置语义。

### 视觉与动态效果

1. 分隔线默认显示为与侧栏边框一致的 `1px` 线，命中区为以该线为中心的 `24px` 透明区域，使用 `col-resize` 光标；不得额外占用布局列宽。
2. hover、拖动和键盘聚焦状态使用 `--accent` 强调可操作线；`focus-visible` 提供不被相邻内容裁剪的 `2px` 轮廓。高对比度模式下不能只依赖颜色区分焦点。
3. 拖动期间禁用 `.shell` 的列宽 transition，宽度必须跟随指针且不产生滞后。`prefers-reduced-motion: reduce` 下调宽、重置、收起和恢复均无宽度动画。
4. 在浏览器 200% 缩放下，按缩放后的 CSS 视口重新判断固定/浮动模式与最大值；不得出现页面级横向溢出。若缩放使视口低于断点，按浮动树合同处理。

## 接口契约

- `internal/theme/templates/layout.tmpl`：在固定侧栏与内容区边界增加单个 `.sidebar-resizer.js-only`；分隔线不复制进浮动树。
- `internal/theme/assets/site.css`：新增 `--sidebar-width-user` 回退链、宽度钳制、左右侧分隔线、24px 命中区、交互状态与 reduced-motion 规则。
- `internal/theme/assets/site.js`：新增 base-path 作用域键、尺寸钳制、Pointer Events、键盘、双击重置和响应式恢复逻辑；初始化必须幂等并兼容 PJAX。
- 内置 zh / en 字符串表增加分隔线可访问名称；`theme.PageData` 不新增字段，配置 schema 与 CLI 表面不变。

## 视觉基线前置

实现前须新增并由维护者冻结可关闭重开的 `.pen` 视觉基线，再把路径、节点 ID 和状态回写本 Spec。基线至少覆盖：

- 1440px 左侧固定树：默认、hover、focus-visible、拖动；
- 1440px 右侧固定树对应状态；
- 最小值、最大值、双击及键盘恢复默认；
- 200% 缩放与 390px 浮动树（分隔线不存在）。

每个状态需要固定 fixture、布局检查与逐张导出证据；探索稿不能约束实现。

基线已于 2026-07-24 冻结：

- 文件：`docs/design/ui/prototypes/repolens.pen`
- 导出证据：`docs/design/ui/prototypes/repolens-baseline/<node-id>.png`
- 布局检查：对全文件执行 `snapshot_layout(problemsOnly: true, maxDepth: 7)`，结果为无布局问题

| 视口 / 状态 | 节点 ID |
| --- | --- |
| 1440px / 左侧 / 默认 | `dVnbT` |
| 1440px / 左侧 / hover | `s6fx1X` |
| 1440px / 左侧 / focus-visible | `qnIg7` |
| 1440px / 左侧 / 拖动中 368px | `J30jxk` |
| 1440px / 左侧 / 最小值 220px | `ZzNCZ` |
| 1440px / 左侧 / 最大值 520px | `vsOlh` |
| 1440px / 左侧 / 双击或键盘恢复作者默认值 | `H9QO0` |
| 1440px / 右侧 / 默认 | `E8PvC` |
| 1440px / 右侧 / hover | `KVarJ` |
| 1440px / 右侧 / focus-visible | `vkgbN` |
| 1440px / 右侧 / 拖动中 368px | `vndyD` |
| 200% 缩放 / 720px CSS 视口 / 浮动树且无分隔线 | `UfK2a` |
| 390px / 浮动树且无分隔线 | `SdapV` |

## 边界与非目标

- 不给浮动树增加独立调宽、拖动或持久化；
- 不引入统一阅读偏好面板、运行时主题或跨设备同步（Issue #39）；
- 不增加仓库身份、账号或服务端偏好存储；
- 不改镜像层，不在用户原始 HTML 中注入分隔线；
- 不在本 Spec 中修复 `view.tree_position` 既有合同缺口（Issue #58）。

## 验收

- 1440px 固定树可用鼠标、触控笔、触摸与键盘调到最小值、范围内值和最大值，ARIA 数值同步；
- `left` / `right` 两种固定布局的方向键、指针方向、边框和命中区正确；
- 双击或聚焦时按 `Enter` 恢复站点作者默认值；无效默认值、无效存储值和 localStorage 不可用时安全回退；
- 同一 origin 的 `/`、`/docs/` 两个部署互不污染；同一 base path 跨页面、刷新、PJAX 与浏览器会话保持；
- 固定树收起、重新固定和视口往返断点后偏好保留；浮动树宽度始终独立；
- 拖动取消不提交，拖动期间无 transition 滞后，减少动态效果设置生效；
- 200% 缩放与 390px 窄屏无页面级横向溢出；分隔线隐藏时不在 Tab 顺序；
- 禁用 JavaScript 时保持站点作者默认宽度的固定侧栏兜底；
- 模板单测覆盖分隔线语义与 zh / en 文案；Playwright 覆盖上述交互、持久化、PJAX、左右布局、缩放、窄屏和禁用 JavaScript；
- `pnpm --dir internal/ui/frontend check`、`pnpm --dir internal/ui/frontend test:e2e`、`gofmt -l .`、`go vet ./...`、`go test ./...`、`go build ./...` 全绿。

## 实现结果

- 模板、CSS 与站点脚本按接口契约落地，未新增配置 schema、CLI 表面或 `theme.PageData` 字段；
- theme 单测锁定 separator 语义、中英文文案、CSS 状态与脚本关键机制；
- Playwright 覆盖鼠标、触控笔、触摸、键盘、取消、重置、左右布局、base-path 隔离、PJAX、断点往返、200% 缩放等价 CSS 视口、390px、reduced-motion、localStorage 失败与禁用 JavaScript；
- 完整质量门禁于 2026-07-24 通过，Playwright 共 19 项通过。
