# Contributing to repolens

[中文](CONTRIBUTING.md) | **English**

Thanks for your interest! repolens is developed docs-first and AI-agent-first: design documents in [`docs/`](docs/README.md) are the source of truth, and [AGENTS.md](AGENTS.md) is the entry point for coding agents. Human contributors are equally welcome — the same rules apply to everyone.

## Development setup

Requirements: Go ≥ 1.25 and `git` on PATH. Changes to the `repolens ui` frontend also require Node.js 24 and pnpm 10.

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

## Before you code

- **Check the roadmap.** Features marked "Out of v1" in [docs/roadmap.md](docs/roadmap.md) are intentionally excluded; open an issue to discuss before implementing.
- **Docs move with code.** Changes to architecture, the config schema, URL conventions, or public CLI behavior must update the corresponding design doc (or add an ADR) in the same PR.
- **New dependencies need justification.** Go direct dependencies stay in the single digits; candidates must be mainstream and actively maintained. Node/pnpm is limited to the `repolens ui` frontend build; releases remain a single binary.

## Commit messages — Conventional Commits

We follow [Conventional Commits 1.0.0](https://www.conventionalcommits.org/):

```
<type>(<optional scope>): <description>

[optional body]

[optional footer]
```

Allowed types: `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `ci`, `chore`, `perf`. Scopes are the package or area touched, e.g. `feat(render): add mermaid support`, `docs(adr): supersede ADR-002`. Use `!` or a `BREAKING CHANGE:` footer for breaking changes. Write messages in English, imperative mood, no trailing period in the subject.

## Pull requests

1. Branch from `main`: `feat/<short-name>`, `fix/<short-name>`, or `docs/<short-name>`.
2. Keep PRs focused — one logical change per PR.
3. Quality gate must pass locally and in CI: `pnpm --dir internal/ui/frontend check`, `pnpm --dir internal/ui/frontend test:e2e`, `gofmt -l .` (no output), `go vet ./...`, `go test ./...`, `go build ./...`.
4. Update `CHANGELOG.md` under **Unreleased** for user-visible changes.

## Reporting issues

Use the issue templates. For bugs, include the repository (or a minimal reproduction), the command you ran, and the output. For feature requests, describe the problem before the solution.

## License

By contributing you agree that your contributions are licensed under the [MIT License](LICENSE).
