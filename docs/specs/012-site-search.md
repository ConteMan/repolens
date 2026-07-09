# 012: 站内搜索（文件名 ＋ 标题级）

- 状态：已实现
- 关联：roadmap M5、spec 010（树顶入口）、spec 011（工具栏入口）、spec 008

## 问题

站点没有任何检索能力，找文件只能逐级点树。90% 的检索场景是"找文件 / 找章节"，用文件名 ＋ 标题级索引即可覆盖，且索引体积可控（KB 级），符合零外部请求约束。

## 行为

1. **构建期索引**：site 层输出 `search.json`（站点根，与 index.json 并列）：

   ```json
   {"docs": [
     {"path": "docs/guide.md", "title": "使用指南", "kind": "markdown",
      "view": "view/docs/guide.md/",
      "headings": [{"text": "安装", "anchor": "安装", "level": 2}]
   ]}
   ```

   收录全部有浏览页的文件（headings 仅 Markdown 有，取自 render 的 TOC 全量——不受页面 TOC 阈值开关影响）；`agent.index_json` 关闭不影响本产物；新增开关 `view.search`（默认开）。
2. **前端交互**：搜索框位于工具栏与树顶（spec 010/011 的入口）；快捷键 `/` 聚焦、`Esc` 关闭、`↑↓` 选择、`Enter` 跳转；结果按文件与章节两组展示，命中高亮；匹配为大小写不敏感的子串 ＋ 简单拼音首字母不做（见边界）。
3. **懒加载**：`search.json` 首次唤起搜索时 fetch，之后内存复用；加载失败时输入框提示不可用，不影响页面其余功能。
4. **无 JS 兜底**：搜索入口隐藏（`hidden` 由 JS 移除），页面其余功能不受影响。

## 接口契约

- `internal/site` 新增 `search.json` 生成（数据全部来自既有 model 与 render 结果，无新包依赖）；
- `internal/config`：`View` 增加 `Search bool`（默认 true），规则级联不适用（站点级开关）；
- 搜索 JS 并入 site.js 预算（010＋011＋012 合计 ~700 行量级；实现落点 715 行，超出部分为终审补充的 pjax 滚动定位与键盘可访问性修复）。

## 边界与非目标

- 全文检索 v1.x 不做（roadmap Out 维持，pagefind 仍是未来候选，届时独立 spec）；
- 不做拼音 / 模糊纠错 / 权重排序（子串命中按路径字典序）；
- 不做搜索历史。

## 验收

- `search.json` 结构如上、`json.Valid`、与站点文件一致（含中文标题）；`view.search: false` 时不生成且入口隐藏；
- 交互：`/` 唤起、键盘导航、文件与章节命中均可跳转（章节带 fragment）；
- 懒加载只发生一次；断网模拟下降级提示正确；
- 对 app-pdf-launcher-doc 真实仓库 dogfood：中文文件名与标题可检索；
- 质量门禁全绿。
