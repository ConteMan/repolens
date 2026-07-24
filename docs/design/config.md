# 配置模型

> 状态：已定稿（2026-07-04）。schema 字段在 v1 实现期间允许微调，语义原则（三条设计原则、信任域划分）变更需修订 ADR-005。

## 设计原则

1. **零配置可用**：`repolens build <repo-url>` 不带任何配置即产出完整站点，配置只做覆盖；
2. **双信任域**：仓库内 `.repolens.yml`（仓库作者写）与构建时外部配置 `--config`（构建者写）。渲染他人仓库时，仓库内配置是**不可信输入**：可以决定"怎么渲染自己"，不能决定"从哪拉取、往哪输出"。`source` / `output` / `access` 三段只认外部配置与 CLI，仓库内出现则忽略并告警。优先级：CLI > 外部配置 > 仓库内配置 > 默认值；
3. **级联规则**：按序全部匹配、后写覆盖先写（editorconfig 语义），全局 `render` 段等价第 0 条规则。

## Schema

```yaml
# ── 仅外部配置有效 ──────────────────────────
source:
  repo: git@github.com:me/docs.git   # 或本地路径；CLI 位置参数可覆盖
  ref: main                          # 分支 / tag / commit

output:
  dir: dist
  base_path: /                       # 全相对链接下通常无需设置

access:
  noindex: true                      # 默认 true：robots.txt + meta noindex
  encrypt:
    enabled: false
    paths: ["internal/**"]           # 空 = 全站
    password_env: REPOLENS_PASSWORD  # 只存环境变量名，密码永不落配置文件

# ── 仓库内外均可，外部覆盖仓库内 ─────────────
site:
  title: ""                          # 默认取仓库名
  language: zh-CN
  home: README.md                    # 站点根 /view/ 渲染的文件

ignore:                              # 两层输出都排除；也是防泄漏的第一道闸
  - "node_modules/**"
  - ".github/**"

render:                              # 全局默认（第 0 条规则）
  markdown:
    toc: true                        # 标题 ≥ 3 个才出 TOC
    anchors: true
    mermaid: true
    math: false
    frontmatter_title: true
  html:
    view: embed                      # embed=iframe 进框架 / direct=只链镜像 / source=按代码展示
  code:
    line_numbers: true
    theme: github                    # chroma 主题名
  max_file_size: 5MB                 # 超限只进镜像层，浏览页给下载链接

rules:                               # 按序级联，覆盖 render 默认
  - match: "reports/**/*.html"
    html: { view: direct }
  - match: "legacy/**"
    render: false                    # 只镜像，不生成浏览页
  - match: "specs/**/*.md"
    markdown: { math: true }

theme:
  vars:                              # 注入 CSS 变量
    accent: "#0969da"
    sidebar-width: 280px             # 桌面固定树的站点作者默认值；访问者可在浏览器本地覆盖
  css: custom.css                    # 追加样式表（仓库内相对路径）
  templates: .repolens/templates/    # 同名覆盖内置模板

view:
  tree_position: left                # left / right
  tree_expand_depth: 2
  toc_panel: floating                # floating / inline；默认 floating
  search: true                       # 站内搜索入口与 search.json；默认 true

agent:
  llms_txt: true
  llms_full:
    enabled: true
    max_size: 2MB
  index_json: true
```

## 语义说明

- **`ignore` vs `rules[].render: false`**：前者"文件当不存在"（两层都排除），后者"可访问但无阅读页"（镜像层保留）。`ignore` 同时是安全功能，文档按此定位撰写；
- **加密与 Agent 视图互斥**：`access.encrypt.paths` 与 agent 输出的交集构建时检测并告警（lint 规则）。私有站点优先推荐平台层认证（Agent 可走 service token），客户端加密仅留给托管平台完全不可控的场景；
- **站内搜索是浏览层能力**：`view.search: true` 时输出站点根 `search.json` 并渲染工具栏 / 树顶搜索入口；设为 `false` 时两者都不生成。该开关不参与 `rules` 级联，且不受 `agent.index_json` 影响；
- **侧栏宽度分属两个层级**：`theme.vars.sidebar-width` 是站点作者随构建产物发布的默认值；访问者对桌面固定树的调宽只保存在按部署 base path 隔离的浏览器本地偏好中，不回写配置。浮动树宽度保持独立；
- **密码只引用环境变量名**，配置文件可安全提交；
- 主题定制三级：`theme.vars`（90% 需求）→ `theme.css` → `theme.templates`（逃生舱，模板名是公开 API，宁缺毋滥）。
