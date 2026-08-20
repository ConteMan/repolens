# repolens

**中文** | [English](README.en.md)

> 把任意 Git 仓库渲染成可浏览的静态站点——零配置。

repolens 以 Git 仓库原样为内容源构建静态网站：Markdown 渲染（TOC、Mermaid）、HTML 文件直接预览、代码文件语法高亮，全部通过常驻文件树导航。访问者在浏览器里阅读一切——无需 clone、无需本地工具。AI Agent 则获得每个文件的干净纯文本视图。

`build` 与 `serve` 可用，当前版本见 [Releases](https://github.com/ConteMan/repolens/releases/latest)；架构与路线图见 [`docs/`](docs/README.md)。

## 为什么

文档往往已经在 Git 仓库里，但舒服地阅读它需要 clone 仓库、本地打开 HTML、找工具预览 Markdown——这把协作者、客户、非技术读者和 AI Agent 都挡在了外面。现有站点生成器（MkDocs、Docusaurus 等）只解决"为它们而组织"的仓库。repolens 渲染**任意仓库，不做任何修改**。

## 设计要点

- **双层输出** —— 仓库的逐字节*镜像层*（你的 HTML 相对链接天然可用；每个文件都有稳定的 raw URL）＋ `/view/` 下带导航框架的*浏览层*。
- **预渲染 MPA** —— 每个页面都是真实 URL，在任何静态托管上开箱可用；一层轻量原生 JS 负责文件树状态与流畅导航。生成站点无前端框架或 Node 运行时。
- **规则化渲染** —— 有序级联的规则模型（`glob → 管线参数`）按路径定制渲染；默认值足够合理，可以完全不写配置。
- **术语标注与解释** —— 正文里把一个词标成术语，读者原地展开解释，不必跳转、不必离站。语法是标准 Markdown 链接，仓库在 GitHub 上照样正常。默认关闭。
- **Agent 友好** —— 每次构建生成 `llms.txt`、可选 `llms-full.txt` 和机器可读的 JSON 站点索引。
- **随处部署** —— 输出是纯静态目录：Cloudflare Pages、GitHub Pages、对象存储、nginx。全相对链接、零外部请求，可置于任何认证代理之后。
- **单一二进制** —— Go 编写，模板与资源全部内嵌。

## 安装

```sh
# macOS
brew install conteman/tap/repolens

# Windows（Scoop；或从 Releases 页直接下载 zip）
scoop bucket add conteman https://github.com/ConteMan/homebrew-tap
scoop install repolens

# 任意平台（需 Go 环境）
go install github.com/ConteMan/repolens/cmd/repolens@latest
```

升级：包管理器用户走 `brew upgrade` / `scoop update`；直接下载的用户执行 `repolens upgrade` 原地自更新。

## 快速开始

```sh
repolens build git@github.com:you/your-docs.git   # → ./dist
repolens build . -o dist --ref v2.0               # 本地仓库，指定 ref
repolens serve . --worktree                       # 本地预览（含未提交内容），改动自动重建
repolens ui                                       # 图形化配置并构建当前工作树
```

构建产物是纯静态目录，直接交给任何静态托管即可，见[部署指南](docs/deploy.md)。`repolens ui` 默认构建到本机缓存，也可为当前页面会话选择安全的自定义绝对输出目录；该选择不会写入仓库配置。渲染行为可通过仓库根的 `.repolens.yml` 定制（可选，零配置即有合理默认），见[配置模型](docs/design/config.md)。

## 术语标注与解释

技术文档的理解成本常常集中在两类词上：读者不熟的行业概念，以及**同名不同义**的词——同一个词在不同平台、不同上下文里指的不是一回事。正文展开解释会打断主线，集中的名词表要求读者中断阅读去跳转，外部链接则把人带离站点、且解释的是通用含义而不是「在这篇文档里它指什么」。

repolens 让作者把解释放在仓库里，读者在原地读到它：

**1. 正文标注**——标准 Markdown 链接，`term:` 加术语 key：

```markdown
买量渠道负责把用户带进 App，[广告聚合](term:mediation)平台负责决定展示哪家广告源的广告。
```

这在 GitHub 和任何 Markdown 渲染器下都是合法链接，仓库不会因为采用它而在别处变难看。显示文本与 key 解耦——正文写「渠道」「Channel」「媒体源」都可以指向同一条术语。

**2. 写术语库**——`.repolens/glossary/<key>.yml`，文件名即 key：

```yaml
title: 广告聚合
alias: Ad mediation
summary: 在多个广告源之间统一请求、竞价或瀑布选择，以提高填充率和收益。
warning: AdMob 既可能是聚合平台内的一个广告源，也可能自己作为主聚合平台。先确认它当前扮演哪种角色。
source: { label: AppLovin MAX 官方说明, url: https://developers.applovin.com/en/max/getting-started/ }
```

文档 front matter 里还能补本文语境的含义，或定义只在本页可见的私有术语：

```yaml
---
glossary:
  mediation:
    page: 主聚合平台决定展示哪家广告源的广告，它不负责判断最初是哪个买量渠道带来了用户。
---
```

`summary` 回答「这个词是什么」，换一篇文档仍然成立；`page` 回答「在这篇文档里它具体指什么」。区分这两者是这套格式的核心。

**3. 开启**——特性默认关闭：

```yaml
render:
  markdown:
    glossary: true
glossary:
  dir: .repolens/glossary
  strict: refs        # off / refs / complete
```

构建出的站点里，被标注的词点击即可展开解释抽屉，页尾同时生成本页术语表。**关掉 JavaScript、打印、被 Agent 抓取时解释都不会丢失**——抽屉读的就是页面里那份术语表，不存在第二份数据。术语还会进入站内搜索与 `llms.txt`。

`strict` 三档区分两种不同的问题：引用了不存在的术语（笔误，任何时候都让构建失败），和解释还没写完（写作中的正常状态，默认放行）。想在合并前拦住没写完的解释，让 CI 用外部配置覆盖成 `complete` 即可，作者本地仍可先标注、后补解释。

仓库里没有用到这个特性时，构建产物与该特性不存在时逐字节一致。

配套的 [Agent 操作规范](skills/repolens-glossary/SKILL.md) 可以直接交给编码 Agent，让它配合你整理术语、写入正确格式。它随二进制分发，一条命令装到你的 Agent 认得的目录：

```sh
repolens skill install glossary
```

命令按仓库里的 Agent 目录探测安装位置（`.claude/`、`.codex/`、`.cursor/`、`.github/`），都没探测到则写入中立路径 `.agents/skills/`；`--global` 装到个人目录。升级 repolens 后 `repolens skill update` 离线把副本更新到当前版本，改动过的副本不会被覆盖。

## 文档

- [架构总览](docs/design/architecture.md)
- [术语标注 Agent Skill](skills/repolens-glossary/SKILL.md)
- [配置模型](docs/design/config.md)
- [部署指南](docs/deploy.md)
- [架构决策记录（ADR）](docs/decisions/README.md)
- [路线图](docs/roadmap.md)

## 参与贡献

见 [CONTRIBUTING.md](CONTRIBUTING.md)。本项目以 AI Agent 优先的方式开发，[AGENTS.md](AGENTS.md) 是编码 Agent 的入口。

## 许可证

[MIT](LICENSE)
