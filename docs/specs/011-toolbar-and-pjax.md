# 011: 顶部工具栏与 pjax 导航

- 状态：草稿
- 关联：roadmap M5、spec 006（pjax 当时缓行）、spec 005（"查看源码"入口当时缓行）、spec 010

## 问题

浏览页顶栏目前只有站点标题和 Raw 链接，缺少常用阅读操作。参照 docu.md 交互，用户期望：TOC 开合、前进后退、当前文件名、缩放、布局宽度、Markdown 源码视图、下载、树开关、搜索——一条完整的工具栏。

## 行为

顶栏从左到右：

1. **树开关**（☰）——spec 010 的入口；
2. **前进 / 后退**——驱动 History API；配合本 spec 的 pjax 使导航不整页刷新；
3. **面包屑 ＋ 当前文件名**（中央弹性区，窄屏折叠为仅文件名）；
4. **TOC 开关**——Markdown 页显示；TOC 从正文侧移为可开合面板，状态持久化；
5. **缩放**——内容字号档位（90 / 100 / 110 / 125%），`−`/`+` 按钮，localStorage 持久化；
6. **布局宽度**——内容区 narrow（68ch，默认）/ wide / full 三档切换，持久化；
7. **源码视图**——新增子页 URL `view/<path>/source/`：Markdown 页跳转到 chroma 高亮的 .md 源码页；HTML embed/direct 页跳到既有 source 形态渲染（补上 spec 005 缓行的"查看源码"入口）；code/image/binary 页无此按钮；
8. **下载菜单**——始终含"原始文件"（镜像路径，`download` 属性）；Markdown 页的源码即原始文件，不重复列出；格式转换（PDF/docx 等）明确不做，见边界；
9. **搜索入口**（spec 012）。

**pjax**：拦截站内 `view/` 链接（树、面包屑、正文内链、目录列表），fetch 目标页替换内容区与顶栏状态 ＋ `pushState`；失败（网络/解析）回退整页跳转；`popstate` 正确还原。禁 JS 时一切链接为普通跳转。

## 接口契约

- `internal/site`：新增 `view/<path>/source/index.html` 产出（仅 markdown 与 html-embed/direct 的文件页）；`theme.PageData` 新增 `SourceHref string`（无源码页时为空，模板据此隐藏按钮）。契约变更同 PR 回写 spec 006。
- pjax 需要可定位的内容区容器与页面元数据（标题、TOC、当前树高亮），通过现有 DOM 结构 ＋ `data-*` 属性传递，不引入 JSON 页面清单。
- site.js 010＋011 合计预算 ~600 行，仍单文件、无框架、无打包器。

## 边界与非目标

- 不做 PDF / docx / epub 等格式导出（v1.x Out；如未来做，走独立 spec 评估 pandoc 类依赖与零外部请求的关系）；
- 不做打印样式优化（浏览器自带打印可用即可）；
- 前进/后退不维护自建历史栈，完全依赖浏览器 History；
- 镜像层不受影响。

## 验收

- 工具栏九项在对应 Kind 页面正确显隐；缩放、布局、TOC 开合三项偏好跨页面持久；
- `view/<path>/source/` 对 md 与 html 页可访问、高亮正确、相对链接自检通过；
- pjax：站内跳转不整页刷新、地址栏与标题正确、树高亮同步、popstate 前进后退正常、fetch 失败回退整页；
- 禁 JS：工具栏中纯 JS 功能（缩放/布局/pjax）隐藏或退化，链接类（源码/下载）仍可用；
- 质量门禁全绿；dogfood 视觉验收由维护者确认。
