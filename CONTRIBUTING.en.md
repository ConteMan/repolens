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
- **Search existing issues first.** Add evidence to an existing issue instead of filing a duplicate, and do not maintain a second task list in repository documents.
- **Docs move with code.** Changes to architecture, the config schema, URL conventions, or public CLI behavior must update the corresponding design doc (or add an ADR) in the same PR.
- **New dependencies need justification.** Go direct dependencies stay in the single digits; candidates must be mainstream and actively maintained. Node/pnpm is limited to the `repolens ui` frontend build; releases remain a single binary.

## Issue and design tracking

Use these labels to route work:

- `bug`: behavior that does not work as documented;
- `design-gap`: an implementation mismatch with a reviewed and frozen Pencil design baseline;
- `enhancement`: an improvement that fits the current Specs and roadmap;
- `spec-needed`: work that requires a new or revised Spec, ADR, or roadmap decision before implementation.

Do not implement `spec-needed` work until a maintainer confirms the direction and the document contract is updated. Attach staged delivery to the relevant milestone. A UI `design-gap` must include the `.pen` path, node ID, screen/state/viewport, and screenshot. An exploratory Pencil file that cannot be closed and reopened or lacks node mapping is not a design source of truth.

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
5. Use `Closes #N` to link and close the issue, or explain in the PR why no issue is associated.

## Reporting issues

Use the issue templates. For bugs, include the repository (or a minimal reproduction), the command you ran, and the output. For feature requests, describe the problem before the solution. Use the `spec-needed` template for large requests and the `design-gap` template for mismatches against a confirmed UI baseline.

## License

By contributing you agree that your contributions are licensed under the [MIT License](LICENSE).
