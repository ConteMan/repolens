# M6 结构评审（2026-07-11）

选择：单项目、路径输入、仓库域字段编辑、写入前 Diff、构建结果页。构建输出进入仓库外缓存根，不承担预览服务生命周期。

放弃：服务端目录浏览器、远程仓库、外部可信配置、内嵌 serve 和新标签预览。规则与主题仍属于仓库可写 schema，因此采用结构化编辑，不应被错误地延期；`access.noindex` 必须延期，因为现有 `config.Load` 已证明仓库内 `access` 会被忽略。

Pencil 阻塞：检测到 `/Applications/Pencil.app`，但 `orca status --json` 报告运行时为 `stale_bootstrap`，`orca computer capabilities --json` 返回 `runtime_unavailable`，无法连接运行中的 Orca app。对目标文件的三次 Pencil `batch_design` 均返回“user cancelled MCP tool call”，没有创建 `.pen` 或节点。替代物为 [Mermaid/ASCII 主流程](../flows/m6-config-ui.md)、[画板清单](../screen-inventory.md)和 [fixture](../fixtures.md)。恢复后，由单一编辑者创建唯一 `docs/design/ui/prototypes/repolens-ui.pen`，含 `00 Foundations`、`01 Components`、1440 与 390 的项目选择/配置编辑/构建结果；运行布局检查后导出 `<画板 ID>-<Pencil 节点 ID>.png` 并回填本记录。

评审资产排除：检查时发现 `reviews/2026-07-11-assets/kJ6aB.png`（1440×1040）和 `reviews/2026-07-11-assets/XKVIY.png`（390×940），两者均为空白画布，且没有对应 `.pen`。它们不代表任何画板，不是本轮视觉交付或实现事实源；保留现状以避免覆盖并发工作，待 Pencil 保存能力恢复后按上述命名规则重新生成有效资产。

待确认：配置保真写回应采用 AST 保留注释，还是规范化 YAML；一期明确不承诺前者。CG-01 至 CG-05 必须先 contract-only 收口并有测试。`view.toc_panel` 应继续作为读取 warning 还是在 UI 写前提升为字段阻断，也需维护者确认。

审计证据：Spec 012 规格文件已标“已实现”；提交 `71eb2ee feat(site,theme): site search with filename and heading index (spec 012)` 修改 `internal/site/search.go`，`internal/site/site_test.go` 含搜索索引与开关测试。因此 `docs/specs/README.md` 已改为“已实现”。本轮未改 Go 产品代码、未改运行行为、未提交 git。
