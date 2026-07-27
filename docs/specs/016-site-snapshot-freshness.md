# 016: 站点快照新鲜度提示

- 状态：已实现
- 关联：roadmap M10、Issue #35、ADR-001、ADR-002、specs 005–007

## 问题

repolens 输出的是静态快照。访问者长期保持旧浏览页打开时，新构建即使已经部署到相同 URL，当前标签页也不会主动发现，因而可能长期停留在过期内容与导航状态。HTTP 缓存策略只能影响后续请求，不能通知已经打开的页面。

## 行为

1. **快照标识**：每次 `site.Builder.Build` 生成一个 128-bit 随机值并编码为 32 位小写十六进制字符串。它是不透明的构建快照标识，不承诺可排序、可反推时间或等同 Git commit；一次构建内所有浏览页和元数据资源必须使用同一个值，不同成功构建使用不同值。
2. **稳定元数据**：构建在站点根输出 `_assets/snapshot.json`，UTF-8 JSON 结构固定为 `{"snapshot":"<id>"}`。浏览页 `<body>` 通过 `data-snapshot-id` 携带当前 ID，并通过相对 `data-snapshot-src` 指向该资源；子路径部署不得需要 `base_path` 或绝对 URL。源仓库若包含同名镜像路径，构建必须在写输出前明确报冲突，不得覆盖仓库文件。
3. **探测调度**：
   - 页面可见且 `navigator.onLine !== false` 时每 60 秒检查；
   - `visibilitychange` 进入可见状态和 `online` 事件触发立即检查；
   - 页面隐藏、明确离线、已发现新快照或已有请求在途时不发请求；
   - 不在页面初次加载时额外请求，首次定时检查从加载后 60 秒开始。
4. **缓存绕过**：每次请求为元数据 URL 增加当前毫秒时间戳查询参数，并使用 `fetch(..., {credentials:"same-origin", cache:"no-store"})`。不要求托管平台采用特定响应头；响应非 2xx、JSON 无效、字段缺失和网络失败均静默降级。
5. **更新提示**：响应中的非空字符串 ID 与当前页面不一致时，显示唯一的固定位置提示。提示使用 `role="status"`、`aria-live="polite"` 与 `aria-atomic="true"`，不自动获得焦点；包含明确的“重新加载”按钮，点击调用 `window.location.reload()`。同一页面只展示一次，不自动刷新、不替换局部内容。
6. **PJAX**：PJAX 取回的目标页若携带不同快照 ID，应立即显示同一更新提示，避免把两个部署快照的局部 DOM 静默混合；正常同快照导航保持原行为。
7. **本地预览**：`repolens serve` 每次成功原子切换的新构建携带新 ID，已打开页面按相同静态探测机制发现更新。重建失败继续服务旧目录和旧 ID，不显示虚假的成功更新。
8. **多语言与主题**：内置中英文字符串分别为“此站点已有新版本。”/“A newer version of this site is available.”和“重新加载”/“Reload”。默认 `layout` 提供完整提示 DOM；自定义 `layout` 若要保留该能力，必须消费 `PageData.SnapshotID` 并保留与默认模板等价的数据属性和可访问提示结构。

## 接口契约

```go
package theme

type PageData struct {
    // 既有字段省略
    SnapshotID string
}
```

`site.Builder.Build` 的公开签名不变；快照 ID 仅是单次构建内部状态。`_assets/snapshot.json` 属 repolens 生成资源，不进入镜像层、Agent 索引或搜索索引。

## 边界与非目标

- 不参与、触发或观察部署流程，不增加运行时服务；
- 不使用 SSE、WebSocket、Service Worker、推送通知或托管平台 API；
- 不自动刷新，不热替换单独页面片段，不跨快照同步客户端状态；
- 本期不增加配置字段，60 秒周期和手动刷新是固定合同；
- 不保证离线时发现更新，不把探测失败展示成用户错误；
- 无 JavaScript 时不提供新鲜度检测，但页面仍完整可读可导航；
- 不对仓库镜像层的原始 HTML 注入检测逻辑。

## 验收

- Go 测试断言每次构建输出合法 `_assets/snapshot.json`，同次构建的所有浏览页 ID 一致，两次构建 ID 不同，镜像层同名路径被明确拒绝；
- 主题测试断言中英文提示、相对元数据 URL、ARIA 属性、CSS 响应式布局和 JS 调度/缓存绕过合同；
- Playwright 在根路径和子路径静态托管下替换元数据 ID，已打开页面显示唯一提示且不抢焦点，重新加载按钮可操作；
- Playwright 验证相同 ID 不提示、临时网络失败静默、PJAX 跨快照显示提示；
- `repolens serve` 既有原子切换测试继续通过，失败重建仍保留旧站点；
- 产物无外部请求，`gofmt -l .`、`go vet ./...`、`go test ./...`、`go build ./...` 全部通过。
