# 017: 术语标注与解释

- 状态：草稿
- 关联：roadmap M11、Issue #73、ADR-007、ADR-001、ADR-002、ADR-005、specs [003](003-markdown-pipeline.md)、[018](018-glossary-skill.md)、[005](005-site-assembly.md)、[006](006-theme-and-templates.md)、[008](008-agent-surface.md)、[011](011-toolbar-and-pjax.md)、[012](012-site-search.md)

## 问题

技术文档大量使用领域术语，读者的理解成本集中在两类词上：一类是行业通用但读者不熟的概念，另一类是同名不同义——同一个词在不同平台、不同上下文里指不同的东西。现有手段都不解决它：正文内展开解释会打断主线；集中的名词表要求读者中断阅读跳转；外部链接把读者带离站点，且解释的是通用含义，不是"本文语境下指什么"。

repolens 面向的是"仓库原样即站点"的文档，作者没有地方安放这类解释。需要一种成本足够低的标注方式：作者在正文中把一个词标记为术语，读者原地即可读到解释，且解释区分"通用含义"与"本文语境下的含义"。

## 行为

### 1. 启用与降级

1. 特性由 `render.markdown.glossary` 控制，默认 `false`，可被 `rules` 级联覆盖。未启用时管线不加载术语库、不注册术语相关的 AST 处理与模板片段，输出与不存在本特性时逐字节一致（ADR-007 准入条件一）。
2. 术语库目录由顶层 `glossary.dir` 指定，默认 `.repolens/glossary`。目录不存在时视为空术语库，不报错。
3. `glossary.strict` 默认 `true`：启用状态下引用了未定义的术语 key 时构建失败，并给出文件路径、行号与 key。`false` 时降级为告警。
4. **未启用时的降级是确定的**：`glossary: false` 的文件中出现术语标注语法时，渲染为纯文本显示文本，不生成链接、不留残余标记。非 strict 模式下引用未定义 key 时同样如此。任何配置组合下都不得输出 `href` 指向 `term:` 的链接。

### 2. 标注语法

1. 语法为标准 Markdown 链接，目标使用 `term:` 伪 scheme：

   ```markdown
   [广告聚合](term:mediation)与变现平台拥有展示级收入事实。
   ```

2. 显示文本与 key 解耦：正文可用"渠道""Channel""媒体源"指向同一个 key，术语的规范名由术语库的 `title` 决定。
3. key 规范为 `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`。引用处大小写不敏感，解析时归一化为小写。目标带 query 或 fragment、key 不合规范时，按未定义 key 处理。
4. 只作用于 `*ast.Link`。`*ast.Image` 的 `term:` 目标不做处理，保持原样。
5. 同一术语可在同页引用任意次，每次都是独立的可点击标注。
6. 该语法在 GitHub 与通用 Markdown 渲染器下是合法链接，正文可读、不产生渲染错误（ADR-007 准入条件二）。现有链接改写逻辑（spec 003 行为 3）对带 scheme 的目标一律跳过，两者互不干扰。

### 3. 术语库与合并

1. 公共术语库：`<glossary.dir>/<key>.yml`（同时接受 `.yaml`），**文件名即 key**，文件内不再重复 key 字段。文件名不符合 key 规范时跳过并告警；`<key>.yml` 与 `<key>.yaml` 同时存在时构建失败。
2. 文档 front matter 的 `glossary` 段提供**本文语境的覆盖与私有术语**：

   ```yaml
   ---
   title: 投放归因与广告收入数据流
   glossary:
     mediation:
       page: 主聚合平台决定展示哪家广告源的广告，并形成展示与收入事实。
     organic:
       title: Organic
       summary: 非广告带来的自然用户。
   ---
   ```

3. 合并语义为**字段级覆盖**：公共库条目为基底，front matter 中出现且非空的字段整体替换同名字段（`source` 作为整体替换，不做字段内深合并）。公共库不存在的 key 视为该文档的私有术语，只在本文档内可见。
4. 合并后 `title` 为空的条目按未定义 key 处理。
5. 术语库文件本身不被隐式排除：照常进入镜像层与浏览层，作者需要排除时使用 `ignore`。

### 4. 字段与安全

1. 条目字段（除 `title` 外均可选）：

   | 字段 | 含义 |
   |---|---|
   | `title` | 规范名，必填 |
   | `alias` | 别名、外文名或常见写法 |
   | `owner` | 归属：行业通用概念，还是某个平台/服务的私有实现 |
   | `summary` | 脱离本文也成立的通用解释 |
   | `page` | **本文语境下**具体指什么；只应出现在 front matter |
   | `warning` | 易混淆点与常见误用 |
   | `source` | `{label, url}`，权威出处 |

2. 所有字段按**纯文本**处理：不解析 Markdown、不允许内嵌 HTML，渲染时整体 HTML 转义。术语库是仓库作者书写的不可信输入（ADR-005、ADR-007）。
3. `source.url` 仅接受 `http` 与 `https`，其他 scheme 忽略该 `source` 并告警。渲染为带 `rel="noreferrer"` 的链接，与"私有站点不泄露访问痕迹"的约束一致。
4. 单条目各字段长度上限 2000 字符，超出截断并告警，避免异常数据撑爆每一个引用页面。

### 5. 页内术语表（单一事实源）

1. 启用且本页至少引用一个有效术语时，浏览页正文之后追加术语表小节，容器为 `<section class="glossary-appendix" id="glossary">`，每个条目 `id="glossary-<key>"`，**只包含本页实际引用到的术语**，按正文中首次出现的顺序排列。
2. 正文中的标注渲染为：

   ```html
   <a class="term" href="#glossary-mediation" data-glossary="mediation">广告聚合</a>
   ```

   无 JavaScript 时它是一个可用的页内锚点链接，跳转到术语表条目；每页无 JS 完整可读的约束（ADR-002）因此成立。
3. 术语表是本页术语数据的**唯一事实源**。不额外内联 JSON、不重复输出术语数据；增强层的抽屉从该 DOM 读取内容。
4. 打印样式：术语表保留并展开，抽屉与浮动入口隐藏。
5. 小节标题为内置多语言字符串：中文"术语"、英文"Glossary"。

### 6. 增强层：解释抽屉

按 ADR-002 的"预渲染 ＋ 薄增强层"定位，抽屉是渐进增强，不承担内容供给：

1. 点击 `.term` 时阻止默认跳转，打开侧向抽屉并直接展示该术语详情；术语表中的锚点仍可通过键盘与直接访问 URL fragment 使用。
2. 页面存在术语表时显示固定位置的浮动入口，打开时展示本页术语索引（`title` ＋ `alias`），点击条目进入详情，详情可返回索引。
3. 详情按序展示 `owner`、`title`、`alias`、`summary`、`page`、`warning`、`source`，缺失字段整块省略而非留空。`warning` 使用与其他块可区分的样式。
4. 可访问性：抽屉为 `role="dialog"` ＋ `aria-modal="true"`，打开时焦点移入、Tab 在抽屉内循环，`Escape` 与点击遮罩关闭，关闭后焦点还原到触发元素。`.term` 提供描述其为术语的 `aria-label`。
5. 抽屉在 pjax 内容替换后随新内容重新初始化（spec 011），不残留上一页的术语数据；替换期间抽屉如为打开状态则关闭。
6. 样式与脚本随 theme 的既有资源 embed 输出，无外部请求。默认 `layout` 提供完整 DOM；自定义 `layout` 若要保留该能力，必须消费 `PageData.Terms` 并保留等价的术语表结构与 class 约定。

### 7. 搜索与 Agent 视图

1. `view.search` 开启时，`search.json` 的每个文档条目增加 `terms` 数组，元素为 `{key, title, alias, anchor}`，`anchor` 为 `glossary-<key>`。搜索命中术语时跳转到对应文档的术语表条目。术语不产生独立的搜索文档。
2. `agent.llms_txt` 开启且术语库非空时，`llms.txt` 增加"术语表"小节，逐条列出 `title`、`alias`、`owner`、`summary` 与定义文件路径。文档私有术语不进入该小节。
3. `llms-full.txt` 行为不变：它是镜像层原始字节的拼接，不注入渲染期产物；术语库 YAML 文件本身已按普通文件参与其中。

## 接口契约

```go
package render

type GlossarySource struct {
    Label string
    URL   string
}

type GlossaryTerm struct {
    Key     string
    Title   string
    Alias   string
    Owner   string
    Summary string
    Page    string
    Warning string
    Source  *GlossarySource
}

// Glossary 是构建期解析完成的公共术语库，按归一化 key 索引；渲染期只读，可并发使用。
type Glossary map[string]GlossaryTerm

type MarkdownOptions struct {
    // 既有字段省略
    Glossary       bool      // 启用术语标注
    GlossaryStrict bool      // 未定义 key 时返回错误
    Terms          Glossary  // 公共术语库，nil 视为空
}

type MarkdownResult struct {
    // 既有字段省略
    // Terms 为本页引用到的术语，已合并 front matter 覆盖，按首次出现顺序去重。
    Terms []GlossaryTerm
}
```

```go
package site

// LoadGlossary 读取公共术语库；warnings 为可恢复问题（非法文件名、非法 URL、字段截断），
// error 仅用于构建必须失败的情况（key 冲突、YAML 解析失败）。
func LoadGlossary(root, dir string) (glossary render.Glossary, warnings []string, err error)
```

```go
package theme

type PageData struct {
    // 既有字段省略
    Terms []render.GlossaryTerm
}
```

```go
package config

type GlossaryConfig struct {
    Dir    string // 默认 .repolens/glossary
    Strict bool   // 默认 true
}
```

`render` 不读取文件系统，术语库由 `site` 加载后经 `MarkdownOptions` 注入，与 `render` 不导入 `internal/config` 的既有约定一致。front matter 覆盖在 `Render` 内部完成，不改变公共术语库。

`Markdown` 当前按 `[anchors][mermaid]` 预构建 goldmark 变体（`variants [2][2]`），本特性再加一维会让该结构继续劣化，实现时改为按选项组合缓存。

## 边界与非目标

- **不做启发式自动标注**：不扫描正文自动识别术语、不基于外部词表自动加注（ADR-007 决策 2）；
- 不提供全站术语索引页面：本期只有页内术语表与本页索引，全站术语页涉及新的 URL 约定（ADR-001），需要单独决策；
- 术语字段不支持 Markdown、不支持嵌套术语引用、不支持图片；
- 不做术语的多语言变体：一个 key 一份内容，站点语言由 `site.language` 决定的只是内置字符串；
- 不引入新的 Go 依赖：YAML 解析复用 `goccy/go-yaml`；
- 不改变镜像层：术语标注只影响浏览层渲染结果；
- 不做悬停即显的 tooltip：触屏与键盘可达性成本高于收益，统一走点击打开抽屉。

## 验收

- 未启用时对含术语标注的仓库构建，产物与移除本特性的构建逐字节一致；
- 表驱动测试覆盖语法解析：合法 key、大小写归一化、非法 key、带 query/fragment、图片节点、未定义 key 在 strict / 非 strict 下的行为、未启用时剥离为纯文本，并断言任何组合下产物中不出现 `href="term:`；
- 合并测试覆盖：公共库单独生效、front matter 字段级覆盖、`source` 整体替换、私有术语只在本文可见、合并后 `title` 为空按未定义处理；
- 安全测试覆盖：字段中的 HTML 与 Markdown 被转义、`javascript:` 等非 http(s) 的 `source.url` 被拒绝并告警、超长字段截断；
- `LoadGlossary` 测试覆盖：目录缺失返回空库无错、非法文件名告警跳过、`.yml` 与 `.yaml` 同 key 冲突报错、YAML 解析失败报错；
- golden-file 测试断言术语表 DOM 结构、`id` 与 `href` 对应关系、只含本页引用术语且按首次出现顺序排列、多语言标题；
- 主题测试断言抽屉 ARIA 属性、焦点行为与打印样式；Playwright 验证点击打开、Escape 关闭、焦点还原、无 JavaScript 时锚点跳转可用、pjax 切换后抽屉状态与数据正确重置；
- `search.json` 测试断言 `terms` 数组结构与 anchor 一致性；`llms.txt` 测试断言术语表小节内容与私有术语不入内；
- 产物无外部请求，`gofmt -l .`、`go vet ./...`、`go test ./...`、`go build ./...` 全部通过。
