# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.8.0] - 2026-08-20

### Added

- `repolens skill` 命令族（`install` / `list` / `update`）：随二进制分发的 Agent 操作规范现在一条命令即可装到编码 Agent 实际扫描的目录，不必再手工复制文件。安装位置按仓库或用户主目录中的 Agent 配置探测（`.claude/`、`.codex/`、`.cursor/`、Copilot 的 `.github/skills/` 与 `.github/copilot-instructions.md`），都没探测到时回退到多家识别的中立路径 `.agents/skills/`；`--target` 显式指定，`--global` 装到个人目录，`--dry-run` 只报告不落盘。安装时把来源、版本与正文摘要写进 SKILL.md front matter 的 `metadata`，随文件走，因此升级 repolens 后 `repolens skill update` 无需联网即可把过期副本刷新到当前版本。被本地改动过的副本、以及位置上同名但非 repolens 安装的 skill 一律跳过并告警，`--force` 才覆盖；写入只碰 `SKILL.md` 一个文件，同目录下使用者自己补充的内容不受影响。

### Changed

- 术语标注 skill 的目录由 `skills/glossary/` 改名为 `skills/repolens-glossary/`。Agent Skills 规范要求 front matter 的 `name` 与父目录名一致，原目录名不符合该约束，改名后可被合规工具正确发现。`SKILL.md` 内容本身未变；此前按旧路径手工复制过该文件的仓库不受影响，改用 `repolens skill install glossary` 即可。

## [1.7.0] - 2026-08-20

### Added

- 术语标注与解释（默认关闭，`render.markdown.glossary`）。正文用 `[显示文本](term:key)` 标注术语——这是标准 Markdown 链接，在 GitHub 与任意 Markdown 渲染器下仍然合法可读；解释写在仓库内 `.repolens/glossary/<key>.yml`，文档 front matter 可覆盖本文语境的含义或定义只在本页可见的私有术语。浏览页在正文后输出仅含本页引用术语的术语表，标注是指向它的锚点，关闭 JavaScript、打印和被 Agent 抓取时解释都不丢失；启用 JavaScript 时点击术语原地展开解释抽屉。`glossary.strict` 三档（`off` / `refs` / `complete`）区分「引用了不存在的术语」与「解释尚未写完」，前者任何时候都让构建失败，后者默认放行、可由构建者在外部配置中覆盖为 `complete` 充当合并门禁。术语同时进入站内搜索索引与 `llms.txt` 术语表小节。未使用该特性的仓库产物逐字节不变。

## [1.6.3] - 2026-08-14

### Fixed

- Markdown 标题锚点保留 Unicode 字母、数字与组合标记；中文及混合语言标题不再丢失非 ASCII 内容或退化为通用 `heading`，TOC 与搜索章节链接同步使用完整 fragment。纯 ASCII fragment 保持兼容；重新构建后，旧版本生成的缺损非 ASCII fragment 需要更新。

## [1.6.2] - 2026-08-06

### Fixed

- Markdown 文件页与由 Markdown 提供正文的目录页不再额外显示派生页面标题，正文直接从文章自身内容开始；浏览器标题、搜索与 Agent 元数据保持不变。
- 窄屏浮动文件树不再残留隐藏侧栏的网格轨道，正文恢复为占满可用宽度的单列布局。

## [1.6.1] - 2026-07-27

### Fixed

- 文件树将目录折叠控件与目录页链接拆为同级交互元素，消除 `<summary>` 内嵌链接的无障碍告警，同时保留键盘操作、PJAX 与禁用 JavaScript 时的目录导航。

## [1.6.0] - 2026-07-27

### Added

- 生成站点可通过同源快照元数据发现新部署，在页面可见且在线时保守检查，并以不抢焦点的中英文提示让访问者主动重新加载；静态托管、子路径部署与 `repolens serve` 共用同一机制。

## [1.5.0] - 2026-07-24

### Added

- 桌面固定文件树支持鼠标、触控笔、触摸与键盘调宽，访问者偏好按部署 base path 持久化，并支持双击或 `Enter` 恢复站点作者默认值。

### Fixed

- `view.tree_position: right` 现在会镜像生成站点的桌面固定文件树、收起列、边框和浮动树出现方向，不再静默保持左侧布局。

## [1.4.1] - 2026-07-23

### Changed

- 文件树侧栏与浮动覆盖层仅滚动文件列表，树级操作和搜索入口固定在顶部，长仓库中无需返回树顶即可继续操作。

## [1.4.0] - 2026-07-23

### Added

- 文件树增加全部展开、全部折叠和定位当前文件操作；固定树与浮动树共享去重后的目录状态，长树项目和面包屑可查看完整仓库相对路径。

## [1.3.1] - 2026-07-22

### Fixed

- Release workflow 将 GoReleaser Action 固定到使用 Node.js 24 的 v7.2.3，不再产生 Node.js 20 弃用 annotation。

## [1.3.0] - 2026-07-22

### Added

- `repolens ui` 的构建页支持会话级自定义绝对输出目录；服务端拒绝危险路径与无所有权目录，覆盖旧产物前要求显式确认，并在同级临时目录构建后原子发布。

### Changed

- `repolens ui` 打开项目时同时提供仓库值、有效默认值、字段来源与读取 warning；配置校验错误可定位到具体字段并恢复焦点。
- 冻结 `repolens ui` 最终 Pencil 设计基线，覆盖 Foundations、核心组件、Project Open、Config Edit、Diff/Write、Build 及 390px 窄屏状态。
- `repolens ui` 按冻结基线落地方向 B 工作区、桌面与窄屏布局、Rule 子页、revision 冲突恢复及构建结果状态，并以真实临时 Git 仓库补齐浏览器回归。

## [1.2.1] - 2026-07-17

### Changed

- `repolens ui` 已删除原生单文件前端，迁移到 React、TypeScript、Vite 与 Base UI；生产资源继续由 Go embed 进入单二进制，最终用户无需安装 Node。

### Fixed

- GFM 表格包含长 JSON、URL、token 或代码内容时不再撑坏页面与其他列布局，并支持在可聚焦区域内横向滚动。

### Security

- 本地 UI 的 CSP 不再允许内联脚本或内联样式，仅加载单二进制内嵌的同源哈希资源。

## [1.2.0] - 2026-07-11

### Added

- 本地回环图形化管理界面：`repolens ui` 可打开本机 Git 工作树，编辑仓库配置域、校验并预览 YAML diff、确认原子写入，以及构建到用户缓存目录。
- UI 配置编辑覆盖 `site`、`ignore`、`render`、有序 `rules`、`theme`、`view` 和 `agent`；规则支持新增、删除与上下移动，`source`、`output`、`access` 不会暴露或改写。

### Security

- UI 仅接受 loopback 监听地址，变更 API 需要启动时生成并注入页面的 CSRF token。

## [1.1.2] - 2026-07-09

### Added

- 站点根生成自包含 `404.html`：修复 Cloudflare Pages 等托管对未命中路径回退根 index.html 时产生的 `view/view/…` 无限重定向（常见诱因：文档里指向仓库之外的相对链接）；仓库自带根 404.html 时镜像优先。

### Changed

- README 不再硬编码版本号，指向 Releases 页（避免随发版遗漏更新）。

## [1.1.1] - 2026-07-09

### Fixed

- Markdown 站内链接显式指向 `<目录>/index.html` 时被改写成不存在的 `view/<目录>/index.html/`（404）；现正确落到该目录的浏览页 `view/<目录>/`（index.html 按 Web 惯例并入目录页，#9 规则）。

## [1.1.0] - 2026-07-09

### Added

- 文件树混合形态（spec 010）：宽屏可一键完全收起获得全宽阅读，收起后 ☰ 浮出覆盖层树（遮罩/Esc/固定还原/点链接自关）；窄屏默认浮动；收起偏好跨会话保持；禁 JS 退化为固定侧栏。
- 顶部工具栏（spec 011）：树开关、前进/后退、面包屑+文件名、TOC 开关、字号缩放（90–125%）、布局宽度三档（窄/默认/全宽）、页面信息面板（路径/类型/大小/最后更新/commit，取代正文页脚 meta）、源码视图、下载菜单、搜索——按页面类型显隐，偏好持久化。
- 源码子页（spec 011）：Markdown 与 HTML 页新增 `view/<path>/source/`，chroma 高亮查看原始源码（补上 v1 缓行的"查看源码"入口）。
- pjax 站内导航（spec 011）：树/面包屑/正文链接换页不整页刷新，前进后退正常，失败自动回退整页；锚点跳转不再被顶栏遮挡。
- 站内搜索（spec 012）：构建期生成 `search.json`（文件名＋标题级索引，中文可检索），`/` 唤起搜索模态，键盘导航，文件与章节分组展示、命中高亮；索引懒加载；`view.search` 可关。
- 主题 UI 内置中英双语文案，由 `site.language` 驱动；新增 `view.toc_panel: floating | inline` 配置（TOC 默认改为右上浮动面板，`inline` 保持 v1 内联形态）。

### Changed

- Markdown 页内联 TOC 盒默认改为浮动面板形态（配置可退回）；正文页脚的更新时间/commit 行移入工具栏 ⓘ 信息面板。

## [1.0.1] - 2026-07-07

### Added

- 多平台预编译分发（spec 009）：GoReleaser 流水线（darwin/linux/windows × amd64/arm64）、Homebrew tap（`brew install conteman/tap/repolens`）、Scoop bucket（寄宿 tap 仓 `bucket/`）、Release 页直接下载。
- `repolens upgrade`：直装用户自更新（GitHub 域白名单 + sha256 校验 + 原子替换）；brew/scoop 安装自动提示对应升级命令；`--check` 只查不更。联网仅发生在显式执行时。

### Fixed

- `go install` 安装的二进制 `repolens version` 不再显示 `dev`（回退读 build info）。
- macOS cask 安装的未签名二进制被 Gatekeeper 拦截：装后钩子去 quarantine。

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

[Unreleased]: https://github.com/ConteMan/repolens/compare/v1.8.0...HEAD
[1.8.0]: https://github.com/ConteMan/repolens/compare/v1.7.0...v1.8.0
[1.7.0]: https://github.com/ConteMan/repolens/compare/v1.6.3...v1.7.0
[1.6.3]: https://github.com/ConteMan/repolens/compare/v1.6.2...v1.6.3
[1.6.2]: https://github.com/ConteMan/repolens/compare/v1.6.1...v1.6.2
[1.6.1]: https://github.com/ConteMan/repolens/compare/v1.6.0...v1.6.1
[1.6.0]: https://github.com/ConteMan/repolens/compare/v1.5.0...v1.6.0
[1.5.0]: https://github.com/ConteMan/repolens/compare/v1.4.1...v1.5.0
[1.4.1]: https://github.com/ConteMan/repolens/compare/v1.4.0...v1.4.1
[1.4.0]: https://github.com/ConteMan/repolens/compare/v1.3.1...v1.4.0
[1.3.1]: https://github.com/ConteMan/repolens/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/ConteMan/repolens/compare/v1.2.1...v1.3.0
[1.2.1]: https://github.com/ConteMan/repolens/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/ConteMan/repolens/compare/v1.1.2...v1.2.0
[1.1.2]: https://github.com/ConteMan/repolens/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/ConteMan/repolens/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/ConteMan/repolens/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/ConteMan/repolens/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/ConteMan/repolens/releases/tag/v1.0.0
