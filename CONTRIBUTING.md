# 参与贡献 repolens

**中文** | [English](CONTRIBUTING.en.md)

感谢关注！repolens 以文档先行、AI Agent 优先的方式开发：[`docs/`](docs/) 中的设计文档是唯一真相源，[AGENTS.md](AGENTS.md) 是编码 Agent 的入口。同样欢迎人类贡献者——规则对所有人一致。

## 开发环境

要求：Go ≥ 1.25，PATH 上有 `git`（repolens 调用系统 git）。

```sh
git clone git@github.com:ConteMan/repolens.git
cd repolens
go build ./...
go test ./...
```

## 动手之前

- **先看路线图。** [docs/roadmap.md](docs/roadmap.md) 中标记 "Out of v1" 的功能是有意排除的；实现前请先开 Issue 讨论。
- **文档与代码同行。** 涉及架构、配置 schema、URL 约定或公开 CLI 行为的变更，必须在同一 PR 内更新对应设计文档（或新增 ADR）。
- **新增依赖需要论证。** 直接依赖保持个位数；候选库必须主流且在维护。永远不引入 Node 工具链。

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
3. 质量门禁本地与 CI 都必须通过：`gofmt -l .`（无输出）、`go vet ./...`、`go test ./...`、`go build ./...`。
4. 用户可见的变更在 `CHANGELOG.md` 的 **Unreleased** 下登记。

## 报告问题

请使用 Issue 模板。Bug 报告需包含：仓库（或最小复现）、执行的命令、输出。功能请求先描述问题，再谈方案。

## 许可证

提交贡献即表示同意以 [MIT License](LICENSE) 授权。
