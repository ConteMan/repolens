# M6 UI 合同状态与缺口

状态基于当前代码静态核对；“已实现”仅表示所列最小合同已有对应实现与测试，不延伸为未核验的浏览器体验声明。

GitHub 跟踪：[#28 ui: align config API contracts before freezing the design baseline](https://github.com/ConteMan/repolens/issues/28)。

| 编号 | 最小合同 | 当前事实 | 状态 | 下一步 |
|---|---|---|---|---|
| CG-01 配置 document | 导出仓库域未合并 document、字段 presence、可预览 YAML；非受控节点不被表单覆盖 | `internal/config/repository_document.go` 已提供 `RepositoryDocument`、指针字段、AST 读写、YAML、revision，并由 `repository_document_test.go` 覆盖未受控节点保留；但 `internal/ui/ui.go` 的 `projectOpenResponse` 只暴露 `settings` 与 `revision`，没有有效默认值、字段来源或读取 warning | 部分实现 | 为 open 响应定义 `effective`、`sources`、`warnings` 的最小结构；明确“文件未设置”与“使用有效默认值”的呈现 |
| CG-02 结构化校验 | 返回 `{path,code,message}` 并明确 blocking/warning，前端可关联具体字段 | `config.RepositoryValidationIssue` 与 `ValidateRepositorySettings` 已结构化校验可写字段；但 `validateConfig`/`prepareWrite`/`commitConfig` 将错误折叠为通用 `validation_failed`，`writeError` 没有 `field` 或 `issues`，前端 `APIError` 与 `Status` 只能显示全局 message | 部分实现 | 保留并序列化 validation issues；确定单个 `field` 或 `issues[]` 格式；前端建立字段错误映射、页面摘要与焦点恢复；warning 另行返回，不与 blocking 混用 |
| CG-03 写入并发 | SHA-256 revision、unified diff、同目录临时文件与 rename 原子提交；冲突不得覆盖 | `RepositoryDocument.Write` 比较 revision 并原子写入；`internal/ui/ui.go` 已提供 prepare diff、明确 confirm、revision 冲突和受信域预览脱敏，相关 Go 测试已覆盖 | 已实现 | UI 仍需把 `revision_conflict` 设计为可执行的“重新读取”恢复路径，并补浏览器级冲突验收 |
| CG-04 构建服务 | 不调用自身 CLI；返回阶段、Stats、warnings、错误及日志尾部 | `internal/ui/build.go` 直接复用 source/config/theme/site，已有 opening、loading_config、loading_theme、building、completed、failed，且 API 返回 Stats、Warnings、Error、OutputPath；当前 operation 没有日志尾部字段 | 部分实现 | 先定义日志尾部是否为结构化事件或有界文本，再补 API、页面展开区和失败 fixture；在此之前设计稿不得把日志标为已交付 |
| CG-05 输出生命周期 | 仓库外 project-hash 缓存、同项目单 operation、失败保留成功产物、页面可定位最近成功结果 | `buildService.outputPath` 使用用户缓存根和 repository SHA-256；`repositories` 阻止同项目并发；临时目录替换失败会回滚已有输出。前端 `lastSuccess` 仅记录当前页面会话内完成的构建，open 新项目时清空，后端没有“按项目查询最近成功 operation”的合同 | 部分实现 | 定义最近成功结果的查询/恢复语义，覆盖页面刷新、进程重启和切换项目边界；补“已有成功后失败”的浏览器验收 |
| CG-06 预览服务 | 可停止的 preview session，不依赖 UI 进程 cwd 或 stdout | Spec 013 明确本期不启动 serve、不提供预览链接 | 延期 | 后续独立 spec，不在当前画板中以 disabled 控件暗示可用 |
| CG-07 路径浏览 | 目录浏览、home 限制、符号链接和权限语义 | 当前只接受既有绝对目录，`loadDocument` 做绝对路径与目录检查；没有目录浏览 API | 延期 | 后续独立 spec；当前 Project Open 只设计文本输入、加载与错误恢复 |

## 当前跨层不一致

以下项目应先收口合同，再把对应 Pencil 状态标为可实现：

1. `docs/specs/013-config-ui.md` 声明 `project/open` 返回原始 document、有效 config、warnings 和 revision；当前 `internal/ui/ui.go` 实际只返回 repository settings 与 revision。
2. Spec 声明 API 错误包含 `field?`，当前 `writeError` 只有 `code` 与 `message`，前端也未保存字段路径。
3. Spec 的构建结果包含“日志尾部”，当前 `buildOperation` 与 `buildResponse` 均无日志字段。
4. “失败保留最近一次成功结果”的文件系统行为已有回滚保护，但 UI 可恢复范围目前仅限当前页面会话，不能据此宣称刷新或重启后仍可查询。

## 收口顺序

1. 先完成 CG-02：字段错误是 Project Open、Config Edit 和 Diff 恢复体验的共同依赖。
2. 再完成 CG-01 的 open warnings / effective defaults / sources，避免把合并值误画成仓库原值。
3. 明确 CG-04 日志尾部是否仍属 v1.2 合同；若保留，先定义有界数据结构再设计页面。
4. 明确 CG-05 最近成功结果的持久范围，再设计失败恢复卡。
5. CG-03 只补浏览器恢复路径与验收，不重复改写已存在的 revision/原子写入实现。
