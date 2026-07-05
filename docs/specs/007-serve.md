# 007: 本地预览 serve（internal/server）

- 状态：已实现
- 关联：roadmap M3、specs 001、005

## 问题

`repolens serve [path]`：本地起 HTTP 服务预览构建产物，文件变化自动重建，用于日常写作与主题调试。

## 行为

1. **首次构建**：与 build 相同管线，输出到临时目录（`os.MkdirTemp`，退出清理）。默认 `--worktree=false` 走 git 树；`--worktree` 直接渲染工作目录（含未提交内容，spec 001 的 worktree 模式）。
2. **静态服务**：`net/http` ＋ `http.FileServer`，监听 `--addr`（默认 `127.0.0.1:8788`）。镜像层文件按扩展名正常给 Content-Type（`.md` 即 text/markdown——与线上静态托管行为一致，所见即所得）。启动打印可点击 URL。
3. **监听重建**：fsnotify 递归监听源目录（worktree 模式）或轮询 git HEAD 变化（git 模式，间隔 2s）；变化后 300ms 防抖全量重建到**新临时目录**，成功后原子切换服务根（失败保留旧站点并打印错误，不中断服务）。忽略 `.git/`、输出目录自身与 `ignore` 命中路径。
4. **退出**：SIGINT/SIGTERM 优雅关闭并清理临时目录。

## 接口契约

```go
package server

type Options struct {
    Addr     string
    Worktree bool
}

// Run 阻塞直到 ctx 取消。rebuild 由 cli 层闭包注入（复用 build 管线），
// 返回新的站点根目录。
func Run(ctx context.Context, opts Options, rebuild func(ctx context.Context) (dir string, err error)) error
```

## 边界与非目标

- 不做浏览器 live-reload 注入（v1 手动刷新；接口不排斥后续加 SSE）；
- 不支持对外暴露的生产服务（明确定位本地预览，默认只绑 127.0.0.1）；
- 不做部分重建。

## 验收

- 集成测试：起 serve（随机端口）→ GET 浏览页与镜像文件断言 200 与 Content-Type → 修改源文件 → 轮询断言新内容生效 → ctx 取消后端口释放、临时目录清理；
- 重建失败（如配置损坏）时旧站点仍可访问；
- `gofmt` / `go vet` / `go test` 通过。
