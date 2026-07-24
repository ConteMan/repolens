# M6 UI 合同状态与缺口

状态基于当前代码静态核对；“已实现”仅表示所列最小合同已有对应实现与测试，不延伸为未核验的浏览器体验声明。

历史跟踪：[#28 配置 API 合同收口](https://github.com/ConteMan/repolens/issues/28)、[#29 最终设计基线](https://github.com/ConteMan/repolens/issues/29)、[#30 React 实现与设计回归](https://github.com/ConteMan/repolens/issues/30)，均已完成。

| 编号 | 最小合同 | 当前事实 | 状态 | 下一步 |
|---|---|---|---|---|
| CG-01 配置 document | 导出同一快照的仓库域未合并 document、字段 presence、有效默认值、来源、warning 与 revision；受控字段可恢复默认值，受信任域不被表单覆盖 | `project/open` 通过 revision 有界重试返回 `settings`、`effective`、`sources`、`warnings` 与 `revision`；全局字段物化有效值，rules 保留 presence-safe 的级联 patch，避免把缺失叶子的 Go 零值误报为有效值；`RepositoryDocument.Replace` 为 UI 完整目标状态提供 nil 删除语义，通用 `Apply` 的 patch 语义不变；`source` / `output` / `access` 不进入 UI DTO 且写入时保留 | 已实现 | 最终基线已区分仓库值、有效值、规则 patch 与来源，不会把有效默认值静默写回 |
| CG-02 结构化校验 | 返回 `{path,code,message,severity}` 并明确 error/warning，前端可关联具体字段 | `validate` / `prepare-write` / `commit` 返回 `issues[]` 与兼容首项 `field`；前端 API 保留问题结构，提供页面摘要、字段内联错误、`aria-invalid` / `aria-describedby` 和首个错误焦点恢复；数组路径归一到对应编辑控件 | 已实现 | 最终 Pencil 基线和浏览器回归已覆盖错误摘要、字段映射与焦点恢复 |
| CG-03 写入并发 | SHA-256 revision、unified diff、同目录临时文件与 rename 原子提交；冲突不得覆盖 | `RepositoryDocument.Write` 比较 revision 并原子写入；`internal/ui/ui.go` 已提供 prepare diff、明确 confirm、revision 冲突和受信域预览脱敏，相关 Go 测试已覆盖 | 已实现 | 最终实现已提供 revision 冲突后的“重新读取”恢复路径，并有浏览器回归覆盖 |
| CG-04 构建服务 | 不调用自身 CLI；返回结构化阶段、Stats、warnings、错误与完整输出路径 | `internal/ui/build.go` 直接复用 source/config/theme/site，已有 opening、loading_config、loading_theme、building、completed、failed，且 API 返回 Stats、Warnings、Error、OutputPath | 已实现（合同收窄） | 2026-07-22 确认 Spec 013 不承诺通用日志尾部；若后续出现真实诊断需求，另开 Spec 定义有界结构和脱敏边界 |
| CG-05 输出生命周期 | 仓库外 project-hash 缓存、同项目单 operation、失败保留成功产物、当前页面会话可定位最近成功结果 | `buildService.outputPath` 使用用户缓存根和 repository SHA-256；`repositories` 阻止同项目并发；临时目录替换失败会回滚已有输出；前端 `lastSuccess` 记录当前页面会话内完成的构建，open 新项目时清空 | 已实现（会话范围） | 2026-07-22 确认刷新页面、进程重启和切换项目后的查询与恢复延期；后续需求不得把磁盘产物存在等同为 operation 可查询 |
| CG-06 预览服务 | 可停止的 preview session，不依赖 UI 进程 cwd 或 stdout | Spec 013 明确本期不启动 serve、不提供预览链接 | 延期 | 后续独立 spec，不在当前画板中以 disabled 控件暗示可用 |
| CG-07 路径浏览 | 目录浏览、home 限制、符号链接和权限语义 | 当前只接受既有绝对目录，`loadDocument` 做绝对路径与目录检查；没有目录浏览 API | 延期 | 后续独立 spec；当前 Project Open 只设计文本输入、加载与错误恢复 |

## 当前跨层状态

- CG-01、CG-02 已收口，并已进入 Issue #29 最终 Pencil 基线和 Issue #30 实现。
- CG-03 的底层 revision/原子写入合同、“重新读取”交互和浏览器回归均已完成。
- CG-04 已明确只返回结构化阶段、Stats、Warnings、Error 与输出路径，不再把通用日志尾部列为本期合同。
- CG-05 已限定为磁盘产物回滚保护与当前页面会话内定位；刷新或重启后的 operation 查询明确延期。

## 完成记录

1. Issue #29 基于已收口的 CG-01、CG-02 建立了最终 Foundations、组件状态与完整页面。
2. Issue #30 实现冻结基线，并补齐 CG-03 的浏览器恢复路径与验收；底层 revision/原子写入合同保持不变。

## 2026-07-22 维护者确认

- CG-01、CG-02 留在 Spec 013，已补齐 API、前端映射与自动化测试。
- CG-03 固定为 revision 冲突后阻止写入并提供“重新读取”恢复路径；浏览器实现与回归由 Issue #30 完成。
- CG-04 的通用日志尾部延期；现有结构化构建结果即本期合同。
- CG-05 限定为当前 UI 页面会话；跨刷新、重启和项目切换的 operation 查询延期。
