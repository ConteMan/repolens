# 018: 术语标注 Agent Skill

- 状态：已实现
- 关联：roadmap M11、Issue #73、[spec 017](017-glossary-terms.md)、[ADR-007](../decisions/ADR-007-optional-repo-conventions.md)

## 问题

术语标注是仓库可选约定（ADR-007），内容必须由作者主动书写。格式本身不复杂，难的是判断：哪些词值得标注、解释写到什么深度、如何避免术语库随时间劣化。

实际使用中这项工作大多交给编码 Agent 完成。缺少明确规范时，Agent 的失败方式是稳定且可预测的：把所有专有名词都标上导致满屏虚线、不检查已有条目就新建语义重复的 key、猜测术语含义写入错误解释、标注写完却忘记开启默认关闭的配置。

这些都不是程序能判定的问题，但可以由 repolens 随版本提供一份操作规范来约束。

## 行为

1. **位置与形态**：skill 存放于仓库顶层 `skills/<name>/`，主文件为 `SKILL.md`——YAML front matter 提供 `name` 与 `description`，正文为纯 Markdown 操作规范。术语标注 skill 为 `skills/glossary/`。
2. **Agent 中立**：不出现具体产品的工具名与加载机制，不依赖任何一家的目录约定或配置格式；需要执行的动作写为 shell 命令，并给出无法执行命令时的替代路径（自查清单）。任何能读写文件的编码 Agent 都可直接使用。
3. **职责边界**：skill 承载判断标准与协作流程——范围确认、读者画像、术语取舍、key 命名、标注密度、增量维护、常见错误。skill 不重新实现校验逻辑：正确性由 `repolens build` 在 `strict` 模式下裁决。
4. **自包含且自洽**：skill 在使用者的仓库中独立工作，那里没有 repolens 的设计文档。因此 SKILL.md **不引用 spec、ADR 或任何 repolens 内部文档**，也不声明规范源与版本——正文即是使用者需要的全部规范。
5. **一致性是维护者的责任**：skill 与实现的一致由 repolens 维护者保证，不转嫁给使用者。spec 017 的格式变更必须在同一 PR 内同步 SKILL.md 与 `example/`。skill 与实现不一致属于 `bug`。
6. **固定样例**：`skills/glossary/example/` 是一个最小的完整仓库片段，包含 `.repolens.yml`、公共术语库条目与一篇使用了标注、front matter 覆盖和私有术语的文档。它同时是文档示例与 CI fixture。
7. **一致性检查**：自动化测试构建 `skills/glossary/example/` 并断言 skill 所教的每一项写法都在产物中真实生效。检查随 spec 017 实现一并启用。

   检查**不是**对产物做字节级 golden 比对——那会被任何样式或模板调整打破，最终只会被一键刷新，失去防漂移的作用。要防的是「skill 教的写法与实现不再一致」，因此断言必须逐项对应 skill 的教学内容：

   - `SKILL.md` 字段表中列出的字段名，必须是实现实际支持的字段（术语库条目与 front matter 覆盖各自的集合）。多出实现不认识的字段、或实现新增字段而 skill 未收录，都应失败——这是最直接的漂移信号；
   - `example/` 中出现的每个字段都在产物里有对应输出：`title`、`alias`、`summary`、`warning`、带 `rel="noreferrer"` 的 `source`；
   - front matter 的 `page` 覆盖生效，私有术语生效且不进入 `llms.txt`；
   - 字段中的行内代码渲染为 `<code>`；
   - `strict: refs` 下构建成功（即 example 中不存在未定义引用）；
   - 产物中不出现 `href="term:`。
8. **不进入二进制**：skill 不 `go:embed`、不随发布二进制分发、不参与站点构建输出。它随源码仓库分发，使用者复制或引用文件路径即可。

## 接口契约

本 spec 不引入 Go 接口。契约是 `skills/glossary/example/` 的内容约定：

```
skills/glossary/
├── SKILL.md                              # front matter 含 name / description
└── example/
    ├── .repolens.yml                     # 开启 render.markdown.glossary 与 glossary 段
    ├── .repolens/glossary/<key>.yml      # 至少两条公共术语，覆盖 source 有无、行内代码有无
    └── docs/<file>.md                    # 含正文标注、front matter 的 page 覆盖与私有术语
```

检查以 Go 测试实现，与其余质量门禁一同在 CI 执行，不新增独立的工作流步骤。

## 边界与非目标

- 不提供 `repolens glossary` 子命令（lint / add / list / candidates）：那是独立的后续决策，skill 不预设它们存在；
- 不为特定 Agent 产品提供适配层、安装脚本或打包格式；
- 不约束 skill 之外的仓库内容，不要求使用者的仓库采用任何目录结构；
- 不做多语言版本：skill 正文与仓库文档语言约定一致（中文）；
- `skills/` 下暂只有术语标注一项，不预先建立通用的 skill 框架或索引机制。

## 验收

- `skills/glossary/SKILL.md` 存在且 front matter 仅含 `name` 与 `description`；
- 全文不含具体 Agent 产品的工具名与加载机制依赖（`skills/README.md` 的接入方式说明除外）；
- 全文不引用 spec、ADR 或其他 repolens 内部文档路径，不出现要求使用者自行核对规范的表述；
- SKILL.md 中的字段表、语法、配置片段与 spec 017 一致（由维护者在同一 PR 内保证）；
- example 覆盖：正文标注、front matter `page` 覆盖、私有术语、含 `source` 与不含 `source` 的条目、字段中的行内代码；
- spec 017 实现后，example 一致性检查通过，且覆盖行为 7 列出的全部断言；
- 字段名比对能真实捕捉漂移：删除或新增实现支持的字段而未同步 `SKILL.md` 时，检查失败；
- `gofmt -l .`、`go vet ./...`、`go test ./...`、`go build ./...` 全部通过。
