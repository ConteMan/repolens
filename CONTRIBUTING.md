# 参与贡献 repolens

**中文** | [English](CONTRIBUTING.en.md)

感谢关注！repolens 以文档先行、AI Agent 优先的方式开发：[`docs/`](docs/README.md) 中的设计文档是唯一真相源，[AGENTS.md](AGENTS.md) 是编码 Agent 的入口。同样欢迎人类贡献者——规则对所有人一致。

## 开发环境

要求：Go ≥ 1.25，PATH 上有 `git`（repolens 调用系统 git）。修改 `repolens ui` 前端时还需要 Node.js 24 与 pnpm 10。

```sh
git clone git@github.com:ConteMan/repolens.git
cd repolens
go build ./...
go test ./...
pnpm --dir internal/ui/frontend install --frozen-lockfile
pnpm --dir internal/ui/frontend check
pnpm --dir internal/ui/frontend exec playwright install chromium
pnpm --dir internal/ui/frontend test:e2e
```

## 动手之前

- **先看路线图。** [docs/roadmap.md](docs/roadmap.md) 中标记 "Out of v1" 的功能是有意排除的；实现前请先开 Issue 讨论。
- **先搜索 Issue。** 重复问题请在原 Issue 补充证据；不要用仓库文档维护另一份任务清单。
- **文档与代码同行。** 涉及架构、配置 schema、URL 约定或公开 CLI 行为的变更，必须在同一 PR 内更新对应设计文档（或新增 ADR）。
- **新增依赖需要论证。** Go 直接依赖保持个位数；候选库必须主流且在维护。Node/pnpm 仅用于 `repolens ui` 前端开发构建，最终产物仍是单二进制。

## Issue 与设计跟踪

Issue 使用以下标签分流：

- `bug`：行为错误；
- `design-gap`：实现与已评审冻结的 Pencil 设计基线不一致；
- `enhancement`：现有 Spec 与 Roadmap 范围内的改进；
- `spec-needed`：必须先新增或修订 Spec、ADR 或 Roadmap 的需求。

`spec-needed` 在维护者确认方向、文档合同更新前不得进入实现。阶段性交付挂到对应 Milestone。UI 的 `design-gap` 必须提供 `.pen` 路径、节点 ID、页面/状态/视口和截图；仍在探索、无法关闭重开或没有节点映射的 Pencil 文件不是设计事实源。

## Commit 规范 — Conventional Commits

遵循 [Conventional Commits 1.0.0](https://www.conventionalcommits.org/)：

```
<type>(<optional scope>): <description>

[optional body]

[optional footer]
```

允许的 type：`feat`、`fix`、`docs`、`refactor`、`test`、`build`、`ci`、`chore`、`perf`。scope 为所涉包或领域，例如 `feat(render): add mermaid support`、`docs(adr): supersede ADR-002`。破坏性变更使用 `!` 或 `BREAKING CHANGE:` footer。Commit message 使用英文、祈使语气、标题不加句号。

## Pull Request

1. 从 `main` 拉分支：`feat/<short-name>`、`fix/<short-name>` 或 `docs/<short-name>`。
2. 保持 PR 聚焦——一个 PR 一个逻辑变更。
3. 质量门禁本地与 CI 都必须通过：`pnpm --dir internal/ui/frontend check`、`pnpm --dir internal/ui/frontend test:e2e`、`gofmt -l .`（无输出）、`go vet ./...`、`go test ./...`、`go build ./...`。
4. 用户可见的变更在 `CHANGELOG.md` 的 **Unreleased** 下登记。
5. 使用 `Closes #N` 关联并关闭 Issue；没有关联 Issue 时在 PR 中说明原因。

## 报告问题

请使用 Issue 模板。Bug 报告需包含：仓库（或最小复现）、执行的命令、输出。功能请求先描述问题，再谈方案。大型需求使用 `spec-needed` 模板；已确认 UI 设计与实现之间的差距使用 `design-gap` 模板。

## 许可证

提交贡献即表示同意以 [MIT License](LICENSE) 授权。
