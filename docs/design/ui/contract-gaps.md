# M6 合同缺口

| 缺口 | 最小合同 | 归属 | 状态 |
|---|---|---|---|
| CG-01 配置 document | `config.Load` 只返回多层合并 `Config`，UI 不能用它回写仓库原始语义 | 导出仓库域未合并 document、字段是否存在、canonical YAML；仓库三禁止域产生警告 | `internal/config` | 待补 |
| CG-02 结构化校验 | 现有未知字段和 `toc_panel` 主要是 warning，坏 glob 在 `OptionsFor` 可静默跳过 | `ValidateRepositoryDocument` 返回 `{Path,Code,Message}`，明确阻断/警告边界 | `internal/config` | 待补 |
| CG-03 写入并发 | 当前没有 revision、diff 或写入 API | SHA-256 revision、unified diff、同目录 temp + rename 原子提交，冲突为 `409` | `internal/config` | 待补 |
| CG-04 构建服务 | CLI build 编排在 Cobra 闭包中，stdout 不是 UI 合同 | `internal/ui` package-private build service 返回阶段、Stats、warnings、错误；不得 shell 调 CLI | `internal/ui` | 待补 |
| CG-05 输出生命周期 | Worktree 构建会遍历仓库；仓库内输出会回流为源内容 | 用户缓存 project-hash 输出根、单项目一个 operation、保留最近成功结果 | `internal/ui` | 待补 |
| CG-06 预览服务 | `server.Run` 依赖 cwd、删除临时根且只把 URL 写 stdout | 本期不做；后续独立定义可停止 preview session | `internal/server` | 延期 |
| CG-07 路径浏览 | 目录树、home 限制、符号链接尚无 API 合同 | 本期只接受绝对路径；后续单独定义浏览与权限语义 | 后续 spec | 延期 |
