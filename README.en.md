# repolens

[中文](README.md) | **English**

> Turn any Git repository into a browsable static site — zero configuration required.

repolens takes a Git repository as-is and builds a static website from it: Markdown rendered with TOC and Mermaid, HTML files previewed directly, code files with syntax highlighting, all navigable through a persistent file tree. Visitors read everything in the browser — no cloning, no local tooling. AI agents get clean plaintext views of every file.

`build` and `serve` are usable — see [Releases](https://github.com/ConteMan/repolens/releases/latest) for the current version; architecture and roadmap live in [`docs/`](docs/README.md).

## Why

Documentation often already lives in a Git repository, but reading it comfortably requires cloning it, opening HTML files locally, and finding a Markdown previewer. That excludes collaborators, clients, non-technical readers, and AI agents. Existing site generators (MkDocs, Docusaurus, …) solve this only for repositories structured around them. repolens renders **any repository, unmodified**.

## Design highlights

- **Two-layer output** — a byte-for-byte *mirror* of the repository (relative links in your HTML just work; every file has a stable raw URL) plus a generated *view* layer with navigation chrome under `/view/`.
- **Pre-rendered MPA** — every page is a real URL that works on any static host, enhanced by a thin vanilla-JS layer for file-tree state and smooth navigation. Generated sites require no frontend framework or Node runtime.
- **Rule-based rendering** — an ordered, cascading rule model (`glob → pipeline options`) customizes rendering per path; sensible defaults need no config at all.
- **Glossary annotations** — mark a word as a term in your prose and readers expand its explanation in place, without leaving the page or the site. The syntax is a plain Markdown link, so the repository still reads fine on GitHub. Off by default.
- **Agent-friendly** — `llms.txt`, optional `llms-full.txt`, and a machine-readable JSON site index generated on every build.
- **Deploy anywhere** — output is a plain static directory: Cloudflare Pages, GitHub Pages, object storage, nginx. Fully relative links, zero external requests, works behind any auth proxy.
- **Single binary** — Go, with templates and assets embedded.

## Install

```sh
# macOS
brew install conteman/tap/repolens

# Windows (Scoop; or grab the zip from the Releases page)
scoop bucket add conteman https://github.com/ConteMan/homebrew-tap
scoop install repolens

# any platform (requires Go)
go install github.com/ConteMan/repolens/cmd/repolens@latest
```

Upgrades: package-manager users run `brew upgrade` / `scoop update`; direct downloads self-update via `repolens upgrade`.

## Quick start

```sh
repolens build git@github.com:you/your-docs.git   # → ./dist
repolens build . -o dist --ref v2.0               # local repository, specific ref
repolens serve . --worktree                       # local preview (incl. uncommitted changes), auto rebuild
repolens ui                                       # configure and build the current worktree in the UI
```

The build output is a plain static directory — hand it to any static host; see the [deployment guide](docs/deploy.md). `repolens ui` builds to a local cache by default and can use a safe custom absolute output directory for the current page session; this choice is never written to repository configuration. Rendering is customizable through an optional `.repolens.yml` at the repository root (sensible defaults need no config); see the [configuration model](docs/design/config.md).

## Glossary annotations

In technical documentation, comprehension cost tends to concentrate on two kinds of words: domain concepts the reader does not know, and **words that mean different things in different places** — the same term refers to something else on another platform or in another context. Explaining inline breaks the main thread; a central glossary page forces readers to interrupt and jump; external links take them off-site and explain the general meaning rather than what the word means *in this document*.

repolens lets authors keep those explanations in the repository, where readers meet them in place:

**1. Annotate** — a plain Markdown link with a `term:` target:

```markdown
User acquisition channels bring users in; the [ad mediation](term:mediation) platform decides which ad source is shown.
```

This stays a valid link on GitHub and in any Markdown renderer — adopting it never makes the repository worse anywhere else. Display text is decoupled from the key, so "mediation", "ad mediation", and "waterfall" can all point at the same entry.

**2. Define** — `.repolens/glossary/<key>.yml`, where the filename is the key:

```yaml
title: Ad mediation
alias: Mediation layer
summary: Unifies requests across multiple ad sources via bidding or waterfall to raise fill rate and revenue.
warning: AdMob can be a source inside another mediation platform, or the mediation platform itself. Confirm which role it plays first.
source: { label: AppLovin MAX docs, url: https://developers.applovin.com/en/max/getting-started/ }
```

A document's front matter can add what the term means *here*, or define private terms visible only on that page:

```yaml
---
glossary:
  mediation:
    page: The primary mediation platform picks which ad source is shown; it does not determine which acquisition channel brought the user.
---
```

`summary` answers "what is this word" and holds across documents; `page` answers "what does it mean in this one". Keeping those separate is the heart of the format.

**3. Enable** — the feature is off by default:

```yaml
render:
  markdown:
    glossary: true
glossary:
  dir: .repolens/glossary
  strict: refs        # off / refs / complete
```

In the generated site, annotated words open an explanation drawer on click, and a glossary section for the page is appended at the end. **Explanations survive with JavaScript disabled, in print, and when fetched by agents** — the drawer reads that very section, so there is never a second copy of the data. Terms also flow into site search and `llms.txt`.

The three `strict` levels separate two different problems: referencing a term that does not exist (a typo — always fails the build), and an explanation that is not written yet (a normal state while drafting — allowed by default). To block unfinished explanations at merge time, have CI override `strict` to `complete` from external config, while authors keep annotating first and explaining later locally.

When a repository does not use the feature, build output is byte-for-byte identical to a build without it.

The bundled [agent instructions](skills/glossary/SKILL.md) can be handed to a coding agent so it helps you collect terms and write them in the right format.

## Documentation

- [Architecture](docs/design/architecture.md)
- [Glossary agent skill](skills/glossary/SKILL.md)
- [Configuration model](docs/design/config.md)
- [Deployment guide](docs/deploy.md)
- [Architecture decision records](docs/decisions/README.md)
- [Roadmap](docs/roadmap.md)

## Contributing

See [CONTRIBUTING.en.md](CONTRIBUTING.en.md). This project is developed AI-agent-first; [AGENTS.md](AGENTS.md) is the entry point for coding agents.

## License

[MIT](LICENSE)
