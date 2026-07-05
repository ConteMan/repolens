# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-07-05

### Added

- `repolens build [repo|path]`：任意 Git 仓库（远程 URL / 本地路径 / `--worktree` 未提交工作区）构建为静态站点，双层输出——逐字节镜像层 ＋ `view/` 浏览层（specs 001、005）。
- Markdown 渲染管线：GFM、TOC、标题锚点、front-matter、Mermaid（vendored v11.16.0，客户端渲染）、AST 层站内链接改写（spec 003）。
- 文件分类与代码高亮：五类判定（Markdown/HTML/Code/Image/Binary）、chroma class 高亮、可链接行号锚点（spec 004）。
- 配置模型：`.repolens.yml` 有序规则级联（`glob → 管线参数`），零配置合理默认；`site.home`、`site.language` 生效（spec 002）。
- 默认主题：layout/page/dirlist/tree 模板集、CSS 变量与深色模式、文件树状态持久化 JS、三级定制（vars / custom.css / 模板覆盖）（spec 006）。
- `repolens serve`：本地预览，worktree fsnotify 监听 / git HEAD 轮询，防抖全量重建原子切换，失败保留旧站点（spec 007）。
- Agent 视图：`llms.txt`、`llms-full.txt`（2MB 截断）、`index.json` 机器索引、Markdown 页 `rel=alternate`（spec 008）。
- 目录页 README 渲染、`index.html` 按 Web 惯例合并进目录页（#9）、相对链接不变量构建自检、`.repolens-build` 哨兵防误删。
- 部署指南（GitHub Pages / Cloudflare Pages / nginx / 对象存储）。

### Historical

- Project skeleton: CLI scaffold, design docs, ADRs, CI quality gate.
- Vendored Mermaid v11.16.0 UMD build for local-only diagram rendering.

[Unreleased]: https://github.com/ConteMan/repolens/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/ConteMan/repolens/releases/tag/v1.0.0
