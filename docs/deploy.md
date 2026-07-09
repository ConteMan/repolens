# 部署指南

`repolens build` 的产物是一个纯静态目录（默认 `dist/`）：全相对链接、零外部请求、无服务端逻辑。任何能托管静态文件的地方都能部署，且**允许部署在子路径下**（如 `example.com/docs/`），无需任何配置。

部署前构建：

```sh
repolens build git@github.com:you/your-docs.git -o dist
```

## GitHub Pages

仓库里加一条 workflow（`.github/workflows/pages.yml`）：

```yaml
name: Deploy docs
on:
  push:
    branches: [main]
permissions:
  contents: read
  pages: write
  id-token: write
jobs:
  deploy:
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0   # git 元数据（最后修改时间）需要完整历史
      - uses: actions/setup-go@v6
        with:
          go-version: 'stable'
      - run: go run github.com/ConteMan/repolens/cmd/repolens@latest build . -o dist
      - uses: actions/upload-pages-artifact@v4
        with:
          path: dist
      - id: deployment
        uses: actions/deploy-pages@v4
```

## Cloudflare Pages

- **构建命令**：`go run github.com/ConteMan/repolens/cmd/repolens@latest build . -o dist`
- **输出目录**：`dist`
- 环境需启用 Go（Pages 构建镜像自带）；或在 CI 里构建后用 `wrangler pages deploy dist` 直传。

## nginx / 任意对象存储

产物即目录，拷过去即可：

```nginx
server {
    listen 443 ssl;
    server_name docs.example.com;
    root /srv/repolens/dist;
    index index.html;
}
```

对象存储（S3 / OSS / COS）开启静态网站托管后同步目录即可（`aws s3 sync dist/ s3://bucket/`）。

## 注意事项

- **访问控制**：站点本身无认证。需要限制访问时置于任何认证代理（Cloudflare Access、oauth2-proxy、内网 nginx auth）之后即可——全相对链接不受路径改写影响。
- **noindex**：配置 `access.noindex: true` 时产物自带 `robots.txt`（`Disallow: /`）与每页 `<meta name="robots" content="noindex">`。
- **MIME 类型**：镜像层按扩展名由托管方决定 Content-Type；无注册类型的文件（如 `.go`）多数托管会按下载处理，属预期行为（浏览页本身提供高亮阅读视图）。
- **404 行为**：产物自带 `404.html`，Cloudflare Pages / GitHub Pages 会对未命中路径直接返回它。没有该文件时部分托管（如 Cloudflare Pages）会把未命中路径回退成根 `index.html`（SPA 约定），与根跳转页叠加造成 `view/view/…` 无限重定向。
- **跨仓库链接**：文档里指向仓库之外的相对链接（如 `../另一个仓库/…`）无法在单仓站点上解析，repolens 会原样保留，线上表现为 404。需要跨仓引用时改用对方站点的完整 URL，或把多个仓库的构建产物按相同的相对结构部署在同一域名下。
- **Agent 入口**：站点根的 `llms.txt`、`llms-full.txt`、`index.json` 随构建生成，Agent 可直接抓取，无需渲染 HTML。
