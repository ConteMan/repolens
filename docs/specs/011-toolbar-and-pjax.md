# 011: 顶部工具栏与 pjax 导航

- 状态：已确认
- 关联：roadmap M5、spec 006（pjax 当时缓行）、spec 005（"查看源码"入口当时缓行）、spec 010
- 视觉契约：[docs/design/mockups/010-011-prototype.html](../design/mockups/010-011-prototype.html)（2026-07-08 三轮评审定稿，实现须与其对齐）

## 问题

浏览页顶栏目前只有站点标题和 Raw 链接，缺少常用阅读操作。参照 docu.md 交互，用户期望：TOC 开合、前进后退、当前文件名、缩放、布局宽度、Markdown 源码视图、下载、树开关、搜索——一条完整的工具栏。

## 行为

顶栏从左到右（十项，原型定稿）：

1. **树开关**（☰）——spec 010 的入口；
2. **前进 / 后退**——驱动 History API；配合本 spec 的 pjax 使导航不整页刷新；
3. **面包屑 ＋ 当前文件名**（中央弹性区，窄屏折叠为仅文件名）；
4. **TOC 开关**——Markdown 页显示；TOC 从正文内联盒改为**右上浮动面板**（原型定稿形态：fixed 卡片、滚动高亮当前章节、层级缩进），开合状态 localStorage 持久化；站点级默认形态可配 `view.toc_panel: floating | inline`（默认 floating，inline 即 v1 现状）；
5. **缩放**——内容字号档位（90 / 100 / 110 / 125%），`−`/`+` 按钮 + 百分比读数，localStorage 持久化；
6. **布局宽度**——内容区 narrow（72ch）/ default（980px，默认）/ full 三档循环切换（图标 + 当前档文字），持久化；
7. **页面信息面板（ⓘ）**——2026-07-08 评审新增并定稿：取代 v1 的正文页脚 meta 行（`footer.meta` 移除，正文底部不再有任何框架级信息）。点击弹出锚定面板：路径、类型、大小、最后更新时间、commit（短 hash 7 位 ＋ subject）、操作区（查看原始文件、复制路径）。点击面板外或 Esc 关闭；
8. **源码视图**——新增子页 URL `view/<path>/source/`：Markdown 页跳转到 chroma 高亮的 .md 源码页；HTML embed/direct 页跳到既有 source 形态渲染（补上 spec 005 缓行的"查看源码"入口）；code/image/binary 页无此按钮；
9. **下载菜单**——始终含"原始文件"（镜像路径，`download` 属性）；Markdown 页的源码即原始文件，不重复列出；格式转换（PDF/docx 等）明确不做，见边界；
10. **搜索入口**（spec 012）。

分组顺序（原型定稿）：`☰ | ← → | 面包屑+文件名（弹性区） | TOC · 缩放 · 宽度 | ⓘ · 源码 · 下载 · 搜索`，段间以细分隔线区隔。

**UI 字符串多语言**：工具栏 title、搜索占位、信息面板标签、"最后更新"等主题固定文案（约 15 条）走内置字符串表（zh / en），由 `site.language` 选择，非 zh 前缀语言回退 en。不做用户内容翻译。

**pjax**：拦截站内 `view/` 链接（树、面包屑、正文内链、目录列表），fetch 目标页替换内容区与顶栏状态 ＋ `pushState`；失败（网络/解析）回退整页跳转；`popstate` 正确还原。禁 JS 时一切链接为普通跳转。

## 接口契约

- `internal/site`：新增 `view/<path>/source/index.html` 产出（仅 markdown 与 html-embed/direct 的文件页）；`theme.PageData` 新增 `SourceHref string`（无源码页时为空，模板据此隐藏按钮）与 `FileSize int64`（ⓘ 面板展示，目录页为 0 隐藏该行）；`LastCommit` 既有 `Subject` 字段由 ⓘ 面板消费。契约变更同 PR 回写 spec 006。
- pjax 需要可定位的内容区容器与页面元数据（标题、TOC、当前树高亮），通过现有 DOM 结构 ＋ `data-*` 属性传递，不引入 JSON 页面清单。
- site.js 010＋011 合计预算 ~600 行，仍单文件、无框架、无打包器。

## 边界与非目标

- 不做 PDF / docx / epub 等格式导出（v1.x Out；如未来做，走独立 spec 评估 pandoc 类依赖与零外部请求的关系）；
- 不做打印样式优化（浏览器自带打印可用即可）；
- 前进/后退不维护自建历史栈，完全依赖浏览器 History；
- 镜像层不受影响。

## 验收

- 工具栏十项在对应 Kind 页面正确显隐；缩放、布局、TOC 开合三项偏好跨页面持久；
- ⓘ 面板字段齐全（路径/类型/大小/更新时间/commit hash+subject）、原始文件与复制路径可用、正文底部无 footer.meta 残留；
- `view.toc_panel: inline` 时退回 v1 内联 TOC 形态；
- `site.language: en` 时 UI 字符串全量英文（抽查 5 条）；
- 与视觉契约原型逐状态对照截图无明显偏差（间距/配色/圆角/图标）；
- `view/<path>/source/` 对 md 与 html 页可访问、高亮正确、相对链接自检通过；
- pjax：站内跳转不整页刷新、地址栏与标题正确、树高亮同步、popstate 前进后退正常、fetch 失败回退整页；
- 禁 JS：工具栏中纯 JS 功能（缩放/布局/pjax）隐藏或退化，链接类（源码/下载）仍可用；
- 质量门禁全绿；dogfood 视觉验收由维护者确认。
