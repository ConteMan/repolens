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
- **Agent 入口**：站点根的 `llms.txt`、`llms-full.txt`、`index.json` 随构建生成，Agent 可直接抓取，无需渲染 HTML。
