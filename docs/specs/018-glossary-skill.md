# 018: 术语标注 Agent Skill

- 状态：草稿
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
7. **一致性检查**：CI 对 `skills/glossary/example/` 执行构建，断言构建成功且产物包含术语表小节。spec 017 的格式若变更而 skill 未同步，该检查失败。检查随 spec 017 实现一并启用；在此之前 example 仅作为文档样例存在。
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

CI 检查项：example 构建成功；产物中存在 `id="glossary"` 的术语表小节；正文标注渲染为指向该小节的链接；产物中不出现 `href="term:`。

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
- spec 017 实现后，CI 对 example 的构建检查通过，且断言上述四个检查项；
- `gofmt -l .`、`go vet ./...`、`go test ./...`、`go build ./...` 全部通过。
