# repolens

[中文](README.md) | **English**

> Turn any Git repository into a browsable static site — zero configuration required.

repolens takes a Git repository as-is and builds a static website from it: Markdown rendered with TOC and Mermaid, HTML files previewed directly, code files with syntax highlighting, all navigable through a persistent file tree. Visitors read everything in the browser — no cloning, no local tooling. AI agents get clean plaintext views of every file.

**Status: early design / pre-v1.** The architecture and roadmap live in [`docs/`](docs/README.md). Nothing is usable yet.

## Why

Documentation often already lives in a Git repository, but reading it comfortably requires cloning it, opening HTML files locally, and finding a Markdown previewer. That excludes collaborators, clients, non-technical readers, and AI agents. Existing site generators (MkDocs, Docusaurus, …) solve this only for repositories structured around them. repolens renders **any repository, unmodified**.

## Design highlights

- **Two-layer output** — a byte-for-byte *mirror* of the repository (relative links in your HTML just work; every file has a stable raw URL) plus a generated *view* layer with navigation chrome under `/view/`.
- **Pre-rendered MPA** — every page is a real URL that works on any static host, enhanced by a thin vanilla-JS layer for file-tree state and smooth navigation. No Node toolchain anywhere.
- **Rule-based rendering** — an ordered, cascading rule model (`glob → pipeline options`) customizes rendering per path; sensible defaults need no config at all.
- **Agent-friendly** — `llms.txt`, optional `llms-full.txt`, and a machine-readable JSON site index generated on every build.
- **Deploy anywhere** — output is a plain static directory: Cloudflare Pages, GitHub Pages, object storage, nginx. Fully relative links, zero external requests, works behind any auth proxy.
- **Single binary** — Go, with templates and assets embedded.

## Quick start (planned CLI)

```sh
repolens build git@github.com:you/your-docs.git   # → ./dist
repolens serve ./local-repo                        # local preview with live reload
```

## Documentation

- [Architecture](docs/design/architecture.md)
- [Configuration model](docs/design/config.md)
- [Architecture decision records](docs/decisions/README.md)
- [Roadmap](docs/roadmap.md)

## Contributing

See [CONTRIBUTING.en.md](CONTRIBUTING.en.md). This project is developed AI-agent-first; [AGENTS.md](AGENTS.md) is the entry point for coding agents.

## License

[MIT](LICENSE)
