# 架构总览

> 状态：已定稿（2026-07-04）。变更本文档描述的约定需先补充或修订 ADR。

## 定位

repolens 是一个"仓库 → 静态输出目录"的**构建器**（不是运行时服务）：以任意 Git 仓库为内容源，不要求仓库做任何配合，生成可部署到任意静态托管（Cloudflare Pages、GitHub Pages、对象存储、nginx）的浏览站点。部署由用户自行完成，repolens 只保证输出在各平台上开箱可用。

三个核心价值，单独看均有已有方案，组合是空白：

1. **零配置、仓库原样渲染**——区别于 MkDocs / Docusaurus 等"仓库为工具而组织"的文档站生成器；
2. **HTML 文件可直接预览**——GitHub 只展示 HTML 源码，现有方案（htmlpreview / raw.githack）临时且不体面；
3. **Agent 友好**——每个文件有稳定纯文本 URL，站点自带机器可读索引。

非目标：商业化、参与部署流程、多租户托管服务。

## 双层输出（ADR-001）

```
dist/
├── <repo 原样路径>            # 镜像层：文件原样复制，路径一字不改
├── view/<repo 路径>[/]        # 浏览层：生成的阅读页面
├── llms.txt / llms-full.txt   # Agent 视图
├── index.json                 # 机器可读站点索引
├── _assets/                   # 站点自身的 CSS / JS / 字体（非仓库内容）
└── robots.txt
```

- **镜像层**：仓库文件逐字节复制。`docs/report.html` 位于 `/docs/report.html`，其相对引用的资源和内链天然成立——"HTML 内链可跳转"靠**不破坏**实现，而非改写。镜像层同时就是 Agent / curl 的 raw 访问层。
- **浏览层**：带站点框架（文件树侧栏、面包屑、顶栏）的生成页面。Markdown 渲染为 HTML，代码文件带行号高亮展示，HTML 文件以 iframe 嵌入（指向镜像层原路径），目录生成 listing 页。
- 为什么不能单层：渲染后的 HTML 若放在 `foo.md` 原路径，静态托管按扩展名返回 `text/markdown`，浏览器行为不可控。路径镜像给原始内容、前缀路径给渲染视图，是唯一在所有静态平台一致成立的方案。代价是内容存两份，对文档类仓库可接受。

## URL 约定

| 内容 | URL | 落地文件 |
|---|---|---|
| 原始文件（镜像层） | `/<repo 路径>` | `dist/<repo 路径>` |
| 文件浏览页 | `/view/<repo 路径>/`（保留原扩展名 ＋ 尾斜杠，如 `/view/docs/foo.md/`） | `view/<repo 路径>/index.html` |
| 目录页 | `/view/<目录>/` | `view/<目录>/index.html` |
| 站点根 | `/` 跳转 `/view/`，渲染 `site.home`（默认根 README） | meta-refresh 页 |

浏览页落地为 `index.html` 而非裸文件：若 `view/docs/foo.md` 作为文件输出，静态托管同样按 `.md` 扩展名给错 Content-Type——镜像层遇到的问题在浏览层同样存在，尾斜杠目录形式是唯一可移植解（2026-07-04 细化，见 ADR-001）。

全站**只使用相对链接**，不烧录 base URL——部署在任意子路径、任意反代之后都成立。

## 浏览层交互（ADR-002）

**预渲染 MPA ＋ 薄增强层**。交互形态参照 docu.md（文件树侧栏 + 内容区即时切换 + HTML 带样式直出），但渲染发生在构建期而非浏览器：

- 每个文件预渲染为独立页面，无 JS 也可读，每页是真实 URL；
- 共享一份手写原生 JS（无框架、无打包器）：sessionStorage 保持文件树展开状态、可选 pjax 式内容区局部替换；
- 文件树带类型图标、当前文件高亮、目录折叠，位置可配置（默认左侧）；
- HTML 浏览页 = iframe 指向镜像层原路径，样式与站点框架互不污染；工具栏提供"全屏打开原文件"与"查看源码"。单仓库单租户，iframe 默认同源不加 sandbox，配置留禁脚本选项；
- 目录页：有 README 渲染 README，无则显示文件列表（名称、类型、大小、git 最后修改时间）；
- Mermaid 例外：构建期渲染需拖 headless browser，不值得。vendor 一份本地 mermaid.js，仅含图表页面懒加载（不违背零外部请求约束，含图表页面的无 JS 可读性打折，接受）。

## 渲染管线（ADR-003）

内容集合来自 `git ls-tree`（而非工作目录）：`.gitignore` 天然生效，构建结果只由 `repo + ref` 决定，可复现。本地未提交内容预览走 `serve --worktree` 显式开关。

规则模型：**(glob 匹配 → 处理管线 → 参数)**，按序级联、后者覆盖前者（editorconfig 语义），全局默认等价第 0 条规则。详见 [config.md](config.md)。

- Markdown：goldmark ＋ 扩展（GFM、脚注、TOC、标题锚点、front-matter、Mermaid）。指向仓库内 `.md` 的相对链接在 **AST 层**（ASTTransformer）改写为 `/view/` 路径，不做渲染后文本替换；
- 代码：chroma 构建期高亮，Markdown 代码块与独立代码文件页共用；
- 用户 HTML：不解析不修改，镜像原样 + iframe 浏览页；
- 超过 `max_file_size` 的文件与二进制文件：只进镜像层，浏览页给下载链接；
- git 元数据：一次 `git log --name-status` 全量建映射（禁止逐文件 git log），每页注入最后修改时间，可选 commit 链接。

## Agent 视图

- `llms.txt`：站点说明 ＋ 路径约定（浏览页在 `/view/`，原始文件在原路径）＋ 按目录树组织的内容清单；
- `llms-full.txt`（可选）：全部文本内容拼接，有大小上限；
- `index.json`：文件树 ＋ 类型 ＋ 大小 ＋ git 最后修改时间，一次请求拿全貌；
- 每个浏览页 `<head>` 含 `<link rel="alternate" type="text/markdown">` 指向镜像层原文件。

## 访问控制

生成器本身认证无关，但输出为"被保护"而设计：

1. **平台层（推荐主方案）**：Cloudflare Access、Vercel 密码、nginx basic auth、内网。落成两条硬约束：全相对链接；**零外部请求**（字体、脚本、样式全部 embed 并随站点输出，私有站点不向任何 CDN 泄露访问痕迹）。Agent 访问私有站走平台 service token；
2. **项目自带（可选）**：客户端加密（PageCrypt 思路：构建期密码派生密钥加密页面，浏览器 WebCrypto 解密）。与 Agent 视图**互斥**——加密路径与 agent 输出的交集在构建时检测并告警；
3. **软保护（默认开）**：`noindex` meta ＋ robots.txt。

## 技术栈（ADR-003 / ADR-004）

Go 单二进制，全部资源 go:embed。直接依赖控制在个位数，每个都有大型项目背书：

| 组件 | 选型 |
|---|---|
| Markdown | yuin/goldmark（＋ abhinav 系扩展） |
| 高亮 | alecthomas/chroma |
| CLI | spf13/cobra |
| YAML | goccy/go-yaml（gopkg.in/yaml.v3 已归档） |
| 文件监听 | fsnotify/fsnotify |
| Git | shell out 系统 git（认证复用用户已有配置，见 ADR-004） |
| 生成站点增强层 | 手写原生 JS/CSS，零运行时依赖 |
| 本地管理界面 | React + TypeScript + Base UI + Vite；静态产物 go:embed（ADR-006） |

## 包结构

| 包 | 职责 |
|---|---|
| `cmd/repolens` | main 入口 |
| `internal/cli` | cobra 命令：build / serve / init / version |
| `internal/config` | 配置加载、双信任域合并、规则级联求值 |
| `internal/source` | git 操作：clone / fetch / ls-tree / log 映射 |
| `internal/render` | goldmark 管线、chroma、链接改写 |
| `internal/site` | 站点组装：镜像层、浏览层、llms.txt、index.json、robots.txt |
| `internal/theme` | 内置模板（layout / page / dirlist / tree）与静态资源，go:embed |
| `internal/server` | 本地预览：net/http ＋ fsnotify 重建 |

主题定制三级：CSS 变量 → 追加 CSS → 模板覆盖（逃生舱）。内置模板保持少而稳定，模板名即公开 API。
