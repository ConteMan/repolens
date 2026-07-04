# repolens

**中文** | [English](README.en.md)

> 把任意 Git 仓库渲染成可浏览的静态站点——零配置。

repolens 以 Git 仓库原样为内容源构建静态网站：Markdown 渲染（TOC、Mermaid）、HTML 文件直接预览、代码文件语法高亮，全部通过常驻文件树导航。访问者在浏览器里阅读一切——无需 clone、无需本地工具。AI Agent 则获得每个文件的干净纯文本视图。

**状态：早期设计 / v1 之前。** 架构与路线图见 [`docs/`](docs/README.md)，目前尚不可用。

## 为什么

文档往往已经在 Git 仓库里，但舒服地阅读它需要 clone 仓库、本地打开 HTML、找工具预览 Markdown——这把协作者、客户、非技术读者和 AI Agent 都挡在了外面。现有站点生成器（MkDocs、Docusaurus 等）只解决"为它们而组织"的仓库。repolens 渲染**任意仓库，不做任何修改**。

## 设计要点

- **双层输出** —— 仓库的逐字节*镜像层*（你的 HTML 相对链接天然可用；每个文件都有稳定的 raw URL）＋ `/view/` 下带导航框架的*浏览层*。
- **预渲染 MPA** —— 每个页面都是真实 URL，在任何静态托管上开箱可用；一层轻量原生 JS 负责文件树状态与流畅导航。全程无 Node 工具链。
- **规则化渲染** —— 有序级联的规则模型（`glob → 管线参数`）按路径定制渲染；默认值足够合理，可以完全不写配置。
- **Agent 友好** —— 每次构建生成 `llms.txt`、可选 `llms-full.txt` 和机器可读的 JSON 站点索引。
- **随处部署** —— 输出是纯静态目录：Cloudflare Pages、GitHub Pages、对象存储、nginx。全相对链接、零外部请求，可置于任何认证代理之后。
- **单一二进制** —— Go 编写，模板与资源全部内嵌。

## 快速开始（规划中的 CLI）

```sh
repolens build git@github.com:you/your-docs.git   # → ./dist
repolens serve ./local-repo                        # 本地预览，实时重建
```

## 文档

- [架构总览](docs/design/architecture.md)
- [配置模型](docs/design/config.md)
- [架构决策记录（ADR）](docs/decisions/README.md)
- [路线图](docs/roadmap.md)

## 参与贡献

见 [CONTRIBUTING.md](CONTRIBUTING.md)。本项目以 AI Agent 优先的方式开发，[AGENTS.md](AGENTS.md) 是编码 Agent 的入口。

## 许可证

[MIT](LICENSE)
