# 013: 图形化管理界面（repolens ui）

- 状态：草稿
- 关联：roadmap M6、specs 002 / 005 / 007、ADR-005（配置双信任域）

## 问题

配置只能手写 `.repolens.yml`，选择项目、触发构建都是命令行操作——对不熟悉终端与 YAML 的用户不友好。需要一个图形界面完成"选目录 → 调配置 → 构建/预览"的完整闭环。

## 行为

1. **入口**：`repolens ui [--addr 127.0.0.1:8799]`，本地起管理界面并自动打开浏览器。界面资源 go:embed，延续单二进制、无 Node 运行时依赖（前端为手写 HTML/CSS/JS，复用主题的设计语言）。
2. **项目管理**：
   - 服务端目录浏览器选择本地仓库路径（限制在用户主目录树内），或粘贴远程 git URL；
   - 最近项目列表（持久化于 `~/.config/repolens/projects.json`，含各项目上次输出目录与配置摘要）。
3. **可视化配置**：按 spec 002 的 schema 生成表单——站点信息（title/home/language）、视图（tree_expand_depth/search）、规则级联（glob 列表的增删排序 ＋ 每条的管线参数）、主题（vars 色板 / custom css 路径 / 模板目录）、访问（noindex）、Agent 产物三开关；
   - 保存写回仓库根 `.repolens.yml`（带注释头标明"由 repolens ui 生成"）；也可另存为外部配置文件（`--config` 信任域）；
   - 写回前展示 YAML diff 供确认。
4. **构建与预览**：界面内触发 build（输出目录可选），实时流式日志与 Stats/Warnings 展示；一键启动 serve 预览（复用 spec 007 管线）并内嵌打开。
5. **安全**：默认只绑 127.0.0.1；所有变更请求校验随机 token（启动时生成、注入页面）；不提供任何远程暴露选项。

## 接口契约

```go
package ui // internal/ui

type Options struct{ Addr string }
func Run(ctx context.Context, opts Options) error // 阻塞直到 ctx 取消
```

- 内部通过 HTTP JSON API 驱动（/api/projects、/api/config、/api/build、/api/serve），API 仅服务于内嵌前端，不承诺对外稳定；
- 复用 `config.Load` / `site.Builder` / `server.Run`，不复制业务逻辑；配置写回需要 config 包新增"结构 → YAML（含注释）"的序列化助手，签名实现期定、回写本 spec。

## 边界与非目标

- 不做原生桌面壳（Wails 等）——先验证 Web UI 交互，见效后再议（用户已确认两步走的第一步）；
- 不做多用户 / 远程部署管理 / 站点托管；
- 不做配置项之外的仓库内容编辑；
- CLI 仍是一等公民，ui 不新增任何 CLI 做不到的能力。

## 验收

- 全新用户路径实测：`repolens ui` → 选本地仓库 → 表单改标题与一条规则 → diff 确认写回 → 构建 → 预览，全程无终端交互；
- 写回的 `.repolens.yml` 可被 CLI 正常加载且语义等价；
- 目录浏览器无法越出主目录树；无 token 的 API 请求被拒；
- 断开 ui 进程不影响已构建产物与已启动的独立 serve；
- 质量门禁全绿（API 层单测 ＋ 一条端到端冒烟）。
