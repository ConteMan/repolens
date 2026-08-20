# 017: 术语标注与解释

- 状态：已实现
- 关联：roadmap M11、Issue #73、ADR-007、ADR-001、ADR-002、ADR-005、specs [003](003-markdown-pipeline.md)、[018](018-glossary-skill.md)、[005](005-site-assembly.md)、[006](006-theme-and-templates.md)、[008](008-agent-surface.md)、[011](011-toolbar-and-pjax.md)、[012](012-site-search.md)

## 问题

技术文档大量使用领域术语，读者的理解成本集中在两类词上：一类是行业通用但读者不熟的概念，另一类是同名不同义——同一个词在不同平台、不同上下文里指不同的东西。现有手段都不解决它：正文内展开解释会打断主线；集中的名词表要求读者中断阅读跳转；外部链接把读者带离站点，且解释的是通用含义，不是"本文语境下指什么"。

repolens 面向的是"仓库原样即站点"的文档，作者没有地方安放这类解释。需要一种成本足够低的标注方式：作者在正文中把一个词标记为术语，读者原地即可读到解释，且解释区分"通用含义"与"本文语境下的含义"。

## 行为

### 1. 启用与降级

1. 特性由 `render.markdown.glossary` 控制，默认 `false`，可被 `rules` 级联覆盖。未启用时不加载术语库、不输出术语表与增强层资源。
   **零影响的判定对象是不含术语标注语法的仓库**——这类仓库的产物与不存在本特性时逐字节一致（ADR-007 准入条件一）；判定不要求「未启用时完全不检查 AST」，因为含标注语法的文件即使未启用也必须处理（见本节第 6 条），否则会输出 `href="term:"` 的坏链接。
   实现上应先对源码做一次廉价的子串预检，不含标注语法时直接跳过全部术语处理，使零影响仓库不承担 AST 遍历开销。
2. 术语库目录由顶层 `glossary.dir` 指定，默认 `.repolens/glossary`。目录不存在时视为空术语库，不报错。
3. `glossary.strict` 为三档枚举，默认 `refs`。它区分两种不完整状态——**未定义**（引用了不存在的 key，属笔误或漏建）与**待补全**（条目存在但解释未写完，属写作中的正常中间状态）：

   | 取值 | 未定义引用 | 待补全条目 |
   |---|---|---|
   | `off` | 告警 | 放行 |
   | `refs`（默认） | 构建失败 | 放行 |
   | `complete` | 构建失败 | 构建失败 |

   构建失败必须给出文件路径、行号与 key；待补全的报告给出术语库文件路径与 key。
4. **待补全的判定**：条目合并文档级覆盖后 `summary` 与 `page` 均为空。只写 `page` 不写 `summary` 视为完整——本文特化的解释已足够。
5. **两档默认值服务两个场景**：`refs` 让作者可以先标注、后补解释，写作中途的构建与预览不被打断；`complete` 由构建者在外部配置中覆盖，作为合并门禁拦住未写完的条目。仓库内配置与外部配置的优先级按 ADR-005 既有语义，不引入新机制。
6. **未启用时的降级是确定的**：`glossary: false` 的文件中出现术语标注语法时，渲染为纯文本显示文本，不生成链接、不留残余标记。`off` 档下引用未定义 key 时同样如此。任何配置组合下都不得输出 `href` 指向 `term:` 的链接。

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
5. **`page` 只在 front matter 中生效**：公共术语库文件中出现 `page` 时忽略该字段并告警。它是本文语境的解释，写进公共库会污染所有引用该术语的文档——`SKILL.md` 已把它列为常见错误，因而更需要明确反馈而非静默忽略。
6. 术语库文件本身不被隐式排除：照常进入镜像层与浏览层，作者需要排除时使用 `ignore`。

### 4. 字段与安全

1. 条目字段（除 `title` 外均可选）：

   | 字段 | 含义 |
   |---|---|
   | `title` | 规范名，必填 |
   | `alias` | 别名、外文名或常见写法 |
   | `summary` | 脱离本文也成立的通用解释；术语的归属（行业通用 / 某平台私有）在此说明，不设独立字段 |
   | `page` | **本文语境下**具体指什么；只应出现在 front matter |
   | `warning` | 易混淆点与常见误用 |
   | `source` | `{label, url}`，权威出处 |

2. 字段内容按**纯文本 ＋ 行内代码**处理。术语库是仓库作者书写的不可信输入（ADR-005、ADR-007），因此不解析完整 Markdown，只识别一种标记：
   - 成对的单反引号之间的内容渲染为 `<code>`，其中的字符一律 HTML 转义，不再解析任何标记；
   - 未配对的反引号按字面字符输出；一对反引号之间内容为空时，两个反引号均按字面输出；
   - 不支持转义符：字面反引号即落单反引号，无 `\``  写法；
   - 反引号之外的一切按纯文本 HTML 转义，`**粗体**`、链接语法与内嵌 HTML 均原样显示。

   之所以只开这一个口子：术语解释高频出现字段名、事件名与 SDK 标识符（`af_ad_revenue`、`OnPaidEventListener`），纯文本渲染它们可读性明显不足；而完整 Markdown 会把链接、图片与原始 HTML 一并引入不可信输入的渲染路径。
3. **每个字段有两种形态**，渲染出口用 HTML 形态，非 HTML 出口一律用纯文本形态（去掉反引号标记后的字符串）：

   | 出口 | 形态 |
   |---|---|
   | 页内术语表、抽屉 | HTML |
   | `.term` 的 `aria-label` | 纯文本 |
   | `search.json` 的 `terms[]` | 纯文本 |
   | `llms.txt` 术语表小节 | 纯文本 |

   搜索索引使用纯文本形态是必须的：否则查询 `af_ad_revenue` 无法命中标题为 `` `af_ad_revenue` `` 的术语。
4. `source.url` 仅接受 `http` 与 `https`，其他 scheme 忽略该 `source` 并告警。渲染为带 `rel="noreferrer"` 的链接，与"私有站点不泄露访问痕迹"的约束一致。
5. 单条目各字段长度上限 2000 个 **Unicode 字符**（按原始字符串计，含反引号；中文文档按字节计会过于苛刻），超出时在字符边界截断并告警，不得切出不完整的多字节字符。
6. **告警有两条渠道，按数据来源分工**：公共术语库的问题由 `LoadGlossary` 在构建期集中产出；front matter 中定义或覆盖的条目由 `Render` 通过 `MarkdownResult.Warnings` 产出，携带文档路径与 key。front matter 的问题不得静默丢弃——私有术语只在本页可见，作者拿不到任何其他反馈。site 层负责汇总两条渠道的告警并输出。
   `Render` 的告警范围与 `MarkdownResult.Terms` 一致，**只针对本页实际引用到的术语**：正文未引用任何术语时不检查 front matter，未被引用的条目也不产生告警——它们不影响本页任何输出。

### 5. 页内术语表（单一事实源）

1. 启用且本页至少引用一个有效术语时，浏览页正文之后追加术语表小节，容器为 `<section class="glossary-appendix" id="glossary">`，每个条目 `id="glossary-<key>"`，**只包含本页实际引用到的术语**，按正文中首次出现的顺序排列。
2. 正文中的标注渲染为：

   ```html
   <a class="term" href="#glossary-mediation" data-glossary="mediation"
      aria-label="广告聚合（术语，查看解释）">广告聚合</a>
   ```

   无 JavaScript 时它是一个可用的页内锚点链接，跳转到术语表条目；每页无 JS 完整可读的约束（ADR-002）因此成立。
3. **`aria-label` 的文案**为「显示文本 ＋ 固定说明」，不是术语标题的重复——`aria-label` 会覆盖链接文本，若二者相同则屏幕阅读器读到的与普通链接无异，读者无从知道这里可以展开解释。内置字符串为中文 `%s（术语，查看解释）`、英文 `%s (term, view definition)`，`%s` 为正文中的显示文本。
   文案是本地化资源，属 theme 层；`render` 保持语言无关，经 `MarkdownOptions.GlossaryTermLabel` 接收格式串，由 `site` 从 `theme.UIStrings` 取值后传入。该字段为空时不输出 `aria-label`。
4. 术语表是本页术语数据的**唯一事实源**。不额外内联 JSON、不重复输出术语数据；增强层的抽屉从该 DOM 读取内容。
5. 打印样式：术语表保留并展开，抽屉与浮动入口隐藏。theme 目前没有 `@media print` 块，需要新建。
6. 小节标题为内置多语言字符串：中文"术语"、英文"Glossary"。

### 6. 增强层：解释抽屉

按 ADR-002 的"预渲染 ＋ 薄增强层"定位，抽屉是渐进增强，不承担内容供给：

1. 点击 `.term` 时阻止默认跳转，打开侧向抽屉并直接展示该术语详情；术语表中的锚点仍可通过键盘与直接访问 URL fragment 使用。
2. 页面存在术语表时显示固定位置的浮动入口，打开时展示本页术语索引（`title` ＋ `alias`），点击条目进入详情，详情可返回索引。
3. 详情按序展示 `title`、`alias`、`summary`、`page`、`warning`、`source`，缺失字段整块省略而非留空。`warning` 使用与其他块可区分的样式。
4. 可访问性：抽屉为 `role="dialog"` ＋ `aria-modal="true"`，打开时焦点移入、Tab 在抽屉内循环，`Escape` 与点击遮罩关闭，关闭后焦点还原到触发元素。`.term` 提供描述其为术语的 `aria-label`。
5. 抽屉在 pjax 内容替换后随新内容重新初始化（spec 011），不残留上一页的术语数据；替换期间抽屉如为打开状态则关闭。
6. 样式与脚本随 theme 的既有资源 embed 输出，无外部请求。默认 `layout` 提供完整 DOM；自定义 `layout` 若要保留该能力，必须消费 `PageData.Terms` 并保留等价的术语表结构与 class 约定。

### 7. 搜索与 Agent 视图

1. `view.search` 开启时，`search.json` 的每个文档条目增加 `terms` 数组，元素为 `{key, title, alias, anchor}`，`anchor` 为 `glossary-<key>`。搜索命中术语时跳转到对应文档的术语表条目。术语不产生独立的搜索文档。
2. `agent.llms_txt` 开启且术语库非空时，`llms.txt` 增加"术语表"小节，逐条列出 `title`、`alias`、`summary` 与 `DefinedIn` 路径。文档私有术语没有独立定义文件，不进入该小节。
3. `llms-full.txt` 行为不变：它是镜像层原始字节的拼接，不注入渲染期产物；术语库 YAML 文件本身已按普通文件参与其中。

## 接口契约

```go
package render

// GlossaryText 是术语字段的两种形态。Text 为去掉行内代码标记后的纯文本，
// 用于 aria-label、搜索索引与 llms.txt；HTML 为渲染结果，行内代码为
// <code>，其余内容 HTML 转义。两者由同一原始字符串产生。
type GlossaryText struct {
    Text string
    HTML template.HTML
}

// ParseGlossaryText 把术语字段的原始字符串转为 Text / HTML 两种形态。
// site.LoadGlossary 与 Render 的 front matter 合并共用此函数，避免两处各写一份解析。
func ParseGlossaryText(raw string) GlossaryText

// ValidGlossaryKey 报告 key 是否符合 ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$。
func ValidGlossaryKey(key string) bool

// GlossaryFrontMatterFields 返回 front matter 覆盖实际认可的 YAML 键，
// 由 spec 018 的字段名一致性检查消费，避免检查方重复维护一份字段清单。
func GlossaryFrontMatterFields() []string

type GlossarySource struct {
    Label GlossaryText
    URL   string
}

type GlossaryTerm struct {
    Key     string
    Title   GlossaryText
    Alias   GlossaryText
    Summary GlossaryText
    Page    GlossaryText
    Warning GlossaryText
    Source  *GlossarySource
    // DefinedIn 是条目的定义文件在仓库中的路径，由 LoadGlossary 填充；
    // front matter 中定义的私有术语留空。llms.txt 术语表小节需要它，
    // 且 .yml 与 .yaml 都合法，路径不能由 key 推导。
    DefinedIn string
}

// IsIncomplete 报告条目是否处于待补全状态：Summary 与 Page 均为空。
// 判定在合并 front matter 覆盖之后进行，因此只对 MarkdownResult.Terms
// 的元素有意义，对公共术语库条目无意义。
func (t GlossaryTerm) IsIncomplete() bool

type GlossaryStrictness string

const (
    GlossaryStrictOff      GlossaryStrictness = "off"
    GlossaryStrictRefs     GlossaryStrictness = "refs"
    GlossaryStrictComplete GlossaryStrictness = "complete"
)

// Glossary 是构建期解析完成的公共术语库，按归一化 key 索引；渲染期只读，可并发使用。
type Glossary map[string]GlossaryTerm

type MarkdownOptions struct {
    // 既有字段省略
    Glossary       bool               // 启用术语标注
    GlossaryStrict GlossaryStrictness // 空值等同 GlossaryStrictRefs
    Terms          Glossary           // 公共术语库，nil 视为空
    // GlossaryTermLabel 是 .term 的 aria-label 格式串，含单个 %s（显示文本）。
    // 由 site 从 theme.UIStrings 取值传入，使 render 保持语言无关；空则不输出 aria-label。
    GlossaryTermLabel string
}

type MarkdownResult struct {
    // 既有字段省略
    // Terms 为本页引用到的术语，已合并 front matter 覆盖，按首次出现顺序去重。
    Terms []GlossaryTerm
    // Warnings 为本次渲染中可恢复的问题：front matter 的非法 source.url /
    // 字段截断，以及 GlossaryStrictOff 下的未定义引用。构建必须失败的情况仍走 error。
    Warnings []string
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
    Strict string // "off" / "refs" / "complete"，加载时校验，默认 "refs"
}
```

`render` 不读取文件系统，术语库由 `site` 加载后经 `MarkdownOptions` 注入，与 `render` 不导入 `internal/config` 的既有约定一致；`config` 也不导入 `render`，`Strict` 在 `config` 中是校验过的字符串，由 `site` 映射为 `GlossaryStrictness`。front matter 覆盖在 `Render` 内部完成，不改变公共术语库。

待补全的判定依赖 front matter 覆盖，因此发生在渲染之后：`site` 遍历各页 `MarkdownResult.Terms`，在 `complete` 档下汇总 `IsIncomplete()` 为真的条目并使构建失败。`LoadGlossary` 不参与该判定。

`Markdown` 当前按 `[anchors][mermaid]` 预构建 goldmark 变体（`variants [2][2]`），本特性再加一维会让该结构继续劣化，实现时改为按选项组合缓存。

## 边界与非目标

- **不做启发式自动标注**：不扫描正文自动识别术语、不基于外部词表自动加注（ADR-007 决策 2）；
- 不提供全站术语索引页面：本期只有页内术语表与本页索引，全站术语页涉及新的 URL 约定（ADR-001），需要单独决策；
- 术语字段只支持行内代码一种标记，不支持其余 Markdown、不支持嵌套术语引用、不支持图片；
- 不提供独立的 lint 命令：`complete` 档的构建失败即是合并门禁，`repolens glossary` 子命令是独立的后续决策；
- 不做术语的多语言变体：一个 key 一份内容，站点语言由 `site.language` 决定的只是内置字符串；
- 不引入新的 Go 依赖：YAML 解析复用 `goccy/go-yaml`；
- 不改变镜像层：术语标注只影响浏览层渲染结果；
- 不做悬停即显的 tooltip：触屏与键盘可达性成本高于收益，统一走点击打开抽屉。

## 验收

- 未启用时对含术语标注的仓库构建，产物与移除本特性的构建逐字节一致；
- 表驱动测试覆盖语法解析：合法 key、大小写归一化、非法 key、带 query/fragment、图片节点、未定义 key 在 `off` / `refs` / `complete` 三档下的行为、未启用时剥离为纯文本，并断言任何组合下产物中不出现 `href="term:`；
- strict 档位测试覆盖：`refs` 下待补全条目放行且构建成功、`complete` 下同一仓库构建失败并报出术语库文件路径与 key、只写 `page` 不写 `summary` 在 `complete` 下视为完整、外部配置覆盖仓库内 `strict` 生效；
- 合并测试覆盖：公共库单独生效、front matter 字段级覆盖、`source` 整体替换、私有术语只在本文可见、合并后 `title` 为空按未定义处理；
- 行内代码测试覆盖：成对反引号渲染为 `<code>` 且内部字符转义、未配对反引号按字面输出、空反引号对按字面输出、无转义符语义，并断言同一字段的 `Text` 形态已去除标记；
- 出口形态测试断言 `aria-label`、`search.json` 的 `terms[]` 与 `llms.txt` 术语表小节使用纯文本形态，页内术语表与抽屉使用 HTML 形态；
- 安全测试覆盖：字段中的 HTML 与行内代码之外的 Markdown 被转义、反引号内的 HTML 被转义、`javascript:` 等非 http(s) 的 `source.url` 被拒绝并告警、超长字段截断；
- 告警渠道测试断言：front matter 中的非法 `source.url` 与字段截断产生 `MarkdownResult.Warnings` 条目且携带 key，公共术语库的同类问题产生 `LoadGlossary` warnings，两者不重复计入；
- 零影响测试断言：不含标注语法的源码不触发术语相关的 AST 遍历（子串预检生效）；
- `LoadGlossary` 测试覆盖：目录缺失返回空库无错、非法文件名告警跳过、`.yml` 与 `.yaml` 同 key 冲突报错、YAML 解析失败报错、库文件中的 `page` 被忽略并告警；
- golden-file 测试断言术语表 DOM 结构、`id` 与 `href` 对应关系、只含本页引用术语且按首次出现顺序排列、多语言标题；
- 主题测试断言抽屉 ARIA 属性、焦点行为与打印样式；Playwright 验证点击打开、Escape 关闭、焦点还原、无 JavaScript 时锚点跳转可用、pjax 切换后抽屉状态与数据正确重置；
- `search.json` 测试断言 `terms` 数组结构与 anchor 一致性；`llms.txt` 测试断言术语表小节内容与私有术语不入内；
- 产物无外部请求，`gofmt -l .`、`go vet ./...`、`go test ./...`、`go build ./...` 全部通过。
