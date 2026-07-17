## What

<!-- One logical change per PR. What does this PR do? -->

## Why

<!-- Use "Closes #N". Link the roadmap item, Spec, ADR, or confirmed design baseline. If there is no issue, explain why. -->

## Evidence

<!-- For UI changes: include the .pen path, node IDs, screen/state/viewport, and exported screenshots. Remove this section when not applicable. -->

## Checklist

- [ ] Commits follow Conventional Commits
- [ ] `gofmt -l .` is clean; `go vet`, `go test`, `go build` pass
- [ ] `pnpm --dir internal/ui/frontend check` and `pnpm --dir internal/ui/frontend test:e2e` pass when the UI is affected
- [ ] Design docs / ADRs updated in this PR if architecture, config schema, URL conventions, or public CLI behavior changed
- [ ] UI behavior is backed by an implemented contract or an explicitly tracked contract gap
- [ ] `CHANGELOG.md` updated for user-visible changes

## Risk and rollback

<!-- State the main risk and how to revert or recover. -->
