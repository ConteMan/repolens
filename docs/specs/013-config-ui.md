# 013: 图形化管理界面（repolens ui）

- 状态：已实现
- 关联：roadmap M6、specs 002 / 005 / 007、ADR-005（配置双信任域）

## 问题

配置只能手写 `.repolens.yml`，选择项目、触发构建都是命令行操作——对不熟悉终端与 YAML 的用户不友好。需要一个图形界面完成“选择项目 → 调配置 → 构建结果”的完整闭环。

## 行为

第一期只交付本地仓库的最短闭环：输入绝对本地工作树路径 → 编辑仓库配置域 → 确认 YAML diff → 写入 → 构建并查看结果。远程仓库、最近项目、目录浏览器、预览服务和外部可信配置均在后续独立收口。

1. **入口**：`repolens ui [--addr 127.0.0.1:8799]`。仅接受 loopback 地址；启动后打开浏览器。界面使用 React、TypeScript、Vite 与 Base UI；构建后的静态资源通过 `go:embed` 编入二进制，用户运行时不需要 Node。
2. **项目选择**：用户输入或粘贴绝对本地目录路径；服务端验证目录可由 `source.Open(..., Worktree: true)` 读取。第一期不实现目录树、主目录限制或远程 URL，因此不伪造尚不存在的文件系统浏览 API。
3. **配置表单**：读取仓库根 `.repolens.yml`，编辑仓库内可写的真实字段：`site`、`ignore`、`render`、`rules`、`theme`、`view`、`agent`。表单同时接收仓库未合并值、有效配置、字段来源与读取 warning：未设置字段显示其有效默认值和 `default` 来源，但保持空值，绝不把有效值静默写回仓库；用户把已设置字段切回“使用默认值”时，写入候选必须删除对应受控 YAML 节点。`source`、`output`、`access` 受信任域仍须原样保留；未知字段继续遵循写入确认中的非保留承诺。加载、空配置、字段校验失败和写入失败均须在页面内可见。
4. **规则编辑**：`rules` 是保序列表，可新增、删除、上移和下移；每条只允许 schema 已有字段 `match`、`render`、`markdown`、`html`、`code`、`max_file_size`。窄屏用独立子页，不提供 drag-and-drop。主题字段仅允许 `vars`、`css`、`templates`。
5. **写入确认**：表单不会直接写盘；先生成候选规范化 YAML 和 unified diff，明确目标 `<repo>/.repolens.yml`，用户明确确认后才原子覆盖。读取时的 revision 与提交时不同则返回冲突，不得覆盖外部修改。输出不承诺保留注释、空行、原键顺序或未知字段，确认页必须显示该影响。
6. **构建结果**：确认后的配置可触发与 `build` 同一管线的工作树构建，输出根由 UI 管理在仓库外的用户缓存目录，绝不写入仓库或 `output` 信任域。页面展示结构化阶段、Stats、Warnings、Error 与完整缓存路径，不复制 CLI stdout 或承诺通用日志尾部。构建采用临时目录原子替换，失败不得破坏磁盘上已有的成功产物；页面只承诺在当前 UI 页面会话内保留最近一次成功结果，刷新页面、重启进程或切换项目后的查询与恢复不在本期合同内。本期不启动 `serve`，不内嵌或新标签预览。
7. **安全**：默认只绑 `127.0.0.1`；进程生成随机 CSRF token 并注入首个 HTML，所有变更 API 要求该 token。无远程暴露开关、无认证旁路、无静默写入。

## 接口契约

```go
package ui // internal/ui

type Options struct{ Addr string }
func Run(ctx context.Context, opts Options) error // 阻塞直到 ctx 取消
```

- CLI 新增 `ui` 子命令，只调用 `ui.Run`；`internal/ui` 不得反向依赖 `internal/cli`。
- 内部 API 限定为 `POST /api/project/open`、`POST /api/config/validate`、`POST /api/config/prepare-write`、`POST /api/config/commit`、`POST /api/build`、`GET /api/build/{id}`。API 仅服务内嵌前端，不承诺稳定；一般错误为 `{code,message,field?}`，配置校验失败额外返回 `issues: [{path,code,message,severity}]`。字段路径使用 YAML 风格（如 `rules[1].match`），`severity` 为 `error` 或 `warning`；只有 `error` 阻止校验、diff 与写入。
- `project/open` 返回同一仓库配置快照的未合并 `settings`、合并后的 `effective`、字段到 `repository | default` 的 `sources`、读取 `warnings` 和 `revision`；读取期间 revision 变化时必须有界重试或拒绝，不能混合两个快照。`effective` 物化全局有效配置，但 `rules` 保持 presence-safe 的级联 patch 形状：没有具体文件路径时不得把规则中未设置的叶子伪装成有效零值。`validate` 返回规范化后的仓库 settings 与结构化问题，绝不写盘；`prepare-write` / `commit` 把请求视为 UI 受控字段的完整目标状态，`null` 删除对应受控 YAML 节点，同时保留 `source`、`output`、`access` 受信任域；未知字段仍不承诺保留。`prepare-write` 返回 before/after/unified diff；`commit` 必须有 `confirm: true` 并比较 revision；`build` 每项目同时最多一个 operation，忙时返回 `409`。
- 复用 `source.Open`、`config.Load`、`theme.New`、`site.NewBuilder(...).Build`，不得调用自身 CLI 或解析 stdout。`internal/ui` 的 package-private build service 返回阶段、`site.Stats`、warning 与错误，不复制 Cobra 的业务逻辑；`context.Context` 贯穿取消。
- `internal/config` 提供仓库域未合并 document 的解析、结构化校验、规范化 YAML、revision 安全写入，以及 UI 完整目标状态的 replace/clear 接口；公开 `Load` 的多层合并结果只用于有效值展示，不能作为回写模型。接口细节见 [UI 合同状态](../design/ui/contract-gaps.md)。

## 边界与非目标

- 不做原生桌面壳、远程仓库 URL、最近项目、服务端目录浏览器、多用户/远程部署、站点托管；
- 不做外部 `--config`、`source` / `output` / `access` 的可信域编辑、YAML 无损 round-trip 或仓库内容编辑；
- 不做内嵌 `serve`/watch、预览 session 或浏览器预览链接；用户自选本地输出目录由 [Spec 014](014-ui-session-output.md) 取代本期非目标；
- 不做通用构建日志流、页面刷新或进程重启后的最近成功 operation 查询；
- CLI 仍是一等公民，UI 不新增 CLI 做不到的构建语义。

## 验收

- 新用户路径：`repolens ui` → 输入本地 Git 工作树绝对路径 → 改标题与一条规则 → 校验 → diff 确认写回 → 构建结果，全程无终端交互；
- 写回后 `config.Load(repoRoot, "", config.Flags{})` 的仓库域语义等价于确认草稿，且不得写入 `source`、`output`、`access`；
- 空配置、有效默认值与字段来源、仓库 warning、已设置字段恢复默认值、三个字段错误及焦点恢复、写入确认、打开期间配置变化、revision 冲突、构建成功含 warning、构建失败都具有真实 fixture 与自动化测试；
- 相对路径、非目录、无 token/错误 token、非 loopback 地址和同项目并发构建均被拒；
- 质量门禁全绿：API 与路径安全单测、配置 round-trip 测试、一条浏览器端到端冒烟。
