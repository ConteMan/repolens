## 变更内容

<!-- 一个 PR 只包含一个逻辑变更。说明本 PR 做了什么。 -->

## 变更原因

<!-- 使用 "Closes #N"。链接 Roadmap 条目、Spec、ADR 或已确认的设计基线。没有关联 Issue 时说明原因。 -->

## 验证证据

<!-- UI 变更需提供 .pen 路径、节点 ID、页面/状态/视口和导出截图。不适用时删除本节。 -->

## 检查清单

- [ ] Commit 遵循 Conventional Commits，描述和正文使用中文
- [ ] `gofmt -l .` 无输出；`go vet`、`go test`、`go build` 通过
- [ ] 涉及 UI 时，`pnpm --dir internal/ui/frontend check` 和 `pnpm --dir internal/ui/frontend test:e2e` 通过
- [ ] 涉及架构、配置 schema、URL 约定或公开 CLI 行为时，已在本 PR 更新设计文档 / ADR
- [ ] UI 行为已有实现合同支持，或已明确跟踪合同缺口
- [ ] 用户可见变更已更新 `CHANGELOG.md`

## 风险与回滚

<!-- 说明主要风险，以及如何回滚或恢复。 -->
